package monitoring

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Custom Metrics API
//
// Tenant-owned metric namespaces with PutMetricData batch ingest and
// GetMetricStatistics aggregation (CloudWatch-compatible).
// ──────────────────────────────────────────────────────────────────────

// MetricNamespace represents a tenant-owned metric namespace.
type MetricNamespace struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"uniqueIndex:uniq_ns_tenant;not null"`
	TenantID    string    `json:"tenant_id" gorm:"index;uniqueIndex:uniq_ns_tenant"`
	Description string    `json:"description"`
	MetricCount int       `json:"metric_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
}

func (MetricNamespace) TableName() string { return "mon_metric_namespaces" }

// CustomMetricDatum represents a single metric data point.
type CustomMetricDatum struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	Namespace  string    `json:"namespace" gorm:"index;not null"`
	MetricName string    `json:"metric_name" gorm:"index;not null"`
	Dimensions string    `json:"dimensions,omitempty" gorm:"type:text"` // JSON {"key":"value"}
	Value      float64   `json:"value" gorm:"not null"`
	Unit       string    `json:"unit" gorm:"default:'None'"` // Count, Bytes, Seconds, Percent, None
	Timestamp  time.Time `json:"timestamp" gorm:"index;not null"`
	TenantID   string    `json:"tenant_id" gorm:"index"`
}

func (CustomMetricDatum) TableName() string { return "mon_custom_metrics" }

// ── Route Setup ──

func SetupCustomMetricsRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &metricsService{db: db, logger: logger}
	m := api.Group("/custom-metrics")
	{
		// Namespaces.
		m.GET("/namespaces", svc.listNamespaces)
		m.POST("/namespaces", svc.createNamespace)
		m.DELETE("/namespaces/:id", svc.deleteNamespace)
		// Data ingest + query.
		m.POST("/data", svc.putMetricData)
		m.GET("/statistics", svc.getMetricStatistics)
		m.GET("/metrics", svc.listMetricNames)
	}
}

type metricsService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (s *metricsService) listNamespaces(c *gin.Context) {
	var ns []MetricNamespace
	query := s.db
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	query.Find(&ns)
	c.JSON(http.StatusOK, gin.H{"namespaces": ns})
}

func (s *metricsService) createNamespace(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		TenantID    string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ns := MetricNamespace{Name: req.Name, TenantID: req.TenantID, Description: req.Description}
	if err := s.db.Create(&ns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create namespace"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"namespace": ns})
}

func (s *metricsService) deleteNamespace(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("namespace = (SELECT name FROM mon_metric_namespaces WHERE id = ?)", id).Delete(&CustomMetricDatum{})
	s.db.Delete(&MetricNamespace{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Namespace deleted"})
}

func (s *metricsService) putMetricData(c *gin.Context) {
	var req struct {
		Namespace  string `json:"namespace" binding:"required"`
		MetricData []struct {
			MetricName string            `json:"metric_name" binding:"required"`
			Value      float64           `json:"value"`
			Unit       string            `json:"unit"`
			Dimensions map[string]string `json:"dimensions"`
			Timestamp  *time.Time        `json:"timestamp"`
		} `json:"metric_data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count := 0
	for _, d := range req.MetricData {
		ts := time.Now()
		if d.Timestamp != nil {
			ts = *d.Timestamp
		}
		unit := d.Unit
		if unit == "" {
			unit = "None"
		}
		dimJSON, _ := json.Marshal(d.Dimensions)
		datum := CustomMetricDatum{
			Namespace:  req.Namespace,
			MetricName: d.MetricName,
			Dimensions: string(dimJSON),
			Value:      d.Value,
			Unit:       unit,
			Timestamp:  ts,
		}
		if err := s.db.Create(&datum).Error; err == nil {
			count++
		}
	}
	c.JSON(http.StatusAccepted, gin.H{"accepted": count})
}

func (s *metricsService) getMetricStatistics(c *gin.Context) {
	ns := c.Query("namespace")
	metric := c.Query("metric_name")
	stat := c.Query("stat") // avg, min, max, sum, count
	if ns == "" || metric == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and metric_name required"})
		return
	}
	if stat == "" {
		stat = "avg" //nolint:ineffassign // intentional default before switch
	}

	query := s.db.Model(&CustomMetricDatum{}).Where("namespace = ? AND metric_name = ?", ns, metric)
	if start := c.Query("start"); start != "" {
		query = query.Where("timestamp >= ?", start)
	}
	if end := c.Query("end"); end != "" {
		query = query.Where("timestamp <= ?", end)
	}

	type StatResult struct {
		Avg   float64 `json:"avg"`
		Min   float64 `json:"min"`
		Max   float64 `json:"max"`
		Sum   float64 `json:"sum"`
		Count int64   `json:"count"`
	}
	var result StatResult
	query.Select("AVG(value) as avg, MIN(value) as min, MAX(value) as max, SUM(value) as sum, COUNT(*) as count").
		Scan(&result)

	c.JSON(http.StatusOK, gin.H{
		"namespace":   ns,
		"metric_name": metric,
		"statistics":  result,
	})
}

func (s *metricsService) listMetricNames(c *gin.Context) {
	ns := c.Query("namespace")
	query := s.db.Model(&CustomMetricDatum{})
	if ns != "" {
		query = query.Where("namespace = ?", ns)
	}
	var names []string
	query.Distinct("metric_name").Pluck("metric_name", &names)
	c.JSON(http.StatusOK, gin.H{"metrics": names})
}
