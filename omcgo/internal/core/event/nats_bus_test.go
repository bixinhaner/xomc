package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/reliability"
)

type queueStatsReaderStub struct {
	streamName string
	streamErr  error

	consumerInfo *nats.ConsumerInfo
	consumerErr  error

	streamInfo            *nats.StreamInfo
	streamInfoErr         error
	streamInfoCompletedAt time.Time
	streamInfoDelay       time.Duration

	nextMessage     *nats.RawStreamMsg
	nextErr         error
	nextStart       uint64
	nextSubject     string
	nextCalls       int
	nextCompletedAt time.Time

	calls int
}

func (s *queueStatsReaderStub) StreamNameBySubject(ctx context.Context, _ string) (string, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.streamName, s.streamErr
}

func (s *queueStatsReaderStub) ConsumerInfo(ctx context.Context, _, _ string) (*nats.ConsumerInfo, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.consumerInfo, s.consumerErr
}

func (s *queueStatsReaderStub) StreamInfo(ctx context.Context, _ string) (*nats.StreamInfo, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.streamInfoDelay > 0 {
		time.Sleep(s.streamInfoDelay)
	}
	s.streamInfoCompletedAt = time.Now()
	return s.streamInfo, s.streamInfoErr
}

func (s *queueStatsReaderStub) NextMessage(ctx context.Context, _ string, start uint64, subject string) (*nats.RawStreamMsg, error) {
	s.calls++
	s.nextCalls++
	s.nextStart = start
	s.nextSubject = subject
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.nextCompletedAt = time.Now()
	return s.nextMessage, s.nextErr
}

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

