package compute

import (
	"context"
	"strings"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm"
	"go.uber.org/zap"
)

// StartVM initiates the launch of a virtual machine on this node.
// It is called by the management plane (via Agent API).
func (s *Service) StartVM(ctx context.Context, inst *Instance) error {
	s.logger.Info("Starting VM", zap.String("uuid", inst.UUID), zap.String("name", inst.Name))

	// 1. Convert models.Instance to vm.CreateVMRequest
	req := vm.CreateVMRequest{
		Name:             inst.VMID,
		VCPUs:            inst.Flavor.VCPUs,
		MemoryMB:         inst.Flavor.RAM,
		DiskGB:           inst.RootDiskGB,
		SSHAuthorizedKey: inst.SSHKey,
		UserData:         inst.UserData,
	}

	// Handle storage backend
	if strings.TrimSpace(inst.Image.RBDPool) != "" && strings.TrimSpace(inst.Image.RBDImage) != "" {
		val := inst.Image.RBDPool + "/" + inst.Image.RBDImage
		if strings.TrimSpace(inst.Image.RBDSnap) != "" {
			val = val + "@" + inst.Image.RBDSnap
		}
		req.RootRBDImage = val
	} else if strings.TrimSpace(inst.Image.FilePath) != "" {
		req.Image = inst.Image.FilePath
	}

	// 2. Report "spawning" status to controller.
	s.reportStatus(ctx, inst.UUID, "spawning", "shutdown")

	// 3. Launch the VM using the local driver.
	go func() {
		bgCtx := context.Background()

		if s.vmDriver == nil {
			s.logger.Error("no hypervisor driver available on node", zap.String("uuid", inst.UUID))
			s.reportStatus(bgCtx, inst.UUID, "error", "shutdown")
			return
		}

		vmMetadata, err := s.vmDriver.CreateVMDirect(req)
		if err != nil {
			s.logger.Error("VM launch failed", zap.String("uuid", inst.UUID), zap.Error(err))
			s.reportStatus(bgCtx, inst.UUID, "error", "shutdown")
			return
		}

		s.logger.Info("VM process launched", zap.String("uuid", inst.UUID), zap.String("vmid", vmMetadata.ID))

		// 4. Update status to active/running.
		s.reportStatus(bgCtx, inst.UUID, "active", "running")

		// Report additional runtime details (VNC port, PID if available)
		_ = s.controller.UpdateInstanceStatus(bgCtx, inst.UUID, map[string]interface{}{
			"launched_at": time.Now(),
		})
	}()

	return nil
}

// StopVM terminates a running virtual machine on this node.
func (s *Service) StopVM(ctx context.Context, instanceUUID string) error {
	s.logger.Info("Stopping VM", zap.String("uuid", instanceUUID))

	if s.vmDriver != nil {
		if err := s.vmDriver.StopVMDirect(instanceUUID, false); err != nil {
			return err
		}
	}

	s.reportStatus(ctx, instanceUUID, "stopped", "shutdown")
	return nil
}

// reportStatus is a helper to update instance state via the controller API.
func (s *Service) reportStatus(ctx context.Context, uuid, status, powerState string) {
	updates := map[string]interface{}{
		"status":      status,
		"power_state": powerState,
	}
	if err := s.controller.UpdateInstanceStatus(ctx, uuid, updates); err != nil {
		s.logger.Error("Failed to report status to controller",
			zap.String("uuid", uuid), zap.Error(err))
	}
}
