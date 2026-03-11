// DRS (Distributed Resource Scheduling) auto-balancer for VC Stack.
// DRS periodically evaluates host utilization across the cluster and
// generates live migration recommendations to balance workloads.
package scheduler

import (
	"context"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- DRS Models ----------

// DRSHostUtilization represents current resource usage for a host.
type DRSHostUtilization struct {
	HostID    string  `json:"host_id"`
	Hostname  string  `json:"hostname"`
	CPUUsage  float64 `json:"cpu_usage"`
	RAMUsage  float64 `json:"ram_usage"`
	DiskUsage float64 `json:"disk_usage"`
	VMCount   int     `json:"vm_count"`
	Score     float64 `json:"score"`
}

// DRSMigrationSuggestion is a suggested VM migration for load balancing.
type DRSMigrationSuggestion struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InstanceID   string    `json:"instance_id" gorm:"index"`
	InstanceName string    `json:"instance_name"`
	SourceHost   string    `json:"source_host"`
	TargetHost   string    `json:"target_host"`
	Reason       string    `json:"reason"`
	Impact       string    `json:"impact" gorm:"default:'medium'"`
	Status       string    `json:"status" gorm:"default:'pending'"`
	CreatedAt    time.Time `json:"created_at"`
}

func (DRSMigrationSuggestion) TableName() string { return "drs_migration_suggestions" }

// DRSSettings controls auto-balancing thresholds.
type DRSSettings struct {
	Enabled            bool          `json:"enabled" mapstructure:"enabled"`
	Interval           time.Duration `json:"interval" mapstructure:"interval"`
	ImbalanceThreshold float64       `json:"imbalance_threshold" mapstructure:"imbalance_threshold"`
	AutoMigrate        bool          `json:"auto_migrate" mapstructure:"auto_migrate"`
	CPUWeight          float64       `json:"cpu_weight" mapstructure:"cpu_weight"`
	RAMWeight          float64       `json:"ram_weight" mapstructure:"ram_weight"`
	DiskWeight         float64       `json:"disk_weight" mapstructure:"disk_weight"`
}

// DRSAnalysisResult is the output of a DRS evaluation cycle.
type DRSAnalysisResult struct {
	Timestamp   time.Time                `json:"timestamp"`
	HostCount   int                      `json:"host_count"`
	Balanced    bool                     `json:"balanced"`
	Imbalance   float64                  `json:"imbalance"`
	Suggestions []DRSMigrationSuggestion `json:"suggestions"`
}

// DRSEngine runs the periodic DRS evaluation loop.
type DRSEngine struct {
	db     *gorm.DB
	logger *zap.Logger
	cfg    DRSSettings
	cancel context.CancelFunc
	mu     sync.RWMutex

	lastAnalysis *DRSAnalysisResult
}

// NewDRSEngine creates and starts the DRS auto-balancer.
func NewDRSEngine(db *gorm.DB, logger *zap.Logger, cfg DRSSettings) (*DRSEngine, error) {
	if db != nil {
		_ = db.AutoMigrate(&DRSMigrationSuggestion{})
	}

	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.ImbalanceThreshold == 0 {
		cfg.ImbalanceThreshold = 0.15
	}
	if cfg.CPUWeight == 0 {
		cfg.CPUWeight = 0.5
	}
	if cfg.RAMWeight == 0 {
		cfg.RAMWeight = 0.4
	}
	if cfg.DiskWeight == 0 {
		cfg.DiskWeight = 0.1
	}

	e := &DRSEngine{
		db:     db,
		logger: logger,
		cfg:    cfg,
	}

	if cfg.Enabled {
		ctx, cancel := context.WithCancel(context.Background())
		e.cancel = cancel
		go e.runLoop(ctx)
		logger.Info("DRS auto-balancer started",
			zap.Duration("interval", cfg.Interval),
			zap.Float64("threshold", cfg.ImbalanceThreshold),
			zap.Bool("auto_migrate", cfg.AutoMigrate))
	} else {
		logger.Info("DRS auto-balancer disabled")
	}

	return e, nil
}

// StopDRS stops the DRS evaluation loop.
func (e *DRSEngine) StopDRS() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *DRSEngine) runLoop(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("DRS loop stopped")
			return
		case <-ticker.C:
			e.evaluate()
		}
	}
}

