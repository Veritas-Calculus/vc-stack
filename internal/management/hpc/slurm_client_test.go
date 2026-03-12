package hpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestBuildSubmitScript(t *testing.T) {
	sc := NewSlurmClient(zap.NewNop())

	job := &HPCJob{
		ID:             "slurm-job-1",
		Name:           "gpu-training",
		ProjectID:      "proj-gpu",
		UserID:         "user-1",
		Scheduler:      "slurm",
		ClusterID:      "sc1",
		CPUs:           16,
		MemoryMB:       65536,
		GPUs:           4,
		GPUType:        "A100",
		Nodes:          2,
		Script:         "python train.py --epochs 100",
		WorkDir:        "/scratch/projects/gpu-training",
		WallTimeLimit:  "48:00:00",
		SlurmPartition: "gpu-large",
	}

	mapping := &SlurmUserMapping{
		SlurmAccount: "gpu-project",
		QOS:          "high",
	}

	submit := sc.BuildSubmitScript(job, mapping)

	// Verify job parameters
	if submit.Job.Name != "gpu-training" {
		t.Errorf("expected name=gpu-training, got %s", submit.Job.Name)
	}
	if submit.Job.Account != "gpu-project" {
		t.Errorf("expected account=gpu-project, got %s", submit.Job.Account)
	}
	if submit.Job.Partition != "gpu-large" {
		t.Errorf("expected partition=gpu-large, got %s", submit.Job.Partition)
	}
	if submit.Job.Nodes != 2 {
		t.Errorf("expected nodes=2, got %d", submit.Job.Nodes)
	}
	if submit.Job.CPUsPerTask != 16 {
		t.Errorf("expected cpus_per_task=16, got %d", submit.Job.CPUsPerTask)
	}
	if submit.Job.Gres != "gpu:a100:4" {
		t.Errorf("expected gres=gpu:a100:4, got %s", submit.Job.Gres)
	}
	if submit.Job.QOS != "high" {
		t.Errorf("expected qos=high, got %s", submit.Job.QOS)
	}
	if submit.Job.TimeLimit != 2880 { // 48h = 2880 min
		t.Errorf("expected time_limit=2880, got %d", submit.Job.TimeLimit)
	}

	// Verify script content
	if !strings.Contains(submit.Script, "#!/bin/bash") {
		t.Error("script should start with shebang")
	}
	if !strings.Contains(submit.Script, "#SBATCH --gres=gpu:a100:4") {
		t.Error("script should contain GRES directive")
	}
	if !strings.Contains(submit.Script, "#SBATCH --nodes=2") {
		t.Error("script should contain nodes directive")
	}
	if !strings.Contains(submit.Script, "MASTER_ADDR") {
		t.Error("multi-node script should set MASTER_ADDR")
	}
	if !strings.Contains(submit.Script, "python train.py --epochs 100") {
		t.Error("script should contain user script")
	}
	if !strings.Contains(submit.Script, "VC_STACK_JOB_ID") {
		t.Error("script should set VC_STACK environment variables")
	}
}

func TestBuildSubmitScriptNoMapping(t *testing.T) {
	sc := NewSlurmClient(zap.NewNop())

	job := &HPCJob{
		ID:        "slurm-job-2",
		Name:      "simple-job",
		ProjectID: "proj-1",
		UserID:    "user-1",
		Scheduler: "slurm",
		ClusterID: "sc1",
		CPUs:      4,
		MemoryMB:  8192,
		Script:    "echo hello",
	}

	submit := sc.BuildSubmitScript(job, nil)

	if submit.Job.Account != "proj-1" {
		t.Errorf("without mapping, account should default to project_id, got %s", submit.Job.Account)
	}
	if submit.Job.QOS != "normal" {
		t.Errorf("without mapping, qos should default to normal, got %s", submit.Job.QOS)
	}
	if submit.Job.Gres != "" {
		t.Errorf("no GPUs, gres should be empty, got %s", submit.Job.Gres)
	}
}