func TestNATSEventBusQueueStats(t *testing.T) {
	now := time.Now()
	firstRetainedAt := now.Add(-3 * time.Minute)
	tests := []struct {
		name          string
		consumer      *nats.ConsumerInfo
		stream        *nats.StreamInfo
		nextMessage   *nats.RawStreamMsg
		nextErr       error
		expectedStart uint64
		assert        func(t *testing.T, stats QueueStats)
	}{
		{
			name:     "empty stream has no oldest pending age",
			consumer: &nats.ConsumerInfo{},
			stream:   &nats.StreamInfo{State: nats.StreamState{}},
			assert: func(t *testing.T, stats QueueStats) {
				assert.Zero(t, stats.Pending)
				assert.Zero(t, stats.AckPending)
				assert.Zero(t, stats.Redelivered)
				assert.Zero(t, stats.OldestPendingAge)
				assert.Zero(t, stats.LastSequence)
				assert.Zero(t, stats.AckSequence)
			},
		},
		{
			name: "pending age ignores older unrelated shared stream messages",
			consumer: &nats.ConsumerInfo{
				NumPending:     7,
				NumAckPending:  2,
				NumRedelivered: 3,
				AckFloor:       nats.SequenceInfo{Consumer: 9, Stream: 11},
				Delivered:      nats.SequenceInfo{Consumer: 18, Stream: 19},
			},
			stream: &nats.StreamInfo{State: nats.StreamState{
				Msgs:      9,
				FirstSeq:  12,
				FirstTime: now.Add(-30 * time.Minute),
				LastSeq:   20,
			}},
			nextMessage:   &nats.RawStreamMsg{Subject: SubjectPMFileReceived, Sequence: 14, Time: firstRetainedAt},
			expectedStart: 12,
			assert: func(t *testing.T, stats QueueStats) {
				assert.Equal(t, uint64(7), stats.Pending)
				assert.Equal(t, 2, stats.AckPending)
				assert.Equal(t, 3, stats.Redelivered)
				assert.InDelta(t, 3*time.Minute, stats.OldestPendingAge, float64(250*time.Millisecond))
				assert.Equal(t, uint64(20), stats.LastSequence)
				assert.Equal(t, uint64(11), stats.AckSequence)
				assert.Equal(t, uint64(18), stats.DeliverySequence)
				assert.Equal(t, uint64(9), stats.AckConsumerSequence)
			},
		},
		{
			name: "advanced and deleted first sequence uses next matching retained message",
			consumer: &nats.ConsumerInfo{
				NumPending: 3,
				AckFloor:   nats.SequenceInfo{Stream: 4},
			},
			stream: &nats.StreamInfo{State: nats.StreamState{
				Msgs:      3,
				FirstSeq:  8,
				FirstTime: now.Add(-30 * time.Minute),
				LastSeq:   11,
				Deleted:   []uint64{9},
			}},
			nextMessage:   &nats.RawStreamMsg{Subject: SubjectPMFileReceived, Sequence: 10, Time: now.Add(-time.Minute)},
			expectedStart: 8,
			assert: func(t *testing.T, stats QueueStats) {
				assert.Equal(t, uint64(3), stats.Pending)
				assert.InDelta(t, time.Minute, stats.OldestPendingAge, float64(250*time.Millisecond))
				assert.Equal(t, uint64(11), stats.LastSequence)
				assert.Equal(t, uint64(4), stats.AckSequence)
			},
		},
		{
			name: "ack only uses consumer acknowledgement floor as the bounded lookup start",
			consumer: &nats.ConsumerInfo{
				NumAckPending: 1,
				AckFloor:      nats.SequenceInfo{Stream: 4},
				Delivered:     nats.SequenceInfo{Stream: 10},
			},
			stream:        &nats.StreamInfo{State: nats.StreamState{FirstSeq: 3, LastSeq: 10}},
			nextMessage:   &nats.RawStreamMsg{Subject: SubjectPMFileReceived, Sequence: 7, Time: now.Add(-2 * time.Minute)},
			expectedStart: 5,
			assert: func(t *testing.T, stats QueueStats) {
				assert.Zero(t, stats.Pending)
				assert.Equal(t, 1, stats.AckPending)
				assert.InDelta(t, 2*time.Minute, stats.OldestPendingAge, float64(250*time.Millisecond))
			},
		},
		{
			name:          "no matching retained message has zero age",
			consumer:      &nats.ConsumerInfo{NumPending: 1, Delivered: nats.SequenceInfo{Stream: 4}},
			stream:        &nats.StreamInfo{State: nats.StreamState{FirstSeq: 1, LastSeq: 8}},
			nextErr:       nats.ErrMsgNotFound,
			expectedStart: 5,
			assert: func(t *testing.T, stats QueueStats) {
				assert.Zero(t, stats.OldestPendingAge)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &queueStatsReaderStub{
				streamName:      "PM",
				consumerInfo:    tt.consumer,
				streamInfo:      tt.stream,
				streamInfoDelay: time.Millisecond,
				nextMessage:     tt.nextMessage,
				nextErr:         tt.nextErr,
			}
			bus := NewNATSEventBus(nil, nil, zap.NewNop())
			bus.queueStatsReader = reader

			stats, err := bus.QueueStats(context.Background(), SubjectPMFileReceived, "pm-workers")
			require.NoError(t, err)
			tt.assert(t, stats)
			assert.False(t, stats.SampledAt.Before(reader.streamInfoCompletedAt), "sample timestamp must be at or after collection completes")
			if reader.nextCalls > 0 {
				assert.False(t, stats.SampledAt.Before(reader.nextCompletedAt), "sample timestamp must be at or after the oldest-message lookup completes")
			}
			if tt.expectedStart > 0 {
				assert.Equal(t, tt.expectedStart, reader.nextStart)
				assert.Equal(t, SubjectPMFileReceived, reader.nextSubject)
			}
		})
	}
}

func TestNATSEventBusQueueStatsMissingConsumer(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())
	bus.queueStatsReader = &queueStatsReaderStub{
		streamName:  "PM",
		consumerErr: nats.ErrConsumerNotFound,
	}

	_, err := bus.QueueStats(context.Background(), SubjectPMFileReceived, "pm-workers")
	require.Error(t, err)
	assert.ErrorIs(t, err, nats.ErrConsumerNotFound)
	assert.Contains(t, err.Error(), "load queue consumer info")
}

func TestNATSEventBusQueueStatsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := &queueStatsReaderStub{}
	bus := NewNATSEventBus(nil, nil, zap.NewNop())
	bus.queueStatsReader = reader

	_, err := bus.QueueStats(ctx, SubjectPMFileReceived, "pm-workers")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, reader.calls, "a canceled context must not make NATS requests")
}

