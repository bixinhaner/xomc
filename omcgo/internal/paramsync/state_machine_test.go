package paramsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachine_RequestAndRunTransitions(t *testing.T) {
	t.Run("request follows an observable lifecycle", func(t *testing.T) {
		status := RequestStatusAccepted
		for _, next := range []RequestStatus{RequestStatusQueued, RequestStatusRunning, RequestStatusSucceeded} {
			require.NoError(t, ValidateRequestTransition(status, next))
			status = next
		}
		assert.True(t, status.Terminal())
	})

	t.Run("terminal request cannot restart", func(t *testing.T) {
		err := ValidateRequestTransition(RequestStatusSucceeded, RequestStatusRunning)
		require.ErrorIs(t, err, ErrInvalidStateTransition)
		assert.Contains(t, err.Error(), "succeeded -> running")
	})

	t.Run("full run waits for all processing before success", func(t *testing.T) {
		status := RunStatusPlanning
		for _, next := range []RunStatus{
			RunStatusEnqueuing,
			RunStatusWaitingDevice,
			RunStatusExecuting,
			RunStatusProcessing,
			RunStatusSucceeded,
		} {
			require.NoError(t, ValidateRunTransition(status, next))
			status = next
		}
		assert.True(t, status.Terminal())
	})

	t.Run("failure converges through cancelling before terminal", func(t *testing.T) {
		require.NoError(t, ValidateRunTransition(RunStatusExecuting, RunStatusCancelling))
		require.NoError(t, ValidateRunTransition(RunStatusCancelling, RunStatusFailed))
		assert.False(t, RunStatusCancelling.Terminal())
	})
}

func TestFeatureFlags_CanaryIsDeterministicAndBounded(t *testing.T) {
	flags := FeatureFlags{RunEnabled: true, CanaryPercent: 37}
	first := flags.EnabledForDevice("3cc8c90e-e37f-4271-84a2-1e079bb1870c")
	for range 100 {
		assert.Equal(t, first, flags.EnabledForDevice("3cc8c90e-e37f-4271-84a2-1e079bb1870c"))
	}

	flags.CanaryPercent = 0
	assert.False(t, flags.EnabledForDevice("device-a"))
	flags.CanaryPercent = 100
	assert.True(t, flags.EnabledForDevice("device-a"))
	flags.RunEnabled = false
	assert.False(t, flags.EnabledForDevice("device-a"))
}

func TestFeatureFlags_Validate(t *testing.T) {
	assert.NoError(t, (FeatureFlags{}).Validate())
	assert.NoError(t, (FeatureFlags{RunEnabled: true, ResultConsumerEnabled: true, StagingEnabled: true, CanaryPercent: 100}).Validate())
	assert.Error(t, (FeatureFlags{CanaryPercent: -1}).Validate())
	assert.Error(t, (FeatureFlags{CanaryPercent: 101}).Validate())
	assert.ErrorContains(t, (FeatureFlags{RunEnabled: true, StagingEnabled: true, CanaryPercent: 10}).Validate(), "result_consumer_enabled")
	assert.ErrorContains(t, (FeatureFlags{RunEnabled: true, ResultConsumerEnabled: true, CanaryPercent: 10}).Validate(), "staging_enabled")
	assert.ErrorContains(t, (FeatureFlags{RunEnabled: true, ResultConsumerEnabled: true, StagingEnabled: true}).Validate(), "canary_percent")
	assert.ErrorContains(t, (FeatureFlags{ResultConsumerEnabled: true}).Validate(), "run_enabled")
}
