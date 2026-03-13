package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestChannelEventBus_PublishSubscribe(t *testing.T) {
	bus := NewChannelEventBus(64, zap.NewNop())
	defer bus.Close()

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	_, err := bus.Subscribe("device.inform.bootstrap", func(ctx context.Context, evt Event) error {
		received = evt
		wg.Done()
		return nil
	})
	require.NoError(t, err)

	evt, err := NewEvent("device.inform.bootstrap", map[string]string{"sn": "TEST001"})
	require.NoError(t, err)

	err = bus.Publish(context.Background(), "device.inform.bootstrap", evt)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		assert.Equal(t, "device.inform.bootstrap", received.Subject)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestChannelEventBus_WildcardSubscribe(t *testing.T) {
	bus := NewChannelEventBus(64, zap.NewNop())
	defer bus.Close()

	var count int32
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	_, err := bus.Subscribe("device.inform.>", func(ctx context.Context, evt Event) error {
		mu.Lock()
		count++
		mu.Unlock()
		wg.Done()
		return nil
	})
	require.NoError(t, err)

	evt1, _ := NewEvent("device.inform.bootstrap", nil)
	evt2, _ := NewEvent("device.inform.periodic", nil)

	bus.Publish(context.Background(), "device.inform.bootstrap", evt1)
	bus.Publish(context.Background(), "device.inform.periodic", evt2)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		mu.Lock()
		assert.Equal(t, int32(2), count)
		mu.Unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for events")
	}
}

func TestMatchSubject(t *testing.T) {
	tests := []struct {
		pattern string
		subject string
		match   bool
	}{
		{"device.inform.bootstrap", "device.inform.bootstrap", true},
		{"device.inform.bootstrap", "device.inform.periodic", false},
		{"device.inform.*", "device.inform.bootstrap", true},
		{"device.inform.*", "device.inform.periodic", true},
		{"device.inform.*", "device.inform", false},
		{"device.>", "device.inform.bootstrap", true},
		{"device.>", "device.inform", true},
		{"device.>", "device", false},
		{"*.inform.bootstrap", "device.inform.bootstrap", true},
		{"*.inform.bootstrap", "alarm.inform.bootstrap", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.subject, func(t *testing.T) {
			assert.Equal(t, tt.match, matchSubject(tt.pattern, tt.subject))
		})
	}
}

func TestChannelEventBus_Close(t *testing.T) {
	bus := NewChannelEventBus(64, zap.NewNop())

	_, err := bus.Subscribe("test", func(ctx context.Context, evt Event) error { return nil })
	require.NoError(t, err)

	err = bus.Close()
	assert.NoError(t, err)

	err = bus.Publish(context.Background(), "test", Event{})
	assert.ErrorIs(t, err, ErrBusClosed)
}
