package trace

import "github.com/prometheus/client_golang/prometheus"

// Metrics 持有 TR069 报文跟踪模块的 5 个 Prometheus 指标（设计文档 §8.1）。
//
// 反例告警约定：
//   - trace_capture_latency_seconds P99 > 0.005（5ms）→ 触发 WARNING
//   - trace_messages_dropped_total 任何非零增量 → 触发 WARNING
//   - trace_active_tasks 持续超过 200 → 容量告警
type Metrics struct {
	// ActiveTasks 当前 running 状态任务数。
	ActiveTasks prometheus.Gauge

	// MessagesCapturedTotal 累计采集报文数（按方向）。
	MessagesCapturedTotal *prometheus.CounterVec

	// MessagesDroppedTotal 因队列满 / 解析失败 / DB 失败被丢弃的报文数（NATS 异常告警信号）。
	MessagesDroppedTotal *prometheus.CounterVec

	// StorageBytesTotal 存储量分布（按 inline / minio 维度）。worker 周期采样更新。
	StorageBytesTotal *prometheus.GaugeVec

	// CaptureLatencySeconds 拦截 hook 在 ACS 内的耗时直方图（P99 < 5ms 反例监控）。
	CaptureLatencySeconds prometheus.Histogram
}

// NewMetrics 构造并注册指标。reg=nil 时返回 nil（允许调用方在未启用 Prometheus 时跳过）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		return nil
	}
	m := &Metrics{
		ActiveTasks: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_trace_active_tasks",
			// L-6：worker Sweeper 周期采样维护，仅 worker 进程的样本可信；
			// app / acs 也注册这个 family 但永远 emit 0（注册表一致性）。
			// 运维侧用 {job="omc-worker"} filter。
			Help: "Number of TR069 message trace tasks currently in running state. Emitted by worker process (job=\"omc-worker\"); other processes register the family for registry uniformity but never observe samples.",
		}),
		MessagesCapturedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_trace_messages_captured_total",
			Help: "Total number of TR069 SOAP messages captured, by direction (in=CPE→ACS, out=ACS→CPE).",
		}, []string{"direction"}),
		MessagesDroppedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_trace_messages_dropped_total",
			Help: "Total TR069 trace messages dropped. Non-zero is an alert signal.",
		}, []string{"reason"}),
		StorageBytesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_trace_storage_bytes",
			Help: "TR069 trace payload storage usage by location (inline=PG TOAST, minio=trace-bulk bucket).",
		}, []string{"location"}),
		CaptureLatencySeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "omc_trace_capture_latency_seconds",
			// L-9：worker / app 也注册这个 metric 但永远 emit 0 — 是注册表一致性
			// 设计选择，运维侧用 {job="omc-acs"} filter 看真实分布。
			Help:    "TR069 trace capture hook latency. Emitted only by ACS process (job=\"omc-acs\"); other processes register the family for registry uniformity but never observe samples. P99 must stay < 0.005 (5ms).",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5},
		}),
	}
	reg.MustRegister(
		m.ActiveTasks,
		m.MessagesCapturedTotal,
		m.MessagesDroppedTotal,
		m.StorageBytesTotal,
		m.CaptureLatencySeconds,
	)
	// Prometheus 行为：CounterVec / GaugeVec 在首次 WithLabelValues 之前不会在
	// /metrics 输出整个 family（连 # HELP 都没有）。运营商监控 / 告警规则会因
	// "series 不存在" 沉默失败。这里 prime 所有已知 label 组合，让进程一启动
	// 就暴露完整 5 个 metric family（0 值也算合法 observation）。
	m.MessagesCapturedTotal.WithLabelValues(string(DirectionIn))
	m.MessagesCapturedTotal.WithLabelValues(string(DirectionOut))
	m.MessagesDroppedTotal.WithLabelValues(DropReasonQueueFull)
	m.MessagesDroppedTotal.WithLabelValues(DropReasonPublishFailed)
	m.MessagesDroppedTotal.WithLabelValues(DropReasonDecodeFailed)
	m.MessagesDroppedTotal.WithLabelValues(DropReasonInsertFailed)
	m.StorageBytesTotal.WithLabelValues(StorageInline)
	m.StorageBytesTotal.WithLabelValues(StorageMinIO)
	return m
}

// DropReason 丢弃报文原因枚举（指标 label 值）。
const (
	DropReasonQueueFull     = "queue_full"
	DropReasonPublishFailed = "publish_failed"
	DropReasonDecodeFailed  = "decode_failed"
	DropReasonInsertFailed  = "insert_failed"
)

// StorageLocation 存储位置枚举。
const (
	StorageInline = "inline"
	StorageMinIO  = "minio"
)
