package loadbalancer

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

func TestCreateLoadBalancer(t *testing.T) {
	svc := setupTest(t)
	lb, err := svc.Create(1, &CreateLBRequest{Name: "web-lb", Algorithm: "round_robin", NetworkID: 1})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lb.Name != "web-lb" {
		t.Errorf("expected 'web-lb', got %q", lb.Name)
	}
	if lb.Status != "active" {
		t.Errorf("expected 'active', got %q", lb.Status)
	}
}

func TestListLoadBalancers(t *testing.T) {
	svc := setupTest(t)
	svc.Create(1, &CreateLBRequest{Name: "lb-1"})
	svc.Create(1, &CreateLBRequest{Name: "lb-2"})

	lbs, err := svc.List(0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(lbs) != 2 {
		t.Errorf("expected 2, got %d", len(lbs))
	}
}

func TestDeleteLoadBalancer(t *testing.T) {
	svc := setupTest(t)
	lb, _ := svc.Create(1, &CreateLBRequest{Name: "del-lb"})

	if err := svc.Delete(lb.ID); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	_, err := svc.Get(lb.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestAddListener(t *testing.T) {
	svc := setupTest(t)
	lb, _ := svc.Create(1, &CreateLBRequest{Name: "listener-lb"})

	l, err := svc.AddListener(lb.ID, &CreateListenerRequest{Name: "http", Port: 80})
	if err != nil {
		t.Fatalf("AddListener error: %v", err)
	}
	if l.Protocol != "HTTP" {
		t.Errorf("expected HTTP, got %q", l.Protocol)
	}
}

func TestAddPoolAndMember(t *testing.T) {
	svc := setupTest(t)
	lb, _ := svc.Create(1, &CreateLBRequest{Name: "pool-lb"})

	pool, err := svc.AddPool(lb.ID, &CreatePoolRequest{
		Name:        "backend",
		HealthCheck: HealthCheck{Path: "/health", ExpectedCodes: "200"},
	})
	if err != nil {
		t.Fatalf("AddPool error: %v", err)
	}
	if pool.HealthCheck.IntervalSeconds != 30 {
		t.Errorf("expected default interval 30, got %d", pool.HealthCheck.IntervalSeconds)
	}

	member, err := svc.AddMember(pool.ID, &AddMemberRequest{Address: "10.0.1.5", Port: 8080, Weight: 3})
	if err != nil {
		t.Fatalf("AddMember error: %v", err)
	}
	if member.Weight != 3 {
		t.Errorf("expected weight 3, got %d", member.Weight)
	}
}

func TestListPools(t *testing.T) {
	svc := setupTest(t)
	lb, _ := svc.Create(1, &CreateLBRequest{Name: "list-pools-lb"})
	svc.AddPool(lb.ID, &CreatePoolRequest{Name: "pool-1"})
	svc.AddPool(lb.ID, &CreatePoolRequest{Name: "pool-2"})

	pools, err := svc.ListPools(lb.ID)
	if err != nil {
		t.Fatalf("ListPools error: %v", err)
	}
	if len(pools) != 2 {
		t.Errorf("expected 2 pools, got %d", len(pools))
	}
}
