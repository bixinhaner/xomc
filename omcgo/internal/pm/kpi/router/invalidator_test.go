package router

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	componentlogger "github.com/omcgo/omcgo/internal/core/components/logger"
)

type invalidationTargetSpy struct {
	calls int
}

func (s *invalidationTargetSpy) InvalidateAll() {
	s.calls++
}

type versionBumperStub struct {
	version int64
	calls   int
	errs    []error
}

type contextAwareBumper struct {
	ctxErr      error
	hasDeadline bool
}

type deadlineBumper struct {
	calls int
}

func (s *deadlineBumper) BumpVersion(ctx context.Context) (int64, error) {
	s.calls++
	<-ctx.Done()
	return 0, ctx.Err()
}

func (s *contextAwareBumper) BumpVersion(ctx context.Context) (int64, error) {
	s.ctxErr = ctx.Err()
	_, s.hasDeadline = ctx.Deadline()
	return 9, nil
}

func (s *versionBumperStub) BumpVersion(context.Context) (int64, error) {
	s.calls++
	if s.calls <= len(s.errs) && s.errs[s.calls-1] != nil {
		return 0, s.errs[s.calls-1]
	}
	return s.version, nil
}

func TestInvalidator_Invalidate_FirstBumpSucceeds(t *testing.T) {
	target := &invalidationTargetSpy{}
	bumper := &versionBumperStub{version: 7}
	reg := prometheus.NewRegistry()
	inv := NewInvalidator(target, bumper, InvalidatorOptions{
		Logger:  zap.NewNop(),
		Metrics: NewMetrics(reg),
	})

	result, err := inv.Invalidate(context.Background(), InvalidationTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, target.calls, "local L1 must be cleared before global invalidation")
	require.Equal(t, 1, bumper.calls)
	require.Equal(t, int64(7), result.CacheVersion)
	require.Equal(t, 1, result.Attempts)
	require.Equal(t, InvalidationScopeGlobal, result.Scope)
	require.True(t, result.MultiProcessSync)
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(`
# HELP omc_kpi_router_invalidations_total KPI route invalidation outcomes by low-cardinality trigger, result, and scope.
# TYPE omc_kpi_router_invalidations_total counter
omc_kpi_router_invalidations_total{result="success",scope="global",trigger="manual"} 1
`), "omc_kpi_router_invalidations_total"))
}

func TestInvalidator_Invalidate_RetryThenSucceeds(t *testing.T) {
	target := &invalidationTargetSpy{}
	bumper := &versionBumperStub{
		version: 8,
		errs:    []error{errors.New("redis unavailable")},
	}
	inv := NewInvalidator(target, bumper, InvalidatorOptions{
		Logger:     zap.NewNop(),
		RetryDelay: time.Nanosecond,
	})

	result, err := inv.Invalidate(context.Background(), InvalidationTriggerIndicatorReload)

	require.NoError(t, err)
	require.Equal(t, 1, target.calls, "retries must not repeat local purge")
	require.Equal(t, 2, bumper.calls)
	require.Equal(t, int64(8), result.CacheVersion)
	require.Equal(t, 2, result.Attempts)
}

