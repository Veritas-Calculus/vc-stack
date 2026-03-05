// Package orchestration provides template-based resource orchestration (stacks),
// similar to OpenStack Heat / AWS CloudFormation.
// Users define infrastructure as YAML templates, and the engine manages the
// full lifecycle: create, update, preview, delete, with dependency resolution.
package orchestration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// --- Constants ---

const (
	// Stack statuses.
	StackStatusCreateInProgress   = "CREATE_IN_PROGRESS"
	StackStatusCreateComplete     = "CREATE_COMPLETE"
	StackStatusCreateFailed       = "CREATE_FAILED"
	StackStatusUpdateInProgress   = "UPDATE_IN_PROGRESS"
	StackStatusUpdateComplete     = "UPDATE_COMPLETE"
	StackStatusUpdateFailed       = "UPDATE_FAILED"
	StackStatusDeleteInProgress   = "DELETE_IN_PROGRESS"
	StackStatusDeleteComplete     = "DELETE_COMPLETE"
	StackStatusDeleteFailed       = "DELETE_FAILED"
	StackStatusRollbackInProgress = "ROLLBACK_IN_PROGRESS"
	StackStatusRollbackComplete   = "ROLLBACK_COMPLETE"

	// Resource statuses.
	ResourceStatusCreateInProgress = "CREATE_IN_PROGRESS"
	ResourceStatusCreateComplete   = "CREATE_COMPLETE"
	ResourceStatusCreateFailed     = "CREATE_FAILED"
	ResourceStatusDeleteInProgress = "DELETE_IN_PROGRESS"
	ResourceStatusDeleteComplete   = "DELETE_COMPLETE"

	// Event types.
	EventCreate = "CREATE"
	EventUpdate = "UPDATE"
	EventDelete = "DELETE"

	// Supported resource types.
	ResourceTypeInstance      = "VC::Compute::Instance"
	ResourceTypeVolume        = "VC::Storage::Volume"
	ResourceTypeNetwork       = "VC::Network::Network"
	ResourceTypeSubnet        = "VC::Network::Subnet"
	ResourceTypeSecurityGroup = "VC::Network::SecurityGroup"
	ResourceTypeFloatingIP    = "VC::Network::FloatingIP"
	ResourceTypeDNSZone       = "VC::DNS::Zone"
	ResourceTypeDNSRecord     = "VC::DNS::RecordSet"
	ResourceTypeBucket        = "VC::ObjectStorage::Bucket"
	ResourceTypeRouter        = "VC::Network::Router"
	ResourceTypeKeypair       = "VC::Compute::KeyPair"

	// Maximum template size.
	MaxTemplateSize = 512 * 1024 // 512 KB
)

// --- Models ---

// Stack represents a deployed set of resources from a template.
type Stack struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name            string    `json:"name" gorm:"not null;index"`
	Description     string    `json:"description"`
	ProjectID       string    `json:"project_id" gorm:"type:varchar(36);index"`
	Status          string    `json:"status" gorm:"default:'CREATE_IN_PROGRESS';index"`
	StatusReason    string    `json:"status_reason"`
	Template        string    `json:"-" gorm:"type:text"` // Raw YAML template (hidden from list)
	TemplateDesc    string    `json:"template_description,omitempty"`
	Parameters      string    `json:"parameters,omitempty" gorm:"type:text"` // JSON key-value
	Outputs         string    `json:"outputs,omitempty" gorm:"type:text"`    // JSON outputs
	Tags            string    `json:"tags,omitempty"`
	Timeout         int       `json:"timeout_mins" gorm:"column:timeout_mins;default:60"` // Deployment timeout in minutes
	DisableRollback bool      `json:"disable_rollback" gorm:"default:false"`
	ParentID        string    `json:"parent_id,omitempty" gorm:"type:varchar(36)"` // For nested stacks
	ResourceCount   int       `json:"resource_count" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Stack) TableName() string { return "orchestration_stacks" }

// StackResource represents a single resource within a stack.
type StackResource struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	StackID      string    `json:"stack_id" gorm:"type:varchar(36);index"`
	LogicalID    string    `json:"logical_id"`            // Name in template (e.g., "web_server")
	PhysicalID   string    `json:"physical_id,omitempty"` // Actual resource ID after creation
	Type         string    `json:"type"`                  // e.g., "VC::Compute::Instance"
	Status       string    `json:"status" gorm:"default:'CREATE_IN_PROGRESS'"`
	StatusReason string    `json:"status_reason"`
	Properties   string    `json:"properties,omitempty" gorm:"type:text"` // JSON properties
	DependsOn    string    `json:"depends_on,omitempty"`                  // Comma-separated logical IDs
	RequiredBy   string    `json:"required_by,omitempty"`                 // Comma-separated logical IDs
	Metadata     string    `json:"metadata,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (StackResource) TableName() string { return "orchestration_resources" }

