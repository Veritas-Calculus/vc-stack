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
// K8s Orchestration Client — manages HPC components on K8s clusters
// ──────────────────────────────────────────────────────────────────

// K8sOrchestrator manages the lifecycle of HPC-specific components on K8s clusters.
// It handles Volcano scheduler, MPI Operator, NVIDIA GPU device plugin, and NCCL/RDMA setup.
type K8sOrchestrator struct {
	logger *zap.Logger
}

// NewK8sOrchestrator creates a new K8s HPC orchestrator.
func NewK8sOrchestrator(logger *zap.Logger) *K8sOrchestrator {
	return &K8sOrchestrator{logger: logger}
}

// HPCComponentSpec defines a deployable HPC component on a K8s cluster.
type HPCComponentSpec struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Version    string            `json:"version"`
	HelmChart  string            `json:"helm_chart"`
	HelmRepo   string            `json:"helm_repo"`
	ValuesYAML string            `json:"values_yaml"`
	Labels     map[string]string `json:"labels"`
	Status     string            `json:"status"` // pending, deploying, ready, failed, removing
	Message    string            `json:"message"`
}

// HPCClusterComponents holds all HPC-specific component states for a cluster.
type HPCClusterComponents struct {
	ClusterID        string            `json:"cluster_id"`
	GPUScheduler     *HPCComponentSpec `json:"gpu_scheduler"`
	GPUDevicePlugin  *HPCComponentSpec `json:"gpu_device_plugin"`
	MPIOperator      *HPCComponentSpec `json:"mpi_operator,omitempty"`
	NCCLPlugin       *HPCComponentSpec `json:"nccl_plugin,omitempty"`
	RDMADevicePlugin *HPCComponentSpec `json:"rdma_device_plugin,omitempty"`
	SharedFS         *HPCComponentSpec `json:"shared_fs,omitempty"`
	Monitoring       *HPCComponentSpec `json:"monitoring,omitempty"`
	LastReconciled   time.Time         `json:"last_reconciled"`
}

// ──────────────────────────────────────────────────────────────────
// Component Specification Builders
// ──────────────────────────────────────────────────────────────────

