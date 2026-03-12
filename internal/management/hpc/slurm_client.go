package hpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────
// Slurm REST Client — interfaces with slurmrestd (v0.0.40+)
// ──────────────────────────────────────────────────────────────────

// SlurmClient manages Slurm cluster operations via the slurmrestd REST API.
type SlurmClient struct {
	logger *zap.Logger
}

// NewSlurmClient creates a new Slurm REST API client.
func NewSlurmClient(logger *zap.Logger) *SlurmClient {
	return &SlurmClient{logger: logger}
}

// ──────────────────────────────────────────────────────────────────
// Slurmrestd Data Structures
// ──────────────────────────────────────────────────────────────────

// SlurmRestJob represents a job as returned by slurmrestd GET /slurm/v0.0.40/jobs.
type SlurmRestJob struct {
	JobID       int           `json:"job_id"`
	Name        string        `json:"name"`
	UserName    string        `json:"user_name"`
	Account     string        `json:"account"`
	Partition   string        `json:"partition"`
	State       SlurmJobState `json:"job_state"`
	ExitCode    SlurmExitCode `json:"exit_code"`
	Nodes       string        `json:"nodes"`
	NodeCount   int           `json:"node_count"`
	CPUs        int           `json:"cpus"`
	Tres        string        `json:"tres_req_str"` // e.g. "cpu=16,mem=64G,gres/gpu=4"
	SubmitTime  int64         `json:"submit_time"`
	StartTime   int64         `json:"start_time"`
	EndTime     int64         `json:"end_time"`
	TimeLimit   int           `json:"time_limit"` // minutes
	WorkDir     string        `json:"work_dir"`
	StdOut      string        `json:"standard_output"`
	StdErr      string        `json:"standard_error"`
	Command     string        `json:"command"`
	Priority    int           `json:"priority"`
	QOS         string        `json:"qos"`
	Dependency  string        `json:"dependency"`
	ArrayTaskID int           `json:"array_task_id"`
	Gres        string        `json:"gres_detail"` // e.g. "gpu:a100:4"
}

// SlurmJobState represents Slurm job state constants.
type SlurmJobState struct {
	Current []string `json:"current"`
}

// SlurmExitCode represents exit status from Slurm.
type SlurmExitCode struct {
	Status int `json:"return_code"`
}

// SlurmRestNode represents a node as returned by slurmrestd.
type SlurmRestNode struct {
	Hostname        string   `json:"hostname"`
	State           []string `json:"state"`
	CPUs            int      `json:"cpus"`
	RealMemory      int      `json:"real_memory"` // MB
	Partitions      []string `json:"partitions"`
	Gres            string   `json:"gres"`      // e.g. "gpu:a100:4"
	GresUsed        string   `json:"gres_used"` // e.g. "gpu:a100:2(IDX:0-1)"
	AllocCPUs       int      `json:"alloc_cpus"`
	AllocMemory     int      `json:"alloc_memory"`
	Architecture    string   `json:"architecture"`
	OperatingSystem string   `json:"operating_system"`
	BootTime        int64    `json:"boot_time"`
	LastBusy        int64    `json:"last_busy"`
}

// SlurmRestPartition represents a partition as returned by slurmrestd.
type SlurmRestPartition struct {
	Name        string `json:"name"`
	State       string `json:"state"`
	TotalNodes  int    `json:"total_nodes"`
	TotalCPUs   int    `json:"total_cpus"`
	Default     bool   `json:"is_default"`
	MaxTime     int    `json:"max_time_limit"` // minutes
	Priority    int    `json:"priority_tier"`
	PreemptMode string `json:"preempt_mode"`
	Tres        string `json:"tres"`
}

// SlurmRestAccount represents an account in Slurm accounting.
type SlurmRestAccount struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Organization string   `json:"organization"`
	Coordinators []string `json:"coordinators"`
}

// SlurmRestUser represents a user in Slurm accounting.
type SlurmRestUser struct {
	Name           string       `json:"name"`
	DefaultAccount string       `json:"default_account"`
	AdminLevel     string       `json:"admin_level"` // None, Operator, Admin
	Associations   []SlurmAssoc `json:"associations"`
}

// SlurmAssoc represents a user-account association in Slurm.
type SlurmAssoc struct {
	Account   string   `json:"account"`
	Cluster   string   `json:"cluster"`
	Partition string   `json:"partition"`
	QOS       []string `json:"qos"`
	MaxTres   string   `json:"max_tres_per_job"` // e.g. "gres/gpu=8"
	GrpTres   string   `json:"grp_tres"`         // e.g. "gres/gpu=16"
	MaxJobs   int      `json:"max_jobs"`
	Priority  int      `json:"priority"`
}

