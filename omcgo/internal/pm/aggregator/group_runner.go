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
// 注意调度时机：GroupRunner 的 cron 时刻应**晚于**对应 Runner 的 cron（避免读到未完成的
// device-level 聚合表）。worker/aggregator.go 中按 hourly:05 → hourly_group:15 错峰 10 分钟。
type GroupRunner struct {
	aggregator   *Aggregator
	jobType      string
	deviceTarget string // 上一级聚合表（如 pm_metrics_hourly）
	groupTarget  string // 本级 group 聚合表（如 pm_group_metrics_hourly）
	granularity  metrics.Granularity
}

// JobType / DeviceTarget / GroupTarget / Granularity — 暴露给外部（路由 + 日志）。
func (r *GroupRunner) JobType() string                  { return r.jobType }
func (r *GroupRunner) DeviceTarget() string             { return r.deviceTarget }
func (r *GroupRunner) GroupTarget() string              { return r.groupTarget }
func (r *GroupRunner) Granularity() metrics.Granularity { return r.granularity }

// Run 解析 payload 拿 [Start, End) → AggregateDeviceGroup 写 groupTarget。
//
// 不计算 KPI（设备组级 KPI 因依赖具体设备 productClass 路由，跨设备时语义复杂，留 v2）。
func (r *GroupRunner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	var p runPayload
	if len(job.Payload) > 0 {
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return nil, fmt.Errorf("group runner %s: unmarshal payload: %w", r.jobType, err)
		}
	}
	if p.Start.IsZero() || p.End.IsZero() {
		return nil, fmt.Errorf("group runner %s: payload missing start/end", r.jobType)
	}
	if !p.End.After(p.Start) {
		return nil, fmt.Errorf("group runner %s: end (%s) must be after start (%s)",
			r.jobType, p.End.Format(time.RFC3339), p.Start.Format(time.RFC3339))
	}

	w := WindowSpec{Granularity: r.granularity, Start: p.Start, End: p.End}

	rows, err := r.aggregator.AggregateDeviceGroup(ctx, r.deviceTarget, r.groupTarget, w)
	if err != nil {
		return nil, fmt.Errorf("group runner %s: aggregate device group: %w", r.jobType, err)
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
