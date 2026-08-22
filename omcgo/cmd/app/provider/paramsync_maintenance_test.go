package provider

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func TestRunPeriodicMaintenanceLoopsDoNotStarveEachOther(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	slowStarted := make(chan struct{}, 1)
	releaseSlow := make(chan struct{})
	var fastCalls atomic.Int32

	go runPeriodicMaintenance(ctx, time.Millisecond, func(context.Context) {
		select {
		case slowStarted <- struct{}{}:
		default:
		}
		select {
		case <-releaseSlow:
		case <-ctx.Done():
		}
	})
	go runPeriodicMaintenance(ctx, time.Millisecond, func(context.Context) {
		fastCalls.Add(1)
	})

	select {
	case <-slowStarted:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("slow maintenance loop did not start")
	}
	require.Eventually(t, func() bool { return fastCalls.Load() >= 3 }, 100*time.Millisecond, time.Millisecond)
	close(releaseSlow)
}

func TestRunPeriodicMaintenanceSchedulesNextRunAfterCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	starts := make(chan time.Time, 2)

	go runPeriodicMaintenance(ctx, 10*time.Millisecond, func(context.Context) {
		starts <- time.Now()
		time.Sleep(15 * time.Millisecond)
	})

	first := <-starts
	second := <-starts
	assert.GreaterOrEqual(t, second.Sub(first), 23*time.Millisecond,
		"a slow task must not trigger an immediate catch-up run")
}

func TestParamSyncMaintenanceWorkerBudgetRemainsBounded(t *testing.T) {
	assert.LessOrEqual(t, paramSyncOutboxWorkers+paramSyncQueuedWorkers, 4)
	assert.LessOrEqual(t, paramSyncOutboxBatchLimit, 10)
	assert.LessOrEqual(t, paramSyncQueuedBatchLimit, 25)
	assert.LessOrEqual(t, paramSyncAdmissionReconcileLimit, 100)
	assert.LessOrEqual(t, paramSyncResultConsumerShards, 4)
	assert.LessOrEqual(t, paramSyncResultConsumerQueue, 32)
	assert.LessOrEqual(t, paramSyncResultConsumerQueueMax, 64)
}

func TestParamSyncPullTuningFromAppClampsOversizedConfig(t *testing.T) {
	tuning := paramSyncPullTuningFromApp(appconfig.ParamSyncConfig{
		ResultConsumerPullBatchSize:   64,
		ResultConsumerPullConcurrency: 64,
		ResultConsumerMaxAckPending:   512,
		ResultConsumerAckWait:         2 * time.Minute,
	})

	assert.Equal(t, paramSyncResultPullBatchSize, tuning.BatchSize)
	assert.Equal(t, paramSyncResultPullConcurrency, tuning.Concurrency)
	assert.Equal(t, paramSyncResultMaxAckPending, tuning.MaxAckPending)
	assert.Equal(t, 2*time.Minute, tuning.AckWait)
}

func TestParamSyncPullTuningFromAppUsesBackpressureDefaults(t *testing.T) {
	tuning := paramSyncPullTuningFromApp(appconfig.ParamSyncConfig{})

	assert.Equal(t, paramSyncResultPullBatchSize, tuning.BatchSize)
	assert.Equal(t, paramSyncResultPullConcurrency, tuning.Concurrency)
	assert.Equal(t, paramSyncResultMaxAckPending, tuning.MaxAckPending)
	assert.Equal(t, paramSyncResultAckWait, tuning.AckWait)
}

func TestBoundedPositiveIntUsesDefaultAndMax(t *testing.T) {
	assert.Equal(t, 8, boundedPositiveInt(0, 8, 16))
	assert.Equal(t, 16, boundedPositiveInt(64, 8, 16))
	assert.Equal(t, 12, boundedPositiveInt(12, 8, 16))
}

func TestParamSyncMaintenanceRecoveryCoversOneTaskPerRunBursts(t *testing.T) {
	cfg := defaultParamSyncMaintenanceConfig()

	assert.Equal(t, 200, cfg.recoveryRunLimit)
	assert.Equal(t, 200, cfg.recoveryTaskBudget)
}

