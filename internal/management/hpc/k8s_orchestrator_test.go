package hpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestBuildComponentsForCluster(t *testing.T) {
	orch := NewK8sOrchestrator(zap.NewNop())

	tests := []struct {
		name           string
		cluster        *HPCKubernetesCluster
		gpuPools       []HPCGPUPool
		wantScheduler  string
		wantMPI        bool
		wantNCCL       bool
		wantRDMA       bool
		wantMonitoring bool
	}{
		{
			name: "volcano with MPI and RDMA",
			cluster: &HPCKubernetesCluster{
				ID:           "c1",
				GPUScheduler: "volcano",
				EnableMPI:    true,
				EnableRDMA:   true,
				TotalGPUs:    8,
			},
			gpuPools:       nil,
			wantScheduler:  "volcano",
			wantMPI:        true,
			wantNCCL:       true,
			wantRDMA:       true,
			wantMonitoring: true,
		},
		{
			name: "kueue without MPI",
			cluster: &HPCKubernetesCluster{
				ID:           "c2",
				GPUScheduler: "kueue",
				EnableMPI:    false,
				EnableRDMA:   false,
				TotalGPUs:    4,
			},
			gpuPools:       nil,
			wantScheduler:  "kueue",
			wantMPI:        false,
			wantNCCL:       true, // multi-GPU → NCCL
			wantRDMA:       false,
			wantMonitoring: true,
		},
		{
			name: "with MIG GPU pool",
			cluster: &HPCKubernetesCluster{
				ID:           "c3",
				GPUScheduler: "volcano",
				TotalGPUs:    8,
			},
			gpuPools: []HPCGPUPool{
				{MIGEnabled: true, MIGProfile: "1g.5gb"},
			},
			wantScheduler:  "volcano",
			wantMPI:        false,
			wantNCCL:       true,
			wantRDMA:       false,
			wantMonitoring: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := orch.BuildComponentsForCluster(tt.cluster, tt.gpuPools)

			if components.ClusterID != tt.cluster.ID {
				t.Errorf("expected cluster_id=%s, got %s", tt.cluster.ID, components.ClusterID)
			}
			if components.GPUScheduler == nil {
				t.Fatal("GPUScheduler should not be nil")
			}
			if components.GPUScheduler.Name != tt.wantScheduler {
				t.Errorf("expected scheduler=%s, got %s", tt.wantScheduler, components.GPUScheduler.Name)
			}
			if (components.MPIOperator != nil) != tt.wantMPI {
				t.Errorf("expected MPI=%v, got %v", tt.wantMPI, components.MPIOperator != nil)
			}
			if (components.NCCLPlugin != nil) != tt.wantNCCL {
				t.Errorf("expected NCCL=%v, got %v", tt.wantNCCL, components.NCCLPlugin != nil)
			}
			if (components.RDMADevicePlugin != nil) != tt.wantRDMA {
				t.Errorf("expected RDMA=%v, got %v", tt.wantRDMA, components.RDMADevicePlugin != nil)
			}
			if (components.Monitoring != nil) != tt.wantMonitoring {
				t.Errorf("expected Monitoring=%v, got %v", tt.wantMonitoring, components.Monitoring != nil)
			}
			if components.GPUDevicePlugin == nil {
				t.Error("GPUDevicePlugin should always be present")
			}
		})
	}
}

func TestBuildVolcanoJobManifest(t *testing.T) {
	orch := NewK8sOrchestrator(zap.NewNop())

	job := &HPCJob{
		ID:         "job-1",
		Name:       "train-gpt",
		ProjectID:  "proj-1",
		UserID:     "user-1",
		Scheduler:  "kubernetes",
		ClusterID:  "c1",
		CPUs:       8,
		MemoryMB:   32768,
		GPUs:       4,
		GPUType:    "A100-80GB",
		Nodes:      2,
		Script:     "python train.py --distributed",
		Image:      "pytorch:2.2-cuda12.1",
		WorkDir:    "/workspace",
		K8sJobName: "train-gpt-job-1",
	}

	manifest := orch.BuildVolcanoJobManifest(job)

	if manifest.APIVersion != "batch.volcano.sh/v1alpha1" {
		t.Errorf("expected apiVersion=batch.volcano.sh/v1alpha1, got %s", manifest.APIVersion)
	}
	if manifest.Kind != "Job" {
		t.Errorf("expected kind=Job, got %s", manifest.Kind)
	}
	if manifest.Metadata["name"] != "train-gpt-job-1" {
		t.Errorf("expected name=train-gpt-job-1, got %v", manifest.Metadata["name"])
	}
	if manifest.Metadata["namespace"] != "hpc-proj-1" {
		t.Errorf("expected namespace=hpc-proj-1, got %v", manifest.Metadata["namespace"])
	}

	// Verify serialization doesn't panic
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if len(data) < 100 {
		t.Error("manifest seems too small")
	}
}

