package selfheal

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
	db      *gorm.DB
	logger  *zap.Logger
	compute interface{} // compute.Interface
	notify  interface{} // notification.Interface
}

func NewService(cfg Config) (*Service, error) {
	return &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}, nil
}

func (s *Service) Name() string { return "selfheal" }
func (s *Service) ServiceInstance() interface{} { return s }

func (s *Service) SetCompute(m interface{})      { s.compute = m }
func (s *Service) SetNotification(m interface{}) { s.notify = m }

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/self-heal")
	{
		api.GET("/events", s.listEvents)
	}
}

func (s *Service) listEvents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"events": []string{}})
}
