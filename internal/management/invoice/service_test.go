package invoice

import (
	"testing"
	"time"

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

func TestGenerate(t *testing.T) {
	svc := setup(t)
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	inv, err := svc.Generate(1, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Number != "INV-2026-03-001" {
		t.Errorf("expected INV-2026-03-001, got %q", inv.Number)
	}
	if inv.Status != "draft" {
		t.Errorf("expected draft, got %q", inv.Status)
	}
}

func TestAddLineItem(t *testing.T) {
	svc := setup(t)
	inv, _ := svc.Generate(1, time.Now(), time.Now())
	li, err := svc.AddLineItem(inv.ID, "instance", "i-123", "m5.large x 720h", 720, 0.096)
	if err != nil {
		t.Fatal(err)
	}
	expected := 720 * 0.096
	if li.Amount != expected {
		t.Errorf("expected %.2f, got %.2f", expected, li.Amount)
	}
}

func TestRecalculateTotals(t *testing.T) {
	svc := setup(t)
	inv, _ := svc.Generate(1, time.Now(), time.Now())
	svc.AddLineItem(inv.ID, "instance", "i-1", "compute", 100, 0.10) // 10.00
	svc.AddLineItem(inv.ID, "volume", "v-1", "storage", 50, 0.05)    // 2.50
	got, _ := svc.Get(inv.ID)
	if got.Total != 12.5 {
		t.Errorf("expected 12.50, got %.2f", got.Total)
	}
	if len(got.LineItems) != 2 {
		t.Errorf("expected 2 items, got %d", len(got.LineItems))
	}
}

func TestIssueAndPay(t *testing.T) {
	svc := setup(t)
	inv, _ := svc.Generate(1, time.Now(), time.Now())
	svc.Issue(inv.ID)
	got, _ := svc.Get(inv.ID)
	if got.Status != "issued" {
		t.Errorf("expected issued, got %q", got.Status)
	}

	svc.MarkPaid(inv.ID)
	got, _ = svc.Get(inv.ID)
	if got.Status != "paid" {
		t.Errorf("expected paid, got %q", got.Status)
	}
}
