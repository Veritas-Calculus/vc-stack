package invoice

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
	usage  interface{} // Injected usage service
}

func NewService(cfg Config) (*Service, error) {
	return &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}, nil
}

func (s *Service) Name() string { return "invoice" }
func (s *Service) ServiceInstance() interface{} { return s }

// SetUsageService injects usage data source.
func (s *Service) SetUsageService(m interface{}) {
	s.usage = m
}

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/invoices")
	{
		api.GET("", s.listInvoices)
	}
}

func (s *Service) listInvoices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"invoices": []string{}})
}
