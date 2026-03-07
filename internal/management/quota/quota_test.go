package quota

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&QuotaSet{}, &QuotaUsage{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupTestService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create quota service: %v", err)
	}
	return svc
}

// TestCheckQuota_UnderLimit verifies that CheckQuota passes when usage is under quota.
func TestCheckQuota_UnderLimit(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	// Set a quota of 10 instances for tenant-1.
	db.Create(&QuotaSet{TenantID: "tenant-1", Instances: 10, VCPUs: 20, RAMMB: 51200, DiskGB: 1000})
	db.Create(&QuotaUsage{TenantID: "tenant-1", Instances: 5})

	if err := svc.CheckQuota("tenant-1", "instances", 1); err != nil {
		t.Errorf("expected quota check to pass, got error: %v", err)
	}
}

// TestCheckQuota_ExceedsLimit verifies that CheckQuota returns an error
// when the operation would exceed the quota.
func TestCheckQuota_ExceedsLimit(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	db.Create(&QuotaSet{TenantID: "tenant-2", Instances: 5, VCPUs: 10, RAMMB: 10240, DiskGB: 500})
	db.Create(&QuotaUsage{TenantID: "tenant-2", Instances: 5})

	err := svc.CheckQuota("tenant-2", "instances", 1)
	if err == nil {
		t.Fatal("expected quota check to fail, got nil")
	}
	if _, ok := err.(*QuotaExceededError); !ok {
		t.Errorf("expected QuotaExceededError, got %T: %v", err, err)
	}
}

// TestCheckQuota_UnlimitedQuota verifies that -1 means unlimited.
func TestCheckQuota_UnlimitedQuota(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	db.Create(&QuotaSet{TenantID: "tenant-3", Instances: -1, VCPUs: -1, RAMMB: -1, DiskGB: -1})
	db.Create(&QuotaUsage{TenantID: "tenant-3", Instances: 999})

	if err := svc.CheckQuota("tenant-3", "instances", 100); err != nil {
		t.Errorf("expected unlimited quota check to pass, got error: %v", err)
	}
}

// TestCheckQuota_DefaultQuota verifies that the default quota kicks in when
// no custom quota is set for the tenant.
func TestCheckQuota_DefaultQuota(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	// No quota set for tenant-4, defaults should apply.
	err := svc.CheckQuota("tenant-4", "instances", 1)
	if err != nil {
		t.Errorf("expected default quota to allow creation, got error: %v", err)
	}
}

// TestCheckQuota_MultipleResourceTypes tests quota checking for different resource types.
func TestCheckQuota_MultipleResourceTypes(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	db.Create(&QuotaSet{TenantID: "tenant-5", Instances: 10, VCPUs: 8, RAMMB: 16384, DiskGB: 200})
	db.Create(&QuotaUsage{TenantID: "tenant-5", Instances: 1, VCPUs: 4, RAMMB: 8192, DiskGB: 100})

	tests := []struct {
		name         string
		resourceType string
		delta        int
		wantErr      bool
	}{
		{"instances under", "instances", 1, false},
		{"vcpus under", "vcpus", 2, false},
		{"vcpus exceed", "vcpus", 5, true},
		{"ram_mb under", "ram_mb", 4096, false},
		{"ram_mb exceed", "ram_mb", 9000, true},
		{"disk_gb under", "disk_gb", 50, false},
		{"disk_gb exceed", "disk_gb", 150, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.CheckQuota("tenant-5", tt.resourceType, tt.delta)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckQuota(%s, %d) = %v, wantErr %v", tt.resourceType, tt.delta, err, tt.wantErr)
			}
		})
	}
}

// TestUpdateUsage verifies that usage is correctly updated with atomic increments.
func TestUpdateUsage(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	db.Create(&QuotaUsage{TenantID: "tenant-6", Instances: 0, VCPUs: 0})

	if err := svc.UpdateUsage("tenant-6", "instances", 3); err != nil {
		t.Fatalf("failed to update usage: %v", err)
	}

	var usage QuotaUsage
	db.Where("tenant_id = ?", "tenant-6").First(&usage)

	if usage.Instances != 3 {
		t.Errorf("expected instances=3, got %d", usage.Instances)
	}

	// Decrement.
	if err := svc.UpdateUsage("tenant-6", "instances", -1); err != nil {
		t.Fatalf("failed to decrement usage: %v", err)
	}

	db.Where("tenant_id = ?", "tenant-6").First(&usage)
	if usage.Instances != 2 {
		t.Errorf("expected instances=2 after decrement, got %d", usage.Instances)
	}
}

// TestUpdateUsage_CreatesRecordIfMissing verifies that UpdateUsage creates
// a usage record if one does not exist.
func TestUpdateUsage_CreatesRecordIfMissing(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	if err := svc.UpdateUsage("new-tenant", "instances", 1); err != nil {
		t.Fatalf("failed to update usage for new tenant: %v", err)
	}

	var count int64
	db.Model(&QuotaUsage{}).Where("tenant_id = ?", "new-tenant").Count(&count)
	if count != 1 {
		t.Errorf("expected usage record created, got count=%d", count)
	}
}

// TestQuotaHTTPEndpoint_GetQuota tests the GET /api/v1/quotas/:tenant_id endpoint.
func TestQuotaHTTPEndpoint_GetQuota(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	db.Create(&QuotaSet{TenantID: "http-tenant", Instances: 20, VCPUs: 40, RAMMB: 102400, DiskGB: 2000})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quotas/tenants/http-tenant", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "http-tenant") {
		t.Errorf("response body should contain tenant_id, got: %s", w.Body.String())
	}
}

// TestQuotaExceededError_Message tests the error message format.
func TestQuotaExceededError_Message(t *testing.T) {
	err := &QuotaExceededError{Resource: "instances", Limit: 10, Current: 10}
	msg := err.Error()
	if !strings.Contains(msg, "instances") {
		t.Errorf("error message should mention the resource, got: %s", msg)
	}
}
