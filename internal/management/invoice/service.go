// Package invoice provides automated monthly invoice generation and
// billing history for projects. Integrates with the existing Usage/Tariff
// system to produce detailed line-item invoices.
package invoice

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

// Invoice represents a billing invoice for a project.
type Invoice struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	ProjectID   uint       `json:"project_id" gorm:"index;not null"`
	Number      string     `json:"number" gorm:"uniqueIndex;not null"` // INV-2026-03-001
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	Subtotal    float64    `json:"subtotal" gorm:"default:0"`
	Tax         float64    `json:"tax" gorm:"default:0"`
	Total       float64    `json:"total" gorm:"default:0"`
	Currency    string     `json:"currency" gorm:"default:'USD'"`
	Status      string     `json:"status" gorm:"default:'draft'"` // draft, issued, paid, void
	LineItems   []LineItem `json:"line_items,omitempty" gorm:"foreignKey:InvoiceID"`
	IssuedAt    *time.Time `json:"issued_at,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// LineItem represents a single charge within an invoice.
type LineItem struct {
	ID           uint    `json:"id" gorm:"primarykey"`
	InvoiceID    uint    `json:"invoice_id" gorm:"index;not null"`
	ResourceType string  `json:"resource_type"` // instance, volume, floating_ip, redis, tidb, etc.
	ResourceID   string  `json:"resource_id"`
	Description  string  `json:"description"`
	Quantity     float64 `json:"quantity"` // hours, GB, requests
	UnitPrice    float64 `json:"unit_price"`
	Amount       float64 `json:"amount"` // quantity * unit_price
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
	if err := cfg.DB.AutoMigrate(&Invoice{}, &LineItem{}); err != nil {
		return nil, fmt.Errorf("invoice auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Invoice Operations ───────────────────────────────────────

func (s *Service) Generate(projectID uint, periodStart, periodEnd time.Time) (*Invoice, error) {
	invNum := fmt.Sprintf("INV-%s-%03d", periodEnd.Format("2006-01"), projectID)
	inv := &Invoice{
		ProjectID: projectID, Number: invNum,
		PeriodStart: periodStart, PeriodEnd: periodEnd,
		Currency: "USD", Status: "draft",
	}
	return inv, s.db.Create(inv).Error
}

func (s *Service) AddLineItem(invoiceID uint, resourceType, resourceID, description string, qty, unitPrice float64) (*LineItem, error) {
	li := &LineItem{
		InvoiceID: invoiceID, ResourceType: resourceType,
		ResourceID: resourceID, Description: description,
		Quantity: qty, UnitPrice: unitPrice, Amount: qty * unitPrice,
	}
	if err := s.db.Create(li).Error; err != nil {
		return nil, err
	}

	// Recalculate totals.
	s.recalculate(invoiceID)
	return li, nil
}

func (s *Service) recalculate(invoiceID uint) {
	var total float64
	s.db.Model(&LineItem{}).Where("invoice_id = ?", invoiceID).Select("COALESCE(SUM(amount),0)").Scan(&total)
	s.db.Model(&Invoice{}).Where("id = ?", invoiceID).Updates(map[string]interface{}{
		"subtotal": total, "total": total,
	})
}

func (s *Service) Issue(id uint) error {
	now := time.Now()
	return s.db.Model(&Invoice{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "issued", "issued_at": now,
	}).Error
}

func (s *Service) MarkPaid(id uint) error {
	now := time.Now()
	return s.db.Model(&Invoice{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "paid", "paid_at": now,
	}).Error
}

func (s *Service) List(projectID uint) ([]Invoice, error) {
	var invoices []Invoice
	q := s.db.Preload("LineItems").Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return invoices, q.Find(&invoices).Error
}

func (s *Service) Get(id uint) (*Invoice, error) {
	var inv Invoice
	return &inv, s.db.Preload("LineItems").First(&inv, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/invoices")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("/generate", s.handleGenerate)
		api.GET("/:id", s.handleGet)
		api.POST("/:id/items", s.handleAddItem)
		api.POST("/:id/issue", s.handleIssue)
		api.POST("/:id/pay", s.handlePay)
	}
}

func (s *Service) handleList(c *gin.Context) {
	is, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoices": is})
}

func (s *Service) handleGenerate(c *gin.Context) {
	var req struct {
		ProjectID   uint   `json:"project_id" binding:"required"`
		PeriodStart string `json:"period_start" binding:"required"`
		PeriodEnd   string `json:"period_end" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	start, _ := time.Parse("2006-01-02", req.PeriodStart)
	end, _ := time.Parse("2006-01-02", req.PeriodEnd)
	inv, err := s.Generate(req.ProjectID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"invoice": inv})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	inv, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoice": inv})
}

func (s *Service) handleAddItem(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		ResourceType string  `json:"resource_type"`
		ResourceID   string  `json:"resource_id"`
		Description  string  `json:"description"`
		Quantity     float64 `json:"quantity"`
		UnitPrice    float64 `json:"unit_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	li, err := s.AddLineItem(uint(id), req.ResourceType, req.ResourceID, req.Description, req.Quantity, req.UnitPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"line_item": li})
}

func (s *Service) handleIssue(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.Issue(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "issued"})
}

func (s *Service) handlePay(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.MarkPaid(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "paid"})
}
