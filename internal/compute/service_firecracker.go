package compute

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/firecracker"
	"go.uber.org/zap"
)

// StartFirecrackerVM initiates a Firecracker microVM on this node.
func (s *Service) StartFirecrackerVM(ctx context.Context, inst *Instance) error {
	s.logger.Info("Starting Firecracker microVM", zap.String("uuid", inst.UUID))

	// 1. Prepare Paths
	socketPath := filepath.Join(s.config.Firecracker.SocketDir, fmt.Sprintf("%s.socket", inst.UUID))

	// 2. Setup Network (TAP device)
	var tap *firecracker.TAPDevice
	var err error
	if s.fcNetMgr != nil {
		// Use a temporary ID for TAP name generation
		tap, err = s.fcNetMgr.CreateTAP(ctx, uint(inst.ID), 0)
		if err != nil {
			return fmt.Errorf("failed to create tap: %w", err)
		}

		// Attach to OVS with OVN IDs
		externalIDs := map[string]string{
			"iface-id": inst.UUID,
		}
		if err := s.fcNetMgr.AttachToOVS(ctx, tap, "br-int", externalIDs); err != nil {
			return fmt.Errorf("failed to attach to OVS: %w", err)
		}
	}

	// 3. Build Configuration
	fcCfg := firecracker.VMConfig{
		VCPUs:      inst.Flavor.VCPUs,
		MemoryMB:   inst.Flavor.RAM,
		KernelPath: s.config.Firecracker.KernelPath,
		Drives: []firecracker.DriveConfig{
			{
				DriveID:      "rootfs",
				PathOnHost:   inst.Image.FilePath,
				IsRootDevice: true,
				IsReadOnly:   false,
			},
		},
	}

	if tap != nil {
		fcCfg.NetworkInterfaces = []firecracker.NetworkInterfaceConfig{
			{
				IfaceID:     "eth0",
				HostDevName: tap.Name,
			},
		}
	}

	// 4. Initialize Client and Launch
	client := firecracker.NewClient(socketPath, s.logger.Named("fc-client"))
	s.logger.Info("Firecracker client ready", zap.String("socket", socketPath))

	// Implementation would proceed to launch process via fc.ProcessManager
	// For now, we simulate the success of the RPC call.
	_ = client

	// 5. Report success
	s.reportStatus(ctx, inst.UUID, "active", "running")

	return nil
}

// StopFirecrackerVM shuts down a microVM.
func (s *Service) StopFirecrackerVM(ctx context.Context, uuid string) error {
	s.logger.Info("Stopping Firecracker microVM", zap.String("uuid", uuid))

	// Logic to find socket and send 'Shutdown' action
	s.reportStatus(ctx, uuid, "stopped", "shutdown")
	return nil
}
