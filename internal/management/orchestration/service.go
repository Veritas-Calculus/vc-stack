// Package orchestration provides template-based resource orchestration (stacks),
// similar to OpenStack Heat / AWS CloudFormation.
// Users define infrastructure as YAML templates, and the engine manages the
// full lifecycle: create, update, preview, delete, with dependency resolution.
//
// File layout:
//   - service.go   — Config, Service struct, constructor, routes
//   - models.go    — Constants, GORM models, template schema, request/response types
//   - handlers.go  — HTTP handlers, template parsing, dependency helpers
package orchestration

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// --- Service ---

// Config configures the orchestration service.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service manages orchestration stacks, resources, events, and templates.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates and initializes the orchestration service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if err := cfg.DB.AutoMigrate(&Stack{}, &StackResource{}, &StackEvent{}, &StackTemplate{}); err != nil {
		cfg.Logger.Error("failed to migrate orchestration tables", zap.Error(err))
		return nil, fmt.Errorf("orchestration migration: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// SetupRoutes registers API endpoints.
func (s *Service) SetupRoutes(rg *gin.RouterGroup) {
	rp := middleware.RequirePermission
	orch := rg.Group("/stacks")
	{
		orch.GET("", rp("orchestration", "list"), s.listStacks)
		orch.POST("", rp("orchestration", "create"), s.createStack)
		orch.GET("/:stack_id", rp("orchestration", "get"), s.getStack)
		orch.PUT("/:stack_id", rp("orchestration", "update"), s.updateStack)
		orch.DELETE("/:stack_id", rp("orchestration", "delete"), s.deleteStack)

		// Resources within a stack.
		orch.GET("/:stack_id/resources", rp("orchestration", "get"), s.listResources)
		orch.GET("/:stack_id/resources/:resource_id", rp("orchestration", "get"), s.getResource)

		// Events.
		orch.GET("/:stack_id/events", rp("orchestration", "get"), s.listEvents)

		// Template preview (dry run).
		orch.POST("/preview", rp("orchestration", "create"), s.previewStack)

		// Get template from a running stack.
		orch.GET("/:stack_id/template", rp("orchestration", "get"), s.getStackTemplate)
	}

	// Reusable templates library.
	tpl := rg.Group("/templates")
	{
		tpl.GET("", rp("orchestration", "list"), s.listTemplates)
		tpl.POST("", rp("orchestration", "create"), s.createTemplate)
		tpl.GET("/:template_id", rp("orchestration", "get"), s.getTemplate)
		tpl.DELETE("/:template_id", rp("orchestration", "delete"), s.deleteTemplate)
	}
}
