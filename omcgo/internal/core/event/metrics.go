package event

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// EventBusMetrics holds Prometheus metrics for NATS JetStream delivery outcomes
// (issue #20). Before this, max-retries termination and parse-drop were only
// logged at ERROR level — a消息被吞掉后没有任何可告警信号，运维只能事后翻日志。
//
// 指标契约：subject 和 durable 只能来自已注册的固定业务枚举；禁止把
// device SN、完整 Redis key 或对象路径放入标签。单位和空值语义如下：
//   - *_pending、*_ack_pending、*_redelivered 是 gauge，单位为消息数，空队列明确暴露 0。
//   - *_oldest_age_seconds 是 gauge，单位为秒；无积压时为 0。
//   - *_sequence 是 gauge，单位为 JetStream sequence；采集失败保留上次值。
//   - *_sample_timestamp_seconds 是 gauge，单位为 Unix 秒；采集失败不更新。
//   - *_sample_failures_total 是 counter；采集失败时递增，不能用 0 伪造成功。
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
	AckFailures   *prometheus.CounterVec
	HandlerTime   *prometheus.HistogramVec
	LocalQueue    *prometheus.GaugeVec

	ConsumerPending                *prometheus.GaugeVec
	ConsumerAckPending             *prometheus.GaugeVec
	ConsumerRedelivered            *prometheus.GaugeVec
	ConsumerOldestAgeSeconds       *prometheus.GaugeVec
	ConsumerLastSequence           *prometheus.GaugeVec
	ConsumerAckSequence            *prometheus.GaugeVec
	ConsumerSampleTimestampSeconds *prometheus.GaugeVec
	ConsumerSampleFailures         *prometheus.CounterVec

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
		AckFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_eventbus_ack_failures_total",
			Help: "Failed NATS JetStream acknowledgement operations by subject, durable, action, and bounded error class.",
		}, []string{"subject", "durable", "action", "error_class"}),
		HandlerTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_eventbus_handler_duration_seconds",
			Help:    "NATS JetStream event handler duration by subject and durable.",
			Buckets: prometheus.DefBuckets,
		}, []string{"subject", "durable"}),
		LocalQueue: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_eventbus_local_queue_depth",
			Help: "Local keyed-dispatch queued and in-flight work by subject and durable.",
		}, []string{"subject", "durable"}),
		ConsumerPending:                newQueueGauge("omc_eventbus_consumer_pending", "JetStream messages pending delivery by observable durable consumer."),
		ConsumerAckPending:             newQueueGauge("omc_eventbus_consumer_ack_pending", "JetStream messages delivered but not yet acknowledged by observable durable consumer."),
		ConsumerRedelivered:            newQueueGauge("omc_eventbus_consumer_redelivered", "JetStream messages currently marked for redelivery by observable durable consumer."),
		ConsumerOldestAgeSeconds:       newQueueGauge("omc_eventbus_consumer_oldest_age_seconds", "Age in seconds of the oldest retained message for an observable durable consumer."),
		ConsumerLastSequence:           newQueueGauge("omc_eventbus_consumer_last_sequence", "Latest stream sequence retained for an observable durable consumer."),
		ConsumerAckSequence:            newQueueGauge("omc_eventbus_consumer_ack_sequence", "Acknowledgement floor stream sequence for an observable durable consumer."),
		ConsumerSampleTimestampSeconds: newQueueGauge("omc_eventbus_consumer_sample_timestamp_seconds", "Unix timestamp of the last successful observable durable consumer sample."),
		ConsumerSampleFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_eventbus_consumer_sample_failures_total",
			Help: "Failed observable durable consumer health sampling attempts.",
		}, []string{"subject", "durable"}),
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
			m.AckFailures,
			m.HandlerTime,
			m.LocalQueue,
			m.ConsumerPending,
			m.ConsumerAckPending,
			m.ConsumerRedelivered,
			m.ConsumerOldestAgeSeconds,
			m.ConsumerLastSequence,
			m.ConsumerAckSequence,
			m.ConsumerSampleTimestampSeconds,
			m.ConsumerSampleFailures,
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
	m.observeConsumerQueueSampleFailure(subject, durable)
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

func (m *EventBusMetrics) incAckFailure(subject, durable, action, errorClass string) {
	if m == nil {
		return
	}
	m.AckFailures.WithLabelValues(subject, durable, action, errorClass).Inc()
}

func (m *EventBusMetrics) observeHandlerDuration(subject, durable string, duration time.Duration) {
	if m == nil {
		return
	}
	m.HandlerTime.WithLabelValues(subject, durable).Observe(duration.Seconds())
}

func (m *EventBusMetrics) observeLocalQueueDepth(subject, durable string, depth int) {
	if m == nil {
		return
	}
	if depth < 0 {
		depth = 0
	}
	m.LocalQueue.WithLabelValues(subject, durable).Set(float64(depth))
}

func (m *EventBusMetrics) observeConsumerQueueSampleFailure(subject, durable string) {
	if m == nil || !observableConsumerQueue(subject, durable) {
		return
	}
	m.ConsumerSampleFailures.WithLabelValues(subject, durable).Inc()
}

func (m *EventBusMetrics) observeQueueStats(subject, durable string, stats QueueStats) {
	if m == nil {
		return
	}
	if observableConsumerQueue(subject, durable) {
		labels := []string{subject, durable}
		m.ConsumerPending.WithLabelValues(labels...).Set(float64(stats.Pending))
		m.ConsumerAckPending.WithLabelValues(labels...).Set(float64(stats.AckPending))
		m.ConsumerRedelivered.WithLabelValues(labels...).Set(float64(stats.Redelivered))
		m.ConsumerOldestAgeSeconds.WithLabelValues(labels...).Set(stats.OldestPendingAge.Seconds())
		m.ConsumerLastSequence.WithLabelValues(labels...).Set(float64(stats.LastSequence))
		m.ConsumerAckSequence.WithLabelValues(labels...).Set(float64(stats.AckSequence))
		m.ConsumerSampleTimestampSeconds.WithLabelValues(labels...).Set(float64(stats.SampledAt.UnixNano()) / float64(time.Second))
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

func observableConsumerQueue(subject, durable string) bool {
	if subject == SubjectPMFileReceived && durable == pmQueueStatsDurable {
		return true
	}
	return subject == SubjectCommandGetParamsResponse && durable != ""
}