// StackEvent records lifecycle events for a stack or resource.
type StackEvent struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	StackID      string    `json:"stack_id" gorm:"type:varchar(36);index"`
	ResourceID   string    `json:"resource_id,omitempty" gorm:"type:varchar(36)"`
	LogicalID    string    `json:"logical_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	EventType    string    `json:"event_type"` // CREATE, UPDATE, DELETE
	Status       string    `json:"status"`
	StatusReason string    `json:"status_reason"`
	PhysicalID   string    `json:"physical_id,omitempty"`
	Timestamp    time.Time `json:"timestamp" gorm:"autoCreateTime"`
}

func (StackEvent) TableName() string { return "orchestration_events" }

// StackTemplate represents a reusable, versioned template.
type StackTemplate struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;index"`
	Description string    `json:"description"`
	ProjectID   string    `json:"project_id" gorm:"type:varchar(36);index"`
	Version     string    `json:"version" gorm:"default:'1.0'"`
	Template    string    `json:"template" gorm:"type:text"` // YAML content
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	Category    string    `json:"category,omitempty"` // web, database, network, etc.
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (StackTemplate) TableName() string { return "orchestration_templates" }

// --- Template Schema (simplified YAML representation in JSON) ---

// TemplateResource defines a resource in a parsed template.
type TemplateResource struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ParsedTemplate represents a parsed stack template.
type ParsedTemplate struct {
	Description string                      `json:"description"`
	Parameters  map[string]TemplateParam    `json:"parameters,omitempty"`
	Resources   map[string]TemplateResource `json:"resources"`
	Outputs     map[string]TemplateOutput   `json:"outputs,omitempty"`
}

// TemplateParam defines a template parameter.
type TemplateParam struct {
	Type        string      `json:"type"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Constraints []string    `json:"constraints,omitempty"`
}

// TemplateOutput defines a template output value.
type TemplateOutput struct {
	Description string      `json:"description,omitempty"`
	Value       interface{} `json:"value"`
}

// --- Request/Response Types ---

type CreateStackRequest struct {
	Name            string            `json:"name" binding:"required"`
	Description     string            `json:"description"`
	Template        string            `json:"template" binding:"required"` // JSON template (simplified)
	Parameters      map[string]string `json:"parameters"`
	Tags            string            `json:"tags"`
	TimeoutMins     int               `json:"timeout_mins"`
	DisableRollback bool              `json:"disable_rollback"`
}

type UpdateStackRequest struct {
	Description string            `json:"description"`
	Template    string            `json:"template"`
	Parameters  map[string]string `json:"parameters"`
	Tags        string            `json:"tags"`
	TimeoutMins int               `json:"timeout_mins"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Template    string `json:"template" binding:"required"`
	Version     string `json:"version"`
	IsPublic    bool   `json:"is_public"`
	Category    string `json:"category"`
}

type PreviewResponse struct {
	Resources []PreviewResource `json:"resources"`
	Warnings  []string          `json:"warnings,omitempty"`
}

type PreviewResource struct {
	LogicalID string `json:"logical_id"`
	Type      string `json:"type"`
	Action    string `json:"action"`
	DependsOn string `json:"depends_on,omitempty"`
}

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
	orch := rg.Group("/stacks")
	{
		orch.GET("", s.listStacks)
		orch.POST("", s.createStack)
		orch.GET("/:stack_id", s.getStack)
		orch.PUT("/:stack_id", s.updateStack)
		orch.DELETE("/:stack_id", s.deleteStack)

		// Resources within a stack.
		orch.GET("/:stack_id/resources", s.listResources)
		orch.GET("/:stack_id/resources/:resource_id", s.getResource)

		// Events.
		orch.GET("/:stack_id/events", s.listEvents)

		// Template preview (dry run).
		orch.POST("/preview", s.previewStack)

		// Get template from a running stack.
		orch.GET("/:stack_id/template", s.getStackTemplate)
	}

	// Reusable templates library.
	tpl := rg.Group("/templates")
	{
		tpl.GET("", s.listTemplates)
		tpl.POST("", s.createTemplate)
		tpl.GET("/:template_id", s.getTemplate)
		tpl.DELETE("/:template_id", s.deleteTemplate)
	}
}

