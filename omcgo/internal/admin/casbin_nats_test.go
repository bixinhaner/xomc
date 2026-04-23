package admin

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// --- Nil-bus degraded path ---

func TestNATSCasbinWatcher_NilBus_UpdateNoOp(t *testing.T) {
	w := newNATSCasbinWatcher(nil, zap.NewNop())
	// 无 bus 时 Update 不应报错，退化为 no-op（周期性刷新兜底）
	assert.NoError(t, w.Update())
	assert.NoError(t, w.Notify())
}

func TestNATSCasbinWatcher_NilBus_StartListenerNoOp(t *testing.T) {
	w := newNATSCasbinWatcher(nil, zap.NewNop())
	// 无 bus 时 StartListener 应立即返回（不启动 goroutine），回调不被触发
	var called int32
	require.NoError(t, w.SetUpdateCallback(func(string) {
		atomic.StoreInt32(&called, 1)
	}))
	w.StartListener()
	time.Sleep(50 * time.Millisecond)
	assert.Zero(t, atomic.LoadInt32(&called), "nil bus 应不触发回调")
	w.Close() // Close 对 nil sub 也安全
}

// --- Happy path with ChannelEventBus ---

func TestNATSCasbinWatcher_PublishSubscribeRoundTrip(t *testing.T) {
	bus := event.NewChannelEventBus(64, zap.NewNop())
	defer bus.Close()

	w := newNATSCasbinWatcher(bus, zap.NewNop())

	received := make(chan string, 1)
	require.NoError(t, w.SetUpdateCallback(func(payload string) {
		received <- payload
	}))

	w.StartListener()
	defer w.Close()

	// Allow the channel subscription to register
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, w.Update())

	select {
	case payload := <-received:
		assert.Equal(t, "reload", payload)
	case <-time.After(time.Second):
		t.Fatal("expected callback to be invoked via NATS subject")
	}
}

func TestNATSCasbinWatcher_NotifyAliasesUpdate(t *testing.T) {
	bus := event.NewChannelEventBus(64, zap.NewNop())
	defer bus.Close()

	w := newNATSCasbinWatcher(bus, zap.NewNop())

	received := make(chan struct{}, 1)
	require.NoError(t, w.SetUpdateCallback(func(string) {
		received <- struct{}{}
	}))

	w.StartListener()
	defer w.Close()
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, w.Notify())

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("Notify should publish to the same subject as Update")
	}
}

// --- Close should unsubscribe ---

func TestNATSCasbinWatcher_CloseUnsubscribes(t *testing.T) {
	bus := event.NewChannelEventBus(64, zap.NewNop())
	defer bus.Close()

	w := newNATSCasbinWatcher(bus, zap.NewNop())

	var count int32
	require.NoError(t, w.SetUpdateCallback(func(string) {
		atomic.AddInt32(&count, 1)
	}))
	w.StartListener()
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, w.Update())
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))

	w.Close()
	// 第二次 Update 不应触发回调
	require.NoError(t, w.Update())
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&count), "Close 后订阅应解除")

	// 再次 Close 应幂等
	w.Close()
}

// --- Publish subject uses the declared constant ---

func TestNATSCasbinWatcher_PublishesToCorrectSubject(t *testing.T) {
	bus := &capturingBus{}
	w := newNATSCasbinWatcher(bus, zap.NewNop())

	require.NoError(t, w.Update())
	require.Len(t, bus.subjects, 1)
	assert.Equal(t, event.SubjectSysCasbinPolicyReload, bus.subjects[0])
}

// capturingBus 最小 EventBus 实现，仅记录 publish 目标主题供断言。
type capturingBus struct {
	subjects []string
}

func (b *capturingBus) Publish(_ context.Context, subject string, _ event.Event) error {
	b.subjects = append(b.subjects, subject)
	return nil
}
func (b *capturingBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &noopCasbinSub{}, nil
}
func (b *capturingBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &noopCasbinSub{}, nil
}
func (b *capturingBus) Close() error { return nil }

type noopCasbinSub struct{}

func (noopCasbinSub) Unsubscribe() error { return nil }

// --- Error propagation ---

func TestNATSCasbinWatcher_PublishError(t *testing.T) {
	w := newNATSCasbinWatcher(&erroringBus{}, zap.NewNop())
	err := w.Update()
	assert.ErrorContains(t, err, "publish casbin reload")
}

type erroringBus struct{}

func (erroringBus) Publish(_ context.Context, _ string, _ event.Event) error {
	return fmt.Errorf("nats down")
}
func (erroringBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &noopCasbinSub{}, nil
}
func (erroringBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &noopCasbinSub{}, nil
}
func (erroringBus) Close() error { return nil }