func TestBuildUserSyncPlan(t *testing.T) {
	sc := NewSlurmClient(zap.NewNop())

	cluster := &SlurmCluster{
		ID:   "sc-sync",
		Name: "test-cluster",
	}

	mappings := []SlurmUserMapping{
		{
			SlurmUser:       "alice",
			SlurmAccount:    "proj-ml",
			SlurmPartitions: "gpu-small,gpu-large",
			QOS:             "high",
			MaxGPUs:         8,
			Status:          "active",
		},
		{
			SlurmUser:    "bob",
			SlurmAccount: "proj-ml",
			QOS:          "normal",
			MaxGPUs:      4,
			Status:       "active",
		},
		{
			SlurmUser: "charlie",
			Status:    "suspended", // should be skipped
		},
	}

	existingUsers := []string{"alice", "old-user"}

	plan := sc.BuildUserSyncPlan(cluster, mappings, existingUsers)

	if plan.ClusterID != "sc-sync" {
		t.Errorf("expected cluster_id=sc-sync, got %s", plan.ClusterID)
	}

	// alice exists -> update, bob is new -> add, charlie suspended -> skip
	if len(plan.UsersToAdd) != 1 {
		t.Errorf("expected 1 user to add, got %d", len(plan.UsersToAdd))
	}
	if plan.UsersToAdd[0].Name != "bob" {
		t.Errorf("expected bob to be added, got %s", plan.UsersToAdd[0].Name)
	}

	if len(plan.UsersToUpdate) != 1 {
		t.Errorf("expected 1 user to update, got %d", len(plan.UsersToUpdate))
	}
	if plan.UsersToUpdate[0].Name != "alice" {
		t.Errorf("expected alice to be updated, got %s", plan.UsersToUpdate[0].Name)
	}

	// old-user not in mappings -> remove
	if len(plan.UsersToRemove) != 1 {
		t.Errorf("expected 1 user to remove, got %d", len(plan.UsersToRemove))
	}
	if plan.UsersToRemove[0] != "old-user" {
		t.Errorf("expected old-user to be removed, got %s", plan.UsersToRemove[0])
	}

	// Both alice and bob map to proj-ml -> 1 unique account
	if len(plan.AccountsToAdd) != 1 {
		t.Errorf("expected 1 account, got %d", len(plan.AccountsToAdd))
	}

	// alice has 2 partitions -> 2 assocs, bob has no partition -> 1 assoc
	if plan.AssocChanges != 3 {
		t.Errorf("expected 3 association changes, got %d", plan.AssocChanges)
	}
}

func TestExecuteUserSync(t *testing.T) {
	sc := NewSlurmClient(zap.NewNop())

	plan := &UserSyncPlan{
		ClusterID: "sc-exec",
		AccountsToAdd: []SlurmRestAccount{
			{Name: "proj-ml"},
		},
		UsersToAdd: []SlurmRestUser{
			{Name: "bob", Associations: []SlurmAssoc{{Account: "proj-ml"}}},
		},
		UsersToUpdate: []SlurmRestUser{
			{Name: "alice", Associations: []SlurmAssoc{{Account: "proj-ml"}}},
		},
		UsersToRemove: []string{"old-user"},
	}

	result := sc.ExecuteUserSync(context.TODO(), plan)

	if result.OverallStatus != "success" {
		t.Errorf("expected status=success, got %s", result.OverallStatus)
	}
	if result.AccountsCreated != 1 {
		t.Errorf("expected 1 account created, got %d", result.AccountsCreated)
	}
	if result.UsersCreated != 1 {
		t.Errorf("expected 1 user created, got %d", result.UsersCreated)
	}
	if result.UsersUpdated != 1 {
		t.Errorf("expected 1 user updated, got %d", result.UsersUpdated)
	}
	if result.UsersRemoved != 1 {
		t.Errorf("expected 1 user removed, got %d", result.UsersRemoved)
	}
	if len(result.Actions) != 4 {
		t.Errorf("expected 4 actions, got %d", len(result.Actions))
	}
}

func TestExecuteUserSyncNoChanges(t *testing.T) {
	sc := NewSlurmClient(zap.NewNop())

	plan := &UserSyncPlan{
		ClusterID:     "sc-noop",
		AccountsToAdd: []SlurmRestAccount{},
		UsersToAdd:    []SlurmRestUser{},
		UsersToUpdate: []SlurmRestUser{},
		UsersToRemove: []string{},
	}

	result := sc.ExecuteUserSync(context.TODO(), plan)

	if result.OverallStatus != "no_changes" {
		t.Errorf("expected status=no_changes, got %s", result.OverallStatus)
	}
}

func TestMapSlurmJobState(t *testing.T) {
	tests := []struct {
		states []string
		want   string
	}{
		{[]string{"PENDING"}, "queued"},
		{[]string{"RUNNING"}, "running"},
		{[]string{"COMPLETING"}, "running"},
		{[]string{"COMPLETED"}, "completed"},
		{[]string{"FAILED"}, "failed"},
		{[]string{"CANCELLED"}, "cancelled"},
		{[]string{"TIMEOUT"}, "failed"},
		{[]string{"OUT_OF_MEMORY"}, "failed"},
		{[]string{"SUSPENDED"}, "queued"},
		{[]string{"REQUEUED"}, "queued"},
		{[]string{}, "pending"},
	}

	for _, tt := range tests {
		got := MapSlurmJobState(tt.states)
		if got != tt.want {
			t.Errorf("MapSlurmJobState(%v) = %s, want %s", tt.states, got, tt.want)
		}
	}
}

func TestMapSlurmNodeState(t *testing.T) {
	tests := []struct {
		states []string
		want   string
	}{
		{[]string{"IDLE"}, "idle"},
		{[]string{"ALLOCATED"}, "alloc"},
		{[]string{"MIXED"}, "mix"},
		{[]string{"DOWN"}, "down"},
		{[]string{"DRAINED"}, "drain"},
		{[]string{"MAINTENANCE"}, "maint"},
		{[]string{}, "unknown"},
	}

	for _, tt := range tests {
		got := MapSlurmNodeState(tt.states)
		if got != tt.want {
			t.Errorf("MapSlurmNodeState(%v) = %s, want %s", tt.states, got, tt.want)
		}
	}
}