// --- Stack Handlers ---

func (s *Service) listStacks(c *gin.Context) {
	var stacks []Stack
	query := s.db.Where("status != ?", StackStatusDeleteComplete).Order("created_at DESC")

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if tag := c.Query("tag"); tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	if err := query.Find(&stacks).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list stacks"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stacks":   stacks,
		"metadata": gin.H{"total_count": len(stacks)},
	})
}

func (s *Service) createStack(c *gin.Context) {
	var req CreateStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	if len(req.Template) > MaxTemplateSize {
		apierrors.Respond(c, apierrors.ErrInvalidParam("template",
			fmt.Sprintf("template exceeds maximum size of %d KB", MaxTemplateSize/1024)))
		return
	}

	// Parse the template.
	parsed, err := parseTemplate(req.Template)
	if err != nil {
		apierrors.Respond(c, apierrors.ErrInvalidParam("template", err.Error()))
		return
	}

	if len(parsed.Resources) == 0 {
		apierrors.Respond(c, apierrors.ErrInvalidParam("template", "template must define at least one resource"))
		return
	}

	// Validate resource types.
	for logicalID, res := range parsed.Resources {
		if !isValidResourceType(res.Type) {
			apierrors.Respond(c, apierrors.ErrInvalidParam("template",
				fmt.Sprintf("unsupported resource type %q for resource %q", res.Type, logicalID)))
			return
		}
	}

	// Validate dependency graph (detect cycles).
	if err := validateDependencies(parsed.Resources); err != nil {
		apierrors.Respond(c, apierrors.ErrInvalidParam("template", err.Error()))
		return
	}

	projectID := c.GetString("tenant_id")
	timeout := req.TimeoutMins
	if timeout <= 0 {
		timeout = 60
	}

	paramsJSON, _ := json.Marshal(req.Parameters)
	outputsJSON, _ := json.Marshal(parsed.Outputs)

	stack := &Stack{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Description:     req.Description,
		ProjectID:       projectID,
		Status:          StackStatusCreateInProgress,
		StatusReason:    "Stack creation initiated",
		Template:        req.Template,
		TemplateDesc:    parsed.Description,
		Parameters:      string(paramsJSON),
		Outputs:         string(outputsJSON),
		Tags:            req.Tags,
		Timeout:         timeout,
		DisableRollback: req.DisableRollback,
		ResourceCount:   len(parsed.Resources),
	}

	// Check for duplicate name within project.
	var existing int64
	s.db.Model(&Stack{}).Where("name = ? AND project_id = ? AND status NOT IN ?",
		req.Name, projectID, []string{StackStatusDeleteComplete}).Count(&existing)
	if existing > 0 {
		apierrors.Respond(c, apierrors.ErrAlreadyExists("stack", req.Name))
		return
	}

	if err := s.db.Create(stack).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("create stack"))
		return
	}

	// Create resources from template.
	order := topologicalSort(parsed.Resources)
	for _, logicalID := range order {
		res := parsed.Resources[logicalID]
		propsJSON, _ := json.Marshal(res.Properties)
		metaJSON, _ := json.Marshal(res.Metadata)

		resource := &StackResource{
			ID:         uuid.New().String(),
			StackID:    stack.ID,
			LogicalID:  logicalID,
			Type:       res.Type,
			Status:     ResourceStatusCreateComplete, // In dev mode, resources are created immediately.
			Properties: string(propsJSON),
			DependsOn:  strings.Join(res.DependsOn, ","),
			Metadata:   string(metaJSON),
			PhysicalID: fmt.Sprintf("sim-%s-%s", strings.ToLower(logicalID), uuid.New().String()[:8]),
		}

		// Build required_by from inverse dependencies.
		var requiredBy []string
		for otherID, otherRes := range parsed.Resources {
			for _, dep := range otherRes.DependsOn {
				if dep == logicalID {
					requiredBy = append(requiredBy, otherID)
				}
			}
		}
		resource.RequiredBy = strings.Join(requiredBy, ",")

		if err := s.db.Create(resource).Error; err != nil {
			s.logger.Error("failed to create resource", zap.Error(err))
		}

		// Record event.
		s.recordEvent(stack.ID, resource.ID, logicalID, res.Type, EventCreate,
			ResourceStatusCreateComplete, "Resource created successfully", resource.PhysicalID)
	}

	// Mark stack as complete.
	s.db.Model(stack).Updates(map[string]interface{}{
		"status":        StackStatusCreateComplete,
		"status_reason": "Stack CREATE completed successfully",
	})
	stack.Status = StackStatusCreateComplete
	stack.StatusReason = "Stack CREATE completed successfully"

	// Record stack-level event.
	s.recordEvent(stack.ID, "", stack.Name, "VC::Orchestration::Stack", EventCreate,
		StackStatusCreateComplete, "Stack CREATE completed successfully", stack.ID)

	s.logger.Info("stack created", zap.String("name", req.Name), zap.String("id", stack.ID),
		zap.Int("resources", len(parsed.Resources)))

	c.JSON(http.StatusCreated, gin.H{"stack": stack})
}

