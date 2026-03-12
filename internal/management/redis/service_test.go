package redis

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

func TestCreateSentinel(t *testing.T) {
	svc := setup(t)
	inst, err := svc.Create(1, &CreateRequest{Name: "cache-prod", Mode: "sentinel", MemoryMB: 2048})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Mode != "sentinel" {
		t.Errorf("expected sentinel, got %q", inst.Mode)
	}
	if inst.MemoryMB != 2048 {
		t.Errorf("expected 2048, got %d", inst.MemoryMB)
	}
	if inst.Shards != 0 {
		t.Errorf("sentinel should have 0 shards, got %d", inst.Shards)
	}
}

func TestCreateCluster(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateRequest{Name: "cache-cluster", Mode: "cluster", Shards: 6})
	if inst.Mode != "cluster" {
		t.Errorf("expected cluster, got %q", inst.Mode)
	}
	if inst.Shards != 6 {
		t.Errorf("expected 6 shards, got %d", inst.Shards)
	}
}

func TestClusterMinShards(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateRequest{Name: "min-cluster", Mode: "cluster", Shards: 1})
	if inst.Shards < 3 {
		t.Errorf("expected min 3 shards, got %d", inst.Shards)
	}
}

func TestScale(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateRequest{Name: "scale-cache"})
	svc.Scale(inst.ID, 4096, 3)
	got, _ := svc.Get(inst.ID)
	if got.MemoryMB != 4096 {
		t.Errorf("expected 4096, got %d", got.MemoryMB)
	}
}

func TestSnapshot(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateRequest{Name: "snap-cache"})
	snap, err := svc.CreateSnapshot(inst.ID, "daily-backup")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Status != "creating" {
		t.Errorf("expected creating, got %q", snap.Status)
	}
	snaps, _ := svc.ListSnapshots(inst.ID)
	if len(snaps) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snaps))
	}
}

func TestDeleteCascade(t *testing.T) {
	svc := setup(t)
	inst, _ := svc.Create(1, &CreateRequest{Name: "del-cache"})
	svc.CreateSnapshot(inst.ID, "snap1")
	svc.Delete(inst.ID)
	_, err := svc.Get(inst.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
