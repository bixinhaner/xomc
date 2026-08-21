package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/core/reliability"
)

func TestQueueSubscribeHonorsConfiguredMaxDeliver(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("QUEUE_MAX_DELIVER_%d", suffix)
	subject := fmt.Sprintf("test.queue.max-deliver.%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	core, observed := observer.New(zap.ErrorLevel)
	bus := NewNATSEventBus(nc, js, zap.New(core))
	t.Cleanup(func() { _ = bus.Close() })
	bus.SetQueueTuning(subject, QueueTuning{
		AckWait:       2 * time.Second,
		MaxDeliver:    2,
		MaxAckPending: 16,
	})

	var attempts atomic.Int64
	_, err = bus.QueueSubscribe(subject, fmt.Sprintf("queue-max-deliver-%d", suffix),
		func(context.Context, Event) error {
			attempts.Add(1)
			return fmt.Errorf("registration pending: %w", reliability.ErrDeferred)
		})
	require.NoError(t, err)

	evt, err := NewEvent(subject, map[string]string{"device_sn": "SN-DEFERRED"})
	require.NoError(t, err)
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return attempts.Load() == 2
	}, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		return observed.FilterMessage("handle event (terminating)").Len() == 1
	}, 2*time.Second, 20*time.Millisecond,
		"QueueSubscribe settlement must use the durable consumer MaxDeliver")
	require.Zero(t, observed.FilterMessage("handle event (retrying)").Len(),
		"expected deferred redelivery must not emit error stacktraces")
}

func TestQueueSubscribeClosePreservesDurableConsumer(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("QUEUE_RESTART_%d", suffix)
	subject := fmt.Sprintf("test.queue.restart.%d", suffix)
	durable := fmt.Sprintf("queue-restart-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	_, err = bus.QueueSubscribe(subject, durable, func(context.Context, Event) error {
		return nil
	})
	require.NoError(t, err)
	_, err = js.ConsumerInfo(stream, durable)
	require.NoError(t, err)

	require.NoError(t, bus.Close())
	_, err = js.ConsumerInfo(stream, durable)
	require.NoError(t, err,
		"graceful service shutdown must detach from, rather than delete, a durable consumer")
}

func TestKeyedQueueSubscribeClosePreservesDurableConsumer(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("KEYED_QUEUE_RESTART_%d", suffix)
	subject := fmt.Sprintf("test.keyed-queue.restart.%d", suffix)
	durable := fmt.Sprintf("keyed-queue-restart-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	_, err = bus.KeyedQueueSubscribe(subject, KeyedQueueConfig{
		Durable:       durable,
		Concurrency:   1,
		QueueDepth:    1,
		AckWait:       time.Second,
		MaxDeliver:    3,
		MaxAckPending: 2,
	}, func(evt Event) (string, error) {
		return evt.ID, nil
	}, func(context.Context, Event) error {
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, bus.Close())
	_, err = js.ConsumerInfo(stream, durable)
	require.NoError(t, err,
		"graceful service shutdown must preserve a keyed durable consumer")
}