func (s *Service) getStack(c *gin.Context) {
	stackID := c.Param("stack_id")
	var stack Stack
	if err := s.db.First(&stack, "id = ? AND status != ?", stackID, StackStatusDeleteComplete).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("stack", stackID))
		return
	}

	// Count resources.
	var resourceCount int64
	s.db.Model(&StackResource{}).Where("stack_id = ?", stackID).Count(&resourceCount)

	c.JSON(http.StatusOK, gin.H{
		"stack":          stack,
		"resource_count": resourceCount,
	})
}

func (s *Service) updateStack(c *gin.Context) {
	stackID := c.Param("stack_id")
	var stack Stack
	if err := s.db.First(&stack, "id = ? AND status NOT IN ?", stackID,
		[]string{StackStatusDeleteComplete, StackStatusDeleteInProgress}).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("stack", stackID))
		return
	}

	var req UpdateStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	updates := map[string]interface{}{
		"status":        StackStatusUpdateInProgress,
		"status_reason": "Stack update initiated",
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}
	if req.TimeoutMins > 0 {
		updates["timeout_mins"] = req.TimeoutMins
	}
	if req.Template != "" {
		parsed, err := parseTemplate(req.Template)
		if err != nil {
			apierrors.Respond(c, apierrors.ErrInvalidParam("template", err.Error()))
			return
		}
		updates["template"] = req.Template
		updates["template_description"] = parsed.Description
		updates["resource_count"] = len(parsed.Resources)
	}
	if len(req.Parameters) > 0 {
		paramsJSON, _ := json.Marshal(req.Parameters)
		updates["parameters"] = string(paramsJSON)
	}

	// Complete the update.
	updates["status"] = StackStatusUpdateComplete
	updates["status_reason"] = "Stack UPDATE completed successfully"

	if err := s.db.Model(&stack).Updates(updates).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("update stack"))
		return
	}

	// Record event.
	s.recordEvent(stackID, "", stack.Name, "VC::Orchestration::Stack", EventUpdate,
		StackStatusUpdateComplete, "Stack UPDATE completed successfully", stackID)

	_ = s.db.First(&stack, "id = ?", stackID).Error
	c.JSON(http.StatusOK, gin.H{"stack": stack})
}

