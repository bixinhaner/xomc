package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type failOnceAlarmStore struct {
	*mockAlarmStore
	failed bool
}

func (s *failOnceAlarmStore) SaveActive(ctx context.Context, alarm *model.Alarm) error {
	if !s.failed {
		s.failed = true
		return errors.New("injected save failure")
	}
	return s.mockAlarmStore.SaveActive(ctx, alarm)
}

func TestExpeditedEventReceiverPublishesOneSyncRequestPerValidBatch(t *testing.T) {
	store := newMockAlarmStore()
	bus := newMockEventBus()
	receiver := newExpeditedReceiverForSyncTest(store, bus)
	evt := newExpeditedBatchEvent(t,
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11109"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Major"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.11.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.11.AlarmIdentifier", "11112"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.11.PerceivedSeverity", "Critical"),
	)

	require.NoError(t, receiver.handleExpeditedAlarmEvent(context.Background(), evt))

	assert.Len(t, store.active, 2)
	require.Len(t, bus.published[event.SubjectAlarmSyncRequested], 1)
}

func TestExpeditedEventReceiverDoesNotSyncInvalidOnlyBatch(t *testing.T) {
	store := newMockAlarmStore()
	bus := newMockEventBus()
	receiver := newExpeditedReceiverForSyncTest(store, bus)
	evt := newExpeditedBatchEvent(t,
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
	)

	require.NoError(t, receiver.handleExpeditedAlarmEvent(context.Background(), evt))

	assert.Empty(t, store.active)
	assert.Empty(t, bus.published[event.SubjectAlarmSyncRequested])
}

func TestExpeditedEventReceiverStillSyncsAfterPartialProcessingFailure(t *testing.T) {
	store := &failOnceAlarmStore{mockAlarmStore: newMockAlarmStore()}
	bus := newMockEventBus()
	receiver := newExpeditedReceiverForSyncTest(store, bus)
	evt := newExpeditedBatchEvent(t,
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11109"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Major"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.11.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.11.AlarmIdentifier", "11112"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.11.PerceivedSeverity", "Critical"),
	)

	require.NoError(t, receiver.handleExpeditedAlarmEvent(context.Background(), evt))

	assert.Len(t, store.active, 1)
	require.Len(t, bus.published[event.SubjectAlarmSyncRequested], 1)
}

func newExpeditedReceiverForSyncTest(store AlarmStore, bus event.EventBus) *ExpeditedEventReceiver {
	engine := newTestEngine(store)
	deviceLookup := &mockDeviceLookup{
		devices: map[string]*model.Device{"SN-TEST": newTestDevice()},
	}
	return NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())
}

func newExpeditedBatchEvent(t *testing.T, params ...tr069.ParameterValueStruct) event.Event {
	t.Helper()
	payload, err := json.Marshal(ExpeditedEventPayload{
		DeviceSN:        "SN-TEST",
		ParameterValues: params,
	})
	require.NoError(t, err)
	return event.Event{
		ID:      "expedited-sync-test",
		Subject: event.SubjectDeviceExpeditedAlarm,
		Payload: payload,
	}
}
