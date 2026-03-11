package circuitbreaker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

var errTest = errors.New("test failure")

func TestBreakerClosedPassesThrough(t *testing.T) {
	cb := New("test", Options{FailureThreshold: 3, ResetTimeout: 100 * time.Millisecond})
	var called bool
	err := cb.Execute(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected function to be called")
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected closed state, got %s", cb.State())
	}
}

func TestBreakerOpensAfterThreshold(t *testing.T) {
	cb := New("test", Options{FailureThreshold: 3, ResetTimeout: 100 * time.Millisecond})

	// 3 failures should open the circuit.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return errTest })
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected open state, got %s", cb.State())
	}

	// Next call should be rejected.
	err := cb.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreakerHalfOpenAfterTimeout(t *testing.T) {
	cb := New("test", Options{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		ResetTimeout:     50 * time.Millisecond,
	})

	// Trip the circuit.
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })
	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for reset timeout.
	time.Sleep(60 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", cb.State())
	}
}

func TestBreakerClosesAfterHalfOpenSuccess(t *testing.T) {
	cb := New("test", Options{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		ResetTimeout:     50 * time.Millisecond,
	})

	// Trip → Open.
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Wait → Half-Open.
	time.Sleep(60 * time.Millisecond)

	// 2 successes should close.
	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return nil })

	if cb.State() != StateClosed {
		t.Fatalf("expected closed, got %s", cb.State())
	}
}

func TestBreakerReopensOnHalfOpenFailure(t *testing.T) {
	cb := New("test", Options{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		ResetTimeout:     50 * time.Millisecond,
	})

	// Trip → Open.
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Wait → Half-Open.
	time.Sleep(60 * time.Millisecond)

	// Failure in half-open → back to Open.
	_ = cb.Execute(func() error { return errTest })

	if cb.State() != StateOpen {
		t.Fatalf("expected open after half-open failure, got %s", cb.State())
	}
}

func TestBreakerMetrics(t *testing.T) {
	cb := New("metrics-test", Options{FailureThreshold: 3, ResetTimeout: time.Minute})

	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return nil })

	m := cb.Metrics()
	if m.Name != "metrics-test" {
		t.Fatalf("expected name 'metrics-test', got %q", m.Name)
	}
	if m.TotalRequests != 3 {
		t.Fatalf("expected 3 total requests, got %d", m.TotalRequests)
	}
	if m.TotalSuccesses != 2 {
		t.Fatalf("expected 2 successes, got %d", m.TotalSuccesses)
	}
	if m.TotalFailures != 1 {
		t.Fatalf("expected 1 failure, got %d", m.TotalFailures)
	}
}

func TestBreakerReset(t *testing.T) {
	cb := New("reset-test", Options{FailureThreshold: 1, ResetTimeout: time.Minute})

	_ = cb.Execute(func() error { return errTest })
	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	cb.Reset()
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after reset, got %s", cb.State())
	}
}

func TestManagerGetOrCreate(t *testing.T) {
	mgr := NewManager(Options{FailureThreshold: 3, ResetTimeout: time.Minute})

	cb1 := mgr.Get("node-1")
	cb2 := mgr.Get("node-1")
	cb3 := mgr.Get("node-2")

	if cb1 != cb2 {
		t.Fatal("expected same breaker for same name")
	}
	if cb1 == cb3 {
		t.Fatal("expected different breaker for different name")
	}
}

func TestManagerExecute(t *testing.T) {
	mgr := NewManager(Options{FailureThreshold: 2, ResetTimeout: time.Minute})

	// Trip node-1.
	_ = mgr.Execute("node-1", func() error { return errTest })
	_ = mgr.Execute("node-1", func() error { return errTest })

	// node-1 should be open.
	err := mgr.Execute("node-1", func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected node-1 to be open, got %v", err)
	}

	// node-2 should still work.
	err = mgr.Execute("node-2", func() error { return nil })
	if err != nil {
		t.Fatalf("expected node-2 to work, got %v", err)
	}
}

func TestManagerAllMetrics(t *testing.T) {
	mgr := NewManager(Options{FailureThreshold: 5, ResetTimeout: time.Minute})

	_ = mgr.Execute("a", func() error { return nil })
	_ = mgr.Execute("b", func() error { return nil })
	_ = mgr.Execute("c", func() error { return nil })

	metrics := mgr.AllMetrics()
	if len(metrics) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(metrics))
	}
}

func TestStateStringer(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("State(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

func TestOnStateChangeCallback(t *testing.T) {
	var callbackCalled atomic.Int32

	cb := New("callback-test", Options{
		FailureThreshold: 1,
		ResetTimeout:     time.Minute,
		OnStateChange: func(name string, from, to State) {
			callbackCalled.Add(1)
		},
	})

	_ = cb.Execute(func() error { return errTest })

	// Callback is async, give it time.
	time.Sleep(50 * time.Millisecond)
	if callbackCalled.Load() == 0 {
		t.Fatal("expected OnStateChange to be called")
	}
}

func TestFormatError(t *testing.T) {
	err := FormatError("compute-1", ErrCircuitOpen)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatal("expected wrapped ErrCircuitOpen")
	}

	err2 := FormatError("compute-1", errors.New("something else"))
	if errors.Is(err2, ErrCircuitOpen) {
		t.Fatal("expected non-circuit error to pass through")
	}
}
