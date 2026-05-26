package task

import "github.com/prometheus/client_golang/prometheus"

// Metrics 收口 F05 MR 任务的可观测性指标。
//
// 全部通过 NewMetrics(reg) 注册到 app/worker 进程的 prometheus.Registerer。
// 命名遵循 omc_<domain>_<metric>_<unit> 风格，与已有 task / pm / alarm 指标对齐。
//
// 指标用途速览：
//
//	mr_task_dispatched_total{op=open|close,result=success|enqueue_failed|unsupport}
//	  每次 dispatcher.Open / Close 增计数，让运维看到下发量与成败分布。
//	mr_task_active_count
//	  当前 task_status='on' 任务数（Gauge，由 scheduler 巡检定期 Set）。
//	mr_file_uploaded_total{cell_code}
//	  每条 mr.file.uploaded 事件递增，单 cell 上报频率监控。
//	mr_heartbeat_missed_total
//	  scheduler 心跳巡检每次 missed +1（含未到阈值的）。
//	mr_heartbeat_abnormal_total
//	  跨过阈值切到 abnormal 时 +1（告警触发点）。
//	mr_files_cleaned_total{kind=minio|pg}
//	  cleaner 每次 RunOnce 删的对象 / PG 行数累计。
type Metrics struct {
	Dispatched       *prometheus.CounterVec
	ActiveCount      prometheus.Gauge
	FileUploaded     *prometheus.CounterVec
	HeartbeatMissed  prometheus.Counter
	HeartbeatAbnormal prometheus.Counter
	FilesCleaned     *prometheus.CounterVec
}

// NewMetrics 注册并返回所有 MR 任务相关指标。reg 为 nil 时返回 nil（调用方降级到无指标模式）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		return nil
	}
	m := &Metrics{
		Dispatched: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "omc_mr_task_dispatched_total",
				Help: "Total MR open/close SPV dispatches by result.",
			},
			[]string{"op", "result"},
		),
		ActiveCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "omc_mr_task_active_count",
				Help: "Current count of MR tasks in 'on' status.",
			},
		),
		FileUploaded: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "omc_mr_file_uploaded_total",
				Help: "Total MR files uploaded by devices, partitioned by cell_code.",
			},
			[]string{"cell_code"},
		),
		HeartbeatMissed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "omc_mr_heartbeat_missed_total",
				Help: "Total MR cell heartbeat misses detected by scheduler patrol (regardless of threshold).",
			},
		),
		HeartbeatAbnormal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "omc_mr_heartbeat_abnormal_total",
				Help: "Total MR cells flipped to abnormal due to consecutive missed heartbeats.",
			},
		),
		FilesCleaned: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "omc_mr_files_cleaned_total",
				Help: "Total MR files cleaned by daily cleaner (kind=minio for objects, kind=pg for rows).",
			},
			[]string{"kind"},
		),
	}
	reg.MustRegister(
		m.Dispatched,
		m.ActiveCount,
		m.FileUploaded,
		m.HeartbeatMissed,
		m.HeartbeatAbnormal,
		m.FilesCleaned,
	)
	return m
}

// 以下小方法是 nil-safe 帮手，让调用方无需重复 if m != nil 判断。

func (m *Metrics) IncDispatched(op, result string) {
	if m == nil {
		return
	}
	m.Dispatched.WithLabelValues(op, result).Inc()
}

func (m *Metrics) SetActiveCount(n float64) {
	if m == nil {
		return
	}
	m.ActiveCount.Set(n)
}

func (m *Metrics) IncFileUploaded(cellCode string) {
	if m == nil {
		return
	}
	m.FileUploaded.WithLabelValues(cellCode).Inc()
}

func (m *Metrics) IncHeartbeatMissed() {
	if m == nil {
		return
	}
	m.HeartbeatMissed.Inc()
}

func (m *Metrics) IncHeartbeatAbnormal() {
	if m == nil {
		return
	}
	m.HeartbeatAbnormal.Inc()
}

func (m *Metrics) AddFilesCleaned(kind string, n float64) {
	if m == nil || n <= 0 {
		return
	}
	m.FilesCleaned.WithLabelValues(kind).Add(n)
}
