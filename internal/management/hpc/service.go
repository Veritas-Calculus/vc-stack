package hpc

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// PermissionChecker is the subset of identity.Service needed for RBAC.
type PermissionChecker interface {
	RequirePermission(resource, action string) gin.HandlerFunc
}

// Config holds the dependencies for the HPC service.
type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string
	Identity  PermissionChecker
}

// Service implements the HPC management module.
type Service struct {
	db           *gorm.DB
	logger       *zap.Logger
	jwtSecret    string
	identity     PermissionChecker
	orchestrator *K8sOrchestrator
	slurmClient  *SlurmClient
}

// NewService creates a new HPC service and runs DB migrations.
func NewService(cfg Config) (*Service, error) {
	s := &Service{
		db:           cfg.DB,
		logger:       cfg.Logger,
		jwtSecret:    cfg.JWTSecret,
		identity:     cfg.Identity,
		orchestrator: NewK8sOrchestrator(cfg.Logger.Named("k8s-orchestrator")),
		slurmClient:  NewSlurmClient(cfg.Logger.Named("slurm-client")),
	}

	if err := cfg.DB.AutoMigrate(
		&HPCKubernetesCluster{},
		&HPCKubernetesNode{},
		&HPCGPUPool{},
		&SlurmCluster{},
		&SlurmNode{},
		&SlurmPartition{},
		&HPCJob{},
		&SlurmUserMapping{},
	); err != nil {
		return nil, fmt.Errorf("hpc: migrate: %w", err)
	}

	s.seedHPCRoles()
	s.logger.Info("HPC service initialized")
	return s, nil
}

// Name implements the Module interface.
func (s *Service) Name() string { return "hpc" }

