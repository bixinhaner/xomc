package event

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// EventBusMetrics holds Prometheus metrics for NATS JetStream delivery outcomes
// (issue #20). Before this, max-retries termination and parse-drop were only
// logged at ERROR level — a消息被吞掉后没有任何可告警信号，运维只能事后翻日志。
//
// 指标语义（按 subject 维度，低基数 = 业务主题固定枚举）：
//   - DeliveryTotal{subject,outcome}  每条消息的最终处置：
//     outcome=ack         handler 成功，消息确认
//     outcome=nak         handler 失败但未达上限，退避重投（会重复计数，每次投递一次）
//     outcome=terminated  handler 失败达 maxDeliveries，Term 终止（消息被丢弃，需告警）
//     outcome=dropped     消息体无法解析（永久错误），Term 丢弃（需告警）
//
// terminated / dropped 是"静默丢消息"的两条路径，告警阈值建议见 PR 遗留段。
type EventBusMetrics struct {
	DeliveryTotal *prometheus.CounterVec

	QueuePending                *prometheus.GaugeVec
	QueueAckPending             *prometheus.GaugeVec
	QueueRedelivered            *prometheus.GaugeVec
	QueueOldestAgeSeconds       *prometheus.GaugeVec
	QueueLastSequence           *prometheus.GaugeVec
	QueueAckSequence            *prometheus.GaugeVec
	QueueSampleTimestampSeconds *prometheus.GaugeVec
	QueueSampleFailures         *prometheus.CounterVec
}

const pmQueueStatsDurable = "pm-workers"

// NewEventBusMetrics creates and registers EventBus delivery metrics.
func NewEventBusMetrics(reg prometheus.Registerer) *EventBusMetrics {
	m := &EventBusMetrics{
		DeliveryTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_eventbus_delivery_total",
			Help: "NATS JetStream message delivery outcomes by subject and outcome (ack/nak/terminated/dropped).",
		}, []string{"subject", "outcome"}),
		QueuePending:                newQueueGauge("omc_pm_queue_pending", "JetStream messages pending delivery for the PM durable consumer."),
		QueueAckPending:             newQueueGauge("omc_pm_queue_ack_pending", "JetStream PM messages delivered but not yet acknowledged."),
		QueueRedelivered:            newQueueGauge("omc_pm_queue_redelivered", "JetStream PM messages currently marked for redelivery."),
		QueueOldestAgeSeconds:       newQueueGauge("omc_pm_queue_oldest_age_seconds", "Age in seconds of the oldest retained PM message while PM work is pending."),
		QueueLastSequence:           newQueueGauge("omc_pm_queue_last_sequence", "Latest sequence retained by the PM JetStream stream."),
		QueueAckSequence:            newQueueGauge("omc_pm_queue_ack_sequence", "Acknowledgement floor sequence for the PM durable consumer."),
		QueueSampleTimestampSeconds: newQueueGauge("omc_pm_queue_sample_timestamp_seconds", "Unix timestamp of the last successful PM queue sample."),
		QueueSampleFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_queue_sample_failures_total",
			Help: "Failed PM queue health sampling attempts.",
		}, []string{"subject", "durable"}),
	}
	m.QueueSampleTimestampSeconds.WithLabelValues(SubjectPMFileReceived, pmQueueStatsDurable).Set(0)
	if reg != nil {
		reg.MustRegister(
			m.DeliveryTotal,
			m.QueuePending,
			m.QueueAckPending,
			m.QueueRedelivered,
			m.QueueOldestAgeSeconds,
			m.QueueLastSequence,
			m.QueueAckSequence,
			m.QueueSampleTimestampSeconds,
			m.QueueSampleFailures,
		)
	}
	return m
}

func (m *EventBusMetrics) observeQueueSampleFailure(subject, durable string) {
	if m == nil || subject != SubjectPMFileReceived || durable != pmQueueStatsDurable {
		return
	}
	m.QueueSampleFailures.WithLabelValues(subject, durable).Inc()
}

// The ACS wiring supplies fixed PM subject and durable names, making these
// labels bounded rather than a reflection of untrusted event payloads.
func newQueueGauge(name, help string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, []string{"subject", "durable"})
}

// outcome 标签取值常量，避免散落字符串拼写漂移。
const (
	deliveryOutcomeAck        = "ack"
	deliveryOutcomeNak        = "nak"
	deliveryOutcomeTerminated = "terminated"
	deliveryOutcomeDropped    = "dropped"
)

// inc 是 EventBusMetrics 的 nil 安全自增入口：metrics 未注入时（单进程 / 单测）静默 no-op。
func (m *EventBusMetrics) inc(subject, outcome string) {
	if m == nil {
		return
	}
	m.DeliveryTotal.WithLabelValues(subject, outcome).Inc()
}

func (m *EventBusMetrics) observeQueueStats(subject, durable string, stats QueueStats) {
	if m == nil {
		return
	}
	// These are PM-only metric names. Do not let the generic QueueStats API turn
	// caller-provided subjects or durable names into an unbounded label space.
	if subject != SubjectPMFileReceived || durable != pmQueueStatsDurable {
		return
	}
	labels := []string{subject, durable}
	m.QueuePending.WithLabelValues(labels...).Set(float64(stats.Pending))
	m.QueueAckPending.WithLabelValues(labels...).Set(float64(stats.AckPending))
	m.QueueRedelivered.WithLabelValues(labels...).Set(float64(stats.Redelivered))
	m.QueueOldestAgeSeconds.WithLabelValues(labels...).Set(stats.OldestPendingAge.Seconds())
	m.QueueLastSequence.WithLabelValues(labels...).Set(float64(stats.LastSequence))
	m.QueueAckSequence.WithLabelValues(labels...).Set(float64(stats.AckSequence))
	m.QueueSampleTimestampSeconds.WithLabelValues(labels...).Set(float64(stats.SampledAt.UnixNano()) / float64(time.Second))
}
