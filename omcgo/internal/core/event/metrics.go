package event

import "github.com/prometheus/client_golang/prometheus"

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
}

// NewEventBusMetrics creates and registers EventBus delivery metrics.
func NewEventBusMetrics(reg prometheus.Registerer) *EventBusMetrics {
	m := &EventBusMetrics{
		DeliveryTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_eventbus_delivery_total",
			Help: "NATS JetStream message delivery outcomes by subject and outcome (ack/nak/terminated/dropped).",
		}, []string{"subject", "outcome"}),
	}
	reg.MustRegister(m.DeliveryTotal)
	return m
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
