package host

// Package host provides host management functionality.
// Inspired by CloudStack's host management architecture.

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Config contains the host service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger

	// HeartbeatTimeout is the duration after which a host is considered down.
	HeartbeatTimeout time.Duration
}

// Service provides host management operations.
type Service struct {
	db               *gorm.DB
	logger           *zap.Logger
	heartbeatTimeout time.Duration
}

// NewService creates a new host management service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.HeartbeatTimeout == 0 {
		cfg.HeartbeatTimeout = 2 * time.Minute
	}

	s := &Service{
		db:               cfg.DB,
		logger:           cfg.Logger,
		heartbeatTimeout: cfg.HeartbeatTimeout,
	}

	// Start background tasks
	go s.monitorHosts()

	return s, nil
}

// SetupRoutes registers HTTP routes for the host service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/hosts/register", s.registerHost)
		api.POST("/hosts/heartbeat", s.heartbeat)
		api.GET("/hosts", s.listHosts)
		api.GET("/hosts/:id", s.getHost)
		api.PUT("/hosts/:id", s.updateHost)
		api.DELETE("/hosts/:id", s.deleteHost)
		api.POST("/hosts/:id/enable", s.enableHost)
		api.POST("/hosts/:id/disable", s.disableHost)
		api.POST("/hosts/:id/maintenance", s.maintenanceMode)
	}
}

// RegisterRequest represents a host registration request.
type RegisterRequest struct {
	Name              string                 `json:"name" binding:"required"`
	Hostname          string                 `json:"hostname" binding:"required"`
	IPAddress         string                 `json:"ip_address" binding:"required"`
	ManagementPort    int                    `json:"management_port"`
	HostType          string                 `json:"host_type"`
	HypervisorType    string                 `json:"hypervisor_type"`
	HypervisorVersion string                 `json:"hypervisor_version"`
	CPUCores          int                    `json:"cpu_cores" binding:"required"`
	CPUSockets        int                    `json:"cpu_sockets"`
	CPUMhz            int64                  `json:"cpu_mhz"`
	RAMMB             int64                  `json:"ram_mb" binding:"required"`
	DiskGB            int64                  `json:"disk_gb" binding:"required"`
	Capabilities      map[string]interface{} `json:"capabilities"`
	Labels            map[string]interface{} `json:"labels"`
	AgentVersion      string                 `json:"agent_version"`
	ZoneID            *uint                  `json:"zone_id"`
	ClusterID         *uint                  `json:"cluster_id"`
}

// HeartbeatRequest represents a heartbeat update from a host.
type HeartbeatRequest struct {
	UUID              string `json:"uuid" binding:"required"`
	CPUAllocated      int    `json:"cpu_allocated"`
	RAMAllocatedMB    int64  `json:"ram_allocated_mb"`
	DiskAllocatedGB   int64  `json:"disk_allocated_gb"`
	HypervisorVersion string `json:"hypervisor_version"`
	AgentVersion      string `json:"agent_version"`
}

// UpdateRequest represents a host update request.
type UpdateRequest struct {
	Name          *string                 `json:"name"`
	Labels        *map[string]interface{} `json:"labels"`
	ResourceState *string                 `json:"resource_state"`
}

