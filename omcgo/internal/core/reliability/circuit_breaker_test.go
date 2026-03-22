package reliability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	assert.Equal(t, StateClosed, cb.State())
	assert.NoError(t, cb.Allow())
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 3, ResetTimeout: 10 * time.Second}
	cb := NewCircuitBreaker(cfg)

	// First two failures keep circuit closed
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())

	// Third failure opens the circuit
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
	assert.ErrorIs(t, cb.Allow(), ErrCircuitOpen)
}

func TestCircuitBreaker_SuccessResetsCount(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 3, ResetTimeout: 10 * time.Second}
	cb := NewCircuitBreaker(cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	// One success resets the count
	cb.RecordSuccess()

	count, _ := cb.Counters()
	assert.Equal(t, 0, count)
	assert.Equal(t, StateClosed, cb.State())

	// Need 3 more consecutive failures to open
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_TransitionToHalfOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	now := time.Now()
	cb.now = func() time.Time { return now }

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Before timeout: still open
	cb.now = func() time.Time { return now.Add(50 * time.Millisecond) }
	assert.ErrorIs(t, cb.Allow(), ErrCircuitOpen)

	// After timeout: transitions to half-open
	cb.now = func() time.Time { return now.Add(150 * time.Millisecond) }
	require.NoError(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestCircuitBreaker_HalfOpenSuccess(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	now := time.Now()
	cb.now = func() time.Time { return now }

	cb.RecordFailure()
	cb.RecordFailure()

	// Transition to half-open
	cb.now = func() time.Time { return now.Add(150 * time.Millisecond) }
	require.NoError(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())

	// Success closes the circuit
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 100 * time.Millisecond}
	cb := NewCircuitBreaker(cfg)

	now := time.Now()
	cb.now = func() time.Time { return now }

	cb.RecordFailure()
	cb.RecordFailure()

	// Transition to half-open
	cb.now = func() time.Time { return now.Add(150 * time.Millisecond) }
	require.NoError(t, cb.Allow())

	// Failure reopens the circuit (count was already at threshold, +1 still >= threshold)
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cfg := CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 10 * time.Second}
	cb := NewCircuitBreaker(cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
	count, _ := cb.Counters()
	assert.Equal(t, 0, count)
}

func TestCircuitState_String(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", CircuitState(99).String())
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	assert.Equal(t, 5, cfg.FailureThreshold)
	assert.Equal(t, 30*time.Second, cfg.ResetTimeout)
}
