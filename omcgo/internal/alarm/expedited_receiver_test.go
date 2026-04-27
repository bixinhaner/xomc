package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockDeviceLookup for ExpeditedEventReceiver tests.
type mockDeviceLookup struct {
	devices map[string]*model.Device
}

func (m *mockDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	d, ok := m.devices[sn]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", sn)
	}
	return d, nil
}

// mockEventBus for ExpeditedEventReceiver tests.
type mockEventBus struct {
	published map[string][]event.Event
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{published: make(map[string][]event.Event)}
}

func (m *mockEventBus) Publish(_ context.Context, subject string, evt event.Event) error {
	m.published[subject] = append(m.published[subject], evt)
	return nil
}

func (m *mockEventBus) Subscribe(_ string, handler event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (m *mockEventBus) QueueSubscribe(_ string, _ string, handler event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (m *mockEventBus) Close() error { return nil }

func newTestDevice() *model.Device {
	return &model.Device{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		SerialNumber: "SN-TEST",
		Carrier:      model.CarrierCMCC,
		Status:       model.DeviceActive,
	}
}

func TestExpeditedEventReceiver_NewAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	bus := newMockEventBus()
	deviceLookup := &mockDeviceLookup{
		devices: map[string]*model.Device{"SN-TEST": newTestDevice()},
	}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "11184"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ProbableCause", "Cell unavailable"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.EventTime", "2026-04-23T09:03:01"),
	}

	payload := ExpeditedEventPayload{
		DeviceSN:        "SN-TEST",
		ParameterValues: params,
	}
	payloadJSON, _ := json.Marshal(payload)
	evt := event.Event{ID: "test-1", Subject: event.SubjectDeviceExpeditedAlarm, Payload: payloadJSON}

	err := receiver.handleExpeditedAlarmEvent(context.Background(), evt)
	require.NoError(t, err)

	// Verify alarm was created
	assert.Len(t, store.active, 1)
	for _, a := range store.active {
		assert.Equal(t, "11184", a.AlarmIdentifier)
		assert.Equal(t, "SN-TEST", a.DeviceSN)
		assert.Equal(t, model.CarrierCMCC, a.Carrier)
	}
}

func TestExpeditedEventReceiver_ChangedAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	bus := newMockEventBus()
	deviceLookup := &mockDeviceLookup{
		devices: map[string]*model.Device{"SN-TEST": newTestDevice()},
	}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())
	ctx := context.Background()

	// Pre-create an existing alarm
	deviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	existing := &model.Alarm{
		DeviceSN:        "SN-TEST",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM-001",
		Severity:        model.AlarmMajor,
		Description:     "original",
		RaisedAt:        mustParseTime(t, "2026-04-23T09:00:00Z"),
	}
	require.NoError(t, engine.Process(ctx, existing))

	// Send ChangedAlarm event
	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "ChangedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "ALM-001"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.PerceivedSeverity", "Critical"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.ProbableCause", "Cell unavailable"),
	}

	payload := ExpeditedEventPayload{
		DeviceSN:        "SN-TEST",
		ParameterValues: params,
	}
	payloadJSON, _ := json.Marshal(payload)
	evt := event.Event{ID: "test-2", Subject: event.SubjectDeviceExpeditedAlarm, Payload: payloadJSON}

	err := receiver.handleExpeditedAlarmEvent(ctx, evt)
	require.NoError(t, err)

	// Verify alarm was updated (not duplicated)
	assert.Len(t, store.active, 1)
	updated, err := store.GetActiveByDeviceAndIdentifier(ctx, "SN-TEST", "ALM-001")
	require.NoError(t, err)
	assert.Equal(t, model.AlarmCritical, updated.Severity)
}

func TestExpeditedEventReceiver_ClearedAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	bus := newMockEventBus()
	deviceLookup := &mockDeviceLookup{
		devices: map[string]*model.Device{"SN-TEST": newTestDevice()},
	}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())
	ctx := context.Background()

	// Pre-create an existing alarm
	deviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	existing := &model.Alarm{
		DeviceSN:        "SN-TEST",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM-001",
		Severity:        model.AlarmCritical,
		RaisedAt:        mustParseTime(t, "2026-04-23T09:00:00Z"),
	}
	require.NoError(t, engine.Process(ctx, existing))

	// Send ClearedAlarm event
	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "ClearedAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "ALM-001"),
	}

	payload := ExpeditedEventPayload{
		DeviceSN:        "SN-TEST",
		ParameterValues: params,
	}
	payloadJSON, _ := json.Marshal(payload)
	evt := event.Event{ID: "test-3", Subject: event.SubjectDeviceExpeditedAlarm, Payload: payloadJSON}

	err := receiver.handleExpeditedAlarmEvent(ctx, evt)
	require.NoError(t, err)

	// Verify alarm was cleared
	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
}

func TestExpeditedEventReceiver_DeviceNotFound(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	bus := newMockEventBus()
	deviceLookup := &mockDeviceLookup{} // no devices
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "NewAlarm"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "ALM-001"),
	}

	payload := ExpeditedEventPayload{
		DeviceSN:        "SN-UNKNOWN",
		ParameterValues: params,
	}
	payloadJSON, _ := json.Marshal(payload)
	evt := event.Event{ID: "test-4", Subject: event.SubjectDeviceExpeditedAlarm, Payload: payloadJSON}

	err := receiver.handleExpeditedAlarmEvent(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device not found")
	assert.Len(t, store.active, 0)
}

func TestExpeditedEventReceiver_InvalidNotificationType(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	bus := newMockEventBus()
	deviceLookup := &mockDeviceLookup{
		devices: map[string]*model.Device{"SN-TEST": newTestDevice()},
	}
	receiver := NewExpeditedEventReceiver(engine, deviceLookup, bus, zap.NewNop())

	params := []tr069.ParameterValueStruct{
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.NotificationType", "InvalidType"),
		makeParam("Device.FaultMgmt.ExpeditedEvent.10.AlarmIdentifier", "ALM-001"),
	}

	payload := ExpeditedEventPayload{
		DeviceSN:        "SN-TEST",
		ParameterValues: params,
	}
	payloadJSON, _ := json.Marshal(payload)
	evt := event.Event{ID: "test-5", Subject: event.SubjectDeviceExpeditedAlarm, Payload: payloadJSON}

	// Should not error — invalid events are skipped, not fatal
	err := receiver.handleExpeditedAlarmEvent(context.Background(), evt)
	require.NoError(t, err)
	assert.Len(t, store.active, 0)
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}
