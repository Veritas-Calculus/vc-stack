package launchtemplate

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

func TestCreateTemplate(t *testing.T) {
	svc := setup(t)
	tmpl, err := svc.CreateTemplate(1, &CreateTemplateRequest{Name: "web-template", FlavorID: 2, ImageID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Version != 1 {
		t.Errorf("expected version 1, got %d", tmpl.Version)
	}
}

func TestListTemplates(t *testing.T) {
	svc := setup(t)
	svc.CreateTemplate(1, &CreateTemplateRequest{Name: "t1"})
	svc.CreateTemplate(1, &CreateTemplateRequest{Name: "t2"})
	templates, _ := svc.ListTemplates()
	if len(templates) != 2 {
		t.Errorf("expected 2, got %d", len(templates))
	}
}

func TestCreateScalingGroup(t *testing.T) {
	svc := setup(t)
	tmpl, _ := svc.CreateTemplate(1, &CreateTemplateRequest{Name: "asg-tmpl"})
	g, err := svc.CreateGroup(1, &CreateGroupRequest{Name: "web-asg", LaunchTemplateID: tmpl.ID, MinSize: 2, MaxSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if g.MinSize != 2 {
		t.Errorf("expected min 2, got %d", g.MinSize)
	}
	if g.MaxSize != 20 {
		t.Errorf("expected max 20, got %d", g.MaxSize)
	}
	if g.CooldownSeconds != 300 {
		t.Errorf("expected 300, got %d", g.CooldownSeconds)
	}
}

func TestAddPolicy(t *testing.T) {
	svc := setup(t)
	tmpl, _ := svc.CreateTemplate(1, &CreateTemplateRequest{Name: "policy-tmpl"})
	g, _ := svc.CreateGroup(1, &CreateGroupRequest{Name: "policy-asg", LaunchTemplateID: tmpl.ID})
	p, err := svc.AddPolicy(g.ID, &CreatePolicyRequest{Name: "cpu-target", MetricName: "cpu_percent", TargetValue: 70})
	if err != nil {
		t.Fatal(err)
	}
	if p.PolicyType != "target_tracking" {
		t.Errorf("expected target_tracking, got %q", p.PolicyType)
	}
	if p.TargetValue != 70 {
		t.Errorf("expected 70, got %f", p.TargetValue)
	}
}

func TestSetDesiredCapacity(t *testing.T) {
	svc := setup(t)
	tmpl, _ := svc.CreateTemplate(1, &CreateTemplateRequest{Name: "cap-tmpl"})
	g, _ := svc.CreateGroup(1, &CreateGroupRequest{Name: "cap-asg", LaunchTemplateID: tmpl.ID, DesiredCapacity: 3})
	svc.SetDesiredCapacity(g.ID, 5)
	got, _ := svc.GetGroup(g.ID)
	if got.DesiredCapacity != 5 {
		t.Errorf("expected 5, got %d", got.DesiredCapacity)
	}
}

func TestDeleteGroupCascade(t *testing.T) {
	svc := setup(t)
	tmpl, _ := svc.CreateTemplate(1, &CreateTemplateRequest{Name: "del-tmpl"})
	g, _ := svc.CreateGroup(1, &CreateGroupRequest{Name: "del-asg", LaunchTemplateID: tmpl.ID})
	svc.AddPolicy(g.ID, &CreatePolicyRequest{Name: "p1", MetricName: "cpu_percent", TargetValue: 80})
	svc.DeleteGroup(g.ID)
	_, err := svc.GetGroup(g.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
