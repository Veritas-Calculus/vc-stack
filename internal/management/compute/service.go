package compute

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/internal/management/workflow"
	"github.com/Veritas-Calculus/vc-stack/pkg/vcredis"
)

// QuotaChecker is the interface for quota enforcement.
type QuotaChecker interface {
	CheckQuota(tenantID, resourceType string, delta int) error
	UpdateUsage(tenantID, resourceType string, delta int) error
}

// PortAllocator is the interface for network port allocation.
type PortAllocator interface {
	AllocateIP(ctx context.Context, instanceUUID string, networkID string) (string, error)
	ReleaseIP(ctx context.Context, instanceUUID string) error
}

// HostManager defines the interface to look up node information.
type HostManager interface {
	GetHostAddress(ctx context.Context, hostID string) (string, error)
}

// Config represents the compute service configuration.
type Config struct {
	DB           *gorm.DB
	Logger       *zap.Logger
	Scheduler    string
	JWTSecret    string
	EventLogger  event.EventLogger
	QuotaService QuotaChecker
	Redis        *vcredis.Manager
}

// Service represents the controller compute service.
type Service struct {
	db            *gorm.DB
	logger        *zap.Logger
	workflow      *workflow.Engine
	scheduler     string
	client        *http.Client
	jwtSecret     string
	internalToken string
	eventLogger   event.EventLogger
	portAllocator PortAllocator
	hostManager   HostManager
	quotaService  QuotaChecker
	agentClient   *AgentClient
}

// NewService creates a new compute service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	wfAudit := &workflowAuditor{logger: cfg.EventLogger}

	s := &Service{
		db:        cfg.DB,
		logger:    cfg.Logger,
		workflow:  workflow.NewEngine(cfg.DB, cfg.Logger, wfAudit, cfg.Redis),
		scheduler: cfg.Scheduler,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwtSecret:    cfg.JWTSecret,
		eventLogger:  cfg.EventLogger,
		quotaService: cfg.QuotaService,
	}

	return s, nil
}

// Name returns the unique identifier for this module.
func (s *Service) Name() string { return "compute" }

// ServiceInstance returns the concrete service instance.
func (s *Service) ServiceInstance() interface{} { return s }

// GetWorkflowEngine returns the underlying task engine.
func (s *Service) GetWorkflowEngine() *workflow.Engine {
	return s.workflow
}

// SetHostManager injects the host service dependency.
func (s *Service) SetHostManager(hm HostManager) {
	s.hostManager = hm
}

func (s *Service) resolveNodeAddress(ctx context.Context, hostID string) string {
	if s.hostManager == nil {
		return ""
	}
	addr, _ := s.hostManager.GetHostAddress(ctx, hostID)
	return addr
}

// RegisterWorkflows defines and registers all VM-related workflows.
func (s *Service) RegisterWorkflows() {
	// 1. Create VM Workflow
	s.workflow.RegisterWorkflow("instance.create", &workflow.Workflow{
		Name: "CreateInstance",
		Steps: []workflow.Step{
			&StepAllocateIP{NetMgr: s.portAllocator.(NetworkManager), Logger: s.logger},
			&StepCreateVolume{Storage: s.quotaService.(StorageManager), Logger: s.logger},
			&StepStartInstance{Agent: s.agentClient, Resolver: s, Logger: s.logger},
		},
	})

	// 2. Stop VM Workflow
	s.workflow.RegisterWorkflow("instance.stop", &workflow.Workflow{
		Name: "StopInstance",
		Steps: []workflow.Step{
			&StepStopInstance{Agent: s.agentClient, Resolver: s, Logger: s.logger},
		},
	})

	// 3. Delete VM Workflow
	s.workflow.RegisterWorkflow("instance.delete", &workflow.Workflow{
		Name: "DeleteInstance",
		Steps: []workflow.Step{
			&StepStopInstance{Agent: s.agentClient, Resolver: s, Logger: s.logger},
			&StepDeleteInstance{Agent: s.agentClient, Resolver: s, Logger: s.logger},
			&StepAllocateIP{NetMgr: s.portAllocator.(NetworkManager), Logger: s.logger},
		},
	})

	s.logger.Info("Compute workflows registered")
}

// SetInternalToken sets the shared secret for M2M authentication and initializes the agent client.
func (s *Service) SetInternalToken(token string) {
	s.internalToken = token
	s.agentClient = NewAgentClient(token)
}

// SetupRoutes registers HTTP routes for the compute service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1")
	{
		// Instances
		api.GET("/instances", rp("instance", "list"), s.listInstances)
		api.POST("/instances", rp("instance", "create"), s.createInstance)
		api.GET("/instances/:id", rp("instance", "get"), s.getInstance)
		api.PUT("/instances/:id", rp("instance", "update"), s.updateInstance)
		api.DELETE("/instances/:id", rp("instance", "delete"), s.deleteInstance)
		api.POST("/instances/:id/start", rp("instance", "update"), s.startInstance)
		api.POST("/instances/:id/stop", rp("instance", "update"), s.stopInstance)
		api.POST("/instances/:id/reboot", rp("instance", "update"), s.rebootInstance)

		// Flavors
		api.GET("/flavors", rp("flavor", "list"), s.listFlavors)
		api.POST("/flavors", rp("flavor", "create"), s.createFlavor)
		api.GET("/flavors/:id", rp("flavor", "get"), s.getFlavor)
		api.DELETE("/flavors/:id", rp("flavor", "delete"), s.deleteFlavor)
	}

	// Internal M2M endpoints (Compute nodes only)
	internalAuth := middleware.InternalAuthMiddleware(s.internalToken, s.logger)
	internal := router.Group("/api/v1/internal")
	internal.Use(internalAuth)
	{
		internal.PATCH("/instances/:uuid/status", s.updateInstanceStatusInternal)
	}
}

// SetPortAllocator injects the network port allocator into the compute service.
func (s *Service) SetPortAllocator(pa PortAllocator) {
	s.portAllocator = pa
}

// emitEvent is a helper to log events.
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) emitEvent(eventType, resourceID, action, status, userID string, details map[string]interface{}, errMsg string) {
	if s.eventLogger != nil {
		go s.eventLogger.LogEvent(eventType, "instance", resourceID, action, status, userID, "", details, errMsg)
	}
}

// workflowAuditor bridges workflow events to the global event logger.
type workflowAuditor struct {
	logger event.EventLogger
}

func (a *workflowAuditor) LogEvent(resource, resourceID, action, status, userID, message string) {
	if a.logger != nil {
		a.logger.LogEvent("workflow", resource, resourceID, action, status, userID, "", nil, message)
	}
}
