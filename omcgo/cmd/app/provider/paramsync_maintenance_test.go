package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingParamSyncMaintainer struct {
	calls []string
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
func (m *failingParamSyncMaintainer) RepublishMissingResults(context.Context, int) (int, error) {
	m.record("republish")
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

	err := runParamSyncMaintenance(context.Background(), maintainer, time.Now())

	require.ErrorContains(t, err, "historical row invalid")
	assert.Equal(t, []string{"sweep", "stalled", "stalled-runs", "cancelling-runs", "counts", "bindings", "republish", "staging", "metrics"}, maintainer.calls)
}

func TestRunParamSyncReconciliationProjectsEvenWhenMaintenanceFails(t *testing.T) {
	maintainer := &failingParamSyncMaintainer{}
	projector := &recordingParamSyncProjector{err: errors.New("projection failed")}

	err := runParamSyncReconciliation(context.Background(), maintainer, projector, time.Now())

	require.ErrorContains(t, err, "historical row invalid")
	require.ErrorContains(t, err, "projection failed")
	assert.True(t, projector.called)
}
