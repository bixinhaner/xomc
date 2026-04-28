package events

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

func newServiceWithFakeStore(t *testing.T) (*EventServiceImpl, *fakeStore) {
	t.Helper()
	store := newFakeStore()
	hub := NewMessageHub(store, zap.NewNop())
	svc := NewEventService(hub, store, zap.NewNop())
	return svc, store
}

func TestNewEventService_PanicsOnNilHub(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil hub")
		}
	}()
	NewEventService(nil, nil, zap.NewNop())
}

func TestNewEventService_PanicsOnNilLogger(t *testing.T) {
	hub := NewMessageHub(nil, zap.NewNop())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil logger")
		}
	}()
	NewEventService(hub, nil, nil)
}

func TestService_Publish_Delivers(t *testing.T) {
	svc, store := newServiceWithFakeStore(t)

	ch, err := svc.SubscribeQuery(context.Background(), "u1")
	require.NoError(t, err)

	msg := &SSEMessage{Event: "alarm.raised", Data: json.RawMessage(`{"id":1}`)}
	require.NoError(t, svc.Publish(context.Background(), "u1", msg))
	assert.NotEmpty(t, msg.ID, "Publish should auto-fill ID when empty")

	select {
	case got := <-ch:
		assert.Equal(t, "alarm.raised", got.Event)
		assert.Equal(t, "u1", got.UserID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for published message")
	}

	assert.Len(t, store.stored["u1"], 1)
}

func TestService_Publish_ValidatesArgs(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	ctx := context.Background()

	err := svc.Publish(ctx, "", &SSEMessage{ID: "m"})
	require.Error(t, err)

	err = svc.Publish(ctx, "u", nil)
	require.Error(t, err)
}

func TestService_PublishJSON_MarshalsAndDelivers(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	ctx := context.Background()

	ch, err := svc.SubscribeQuery(ctx, "u-json")
	require.NoError(t, err)

	type payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, svc.PublishJSON(ctx, "u-json", "task.done", payload{Code: 0, Msg: "ok"}))

	select {
	case got := <-ch:
		assert.Equal(t, "task.done", got.Event)
		assert.JSONEq(t, `{"code":0,"msg":"ok"}`, string(got.Data))
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestService_PublishJSON_MarshalErrorPropagates(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	// channels can't be JSON-marshalled.
	err := svc.PublishJSON(context.Background(), "u", "evt", make(chan int))
	require.Error(t, err)
}

func TestService_PublishGlobal_DeliversToAllAndAutoIDs(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	ctx := context.Background()

	chA, err := svc.SubscribeQuery(ctx, "a")
	require.NoError(t, err)
	chB, err := svc.SubscribeQuery(ctx, "b")
	require.NoError(t, err)

	msg := &SSEMessage{Event: "broadcast", Data: json.RawMessage(`"hi"`)}
	require.NoError(t, svc.PublishGlobal(ctx, msg))
	assert.NotEmpty(t, msg.ID)

	for _, ch := range []<-chan *SSEMessage{chA, chB} {
		select {
		case got := <-ch:
			assert.Equal(t, "broadcast", got.Event)
		case <-time.After(time.Second):
			t.Fatal("global broadcast missed a subscriber")
		}
	}
}

func TestService_PublishGlobal_NilMsg(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	require.Error(t, svc.PublishGlobal(context.Background(), nil))
}

func TestService_SubscribeQuery_EmptyUserID(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	_, err := svc.SubscribeQuery(context.Background(), "")
	require.Error(t, err)
}

func TestService_Unsubscribe_ClosesChannel(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)

	ch, err := svc.SubscribeQuery(context.Background(), "u")
	require.NoError(t, err)

	svc.Unsubscribe("u")
	svc.Unsubscribe("") // no-op for empty userID

	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("Unsubscribe did not close channel")
	}
}

func TestService_QueryHistory_ReturnsAll(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	ctx := context.Background()

	for _, id := range []string{"m1", "m2", "m3"} {
		require.NoError(t, svc.Publish(ctx, "u", &SSEMessage{ID: id, Event: "x"}))
	}

	got, err := svc.QueryHistory(ctx, "u", "", 10)
	require.NoError(t, err)
	require.Len(t, got, 3)
}

func TestService_QueryHistory_AfterID(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	ctx := context.Background()

	for _, id := range []string{"m1", "m2", "m3"} {
		require.NoError(t, svc.Publish(ctx, "u", &SSEMessage{ID: id}))
	}

	got, err := svc.QueryHistory(ctx, "u", "m1", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "m2", got[0].ID)
}

func TestService_QueryHistory_DefaultsLimitTo50(t *testing.T) {
	svc, store := newServiceWithFakeStore(t)
	ctx := context.Background()

	// Push 80 messages. Default limit is 50 → only 50 returned.
	for i := 0; i < 80; i++ {
		_ = store.Store(ctx, "u", &SSEMessage{ID: string(rune('a' + i%26))})
	}

	got, err := svc.QueryHistory(ctx, "u", "", 0) // 0 → default 50
	require.NoError(t, err)
	assert.Len(t, got, 50)
}

func TestService_QueryHistory_NilStore(t *testing.T) {
	hub := NewMessageHub(nil, zap.NewNop())
	svc := NewEventService(hub, nil, zap.NewNop())

	got, err := svc.QueryHistory(context.Background(), "u", "", 10)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_QueryHistory_EmptyUserID(t *testing.T) {
	svc, _ := newServiceWithFakeStore(t)
	_, err := svc.QueryHistory(context.Background(), "", "", 10)
	require.Error(t, err)
}

func TestService_QueryHistory_StoreErrorPropagates(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("redis offline")
	hub := NewMessageHub(store, zap.NewNop())
	svc := NewEventService(hub, store, zap.NewNop())

	_, err := svc.QueryHistory(context.Background(), "u", "", 10)
	require.Error(t, err)
}

func TestService_HubAndStoreAccessors(t *testing.T) {
	svc, store := newServiceWithFakeStore(t)
	assert.NotNil(t, svc.Hub())
	assert.Equal(t, MessageStore(store), svc.Store())
}

func TestService_InterfaceContract(t *testing.T) {
	// EventServiceImpl must satisfy EventService for any consumer that
	// declares dependency on the interface.
	var iface EventService = (*EventServiceImpl)(nil)
	_ = iface
}
