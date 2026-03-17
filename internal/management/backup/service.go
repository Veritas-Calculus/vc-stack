package backup

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
	storage interface{} // Injected storage.Interface
}

func NewService(cfg Config) (*Service, error) {
	return &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}, nil
}

func (s *Service) Name() string { return "backup" }
func (s *Service) ServiceInstance() interface{} { return s }

// SetStorageManager injects the storage module for snapshot operations.
func (s *Service) SetStorageManager(m interface{}) {
	s.storage = m
}

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/backups")
	{
		api.GET("", s.listBackups)
	}
}

func (s *Service) listBackups(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"backups": []string{}})
}
