package alarm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAlarmEmailSchedulerRepository struct {
	setting       AlarmEmailGlobalSetting
	subscriptions map[int][]AlarmEmailSubscription
	latest        map[uuid.UUID]time.Time
	enqueued      []AlarmEmailWindow
	scheduled     []time.Time
}

func (f *fakeAlarmEmailSchedulerRepository) GetGlobalSetting(context.Context) (*AlarmEmailGlobalSetting, error) {
	setting := f.setting
	return &setting, nil
}

func (f *fakeAlarmEmailSchedulerRepository) ListEnabledSubscriptions(_ context.Context, interval int) ([]AlarmEmailSubscription, error) {
	return f.subscriptions[interval], nil
}

func (f *fakeAlarmEmailSchedulerRepository) LatestPeriodicWindowEnd(_ context.Context, subscriptionID uuid.UUID) (*time.Time, error) {
	latest, ok := f.latest[subscriptionID]
	if !ok {
		return nil, nil
	}
	return &latest, nil
}

func (f *fakeAlarmEmailSchedulerRepository) EnqueueRun(_ context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string, window AlarmEmailWindow, scheduledAt time.Time) (*AlarmEmailRun, bool, error) {
	f.enqueued = append(f.enqueued, window)
	f.scheduled = append(f.scheduled, scheduledAt)
	recipients := append([]string(nil), subscription.Recipients...)
	if subscription.IncludeDefaultRecipients {
		recipients = append(recipients, defaultRecipients...)
	}
	if window.End.Sub(window.Start) >= time.Minute {
		if f.latest == nil {
			f.latest = make(map[uuid.UUID]time.Time)
		}
		if latest, ok := f.latest[subscription.ID]; !ok || window.End.After(latest) {
			f.latest[subscription.ID] = window.End
		}
	}
	return &AlarmEmailRun{
		ID: uuid.New(), SubscriptionID: subscription.ID, Window: window,
		SubscriptionSnapshot: *subscription, RecipientsSnapshot: recipients,
	}, true, nil
}

func TestAlarmEmailSchedulerEnqueuesOneWindowPerBucket(t *testing.T) {
	subscriptionID := uuid.New()
	repository := &fakeAlarmEmailSchedulerRepository{
		setting: AlarmEmailGlobalSetting{Enabled: true},
		subscriptions: map[int][]AlarmEmailSubscription{
			AlarmEmailInterval10Min: {{
				ID: subscriptionID, IntervalMinutes: 10, ToleranceMinutes: 10, Enabled: true,
			}},
		},
	}
	scheduler := NewAlarmEmailScheduler(repository, nil)

	now := time.Date(2026, 8, 17, 12, 3, 0, 0, time.UTC)
	count, err := scheduler.RunOnce(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, repository.enqueued, 1)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 40, 0, 0, time.UTC), repository.enqueued[0].Start)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 50, 0, 0, time.UTC), repository.enqueued[0].End)

	count, err = scheduler.RunOnce(context.Background(), now.Add(5*time.Minute))
	require.NoError(t, err)
	assert.Zero(t, count)
	assert.Len(t, repository.enqueued, 1)

	count, err = scheduler.RunOnce(context.Background(), now.Add(7*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, repository.enqueued, 2)
}

func TestAlarmEmailSchedulerDoesNothingWhenGloballyDisabled(t *testing.T) {
	repository := &fakeAlarmEmailSchedulerRepository{setting: AlarmEmailGlobalSetting{Enabled: false}}
	count, err := NewAlarmEmailScheduler(repository, nil).RunOnce(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Zero(t, count)
	assert.Empty(t, repository.enqueued)
}

func TestAlarmEmailSchedulerCatchesUpPersistedMissingWindows(t *testing.T) {
	subscriptionID := uuid.New()
	repository := &fakeAlarmEmailSchedulerRepository{
		setting: AlarmEmailGlobalSetting{Enabled: true},
		subscriptions: map[int][]AlarmEmailSubscription{
			AlarmEmailInterval10Min: {{ID: subscriptionID, IntervalMinutes: 10, Enabled: true}},
		},
		latest: map[uuid.UUID]time.Time{
			subscriptionID: time.Date(2026, 8, 17, 11, 20, 0, 0, time.UTC),
		},
	}

	count, err := NewAlarmEmailScheduler(repository, nil).RunOnce(
		context.Background(),
		time.Date(2026, 8, 17, 11, 55, 0, 0, time.UTC),
	)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Len(t, repository.enqueued, 3)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 20, 0, 0, time.UTC), repository.enqueued[0].Start)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 50, 0, 0, time.UTC), repository.enqueued[2].End)
}

func TestAlarmEmailSchedulerSkipsStaleCatchUpBacklog(t *testing.T) {
	subscriptionID := uuid.New()
	repository := &fakeAlarmEmailSchedulerRepository{
		setting: AlarmEmailGlobalSetting{Enabled: true},
		subscriptions: map[int][]AlarmEmailSubscription{
			AlarmEmailInterval10Min: {{ID: subscriptionID, IntervalMinutes: 10, Enabled: true}},
		},
		latest: map[uuid.UUID]time.Time{
			subscriptionID: time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC),
		},
	}

	count, err := NewAlarmEmailScheduler(repository, nil).RunOnce(
		context.Background(),
		time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC),
	)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, repository.enqueued, 1)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 50, 0, 0, time.UTC), repository.enqueued[0].Start)
}

func TestAlarmEmailSchedulerPreservesFirstPartialWindowAfterSubscriptionActivation(t *testing.T) {
	subscriptionID := uuid.New()
	updatedAt := time.Date(2026, 8, 17, 11, 45, 0, 0, time.UTC)
	repository := &fakeAlarmEmailSchedulerRepository{
		setting: AlarmEmailGlobalSetting{Enabled: true},
		subscriptions: map[int][]AlarmEmailSubscription{
			AlarmEmailInterval10Min: {{
				ID: subscriptionID, IntervalMinutes: 10, Enabled: true, UpdatedAt: updatedAt,
			}},
		},
	}
	scheduler := NewAlarmEmailScheduler(repository, nil)

	count, err := scheduler.RunOnce(context.Background(), time.Date(2026, 8, 17, 11, 55, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, repository.enqueued, 1)
	assert.Equal(t, updatedAt, repository.enqueued[0].Start)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 50, 0, 0, time.UTC), repository.enqueued[0].End)

	count, err = scheduler.RunOnce(context.Background(), time.Date(2026, 8, 17, 12, 5, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, repository.enqueued, 2)
	assert.Equal(t, time.Date(2026, 8, 17, 11, 50, 0, 0, time.UTC), repository.enqueued[1].Start)
}
