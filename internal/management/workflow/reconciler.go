package workflow

import (
	"context"

	"go.uber.org/zap"
)

// ReconcileTasks finds unfinished tasks and resumes them.
func (e *Engine) ReconcileTasks() {
	e.logger.Info("Starting task reconciliation...")

	var tasks []Task
	// Find tasks that are stuck in 'running' or 'pending' state.
	err := e.db.Where("status IN ?", []TaskStatus{StatusPending, StatusRunning, StatusRollingBack}).
		Find(&tasks).Error

	if err != nil {
		e.logger.Error("Failed to query unfinished tasks", zap.Error(err))
		return
	}

	if len(tasks) == 0 {
		e.logger.Info("No unfinished tasks found.")
		return
	}

	e.logger.Info("Found unfinished tasks", zap.Int("count", len(tasks)))

	for _, t := range tasks {
		go func(task Task) {
			wf, ok := e.registry[task.Type]
			if !ok {
				e.logger.Error("No workflow registered for task type", zap.String("type", task.Type))
				return
			}

			e.logger.Info("Resuming task", zap.Uint("id", task.ID), zap.String("type", task.Type))
			ctx := context.Background()

			if task.Status == StatusRollingBack {
				_ = e.rollback(ctx, &task, wf, task.CurrentStep, e.logger)
			} else {
				_ = e.Run(ctx, &task, wf, e.logger)
			}
		}(t)
	}
}
