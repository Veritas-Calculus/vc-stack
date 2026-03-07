package compute

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// processDeletionQueue continuously processes pending deletion tasks with retry support.
func (s *Service) processDeletionQueue() {
	// Skip if database is not available.
	if s.db == nil {
		s.logger.Warn("Deletion queue processor disabled (database not available)")
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	s.logger.Info("Deletion queue processor started")

	for range ticker.C {
		// Find pending or failed tasks ready for retry.
		var tasks []DeletionTask
		err := s.db.Where("status IN (?, ?) AND retry_count < max_retries", "pending", "failed").
			Order("created_at ASC").
			Limit(10).
			Find(&tasks).Error

		if err != nil {
			s.logger.Error("Failed to fetch deletion tasks", zap.Error(err))
			continue
		}

		for _, task := range tasks {
			s.processDeletionTask(&task)
		}
	}
}

// processDeletionTask processes a single deletion task with retry logic.
func (s *Service) processDeletionTask(task *DeletionTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Update status to processing.
	now := time.Now()
	updates := map[string]interface{}{
		"status":     "processing",
		"started_at": &now,
	}
	s.db.Model(task).Updates(updates)

	s.logger.Info("Processing deletion task",
		zap.Uint("task_id", task.ID),
		zap.String("instance_uuid", task.InstanceUUID),
		zap.String("vmid", task.VMID),
		zap.Int("retry", task.RetryCount))

	// Step 1: Delete VM from hypervisor.
	var deleteErr error
	if task.LiteAddr != "" && task.VMID != "" {
		deleteErr = s.nodeDeleteVM(ctx, task.LiteAddr, task.VMID)
		if deleteErr != nil {
			s.logger.Error("Lite delete failed",
				zap.String("vmid", task.VMID),
				zap.String("lite", task.LiteAddr),
				zap.Error(deleteErr))
		} else {
			s.logger.Info("Lite delete succeeded", zap.String("vmid", task.VMID))
		}
	} else {
		s.logger.Warn("Skipping lite delete: missing address or vmid",
			zap.String("lite", task.LiteAddr),
			zap.String("vmid", task.VMID))
	}

	// Step 2: Verify deletion (check if VM still exists)
	verified := false
	if deleteErr == nil && task.LiteAddr != "" && task.VMID != "" {
		verified = s.verifyVMDeletion(ctx, task.LiteAddr, task.VMID)
		if !verified {
			deleteErr = fmt.Errorf("VM still exists after deletion attempt")
			s.logger.Warn("Deletion verification failed", zap.String("vmid", task.VMID))
		}
	}

	// Step 3: Handle result.
	if deleteErr != nil {
		// Deletion failed, increment retry count.
		task.RetryCount++
		task.LastError = deleteErr.Error()

		if task.RetryCount >= task.MaxRetries {
			// Max retries reached, mark as failed.
			completedAt := time.Now()
			s.db.Model(task).Updates(map[string]interface{}{
				"status":       "failed",
				"retry_count":  task.RetryCount,
				"last_error":   task.LastError,
				"completed_at": &completedAt,
			})

			s.logger.Error("Deletion task failed after max retries",
				zap.Uint("task_id", task.ID),
				zap.String("instance_uuid", task.InstanceUUID),
				zap.Int("retries", task.RetryCount),
				zap.String("error", task.LastError))

			// Update instance status to error.
			s.db.Model(&Instance{}).
				Where("uuid = ?", task.InstanceUUID).
				Updates(map[string]interface{}{
					"status":     "error",
					"task_state": "deletion_failed",
				})
		} else {
			// Schedule for retry.
			s.db.Model(task).Updates(map[string]interface{}{
				"status":      "failed",
				"retry_count": task.RetryCount,
				"last_error":  task.LastError,
			})

			s.logger.Warn("Deletion task will retry",
				zap.Uint("task_id", task.ID),
				zap.Int("retry", task.RetryCount),
				zap.Int("max_retries", task.MaxRetries))
		}
	} else {
		// Deletion successful.
		completedAt := time.Now()
		s.db.Model(task).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": &completedAt,
			"last_error":   "",
		})

		// Update instance status to deleted.
		s.db.Model(&Instance{}).
			Where("uuid = ?", task.InstanceUUID).
			Updates(map[string]interface{}{
				"status":        "deleted",
				"power_state":   "shutdown",
				"terminated_at": &completedAt,
			})

		s.logger.Info("Deletion task completed successfully",
			zap.Uint("task_id", task.ID),
			zap.String("instance_uuid", task.InstanceUUID))
	}
}

// verifyVMDeletion checks if a VM still exists on the hypervisor.
// Returns true if VM is confirmed deleted, false if it still exists.
func (s *Service) verifyVMDeletion(ctx context.Context, nodeAddr, vmID string) bool {
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID
	req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)

	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		// Network error, can't verify - assume not deleted.
		s.logger.Warn("Verification failed due to network error", zap.Error(err))
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	// 404 means VM doesn't exist (deleted successfully)
	// 200 means VM still exists (not deleted)
	// Other codes are ambiguous.
	if resp.StatusCode == http.StatusNotFound {
		return true
	}

	return false
}

// GetDeletionTask retrieves a deletion task by instance UUID.
func (s *Service) GetDeletionTask(ctx context.Context, instanceUUID string) (*DeletionTask, error) {
	var task DeletionTask
	err := s.db.Where("instance_uuid = ?", instanceUUID).
		Order("created_at DESC").
		First(&task).Error

	if err != nil {
		return nil, err
	}
	return &task, nil
}
