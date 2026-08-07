package pageconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

func TestAlarmEventConsumerUsesSingleSubjectDurablesAndFanout(t *testing.T) {
	bus := &alarmEventConsumerRecordingBus{handlers: map[string]event.EventHandler{}}
	var calls []string
	consumer := NewAlarmEventConsumer(
		bus,
		zap.NewNop(),
		func(_ context.Context, subject string, _ event.Event) error {
			calls = append(calls, "first:"+subject)
			return errors.New("first handler failed")
		},
		func(_ context.Context, subject string, _ event.Event) error {
			calls = append(calls, "second:"+subject)
			return nil
		},
	)

	require.NoError(t, consumer.Start())
	require.Equal(t, []string{event.SubjectAlarmRaised, event.SubjectAlarmCleared}, bus.subjects)
	require.Equal(t, []string{snmpForwarderQueue, snmpForwarderQueue + "-cleared"}, bus.queues)

	evt, err := event.NewEvent(event.SubjectAlarmRaised, map[string]string{"alarm": "raised"})
	require.NoError(t, err)
	require.NoError(t, bus.handlers[event.SubjectAlarmRaised](context.Background(), evt))
	require.Equal(t, []string{"first:alarm.raised", "second:alarm.raised"}, calls)
	require.NoError(t, consumer.Stop())
}

type alarmEventConsumerRecordingBus struct {
	subjects []string
	queues   []string
	handlers map[string]event.EventHandler
}

func (b *alarmEventConsumerRecordingBus) Publish(context.Context, string, event.Event) error {
	return nil
}

func (b *alarmEventConsumerRecordingBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return alarmEventConsumerSubscription{}, nil
}

func (b *alarmEventConsumerRecordingBus) QueueSubscribe(subject string, queue string, handler event.EventHandler) (event.Subscription, error) {
	b.subjects = append(b.subjects, subject)
	b.queues = append(b.queues, queue)
	b.handlers[subject] = handler
	return alarmEventConsumerSubscription{}, nil
}

func (b *alarmEventConsumerRecordingBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return alarmEventConsumerSubscription{}, nil
}

func (b *alarmEventConsumerRecordingBus) Close() error {
	return nil
}

type alarmEventConsumerSubscription struct{}

func (alarmEventConsumerSubscription) Unsubscribe() error {
	return nil
}
