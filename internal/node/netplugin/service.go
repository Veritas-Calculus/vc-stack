package netplugin

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Config struct {
	Logger      *zap.Logger
	OVNNBSocket string // Optional: will auto-detect if empty
}

type Service struct {
	logger      *zap.Logger
	ovnNBSocket string
	dhcpMutex   sync.Mutex // Prevent concurrent DHCP option creation
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Auto-detect OVN northbound socket location
	ovnNBSocket := cfg.OVNNBSocket
	if ovnNBSocket == "" {
		ovnNBSocket = detectOVNSocket()
		cfg.Logger.Info("Auto-detected OVN socket", zap.String("socket", ovnNBSocket))
	}

	return &Service{
		logger:      cfg.Logger,
		ovnNBSocket: ovnNBSocket,
	}, nil
}

// detectOVNSocket tries common OVN socket locations
func detectOVNSocket() string {
	locations := []string{
		"unix:/var/run/ovn/ovnnb_db.sock",         // Debian/Ubuntu
		"unix:/run/ovn/ovnnb_db.sock",             // Some modern systems
		"unix:/var/run/openvswitch/ovnnb_db.sock", // Alternative
		"tcp:127.0.0.1:6641",                      // TCP fallback
	}

	for _, loc := range locations {
		if strings.HasPrefix(loc, "unix:") {
			path := strings.TrimPrefix(loc, "unix:")
			if _, err := os.Stat(path); err == nil {
				return loc
			}
		} else if strings.HasPrefix(loc, "tcp:") {
			// For TCP, just return it as last resort
			return loc
		}
	}

	// Default fallback
	return "unix:/var/run/ovn/ovnnb_db.sock"
}

func (s *Service) SetupRoutes(r *gin.Engine) {
	r.GET("/api/netplugin/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "vc-network-plugin", "ovn_socket": s.ovnNBSocket})
	})
	v1 := r.Group("/api/v1")
	{
		// Network management
		v1.POST("/networks", s.createNetwork)
		v1.DELETE("/networks/:id", s.deleteNetwork)

		// Port management
		v1.POST("/ports", s.createPort)
		v1.DELETE("/ports/:id", s.deletePort)

		// DHCP management
		v1.POST("/dhcp", s.configureDHCP)

		// Router and gateway management (for plugin driver)
		v1.POST("/routers", s.ensureRouter)
		v1.DELETE("/routers/:name", s.deleteRouter)
		v1.POST("/routers/:name/connect-subnet", s.connectSubnetToRouter)
		v1.POST("/routers/:name/disconnect-subnet", s.disconnectSubnetFromRouter)
		v1.POST("/routers/:name/set-gateway", s.setRouterGateway)
		v1.POST("/routers/:name/clear-gateway", s.clearRouterGateway)
		v1.POST("/routers/:name/snat", s.setRouterSNAT)

		// Legacy endpoints
		v1.POST("/net/ovs/setup", s.setupOVS)
		v1.POST("/net/bridge/setup", s.setupBridge)
		v1.POST("/net/tap/create", s.createTap)
	}
}

// nbctl executes ovn-nbctl command
func (s *Service) nbctl(args ...string) (string, error) {
	if s.ovnNBSocket != "" {
		args = append([]string{"--db", s.ovnNBSocket}, args...)
	}
	s.logger.Debug("ovn-nbctl", zap.Strings("args", args))
	cmd := exec.Command("ovn-nbctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ovn-nbctl failed: %v, output: %s", err, string(out))
	}
	return string(out), nil
}

func (s *Service) setupOVS(c *gin.Context)    { c.JSON(http.StatusOK, gin.H{"ok": true}) }
func (s *Service) setupBridge(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
func (s *Service) createTap(c *gin.Context)   { c.JSON(http.StatusOK, gin.H{"tap": "tap0"}) }