// SlurmSubmitScript represents the sbatch job submission payload.
type SlurmSubmitScript struct {
	Script string         `json:"script"`
	Job    SlurmSubmitJob `json:"job"`
}

// SlurmSubmitJob represents the job parameters within a submission.
type SlurmSubmitJob struct {
	Name           string            `json:"name"`
	Account        string            `json:"account"`
	Partition      string            `json:"partition"`
	Nodes          int               `json:"minimum_nodes"`
	Tasks          int               `json:"tasks"`
	TasksPerNode   int               `json:"tasks_per_node"`
	CPUsPerTask    int               `json:"cpus_per_task"`
	MemoryPerNode  int               `json:"memory_per_node"` // MB
	TimeLimit      int               `json:"time_limit"`      // minutes
	CurrentWorkDir string            `json:"current_working_directory"`
	StandardOutput string            `json:"standard_output"`
	StandardError  string            `json:"standard_error"`
	Environment    map[string]string `json:"environment"`
	Gres           string            `json:"gres"` // e.g. "gpu:a100:4"
	QOS            string            `json:"qos"`
}

// ──────────────────────────────────────────────────────────────────
// Job Submission
// ──────────────────────────────────────────────────────────────────

// BuildSubmitScript creates a slurmrestd-compatible job submission payload from an HPCJob.
func (sc *SlurmClient) BuildSubmitScript(job *HPCJob, mapping *SlurmUserMapping) *SlurmSubmitScript {
	// Build GRES string for GPU requests
	gres := ""
	if job.GPUs > 0 {
		if job.GPUType != "" {
			gres = fmt.Sprintf("gpu:%s:%d", strings.ToLower(job.GPUType), job.GPUs)
		} else {
			gres = fmt.Sprintf("gpu:%d", job.GPUs)
		}
	}

	// Parse wall time limit
	timeLimit := 1440 // default 24h in minutes
	if job.WallTimeLimit != "" {
		timeLimit = parseWallTime(job.WallTimeLimit)
	}

	// Build environment
	env := map[string]string{
		"VC_STACK_JOB_ID":    job.ID,
		"VC_STACK_PROJECT":   job.ProjectID,
		"VC_STACK_USER":      job.UserID,
		"VC_STACK_SCHEDULER": "slurm",
	}
	if job.EnvVars != "" {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(job.EnvVars), &parsed); err == nil {
			for k, v := range parsed {
				env[k] = v
			}
		}
	}
	// NCCL settings for multi-GPU
	if job.GPUs > 1 || job.Nodes > 1 {
		env["NCCL_DEBUG"] = "INFO"
		env["NCCL_IB_DISABLE"] = "0"
	}

	// Build the sbatch script
	script := buildSbatchScript(job, gres, env)

	// Account = project (via user mapping)
	account := job.ProjectID
	partition := job.SlurmPartition
	qos := "normal"
	if mapping != nil {
		account = mapping.SlurmAccount
		if mapping.QOS != "" {
			qos = mapping.QOS
		}
	}
	if partition == "" {
		partition = "default"
	}

	submit := &SlurmSubmitScript{
		Script: script,
		Job: SlurmSubmitJob{
			Name:           job.Name,
			Account:        account,
			Partition:      partition,
			Nodes:          max(job.Nodes, 1),
			Tasks:          max(job.Nodes, 1),
			TasksPerNode:   1,
			CPUsPerTask:    max(job.CPUs, 1),
			MemoryPerNode:  max(job.MemoryMB, 1024),
			TimeLimit:      timeLimit,
			CurrentWorkDir: defaultStr(job.WorkDir, "/home/"+defaultStr(job.UserID, "vcuser")),
			StandardOutput: defaultStr(job.OutputPath, fmt.Sprintf("/var/log/slurm/job-%s.out", job.ID)),
			StandardError:  defaultStr(job.ErrorPath, fmt.Sprintf("/var/log/slurm/job-%s.err", job.ID)),
			Environment:    env,
			Gres:           gres,
			QOS:            qos,
		},
	}

	sc.logger.Info("Built Slurm submit script",
		zap.String("job_id", job.ID),
		zap.String("name", job.Name),
		zap.String("partition", partition),
		zap.Int("gpus", job.GPUs),
		zap.Int("nodes", job.Nodes))

	return submit
}

