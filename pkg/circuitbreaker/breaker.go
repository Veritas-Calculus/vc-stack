// Package circuitbreaker provides a reusable circuit breaker for protecting
// remote service calls (Management → Compute, Gateway → Scheduler, etc.).
//
// It follows the standard three-state model:
//   - Closed: requests flow normally, failures are counted.
//   - Open: all requests are rejected immediately (fail-fast).
//   - Half-Open: a limited number of probe requests are allowed to test recovery.
//
// Usage:
//
//	cb := circuitbreaker.New("compute-node-1", circuitbreaker.Options{
//	    FailureThreshold: 5,
//	    ResetTimeout:     30 * time.Second,
//	})
//	err := cb.Execute(func() error {
//	    return httpClient.Do(req)
//	})
package circuitbreaker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state.
type State int32

const (
	// StateClosed allows requests through normally.
	StateClosed State = iota
	// StateOpen rejects all requests immediately.
	StateOpen
	// StateHalfOpen allows limited probe requests to test recovery.
	StateHalfOpen
)

// String returns the human-readable state name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is in the Open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// ErrTooManyRequests is returned when too many requests are made in Half-Open state.
var ErrTooManyRequests = errors.New("circuit breaker: too many requests in half-open state")

// Options configures the circuit breaker behavior.
type Options struct {
	// FailureThreshold is the number of consecutive failures before opening. Default: 5.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in Half-Open to close. Default: 2.
	SuccessThreshold int
	// ResetTimeout is how long the circuit stays open before transitioning to Half-Open. Default: 30s.
	ResetTimeout time.Duration
	// MaxHalfOpenRequests is the max concurrent requests allowed in Half-Open state. Default: 1.
	MaxHalfOpenRequests int
	// OnStateChange is an optional callback when the state transitions.
	OnStateChange func(name string, from, to State)
	// Logger for structured logging. If nil, a no-op logger is used.
	Logger *zap.Logger
}

func (o *Options) defaults() {
	if o.FailureThreshold <= 0 {
		o.FailureThreshold = 5
	}
	if o.SuccessThreshold <= 0 {
		o.SuccessThreshold = 2
	}
	if o.ResetTimeout <= 0 {
		o.ResetTimeout = 30 * time.Second
	}
	if o.MaxHalfOpenRequests <= 0 {
		o.MaxHalfOpenRequests = 1
	}
	if o.Logger == nil {
		o.Logger = zap.NewNop()
	}
}

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	name string
	opts Options

	mu               sync.Mutex
	state            State
	failures         int
	successes        int
	halfOpenRequests int32 // atomic counter for half-open concurrency control
	lastFailureTime  time.Time
	lastStateChange  time.Time

	// Metrics (atomic for lock-free reads).
	totalRequests  atomic.Int64
	totalSuccesses atomic.Int64
	totalFailures  atomic.Int64
	totalRejected  atomic.Int64
}

// New creates a new circuit breaker with the given name and options.
func New(name string, opts Options) *Breaker {
	opts.defaults()
	return &Breaker{
		name:            name,
		opts:            opts,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function through the circuit breaker.
// If the circuit is open, it returns ErrCircuitOpen without calling fn.
// If the circuit is half-open and max probes are reached, returns ErrTooManyRequests.
func (b *Breaker) Execute(fn func() error) error {
	b.totalRequests.Add(1)

	if err := b.beforeRequest(); err != nil {
		b.totalRejected.Add(1)
		return err
	}

	err := fn()

	b.afterRequest(err)
	return err
}

// State returns the current circuit breaker state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.checkAutoTransition()
	return b.state
}

// Metrics returns a snapshot of the circuit breaker metrics.
func (b *Breaker) Metrics() BreakerMetrics {
	b.mu.Lock()
	state := b.state
	failures := b.failures
	lastFailure := b.lastFailureTime
	lastChange := b.lastStateChange
	b.mu.Unlock()

	return BreakerMetrics{
		Name:            b.name,
		State:           state.String(),
		ConsecFailures:  failures,
		TotalRequests:   b.totalRequests.Load(),
		TotalSuccesses:  b.totalSuccesses.Load(),
		TotalFailures:   b.totalFailures.Load(),
		TotalRejected:   b.totalRejected.Load(),
		LastFailureTime: lastFailure,
		LastStateChange: lastChange,
	}
}

// BreakerMetrics holds observable metrics for the circuit breaker.
type BreakerMetrics struct {
	Name            string    `json:"name"`
	State           string    `json:"state"`
	ConsecFailures  int       `json:"consecutive_failures"`
	TotalRequests   int64     `json:"total_requests"`
	TotalSuccesses  int64     `json:"total_successes"`
	TotalFailures   int64     `json:"total_failures"`
	TotalRejected   int64     `json:"total_rejected"`
	LastFailureTime time.Time `json:"last_failure_time,omitempty"`
	LastStateChange time.Time `json:"last_state_change"`
}

// beforeRequest checks if the request should proceed.
func (b *Breaker) beforeRequest() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.checkAutoTransition()

	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		// Limit concurrent probes.
		current := atomic.LoadInt32(&b.halfOpenRequests)
		if int(current) >= b.opts.MaxHalfOpenRequests {
			return ErrTooManyRequests
		}
		atomic.AddInt32(&b.halfOpenRequests, 1)
		return nil
	}

	return nil
}

