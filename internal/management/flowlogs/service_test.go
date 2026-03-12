package flowlogs

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTest(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestCreateConfig(t *testing.T) {
	svc := setupTest(t)
	cfg, err := svc.CreateConfig(1, &CreateConfigRequest{Name: "web-flow", NetworkID: 1, Direction: "ingress"})
	if err != nil {
		t.Fatalf("CreateConfig: %v", err)
	}
	if cfg.Direction != "ingress" {
		t.Errorf("expected ingress, got %q", cfg.Direction)
	}
	if !cfg.Enabled {
		t.Error("expected enabled by default")
	}
}

func TestListConfigs(t *testing.T) {
	svc := setupTest(t)
	svc.CreateConfig(1, &CreateConfigRequest{Name: "cfg-1", NetworkID: 1})
	svc.CreateConfig(1, &CreateConfigRequest{Name: "cfg-2", NetworkID: 2})

	configs, err := svc.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2, got %d", len(configs))
	}
}

func TestRecordAndQueryFlows(t *testing.T) {
	svc := setupTest(t)

	svc.RecordFlow(&FlowLogEntry{
		ConfigID: 1, Timestamp: time.Now(), Direction: "IN", Action: "ACCEPT",
		Protocol: "TCP", SrcIP: "10.0.1.5", SrcPort: 45678, DstIP: "10.0.1.10", DstPort: 80,
		Bytes: 1024, Packets: 10, NetworkID: 1,
	})
	svc.RecordFlow(&FlowLogEntry{
		ConfigID: 1, Timestamp: time.Now(), Direction: "IN", Action: "REJECT",
		Protocol: "TCP", SrcIP: "10.0.2.5", SrcPort: 12345, DstIP: "10.0.1.10", DstPort: 22,
		Bytes: 64, Packets: 1, NetworkID: 1,
	})

	// Query all
	entries, total, err := svc.QueryFlows(&FlowLogQuery{})
	if err != nil {
		t.Fatalf("QueryFlows: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 total, got %d", total)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// Filter by action
	entries, total, _ = svc.QueryFlows(&FlowLogQuery{Action: "REJECT"})
	if total != 1 {
		t.Errorf("expected 1 rejected, got %d", total)
	}

	// Filter by src IP
	entries, _, _ = svc.QueryFlows(&FlowLogQuery{SrcIP: "10.0.1.5"})
	if len(entries) != 1 {
		t.Errorf("expected 1 from 10.0.1.5, got %d", len(entries))
	}
}

func TestToggleConfig(t *testing.T) {
	svc := setupTest(t)
	cfg, _ := svc.CreateConfig(1, &CreateConfigRequest{Name: "toggle", NetworkID: 1})

	svc.ToggleConfig(cfg.ID, false)
	configs, _ := svc.ListConfigs()
	for _, c := range configs {
		if c.ID == cfg.ID && c.Enabled {
			t.Error("expected disabled")
		}
	}
}
