package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// rcvMock prefix to avoid conflicts with engine_test / handler_test
// ---------------------------------------------------------------------------

type rcvMockSubscription struct{}

func (s *rcvMockSubscription) Unsubscribe() error { return nil }

type rcvMockEventBus struct {
	subscribedSubject string
	subscribedQueue   string
	subscribedHandler event.EventHandler
	subscribeErr      error
	published         map[string][]event.Event
}

func (b *rcvMockEventBus) Publish(_ context.Context, subject string, evt event.Event) error {
	if b.published == nil {
		b.published = map[string][]event.Event{}
	}
	b.published[subject] = append(b.published[subject], evt)
	return nil
}
func (b *rcvMockEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &rcvMockSubscription{}, nil
}
func (b *rcvMockEventBus) QueueSubscribe(subject, queue string, handler event.EventHandler) (event.Subscription, error) {
	b.subscribedSubject = subject
	b.subscribedQueue = queue
	b.subscribedHandler = handler
	if b.subscribeErr != nil {
		return nil, b.subscribeErr
	}
	return &rcvMockSubscription{}, nil
}
func (b *rcvMockEventBus) PullSubscribe(subject, queue string, handler event.EventHandler) (event.Subscription, error) {
	return b.QueueSubscribe(subject, queue, handler)
}
func (b *rcvMockEventBus) Close() error { return nil }

type rcvMockDeviceReader struct {
	deviceByID map[uuid.UUID]*model.Device
	deviceBySN map[string]*model.Device
}

func (r *rcvMockDeviceReader) GetByID(_ context.Context, id uuid.UUID) (*model.Device, error) {
	if r.deviceByID == nil {
		return nil, nil
	}
	return r.deviceByID[id], nil
}

func (r *rcvMockDeviceReader) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	if r.deviceBySN == nil {
		return nil, nil
	}
	return r.deviceBySN[sn], nil
}

func makeRcvEvent(t *testing.T, payload interface{}) event.Event {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return event.Event{
		ID:        uuid.New().String(),
		Subject:   event.SubjectDeviceAlarm,
		Payload:   data,
		Timestamp: time.Now(),
	}
}

type stubAlarmDefRepo struct {
	defs []definition.ResolvedDefinition
	err  error
}

func (r *stubAlarmDefRepo) ListAll(context.Context) ([]definition.ResolvedDefinition, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.defs, nil
}

func (r *stubAlarmDefRepo) ListSeverityLevels(context.Context) ([]definition.SeverityLevel, error) {
	return nil, nil
}

func newTestAlarmDefRegistry(t *testing.T, defs ...definition.ResolvedDefinition) *definition.Registry {
	t.Helper()
	registry := definition.NewRegistry(&stubAlarmDefRepo{defs: defs}, nil, zap.NewNop())
	require.NoError(t, registry.Refresh(context.Background()))
	return registry
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewAlarmReceiver(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop())

	require.NotNil(t, receiver)
	assert.Equal(t, engine, receiver.engine)
}

func TestSubscribe_Success(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop())

	bus := &rcvMockEventBus{}
	err := receiver.Subscribe(bus)
	require.NoError(t, err)
	assert.Equal(t, event.SubjectDeviceAlarm, bus.subscribedSubject)
	assert.Equal(t, "alarm-workers", bus.subscribedQueue)
	assert.NotNil(t, bus.subscribedHandler)
}

func TestSubscribe_Error(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop())

	bus := &rcvMockEventBus{subscribeErr: errors.New("connection refused")}
	err := receiver.Subscribe(bus)
	require.Error(t, err)
}

func TestHandleAlarmEvent_Success(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop())

	deviceID := uuid.New()
	now := time.Now().Truncate(time.Second)

	payload := AlarmPayload{
		DeviceID:        deviceID.String(),
		DeviceSN:        "SN-RCV-001",
		Carrier:         "cmcc",
		AlarmIdentifier: "ALM_TEST_01",
		AlarmType:       "equipment",
		Description:     "test alarm from receiver",
		Severity:        2,
		RaisedAt:        now,
		Additional:      map[string]string{"key": "value"},
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.NoError(t, err)

	assert.Len(t, store.active, 1)
	for _, alarm := range store.active {
		assert.Equal(t, deviceID, alarm.DeviceID)
		assert.Equal(t, "SN-RCV-001", alarm.DeviceSN)
		assert.Equal(t, model.CarrierCode("cmcc"), alarm.Carrier)
		assert.Equal(t, "ALM_TEST_01", alarm.AlarmIdentifier)
	}
}

func TestHandleAlarmEvent_InvalidDeviceID(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop())

	payload := AlarmPayload{
		DeviceID:        "not-a-valid-uuid",
		DeviceSN:        "SN-RCV-002",
		AlarmIdentifier: "ALM_TEST_02",
		Severity:        1,
		RaisedAt:        time.Now(),
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.Error(t, err)
	assert.Len(t, store.active, 0)
}

