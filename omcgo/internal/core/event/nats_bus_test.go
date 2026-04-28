package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Constructor tests ---

func TestNewNATSEventBus_ReturnsNonNil(t *testing.T) {
	// NewNATSEventBus accepts nil conn/js for constructor — it's the caller's
	// responsibility to pass valid connections. Verify struct is populated.
	logger := zap.NewNop()
	bus := NewNATSEventBus(nil, nil, logger)

	require.NotNil(t, bus)
	assert.Nil(t, bus.conn)
	assert.Nil(t, bus.js)
	assert.NotNil(t, bus.logger)
	assert.Nil(t, bus.subs)
}

// --- Event serialization round-trip ---

func TestEvent_JSONRoundTrip(t *testing.T) {
	type payload struct {
		DeviceID string `json:"device_id"`
		Status   string `json:"status"`
	}

	original, err := NewEvent("device.registered", payload{
		DeviceID: "dev-001",
		Status:   "active",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, original.ID)
	assert.Equal(t, "device.registered", original.Subject)
	assert.False(t, original.Timestamp.IsZero())

	// Marshal to JSON (as Publish would do)
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal (as wrapHandler would do)
	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Subject, decoded.Subject)
	assert.WithinDuration(t, original.Timestamp, decoded.Timestamp, time.Second)

	// Decode payload
	var p payload
	err = decoded.DecodePayload(&p)
	require.NoError(t, err)
	assert.Equal(t, "dev-001", p.DeviceID)
	assert.Equal(t, "active", p.Status)
}

func TestEvent_JSONRoundTrip_NilPayload(t *testing.T) {
	evt, err := NewEvent("test.subject", nil)
	require.NoError(t, err)

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "test.subject", decoded.Subject)
}

func TestEvent_JSONRoundTrip_WithMetadata(t *testing.T) {
	evt := Event{
		ID:      "test-id",
		Subject: "alarm.raised",
		Payload: json.RawMessage(`{"severity":"critical"}`),
		Metadata: map[string]string{
			"carrier": "cmcc",
			"region":  "beijing",
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "cmcc", decoded.Metadata["carrier"])
	assert.Equal(t, "beijing", decoded.Metadata["region"])
}

// --- Close with no subscriptions ---

func TestNATSEventBus_Close_NoSubscriptions(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())
	err := bus.Close()
	assert.NoError(t, err)
	assert.Nil(t, bus.subs)
}

// --- Publish with nil JetStream ---

func TestNATSEventBus_Publish_NilJS_ReturnsError(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())

	evt, err := NewEvent("test.subject", map[string]string{"key": "value"})
	require.NoError(t, err)

	// Publish should fail because js is nil — this will panic or error
	// depending on implementation. Since js.Publish is called on nil,
	// this verifies the bus doesn't silently swallow the issue.
	assert.Panics(t, func() {
		bus.Publish(context.Background(), "test.subject", evt)
	})
}

// --- Subscribe with nil JetStream ---

func TestNATSEventBus_Subscribe_NilJS_ReturnsError(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())

	// Subscribe should fail because js is nil
	assert.Panics(t, func() {
		bus.Subscribe("test.subject", func(ctx context.Context, evt Event) error {
			return nil
		})
	})
}

// --- QueueSubscribe with nil JetStream ---

func TestNATSEventBus_QueueSubscribe_NilJS_Panics(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())

	// QueueSubscribe should panic because js is nil
	assert.Panics(t, func() {
		_, _ = bus.QueueSubscribe("test.subject", "test-queue", func(ctx context.Context, evt Event) error {
			return nil
		})
	})
}

// --- ackAction enum sanity ---

func TestAckAction_Distinct(t *testing.T) {
	// Verify enum values are distinct so wrapHandler switch works correctly.
	actions := map[ackAction]string{
		ackActionAck:  "ack",
		ackActionNak:  "nak",
		ackActionTerm: "term",
	}
	assert.Len(t, actions, 3, "all three actions should be distinct")
}

// --- wrapHandler smoke ---
//
// wrapHandler 返回的闭包接收 *nats.Msg，构造真实 nats.Msg 需要 NATS 连接，
// 这里仅验证 wrapHandler 在 nil handler / 各类 EventHandler 输入下不 panic 地
// 返回函数（而不是测试其消息处理流程，那需要 embed NATS server）。

func TestWrapHandler_ReturnsNonNilMsgHandler(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())
	handler := func(ctx context.Context, evt Event) error { return nil }

	msgHandler := bus.wrapHandler(handler)
	assert.NotNil(t, msgHandler)
}

// --- wrapHandler logic ---
// The wrapHandler method is tightly coupled to nats.Msg, which requires a
// real NATS connection to construct properly. The following tests verify
// the handler wrapping logic indirectly through event serialization.
//
// Integration tests with an embedded NATS server would be needed to fully test:
// - Message acknowledgement (Ack/Nak/Term)
// - Exponential backoff retry (NakWithDelay)
// - Max delivery termination (maxDeliveries = 5)
// - Queue subscription load balancing
// - Subject pattern matching with NATS wildcards

func TestWrapHandler_UnmarshalLogic(t *testing.T) {
	// Verify that the JSON format used by Publish is compatible
	// with what wrapHandler expects to unmarshal.
	type testPayload struct {
		Name string `json:"name"`
	}

	evt, err := NewEvent("test.wrap", testPayload{Name: "hello"})
	require.NoError(t, err)

	// Simulate what Publish does
	evt.Subject = "test.wrap"
	data, err := json.Marshal(evt)
	require.NoError(t, err)

	// Simulate what wrapHandler does (unmarshal)
	var received Event
	err = json.Unmarshal(data, &received)
	require.NoError(t, err)

	assert.Equal(t, evt.ID, received.ID)
	assert.Equal(t, "test.wrap", received.Subject)

	var p testPayload
	err = received.DecodePayload(&p)
	require.NoError(t, err)
	assert.Equal(t, "hello", p.Name)
}

