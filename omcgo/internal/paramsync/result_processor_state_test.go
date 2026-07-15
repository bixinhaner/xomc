package paramsync

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/task"
)

func TestValidateDurableTaskStateRetriesUntilPostgresTerminalStateIsVisible(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  task.TaskStatus
		success bool
	}{
		{name: "success event while task pending", status: task.TaskStatusPending, success: true},
		{name: "failure event while task sent", status: task.TaskStatusSent, success: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDurableTaskState(tc.status, tc.success)
			assert.ErrorIs(t, err, ErrTaskResultNotDurable)
		})
	}
}

func TestValidateDurableTaskStateAcceptsMatchingTerminalState(t *testing.T) {
	assert.NoError(t, validateDurableTaskState(task.TaskStatusCompleted, true))
	for _, status := range []task.TaskStatus{task.TaskStatusFailed, task.TaskStatusExpired, task.TaskStatusCancelled} {
		err := validateDurableTaskState(status, false)
		assert.NoError(t, err, status)
	}
}

func TestValidateDurableTaskStateRejectsContradictoryTerminalState(t *testing.T) {
	err := validateDurableTaskState(task.TaskStatusFailed, true)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrTaskResultNotDurable))
}
