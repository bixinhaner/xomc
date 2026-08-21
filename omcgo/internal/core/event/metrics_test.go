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

func TestEventBusMetricsObserveCommandConsumerQueueStats(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)
	sampledAt := time.Date(2026, time.August, 21, 10, 0, 0, 0, time.UTC)

	m.observeQueueStats(SubjectCommandGetParamsResponse, "device-rpc-gpv", QueueStats{
		Pending:          5,
		AckPending:       1,
		Redelivered:      2,
		OldestPendingAge: 30 * time.Second,
		LastSequence:     99,
		AckSequence:      88,
		SampledAt:        sampledAt,
	})
	m.observeConsumerQueueSampleFailure(SubjectCommandGetParamsResponse, "device-rpc-gpv")

	labels := []string{SubjectCommandGetParamsResponse, "device-rpc-gpv"}
	assert.Equal(t, float64(5), testutil.ToFloat64(m.ConsumerPending.WithLabelValues(labels...)))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.ConsumerAckPending.WithLabelValues(labels...)))
	assert.Equal(t, float64(2), testutil.ToFloat64(m.ConsumerRedelivered.WithLabelValues(labels...)))
	assert.Equal(t, float64(30), testutil.ToFloat64(m.ConsumerOldestAgeSeconds.WithLabelValues(labels...)))
	assert.Equal(t, float64(99), testutil.ToFloat64(m.ConsumerLastSequence.WithLabelValues(labels...)))
	assert.Equal(t, float64(88), testutil.ToFloat64(m.ConsumerAckSequence.WithLabelValues(labels...)))
	assert.Equal(t, float64(sampledAt.Unix()), testutil.ToFloat64(m.ConsumerSampleTimestampSeconds.WithLabelValues(labels...)))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.ConsumerSampleFailures.WithLabelValues(labels...)))
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
		"omc_eventbus_consumer_pending",
		"omc_eventbus_consumer_ack_pending",
		"omc_eventbus_consumer_redelivered",
		"omc_eventbus_consumer_oldest_age_seconds",
		"omc_eventbus_consumer_last_sequence",
		"omc_eventbus_consumer_ack_sequence",
		"omc_eventbus_consumer_sample_timestamp_seconds",
		"omc_eventbus_consumer_sample_failures_total",
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
	m.incAckFailure("pm.file.received", "pm-workers", "ack", "other")
	m.observeHandlerDuration("pm.file.received", "pm-workers", time.Millisecond)
	m.observeLocalQueueDepth("pm.file.received", "pm-workers", 1)
	families, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range families {
		names[f.GetName()] = true
	}
	assert.True(t, names["omc_eventbus_delivery_total"], "delivery counter should be registered")
	assert.True(t, names["omc_eventbus_ack_failures_total"], "ack failure counter should be registered")
	assert.True(t, names["omc_eventbus_handler_duration_seconds"], "handler duration histogram should be registered")
	assert.True(t, names["omc_eventbus_local_queue_depth"], "local queue gauge should be registered")
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

func TestEventBusMetrics_IncAckFailure_ByActionAndClass(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)

	m.incAckFailure("device.command.get_params.response", "device-rpc-gpv", "ack", "other")
	m.incAckFailure("device.command.get_params.response", "device-rpc-gpv", "ack", "other")
	m.incAckFailure("device.command.get_params.response", "device-rpc-gpv", "in_progress", "deadline")

	assert.Equal(t, float64(2),
		testutil.ToFloat64(m.AckFailures.WithLabelValues("device.command.get_params.response", "device-rpc-gpv", "ack", "other")))
	assert.Equal(t, float64(1),
		testutil.ToFloat64(m.AckFailures.WithLabelValues("device.command.get_params.response", "device-rpc-gpv", "in_progress", "deadline")))
}

func TestEventBusMetrics_ContractUsesOnlyLowCardinalityLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)

	m.observeQueueStats("acs:taskq:DEVICE-SN-001", "redis:key:acs:taskq:DEVICE-SN-001", QueueStats{SampledAt: time.Now()})
	m.observeQueueStats(SubjectPMFileReceived, pmQueueStatsDurable, QueueStats{SampledAt: time.Now()})
	m.observeQueueSampleFailure(SubjectPMFileReceived, pmQueueStatsDurable)
	m.incAckFailure("device.command.get_params.response", "device-rpc-gpv", "ack", "other")

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
		m.incAckFailure("any.subject", "durable", "ack", "other")
		m.observeHandlerDuration("any.subject", "durable", time.Millisecond)
		m.observeLocalQueueDepth("any.subject", "durable", 1)
		m.observeConsumerQueueSampleFailure("any.subject", "durable")
		m.observeQueueStats("any.subject", "durable", QueueStats{})
	})
}
