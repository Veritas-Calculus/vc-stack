package usage

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	events EventManager // Using interface
}

// EventManager defines the pub/sub interface needed for billing.
type EventManager interface {
	Subscribe(topic string, handler func(payload string))
}

func NewService(cfg Config) (*Service, error) {
	return &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}, nil
}

func (s *Service) Name() string { return "usage" }
func (s *Service) ServiceInstance() interface{} { return s }

// SetEventManager injects the event bus and starts listeners.
func (s *Service) SetEventManager(m interface{}) {
	if em, ok := m.(EventManager); ok {
		s.events = em
		s.startListeners()
	}
}

func (s *Service) startListeners() {
	s.logger.Info("Usage: Starting billing listeners for VM events")
	s.events.Subscribe("compute.instance.created", s.handleVMCreateEvent)
}

func (s *Service) handleVMCreateEvent(payload string) {
	s.logger.Info("Usage: VM creation detected, starting billable period", zap.String("payload", payload))
	// Logic to insert record into usage_records table
}

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/usage")
	{
		api.GET("/records", s.listUsage)
	}
}

func (s *Service) listUsage(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"records": []string{}})
}
