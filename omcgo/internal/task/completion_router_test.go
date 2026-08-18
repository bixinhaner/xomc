package task

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type failingReliableHandler struct{ err error }

func (h *failingReliableHandler) OnTaskCompleted(context.Context, *Task) {}
func (h *failingReliableHandler) OnTaskCompletedReliable(context.Context, *Task) error {
	return h.err
}

// recordingHandler 记录被调用的 Task，供断言比对。
type recordingHandler struct {
	got []*Task
}

func (h *recordingHandler) OnTaskCompleted(_ context.Context, t *Task) {
	h.got = append(h.got, t)
}

func Test_CompletionRouter_DispatchesToRegistered(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)
	mml := &recordingHandler{}
	r.Register(TaskSourceMML, mml)

	r.Dispatch(context.Background(), &Task{ID: "t1", Source: TaskSourceMML, SourceID: "x"})
	r.Dispatch(context.Background(), &Task{ID: "t2", Source: TaskSourceAPI, SourceID: "y"})

	assert.Len(t, mml.got, 1, "MML handler 只接收 source=mml 的任务")
	assert.Equal(t, "t1", mml.got[0].ID)
}

func Test_CompletionRouter_UnknownSourceFallsBackToWarn(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)

	custom := &recordingHandler{}
	r.SetUnknownHandler(custom)

	r.Dispatch(context.Background(), &Task{ID: "t3", Source: TaskSourceAPI, SourceID: "z"})

	assert.Len(t, custom.got, 1, "未注册 source 走 unknownHandler")
}

func Test_CompletionRouter_MultipleHandlersInvokedInOrder(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)

	a, b := &recordingHandler{}, &recordingHandler{}
	r.Register(TaskSourceMML, a)
	r.Register(TaskSourceMML, b)

	r.Dispatch(context.Background(), &Task{ID: "t4", Source: TaskSourceMML, SourceID: "x"})

	assert.Len(t, a.got, 1)
	assert.Len(t, b.got, 1)
}

func Test_CompletionRouter_ReliableErrorReturnedToConsumer(t *testing.T) {
	r := NewCompletionRouter(zap.NewNop())
	want := errors.New("projection unavailable")
	r.Register(TaskSourceDeviceAccess, &failingReliableHandler{err: want})
	observer := &recordingHandler{}
	r.RegisterObserver(observer)

	err := r.DispatchReliable(context.Background(), &Task{ID: "t-retry", Source: TaskSourceDeviceAccess})

	assert.ErrorIs(t, err, want)
	assert.Empty(t, observer.got, "retryable projection failure must not append a duplicate terminal observation")
}

func Test_CompletionRouter_HandlerPanicIsolated(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)

	r.Register(TaskSourceMML, panickingHandler{})
	safe := &recordingHandler{}
	r.Register(TaskSourceMML, safe)

	// 不应 panic；后续 handler 仍被调用。
	r.Dispatch(context.Background(), &Task{ID: "t5", Source: TaskSourceMML, SourceID: "x"})

	assert.Len(t, safe.got, 1, "前置 handler panic 不应阻断后续 handler")
}

type panickingHandler struct{}

func (panickingHandler) OnTaskCompleted(_ context.Context, _ *Task) {
	panic("boom")
}

func Test_CompletionRouter_NilTaskNoop(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)
	h := &recordingHandler{}
	r.Register(TaskSourceMML, h)

	// 不应 panic；handler 不被调用。
	r.Dispatch(context.Background(), nil)
	assert.Empty(t, h.got)
}

// #122：source 无关观察者应对每个终态任务都触发，无论 source、无论是否命中
// per-source handler（验证 sys_task_logs 写入链路覆盖全部来源）。
func Test_CompletionRouter_ObserverFiresForEverySource(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)

	obs := &recordingHandler{}
	r.RegisterObserver(obs)

	// 一个有 per-source handler，一个无（走 unknown）——observer 两个都应收到。
	mml := &recordingHandler{}
	r.Register(TaskSourceMML, mml)

	r.Dispatch(context.Background(), &Task{ID: "o1", Source: TaskSourceMML, SourceID: "x"})
	r.Dispatch(context.Background(), &Task{ID: "o2", Source: TaskSourceAPI, SourceID: "y"})

	assert.Len(t, obs.got, 2, "observer 对每个 source 的终态任务都触发")
	assert.Equal(t, "o1", obs.got[0].ID)
	assert.Equal(t, "o2", obs.got[1].ID)
	assert.Len(t, mml.got, 1, "per-source handler 仍只收自己 source 的任务")
}

// observer panic 必须被隔离，不得阻断 per-source handler。
func Test_CompletionRouter_ObserverPanicIsolated(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)

	r.RegisterObserver(panickingHandler{})
	safe := &recordingHandler{}
	r.Register(TaskSourceMML, safe)

	assert.NotPanics(t, func() {
		r.Dispatch(context.Background(), &Task{ID: "o3", Source: TaskSourceMML, SourceID: "x"})
	})
	assert.Len(t, safe.got, 1, "observer panic 不应阻断 per-source handler")
}

// 传 nil observer 应被忽略，不得在 Dispatch 时 panic。
func Test_CompletionRouter_RegisterNilObserverIgnored(t *testing.T) {
	logger := zap.NewNop()
	r := NewCompletionRouter(logger)
	r.RegisterObserver(nil)

	assert.NotPanics(t, func() {
		r.Dispatch(context.Background(), &Task{ID: "o4", Source: TaskSourceAPI, SourceID: "z"})
	})
}