// SetupRoutes registers all HPC API routes with authentication and RBAC.
func (s *Service) SetupRoutes(router *gin.Engine) {
	hpc := router.Group("/api/v1/hpc")

	// Global auth middleware for all HPC routes (skipped when jwtSecret is empty, e.g. in tests).
	if s.jwtSecret != "" {
		hpc.Use(
			middleware.AuthMiddleware(s.jwtSecret, s.logger),
		)
	}

	// Status endpoint (any authenticated user).
	hpc.GET("/status", s.getStatus)

	// ── HPC Kubernetes Clusters ───────────────────────────────────
	k8s := hpc.Group("/kubernetes")
	{
		k8s.GET("/clusters",
			s.identity.RequirePermission("hpc_cluster", "list"),
			s.listK8sClusters)
		k8s.POST("/clusters",
			s.identity.RequirePermission("hpc_cluster", "create"),
			s.createK8sCluster)
		k8s.GET("/clusters/:id",
			s.identity.RequirePermission("hpc_cluster", "get"),
			s.getK8sCluster)
		k8s.PUT("/clusters/:id",
			s.identity.RequirePermission("hpc_cluster", "update"),
			s.updateK8sCluster)
		k8s.DELETE("/clusters/:id",
			s.identity.RequirePermission("hpc_cluster", "delete"),
			s.deleteK8sCluster)
		k8s.POST("/clusters/:id/scale",
			s.identity.RequirePermission("hpc_cluster", "scale"),
			s.scaleK8sCluster)

		// GPU Pools
		k8s.GET("/clusters/:id/gpu-pools",
			s.identity.RequirePermission("hpc_gpu", "list"),
			s.listGPUPools)
		k8s.POST("/clusters/:id/gpu-pools",
			s.identity.RequirePermission("hpc_gpu", "create"),
			s.createGPUPool)
		k8s.DELETE("/clusters/:id/gpu-pools/:poolId",
			s.identity.RequirePermission("hpc_gpu", "delete"),
			s.deleteGPUPool)

		// Nodes
		k8s.GET("/clusters/:id/nodes",
			s.identity.RequirePermission("hpc_cluster", "get"),
			s.listK8sNodes)
		k8s.POST("/clusters/:id/nodes",
			s.identity.RequirePermission("hpc_cluster", "scale"),
			s.addK8sNode)
		k8s.DELETE("/clusters/:id/nodes/:nodeId",
			s.identity.RequirePermission("hpc_cluster", "scale"),
			s.removeK8sNode)

		// HPC Components (P1)
		k8s.GET("/clusters/:id/components",
			s.identity.RequirePermission("hpc_cluster", "get"),
			s.getClusterComponents)
		k8s.POST("/clusters/:id/reconcile",
			s.identity.RequirePermission("hpc_cluster", "update"),
			s.reconcileCluster)

		// GPU Topology (P1)
		k8s.GET("/clusters/:id/nodes/:nodeId/gpu-topology",
			s.identity.RequirePermission("hpc_gpu", "get"),
			s.getNodeGPUTopology)
	}

	// ── Slurm Clusters ────────────────────────────────────────────
	slurm := hpc.Group("/slurm")
	{
		slurm.GET("/clusters",
			s.identity.RequirePermission("slurm_cluster", "list"),
			s.listSlurmClusters)
		slurm.POST("/clusters",
			s.identity.RequirePermission("slurm_cluster", "create"),
			s.createSlurmCluster)
		slurm.GET("/clusters/:id",
			s.identity.RequirePermission("slurm_cluster", "get"),
			s.getSlurmCluster)
		slurm.DELETE("/clusters/:id",
			s.identity.RequirePermission("slurm_cluster", "delete"),
			s.deleteSlurmCluster)

		// Partitions
		slurm.GET("/clusters/:id/partitions",
			s.identity.RequirePermission("slurm_partition", "list"),
			s.listPartitions)
		slurm.POST("/clusters/:id/partitions",
			s.identity.RequirePermission("slurm_partition", "create"),
			s.createPartition)
		slurm.DELETE("/clusters/:id/partitions/:partId",
			s.identity.RequirePermission("slurm_partition", "delete"),
			s.deletePartition)

		// Slurm User Sync
		slurm.GET("/clusters/:id/users",
			s.identity.RequirePermission("slurm_user", "list"),
			s.listSlurmUsers)
		slurm.POST("/clusters/:id/users",
			s.identity.RequirePermission("slurm_user", "create"),
			s.syncSlurmUser)

		// P2: User Sync Pipeline
		slurm.POST("/clusters/:id/sync-users",
			s.identity.RequirePermission("slurm_user", "create"),
			s.executeSlurmUserSync)

		// P2: Slurm Node State
		slurm.GET("/clusters/:id/nodes",
			s.identity.RequirePermission("slurm_cluster", "get"),
			s.listSlurmNodes)
		slurm.POST("/clusters/:id/nodes",
			s.identity.RequirePermission("slurm_cluster", "update"),
			s.addSlurmNode)
		slurm.DELETE("/clusters/:id/users/:mappingId",
			s.identity.RequirePermission("slurm_user", "delete"),
			s.removeSlurmUser)
	}

	// ── Unified Jobs ──────────────────────────────────────────────
	jobs := hpc.Group("/jobs")
	{
		jobs.GET("",
			s.identity.RequirePermission("hpc_job", "list"),
			s.listJobs)
		jobs.POST("",
			s.identity.RequirePermission("hpc_job", "create"),
			s.submitJob)
		jobs.GET("/:id",
			s.identity.RequirePermission("hpc_job", "get"),
			s.getJob)
		jobs.DELETE("/:id",
			s.identity.RequirePermission("hpc_job", "delete"),
			s.cancelJob)
		jobs.GET("/:id/manifest",
			s.identity.RequirePermission("hpc_job", "get"),
			s.getJobManifest)
		jobs.GET("/:id/sbatch",
			s.identity.RequirePermission("hpc_job", "get"),
			s.getSlurmSubmitScript)
	}
}

// ──────────────────────────────────────────────────────────────────
// Status
// ──────────────────────────────────────────────────────────────────

func (s *Service) getStatus(c *gin.Context) {
	var k8sCount, slurmCount, jobCount, gpuPoolCount int64
	s.db.Model(&HPCKubernetesCluster{}).Count(&k8sCount)
	s.db.Model(&SlurmCluster{}).Count(&slurmCount)
	s.db.Model(&HPCJob{}).Count(&jobCount)
	s.db.Model(&HPCGPUPool{}).Count(&gpuPoolCount)

	var activeJobs int64
	s.db.Model(&HPCJob{}).Where("status IN ?", []string{"running", "queued"}).Count(&activeJobs)

	var totalGPUs int64
	s.db.Model(&HPCGPUPool{}).Select("COALESCE(SUM(gpu_count), 0)").Scan(&totalGPUs)

	c.JSON(http.StatusOK, gin.H{
		"status":              "operational",
		"kubernetes_clusters": k8sCount,
		"slurm_clusters":      slurmCount,
		"total_jobs":          jobCount,
		"active_jobs":         activeJobs,
		"gpu_pools":           gpuPoolCount,
		"total_gpus":          totalGPUs,
	})
}

// ──────────────────────────────────────────────────────────────────
// HPC Kubernetes Cluster Handlers
// ──────────────────────────────────────────────────────────────────

