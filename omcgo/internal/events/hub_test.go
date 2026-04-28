package events

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeStore is an in-memory MessageStore for unit tests.
type fakeStore struct {
	mu       sync.Mutex
	stored   map[string][]*SSEMessage
	storeErr error
	getErr   error
}

func newFakeStore() *fakeStore {
	return &fakeStore{stored: make(map[string][]*SSEMessage)}
}

func (f *fakeStore) Store(_ context.Context, userID string, msg *SSEMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.storeErr != nil {
		return f.storeErr
	}
	f.stored[userID] = append(f.stored[userID], msg)
	return nil
}

func (f *fakeStore) GetSince(_ context.Context, userID string, afterID string, limit int) ([]*SSEMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	src := f.stored[userID]
	if afterID == "" {
		out := make([]*SSEMessage, 0, len(src))
		for i, m := range src {
			if i >= limit {
				break
			}
			out = append(out, m)
		}
		return out, nil
	}
	out := []*SSEMessage{}
	found := false
	for _, m := range src {
		if found {
			out = append(out, m)
			if len(out) >= limit {
				break
			}
			continue
		}
		if m.ID == afterID {
			found = true
		}
	}
	return out, nil
}

func TestHub_PublishAndSubscribe_DeliversMessage(t *testing.T) {
	store := newFakeStore()
	hub := NewMessageHub(store, zap.NewNop())

	ch, err := hub.Subscribe("user-1")
	require.NoError(t, err)

	msg := &SSEMessage{ID: "m1", Event: "alarm", Data: json.RawMessage(`{"x":1}`)}
	require.NoError(t, hub.Publish("user-1", msg))

	select {
	case got := <-ch:
		assert.Equal(t, "m1", got.ID)
		assert.Equal(t, "user-1", got.UserID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Persisted exactly once.
	assert.Len(t, store.stored["user-1"], 1)
}

func TestHub_Publish_NoSubscriber_PersistsOnly(t *testing.T) {
	store := newFakeStore()
	hub := NewMessageHub(store, zap.NewNop())

	msg := &SSEMessage{ID: "m1", Event: "test", Data: json.RawMessage(`{}`)}
	require.NoError(t, hub.Publish("offline-user", msg))

	// No panic, message persisted.
	assert.Len(t, store.stored["offline-user"], 1)
}

func TestHub_Publish_StoreErrorIsLoggedNotReturned(t *testing.T) {
	store := newFakeStore()
	store.storeErr = assertAnyError{}
	hub := NewMessageHub(store, zap.NewNop())

	// Even with store error, hub.Publish should not return error
	// (it warns and continues).
	require.NoError(t, hub.Publish("u", &SSEMessage{ID: "m"}))
}

type assertAnyError struct{}

func (assertAnyError) Error() string { return "store down" }

func TestHub_PublishSimple_GeneratesIDAndDelivers(t *testing.T) {
	hub := NewMessageHub(nil, zap.NewNop())

	ch, err := hub.Subscribe("user-2")
	require.NoError(t, err)

	hub.PublishSimple("user-2", "hello", []byte(`"world"`))

	select {
	case got := <-ch:
		assert.NotEmpty(t, got.ID, "PublishSimple should auto-generate ID")
		assert.Equal(t, "hello", got.Event)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestHub_Subscribe_KicksExistingChannel(t *testing.T) {
	hub := NewMessageHub(nil, zap.NewNop())

	ch1, err := hub.Subscribe("user-3")
	require.NoError(t, err)

	// Re-subscribe same user → ch1 should be closed.
	ch2, err := hub.Subscribe("user-3")
	require.NoError(t, err)
	require.NotNil(t, ch2)

	select {
	case _, ok := <-ch1:
		assert.False(t, ok, "old channel must be closed")
	case <-time.After(time.Second):
		t.Fatal("old channel was not closed in time")
	}
}

func TestHub_Unsubscribe_RemovesChannel(t *testing.T) {
	hub := NewMessageHub(nil, zap.NewNop())

	ch, err := hub.Subscribe("user-4")
	require.NoError(t, err)

	hub.Unsubscribe("user-4")

	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("channel was not closed by Unsubscribe")
	}

	// Unsubscribe a non-existent user is a no-op.
	hub.Unsubscribe("user-4")
	hub.Unsubscribe("never-existed")
}

func TestHub_PublishGlobal_DeliversToAllSubscribers(t *testing.T) {
	hub := NewMessageHub(nil, zap.NewNop())

	chA, err := hub.Subscribe("a")
	require.NoError(t, err)
	chB, err := hub.Subscribe("b")
	require.NoError(t, err)

	msg := &SSEMessage{ID: "g1", Event: "broadcast"}
	require.NoError(t, hub.PublishGlobal(msg))

	for _, ch := range []<-chan *SSEMessage{chA, chB} {
		select {
		case got := <-ch:
			assert.Equal(t, "g1", got.ID)
		case <-time.After(time.Second):
			t.Fatal("global publish missed a subscriber")
		}
	}
}

func TestHub_Publish_FullChannelDropsOldest(t *testing.T) {
	// Build a hub and saturate one user's channel beyond capacity (256).
	hub := NewMessageHub(nil, zap.NewNop())
	_, err := hub.Subscribe("u")
	require.NoError(t, err)

	// We don't drain. Publishing 300 messages should not block or panic;
	// hub drops oldest and keeps moving.
	for i := 0; i < 300; i++ {
		require.NoError(t, hub.Publish("u", &SSEMessage{ID: "n", Data: json.RawMessage(`{}`)}))
	}
}
