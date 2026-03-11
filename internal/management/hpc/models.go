// Package hpc implements High Performance Computing services for VC Stack.
// Provides Kubernetes HPC cluster management (GPU scheduling, MPI, Volcano/Kueue),
// Slurm workload manager integration, and a unified job abstraction layer.
package hpc

import "time"

// ---------- HPC Kubernetes Cluster ----------

// HPCKubernetesCluster represents a GPU-aware Kubernetes cluster for HPC workloads.
type HPCKubernetesCluster struct {
	ID                string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name              string `json:"name" gorm:"not null;uniqueIndex:uniq_hpc_k8s_tenant"`
	Description       string `json:"description"`
	ProjectID         string `json:"project_id" gorm:"not null;index;uniqueIndex:uniq_hpc_k8s_tenant"`
	Status            string `json:"status" gorm:"default:'pending'"` // pending, provisioning, active, error, deleting, upgrading
	KubernetesVersion string `json:"kubernetes_version" gorm:"not null;default:'1.30'"`

	// Node configuration
	NetworkID          string `json:"network_id" gorm:"index"`
	SubnetID           string `json:"subnet_id"`
	ControlPlaneCount  int    `json:"control_plane_count" gorm:"not null;default:1"`
	WorkerCount        int    `json:"worker_count" gorm:"not null;default:3"`
	ControlPlaneFlavor string `json:"control_plane_flavor" gorm:"default:'m1.medium'"`
	WorkerFlavor       string `json:"worker_flavor" gorm:"default:'m1.xlarge'"`
	APIEndpoint        string `json:"api_endpoint"`

	// HPC-specific: GPU scheduling
	GPUScheduler string `json:"gpu_scheduler" gorm:"default:'volcano'"` // volcano, kueue, default
	EnableMPI    bool   `json:"enable_mpi" gorm:"default:false"`
	EnableRDMA   bool   `json:"enable_rdma" gorm:"default:false"`

	// HPC-specific: shared filesystem
	SharedFSType     string `json:"shared_fs_type"`     // cephfs, lustre, nfs, none
	SharedFSEndpoint string `json:"shared_fs_endpoint"` // mount endpoint

	// GPU inventory (aggregated from nodes)
	TotalGPUs     int    `json:"total_gpus" gorm:"default:0"`
	AllocatedGPUs int    `json:"allocated_gpus" gorm:"default:0"`
	GPUTypes      string `json:"gpu_types"` // JSON array: ["A100","H100"]

	// HA
	HAEnabled bool `json:"ha_enabled" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName sets the table name for HPCKubernetesCluster.
func (HPCKubernetesCluster) TableName() string { return "hpc_kubernetes_clusters" }

// HPCKubernetesNode represents a node in an HPC K8s cluster.
type HPCKubernetesNode struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID  string `json:"cluster_id" gorm:"not null;index"`
	Name       string `json:"name" gorm:"not null"`
	Role       string `json:"role" gorm:"not null"`     // control-plane, worker, gpu-worker
	InstanceID string `json:"instance_id" gorm:"index"` // VC Stack VM or BMaaS node ID
	IPAddress  string `json:"ip_address"`
	FlavorName string `json:"flavor_name"`
	Status     string `json:"status" gorm:"default:'pending'"` // pending, provisioning, ready, not-ready, draining

	// GPU info (for gpu-worker nodes)
	GPUCount  int    `json:"gpu_count" gorm:"default:0"`
	GPUType   string `json:"gpu_type"`   // e.g. "A100-80GB"
	GPUDriver string `json:"gpu_driver"` // nvidia, amd

	K8sVersion string               `json:"k8s_version"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	Cluster    HPCKubernetesCluster `json:"-" gorm:"foreignKey:ClusterID"`
}

// TableName sets the table name for HPCKubernetesNode.
func (HPCKubernetesNode) TableName() string { return "hpc_kubernetes_nodes" }

