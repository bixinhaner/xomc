package paramsync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryDeadlinePreservesOriginalTimeoutWindow(t *testing.T) {
	createdAt := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	originalDeadline := createdAt.Add(12 * time.Minute)
	retryAt := createdAt.Add(time.Hour)

	deadline := retryDeadline(&SyncRequest{CreatedAt: createdAt, DeadlineAt: &originalDeadline}, retryAt)

	require.NotNil(t, deadline)
	assert.Equal(t, retryAt.Add(12*time.Minute), *deadline)
}

func TestRetryDeadlineLeavesRequestsWithoutTimeoutUnbounded(t *testing.T) {
	assert.Nil(t, retryDeadline(&SyncRequest{}, time.Now()))
}

func TestRequestCanOnlyBeCancelledWhileActive(t *testing.T) {
	for _, status := range []RequestStatus{RequestStatusAccepted, RequestStatusQueued, RequestStatusRunning} {
		assert.True(t, requestCanBeCancelled(status), string(status))
	}
	for _, status := range []RequestStatus{
		RequestStatusSucceeded, RequestStatusFailed, RequestStatusTimedOut,
		RequestStatusCancelled, RequestStatusDeduplicated, RequestStatusRejected,
	} {
		assert.False(t, requestCanBeCancelled(status), string(status))
	}
}
