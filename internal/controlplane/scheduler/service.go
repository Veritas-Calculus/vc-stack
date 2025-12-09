package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	mu     sync.RWMutex
	nodes  map[string]*Node
}

// Node represents a compute node registration and live metrics.
type Node struct {
	ID            string            `json:"id"` // unique name/ID
	Hostname      string            `json:"hostname"`
	Address       string            `json:"address"` // e.g., http://host:8091 for vc-lite
	Capacity      Resource          `json:"capacity"`
	Usage         Resource          `json:"usage"`
	Labels        map[string]string `json:"labels"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
}

type Resource struct {
	CPUs   int `json:"cpus"`
	RAMMB  int `json:"ram_mb"`
	DiskGB int `json:"disk_gb"`
}

type RegisterNodeRequest struct {
	ID       string            `json:"id"` // required
	Hostname string            `json:"hostname"`
	Address  string            `json:"address"` // http://host:8091
	Capacity Resource          `json:"capacity"`
	Labels   map[string]string `json:"labels"`
}

type HeartbeatRequest struct {
	ID    string   `json:"id"`
	Usage Resource `json:"usage"`
}

type ScheduleRequest struct {
	VCPUs  int               `json:"vcpus"`
	RAMMB  int               `json:"ram_mb"`
	DiskGB int               `json:"disk_gb"`
	Labels map[string]string `json:"labels"`
}

type ScheduleResponse struct {
	NodeID string `json:"node"`
	Reason string `json:"reason"`
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, nodes: make(map[string]*Node)}, nil
}

func (s *Service) SetupRoutes(r *gin.Engine) {
	// Health check under scheduler prefix to avoid conflicts
	r.GET("/api/scheduler/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "vc-scheduler"}) })
	v1 := r.Group("/api/v1")
	{
		v1.POST("/schedule", s.schedule)
		v1.POST("/dispatch/vms", s.dispatchVMCreate)
		v1.POST("/nodes/register", s.registerNode)
		v1.POST("/nodes/heartbeat", s.heartbeat)
		v1.GET("/nodes", s.listNodes)
		v1.GET("/nodes/:id", s.getNode)
		v1.DELETE("/nodes/:id", s.deleteNode)
	}
}

// dispatchVMCreate selects a node and forwards the VM create request to that node's vc-lite.
func (s *Service) dispatchVMCreate(c *gin.Context) {
	s.logger.Info("dispatch request received", zap.String("client_ip", c.ClientIP()))

	// Read raw body for forwarding and decode minimal fields for scheduling
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		s.logger.Warn("dispatch invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid create payload"})
		return
	}
	// Extract resources
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

	s.logger.Info("dispatch scheduling", zap.Int("vcpus", req.VCPUs), zap.Int("ram_mb", req.RAMMB), zap.Int("disk_gb", req.DiskGB))

	nodeID, reason := s.selectNode(req)
	if nodeID == "" {
		s.logger.Warn("dispatch no nodes available", zap.String("reason", reason))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": reason})
		return
	}

	s.logger.Info("dispatch selected node", zap.String("node_id", nodeID), zap.String("reason", reason))
	// Lookup node address
	s.mu.RLock()
	node := s.nodes[nodeID]
	s.mu.RUnlock()
	if node == nil || node.Address == "" {
		s.logger.Error("dispatch node not available", zap.String("node_id", nodeID))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "chosen node not available"})
		return
	}

	s.logger.Info("dispatch forwarding to node", zap.String("node_id", nodeID), zap.String("address", node.Address))

	// Forward request to vc-lite
	addr := strings.TrimRight(node.Address, "/") + "/api/v1/vms"
	// Re-encode payload to JSON
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		s.logger.Error("Failed to encode dispatch payload", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode payload"})
		return
	}
	reqHTTP, _ := http.NewRequest("POST", addr, buf)
	reqHTTP.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		s.logger.Error("dispatch forward failed", zap.String("addr", addr), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "forward to node failed"})
		return
	}
	defer resp.Body.Close()

	s.logger.Info("dispatch forward response", zap.String("status", resp.Status), zap.Int("status_code", resp.StatusCode))

	// Read upstream body
	var upstream map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&upstream); err != nil {
		upstream = map[string]any{"error": "invalid upstream response"}
	}
	// Compose response including chosen node
	out := map[string]any{"node": nodeID}
	for k, v := range upstream {
		out[k] = v
	}

	s.logger.Info("dispatch completed", zap.String("node_id", nodeID), zap.Int("response_code", resp.StatusCode))
	c.JSON(resp.StatusCode, out)
}

func (s *Service) registerNode(c *gin.Context) {
	var req RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &Node{ID: req.ID, Hostname: req.Hostname, Address: req.Address, Capacity: req.Capacity, Labels: req.Labels, LastHeartbeat: time.Now()}
	if n.Labels == nil {
		n.Labels = map[string]string{}
	}
	s.nodes[req.ID] = n
	s.logger.Info("node registered", zap.String("id", req.ID), zap.String("addr", req.Address))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.nodes[req.ID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	n.Usage = req.Usage
	n.LastHeartbeat = time.Now()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) listNodes(c *gin.Context) {
	s.mu.RLock()
	list := make([]*Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		list = append(list, n)
	}
	s.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"nodes": list})
}

// getNode retrieves a single node by ID.
func (s *Service) getNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	s.mu.RLock()
	node, ok := s.nodes[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": node})
}

// deleteNode removes a node from the scheduler registry.
func (s *Service) deleteNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	delete(s.nodes, id)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// schedule chooses the least-allocated node that fits the requested resources.
func (s *Service) schedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	nodeID, reason := s.selectNode(req)
	if nodeID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": reason})
		return
	}
	c.JSON(http.StatusOK, ScheduleResponse{NodeID: nodeID, Reason: reason})
}

func (s *Service) selectNode(req ScheduleRequest) (nodeID, reason string) {
	s.mu.RLock()
	candidates := make([]*Node, 0, len(s.nodes))
	now := time.Now()
	for _, n := range s.nodes {
		// simple liveness: heartbeat within 1m
		if now.Sub(n.LastHeartbeat) > time.Minute {
			continue
		}
		// fit check
		freeCPU := n.Capacity.CPUs - n.Usage.CPUs
		freeRAM := n.Capacity.RAMMB - n.Usage.RAMMB
		freeDisk := n.Capacity.DiskGB - n.Usage.DiskGB
		if freeCPU < req.VCPUs || freeRAM < req.RAMMB || freeDisk < req.DiskGB {
			continue
		}
		candidates = append(candidates, n)
	}
	s.mu.RUnlock()
	if len(candidates) == 0 {
		return "", "no nodes available"
	}
	// sort by least-allocated (CPU, then RAM)
	sort.Slice(candidates, func(i, j int) bool {
		ni, nj := candidates[i], candidates[j]
		uCPU := ni.Usage.CPUs * 1000 / maxInt(1, ni.Capacity.CPUs)
		vCPU := nj.Usage.CPUs * 1000 / maxInt(1, nj.Capacity.CPUs)
		if uCPU != vCPU {
			return uCPU < vCPU
		}
		uRAM := ni.Usage.RAMMB * 1000 / maxInt(1, ni.Capacity.RAMMB)
		vRAM := nj.Usage.RAMMB * 1000 / maxInt(1, nj.Capacity.RAMMB)
		return uRAM < vRAM
	})
	chosen := candidates[0]
	nodeID = chosen.ID
	reason = fmt.Sprintf("least-allocated: cpu=%d/%d ram=%d/%d", chosen.Usage.CPUs, chosen.Capacity.CPUs, chosen.Usage.RAMMB, chosen.Capacity.RAMMB)
	return nodeID, reason
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
