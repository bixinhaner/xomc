package mmlstandardloader

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics — 4 个 Prometheus 指标，运维通过 Grafana 面板观察 Loader 状态。
//
// 决策依据：docs/design/mml-rebuild-plan-20260513.md §5.5（Q9 决议）。
//   - omc_mml_loader_status        gauge   0=fail / 1=ok 最近一次跑结果
//   - omc_mml_loader_last_run_at   gauge   unix 时间戳，秒
//   - omc_mml_loader_duration      histogram 单次跑耗时
//   - omc_mml_loader_total         counter labels: result, phase
//   - omc_mml_loader_rows          gauge   labels: table  最近成功跑的行数
type Metrics struct {
	Status    prometheus.Gauge
	LastRunAt prometheus.Gauge
	Duration  prometheus.Histogram
	Total     *prometheus.CounterVec
	Rows      *prometheus.GaugeVec
}

var registerOnce sync.Once

// NewMetrics 注册 metric 到 reg；reg=nil 时返回 no-op metric。
// 同一进程只能注册一次（防 Prometheus DuplicateRegistry panic）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Status: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_mml_loader_status",
			Help: "MML standard loader last run status (0=fail, 1=ok)",
		}),
		LastRunAt: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_mml_loader_last_run_at_seconds",
			Help: "MML standard loader last successful run unix timestamp",
		}),
		Duration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_mml_loader_duration_seconds",
			Help:    "MML standard loader run duration",
			Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
		}),
		Total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_mml_loader_total",
			Help: "MML standard loader run total by result and phase",
		}, []string{"result", "phase"}),
		Rows: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_mml_loader_rows_upserted",
			Help: "MML standard loader last successful upserted row count per table",
		}, []string{"table"}),
	}
	if reg != nil {
		registerOnce.Do(func() {
			reg.MustRegister(m.Status, m.LastRunAt, m.Duration, m.Total, m.Rows)
		})
	}
	return m
}

func (m *Metrics) recordSuccess(elapsed time.Duration, params, groups, commands int) {
	m.Status.Set(1)
	m.LastRunAt.SetToCurrentTime()
	m.Duration.Observe(elapsed.Seconds())
	m.Total.WithLabelValues("success", "all").Inc()
	m.Rows.WithLabelValues("mml_params").Set(float64(params))
	m.Rows.WithLabelValues("mml_command_groups").Set(float64(groups))
	m.Rows.WithLabelValues("mml_commands").Set(float64(commands))
}

func (m *Metrics) recordFailure(phase string) {
	m.Status.Set(0)
	m.Total.WithLabelValues("fail", phase).Inc()
}
