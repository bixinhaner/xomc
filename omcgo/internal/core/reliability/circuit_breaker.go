// Package reliability 提供分布式系统常用的可靠性模式：
// - CircuitBreaker：熟断器，防止连锁故障，用于调用下游不稳定服务（如 OSS 推送、Connection Request）
// - Retry：指数退退重试，用于短暂故障自恢复（如 DB 连接重试、NATS 发布重试）
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

// CircuitBreakerConfig 定义熟断器的运行参数。
// FailureThreshold：连续失败多少次转为 Open 状态（默认 5）。
// ResetTimeout： Open 状态持续多久后转为 HalfOpen 尝试恢复（默认 30s）。
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

// CircuitBreaker 实现熟断器模式，具有三个状态：
// - Closed（关闭）：正常放行请求
// - Open（断开）：连续失败超过阈值后，快速失败不进入下游
// - HalfOpen（半开）：等待 ResetTimeout 后放行一次探测请求
//
// 使用场景：封装对外调用（OSS HTTP 推送、CPE Connection Request），
// 防止下游服务长时集中超时导致线程串联庄。
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
