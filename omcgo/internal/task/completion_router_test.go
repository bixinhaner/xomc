package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

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
