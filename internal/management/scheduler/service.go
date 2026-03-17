package scheduler

import (
	"context"
	"net/http"
	"sort"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

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
	leader     dlock.LeaderElector //nolint:unused
	isLeader   atomic.Bool
	mq         mq.MessageBus
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Normalize overcommit ratios: must be >= 1.0.
	if cfg.Overcommit.CPURatio < 1.0 {
		cfg.Overcommit.CPURatio = 1.0
	}
	if cfg.Overcommit.RAMRatio < 1.0 {
		cfg.Overcommit.RAMRatio = 1.0
	}
	if cfg.Overcommit.DiskRatio < 1.0 {
		cfg.Overcommit.DiskRatio = 1.0
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
	var hosts []models.Host

	if s.hosts != nil {
		var err error
		hosts, err = s.hosts.ListHosts(ctx)
		if err != nil {
			return nil, ScheduleResponse{Reason: "no hosts available"}
		}
	} else if s.db != nil {
		// Fallback: query database directly when no HostProvider configured.
		s.db.WithContext(ctx).
			Where("status = ? AND resource_state = ?", models.HostStatusUp, models.ResourceStateEnabled).
			Find(&hosts)
	}

	if len(hosts) == 0 {
		return nil, ScheduleResponse{Reason: "no hosts available"}
	}

	// Default strategy.
	strategy := req.Strategy
	if strategy == "" {
		strategy = StrategyLeastAllocated
	}

	// Filter hosts with sufficient capacity.
	var eligible []models.Host
	for _, h := range hosts {
		cpuFree := float64(h.CPUCores)*s.overcommit.CPURatio - float64(h.CPUAllocated)
		ramFree := float64(h.RAMMB)*s.overcommit.RAMRatio - float64(h.RAMAllocatedMB)
		if cpuFree >= float64(req.VCPUs) && ramFree >= float64(req.RAMMB) {
			eligible = append(eligible, h)
		}
	}
	if len(eligible) == 0 {
		return nil, ScheduleResponse{Reason: "no hosts with sufficient capacity"}
	}

	// Sort by strategy.
	sort.Slice(eligible, func(i, j int) bool {
		loadI := float64(eligible[i].CPUAllocated) / float64(max(eligible[i].CPUCores, 1))
		loadJ := float64(eligible[j].CPUAllocated) / float64(max(eligible[j].CPUCores, 1))
		if strategy == StrategyMostAllocated {
			return loadI > loadJ // pack: prefer more loaded.
		}
		return loadI < loadJ // spread (default): prefer less loaded.
	})

	return &eligible[0], ScheduleResponse{NodeID: eligible[0].UUID, Strategy: strategy}
}

func (s *Service) listNodes(c *gin.Context) {
	var hosts []models.Host

	if s.hosts != nil {
		var err error
		hosts, err = s.hosts.ListHosts(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if s.db != nil {
		s.db.WithContext(c.Request.Context()).Find(&hosts)
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
