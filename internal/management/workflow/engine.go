package workflow

import (
	"context"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/vcredis"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TaskStatus represents the current state of a workflow task.
type TaskStatus string

const (
	StatusPending     TaskStatus = "pending"
	StatusRunning     TaskStatus = "running"
	StatusCompleted   TaskStatus = "completed"
	StatusFailed      TaskStatus = "failed"
	StatusRollingBack TaskStatus = "rolling_back"
	StatusRolledBack  TaskStatus = "rolled_back"
)

// Task represents a persistent workflow record.
type Task struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Type         string     `gorm:"index" json:"type"`
	ResourceUUID string     `gorm:"index" json:"resource_uuid"`
	Status       TaskStatus `gorm:"default:'pending'" json:"status"`
	CurrentStep  int        `json:"current_step"`
	Payload      string     `gorm:"type:text" json:"payload"`
	Error        string     `json:"error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

func (Task) TableName() string { return "workflow_tasks" }

// Step defines a single action in a workflow.
type Step interface {
	Name() string
	Execute(ctx context.Context, task *Task) error
	Compensate(ctx context.Context, task *Task) error
}

// Workflow defines a sequence of steps.
type Workflow struct {
	Name  string
	Steps []Step
}

// AuditLogger defines the interface for recording workflow events.
type AuditLogger interface {
	LogEvent(resource, resourceID, action, status, userID, message string)
}

// Engine manages task execution and persistence.
type Engine struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry map[string]*Workflow
	audit    AuditLogger
	redis    *vcredis.Manager
}

func NewEngine(db *gorm.DB, logger *zap.Logger, audit AuditLogger, redis *vcredis.Manager) *Engine {
	return &Engine{
		db:       db,
		logger:   logger,
		registry: make(map[string]*Workflow),
		audit:    audit,
		redis:    redis,
	}
}

// RegisterWorkflow registers a workflow template by type.
func (e *Engine) RegisterWorkflow(taskType string, wf *Workflow) {
	e.registry[taskType] = wf
}

// AcquireLock attempts to lock a resource for an operation.
func (e *Engine) AcquireLock(ctx context.Context, resourceUUID string, ttl time.Duration) (bool, error) {
	if e.redis == nil {
		return true, nil
	}
	key := "lock:wf:" + resourceUUID
	return e.redis.Client().SetNX(ctx, key, "locked", ttl).Result()
}

// ReleaseLock removes the resource lock.
func (e *Engine) ReleaseLock(ctx context.Context, resourceUUID string) {
	if e.redis != nil {
		_ = e.redis.Client().Del(context.Background(), "lock:wf:"+resourceUUID)
	}
}
