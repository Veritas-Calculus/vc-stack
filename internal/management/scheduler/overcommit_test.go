package scheduler

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

func setupOvercommitDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&models.Host{}, &models.ServerGroup{}, &models.ServerGroupMember{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupOvercommitService(t *testing.T, db *gorm.DB, oc OvercommitConfig) *Service {
	t.Helper()
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop(), Overcommit: oc})
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}
	return svc
}

// TestOvercommit_CPURatio verifies that overcommit allows scheduling beyond physical capacity.
func TestOvercommit_CPURatio(t *testing.T) {
	db := setupOvercommitDB(t)

	// Host with 4 cores, 3 allocated -> 1 free physical core.
	seedHosts(t, db, []models.Host{
		{
			UUID: "host-1", Name: "node-1", Hostname: "n1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 4, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 3, RAMAllocatedMB: 0, DiskAllocatedGB: 0,
		},
	})

	t.Run("no overcommit rejects", func(t *testing.T) {
		svc := setupOvercommitService(t, db, OvercommitConfig{CPURatio: 1.0, RAMRatio: 1.0, DiskRatio: 1.0})
		// Need 4 vCPUs but only 1 free -> should fail.
		host, _ := svc.selectHost(context.Background(), ScheduleRequest{VCPUs: 4, RAMMB: 1024, DiskGB: 10})
		if host != nil {
			t.Error("expected nil host without overcommit")
		}
	})

	t.Run("4x overcommit allows", func(t *testing.T) {
		svc := setupOvercommitService(t, db, OvercommitConfig{CPURatio: 4.0, RAMRatio: 1.0, DiskRatio: 1.0})
		// Effective capacity: 4 * 4.0 = 16 vCPUs, 3 allocated -> 13 free. Need 4 -> OK.
		host, resp := svc.selectHost(context.Background(), ScheduleRequest{VCPUs: 4, RAMMB: 1024, DiskGB: 10})
		if host == nil {
			t.Fatalf("expected host with 4x CPU overcommit, got nil: %s", resp.Reason)
		}
		if host.UUID != "host-1" {
			t.Errorf("expected host-1, got %s", host.UUID)
		}
	})
}

// TestOvercommit_RAMRatio verifies RAM overcommit.
func TestOvercommit_RAMRatio(t *testing.T) {
	db := setupOvercommitDB(t)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-1", Name: "node-1", Hostname: "n1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 8192, DiskGB: 500,
			CPUAllocated: 0, RAMAllocatedMB: 7000, DiskAllocatedGB: 0,
		},
	})

	t.Run("no overcommit rejects", func(t *testing.T) {
		svc := setupOvercommitService(t, db, OvercommitConfig{CPURatio: 1.0, RAMRatio: 1.0, DiskRatio: 1.0})
		host, _ := svc.selectHost(context.Background(), ScheduleRequest{VCPUs: 1, RAMMB: 4096, DiskGB: 10})
		if host != nil {
			t.Error("expected nil host without RAM overcommit")
		}
	})

	t.Run("1.5x overcommit allows", func(t *testing.T) {
		svc := setupOvercommitService(t, db, OvercommitConfig{CPURatio: 1.0, RAMRatio: 1.5, DiskRatio: 1.0})
		// Effective RAM: 8192 * 1.5 = 12288 MB. 7000 allocated -> 5288 free. Need 4096 -> OK.
		host, resp := svc.selectHost(context.Background(), ScheduleRequest{VCPUs: 1, RAMMB: 4096, DiskGB: 10})
		if host == nil {
			t.Fatalf("expected host with 1.5x RAM overcommit, got nil: %s", resp.Reason)
		}
	})
}

// TestOvercommit_NormalizesInvalidRatios verifies that ratios < 1.0 are clamped.
func TestOvercommit_NormalizesInvalidRatios(t *testing.T) {
	svc, err := NewService(Config{
		DB:     nil,
		Logger: zap.NewNop(),
		Overcommit: OvercommitConfig{
			CPURatio:  0.5, // invalid, should become 1.0
			RAMRatio:  -1,  // invalid, should become 1.0
			DiskRatio: 0,   // invalid, should become 1.0
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if svc.overcommit.CPURatio != 1.0 {
		t.Errorf("expected CPU ratio 1.0, got %f", svc.overcommit.CPURatio)
	}
	if svc.overcommit.RAMRatio != 1.0 {
		t.Errorf("expected RAM ratio 1.0, got %f", svc.overcommit.RAMRatio)
	}
	if svc.overcommit.DiskRatio != 1.0 {
		t.Errorf("expected Disk ratio 1.0, got %f", svc.overcommit.DiskRatio)
	}
}


// TestServerGroupPolicyValidation tests policy string validation.
func TestServerGroupPolicyValidation(t *testing.T) {
	valid := []string{"affinity", "anti-affinity", "soft-affinity", "soft-anti-affinity"}
	for _, p := range valid {
		if !models.ValidateServerGroupPolicy(p) {
			t.Errorf("expected %q to be valid", p)
		}
	}

	invalid := []string{"", "random", "foo", "AFFINITY"}
	for _, p := range invalid {
		if models.ValidateServerGroupPolicy(p) {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}
