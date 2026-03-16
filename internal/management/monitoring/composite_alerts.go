package monitoring

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Composite Alerts with Anomaly Detection
// ──────────────────────────────────────────────────────────────────────

// CompositeAlertRule defines a multi-condition alert rule.
type CompositeAlertRule struct {
	ID                   uint             `json:"id" gorm:"primarykey"`
	Name                 string           `json:"name" gorm:"not null"`
	Description          string           `json:"description"`
	Expression           string           `json:"expression" gorm:"not null"`             // Boolean: "AND" or "OR"
	Severity             string           `json:"severity" gorm:"default:'warning'"`      // info, warning, critical
	NotificationChannels string           `json:"notification_channels" gorm:"type:text"` // JSON: ["email:x", "slack:#y", "webhook:url"]
	EvaluationInterval   int              `json:"evaluation_interval" gorm:"default:60"`  // seconds
	Enabled              bool             `json:"enabled" gorm:"default:true"`
	Status               string           `json:"status" gorm:"default:'ok'"` // ok, firing, pending
	LastEvaluated        *time.Time       `json:"last_evaluated"`
	TenantID             string           `json:"tenant_id" gorm:"index"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
	Conditions           []AlertCondition `json:"conditions,omitempty" gorm:"foreignKey:AlertRuleID"`
}

func (CompositeAlertRule) TableName() string { return "mon_composite_alerts" }

// AlertCondition is a single metric condition within a composite alert.
type AlertCondition struct {
	ID              uint    `json:"id" gorm:"primarykey"`
	AlertRuleID     uint    `json:"alert_rule_id" gorm:"index;not null"`
	MetricNamespace string  `json:"metric_namespace"`
	MetricName      string  `json:"metric_name" gorm:"not null"`
	Operator        string  `json:"operator" gorm:"not null"` // gt, lt, gte, lte, eq, ne
	Threshold       float64 `json:"threshold"`
	Aggregation     string  `json:"aggregation" gorm:"default:'avg'"` // avg, min, max, sum, count, p95, p99
	Period          int     `json:"period" gorm:"default:300"`        // evaluation window in seconds
	// Anomaly detection.
	AnomalyMode string  `json:"anomaly_mode,omitempty"` // stddev, trend
	AnomalyBand float64 `json:"anomaly_band,omitempty"` // Number of std deviations
}

func (AlertCondition) TableName() string { return "mon_alert_conditions" }

// AlertHistory records alert state transitions.
type AlertHistory struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	AlertRuleID uint       `json:"alert_rule_id" gorm:"index;not null"`
	Status      string     `json:"status"`                  // firing, resolved
	Values      string     `json:"values" gorm:"type:text"` // JSON: triggering metric values
	Message     string     `json:"message"`
	FiredAt     time.Time  `json:"fired_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

func (AlertHistory) TableName() string { return "mon_alert_history" }

// ── Route Setup ──

func SetupCompositeAlertRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &alertService{db: db, logger: logger}
	a := api.Group("/alerts")
	{
		a.GET("", svc.listAlerts)
		a.POST("", svc.createAlert)
		a.GET("/:id", svc.getAlert)
		a.PUT("/:id", svc.updateAlert)
		a.DELETE("/:id", svc.deleteAlert)
		a.POST("/:id/evaluate", svc.evaluateAlert)
		a.GET("/:id/history", svc.alertHistory)
	}
}

