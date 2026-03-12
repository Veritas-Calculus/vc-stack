package storageqos

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

func TestCreatePolicy(t *testing.T) {
	svc := setup(t)
	p, err := svc.CreatePolicy(&CreatePolicyRequest{Name: "gp3", MaxIOPS: 16000, Tier: "premium"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Tier != "premium" {
		t.Errorf("expected premium, got %q", p.Tier)
	}
	if p.MaxIOPS != 16000 {
		t.Errorf("expected 16000, got %d", p.MaxIOPS)
	}
}

func TestDefaultValues(t *testing.T) {
	svc := setup(t)
	p, _ := svc.CreatePolicy(&CreatePolicyRequest{Name: "default-test"})
	if p.MaxIOPS != 3000 {
		t.Errorf("expected default 3000, got %d", p.MaxIOPS)
	}
	if p.Tier != "standard" {
		t.Errorf("expected standard, got %q", p.Tier)
	}
}

func TestAssignPolicyScaling(t *testing.T) {
	svc := setup(t)
	p, _ := svc.CreatePolicy(&CreatePolicyRequest{Name: "scale-test", MaxIOPS: 16000, PerGBIOPS: 50})
	// 100 GB * 50 IOPS/GB = 5000 effective IOPS
	vq, err := svc.AssignPolicy(1, p.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if vq.EffectiveIOPS != 5000 {
		t.Errorf("expected 5000, got %d", vq.EffectiveIOPS)
	}
}

func TestAssignPolicyCapped(t *testing.T) {
	svc := setup(t)
	p, _ := svc.CreatePolicy(&CreatePolicyRequest{Name: "cap-test", MaxIOPS: 3000, PerGBIOPS: 50})
	// 200 GB * 50 = 10000, capped at 3000
	vq, _ := svc.AssignPolicy(2, p.ID, 200)
	if vq.EffectiveIOPS != 3000 {
		t.Errorf("expected 3000 (capped), got %d", vq.EffectiveIOPS)
	}
}

func TestAssignPolicyMinIOPS(t *testing.T) {
	svc := setup(t)
	p, _ := svc.CreatePolicy(&CreatePolicyRequest{Name: "min-test", MinIOPS: 500, PerGBIOPS: 3})
	// 10 GB * 3 = 30, but min is 500
	vq, _ := svc.AssignPolicy(3, p.ID, 10)
	if vq.EffectiveIOPS != 500 {
		t.Errorf("expected min 500, got %d", vq.EffectiveIOPS)
	}
}
