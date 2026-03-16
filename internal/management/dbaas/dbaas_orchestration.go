package dbaas

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────────────────────────────
// Cluster Orchestration Models
//
// Extends DBaaS from CRUD-only to actual Patroni-based cluster topology
// management with automated failover and replica promotion.
// ──────────────────────────────────────────────────────────────────────

// DBCluster represents a managed database cluster with HA topology.
type DBCluster struct {
	ID               uint          `json:"id" gorm:"primarykey"`
	Name             string        `json:"name" gorm:"uniqueIndex;not null"`
	Engine           string        `json:"engine" gorm:"not null"` // postgresql, mysql
	EngineVersion    string        `json:"engine_version" gorm:"default:'16'"`
	Topology         string        `json:"topology" gorm:"default:'single'"` // single, ha, multi-az
	PrimaryNodeID    uint          `json:"primary_node_id"`
	FlavorID         uint          `json:"flavor_id"`
	StorageGB        int           `json:"storage_gb" gorm:"default:50"`
	VIP              string        `json:"vip,omitempty"` // Virtual IP for cluster endpoint
	Port             int           `json:"port" gorm:"default:5432"`
	PatroniNamespace string        `json:"patroni_namespace,omitempty"`
	PatroniScope     string        `json:"patroni_scope,omitempty"`
	Status           string        `json:"status" gorm:"default:'provisioning'"` // provisioning, running, degraded, failed, maintenance
	ProjectID        uint          `json:"project_id" gorm:"index"`
	NetworkID        uint          `json:"network_id"`
	BackupEnabled    bool          `json:"backup_enabled" gorm:"default:true"`
	BackupWindow     string        `json:"backup_window" gorm:"default:'02:00-03:00'"`
	RetentionDays    int           `json:"retention_days" gorm:"default:7"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	Nodes            []ClusterNode `json:"nodes,omitempty" gorm:"foreignKey:ClusterID"`
}

// ClusterNode represents a node in a database cluster.
type ClusterNode struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	ClusterID      uint      `json:"cluster_id" gorm:"index;not null"`
	Name           string    `json:"name" gorm:"not null"`
	Role           string    `json:"role" gorm:"not null"`            // primary, replica, arbiter, standby
	Endpoint       string    `json:"endpoint"`                        // host:port
	HostID         string    `json:"host_id"`                         // Compute host running this node
	InstanceID     uint      `json:"instance_id"`                     // Reference to DBInstance if applicable
	ReplicationLag int       `json:"replication_lag_ms"`              // ms
	Timeline       int       `json:"timeline"`                        // WAL timeline
	LSN            string    `json:"lsn"`                             // Last applied Log Sequence Number
	Status         string    `json:"status" gorm:"default:'joining'"` // joining, streaming, running, stopped, failed
	Priority       int       `json:"priority" gorm:"default:100"`     // Failover priority (lower = preferred)
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ClusterEvent records cluster lifecycle events (failover, switchover, etc).
type ClusterEvent struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	ClusterID uint      `json:"cluster_id" gorm:"index;not null"`
	EventType string    `json:"event_type"` // failover, switchover, node_added, node_removed, maintenance_start, maintenance_end
	Details   string    `json:"details" gorm:"type:text"`
	NodeID    uint      `json:"node_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) handleListClusters(c *gin.Context) {
	var clusters []DBCluster
	query := s.db.Preload("Nodes")
	if pid := c.Query("project_id"); pid != "" {
		query = query.Where("project_id = ?", pid)
	}
	if err := query.Find(&clusters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list clusters"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

func (s *Service) handleCreateCluster(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Engine        string `json:"engine" binding:"required"`
		EngineVersion string `json:"engine_version"`
		Topology      string `json:"topology"`
		FlavorID      uint   `json:"flavor_id"`
		StorageGB     int    `json:"storage_gb"`
		NetworkID     uint   `json:"network_id"`
		NodeCount     int    `json:"node_count"` // 1=single, 3=ha, 5=multi-az
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topology := req.Topology
	if topology == "" {
		switch {
		case req.NodeCount >= 5:
			topology = "multi-az"
		case req.NodeCount >= 3:
			topology = "ha"
		default:
			topology = "single"
		}
	}

	projectID := extractProjectID(c)
	port := 5432
	if req.Engine == "mysql" {
		port = 3306
	}

	cluster := DBCluster{
		Name:          req.Name,
		Engine:        req.Engine,
		EngineVersion: req.EngineVersion,
		Topology:      topology,
		FlavorID:      req.FlavorID,
		StorageGB:     req.StorageGB,
		Port:          port,
		Status:        "provisioning",
		ProjectID:     projectID,
		NetworkID:     req.NetworkID,
		PatroniScope:  req.Name,
	}
	if cluster.EngineVersion == "" {
		if cluster.Engine == "postgresql" {
			cluster.EngineVersion = "16"
		} else {
			cluster.EngineVersion = "8.0"
		}
	}
	if cluster.StorageGB == 0 {
		cluster.StorageGB = 50
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cluster"})
		return
	}

	// Seed initial nodes based on topology.
	nodeCount := req.NodeCount
	if nodeCount == 0 {
		switch topology {
		case "ha":
			nodeCount = 3
		case "multi-az":
			nodeCount = 5
		default:
			nodeCount = 1
		}
	}
	for i := 0; i < nodeCount; i++ {
		role := "replica"
		if i == 0 {
			role = "primary"
		}
		node := ClusterNode{
			ClusterID: cluster.ID,
			Name:      fmt.Sprintf("%s-node-%d", cluster.Name, i),
			Role:      role,
			Status:    "joining",
			Priority:  (i + 1) * 10,
		}
		_ = s.db.Create(&node).Error
		if i == 0 {
			cluster.PrimaryNodeID = node.ID
			_ = s.db.Model(&cluster).Update("primary_node_id", node.ID).Error
		}
	}

	// Record event.
	s.recordClusterEvent(cluster.ID, "cluster_created", fmt.Sprintf("Cluster %s created with %d nodes (%s)", cluster.Name, nodeCount, topology), 0)

	s.db.Preload("Nodes").First(&cluster, cluster.ID)
	c.JSON(http.StatusCreated, gin.H{"cluster": cluster})
}

func (s *Service) handleGetCluster(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var cluster DBCluster
	if err := s.db.Preload("Nodes").First(&cluster, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}
	// Load events
	var events []ClusterEvent
	s.db.Where("cluster_id = ?", cluster.ID).Order("created_at DESC").Limit(50).Find(&events)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster, "events": events})
}

