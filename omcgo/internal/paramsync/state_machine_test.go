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
