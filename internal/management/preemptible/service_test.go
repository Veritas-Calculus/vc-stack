package preemptible

import (
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setup(t *testing.T) *Service {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRegister(t *testing.T) {
	svc := setup(t)
	pi, err := svc.Register(1, "i-spot001", 2, 0.05, 12)
	if err != nil {
		t.Fatal(err)
	}
	if pi.Status != "running" {
		t.Errorf("expected running, got %q", pi.Status)
	}
	if pi.ExpiresAt == nil {
		t.Error("expected expiry set")
	}
}

func TestTerminate(t *testing.T) {
	svc := setup(t)
	svc.Register(1, "i-term001", 2, 0.05, 0)
	err := svc.Terminate("i-term001", "capacity_reclaim")
	if err != nil {
		t.Fatal(err)
	}
	instances, _ := svc.List(0)
	for _, i := range instances {
		if i.InstanceID == "i-term001" && i.Status != "terminated" {
			t.Errorf("expected terminated, got %q", i.Status)
		}
	}
}

func TestGetConfigDefaults(t *testing.T) {
	svc := setup(t)
	cfg, err := svc.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DiscountPct != 70 {
		t.Errorf("expected 70, got %f", cfg.DiscountPct)
	}
	if cfg.MaxLifetimeH != 24 {
		t.Errorf("expected 24, got %d", cfg.MaxLifetimeH)
	}
}

func TestUpdateConfig(t *testing.T) {
	svc := setup(t)
	svc.GetConfig() // initialize defaults
	svc.UpdateConfig(false, 50, 6)
	cfg, _ := svc.GetConfig()
	if cfg.DiscountPct != 50 {
		t.Errorf("expected 50, got %f", cfg.DiscountPct)
	}
	if cfg.MaxLifetimeH != 6 {
		t.Errorf("expected 6, got %d", cfg.MaxLifetimeH)
	}
}