// afterRequest processes the result of a request.
func (b *Breaker) afterRequest(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == StateHalfOpen {
		atomic.AddInt32(&b.halfOpenRequests, -1)
	}

	if err != nil {
		b.onFailure()
	} else {
		b.onSuccess()
	}
}

// onFailure handles a failed request.
func (b *Breaker) onFailure() {
	b.totalFailures.Add(1)
	b.failures++
	b.successes = 0
	b.lastFailureTime = time.Now()

	switch b.state {
	case StateClosed:
		if b.failures >= b.opts.FailureThreshold {
			b.transition(StateOpen)
		}
	case StateHalfOpen:
		// Any failure in half-open immediately reopens.
		b.transition(StateOpen)
	}
}

// onSuccess handles a successful request.
func (b *Breaker) onSuccess() {
	b.totalSuccesses.Add(1)

	switch b.state {
	case StateClosed:
		b.failures = 0
		b.successes = 0
	case StateHalfOpen:
		b.successes++
		if b.successes >= b.opts.SuccessThreshold {
			b.transition(StateClosed)
		}
	}
}

// checkAutoTransition checks if a time-based transition should occur.
func (b *Breaker) checkAutoTransition() {
	if b.state == StateOpen && time.Since(b.lastFailureTime) >= b.opts.ResetTimeout {
		b.transition(StateHalfOpen)
	}
}

// transition changes the circuit breaker state.
func (b *Breaker) transition(to State) {
	from := b.state
	if from == to {
		return
	}

	b.state = to
	b.lastStateChange = time.Now()
	b.failures = 0
	b.successes = 0
	atomic.StoreInt32(&b.halfOpenRequests, 0)

	b.opts.Logger.Info("circuit breaker state change",
		zap.String("breaker", b.name),
		zap.String("from", from.String()),
		zap.String("to", to.String()))

	if b.opts.OnStateChange != nil {
		go b.opts.OnStateChange(b.name, from, to)
	}
}

// Reset forces the circuit breaker back to the Closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transition(StateClosed)
}

// Manager manages multiple named circuit breakers (e.g., one per compute node).
type Manager struct {
	mu       sync.RWMutex
	breakers map[string]*Breaker
	opts     Options
}

// NewManager creates a circuit breaker manager with shared default options.
func NewManager(opts Options) *Manager {
	opts.defaults()
	return &Manager{
		breakers: make(map[string]*Breaker),
		opts:     opts,
	}
}

// Get returns the circuit breaker for the given name, creating it if needed.
func (m *Manager) Get(name string) *Breaker {
	m.mu.RLock()
	cb, ok := m.breakers[name]
	m.mu.RUnlock()
	if ok {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock.
	if cb, ok = m.breakers[name]; ok {
		return cb
	}
	cb = New(name, m.opts)
	m.breakers[name] = cb
	return cb
}

// AllMetrics returns metrics for all managed circuit breakers.
func (m *Manager) AllMetrics() []BreakerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make([]BreakerMetrics, 0, len(m.breakers))
	for _, cb := range m.breakers {
		metrics = append(metrics, cb.Metrics())
	}
	return metrics
}

// Execute runs a function through the named circuit breaker.
func (m *Manager) Execute(name string, fn func() error) error {
	return m.Get(name).Execute(fn)
}

// FormatError wraps a circuit breaker error with the target name for context.
func FormatError(target string, err error) error {
	if errors.Is(err, ErrCircuitOpen) {
		return fmt.Errorf("service %s is temporarily unavailable (circuit open): %w", target, err)
	}
	if errors.Is(err, ErrTooManyRequests) {
		return fmt.Errorf("service %s is recovering (half-open, limited probes): %w", target, err)
	}
	return err
}
