package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// GroupRunner 是设备组维度聚合的 asyncjob.JobRunner 实现（T-0164 收尾 G5-Gap-1）。
//
// 与 Runner 的区别：
//   - Runner（设备维度）：source=15min/上级粒度表 → target=device-level 聚合表
//   - GroupRunner（设备组维度）：source=device-level 聚合表 → target=group-level 聚合表，
//     执行 JOIN devices + device_group_members 二阶段聚合
//
// 触发时机（#479 改动三）：GroupRunner 不再有独立 cron / 固定错峰。改为对应粒度的设备级
// Runner 该桶聚合**成功提交后**确定性 chain 出本组任务（同一 [Start,End)），因此读到的必是
// 已提交的完整 device-level 聚合数据。装配见 worker/aggregator.go（Runner.SetGroupChain）。
type GroupRunner struct {
	aggregator   *Aggregator
	jobType      string
	deviceTarget string // 上一级聚合表（如 pm_metrics_hourly）
	groupTarget  string // 本级 group 聚合表（如 pm_group_metrics_hourly）
	granularity  metrics.Granularity
	metrics      *Metrics // G5-Gap-3 Prometheus hook；nil 则不记

	// 完成水位写入（#528 P1）。watermarkExec 非 nil 时，组级该桶聚合**成功后**
	// 把 (granularity, group) 水位 UPSERT 推进到本格起点（取 max 不回退、空格也推进）。
	// 由 worker 装配时注入（传主库 PgPool）。nil 则退化为不写水位。
	watermarkExec WatermarkExecer
}

// SetWatermarkExec 注入组级完成水位写入执行体（#528 P1，传主库 PgPool）。nil 关闭水位写入。
func (r *GroupRunner) SetWatermarkExec(exec WatermarkExecer) { r.watermarkExec = exec }

// JobType / DeviceTarget / GroupTarget / Granularity — 暴露给外部（路由 + 日志）。
func (r *GroupRunner) JobType() string                  { return r.jobType }
func (r *GroupRunner) DeviceTarget() string             { return r.deviceTarget }
func (r *GroupRunner) GroupTarget() string              { return r.groupTarget }
func (r *GroupRunner) Granularity() metrics.Granularity { return r.granularity }

// SetMetrics 注入 Prometheus 指标采集器；nil 关闭采集（G5-Gap-3）。
func (r *GroupRunner) SetMetrics(m *Metrics) { r.metrics = m }

// Run 解析 payload 拿 [Start, End) → AggregateDeviceGroup 写 groupTarget。
//
// 不计算 KPI（设备组级 KPI 因依赖具体设备 productClass 路由，跨设备时语义复杂，留 v2）。
func (r *GroupRunner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	startedAt := time.Now()
	var runStatus = "succeeded"
	defer func() {
		r.metrics.ObserveDuration(r.jobType, time.Since(startedAt).Seconds())
		r.metrics.IncRun(r.jobType, runStatus)
	}()

	var p runPayload
	if len(job.Payload) > 0 {
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			runStatus = "failed"
			return nil, fmt.Errorf("group runner %s: unmarshal payload: %w", r.jobType, err)
		}
	}
	if p.Start.IsZero() || p.End.IsZero() {
		runStatus = "failed"
		return nil, fmt.Errorf("group runner %s: payload missing start/end", r.jobType)
	}
	if !p.End.After(p.Start) {
		runStatus = "failed"
		return nil, fmt.Errorf("group runner %s: end (%s) must be after start (%s)",
			r.jobType, p.End.Format(time.RFC3339), p.Start.Format(time.RFC3339))
	}

	w := WindowSpec{Granularity: r.granularity, Start: p.Start, End: p.End}

	rows, err := r.aggregator.AggregateDeviceGroup(ctx, r.deviceTarget, r.groupTarget, w)
	if err != nil {
		runStatus = "failed"
		return nil, fmt.Errorf("group runner %s: aggregate device group: %w", r.jobType, err)
	}
	r.metrics.AddRows(r.jobType, "group", rows)
	r.metrics.SetBucketLag(r.jobType, time.Since(p.End).Seconds())

	// 组级该桶已成功处理完——推进组级完成水位（#528 P1）。
	// 语义「已处理」非「有数据」：rows 可能为 0（空格），水位照样推进。UPSERT 取 max 不回退。
	if r.watermarkExec != nil {
		if werr := UpsertWatermark(ctx, r.watermarkExec, r.granularity, WatermarkLevelGroup, p.Start); werr != nil {
			runStatus = "failed"
			return nil, fmt.Errorf("group runner %s: upsert group watermark: %w", r.jobType, werr)
		}
	}

	out := map[string]any{
		"rows":          rows,
		"start":         p.Start.Format(time.RFC3339),
		"end":           p.End.Format(time.RFC3339),
		"device_target": r.deviceTarget,
		"group_target":  r.groupTarget,
		"granularity":   string(r.granularity),
	}
	return json.Marshal(out)
}
