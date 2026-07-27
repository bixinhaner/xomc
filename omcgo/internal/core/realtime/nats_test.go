package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCoreNATS struct {
	mu                 sync.Mutex
	handlers           map[string][]func([]byte)
	flushed            bool
	flushCalls         int
	publishBeforeFlush bool
}

func (f *fakeCoreNATS) publish(subject string, data []byte) error {
	f.mu.Lock()
	if !f.flushed {
		f.publishBeforeFlush = true
	}
	handlers := append([]func([]byte){}, f.handlers[subject]...)
	f.mu.Unlock()
	for _, handler := range handlers {
		handler(data)
	}
	return nil
}

func (f *fakeCoreNATS) subscribe(subject string, handler func([]byte)) (Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[subject] = append(f.handlers[subject], handler)
	f.flushed = false
	return noopSubscription{}, nil
}

func (f *fakeCoreNATS) flush(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flushed = true
	f.flushCalls++
	return nil
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error { return nil }

type errorSubscription struct{ err error }

func (s errorSubscription) Unsubscribe() error { return s.err }

func TestCoreNATSAdapter_BroadcastSeamDeliversDirectPayloadToEverySubscriber(t *testing.T) {
	transport := &fakeCoreNATS{handlers: make(map[string][]func([]byte))}
	bus := newCoreNATS(transport.publish, transport.subscribe, transport.flush)

	receivedA := make(chan map[string]any, 1)
	receivedB := make(chan map[string]any, 1)
	for _, received := range []chan map[string]any{receivedA, receivedB} {
		_, err := bus.Subscribe("pm_realtime.adhoc.progress", func(data []byte) {
			var payload map[string]any
			require.NoError(t, json.Unmarshal(data, &payload))
			received <- payload
		})
		require.NoError(t, err)
	}

	payload := map[string]any{"task_id": "task-1", "progress": 35}
	require.Equal(t, 2, transport.flushCalls, "each subscription must be registered before publish")
	require.NoError(t, bus.Publish(context.Background(), "pm_realtime.adhoc.progress", payload))
	assert.False(t, transport.publishBeforeFlush, "subscriptions must be flushed before publish")
	assert.Equal(t, 2, transport.flushCalls, "publish must not add a synchronous flush round trip")

	for _, received := range []chan map[string]any{receivedA, receivedB} {
		select {
		case got := <-received:
			assert.Equal(t, "task-1", got["task_id"])
			assert.Equal(t, float64(35), got["progress"])
			assert.NotContains(t, got, "payload", "Core NATS payload must not add an event envelope")
		case <-time.After(time.Second):
			t.Fatal("subscriber did not receive realtime message")
		}
	}
}

func TestCoreNATSAdapter_FlushFailureIncludesCleanupFailure(t *testing.T) {
	flushErr := errors.New("flush failed")
	cleanupErr := errors.New("unsubscribe failed")
	bus := newCoreNATS(
		func(string, []byte) error { return nil },
		func(string, func([]byte)) (Subscription, error) {
			return errorSubscription{err: cleanupErr}, nil
		},
		func(context.Context) error { return flushErr },
	)

	_, err := bus.Subscribe("pm_realtime.adhoc.progress", func([]byte) {})
	require.Error(t, err)
	assert.ErrorIs(t, err, flushErr)
	assert.ErrorIs(t, err, cleanupErr)
	assert.Contains(t, err.Error(), "flush realtime subscription pm_realtime.adhoc.progress")
	assert.Contains(t, err.Error(), "unsubscribe realtime subject pm_realtime.adhoc.progress after flush failure")
}