// BuildVolcanoSpec creates the Volcano scheduler component spec.
func (o *K8sOrchestrator) BuildVolcanoSpec() *HPCComponentSpec {
	return &HPCComponentSpec{
		Name:      "volcano",
		Namespace: "volcano-system",
		Version:   "1.9.0",
		HelmChart: "volcano",
		HelmRepo:  "https://volcano-sh.github.io/charts",
		ValuesYAML: marshallValues(map[string]interface{}{
			"scheduler": map[string]interface{}{
				"replicas": 1,
				"plugins": map[string]interface{}{
					"gang":          map[string]interface{}{"enabled": true},
					"priority":      map[string]interface{}{"enabled": true},
					"drf":           map[string]interface{}{"enabled": true},
					"predicates":    map[string]interface{}{"enabled": true},
					"proportion":    map[string]interface{}{"enabled": true},
					"nodeorder":     map[string]interface{}{"enabled": true},
					"binpack":       map[string]interface{}{"enabled": true},
					"tdm":           map[string]interface{}{"enabled": true},
					"gpudevice":     map[string]interface{}{"enabled": true},
					"sla":           map[string]interface{}{"enabled": true},
					"resourcequota": map[string]interface{}{"enabled": true},
				},
			},
			"controller": map[string]interface{}{
				"replicas": 1,
			},
		}),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// BuildKueueSpec creates the Kueue scheduler component spec.
func (o *K8sOrchestrator) BuildKueueSpec() *HPCComponentSpec {
	return &HPCComponentSpec{
		Name:      "kueue",
		Namespace: "kueue-system",
		Version:   "0.8.1",
		HelmChart: "kueue",
		HelmRepo:  "https://kubernetes-sigs.github.io/kueue/charts",
		ValuesYAML: marshallValues(map[string]interface{}{
			"enableVisibilityAPIs": true,
			"controller": map[string]interface{}{
				"replicas": 1,
			},
		}),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// BuildNVIDIADevicePluginSpec creates the NVIDIA GPU device plugin spec.
func (o *K8sOrchestrator) BuildNVIDIADevicePluginSpec(migEnabled bool, migProfile string) *HPCComponentSpec {
	values := map[string]interface{}{
		"nfd": map[string]interface{}{
			"enabled": true,
		},
		"gfd": map[string]interface{}{
			"enabled": true,
		},
		"devicePlugin": map[string]interface{}{
			"enabled": true,
		},
		"toolkit": map[string]interface{}{
			"enabled": true,
			"version": "v1.16.2-ubuntu22.04",
		},
	}
	if migEnabled {
		values["migStrategy"] = "mixed"
		values["migManager"] = map[string]interface{}{
			"enabled": true,
			"config": map[string]interface{}{
				"default": migProfile,
			},
		}
	}
	return &HPCComponentSpec{
		Name:       "nvidia-gpu-operator",
		Namespace:  "gpu-operator",
		Version:    "24.9.0",
		HelmChart:  "gpu-operator",
		HelmRepo:   "https://helm.ngc.nvidia.com/nvidia",
		ValuesYAML: marshallValues(values),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// BuildMPIOperatorSpec creates the MPI Operator component spec.
func (o *K8sOrchestrator) BuildMPIOperatorSpec() *HPCComponentSpec {
	return &HPCComponentSpec{
		Name:      "mpi-operator",
		Namespace: "mpi-operator",
		Version:   "0.5.0",
		HelmChart: "mpi-operator",
		HelmRepo:  "https://kubeflow.github.io/mpi-operator",
		ValuesYAML: marshallValues(map[string]interface{}{
			"replicas":              1,
			"launcherRunsWorkloads": false,
		}),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// BuildNCCLPluginSpec creates the NVIDIA NCCL plugin spec for multi-GPU comm.
func (o *K8sOrchestrator) BuildNCCLPluginSpec() *HPCComponentSpec {
	return &HPCComponentSpec{
		Name:      "nccl-plugin",
		Namespace: "kube-system",
		Version:   "2.21.5",
		HelmChart: "nccl-plugin",
		HelmRepo:  "https://helm.ngc.nvidia.com/nvidia",
		ValuesYAML: marshallValues(map[string]interface{}{
			"nccl": map[string]interface{}{
				"sharpEnabled": false,
			},
		}),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// BuildRDMADevicePluginSpec creates the RDMA/InfiniBand device plugin spec.
func (o *K8sOrchestrator) BuildRDMADevicePluginSpec() *HPCComponentSpec {
	return &HPCComponentSpec{
		Name:      "rdma-shared-device-plugin",
		Namespace: "kube-system",
		Version:   "1.5.1",
		HelmChart: "network-operator",
		HelmRepo:  "https://helm.ngc.nvidia.com/nvidia",
		ValuesYAML: marshallValues(map[string]interface{}{
			"rdmaSharedDevicePlugin": map[string]interface{}{
				"enabled": true,
				"resources": []map[string]interface{}{
					{
						"name":    "rdma_shared_device_a",
						"vendors": []string{"15b3"},
					},
				},
			},
			"sriovDevicePlugin": map[string]interface{}{
				"enabled": false,
			},
		}),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// BuildGPUMonitoringSpec creates the DCGM-Exporter monitoring component spec.
func (o *K8sOrchestrator) BuildGPUMonitoringSpec() *HPCComponentSpec {
	return &HPCComponentSpec{
		Name:      "dcgm-exporter",
		Namespace: "monitoring",
		Version:   "3.3.9",
		HelmChart: "dcgm-exporter",
		HelmRepo:  "https://nvidia.github.io/dcgm-exporter/helm-charts",
		ValuesYAML: marshallValues(map[string]interface{}{
			"serviceMonitor": map[string]interface{}{
				"enabled": true,
			},
			"extraEnv": []map[string]interface{}{
				{"name": "DCGM_EXPORTER_KUBERNETES", "value": "true"},
				{"name": "DCGM_EXPORTER_LISTEN", "value": ":9400"},
			},
		}),
		Labels: map[string]string{
			"app.kubernetes.io/part-of":    "vc-hpc",
			"app.kubernetes.io/managed-by": "vc-stack",
		},
		Status: "pending",
	}
}

// ──────────────────────────────────────────────────────────────────
// Cluster Provisioning Plan
// ──────────────────────────────────────────────────────────────────

// BuildComponentsForCluster creates the full HPC component plan for a cluster.
func (o *K8sOrchestrator) BuildComponentsForCluster(cluster *HPCKubernetesCluster, gpuPools []HPCGPUPool) *HPCClusterComponents {
	components := &HPCClusterComponents{
		ClusterID:      cluster.ID,
		LastReconciled: time.Now(),
	}

	// 1. GPU Scheduler — Volcano or Kueue
	switch strings.ToLower(cluster.GPUScheduler) {
	case "volcano":
		components.GPUScheduler = o.BuildVolcanoSpec()
	case "kueue":
		components.GPUScheduler = o.BuildKueueSpec()
	default:
		components.GPUScheduler = o.BuildVolcanoSpec()
	}

	// 2. GPU Device Plugin — always needed for GPU workloads
	hasMIG := false
	migProfile := ""
	for _, pool := range gpuPools {
		if pool.MIGEnabled {
			hasMIG = true
			migProfile = pool.MIGProfile
			break
		}
	}
	components.GPUDevicePlugin = o.BuildNVIDIADevicePluginSpec(hasMIG, migProfile)

	// 3. MPI Operator — if MPI enabled
	if cluster.EnableMPI {
		components.MPIOperator = o.BuildMPIOperatorSpec()
	}

	// 4. NCCL Plugin — if multi-GPU or MPI enabled
	if cluster.EnableMPI || cluster.TotalGPUs > 1 {
		components.NCCLPlugin = o.BuildNCCLPluginSpec()
	}

	// 5. RDMA/InfiniBand — if RDMA enabled
	if cluster.EnableRDMA {
		components.RDMADevicePlugin = o.BuildRDMADevicePluginSpec()
	}

	// 6. GPU Monitoring — DCGM Exporter (always)
	components.Monitoring = o.BuildGPUMonitoringSpec()

	o.logger.Info("HPC component plan built",
		zap.String("cluster_id", cluster.ID),
		zap.String("scheduler", cluster.GPUScheduler),
		zap.Bool("mpi", cluster.EnableMPI),
		zap.Bool("rdma", cluster.EnableRDMA),
		zap.Bool("mig", hasMIG))

	return components
}

// ──────────────────────────────────────────────────────────────────
// Job Manifest Generation
// ──────────────────────────────────────────────────────────────────

// VolcanoJobManifest represents a Volcano Job (vcjob) manifest.
type VolcanoJobManifest struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
}

// MPIJobManifest represents a Kubeflow MPIJob manifest.
type MPIJobManifest struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
}

// BuildVolcanoJobManifest creates a Volcano vcjob manifest for an HPC job.
func (o *K8sOrchestrator) BuildVolcanoJobManifest(job *HPCJob) *VolcanoJobManifest {
	namespace := job.K8sNamespace
	if namespace == "" {
		namespace = fmt.Sprintf("hpc-%s", job.ProjectID)
	}

	// Build resource requests
	resources := map[string]interface{}{
		"cpu":    fmt.Sprintf("%d", max(job.CPUs, 1)),
		"memory": fmt.Sprintf("%dMi", max(job.MemoryMB, 1024)),
	}
	if job.GPUs > 0 {
		resources["nvidia.com/gpu"] = fmt.Sprintf("%d", job.GPUs)
	}

	// Build environment variables
	envVars := []map[string]interface{}{}
	if job.EnvVars != "" {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(job.EnvVars), &parsed); err == nil {
			for k, v := range parsed {
				envVars = append(envVars, map[string]interface{}{
					"name": k, "value": v,
				})
			}
		}
	}
	// Standard NCCL env vars for multi-GPU
	if job.GPUs > 1 || job.Nodes > 1 {
		envVars = append(envVars,
			map[string]interface{}{"name": "NCCL_DEBUG", "value": "INFO"},
			map[string]interface{}{"name": "NCCL_IB_DISABLE", "value": "0"},
			map[string]interface{}{"name": "NCCL_SOCKET_IFNAME", "value": "eth0"},
		)
	}

	// Build volume mounts for shared FS
	volumeMounts := []map[string]interface{}{}
	volumes := []map[string]interface{}{}
	if job.WorkDir != "" {
		volumeMounts = append(volumeMounts, map[string]interface{}{
			"name":      "workspace",
			"mountPath": job.WorkDir,
		})
		volumes = append(volumes, map[string]interface{}{
			"name": "workspace",
			"persistentVolumeClaim": map[string]interface{}{
				"claimName": fmt.Sprintf("hpc-workspace-%s", job.ID),
			},
		})
	}
	// Shared memory for NCCL (required for multi-GPU training)
	if job.GPUs > 0 {
		volumeMounts = append(volumeMounts, map[string]interface{}{
			"name":      "dshm",
			"mountPath": "/dev/shm",
		})
		volumes = append(volumes, map[string]interface{}{
			"name": "dshm",
			"emptyDir": map[string]interface{}{
				"medium":    "Memory",
				"sizeLimit": "16Gi",
			},
		})
	}

	// Build task spec
	taskSpec := map[string]interface{}{
		"replicas": max(job.Nodes, 1),
		"name":     "worker",
		"template": map[string]interface{}{
			"spec": map[string]interface{}{
				"restartPolicy": "Never",
				"containers": []map[string]interface{}{
					{
						"name":    "main",
						"image":   job.Image,
						"command": []string{"/bin/bash", "-c", job.Script},
						"resources": map[string]interface{}{
							"limits":   resources,
							"requests": resources,
						},
						"env":          envVars,
						"volumeMounts": volumeMounts,
					},
				},
				"volumes": volumes,
			},
		},
	}

	// Add scheduler name for GPU-specific scheduling
	if job.GPUType != "" {
		taskSpec["template"].(map[string]interface{})["spec"].(map[string]interface{})["nodeSelector"] = map[string]interface{}{
			"nvidia.com/gpu.product": job.GPUType,
		}
	}

	// Build main policies
	policies := []map[string]interface{}{
		{"event": "PodEvicted", "action": "RestartJob"},
		{"event": "PodFailed", "action": "RestartJob"},
		{"event": "TaskCompleted", "action": "CompleteJob"},
	}

	manifest := &VolcanoJobManifest{
		APIVersion: "batch.volcano.sh/v1alpha1",
		Kind:       "Job",
		Metadata: map[string]interface{}{
			"name":      job.K8sJobName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "vc-stack",
				"vc-stack.io/project-id":       job.ProjectID,
				"vc-stack.io/job-id":           job.ID,
				"vc-stack.io/user-id":          job.UserID,
			},
		},
		Spec: map[string]interface{}{
			"schedulerName":           "volcano",
			"minAvailable":            max(job.Nodes, 1),
			"maxRetry":                3,
			"queue":                   "default",
			"ttlSecondsAfterFinished": 3600,
			"tasks":                   []interface{}{taskSpec},
			"policies":                policies,
		},
	}

	o.logger.Info("Built Volcano job manifest",
		zap.String("job_id", job.ID),
		zap.String("name", job.K8sJobName),
		zap.Int("gpus", job.GPUs),
		zap.Int("nodes", job.Nodes))

	return manifest
}

// BuildMPIJobManifest creates a Kubeflow MPIJob manifest for distributed training.
func (o *K8sOrchestrator) BuildMPIJobManifest(job *HPCJob) *MPIJobManifest {
	namespace := job.K8sNamespace
	if namespace == "" {
		namespace = fmt.Sprintf("hpc-%s", job.ProjectID)
	}

	// Build resource requests
	resources := map[string]interface{}{
		"cpu":    fmt.Sprintf("%d", max(job.CPUs, 1)),
		"memory": fmt.Sprintf("%dMi", max(job.MemoryMB, 1024)),
	}
	if job.GPUs > 0 {
		resources["nvidia.com/gpu"] = fmt.Sprintf("%d", job.GPUs)
	}

	// Build environment variables
	envVars := []map[string]interface{}{
		{"name": "NCCL_DEBUG", "value": "INFO"},
		{"name": "OMPI_MCA_btl_tcp_if_include", "value": "eth0"},
	}
	if job.EnvVars != "" {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(job.EnvVars), &parsed); err == nil {
			for k, v := range parsed {
				envVars = append(envVars, map[string]interface{}{
					"name": k, "value": v,
				})
			}
		}
	}

	// Volume mounts for shared memory
	volumeMounts := []map[string]interface{}{
		{"name": "dshm", "mountPath": "/dev/shm"},
	}
	volumes := []map[string]interface{}{
		{"name": "dshm", "emptyDir": map[string]interface{}{
			"medium":    "Memory",
			"sizeLimit": "16Gi",
		}},
	}

	if job.WorkDir != "" {
		volumeMounts = append(volumeMounts, map[string]interface{}{
			"name":      "workspace",
			"mountPath": job.WorkDir,
		})
		volumes = append(volumes, map[string]interface{}{
			"name": "workspace",
			"persistentVolumeClaim": map[string]interface{}{
				"claimName": fmt.Sprintf("hpc-workspace-%s", job.ID),
			},
		})
	}

	// Worker container spec
	workerContainer := map[string]interface{}{
		"name":    "worker",
		"image":   job.Image,
		"command": []string{"mpirun"},
		"args": []string{
			"--allow-run-as-root",
			"-np", fmt.Sprintf("%d", max(job.Nodes, 1)*max(job.GPUs, 1)),
			"--bind-to", "none",
			"-map-by", "slot",
			"-x", "NCCL_DEBUG=INFO",
			"-x", "LD_LIBRARY_PATH",
			"-x", "PATH",
			"-mca", "pml", "ob1",
			"-mca", "btl", "^openib",
			"/bin/bash", "-c", job.Script,
		},
		"resources": map[string]interface{}{
			"limits":   resources,
			"requests": resources,
		},
		"env":          envVars,
		"volumeMounts": volumeMounts,
	}

	// Node selector for GPU type
	nodeSelector := map[string]interface{}{}
	if job.GPUType != "" {
		nodeSelector["nvidia.com/gpu.product"] = job.GPUType
	}

	manifest := &MPIJobManifest{
		APIVersion: "kubeflow.org/v2beta1",
		Kind:       "MPIJob",
		Metadata: map[string]interface{}{
			"name":      job.K8sJobName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "vc-stack",
				"vc-stack.io/project-id":       job.ProjectID,
				"vc-stack.io/job-id":           job.ID,
				"vc-stack.io/user-id":          job.UserID,
			},
		},
		Spec: map[string]interface{}{
			"slotsPerWorker": max(job.GPUs, 1),
			"runPolicy": map[string]interface{}{
				"cleanPodPolicy":          "Running",
				"ttlSecondsAfterFinished": 3600,
				"backoffLimit":            3,
			},
			"mpiReplicaSpecs": map[string]interface{}{
				"Launcher": map[string]interface{}{
					"replicas": 1,
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{workerContainer},
							"volumes":    volumes,
						},
					},
				},
				"Worker": map[string]interface{}{
					"replicas": max(job.Nodes, 1),
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers":   []interface{}{workerContainer},
							"volumes":      volumes,
							"nodeSelector": nodeSelector,
						},
					},
				},
			},
		},
	}

	o.logger.Info("Built MPIJob manifest",
		zap.String("job_id", job.ID),
		zap.String("name", job.K8sJobName),
		zap.Int("gpus_per_worker", job.GPUs),
		zap.Int("workers", job.Nodes))

	return manifest
}

// ──────────────────────────────────────────────────────────────────
// Reconciliation Loop
// ──────────────────────────────────────────────────────────────────

// ReconcileResult holds the result of a cluster reconciliation.
type ReconcileResult struct {
	ClusterID       string            `json:"cluster_id"`
	ComponentsReady int               `json:"components_ready"`
	ComponentsTotal int               `json:"components_total"`
	Failed          []string          `json:"failed,omitempty"`
	Actions         []ReconcileAction `json:"actions"`
	OverallStatus   string            `json:"overall_status"` // ready, progressing, degraded, error
}

// ReconcileAction represents a single action taken during reconciliation.
type ReconcileAction struct {
	Component string `json:"component"`
	Action    string `json:"action"` // install, upgrade, repair, skip
	Status    string `json:"status"` // success, failed, skipped
	Message   string `json:"message"`
}

// ReconcileCluster performs a full reconciliation of HPC components on a cluster.
// In production, this talks to the K8s API; here it simulates the reconciliation.
func (o *K8sOrchestrator) ReconcileCluster(_ context.Context, components *HPCClusterComponents) *ReconcileResult {
	result := &ReconcileResult{
		ClusterID: components.ClusterID,
		Actions:   []ReconcileAction{},
	}

	// Process each component
	specs := []*HPCComponentSpec{
		components.GPUScheduler,
		components.GPUDevicePlugin,
		components.MPIOperator,
		components.NCCLPlugin,
		components.RDMADevicePlugin,
		components.SharedFS,
		components.Monitoring,
	}

	for _, spec := range specs {
		if spec == nil {
			continue
		}
		result.ComponentsTotal++

		action := ReconcileAction{
			Component: spec.Name,
		}

		switch spec.Status {
		case "pending":
			// Install the component
			action.Action = "install"
			spec.Status = "deploying"
			spec.Message = fmt.Sprintf("Installing %s v%s via Helm chart %s", spec.Name, spec.Version, spec.HelmChart)
			// Simulate successful installation
			spec.Status = "ready"
			spec.Message = ""
			action.Status = "success"
			action.Message = fmt.Sprintf("Installed %s v%s", spec.Name, spec.Version)
			result.ComponentsReady++

		case "deploying":
			// Check deployment progress
			action.Action = "skip"
			action.Status = "skipped"
			action.Message = "Deployment in progress"

		case "ready":
			// Already ready, skip
			action.Action = "skip"
			action.Status = "skipped"
			action.Message = "Already running"
			result.ComponentsReady++

		case "failed":
			// Attempt repair
			action.Action = "repair"
			spec.Status = "deploying"
			spec.Message = "Repairing failed component"
			// Simulate repair
			spec.Status = "ready"
			spec.Message = ""
			action.Status = "success"
			action.Message = fmt.Sprintf("Repaired %s", spec.Name)
			result.ComponentsReady++

		default:
			action.Action = "skip"
			action.Status = "skipped"
			action.Message = fmt.Sprintf("Unknown status: %s", spec.Status)
		}

		result.Actions = append(result.Actions, action)
	}

	// Determine overall status
	if result.ComponentsReady == result.ComponentsTotal {
		result.OverallStatus = "ready"
	} else if len(result.Failed) > 0 {
		result.OverallStatus = "degraded"
	} else {
		result.OverallStatus = "progressing"
	}

	components.LastReconciled = time.Now()

	o.logger.Info("Cluster reconciliation complete",
		zap.String("cluster_id", components.ClusterID),
		zap.String("status", result.OverallStatus),
		zap.Int("ready", result.ComponentsReady),
		zap.Int("total", result.ComponentsTotal))

	return result
}

// ──────────────────────────────────────────────────────────────────
// GPU Topology Discovery
// ──────────────────────────────────────────────────────────────────

// GPUTopology describes the GPU hardware topology of a node.
type GPUTopology struct {
	NodeName    string       `json:"node_name"`
	GPUDevices  []GPUDevice  `json:"gpu_devices"`
	NVLinkPairs []NVLinkPair `json:"nvlink_pairs,omitempty"`
	PCIeTree    string       `json:"pcie_tree,omitempty"` // textual PCIe topology
}

// GPUDevice represents a single GPU device on a node.
type GPUDevice struct {
	Index       int    `json:"index"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"` // e.g. "NVIDIA A100-SXM4-80GB"
	MemoryMB    int    `json:"memory_mb"`
	PCIeBus     string `json:"pcie_bus"`
	MIGEnabled  bool   `json:"mig_enabled"`
	MIGDevices  int    `json:"mig_devices"`
	Temperature int    `json:"temperature_c"`
	PowerDraw   int    `json:"power_draw_w"`
	Utilization int    `json:"utilization_pct"`
}

// NVLinkPair represents an NVLink connection between two GPUs.
type NVLinkPair struct {
	GPU0      int    `json:"gpu0"`
	GPU1      int    `json:"gpu1"`
	Version   int    `json:"version"` // NVLink version (3, 4, etc.)
	Bandwidth string `json:"bandwidth"`
}

// DiscoverGPUTopology discovers GPU topology for a node.
// In production, this queries nvidia-smi or DCGM on the target node.
func (o *K8sOrchestrator) DiscoverGPUTopology(nodeName string, gpuCount int, gpuType string) *GPUTopology {
	topology := &GPUTopology{
		NodeName:   nodeName,
		GPUDevices: make([]GPUDevice, 0, gpuCount),
	}

	memoryMB := 81920 // Default A100-80GB
	switch {
	case strings.Contains(gpuType, "H100"):
		memoryMB = 81920
	case strings.Contains(gpuType, "A100-40"):
		memoryMB = 40960
	case strings.Contains(gpuType, "A100"):
		memoryMB = 81920
	case strings.Contains(gpuType, "V100"):
		memoryMB = 32768
	case strings.Contains(gpuType, "L40"):
		memoryMB = 49152
	case strings.Contains(gpuType, "A10"):
		memoryMB = 24576
	}

	for i := 0; i < gpuCount; i++ {
		topology.GPUDevices = append(topology.GPUDevices, GPUDevice{
			Index:       i,
			UUID:        fmt.Sprintf("GPU-%s-%d", nodeName, i),
			Name:        fmt.Sprintf("NVIDIA %s", gpuType),
			MemoryMB:    memoryMB,
			PCIeBus:     fmt.Sprintf("0000:%02x:00.0", i+1),
			Temperature: 35,
			PowerDraw:   75,
			Utilization: 0,
		})
	}

	// Build NVLink pairs (fully connected mesh for DGX-like configs)
	if gpuCount >= 4 && (strings.Contains(gpuType, "A100") || strings.Contains(gpuType, "H100")) {
		nvlinkVersion := 3
		bandwidth := "600 GB/s"
		if strings.Contains(gpuType, "H100") {
			nvlinkVersion = 4
			bandwidth = "900 GB/s"
		}
		for i := 0; i < gpuCount; i++ {
			for j := i + 1; j < gpuCount; j++ {
				topology.NVLinkPairs = append(topology.NVLinkPairs, NVLinkPair{
					GPU0: i, GPU1: j,
					Version:   nvlinkVersion,
					Bandwidth: bandwidth,
				})
			}
		}
	}

	o.logger.Debug("Discovered GPU topology",
		zap.String("node", nodeName),
		zap.Int("gpus", gpuCount),
		zap.Int("nvlink_pairs", len(topology.NVLinkPairs)))

	return topology
}

// ──────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────

func marshallValues(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
