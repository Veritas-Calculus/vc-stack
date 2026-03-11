package host

// Package host provides host management functionality.
// Inspired by CloudStack's host management architecture.

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
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

	// ExternalURL is the publicly reachable URL of the management server.
	// Used to generate install scripts for compute nodes.
	ExternalURL string

	// EvacuateCallback is called when instances need to be rescheduled
	// after a host goes down. If nil, instances are only marked as error.
	EvacuateCallback func(hostUUID string, instanceIDs []uint)
}

// Service provides host management operations.
type Service struct {
	db               *gorm.DB
	logger           *zap.Logger
	heartbeatTimeout time.Duration
	externalURL      string
	evacuateCallback func(hostUUID string, instanceIDs []uint)
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
		externalURL:      cfg.ExternalURL,
		evacuateCallback: cfg.EvacuateCallback,
	}

	// Start background tasks
	go s.monitorHosts()

	return s, nil
}

// SetupRoutes registers HTTP routes for the host service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1")
	{
		api.POST("/hosts/register", rp("host", "create"), s.registerHost)
		api.POST("/hosts/heartbeat", rp("host", "create"), s.heartbeat)
		api.POST("/hosts/test-connection", rp("host", "create"), s.testConnection)
		api.GET("/hosts", rp("host", "list"), s.listHosts)
		api.GET("/hosts/install-script", rp("host", "list"), s.generateInstallScript)
		api.POST("/hosts/deploy", rp("host", "create"), s.deployHost)
		api.GET("/hosts/:id", rp("host", "get"), s.getHost)
		api.PUT("/hosts/:id", rp("host", "update"), s.updateHost)
		api.DELETE("/hosts/:id", rp("host", "delete"), s.deleteHost)
		api.POST("/hosts/:id/enable", rp("host", "create"), s.enableHost)
		api.POST("/hosts/:id/disable", rp("host", "create"), s.disableHost)
		api.POST("/hosts/:id/maintenance", rp("host", "create"), s.maintenanceMode)
		api.POST("/hosts/:id/evacuate", rp("host", "create"), s.evacuateHostHTTP)
	}
}

