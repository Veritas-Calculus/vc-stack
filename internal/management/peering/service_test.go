package peering

import (
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTest(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestCreatePeering(t *testing.T) {
	svc := setupTest(t)
	p, err := svc.Create(1, 10, &CreatePeeringRequest{
		Name: "app-to-db", RequesterNetworkID: 1, AccepterNetworkID: 2,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Status != "pending" {
		t.Errorf("expected pending, got %q", p.Status)
	}
}

func TestCannotSelfPeer(t *testing.T) {
	svc := setupTest(t)
	_, err := svc.Create(1, 10, &CreatePeeringRequest{
		Name: "self", RequesterNetworkID: 1, AccepterNetworkID: 1,
	})
	if err == nil {
		t.Error("expected error for self-peering")
	}
}

func TestAcceptPeering(t *testing.T) {
	svc := setupTest(t)
	p, _ := svc.Create(1, 10, &CreatePeeringRequest{
		Name: "accept-test", RequesterNetworkID: 1, AccepterNetworkID: 2,
	})

	accepted, err := svc.Accept(p.ID, 2)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if accepted.Status != "active" {
		t.Errorf("expected active, got %q", accepted.Status)
	}
}

func TestRejectPeering(t *testing.T) {
	svc := setupTest(t)
	p, _ := svc.Create(1, 10, &CreatePeeringRequest{
		Name: "reject-test", RequesterNetworkID: 1, AccepterNetworkID: 2,
	})

	if err := svc.Reject(p.ID); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	got, _ := svc.Get(p.ID)
	if got.Status != "rejected" {
		t.Errorf("expected rejected, got %q", got.Status)
	}
}

func TestDoubleAcceptFails(t *testing.T) {
	svc := setupTest(t)
	p, _ := svc.Create(1, 10, &CreatePeeringRequest{
		Name: "double", RequesterNetworkID: 1, AccepterNetworkID: 2,
	})
	svc.Accept(p.ID, 2)

	_, err := svc.Accept(p.ID, 3)
	if err == nil {
		t.Error("expected error on double accept")
	}
}

func TestListPeerings(t *testing.T) {
	svc := setupTest(t)
	svc.Create(1, 10, &CreatePeeringRequest{Name: "p1", RequesterNetworkID: 1, AccepterNetworkID: 2})
	svc.Create(1, 10, &CreatePeeringRequest{Name: "p2", RequesterNetworkID: 3, AccepterNetworkID: 4})

	peerings, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(peerings) != 2 {
		t.Errorf("expected 2, got %d", len(peerings))
	}
}
