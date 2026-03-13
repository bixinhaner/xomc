package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
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
}

func (b *rcvMockEventBus) Publish(_ context.Context, _ string, _ event.Event) error { return nil }
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
func (b *rcvMockEventBus) Close() error { return nil }

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

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewAlarmReceiver(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, zap.NewNop())

	require.NotNil(t, receiver)
	assert.Equal(t, engine, receiver.engine)
}

func TestSubscribe_Success(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, zap.NewNop())

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
	receiver := NewAlarmReceiver(engine, zap.NewNop())

	bus := &rcvMockEventBus{subscribeErr: errors.New("connection refused")}
	err := receiver.Subscribe(bus)
	require.Error(t, err)
}

func TestHandleAlarmEvent_Success(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, zap.NewNop())

	deviceID := uuid.New()
	now := time.Now().Truncate(time.Second)

	payload := AlarmPayload{
		DeviceID:    deviceID.String(),
		DeviceSN:    "SN-RCV-001",
		Carrier:     "cmcc",
		AlarmCode:   "ALM_TEST_01",
		AlarmType:   "equipment",
		Description: "test alarm from receiver",
		Severity:    2,
		RaisedAt:    now,
		Additional:  map[string]string{"key": "value"},
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.NoError(t, err)

	assert.Len(t, store.active, 1)
	for _, alarm := range store.active {
		assert.Equal(t, deviceID, alarm.DeviceID)
		assert.Equal(t, "SN-RCV-001", alarm.DeviceSN)
		assert.Equal(t, model.CarrierCode("cmcc"), alarm.Carrier)
		assert.Equal(t, "ALM_TEST_01", alarm.AlarmCode)
	}
}

func TestHandleAlarmEvent_InvalidDeviceID(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	receiver := NewAlarmReceiver(engine, zap.NewNop())

	payload := AlarmPayload{
		DeviceID:  "not-a-valid-uuid",
		DeviceSN:  "SN-RCV-002",
		AlarmCode: "ALM_TEST_02",
		Severity:  1,
		RaisedAt:  time.Now(),
	}

	evt := makeRcvEvent(t, payload)
	err := receiver.handleAlarmEvent(context.Background(), evt)
	require.Error(t, err)
	assert.Len(t, store.active, 0)
}
