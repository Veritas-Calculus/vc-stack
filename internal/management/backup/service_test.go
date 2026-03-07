package backup

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
		{"BackupOffering", BackupOffering{}.TableName()},
		{"Backup", Backup{}.TableName()},
		{"BackupSchedule", BackupSchedule{}.TableName()},
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
		"GET:/api/v1/backup-offerings",
		"POST:/api/v1/backup-offerings",
		"DELETE:/api/v1/backup-offerings/:id",
		"GET:/api/v1/backups",
		"POST:/api/v1/backups",
		"POST:/api/v1/backups/:id/restore",
		"DELETE:/api/v1/backups/:id",
		"GET:/api/v1/backup-schedules",
		"POST:/api/v1/backup-schedules",
		"PUT:/api/v1/backup-schedules/:id",
		"DELETE:/api/v1/backup-schedules/:id",
	}
	for _, r := range required {
		if !paths[r] {
			t.Errorf("missing route: %s", r)
		}
	}
}

func TestSeedDefaults(t *testing.T) {
	db := testDB(t)
	l, _ := zap.NewDevelopment()
	svc, _ := NewService(Config{DB: db, Logger: l})

	// seedDefaults is called during NewService.
	var count int64
	svc.db.Model(&BackupOffering{}).Count(&count)
	if count == 0 {
		t.Error("seedDefaults did not create any backup offerings")
	}
}

func TestBkID(t *testing.T) {
	id := bkID()
	if len(id) == 0 {
		t.Error("bkID() returned empty string")
	}
	id2 := bkID()
	if id == id2 {
		t.Error("bkID() returned duplicate IDs")
	}
}
