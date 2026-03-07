package domain

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

func TestModel_TableName(t *testing.T) {
	d := Domain{}
	if d.TableName() == "" {
		t.Error("Domain.TableName() is empty")
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
		"GET:/api/v1/domains",
		"GET:/api/v1/domains/tree",
		"GET:/api/v1/domains/:id",
		"POST:/api/v1/domains",
		"PUT:/api/v1/domains/:id",
		"DELETE:/api/v1/domains/:id",
	}
	for _, r := range required {
		if !paths[r] {
			t.Errorf("missing route: %s", r)
		}
	}
}

func TestSeedRoot(t *testing.T) {
	db := testDB(t)
	l, _ := zap.NewDevelopment()
	_, _ = NewService(Config{DB: db, Logger: l})

	// seedRoot is called during NewService.
	var root Domain
	result := db.Where("name = ?", "ROOT").First(&root)
	if result.Error != nil {
		t.Errorf("ROOT domain not found: %v", result.Error)
	}
	if root.Name != "ROOT" {
		t.Errorf("root domain name = %q, want ROOT", root.Name)
	}
}
