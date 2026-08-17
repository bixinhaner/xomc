package alarm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlarmEmailRealtimeSubscriberEnqueuesRaisedAlarmAfterTolerance(t *testing.T) {
	subscriptionID := uuid.New()
	repository := &fakeAlarmEmailSchedulerRepository{
		setting: AlarmEmailGlobalSetting{Enabled: true},
		subscriptions: map[int][]AlarmEmailSubscription{
			AlarmEmailIntervalRealtime: {{
				ID: subscriptionID, IntervalMinutes: 0, ToleranceMinutes: 10, Enabled: true,
			}},
		},
	}
	raisedAt := time.Date(2026, 8, 17, 12, 0, 0, 123456789, time.UTC)
	evt, err := event.NewEvent(event.SubjectAlarmRaised, model.Alarm{
		ID: uuid.New(), RaisedAt: raisedAt,
	})
	require.NoError(t, err)

	err = NewAlarmEmailRealtimeSubscriber(repository, nil).handleRaised(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, repository.enqueued, 1)
	assert.Equal(t, raisedAt.Truncate(time.Microsecond), repository.enqueued[0].Start)
	assert.Equal(t, raisedAt.Truncate(time.Microsecond).Add(time.Microsecond), repository.enqueued[0].End)
	assert.Equal(t, raisedAt.Add(10*time.Minute), repository.scheduled[0])
}

func TestAlarmEmailRealtimeSubscriberSkipsClearCoveredByToleranceWindow(t *testing.T) {
	repository := newRealtimeTestRepository(10)
	raisedAt := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	clearedAt := raisedAt.Add(5 * time.Minute)
	evt, err := event.NewEvent(event.SubjectAlarmCleared, model.Alarm{
		ID: uuid.New(), RaisedAt: raisedAt, ClearedAt: &clearedAt,
	})
	require.NoError(t, err)

	err = NewAlarmEmailRealtimeSubscriber(repository, nil).handleCleared(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, repository.enqueued)
}

func TestAlarmEmailRealtimeSubscriberEnqueuesClearAfterToleranceWindow(t *testing.T) {
	repository := newRealtimeTestRepository(10)
	raisedAt := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	clearedAt := raisedAt.Add(15 * time.Minute)
	evt, err := event.NewEvent(event.SubjectAlarmCleared, model.Alarm{
		ID: uuid.New(), RaisedAt: raisedAt, ClearedAt: &clearedAt,
	})
	require.NoError(t, err)

	err = NewAlarmEmailRealtimeSubscriber(repository, nil).handleCleared(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, repository.enqueued, 1)
	assert.Equal(t, clearedAt, repository.enqueued[0].Start)
	assert.Equal(t, clearedAt.Add(time.Microsecond), repository.enqueued[0].End)
	assert.Equal(t, clearedAt, repository.scheduled[0])
}

func TestAlarmEmailRealtimeWindowsDoNotOverlap(t *testing.T) {
	first := alarmEmailRealtimeWindow(time.Date(2026, 8, 17, 12, 0, 0, 123456100, time.UTC))
	second := alarmEmailRealtimeWindow(time.Date(2026, 8, 17, 12, 0, 0, 123457100, time.UTC))

	assert.Equal(t, first.End, second.Start)
}

func TestAlarmEmailRealtimeSubscriberUsesDistinctDurables(t *testing.T) {
	bus := &recordingAlarmEmailEventBus{}
	repository := &fakeAlarmEmailSchedulerRepository{}

	err := NewAlarmEmailRealtimeSubscriber(repository, nil).Subscribe(bus)

	require.NoError(t, err)
	require.Len(t, bus.queues, 2)
	assert.Equal(t, []string{event.SubjectAlarmEmailRaised, event.SubjectAlarmEmailCleared}, bus.subjects)
	assert.Equal(t, alarmEmailRealtimeRaisedQueue, bus.queues[0])
	assert.Equal(t, alarmEmailRealtimeClearedQueue, bus.queues[1])
	assert.NotEqual(t, bus.queues[0], bus.queues[1])
}

func TestAlarmEmailRealtimeSubscriberCleansUpPartialSubscription(t *testing.T) {
	bus := &recordingAlarmEmailEventBus{failSubject: event.SubjectAlarmEmailCleared}

	err := NewAlarmEmailRealtimeSubscriber(&fakeAlarmEmailSchedulerRepository{}, nil).Subscribe(bus)

	require.Error(t, err)
	require.Len(t, bus.subscriptions, 1)
	assert.True(t, bus.subscriptions[0].unsubscribed)
}

type recordingAlarmEmailSubscription struct {
	unsubscribed bool
}

func (s *recordingAlarmEmailSubscription) Unsubscribe() error {
	s.unsubscribed = true
	return nil
}

type recordingAlarmEmailEventBus struct {
	queues        []string
	subjects      []string
	published     []string
	subscriptions []*recordingAlarmEmailSubscription
	failSubject   string
}

func (b *recordingAlarmEmailEventBus) Publish(_ context.Context, subject string, _ event.Event) error {
	b.published = append(b.published, subject)
	return nil
}
func (b *recordingAlarmEmailEventBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (b *recordingAlarmEmailEventBus) QueueSubscribe(subject, queue string, _ event.EventHandler) (event.Subscription, error) {
	if subject == b.failSubject {
		return nil, errors.New("subscribe failed")
	}
	subscription := &recordingAlarmEmailSubscription{}
	b.queues = append(b.queues, queue)
	b.subjects = append(b.subjects, subject)
	b.subscriptions = append(b.subscriptions, subscription)
	return subscription, nil
}
func (b *recordingAlarmEmailEventBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (b *recordingAlarmEmailEventBus) Close() error { return nil }

func newRealtimeTestRepository(toleranceMinutes int) *fakeAlarmEmailSchedulerRepository {
	return &fakeAlarmEmailSchedulerRepository{
		setting: AlarmEmailGlobalSetting{Enabled: true},
		subscriptions: map[int][]AlarmEmailSubscription{
			AlarmEmailIntervalRealtime: {{
				ID: uuid.New(), IntervalMinutes: 0, ToleranceMinutes: toleranceMinutes, Enabled: true,
			}},
		},
	}
}
