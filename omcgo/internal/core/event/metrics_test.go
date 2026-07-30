package event

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBusMetricsObserveQueueStats(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)
	sampledAt := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)

	m.observeQueueStats("pm.file.received", "pm-workers", QueueStats{
		Pending:          7,
		AckPending:       2,
		Redelivered:      3,
		OldestPendingAge: 45 * time.Second,
		LastSequence:     20,
		AckSequence:      11,
		SampledAt:        sampledAt,
	})

	labels := []string{"pm.file.received", "pm-workers"}
	assert.Equal(t, float64(7), testutil.ToFloat64(m.QueuePending.WithLabelValues(labels...)))
	assert.Equal(t, float64(2), testutil.ToFloat64(m.QueueAckPending.WithLabelValues(labels...)))
	assert.Equal(t, float64(3), testutil.ToFloat64(m.QueueRedelivered.WithLabelValues(labels...)))
	assert.Equal(t, float64(45), testutil.ToFloat64(m.QueueOldestAgeSeconds.WithLabelValues(labels...)))
	assert.Equal(t, float64(20), testutil.ToFloat64(m.QueueLastSequence.WithLabelValues(labels...)))
	assert.Equal(t, float64(11), testutil.ToFloat64(m.QueueAckSequence.WithLabelValues(labels...)))
	assert.Equal(t, float64(sampledAt.Unix()), testutil.ToFloat64(m.QueueSampleTimestampSeconds.WithLabelValues(labels...)))
}

func TestEventBusMetricsObserveQueueStatsBoundsPMLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)
	stats := QueueStats{Pending: 1, SampledAt: time.Now()}

	m.observeQueueStats(SubjectPMFileReceived, "pm-workers", stats)
	m.observeQueueStats("unbounded.subject", "unbounded-durable", stats)

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "omc_pm_queue_pending" {
			assert.Len(t, family.Metric, 1, "only the fixed PM queue may create a queue metric series")
			return
		}
	}
	t.Fatal("omc_pm_queue_pending was not registered")
}

func TestEventBusMetricsRegistersPMQueueMetricContract(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)
	m.observeQueueStats(SubjectPMFileReceived, "pm-workers", QueueStats{SampledAt: time.Now()})
	m.observeQueueSampleFailure(SubjectPMFileReceived, "pm-workers")

	families, err := reg.Gather()
	require.NoError(t, err)
	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, name := range []string{
		"omc_pm_queue_pending",
		"omc_pm_queue_ack_pending",
		"omc_pm_queue_redelivered",
		"omc_pm_queue_oldest_age_seconds",
		"omc_pm_queue_last_sequence",
		"omc_pm_queue_ack_sequence",
		"omc_pm_queue_sample_timestamp_seconds",
		"omc_pm_queue_sample_failures_total",
	} {
		assert.True(t, names[name], "queue metric %s must be registered", name)
	}
}

func TestNewEventBusMetrics_Registered(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)
	require.NotNil(t, m)
	assert.NotNil(t, m.DeliveryTotal)

	// CounterVec 无观测时 Gather 不返回，先打一个点再断言注册成功。
	m.inc("pm.file.received", deliveryOutcomeAck)
	families, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range families {
		names[f.GetName()] = true
	}
	assert.True(t, names["omc_eventbus_delivery_total"], "delivery counter should be registered")
}

func TestEventBusMetrics_Inc_ByOutcome(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)

	m.inc("alarm.raised", deliveryOutcomeTerminated)
	m.inc("alarm.raised", deliveryOutcomeTerminated)
	m.inc("alarm.raised", deliveryOutcomeDropped)

	assert.Equal(t, float64(2),
		testutil.ToFloat64(m.DeliveryTotal.WithLabelValues("alarm.raised", deliveryOutcomeTerminated)))
	assert.Equal(t, float64(1),
		testutil.ToFloat64(m.DeliveryTotal.WithLabelValues("alarm.raised", deliveryOutcomeDropped)))
}

func TestEventBusMetrics_ContractUsesOnlyLowCardinalityLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)

	m.observeQueueStats("acs:taskq:DEVICE-SN-001", "redis:key:acs:taskq:DEVICE-SN-001", QueueStats{SampledAt: time.Now()})
	m.observeQueueStats(SubjectPMFileReceived, pmQueueStatsDurable, QueueStats{SampledAt: time.Now()})
	m.observeQueueSampleFailure(SubjectPMFileReceived, pmQueueStatsDurable)

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				assert.NotContains(t, label.GetName(), "device_sn")
				assert.NotContains(t, label.GetName(), "redis_key")
				assert.NotContains(t, label.GetName(), "object_path")
				assert.False(t, strings.Contains(label.GetValue(), "DEVICE-SN-001"), "high-cardinality device value must not be a label")
				assert.False(t, strings.Contains(label.GetValue(), "acs:taskq:"), "complete Redis key must not be a label")
			}
		}
	}
}

// nil 安全：metrics 未注入（单进程 / 单测）时 inc 不 panic。
func TestEventBusMetrics_NilSafe(t *testing.T) {
	var m *EventBusMetrics
	assert.NotPanics(t, func() {
		m.inc("any.subject", deliveryOutcomeAck)
	})
}
