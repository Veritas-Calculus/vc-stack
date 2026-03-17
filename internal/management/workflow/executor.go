package workflow

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RunByType executes a workflow by finding the template in the registry.
func (e *Engine) RunByType(ctx context.Context, task *Task, logger *zap.Logger) error {
	wf, ok := e.registry[task.Type]
	if !ok {
		return fmt.Errorf("no workflow registered for type: %s", task.Type)
	}
	return e.Run(ctx, task, wf, logger)
}

// Run executes a workflow for a given task.
func (e *Engine) Run(ctx context.Context, task *Task, wf *Workflow, logger *zap.Logger) error {
	task.Status = StatusRunning
	e.db.Save(task)

	for i := task.CurrentStep; i < len(wf.Steps); i++ {
		step := wf.Steps[i]
		logger.Info("Executing step", zap.String("workflow", wf.Name), zap.String("step", step.Name()))

		if err := step.Execute(ctx, task); err != nil {
			logger.Error("Step failed, initiating rollback", zap.String("step", step.Name()), zap.Error(err))
			task.Error = err.Error()

			// Audit failure
			if e.audit != nil {
				e.audit.LogEvent(task.Type, task.ResourceUUID, step.Name(), "failed", "", err.Error())
			}

			return e.rollback(ctx, task, wf, i, logger)
		}

		// Audit step success
		if e.audit != nil {
			e.audit.LogEvent(task.Type, task.ResourceUUID, step.Name(), "success", "", "")
		}

		task.CurrentStep = i + 1
		e.db.Save(task)
	}

	now := time.Now()
	task.Status = StatusCompleted
	task.CompletedAt = &now
	e.db.Save(task)

	logger.Info("Workflow completed successfully", zap.String("workflow", wf.Name), zap.String("resource", task.ResourceUUID))
	return nil
}

// rollback executes Compensation logic for all completed steps in reverse order.
func (e *Engine) rollback(ctx context.Context, task *Task, wf *Workflow, failedAt int, logger *zap.Logger) error {
	task.Status = StatusRollingBack
	e.db.Save(task)

	for i := failedAt; i >= 0; i-- {
		step := wf.Steps[i]
		logger.Info("Compensating step", zap.String("step", step.Name()))
		if err := step.Compensate(ctx, task); err != nil {
			logger.Error("Compensation failed", zap.String("step", step.Name()), zap.Error(err))
		}
	}

	task.Status = StatusRolledBack
	e.db.Save(task)

	if e.audit != nil {
		e.audit.LogEvent(task.Type, task.ResourceUUID, "rollback", "completed", "", "workflow rolled back due to error")
	}

	return fmt.Errorf("workflow failed at step %s: %s", wf.Steps[failedAt].Name(), task.Error)
}
