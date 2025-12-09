// Package compute provides resource management utilities.
package compute

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourceTracker tracks resource allocations and deallocations.
type ResourceTracker struct {
	mu        sync.RWMutex
	allocated map[string]*ResourceAllocation
	logger    *zap.Logger
}

// ResourceAllocation represents an allocated resource.
type ResourceAllocation struct {
	ResourceID   string
	ResourceType string
	ProjectID    uint
	UserID       uint
	AllocatedAt  time.Time
	Metadata     map[string]string
}

// NewResourceTracker creates a new resource tracker.
func NewResourceTracker(logger *zap.Logger) *ResourceTracker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ResourceTracker{
		allocated: make(map[string]*ResourceAllocation),
		logger:    logger,
	}
}

// Track records a new resource allocation.
func (rt *ResourceTracker) Track(alloc *ResourceAllocation) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.allocated[alloc.ResourceID] = alloc
	rt.logger.Info("resource allocated",
		zap.String("resource_id", alloc.ResourceID),
		zap.String("resource_type", alloc.ResourceType),
		zap.Uint("project_id", alloc.ProjectID))
}

// Release removes a resource allocation.
func (rt *ResourceTracker) Release(resourceID string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if alloc, exists := rt.allocated[resourceID]; exists {
		rt.logger.Info("resource released",
			zap.String("resource_id", resourceID),
			zap.String("resource_type", alloc.ResourceType),
			zap.Duration("lifetime", time.Since(alloc.AllocatedAt)))
		delete(rt.allocated, resourceID)
	}
}

// Get retrieves a resource allocation.
func (rt *ResourceTracker) Get(resourceID string) (*ResourceAllocation, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	alloc, exists := rt.allocated[resourceID]
	return alloc, exists
}

// ListByProject returns all allocations for a project.
func (rt *ResourceTracker) ListByProject(projectID uint) []*ResourceAllocation {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var result []*ResourceAllocation
	for _, alloc := range rt.allocated {
		if alloc.ProjectID == projectID {
			result = append(result, alloc)
		}
	}
	return result
}

// Count returns the total number of tracked resources.
func (rt *ResourceTracker) Count() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return len(rt.allocated)
}

// OperationContext wraps context with operation metadata.
type OperationContext struct {
	ctx        context.Context
	OpID       string
	OpType     string
	ResourceID string
	UserID     uint
	ProjectID  uint
	StartTime  time.Time
	Logger     *zap.Logger
}

// NewOperationContext creates a new operation context.
func NewOperationContext(ctx context.Context, opType, resourceID string, logger *zap.Logger) *OperationContext {
	return &OperationContext{
		ctx:        ctx,
		OpID:       generateOpID(),
		OpType:     opType,
		ResourceID: resourceID,
		StartTime:  time.Now(),
		Logger:     logger.With(zap.String("op_id", generateOpID())),
	}
}

// Context returns the underlying context.
func (oc *OperationContext) Context() context.Context {
	return oc.ctx
}

// Duration returns elapsed time since operation start.
func (oc *OperationContext) Duration() time.Duration {
	return time.Since(oc.StartTime)
}

// LogSuccess logs successful operation completion.
func (oc *OperationContext) LogSuccess() {
	oc.Logger.Info("operation completed",
		zap.String("op_type", oc.OpType),
		zap.String("resource_id", oc.ResourceID),
		zap.Duration("duration", oc.Duration()))
}

// LogError logs operation failure.
func (oc *OperationContext) LogError(err error) {
	oc.Logger.Error("operation failed",
		zap.String("op_type", oc.OpType),
		zap.String("resource_id", oc.ResourceID),
		zap.Duration("duration", oc.Duration()),
		zap.Error(err))
}

// generateOpID generates a unique operation ID.
func generateOpID() string {
	return fmt.Sprintf("op-%d", time.Now().UnixNano())
}

// TaskQueue manages background task execution.
type TaskQueue struct {
	mu      sync.Mutex
	queue   []Task
	workers int
	stopCh  chan struct{}
	taskCh  chan Task
	wg      sync.WaitGroup
	logger  *zap.Logger
}

