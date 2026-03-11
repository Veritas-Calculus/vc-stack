package hpc

import (
	"fmt"
	"net/http"
	"time"

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

func (s *Service) listGPUPools(c *gin.Context) {
	var pools []HPCGPUPool
	s.db.Where("cluster_id = ?", c.Param("id")).Find(&pools)
	c.JSON(http.StatusOK, gin.H{"gpu_pools": pools})
}

func (s *Service) createGPUPool(c *gin.Context) {
	clusterID := c.Param("id")
	var req struct {
		Name       string `json:"name" binding:"required"`
		GPUType    string `json:"gpu_type" binding:"required"`
		GPUCount   int    `json:"gpu_count" binding:"required"`
		MIGEnabled bool   `json:"mig_enabled"`
		MIGProfile string `json:"mig_profile"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pool := HPCGPUPool{
		ID:         uuid.New().String(),
		ClusterID:  clusterID,
		Name:       req.Name,
		GPUType:    req.GPUType,
		GPUCount:   req.GPUCount,
		Available:  req.GPUCount,
		MIGEnabled: req.MIGEnabled,
		MIGProfile: req.MIGProfile,
	}
	s.db.Create(&pool)
	c.JSON(http.StatusCreated, gin.H{"gpu_pool": pool})
}

func (s *Service) deleteGPUPool(c *gin.Context) {
	s.db.Where("id = ? AND cluster_id = ?", c.Param("poolId"), c.Param("id")).Delete(&HPCGPUPool{})
	c.JSON(http.StatusOK, gin.H{"message": "GPU pool deleted"})
}

// ──────────────────────────────────────────────────────────────────
// Slurm Cluster Handlers
// ──────────────────────────────────────────────────────────────────

func (s *Service) listSlurmClusters(c *gin.Context) {
	var clusters []SlurmCluster
	query := s.db.Order("created_at DESC")
	if projectID := c.GetString("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	query.Find(&clusters)
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

func (s *Service) createSlurmCluster(c *gin.Context) {
	var req struct {
		Name              string `json:"name" binding:"required"`
		Description       string `json:"description"`
		SlurmVersion      string `json:"slurm_version"`
		AccountingEnabled *bool  `json:"accounting_enabled"`
		FairShareEnabled  *bool  `json:"fairshare_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SlurmVersion == "" {
		req.SlurmVersion = "24.05"
	}
	acctEnabled := true
	if req.AccountingEnabled != nil {
		acctEnabled = *req.AccountingEnabled
	}
	fsEnabled := true
	if req.FairShareEnabled != nil {
		fsEnabled = *req.FairShareEnabled
	}

	projectID := c.GetString("project_id")

	cluster := SlurmCluster{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Description:       req.Description,
		ProjectID:         projectID,
		Status:            "provisioning",
		SlurmVersion:      req.SlurmVersion,
		AccountingEnabled: acctEnabled,
		FairShareEnabled:  fsEnabled,
	}
	if err := s.db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "slurm cluster name already exists in this project"})
		return
	}

	// Simulate provisioning completion.
	s.db.Model(&cluster).Updates(map[string]interface{}{
		"status":       "active",
		"api_endpoint": fmt.Sprintf("http://%s-slurmctld:6820", cluster.Name),
	})

	s.db.First(&cluster, "id = ?", cluster.ID)
	s.logger.Info("Slurm cluster created",
		zap.String("cluster_id", cluster.ID),
		zap.String("name", cluster.Name))

	c.JSON(http.StatusCreated, gin.H{"cluster": cluster})
}

func (s *Service) getSlurmCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster SlurmCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "slurm cluster not found"})
		return
	}
	var partitions []SlurmPartition
	s.db.Where("cluster_id = ?", id).Find(&partitions)
	var nodes []SlurmNode
	s.db.Where("cluster_id = ?", id).Find(&nodes)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster, "partitions": partitions, "nodes": nodes})
}