// registerHost handles host registration.
func (s *Service) registerHost(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if host already exists by IP and port
	var existing models.Host
	result := s.db.Where("ip_address = ? AND management_port = ?",
		req.IPAddress, req.ManagementPort).First(&existing)

	now := time.Now()
	var host *models.Host

	if result.Error == nil {
		// Host already exists, update it
		host = &existing
		host.Name = req.Name
		host.Hostname = req.Hostname
		host.CPUCores = req.CPUCores
		if req.CPUSockets > 0 {
			host.CPUSockets = req.CPUSockets
		}
		host.CPUMhz = req.CPUMhz
		host.RAMMB = req.RAMMB
		host.DiskGB = req.DiskGB
		host.HypervisorType = req.HypervisorType
		host.HypervisorVersion = req.HypervisorVersion
		host.AgentVersion = req.AgentVersion
		host.Capabilities = req.Capabilities
		host.Labels = req.Labels
		host.ZoneID = req.ZoneID
		host.ClusterID = req.ClusterID
		host.LastHeartbeat = &now
		host.Status = models.HostStatusUp

		if err := s.db.Save(host).Error; err != nil {
			s.logger.Error("failed to update existing host", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update host"})
			return
		}

		s.logger.Info("host re-registered",
			zap.String("uuid", host.UUID),
			zap.String("name", host.Name),
			zap.String("ip", host.IPAddress))
	} else {
		// Create new host
		hostType := models.HostTypeCompute
		if req.HostType != "" {
			hostType = models.HostType(req.HostType)
		}

		managementPort := 8091
		if req.ManagementPort > 0 {
			managementPort = req.ManagementPort
		}

		cpuSockets := 1
		if req.CPUSockets > 0 {
			cpuSockets = req.CPUSockets
		}

		host = &models.Host{
			UUID:              uuid.New().String(),
			Name:              req.Name,
			HostType:          hostType,
			Status:            models.HostStatusUp,
			ResourceState:     models.ResourceStateEnabled,
			Hostname:          req.Hostname,
			IPAddress:         req.IPAddress,
			ManagementPort:    managementPort,
			HypervisorType:    req.HypervisorType,
			HypervisorVersion: req.HypervisorVersion,
			CPUCores:          req.CPUCores,
			CPUSockets:        cpuSockets,
			CPUMhz:            req.CPUMhz,
			RAMMB:             req.RAMMB,
			DiskGB:            req.DiskGB,
			Capabilities:      req.Capabilities,
			Labels:            req.Labels,
			LastHeartbeat:     &now,
			AgentVersion:      req.AgentVersion,
			ZoneID:            req.ZoneID,
			ClusterID:         req.ClusterID,
		}

		if err := s.db.Create(host).Error; err != nil {
			s.logger.Error("failed to register host", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register host"})
			return
		}

		s.logger.Info("host registered",
			zap.String("uuid", host.UUID),
			zap.String("name", host.Name),
			zap.String("ip", host.IPAddress),
			zap.Int("cpu_cores", host.CPUCores),
			zap.Int64("ram_mb", host.RAMMB),
			zap.Int64("disk_gb", host.DiskGB))
	}

	c.JSON(http.StatusOK, gin.H{
		"uuid":    host.UUID,
		"host_id": host.ID,
		"status":  "registered",
	})
}

// heartbeat handles heartbeat updates from hosts.
func (s *Service) heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var host models.Host
	if err := s.db.Where("uuid = ?", req.UUID).First(&host).Error; err != nil {
		s.logger.Warn("heartbeat from unknown host", zap.String("uuid", req.UUID))
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_heartbeat":    now,
		"cpu_allocated":     req.CPUAllocated,
		"ram_allocated_mb":  req.RAMAllocatedMB,
		"disk_allocated_gb": req.DiskAllocatedGB,
	}

	if req.HypervisorVersion != "" {
		updates["hypervisor_version"] = req.HypervisorVersion
	}
	if req.AgentVersion != "" {
		updates["agent_version"] = req.AgentVersion
	}

	// Update status to up if it was down
	if host.Status == models.HostStatusDown || host.Status == models.HostStatusDisconnected {
		updates["status"] = models.HostStatusUp
		s.logger.Info("host came back online", zap.String("uuid", host.UUID))
	}

	if err := s.db.Model(&host).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update heartbeat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update heartbeat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// listHosts returns all registered hosts.
func (s *Service) listHosts(c *gin.Context) {
	var hosts []models.Host

	query := s.db.Model(&models.Host{})

	// Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by resource state
	if resourceState := c.Query("resource_state"); resourceState != "" {
		query = query.Where("resource_state = ?", resourceState)
	}

	// Filter by host type
	if hostType := c.Query("host_type"); hostType != "" {
		query = query.Where("host_type = ?", hostType)
	}

	// Filter by zone
	if zoneID := c.Query("zone_id"); zoneID != "" {
		query = query.Where("zone_id = ?", zoneID)
	}

	if err := query.Find(&hosts).Error; err != nil {
		s.logger.Error("failed to list hosts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list hosts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"hosts": hosts})
}

// getHost returns a single host by ID or UUID.
func (s *Service) getHost(c *gin.Context) {
	id := c.Param("id")

	var host models.Host
	query := s.db.Model(&models.Host{})

	// Try UUID first, then ID
	if _, err := uuid.Parse(id); err == nil {
		query = query.Where("uuid = ?", id)
	} else {
		query = query.Where("id = ?", id)
	}

	if err := query.First(&host).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		} else {
			s.logger.Error("failed to get host", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get host"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"host": host})
}

// updateHost updates a host's configuration.
func (s *Service) updateHost(c *gin.Context) {
	id := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var host models.Host
	if err := s.db.Where("uuid = ? OR id = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Labels != nil {
		updates["labels"] = *req.Labels
	}
	if req.ResourceState != nil {
		updates["resource_state"] = *req.ResourceState
	}

	if len(updates) > 0 {
		if err := s.db.Model(&host).Updates(updates).Error; err != nil {
			s.logger.Error("failed to update host", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update host"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"host": host})
}

// deleteHost removes a host from the system.
func (s *Service) deleteHost(c *gin.Context) {
	id := c.Param("id")

	var host models.Host
	if err := s.db.Where("uuid = ? OR id = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	if err := s.db.Delete(&host).Error; err != nil {
		s.logger.Error("failed to delete host", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete host"})
		return
	}

	s.logger.Info("host deleted", zap.String("uuid", host.UUID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// enableHost enables a host for resource allocation.
func (s *Service) enableHost(c *gin.Context) {
	s.updateHostState(c, models.ResourceStateEnabled, models.HostStatusUp)
}

// disableHost disables a host from resource allocation.
func (s *Service) disableHost(c *gin.Context) {
	s.updateHostState(c, models.ResourceStateDisabled, models.HostStatusDisabled)
}

// maintenanceMode puts a host into maintenance mode.
func (s *Service) maintenanceMode(c *gin.Context) {
	s.updateHostState(c, models.ResourceStateMaintenance, models.HostStatusMaintenance)
}

// updateHostState updates the state of a host.
func (s *Service) updateHostState(c *gin.Context, resourceState models.HostResourceState, status models.HostStatus) {
	id := c.Param("id")

	var host models.Host
	if err := s.db.Where("uuid = ? OR id = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	updates := map[string]interface{}{
		"resource_state": resourceState,
		"status":         status,
	}

	if err := s.db.Model(&host).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update host state", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update host state"})
		return
	}

	s.logger.Info("host state updated",
		zap.String("uuid", host.UUID),
		zap.String("resource_state", string(resourceState)),
		zap.String("status", string(status)))

	c.JSON(http.StatusOK, gin.H{"ok": true, "status": status, "resource_state": resourceState})
}

// monitorHosts is a background task that monitors host health.
func (s *Service) monitorHosts() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.checkHostHealth()
	}
}

// checkHostHealth checks the health of all hosts based on heartbeat.
func (s *Service) checkHostHealth() {
	threshold := time.Now().Add(-s.heartbeatTimeout)

	var hosts []models.Host
	if err := s.db.Where("status IN (?, ?) AND (last_heartbeat IS NULL OR last_heartbeat < ?)",
		models.HostStatusUp, models.HostStatusConnecting, threshold).Find(&hosts).Error; err != nil {
		s.logger.Error("failed to check host health", zap.Error(err))
		return
	}

	for _, host := range hosts {
		now := time.Now()
		updates := map[string]interface{}{
			"status":          models.HostStatusDown,
			"disconnected_at": now,
		}

		if err := s.db.Model(&host).Updates(updates).Error; err != nil {
			s.logger.Error("failed to mark host as down",
				zap.String("uuid", host.UUID),
				zap.Error(err))
			continue
		}

		s.logger.Warn("host marked as down due to heartbeat timeout",
			zap.String("uuid", host.UUID),
			zap.String("name", host.Name),
			zap.Time("last_heartbeat", *host.LastHeartbeat))
	}
}

// GetAvailableHosts returns hosts available for scheduling.
func (s *Service) GetAvailableHosts() ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.Where("status = ? AND resource_state = ?",
		models.HostStatusUp, models.ResourceStateEnabled).
		Order("cpu_allocated ASC, ram_allocated_mb ASC").
		Find(&hosts).Error
	return hosts, err
}
