package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGeofenceAlarmMonitorProjectsDeviceExitToOneActiveAlarm(
	t *testing.T,
) {
	tests := []struct {
		name        string
		actionLevel string
		severity    model.AlarmSeverity
	}{
		{
			name:        "notify only",
			actionLevel: "notify_only",
			severity:    model.AlarmWarning,
		},
		{
			name:        "manual review",
			actionLevel: "manual_review",
			severity:    model.AlarmMajor,
		},
		{
			name:        "deactivate",
			actionLevel: "deactivate",
			severity:    model.AlarmCritical,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockAlarmStore()
			monitor := NewGeofenceAlarmMonitor(
				newTestEngine(store),
				zap.NewNop(),
			)
			exited := geofenceDeviceStateEvent(
				t,
				event.SubjectGeofenceDeviceExited,
				"outside",
				tt.actionLevel,
			)

			require.NoError(t, monitor.handle(context.Background(), exited))
			require.NoError(t, monitor.handle(context.Background(), exited))

			require.Len(t, store.active, 1)
			for _, saved := range store.active {
				require.Equal(t, AlarmCodeGeofenceLocationOutside, saved.AlarmIdentifier)
				require.Equal(t, model.CarrierCMCC, saved.Carrier)
				require.Equal(t, tt.severity, saved.Severity)
				require.Equal(t, "geofence", saved.AlarmType)
				require.Equal(t, "omc", *saved.AlarmSource)
				require.Equal(t, "location_outside", *saved.EventType)
				require.Equal(t, "12", saved.AdditionalInfo["observation_version"])
				require.Equal(t, "4", saved.AdditionalInfo["effective_state_version"])
				require.Equal(t, tt.actionLevel, saved.AdditionalInfo["required_action_level"])
			}
		})
	}
}

func TestGeofenceAlarmMonitorClearsActiveAlarmOnDeviceEntry(
	t *testing.T,
) {
	store := newMockAlarmStore()
	monitor := NewGeofenceAlarmMonitor(newTestEngine(store), zap.NewNop())

	require.NoError(t, monitor.handle(
		context.Background(),
		geofenceDeviceStateEvent(
			t,
			event.SubjectGeofenceDeviceExited,
			"outside",
			"notify_only",
		),
	))
	require.NoError(t, monitor.handle(
		context.Background(),
		geofenceDeviceStateEvent(
			t,
			event.SubjectGeofenceDeviceEntered,
			"inside",
			"none",
		),
	))

	require.Empty(t, store.active)
	require.Len(t, store.history, 1)
	require.Equal(t, model.AlarmCleared, store.history[0].Status)
	require.Equal(t, "system", *store.history[0].ClearedBy)
}

func TestGeofenceAlarmMonitorDropsMalformedOrIncompleteEvents(
	t *testing.T,
) {
	store := newMockAlarmStore()
	monitor := NewGeofenceAlarmMonitor(newTestEngine(store), zap.NewNop())

	require.NoError(t, monitor.handle(context.Background(), event.Event{
		Subject: event.SubjectGeofenceDeviceExited,
		Payload: json.RawMessage(`{"serial_number":`),
	}))
	require.NoError(t, monitor.handle(
		context.Background(),
		geofenceDeviceStateEvent(
			t,
			event.SubjectGeofenceDeviceExited,
			"inside",
			"notify_only",
		),
	))

	require.Empty(t, store.active)
	require.Empty(t, store.history)
}

func TestGeofenceAlarmMonitorSubscribesExitAndEntryEdges(
	t *testing.T,
) {
	store := newMockAlarmStore()
	monitor := NewGeofenceAlarmMonitor(newTestEngine(store), zap.NewNop())
	bus := &geofenceAlarmTestBus{
		handlers: make(map[string]event.EventHandler),
		queues:   make(map[string]string),
	}

	require.NoError(t, monitor.Subscribe(bus))
	require.Equal(
		t,
		GeofenceAlarmExitedQueue,
		bus.queues[event.SubjectGeofenceDeviceExited],
	)
	require.Equal(
		t,
		GeofenceAlarmEnteredQueue,
		bus.queues[event.SubjectGeofenceDeviceEntered],
	)
	require.NotEqual(
		t,
		bus.queues[event.SubjectGeofenceDeviceExited],
		bus.queues[event.SubjectGeofenceDeviceEntered],
	)

	require.NoError(t, bus.handle(
		context.Background(),
		geofenceDeviceStateEvent(
			t,
			event.SubjectGeofenceDeviceExited,
			"outside",
			"notify_only",
		),
	))
	require.Len(t, store.active, 1)
	require.NoError(t, bus.handle(
		context.Background(),
		geofenceDeviceStateEvent(
			t,
			event.SubjectGeofenceDeviceEntered,
			"inside",
			"none",
		),
	))
	require.Empty(t, store.active)
	require.Len(t, store.history, 1)
}

func geofenceDeviceStateEvent(
	t *testing.T,
	subject string,
	state string,
	actionLevel string,
) event.Event {
	t.Helper()
	bindingID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	occurredAt := time.Date(2026, 7, 31, 8, 5, 0, 0, time.UTC)
	evt, err := event.NewEvent(subject, event.GeofenceDeviceStatePayload{
		DeviceID:              uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		SerialNumber:          "SN001",
		Carrier:               "cmcc",
		TriggerBindingID:      &bindingID,
		ObservationVersion:    12,
		EffectiveState:        state,
		RequiredActionLevel:   actionLevel,
		EffectiveStateVersion: 4,
		EvaluationHealth:      "healthy",
		ReasonCode:            "confirmed_state_changed",
		OccurredAt:            occurredAt,
	})
	require.NoError(t, err)
	return evt
}

type geofenceAlarmTestBus struct {
	handlers map[string]event.EventHandler
	queues   map[string]string
}

func (b *geofenceAlarmTestBus) Publish(
	context.Context,
	string,
	event.Event,
) error {
	return nil
}

func (b *geofenceAlarmTestBus) Subscribe(
	string,
	event.EventHandler,
) (event.Subscription, error) {
	return nil, fmt.Errorf("unexpected broadcast subscription")
}

func (b *geofenceAlarmTestBus) QueueSubscribe(
	subject string,
	queue string,
	handler event.EventHandler,
) (event.Subscription, error) {
	b.handlers[subject] = handler
	b.queues[subject] = queue
	return geofenceAlarmTestSubscription{}, nil
}

func (b *geofenceAlarmTestBus) PullSubscribe(
	string,
	string,
	event.EventHandler,
) (event.Subscription, error) {
	return nil, fmt.Errorf("unexpected pull subscription")
}

func (b *geofenceAlarmTestBus) Close() error {
	return nil
}

func (b *geofenceAlarmTestBus) handle(
	ctx context.Context,
	evt event.Event,
) error {
	handler, ok := b.handlers[evt.Subject]
	if !ok {
		return fmt.Errorf("no handler for %s", evt.Subject)
	}
	return handler(ctx, evt)
}

type geofenceAlarmTestSubscription struct{}

func (geofenceAlarmTestSubscription) Unsubscribe() error {
	return nil
}
