// Package budget provides project-level budget management and threshold alerts.
//
// It tracks budget limits per project and generates alerts when usage
// reaches configured thresholds (e.g. 50%, 80%, 100%).
package budget

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// Budget represents a project-level spending budget.
type Budget struct {
	ID           uint              `json:"id" gorm:"primarykey"`
	Name         string            `json:"name" gorm:"not null"`
	ProjectID    uint              `json:"project_id" gorm:"index;not null"`
	LimitAmount  float64           `json:"limit_amount"` // monthly budget in base currency
	Currency     string            `json:"currency" gorm:"default:'USD'"`
	Period       string            `json:"period" gorm:"default:'monthly'"` // monthly, quarterly
	CurrentSpend float64           `json:"current_spend" gorm:"default:0"`
	Thresholds   []BudgetThreshold `json:"thresholds,omitempty" gorm:"foreignKey:BudgetID"`
	Alerts       []BudgetAlert     `json:"alerts,omitempty" gorm:"foreignKey:BudgetID"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// BudgetThreshold defines an alert trigger point.
type BudgetThreshold struct {
	ID        uint    `json:"id" gorm:"primarykey"`
	BudgetID  uint    `json:"budget_id" gorm:"index;not null"`
	Percent   float64 `json:"percent"`                        // e.g. 80.0 for 80%
	Channel   string  `json:"channel" gorm:"default:'email'"` // email, webhook, slack
	Triggered bool    `json:"triggered" gorm:"default:false"`
}

// BudgetAlert records triggered budget notifications.
type BudgetAlert struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	BudgetID  uint      `json:"budget_id" gorm:"index"`
	Percent   float64   `json:"percent"`
	Spend     float64   `json:"spend"`
	Limit     float64   `json:"limit"`
	Channel   string    `json:"channel"`
	CreatedAt time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&Budget{}, &BudgetThreshold{}, &BudgetAlert{}); err != nil {
		return nil, fmt.Errorf("budget auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Budget CRUD ──────────────────────────────────────────────

type CreateBudgetRequest struct {
	Name        string    `json:"name" binding:"required"`
	ProjectID   uint      `json:"project_id" binding:"required"`
	LimitAmount float64   `json:"limit_amount" binding:"required"`
	Currency    string    `json:"currency"`
	Period      string    `json:"period"`
	Thresholds  []float64 `json:"thresholds"` // e.g. [50, 80, 100]
}

func (s *Service) Create(req *CreateBudgetRequest) (*Budget, error) {
	b := &Budget{
		Name: req.Name, ProjectID: req.ProjectID,
		LimitAmount: req.LimitAmount,
		Currency:    defaultS(req.Currency, "USD"),
		Period:      defaultS(req.Period, "monthly"),
	}

	return b, s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(b).Error; err != nil {
			return err
		}
		for _, pct := range req.Thresholds {
			tx.Create(&BudgetThreshold{BudgetID: b.ID, Percent: pct})
		}
		return nil
	})
}

func (s *Service) List(projectID uint) ([]Budget, error) {
	var budgets []Budget
	q := s.db.Preload("Thresholds").Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return budgets, q.Find(&budgets).Error
}

func (s *Service) Get(id uint) (*Budget, error) {
	var b Budget
	return &b, s.db.Preload("Thresholds").Preload("Alerts", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC").Limit(10)
	}).First(&b, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("budget_id = ?", id).Delete(&BudgetAlert{})
		tx.Where("budget_id = ?", id).Delete(&BudgetThreshold{})
		return tx.Delete(&Budget{}, id).Error
	})
}

// UpdateSpend updates the current spend and evaluates thresholds.
func (s *Service) UpdateSpend(id uint, spend float64) ([]BudgetAlert, error) {
	var b Budget
	if err := s.db.Preload("Thresholds").First(&b, id).Error; err != nil {
		return nil, err
	}
	s.db.Model(&Budget{}).Where("id = ?", id).Update("current_spend", spend)

	var alerts []BudgetAlert
	pct := (spend / b.LimitAmount) * 100
	for _, th := range b.Thresholds {
		if pct >= th.Percent && !th.Triggered {
			alert := BudgetAlert{BudgetID: id, Percent: th.Percent, Spend: spend, Limit: b.LimitAmount, Channel: th.Channel}
			s.db.Create(&alert)
			s.db.Model(&BudgetThreshold{}).Where("id = ?", th.ID).Update("triggered", true)
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/budgets")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		api.POST("/:id/spend", s.handleUpdateSpend)
	}
}

func (s *Service) handleList(c *gin.Context) {
	budgets, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := s.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"budget": b})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	b, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"budget": b})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleUpdateSpend(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Spend float64 `json:"spend"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	alerts, err := s.UpdateSpend(uint(id), req.Spend)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"triggered_alerts": len(alerts), "alerts": alerts})
}

func defaultS(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
