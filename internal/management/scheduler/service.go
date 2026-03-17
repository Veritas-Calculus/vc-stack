package scheduler

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/circuitbreaker"
	"github.com/Veritas-Calculus/vc-stack/pkg/dlock"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"github.com/Veritas-Calculus/vc-stack/pkg/mq"
)

// Scheduling strategies.
const (
	StrategyLeastAllocated = "spread"
	StrategyMostAllocated  = "pack"
)

// HostProvider defines the interface for resource discovery.
type HostProvider interface {
	ListHosts(ctx context.Context) ([]models.Host, error)
	GetHost(ctx context.Context, id string) (*models.Host, error)
	DeleteHost(ctx context.Context, id string) error
}

type Config struct {
	DB         *gorm.DB
	Logger     *zap.Logger
	Overcommit OvercommitConfig
	Hosts      HostProvider
	DLock      *dlock.Manager
	MQ         mq.MessageBus
}

type OvercommitConfig struct {
	CPURatio  float64
	RAMRatio  float64
	DiskRatio float64
}

// Service provides scheduling.
type Service struct {
	db         *gorm.DB
	logger     *zap.Logger
	hosts      HostProvider
	overcommit OvercommitConfig
	cbManager  *circuitbreaker.Manager
	leader     dlock.LeaderElector
	isLeader   atomic.Bool
	mq         mq.MessageBus
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{
		db:         cfg.DB,
		logger:     cfg.Logger,
		hosts:      cfg.Hosts,
		overcommit: cfg.Overcommit,
		mq:         cfg.MQ,
	}
	svc.isLeader.Store(true)
	return svc, nil
}

func (s *Service) Name() string                 { return "scheduler" }
func (s *Service) ServiceInstance() interface{} { return s }

func (s *Service) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.POST("/schedule", s.schedule)
		api.GET("/nodes", s.listNodes)
	}
}

func (s *Service) schedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	host, resp := s.selectHost(c.Request.Context(), req)
	if host == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": resp.Reason})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Service) selectHost(ctx context.Context, req ScheduleRequest) (*models.Host, ScheduleResponse) {
	hosts, err := s.hosts.ListHosts(ctx)
	if err != nil || len(hosts) == 0 {
		return nil, ScheduleResponse{Reason: "no hosts available"}
	}

	// Default: return first host (Spread)
	return &hosts[0], ScheduleResponse{NodeID: hosts[0].UUID, Strategy: req.Strategy}
}

func (s *Service) listNodes(c *gin.Context) {
	hosts, err := s.hosts.ListHosts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": hosts})
}

type ScheduleRequest struct {
	VCPUs    int    `json:"vcpus"`
	RAMMB    int    `json:"ram_mb"`
	DiskGB   int    `json:"disk_gb"`
	Strategy string `json:"strategy"`
}

type ScheduleResponse struct {
	NodeID   string `json:"node"`
	Reason   string `json:"reason"`
	Strategy string `json:"strategy"`
}