// Task represents a background task.
type Task interface {
	Execute(ctx context.Context) error
	GetID() string
	GetType() string
}

// NewTaskQueue creates a new task queue.
func NewTaskQueue(workers int, logger *zap.Logger) *TaskQueue {
	if workers <= 0 {
		workers = 5
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	tq := &TaskQueue{
		workers: workers,
		stopCh:  make(chan struct{}),
		taskCh:  make(chan Task, 100),
		logger:  logger,
	}

	tq.start()
	return tq
}

// Submit adds a task to the queue.
func (tq *TaskQueue) Submit(task Task) error {
	select {
	case tq.taskCh <- task:
		tq.logger.Debug("task queued",
			zap.String("task_id", task.GetID()),
			zap.String("task_type", task.GetType()))
		return nil
	case <-tq.stopCh:
		return fmt.Errorf("queue is closed")
	default:
		return fmt.Errorf("queue is full")
	}
}

// Stop gracefully shuts down the task queue.
func (tq *TaskQueue) Stop(ctx context.Context) error {
	close(tq.stopCh)

	done := make(chan struct{})
	go func() {
		tq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		tq.logger.Info("task queue stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// start initializes worker goroutines.
func (tq *TaskQueue) start() {
	for i := 0; i < tq.workers; i++ {
		tq.wg.Add(1)
		go tq.worker(i)
	}
	tq.logger.Info("task queue started", zap.Int("workers", tq.workers))
}

// worker processes tasks from the queue.
func (tq *TaskQueue) worker(id int) {
	defer tq.wg.Done()

	logger := tq.logger.With(zap.Int("worker_id", id))
	logger.Info("worker started")

	for {
		select {
		case task := <-tq.taskCh:
			if task == nil {
				continue
			}
			tq.executeTask(task, logger)
		case <-tq.stopCh:
			logger.Info("worker stopped")
			return
		}
	}
}

// executeTask runs a task with timeout and error handling.
func (tq *TaskQueue) executeTask(task Task, logger *zap.Logger) {
	logger = logger.With(
		zap.String("task_id", task.GetID()),
		zap.String("task_type", task.GetType()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	if err := task.Execute(ctx); err != nil {
		logger.Error("task failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)))
		return
	}

	logger.Info("task completed", zap.Duration("duration", time.Since(start)))
}

// CircuitBreaker implements circuit breaker pattern.
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        string // closed, open, half-open
	failures     int
	successes    int
	threshold    int
	timeout      time.Duration
	lastFailTime time.Time
	logger       *zap.Logger
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(threshold int, timeout time.Duration, logger *zap.Logger) *CircuitBreaker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CircuitBreaker{
		state:     "closed",
		threshold: threshold,
		timeout:   timeout,
		logger:    logger,
	}
}

// Call executes a function through the circuit breaker.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if we should transition from open to half-open.
	if cb.state == "open" && time.Since(cb.lastFailTime) >= cb.timeout {
		cb.state = "half-open"
		cb.successes = 0
		cb.logger.Info("circuit breaker transitioned to half-open")
	}

	// Reject calls when circuit is open.
	if cb.state == "open" {
		cb.mu.Unlock()
		return fmt.Errorf("circuit breaker is open")
	}

	cb.mu.Unlock()

	// Execute function.
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		// Transition to open if threshold exceeded.
		if cb.failures >= cb.threshold {
			cb.state = "open"
			cb.logger.Warn("circuit breaker opened",
				zap.Int("failures", cb.failures),
				zap.Int("threshold", cb.threshold))
		}
		return err
	}

	// Success handling.
	if cb.state == "half-open" {
		cb.successes++
		// Close circuit if enough consecutive successes.
		if cb.successes >= 2 {
			cb.state = "closed"
			cb.failures = 0
			cb.successes = 0
			cb.logger.Info("circuit breaker closed")
		}
	} else {
		cb.failures = 0
	}

	return nil
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
