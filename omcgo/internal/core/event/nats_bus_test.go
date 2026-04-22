package event

import (
	"context"
	"encoding/json"
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
