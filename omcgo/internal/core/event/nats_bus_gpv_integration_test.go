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
	require.NoError(t, firstSub.Unsubscribe())
	firstMu.Lock()
	require.ElementsMatch(t, []int{2, 3, 4}, firstGot)
	firstMu.Unlock()

	evt, err := NewEvent(subject, map[string]any{"device_sn": "SN-5", "sequence": 5})
	require.NoError(t, err)
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	_, err = js.Publish(subject, data)
	require.NoError(t, err)

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