// findHostByID looks up a host by UUID or numeric ID, handling PostgreSQL
// UUID type safely (avoids 'invalid input syntax for type uuid' errors).
func (s *Service) findHostByID(id string) (*models.Host, error) {
	var host models.Host
	query := s.db.Model(&models.Host{})
	if _, err := uuid.Parse(id); err == nil {
		query = query.Where("uuid = ?", id)
	} else {
		query = query.Where("id = ?", id)
	}
	err := query.First(&host).Error
	return &host, err
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
	ZoneID            *string                `json:"zone_id"`
	ClusterID         *string                `json:"cluster_id"`
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

	// Validate / resolve IP address — the DB column is type inet and requires a valid IP
	if ip := net.ParseIP(req.IPAddress); ip == nil {
		// Not a valid IP, try to resolve as hostname
		addrs, err := net.LookupHost(req.IPAddress)
		if err != nil || len(addrs) == 0 {
			s.logger.Warn("invalid ip_address and DNS resolution failed",
				zap.String("ip_address", req.IPAddress), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("ip_address '%s' is not a valid IP and could not be resolved via DNS", req.IPAddress),
			})
			return
		}
		s.logger.Info("resolved hostname to IP",
			zap.String("hostname", req.IPAddress),
			zap.String("resolved_ip", addrs[0]))
		req.IPAddress = addrs[0]
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

		managementPort := 8081
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
			Status:            models.HostStatusConnecting,
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

	// Update status to up if it was down, disconnected, or connecting
	if host.Status == models.HostStatusDown || host.Status == models.HostStatusDisconnected || host.Status == models.HostStatusConnecting {
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

	host, err := s.findHostByID(id)
	if err != nil {
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
		if err := s.db.Model(host).Updates(updates).Error; err != nil {
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

	host, err := s.findHostByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	if err := s.db.Delete(host).Error; err != nil {
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

	host, err := s.findHostByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	updates := map[string]interface{}{
		"resource_state": resourceState,
		"status":         status,
	}

	if err := s.db.Model(host).Updates(updates).Error; err != nil {
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

	// Run stale host cleanup less frequently (every 10 minutes).
	cleanupTicker := time.NewTicker(10 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkHostHealth()
		case <-cleanupTicker.C:
			s.cleanupStaleHosts()
		}
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

		// Trigger automatic evacuation for downed host.
		go s.evacuateHost(host.UUID, host.Name)
	}
}

// cleanupStaleHosts soft-deletes hosts that have been down for more than 7 days.
// This prevents zombie host records from accumulating in the database.
func (s *Service) cleanupStaleHosts() {
	staleThreshold := time.Now().Add(-7 * 24 * time.Hour) // 7 days

	var staleHosts []models.Host
	if err := s.db.Where("status = ? AND disconnected_at IS NOT NULL AND disconnected_at < ?",
		models.HostStatusDown, staleThreshold).Find(&staleHosts).Error; err != nil {
		s.logger.Error("failed to query stale hosts", zap.Error(err))
		return
	}

	for _, host := range staleHosts {
		if err := s.db.Delete(&host).Error; err != nil {
			s.logger.Error("failed to cleanup stale host",
				zap.String("uuid", host.UUID),
				zap.String("name", host.Name),
				zap.Error(err))
			continue
		}

		s.logger.Info("stale host auto-cleaned",
			zap.String("uuid", host.UUID),
			zap.String("name", host.Name),
			zap.Time("disconnected_at", *host.DisconnectedAt))
	}

	if len(staleHosts) > 0 {
		s.logger.Info("stale host cleanup complete",
			zap.Int("cleaned", len(staleHosts)))
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

// evacuateHost marks all active/building instances on a downed host as needing recovery.
// If an EvacuateCallback is configured, it triggers rescheduling.
func (s *Service) evacuateHost(hostUUID, hostName string) {
	var instanceIDs []uint
	if err := s.db.Table("instances").
		Select("id").
		Where("host_id = ? AND status IN (?, ?) AND deleted_at IS NULL", hostUUID, "active", "building").
		Pluck("id", &instanceIDs).Error; err != nil {
		s.logger.Error("failed to query instances for evacuation",
			zap.String("host_uuid", hostUUID), zap.Error(err))
		return
	}

	if len(instanceIDs) == 0 {
		s.logger.Info("no instances to evacuate on downed host",
			zap.String("host_uuid", hostUUID),
			zap.String("host_name", hostName))
		return
	}

	s.logger.Warn("evacuating instances from downed host",
		zap.String("host_uuid", hostUUID),
		zap.String("host_name", hostName),
		zap.Int("instance_count", len(instanceIDs)))

	// Mark all affected instances as error/host_down so they are
	// visible in the UI and eligible for rescheduling.
	result := s.db.Table("instances").
		Where("id IN ? AND status IN (?, ?)", instanceIDs, "active", "building").
		Updates(map[string]interface{}{
			"status":      "error",
			"power_state": "host_down",
		})
	if result.Error != nil {
		s.logger.Error("failed to mark instances for evacuation", zap.Error(result.Error))
		return
	}

	s.logger.Info("instances marked for evacuation",
		zap.Int64("affected", result.RowsAffected),
		zap.String("host_uuid", hostUUID))

	// Log an evacuation event for audit.
	s.db.Exec(`INSERT INTO system_events (resource_type, resource_id, action, status, message, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())`,
		"host", hostUUID, "evacuate", "success",
		fmt.Sprintf("Host %s marked down. %d instance(s) marked for evacuation.", hostName, result.RowsAffected))

	// If a callback is configured (e.g. compute service rescheduling), invoke it.
	if s.evacuateCallback != nil {
		s.evacuateCallback(hostUUID, instanceIDs)
	}
}

// evacuateHostHTTP handles POST /api/v1/hosts/:id/evacuate.
// Allows operators to manually trigger host evacuation.
func (s *Service) evacuateHostHTTP(c *gin.Context) {
	id := c.Param("id")

	host, err := s.findHostByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	// Count affected instances.
	var count int64
	s.db.Table("instances").
		Where("host_id = ? AND status IN (?, ?) AND deleted_at IS NULL", host.UUID, "active", "building").
		Count(&count)

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "no instances to evacuate", "affected": 0})
		return
	}

	// Trigger evacuation asynchronously.
	go s.evacuateHost(host.UUID, host.Name)

	c.JSON(http.StatusAccepted, gin.H{
		"ok":       true,
		"message":  fmt.Sprintf("evacuating %d instance(s) from host %s", count, host.Name),
		"affected": count,
	})
}

// testConnection checks TCP connectivity to a host:port.
func (s *Service) testConnection(c *gin.Context) {
	var req struct {
		IP   string `json:"ip" binding:"required"`
		Port int    `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	port := req.Port
	if port == 0 {
		port = 8081
	}

	// Resolve hostname if needed
	ip := req.IP
	if parsed := net.ParseIP(ip); parsed == nil {
		addrs, err := net.LookupHost(ip)
		if err != nil || len(addrs) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"reachable": false,
				"error":     fmt.Sprintf("cannot resolve hostname '%s'", ip),
			})
			return
		}
		ip = addrs[0]
	}

	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		s.logger.Info("test connection failed",
			zap.String("addr", addr), zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"reachable": false,
			"error":     fmt.Sprintf("connection to %s failed: %v", addr, err),
		})
		return
	}
	_ = conn.Close()

	s.logger.Info("test connection succeeded", zap.String("addr", addr))
	c.JSON(http.StatusOK, gin.H{
		"reachable":   true,
		"resolved_ip": ip,
	})
}