func (e *DRSEngine) evaluate() {
	hosts := e.collectUtilization()
	if len(hosts) < 2 {
		return
	}

	for i := range hosts {
		hosts[i].Score = hosts[i].CPUUsage*e.cfg.CPUWeight +
			hosts[i].RAMUsage*e.cfg.RAMWeight +
			hosts[i].DiskUsage*e.cfg.DiskWeight
	}

	var sum float64
	for _, h := range hosts {
		sum += h.Score
	}
	mean := sum / float64(len(hosts))

	var variance float64
	for _, h := range hosts {
		variance += (h.Score - mean) * (h.Score - mean)
	}
	stddev := math.Sqrt(variance / float64(len(hosts)))

	balanced := stddev < e.cfg.ImbalanceThreshold

	result := &DRSAnalysisResult{
		Timestamp: time.Now(),
		HostCount: len(hosts),
		Balanced:  balanced,
		Imbalance: stddev,
	}

	if !balanced {
		e.logger.Info("DRS detected imbalance",
			zap.Float64("stddev", stddev),
			zap.Float64("threshold", e.cfg.ImbalanceThreshold))
	}

	e.mu.Lock()
	e.lastAnalysis = result
	e.mu.Unlock()
}

func (e *DRSEngine) collectUtilization() []DRSHostUtilization {
	if e.db == nil {
		return nil
	}

	type hostRow struct {
		ID       string
		Hostname string
		CPUCores int
		RAMGB    int
		Status   string
	}

	var rows []hostRow
	e.db.Table("hosts").Where("status = ?", "active").Find(&rows)

	hosts := make([]DRSHostUtilization, 0, len(rows))
	for _, r := range rows {
		var vmCount int64
		e.db.Table("instances").Where("host_id = ? AND status = ?", r.ID, "active").Count(&vmCount)

		cpuCores := r.CPUCores
		if cpuCores < 1 {
			cpuCores = 1
		}
		ramGB := r.RAMGB
		if ramGB < 1 {
			ramGB = 1
		}

		cpuUsage := float64(vmCount) * 2.0 / float64(cpuCores)
		ramUsage := float64(vmCount) * 4.0 / float64(ramGB)
		if cpuUsage > 1.0 {
			cpuUsage = 1.0
		}
		if ramUsage > 1.0 {
			ramUsage = 1.0
		}

		hosts = append(hosts, DRSHostUtilization{
			HostID:    r.ID,
			Hostname:  r.Hostname,
			CPUUsage:  cpuUsage,
			RAMUsage:  ramUsage,
			DiskUsage: 0.3,
			VMCount:   int(vmCount),
		})
	}
	return hosts
}

// ---------- DRS Routes (registered by scheduler) ----------

// SetupDRSRoutes adds DRS API endpoints to the router.
func (e *DRSEngine) SetupDRSRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/drs")
	{
		api.GET("/status", rp("scheduler", "get"), e.drsStatus)
		api.GET("/analysis", rp("scheduler", "get"), e.drsAnalysis)
		api.GET("/suggestions", rp("scheduler", "list"), e.drsSuggestions)
		api.POST("/evaluate", rp("scheduler", "create"), e.drsEvaluate)
	}
}

func (e *DRSEngine) drsStatus(c *gin.Context) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := gin.H{
		"enabled":             e.cfg.Enabled,
		"interval":            e.cfg.Interval.String(),
		"imbalance_threshold": e.cfg.ImbalanceThreshold,
		"auto_migrate":        e.cfg.AutoMigrate,
	}
	if e.lastAnalysis != nil {
		status["last_analysis"] = e.lastAnalysis.Timestamp
		status["balanced"] = e.lastAnalysis.Balanced
		status["imbalance"] = e.lastAnalysis.Imbalance
	}
	c.JSON(http.StatusOK, status)
}

func (e *DRSEngine) drsAnalysis(c *gin.Context) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.lastAnalysis == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no analysis available yet"})
		return
	}
	c.JSON(http.StatusOK, e.lastAnalysis)
}

func (e *DRSEngine) drsSuggestions(c *gin.Context) {
	var suggestions []DRSMigrationSuggestion
	e.db.Order("created_at DESC").Limit(50).Find(&suggestions)
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions, "count": len(suggestions)})
}

func (e *DRSEngine) drsEvaluate(c *gin.Context) {
	go e.evaluate()
	c.JSON(http.StatusAccepted, gin.H{"message": "DRS evaluation triggered"})
}