func (s *Service) listK8sClusters(c *gin.Context) {
	var clusters []HPCKubernetesCluster
	query := s.db.Order("created_at DESC")
	if projectID := c.GetString("project_id"); projectID != "" {
		if c.Query("all_projects") == "true" {
			if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
				// admin: no project filter
			} else {
				query = query.Where("project_id = ?", projectID)
			}
		} else {
			query = query.Where("project_id = ?", projectID)
		}
	}
	query.Find(&clusters)
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

func (s *Service) createK8sCluster(c *gin.Context) {
	var req struct {
		Name               string `json:"name" binding:"required"`
		Description        string `json:"description"`
		KubernetesVersion  string `json:"kubernetes_version"`
		NetworkID          string `json:"network_id"`
		SubnetID           string `json:"subnet_id"`
		ControlPlaneCount  int    `json:"control_plane_count"`
		WorkerCount        int    `json:"worker_count"`
		ControlPlaneFlavor string `json:"control_plane_flavor"`
		WorkerFlavor       string `json:"worker_flavor"`
		HAEnabled          bool   `json:"ha_enabled"`
		GPUScheduler       string `json:"gpu_scheduler"`
		EnableMPI          bool   `json:"enable_mpi"`
		EnableRDMA         bool   `json:"enable_rdma"`
		SharedFSType       string `json:"shared_fs_type"`
		SharedFSEndpoint   string `json:"shared_fs_endpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Defaults
	if req.KubernetesVersion == "" {
		req.KubernetesVersion = "1.30"
	}
	if req.ControlPlaneCount == 0 {
		req.ControlPlaneCount = 1
	}
	if req.WorkerCount == 0 {
		req.WorkerCount = 3
	}
	if req.HAEnabled && req.ControlPlaneCount < 3 {
		req.ControlPlaneCount = 3
	}
	if req.GPUScheduler == "" {
		req.GPUScheduler = "volcano"
	}

	projectID := c.GetString("project_id")

	cluster := HPCKubernetesCluster{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		ProjectID:          projectID,
		Status:             "provisioning",
		KubernetesVersion:  req.KubernetesVersion,
		NetworkID:          req.NetworkID,
		SubnetID:           req.SubnetID,
		ControlPlaneCount:  req.ControlPlaneCount,
		WorkerCount:        req.WorkerCount,
		ControlPlaneFlavor: defaultStr(req.ControlPlaneFlavor, "m1.medium"),
		WorkerFlavor:       defaultStr(req.WorkerFlavor, "m1.xlarge"),
		HAEnabled:          req.HAEnabled,
		GPUScheduler:       req.GPUScheduler,
		EnableMPI:          req.EnableMPI,
		EnableRDMA:         req.EnableRDMA,
		SharedFSType:       req.SharedFSType,
		SharedFSEndpoint:   req.SharedFSEndpoint,
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "cluster name already exists in this project"})
		return
	}

	// Simulate provisioning (placeholder for actual orchestration).
	s.provisionK8sNodes(&cluster)

	s.db.First(&cluster, "id = ?", cluster.ID)
	s.logger.Info("HPC K8s cluster created",
		zap.String("cluster_id", cluster.ID),
		zap.String("name", cluster.Name),
		zap.String("gpu_scheduler", cluster.GPUScheduler))

	c.JSON(http.StatusCreated, gin.H{"cluster": cluster})
}

func (s *Service) provisionK8sNodes(cluster *HPCKubernetesCluster) {
	for i := 0; i < cluster.ControlPlaneCount; i++ {
		node := HPCKubernetesNode{
			ID:         uuid.New().String(),
			ClusterID:  cluster.ID,
			Name:       fmt.Sprintf("%s-cp-%d", cluster.Name, i+1),
			Role:       "control-plane",
			IPAddress:  fmt.Sprintf("10.0.1.%d", 10+i),
			FlavorName: cluster.ControlPlaneFlavor,
			Status:     "ready",
			K8sVersion: cluster.KubernetesVersion,
		}
		_ = s.db.Create(&node)
	}
	for i := 0; i < cluster.WorkerCount; i++ {
		node := HPCKubernetesNode{
			ID:         uuid.New().String(),
			ClusterID:  cluster.ID,
			Name:       fmt.Sprintf("%s-gpu-worker-%d", cluster.Name, i+1),
			Role:       "gpu-worker",
			IPAddress:  fmt.Sprintf("10.0.1.%d", 20+i),
			FlavorName: cluster.WorkerFlavor,
			Status:     "ready",
			K8sVersion: cluster.KubernetesVersion,
		}
		_ = s.db.Create(&node)
	}

	apiIP := "10.0.1.10"
	if cluster.HAEnabled {
		apiIP = "10.0.1.100"
	}
	s.db.Model(cluster).Updates(map[string]interface{}{
		"status":       "active",
		"api_endpoint": fmt.Sprintf("https://%s:6443", apiIP),
	})
}

func (s *Service) getK8sCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var nodes []HPCKubernetesNode
	s.db.Where("cluster_id = ?", id).Find(&nodes)
	var gpuPools []HPCGPUPool
	s.db.Where("cluster_id = ?", id).Find(&gpuPools)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster, "nodes": nodes, "gpu_pools": gpuPools})
}

func (s *Service) updateK8sCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var req struct {
		Description      string `json:"description"`
		GPUScheduler     string `json:"gpu_scheduler"`
		EnableMPI        bool   `json:"enable_mpi"`
		EnableRDMA       bool   `json:"enable_rdma"`
		SharedFSType     string `json:"shared_fs_type"`
		SharedFSEndpoint string `json:"shared_fs_endpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.GPUScheduler != "" {
		updates["gpu_scheduler"] = req.GPUScheduler
	}
	updates["enable_mpi"] = req.EnableMPI
	updates["enable_rdma"] = req.EnableRDMA
	if req.SharedFSType != "" {
		updates["shared_fs_type"] = req.SharedFSType
	}
	if req.SharedFSEndpoint != "" {
		updates["shared_fs_endpoint"] = req.SharedFSEndpoint
	}
	s.db.Model(&cluster).Updates(updates)
	s.db.First(&cluster, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster})
}

