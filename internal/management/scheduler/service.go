package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides scheduling and VM dispatch.
// It reads host data from the persistent `hosts` table instead of
// keeping a volatile in-memory map that would be lost on restart.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

type ScheduleRequest struct {
	VCPUs     int    `json:"vcpus"`
	RAMMB     int    `json:"ram_mb"`
	DiskGB    int    `json:"disk_gb"`
	ZoneID    string `json:"zone_id"`
	ClusterID string `json:"cluster_id"`
}

type ScheduleResponse struct {
	NodeID string `json:"node"`
	Reason string `json:"reason"`
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

func (s *Service) SetupRoutes(r *gin.Engine) {
	// Health check under scheduler prefix to avoid conflicts
	r.GET("/api/scheduler/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "vc-scheduler"})
	})
	v1 := r.Group("/api/v1")
	{
		v1.POST("/schedule", s.schedule)
		v1.POST("/dispatch/vms", s.dispatchVMCreate)

		// Legacy /nodes endpoints — delegate to hosts table.
		// These exist for backwards compatibility with older compute agents.
		v1.POST("/nodes/register", s.legacyRegisterNode)
		v1.POST("/nodes/heartbeat", s.legacyHeartbeat)
		v1.GET("/nodes", s.listNodes)
		v1.GET("/nodes/:id", s.getNode)
		v1.DELETE("/nodes/:id", s.deleteNode)
	}
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

	host, reason := s.selectHost(req)
	if host == nil {
		s.logger.Warn("dispatch no hosts available", zap.String("reason", reason))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": reason})
		return
	}

	s.logger.Info("dispatch selected host",
		zap.String("uuid", host.UUID), zap.String("name", host.Name),
		zap.String("reason", reason))

	// Forward request to the selected host's VM driver.
	addr := strings.TrimRight(host.GetManagementURL(), "/") + "/api/v1/vms"
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode payload"})
		return
	}
	reqHTTP, _ := http.NewRequest("POST", addr, buf)
	reqHTTP.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(reqHTTP) // #nosec
	if err != nil {
		s.logger.Error("dispatch forward failed", zap.String("addr", addr), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "forward to node failed"})
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var upstream map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&upstream); err != nil {
		upstream = map[string]any{"error": "invalid upstream response"}
	}
	out := map[string]any{"node": host.UUID}
	for k, v := range upstream {
		out[k] = v
	}
	c.JSON(resp.StatusCode, out)
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

// schedule chooses the least-allocated host that fits the requested resources.
func (s *Service) schedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	host, reason := s.selectHost(req)
	if host == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": reason})
		return
	}
	c.JSON(http.StatusOK, ScheduleResponse{NodeID: host.UUID, Reason: reason})
}

// selectHost finds the best available host from the database.
// Criteria: status=up, resource_state=enabled, sufficient resources.
// Sorts by least-allocated (CPU, then RAM).
func (s *Service) selectHost(req ScheduleRequest) (*models.Host, string) {
	var hosts []models.Host
	query := s.db.Where("status = ? AND resource_state = ?",
		models.HostStatusUp, models.ResourceStateEnabled)

	// Zone/cluster affinity filtering.
	if req.ZoneID != "" {
		query = query.Where("zone_id = ?", req.ZoneID)
	}
	if req.ClusterID != "" {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	if err := query.Find(&hosts).Error; err != nil {
		return nil, "database error"
	}

	// Filter by resource fit.
	candidates := make([]models.Host, 0, len(hosts))
	for _, h := range hosts {
		if h.HasEnoughResources(req.VCPUs, int64(req.RAMMB), int64(req.DiskGB)) {
			candidates = append(candidates, h)
		}
	}

	if len(candidates) == 0 {
		return nil, "no hosts available with sufficient resources"
	}

	// Sort by least-allocated (CPU%, then RAM%).
	sort.Slice(candidates, func(i, j int) bool {
		ci, cj := candidates[i], candidates[j]
		cpuI, ramI, _ := ci.GetUsagePercent()
		cpuJ, ramJ, _ := cj.GetUsagePercent()
		if cpuI != cpuJ {
			return cpuI < cpuJ
		}
		return ramI < ramJ
	})

	chosen := &candidates[0]
	reason := fmt.Sprintf("least-allocated: cpu=%d/%d ram=%d/%d",
		chosen.CPUAllocated, chosen.CPUCores, chosen.RAMAllocatedMB, chosen.RAMMB)
	return chosen, reason
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
