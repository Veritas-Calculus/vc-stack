package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TrafficType defines the purpose of a physical network interface.
type TrafficType string

const (
	TrafficManagement TrafficType = "management"
	TrafficGuest      TrafficType = "guest"
	TrafficPublic     TrafficType = "public"
	TrafficStorage    TrafficType = "storage"
)

// Service represents the network service.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	config Config
	driver Driver
	ipam   *IPAM
}

// NewService creates a new network service instance.
func NewService(config Config) (*Service, error) {
	service := &Service{
		db:     config.DB,
		logger: config.Logger,
		config: config,
	}

	ovnCfg := config.SDN.OVN
	if env := os.Getenv("OVN_NB_ADDRESS"); strings.TrimSpace(env) != "" {
		ovnCfg.NBAddress = env
	}
	ovnCfg.BridgeMappings = config.SDN.BridgeMappings
	
	service.driver = NewOVNDriver(config.Logger, ovnCfg)
	service.ipam = NewIPAM(config.DB, config.Logger)

	return service, nil
}

// --- IoC Module Implementation ---

func (s *Service) Name() string { return "network" }
func (s *Service) ServiceInstance() interface{} { return Interface(s) }

type Interface interface {
	AllocateIP(ctx context.Context, instanceUUID string, networkID string) (string, error)
	ReleaseIP(ctx context.Context, instanceUUID string) error
}

func (s *Service) AllocateIP(ctx context.Context, instanceUUID string, networkID string) (string, error) {
	var subnet Subnet
	if err := s.db.Where("network_id = ?", networkID).First(&subnet).Error; err != nil {
		return "", err
	}
	return s.ipam.Allocate(ctx, &subnet, instanceUUID)
}

func (s *Service) ReleaseIP(ctx context.Context, instanceUUID string) error {
	var port NetworkPort
	if err := s.db.Where("device_id = ?", instanceUUID).First(&port).Error; err == nil {
		for _, fip := range port.FixedIPs {
			_ = s.ipam.Release(ctx, port.SubnetID, fip.IP)
		}
	}
	return nil
}

// --- Routing ---

func (s *Service) SetupRoutes(router *gin.Engine) {
	s.setupRoutes(router)
	s.setupRouterRoutes(router)
	s.setupSecurityRoutes(router)
}

// --- Internal Helpers (Shared across files in package) ---

func (s *Service) getOVNDriver() *OVNDriver {
	if drv, ok := s.driver.(*OVNDriver); ok {
		return drv
	}
	return nil
}

func GenerateMAC() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	buf[0] = (buf[0] | 2) & 0xfe
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}

func ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	return err
}

func (s *Service) migrateDatabase() error {
	return s.db.AutoMigrate(
		&Network{}, &Subnet{}, &NetworkPort{}, &Router{}, &RouterInterface{},
		&FloatingIP{}, &SecurityGroup{}, &SecurityGroupRule{}, &PhysicalNetwork{},
		&IPAllocation{},
	)
}

// --- Config Types ---

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
	SDN    SDNConfig
	IPAM   IPAMOptions
}

type SDNConfig struct {
	Provider       string
	Bridge         string
	OVN            OVNConfig
	PluginEndpoint string 
	BridgeMappings string
}

type IPAMOptions struct {
	ReserveGateway bool
	ReservedFirst  int
	ReservedLast   int
}

// OVNConfig holds OVN northbound database connection settings.
type OVNConfig struct {
	NBAddress      string // e.g. tcp:127.0.0.1:6641
	BridgeMappings string // OVS bridge mappings
}
