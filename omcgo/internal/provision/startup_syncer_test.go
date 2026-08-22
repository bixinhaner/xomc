package provision

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type startupPageLister struct {
	pages [][]*model.Device
	calls []uuid.UUID
	errs  []error
	next  int
}

func (l *startupPageLister) ListOnlineDevices(_ context.Context, afterID uuid.UUID, _ int) ([]*model.Device, error) {
	l.calls = append(l.calls, afterID)
	if len(l.errs) > 0 {
		err := l.errs[0]
		l.errs = l.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	if l.next >= len(l.pages) {
		return nil, nil
	}
	page := l.pages[l.next]
	l.next++
	return page, nil
}

type startupSubmitter struct {
	calls   []string
	keys    []string
	errs    []error
	results []*DeviceOnlineFullSyncResult
}

func (s *startupSubmitter) SubmitStartupDeviceOnlineFullSync(_ context.Context, dev *model.Device, key, _ string) (*DeviceOnlineFullSyncResult, error) {
	s.calls = append(s.calls, dev.SerialNumber)
	s.keys = append(s.keys, key)
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	if len(s.results) > 0 {
		result := s.results[0]
		s.results = s.results[1:]
		return result, nil
	}
	return &DeviceOnlineFullSyncResult{RequestID: uuid.New(), Status: "accepted"}, nil
}

func startupDevices(n int) []*model.Device {
	result := make([]*model.Device, n)
	for i := range result {
		result[i] = &model.Device{ID: uuid.New(), SerialNumber: uuid.NewString()}
	}
	return result
}

func TestStartupSyncerPagesThroughAllOnlineDevices(t *testing.T) {
	lister := &startupPageLister{pages: [][]*model.Device{startupDevices(2), startupDevices(2), startupDevices(1)}}
	submitter := &startupSubmitter{}
	leader := &fakeLeader{acquired: true}
	s := NewStartupSyncer(lister, submitter, leader, 2, zap.NewNop())
	require.NoError(t, s.Run(context.Background()))
	require.Len(t, lister.calls, 3)
	require.Equal(t, uuid.Nil, lister.calls[0])
	require.Len(t, submitter.calls, 5)
	require.True(t, leader.released.Load())
	for _, key := range submitter.keys {
		require.True(t, strings.HasPrefix(key, "omc-redeploy:"))
	}
}

func TestStartupSyncerStopsAtSubmissionBudget(t *testing.T) {
	lister := &startupPageLister{pages: [][]*model.Device{startupDevices(2), startupDevices(2), startupDevices(1)}}
	submitter := &startupSubmitter{}
	s := NewStartupSyncer(lister, submitter, nil, 2, zap.NewNop()).
		WithSubmissionBudget(3, 0)

	require.NoError(t, s.Run(context.Background()))
	require.Len(t, lister.calls, 2)
	require.Len(t, submitter.calls, 3)
}

func TestStartupSyncerCountsPendingAgainstSubmissionBudget(t *testing.T) {
	lister := &startupPageLister{pages: [][]*model.Device{startupDevices(2), startupDevices(2), startupDevices(1)}}
	submitter := &startupSubmitter{results: []*DeviceOnlineFullSyncResult{nil, nil, nil, nil, nil, nil}}
	s := NewStartupSyncer(lister, submitter, nil, 2, zap.NewNop()).
		WithSubmissionBudget(3, 0)
	s.retryWait = func(context.Context, time.Duration) error { return nil }

	require.NoError(t, s.Run(context.Background()))
	require.Len(t, lister.calls, 2)
	require.Len(t, submitter.calls, 3)
}

func TestStartupSyncerLimitsPendingRetryRounds(t *testing.T) {
	lister := &startupPageLister{pages: [][]*model.Device{startupDevices(1)}}
	submitter := &startupSubmitter{results: []*DeviceOnlineFullSyncResult{nil, nil, nil}}
	s := NewStartupSyncer(lister, submitter, nil, 2, zap.NewNop()).
		WithSubmissionBudget(10, 0)
	s.retryWait = func(context.Context, time.Duration) error { return nil }

	require.NoError(t, s.Run(context.Background()))
	require.Len(t, submitter.calls, 2)
}

func TestStartupSyncerWaitsBetweenBudgetedSubmissions(t *testing.T) {
	lister := &startupPageLister{pages: [][]*model.Device{startupDevices(2)}}
	submitter := &startupSubmitter{}
	s := NewStartupSyncer(lister, submitter, nil, 2, zap.NewNop()).
		WithSubmissionBudget(2, time.Second)
	var waits []time.Duration
	s.retryWait = func(_ context.Context, delay time.Duration) error {
		waits = append(waits, delay)
		return nil
	}

	require.NoError(t, s.Run(context.Background()))
	require.Equal(t, []time.Duration{time.Second}, waits)
}

func TestStartupSyncerLogsOMCRedeployLifecycle(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	s := NewStartupSyncer(
		&startupPageLister{pages: [][]*model.Device{startupDevices(1)}},
		&startupSubmitter{}, nil, 200, zap.New(core),
	)

	require.NoError(t, s.Run(context.Background()))
	started := logs.FilterMessage("OMC redeploy full parameter sync started").All()
	completed := logs.FilterMessage("OMC redeploy full parameter sync completed").All()
	require.Len(t, started, 1)
	require.Len(t, completed, 1)
	require.Equal(t, "omc_redeploy", started[0].ContextMap()["trigger"])
	require.NotEmpty(t, started[0].ContextMap()["redeploy_id"])
	require.Equal(t, started[0].ContextMap()["redeploy_id"], completed[0].ContextMap()["redeploy_id"])
}

func TestStartupSyncerSkipsUPSDevices(t *testing.T) {
	ups := &model.Device{ID: uuid.New(), SerialNumber: "ups-device", ProductClass: "UPS_M3_BMU"}
	radio := &model.Device{ID: uuid.New(), SerialNumber: "radio-device", ProductClass: "FAP/TEST"}
	submitter := &startupSubmitter{}
	s := NewStartupSyncer(
		&startupPageLister{pages: [][]*model.Device{{ups, radio}}},
		submitter, nil, 200, zap.NewNop(),
	)

	require.NoError(t, s.Run(context.Background()))
	require.Equal(t, []string{"radio-device"}, submitter.calls)
}

func TestStartupSyncerNonLeaderDoesNothing(t *testing.T) {
	lister := &startupPageLister{}
	s := NewStartupSyncer(lister, &startupSubmitter{}, &fakeLeader{acquired: false}, 200, zap.NewNop())
	require.NoError(t, s.Run(context.Background()))
	require.Empty(t, lister.calls)
}

func TestStartupSyncerRetriesListFailureUntilRecovered(t *testing.T) {
	devices := startupDevices(1)
	lister := &startupPageLister{
		pages: [][]*model.Device{devices},
		errs:  []error{errors.New("database unavailable")},
	}
	submitter := &startupSubmitter{}
	s := NewStartupSyncer(lister, submitter, nil, 200, zap.NewNop())
	s.retryWait = func(context.Context, time.Duration) error { return nil }

	require.NoError(t, s.Run(context.Background()))
	require.Len(t, lister.calls, 2)
	require.Len(t, submitter.calls, 1)
}

func TestStartupSyncerUsesStableIdempotencyKeysOnRetry(t *testing.T) {
	dev := startupDevices(1)[0]
	lister := &startupPageLister{pages: [][]*model.Device{{dev}}}
	submitter := &startupSubmitter{}
	s := NewStartupSyncer(lister, submitter, nil, 200, zap.NewNop())
	require.NoError(t, s.Run(context.Background()))
	lister.calls = nil
	lister.next = 0
	require.NoError(t, s.Run(context.Background()))
	require.Equal(t, submitter.keys[0], submitter.keys[1])
}

func TestStartupSyncerRetriesOnlyUnpersistedSubmissionErrors(t *testing.T) {
	dev := startupDevices(1)[0]
	lister := &startupPageLister{pages: [][]*model.Device{{dev}}}
	submitter := &startupSubmitter{errs: []error{errors.New("database unavailable"), nil}}
	s := NewStartupSyncer(lister, submitter, nil, 200, zap.NewNop())
	s.retryWait = func(context.Context, time.Duration) error { return nil }
	require.NoError(t, s.Run(context.Background()))
	require.Len(t, submitter.calls, 2)
	require.Equal(t, submitter.keys[0], submitter.keys[1])
}

func TestStartupSyncerRetriesFailedDeviceUntilRecoveredWithStableKey(t *testing.T) {
	dev := startupDevices(1)[0]
	lister := &startupPageLister{pages: [][]*model.Device{{dev}}}
	submitter := &startupSubmitter{errs: []error{
		errors.New("attempt 1"), errors.New("attempt 2"), errors.New("attempt 3"), nil,
	}}
	s := NewStartupSyncer(lister, submitter, nil, 200, zap.NewNop())
	s.retryWait = func(context.Context, time.Duration) error { return nil }

	require.NoError(t, s.Run(context.Background()))
	require.Len(t, submitter.calls, 4)
	for _, key := range submitter.keys[1:] {
		require.Equal(t, submitter.keys[0], key)
	}
}

func TestStartupSyncerCancellationDuringRetryReleasesLeader(t *testing.T) {
	dev := startupDevices(1)[0]
	lister := &startupPageLister{pages: [][]*model.Device{{dev}}}
	submitter := &startupSubmitter{errs: []error{errors.New("database unavailable")}}
	leader := &fakeLeader{acquired: true}
	s := NewStartupSyncer(lister, submitter, leader, 200, zap.NewNop())
	retryStarted := make(chan struct{})
	s.retryWait = func(ctx context.Context, _ time.Duration) error {
		close(retryStarted)
		<-ctx.Done()
		return ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	<-retryStarted
	cancel()

	require.ErrorIs(t, <-done, context.Canceled)
	require.True(t, leader.released.Load())
}
