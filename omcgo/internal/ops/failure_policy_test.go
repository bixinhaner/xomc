package ops

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPolicyDecider_Abort(t *testing.T) {
	d := NewPolicyDecider(FailureAbort, 0)
	for _, attempt := range []int{0, 1, 5, 100} {
		action, delay := d.Decide(attempt)
		assert.Equal(t, ActionAbort, action)
		assert.Equal(t, time.Duration(0), delay)
	}
}

func TestPolicyDecider_Continue(t *testing.T) {
	d := NewPolicyDecider(FailureContinue, 0)
	action, delay := d.Decide(0)
	assert.Equal(t, ActionContinue, action)
	assert.Equal(t, time.Duration(0), delay)
}

func TestPolicyDecider_Rollback(t *testing.T) {
	d := NewPolicyDecider(FailureRollback, 0)
	action, delay := d.Decide(0)
	assert.Equal(t, ActionRollback, action)
	assert.Equal(t, time.Duration(0), delay)
}

func TestPolicyDecider_Retry_WithExpBackoff(t *testing.T) {
	d := NewPolicyDecider(FailureRetry, 5)

	tests := []struct {
		attempt   int
		wantAction FailureAction
		wantDelay  time.Duration
	}{
		{0, ActionRetry, 1 * time.Second},     // 2^0 = 1s
		{1, ActionRetry, 2 * time.Second},     // 2^1 = 2s
		{2, ActionRetry, 4 * time.Second},     // 2^2 = 4s
		{3, ActionRetry, 8 * time.Second},     // 2^3 = 8s
		{4, ActionRetry, 16 * time.Second},    // 2^4 = 16s
		{5, ActionAbort, 0},                   // 重试用尽
		{6, ActionAbort, 0},                   // 持续 abort
	}
	for _, tc := range tests {
		action, delay := d.Decide(tc.attempt)
		assert.Equal(t, tc.wantAction, action, "attempt=%d", tc.attempt)
		assert.Equal(t, tc.wantDelay, delay, "attempt=%d delay", tc.attempt)
	}
}

func TestPolicyDecider_Retry_BackoffCappedAt30s(t *testing.T) {
	d := NewPolicyDecider(FailureRetry, 100)
	// 2^7 = 128s 应钳到 30s
	_, delay := d.Decide(7)
	assert.Equal(t, 30*time.Second, delay, "backoff 须钳在 30s")
}

func TestPolicyDecider_Retry_DefaultMaxWhenZero(t *testing.T) {
	d := NewPolicyDecider(FailureRetry, 0)
	// max <1 时默认 3
	for i := 0; i < 3; i++ {
		action, _ := d.Decide(i)
		assert.Equal(t, ActionRetry, action, "attempt=%d 应仍 retry", i)
	}
	action, _ := d.Decide(3)
	assert.Equal(t, ActionAbort, action, "attempt=3 重试用尽")
}

func TestPolicyDecider_UnknownPolicyDefaultsAbort(t *testing.T) {
	d := NewPolicyDecider(FailurePolicy("invalid_xyz"), 0)
	action, _ := d.Decide(0)
	assert.Equal(t, ActionAbort, action, "未知 policy 走最保守 abort")
}

func TestPolicyDecider_Describe(t *testing.T) {
	assert.Equal(t, "retry-max=5", NewPolicyDecider(FailureRetry, 5).Describe())
	assert.Equal(t, "abort", NewPolicyDecider(FailureAbort, 0).Describe())
	assert.Equal(t, "continue", NewPolicyDecider(FailureContinue, 0).Describe())
	assert.Equal(t, "rollback", NewPolicyDecider(FailureRollback, 0).Describe())
}