func TestHandleAlarmEvent_BackfillsTechnologyFromDevice(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceByID: map[uuid.UUID]*model.Device{
			deviceID: {
				ID:           deviceID,
				SerialNumber: "SN-RCV-003",
				Technology:   model.TechNR,
			},
		},
	})

	payload := AlarmPayload{
		DeviceID:        deviceID.String(),
		DeviceSN:        "SN-RCV-003",
		Carrier:         "cmcc",
		AlarmIdentifier: "ALM_TEST_03",
		AlarmType:       "equipment",
		Description:     "alarm without technology payload",
		Severity:        2,
		RaisedAt:        time.Now(),
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, store.active, 1)
	for _, alarm := range store.active {
		require.NotNil(t, alarm.Technology)
		assert.Equal(t, string(model.TechNR), *alarm.Technology)
	}
}

func TestHandleAlarmEvent_UPSAlarmSourceBackfillsTechnology(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop())

	payload := AlarmPayload{
		DeviceID:        deviceID.String(),
		DeviceSN:        "SN-UPS-RCV-001",
		Carrier:         "cmcc",
		AlarmIdentifier: "42000",
		AlarmType:       "equipment",
		AlarmSource:     "UPS/FSU2024",
		Description:     "UPS alarm without technology payload",
		Severity:        2,
		RaisedAt:        time.Now(),
	}

	err := receiver.handleAlarmEvent(context.Background(), makeRcvEvent(t, payload))
	require.NoError(t, err)

	require.Len(t, store.active, 1)
	for _, alarm := range store.active {
		require.NotNil(t, alarm.Technology)
		assert.Equal(t, "UPS", *alarm.Technology)
	}
}

func TestHandleAlarmEvent_GenericInformPayloadAlarmInfo(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	deviceSN := "1202000534228JB0007"
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceBySN: map[string]*model.Device{
			deviceSN: {
				ID:           deviceID,
				SerialNumber: deviceSN,
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechNR,
				ProductClass: "GNB",
				DeviceName:   "gNB-7",
			},
		},
	})

	payload := informAlarmEventPayload{
		DeviceID: tr069.DeviceId{
			ProductClass: "GNB",
			SerialNumber: deviceSN,
		},
		Events:      []string{"101 ALARM"},
		CurrentTime: time.Date(2026, 5, 28, 13, 27, 27, 0, time.UTC),
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.NotificationType", Value: NotificationNewAlarm},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AlarmIdentifier", Value: "50003"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.PerceivedSeverity", Value: "Minor"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventType", Value: "equipment"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.SpecificProblem", Value: "时间同步失败告警"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventTime", Value: "2026-05-28T13:27:27Z"},
		},
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, store.active, 1)
	for _, alarm := range store.active {
		assert.Equal(t, deviceID, alarm.DeviceID)
		assert.Equal(t, deviceSN, alarm.DeviceSN)
		assert.Equal(t, model.CarrierCMCC, alarm.Carrier)
		assert.Equal(t, "50003", alarm.AlarmIdentifier)
		assert.Equal(t, "equipment", alarm.AlarmType)
		assert.Equal(t, "时间同步失败告警", alarm.Description)
		assert.Equal(t, model.AlarmMinor, alarm.Severity)
		require.NotNil(t, alarm.Technology)
		assert.Equal(t, string(model.TechNR), *alarm.Technology)
	}
}

func TestHandleAlarmEvent_UPSCurrentAlarmAddsSerialEquipmentInfo(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	bus := &rcvMockEventBus{}
	deviceID := uuid.New()
	deviceSN := "SN-UPS-ALARM-001"
	receiver := NewAlarmReceiver(engine, bus, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceBySN: map[string]*model.Device{
			deviceSN: {
				ID:           deviceID,
				SerialNumber: deviceSN,
				Carrier:      model.CarrierCMCC,
				ProductClass: "UPS_M3_BMU",
				DeviceName:   "UPS-1",
			},
		},
	})

	payload := informAlarmEventPayload{
		DeviceID: tr069.DeviceId{
			ProductClass: "UPS_M3_BMU",
			SerialNumber: deviceSN,
		},
		Events:      []string{"101 ALARM"},
		CurrentTime: time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC),
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.AlarmIdentifier", Value: "42000"},
			{Name: "InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.PerceivedSeverity", Value: "Major"},
			{Name: "InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.EventType", Value: "equipment"},
			{Name: "InternetGatewayDevice.FaultMgmt.CurrentAlarm.1.SpecificProblem", Value: "UPS battery fault"},
		},
	}

	err := receiver.handleAlarmEvent(context.Background(), makeRcvEvent(t, payload))
	require.NoError(t, err)

	require.Len(t, store.active, 1)
	for _, alarm := range store.active {
		assert.Equal(t, deviceID, alarm.DeviceID)
		assert.Equal(t, deviceSN, alarm.DeviceSN)
		assert.Equal(t, "42000", alarm.AlarmIdentifier)
		assert.Equal(t, "UPS_M3_BMU", *alarm.AlarmSource)
		require.NotNil(t, alarm.Technology)
		assert.Equal(t, "UPS", *alarm.Technology)
		assert.Equal(t, "SN="+deviceSN, alarm.AdditionalInfo[additionalInfoEquipmentInfo])
	}
	assert.Empty(t, bus.published[event.SubjectAlarmSyncRequested])
}