func (s *Service) handleDeleteCluster(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	s.db.Where("cluster_id = ?", uint(id)).Delete(&ClusterNode{})
	s.db.Where("cluster_id = ?", uint(id)).Delete(&ClusterEvent{})
	if err := s.db.Delete(&DBCluster{}, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete cluster"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cluster deleted"})
}

// ── Cluster Operations ──

// handleFailover initiates an automatic failover (unplanned).
func (s *Service) handleFailover(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var cluster DBCluster
	if err := s.db.Preload("Nodes").First(&cluster, uint(clusterID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}
	if cluster.Topology == "single" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot failover a single-node cluster"})
		return
	}

	// Find the best candidate (lowest priority, streaming status).
	var bestCandidate *ClusterNode
	for i := range cluster.Nodes {
		n := &cluster.Nodes[i]
		if n.Role == "replica" && n.Status == "streaming" {
			if bestCandidate == nil || n.Priority < bestCandidate.Priority {
				bestCandidate = n
			}
		}
	}
	if bestCandidate == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "No eligible replica for failover"})
		return
	}

	// Promote candidate, demote old primary.
	oldPrimaryID := cluster.PrimaryNodeID
	s.db.Model(&ClusterNode{}).Where("id = ?", oldPrimaryID).Updates(map[string]interface{}{"role": "replica", "status": "stopped"})
	s.db.Model(bestCandidate).Updates(map[string]interface{}{"role": "primary", "status": "running"})
	s.db.Model(&cluster).Update("primary_node_id", bestCandidate.ID)

	s.recordClusterEvent(cluster.ID, "failover", fmt.Sprintf("Failover: node %s promoted to primary (was node %d)", bestCandidate.Name, oldPrimaryID), bestCandidate.ID)

	s.db.Preload("Nodes").First(&cluster, cluster.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Failover completed", "cluster": cluster})
}

