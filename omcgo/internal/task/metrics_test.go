package task

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskMetrics_Registered(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)

	require.NotNil(t, m)
	assert.NotNil(t, m.PendingTotal)
	assert.NotNil(t, m.CompletedTotal)
	assert.NotNil(t, m.DurationSeconds)
	assert.NotNil(t, m.NoHandlerTotal)
	// issue #20 新增
	assert.NotNil(t, m.BacklogTotal)
	assert.NotNil(t, m.StaleDetectedTotal)
	assert.NotNil(t, m.RecoveryActionTotal)

	// 确认指标真的注册了
	families, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range families {
		names[f.GetName()] = true
	}
	// CounterVec/HistogramVec 在没有 observation 的情况下不会被 Gather 返回
	// 但 Gauge 会，所以 PendingTotal / BacklogTotal 一定出现
	assert.True(t, names["omc_tasks_pending_total"], "PendingTotal gauge should be registered")
	assert.True(t, names["omc_tasks_backlog_total"], "BacklogTotal gauge should be registered")
}

func TestNewTaskMetrics_CounterIncrements(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)

	m.PendingTotal.Inc()
	m.CompletedTotal.WithLabelValues("mml", "completed").Inc()
	m.DurationSeconds.WithLabelValues("mml").Observe(1.5)
	m.NoHandlerTotal.WithLabelValues("unknown-source").Inc()

	families, err := reg.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, families)
}

// issue #20：backlog gauge / stale-detection counter / recovery-action counter
// 的读写行为（用 testutil 直接断言值）。
func TestTaskMetrics_LifecycleObservability(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)

	m.BacklogTotal.Set(42)
	m.StaleDetectedTotal.Inc()
	m.StaleDetectedTotal.Inc()
	m.RecoveryActionTotal.WithLabelValues(RecoveryActionRestorePending).Add(3)
	m.RecoveryActionTotal.WithLabelValues(RecoveryActionReconcileRepair).Inc()

	assert.Equal(t, float64(42), testutil.ToFloat64(m.BacklogTotal))
	assert.Equal(t, float64(2), testutil.ToFloat64(m.StaleDetectedTotal))
	assert.Equal(t, float64(3),
		testutil.ToFloat64(m.RecoveryActionTotal.WithLabelValues(RecoveryActionRestorePending)))
	assert.Equal(t, float64(1),
		testutil.ToFloat64(m.RecoveryActionTotal.WithLabelValues(RecoveryActionReconcileRepair)))
}

func TestTaskMetrics_ContractUsesOnlyLowCardinalityLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)

	m.CompletedTotal.WithLabelValues("mml", "completed").Inc()
	m.DurationSeconds.WithLabelValues("mml").Observe(1)
	m.NoHandlerTotal.WithLabelValues("unknown-source").Inc()

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				assert.NotContains(t, label.GetName(), "device_sn")
				assert.NotContains(t, label.GetName(), "redis_key")
				assert.NotContains(t, label.GetName(), "object_path")
				assert.NotContains(t, label.GetValue(), "device-sn-001")
				assert.NotContains(t, label.GetValue(), "redis:acs:taskq:")
			}
		}
	}
}

func TestTaskMetrics_PersistentQueueMetricContract(t *testing.T) {
	assert.Equal(t, []string{
		"device_tasks",
		"async_jobs",
		"parameter_sync_outbox",
		"northbound_outbox",
		"pm_kpi_export",
		"trace_export",
		"backup_tasks",
		"dead_letters",
	}, PersistentQueueNames())
	assert.Equal(t, "pending", PersistentQueueMetricPending)
	assert.Equal(t, "oldest_age_seconds", PersistentQueueMetricOldestAgeSeconds)
	assert.Equal(t, "overdue_oldest_age_seconds", PersistentQueueMetricOverdueOldestAgeSeconds)
	assert.Equal(t, "failed_total", PersistentQueueMetricFailedTotal)
	assert.Equal(t, "dead_letter_total", PersistentQueueMetricDeadLetterTotal)
	assert.Equal(t, "processed_total", PersistentQueueMetricProcessedTotal)
	assert.Equal(t, "observer_failures_total", PersistentQueueMetricObserverFailuresTotal)
	assert.Equal(t, []string{"pending", "sent", "running", "succeeded", "failed", "dead_letter"}, PersistentQueueStatuses())
}

func TestPersistentQueueLabelsRejectUnregisteredValues(t *testing.T) {
	for name, values := range map[string][3]string{
		"queue":  {"device-sn-001", "pending", ""},
		"status": {"device_tasks", "redis-key:abc", ""},
		"result": {"device_tasks", "succeeded", "/object/path/file"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewPersistentQueueLabels(values[0], values[1], values[2])
			require.Error(t, err)
		})
	}
}

func TestPersistentQueueLabelsExposeOnlyValidatedValues(t *testing.T) {
	labels, err := NewPersistentQueueLabels("device_tasks", "pending", "")
	require.NoError(t, err)
	assert.Equal(t, [3]string{"device_tasks", "pending", ""}, labels.Values())

	names := PersistentQueueNames()
	names[0] = "device-sn-001"
	assert.Equal(t, "device_tasks", PersistentQueueNames()[0], "queue registry must not be externally mutable")
}

func TestPersistentQueueLabelsOnlyProduceValidatedMetricValues(t *testing.T) {
	labels, err := NewPersistentQueueLabels("device_tasks", "failed", "failed")
	require.NoError(t, err)
	values, err := labels.LabelValues()
	require.NoError(t, err)
	assert.Equal(t, []string{"device_tasks", "failed", "failed"}, values)

	_, err = (PersistentQueueLabels{}).LabelValues()
	require.Error(t, err)
}
