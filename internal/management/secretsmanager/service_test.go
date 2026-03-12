package secretsmanager

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

func TestCreateSecret(t *testing.T) {
	svc := setup(t)
	sec, err := svc.Create(1, "db-password", "primary DB password", "s3cret", 0)
	if err != nil {
		t.Fatal(err)
	}
	if sec.VersionID != 1 {
		t.Errorf("expected v1, got %d", sec.VersionID)
	}
}

func TestGetValue(t *testing.T) {
	svc := setup(t)
	sec, _ := svc.Create(1, "val-test", "", "myvalue", 0)
	val, ver, err := svc.GetValue(sec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if val != "myvalue" {
		t.Errorf("expected myvalue, got %q", val)
	}
	if ver != 1 {
		t.Errorf("expected v1, got %d", ver)
	}
}

func TestPutValue(t *testing.T) {
	svc := setup(t)
	sec, _ := svc.Create(1, "put-test", "", "old", 0)
	svc.PutValue(sec.ID, "new")
	val, ver, _ := svc.GetValue(sec.ID)
	if val != "new" {
		t.Errorf("expected new, got %q", val)
	}
	if ver != 2 {
		t.Errorf("expected v2, got %d", ver)
	}
}

func TestRotate(t *testing.T) {
	svc := setup(t)
	sec, _ := svc.Create(1, "rotate-test", "", "initial", 0)
	v, err := svc.Rotate(sec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != 2 {
		t.Errorf("expected v2, got %d", v.Version)
	}
	val, _, _ := svc.GetValue(sec.ID)
	if val == "initial" {
		t.Error("value should have changed after rotate")
	}
}

func TestDeleteSecret(t *testing.T) {
	svc := setup(t)
	sec, _ := svc.Create(1, "del-test", "", "val", 0)
	svc.Delete(sec.ID)
	_, err := svc.Get(sec.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