// HPCGPUPool represents a pool of GPU resources within an HPC K8s cluster.
type HPCGPUPool struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID  string `json:"cluster_id" gorm:"not null;index"`
	Name       string `json:"name" gorm:"not null"`
	GPUType    string `json:"gpu_type" gorm:"not null"` // A100, H100, V100, etc.
	GPUCount   int    `json:"gpu_count" gorm:"not null"`
	Available  int    `json:"available" gorm:"default:0"`
	MIGEnabled bool   `json:"mig_enabled" gorm:"default:false"` // NVIDIA Multi-Instance GPU
	MIGProfile string `json:"mig_profile"`                      // e.g. "1g.5gb", "2g.10gb"

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName sets the table name for HPCGPUPool.
func (HPCGPUPool) TableName() string { return "hpc_gpu_pools" }

// ---------- Slurm Cluster ----------

// SlurmCluster represents a managed Slurm workload manager cluster.
type SlurmCluster struct {
	ID           string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string `json:"name" gorm:"not null;uniqueIndex:uniq_slurm_tenant"`
	Description  string `json:"description"`
	ProjectID    string `json:"project_id" gorm:"not null;index;uniqueIndex:uniq_slurm_tenant"`
	Status       string `json:"status" gorm:"default:'pending'"` // pending, provisioning, active, error, deleting
	SlurmVersion string `json:"slurm_version" gorm:"default:'24.05'"`

	// Controller nodes
	ControllerNodeID   string `json:"controller_node_id"`   // slurmctld host
	DBNodeID           string `json:"db_node_id"`           // slurmdbd host
	BackupControllerID string `json:"backup_controller_id"` // backup slurmctld

	// REST API endpoint (slurmrestd)
	APIEndpoint string `json:"api_endpoint"` // e.g. http://slurmctld:6820

	// Accounting
	AccountingEnabled bool `json:"accounting_enabled" gorm:"default:true"`
	FairShareEnabled  bool `json:"fairshare_enabled" gorm:"default:true"`

	// Compute node count
	ComputeNodeCount int `json:"compute_node_count" gorm:"default:0"`

	// GPU summary
	TotalGPUs     int    `json:"total_gpus" gorm:"default:0"`
	AllocatedGPUs int    `json:"allocated_gpus" gorm:"default:0"`
	GPUTypes      string `json:"gpu_types"` // JSON array

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName sets the table name for SlurmCluster.
func (SlurmCluster) TableName() string { return "hpc_slurm_clusters" }

// SlurmNode represents a compute node in a Slurm cluster.
type SlurmNode struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID  string `json:"cluster_id" gorm:"not null;index"`
	Hostname   string `json:"hostname" gorm:"not null"`
	InstanceID string `json:"instance_id" gorm:"index"` // VC Stack VM or BMaaS node ID
	IPAddress  string `json:"ip_address"`
	State      string `json:"state" gorm:"default:'idle'"` // idle, alloc, mix, down, drain
	CPUs       int    `json:"cpus" gorm:"default:0"`
	MemoryMB   int    `json:"memory_mb" gorm:"default:0"`
	GPUCount   int    `json:"gpu_count" gorm:"default:0"`
	GPUType    string `json:"gpu_type"`
	Partitions string `json:"partitions"` // comma-separated partition names

	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Cluster   SlurmCluster `json:"-" gorm:"foreignKey:ClusterID"`
}

// TableName sets the table name for SlurmNode.
func (SlurmNode) TableName() string { return "hpc_slurm_nodes" }

// SlurmPartition represents a Slurm partition (queue).
type SlurmPartition struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID   string `json:"cluster_id" gorm:"not null;index"`
	Name        string `json:"name" gorm:"not null"`
	State       string `json:"state" gorm:"default:'UP'"` // UP, DOWN, DRAIN, INACTIVE
	Default     bool   `json:"default" gorm:"default:false"`
	MaxTime     string `json:"max_time" gorm:"default:'INFINITE'"`
	MaxNodes    int    `json:"max_nodes" gorm:"default:0"`
	DefaultMem  int    `json:"default_mem_mb" gorm:"default:0"`
	GPUType     string `json:"gpu_type"`
	GPUsPerNode int    `json:"gpus_per_node" gorm:"default:0"`
	Priority    int    `json:"priority" gorm:"default:1"`
	PreemptMode string `json:"preempt_mode" gorm:"default:'off'"` // off, cancel, requeue, suspend
	NodeCount   int    `json:"node_count" gorm:"default:0"`

	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Cluster   SlurmCluster `json:"-" gorm:"foreignKey:ClusterID"`
}

