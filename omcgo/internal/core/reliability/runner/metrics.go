package runner

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
)

// Result 标识 retry 最终结果（不是每次 retry 计数，而是最终成功/失败）。
const (
	ResultSuccess = "success"
	ResultFailed  = "failed"
)

// Replay 标识 admin replay 请求的结果。
const (
	ReplayPublished = "published"
	ReplayFailed    = "failed"
)

// Metrics 是 worker retry / DLQ / replay 的 Prometheus 指标集合。
//
// 4 个指标按 PRD §7：
//
//	worker_retry_attempts_total {module, subject, result} — counter
//	worker_dlq_entries_total    {module, subject}         — counter
//	worker_dlq_replays_total    {module, subject, result} — counter
//	worker_dlq_size             {module}                  — gauge（懒拉取）
//
// nil 接收者降级到 no-op，便于轻量部署 / 测试不注册 registry。
type Metrics struct {
	retryAttempts *prometheus.CounterVec
	dlqEntries    *prometheus.CounterVec
	dlqReplays    *prometheus.CounterVec
	dlqSize       *prometheus.GaugeVec
}

// NewMetrics 注册 4 个指标到 reg。reg 为 nil 时不注册（仅允许零值降级）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		retryAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "worker_retry_attempts_total",
			Help: "Final retry result counter labelled by module/subject (success or failed).",
		}, []string{"module", "subject", "result"}),
		dlqEntries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "worker_dlq_entries_total",
			Help: "Count of events that exhausted retries and entered the dead-letter queue.",
		}, []string{"module", "subject"}),
		dlqReplays: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "worker_dlq_replays_total",
			Help: "Count of admin replay attempts, labelled by module/subject and result.",
		}, []string{"module", "subject", "result"}),
		dlqSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "worker_dlq_size",
			Help: "Current number of dead-letter records grouped by module (refreshed on demand).",
		}, []string{"module"}),
	}
	if reg != nil {
		reg.MustRegister(m.retryAttempts, m.dlqEntries, m.dlqReplays, m.dlqSize)
	}
	return m
}

// RecordRetryResult 记录最终 retry 结果。result ∈ {success, failed}。
func (m *Metrics) RecordRetryResult(module, subject, result string) {
	if m == nil {
		return
	}
	m.retryAttempts.WithLabelValues(module, subject, result).Inc()
}

// RecordDLQEntry 记录一条死信入库。
func (m *Metrics) RecordDLQEntry(module, subject string) {
	if m == nil {
		return
	}
	m.dlqEntries.WithLabelValues(module, subject).Inc()
}

// RecordReplayResult 记录 admin replay 的结果。result ∈ {published, failed}。
func (m *Metrics) RecordReplayResult(module, subject, result string) {
	if m == nil {
		return
	}
	m.dlqReplays.WithLabelValues(module, subject, result).Inc()
}

// SetDLQSize 设置某 module 的 DLQ 当前大小。由 admin handler / 监控 cron 在合适
// 时机调用 dlq.Repository.Count 后调用此方法刷新 gauge。
func (m *Metrics) SetDLQSize(module string, n int64) {
	if m == nil {
		return
	}
	m.dlqSize.WithLabelValues(module).Set(float64(n))
}

// RefreshDLQSize 是个便捷封装：从 repo Count 后刷新 gauge。
// 失败仅返回错误，不 panic；不抑制 prometheus gauge（保留前一次值）。
func (m *Metrics) RefreshDLQSize(ctx context.Context, repo dlq.Repository, module string) error {
	if m == nil || repo == nil {
		return nil
	}
	n, err := repo.Count(ctx, module)
	if err != nil {
		return err
	}
	m.SetDLQSize(module, n)
	return nil
}
