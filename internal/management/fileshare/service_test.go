package fileshare

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

func TestCreateFileShare(t *testing.T) {
	svc := setup(t)
	fs, err := svc.Create(1, "shared-data", "nfs", 200)
	if err != nil {
		t.Fatal(err)
	}
	if fs.ExportPath != "/exports/shared-data" {
		t.Errorf("expected /exports/shared-data, got %q", fs.ExportPath)
	}
	if fs.Protocol != "nfs" {
		t.Errorf("expected nfs, got %q", fs.Protocol)
	}
}

func TestMinSize(t *testing.T) {
	svc := setup(t)
	fs, _ := svc.Create(1, "small", "", 5)
	if fs.SizeGB < 10 {
		t.Errorf("expected min 10, got %d", fs.SizeGB)
	}
}

func TestResize(t *testing.T) {
	svc := setup(t)
	fs, _ := svc.Create(1, "resize-share", "", 100)
	svc.Resize(fs.ID, 500)
	got, _ := svc.Get(fs.ID)
	if got.SizeGB != 500 {
		t.Errorf("expected 500, got %d", got.SizeGB)
	}
}

func TestAddAccessRule(t *testing.T) {
	svc := setup(t)
	fs, _ := svc.Create(1, "acl-share", "nfs", 50)
	rule, err := svc.AddAccessRule(fs.ID, "10.0.0.0/24", "ro")
	if err != nil {
		t.Fatal(err)
	}
	if rule.AccessLevel != "ro" {
		t.Errorf("expected ro, got %q", rule.AccessLevel)
	}
}

func TestDeleteCascade(t *testing.T) {
	svc := setup(t)
	fs, _ := svc.Create(1, "del-share", "", 50)
	svc.AddAccessRule(fs.ID, "10.0.0.0/24", "")
	svc.Delete(fs.ID)
	_, err := svc.Get(fs.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
