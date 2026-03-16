package redis

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RedisCluster represents a managed Redis cluster.
type RedisCluster struct {
	ID             uint        `json:"id" gorm:"primarykey"`
	Name           string      `json:"name" gorm:"uniqueIndex;not null"`
	Mode           string      `json:"mode" gorm:"default:'standalone'"` // standalone, sentinel, cluster
	Version        string      `json:"version" gorm:"default:'7.2'"`
	MemoryMB       int         `json:"memory_mb" gorm:"default:256"`
	MaxClients     int         `json:"max_clients" gorm:"default:10000"`
	Password       string      `json:"-" gorm:""`
	Endpoint       string      `json:"endpoint,omitempty"`
	Port           int         `json:"port" gorm:"default:6379"`
	ProjectID      uint        `json:"project_id" gorm:"index"`
	NetworkID      uint        `json:"network_id"`
	Status         string      `json:"status" gorm:"default:'provisioning'"`
	Persistence    string      `json:"persistence" gorm:"default:'rdb'"` // none, rdb, aof, rdb+aof
	EvictionPolicy string      `json:"eviction_policy" gorm:"default:'allkeys-lru'"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	Nodes          []RedisNode `json:"nodes,omitempty" gorm:"foreignKey:ClusterID"`
}

// RedisNode represents a node in a Redis cluster.
type RedisNode struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	ClusterID  uint      `json:"cluster_id" gorm:"index;not null"`
	Name       string    `json:"name"`
	Role       string    `json:"role" gorm:"not null"` // master, replica, sentinel
	Endpoint   string    `json:"endpoint"`
	SlotStart  int       `json:"slot_start,omitempty"` // For cluster mode: 0-16383
	SlotEnd    int       `json:"slot_end,omitempty"`
	MasterID   uint      `json:"master_id,omitempty"` // For replicas
	Status     string    `json:"status" gorm:"default:'joining'"`
	MemoryUsed int64     `json:"memory_used_bytes"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── Handlers ──

func (s *Service) handleListClusters(c *gin.Context) {
	var clusters []RedisCluster
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
		Name        string `json:"name" binding:"required"`
		Mode        string `json:"mode"`
		Version     string `json:"version"`
		MemoryMB    int    `json:"memory_mb"`
		NetworkID   uint   `json:"network_id"`
		NodeCount   int    `json:"node_count"`
		Persistence string `json:"persistence"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "standalone"
	}
	version := req.Version
	if version == "" {
		version = "7.2"
	}
	mem := req.MemoryMB
	if mem == 0 {
		mem = 256
	}

	projectID := uint(0)
	if pid := c.Query("project_id"); pid != "" {
		v, _ := strconv.ParseUint(pid, 10, 32)
		projectID = uint(v)
	}

	cluster := RedisCluster{
		Name:        req.Name,
		Mode:        mode,
		Version:     version,
		MemoryMB:    mem,
		NetworkID:   req.NetworkID,
		ProjectID:   projectID,
		Persistence: req.Persistence,
		Status:      "provisioning",
	}
	if cluster.Persistence == "" {
		cluster.Persistence = "rdb"
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cluster"})
		return
	}

	// Seed nodes.
	nodeCount := req.NodeCount
	if nodeCount == 0 {
		switch mode {
		case "sentinel":
			nodeCount = 3
		case "cluster":
			nodeCount = 6
		default:
			nodeCount = 1
		}
	}

	slotsPerShard := 0
	masterCount := nodeCount
	if mode == "cluster" {
		masterCount = nodeCount / 2
		if masterCount == 0 {
			masterCount = 3
		}
		slotsPerShard = 16384 / masterCount
	}

	masterIdx := 0
	for i := 0; i < nodeCount; i++ {
		role := "master"
		slotStart, slotEnd := 0, 0
		var masterRef uint

		switch mode {
		case "sentinel":
			if i == 0 {
				role = "master"
			} else if i <= 1 {
				role = "replica"
			} else {
				role = "sentinel"
			}
		case "cluster":
			if i < masterCount {
				role = "master"
				slotStart = i * slotsPerShard
				slotEnd = (i+1)*slotsPerShard - 1
				if i == masterCount-1 {
					slotEnd = 16383
				}
			} else {
				role = "replica"
				masterRef = cluster.Nodes[masterIdx].ID
				masterIdx++
				if masterIdx >= masterCount {
					masterIdx = 0
				}
			}
		}

		node := RedisNode{
			ClusterID: cluster.ID,
			Name:      fmt.Sprintf("%s-node-%d", cluster.Name, i),
			Role:      role,
			SlotStart: slotStart,
			SlotEnd:   slotEnd,
			MasterID:  masterRef,
			Status:    "joining",
		}
		s.db.Create(&node)
		cluster.Nodes = append(cluster.Nodes, node)
	}

	c.JSON(http.StatusCreated, gin.H{"cluster": cluster})
}

func (s *Service) handleGetCluster(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("clusterId"), 10, 32)
	var cluster RedisCluster
	if err := s.db.Preload("Nodes").First(&cluster, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cluster": cluster})
}

func (s *Service) handleDeleteCluster(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("clusterId"), 10, 32)
	s.db.Where("cluster_id = ?", uint(id)).Delete(&RedisNode{})
	s.db.Delete(&RedisCluster{}, uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "Cluster deleted"})
}

func (s *Service) handleFailover(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("clusterId"), 10, 32)
	var cluster RedisCluster
	if err := s.db.Preload("Nodes").First(&cluster, uint(clusterID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}
	if cluster.Mode == "standalone" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot failover standalone instance"})
		return
	}
	// Promote first available replica.
	for _, n := range cluster.Nodes {
		if n.Role == "replica" && n.Status != "failed" {
			s.db.Model(&RedisNode{}).Where("cluster_id = ? AND role = 'master'", clusterID).Update("role", "replica")
			s.db.Model(&n).Update("role", "master")
			s.logger.Info("Redis failover completed", zap.String("promoted", n.Name))
			c.JSON(http.StatusOK, gin.H{"message": "Failover completed", "promoted_node": n.Name})
			return
		}
	}
	c.JSON(http.StatusConflict, gin.H{"error": "No eligible replica for failover"})
}

func (s *Service) handleRebalance(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("clusterId"), 10, 32)
	var nodes []RedisNode
	s.db.Where("cluster_id = ? AND role = 'master'", uint(clusterID)).Find(&nodes)
	if len(nodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No master nodes found"})
		return
	}
	slotsPerNode := 16384 / len(nodes)
	for i, n := range nodes {
		n.SlotStart = i * slotsPerNode
		n.SlotEnd = (i+1)*slotsPerNode - 1
		if i == len(nodes)-1 {
			n.SlotEnd = 16383
		}
		s.db.Save(&n)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Slots rebalanced", "master_count": len(nodes)})
}
