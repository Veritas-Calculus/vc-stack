// Package compute provides the compute agent service.
// It manages the local virtual machine lifecycle and reports status back to vc-management.
package compute

import (

	"go.uber.org/zap"

	fc "github.com/Veritas-Calculus/vc-stack/internal/compute/firecracker"
	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Service represents the compute agent.
type Service struct {
	logger     *zap.Logger
	config     Config
	controller *ControllerClient
	
	// vmDriver provides direct access to the QEMU/KVM driver.
	vmDriver *vm.Service
	
	// rbdManager manages local Ceph/RBD attachments.
	rbdManager *RBDManager
	
	// fcRegistry tracks running Firecracker microVM processes.
	fcRegistry *fc.Registry
	// fcNetMgr manages TAP devices for Firecracker.
	fcNetMgr *fc.NetworkManager
}

// Config represents the compute agent configuration.
type Config struct {
	Logger        *zap.Logger
	ControllerURL string
	InternalToken string
	VMDriver      *vm.Service
	Hypervisor    HypervisorConfig
	Orchestrator  OrchestratorConfig
	Images        ImagesConfig
	Volumes       VolumesConfig
	Backups       BackupsConfig
	Firecracker   FirecrackerConfig // Added
}

type FirecrackerConfig struct {
	BinaryPath   string
	SocketDir    string
	KernelPath   string
	RootFSPath   string
	DefaultVCPUs int
	DefaultRAM   int
}

type HypervisorConfig struct {
	Type string // kvm, firecracker
}

type OrchestratorConfig struct {
	SchedulerURL string
	LiteURL      string
}

type ImagesConfig struct {
	DefaultBackend string
	LocalPath      string
	RBDPool        string
	RBDClient      string
	CephConf       string
	Keyring        string
}

type VolumesConfig struct {
	DefaultBackend string
	LocalPath      string
	RBDPool        string
	RBDClient      string
	CephConf       string
	Keyring        string
}

type BackupsConfig struct {
	RBDPool   string
	RBDClient string
	CephConf  string
	Keyring   string
}

// Core compute models are re-exported for local logic.
type Instance = models.Instance
type Volume = models.Volume
type Flavor = models.Flavor
type Image = models.Image

// NetworkRequest represents network configuration for instance creation.
type NetworkRequest struct {
	UUID    string `json:"uuid"`
	Port    string `json:"port"`
	FixedIP string `json:"fixed_ip"`
}

// NewService creates a new compute agent service.
func NewService(config Config) (*Service, error) {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	service := &Service{
		logger:     config.Logger,
		config:     config,
		vmDriver:   config.VMDriver,
		controller: NewControllerClient(config.ControllerURL, config.InternalToken, config.Logger),
	}

	// Initialize RBD manager for local storage ops.
	service.rbdManager = NewRBDManager(
		config.Logger,
		config.Images,
		config.Volumes,
		BackupsConfig{}, // Backups usually managed by dedicated service now
	)

	return service, nil
}

// SetupRoutes registers the minimal API for the controller to call.
func (s *Service) SetupRoutes(router interface{}) {
	// Logic to register HTTP routes (Start, Stop, Metrics) will be here.
}
