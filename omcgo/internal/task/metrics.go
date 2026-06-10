package task

import "github.com/prometheus/client_golang/prometheus"

// TaskMetrics holds Prometheus metrics for the task queue module.
//
// P4 扩展（docs/design/mml-task-flow-design-20260424.md §3.3 C10）：
//   - CompletedTotal 新增 `source` 标签，便于按上游分组
//   - DurationSeconds 记录 device_task 从 sent → 终态的耗时，histogram
//   - NoHandlerTotal 记录 CompletionRouter 未匹配 source 的次数，帮助排查
//     漏注册问题
type TaskMetrics struct {
	PendingTotal     prometheus.Gauge
	CompletedTotal   *prometheus.CounterVec
	DurationSeconds  *prometheus.HistogramVec
	NoHandlerTotal   *prometheus.CounterVec

	// ReconcileTotal 记录 Reconciler 检测/修复的 Redis↔PG 状态分叉次数（#13）。
	// outcome 标签：
	//   - "repaired"      PG 滞后于 Redis 终态，已把 PG 同步到终态
	//   - "repair_failed" 检出分叉但 PG 回写失败（下一轮重试）
	// 持续大于 0 说明双写 sync 路径有非瞬态故障，需要排查。
	ReconcileTotal *prometheus.CounterVec

	// DualWriteFailTotal 记录双写中断（写一半失败）次数（#13），用于在分叉发生的
	// 第一现场可观测，而非等 Reconciler 事后对账才发现。op 标签标识失败发生在哪一步：
	//   - "create_rollback" CreateTask 入队失败回滚 PG.Delete 也失败 → PG pending 孤儿
	//   - "sync_terminal"   MarkTaskCompleted/Failed PG.Update 失败 → PG 滞后于 Redis 终态
	//   - "sync_sent"       MarkTaskSent PG.Update 失败 → PG 滞后于 Redis sent
	DualWriteFailTotal *prometheus.CounterVec
}

// NewTaskMetrics creates and registers task metrics.
func NewTaskMetrics(reg prometheus.Registerer) *TaskMetrics {
	m := &TaskMetrics{
		PendingTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_tasks_pending_total",
			Help: "Current number of pending tasks",
		}),
		CompletedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mml_task_total",
			Help: "Total number of device_task completions by source and terminal status",
		}, []string{"source", "status"}),
		DurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mml_task_duration_seconds",
			Help:    "device_task latency from sent to terminal state, by source",
			Buckets: []float64{0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 180, 600},
		}, []string{"source"}),
		NoHandlerTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "completion_no_handler_total",
			Help: "CompletionRouter received task events whose source has no registered handler",
		}, []string{"source"}),
		ReconcileTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "task_reconcile_total",
			Help: "Redis↔PG task state divergences detected/repaired by the reconciler, by outcome",
		}, []string{"outcome"}),
		DualWriteFailTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "task_dual_write_fail_total",
			Help: "Redis↔PG dual-write interruptions (one side failed), by operation",
		}, []string{"op"}),
	}

	reg.MustRegister(
		m.PendingTotal,
		m.CompletedTotal,
		m.DurationSeconds,
		m.NoHandlerTotal,
		m.ReconcileTotal,
		m.DualWriteFailTotal,
	)
	return m
}