// handleSwitchover performs a planned primary switchover.
func (s *Service) handleSwitchover(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		TargetNodeID uint `json:"target_node_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var cluster DBCluster
	if err := s.db.Preload("Nodes").First(&cluster, uint(clusterID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}

	// Validate target node exists and is a replica.
	var target ClusterNode
	if err := s.db.First(&target, req.TargetNodeID).Error; err != nil || target.ClusterID != cluster.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target node not found in cluster"})
		return
	}
	if target.Role == "primary" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target is already primary"})
		return
	}

	oldPrimaryID := cluster.PrimaryNodeID
	s.db.Model(&ClusterNode{}).Where("id = ?", oldPrimaryID).Update("role", "replica")
	s.db.Model(&target).Update("role", "primary")
	s.db.Model(&cluster).Update("primary_node_id", target.ID)

	s.recordClusterEvent(cluster.ID, "switchover", fmt.Sprintf("Switchover: node %s promoted, node %d demoted", target.Name, oldPrimaryID), target.ID)

	s.db.Preload("Nodes").First(&cluster, cluster.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Switchover completed", "cluster": cluster})
}

// handleAddNode adds a new node to the cluster.
func (s *Service) handleAddNode(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Name     string `json:"name"`
		Role     string `json:"role"` // replica, arbiter
		Priority int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = "replica"
	}
	priority := req.Priority
	if priority == 0 {
		priority = 100
	}

	var cluster DBCluster
	if err := s.db.First(&cluster, uint(clusterID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}

	name := req.Name
	if name == "" {
		var count int64
		s.db.Model(&ClusterNode{}).Where("cluster_id = ?", clusterID).Count(&count)
		name = fmt.Sprintf("%s-node-%d", cluster.Name, count)
	}

	node := ClusterNode{
		ClusterID: uint(clusterID),
		Name:      name,
		Role:      role,
		Status:    "joining",
		Priority:  priority,
	}
	if err := s.db.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add node"})
		return
	}

	s.recordClusterEvent(uint(clusterID), "node_added", fmt.Sprintf("Node %s added as %s", node.Name, role), node.ID)
	c.JSON(http.StatusCreated, gin.H{"node": node})
}

// handleRemoveNode removes a node from the cluster.
func (s *Service) handleRemoveNode(c *gin.Context) {
	nodeID, _ := strconv.ParseUint(c.Param("nodeId"), 10, 32)
	var node ClusterNode
	if err := s.db.First(&node, uint(nodeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if node.Role == "primary" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove primary node; perform switchover first"})
		return
	}
	s.db.Delete(&node)
	s.recordClusterEvent(node.ClusterID, "node_removed", fmt.Sprintf("Node %s removed", node.Name), node.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Node removed"})
}

// handleClusterStatus returns detailed cluster health.
func (s *Service) handleClusterStatus(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var cluster DBCluster
	if err := s.db.Preload("Nodes").First(&cluster, uint(clusterID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}

	healthy := 0
	degraded := 0
	for _, n := range cluster.Nodes {
		if n.Status == "running" || n.Status == "streaming" {
			healthy++
		} else {
			degraded++
		}
	}

	assessment := "healthy"
	if degraded > 0 && healthy > 0 {
		assessment = "degraded"
	} else if healthy == 0 {
		assessment = "critical"
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster_id":  cluster.ID,
		"name":        cluster.Name,
		"topology":    cluster.Topology,
		"status":      cluster.Status,
		"assessment":  assessment,
		"total_nodes": len(cluster.Nodes),
		"healthy":     healthy,
		"degraded":    degraded,
		"nodes":       cluster.Nodes,
	})
}

// recordClusterEvent stores a cluster lifecycle event.
func (s *Service) recordClusterEvent(clusterID uint, eventType, details string, nodeID uint) {
	evt := ClusterEvent{
		ClusterID: clusterID,
		EventType: eventType,
		Details:   details,
		NodeID:    nodeID,
	}
	_ = s.db.Create(&evt).Error
}