func TestParseGresString(t *testing.T) {
	tests := []struct {
		gres      string
		wantType  string
		wantCount int
	}{
		{"gpu:a100:4", "A100", 4},
		{"gpu:h100:8", "H100", 8},
		{"gpu:2", "", 2},
		{"", "", 0},
		{"cpu:8", "", 0},
	}

	for _, tt := range tests {
		gotType, gotCount := ParseGresString(tt.gres)
		if gotType != tt.wantType || gotCount != tt.wantCount {
			t.Errorf("ParseGresString(%q) = (%s, %d), want (%s, %d)",
				tt.gres, gotType, gotCount, tt.wantType, tt.wantCount)
		}
	}
}

func TestParseWallTime(t *testing.T) {
	tests := []struct {
		input string
		want  int // minutes
	}{
		{"24:00:00", 1440},
		{"48:00:00", 2880},
		{"1:30:00", 90},
		{"2-12:00:00", 3600}, // 2 days + 12h
		{"12:30", 750},
		{"60", 60},
	}

	for _, tt := range tests {
		got := parseWallTime(tt.input)
		if got != tt.want {
			t.Errorf("parseWallTime(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSlurmUserSyncAPIEndpoint(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create Slurm cluster
	body, _ := json.Marshal(map[string]interface{}{"name": "sync-cluster"})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Add a user mapping
	userBody, _ := json.Marshal(map[string]interface{}{
		"username": "testuser",
		"qos":      "normal",
		"max_gpus": 4,
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters/"+clusterID+"/users", bytes.NewBuffer(userBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("add user: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	// Trigger user sync
	req3, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters/"+clusterID+"/sync-users", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("sync: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
	var syncResp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &syncResp)
	result := syncResp["result"].(map[string]interface{})
	if result["overall_status"] != "success" {
		t.Errorf("expected sync status=success, got %v", result["overall_status"])
	}
}

func TestSlurmNodeManagement(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create Slurm cluster
	body, _ := json.Marshal(map[string]interface{}{"name": "node-cluster"})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Add a Slurm compute node
	nodeBody, _ := json.Marshal(map[string]interface{}{
		"hostname":   "gpu-node-01",
		"ip_address": "10.0.2.10",
		"cpus":       64,
		"memory_mb":  524288,
		"gpu_count":  8,
		"gpu_type":   "A100",
		"partitions": "gpu-large",
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters/"+clusterID+"/nodes", bytes.NewBuffer(nodeBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("add node: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	// List Slurm nodes
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/slurm/clusters/"+clusterID+"/nodes", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("list nodes: expected 200, got %d", w3.Code)
	}
	var nodeResp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &nodeResp)
	nodes := nodeResp["nodes"].([]interface{})
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	node := nodes[0].(map[string]interface{})
	if node["hostname"] != "gpu-node-01" {
		t.Errorf("expected hostname=gpu-node-01, got %v", node["hostname"])
	}
}

func TestSlurmSbatchEndpoint(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create Slurm cluster
	body, _ := json.Marshal(map[string]interface{}{"name": "sbatch-cluster"})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Submit a Slurm job
	jobBody, _ := json.Marshal(map[string]interface{}{
		"name":            "slurm-train",
		"scheduler":       "slurm",
		"cluster_id":      clusterID,
		"gpus":            4,
		"gpu_type":        "A100",
		"cpus":            16,
		"memory_mb":       65536,
		"nodes":           2,
		"script":          "python train.py",
		"wall_time_limit": "24:00:00",
		"slurm_partition": "gpu",
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/jobs", bytes.NewBuffer(jobBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("submit: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
	var jobResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &jobResp)
	jobID := jobResp["job"].(map[string]interface{})["id"].(string)

	// Get sbatch script
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/jobs/"+jobID+"/sbatch", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("sbatch: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
	var sbatchResp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &sbatchResp)
	script := sbatchResp["sbatch_script"].(string)
	if !strings.Contains(script, "#SBATCH --gres=gpu:a100:4") {
		t.Error("sbatch script should contain GRES directive")
	}
	if !strings.Contains(script, "#SBATCH --partition=gpu") {
		t.Error("sbatch script should contain partition directive")
	}

	// K8s job should reject sbatch endpoint
	k8sBody, _ := json.Marshal(map[string]interface{}{
		"name":       "k8s-only",
		"scheduler":  "kubernetes",
		"cluster_id": clusterID,
		"script":     "echo hi",
		"image":      "alpine",
	})
	req4, _ := http.NewRequest("POST", "/api/v1/hpc/jobs", bytes.NewBuffer(k8sBody))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	var k8sResp map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &k8sResp)
	k8sJobID := k8sResp["job"].(map[string]interface{})["id"].(string)

	req5, _ := http.NewRequest("GET", "/api/v1/hpc/jobs/"+k8sJobID+"/sbatch", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusBadRequest {
		t.Errorf("k8s job sbatch: expected 400, got %d", w5.Code)
	}
}
