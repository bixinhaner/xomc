package task

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// PersistentQueueName constants are the bounded queue dimension for the
// unified persistent-queue metrics. Values must be registered queue names,
// never a device identity, complete Redis key, object path, or database row ID.
const (
	PersistentQueueDeviceTasks         = "device_tasks"
	PersistentQueueAsyncJobs           = "async_jobs"
	PersistentQueueParameterSyncOutbox = "parameter_sync_outbox"
	PersistentQueueNorthboundOutbox    = "northbound_outbox"
	PersistentQueuePMKPIExport         = "pm_kpi_export"
	PersistentQueueTraceExport         = "trace_export"
	PersistentQueueBackupTasks         = "backup_tasks"
	PersistentQueueDeadLetters         = "dead_letters"
	PersistentQueueStatusPending       = "pending"
	PersistentQueueStatusSent          = "sent"
	PersistentQueueStatusRunning       = "running"
	PersistentQueueStatusSucceeded     = "succeeded"
	PersistentQueueStatusFailed        = "failed"
	PersistentQueueStatusDeadLetter    = "dead_letter"
	PersistentQueueResultSucceeded     = "succeeded"
	PersistentQueueResultFailed        = "failed"
)

// PersistentQueueNames returns a fresh copy so callers cannot mutate the
// registry used by future metric constructors.
func PersistentQueueNames() []string {
	return []string{
		PersistentQueueDeviceTasks,
		PersistentQueueAsyncJobs,
		PersistentQueueParameterSyncOutbox,
		PersistentQueueNorthboundOutbox,
		PersistentQueuePMKPIExport,
		PersistentQueueTraceExport,
		PersistentQueueBackupTasks,
		PersistentQueueDeadLetters,
	}
}

// Persistent queue metric suffixes are shared by PM, task, and storage-backed
// queue observers. Units are encoded in the suffix where applicable.
const (
	PersistentQueueMetricPending               = "pending"
	PersistentQueueMetricOldestAgeSeconds      = "oldest_age_seconds"
	PersistentQueueMetricFailedTotal           = "failed_total"
	PersistentQueueMetricDeadLetterTotal       = "dead_letter_total"
	PersistentQueueMetricProcessedTotal        = "processed_total"
	PersistentQueueMetricObserverFailuresTotal = "observer_failures_total"
)

// PersistentQueueStatuses returns a fresh copy of the only status vocabulary
// for queue snapshots. Empty queues are represented by pending=0; observer
// failures retain the previous value and increment observer_failures_total.
func PersistentQueueStatuses() []string {
	return []string{
		PersistentQueueStatusPending,
		PersistentQueueStatusSent,
		PersistentQueueStatusRunning,
		PersistentQueueStatusSucceeded,
		PersistentQueueStatusFailed,
		PersistentQueueStatusDeadLetter,
	}
}

// Persistent queue labels are deliberately bounded. Do not add device_sn,
// redis_key, object_path, row ID, or other per-item dimensions.
const (
	PersistentQueueLabelQueue  = "queue"
	PersistentQueueLabelStatus = "status"
	PersistentQueueLabelResult = "result"
)

// PersistentQueueLabels contains only validated bounded label values. New
// persistent-queue metric constructors should obtain labels through
// NewPersistentQueueLabels rather than accepting arbitrary strings.
type PersistentQueueLabels struct {
	queue  string
	status string
	result string
}

// NewPersistentQueueLabels rejects unregistered queue, status, and result
// values before they can become Prometheus label values. result may be empty
// for metrics that do not use the result dimension.
func NewPersistentQueueLabels(queue, status, result string) (PersistentQueueLabels, error) {
	if !isPersistentQueueValue(queue, PersistentQueueNames()) {
		return PersistentQueueLabels{}, fmt.Errorf("invalid persistent queue %q", queue)
	}
	if !isPersistentQueueValue(status, PersistentQueueStatuses()) {
		return PersistentQueueLabels{}, fmt.Errorf("invalid persistent queue status %q", status)
	}
	if result != "" && result != PersistentQueueResultSucceeded && result != PersistentQueueResultFailed {
		return PersistentQueueLabels{}, fmt.Errorf("invalid persistent queue result %q", result)
	}
	return PersistentQueueLabels{queue: queue, status: status, result: result}, nil
}

func isPersistentQueueValue(value string, registered []string) bool {
	for _, candidate := range registered {
		if value == candidate {
			return true
		}
	}
	return false
}

