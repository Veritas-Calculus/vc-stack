// Package compute provides metrics collection.
package compute

import (
	"sync"
	"time"
)

// Metrics tracks service metrics.
type Metrics struct {
	mu sync.RWMutex

	// Instance metrics.
	InstancesCreated int64
	InstancesDeleted int64
	InstancesFailed  int64
	InstancesRunning int64
	InstancesStopped int64

	// Volume metrics.
	VolumesCreated  int64
	VolumesDeleted  int64
	VolumesAttached int64
	VolumesDetached int64

	// Operation metrics.
	OperationsTotal   int64
	OperationsSuccess int64
	OperationsFailed  int64

	// Task metrics.
	TasksQueued     int64
	TasksProcessing int64
	TasksCompleted  int64
	TasksFailed     int64

	// Timing metrics.
	AvgCreateTime time.Duration
	AvgDeleteTime time.Duration
	AvgStartTime  time.Duration
	AvgStopTime   time.Duration
}

// NewMetrics creates a new metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncInstancesCreated increments instances created counter.
func (m *Metrics) IncInstancesCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InstancesCreated++
}

// IncInstancesDeleted increments instances deleted counter.
func (m *Metrics) IncInstancesDeleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InstancesDeleted++
}

// IncInstancesFailed increments instances failed counter.
func (m *Metrics) IncInstancesFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InstancesFailed++
}

// SetInstancesRunning sets running instances gauge.
func (m *Metrics) SetInstancesRunning(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InstancesRunning = count
}

// SetInstancesStopped sets stopped instances gauge.
func (m *Metrics) SetInstancesStopped(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InstancesStopped = count
}

// IncVolumesCreated increments volumes created counter.
func (m *Metrics) IncVolumesCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.VolumesCreated++
}

// IncVolumesDeleted increments volumes deleted counter.
func (m *Metrics) IncVolumesDeleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.VolumesDeleted++
}

// IncVolumesAttached increments volumes attached counter.
func (m *Metrics) IncVolumesAttached() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.VolumesAttached++
}

// IncVolumesDetached increments volumes detached counter.
func (m *Metrics) IncVolumesDetached() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.VolumesDetached++
}

// RecordOperation records an operation result.
func (m *Metrics) RecordOperation(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OperationsTotal++
	if success {
		m.OperationsSuccess++
	} else {
		m.OperationsFailed++
	}
}

// RecordCreateTime records instance creation time.
func (m *Metrics) RecordCreateTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple moving average (can be improved with more sophisticated algorithms).
	if m.AvgCreateTime == 0 {
		m.AvgCreateTime = duration
	} else {
		m.AvgCreateTime = (m.AvgCreateTime + duration) / 2
	}
}

// RecordDeleteTime records instance deletion time.
func (m *Metrics) RecordDeleteTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AvgDeleteTime == 0 {
		m.AvgDeleteTime = duration
	} else {
		m.AvgDeleteTime = (m.AvgDeleteTime + duration) / 2
	}
}

// RecordStartTime records instance start time.
func (m *Metrics) RecordStartTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AvgStartTime == 0 {
		m.AvgStartTime = duration
	} else {
		m.AvgStartTime = (m.AvgStartTime + duration) / 2
	}
}

// RecordStopTime records instance stop time.
func (m *Metrics) RecordStopTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AvgStopTime == 0 {
		m.AvgStopTime = duration
	} else {
		m.AvgStopTime = (m.AvgStopTime + duration) / 2
	}
}

// IncTasksQueued increments tasks queued counter.
func (m *Metrics) IncTasksQueued() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksQueued++
}

// IncTasksProcessing increments tasks processing counter.
func (m *Metrics) IncTasksProcessing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksProcessing++
}

// DecTasksProcessing decrements tasks processing counter.
func (m *Metrics) DecTasksProcessing() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.TasksProcessing > 0 {
		m.TasksProcessing--
	}
}

// IncTasksCompleted increments tasks completed counter.
func (m *Metrics) IncTasksCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksCompleted++
}

// IncTasksFailed increments tasks failed counter.
func (m *Metrics) IncTasksFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksFailed++
}

// Snapshot returns a copy of current metrics.
func (m *Metrics) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Metrics{
		InstancesCreated:  m.InstancesCreated,
		InstancesDeleted:  m.InstancesDeleted,
		InstancesFailed:   m.InstancesFailed,
		InstancesRunning:  m.InstancesRunning,
		InstancesStopped:  m.InstancesStopped,
		VolumesCreated:    m.VolumesCreated,
		VolumesDeleted:    m.VolumesDeleted,
		VolumesAttached:   m.VolumesAttached,
		VolumesDetached:   m.VolumesDetached,
		OperationsTotal:   m.OperationsTotal,
		OperationsSuccess: m.OperationsSuccess,
		OperationsFailed:  m.OperationsFailed,
		TasksQueued:       m.TasksQueued,
		TasksProcessing:   m.TasksProcessing,
		TasksCompleted:    m.TasksCompleted,
		TasksFailed:       m.TasksFailed,
		AvgCreateTime:     m.AvgCreateTime,
		AvgDeleteTime:     m.AvgDeleteTime,
		AvgStartTime:      m.AvgStartTime,
		AvgStopTime:       m.AvgStopTime,
	}
}

// ToMap converts metrics to a map for JSON serialization.
func (m *Metrics) ToMap() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"instances": map[string]interface{}{
			"created": m.InstancesCreated,
			"deleted": m.InstancesDeleted,
			"failed":  m.InstancesFailed,
			"running": m.InstancesRunning,
			"stopped": m.InstancesStopped,
		},
		"volumes": map[string]interface{}{
			"created":  m.VolumesCreated,
			"deleted":  m.VolumesDeleted,
			"attached": m.VolumesAttached,
			"detached": m.VolumesDetached,
		},
		"operations": map[string]interface{}{
			"total":   m.OperationsTotal,
			"success": m.OperationsSuccess,
			"failed":  m.OperationsFailed,
		},
		"tasks": map[string]interface{}{
			"queued":     m.TasksQueued,
			"processing": m.TasksProcessing,
			"completed":  m.TasksCompleted,
			"failed":     m.TasksFailed,
		},
		"timing": map[string]interface{}{
			"avg_create_ms": m.AvgCreateTime.Milliseconds(),
			"avg_delete_ms": m.AvgDeleteTime.Milliseconds(),
			"avg_start_ms":  m.AvgStartTime.Milliseconds(),
			"avg_stop_ms":   m.AvgStopTime.Milliseconds(),
		},
	}
}

// Reset resets all metrics to zero.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InstancesCreated = 0
	m.InstancesDeleted = 0
	m.InstancesFailed = 0
	m.InstancesRunning = 0
	m.InstancesStopped = 0
	m.VolumesCreated = 0
	m.VolumesDeleted = 0
	m.VolumesAttached = 0
	m.VolumesDetached = 0
	m.OperationsTotal = 0
	m.OperationsSuccess = 0
	m.OperationsFailed = 0
	m.TasksQueued = 0
	m.TasksProcessing = 0
	m.TasksCompleted = 0
	m.TasksFailed = 0
	m.AvgCreateTime = 0
	m.AvgDeleteTime = 0
	m.AvgStartTime = 0
	m.AvgStopTime = 0
}