func (s *Service) deleteStack(c *gin.Context) {
	stackID := c.Param("stack_id")
	var stack Stack
	if err := s.db.First(&stack, "id = ? AND status != ?", stackID, StackStatusDeleteComplete).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("stack", stackID))
		return
	}

	// Mark as deleting.
	s.db.Model(&stack).Updates(map[string]interface{}{
		"status":        StackStatusDeleteInProgress,
		"status_reason": "Stack deletion initiated",
	})

	// Delete resources in reverse dependency order.
	var resources []StackResource
	s.db.Where("stack_id = ?", stackID).Find(&resources)
	for i := len(resources) - 1; i >= 0; i-- {
		r := resources[i]
		s.db.Model(&r).Updates(map[string]interface{}{
			"status":        ResourceStatusDeleteComplete,
			"status_reason": "Resource deleted",
		})
		s.recordEvent(stackID, r.ID, r.LogicalID, r.Type, EventDelete,
			ResourceStatusDeleteComplete, "Resource DELETE completed", r.PhysicalID)
	}

	// Clean up resources and mark stack as deleted.
	s.db.Where("stack_id = ?", stackID).Delete(&StackResource{})
	s.db.Model(&stack).Updates(map[string]interface{}{
		"status":        StackStatusDeleteComplete,
		"status_reason": "Stack DELETE completed successfully",
	})

	s.recordEvent(stackID, "", stack.Name, "VC::Orchestration::Stack", EventDelete,
		StackStatusDeleteComplete, "Stack DELETE completed successfully", stackID)

	s.logger.Info("stack deleted", zap.String("name", stack.Name), zap.String("id", stackID))
	c.JSON(http.StatusOK, gin.H{"message": "stack deleted", "id": stackID})
}

// --- Resource Handlers ---

func (s *Service) listResources(c *gin.Context) {
	stackID := c.Param("stack_id")
	if !s.stackExists(stackID) {
		apierrors.Respond(c, apierrors.ErrNotFound("stack", stackID))
		return
	}

	var resources []StackResource
	query := s.db.Where("stack_id = ?", stackID).Order("created_at ASC")
	if resType := c.Query("type"); resType != "" {
		query = query.Where("type = ?", resType)
	}
	_ = query.Find(&resources).Error

	c.JSON(http.StatusOK, gin.H{
		"resources": resources,
		"metadata":  gin.H{"total_count": len(resources)},
	})
}

func (s *Service) getResource(c *gin.Context) {
	resourceID := c.Param("resource_id")
	var resource StackResource
	if err := s.db.First(&resource, "id = ?", resourceID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("resource", resourceID))
		return
	}
	c.JSON(http.StatusOK, gin.H{"resource": resource})
}

// --- Event Handlers ---

func (s *Service) listEvents(c *gin.Context) {
	stackID := c.Param("stack_id")
	if !s.stackExists(stackID) {
		apierrors.Respond(c, apierrors.ErrNotFound("stack", stackID))
		return
	}

	var events []StackEvent
	query := s.db.Where("stack_id = ?", stackID).Order("timestamp DESC").Limit(200)
	if resType := c.Query("resource_type"); resType != "" {
		query = query.Where("resource_type = ?", resType)
	}
	_ = query.Find(&events).Error

	c.JSON(http.StatusOK, gin.H{
		"events":   events,
		"metadata": gin.H{"total_count": len(events)},
	})
}

// --- Template/Preview Handlers ---

func (s *Service) previewStack(c *gin.Context) {
	var req CreateStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	parsed, err := parseTemplate(req.Template)
	if err != nil {
		apierrors.Respond(c, apierrors.ErrInvalidParam("template", err.Error()))
		return
	}

	var warnings []string
	var resources []PreviewResource

	order := topologicalSort(parsed.Resources)
	for _, logicalID := range order {
		res := parsed.Resources[logicalID]
		if !isValidResourceType(res.Type) {
			warnings = append(warnings, fmt.Sprintf("unsupported type %q for %q", res.Type, logicalID))
		}
		resources = append(resources, PreviewResource{
			LogicalID: logicalID,
			Type:      res.Type,
			Action:    "CREATE",
			DependsOn: strings.Join(res.DependsOn, ", "),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"preview": PreviewResponse{
			Resources: resources,
			Warnings:  warnings,
		},
	})
}

func (s *Service) getStackTemplate(c *gin.Context) {
	stackID := c.Param("stack_id")
	var stack Stack
	if err := s.db.First(&stack, "id = ?", stackID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("stack", stackID))
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": stack.Template})
}

// --- Template Library Handlers ---

func (s *Service) listTemplates(c *gin.Context) {
	var templates []StackTemplate
	query := s.db.Order("name ASC")

	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if isPublic := c.Query("public"); isPublic == "true" {
		query = query.Where("is_public = ?", true)
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	_ = query.Find(&templates).Error
	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"metadata":  gin.H{"total_count": len(templates)},
	})
}