func TestBuildMPIJobManifest(t *testing.T) {
	orch := NewK8sOrchestrator(zap.NewNop())

	job := &HPCJob{
		ID:         "mpi-job-1",
		Name:       "nccl-test",
		ProjectID:  "proj-1",
		UserID:     "user-1",
		Scheduler:  "kubernetes",
		ClusterID:  "c1",
		CPUs:       16,
		MemoryMB:   65536,
		GPUs:       8,
		GPUType:    "H100",
		Nodes:      4,
		Script:     "torchrun --nproc_per_node=8 train.py",
		Image:      "nvcr.io/nvidia/pytorch:24.01-py3",
		K8sJobName: "nccl-test-mpi-1",
	}

	manifest := orch.BuildMPIJobManifest(job)

	if manifest.APIVersion != "kubeflow.org/v2beta1" {
		t.Errorf("expected apiVersion=kubeflow.org/v2beta1, got %s", manifest.APIVersion)
	}
	if manifest.Kind != "MPIJob" {
		t.Errorf("expected kind=MPIJob, got %s", manifest.Kind)
	}

	spec := manifest.Spec
	slotsPerWorker := spec["slotsPerWorker"]
	if slotsPerWorker != 8 {
		t.Errorf("expected slotsPerWorker=8, got %v", slotsPerWorker)
	}

	replicas := spec["mpiReplicaSpecs"].(map[string]interface{})
	if _, ok := replicas["Worker"]; !ok {
		t.Error("expected Worker replica spec")
	}
	if _, ok := replicas["Launcher"]; !ok {
		t.Error("expected Launcher replica spec")
	}
}

func TestReconcileCluster(t *testing.T) {
	orch := NewK8sOrchestrator(zap.NewNop())

	cluster := &HPCKubernetesCluster{
		ID:           "c-recon",
		GPUScheduler: "volcano",
		EnableMPI:    true,
		TotalGPUs:    4,
	}
	components := orch.BuildComponentsForCluster(cluster, nil)
	result := orch.ReconcileCluster(context.TODO(), components)

	if result.OverallStatus != "ready" {
		t.Errorf("expected overall_status=ready, got %s", result.OverallStatus)
	}
	if result.ComponentsReady != result.ComponentsTotal {
		t.Errorf("expected all components ready (%d/%d)",
			result.ComponentsReady, result.ComponentsTotal)
	}
	if len(result.Actions) == 0 {
		t.Error("expected at least one reconcile action")
	}

	// Verify each action was install + success
	for _, a := range result.Actions {
		if a.Action != "install" {
			t.Errorf("expected action=install, got %s for %s", a.Action, a.Component)
		}
		if a.Status != "success" {
			t.Errorf("expected status=success, got %s for %s", a.Status, a.Component)
		}
	}
}

func TestReconcileIdempotent(t *testing.T) {
	orch := NewK8sOrchestrator(zap.NewNop())

	cluster := &HPCKubernetesCluster{ID: "c-idem", GPUScheduler: "volcano", TotalGPUs: 2}
	components := orch.BuildComponentsForCluster(cluster, nil)

	// First reconcile — installs
	result1 := orch.ReconcileCluster(context.TODO(), components)
	if result1.OverallStatus != "ready" {
		t.Fatalf("first reconcile should be ready, got %s", result1.OverallStatus)
	}

	// Second reconcile — idempotent (all skip)
	result2 := orch.ReconcileCluster(context.TODO(), components)
	if result2.OverallStatus != "ready" {
		t.Errorf("second reconcile should be ready, got %s", result2.OverallStatus)
	}
	for _, a := range result2.Actions {
		if a.Action != "skip" {
			t.Errorf("expected action=skip on second reconcile, got %s for %s", a.Action, a.Component)
		}
	}
}

