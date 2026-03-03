// Package usage provides resource usage tracking and billing.
package usage

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the usage service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides usage tracking and billing operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// UsageRecord tracks resource consumption over time.
type UsageRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AccountID    uint      `gorm:"index" json:"account_id"`
	ProjectID    uint      `gorm:"index" json:"project_id"`
	DomainID     uint      `gorm:"index" json:"domain_id"`
	ResourceType string    `gorm:"not null;index" json:"resource_type"` // running_vm, allocated_vm, volume, snapshot, network, ip_address, template
	ResourceID   string    `json:"resource_id"`
	Description  string    `json:"description"`
	UsageValue   float64   `gorm:"not null;default:0" json:"usage_value"` // quantity / hours
	Unit         string    `gorm:"not null;default:'hours'" json:"unit"`  // hours, gb, count
	StartDate    time.Time `gorm:"not null;index" json:"start_date"`
	EndDate      time.Time `gorm:"not null" json:"end_date"`
	CreatedAt    time.Time `json:"created_at"`
}

// Tariff represents a billing rate for a resource type.
type Tariff struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	ResourceType string    `gorm:"not null;uniqueIndex:idx_tariff_type_effective" json:"resource_type"`
	Rate         float64   `gorm:"not null;default:0" json:"rate"` // price per unit
	Currency     string    `gorm:"not null;default:'USD'" json:"currency"`
	Unit         string    `gorm:"not null;default:'per_hour'" json:"unit"` // per_hour, per_gb_month, per_count
	EffectiveOn  time.Time `gorm:"not null;uniqueIndex:idx_tariff_type_effective" json:"effective_on"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// QuotaSummary represents a per-account billing summary.
type QuotaSummary struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AccountID uint      `gorm:"not null;uniqueIndex:idx_qs_account_period" json:"account_id"`
	Period    string    `gorm:"not null;uniqueIndex:idx_qs_account_period" json:"period"` // YYYY-MM
	Balance   float64   `gorm:"not null;default:0" json:"balance"`
	Credit    float64   `gorm:"not null;default:0" json:"credit"` // added credits
	Usage     float64   `gorm:"not null;default:0" json:"usage"`  // total charges
	Currency  string    `gorm:"not null;default:'USD'" json:"currency"`
	State     string    `gorm:"not null;default:'enabled'" json:"state"` // enabled, locked, disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewService creates a new usage service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&UsageRecord{}, &Tariff{}, &QuotaSummary{}); err != nil {
		return nil, err
	}
	s.seedDefaultTariffs()
	return s, nil
}

// seedDefaultTariffs creates default billing rates.
func (s *Service) seedDefaultTariffs() {
	var count int64
	s.db.Model(&Tariff{}).Count(&count)
	if count > 0 {
		return
	}
	now := time.Now()
	defaults := []Tariff{
		{Name: "Running VM", ResourceType: "running_vm", Rate: 0.05, Unit: "per_hour", EffectiveOn: now},
		{Name: "Allocated VM", ResourceType: "allocated_vm", Rate: 0.01, Unit: "per_hour", EffectiveOn: now},
		{Name: "Volume Storage", ResourceType: "volume", Rate: 0.10, Unit: "per_gb_month", EffectiveOn: now},
		{Name: "Snapshot Storage", ResourceType: "snapshot", Rate: 0.05, Unit: "per_gb_month", EffectiveOn: now},
		{Name: "Network", ResourceType: "network", Rate: 5.00, Unit: "per_count", EffectiveOn: now},
		{Name: "Public IP", ResourceType: "ip_address", Rate: 3.00, Unit: "per_count", EffectiveOn: now},
		{Name: "Template Storage", ResourceType: "template", Rate: 0.02, Unit: "per_gb_month", EffectiveOn: now},
	}
	for _, t := range defaults {
		t.Currency = "USD"
		if err := s.db.Create(&t).Error; err != nil {
			s.logger.Warn("failed to seed tariff", zap.String("name", t.Name), zap.Error(err))
		}
	}
}

// SetupRoutes registers HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	if s == nil {
		return
	}
	api := router.Group("/api/v1")
	{
		// Usage records
		usage := api.Group("/usage")
		{
			usage.GET("", s.listUsageRecords)
			usage.GET("/summary", s.getUsageSummary)
		}
		// Tariffs
		tariffs := api.Group("/tariffs")
		{
			tariffs.GET("", s.listTariffs)
			tariffs.POST("", s.createTariff)
			tariffs.PUT("/:id", s.updateTariff)
			tariffs.DELETE("/:id", s.deleteTariff)
		}
		// Quota / Billing summary
		billing := api.Group("/billing")
		{
			billing.GET("/summary", s.listQuotaSummaries)
			billing.POST("/credit", s.addCredit)
		}
	}
}

// --- Usage handlers ---

func (s *Service) listUsageRecords(c *gin.Context) {
	var records []UsageRecord
	query := s.db.Order("start_date DESC").Limit(500)
	if accountID := c.Query("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("start_date >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("end_date <= ?", endDate)
	}
	if err := query.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list usage records"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records, "count": len(records)})
}

func (s *Service) getUsageSummary(c *gin.Context) {
	type summary struct {
		ResourceType string  `json:"resource_type"`
		TotalUsage   float64 `json:"total_usage"`
		Unit         string  `json:"unit"`
	}
	var results []summary
	query := s.db.Model(&UsageRecord{}).
		Select("resource_type, SUM(usage_value) as total_usage, unit").
		Group("resource_type, unit")
	if accountID := c.Query("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage summary"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": results})
}

// --- Tariff handlers ---

func (s *Service) listTariffs(c *gin.Context) {
	var tariffs []Tariff
	if err := s.db.Order("resource_type, effective_on DESC").Find(&tariffs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tariffs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tariffs": tariffs})
}

func (s *Service) createTariff(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		ResourceType string  `json:"resource_type" binding:"required"`
		Rate         float64 `json:"rate" binding:"required"`
		Currency     string  `json:"currency"`
		Unit         string  `json:"unit"`
		Description  string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tariff := Tariff{
		Name:         req.Name,
		ResourceType: req.ResourceType,
		Rate:         req.Rate,
		Currency:     req.Currency,
		Unit:         req.Unit,
		Description:  req.Description,
		EffectiveOn:  time.Now(),
	}
	if tariff.Currency == "" {
		tariff.Currency = "USD"
	}
	if tariff.Unit == "" {
		tariff.Unit = "per_hour"
	}
	if err := s.db.Create(&tariff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tariff"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"tariff": tariff})
}

func (s *Service) updateTariff(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var tariff Tariff
	if err := s.db.First(&tariff, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tariff not found"})
		return
	}
	var req struct {
		Rate        *float64 `json:"rate"`
		Description *string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Rate != nil {
		updates["rate"] = *req.Rate
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) > 0 {
		s.db.Model(&tariff).Updates(updates)
	}
	s.db.First(&tariff, id)
	c.JSON(http.StatusOK, gin.H{"tariff": tariff})
}

func (s *Service) deleteTariff(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Tariff{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tariff"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Billing handlers ---

func (s *Service) listQuotaSummaries(c *gin.Context) {
	var summaries []QuotaSummary
	query := s.db.Order("period DESC")
	if accountID := c.Query("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	if err := query.Limit(100).Find(&summaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list billing summaries"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"summaries": summaries})
}

func (s *Service) addCredit(c *gin.Context) {
	var req struct {
		AccountID uint    `json:"account_id" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		Period    string  `json:"period"` // YYYY-MM, defaults to current
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	period := req.Period
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	var qs QuotaSummary
	if err := s.db.Where("account_id = ? AND period = ?", req.AccountID, period).First(&qs).Error; err != nil {
		qs = QuotaSummary{
			AccountID: req.AccountID,
			Period:    period,
			Credit:    req.Amount,
			Balance:   req.Amount,
			Currency:  "USD",
			State:     "enabled",
		}
		if err := s.db.Create(&qs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add credit"})
			return
		}
	} else {
		s.db.Model(&qs).Updates(map[string]interface{}{
			"credit":  gorm.Expr("credit + ?", req.Amount),
			"balance": gorm.Expr("balance + ?", req.Amount),
		})
		s.db.First(&qs, qs.ID)
	}
	c.JSON(http.StatusOK, gin.H{"summary": qs})
}
