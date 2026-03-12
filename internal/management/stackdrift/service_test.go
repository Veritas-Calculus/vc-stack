package stackdrift

import (
	"encoding/json"
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

func tmpl(resources ...map[string]interface{}) string {
	b, _ := json.Marshal(map[string]interface{}{"resources": resources})
	return string(b)
}

func TestCreateVersion(t *testing.T) {
	svc := setup(t)
	v, err := svc.CreateVersion(1, tmpl(map[string]interface{}{"id": "vpc-1", "type": "vpc", "name": "main"}))
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != 1 {
		t.Errorf("expected v1, got v%d", v.Version)
	}
}

func TestVersioningSupersedes(t *testing.T) {
	svc := setup(t)
	svc.CreateVersion(1, tmpl())
	v2, _ := svc.CreateVersion(1, tmpl())
	if v2.Version != 2 {
		t.Errorf("expected v2, got v%d", v2.Version)
	}

	vs, _ := svc.ListVersions(1)
	active := 0
	for _, v := range vs {
		if v.Status == "active" {
			active++
		}
	}
	if active != 1 {
		t.Errorf("expected 1 active, got %d", active)
	}
}

func TestRollback(t *testing.T) {
	svc := setup(t)
	v1, _ := svc.CreateVersion(1, `{"resources":[{"id":"r1"}]}`)
	svc.CreateVersion(1, `{"resources":[{"id":"r1"},{"id":"r2"}]}`)
	v3, _ := svc.Rollback(1, v1.Version)
	if v3.Version != 3 {
		t.Errorf("rollback should create v3, got v%d", v3.Version)
	}
	if v3.Template != v1.Template {
		t.Error("rollback template should match v1")
	}
}

func TestDetectDrift(t *testing.T) {
	svc := setup(t)
	svc.CreateVersion(1, tmpl(
		map[string]interface{}{"id": "vm-1", "type": "instance", "name": "web"},
		map[string]interface{}{"id": "vol-1", "type": "volume", "name": "data"},
	))
	report, err := svc.DetectDrift(1)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalRes != 2 {
		t.Errorf("expected 2 resources, got %d", report.TotalRes)
	}
}

func TestDepGraph(t *testing.T) {
	svc := setup(t)
	svc.CreateVersion(1, tmpl(
		map[string]interface{}{"id": "vpc-1", "type": "vpc", "name": "main"},
		map[string]interface{}{"id": "subnet-1", "type": "subnet", "name": "web", "depends_on": []string{"vpc-1"}},
	))
	nodes, err := svc.GetDepGraph(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
	if len(nodes[1].DependsOn) != 1 {
		t.Errorf("subnet should depend on vpc")
	}
}
