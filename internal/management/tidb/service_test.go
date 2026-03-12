package tidb

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

func TestCreateCluster(t *testing.T) {
	svc := setup(t)
	c, err := svc.Create(1, &CreateClusterRequest{Name: "tidb-prod", TiKVNodes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if c.TiKVNodes != 5 {
		t.Errorf("expected 5 tikv, got %d", c.TiKVNodes)
	}
	if c.PDNodes < 3 {
		t.Errorf("expected min 3 pd, got %d", c.PDNodes)
	}
	if c.Port != 4000 {
		t.Errorf("expected 4000, got %d", c.Port)
	}
}

func TestMinNodes(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "tidb-min", TiKVNodes: 1, PDNodes: 1})
	if c.TiKVNodes < 3 {
		t.Errorf("expected min 3 tikv, got %d", c.TiKVNodes)
	}
	if c.PDNodes < 3 {
		t.Errorf("expected min 3 pd, got %d", c.PDNodes)
	}
}

func TestScaleTiDB(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "tidb-scale"})
	svc.ScaleTiDB(c.ID, 4)
	got, _ := svc.Get(c.ID)
	if got.TiDBNodes != 4 {
		t.Errorf("expected 4, got %d", got.TiDBNodes)
	}
}

func TestAddTiFlash(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "tidb-flash"})
	svc.AddTiFlash(c.ID, 2)
	got, _ := svc.Get(c.ID)
	if got.TiFlashNodes != 2 {
		t.Errorf("expected 2, got %d", got.TiFlashNodes)
	}
}

func TestBackup(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "tidb-backup"})
	b, err := svc.CreateBackup(c.ID, "full")
	if err != nil {
		t.Fatal(err)
	}
	if b.Type != "full" {
		t.Errorf("expected full, got %q", b.Type)
	}
}

func TestDeleteCascade(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "tidb-del"})
	svc.CreateBackup(c.ID, "full")
	svc.Delete(c.ID)
	_, err := svc.Get(c.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