func (s *Service) deleteK8sCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	// Cascade delete.
	s.db.Where("cluster_id = ?", id).Delete(&HPCKubernetesNode{})
	s.db.Where("cluster_id = ?", id).Delete(&HPCGPUPool{})
	s.db.Delete(&cluster)
	s.logger.Info("HPC K8s cluster deleted", zap.String("cluster_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "cluster deleted"})
}

func (s *Service) scaleK8sCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var req struct {
		WorkerCount int    `json:"worker_count" binding:"required"`
		GPUType     string `json:"gpu_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.WorkerCount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker_count must be >= 0"})
		return
	}
	s.db.Model(&cluster).Update("worker_count", req.WorkerCount)
	s.db.First(&cluster, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster, "message": "scale operation initiated"})
}

// ── HPC K8s Nodes ────────────────────────────────────────────────

func (s *Service) listK8sNodes(c *gin.Context) {
	var nodes []HPCKubernetesNode
	s.db.Where("cluster_id = ?", c.Param("id")).Order("role, name").Find(&nodes)
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (s *Service) addK8sNode(c *gin.Context) {
	clusterID := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var req struct {
		Count    int    `json:"count"`
		Role     string `json:"role"`
		Flavor   string `json:"flavor"`
		GPUCount int    `json:"gpu_count"`
		GPUType  string `json:"gpu_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Count == 0 {
		req.Count = 1
	}
	if req.Role == "" {
		req.Role = "gpu-worker"
	}

	var existingCount int64
	s.db.Model(&HPCKubernetesNode{}).Where("cluster_id = ?", clusterID).Count(&existingCount)

	var newNodes []HPCKubernetesNode
	for i := 0; i < req.Count; i++ {
		idx := int(existingCount) + i + 1
		node := HPCKubernetesNode{
			ID:         uuid.New().String(),
			ClusterID:  clusterID,
			Name:       fmt.Sprintf("%s-%s-%d", cluster.Name, req.Role, idx),
			Role:       req.Role,
			IPAddress:  fmt.Sprintf("10.0.1.%d", 30+idx),
			FlavorName: defaultStr(req.Flavor, cluster.WorkerFlavor),
			Status:     "ready",
			K8sVersion: cluster.KubernetesVersion,
			GPUCount:   req.GPUCount,
			GPUType:    req.GPUType,
		}
		_ = s.db.Create(&node)
		newNodes = append(newNodes, node)
	}
	if req.Role == "gpu-worker" || req.Role == "worker" {
		s.db.Model(&cluster).Update("worker_count", gorm.Expr("worker_count + ?", req.Count))
	}
	c.JSON(http.StatusCreated, gin.H{"nodes": newNodes})
}

func (s *Service) removeK8sNode(c *gin.Context) {
	nodeID := c.Param("nodeId")
	var node HPCKubernetesNode
	if err := s.db.First(&node, "id = ? AND cluster_id = ?", nodeID, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	if node.Role == "control-plane" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove control-plane node directly"})
		return
	}
	s.db.Delete(&node)
	s.db.Model(&HPCKubernetesCluster{}).Where("id = ?", c.Param("id")).
		Update("worker_count", gorm.Expr("worker_count - 1"))
	c.JSON(http.StatusOK, gin.H{"message": "node removed"})
}

// ── GPU Pools ────────────────────────────────────────────────────
