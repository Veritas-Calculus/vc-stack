package network

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListNetworks(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	logger := zap.NewNop()

	// Migrate only the models needed for this test (avoid PostgreSQL-specific types)
	err := db.AutoMigrate(&Network{}, &Subnet{}, &NetworkPort{}, &Router{}, &RouterInterface{},
		&FloatingIP{}, &SecurityGroup{}, &SecurityGroupRule{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	svc := &Service{
		db:     db,
		logger: logger,
		config: Config{DB: db, Logger: logger},
		driver: NewOVNDriver(logger, OVNConfig{}),
		ipam:   NewIPAM(db, logger),
	}

	// Seed
	svc.db.Create(&Network{ID: "net-1", Name: "default"})

	r := gin.New()
	svc.SetupRoutes(r)

	req, _ := http.NewRequest("GET", "/api/v1/networks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPAMConcurrency(t *testing.T) {
	// Use a file-based sqlite for concurrency testing to allow shared state
	dbFile := "test_ipam_concurrent.db"
	defer os.Remove(dbFile)

	db, _ := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})

	// Optimization for concurrent SQLite access
	sqlDB, _ := db.DB()
	sqlDB.Exec("PRAGMA journal_mode=WAL;")
	sqlDB.Exec("PRAGMA busy_timeout=5000;") // 5 seconds wait
	sqlDB.SetMaxOpenConns(1)                // Force serialize for SQLite

	logger := zap.NewNop()
	svc := &Service{
		db:     db,
		logger: logger,
		config: Config{DB: db, Logger: logger},
		driver: NewOVNDriver(logger, OVNConfig{}),
		ipam:   NewIPAM(db, logger),
	}

	// Ensure tables exist (avoid migrateDatabase which includes PostgreSQL-specific types)
	db.AutoMigrate(&Network{}, &Subnet{}, &NetworkPort{}, &IPAllocation{})

	// 1. Create a subnet
	subnet := Subnet{
		ID:        "sub-concurrent",
		NetworkID: "net-1",
		CIDR:      "192.168.1.0/24",
		Gateway:   "192.168.1.1",
	}
	db.Create(&subnet)

	// 2. Launch 20 concurrent allocation requests
	count := 20
	results := make(chan string, count)
	errs := make(chan error, count)

	for k := 0; k < count; k++ {
		go func(idx int) {
			// Each goroutine uses the same service instance (shared DB pool)
			ip, err := svc.ipam.Allocate(context.Background(), &subnet, fmt.Sprintf("port-%d", idx))
			if err != nil {
				errs <- err
			} else {
				results <- ip
			}
		}(k)
	}

	// 3. Collect results
	allocated := make(map[string]bool)
	for k := 0; k < count; k++ {
		select {
		case ip := <-results:
			if allocated[ip] {
				t.Errorf("Duplicate IP allocated: %s", ip)
			}
			allocated[ip] = true
		case err := <-errs:
			t.Errorf("Allocation failed: %v", err)
		}
	}

	assert.Len(t, allocated, count)
}
