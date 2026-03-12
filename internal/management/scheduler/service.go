package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/pkg/circuitbreaker"
	"github.com/Veritas-Calculus/vc-stack/pkg/dlock"
	"github.com/Veritas-Calculus/vc-stack/pkg/mq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Scheduling strategies.
const (
	// StrategyLeastAllocated picks the host with the lowest resource usage (default, spread).
	StrategyLeastAllocated = "spread"
	// StrategyMostAllocated picks the host with the highest usage (bin-packing).
	StrategyMostAllocated = "pack"
	// StrategyZoneAffinity prefers the specified zone but falls back to other zones.
	StrategyZoneAffinity = "zone-affinity"
	// StrategyZoneRequired strictly requires placement in the specified zone.
	StrategyZoneRequired = "zone-required"
)

// Zone health states.
const (
	ZoneHealthy  = "healthy"  // >50% hosts up
	ZoneDegraded = "degraded" // 1-50% hosts up
	ZoneDown     = "down"     // 0 hosts up
)

// OvercommitConfig defines resource overcommit ratios.
type OvercommitConfig struct {
	CPURatio  float64 // e.g., 4.0 means 1 physical CPU supports 4 vCPUs. 0 or 1.0 = no overcommit.
	RAMRatio  float64 // e.g., 1.5 means 64GB physical can schedule 96GB of VM RAM.
	DiskRatio float64 // e.g., 2.0 for thin-provisioned storage.
}

type Config struct {
	DB         *gorm.DB
	Logger     *zap.Logger
	Overcommit OvercommitConfig

	// DLock is an optional distributed lock manager for leader election.
	// If nil, this instance acts as the sole scheduler (single-instance mode).
	DLock *dlock.Manager

	// MQ is an optional message bus. If set, VM commands are published
	// to Kafka instead of dispatched via synchronous REST.
	MQ mq.MessageBus
}

// Service provides scheduling and VM dispatch.
// It reads host data from the persistent `hosts` table instead of
// keeping a volatile in-memory map that would be lost on restart.
//
// In multi-replica mode, only the elected leader performs scheduling.
// Follower replicas serve read-only endpoints but reject write operations.
type Service struct {
	db         *gorm.DB
	logger     *zap.Logger
	overcommit OvercommitConfig
	cbManager  *circuitbreaker.Manager

	// Leader election (nil = always leader).
	leader   dlock.LeaderElector
	isLeader atomic.Bool

	// Message bus (nil = synchronous REST dispatch).
	mq mq.MessageBus
}

// ScheduleRequest describes what resources to schedule and placement preferences.
type ScheduleRequest struct {
	VCPUs         int    `json:"vcpus"`
	RAMMB         int    `json:"ram_mb"`
	DiskGB        int    `json:"disk_gb"`
	ZoneID        string `json:"zone_id"`
	ClusterID     string `json:"cluster_id"`
	Strategy      string `json:"strategy"`        // spread (default), pack, zone-affinity, zone-required
	ServerGroupID string `json:"server_group_id"` // optional server group for affinity/anti-affinity
}

// ScheduleResponse describes the scheduling result.
type ScheduleResponse struct {
	NodeID    string `json:"node"`
	ZoneID    string `json:"zone_id,omitempty"`
	ClusterID string `json:"cluster_id,omitempty"`
	Strategy  string `json:"strategy"`
	Reason    string `json:"reason"`
}