func TestNATSEventBusPendingCountUsesQueueStats(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())
	bus.queueStatsReader = &queueStatsReaderStub{
		streamName: "PM",
		consumerInfo: &nats.ConsumerInfo{
			NumPending:    7,
			NumAckPending: 2,
		},
		streamInfo: &nats.StreamInfo{},
	}

	count, err := bus.PendingCount(SubjectPMFileReceived, "pm-workers")
	require.NoError(t, err)
	assert.Equal(t, uint64(9), count)
}

func TestNATSEventBusQueueStatsDoesNotObservePMMetricsDirectly(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewEventBusMetrics(reg)
	bus := NewNATSEventBus(nil, nil, zap.NewNop())
	bus.SetMetrics(metrics)
	bus.queueStatsReader = &queueStatsReaderStub{
		streamName: "PM",
		consumerInfo: &nats.ConsumerInfo{
			NumPending: 5,
		},
		streamInfo: &nats.StreamInfo{},
	}

	_, err := bus.QueueStats(context.Background(), SubjectPMFileReceived, "pm-workers")
	require.NoError(t, err)
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	assert.Equal(t, "omc_pm_queue_sample_timestamp_seconds", families[0].GetName())
	assert.Zero(t, families[0].Metric[0].GetGauge().GetValue(),
		"QueueStats itself must not turn an initialized timestamp into a successful observation")
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

// --- Publish while disconnected ---

func TestNATSEventBus_Publish_DisconnectedReturnsError(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())

	evt, err := NewEvent("test.subject", map[string]string{"key": "value"})
	require.NoError(t, err)

	err = bus.Publish(context.Background(), "test.subject", evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection is not available")
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

	msgHandler := bus.wrapHandler(handler, maxDeliveries)
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
		SubjectPMFileDeferred,
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

func TestMaxDeliveriesForRetryHorizon(t *testing.T) {
	tests := []struct {
		name    string
		horizon time.Duration
		want    int
	}{
		{name: "no wait", horizon: 0, want: 1},
		{name: "first retry", horizon: time.Second, want: 2},
		{name: "thirty minute registration grace", horizon: 30 * time.Minute, want: 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MaxDeliveriesForRetryHorizon(tt.horizon))
		})
	}
}

func TestDecideAck_ExtremeDeliveries_BackoffCapped(t *testing.T) {
	// 极端 deliveries 不应触发位移溢出（shift cap=30）
	d := decideAck(errors.New("handler failed"), 100, 1000)
	assert.Equal(t, ackActionNak, d.action)
	// 1<<30 seconds 是个大但有限的值，不应是负数或 0
	assert.Greater(t, d.backoff, time.Duration(0))
}

func TestDecideAck_PermanentError_ReturnsTermRegardlessOfDeliveries(t *testing.T) {
	// 包装了 reliability.ErrPermanent 的明确不可恢复错误无论 deliveries 多少，
	// 都应立即 Term，不走正常的指数退避 Nak 重投。
	err := fmt.Errorf("device not found: %w", reliability.ErrPermanent)
	d := decideAck(err, 1, 5)
	assert.Equal(t, ackActionTerm, d.action)
	assert.Zero(t, d.backoff)
}

