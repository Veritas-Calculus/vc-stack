package elasticsearch

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
	c, err := svc.Create(1, &CreateClusterRequest{Name: "es-logs", KibanaEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if c.DataNodes < 3 {
		t.Errorf("expected min 3 data, got %d", c.DataNodes)
	}
	if !c.KibanaEnabled {
		t.Error("expected kibana enabled")
	}
	if c.Endpoint == "" {
		t.Error("expected endpoint set")
	}
}

func TestCreateIndex(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "es-idx"})
	idx, err := svc.CreateIndex(c.ID, "logs-2026-03", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Shards != 5 {
		t.Errorf("expected 5, got %d", idx.Shards)
	}
}

func TestListIndices(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "es-list"})
	svc.CreateIndex(c.ID, "idx-a", 3, 1)
	svc.CreateIndex(c.ID, "idx-b", 3, 1)
	indices, _ := svc.ListIndices(c.ID)
	if len(indices) != 2 {
		t.Errorf("expected 2, got %d", len(indices))
	}
}

func TestSnapshot(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "es-snap"})
	snap, err := svc.CreateSnapshot(c.ID, "daily")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Indices != "*" {
		t.Errorf("expected *, got %q", snap.Indices)
	}
}

func TestDeleteCascade(t *testing.T) {
	svc := setup(t)
	c, _ := svc.Create(1, &CreateClusterRequest{Name: "es-del"})
	svc.CreateIndex(c.ID, "idx", 1, 0)
	svc.CreateSnapshot(c.ID, "snap")
	svc.Delete(c.ID)
	_, err := svc.Get(c.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
