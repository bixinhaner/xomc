package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type deliveryRunOnceStub struct {
	count int
	err   error
}

func (s *deliveryRunOnceStub) RunOnce(context.Context) (int, error) { return s.count, s.err }

func TestDeliveryRuntime_RunOnceRunsSchedulerBeforeWorkerAndJoinsErrors(t *testing.T) {
	scheduleErr, workerErr := errors.New("schedule failed"), errors.New("worker failed")
	runtime := NewDeliveryRuntime(
		&deliveryRunOnceStub{count: 2, err: scheduleErr},
		&deliveryRunOnceStub{count: 3, err: workerErr}, nil,
	)

	processed, err := runtime.RunOnce(context.Background())

	require.Equal(t, 5, processed)
	require.ErrorIs(t, err, scheduleErr)
	require.ErrorIs(t, err, workerErr)
}
