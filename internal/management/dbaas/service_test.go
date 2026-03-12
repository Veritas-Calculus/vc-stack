package dbaas

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

func TestCreateDBInstance(t *testing.T) {
	svc := setup(t)
	inst, err := svc.Create(1, &CreateInstanceRequest{Name: "mydb", Engine: "postgresql", StorageGB: 50})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Engine != "postgresql" {
		t.Errorf("got %q", inst.Engine)
	}
	if inst.Port != 5432 {
		t.Errorf("expected 5432, got %d", inst.Port)
	}
	if inst.StorageGB != 50 {
		t.Errorf("expected 50, got %d", inst.StorageGB)
	}
	if inst.Status != "provisioning" {
		t.Errorf("expected provisioning, got %q", inst.Status)
	}
}

func TestMySQLPort(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateInstanceRequest{Name: "mysqldb", Engine: "mysql"})
	if inst.Port != 3306 {
		t.Errorf("expected 3306, got %d", inst.Port)
	}
	if inst.EngineVersion != "8.0" {
		t.Errorf("expected 8.0, got %q", inst.EngineVersion)
	}
}

func TestListDBInstances(t *testing.T) {
	svc := setup(t)
	svc.Create(1, &CreateInstanceRequest{Name: "db1", Engine: "postgresql"})
	svc.Create(1, &CreateInstanceRequest{Name: "db2", Engine: "mysql"})
	instances, _ := svc.List(0)
	if len(instances) != 2 {
		t.Errorf("expected 2, got %d", len(instances))
	}
}

func TestAddReplica(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateInstanceRequest{Name: "primary", Engine: "postgresql"})
	r, err := svc.AddReplica(inst.ID, "read-replica-1")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "provisioning" {
		t.Errorf("expected provisioning, got %q", r.Status)
	}
}

func TestCreateBackup(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateInstanceRequest{Name: "backup-db", Engine: "postgresql"})
	b, err := svc.CreateBackup(inst.ID, "manual-2026-03-12")
	if err != nil {
		t.Fatal(err)
	}
	if b.Type != "manual" {
		t.Errorf("expected manual, got %q", b.Type)
	}
}

func TestListBackups(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateInstanceRequest{Name: "bu-db", Engine: "postgresql"})
	svc.CreateBackup(inst.ID, "bu1")
	svc.CreateBackup(inst.ID, "bu2")
	backups, _ := svc.ListBackups(inst.ID)
	if len(backups) != 2 {
		t.Errorf("expected 2, got %d", len(backups))
	}
}

func TestResize(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateInstanceRequest{Name: "resize-db", Engine: "postgresql", StorageGB: 20})
	svc.Resize(inst.ID, 100, 0)
	got, _ := svc.Get(inst.ID)
	if got.StorageGB != 100 {
		t.Errorf("expected 100, got %d", got.StorageGB)
	}
}
