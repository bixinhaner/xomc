package ops

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

func TestStepRouter_DefaultHandlersRegistered(t *testing.T) {
	r := NewStepRouter(zap.NewNop())
	ctx := context.Background()

	// 默认 5 handler 都应可分发不报错
	for _, st := range []StepType{StepRPC, StepMML, StepWait, StepLoop, StepBranch} {
		err := r.Dispatch(ctx, Step{Type: st, Name: "test"})
		require.NoError(t, err, "default %s handler 应 stub 通过", st)
	}
}

func TestStepRouter_UnknownTypeRejected(t *testing.T) {
	r := NewStepRouter(zap.NewNop())
	err := r.Dispatch(context.Background(), Step{Type: "unknown_xyz"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownStepType, "未注册类型须返 ErrUnknownStepType")
}

func TestStepRouter_RegisterOverride(t *testing.T) {
	r := NewStepRouter(zap.NewNop())
	var called bool
	r.Register(StepRPC, func(_ context.Context, _ Step) error {
		called = true
		return nil
	})
	err := r.Dispatch(context.Background(), Step{Type: StepRPC})
	require.NoError(t, err)
	assert.True(t, called, "Register 应替换 default handler")
}

func TestStepRouter_WaitHandler_ActuallyWaits(t *testing.T) {
	r := NewStepRouter(zap.NewNop())
	start := time.Now()
	err := r.Dispatch(context.Background(), Step{
		Type:   StepWait,
		Params: json.RawMessage(`{"seconds":1}`),
	})
	require.NoError(t, err)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(1000), "wait 1s 应实际等 1s")
}

func TestStepRouter_WaitHandler_CtxCancelImmediate(t *testing.T) {
	r := NewStepRouter(zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := r.Dispatch(ctx, Step{
		Type:   StepWait,
		Params: json.RawMessage(`{"seconds":10}`),
	})
	require.Error(t, err, "ctx 超时时 wait 应立即返")
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"应是 ctx 取消错误")
	elapsed := time.Since(start)
	assert.Less(t, elapsed.Milliseconds(), int64(200), "应在 ctx 超时附近返不等满 10s")
}

func TestStepRouter_WaitHandler_NoParams(t *testing.T) {
	r := NewStepRouter(zap.NewNop())
	// 无 params / seconds<=0 应 noop 立即返
	err := r.Dispatch(context.Background(), Step{Type: StepWait, Name: "noop"})
	require.NoError(t, err)
}