func TestPullTuningForSubject_DefaultsAndOverride(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())

	gpv := bus.pullTuningForSubject(SubjectCommandGetParamsResponse)
	assert.Equal(t, 2, gpv.Concurrency)
	assert.Equal(t, gpvPullBatchSize, gpv.BatchSize)
	assert.Equal(t, gpvPullAckWait, gpv.AckWait)
	assert.Equal(t, defaultPullMaxAckPending, gpv.MaxAckPending)

	gpvQueue := bus.queueTuningForSubject(SubjectCommandGetParamsResponse)
	assert.Equal(t, 30*time.Second, gpvQueue.AckWait)
	assert.Equal(t, 5, gpvQueue.MaxDeliver)
	assert.Equal(t, 2000, gpvQueue.MaxAckPending)

	paramSync := bus.pullTuningForSubject(SubjectParamSyncTaskResult)
	assert.Equal(t, paramSyncResultPullConcurrent, paramSync.Concurrency)
	assert.Equal(t, paramSyncResultPullAckWait, paramSync.AckWait)
	assert.Equal(t, paramSyncResultMaxAckPending, paramSync.MaxAckPending)

	bus.SetPullTuning(SubjectParamSyncTaskResult, PullTuning{BatchSize: 12, Concurrency: 3, AckWait: 45 * time.Second, MaxAckPending: 99})
	overridden := bus.pullTuningForSubject(SubjectParamSyncTaskResult)
	assert.Equal(t, 12, overridden.BatchSize)
	assert.Equal(t, 3, overridden.Concurrency)
	assert.Equal(t, 45*time.Second, overridden.AckWait)
	assert.Equal(t, 99, overridden.MaxAckPending)
}

func TestDurableDeliveryPlanNeverSkipsOrReplaysWhenMigratingFromAckFloor(t *testing.T) {
	plan := durableDeliveryPlan(nil, 416825)

	require.Equal(t, durableDeliveryFromSequence, plan.policy)
	require.Equal(t, uint64(416825), plan.startSequence)
}

func TestDurableDeliveryPlanBindsExistingConsumerWithoutResettingItsPosition(t *testing.T) {
	existing := &nats.ConsumerInfo{
		Name: "device-rpc-gpv",
		Config: nats.ConsumerConfig{
			Durable:       "device-rpc-gpv",
			DeliverPolicy: nats.DeliverByStartSequencePolicy,
			OptStartSeq:   416825,
		},
	}

	plan := durableDeliveryPlan(existing, 1)

	require.Equal(t, durableDeliveryBindExisting, plan.policy)
	require.Zero(t, plan.startSequence)
}

func TestDurableDeliveryPlanFreshConsumerStartsAtCurrentStreamTail(t *testing.T) {
	plan := durableDeliveryPlan(nil, 0)

	require.Equal(t, durableDeliveryNew, plan.policy)
	require.Zero(t, plan.startSequence)
}

func TestKeyedMaxAckPendingIsBoundedByDispatcherCapacity(t *testing.T) {
	require.Equal(t, 6, keyedMaxAckPending(2, 2, 2000))
	require.Equal(t, 4, keyedMaxAckPending(2, 2, 4))
	require.Equal(t, 1, keyedMaxAckPending(1, 0, 0))
}

func TestKeyedDispatcherRunsDifferentDevicesInParallelAndSameDeviceInOrder(t *testing.T) {
	dispatcher := newKeyedDispatcher(2, 2)
	t.Cleanup(dispatcher.Close)

	keyA, keyB := keysForDifferentShards(2)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	otherStarted := make(chan struct{})

	require.NoError(t, dispatcher.Submit(context.Background(), keyA, func() {
		close(firstStarted)
		<-releaseFirst
	}))
	<-firstStarted
	require.NoError(t, dispatcher.Submit(context.Background(), keyA, func() {
		close(secondStarted)
	}))
	require.NoError(t, dispatcher.Submit(context.Background(), keyB, func() {
		close(otherStarted)
	}))

	select {
	case <-otherStarted:
	case <-time.After(time.Second):
		t.Fatal("different-device work did not run in parallel")
	}
	select {
	case <-secondStarted:
		t.Fatal("same-device work ran out of order")
	default:
	}

	close(releaseFirst)
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("same-device second item did not run after the first")
	}
}