type alertService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (s *alertService) listAlerts(c *gin.Context) {
	var alerts []CompositeAlertRule
	query := s.db.Preload("Conditions")
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&alerts)
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (s *alertService) createAlert(c *gin.Context) {
	var req struct {
		Name                 string           `json:"name" binding:"required"`
		Description          string           `json:"description"`
		Expression           string           `json:"expression"`
		Severity             string           `json:"severity"`
		NotificationChannels []string         `json:"notification_channels"`
		EvaluationInterval   int              `json:"evaluation_interval"`
		TenantID             string           `json:"tenant_id"`
		Conditions           []AlertCondition `json:"conditions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	expr := req.Expression
	if expr == "" {
		expr = "AND"
	}
	sev := req.Severity
	if sev == "" {
		sev = "warning"
	}
	interval := req.EvaluationInterval
	if interval == 0 {
		interval = 60
	}
	channels, _ := json.Marshal(req.NotificationChannels)

	rule := CompositeAlertRule{
		Name: req.Name, Description: req.Description,
		Expression: expr, Severity: sev,
		NotificationChannels: string(channels),
		EvaluationInterval:   interval,
		Enabled:              true, Status: "ok", TenantID: req.TenantID,
		Conditions: req.Conditions,
	}
	s.db.Create(&rule)
	c.JSON(http.StatusCreated, gin.H{"alert": rule})
}

func (s *alertService) getAlert(c *gin.Context) {
	id := c.Param("id")
	var rule CompositeAlertRule
	if err := s.db.Preload("Conditions").First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alert": rule})
}

func (s *alertService) updateAlert(c *gin.Context) {
	id := c.Param("id")
	var rule CompositeAlertRule
	if err := s.db.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	s.db.Model(&rule).Updates(req)
	s.db.Preload("Conditions").First(&rule, id)
	c.JSON(http.StatusOK, gin.H{"alert": rule})
}

func (s *alertService) deleteAlert(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("alert_rule_id = ?", id).Delete(&AlertCondition{})
	s.db.Where("alert_rule_id = ?", id).Delete(&AlertHistory{})
	s.db.Delete(&CompositeAlertRule{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}

func (s *alertService) evaluateAlert(c *gin.Context) {
	id := c.Param("id")
	var rule CompositeAlertRule
	if err := s.db.Preload("Conditions").First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	results := make(map[string]bool)
	values := make(map[string]float64)

	for _, cond := range rule.Conditions {
		val := s.evaluateCondition(&cond)
		values[cond.MetricName] = val

		var triggered bool
		if cond.AnomalyMode != "" {
			triggered = s.evaluateAnomaly(&cond, val)
		} else {
			triggered = evaluateThreshold(val, cond.Operator, cond.Threshold)
		}
		results[cond.MetricName] = triggered
	}

	// Apply expression (AND/OR).
	firing := false
	if rule.Expression == "OR" {
		for _, v := range results {
			if v {
				firing = true
				break
			}
		}
	} else { // AND
		firing = true
		for _, v := range results {
			if !v {
				firing = false
				break
			}
		}
	}

	now := time.Now()
	oldStatus := rule.Status
	if firing {
		rule.Status = "firing"
	} else {
		rule.Status = "ok"
	}
	rule.LastEvaluated = &now
	s.db.Save(&rule)

	// Record history on state change.
	if oldStatus != rule.Status {
		valJSON, _ := json.Marshal(values)
		hist := AlertHistory{
			AlertRuleID: rule.ID, Status: rule.Status,
			Values: string(valJSON), FiredAt: now,
		}
		if rule.Status == "ok" {
			hist.ResolvedAt = &now
		}
		s.db.Create(&hist)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": rule.Status, "firing": firing,
		"conditions": results, "values": values,
	})
}

func (s *alertService) alertHistory(c *gin.Context) {
	id := c.Param("id")
	var history []AlertHistory
	s.db.Where("alert_rule_id = ?", id).Order("fired_at DESC").Limit(100).Find(&history)
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// evaluateCondition fetches metric value for a condition.
func (s *alertService) evaluateCondition(cond *AlertCondition) float64 {
	var result float64
	agg := cond.Aggregation
	if agg == "" {
		agg = "avg"
	}
	windowStart := time.Now().Add(-time.Duration(cond.Period) * time.Second)
	s.db.Model(&CustomMetricDatum{}).
		Where("metric_name = ? AND timestamp >= ?", cond.MetricName, windowStart).
		Select(agg + "(value)").Scan(&result)
	return result
}

// evaluateAnomaly checks for anomalous values using stddev or trend.
func (s *alertService) evaluateAnomaly(cond *AlertCondition, currentVal float64) bool {
	windowStart := time.Now().Add(-time.Duration(cond.Period*6) * time.Second) // 6x window for baseline
	var avg, stddev float64
	row := s.db.Model(&CustomMetricDatum{}).
		Where("metric_name = ? AND timestamp >= ?", cond.MetricName, windowStart).
		Select("AVG(value), STDDEV(value)").Row()
	if err := row.Scan(&avg, &stddev); err != nil {
		return false
	}

	band := cond.AnomalyBand
	if band == 0 {
		band = 2.0
	}
	deviation := math.Abs(currentVal - avg)
	return deviation > band*stddev
}

func evaluateThreshold(value float64, op string, threshold float64) bool {
	switch op {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	}
	return false
}