func TestInvalidator_Invalidate_ThreeFailuresAreObservable(t *testing.T) {
	target := &invalidationTargetSpy{}
	bumpErr := errors.New("redis unavailable")
	bumper := &versionBumperStub{errs: []error{bumpErr, bumpErr, bumpErr}}
	reg := prometheus.NewRegistry()
	core, logs := observer.New(zap.ErrorLevel)
	inv := NewInvalidator(target, bumper, InvalidatorOptions{
		Logger:     zap.New(core),
		Metrics:    NewMetrics(reg),
		RetryDelay: time.Nanosecond,
	})

	result, err := inv.Invalidate(context.Background(), InvalidationTriggerIndicatorReload)

	require.ErrorContains(t, err, "failed after 3 attempts")
	require.ErrorIs(t, err, bumpErr)
	require.Equal(t, 1, target.calls, "local L1 must stay cleared when Redis is unavailable")
	require.Equal(t, 3, bumper.calls)
	require.Equal(t, 3, result.Attempts)
	require.Len(t, logs.All(), 1)
	require.Equal(t, "KPI route invalidation failed", logs.All()[0].Message)
	require.Equal(t, "indicator_reload", logs.All()[0].ContextMap()["trigger"])
	require.Equal(t, int64(3), logs.All()[0].ContextMap()["attempts"])
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(`
# HELP omc_kpi_router_invalidations_total KPI route invalidation outcomes by low-cardinality trigger, result, and scope.
# TYPE omc_kpi_router_invalidations_total counter
omc_kpi_router_invalidations_total{result="failure",scope="global",trigger="indicator_reload"} 1
`), "omc_kpi_router_invalidations_total"))
}

func TestInvalidator_Invalidate_WithoutRedisClearsOnlyLocalL1(t *testing.T) {
	target := &invalidationTargetSpy{}
	reg := prometheus.NewRegistry()
	inv := NewInvalidator(target, nil, InvalidatorOptions{
		Logger:  zap.NewNop(),
		Metrics: NewMetrics(reg),
	})

	result, err := inv.Invalidate(context.Background(), InvalidationTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, target.calls)
	require.Zero(t, result.CacheVersion)
	require.Zero(t, result.Attempts)
	require.Equal(t, InvalidationScopeLocal, result.Scope)
	require.False(t, result.MultiProcessSync, "without Redis, other processes cannot be synchronized immediately")
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(`
# HELP omc_kpi_router_invalidations_total KPI route invalidation outcomes by low-cardinality trigger, result, and scope.
# TYPE omc_kpi_router_invalidations_total counter
omc_kpi_router_invalidations_total{result="success",scope="local",trigger="manual"} 1
`), "omc_kpi_router_invalidations_total"))
}

func TestInvalidator_Invalidate_UsesBoundedBackgroundContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	bumper := &contextAwareBumper{}
	inv := NewInvalidator(nil, bumper, InvalidatorOptions{
		Logger:  zap.NewNop(),
		Timeout: time.Second,
	})

	result, err := inv.Invalidate(parent, InvalidationTriggerIndicatorReload)

	require.NoError(t, err)
	require.Equal(t, int64(9), result.CacheVersion)
	require.NoError(t, bumper.ctxErr, "a committed write must not lose its invalidation because the request was canceled")
	require.True(t, bumper.hasDeadline, "background invalidation must remain bounded")
}

func TestInvalidator_SetLocalTargetBindsRouterAfterConstruction(t *testing.T) {
	target := &invalidationTargetSpy{}
	inv := NewInvalidator(nil, &versionBumperStub{version: 10}, InvalidatorOptions{})
	inv.SetLocalTarget(target)

	_, err := inv.Invalidate(context.Background(), InvalidationTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, target.calls)
}

func TestInvalidator_Invalidate_TimeoutIsBoundedAndLoggedWithRequestID(t *testing.T) {
	bumper := &deadlineBumper{}
	core, logs := observer.New(zap.ErrorLevel)
	inv := NewInvalidator(nil, bumper, InvalidatorOptions{
		Logger:     zap.New(core),
		RetryDelay: time.Second,
		Timeout:    time.Millisecond,
	})
	ctx := componentlogger.WithRequestID(context.Background(), "req-issue-41")

	result, err := inv.Invalidate(ctx, InvalidationTrigger("unbounded-user-input"))

	require.ErrorContains(t, err, "stopped after 1 attempts")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, bumper.calls)
	require.Equal(t, 1, result.Attempts)
	require.False(t, result.MultiProcessSync)
	require.Len(t, logs.All(), 1)
	require.Equal(t, "other", logs.All()[0].ContextMap()["trigger"], "unknown trigger values must not create metric cardinality")
	require.Equal(t, "req-issue-41", logs.All()[0].ContextMap()["request_id"])
}