func TestKeyedDispatcherAppliesBoundedBackpressure(t *testing.T) {
	dispatcher := newKeyedDispatcher(1, 1)
	t.Cleanup(dispatcher.Close)

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	require.NoError(t, dispatcher.Submit(context.Background(), "SN-1", func() {
		close(firstStarted)
		<-releaseFirst
	}))
	<-firstStarted
	require.NoError(t, dispatcher.Submit(context.Background(), "SN-1", func() {}))

	thirdSubmitted := make(chan error, 1)
	go func() {
		thirdSubmitted <- dispatcher.Submit(context.Background(), "SN-1", func() {})
	}()
	select {
	case err := <-thirdSubmitted:
		t.Fatalf("third submit bypassed bounded queue: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	close(releaseFirst)
	select {
	case err := <-thirdSubmitted:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("blocked submit did not resume when capacity became available")
	}
}

func keysForDifferentShards(shards int) (string, string) {
	first := "SN-0"
	firstShard := keyedShardIndex(first, shards)
	for i := 1; i < 100; i++ {
		candidate := fmt.Sprintf("SN-%d", i)
		if keyedShardIndex(candidate, shards) != firstShard {
			return first, candidate
		}
	}
	panic("failed to find keys for distinct shards")
}

func TestUpdatedPullConsumerConfig_OverwritesMutableTuning(t *testing.T) {
	desired := PullTuning{
		BatchSize: 64, Concurrency: 64, AckWait: 2 * time.Minute,
		MaxDeliver: 7, MaxAckPending: 512,
	}
	existing := &nats.ConsumerInfo{
		Name:   "param-sync-results-pull",
		Config: nats.ConsumerConfig{AckWait: 30 * time.Second, MaxDeliver: 5, MaxAckPending: 2048},
	}

	got, changed := updatedPullConsumerConfig(existing, desired)

	require.True(t, changed)
	assert.Equal(t, "param-sync-results-pull", got.Durable)
	assert.Equal(t, 2*time.Minute, got.AckWait)
	assert.Equal(t, 7, got.MaxDeliver)
	assert.Equal(t, 512, got.MaxAckPending)
}

func TestUpdatedPullConsumerConfig_NoChangeWhenAlreadyAligned(t *testing.T) {
	desired := PullTuning{
		BatchSize: 64, Concurrency: 64, AckWait: 2 * time.Minute,
		MaxDeliver: 7, MaxAckPending: 512,
	}
	existing := &nats.ConsumerInfo{
		Name: "param-sync-results-pull",
		Config: nats.ConsumerConfig{
			Durable: "param-sync-results-pull", AckWait: 2 * time.Minute,
			MaxDeliver: 7, MaxAckPending: 512,
		},
	}

	got, changed := updatedPullConsumerConfig(existing, desired)

	require.False(t, changed)
	assert.Equal(t, existing.Config, got)
}

func TestQueueTuningForSubjectDefaultsAndOverride(t *testing.T) {
	bus := NewNATSEventBus(nil, nil, zap.NewNop())

	got := bus.queueTuningForSubject(SubjectPMFileReceived)
	assert.Equal(t, queueSubscribeAckWait, got.AckWait)
	assert.Equal(t, maxDeliveries, got.MaxDeliver)
	assert.Equal(t, defaultQueueMaxAckPending, got.MaxAckPending)

	bus.SetQueueTuning(SubjectPMFileReceived, QueueTuning{
		AckWait: 3 * time.Minute, MaxDeliver: 7, MaxAckPending: 16,
	})
	got = bus.queueTuningForSubject(SubjectPMFileReceived)
	assert.Equal(t, 3*time.Minute, got.AckWait)
	assert.Equal(t, 7, got.MaxDeliver)
	assert.Equal(t, 16, got.MaxAckPending)
}

func TestUpdatedQueueConsumerConfigOverwritesMutableTuning(t *testing.T) {
	desired := QueueTuning{AckWait: 2 * time.Minute, MaxDeliver: 5, MaxAckPending: 16}
	existing := &nats.ConsumerInfo{
		Name: "pm-workers",
		Config: nats.ConsumerConfig{
			AckWait: 30 * time.Second, MaxDeliver: -1, MaxAckPending: 1000,
		},
	}

	got, changed := updatedQueueConsumerConfig(existing, desired)

	require.True(t, changed)
	assert.Equal(t, "pm-workers", got.Durable)
	assert.Equal(t, desired.AckWait, got.AckWait)
	assert.Equal(t, desired.MaxDeliver, got.MaxDeliver)
	assert.Equal(t, desired.MaxAckPending, got.MaxAckPending)
}

func TestReconcilePullTuningWithExisting_NilKeepsDesired(t *testing.T) {
	desired := PullTuning{BatchSize: 64, Concurrency: 64, AckWait: 2 * time.Minute, MaxAckPending: 512}

	assert.Equal(t, desired, reconcilePullTuningWithExisting(desired, nil))
}

func TestReconcilePullTuningWithExisting_FallbackPreservesServerConsumerConfig(t *testing.T) {
	desired := PullTuning{
		BatchSize: 64, Concurrency: 64, AckWait: 2 * time.Minute,
		MaxDeliver: 7, MaxAckPending: 512,
	}
	existing := &nats.ConsumerInfo{Config: nats.ConsumerConfig{
		AckWait: 30 * time.Second, MaxDeliver: 3, MaxAckPending: 2048,
	}}

	got := reconcilePullTuningWithExisting(desired, existing)

	assert.Equal(t, desired.BatchSize, got.BatchSize)
	assert.Equal(t, desired.Concurrency, got.Concurrency)
	assert.Equal(t, 30*time.Second, got.AckWait)
	assert.Equal(t, 3, got.MaxDeliver)
	assert.Equal(t, 2048, got.MaxAckPending)
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
		"DeviceBootstrap":          SubjectDeviceBootstrap,
		"DevicePeriodic":           SubjectDevicePeriodic,
		"CommandGetParamsResponse": SubjectCommandGetParamsResponse,
		"TaskCompleted":            SubjectTaskCompleted,
		"PMFileReceived":           SubjectPMFileReceived,
		"AlarmRaised":              SubjectAlarmRaised,
		"ProvisionStarted":         SubjectProvisionStarted,
		"FirmwareUploaded":         SubjectFirmwareUploaded,
		"BackupTaskCreated":        SubjectBackupTaskCreated,
		"OSSAlarmForward":          SubjectOSSAlarmForward,
		"NEDirectRegister":         SubjectNEDirectRegister,
	}

	for name, subject := range subjects {
		assert.Contains(t, subject, ".", "%s subject should be dot-separated: %s", name, subject)
	}
}

// --- pullDurableName tests ---

func TestPullDurableName_EmptyQueue_ReturnsPull(t *testing.T) {
	assert.Equal(t, "pull", pullDurableName(""))
}

func TestPullDurableName_AlreadyHasSuffix_ReturnsSame(t *testing.T) {
	assert.Equal(t, "provision-gpv-pull", pullDurableName("provision-gpv-pull"))
}

func TestPullDurableName_AppendsSuffix(t *testing.T) {
	assert.Equal(t, "provision-gpv-pull", pullDurableName("provision-gpv"))
}

func TestPullDurableName_ShortName(t *testing.T) {
	assert.Equal(t, "foo-pull", pullDurableName("foo"))
}

// --- defaultPullTuningForSubject tests ---

func TestDefaultPullTuningForSubject_GPVSubject_Returns64(t *testing.T) {
	assert.Equal(t, gpvPullBatchSize, defaultPullTuningForSubject(SubjectCommandGetParamsResponse).BatchSize)
}

func TestDefaultPullTuningForSubject_OtherSubject_Returns32(t *testing.T) {
	assert.Equal(t, 32, defaultPullTuningForSubject("some.other.subject").BatchSize)
}

func TestPullFetchBatchForAvailableSlots_LimitsFetchToFreeConcurrency(t *testing.T) {
	assert.Equal(t, 64, pullFetchBatchForAvailableSlots(64, 64, 0))
	assert.Equal(t, 1, pullFetchBatchForAvailableSlots(64, 64, 63))
	assert.Equal(t, 0, pullFetchBatchForAvailableSlots(64, 64, 64))
	assert.Equal(t, 8, pullFetchBatchForAvailableSlots(8, 64, 1))
}

func TestPullFetchBatchForAvailableSlots_InvalidInputsReturnZero(t *testing.T) {
	assert.Equal(t, 0, pullFetchBatchForAvailableSlots(0, 64, 0))
	assert.Equal(t, 0, pullFetchBatchForAvailableSlots(64, 0, 0))
	assert.Equal(t, 0, pullFetchBatchForAvailableSlots(64, 64, 99))
}