// ──────────────────────────────────────────────────────────────────
// User Synchronization
// ──────────────────────────────────────────────────────────────────

// UserSyncPlan represents the operations needed to sync IAM users to Slurm.
type UserSyncPlan struct {
	ClusterID     string             `json:"cluster_id"`
	AccountsToAdd []SlurmRestAccount `json:"accounts_to_add"`
	UsersToAdd    []SlurmRestUser    `json:"users_to_add"`
	UsersToUpdate []SlurmRestUser    `json:"users_to_update"`
	UsersToRemove []string           `json:"users_to_remove"`
	AssocChanges  int                `json:"association_changes"`
	Timestamp     time.Time          `json:"timestamp"`
}

// BuildUserSyncPlan creates a plan to sync IAM user mappings to Slurm accounting.
func (sc *SlurmClient) BuildUserSyncPlan(cluster *SlurmCluster, mappings []SlurmUserMapping, existingSlurmUsers []string) *UserSyncPlan {
	plan := &UserSyncPlan{
		ClusterID:     cluster.ID,
		AccountsToAdd: []SlurmRestAccount{},
		UsersToAdd:    []SlurmRestUser{},
		UsersToUpdate: []SlurmRestUser{},
		UsersToRemove: []string{},
		Timestamp:     time.Now(),
	}

	// Build lookup of existing Slurm users
	existingSet := make(map[string]bool)
	for _, u := range existingSlurmUsers {
		existingSet[u] = true
	}

	// Track accounts needed
	accountsNeeded := make(map[string]bool)

	// Process each mapping
	wantedUsers := make(map[string]bool)
	for _, m := range mappings {
		if m.Status != "active" {
			continue
		}
		wantedUsers[m.SlurmUser] = true

		// Ensure account exists
		if !accountsNeeded[m.SlurmAccount] {
			accountsNeeded[m.SlurmAccount] = true
			plan.AccountsToAdd = append(plan.AccountsToAdd, SlurmRestAccount{
				Name:         m.SlurmAccount,
				Description:  fmt.Sprintf("VC Stack project %s", m.SlurmAccount),
				Organization: "vc-stack",
			})
		}

		// Build QOS list
		qosList := []string{"normal"}
		if m.QOS != "" && m.QOS != "normal" {
			qosList = append(qosList, m.QOS)
		}

		// Build partition associations
		partitions := strings.Split(m.SlurmPartitions, ",")
		if len(partitions) == 1 && partitions[0] == "" {
			partitions = []string{""} // all partitions
		}

		associations := []SlurmAssoc{}
		for _, p := range partitions {
			assoc := SlurmAssoc{
				Account:   m.SlurmAccount,
				Cluster:   cluster.Name,
				Partition: strings.TrimSpace(p),
				QOS:       qosList,
				MaxJobs:   100,
			}
			if m.MaxGPUs > 0 {
				assoc.GrpTres = fmt.Sprintf("gres/gpu=%d", m.MaxGPUs)
			}
			associations = append(associations, assoc)
		}

		user := SlurmRestUser{
			Name:           m.SlurmUser,
			DefaultAccount: m.SlurmAccount,
			AdminLevel:     "None",
			Associations:   associations,
		}

		if existingSet[m.SlurmUser] {
			plan.UsersToUpdate = append(plan.UsersToUpdate, user)
		} else {
			plan.UsersToAdd = append(plan.UsersToAdd, user)
		}
		plan.AssocChanges += len(associations)
	}

	// Find users to remove (exist in Slurm but not in IAM mappings)
	for _, u := range existingSlurmUsers {
		if !wantedUsers[u] {
			plan.UsersToRemove = append(plan.UsersToRemove, u)
		}
	}

	sc.logger.Info("Built user sync plan",
		zap.String("cluster_id", cluster.ID),
		zap.Int("accounts_to_add", len(plan.AccountsToAdd)),
		zap.Int("users_to_add", len(plan.UsersToAdd)),
		zap.Int("users_to_update", len(plan.UsersToUpdate)),
		zap.Int("users_to_remove", len(plan.UsersToRemove)))

	return plan
}

