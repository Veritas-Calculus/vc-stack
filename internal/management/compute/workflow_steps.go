package compute

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Veritas-Calculus/vc-stack/internal/management/workflow"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"go.uber.org/zap"
)

// NetworkManager defines the subset of network module needed for workflows.
type NetworkManager interface {
	AllocateIP(ctx context.Context, instanceUUID string, networkID string) (string, error)
	ReleaseIP(ctx context.Context, instanceUUID string) error
}

// ComputeAgentClient defines the communication with the remote compute agent.
type ComputeAgentClient interface {
	StartVM(ctx context.Context, hostAddr string, inst *models.Instance) error
	StopVM(ctx context.Context, hostAddr string, instanceUUID string) error
	ConfigureNetwork(ctx context.Context, hostAddr string, bridgeMappings string) error
}

// StorageManager defines the subset of storage module needed for workflows.
type StorageManager interface {
	CreateVolume(ctx context.Context, vol *models.Volume) error
	DeleteVolume(ctx context.Context, volumeID uint) error
}

// HostResolver defines the interface to look up node addresses.
type HostResolver interface {
	resolveNodeAddress(ctx context.Context, hostID string) string
}

// ── Step: AllocateIP ───────────────────────────────────────────────────

type StepAllocateIP struct {
	NetMgr NetworkManager
	Logger *zap.Logger
}

func (s *StepAllocateIP) Name() string { return "AllocateIP" }

func (s *StepAllocateIP) Execute(ctx context.Context, t *workflow.Task) error {
	var inst models.Instance
	if err := json.Unmarshal([]byte(t.Payload), &inst); err != nil {
		return fmt.Errorf("invalid task payload: %w", err)
	}

	if inst.IPAddress != "" {
		s.Logger.Info("IP already assigned, skipping", zap.String("ip", inst.IPAddress))
		return nil
	}

	networkID := "default"
	if len(inst.Networks) > 0 {
		networkID = inst.Networks[0].UUID
	}

	ip, err := s.NetMgr.AllocateIP(ctx, inst.UUID, networkID)
	if err != nil {
		return fmt.Errorf("failed to allocate IP: %w", err)
	}

	inst.IPAddress = ip
	newPayload, _ := json.Marshal(inst)
	t.Payload = string(newPayload)
	return nil
}

func (s *StepAllocateIP) Compensate(ctx context.Context, t *workflow.Task) error {
	s.Logger.Warn("Workflow: Releasing IP", zap.String("instance", t.ResourceUUID))
	return s.NetMgr.ReleaseIP(ctx, t.ResourceUUID)
}

// ── Step: CreateVolume ──────────────────────────────────────────────────

type StepCreateVolume struct {
	Storage StorageManager
	Logger  *zap.Logger
}

func (s *StepCreateVolume) Name() string { return "CreateVolume" }

func (s *StepCreateVolume) Execute(ctx context.Context, t *workflow.Task) error {
	var inst models.Instance
	if err := json.Unmarshal([]byte(t.Payload), &inst); err != nil {
		return err
	}

	vol := &models.Volume{
		Name:      inst.Name + "-root",
		SizeGB:    inst.RootDiskGB,
		RBDPool:   "vcstack-volumes",
		RBDImage:  fmt.Sprintf("vol-%s", inst.UUID),
		UserID:    inst.UserID,
		ProjectID: inst.ProjectID,
		Status:    "creating",
		Bootable:  true,
	}

	if err := s.Storage.CreateVolume(ctx, vol); err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	inst.RootRBDImage = vol.RBDPool + "/" + vol.RBDImage
	newPayload, _ := json.Marshal(inst)
	t.Payload = string(newPayload)
	return nil
}

func (s *StepCreateVolume) Compensate(ctx context.Context, t *workflow.Task) error {
	return nil // Future: call Storage.DeleteVolume
}

// ── Step: StartInstance ───────────────────────────────────────────────

type StepStartInstance struct {
	Agent    ComputeAgentClient
	Resolver HostResolver
	Logger   *zap.Logger
}