// TableName sets the table name for SlurmPartition.
func (SlurmPartition) TableName() string { return "hpc_slurm_partitions" }

// ---------- Unified HPC Job ----------

// HPCJob represents a unified HPC job that abstracts across schedulers.
type HPCJob struct {
	ID        string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string `json:"name" gorm:"not null"`
	ProjectID string `json:"project_id" gorm:"not null;index"`
	UserID    string `json:"user_id" gorm:"not null;index"`
	Scheduler string `json:"scheduler" gorm:"not null"` // kubernetes, slurm
	ClusterID string `json:"cluster_id" gorm:"not null;index"`
	Status    string `json:"status" gorm:"default:'pending'"` // pending, queued, running, completed, failed, cancelled

	// Resource requests
	CPUs     int    `json:"cpus" gorm:"default:1"`
	MemoryMB int    `json:"memory_mb" gorm:"default:1024"`
	GPUs     int    `json:"gpus" gorm:"default:0"`
	GPUType  string `json:"gpu_type"`
	Nodes    int    `json:"nodes" gorm:"default:1"` // for MPI/multi-node jobs

	// Job script / container image
	Script  string `json:"script" gorm:"type:text"`   // sbatch script or command
	Image   string `json:"image"`                     // container image (for K8s jobs)
	WorkDir string `json:"work_dir"`                  // working directory
	EnvVars string `json:"env_vars" gorm:"type:text"` // JSON object of env vars

	// Timing
	SubmittedAt   time.Time  `json:"submitted_at"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	WallTimeLimit string     `json:"wall_time_limit"` // e.g. "24:00:00"

	// Scheduler-specific identifiers
	K8sJobName     string `json:"k8s_job_name"`                  // Kubernetes Job/MPIJob name
	K8sNamespace   string `json:"k8s_namespace"`                 // Kubernetes namespace
	SlurmJobID     int    `json:"slurm_job_id" gorm:"default:0"` // Slurm JOBID
	SlurmPartition string `json:"slurm_partition"`               // Slurm partition name

	// Results
	ExitCode   int    `json:"exit_code" gorm:"default:0"`
	OutputPath string `json:"output_path"`
	ErrorPath  string `json:"error_path"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName sets the table name for HPCJob.
func (HPCJob) TableName() string { return "hpc_jobs" }

// ---------- Slurm User Mapping ----------

// SlurmUserMapping maps a VC Stack IAM user to a Slurm slurmdb user.
type SlurmUserMapping struct {
	ID              string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID       string `json:"cluster_id" gorm:"not null;index"`
	UserID          string `json:"user_id" gorm:"not null;index"`
	Username        string `json:"username" gorm:"not null"`       // VC Stack username
	SlurmUser       string `json:"slurm_user" gorm:"not null"`     // Slurm slurmdb user
	SlurmAccount    string `json:"slurm_account" gorm:"not null"`  // Slurm account (= project)
	SlurmPartitions string `json:"slurm_partitions"`               // comma-separated allowed partitions
	QOS             string `json:"qos" gorm:"default:'normal'"`    // quality of service
	MaxGPUs         int    `json:"max_gpus" gorm:"default:0"`      // GrpTRES limit
	Status          string `json:"status" gorm:"default:'active'"` // active, suspended, deleted

	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Cluster   SlurmCluster `json:"-" gorm:"foreignKey:ClusterID"`
}

// TableName sets the table name for SlurmUserMapping.
func (SlurmUserMapping) TableName() string { return "hpc_slurm_user_mappings" }