func (s *Service) deleteSlurmCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster SlurmCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "slurm cluster not found"})
		return
	}
	// Cascade delete.
	s.db.Where("cluster_id = ?", id).Delete(&SlurmNode{})
	s.db.Where("cluster_id = ?", id).Delete(&SlurmPartition{})
	s.db.Where("cluster_id = ?", id).Delete(&SlurmUserMapping{})
	s.db.Delete(&cluster)
	s.logger.Info("Slurm cluster deleted", zap.String("cluster_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "slurm cluster deleted"})
}

// ── Slurm Partitions ─────────────────────────────────────────────

func (s *Service) listPartitions(c *gin.Context) {
	var partitions []SlurmPartition
	s.db.Where("cluster_id = ?", c.Param("id")).Order("priority DESC, name").Find(&partitions)
	c.JSON(http.StatusOK, gin.H{"partitions": partitions})
}

func (s *Service) createPartition(c *gin.Context) {
	clusterID := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		MaxTime     string `json:"max_time"`
		MaxNodes    int    `json:"max_nodes"`
		DefaultMem  int    `json:"default_mem_mb"`
		GPUType     string `json:"gpu_type"`
		GPUsPerNode int    `json:"gpus_per_node"`
		Priority    int    `json:"priority"`
		PreemptMode string `json:"preempt_mode"`
		Default     bool   `json:"default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	partition := SlurmPartition{
		ID:          uuid.New().String(),
		ClusterID:   clusterID,
		Name:        req.Name,
		State:       "UP",
		Default:     req.Default,
		MaxTime:     defaultStr(req.MaxTime, "INFINITE"),
		MaxNodes:    req.MaxNodes,
		DefaultMem:  req.DefaultMem,
		GPUType:     req.GPUType,
		GPUsPerNode: req.GPUsPerNode,
		Priority:    defaultInt(req.Priority, 1),
		PreemptMode: defaultStr(req.PreemptMode, "off"),
	}
	s.db.Create(&partition)
	c.JSON(http.StatusCreated, gin.H{"partition": partition})
}

func (s *Service) deletePartition(c *gin.Context) {
	s.db.Where("id = ? AND cluster_id = ?", c.Param("partId"), c.Param("id")).Delete(&SlurmPartition{})
	c.JSON(http.StatusOK, gin.H{"message": "partition deleted"})
}

// ── Slurm User Sync ──────────────────────────────────────────────

func (s *Service) listSlurmUsers(c *gin.Context) {
	var mappings []SlurmUserMapping
	s.db.Where("cluster_id = ?", c.Param("id")).Find(&mappings)
	c.JSON(http.StatusOK, gin.H{"user_mappings": mappings})
}

func (s *Service) syncSlurmUser(c *gin.Context) {
	clusterID := c.Param("id")
	var req struct {
		Username        string `json:"username" binding:"required"`
		SlurmPartitions string `json:"slurm_partitions"`
		QOS             string `json:"qos"`
		MaxGPUs         int    `json:"max_gpus"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID := c.GetString("project_id")
	userID := fmt.Sprintf("%v", c.MustGet("user_id"))

	mapping := SlurmUserMapping{
		ID:              uuid.New().String(),
		ClusterID:       clusterID,
		UserID:          userID,
		Username:        req.Username,
		SlurmUser:       req.Username,
		SlurmAccount:    projectID,
		SlurmPartitions: req.SlurmPartitions,
		QOS:             defaultStr(req.QOS, "normal"),
		MaxGPUs:         req.MaxGPUs,
		Status:          "active",
	}
	s.db.Create(&mapping)
	c.JSON(http.StatusCreated, gin.H{"user_mapping": mapping})
}

func (s *Service) removeSlurmUser(c *gin.Context) {
	s.db.Where("id = ? AND cluster_id = ?", c.Param("mappingId"), c.Param("id")).
		Delete(&SlurmUserMapping{})
	c.JSON(http.StatusOK, gin.H{"message": "user mapping removed"})
}

// ──────────────────────────────────────────────────────────────────
// Unified Job Handlers
// ──────────────────────────────────────────────────────────────────

func (s *Service) listJobs(c *gin.Context) {
	var jobs []HPCJob
	query := s.db.Order("submitted_at DESC")
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
	if scheduler := c.Query("scheduler"); scheduler != "" {
		query = query.Where("scheduler = ?", scheduler)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&jobs)
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (s *Service) submitJob(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Scheduler      string `json:"scheduler" binding:"required"` // kubernetes, slurm
		ClusterID      string `json:"cluster_id" binding:"required"`
		CPUs           int    `json:"cpus"`
		MemoryMB       int    `json:"memory_mb"`
		GPUs           int    `json:"gpus"`
		GPUType        string `json:"gpu_type"`
		Nodes          int    `json:"nodes"`
		Script         string `json:"script"`
		Image          string `json:"image"`
		WorkDir        string `json:"work_dir"`
		EnvVars        string `json:"env_vars"`
		WallTimeLimit  string `json:"wall_time_limit"`
		SlurmPartition string `json:"slurm_partition"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Scheduler != "kubernetes" && req.Scheduler != "slurm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scheduler must be 'kubernetes' or 'slurm'"})
		return
	}

	projectID := c.GetString("project_id")
	userID := fmt.Sprintf("%v", c.MustGet("user_id"))

	job := HPCJob{
		ID:             uuid.New().String(),
		Name:           req.Name,
		ProjectID:      projectID,
		UserID:         userID,
		Scheduler:      req.Scheduler,
		ClusterID:      req.ClusterID,
		Status:         "pending",
		CPUs:           defaultInt(req.CPUs, 1),
		MemoryMB:       defaultInt(req.MemoryMB, 1024),
		GPUs:           req.GPUs,
		GPUType:        req.GPUType,
		Nodes:          defaultInt(req.Nodes, 1),
		Script:         req.Script,
		Image:          req.Image,
		WorkDir:        req.WorkDir,
		EnvVars:        req.EnvVars,
		WallTimeLimit:  req.WallTimeLimit,
		SlurmPartition: req.SlurmPartition,
	}
	job.SubmittedAt = time.Now()

	s.db.Create(&job)
	s.logger.Info("HPC job submitted",
		zap.String("job_id", job.ID),
		zap.String("scheduler", job.Scheduler),
		zap.String("cluster_id", job.ClusterID))

	c.JSON(http.StatusCreated, gin.H{"job": job})
}

func (s *Service) getJob(c *gin.Context) {
	id := c.Param("id")
	var job HPCJob
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": job})
}

func (s *Service) cancelJob(c *gin.Context) {
	id := c.Param("id")
	var job HPCJob
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job is already in terminal state"})
		return
	}
	now := time.Now()
	s.db.Model(&job).Updates(map[string]interface{}{
		"status":       "cancelled",
		"completed_at": &now,
	})
	s.logger.Info("HPC job cancelled", zap.String("job_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
}

// ──────────────────────────────────────────────────────────────────
// P1: HPC Cluster Components & Reconciliation
// ──────────────────────────────────────────────────────────────────

func (s *Service) getClusterComponents(c *gin.Context) {
	id := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var gpuPools []HPCGPUPool
	s.db.Where("cluster_id = ?", id).Find(&gpuPools)

	components := s.orchestrator.BuildComponentsForCluster(&cluster, gpuPools)
	c.JSON(http.StatusOK, gin.H{"components": components})
}

func (s *Service) reconcileCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster HPCKubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	if cluster.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster must be active to reconcile"})
		return
	}

	var gpuPools []HPCGPUPool
	s.db.Where("cluster_id = ?", id).Find(&gpuPools)

	components := s.orchestrator.BuildComponentsForCluster(&cluster, gpuPools)
	result := s.orchestrator.ReconcileCluster(c.Request.Context(), components)

	c.JSON(http.StatusOK, gin.H{
		"result":     result,
		"components": components,
	})
}

func (s *Service) getNodeGPUTopology(c *gin.Context) {
	nodeID := c.Param("nodeId")
	var node HPCKubernetesNode
	if err := s.db.First(&node, "id = ? AND cluster_id = ?", nodeID, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	if node.GPUCount == 0 {
		c.JSON(http.StatusOK, gin.H{"topology": nil, "message": "node has no GPUs"})
		return
	}

	topology := s.orchestrator.DiscoverGPUTopology(node.Name, node.GPUCount, node.GPUType)
	c.JSON(http.StatusOK, gin.H{"topology": topology})
}

// getJobManifest generates the K8s manifest for a job (Volcano vcjob or MPIJob).
func (s *Service) getJobManifest(c *gin.Context) {
	id := c.Param("id")
	var job HPCJob
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.Scheduler != "kubernetes" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manifest generation is only available for kubernetes jobs"})
		return
	}

	// Check if the cluster has MPI enabled
	var cluster HPCKubernetesCluster
	s.db.First(&cluster, "id = ?", job.ClusterID)

	if cluster.EnableMPI && job.Nodes > 1 {
		manifest := s.orchestrator.BuildMPIJobManifest(&job)
		c.JSON(http.StatusOK, gin.H{"kind": "MPIJob", "manifest": manifest})
		return
	}

	manifest := s.orchestrator.BuildVolcanoJobManifest(&job)
	c.JSON(http.StatusOK, gin.H{"kind": "VolcanoJob", "manifest": manifest})
}

// ──────────────────────────────────────────────────────────────────
// P2: Slurm Integration Handlers
// ──────────────────────────────────────────────────────────────────

func (s *Service) executeSlurmUserSync(c *gin.Context) {
	clusterID := c.Param("id")
	var cluster SlurmCluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "slurm cluster not found"})
		return
	}
	if cluster.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster must be active to sync users"})
		return
	}

	// Get current IAM mappings
	var mappings []SlurmUserMapping
	s.db.Where("cluster_id = ? AND status = ?", clusterID, "active").Find(&mappings)

	// Get existing Slurm users (simulated — in production, query slurmrestd)
	existingUsers := []string{}

	// Build and execute sync plan
	plan := s.slurmClient.BuildUserSyncPlan(&cluster, mappings, existingUsers)
	result := s.slurmClient.ExecuteUserSync(c.Request.Context(), plan)

	c.JSON(http.StatusOK, gin.H{
		"plan":   plan,
		"result": result,
	})
}

func (s *Service) listSlurmNodes(c *gin.Context) {
	var nodes []SlurmNode
	s.db.Where("cluster_id = ?", c.Param("id")).Order("hostname").Find(&nodes)
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (s *Service) addSlurmNode(c *gin.Context) {
	clusterID := c.Param("id")
	var cluster SlurmCluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "slurm cluster not found"})
		return
	}
	var req struct {
		Hostname   string `json:"hostname" binding:"required"`
		IPAddress  string `json:"ip_address"`
		InstanceID string `json:"instance_id"`
		CPUs       int    `json:"cpus"`
		MemoryMB   int    `json:"memory_mb"`
		GPUCount   int    `json:"gpu_count"`
		GPUType    string `json:"gpu_type"`
		Partitions string `json:"partitions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node := SlurmNode{
		ID:         fmt.Sprintf("sn-%s", time.Now().Format("20060102150405")),
		ClusterID:  clusterID,
		Hostname:   req.Hostname,
		IPAddress:  req.IPAddress,
		InstanceID: req.InstanceID,
		State:      "idle",
		CPUs:       defaultInt(req.CPUs, 4),
		MemoryMB:   defaultInt(req.MemoryMB, 8192),
		GPUCount:   req.GPUCount,
		GPUType:    req.GPUType,
		Partitions: req.Partitions,
	}
	s.db.Create(&node)
	s.db.Model(&cluster).Update("compute_node_count", gorm.Expr("compute_node_count + 1"))

	c.JSON(http.StatusCreated, gin.H{"node": node})
}

// getSlurmSubmitScript generates the sbatch submission payload for a Slurm job.
func (s *Service) getSlurmSubmitScript(c *gin.Context) {
	id := c.Param("id")
	var job HPCJob
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.Scheduler != "slurm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sbatch script generation is only available for slurm jobs"})
		return
	}

	// Get user mapping if available
	var mapping SlurmUserMapping
	s.db.Where("cluster_id = ? AND user_id = ?", job.ClusterID, job.UserID).First(&mapping)

	var mappingPtr *SlurmUserMapping
	if mapping.ID != "" {
		mappingPtr = &mapping
	}

	submit := s.slurmClient.BuildSubmitScript(&job, mappingPtr)
	c.JSON(http.StatusOK, gin.H{
		"submit_script": submit,
		"sbatch_script": submit.Script,
	})
}

// ──────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────

func defaultStr(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func defaultInt(val, fallback int) int {
	if val == 0 {
		return fallback
	}
	return val
}
