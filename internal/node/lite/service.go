// Package lite provides the node agent that reports node state,
// accepts scheduling assignments, and manages VMs via libvirt.
package lite

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	Logger             *zap.Logger
	DB                 *gorm.DB // Database connection for network/port lookup
	LibvirtURI         string
	SmbiosManufacturer string
	SmbiosProduct      string
	NetworkName        string // Deprecated: fallback network name
	OVMFCodePath       string
	OVMFVarsPath       string
	NvramDir           string
	TPMModel           string
	TPMBackend         string
	TPMVersion         string
	// Ceph (RBD) integration.
	CephEnabled     bool
	CephMonitors    []string
	CephUser        string // Images pool user
	CephSecretUUID  string
	CephKeyringPath string // Images pool keyring
	CephConf        string // Path to ceph.conf
	CephConfigDir   string // Deprecated: directory containing ceph.conf
	// RBD lifecycle.
	CephDefaultPool     string // Pool for source images (vcstack-images)
	CephVolumesPool     string // Pool for VM volumes (vcstack-volumes)
	CephVolumesUser     string // Volumes pool user (default: vcstack)
	CephVolumesKeyring  string // Volumes pool keyring
	CephVMImagePrefix   string
	CephDeleteOnDestroy bool
	CephCloneFlatten    bool
	RbdCommand          string
	RbdTimeoutSeconds   int
	// QEMU/KVM driver configuration.
	UseQEMU     bool   // Use QEMU driver instead of libvirt
	QEMURunDir  string // Runtime directory for QEMU
	QEMUCfgDir  string // Configuration directory for QEMU
	QEMUTmplDir string // Template directory for QEMU
}

type Service struct {
	logger *zap.Logger
	drv    Driver
	mu     sync.RWMutex
	vms    map[string]*VM
	met    *LiteMetrics
	// console tokens: token -> VNC address and expiry.
	tokens map[string]consoleToken
}

type consoleToken struct {
	VNCAddr   string
	ExpiresAt time.Time
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	drv, err := newDriver(cfg)
	if err != nil {
		return nil, err
	}
	s := &Service{logger: cfg.Logger, drv: drv, vms: make(map[string]*VM), met: NewLiteMetrics(), tokens: make(map[string]consoleToken)}
	SetMetrics(s.met)
	return s, nil
}

// SetupRoutes registers HTTP endpoints for vc-lite.
func (s *Service) SetupRoutes(r *gin.Engine) {
	r.GET("/api/lite/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "vc-lite"}) })
	r.GET("/metrics", func(c *gin.Context) { c.String(http.StatusOK, s.met.RenderProm()) })
	// WebSocket endpoint for noVNC/websockify.
	r.GET("/ws/console", s.consoleWS)

	api := r.Group("/api/v1")
	{
		// Node heartbeat and resource report.
		api.POST("/nodes/heartbeat", s.heartbeat)
		api.GET("/nodes/status", s.nodeStatus)

		// VM control plane (proxy to libvirt)
		vms := api.Group("/vms")
		{
			vms.POST("", s.createVM)
			vms.GET("/:id", s.getVM)
			vms.DELETE("/:id", s.deleteVM)
			vms.POST("/:id/start", s.startVM)
			vms.POST("/:id/stop", s.stopVM)
			vms.POST("/:id/reboot", s.rebootVM)
			vms.POST("/:id/force-stop", s.forceStopVM)
			vms.POST("/:id/force-reboot", s.forceRebootVM)
		}

		// Console ticket endpoint (returns ws path and token)
		api.POST("/vms/:id/console", s.consoleTicket)
	}
}

func (s *Service) heartbeat(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"ok": true}) }
func (s *Service) nodeStatus(c *gin.Context) { c.JSON(http.StatusOK, s.drv.Status()) }

