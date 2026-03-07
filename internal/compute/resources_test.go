package compute

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// --- ResourceTracker Tests ---

func TestResourceTracker_TrackAndGet(t *testing.T) {
	rt := NewResourceTracker(zap.NewNop())

	now := time.Now()
	alloc := &ResourceAllocation{
		ResourceID:   "inst-1",
		ResourceType: "instance",
		ProjectID:    10,
		UserID:       5,
		AllocatedAt:  now,
		Metadata:     map[string]string{"flavor": "m1.small"},
	}
	rt.Track(alloc)

	got, ok := rt.Get("inst-1")
	if !ok {
		t.Fatal("Should find tracked resource")
	}
	if got.ResourceType != "instance" {
		t.Errorf("ResourceType = %q", got.ResourceType)
	}
	if got.AllocatedAt != now {
		t.Error("AllocatedAt should be preserved")
	}

	// Not found.
	_, ok = rt.Get("nonexistent")
	if ok {
		t.Error("Should not find nonexistent resource")
	}
}

func TestResourceTracker_Release(t *testing.T) {
	rt := NewResourceTracker(zap.NewNop())
	rt.Track(&ResourceAllocation{ResourceID: "r-1"})
	rt.Track(&ResourceAllocation{ResourceID: "r-2"})

	if rt.Count() != 2 {
		t.Errorf("Count before release = %d, want 2", rt.Count())
	}

	rt.Release("r-1")
	if rt.Count() != 1 {
		t.Error("Count after release should be 1")
	}

	_, ok := rt.Get("r-1")
	if ok {
		t.Error("Released resource should not be found")
	}

	// Releasing nonexistent should not panic.
	rt.Release("nonexistent")
}

func TestResourceTracker_ListByProject(t *testing.T) {
	rt := NewResourceTracker(zap.NewNop())
	rt.Track(&ResourceAllocation{ResourceID: "a-1", ProjectID: 10})
	rt.Track(&ResourceAllocation{ResourceID: "a-2", ProjectID: 10})
	rt.Track(&ResourceAllocation{ResourceID: "a-3", ProjectID: 20})

	proj10 := rt.ListByProject(10)
	if len(proj10) != 2 {
		t.Errorf("ListByProject(10) = %d, want 2", len(proj10))
	}

	proj20 := rt.ListByProject(20)
	if len(proj20) != 1 {
		t.Errorf("ListByProject(20) = %d, want 1", len(proj20))
	}

	proj99 := rt.ListByProject(99)
	if len(proj99) != 0 {
		t.Errorf("ListByProject(99) = %d, want 0", len(proj99))
	}
}

func TestResourceTracker_ConcurrentSafety(t *testing.T) {
	rt := NewResourceTracker(zap.NewNop())
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rid := "r-" + string(rune('a'+id%26))
			rt.Track(&ResourceAllocation{ResourceID: rid, ProjectID: uint(id % 5)})
			rt.Get(rid)
			rt.ListByProject(uint(id % 5))
			rt.Count()
		}(i)
	}
	wg.Wait()

	if rt.Count() == 0 {
		t.Error("Should have some tracked resources")
	}
}

// --- OperationContext Tests ---

func TestOperationContext_Basic(t *testing.T) {
	ctx := context.Background()
	opCtx := NewOperationContext(ctx, "create_instance", "inst-1", zap.NewNop())

	if opCtx.OpType != "create_instance" {
		t.Errorf("OpType = %q", opCtx.OpType)
	}
	if opCtx.ResourceID != "inst-1" {
		t.Errorf("ResourceID = %q", opCtx.ResourceID)
	}
	if opCtx.OpID == "" {
		t.Error("OpID should be generated")
	}
	if opCtx.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if opCtx.Context() != ctx {
		t.Error("Context() should return underlying context")
	}
}

func TestOperationContext_Duration(t *testing.T) {
	opCtx := NewOperationContext(context.Background(), "test", "r-1", zap.NewNop())
	time.Sleep(5 * time.Millisecond)
	d := opCtx.Duration()
	if d < 5*time.Millisecond {
		t.Errorf("Duration = %v, too short", d)
	}
}

func TestOperationContext_LogSuccessAndError(t *testing.T) {
	opCtx := NewOperationContext(context.Background(), "test", "r-1", zap.NewNop())
	// Should not panic.
	opCtx.LogSuccess()
	opCtx.LogError(NewInternalError(nil))
}

// --- TaskQueue Tests ---

type testTask struct {
	id string
	fn func(ctx context.Context) error
}

func (t *testTask) Execute(ctx context.Context) error { return t.fn(ctx) }
func (t *testTask) GetID() string                     { return t.id }
func (t *testTask) GetType() string                   { return "test" }

func TestTaskQueue_SubmitAndProcess(t *testing.T) {
	tq := NewTaskQueue(2, zap.NewNop())

	executed := make(chan string, 5)

	for i := 0; i < 3; i++ {
		id := "task-" + string(rune('0'+i))
		task := &testTask{
			id: id,
			fn: func(ctx context.Context) error {
				executed <- id
				return nil
			},
		}
		if err := tq.Submit(task); err != nil {
			t.Fatalf("Submit(%s) error: %v", id, err)
		}
	}

	// Wait for tasks to complete.
	time.Sleep(200 * time.Millisecond)
	if err := tq.Stop(context.Background()); err != nil {
		t.Fatalf("Stop error: %v", err)
	}

	close(executed)
	count := 0
	for range executed {
		count++
	}
	if count != 3 {
		t.Errorf("Executed %d tasks, want 3", count)
	}
}

func TestTaskQueue_StopGracefully(t *testing.T) {
	tq := NewTaskQueue(1, zap.NewNop())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := tq.Stop(ctx)
	if err != nil {
		t.Errorf("Stop with no pending tasks should not error: %v", err)
	}
}

func TestTaskQueue_SubmitAfterStop(t *testing.T) {
	tq := NewTaskQueue(1, zap.NewNop())
	_ = tq.Stop(context.Background())

	// After stop, submitting tasks should eventually fail.
	// The stop channel is closed so the select may pick either the channel
	// write (if buffer available) or stopCh. Try multiple times to ensure
	// at least one returns an error.
	var errCount int
	for i := 0; i < 10; i++ {
		err := tq.Submit(&testTask{id: "late", fn: func(ctx context.Context) error { return nil }})
		if err != nil {
			errCount++
		}
	}
	if errCount == 0 {
		t.Error("Submit after Stop should return error at least once in 10 attempts")
	}
}