func TestPullSubscribeClosePreservesDurableConsumer(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("PULL_RESTART_%d", suffix)
	subject := fmt.Sprintf("test.pull.restart.%d", suffix)
	queue := fmt.Sprintf("pull-restart-%d", suffix)
	durable := pullDurableName(queue)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	_, err = bus.PullSubscribe(subject, queue, func(context.Context, Event) error {
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, bus.Close())
	_, err = js.ConsumerInfo(stream, durable)
	require.NoError(t, err,
		"graceful service shutdown must preserve a pull durable consumer")
}

func TestQueueSubscribeDeferredLaneDoesNotBlockMainConsumer(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("PM_DEFERRED_LANE_%d", suffix)
	mainSubject := fmt.Sprintf("test.pm.received.%d", suffix)
	deferredSubject := fmt.Sprintf("test.pm.deferred.%d", suffix)
	mainDurable := fmt.Sprintf("pm-main-%d", suffix)
	deferredDurable := fmt.Sprintf("pm-wait-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{mainSubject, deferredSubject},
		Storage:   nats.MemoryStorage,
		Retention: nats.WorkQueuePolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	tuning := QueueTuning{
		AckWait:       2 * time.Second,
		MaxDeliver:    3,
		MaxAckPending: 1,
	}
	bus.SetQueueTuning(mainSubject, tuning)
	bus.SetQueueTuning(deferredSubject, tuning)

	var deferredAttempts atomic.Int64
	_, err = bus.QueueSubscribe(deferredSubject, deferredDurable,
		func(context.Context, Event) error {
			deferredAttempts.Add(1)
			return fmt.Errorf("registration pending: %w", reliability.ErrDeferred)
		})
	require.NoError(t, err)

	var healthyProcessed atomic.Int64
	_, err = bus.QueueSubscribe(mainSubject, mainDurable,
		func(ctx context.Context, evt Event) error {
			var payload struct {
				WaitForRegistration bool `json:"wait_for_registration"`
			}
			if decodeErr := evt.DecodePayload(&payload); decodeErr != nil {
				return decodeErr
			}
			if payload.WaitForRegistration {
				return bus.Publish(ctx, deferredSubject, evt)
			}
			healthyProcessed.Add(1)
			return nil
		})
	require.NoError(t, err)

	for _, wait := range []bool{true, false} {
		evt, eventErr := NewEvent(mainSubject, map[string]bool{
			"wait_for_registration": wait,
		})
		require.NoError(t, eventErr)
		require.NoError(t, bus.Publish(context.Background(), mainSubject, evt))
	}

	require.Eventually(t, func() bool {
		return healthyProcessed.Load() == 1 && deferredAttempts.Load() >= 1
	}, 3*time.Second, 20*time.Millisecond,
		"the deferred event may fill its wait consumer but must not block the next main event")
	require.Eventually(t, func() bool {
		mainInfo, infoErr := js.ConsumerInfo(stream, mainDurable)
		return infoErr == nil && mainInfo.NumPending == 0 && mainInfo.NumAckPending == 0
	}, 2*time.Second, 20*time.Millisecond)
	deferredInfo, err := js.ConsumerInfo(stream, deferredDurable)
	require.NoError(t, err)
	require.Equal(t, 1, deferredInfo.NumAckPending)
}

func TestKeyedQueueDurableHandoffDoesNotSkipOrReplay(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_HANDOFF_%d", suffix)
	subject := fmt.Sprintf("test.gpv.handoff.%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	for sequence := 1; sequence <= 4; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": fmt.Sprintf("SN-%d", sequence),
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	var (
		firstMu  sync.Mutex
		firstGot []int
	)
	firstSub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable:       fmt.Sprintf("gpv-rpc-%d", suffix),
			StartSequence: 2,
			Concurrency:   2,
			QueueDepth:    4,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			MaxAckPending: 2000,
		},
		func(evt Event) (string, error) {
			var payload struct {
				DeviceSN string `json:"device_sn"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				return "", err
			}
			return payload.DeviceSN, nil
		},
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				return err
			}
			firstMu.Lock()
			firstGot = append(firstGot, payload.Sequence)
			firstMu.Unlock()
			return nil
		},
	)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		firstMu.Lock()
		defer firstMu.Unlock()
		return len(firstGot) == 3
	}, 5*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		info, infoErr := js.ConsumerInfo(stream, fmt.Sprintf("gpv-rpc-%d", suffix))
		return infoErr == nil && info.AckFloor.Stream == 4
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, firstSub.Unsubscribe())
	// Unsubscribe 只把协议命令写入客户端缓冲；复用同一 durable 前必须等待服务端
	// 确认，否则新消息可能仍被投递到旧 inbox，直到 AckWait 后才重投，造成测试竞态。
	require.NoError(t, nc.Flush())
	firstMu.Lock()
	require.ElementsMatch(t, []int{2, 3, 4}, firstGot)
	firstMu.Unlock()

	var secondGot atomic.Int64
	secondSub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable:       fmt.Sprintf("gpv-rpc-%d", suffix),
			StartSequence: 1,
			Concurrency:   2,
			QueueDepth:    4,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			MaxAckPending: 2000,
		},
		func(evt Event) (string, error) {
			var payload struct {
				DeviceSN string `json:"device_sn"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				return "", err
			}
			return payload.DeviceSN, nil
		},
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int64 `json:"sequence"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				return err
			}
			secondGot.Store(payload.Sequence)
			return nil
		},
	)
	require.NoError(t, err)
	// QueueSubscribe 创建本地 inbox 订阅后再绑定既有 durable；发布切换窗口探针前
	// 等待服务端确认新 inbox 已生效，避免发布与订阅命令竞争。
	require.NoError(t, nc.Flush())
	evt, err := NewEvent(subject, map[string]any{"device_sn": "SN-5", "sequence": 5})
	require.NoError(t, err)
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return secondGot.Load() == 5 }, 5*time.Second, 10*time.Millisecond)
	require.Never(t, func() bool {
		got := secondGot.Load()
		return got > 0 && got != 5
	}, 100*time.Millisecond, 10*time.Millisecond)
	require.NoError(t, secondSub.Unsubscribe())
}

func TestKeyedQueueHandlerFailureNaksThenSuccessAcks(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_ACK_%d", suffix)
	subject := fmt.Sprintf("test.gpv.ack.%d", suffix)
	durable := fmt.Sprintf("gpv-rpc-ack-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	evt, err := NewEvent(subject, map[string]any{"device_sn": "SN-RETRY"})
	require.NoError(t, err)
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)

	reg := prometheus.NewRegistry()
	metrics := NewEventBusMetrics(reg)
	bus := NewNATSEventBus(nc, js, zap.NewNop())
	bus.SetMetrics(metrics)
	t.Cleanup(func() { _ = bus.Close() })
	var attempts atomic.Int64
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable:       durable,
			StartSequence: 1,
			Concurrency:   2,
			QueueDepth:    4,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			MaxAckPending: 2000,
		},
		func(Event) (string, error) { return "SN-RETRY", nil },
		func(context.Context, Event) error {
			if attempts.Add(1) == 1 {
				return errors.New("transient database failure")
			}
			return nil
		},
	)
	require.NoError(t, err)
	require.Eventuallyf(t, func() bool { return attempts.Load() >= 2 }, 5*time.Second, 20*time.Millisecond,
		"handler attempts never reached the successful retry; attempts=%d", attempts.Load())
	require.Eventually(t, func() bool {
		info, infoErr := js.ConsumerInfo(stream, durable)
		return infoErr == nil &&
			info.NumPending == 0 &&
			info.NumAckPending == 0 &&
			info.AckFloor.Consumer >= 1
	}, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, int64(2), attempts.Load(), "one transient failure must produce exactly one in-lane retry")
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DeliveryTotal.WithLabelValues(subject, deliveryOutcomeNak)),
		"one transient handler failure must be observable as one delivery retry")
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.DeliveryTotal.WithLabelValues(subject, deliveryOutcomeAck)),
		"the final successful delivery must be acknowledged exactly once")
	stats, err := bus.QueueStats(context.Background(), subject, durable)
	require.NoError(t, err)
	require.Zero(t, stats.AckGap, "ack rate must catch up to delivered rate after retry success")
	require.GreaterOrEqual(t, stats.DeliverySequence, stats.AckConsumerSequence)
	require.NoError(t, sub.Unsubscribe())
}

func TestKeyedQueueTransientHeadFailureDoesNotReleaseSameDeviceFollower(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_PUSH_HEAD_%d", suffix)
	subject := fmt.Sprintf("test.gpv.push.head.%d", suffix)
	durable := fmt.Sprintf("gpv-rpc-head-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	for sequence := 1; sequence <= 2; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": "SN-HEAD",
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	var (
		mu       sync.Mutex
		order    []int
		attempts = make(map[int]int)
	)
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable:       durable,
			StartSequence: 1,
			Concurrency:   1,
			QueueDepth:    2,
			AckWait:       300 * time.Millisecond,
			MaxDeliver:    3,
			MaxAckPending: 3,
		},
		func(Event) (string, error) { return "SN-HEAD", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			require.NoError(t, evt.DecodePayload(&payload))
			mu.Lock()
			attempts[payload.Sequence]++
			order = append(order, payload.Sequence)
			attempt := attempts[payload.Sequence]
			mu.Unlock()
			if payload.Sequence == 1 && attempt == 1 {
				return errors.New("transient database failure")
			}
			return nil
		},
	)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 3
	}, 5*time.Second, 20*time.Millisecond)
	mu.Lock()
	require.Equal(t, []int{1, 1, 2}, order)
	mu.Unlock()
	require.NoError(t, sub.Unsubscribe())
}

func TestKeyedQueueTerminalHeadFailureExhaustsMaxDeliverBeforeFollower(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_PUSH_TERM_%d", suffix)
	subject := fmt.Sprintf("test.gpv.push.term.%d", suffix)
	durable := fmt.Sprintf("gpv-rpc-term-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	for sequence := 1; sequence <= 2; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": "SN-TERM",
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	var (
		mu    sync.Mutex
		order []int
	)
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable: durable, StartSequence: 1, Concurrency: 1, QueueDepth: 2,
			AckWait: 300 * time.Millisecond, MaxDeliver: 3, MaxAckPending: 3,
		},
		func(Event) (string, error) { return "SN-TERM", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			require.NoError(t, evt.DecodePayload(&payload))
			mu.Lock()
			order = append(order, payload.Sequence)
			mu.Unlock()
			if payload.Sequence == 1 {
				return errors.New("persistent database failure")
			}
			return nil
		},
	)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 4
	}, 5*time.Second, 20*time.Millisecond)
	mu.Lock()
	require.Equal(t, []int{1, 1, 1, 2}, order)
	mu.Unlock()
	require.Eventually(t, func() bool {
		info, infoErr := js.ConsumerInfo(stream, durable)
		return infoErr == nil && info.NumPending == 0 && info.NumAckPending == 0
	}, 5*time.Second, 20*time.Millisecond)
	require.NoError(t, sub.Unsubscribe())
}

func TestKeyedQueueSlowBacklogKeepsAckAliveAndSameDeviceFIFO(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_SLOW_%d", suffix)
	subject := fmt.Sprintf("test.gpv.slow.%d", suffix)
	durable := fmt.Sprintf("gpv-rpc-slow-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	for sequence := 1; sequence <= 3; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": "SN-SLOW",
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}

	const ackWait = 500 * time.Millisecond
	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	var (
		mu       sync.Mutex
		order    []int
		attempts = make(map[int]int)
	)
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable:       durable,
			StartSequence: 1,
			Concurrency:   1,
			QueueDepth:    2,
			AckWait:       ackWait,
			MaxDeliver:    5,
			MaxAckPending: 2000,
		},
		func(Event) (string, error) { return "SN-SLOW", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			require.NoError(t, evt.DecodePayload(&payload))
			mu.Lock()
			attempts[payload.Sequence]++
			order = append(order, payload.Sequence)
			mu.Unlock()
			if payload.Sequence == 1 {
				time.Sleep(3 * ackWait)
			}
			return nil
		},
	)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 3
	}, 5*time.Second, 20*time.Millisecond)

	mu.Lock()
	require.Equal(t, []int{1, 2, 3}, order)
	require.Equal(t, map[int]int{1: 1, 2: 1, 3: 1}, attempts)
	mu.Unlock()
	info, err := js.ConsumerInfo(stream, durable)
	require.NoError(t, err)
	require.Equal(t, 3, info.Config.MaxAckPending,
		"server in-flight must equal one running plus two queued shard slots")
	require.Zero(t, info.NumRedelivered)
	require.NoError(t, sub.Unsubscribe())
}

func TestKeyedPullAdjacentFetchPreservesSameDeviceFIFO(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_PULL_ORDER_%d", suffix)
	subject := fmt.Sprintf("test.gpv.pull.order.%d", suffix)
	queue := fmt.Sprintf("gpv-provision-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	bus.SetPullTuning(subject, PullTuning{
		BatchSize: 6, Concurrency: 2, AckWait: time.Second, MaxAckPending: 100,
	})
	t.Cleanup(func() { _ = bus.Close() })
	var (
		mu    sync.Mutex
		order []int
	)
	sub, err := bus.KeyedPullSubscribe(
		subject,
		queue,
		4,
		func(Event) (string, error) { return "SN-ORDERED", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			require.NoError(t, evt.DecodePayload(&payload))
			if payload.Sequence == 1 {
				time.Sleep(100 * time.Millisecond)
			}
			mu.Lock()
			order = append(order, payload.Sequence)
			mu.Unlock()
			return nil
		},
	)
	require.NoError(t, err)

	for sequence := 1; sequence <= 6; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": "SN-ORDERED",
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 6
	}, 5*time.Second, 20*time.Millisecond)
	mu.Lock()
	require.Equal(t, []int{1, 2, 3, 4, 5, 6}, order)
	mu.Unlock()
	info, err := js.ConsumerInfo(stream, pullDurableName(queue))
	require.NoError(t, err)
	require.Equal(t, 10, info.Config.MaxAckPending)
	require.Zero(t, info.NumRedelivered)
	require.NoError(t, sub.Unsubscribe())
}

func TestKeyedPullTransientHeadFailureDoesNotReleaseSameDeviceFollower(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_PULL_HEAD_%d", suffix)
	subject := fmt.Sprintf("test.gpv.pull.head.%d", suffix)
	queue := fmt.Sprintf("gpv-provision-head-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	bus.SetPullTuning(subject, PullTuning{
		BatchSize: 2, Concurrency: 1, AckWait: 300 * time.Millisecond,
		MaxDeliver: 3, MaxAckPending: 2,
	})
	t.Cleanup(func() { _ = bus.Close() })
	var (
		mu       sync.Mutex
		order    []int
		attempts = make(map[int]int)
	)
	sub, err := bus.KeyedPullSubscribe(
		subject,
		queue,
		2,
		func(Event) (string, error) { return "SN-HEAD", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			require.NoError(t, evt.DecodePayload(&payload))
			mu.Lock()
			attempts[payload.Sequence]++
			order = append(order, payload.Sequence)
			attempt := attempts[payload.Sequence]
			mu.Unlock()
			if payload.Sequence == 1 && attempt == 1 {
				return errors.New("transient database failure")
			}
			return nil
		},
	)
	require.NoError(t, err)

	for sequence := 1; sequence <= 2; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": "SN-HEAD",
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 3
	}, 5*time.Second, 20*time.Millisecond)
	mu.Lock()
	require.Equal(t, []int{1, 1, 2}, order)
	mu.Unlock()
	info, err := js.ConsumerInfo(stream, pullDurableName(queue))
	require.NoError(t, err)
	require.Equal(t, 3, info.Config.MaxDeliver)
	require.NoError(t, sub.Unsubscribe())
}

func TestPrepareGPVHandoffFromActiveLegacyConsumerCapturesSwitchWindow(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_RELEASE_HANDOFF_%d", suffix)
	subject := fmt.Sprintf("test.gpv.release.handoff.%d", suffix)
	target := fmt.Sprintf("device-rpc-gpv-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	for sequence := 1; sequence <= 3; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{"device_sn": "SN-HANDOFF", "sequence": sequence})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}
	legacy, err := js.SubscribeSync(subject, nats.DeliverAll(), nats.AckExplicit())
	require.NoError(t, err)
	for sequence := 1; sequence <= 2; sequence++ {
		msg, nextErr := legacy.NextMsg(time.Second)
		require.NoError(t, nextErr)
		require.NoError(t, msg.AckSync())
	}
	legacyInfo, err := legacy.ConsumerInfo()
	require.NoError(t, err)
	require.Equal(t, uint64(2), legacyInfo.AckFloor.Stream)

	result, err := PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject:       subject,
		TargetDurable: target,
		AckWait:       time.Second,
		MaxDeliver:    5,
		MaxAckPending: 10,
	})
	require.NoError(t, err)
	require.Equal(t, legacyInfo.Name, result.SourceConsumer)
	require.Equal(t, uint64(3), result.StartSequence)

	duringSwitch, err := NewEvent(subject, map[string]any{"device_sn": "SN-HANDOFF", "sequence": 4})
	require.NoError(t, err)
	data, err := json.Marshal(duringSwitch)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)
	require.NoError(t, legacy.Unsubscribe())

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	var (
		mu  sync.Mutex
		got []int
	)
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable: target, Concurrency: 1, QueueDepth: 4,
			AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
		},
		func(Event) (string, error) { return "SN-HANDOFF", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			require.NoError(t, evt.DecodePayload(&payload))
			mu.Lock()
			got = append(got, payload.Sequence)
			mu.Unlock()
			return nil
		},
	)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == 2
	}, 5*time.Second, 20*time.Millisecond)
	mu.Lock()
	require.Equal(t, []int{3, 4}, got, "must capture the switch window without replaying acknowledged history")
	mu.Unlock()
	require.NoError(t, sub.Unsubscribe())
}

func TestPrepareGPVHandoffRejectsConfiguredNonRPCDurable(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_CONFIGURED_HANDOFF_%d", suffix)
	subject := fmt.Sprintf("test.gpv.configured.handoff.%d", suffix)
	source := fmt.Sprintf("legacy-rpc-%d", suffix)
	target := fmt.Sprintf("device-rpc-gpv-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	legacy, err := js.QueueSubscribeSync(
		subject,
		source,
		nats.Durable(source),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	require.NoError(t, err)
	for sequence := 1; sequence <= 3; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{"device_sn": "SN-CONFIGURED", "sequence": sequence})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}
	for sequence := 1; sequence <= 2; sequence++ {
		msg, nextErr := legacy.NextMsg(time.Second)
		require.NoError(t, nextErr)
		require.NoError(t, msg.AckSync())
	}

	_, err = PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject: subject, TargetDurable: target, SourceConsumer: source,
		AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
	})
	require.ErrorContains(t, err, "is not a strict legacy RPC ephemeral")
	_, targetErr := js.ConsumerInfo(stream, target)
	require.ErrorIs(t, targetErr, nats.ErrConsumerNotFound)
	require.NoError(t, legacy.Unsubscribe())
}

func TestPrepareGPVHandoffAutoDiscoveryIgnoresOtherGPVPushDurables(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_PRODUCTION_TOPOLOGY_%d", suffix)
	subject := fmt.Sprintf("test.gpv.production.topology.%d", suffix)
	target := fmt.Sprintf("device-rpc-gpv-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	legacyRPC, err := js.SubscribeSync(subject, nats.DeliverAll(), nats.AckExplicit())
	require.NoError(t, err)
	t.Cleanup(func() { _ = legacyRPC.Unsubscribe() })
	alarmSync, err := js.QueueSubscribeSync(
		subject,
		"alarm-sync-gpv",
		nats.Durable("alarm-sync-gpv"),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = alarmSync.Unsubscribe() })
	softwareRollback, err := js.QueueSubscribeSync(
		subject,
		"software-rollback-gpv-resp",
		nats.Durable("software-rollback-gpv-resp"),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = softwareRollback.Unsubscribe() })

	for sequence := 1; sequence <= 2; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{"device_sn": "SN-TOPOLOGY", "sequence": sequence})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}
	msg, err := legacyRPC.NextMsg(time.Second)
	require.NoError(t, err)
	require.NoError(t, msg.AckSync())
	legacyInfo, err := legacyRPC.ConsumerInfo()
	require.NoError(t, err)

	result, err := PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject: subject, TargetDurable: target,
		AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
	})
	require.NoError(t, err)
	require.Equal(t, legacyInfo.Name, result.SourceConsumer)
	require.Equal(t, uint64(2), result.StartSequence)
}

func TestPrepareGPVHandoffRejectsMultipleLegacyRPCEphemerals(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_AMBIGUOUS_HANDOFF_%d", suffix)
	subject := fmt.Sprintf("test.gpv.ambiguous.handoff.%d", suffix)
	target := fmt.Sprintf("device-rpc-gpv-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	first, err := js.SubscribeSync(subject, nats.DeliverAll(), nats.AckExplicit())
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Unsubscribe() })
	second, err := js.SubscribeSync(subject, nats.DeliverAll(), nats.AckExplicit())
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Unsubscribe() })

	_, err = PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject: subject, TargetDurable: target,
		AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
	})
	require.ErrorContains(t, err, "multiple strict legacy RPC ephemeral consumers")
	_, targetErr := js.ConsumerInfo(stream, target)
	require.ErrorIs(t, targetErr, nats.ErrConsumerNotFound)
}

func TestPrepareGPVHandoffWithoutSourceRejectsNonEmptyStream(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_UNSAFE_TAIL_HANDOFF_%d", suffix)
	subject := fmt.Sprintf("test.gpv.unsafe.tail.handoff.%d", suffix)
	target := fmt.Sprintf("device-rpc-gpv-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	evt, err := NewEvent(subject, map[string]any{"device_sn": "SN-UNSAFE", "sequence": 1})
	require.NoError(t, err)
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)

	_, err = PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject: subject, TargetDurable: target,
		AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
	})
	require.ErrorContains(t, err, "has messages but no strict legacy RPC source")
	_, targetErr := js.ConsumerInfo(stream, target)
	require.ErrorIs(t, targetErr, nats.ErrConsumerNotFound)
}

func TestPrepareGPVHandoffFreshInstallStartsAtTailAndCapturesFutureMessages(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_FRESH_HANDOFF_%d", suffix)
	subject := fmt.Sprintf("test.gpv.fresh.handoff.%d", suffix)
	target := fmt.Sprintf("device-rpc-gpv-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	oldEvent, err := NewEvent(subject, map[string]any{"device_sn": "SN-FRESH", "sequence": 1})
	require.NoError(t, err)
	data, err := json.Marshal(oldEvent)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)

	result, err := PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject: subject, TargetDurable: target, FreshInstall: true,
		AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
	})
	require.NoError(t, err)
	require.Empty(t, result.SourceConsumer)
	require.Zero(t, result.StartSequence)

	futureEvent, err := NewEvent(subject, map[string]any{"device_sn": "SN-FRESH", "sequence": 2})
	require.NoError(t, err)
	data, err = json.Marshal(futureEvent)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	got := make(chan int, 2)
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable: target, Concurrency: 1, QueueDepth: 2,
			AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
		},
		func(Event) (string, error) { return "SN-FRESH", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				return err
			}
			got <- payload.Sequence
			return nil
		},
	)
	require.NoError(t, err)
	select {
	case sequence := <-got:
		require.Equal(t, 2, sequence)
	case <-time.After(5 * time.Second):
		t.Fatal("future message was not captured by pre-created durable")
	}
	require.Never(t, func() bool { return len(got) > 0 }, 100*time.Millisecond, 10*time.Millisecond)
	require.NoError(t, sub.Unsubscribe())
}

func TestGPVHandoffEnqueueAdvancesSourceAckFloorWhenDownstreamIsBlocked(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_HANDOFF_ACK_%d", suffix)
	subject := fmt.Sprintf("test.gpv.handoff-ack.%d", suffix)
	durable := fmt.Sprintf("gpv-handoff-ack-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      stream,
		Subjects:  []string{subject},
		Storage:   nats.MemoryStorage,
		Retention: nats.LimitsPolicy,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	repo := &countingGPVHandoffRepo{seen: make(chan Event, 8)}
	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable:       durable,
			Concurrency:   2,
			QueueDepth:    8,
			AckWait:       time.Second,
			MaxDeliver:    3,
			MaxAckPending: 16,
		},
		testDeviceKey,
		GPVHandoffEnqueueHandler(repo, durable, testDeviceKey),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	for _, deviceSN := range []string{"SN-SLOW-HEAD", "SN-FAST-1", "SN-FAST-2"} {
		evt, err := NewEvent(subject, map[string]any{"device_sn": deviceSN})
		require.NoError(t, err)
		data, err := json.Marshal(evt)
		require.NoError(t, err)
		_, err = js.Publish(subject, data)
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		return repo.Count() == 3
	}, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		stats, err := bus.QueueStats(context.Background(), subject, durable)
		if err != nil {
			return false
		}
		return stats.AckPending == 0 &&
			stats.DeliverySequence >= 3 &&
			stats.AckConsumerSequence >= stats.DeliverySequence &&
			stats.AckGap == 0
	}, 5*time.Second, 20*time.Millisecond,
		"source durable AckFloor and ack gap must settle after durable handoff insert, even though downstream processing has not run")
}

func TestCommandGPVQueueStatsShowsAckGapWhileAckRateLagsAndSettles(t *testing.T) {
	url := os.Getenv("GPV_NATS_TEST_URL")
	if url == "" {
		t.Skip("set GPV_NATS_TEST_URL to run the JetStream integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)

	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("GPV_RATE_%d", suffix)
	subject := fmt.Sprintf("issue372.command.gpv.%d", suffix)
	durable := fmt.Sprintf("device-rpc-gpv-rate-%d", suffix)
	_, err = js.AddStream(&nats.StreamConfig{
		Name: stream, Subjects: []string{subject}, Storage: nats.MemoryStorage,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = js.DeleteStream(stream) })

	bus := NewNATSEventBus(nc, js, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	bus.SetQueueTuning(subject, QueueTuning{
		AckWait:       2 * time.Second,
		MaxDeliver:    3,
		MaxAckPending: 3,
	})
	blocked := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseSlowHandler := func() {
		releaseOnce.Do(func() { close(release) })
	}
	var processed atomic.Int64
	sub, err := bus.KeyedQueueSubscribe(
		subject,
		KeyedQueueConfig{
			Durable: durable, Concurrency: 1, QueueDepth: 2,
			AckWait: 2 * time.Second, MaxDeliver: 3, MaxAckPending: 3,
		},
		func(Event) (string, error) { return "SN-SLOW-ISSUE-372", nil },
		func(_ context.Context, evt Event) error {
			var payload struct {
				Sequence int `json:"sequence"`
			}
			if err := evt.DecodePayload(&payload); err != nil {
				return err
			}
			if payload.Sequence == 1 {
				close(blocked)
				<-release
			}
			processed.Add(1)
			return nil
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	t.Cleanup(releaseSlowHandler)

	for sequence := 1; sequence <= 3; sequence++ {
		evt, eventErr := NewEvent(subject, map[string]any{
			"device_sn": "SN-SLOW-ISSUE-372",
			"sequence":  sequence,
		})
		require.NoError(t, eventErr)
		data, marshalErr := json.Marshal(evt)
		require.NoError(t, marshalErr)
		_, publishErr := js.Publish(subject, data)
		require.NoError(t, publishErr)
	}
	select {
	case <-blocked:
	case <-time.After(5 * time.Second):
		t.Fatal("slow GPV handler did not receive the head message")
	}
	var slowStats QueueStats
	var slowStatsErr error
	require.Eventually(t, func() bool {
		stats, statsErr := bus.QueueStats(context.Background(), subject, durable)
		slowStatsErr = statsErr
		if statsErr == nil {
			slowStats = stats
		}
		return statsErr == nil &&
			stats.LastSequence == 3 &&
			stats.DeliverySequence > stats.AckConsumerSequence &&
			stats.AckGap > 0 &&
			stats.AckPending > 0
	}, 5*time.Second, 20*time.Millisecond,
		"while the slow device head is unacked, delivery/publish progress must be visible ahead of ACK floor; last stats=%+v err=%v", slowStats, slowStatsErr)

	releaseSlowHandler()
	require.Eventually(t, func() bool { return processed.Load() == 3 }, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		stats, statsErr := bus.QueueStats(context.Background(), subject, durable)
		return statsErr == nil &&
			stats.LastSequence == 3 &&
			stats.AckSequence == 3 &&
			stats.Pending == 0 &&
			stats.AckPending == 0 &&
			stats.AckGap == 0 &&
			stats.AckConsumerSequence >= stats.DeliverySequence
	}, 5*time.Second, 20*time.Millisecond,
		"after the slow handler completes, ACK floor must catch up with delivered/published progress")
}

func testDeviceKey(evt Event) (string, error) {
	var payload struct {
		DeviceSN string `json:"device_sn"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return "", err
	}
	return payload.DeviceSN, nil
}

type countingGPVHandoffRepo struct {
	mu    sync.Mutex
	count int
	seen  chan Event
}

func (r *countingGPVHandoffRepo) Enqueue(_ context.Context, req GPVHandoffEnqueue) error {
	r.mu.Lock()
	r.count++
	r.mu.Unlock()
	r.seen <- req.Event
	return nil
}

func (r *countingGPVHandoffRepo) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

func (r *countingGPVHandoffRepo) ClaimDue(context.Context, GPVHandoffClaimOptions) ([]GPVHandoffEntry, error) {
	return nil, nil
}

func (r *countingGPVHandoffRepo) MarkDelivered(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error) {
	return true, nil
}

func (r *countingGPVHandoffRepo) MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error) {
	return true, nil
}

func (r *countingGPVHandoffRepo) MarkDead(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error) {
	return true, nil
}
