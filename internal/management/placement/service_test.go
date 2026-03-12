package placement

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

func TestCreatePlacementGroup(t *testing.T) {
	svc := setup(t)
	pg, err := svc.Create(1, "web-spread", "anti-affinity")
	if err != nil {
		t.Fatal(err)
	}
	if pg.Strategy != "anti-affinity" {
		t.Errorf("expected anti-affinity, got %q", pg.Strategy)
	}
}

func TestInvalidStrategy(t *testing.T) {
	svc := setup(t)
	_, err := svc.Create(1, "bad", "random")
	if err == nil {
		t.Error("expected error for invalid strategy")
	}
}

func TestAddMember(t *testing.T) {
	svc := setup(t)
	pg, _ := svc.Create(1, "member-group", "affinity")
	m, err := svc.AddMember(pg.ID, "i-abc123")
	if err != nil {
		t.Fatal(err)
	}
	if m.InstanceID != "i-abc123" {
		t.Errorf("expected i-abc123, got %q", m.InstanceID)
	}
}

func TestListWithMembers(t *testing.T) {
	svc := setup(t)
	pg, _ := svc.Create(1, "list-group", "anti-affinity")
	svc.AddMember(pg.ID, "i-1")
	svc.AddMember(pg.ID, "i-2")
	groups, _ := svc.List(0)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(groups[0].Members))
	}
}

func TestDeleteGroupCascade(t *testing.T) {
	svc := setup(t)
	pg, _ := svc.Create(1, "del-group", "soft-anti-affinity")
	svc.AddMember(pg.ID, "i-1")
	svc.Delete(pg.ID)
	_, err := svc.Get(pg.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