// Values returns the validated queue, status, and optional result values in
// the order used by the bounded label contract.
func (l PersistentQueueLabels) Values() [3]string {
	return [3]string{l.queue, l.status, l.result}
}

// TaskMetrics holds Prometheus metrics for the task queue module.
//
// P4 扩展（docs/design/mml-task-flow-design-20260424.md §3.3 C10）：
//   - CompletedTotal 新增 `source` 标签，便于按上游分组
//   - DurationSeconds 记录 device_task 从 sent → 终态的耗时，histogram
//   - NoHandlerTotal 记录 CompletionRouter 未匹配 source 的次数，帮助排查
//     漏注册问题
type TaskMetrics struct {
	PendingTotal    prometheus.Gauge
	CompletedTotal  *prometheus.CounterVec
	DurationSeconds *prometheus.HistogramVec
	NoHandlerTotal  *prometheus.CounterVec

	// WakeDropped 记录 wakeDevice 因并发上界（背压）被丢弃的唤醒次数（issue #12）。
	// 持续增长说明唤醒并发上界偏低 / Connection Request 后端变慢，需调 wake_concurrency。
	WakeDropped prometheus.Counter

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

	// --- issue #20: 任务生命周期可观测 ---

	// BacklogTotal 是周期扫描观测到的活跃态(pending/sent)任务积压绝对快照。
	// 与 PendingTotal（增量计数，随 enqueue/complete 加减、可能因双写中断漂移）互补：
	// 本指标由 Reconciler 每轮用 PG 实查的活跃任务数 Set，是不会漂移的权威背压信号。
	// 持续走高说明派发跟不上入队（CPE 不上线 / Connection Request 失败 / 下发变慢）。
	BacklogTotal prometheus.Gauge

	// StaleDetectedTotal 记录检测到的"PG 活跃态 vs Redis 终态"分叉任务数（陈旧 PG 行）。
	// 与 ReconcileTotal{repaired/repair_failed}（修复结果）区分：本指标按"检出即计"，
	// 即便后续修复失败也先计入，反映分叉发生频率本身；持续 >0 说明双写 sync 路径
	// 有非瞬态故障，是 reconcile 修复动作的上游信号。
	StaleDetectedTotal prometheus.Counter

	// RecoveryActionTotal 记录各类恢复动作执行次数，按 action 标签：
	//   - "restore_pending"  worker 启动期 RestorePendingQueues 把 PG pending 重灌 Redis
	//   - "reconcile_repair" Reconciler 把 PG 滞后态同步到 Redis 终态
	// 让"系统自愈了多少次"可观测——平时应为 0/低频，突增说明上游有故障在被兜底掩盖。
	RecoveryActionTotal *prometheus.CounterVec
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
		WakeDropped: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_task_wake_dropped_total",
			Help: "Connection Request wake-ups dropped due to concurrency limit (backpressure)",
		}),
		ReconcileTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "task_reconcile_total",
			Help: "Redis↔PG task state divergences detected/repaired by the reconciler, by outcome",
		}, []string{"outcome"}),
		DualWriteFailTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "task_dual_write_fail_total",
			Help: "Redis↔PG dual-write interruptions (one side failed), by operation",
		}, []string{"op"}),
		BacklogTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_tasks_backlog_total",
			Help: "Active (pending/sent) task backlog observed by the latest reconcile scan (authoritative snapshot, does not drift).",
		}),
		StaleDetectedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_tasks_stale_detected_total",
			Help: "Tasks detected stale (PG active vs Redis terminal divergence), counted at detection regardless of repair outcome.",
		}),
		RecoveryActionTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_tasks_recovery_action_total",
			Help: "Self-healing recovery actions executed, by action (restore_pending/reconcile_repair).",
		}, []string{"action"}),
	}

	reg.MustRegister(
		m.PendingTotal,
		m.CompletedTotal,
		m.DurationSeconds,
		m.NoHandlerTotal,
		m.WakeDropped,
		m.ReconcileTotal,
		m.DualWriteFailTotal,
		m.BacklogTotal,
		m.StaleDetectedTotal,
		m.RecoveryActionTotal,
	)
	return m
}

// Recovery action 标签常量，避免散落字符串。
const (
	RecoveryActionRestorePending  = "restore_pending"
	RecoveryActionReconcileRepair = "reconcile_repair"
)
