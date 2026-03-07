package autoscale

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func TestNewService(t *testing.T) {
	db := testDB(t)
	l, _ := zap.NewDevelopment()
	svc, err := NewService(Config{DB: db, Logger: l})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestModels_TableName(t *testing.T) {
	tests := []struct {
		name  string
		table string
	}{
		{"AutoScaleVMGroup", AutoScaleVMGroup{}.TableName()},
		{"AutoScalePolicy", AutoScalePolicy{}.TableName()},
		{"AutoScaleActivity", AutoScaleActivity{}.TableName()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.table == "" {
				t.Errorf("%s.TableName() is empty", tt.name)
			}
		})
	}
}

func TestSetupRoutes(t *testing.T) {
	db := testDB(t)
	l, _ := zap.NewDevelopment()
	svc, _ := NewService(Config{DB: db, Logger: l})

	router := gin.New()
	svc.SetupRoutes(router)

	routes := router.Routes()
	paths := make(map[string]bool)
	for _, r := range routes {
		paths[r.Method+":"+r.Path] = true
	}

	required := []string{
		"GET:/api/v1/autoscale-groups",
		"POST:/api/v1/autoscale-groups",
		"GET:/api/v1/autoscale-groups/:id",
		"PUT:/api/v1/autoscale-groups/:id",
		"DELETE:/api/v1/autoscale-groups/:id",
		"GET:/api/v1/autoscale-groups/:id/policies",
		"POST:/api/v1/autoscale-groups/:id/policies",
		"GET:/api/v1/autoscale-groups/:id/activity",
		"POST:/api/v1/autoscale-groups/:id/scale",
	}
	for _, r := range required {
		if !paths[r] {
			t.Errorf("missing route: %s", r)
		}
	}
}
