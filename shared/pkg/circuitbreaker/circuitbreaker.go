package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrTooManyRequests is returned when too many requests are attempted in half-open state
	ErrTooManyRequests = errors.New("too many requests in half-open state")

	// ErrTimeout is returned when a request times out
	ErrTimeout = errors.New("request timeout")
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed State = iota
	// StateOpen means the circuit is open and requests are blocked
	StateOpen
	// StateHalfOpen means the circuit is testing if the service has recovered
	StateHalfOpen
)

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

// Config holds circuit breaker configuration
type Config struct {
	// MaxRequests is the maximum number of requests allowed in half-open state
	MaxRequests uint32

	// Interval is the cyclic period in closed state to clear the internal counts
	// If 0, the circuit breaker doesn't clear counts
	Interval time.Duration

	// Timeout is the period after which the circuit moves from open to half-open
	Timeout time.Duration

	// ReadyToTrip is called with the counts when the circuit is about to trip
	// If ReadyToTrip returns true, the circuit trips from closed to open
	ReadyToTrip func(counts Counts) bool

	// OnStateChange is called whenever the state changes
	OnStateChange func(name string, from State, to State)

	// IsSuccessful determines if the result is considered a success
	// If nil, non-nil error is considered a failure
	IsSuccessful func(err error) bool
}

// Counts holds the statistics of the circuit breaker
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker is a state machine to prevent cascading failures
type CircuitBreaker struct {
	name       string
	config     Config
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
	mutex      sync.RWMutex
}

// New creates a new CircuitBreaker with the given configuration
func New(name string, config Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:   name,
		config: config,
		state:  StateClosed,
		expiry: time.Now().Add(config.Interval),
	}

	// Set default values
	if cb.config.MaxRequests == 0 {
		cb.config.MaxRequests = 1
	}

	if cb.config.ReadyToTrip == nil {
		cb.config.ReadyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	}

	if cb.config.IsSuccessful == nil {
		cb.config.IsSuccessful = func(err error) bool {
			return err == nil
		}
	}

	return cb
}

// Execute runs the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(fn func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(generation, false)
			panic(r)
		}
	}()

	err = fn()
	cb.afterRequest(generation, cb.config.IsSuccessful(err))
	return err
}

// ExecuteWithContext runs the given function with context if the circuit breaker allows it
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(generation, false)
			panic(r)
		}
	}()

	// Create a channel to receive the result
	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic recovered: %v", r)
			}
		}()
		done <- fn(ctx)
	}()

	select {
	case <-ctx.Done():
		cb.afterRequest(generation, false)
		return ErrTimeout
	case err := <-done:
		cb.afterRequest(generation, cb.config.IsSuccessful(err))
		return err
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Counts returns the current counts of the circuit breaker
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.toNewGeneration(time.Now())
	cb.setState(StateClosed)
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	switch state {
	case StateOpen:
		return 0, ErrCircuitOpen
	case StateHalfOpen:
		if cb.counts.Requests >= cb.config.MaxRequests {
			return 0, ErrTooManyRequests
		}
	}

	cb.counts.Requests++
	return cb.generation, nil
}

func (cb *CircuitBreaker) afterRequest(generation uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	// Ignore stale requests
	if generation != cb.generation {
		return
	}

	if success {
		cb.onSuccess(now)
	} else {
		cb.onFailure(now)
	}
}

func (cb *CircuitBreaker) onSuccess(now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	switch cb.state {
	case StateClosed:
		if cb.config.Interval > 0 && now.After(cb.expiry) {
			cb.toNewGeneration(now)
		}
	case StateHalfOpen:
		if cb.counts.ConsecutiveSuccesses >= cb.config.MaxRequests {
			cb.setState(StateClosed)
			cb.toNewGeneration(now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	switch cb.state {
	case StateClosed:
		if cb.config.ReadyToTrip(cb.counts) {
			cb.setState(StateOpen)
			cb.toNewGeneration(now)
		} else if cb.config.Interval > 0 && now.After(cb.expiry) {
			cb.toNewGeneration(now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
		cb.toNewGeneration(now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) State {
	switch cb.state {
	case StateClosed:
		if cb.config.Interval > 0 && now.After(cb.expiry) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if now.After(cb.expiry) {
			cb.setState(StateHalfOpen)
			cb.toNewGeneration(now)
		}
	}
	return cb.state
}

func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var expiry time.Time
	switch cb.state {
	case StateClosed:
		if cb.config.Interval > 0 {
			expiry = now.Add(cb.config.Interval)
		}
	case StateOpen:
		expiry = now.Add(cb.config.Timeout)
	default:
		expiry = time.Time{}
	}

	cb.expiry = expiry
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers sync.Map
	config   Config
}

// NewManager creates a new circuit breaker manager
func NewManager(config Config) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		config: config,
	}
}

// Get returns a circuit breaker for the given name, creating one if it doesn't exist
func (m *CircuitBreakerManager) Get(name string) *CircuitBreaker {
	if cb, ok := m.breakers.Load(name); ok {
		return cb.(*CircuitBreaker)
	}

	cb := New(name, m.config)
	actual, _ := m.breakers.LoadOrStore(name, cb)
	return actual.(*CircuitBreaker)
}

// Remove removes a circuit breaker
func (m *CircuitBreakerManager) Remove(name string) {
	m.breakers.Delete(name)
}

// Reset resets all circuit breakers
func (m *CircuitBreakerManager) Reset() {
	m.breakers.Range(func(key, value any) bool {
		if cb, ok := value.(*CircuitBreaker); ok {
			cb.Reset()
		}
		return true
	})
}

// GetAll returns all circuit breakers
func (m *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	result := make(map[string]*CircuitBreaker)
	m.breakers.Range(func(key, value any) bool {
		if name, ok := key.(string); ok {
			if cb, ok := value.(*CircuitBreaker); ok {
				result[name] = cb
			}
		}
		return true
	})
	return result
}