func (s *StepStartInstance) Name() string { return "StartInstance" }

func (s *StepStartInstance) Execute(ctx context.Context, t *workflow.Task) error {
	var inst models.Instance
	if err := json.Unmarshal([]byte(t.Payload), &inst); err != nil {
		return err
	}

	hostAddr := s.Resolver.resolveNodeAddress(ctx, inst.HostID)
	if hostAddr == "" {
		return fmt.Errorf("could not resolve address for host: %s", inst.HostID)
	}

	return s.Agent.StartVM(ctx, hostAddr, &inst)
}

func (s *StepStartInstance) Compensate(ctx context.Context, t *workflow.Task) error {
	var inst models.Instance
	_ = json.Unmarshal([]byte(t.Payload), &inst)
	hostAddr := s.Resolver.resolveNodeAddress(ctx, inst.HostID)
	if hostAddr == "" {
		return nil
	}
	return s.Agent.StopVM(ctx, hostAddr, t.ResourceUUID)
}

// ── Step: StopInstance ────────────────────────────────────────────────

type StepStopInstance struct {
	Agent    ComputeAgentClient
	Resolver HostResolver
	Logger   *zap.Logger
}

func (s *StepStopInstance) Name() string { return "StopInstance" }
func (s *StepStopInstance) Execute(ctx context.Context, t *workflow.Task) error {
	var inst models.Instance
	if err := json.Unmarshal([]byte(t.Payload), &inst); err != nil {
		return err
	}
	hostAddr := s.Resolver.resolveNodeAddress(ctx, inst.HostID)
	if hostAddr == "" {
		return fmt.Errorf("host unreachable")
	}
	return s.Agent.StopVM(ctx, hostAddr, t.ResourceUUID)
}
func (s *StepStopInstance) Compensate(ctx context.Context, t *workflow.Task) error { return nil }

// ── Step: DeleteInstance ──────────────────────────────────────────────

type StepDeleteInstance struct {
	Agent    ComputeAgentClient
	Resolver HostResolver
	Logger   *zap.Logger
}

func (s *StepDeleteInstance) Name() string { return "DeleteInstance" }
func (s *StepDeleteInstance) Execute(ctx context.Context, t *workflow.Task) error {
	var inst models.Instance
	if err := json.Unmarshal([]byte(t.Payload), &inst); err != nil {
		return err
	}
	hostAddr := s.Resolver.resolveNodeAddress(ctx, inst.HostID)
	if hostAddr == "" {
		return fmt.Errorf("host unreachable")
	}
	return s.Agent.StopVM(ctx, hostAddr, t.ResourceUUID)
}
func (s *StepDeleteInstance) Compensate(ctx context.Context, t *workflow.Task) error { return nil }

// ── Step: ConfigureNodeNetwork ────────────────────────────────────────

type StepConfigureNodeNetwork struct {
	Agent    ComputeAgentClient
	Resolver HostResolver
	Logger   *zap.Logger
}

func (s *StepConfigureNodeNetwork) Name() string { return "ConfigureNodeNetwork" }

func (s *StepConfigureNodeNetwork) Execute(ctx context.Context, t *workflow.Task) error {
	hostAddr := s.Resolver.resolveNodeAddress(ctx, t.ResourceUUID)
	if hostAddr == "" {
		return fmt.Errorf("could not resolve address for host: %s", t.ResourceUUID)
	}

	mappings := "physnet1:br-ex" // Default logic
	s.Logger.Info("Workflow: Configuring node OVS", zap.String("mappings", mappings))
	return s.Agent.ConfigureNetwork(ctx, hostAddr, mappings)
}

func (s *StepConfigureNodeNetwork) Compensate(ctx context.Context, t *workflow.Task) error {
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────

//nolint:unused // TODO: wire into routes when feature is enabled
func updatePayloadField(payload, key, value string) string {
	var data map[string]interface{}
	_ = json.Unmarshal([]byte(payload), &data)
	data[key] = value
	newData, _ := json.Marshal(data)
	return string(newData)
}
