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

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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

	bus := NewNATSEventBus(nc, js, zap.NewNop())
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
	require.Eventually(t, func() bool { return attempts.Load() == 2 }, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		info, infoErr := js.ConsumerInfo(stream, durable)
		return infoErr == nil &&
			info.NumPending == 0 &&
			info.NumAckPending == 0 &&
			info.AckFloor.Stream == 1
	}, 5*time.Second, 20*time.Millisecond)
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

func TestPrepareGPVHandoffFromConfiguredDurableUsesItsAckFloor(t *testing.T) {
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

	result, err := PrepareGPVHandoff(context.Background(), js, GPVHandoffConfig{
		Subject: subject, TargetDurable: target, SourceConsumer: source,
		AckWait: time.Second, MaxDeliver: 5, MaxAckPending: 10,
	})
	require.NoError(t, err)
	require.Equal(t, source, result.SourceConsumer)
	require.Equal(t, uint64(3), result.StartSequence)
	require.NoError(t, legacy.Unsubscribe())
}

func TestPrepareGPVHandoffWithoutSourceStartsAtTailAndCapturesFutureMessages(t *testing.T) {
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
		Subject: subject, TargetDurable: target,
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