func TestHandleAlarmEvent_GenericInformPayload_OverridesSeverityFromDefinition(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	deviceSN := "1202000534228JB0008"
	registry := newTestAlarmDefRegistry(t, definition.ResolvedDefinition{
		AlarmDefinition: definition.AlarmDefinition{Identifier: "50003"},
		SeverityCode:    31001,
		SeverityName:    "Critical",
	})
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop()).
		WithDeviceReader(&rcvMockDeviceReader{
			deviceBySN: map[string]*model.Device{
				deviceSN: {
					ID:           deviceID,
					SerialNumber: deviceSN,
					Carrier:      model.CarrierCMCC,
					Technology:   model.TechNR,
					ProductClass: "GNB",
					DeviceName:   "gNB-8",
				},
			},
		}).
		WithAlarmDefRegistry(registry, nil)

	payload := informAlarmEventPayload{
		DeviceID: tr069.DeviceId{
			ProductClass: "GNB",
			SerialNumber: deviceSN,
		},
		Events:      []string{"101 ALARM"},
		CurrentTime: time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC),
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.NotificationType", Value: NotificationNewAlarm},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.AlarmIdentifier", Value: "50003"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.PerceivedSeverity", Value: "Minor"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventType", Value: "equipment"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.SpecificProblem", Value: "时间同步失败告警"},
			{Name: "Device.Services.FAPService.1.FAPControl.LTE.AlarmInfo.1.EventTime", Value: "2026-06-05T09:00:00Z"},
		},
	}

	err := receiver.handleAlarmEvent(context.Background(), makeRcvEvent(t, payload))
	require.NoError(t, err)

	require.Len(t, store.active, 1)
	for _, alarm := range store.active {
		assert.Equal(t, model.AlarmSeverity(31001), alarm.Severity)
	}
}

func TestHandleAlarmEvent_GenericInformPayloadExpeditedEvent(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	deviceID := uuid.New()
	deviceSN := "1202000534228JB0007"
	receiver := NewAlarmReceiver(engine, nil, zap.NewNop()).WithDeviceReader(&rcvMockDeviceReader{
		deviceBySN: map[string]*model.Device{
			deviceSN: {
				ID:           deviceID,
				SerialNumber: deviceSN,
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechNR,
				ProductClass: "FAP/BSC7041C243",
				DeviceName:   "gNB-7",
			},
		},
	})

	payload := informAlarmEventPayload{
		DeviceID: tr069.DeviceId{
			ProductClass: "FAP/BSC7041C243",
			SerialNumber: deviceSN,
		},
		Events:      []string{"101 ALARM"},
		CurrentTime: time.Date(2026, 5, 28, 13, 50, 38, 0, time.UTC),
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.NotificationType", Value: NotificationNewAlarm},
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.AlarmIdentifier", Value: "50003"},
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.PerceivedSeverity", Value: "Minor"},
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.EventType", Value: "Communications Alarm"},
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.ProbableCause", Value: "Time synchronization failed"},
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.SpecificProblem", Value: "specificProblem"},
			{Name: "Device.FaultMgmt.ExpeditedEvent.8.EventTime", Value: "2026-05-28T05:50:32Z"},
		},
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, store.active, 1)
	for _, alarm := range store.active {
		assert.Equal(t, deviceID, alarm.DeviceID)
		assert.Equal(t, deviceSN, alarm.DeviceSN)
		assert.Equal(t, model.CarrierCMCC, alarm.Carrier)
		assert.Equal(t, "50003", alarm.AlarmIdentifier)
		assert.Equal(t, "Communications Alarm", alarm.AlarmType)
		assert.Equal(t, "specificProblem", alarm.Description)
		assert.Equal(t, model.AlarmMinor, alarm.Severity)
		require.NotNil(t, alarm.ProbableCause)
		assert.Equal(t, "Time synchronization failed", *alarm.ProbableCause)
		require.NotNil(t, alarm.Technology)
		assert.Equal(t, string(model.TechNR), *alarm.Technology)
	}
}
