package push

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type retryingLifecycleEnqueuer struct {
	attempts atomic.Int64
	success  chan event.Event
}

func (e *retryingLifecycleEnqueuer) EnqueueEvent(_ context.Context, evt event.Event) error {
	if e.attempts.Add(1) == 1 {
		return errors.New("injected northbound outbox failure")
	}
	e.success <- evt
	return nil
}

func TestAlarmLifecycleConsumer_JetStreamRetriesBeforeAck(t *testing.T) {
	natsURL := os.Getenv("ALARM_LIFECYCLE_NATS_TEST_URL")
	if natsURL == "" {
		t.Skip("set ALARM_LIFECYCLE_NATS_TEST_URL to a dedicated NATS instance")
	}

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	_, err = js.AddStream(&nats.StreamConfig{
		Name:        "DOMAIN_ALARM",
		Subjects:    []string{"domain.alarm.>"},
		Retention:   nats.LimitsPolicy,
		Storage:     nats.MemoryStorage,
		AllowDirect: true,
		MaxAge:      time.Hour,
		MaxBytes:    16 << 20,
	})
	require.NoError(t, err, "integration test requires an empty dedicated NATS instance")
	t.Cleanup(func() { _ = js.DeleteStream("DOMAIN_ALARM") })

	bus := event.NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	enqueuer := &retryingLifecycleEnqueuer{success: make(chan event.Event, 1)}
	consumer := NewAlarmLifecycleConsumer(enqueuer, bus, 1)
	require.NoError(t, consumer.Subscribe())
	t.Cleanup(func() { _ = consumer.Close() })

	occurredAt := time.Now().UTC().Truncate(time.Millisecond)
	alarm := model.Alarm{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "ENB-JS-001",
		Severity: model.AlarmCritical, Status: model.AlarmActive, RaisedAt: occurredAt,
	}
	payload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised, alarm, nil, 1, occurredAt, nil,
	)
	require.NoError(t, err)
	envelope := event.Event{
		ID: payload.EventID.String(), Subject: event.SubjectDomainAlarmLifecycleRaised,
		Timestamp: occurredAt,
	}
	envelope.Payload, err = json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, bus.Publish(context.Background(), envelope.Subject, envelope))

	var forwarded event.Event
	select {
	case forwarded = <-enqueuer.success:
	case <-time.After(5 * time.Second):
		t.Fatal("canonical lifecycle event was not retried into the northbound outbox")
	}
	require.Equal(t, int64(2), enqueuer.attempts.Load())
	require.Equal(t, payload.EventID.String(), forwarded.ID)
	var snapshot event.AlarmLifecycleSnapshot
	require.NoError(t, forwarded.DecodePayload(&snapshot))
	require.Equal(t, payload.Snapshot, snapshot)

	require.Eventually(t, func() bool {
		return consumer.Ready(context.Background()) == nil
	}, 3*time.Second, 20*time.Millisecond)
	info, err := js.ConsumerInfo("DOMAIN_ALARM", AlarmLifecycleConsumerDurable)
	require.NoError(t, err)
	require.Equal(t, event.SubjectDomainAlarmLifecycleAll, info.Config.FilterSubject)
	require.Equal(t, 512, info.Config.MaxAckPending)
	require.Equal(t, uint64(1), info.Config.OptStartSeq)
}