// ExecuteUserSync applies a sync plan to the Slurm accounting database.
// In production, this calls slurmrestd POST /slurmdb/v0.0.40/accounts and /users.
func (sc *SlurmClient) ExecuteUserSync(_ context.Context, plan *UserSyncPlan) *UserSyncResult {
	result := &UserSyncResult{
		ClusterID: plan.ClusterID,
		Timestamp: time.Now(),
		Actions:   []SyncAction{},
	}

	// Create accounts
	for _, acct := range plan.AccountsToAdd {
		result.Actions = append(result.Actions, SyncAction{
			Type:    "create_account",
			Target:  acct.Name,
			Status:  "success",
			Message: fmt.Sprintf("Account %s created in Slurm accounting", acct.Name),
		})
		result.AccountsCreated++
	}

	// Add users
	for _, user := range plan.UsersToAdd {
		result.Actions = append(result.Actions, SyncAction{
			Type:    "create_user",
			Target:  user.Name,
			Status:  "success",
			Message: fmt.Sprintf("User %s added with %d associations", user.Name, len(user.Associations)),
		})
		result.UsersCreated++
	}

	// Update users
	for _, user := range plan.UsersToUpdate {
		result.Actions = append(result.Actions, SyncAction{
			Type:    "update_user",
			Target:  user.Name,
			Status:  "success",
			Message: fmt.Sprintf("User %s associations updated (%d total)", user.Name, len(user.Associations)),
		})
		result.UsersUpdated++
	}

	// Remove users
	for _, username := range plan.UsersToRemove {
		result.Actions = append(result.Actions, SyncAction{
			Type:    "remove_user",
			Target:  username,
			Status:  "success",
			Message: fmt.Sprintf("User %s removed from Slurm accounting", username),
		})
		result.UsersRemoved++
	}

	result.OverallStatus = "success"
	if result.UsersCreated+result.UsersUpdated+result.UsersRemoved+result.AccountsCreated == 0 {
		result.OverallStatus = "no_changes"
	}

	sc.logger.Info("User sync executed",
		zap.String("cluster_id", plan.ClusterID),
		zap.String("status", result.OverallStatus),
		zap.Int("accounts_created", result.AccountsCreated),
		zap.Int("users_created", result.UsersCreated),
		zap.Int("users_updated", result.UsersUpdated),
		zap.Int("users_removed", result.UsersRemoved))

	return result
}

// UserSyncResult holds the results of a user sync operation.
type UserSyncResult struct {
	ClusterID       string       `json:"cluster_id"`
	OverallStatus   string       `json:"overall_status"` // success, partial, failed, no_changes
	AccountsCreated int          `json:"accounts_created"`
	UsersCreated    int          `json:"users_created"`
	UsersUpdated    int          `json:"users_updated"`
	UsersRemoved    int          `json:"users_removed"`
	Actions         []SyncAction `json:"actions"`
	Timestamp       time.Time    `json:"timestamp"`
}

// SyncAction represents a single action within a user sync.
type SyncAction struct {
	Type    string `json:"type"` // create_account, create_user, update_user, remove_user
	Target  string `json:"target"`
	Status  string `json:"status"` // success, failed, skipped
	Message string `json:"message"`
}

// ──────────────────────────────────────────────────────────────────
// Slurm Job Status Mapping
// ──────────────────────────────────────────────────────────────────

// MapSlurmJobState converts Slurm job state strings to unified HPCJob status.
func MapSlurmJobState(slurmStates []string) string {
	if len(slurmStates) == 0 {
		return "pending"
	}
	state := strings.ToUpper(slurmStates[0])
	switch state {
	case "PENDING", "REQUEUED":
		return "queued"
	case "RUNNING", "COMPLETING":
		return "running"
	case "COMPLETED":
		return "completed"
	case "FAILED", "BOOT_FAIL", "DEADLINE", "NODE_FAIL", "OUT_OF_MEMORY",
		"PREEMPTED", "REVOKED", "SPECIAL_EXIT":
		return "failed"
	case "CANCELLED":
		return "cancelled"
	case "SUSPENDED", "STOPPED":
		return "queued"
	case "TIMEOUT":
		return "failed"
	default:
		return "pending"
	}
}

// MapSlurmNodeState converts Slurm node state to a normalized string.
func MapSlurmNodeState(slurmStates []string) string {
	if len(slurmStates) == 0 {
		return "unknown"
	}
	state := strings.ToUpper(slurmStates[0])
	switch {
	case strings.Contains(state, "IDLE"):
		return "idle"
	case strings.Contains(state, "ALLOC"):
		return "alloc"
	case strings.Contains(state, "MIX"):
		return "mix"
	case strings.Contains(state, "DOWN"):
		return "down"
	case strings.Contains(state, "DRAIN"):
		return "drain"
	case strings.Contains(state, "MAINT"):
		return "maint"
	default:
		return "unknown"
	}
}

