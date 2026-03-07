package compute

import (
	"sync"
	"testing"
	"time"
)

func TestMetrics_InstanceCounters(t *testing.T) {
	m := NewMetrics()

	m.IncInstancesCreated()
	m.IncInstancesCreated()
	m.IncInstancesDeleted()
	m.IncInstancesFailed()

	snap := m.Snapshot()
	if snap.InstancesCreated != 2 {
		t.Errorf("InstancesCreated = %d, want 2", snap.InstancesCreated)
	}
	if snap.InstancesDeleted != 1 {
		t.Errorf("InstancesDeleted = %d, want 1", snap.InstancesDeleted)
	}
	if snap.InstancesFailed != 1 {
		t.Errorf("InstancesFailed = %d, want 1", snap.InstancesFailed)
	}
}

func TestMetrics_InstanceGauges(t *testing.T) {
	m := NewMetrics()

	m.SetInstancesRunning(5)
	m.SetInstancesStopped(3)

	snap := m.Snapshot()
	if snap.InstancesRunning != 5 {
		t.Errorf("InstancesRunning = %d, want 5", snap.InstancesRunning)
	}
	if snap.InstancesStopped != 3 {
		t.Errorf("InstancesStopped = %d, want 3", snap.InstancesStopped)
	}
}

func TestMetrics_VolumeCounters(t *testing.T) {
	m := NewMetrics()

	m.IncVolumesCreated()
	m.IncVolumesDeleted()
	m.IncVolumesAttached()
	m.IncVolumesDetached()

	snap := m.Snapshot()
	if snap.VolumesCreated != 1 || snap.VolumesDeleted != 1 ||
		snap.VolumesAttached != 1 || snap.VolumesDetached != 1 {
		t.Error("Volume counters mismatch")
	}
}

func TestMetrics_RecordOperation(t *testing.T) {
	m := NewMetrics()

	m.RecordOperation(true)
	m.RecordOperation(true)
	m.RecordOperation(false)

	snap := m.Snapshot()
	if snap.OperationsTotal != 3 {
		t.Errorf("OperationsTotal = %d, want 3", snap.OperationsTotal)
	}
	if snap.OperationsSuccess != 2 {
		t.Errorf("OperationsSuccess = %d, want 2", snap.OperationsSuccess)
	}
	if snap.OperationsFailed != 1 {
		t.Errorf("OperationsFailed = %d, want 1", snap.OperationsFailed)
	}
}

func TestMetrics_RecordTimings(t *testing.T) {
	m := NewMetrics()

	m.RecordCreateTime(100 * time.Millisecond)
	if m.Snapshot().AvgCreateTime != 100*time.Millisecond {
		t.Error("First record should set value directly")
	}

	m.RecordCreateTime(200 * time.Millisecond)
	avg := m.Snapshot().AvgCreateTime
	if avg != 150*time.Millisecond {
		t.Errorf("AvgCreateTime = %v, want 150ms", avg)
	}

	m.RecordDeleteTime(50 * time.Millisecond)
	m.RecordStartTime(30 * time.Millisecond)
	m.RecordStopTime(20 * time.Millisecond)

	snap := m.Snapshot()
	if snap.AvgDeleteTime != 50*time.Millisecond {
		t.Errorf("AvgDeleteTime = %v", snap.AvgDeleteTime)
	}
	if snap.AvgStartTime != 30*time.Millisecond {
		t.Errorf("AvgStartTime = %v", snap.AvgStartTime)
	}
	if snap.AvgStopTime != 20*time.Millisecond {
		t.Errorf("AvgStopTime = %v", snap.AvgStopTime)
	}
}

func TestMetrics_TaskCounters(t *testing.T) {
	m := NewMetrics()

	m.IncTasksQueued()
	m.IncTasksQueued()
	m.IncTasksProcessing()
	m.IncTasksCompleted()
	m.IncTasksFailed()

	snap := m.Snapshot()
	if snap.TasksQueued != 2 {
		t.Errorf("TasksQueued = %d", snap.TasksQueued)
	}
	if snap.TasksProcessing != 1 {
		t.Errorf("TasksProcessing = %d", snap.TasksProcessing)
	}

	m.DecTasksProcessing()
	m.DecTasksProcessing() // Should not go below 0.
	snap = m.Snapshot()
	if snap.TasksProcessing != 0 {
		t.Errorf("TasksProcessing after double dec = %d", snap.TasksProcessing)
	}
}

func TestMetrics_ToMap(t *testing.T) {
	m := NewMetrics()
	m.IncInstancesCreated()
	m.RecordCreateTime(100 * time.Millisecond)

	result := m.ToMap()
	instances, ok := result["instances"].(map[string]interface{})
	if !ok {
		t.Fatal("instances key missing or wrong type")
	}
	if instances["created"] != int64(1) {
		t.Errorf("instances.created = %v", instances["created"])
	}

	timing, ok := result["timing"].(map[string]interface{})
	if !ok {
		t.Fatal("timing key missing")
	}
	if timing["avg_create_ms"] != int64(100) {
		t.Errorf("avg_create_ms = %v", timing["avg_create_ms"])
	}
}

func TestMetrics_Reset(t *testing.T) {
	m := NewMetrics()
	m.IncInstancesCreated()
	m.IncVolumesCreated()
	m.RecordOperation(true)
	m.IncTasksQueued()
	m.RecordCreateTime(50 * time.Millisecond)

	m.Reset()
	snap := m.Snapshot()

	if snap.InstancesCreated != 0 || snap.VolumesCreated != 0 ||
		snap.OperationsTotal != 0 || snap.TasksQueued != 0 ||
		snap.AvgCreateTime != 0 {
		t.Error("Reset should zero all metrics")
	}
}

func TestMetrics_ConcurrentSafety(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.IncInstancesCreated()
			m.IncInstancesDeleted()
			m.RecordOperation(true)
			m.RecordCreateTime(10 * time.Millisecond)
			_ = m.Snapshot()
			_ = m.ToMap()
		}()
	}

	wg.Wait()
	snap := m.Snapshot()
	if snap.InstancesCreated != 100 {
		t.Errorf("InstancesCreated = %d after 100 concurrent incs", snap.InstancesCreated)
	}
	if snap.OperationsTotal != 100 {
		t.Errorf("OperationsTotal = %d after 100 concurrent ops", snap.OperationsTotal)
	}
}
