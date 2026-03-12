package organization

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

func TestCreateOrg(t *testing.T) {
	svc := setup(t)
	org, err := svc.CreateOrg("acme-corp", "ACME Corporation", "Test org", 1)
	if err != nil {
		t.Fatal(err)
	}
	if org.Status != "active" {
		t.Errorf("expected active, got %q", org.Status)
	}
}

func TestCreateOrgCreatesRootOU(t *testing.T) {
	svc := setup(t)
	org, _ := svc.CreateOrg("root-test", "Root Test", "", 1)
	ous, _ := svc.ListOUs(org.ID)
	if len(ous) != 1 {
		t.Errorf("expected 1 root OU, got %d", len(ous))
	}
	if ous[0].Name != "Root" {
		t.Errorf("expected Root, got %q", ous[0].Name)
	}
}

func TestCreateNestedOU(t *testing.T) {
	svc := setup(t)
	org, _ := svc.CreateOrg("nested-org", "Nested", "", 1)
	ous, _ := svc.ListOUs(org.ID)
	rootID := ous[0].ID
	child, err := svc.CreateOU(org.ID, &rootID, "Engineering")
	if err != nil {
		t.Fatal(err)
	}
	if child.Path != "Root/Engineering" {
		t.Errorf("expected Root/Engineering, got %q", child.Path)
	}
}

func TestMoveProject(t *testing.T) {
	svc := setup(t)
	org, _ := svc.CreateOrg("move-org", "Move", "", 1)
	ous, _ := svc.ListOUs(org.ID)
	err := svc.MoveProject(42, org.ID, ous[0].ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteOrgCascade(t *testing.T) {
	svc := setup(t)
	org, _ := svc.CreateOrg("del-org", "Del", "", 1)
	svc.DeleteOrg(org.ID)
	_, err := svc.GetOrg(org.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
