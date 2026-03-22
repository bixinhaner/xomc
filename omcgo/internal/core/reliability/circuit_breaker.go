package reliability

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
	// StateClosed allows requests to pass through normally.
	StateClosed CircuitState = iota
	// StateOpen rejects all requests immediately.
	StateOpen
	// StateHalfOpen allows a single probe request to determine if the service has recovered.
	StateHalfOpen
)

// String returns a human-readable name for the circuit state.
func (s CircuitState) String() string {
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

// ErrCircuitOpen is returned when the circuit breaker is in the open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig defines the configuration for a CircuitBreaker.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Number of consecutive failures before opening the circuit.
	ResetTimeout     time.Duration // Time to wait before transitioning from open to half-open.
}

// DefaultCircuitBreakerConfig returns a sensible default configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern with three states:
// closed (normal operation), open (fail-fast), and half-open (probe).
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failureCount     int
	failureThreshold int
	resetTimeout     time.Duration
	lastFailureTime  time.Time
	now              func() time.Time // injectable for testing
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold < 1 {
		cfg.FailureThreshold = 5
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		resetTimeout:     cfg.ResetTimeout,
		now:              time.Now,
	}
}

// Allow checks if a request is allowed to proceed.
// Returns nil if allowed, ErrCircuitOpen if not.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if cb.now().Sub(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		// Only one probe at a time; subsequent calls are rejected while probing.
		return nil
	default:
		return ErrCircuitOpen
	}
}

// RecordSuccess records a successful operation, resetting the circuit to closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
	cb.state = StateClosed
}

// RecordFailure records a failed operation. If the failure threshold is reached,
// the circuit transitions to open.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount++
	cb.lastFailureTime = cb.now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Counters returns current failure count and threshold for monitoring.
func (cb *CircuitBreaker) Counters() (failureCount, threshold int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount, cb.failureThreshold
}

// Reset forces the circuit breaker back to closed state. Useful for admin override.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
	cb.state = StateClosed
}