// GetStatus exposes the node status for internal callers (e.g., registration/heartbeat).
func (s *Service) GetStatus() NodeStatus { return s.drv.Status() }
func (s *Service) createVM(c *gin.Context) {
	var req CreateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("createVM invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	s.logger.Info("createVM request received",
		zap.String("name", req.Name),
		zap.Int("vcpus", req.VCPUs),
		zap.Int("memory_mb", req.MemoryMB),
		zap.Int("disk_gb", req.DiskGB),
		zap.String("image", req.Image),
		zap.String("root_rbd_image", req.RootRBDImage),
		zap.String("client_ip", c.ClientIP()))

	// Manual validation: at least one image source must be provided.
	if strings.TrimSpace(req.Image) == "" && strings.TrimSpace(req.RootRBDImage) == "" && strings.TrimSpace(req.ISO) == "" && strings.TrimSpace(req.IsoRBDImage) == "" {
		s.logger.Warn("createVM missing image source", zap.Any("request", req))
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing image source (image or root_rbd_image or iso)"})
		return
	}

	start := time.Now()
	s.logger.Info("createVM calling driver", zap.String("name", req.Name))
	vm, err := s.drv.CreateVM(req)
	if err != nil {
		s.logger.Error("createVM driver failed", zap.String("name", req.Name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("createVM driver succeeded",
		zap.String("vm_id", vm.ID),
		zap.String("name", vm.Name),
		zap.String("status", vm.Status),
		zap.String("power", vm.Power),
		zap.Duration("duration", time.Since(start)))

	s.met.Inc(MVMCreateTotal, 1)
	s.met.ObserveMs(MVMCreateTotal, float64(time.Since(start).Milliseconds()))
	s.mu.Lock()
	s.vms[vm.ID] = vm
	s.mu.Unlock()

	s.logger.Info("createVM completed", zap.String("vm_id", vm.ID))
	c.JSON(http.StatusAccepted, gin.H{"vm": vm})
}
func (s *Service) deleteVM(c *gin.Context) {
	id := c.Param("id")
	start := time.Now()

	// Attempt deletion.
	if err := s.drv.DeleteVM(id, false); err != nil {
		s.logger.Error("DeleteVM failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Verify deletion - check if VM still exists.
	if exists, _ := s.drv.VMStatus(id); exists {
		s.logger.Error("VM still exists after deletion", zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "VM deletion verification failed - VM still exists",
		})
		return
	}

	// Update metrics.
	s.met.Inc(MVMDeleteTotal, 1)
	s.met.ObserveMs(MVMDeleteMs, float64(time.Since(start).Milliseconds()))

	// Remove from memory map.
	s.mu.Lock()
	delete(s.vms, id)
	s.mu.Unlock()

	s.logger.Info("VM deleted successfully", zap.String("id", id), zap.Duration("duration", time.Since(start)))
	c.JSON(http.StatusOK, gin.H{
		"deleted":     id,
		"verified":    true,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}
func (s *Service) startVM(c *gin.Context)       { s.powerOp(c, "start") }
func (s *Service) stopVM(c *gin.Context)        { s.powerOp(c, "stop") }
func (s *Service) rebootVM(c *gin.Context)      { s.powerOp(c, "reboot") }
func (s *Service) forceStopVM(c *gin.Context)   { s.powerOp(c, "force-stop") }
func (s *Service) forceRebootVM(c *gin.Context) { s.powerOp(c, "force-reboot") }

// getVM returns VM metadata by ID if present in memory and optionally checks libvirt state.
func (s *Service) getVM(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	vm, ok := s.vms[id]
	s.mu.RUnlock()
	if !ok {
		s.logger.Warn("getVM: not found in memory", zap.String("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "vm not found"})
		return
	}
	// Optional: verify VM still exists in libvirt (best-effort)
	// This helps catch cases where VM was defined but failed to start.
	if exists, running := s.drv.VMStatus(id); !exists {
		s.logger.Warn("getVM: vm in memory but not in libvirt", zap.String("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "vm not found in hypervisor"})
		return
	} else {
		// Update power state from live query.
		if running {
			vm.Power = "running"
			vm.Status = "active"
		} else {
			vm.Power = "shutdown"
		}
	}
	c.JSON(http.StatusOK, gin.H{"vm": vm})
}
func (s *Service) powerOp(c *gin.Context, op string) {
	id := c.Param("id")
	var err error
	switch op {
	case "start":
		err = s.drv.StartVM(id)
	case "stop":
		err = s.drv.StopVM(id, false)
	case "reboot":
		err = s.drv.RebootVM(id, false)
	case "force-stop":
		err = s.drv.StopVM(id, true)
	case "force-reboot":
		err = s.drv.RebootVM(id, true)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "op": op, "id": id})
}
func (s *Service) consoleTicket(c *gin.Context) {
	id := c.Param("id")
	// Ask driver for VNC address (e.g., vnc://127.0.0.1:5901)
	vncURL, err := s.drv.ConsoleURL(id, 5*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Generate ephemeral token.
	token := genToken(16)
	s.mu.Lock()
	s.tokens[token] = consoleToken{VNCAddr: vncURL, ExpiresAt: time.Now().Add(5 * time.Minute)}
	s.mu.Unlock()
	// Return relative WS path so clients can connect via gateway.
	c.JSON(http.StatusOK, gin.H{"ws": "/ws/console?token=" + token, "token_expires_in": 300})
}

// consoleWS upgrades to websocket and bridges traffic to the libvirt VNC TCP socket.
func (s *Service) consoleWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}
	s.mu.Lock()
	tk, ok := s.tokens[token]
	if ok && time.Now().After(tk.ExpiresAt) {
		ok = false
		delete(s.tokens, token)
	}
	// single-use token.
	if ok {
		delete(s.tokens, token)
	}
	s.mu.Unlock()
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Parse VNC address and dial TCP.
	addr := tk.VNCAddr
	addr = strings.TrimPrefix(addr, "vnc://")
	// Accept only host:port.
	if _, _, err := net.SplitHostPort(addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid VNC address"})
		return
	}
	backend, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "connect VNC backend failed"})
		return
	}
	defer backend.Close()

	up := websocket.Upgrader{
		CheckOrigin:  func(r *http.Request) bool { return true },
		Subprotocols: []string{"binary"},
	}
	ws, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// Pump data between WS and TCP.
	errc := make(chan error, 2)
	go func() {
		// WS -> TCP.
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			// Forward only binary/text payloads.
			if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
				if _, err := backend.Write(data); err != nil {
					errc <- err
					return
				}
			}
		}
	}()
	go func() {
		// TCP -> WS.
		buf := make([]byte, 32*1024)
		for {
			n, err := backend.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					errc <- werr
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					errc <- err
				} else {
					errc <- nil
				}
				return
			}
		}
	}()
	<-errc
}

func genToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// fallback to time-based.
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}
