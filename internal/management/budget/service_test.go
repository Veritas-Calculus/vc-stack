package budget

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

func TestCreateBudget(t *testing.T) {
	svc := setup(t)
	b, err := svc.Create(&CreateBudgetRequest{Name: "dev-budget", ProjectID: 1, LimitAmount: 1000, Thresholds: []float64{50, 80, 100}})
	if err != nil {
		t.Fatal(err)
	}
	if b.LimitAmount != 1000 {
		t.Errorf("expected 1000, got %f", b.LimitAmount)
	}
	if b.Currency != "USD" {
		t.Errorf("expected USD, got %q", b.Currency)
	}
}

func TestThresholdCreation(t *testing.T) {
	svc := setup(t)
	b, _ := svc.Create(&CreateBudgetRequest{Name: "th-budget", ProjectID: 1, LimitAmount: 500, Thresholds: []float64{80, 100}})
	got, _ := svc.Get(b.ID)
	if len(got.Thresholds) != 2 {
		t.Errorf("expected 2 thresholds, got %d", len(got.Thresholds))
	}
}

func TestUpdateSpendTriggersAlert(t *testing.T) {
	svc := setup(t)
	b, _ := svc.Create(&CreateBudgetRequest{Name: "alert-budget", ProjectID: 1, LimitAmount: 100, Thresholds: []float64{50, 80}})
	alerts, err := svc.UpdateSpend(b.ID, 85)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts (50%% and 80%%), got %d", len(alerts))
	}
}

func TestNoDoubleAlert(t *testing.T) {
	svc := setup(t)
	b, _ := svc.Create(&CreateBudgetRequest{Name: "double-budget", ProjectID: 1, LimitAmount: 100, Thresholds: []float64{50}})
	svc.UpdateSpend(b.ID, 60)              // triggers 50%
	alerts, _ := svc.UpdateSpend(b.ID, 70) // should not re-trigger
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts on re-trigger, got %d", len(alerts))
	}
}

func TestDeleteBudget(t *testing.T) {
	svc := setup(t)
	b, _ := svc.Create(&CreateBudgetRequest{Name: "del-budget", ProjectID: 1, LimitAmount: 100})
	svc.Delete(b.ID)
	_, err := svc.Get(b.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
