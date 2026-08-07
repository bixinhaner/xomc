package snmp

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTestAlarm() *AlarmEvent {
	return &AlarmEvent{
		AlarmID:      "alarm-001",
		DeviceSerial: "D-001",
		Severity:     "critical",
		AlarmType:    "POWER_FAIL",
		OccurTime:    time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
		Carrier:      "cmcc",
	}
}

func TestNewEngine_Validation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	snd := &mockSender{}

	_, err := NewEngine(nil, reg, logger)
	require.Error(t, err)

	_, err = NewEngine(snd, nil, logger)
	require.Error(t, err)

	e, err := NewEngine(snd, reg, nil) // nil logger ok
	require.NoError(t, err)
	assert.NotNil(t, e)
}

func TestEngine_Process_NilAlarmReturnsNil(t *testing.T) {
	logger := zaptest.NewLogger(t)
	e, err := NewEngine(&mockSender{}, NewInMemoryRegistry(), logger)
	require.NoError(t, err)
	assert.Nil(t, e.Process(context.Background(), nil))
}

func TestEngine_Process_NoTargets_EmptyResults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	snd := &mockSender{}
	e, err := NewEngine(snd, NewInMemoryRegistry(), logger)
	require.NoError(t, err)

	res := e.Process(context.Background(), newTestAlarm())
	assert.NotNil(t, res, "must return non-nil empty slice, not nil")
	assert.Len(t, res, 0)
	assert.Equal(t, 0, snd.calls(), "no targets → no Send")
}

func TestEngine_Process_SkipsDisabledTargets(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("on", true))
	_, _ = reg.Add(newTestTarget("off", false))

	snd := &mockSender{}
	e, err := NewEngine(snd, reg, logger)
	require.NoError(t, err)

	res := e.Process(context.Background(), newTestAlarm())
	assert.Len(t, res, 1, "disabled target must not be attempted")
	assert.Equal(t, "on", res[0].OSSName)
	assert.True(t, res[0].Success)
	assert.Equal(t, 1, snd.calls())
}

func TestEngine_Process_MultiTarget_FailureIsolation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("ok-1", true))
	_, _ = reg.Add(newTestTarget("ok-2", true))
	_, _ = reg.Add(newTestTarget("bad", true))

	wantErr := errors.New("network unreachable")
	var calls int32
	snd := &mockSender{
		returnFn: func(ctx context.Context, target *TrapTarget, vars []Variable) error {
			atomic.AddInt32(&calls, 1)
			if target.OSSName == "bad" {
				return wantErr
			}
			return nil
		},
	}

	e, err := NewEngine(snd, reg, logger)
	require.NoError(t, err)

	res := e.Process(context.Background(), newTestAlarm())
	require.Len(t, res, 3)

	byName := map[string]SendResult{}
	for _, r := range res {
		byName[r.OSSName] = r
	}
	assert.True(t, byName["ok-1"].Success)
	assert.True(t, byName["ok-2"].Success)
	assert.False(t, byName["bad"].Success)
	require.Error(t, byName["bad"].Err)
	assert.True(t, errors.Is(byName["bad"].Err, wantErr))
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestEngine_Process_MapperFailure_AllTargetsMarkedFailed(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("a", true))
	_, _ = reg.Add(newTestTarget("b", true))

	snd := &mockSender{}
	e, err := NewEngine(snd, reg, logger,
		WithMapper(failingMapper{wantErr: errors.New("bad alarm")}),
	)
	require.NoError(t, err)

	res := e.Process(context.Background(), newTestAlarm())
	require.Len(t, res, 2)
	for _, r := range res {
		assert.False(t, r.Success)
		require.Error(t, r.Err)
	}
	assert.Equal(t, 0, snd.calls(), "Sender must not be invoked when mapper fails")
}

func TestEngine_Process_PassesPDUsToSender(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("only", true))

	snd := &mockSender{}
	e, err := NewEngine(snd, reg, logger)
	require.NoError(t, err)

	_ = e.Process(context.Background(), newTestAlarm())
	require.NotNil(t, snd.lastVars)
	// default mapper follows omcAlarmMIB.mib and always emits 18 VarBinds.
	assert.Len(t, snd.lastVars, 18)
}

func TestEngine_Process_RecordsLatency(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("slow", true))

	snd := &mockSender{
		returnFn: func(ctx context.Context, target *TrapTarget, vars []Variable) error {
			select {
			case <-time.After(20 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	e, err := NewEngine(snd, reg, logger)
	require.NoError(t, err)

	res := e.Process(context.Background(), newTestAlarm())
	require.Len(t, res, 1)
	assert.True(t, res[0].Success)
	assert.GreaterOrEqual(t, res[0].Latency, 20*time.Millisecond)
}

func TestEngine_Process_PerSendTimeoutEnforced(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("hang", true))

	snd := &mockSender{
		returnFn: func(ctx context.Context, target *TrapTarget, vars []Variable) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	e, err := NewEngine(snd, reg, logger,
		WithPerSendTimeout(15*time.Millisecond),
	)
	require.NoError(t, err)

	start := time.Now()
	res := e.Process(context.Background(), newTestAlarm())
	elapsed := time.Since(start)

	require.Len(t, res, 1)
	assert.False(t, res[0].Success)
	require.Error(t, res[0].Err)
	// generous upper bound to keep CI stable
	assert.Less(t, elapsed, 500*time.Millisecond,
		"Process should not exceed per-send timeout by orders of magnitude")
}

// failingMapper is an AlarmMapper test double that always returns wantErr.
type failingMapper struct {
	wantErr error
}

func (f failingMapper) MapAlarmToTrapPDU(_ *AlarmEvent) ([]Variable, error) {
	return nil, f.wantErr
}

func TestWithMapper_NilIgnored(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("a", true))
	snd := &mockSender{}
	e, err := NewEngine(snd, reg, logger, WithMapper(nil))
	require.NoError(t, err)

	// Should still use the default mapper rather than panic.
	res := e.Process(context.Background(), newTestAlarm())
	require.Len(t, res, 1)
	assert.True(t, res[0].Success)
}

func TestWithPerSendTimeout_ZeroIgnored(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reg := NewInMemoryRegistry()
	_, _ = reg.Add(newTestTarget("a", true))
	e, err := NewEngine(&mockSender{}, reg, logger, WithPerSendTimeout(0))
	require.NoError(t, err)
	// 0 is rejected → default 5s remains; verify by sending and inspecting that
	// the call returns success (no premature deadline cancellation).
	res := e.Process(context.Background(), newTestAlarm())
	assert.True(t, res[0].Success)
}
