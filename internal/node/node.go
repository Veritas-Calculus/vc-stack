package node

import (
	"os"

	"github.com/Veritas-Calculus/vc-stack/internal/node/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/node/lite"
	"github.com/Veritas-Calculus/vc-stack/internal/node/netplugin"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config aggregates dependencies required by node components.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service composes compute, lite and netplugin services and exposes
// SetupRoutes to register their routes on a router.
type Service struct {
	Compute   *compute.Service
	Lite      *lite.Service
	NetPlugin *netplugin.Service
}

// New composes node services.
func New(cfg Config) (*Service, error) {
	// Since lite runs on the same node, default to localhost:8081 (or NODE_PORT if set)
	liteURL := "http://localhost:8081"
	if port := os.Getenv("NODE_PORT"); port != "" {
		liteURL = "http://localhost:" + port
	}

	compSvc, err := compute.NewService(compute.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger,
		Hypervisor: compute.HypervisorConfig{
			Type:       "kvm",
			LibvirtURI: "",
		},
		Orchestrator: compute.OrchestratorConfig{
			LiteURL: liteURL,
		},
	})
	if err != nil {
		return nil, err
	}

	liteSvc, err := lite.NewService(lite.Config{Logger: cfg.Logger, DB: cfg.DB})
	if err != nil {
		return nil, err
	}

	npSvc, err := netplugin.NewService(netplugin.Config{Logger: cfg.Logger})
	if err != nil {
		return nil, err
	}

	return &Service{Compute: compSvc, Lite: liteSvc, NetPlugin: npSvc}, nil
}

// SetupRoutes registers node service routes on the provided router
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Add a simple health check endpoint for gateway health checking
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	s.Compute.SetupRoutes(router)
	s.Lite.SetupRoutes(router)
	s.NetPlugin.SetupRoutes(router)
}
