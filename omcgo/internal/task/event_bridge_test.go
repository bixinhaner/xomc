package task

import (
	"context"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeEventBus 实现 event.EventBus，记录订阅信息以便断言。
type fakeEventBus struct {
	queueSubs   []string
	queueGroups []string
	queueErr    error
}

func (b *fakeEventBus) Publish(_ context.Context, _ string, _ event.Event) error { return nil }
func (b *fakeEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &fakeSub{}, nil
}
func (b *fakeEventBus) QueueSubscribe(subject, group string, _ event.EventHandler) (event.Subscription, error) {
	if b.queueErr != nil {
		return nil, b.queueErr
	}
	b.queueSubs = append(b.queueSubs, subject)
	b.queueGroups = append(b.queueGroups, group)
	return &fakeSub{}, nil
}
func (b *fakeEventBus) PullSubscribe(subject, group string, handler event.EventHandler) (event.Subscription, error) {
	return b.QueueSubscribe(subject, group, handler)
}
func (b *fakeEventBus) Close() error { return nil }

type fakeSub struct{}

func (fakeSub) Unsubscribe() error { return nil }

func TestCompletionEventBridge_NewBridge(t *testing.T) {
	router := NewCompletionRouter(zap.NewNop())
	bridge := NewCompletionEventBridge(zap.NewNop(), router, nil)
	require.NotNil(t, bridge)
}

func TestCompletionEventBridge_SubscribeNilBus(t *testing.T) {
	router := NewCompletionRouter(zap.NewNop())
	bridge := NewCompletionEventBridge(zap.NewNop(), router, nil)

	err := bridge.Subscribe(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event bus")
}

func TestCompletionEventBridge_SubscribeNilRouter(t *testing.T) {
	bridge := NewCompletionEventBridge(zap.NewNop(), nil, nil)
	err := bridge.Subscribe(&fakeEventBus{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "completion router")
}

func TestCompletionEventBridge_SubscribeRegistersBothSubjects(t *testing.T) {
	router := NewCompletionRouter(zap.NewNop())
	bridge := NewCompletionEventBridge(zap.NewNop(), router, nil)
	bus := &fakeEventBus{}

	require.NoError(t, bridge.Subscribe(bus))
	assert.Contains(t, bus.queueSubs, event.SubjectTaskCompleted)
	assert.Contains(t, bus.queueSubs, event.SubjectTaskFailed)
	// 每个 subject 用独立 queue 名（C1 阶段 2 拆分），避免 NATS Durable Consumer
	// "subject does not match consumer" 拒绝。
	expected := map[string]string{
		event.SubjectTaskCompleted: "task-completion-bridge-completed",
		event.SubjectTaskFailed:    "task-completion-bridge-failed",
	}
	require.Equal(t, len(bus.queueSubs), len(bus.queueGroups))
	for i, subject := range bus.queueSubs {
		assert.Equal(t, expected[subject], bus.queueGroups[i], "queue name for %s", subject)
	}
}

func TestCompletionEventBridge_SubscribeBusError(t *testing.T) {
	router := NewCompletionRouter(zap.NewNop())
	bridge := NewCompletionEventBridge(zap.NewNop(), router, nil)
	bus := &fakeEventBus{queueErr: errors.New("nats down")}

	err := bridge.Subscribe(bus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscribe")
}

func TestCompletionEventBridge_HandleDispatchesToRouter(t *testing.T) {
	router := NewCompletionRouter(zap.NewNop())
	rec := &recordingHandler{}
	router.Register(TaskSourceMML, rec)

	bridge := NewCompletionEventBridge(zap.NewNop(), router, nil)

	tk := &Task{ID: "t-bridge", Source: TaskSourceMML, SourceID: "mml-1", Status: TaskStatusCompleted}
	evt, err := event.NewEvent(event.SubjectTaskCompleted, tk)
	require.NoError(t, err)

	require.NoError(t, bridge.handle(context.Background(), evt))
	require.Len(t, rec.got, 1)
	assert.Equal(t, "t-bridge", rec.got[0].ID)
}

func TestCompletionEventBridge_HandleBadPayload(t *testing.T) {
	router := NewCompletionRouter(zap.NewNop())
	bridge := NewCompletionEventBridge(zap.NewNop(), router, nil)

	// 制造一个 payload 解码会失败的事件：直接传一个无法 marshal 成 Task 的事件。
	// event.NewEvent 接受 interface{}; 用一个无法被 Task 反序列化的对象（数字）。
	evt, err := event.NewEvent(event.SubjectTaskCompleted, 12345)
	require.NoError(t, err)

	require.Error(t, bridge.handle(context.Background(), evt))
}
