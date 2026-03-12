package abac

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
	conds := []Condition{{Key: "resource.tags.env", Operator: "equals", Value: "production"}}
	p, err := svc.CreatePolicy("prod-only", "Only production", "allow", "instance:*", "create,delete", conds, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p.Effect != "allow" {
		t.Errorf("expected allow, got %q", p.Effect)
	}
}

func TestInvalidEffect(t *testing.T) {
	svc := setup(t)
	_, err := svc.CreatePolicy("bad", "", "maybe", "*", "*", nil, 0)
	if err == nil {
		t.Error("expected error for invalid effect")
	}
}

func TestEvalAllowByTag(t *testing.T) {
	svc := setup(t)
	svc.CreatePolicy("env-prod", "", "allow", "instance:*", "create", []Condition{
		{Key: "resource.tags.env", Operator: "equals", Value: "production"},
	}, 10)

	result, err := svc.Evaluate(&EvalRequest{
		Action: "create", Resource: "instance:i-123",
		ResourceTags: map[string]string{"env": "production"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Error("expected allowed")
	}
	if result.MatchedPolicy != "env-prod" {
		t.Errorf("expected env-prod, got %q", result.MatchedPolicy)
	}
}

func TestEvalDenyByTag(t *testing.T) {
	svc := setup(t)
	svc.CreatePolicy("deny-dev", "", "deny", "instance:*", "*", []Condition{
		{Key: "resource.tags.env", Operator: "equals", Value: "dev"},
	}, 5) // higher priority

	result, _ := svc.Evaluate(&EvalRequest{
		Action: "delete", Resource: "instance:i-456",
		ResourceTags: map[string]string{"env": "dev"},
	})
	if result.Allowed {
		t.Error("expected denied")
	}
}

func TestEvalNoMatch(t *testing.T) {
	svc := setup(t)
	svc.CreatePolicy("specific", "", "deny", "volume:*", "delete", []Condition{
		{Key: "resource.tags.tier", Operator: "equals", Value: "critical"},
	}, 10)

	result, _ := svc.Evaluate(&EvalRequest{
		Action: "create", Resource: "instance:i-789",
		ResourceTags: map[string]string{"tier": "standard"},
	})
	if !result.Allowed {
		t.Error("expected default allow when no policy matches")
	}
}

func TestEvalInOperator(t *testing.T) {
	svc := setup(t)
	svc.CreatePolicy("region-restrict", "", "allow", "*", "*", []Condition{
		{Key: "resource.tags.region", Operator: "in", Value: `["us-east-1","eu-west-1"]`},
	}, 10)

	result, _ := svc.Evaluate(&EvalRequest{
		Action: "create", Resource: "instance:i-100",
		ResourceTags: map[string]string{"region": "us-east-1"},
	})
	if !result.Allowed {
		t.Error("expected allowed for in-list region")
	}
}

func TestEvalStartsWith(t *testing.T) {
	svc := setup(t)
	svc.CreatePolicy("team-prefix", "", "allow", "instance:*", "read", []Condition{
		{Key: "user.team", Operator: "starts_with", Value: "platform"},
	}, 10)

	result, _ := svc.Evaluate(&EvalRequest{
		Action: "read", Resource: "instance:i-200",
		UserAttrs: map[string]string{"team": "platform-infra"},
	})
	if !result.Allowed {
		t.Error("expected allowed for starts_with match")
	}
}

func TestTogglePolicy(t *testing.T) {
	svc := setup(t)
	p, _ := svc.CreatePolicy("toggle-test", "", "deny", "*", "*", []Condition{
		{Key: "resource.tags.env", Operator: "equals", Value: "test"},
	}, 1)
	svc.TogglePolicy(p.ID, false)

	// Disabled policy should not match
	result, _ := svc.Evaluate(&EvalRequest{
		Action: "delete", Resource: "instance:i-300",
		ResourceTags: map[string]string{"env": "test"},
	})
	if !result.Allowed {
		t.Error("disabled policy should not deny")
	}
}
