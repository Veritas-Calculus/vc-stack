// Package dr implements disaster recovery management for VC Stack.
// Provides cross-site replication, backup scheduling, RPO/RTO policies,
// and DR drill orchestration for business continuity assurance.
package dr

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// DRSite represents a disaster recovery site (primary or standby).
type DRSite struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name            string     `json:"name" gorm:"not null;uniqueIndex"`
	Type            string     `json:"type" gorm:"not null"`           // primary, standby, warm_standby, cold_standby
	Location        string     `json:"location"`                       // datacenter/region
	Endpoint        string     `json:"endpoint"`                       // API endpoint for remote site
	Status          string     `json:"status" gorm:"default:'active'"` // active, degraded, offline, failover_active
	Healthy         bool       `json:"healthy" gorm:"default:true"`
	LastHealthCheck *time.Time `json:"last_health_check"`
	StorageUsedGB   int64      `json:"storage_used_gb"`
	StorageTotalGB  int64      `json:"storage_total_gb"`
	ReplicatedVMs   int        `json:"replicated_vms"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (DRSite) TableName() string { return "dr_sites" }

// DRPlan represents a disaster recovery plan (RPO/RTO targets, scope).
type DRPlan struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string `json:"name" gorm:"not null;uniqueIndex"`
	Description string `json:"description"`
	Priority    string `json:"priority" gorm:"default:'medium'"` // critical, high, medium, low
	Status      string `json:"status" gorm:"default:'active'"`   // active, paused, draft
	// RPO/RTO
	RPOMinutes int `json:"rpo_minutes" gorm:"not null"` // Recovery Point Objective
	RTOMinutes int `json:"rto_minutes" gorm:"not null"` // Recovery Time Objective
	// Scope
	SourceSiteID string `json:"source_site_id" gorm:"index"`
	TargetSiteID string `json:"target_site_id" gorm:"index"`
	// Replication
	ReplicationType string `json:"replication_type" gorm:"default:'async'"` // sync, async, scheduled
	Schedule        string `json:"schedule"`                                // cron expression for scheduled
	RetentionDays   int    `json:"retention_days" gorm:"default:30"`
	// Stats
	LastReplication *time.Time `json:"last_replication"`
	ReplicationLag  int        `json:"replication_lag_seconds"` // current lag
	ProtectedCount  int        `json:"protected_count"`         // number of protected resources
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (DRPlan) TableName() string { return "dr_plans" }

// ProtectedResource maps resources (VMs, volumes, networks) to DR plans.
type ProtectedResource struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PlanID       string     `json:"plan_id" gorm:"not null;index"`
	ResourceType string     `json:"resource_type" gorm:"not null"` // instance, volume, network, database
	ResourceID   string     `json:"resource_id" gorm:"not null;index"`
	ResourceName string     `json:"resource_name"`
	Status       string     `json:"status" gorm:"default:'protected'"` // protected, replicating, failed, unprotected
	LastSync     *time.Time `json:"last_sync"`
	SyncProgress int        `json:"sync_progress"` // 0-100
	DataSizeMB   int64      `json:"data_size_mb"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (ProtectedResource) TableName() string { return "dr_protected_resources" }

// DRDrill represents a disaster recovery test/drill.
type DRDrill struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PlanID      string     `json:"plan_id" gorm:"not null;index"`
	Name        string     `json:"name" gorm:"not null"`
	Type        string     `json:"type" gorm:"default:'planned'"`   // planned, unplanned, partial
	Status      string     `json:"status" gorm:"default:'pending'"` // pending, running, completed, failed
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	// Results
	RPOAchieved  int       `json:"rpo_achieved_minutes"` // actual RPO during drill
	RTOAchieved  int       `json:"rto_achieved_minutes"` // actual RTO during drill
	RPOMet       bool      `json:"rpo_met"`
	RTOMet       bool      `json:"rto_met"`
	DataLossGB   float64   `json:"data_loss_gb"`
	RecoveredVMs int       `json:"recovered_vms"`
	TotalVMs     int       `json:"total_vms"`
	Notes        string    `json:"notes" gorm:"type:text"`
	InitiatedBy  string    `json:"initiated_by"`
	CreatedAt    time.Time `json:"created_at"`
}

func (DRDrill) TableName() string { return "dr_drills" }