func TestGPUTopologyDiscovery(t *testing.T) {
	orch := NewK8sOrchestrator(zap.NewNop())

	// A100 8-GPU node — should have NVLink pairs
	topo := orch.DiscoverGPUTopology("gpu-node-1", 8, "A100-80GB")

	if len(topo.GPUDevices) != 8 {
		t.Errorf("expected 8 GPUs, got %d", len(topo.GPUDevices))
	}
	if len(topo.NVLinkPairs) == 0 {
		t.Error("expected NVLink pairs for 8-GPU A100 node")
	}
	// 8 GPUs: C(8,2) = 28 pairs
	if len(topo.NVLinkPairs) != 28 {
		t.Errorf("expected 28 NVLink pairs, got %d", len(topo.NVLinkPairs))
	}
	if topo.NVLinkPairs[0].Version != 3 {
		t.Errorf("expected NVLink v3 for A100, got %d", topo.NVLinkPairs[0].Version)
	}
	if topo.GPUDevices[0].MemoryMB != 81920 {
		t.Errorf("expected 81920 MB for A100-80GB, got %d", topo.GPUDevices[0].MemoryMB)
	}

	// V100 2-GPU node — no NVLink (< 4 GPUs)
	topo2 := orch.DiscoverGPUTopology("v100-node", 2, "V100")
	if len(topo2.NVLinkPairs) != 0 {
		t.Errorf("expected 0 NVLink pairs for 2-GPU V100, got %d", len(topo2.NVLinkPairs))
	}
	if topo2.GPUDevices[0].MemoryMB != 32768 {
		t.Errorf("expected 32768 MB for V100, got %d", topo2.GPUDevices[0].MemoryMB)
	}
}

func TestComponentsAPIEndpoint(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create cluster
	body, _ := json.Marshal(map[string]interface{}{
		"name":          "comp-test",
		"gpu_scheduler": "volcano",
		"enable_mpi":    true,
	})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Get components
	req2, _ := http.NewRequest("GET", "/api/v1/hpc/kubernetes/clusters/"+clusterID+"/components", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("components: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var compResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &compResp)
	components := compResp["components"].(map[string]interface{})
	if components["gpu_scheduler"] == nil {
		t.Error("expected gpu_scheduler component")
	}
	if components["mpi_operator"] == nil {
		t.Error("expected mpi_operator for MPI-enabled cluster")
	}
}

func TestReconcileAPIEndpoint(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create cluster
	body, _ := json.Marshal(map[string]interface{}{"name": "recon-test"})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Reconcile
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters/"+clusterID+"/reconcile", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("reconcile: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var reconResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &reconResp)
	result := reconResp["result"].(map[string]interface{})
	if result["overall_status"] != "ready" {
		t.Errorf("expected overall_status=ready, got %v", result["overall_status"])
	}
}

func TestJobManifestGeneration(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create cluster with MPI
	body, _ := json.Marshal(map[string]interface{}{
		"name":       "manifest-cluster",
		"enable_mpi": true,
	})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Submit a single-node job → Volcano
	jobBody, _ := json.Marshal(map[string]interface{}{
		"name":       "single-gpu-job",
		"scheduler":  "kubernetes",
		"cluster_id": clusterID,
		"gpus":       1,
		"image":      "pytorch:latest",
		"script":     "python train.py",
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/jobs", bytes.NewBuffer(jobBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var jobResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &jobResp)
	jobID := jobResp["job"].(map[string]interface{})["id"].(string)

	// Get Volcano manifest
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/jobs/"+jobID+"/manifest", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("manifest: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
	var manResp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &manResp)
	if manResp["kind"] != "VolcanoJob" {
		t.Errorf("expected kind=VolcanoJob for single-node, got %v", manResp["kind"])
	}

	// Submit a multi-node MPI job → MPIJob
	mpiBody, _ := json.Marshal(map[string]interface{}{
		"name":       "distributed-train",
		"scheduler":  "kubernetes",
		"cluster_id": clusterID,
		"gpus":       8,
		"nodes":      4,
		"image":      "pytorch:latest",
		"script":     "torchrun train.py",
	})
	req4, _ := http.NewRequest("POST", "/api/v1/hpc/jobs", bytes.NewBuffer(mpiBody))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	var mpiResp map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &mpiResp)
	mpiJobID := mpiResp["job"].(map[string]interface{})["id"].(string)

	req5, _ := http.NewRequest("GET", "/api/v1/hpc/jobs/"+mpiJobID+"/manifest", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("mpi manifest: expected 200, got %d: %s", w5.Code, w5.Body.String())
	}
	var mpiManResp map[string]interface{}
	json.Unmarshal(w5.Body.Bytes(), &mpiManResp)
	if mpiManResp["kind"] != "MPIJob" {
		t.Errorf("expected kind=MPIJob for multi-node MPI, got %v", mpiManResp["kind"])
	}
}
