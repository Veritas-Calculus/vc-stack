package alerting

import (
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) *Service {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{
		DB:     db,
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("failed to create alerting service: %v", err)
	}
	t.Cleanup(func() { svc.Stop() })
	return svc
}

func TestCreateRule(t *testing.T) {
	svc := setupTestService(t)

	rule, err := svc.CreateRule(1, &CreateRuleRequest{
		Name:      "High CPU",
		Metric:    "cpu_percent",
		Operator:  "gt",
		Threshold: 80,
		Duration:  "5m",
		Severity:  "warning",
		Channel:   "webhook",
	})
	if err != nil {
		t.Fatalf("CreateRule error: %v", err)
	}
	if rule.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if rule.Name != "High CPU" {
		t.Errorf("expected name 'High CPU', got %q", rule.Name)
	}
	if rule.State != "ok" {
		t.Errorf("expected state 'ok', got %q", rule.State)
	}
	if !rule.Enabled {
		t.Error("expected rule to be enabled by default")
	}
}

func TestListRules(t *testing.T) {
	svc := setupTestService(t)

	for _, name := range []string{"rule-1", "rule-2", "rule-3"} {
		_, _ = svc.CreateRule(1, &CreateRuleRequest{
			Name: name, Metric: "cpu_percent", Operator: "gt", Threshold: 90,
		})
	}

	rules, err := svc.ListRules()
	if err != nil {
		t.Fatalf("ListRules error: %v", err)
	}
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestGetRule(t *testing.T) {
	svc := setupTestService(t)
	created, _ := svc.CreateRule(1, &CreateRuleRequest{
		Name: "get-test", Metric: "memory_percent", Operator: "gt", Threshold: 90,
	})

	rule, err := svc.GetRule(created.ID)
	if err != nil {
		t.Fatalf("GetRule error: %v", err)
	}
	if rule.Name != "get-test" {
		t.Errorf("expected name 'get-test', got %q", rule.Name)
	}
}

func TestUpdateRule(t *testing.T) {
	svc := setupTestService(t)
	created, _ := svc.CreateRule(1, &CreateRuleRequest{
		Name: "update-test", Metric: "cpu_percent", Operator: "gt", Threshold: 80,
	})

	updated, err := svc.UpdateRule(created.ID, &CreateRuleRequest{
		Name: "updated-name", Metric: "memory_percent", Operator: "gte", Threshold: 95,
	})
	if err != nil {
		t.Fatalf("UpdateRule error: %v", err)
	}
	if updated.Metric != "memory_percent" {
		t.Errorf("expected metric 'memory_percent', got %q", updated.Metric)
	}
}

func TestDeleteRule(t *testing.T) {
	svc := setupTestService(t)
	created, _ := svc.CreateRule(1, &CreateRuleRequest{
		Name: "delete-test", Metric: "cpu_percent", Operator: "gt", Threshold: 80,
	})

	if err := svc.DeleteRule(created.ID); err != nil {
		t.Fatalf("DeleteRule error: %v", err)
	}

	_, err := svc.GetRule(created.ID)
	if err == nil {
		t.Error("expected error after deletion, got nil")
	}
}

func TestToggleRule(t *testing.T) {
	svc := setupTestService(t)
	created, _ := svc.CreateRule(1, &CreateRuleRequest{
		Name: "toggle-test", Metric: "cpu_percent", Operator: "gt", Threshold: 80,
	})

	if err := svc.ToggleRule(created.ID, false); err != nil {
		t.Fatalf("toggle off: %v", err)
	}

	rule, _ := svc.GetRule(created.ID)
	if rule.Enabled {
		t.Error("expected rule to be disabled")
	}

	if err := svc.ToggleRule(created.ID, true); err != nil {
		t.Fatalf("toggle on: %v", err)
	}
	rule, _ = svc.GetRule(created.ID)
	if !rule.Enabled {
		t.Error("expected rule to be enabled")
	}
}

func TestEvaluateThresholds(t *testing.T) {
	tests := []struct {
		value     float64
		operator  string
		threshold float64
		expected  bool
	}{
		{85, "gt", 80, true},
		{80, "gt", 80, false},
		{80, "gte", 80, true},
		{79, "gte", 80, false},
		{70, "lt", 80, true},
		{80, "lt", 80, false},
		{80, "lte", 80, true},
		{81, "lte", 80, false},
		{80, "eq", 80, true},
		{81, "eq", 80, false},
		{50, "unknown", 80, false},
	}

	for _, tt := range tests {
		result := evaluate(tt.value, tt.operator, tt.threshold)
		if result != tt.expected {
			t.Errorf("evaluate(%v, %q, %v) = %v, want %v",
				tt.value, tt.operator, tt.threshold, result, tt.expected)
		}
	}
}

func TestListHistory(t *testing.T) {
	svc := setupTestService(t)
	history, err := svc.ListHistory(0, 10)
	if err != nil {
		t.Fatalf("ListHistory error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 history items, got %d", len(history))
	}
}