func TestMaxDeliveries_Constant(t *testing.T) {
	// Verify the constant is set to expected value
	assert.Equal(t, 5, maxDeliveries)
}

// --- natsSubscription ---
// natsSubscription wraps *nats.Subscription and delegates Unsubscribe.
// Testing requires a real NATS connection, so this is noted for integration tests.

// --- Subject constants validation ---

func TestSubjectConstants_NotEmpty(t *testing.T) {
	subjects := []string{
		SubjectDeviceBootstrap,
		SubjectDevicePeriodic,
		SubjectDeviceValueChange,
		SubjectDeviceAlarm,
		SubjectDeviceRegistered,
		SubjectCommandGetParamsResponse,
		SubjectCommandSetParamsResponse,
		SubjectTaskCompleted,
		SubjectTaskFailed,
		SubjectPMFileReceived,
		SubjectPMFileParsed,
		SubjectAlarmRaised,
		SubjectAlarmCleared,
		SubjectAlarmAcknowledged,
		SubjectProvisionStarted,
		SubjectProvisionCompleted,
		SubjectProvisionFailed,
	}

	for _, s := range subjects {
		assert.NotEmpty(t, s, "subject constant should not be empty")
	}
}

// --- decideAck pure-function tests ---
//
// decideAck 是 wrapHandler 的纯逻辑核心，无 NATS 依赖，可独立测试 Ack/Nak/Term
// 决策与指数退避计算的所有分支。

func TestDecideAck_NoError_ReturnsAck(t *testing.T) {
	d := decideAck(nil, 1, 5)
	assert.Equal(t, ackActionAck, d.action)
	assert.Zero(t, d.backoff)
}

func TestDecideAck_ErrorBelowMax_ReturnsNakWithBackoff(t *testing.T) {
	tests := []struct {
		name        string
		deliveries  uint64
		wantBackoff time.Duration
	}{
		{"delivery_1_backoff_1s", 1, 1 * time.Second},
		{"delivery_2_backoff_2s", 2, 2 * time.Second},
		{"delivery_3_backoff_4s", 3, 4 * time.Second},
		{"delivery_4_backoff_8s", 4, 8 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := decideAck(errors.New("handler failed"), tt.deliveries, 5)
			assert.Equal(t, ackActionNak, d.action)
			assert.Equal(t, tt.wantBackoff, d.backoff)
		})
	}
}

func TestDecideAck_ErrorAtMax_ReturnsTerm(t *testing.T) {
	d := decideAck(errors.New("handler failed"), 5, 5)
	assert.Equal(t, ackActionTerm, d.action)
	assert.Zero(t, d.backoff)
}

func TestDecideAck_ErrorAboveMax_ReturnsTerm(t *testing.T) {
	d := decideAck(errors.New("handler failed"), 100, 5)
	assert.Equal(t, ackActionTerm, d.action)
}

func TestDecideAck_ZeroDelivery_TreatedAsOne(t *testing.T) {
	// 防御性：deliveries=0 也走 1 次的 backoff（避免 1<<-1 溢出）
	d := decideAck(errors.New("handler failed"), 0, 5)
	assert.Equal(t, ackActionNak, d.action)
	assert.Equal(t, 1*time.Second, d.backoff)
}

func TestDecideAck_ExtremeDeliveries_BackoffCapped(t *testing.T) {
	// 极端 deliveries 不应触发位移溢出（shift cap=30）
	d := decideAck(errors.New("handler failed"), 100, 1000)
	assert.Equal(t, ackActionNak, d.action)
	// 1<<30 seconds 是个大但有限的值，不应是负数或 0
	assert.Greater(t, d.backoff, time.Duration(0))
}

// --- decodeEventBytes pure-function tests ---

func TestDecodeEventBytes_ValidJSON(t *testing.T) {
	original := Event{
		ID:        "test-id",
		Subject:   "device.inform.bootstrap",
		Payload:   json.RawMessage(`{"sn":"ABC123"}`),
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(original)
	require.NoError(t, err)

	decoded, err := decodeEventBytes(data)
	require.NoError(t, err)
	assert.Equal(t, "test-id", decoded.ID)
	assert.Equal(t, "device.inform.bootstrap", decoded.Subject)
}

func TestDecodeEventBytes_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := decodeEventBytes([]byte("not-valid-json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal event")
}

func TestDecodeEventBytes_EmptyBytes_ReturnsError(t *testing.T) {
	_, err := decodeEventBytes([]byte(""))
	require.Error(t, err)
}

func TestSubjectConstants_DotSeparated(t *testing.T) {
	// All subjects should follow dot-separated naming convention
	subjects := map[string]string{
		"DeviceBootstrap":     SubjectDeviceBootstrap,
		"DevicePeriodic":      SubjectDevicePeriodic,
		"CommandGetParamsResponse": SubjectCommandGetParamsResponse,
		"TaskCompleted":            SubjectTaskCompleted,
		"PMFileReceived":      SubjectPMFileReceived,
		"AlarmRaised":         SubjectAlarmRaised,
		"ProvisionStarted":    SubjectProvisionStarted,
		"FirmwareUploaded":    SubjectFirmwareUploaded,
		"BackupTaskCreated":   SubjectBackupTaskCreated,
		"OSSAlarmForward":     SubjectOSSAlarmForward,
		"NEDirectRegister":    SubjectNEDirectRegister,
	}

	for name, subject := range subjects {
		assert.Contains(t, subject, ".", "%s subject should be dot-separated: %s", name, subject)
	}
}
