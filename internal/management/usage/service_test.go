package usage

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
		"GET:/api/v1/usage",
		"GET:/api/v1/usage/summary",
		"GET:/api/v1/tariffs",
		"POST:/api/v1/tariffs",
		"PUT:/api/v1/tariffs/:id",
		"DELETE:/api/v1/tariffs/:id",
		"GET:/api/v1/billing/summary",
		"POST:/api/v1/billing/credit",
	}
	for _, r := range required {
		if !paths[r] {
			t.Errorf("missing route: %s", r)
		}
	}
}

func TestSeedDefaultTariffs(t *testing.T) {
	db := testDB(t)
	l, _ := zap.NewDevelopment()
	svc, _ := NewService(Config{DB: db, Logger: l})

	// seedDefaultTariffs is called during NewService.
	// Verify tariffs were created.
	var count int64
	svc.db.Model(&Tariff{}).Count(&count)
	if count == 0 {
		t.Error("seedDefaultTariffs did not create any tariffs")
	}
}
