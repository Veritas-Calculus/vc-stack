// Package task provides a unified async task tracking framework.
// Every long-running operation (instance creation, migration, snapshot, backup)
// is represented as a Task with a well-defined lifecycle:
//
//	pending -> running -> completed / failed / cancelled
//
// Tasks can be queried for progress, filtered by resource, and cancelled.
package task

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Task represents an async operation in the system.
type Task struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UUID         string     `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Type         string     `gorm:"type:varchar(64);not null;index" json:"type"`                     // create_instance, delete_instance, migrate, snapshot, backup, resize
	Status       string     `gorm:"type:varchar(32);not null;default:'pending';index" json:"status"` // pending, running, completed, failed, cancelled
	Progress     int        `gorm:"default:0" json:"progress"`                                       // 0-100
	ResourceType string     `gorm:"type:varchar(64);index" json:"resource_type"`                     // instance, volume, snapshot, host
	ResourceID   string     `gorm:"type:varchar(36);index" json:"resource_id"`                       // UUID of the target resource
	ResourceName string     `json:"resource_name"`
	UserID       uint       `gorm:"index" json:"user_id"`
	ProjectID    uint       `gorm:"index" json:"project_id"`
	Message      string     `json:"message,omitempty"`                 // Human-readable status message
	ErrorMessage string     `json:"error_message,omitempty"`           // Error details on failure
	Result       string     `gorm:"type:text" json:"result,omitempty"` // JSON result on completion
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName overrides the default table name.
func (Task) TableName() string { return "sys_tasks" }

// Config contains the task service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides task tracking operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new task tracking service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{db: cfg.DB, logger: cfg.Logger}

	// Auto-migrate task table.
	if err := cfg.DB.AutoMigrate(&Task{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tasks table: %w", err)
	}

	return svc, nil
}

// SetupRoutes registers HTTP routes for the task service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1")
	{
		api.GET("/tasks", rp("task", "list"), s.listTasks)
		api.GET("/tasks/:id", rp("task", "get"), s.getTask)
		api.POST("/tasks/:id/cancel", rp("task", "create"), s.cancelTask)
		api.DELETE("/tasks/:id", rp("task", "delete"), s.deleteTask)
	}
}

// --- Public API for other services to create/update tasks ---

// CreateTask creates a new task record and returns it.
func (s *Service) CreateTask(taskType, resourceType, resourceID, resourceName string, userID, projectID uint) (*Task, error) {
	task := &Task{
		UUID:         uuid.New().String(),
		Type:         taskType,
		Status:       "pending",
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		UserID:       userID,
		ProjectID:    projectID,
		Message:      fmt.Sprintf("%s operation queued", taskType),
	}
	if err := s.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	s.logger.Info("task created",
		zap.String("uuid", task.UUID),
		zap.String("type", taskType),
		zap.String("resource", resourceType+"/"+resourceID))
	return task, nil
}

// StartTask marks a task as running.
func (s *Service) StartTask(taskUUID string) {
	now := time.Now()
	_ = s.db.Model(&Task{}).Where("uuid = ?", taskUUID).Updates(map[string]interface{}{
		"status":     "running",
		"progress":   0,
		"started_at": now,
		"message":    "in progress",
	}).Error
}

// UpdateProgress updates the progress percentage and message.
func (s *Service) UpdateProgress(taskUUID string, progress int, message string) {
	updates := map[string]interface{}{
		"progress": progress,
	}
	if message != "" {
		updates["message"] = message
	}
	_ = s.db.Model(&Task{}).Where("uuid = ?", taskUUID).Updates(updates).Error
}

// CompleteTask marks a task as completed successfully.
func (s *Service) CompleteTask(taskUUID, result string) {
	now := time.Now()
	_ = s.db.Model(&Task{}).Where("uuid = ?", taskUUID).Updates(map[string]interface{}{
		"status":       "completed",
		"progress":     100,
		"completed_at": now,
		"message":      "completed successfully",
		"result":       result,
	}).Error
	s.logger.Info("task completed", zap.String("uuid", taskUUID))
}

// FailTask marks a task as failed.
func (s *Service) FailTask(taskUUID, errorMessage string) {
	now := time.Now()
	_ = s.db.Model(&Task{}).Where("uuid = ?", taskUUID).Updates(map[string]interface{}{
		"status":        "failed",
		"completed_at":  now,
		"message":       "failed",
		"error_message": errorMessage,
	}).Error
	s.logger.Error("task failed", zap.String("uuid", taskUUID), zap.String("error", errorMessage))
}

// CancelTask marks a task as cancelled.
func (s *Service) CancelTask(taskUUID string) error {
	var task Task
	if err := s.db.Where("uuid = ?", taskUUID).First(&task).Error; err != nil {
		return fmt.Errorf("task not found")
	}
	if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
		return fmt.Errorf("task is already %s", task.Status)
	}
	now := time.Now()
	return s.db.Model(&task).Updates(map[string]interface{}{
		"status":       "cancelled",
		"completed_at": now,
		"message":      "cancelled by user",
	}).Error
}

// GetTaskByUUID retrieves a task by its UUID.
func (s *Service) GetTaskByUUID(taskUUID string) (*Task, error) {
	var task Task
	if err := s.db.Where("uuid = ?", taskUUID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// --- HTTP Handlers ---

// listTasks handles GET /api/v1/tasks.
func (s *Service) listTasks(c *gin.Context) {
	var tasks []Task
	query := s.db.Order("id DESC")

	// Filters.
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if taskType := c.Query("type"); taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if resourceID := c.Query("resource_id"); resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// Pagination.
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var total int64
	query.Model(&Task{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		s.logger.Error("failed to list tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// getTask handles GET /api/v1/tasks/:id.
func (s *Service) getTask(c *gin.Context) {
	id := c.Param("id")
	var task Task

	// Try UUID first, then numeric ID.
	err := s.db.Where("uuid = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&task, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

// cancelTask handles POST /api/v1/tasks/:id/cancel.
func (s *Service) cancelTask(c *gin.Context) {
	id := c.Param("id")
	var task Task

	err := s.db.Where("uuid = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&task, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("task is already %s", task.Status),
		})
		return
	}

	now := time.Now()
	_ = s.db.Model(&task).Updates(map[string]interface{}{
		"status":       "cancelled",
		"completed_at": now,
		"message":      "cancelled by user",
	}).Error

	c.JSON(http.StatusOK, gin.H{"ok": true, "task": task})
}

// deleteTask handles DELETE /api/v1/tasks/:id.
// Only completed/failed/cancelled tasks can be deleted.
func (s *Service) deleteTask(c *gin.Context) {
	id := c.Param("id")
	var task Task

	err := s.db.Where("uuid = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&task, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	if task.Status == "pending" || task.Status == "running" {
		c.JSON(http.StatusConflict, gin.H{
			"error": "cannot delete an active task; cancel it first",
		})
		return
	}

	if err := s.db.Delete(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