func (s *Service) createTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	// Validate the template content.
	if _, err := parseTemplate(req.Template); err != nil {
		apierrors.Respond(c, apierrors.ErrInvalidParam("template", err.Error()))
		return
	}

	version := req.Version
	if version == "" {
		version = "1.0"
	}

	tpl := &StackTemplate{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		ProjectID:   c.GetString("tenant_id"),
		Version:     version,
		Template:    req.Template,
		IsPublic:    req.IsPublic,
		Category:    req.Category,
	}

	if err := s.db.Create(tpl).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("create template"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"template": tpl})
}

func (s *Service) getTemplate(c *gin.Context) {
	templateID := c.Param("template_id")
	var tpl StackTemplate
	if err := s.db.First(&tpl, "id = ?", templateID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("template", templateID))
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": tpl})
}

func (s *Service) deleteTemplate(c *gin.Context) {
	templateID := c.Param("template_id")
	if err := s.db.Delete(&StackTemplate{}, "id = ?", templateID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("template", templateID))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

// --- Helpers ---

func (s *Service) stackExists(stackID string) bool {
	var count int64
	s.db.Model(&Stack{}).Where("id = ? AND status != ?", stackID, StackStatusDeleteComplete).Count(&count)
	return count > 0
}

func (s *Service) recordEvent(stackID, resourceID, logicalID, resourceType, eventType, status, reason, physicalID string) {
	event := &StackEvent{
		StackID:      stackID,
		ResourceID:   resourceID,
		LogicalID:    logicalID,
		ResourceType: resourceType,
		EventType:    eventType,
		Status:       status,
		StatusReason: reason,
		PhysicalID:   physicalID,
	}
	if err := s.db.Create(event).Error; err != nil {
		s.logger.Error("failed to record event", zap.Error(err))
	}
}

// parseTemplate parses a JSON template string.
func parseTemplate(raw string) (*ParsedTemplate, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("template is empty")
	}

	var parsed ParsedTemplate
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid template JSON: %w", err)
	}

	if len(parsed.Resources) == 0 {
		return nil, fmt.Errorf("template must contain at least one resource")
	}

	return &parsed, nil
}

// isValidResourceType checks if a resource type is supported.
func isValidResourceType(t string) bool {
	valid := map[string]bool{
		ResourceTypeInstance:      true,
		ResourceTypeVolume:        true,
		ResourceTypeNetwork:       true,
		ResourceTypeSubnet:        true,
		ResourceTypeSecurityGroup: true,
		ResourceTypeFloatingIP:    true,
		ResourceTypeDNSZone:       true,
		ResourceTypeDNSRecord:     true,
		ResourceTypeBucket:        true,
		ResourceTypeRouter:        true,
		ResourceTypeKeypair:       true,
	}
	return valid[t]
}

// validateDependencies checks for circular dependencies in the resource graph.
func validateDependencies(resources map[string]TemplateResource) error {
	// Check that all dependencies reference existing resources.
	for logicalID, res := range resources {
		for _, dep := range res.DependsOn {
			if _, exists := resources[dep]; !exists {
				return fmt.Errorf("resource %q depends on undefined resource %q", logicalID, dep)
			}
		}
	}

	// Detect cycles using DFS.
	visited := map[string]int{} // 0=unvisited, 1=visiting, 2=visited
	var dfs func(string) error
	dfs = func(node string) error {
		if visited[node] == 1 {
			return fmt.Errorf("circular dependency detected involving %q", node)
		}
		if visited[node] == 2 {
			return nil
		}
		visited[node] = 1
		for _, dep := range resources[node].DependsOn {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		visited[node] = 2
		return nil
	}

	for node := range resources {
		if visited[node] == 0 {
			if err := dfs(node); err != nil {
				return err
			}
		}
	}
	return nil
}

// topologicalSort returns resource creation order respecting dependencies.
func topologicalSort(resources map[string]TemplateResource) []string {
	visited := map[string]bool{}
	var order []string

	var visit func(string)
	visit = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true
		for _, dep := range resources[node].DependsOn {
			visit(dep)
		}
		order = append(order, node)
	}

	// Sort keys for deterministic order.
	keys := make([]string, 0, len(resources))
	for k := range resources {
		keys = append(keys, k)
	}
	// Simple sort.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, k := range keys {
		visit(k)
	}
	return order
}
