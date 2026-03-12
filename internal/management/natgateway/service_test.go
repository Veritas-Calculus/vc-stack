package natgateway

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

func TestCreateNATGateway(t *testing.T) {
	svc := setup(t)
	gw, err := svc.Create(1, "nat-prod", 10, 200)
	if err != nil {
		t.Fatal(err)
	}
	if gw.BandwidthMbps != 200 {
		t.Errorf("expected 200, got %d", gw.BandwidthMbps)
	}
	if gw.Status != "creating" {
		t.Errorf("expected creating, got %q", gw.Status)
	}
}

func TestDefaultBandwidth(t *testing.T) {
	svc := setup(t)
	gw, _ := svc.Create(1, "nat-default", 10, 0)
	if gw.BandwidthMbps != 100 {
		t.Errorf("expected default 100, got %d", gw.BandwidthMbps)
	}
}

func TestAddSNATRule(t *testing.T) {
	svc := setup(t)
	gw, _ := svc.Create(1, "nat-snat", 10, 100)
	rule, err := svc.AddRule(gw.ID, "snat", "10.0.1.0/24", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Type != "snat" {
		t.Errorf("expected snat, got %q", rule.Type)
	}
}

func TestListRules(t *testing.T) {
	svc := setup(t)
	gw, _ := svc.Create(1, "nat-rules", 10, 100)
	svc.AddRule(gw.ID, "snat", "10.0.1.0/24", "", 0)
	svc.AddRule(gw.ID, "dnat", "", "192.168.1.10", 8080)
	rules, _ := svc.ListRules(gw.ID)
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
}

func TestDeleteCascade(t *testing.T) {
	svc := setup(t)
	gw, _ := svc.Create(1, "nat-del", 10, 100)
	svc.AddRule(gw.ID, "snat", "10.0.0.0/8", "", 0)
	svc.Delete(gw.ID)
	_, err := svc.Get(gw.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