type failingParamSyncMaintainer struct {
	calls []string
}

type blockingRecoveryParamSyncMaintainer struct {
	failingParamSyncMaintainer
}

func (m *blockingRecoveryParamSyncMaintainer) RecoverMissingResults(ctx context.Context, _, _, _ int) (int, error) {
	m.record("recover")
	<-ctx.Done()
	return 0, ctx.Err()
}

type recordingParamSyncProjector struct {
	called bool
	err    error
}

func (p *recordingParamSyncProjector) ReconcilePending(context.Context, int) (int, error) {
	p.called = true
	return 0, p.err
}

func (m *failingParamSyncMaintainer) record(name string) { m.calls = append(m.calls, name) }

func (m *failingParamSyncMaintainer) SweepExpiredRequests(context.Context, int) (int64, error) {
	m.record("sweep")
	return 0, nil
}
func (m *failingParamSyncMaintainer) ReconcileStalledRequests(context.Context, time.Time, int) (int64, error) {
	m.record("stalled")
	return 0, nil
}
func (m *failingParamSyncMaintainer) ReconcileStalledRuns(context.Context, time.Time, int) (int64, error) {
	m.record("stalled-runs")
	return 0, nil
}
func (m *failingParamSyncMaintainer) ReconcileCancellingRuns(context.Context, int) (int64, error) {
	m.record("cancelling-runs")
	return 0, nil
}
func (m *failingParamSyncMaintainer) ReconcileRunCounts(context.Context) (int64, error) {
	m.record("counts")
	return 0, errors.New("historical row invalid")
}
func (m *failingParamSyncMaintainer) ReconcileTerminalBindings(context.Context) (int64, error) {
	m.record("bindings")
	return 0, nil
}
func (m *failingParamSyncMaintainer) RecoverMissingResults(context.Context, int, int, int) (int, error) {
	m.record("recover")
	return 0, nil
}
func (m *failingParamSyncMaintainer) CleanStaging(context.Context, time.Time, int) (int64, error) {
	m.record("staging")
	return 0, nil
}
func (m *failingParamSyncMaintainer) CollectMetrics(context.Context) error {
	m.record("metrics")
	return nil
}

func TestRunParamSyncMaintenanceDoesNotShortCircuitIndependentRepairs(t *testing.T) {
	maintainer := &failingParamSyncMaintainer{}

	err := runParamSyncMaintenance(context.Background(), maintainer, time.Now(), defaultParamSyncMaintenanceConfig())

	require.ErrorContains(t, err, "historical row invalid")
	assert.Equal(t, []string{"counts", "sweep", "stalled", "stalled-runs", "cancelling-runs", "recover", "bindings", "staging", "metrics"}, maintainer.calls)
}

func TestRunParamSyncMaintenanceRecoveryTimeoutDoesNotCancelLaterSteps(t *testing.T) {
	maintainer := &blockingRecoveryParamSyncMaintainer{}
	cfg := defaultParamSyncMaintenanceConfig()
	cfg.stepTimeout = 50 * time.Millisecond
	cfg.recoveryTimeout = time.Millisecond

	err := runParamSyncMaintenance(context.Background(), maintainer, time.Now(), cfg)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, []string{"counts", "sweep", "stalled", "stalled-runs", "cancelling-runs", "recover", "bindings", "staging", "metrics"}, maintainer.calls)
}

func TestRunParamSyncReconciliationProjectsEvenWhenMaintenanceFails(t *testing.T) {
	maintainer := &failingParamSyncMaintainer{}
	projector := &recordingParamSyncProjector{err: errors.New("projection failed")}

	err := runParamSyncReconciliation(context.Background(), maintainer, projector, time.Now(), defaultParamSyncMaintenanceConfig())

	require.ErrorContains(t, err, "historical row invalid")
	require.ErrorContains(t, err, "projection failed")
	assert.True(t, projector.called)
}
