package hpc

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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
