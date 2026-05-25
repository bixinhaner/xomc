package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// Runner 是单粒度（hourly/daily/weekly/monthly）的 asyncjob.JobRunner 实现。
//
// 由 worker/main.go 创建 4 个实例（用 NewHourlyRunner / NewDailyRunner / ... 构造），
// 注册到 asyncjob.Registry；cron 触发器按 schedule 写入 async_jobs.payload
// 后 worker 的 RunNext goroutine 抢一个就跑一遍 Run。
//
// payload 形式：JSON `{"start":"2026-05-22T10:00:00Z","end":"2026-05-22T11:00:00Z"}`
// （cron 调度器按整点对齐生成，runner 不再做时间对齐，方便回填）。
type Runner struct {
	aggregator  *Aggregator
	jobType     string
	source      string
	target      string
	granularity metrics.Granularity
	metrics     *Metrics // G5-Gap-3 Prometheus hook；nil 则不记
}

// JobType 返回 asyncjob 注册主键。
func (r *Runner) JobType() string { return r.jobType }

// Source / Target / Granularity 暴露给外部（路由配置 / debug log 用）。
func (r *Runner) Source() string                   { return r.source }
func (r *Runner) Target() string                   { return r.target }
func (r *Runner) Granularity() metrics.Granularity { return r.granularity }

// SetMetrics 注入 Prometheus 指标采集器；nil 关闭采集（G5-Gap-3）。
func (r *Runner) SetMetrics(m *Metrics) { r.metrics = m }

// runPayload 是 cron 注入到 async_jobs.payload 的字段集。Start/End 半开区间 [Start, End)。
type runPayload struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Run 实现 asyncjob.JobRunner.Run。
//
// 步骤：
//  1. 解析 payload 拿 Start/End
//  2. AggregateCounters（source→target，CASE WHEN SUM/AVG/MAX）
//  3. AggregateKPIs（target 内 device → KPI route → arithmetic 求值）
//  4. 返回 {"counter_rows": N, "kpi_rows": M, "start": ..., "end": ...}
func (r *Runner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	startedAt := time.Now()
	// G5-Gap-3: nil metrics 安全；defer 写 Duration / 终态 status
	var runStatus = "succeeded"
	defer func() {
		r.metrics.ObserveDuration(r.jobType, time.Since(startedAt).Seconds())
		r.metrics.IncRun(r.jobType, runStatus)
	}()

	var p runPayload
	if len(job.Payload) > 0 {
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			runStatus = "failed"
			return nil, fmt.Errorf("runner %s: unmarshal payload: %w", r.jobType, err)
		}
	}
	if p.Start.IsZero() || p.End.IsZero() {
		runStatus = "failed"
		return nil, fmt.Errorf("runner %s: payload missing start/end", r.jobType)
	}
	if !p.End.After(p.Start) {
		runStatus = "failed"
		return nil, fmt.Errorf("runner %s: end (%s) must be after start (%s)",
			r.jobType, p.End.Format(time.RFC3339), p.Start.Format(time.RFC3339))
	}

	w := WindowSpec{Granularity: r.granularity, Start: p.Start, End: p.End}

	counterRows, err := r.aggregator.AggregateCounters(ctx, r.source, r.target, w)
	if err != nil {
		runStatus = "failed"
		return nil, fmt.Errorf("runner %s: aggregate counters: %w", r.jobType, err)
	}
	r.metrics.AddRows(r.jobType, "counter", counterRows)

	kpiRows, err := r.aggregator.AggregateKPIs(ctx, r.target, w)
	if err != nil {
		runStatus = "failed"
		return nil, fmt.Errorf("runner %s: aggregate kpis: %w", r.jobType, err)
	}
	r.metrics.AddRows(r.jobType, "kpi", kpiRows)

	// 更新桶滞后秒数（end 到现在的秒差），Grafana 用它判断是否堵塞
	r.metrics.SetBucketLag(r.jobType, time.Since(p.End).Seconds())

	out := map[string]any{
		"counter_rows": counterRows,
		"kpi_rows":     kpiRows,
		"start":        p.Start.Format(time.RFC3339),
		"end":          p.End.Format(time.RFC3339),
		"source":       r.source,
		"target":       r.target,
		"granularity":  string(r.granularity),
	}
	return json.Marshal(out)
}

// BuildPayload 是 cron 触发器装 async_jobs.payload 时的助手；保证字段名与 runPayload 对齐。
func BuildPayload(start, end time.Time) (json.RawMessage, error) {
	return json.Marshal(runPayload{Start: start, End: end})
}