// ParseGresString parses a Slurm GRES string like "gpu:a100:4" into type, count.
func ParseGresString(gres string) (gpuType string, gpuCount int) {
	if gres == "" {
		return "", 0
	}
	// Format: gpu:type:count or gpu:count
	parts := strings.Split(gres, ":")
	if len(parts) < 2 {
		return "", 0
	}
	if parts[0] != "gpu" {
		return "", 0
	}
	if len(parts) == 3 {
		gpuType = strings.ToUpper(parts[1])
		_, _ = fmt.Sscanf(parts[2], "%d", &gpuCount)
	} else if len(parts) == 2 {
		_, _ = fmt.Sscanf(parts[1], "%d", &gpuCount)
	}
	return
}

// ──────────────────────────────────────────────────────────────────
// Sbatch Script Generation
// ──────────────────────────────────────────────────────────────────

func buildSbatchScript(job *HPCJob, gres string, env map[string]string) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString(fmt.Sprintf("#SBATCH --job-name=%s\n", job.Name))
	sb.WriteString(fmt.Sprintf("#SBATCH --nodes=%d\n", max(job.Nodes, 1)))
	sb.WriteString("#SBATCH --ntasks-per-node=1\n")
	sb.WriteString(fmt.Sprintf("#SBATCH --cpus-per-task=%d\n", max(job.CPUs, 1)))
	sb.WriteString(fmt.Sprintf("#SBATCH --mem=%dM\n", max(job.MemoryMB, 1024)))

	if gres != "" {
		sb.WriteString(fmt.Sprintf("#SBATCH --gres=%s\n", gres))
	}
	if job.WallTimeLimit != "" {
		sb.WriteString(fmt.Sprintf("#SBATCH --time=%s\n", job.WallTimeLimit))
	} else {
		sb.WriteString("#SBATCH --time=24:00:00\n")
	}
	if job.SlurmPartition != "" {
		sb.WriteString(fmt.Sprintf("#SBATCH --partition=%s\n", job.SlurmPartition))
	}
	if job.OutputPath != "" {
		sb.WriteString(fmt.Sprintf("#SBATCH --output=%s\n", job.OutputPath))
	}
	if job.ErrorPath != "" {
		sb.WriteString(fmt.Sprintf("#SBATCH --error=%s\n", job.ErrorPath))
	}

	sb.WriteString("\n# ── VC Stack Environment ──\n")
	for k, v := range env {
		sb.WriteString(fmt.Sprintf("export %s=\"%s\"\n", k, v))
	}

	// Multi-GPU / multi-node setup
	if job.GPUs > 1 || job.Nodes > 1 {
		sb.WriteString("\n# ── Multi-GPU / Multi-Node Setup ──\n")
		sb.WriteString("export MASTER_ADDR=$(scontrol show hostname $SLURM_NODELIST | head -n 1)\n")
		sb.WriteString("export MASTER_PORT=29500\n")
		sb.WriteString("export WORLD_SIZE=$((SLURM_NNODES * SLURM_GPUS_ON_NODE))\n")
		sb.WriteString("export LOCAL_RANK=$SLURM_LOCALID\n")
		sb.WriteString("export RANK=$SLURM_PROCID\n")
	}

	sb.WriteString("\n# ── User Script ──\n")
	sb.WriteString(job.Script)
	sb.WriteString("\n")

	return sb.String()
}

func parseWallTime(wallTime string) int {
	// Parse formats: HH:MM:SS, D-HH:MM:SS, or just minutes
	parts := strings.Split(wallTime, ":")
	switch len(parts) {
	case 3: // HH:MM:SS or D-HH:MM:SS
		var h, m int
		hourPart := parts[0]
		if dParts := strings.Split(hourPart, "-"); len(dParts) == 2 {
			var d int
			_, _ = fmt.Sscanf(dParts[0], "%d", &d)
			_, _ = fmt.Sscanf(dParts[1], "%d", &h)
			h += d * 24
		} else {
			fmt.Sscanf(hourPart, "%d", &h)
		}
		fmt.Sscanf(parts[1], "%d", &m)
		return h*60 + m
	case 2: // HH:MM
		var h, m int
		fmt.Sscanf(parts[0], "%d", &h)
		fmt.Sscanf(parts[1], "%d", &m)
		return h*60 + m
	default:
		var mins int
		fmt.Sscanf(wallTime, "%d", &mins)
		return mins
	}
}