// ZoneCapacity aggregates resource capacity and usage for a zone.
type ZoneCapacity struct {
	ZoneID          string  `json:"zone_id"`
	ZoneName        string  `json:"zone_name,omitempty"`
	Health          string  `json:"health"` // healthy, degraded, down
	TotalHosts      int     `json:"total_hosts"`
	ActiveHosts     int     `json:"active_hosts"`
	TotalCPU        int     `json:"total_cpu"`
	AllocatedCPU    int     `json:"allocated_cpu"`
	AvailableCPU    int     `json:"available_cpu"`
	TotalRAMMB      int64   `json:"total_ram_mb"`
	AllocatedRAMMB  int64   `json:"allocated_ram_mb"`
	AvailableRAMMB  int64   `json:"available_ram_mb"`
	TotalDiskGB     int64   `json:"total_disk_gb"`
	AllocatedDiskGB int64   `json:"allocated_disk_gb"`
	AvailableDiskGB int64   `json:"available_disk_gb"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	RAMUsagePercent float64 `json:"ram_usage_percent"`
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Normalize overcommit ratios — values < 1.0 are treated as no overcommit.
	if cfg.Overcommit.CPURatio < 1.0 {
		cfg.Overcommit.CPURatio = 1.0
	}
	if cfg.Overcommit.RAMRatio < 1.0 {
		cfg.Overcommit.RAMRatio = 1.0
	}
	if cfg.Overcommit.DiskRatio < 1.0 {
		cfg.Overcommit.DiskRatio = 1.0
	}

	cbMgr := circuitbreaker.NewManager(circuitbreaker.Options{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		ResetTimeout:     30 * time.Second,
		Logger:           cfg.Logger.Named("circuit-breaker"),
	})

	cfg.Logger.Info("scheduler initialized",
		zap.Float64("cpu_overcommit", cfg.Overcommit.CPURatio),
		zap.Float64("ram_overcommit", cfg.Overcommit.RAMRatio),
		zap.Float64("disk_overcommit", cfg.Overcommit.DiskRatio))

	svc := &Service{
		db:         cfg.DB,
		logger:     cfg.Logger,
		overcommit: cfg.Overcommit,
		cbManager:  cbMgr,
		mq:         cfg.MQ,
	}

	// If distributed lock manager is available, start leader election.
	if cfg.DLock != nil {
		leader, err := cfg.DLock.NewLeaderElection("/vc/leader/scheduler", 15*time.Second)
		if err != nil {
			return nil, fmt.Errorf("scheduler: leader election init failed: %w", err)
		}
		svc.leader = leader

		// Get a unique identifier for this instance.
		hostname, _ := os.Hostname()
		instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

		// Campaign in the background — blocks until elected.
		go func() {
			cfg.Logger.Info("scheduler: starting leader election campaign",
				zap.String("instance", instanceID))
			if err := leader.Campaign(context.Background(), instanceID); err != nil {
				cfg.Logger.Error("scheduler: leader campaign failed", zap.Error(err))
				return
			}
			svc.isLeader.Store(true)
			cfg.Logger.Info("scheduler: elected as leader",
				zap.String("instance", instanceID))
		}()
	} else {
		// Single-instance mode: always leader.
		svc.isLeader.Store(true)
		cfg.Logger.Info("scheduler: running in single-instance mode (always leader)")
	}

	return svc, nil
}

func (s *Service) SetupRoutes(r *gin.Engine) {
	rp := middleware.RequirePermission
	// Health check under scheduler prefix to avoid conflicts
	r.GET("/api/scheduler/health", func(c *gin.Context) {
		status := "follower"
		if s.isLeader.Load() {
			status = "leader"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "vc-scheduler",
			"role":      status,
			"is_leader": s.isLeader.Load(),
		})
	})
	v1 := r.Group("/api/v1")
	{
		// Write endpoints — leader only.
		v1.POST("/schedule", rp("scheduler", "create"), s.requireLeader(), s.schedule)
		v1.POST("/dispatch/vms", rp("scheduler", "create"), s.requireLeader(), s.dispatchVMCreate)

		// Read endpoints — all replicas can serve.
		v1.GET("/scheduler/zones", rp("scheduler", "list"), s.listZoneCapacities)
		v1.GET("/scheduler/zones/:zone_id", rp("scheduler", "get"), s.getZoneCapacity)
		v1.GET("/scheduler/stats", rp("scheduler", "get"), s.schedulerStats)

		// Server group management.
		v1.POST("/server-groups", rp("scheduler", "create"), s.requireLeader(), s.createServerGroup)
		v1.GET("/server-groups", rp("scheduler", "list"), s.listServerGroups)
		v1.GET("/server-groups/:id", rp("scheduler", "get"), s.getServerGroup)
		v1.DELETE("/server-groups/:id", rp("scheduler", "delete"), s.requireLeader(), s.deleteServerGroup)

		// Diagnostics — all replicas.
		v1.GET("/scheduler/circuit-breakers", rp("scheduler", "get"), s.listCircuitBreakers)
		v1.GET("/scheduler/leader", rp("scheduler", "get"), s.leaderStatus)

		// Legacy /nodes endpoints — delegate to hosts table.
		v1.POST("/nodes/register", rp("scheduler", "create"), s.requireLeader(), s.legacyRegisterNode)
		v1.POST("/nodes/heartbeat", rp("scheduler", "create"), s.legacyHeartbeat)
		v1.GET("/nodes", rp("scheduler", "list"), s.listNodes)
		v1.GET("/nodes/:id", rp("scheduler", "get"), s.getNode)
		v1.DELETE("/nodes/:id", rp("scheduler", "delete"), s.requireLeader(), s.deleteNode)
	}
}

// requireLeader is a Gin middleware that rejects write requests when
// this instance is not the scheduler leader.
func (s *Service) requireLeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.isLeader.Load() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":     "this instance is not the scheduler leader",
				"code":      "SCHEDULER_NOT_LEADER",
				"is_leader": false,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// leaderStatus returns the current leader election status.
func (s *Service) leaderStatus(c *gin.Context) {
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	resp := gin.H{
		"is_leader":   s.isLeader.Load(),
		"instance_id": instanceID,
	}

	if s.leader != nil {
		resp["leader_value"] = s.leader.LeaderValue()
		resp["mode"] = "distributed"
	} else {
		resp["mode"] = "single-instance"
	}

	c.JSON(http.StatusOK, resp)
}

// dispatchVMCreate selects a host and forwards the VM create request.
func (s *Service) dispatchVMCreate(c *gin.Context) {
	s.logger.Info("dispatch request received", zap.String("client_ip", c.ClientIP()))

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		s.logger.Warn("dispatch invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid create payload"})
		return
	}

	getInt := func(k string) int {
		if v, ok := payload[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			case int64:
				return int(t)
			}
		}
		return 0
	}
	req := ScheduleRequest{VCPUs: getInt("vcpus"), RAMMB: getInt("memory_mb"), DiskGB: getInt("disk_gb")}
	if req.VCPUs <= 0 || req.RAMMB <= 0 || req.DiskGB <= 0 {
		s.logger.Warn("dispatch missing resources", zap.Any("request", req))
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing vcpus/memory_mb/disk_gb in payload"})
		return
	}

	host, schedResp := s.selectHost(req)
	if host == nil {
		s.logger.Warn("dispatch no hosts available", zap.String("reason", schedResp.Reason))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": schedResp.Reason})
		return
	}

	s.logger.Info("dispatch selected host",
		zap.String("uuid", host.UUID), zap.String("name", host.Name),
		zap.String("reason", schedResp.Reason))

	// Forward request to the selected host's VM driver via circuit breaker.
	addr := strings.TrimRight(host.GetManagementURL(), "/") + "/api/v1/vms"
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode payload"})
		return
	}

	var httpResp *http.Response
	cbErr := s.cbManager.Execute(host.UUID, func() error {
		reqHTTP, _ := http.NewRequest("POST", addr, buf)
		reqHTTP.Header.Set("Content-Type", "application/json")
		var reqErr error
		httpResp, reqErr = http.DefaultClient.Do(reqHTTP) //nolint:bodyclose // closed on both error and success paths
		return reqErr
	})
	if cbErr != nil {
		if httpResp != nil {
			_ = httpResp.Body.Close()
		}
		s.logger.Error("dispatch forward failed",
			zap.String("addr", addr),
			zap.String("host", host.UUID),
			zap.Error(cbErr))
		c.JSON(http.StatusBadGateway, gin.H{"error": circuitbreaker.FormatError(host.Name, cbErr).Error()})
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	var upstream map[string]any
	if err := json.NewDecoder(httpResp.Body).Decode(&upstream); err != nil {
		upstream = map[string]any{"error": "invalid upstream response"}
	}
	out := map[string]any{"node": host.UUID}
	for k, v := range upstream {
		out[k] = v
	}
	c.JSON(httpResp.StatusCode, out)
}

// listNodes returns hosts from DB (replaces old in-memory list).
func (s *Service) listNodes(c *gin.Context) {
	var hosts []models.Host
	if err := s.db.Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list nodes"})
		return
	}

	// Map to legacy node format for backwards compatibility.
	nodes := make([]map[string]any, 0, len(hosts))
	for _, h := range hosts {
		nodes = append(nodes, hostToNode(h))
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

// getNode retrieves a single host from DB.
func (s *Service) getNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	var host models.Host
	if err := s.db.Where("uuid = ? OR name = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": hostToNode(host)})
}

// deleteNode removes a host from DB (soft delete).
func (s *Service) deleteNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	result := s.db.Where("uuid = ? OR name = ?", id, id).Delete(&models.Host{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// legacyRegisterNode handles old-style /nodes/register by forwarding to /hosts/register.
// This is a compatibility stub — real registration goes through the host service.
func (s *Service) legacyRegisterNode(c *gin.Context) {
	s.logger.Warn("deprecated: /nodes/register called, use /hosts/register instead")
	c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "use /hosts/register instead"})
}

// legacyHeartbeat handles old-style /nodes/heartbeat.
func (s *Service) legacyHeartbeat(c *gin.Context) {
	s.logger.Warn("deprecated: /nodes/heartbeat called, use /hosts/heartbeat instead")
	c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "use /hosts/heartbeat instead"})
}

// schedule chooses a host based on the requested resources and scheduling strategy.
func (s *Service) schedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}
	host, resp := s.selectHost(req)
	if host == nil {
		apierrors.Respond(c, apierrors.ErrNoHostAvailable().WithDetail(resp.Reason))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// selectHost finds the best available host using the requested scheduling strategy.
func (s *Service) selectHost(req ScheduleRequest) (*models.Host, ScheduleResponse) {
	strategy := req.Strategy
	if strategy == "" {
		strategy = StrategyLeastAllocated
	}

	emptyResp := ScheduleResponse{Strategy: strategy}

	// Build base query: status=up, resource_state=enabled.
	baseQuery := s.db.Where("status = ? AND resource_state = ?",
		models.HostStatusUp, models.ResourceStateEnabled)

	// Cluster affinity always applies strictly.
	if req.ClusterID != "" {
		baseQuery = baseQuery.Where("cluster_id = ?", req.ClusterID)
	}

	var candidates []models.Host

	switch strategy {
	case StrategyZoneRequired:
		// Strict zone placement — zone must be specified, no fallback.
		if req.ZoneID == "" {
			emptyResp.Reason = "zone_id is required for zone-required strategy"
			return nil, emptyResp
		}
		candidates = s.findHostsInZone(baseQuery, req)
		if len(candidates) == 0 {
			emptyResp.Reason = fmt.Sprintf("no hosts available in zone %s with sufficient resources", req.ZoneID)
			return nil, emptyResp
		}

	case StrategyZoneAffinity:
		// Prefer specified zone, fall back to other zones.
		if req.ZoneID != "" {
			candidates = s.findHostsInZone(baseQuery, req)
		}
		if len(candidates) == 0 {
			// Fallback: try all zones.
			s.logger.Info("zone-affinity fallback", zap.String("preferred_zone", req.ZoneID))
			candidates = s.findHostsAllZones(baseQuery, req)
		}
		if len(candidates) == 0 {
			emptyResp.Reason = "no hosts available with sufficient resources (including fallback zones)"
			return nil, emptyResp
		}

	default: // spread or pack
		// Zone filter is optional hint, not strict.
		if req.ZoneID != "" {
			candidates = s.findHostsInZone(baseQuery, req)
		}
		if len(candidates) == 0 {
			candidates = s.findHostsAllZones(baseQuery, req)
		}
		if len(candidates) == 0 {
			emptyResp.Reason = "no hosts available with sufficient resources"
			return nil, emptyResp
		}
	}

	// Apply server group constraints if specified.
	if req.ServerGroupID != "" {
		candidates = s.applyServerGroupFilter(candidates, req.ServerGroupID)
		if len(candidates) == 0 {
			emptyResp.Reason = "no hosts available that satisfy server group policy"
			return nil, emptyResp
		}
	}

	// Sort candidates based on strategy.
	s.sortCandidates(candidates, strategy)

	chosen := &candidates[0]

	// Build reason string with overcommit info if active.
	overcommitInfo := ""
	if s.overcommit.CPURatio > 1.0 || s.overcommit.RAMRatio > 1.0 {
		overcommitInfo = fmt.Sprintf(" overcommit(cpu=%.1fx,ram=%.1fx)", s.overcommit.CPURatio, s.overcommit.RAMRatio)
	}
	reason := fmt.Sprintf("%s: cpu=%d/%d ram=%d/%d%s",
		strategy, chosen.CPUAllocated, chosen.CPUCores, chosen.RAMAllocatedMB, chosen.RAMMB, overcommitInfo)

	resp := ScheduleResponse{
		NodeID:   chosen.UUID,
		Strategy: strategy,
		Reason:   reason,
	}
	if chosen.ZoneID != nil {
		resp.ZoneID = *chosen.ZoneID
	}
	if chosen.ClusterID != nil {
		resp.ClusterID = *chosen.ClusterID
	}
	return chosen, resp
}

// findHostsInZone queries hosts within a specific zone that fit the request.
func (s *Service) findHostsInZone(baseQuery *gorm.DB, req ScheduleRequest) []models.Host {
	var hosts []models.Host
	q := baseQuery.Session(&gorm.Session{NewDB: true}).
		Where("status = ? AND resource_state = ?", models.HostStatusUp, models.ResourceStateEnabled).
		Where("zone_id = ?", req.ZoneID)
	if req.ClusterID != "" {
		q = q.Where("cluster_id = ?", req.ClusterID)
	}
	if err := q.Find(&hosts).Error; err != nil {
		s.logger.Error("failed to query zone hosts", zap.Error(err))
		return nil
	}
	return s.filterByResources(hosts, req)
}

// findHostsAllZones queries hosts across all zones that fit the request.
func (s *Service) findHostsAllZones(baseQuery *gorm.DB, req ScheduleRequest) []models.Host {
	var hosts []models.Host
	q := baseQuery.Session(&gorm.Session{NewDB: true}).
		Where("status = ? AND resource_state = ?", models.HostStatusUp, models.ResourceStateEnabled)
	if req.ClusterID != "" {
		q = q.Where("cluster_id = ?", req.ClusterID)
	}
	if err := q.Find(&hosts).Error; err != nil {
		s.logger.Error("failed to query hosts", zap.Error(err))
		return nil
	}
	return s.filterByResources(hosts, req)
}

// filterByResources filters hosts by available resources, applying overcommit ratios.
func (s *Service) filterByResources(hosts []models.Host, req ScheduleRequest) []models.Host {
	candidates := make([]models.Host, 0, len(hosts))
	for _, h := range hosts {
		if s.hasEnoughWithOvercommit(h, req.VCPUs, int64(req.RAMMB), int64(req.DiskGB)) {
			candidates = append(candidates, h)
		}
	}
	return candidates
}

// hasEnoughWithOvercommit checks if a host has enough resources after applying overcommit ratios.
func (s *Service) hasEnoughWithOvercommit(h models.Host, cpus int, ramMB, diskGB int64) bool {
	effectiveCPU := int(float64(h.CPUCores) * s.overcommit.CPURatio)
	effectiveRAM := int64(float64(h.RAMMB) * s.overcommit.RAMRatio)
	effectiveDisk := int64(float64(h.DiskGB) * s.overcommit.DiskRatio)

	freeCPU := effectiveCPU - h.CPUAllocated
	freeRAM := effectiveRAM - h.RAMAllocatedMB
	freeDisk := effectiveDisk - h.DiskAllocatedGB

	return freeCPU >= cpus && freeRAM >= ramMB && freeDisk >= diskGB
}

// sortCandidates sorts hosts based on the scheduling strategy.
func (s *Service) sortCandidates(candidates []models.Host, strategy string) {
	switch strategy {
	case StrategyMostAllocated:
		// Bin-packing: prefer fully-loaded hosts.
		sort.Slice(candidates, func(i, j int) bool {
			ci, cj := candidates[i], candidates[j]
			cpuI, ramI, _ := ci.GetUsagePercent()
			cpuJ, ramJ, _ := cj.GetUsagePercent()
			if cpuI != cpuJ {
				return cpuI > cpuJ // higher usage first
			}
			return ramI > ramJ
		})
	default:
		// Spread (least-allocated): prefer empty hosts.
		sort.Slice(candidates, func(i, j int) bool {
			ci, cj := candidates[i], candidates[j]
			cpuI, ramI, _ := ci.GetUsagePercent()
			cpuJ, ramJ, _ := cj.GetUsagePercent()
			if cpuI != cpuJ {
				return cpuI < cpuJ // lower usage first
			}
			return ramI < ramJ
		})
	}
}

// --- Zone Capacity APIs ---

// listZoneCapacities returns capacity summary for all zones.
func (s *Service) listZoneCapacities(c *gin.Context) {
	capacities, err := s.computeZoneCapacities()
	if err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list zone capacities"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"zones": capacities})
}

// getZoneCapacity returns capacity for a specific zone.
func (s *Service) getZoneCapacity(c *gin.Context) {
	zoneID := c.Param("zone_id")
	capacities, err := s.computeZoneCapacities()
	if err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("get zone capacity"))
		return
	}
	for _, cap := range capacities {
		if cap.ZoneID == zoneID {
			c.JSON(http.StatusOK, cap)
			return
		}
	}
	apierrors.Respond(c, apierrors.ErrNotFound("zone", zoneID))
}

// schedulerStats returns overall scheduling statistics.
func (s *Service) schedulerStats(c *gin.Context) {
	var totalHosts, activeHosts int64
	s.db.Model(&models.Host{}).Count(&totalHosts)
	s.db.Model(&models.Host{}).Where("status = ? AND resource_state = ?",
		models.HostStatusUp, models.ResourceStateEnabled).Count(&activeHosts)

	zones, _ := s.computeZoneCapacities()

	// Aggregate totals.
	var totalCPU, allocCPU int
	var totalRAM, allocRAM, totalDisk, allocDisk int64
	for _, z := range zones {
		totalCPU += z.TotalCPU
		allocCPU += z.AllocatedCPU
		totalRAM += z.TotalRAMMB
		allocRAM += z.AllocatedRAMMB
		totalDisk += z.TotalDiskGB
		allocDisk += z.AllocatedDiskGB
	}

	c.JSON(http.StatusOK, gin.H{
		"total_hosts":       totalHosts,
		"active_hosts":      activeHosts,
		"zone_count":        len(zones),
		"total_cpu":         totalCPU,
		"allocated_cpu":     allocCPU,
		"available_cpu":     totalCPU - allocCPU,
		"total_ram_mb":      totalRAM,
		"allocated_ram_mb":  allocRAM,
		"available_ram_mb":  totalRAM - allocRAM,
		"total_disk_gb":     totalDisk,
		"allocated_disk_gb": allocDisk,
		"available_disk_gb": totalDisk - allocDisk,
		"zones":             zones,
		"strategies": []string{
			StrategyLeastAllocated, StrategyMostAllocated,
			StrategyZoneAffinity, StrategyZoneRequired,
		},
	})
}

// computeZoneCapacities aggregates host data into per-zone capacity.
func (s *Service) computeZoneCapacities() ([]ZoneCapacity, error) {
	var hosts []models.Host
	if err := s.db.Find(&hosts).Error; err != nil {
		return nil, err
	}

	// Group by zone (nil zone_id -> "unassigned").
	zones := map[string]*ZoneCapacity{}
	for _, h := range hosts {
		zid := "unassigned"
		if h.ZoneID != nil && *h.ZoneID != "" {
			zid = *h.ZoneID
		}
		zc, ok := zones[zid]
		if !ok {
			zc = &ZoneCapacity{ZoneID: zid}
			zones[zid] = zc
		}
		zc.TotalHosts++
		if h.Status == models.HostStatusUp && h.ResourceState == models.ResourceStateEnabled {
			zc.ActiveHosts++
		}
		zc.TotalCPU += h.CPUCores
		zc.AllocatedCPU += h.CPUAllocated
		zc.TotalRAMMB += h.RAMMB
		zc.AllocatedRAMMB += h.RAMAllocatedMB
		zc.TotalDiskGB += h.DiskGB
		zc.AllocatedDiskGB += h.DiskAllocatedGB
	}

	// Compute derived fields.
	result := make([]ZoneCapacity, 0, len(zones))
	for _, zc := range zones {
		zc.AvailableCPU = zc.TotalCPU - zc.AllocatedCPU
		zc.AvailableRAMMB = zc.TotalRAMMB - zc.AllocatedRAMMB
		zc.AvailableDiskGB = zc.TotalDiskGB - zc.AllocatedDiskGB
		if zc.TotalCPU > 0 {
			zc.CPUUsagePercent = float64(zc.AllocatedCPU) / float64(zc.TotalCPU) * 100
		}
		if zc.TotalRAMMB > 0 {
			zc.RAMUsagePercent = float64(zc.AllocatedRAMMB) / float64(zc.TotalRAMMB) * 100
		}
		// Zone health: based on ratio of active hosts.
		switch {
		case zc.ActiveHosts == 0:
			zc.Health = ZoneDown
		case float64(zc.ActiveHosts)/float64(zc.TotalHosts) > 0.5:
			zc.Health = ZoneHealthy
		default:
			zc.Health = ZoneDegraded
		}
		result = append(result, *zc)
	}

	// Sort by zone ID for stable output.
	sort.Slice(result, func(i, j int) bool { return result[i].ZoneID < result[j].ZoneID })
	return result, nil
}

// hostToNode converts a Host model to the legacy node JSON format.
func hostToNode(h models.Host) map[string]any {
	node := map[string]any{
		"id":               h.UUID,
		"hostname":         h.Hostname,
		"address":          h.GetManagementURL(),
		"status":           h.Status,
		"resource_state":   h.ResourceState,
		"ip_address":       h.IPAddress,
		"name":             h.Name,
		"host_type":        h.HostType,
		"hypervisor_type":  h.HypervisorType,
		"cpu_cores":        h.CPUCores,
		"ram_mb":           h.RAMMB,
		"disk_gb":          h.DiskGB,
		"cpu_allocated":    h.CPUAllocated,
		"ram_allocated_mb": h.RAMAllocatedMB,
		"agent_version":    h.AgentVersion,
	}
	if h.LastHeartbeat != nil {
		node["last_heartbeat"] = *h.LastHeartbeat
	}
	if h.ZoneID != nil {
		node["zone_id"] = *h.ZoneID
	}
	if h.ClusterID != nil {
		node["cluster_id"] = *h.ClusterID
	}
	return node
}