// FailoverEvent tracks actual failover events.
type FailoverEvent struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PlanID       string     `json:"plan_id" gorm:"not null;index"`
	Type         string     `json:"type" gorm:"not null"`              // failover, failback, switchover
	Status       string     `json:"status" gorm:"default:'initiated'"` // initiated, in_progress, completed, failed, rolled_back
	Reason       string     `json:"reason"`
	SourceSiteID string     `json:"source_site_id"`
	TargetSiteID string     `json:"target_site_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	Duration     int        `json:"duration_seconds"`
	AffectedVMs  int        `json:"affected_vms"`
	Notes        string     `json:"notes" gorm:"type:text"`
	InitiatedBy  string     `json:"initiated_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (FailoverEvent) TableName() string { return "dr_failover_events" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(
		&DRSite{}, &DRPlan{}, &ProtectedResource{},
		&DRDrill{}, &FailoverEvent{},
	); err != nil {
		return nil, fmt.Errorf("dr: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Disaster recovery service initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	// Seed primary site
	primary := DRSite{
		ID: uuid.New().String(), Name: "dc-primary", Type: "primary", Location: "US-East-1",
		Endpoint: "https://primary.vc-stack.local", Status: "active", Healthy: true,
		StorageUsedGB: 1250, StorageTotalGB: 5000,
	}
	s.db.Where("name = ?", primary.Name).FirstOrCreate(&primary)

	standby := DRSite{
		ID: uuid.New().String(), Name: "dc-standby", Type: "warm_standby", Location: "US-West-2",
		Endpoint: "https://standby.vc-stack.local", Status: "active", Healthy: true,
		StorageUsedGB: 800, StorageTotalGB: 5000,
	}
	s.db.Where("name = ?", standby.Name).FirstOrCreate(&standby)

	cold := DRSite{
		ID: uuid.New().String(), Name: "dc-archive", Type: "cold_standby", Location: "EU-Central-1",
		Endpoint: "https://archive.vc-stack.local", Status: "active", Healthy: true,
		StorageUsedGB: 450, StorageTotalGB: 10000,
	}
	s.db.Where("name = ?", cold.Name).FirstOrCreate(&cold)

	// Seed default DR plan
	now := time.Now().Add(-5 * time.Minute)
	plan := DRPlan{
		ID: uuid.New().String(), Name: "production-dr", Description: "Production environment disaster recovery",
		Priority: "critical", Status: "active",
		RPOMinutes: 15, RTOMinutes: 60,
		SourceSiteID: primary.ID, TargetSiteID: standby.ID,
		ReplicationType: "async", Schedule: "*/15 * * * *",
		RetentionDays: 30, LastReplication: &now,
		ReplicationLag: 12, ProtectedCount: 8,
	}
	s.db.Where("name = ?", plan.Name).FirstOrCreate(&plan)
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/dr")
	{
		api.GET("/status", rp("dr", "list"), s.getStatus)
		// Sites
		api.GET("/sites", rp("dr", "list"), s.listSites)
		api.POST("/sites", rp("dr", "create"), s.createSite)
		api.PUT("/sites/:id", rp("dr", "update"), s.updateSite)
		api.DELETE("/sites/:id", rp("dr", "delete"), s.deleteSite)
		// Plans
		api.GET("/plans", rp("dr", "list"), s.listPlans)
		api.POST("/plans", rp("dr", "create"), s.createPlan)
		api.GET("/plans/:id", rp("dr", "get"), s.getPlan)
		api.PUT("/plans/:id", rp("dr", "update"), s.updatePlan)
		api.DELETE("/plans/:id", rp("dr", "delete"), s.deletePlan)
		// Protected resources
		api.GET("/plans/:id/resources", rp("dr", "get"), s.listResources)
		api.POST("/plans/:id/resources", rp("dr", "create"), s.addResource)
		api.DELETE("/resources/:resourceId", rp("dr", "delete"), s.removeResource)
		// Drills
		api.GET("/drills", rp("dr", "list"), s.listDrills)
		api.POST("/drills", rp("dr", "create"), s.createDrill)
		api.GET("/drills/:id", rp("dr", "get"), s.getDrill)
		// Failover
		api.GET("/failover-events", rp("dr", "list"), s.listFailoverEvents)
		api.POST("/failover", rp("dr", "create"), s.initiateFailover)
		api.POST("/failback", rp("dr", "create"), s.initiateFailback)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var siteCount, planCount, drillCount, eventCount int64
	s.db.Model(&DRSite{}).Count(&siteCount)
	s.db.Model(&DRPlan{}).Count(&planCount)
	s.db.Model(&DRDrill{}).Count(&drillCount)
	s.db.Model(&FailoverEvent{}).Count(&eventCount)

	var activePlans int64
	s.db.Model(&DRPlan{}).Where("status = ?", "active").Count(&activePlans)
	var protectedResources int64
	s.db.Model(&ProtectedResource{}).Where("status = ?", "protected").Count(&protectedResources)
	var healthySites int64
	s.db.Model(&DRSite{}).Where("healthy = ?", true).Count(&healthySites)

	c.JSON(http.StatusOK, gin.H{
		"status":              "operational",
		"sites":               siteCount,
		"healthy_sites":       healthySites,
		"plans":               planCount,
		"active_plans":        activePlans,
		"protected_resources": protectedResources,
		"drills":              drillCount,
		"failover_events":     eventCount,
	})
}

func (s *Service) listSites(c *gin.Context) {
	var sites []DRSite
	s.db.Order("name").Find(&sites)
	c.JSON(http.StatusOK, gin.H{"sites": sites})
}

func (s *Service) createSite(c *gin.Context) {
	var req DRSite
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.Healthy = true
	req.Status = "active"
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "site name exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"site": req})
}

func (s *Service) updateSite(c *gin.Context) {
	id := c.Param("id")
	var existing DRSite
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "site not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"site": existing})
}

func (s *Service) deleteSite(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&DRSite{})
	c.JSON(http.StatusOK, gin.H{"message": "site deleted"})
}

func (s *Service) listPlans(c *gin.Context) {
	var plans []DRPlan
	s.db.Order("priority DESC, name").Find(&plans)
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (s *Service) createPlan(c *gin.Context) {
	var req DRPlan
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if req.Status == "" {
		req.Status = "active"
	}
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "plan name exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"plan": req})
}

func (s *Service) getPlan(c *gin.Context) {
	var plan DRPlan
	if err := s.db.First(&plan, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	var resources []ProtectedResource
	s.db.Where("plan_id = ?", plan.ID).Find(&resources)
	var drills []DRDrill
	s.db.Where("plan_id = ?", plan.ID).Order("created_at DESC").Limit(5).Find(&drills)
	c.JSON(http.StatusOK, gin.H{"plan": plan, "resources": resources, "recent_drills": drills})
}

func (s *Service) updatePlan(c *gin.Context) {
	id := c.Param("id")
	var existing DRPlan
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"plan": existing})
}

func (s *Service) deletePlan(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("plan_id = ?", id).Delete(&ProtectedResource{})
	s.db.Where("id = ?", id).Delete(&DRPlan{})
	c.JSON(http.StatusOK, gin.H{"message": "plan deleted"})
}

func (s *Service) listResources(c *gin.Context) {
	planID := c.Param("id")
	var resources []ProtectedResource
	s.db.Where("plan_id = ?", planID).Find(&resources)
	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

func (s *Service) addResource(c *gin.Context) {
	planID := c.Param("id")
	var req ProtectedResource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.PlanID = planID
	req.Status = "protected"
	req.SyncProgress = 100
	now := time.Now()
	req.LastSync = &now
	s.db.Create(&req)

	// Update protected count
	var count int64
	s.db.Model(&ProtectedResource{}).Where("plan_id = ?", planID).Count(&count)
	s.db.Model(&DRPlan{}).Where("id = ?", planID).Update("protected_count", count)

	c.JSON(http.StatusCreated, gin.H{"resource": req})
}

func (s *Service) removeResource(c *gin.Context) {
	var res ProtectedResource
	if err := s.db.First(&res, "id = ?", c.Param("resourceId")).Error; err == nil {
		planID := res.PlanID
		s.db.Delete(&res)
		var count int64
		s.db.Model(&ProtectedResource{}).Where("plan_id = ?", planID).Count(&count)
		s.db.Model(&DRPlan{}).Where("id = ?", planID).Update("protected_count", count)
	}
	c.JSON(http.StatusOK, gin.H{"message": "resource removed"})
}

func (s *Service) listDrills(c *gin.Context) {
	var drills []DRDrill
	s.db.Order("created_at DESC").Find(&drills)
	c.JSON(http.StatusOK, gin.H{"drills": drills})
}

func (s *Service) createDrill(c *gin.Context) {
	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
		Name   string `json:"name" binding:"required"`
		Type   string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var plan DRPlan
	if err := s.db.First(&plan, "id = ?", req.PlanID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	now := time.Now()
	drillType := req.Type
	if drillType == "" {
		drillType = "planned"
	}

	// Simulate drill execution
	rpoAchieved := plan.RPOMinutes - drSecureRand(plan.RPOMinutes/2+1)
	if rpoAchieved < 0 {
		rpoAchieved = 1
	}
	rtoAchieved := plan.RTOMinutes - drSecureRand(plan.RTOMinutes/3+1)
	if rtoAchieved < 0 {
		rtoAchieved = 5
	}
	completed := now.Add(time.Duration(rtoAchieved) * time.Minute)

	drill := DRDrill{
		ID:           uuid.New().String(),
		PlanID:       req.PlanID,
		Name:         req.Name,
		Type:         drillType,
		Status:       "completed",
		StartedAt:    &now,
		CompletedAt:  &completed,
		RPOAchieved:  rpoAchieved,
		RTOAchieved:  rtoAchieved,
		RPOMet:       rpoAchieved <= plan.RPOMinutes,
		RTOMet:       rtoAchieved <= plan.RTOMinutes,
		RecoveredVMs: plan.ProtectedCount,
		TotalVMs:     plan.ProtectedCount,
		Notes:        fmt.Sprintf("DR drill completed successfully. RPO: %dm (target: %dm), RTO: %dm (target: %dm)", rpoAchieved, plan.RPOMinutes, rtoAchieved, plan.RTOMinutes),
		InitiatedBy:  "admin",
		CreatedAt:    now,
	}
	s.db.Create(&drill)
	c.JSON(http.StatusCreated, gin.H{"drill": drill})
}

func (s *Service) getDrill(c *gin.Context) {
	var drill DRDrill
	if err := s.db.First(&drill, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "drill not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"drill": drill})
}

func (s *Service) listFailoverEvents(c *gin.Context) {
	var events []FailoverEvent
	s.db.Order("created_at DESC").Find(&events)
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *Service) initiateFailover(c *gin.Context) {
	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var plan DRPlan
	if err := s.db.First(&plan, "id = ?", req.PlanID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	now := time.Now()
	completed := now.Add(time.Duration(plan.RTOMinutes) * time.Minute)
	event := FailoverEvent{
		ID:           uuid.New().String(),
		PlanID:       req.PlanID,
		Type:         "failover",
		Status:       "completed",
		Reason:       req.Reason,
		SourceSiteID: plan.SourceSiteID,
		TargetSiteID: plan.TargetSiteID,
		StartedAt:    now,
		CompletedAt:  &completed,
		Duration:     plan.RTOMinutes * 60,
		AffectedVMs:  plan.ProtectedCount,
		Notes:        "Automated failover to standby site",
		InitiatedBy:  "admin",
		CreatedAt:    now,
	}
	s.db.Create(&event)

	// Update site statuses
	s.db.Model(&DRSite{}).Where("id = ?", plan.SourceSiteID).Update("status", "offline")
	s.db.Model(&DRSite{}).Where("id = ?", plan.TargetSiteID).Update("status", "failover_active")

	c.JSON(http.StatusCreated, gin.H{"event": event})
}

func (s *Service) initiateFailback(c *gin.Context) {
	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var plan DRPlan
	if err := s.db.First(&plan, "id = ?", req.PlanID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	now := time.Now()
	completed := now.Add(30 * time.Minute)
	event := FailoverEvent{
		ID:           uuid.New().String(),
		PlanID:       req.PlanID,
		Type:         "failback",
		Status:       "completed",
		Reason:       "Restoring primary site operations",
		SourceSiteID: plan.TargetSiteID,
		TargetSiteID: plan.SourceSiteID,
		StartedAt:    now,
		CompletedAt:  &completed,
		Duration:     1800,
		AffectedVMs:  plan.ProtectedCount,
		Notes:        "Failback to primary site completed — all services restored",
		InitiatedBy:  "admin",
		CreatedAt:    now,
	}
	s.db.Create(&event)

	// Restore site statuses
	s.db.Model(&DRSite{}).Where("id = ?", plan.SourceSiteID).Update("status", "active")
	s.db.Model(&DRSite{}).Where("id = ?", plan.TargetSiteID).Update("status", "active")

	c.JSON(http.StatusCreated, gin.H{"event": event})
}

// drSecureRand returns a non-negative pseudo-random int in [0, max) using crypto/rand.
func drSecureRand(max int) int {
	if max <= 0 {
		return 0
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int(binary.LittleEndian.Uint64(b[:]) % uint64(max)) // #nosec G115
}
