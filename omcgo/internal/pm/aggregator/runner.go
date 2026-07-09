package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// JobEnqueuer 是 Runner 串联设备组聚合时入队后继任务所需的最小依赖（#479 改动三）。
//
// 之所以抽成接口而不直接依赖 asyncjob.Repository：
//   - Runner 只需要"入队一个 pending 任务"这一个能力，不需要整个 Repository（锁/收尾等）；
//   - 单测可用轻量 stub 断言"是否、用什么 payload 入队了组任务"，无需起库。
//
// 由 worker 装配时注入（传 asyncjob 的 PgRepository 即满足）。
type JobEnqueuer interface {
	Insert(ctx context.Context, req asyncjob.InsertRequest) (uuid.UUID, error)
}

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

	// 设备级→设备组 确定性串联（#479 改动三）。
	// chainGroupJobType 非空且 enqueuer 非 nil 时，设备级该桶聚合**成功提交后**
	// 立即用同一 [Start,End) 入队对应的设备组聚合任务——组任务只在设备级该桶落库后才产生，
	// 因此读到的必是已提交的完整设备级数据。取代旧的"固定 10 分钟错峰猜测"（去掉组级独立 cron）。
	//
	// 百万级预留（不实做）：将来若设备级按分片并行跑同一桶，应改为"最后一个分片完成时才 chain
	// 组任务"或引入分片完成屏障计数；当前单任务串行下，本桶设备级 Run 返回即代表该桶设备级全部完成，
	// 直接 chain 即可。chain 点（此处）就是将来插入屏障的唯一位置。
	chainGroupJobType string
	enqueuer          JobEnqueuer

	// 设备级跨粒度确定性串联。只由上游桶成功提交后触发下游桶，避免 daily/weekly/monthly
	// 独立 cron 在源桶未完成时提前执行。业务边界由 locFn 给出；nil 时回落 UTC。
	rollupEnqueuer JobEnqueuer
	locFn          func() *time.Location

	// 完成水位写入（#528 P1）。watermarkExec 非 nil 时，设备级该桶聚合**成功提交后**
	// 把 (granularity, device) 水位 UPSERT 推进到本格起点（取 max 不回退、空格也推进）。
	// 语义是「已处理」非「有数据」：只要 Run 走到成功收尾就推进，与该桶是否产生聚合行无关。
	// 由 worker 装配时注入（传主库 PgPool）。nil 则退化为不写水位（不回归既有行为）。
	watermarkExec WatermarkExecer
}

// JobType 返回 asyncjob 注册主键。
func (r *Runner) JobType() string { return r.jobType }

// Source / Target / Granularity 暴露给外部（路由配置 / debug log 用）。
func (r *Runner) Source() string                   { return r.source }
func (r *Runner) Target() string                   { return r.target }
func (r *Runner) Granularity() metrics.Granularity { return r.granularity }

// SetMetrics 注入 Prometheus 指标采集器；nil 关闭采集（G5-Gap-3）。
func (r *Runner) SetMetrics(m *Metrics) { r.metrics = m }

// SetGroupChain 配置设备级→设备组的确定性串联（#479 改动三）。
//
// groupJobType 为对应粒度的设备组聚合 job_type（如 hourly → JobTypeHourlyGroup）；
// enq 为入队器（worker 装配时传 asyncjob 仓库）。两者任一为空则不串联（退化为无组聚合）。
func (r *Runner) SetGroupChain(groupJobType string, enq JobEnqueuer) {
	r.chainGroupJobType = groupJobType
	r.enqueuer = enq
}

// SetRollupChain 配置设备级跨粒度串联（hourly→daily，daily→weekly/monthly）。
func (r *Runner) SetRollupChain(enq JobEnqueuer, locFn func() *time.Location) {
	r.rollupEnqueuer = enq
	r.locFn = locFn
}

// SetWatermarkExec 注入完成水位写入执行体（#528 P1，传主库 PgPool）。
//
// 设备级该桶聚合成功后用它把 (granularity, device) 水位推进到本格起点。
// nil 关闭水位写入（退化为既有行为，不回归）。
func (r *Runner) SetWatermarkExec(exec WatermarkExecer) { r.watermarkExec = exec }

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
func (r *Runner) Run(ctx context.Context, job *asyncjob.Job) (result json.RawMessage, err error) {
	startedAt := time.Now()
	// G5-Gap-3: nil metrics 安全；defer 写 Duration / 终态 status
	var runStatus = "succeeded"
	var p runPayload
	var counterRows, kpiRows int
	chainedJobTypes := make([]string, 0, 3)
	chainedJobIDs := make([]string, 0, 3)
	defer func() {
		if err != nil {
			runStatus = "failed"
		}
		r.metrics.ObserveDuration(r.jobType, time.Since(startedAt).Seconds())
		r.metrics.IncRun(r.jobType, runStatus)
		if !p.Start.IsZero() || !p.End.IsZero() {
			fields := []zap.Field{
				zap.String("job_type", r.jobType),
				zap.String("job_id", job.ID.String()),
				zap.Time("bucket_start", p.Start),
				zap.Time("bucket_end", p.End),
				zap.Time("scheduled_at", job.ScheduledAt),
				zap.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
				zap.String("status", runStatus),
				zap.Int("attempt", job.Attempt),
				zap.Int("counter_rows", counterRows),
				zap.Int("kpi_rows", kpiRows),
				zap.Strings("chained_job_types", chainedJobTypes),
				zap.Strings("chained_job_ids", chainedJobIDs),
			}
			if err != nil {
				fields = append(fields, zap.Error(err))
				r.aggregator.logger.Warn("pm aggregation bucket completed", fields...)
				return
			}
			r.aggregator.logger.Info("pm aggregation bucket completed", fields...)
		}
	}()

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

	counterRows, err = r.aggregator.AggregateCounters(ctx, r.source, r.target, w)
	if err != nil {
		runStatus = "failed"
		return nil, fmt.Errorf("runner %s: aggregate counters: %w", r.jobType, err)
	}
	r.metrics.AddRows(r.jobType, "counter", counterRows)

	kpiRows, err = r.aggregator.AggregateKPIs(ctx, r.target, w)
	if err != nil {
		runStatus = "failed"
		return nil, fmt.Errorf("runner %s: aggregate kpis: %w", r.jobType, err)
	}
	r.metrics.AddRows(r.jobType, "kpi", kpiRows)

	// 更新桶滞后秒数（end 到现在的秒差），Grafana 用它判断是否堵塞
	r.metrics.SetBucketLag(r.jobType, time.Since(p.End).Seconds())

	// 设备级该桶已成功处理完（counter + KPI 均落库）——推进设备级完成水位（#528 P1）。
	// 语义是「已处理」非「有数据」：counterRows/kpiRows 可能为 0（空格），水位照样推进，
	// 否则下游会卡死在真正空的格上。UPSERT 取 max 不回退，重试/乱序补跑幂等。
	// 水位写在 chain 组任务之前：保证下游消费设备级水位时，组任务尚未必产出（device 先于 group）。
	if r.watermarkExec != nil {
		if werr := UpsertWatermark(ctx, r.watermarkExec, r.granularity, WatermarkLevelDevice, p.Start); werr != nil {
			runStatus = "failed"
			return nil, fmt.Errorf("runner %s: upsert device watermark: %w", r.jobType, werr)
		}
	}

	// 设备级该桶已成功提交（counter + KPI 均落库）——确定性串联设备组聚合（#479 改动三）。
	// 此时 chain，组任务必读到本桶完整的、已提交的设备级数据；不再靠固定错峰猜先后。
	// chain 入队失败让本设备级任务整体失败：asyncjob 会按 max_attempts 重试本桶（UPSERT 幂等、重复安全），
	// 重试时重新 chain——保证组任务不会因一次入队抖动而静默丢失（单一触发链、游标不漂移）。
	var chainedGroupJobID string
	if r.chainGroupJobType != "" && r.enqueuer != nil {
		gid, ierr := enqueueChainedBucket(ctx, r.enqueuer, r.chainGroupJobType, p.Start, p.End)
		if ierr != nil {
			runStatus = "failed"
			return nil, fmt.Errorf("runner %s: enqueue group chain %s: %w", r.jobType, r.chainGroupJobType, ierr)
		}
		chainedGroupJobID = gid.String()
		chainedJobTypes = append(chainedJobTypes, r.chainGroupJobType)
		chainedJobIDs = append(chainedJobIDs, chainedGroupJobID)
	}

	if r.rollupEnqueuer != nil {
		for _, next := range RollupBucketsFor(r.jobType, p.Start, p.End, r.location()) {
			nid, ierr := enqueueChainedBucket(ctx, r.rollupEnqueuer, next.JobType, next.Start, next.End)
			if ierr != nil {
				runStatus = "failed"
				return nil, fmt.Errorf("runner %s: enqueue rollup chain %s: %w", r.jobType, next.JobType, ierr)
			}
			chainedJobTypes = append(chainedJobTypes, next.JobType)
			chainedJobIDs = append(chainedJobIDs, nid.String())
		}
	}

	out := map[string]any{
		"counter_rows": counterRows,
		"kpi_rows":     kpiRows,
		"start":        p.Start.Format(time.RFC3339),
		"end":          p.End.Format(time.RFC3339),
		"source":       r.source,
		"target":       r.target,
		"granularity":  string(r.granularity),
	}
	if r.chainGroupJobType != "" {
		out["chained_group_job_type"] = r.chainGroupJobType
		out["chained_group_job_id"] = chainedGroupJobID
	}
	if len(chainedJobTypes) > 0 {
		out["chained_job_types"] = chainedJobTypes
		out["chained_job_ids"] = chainedJobIDs
	}
	result, err = json.Marshal(out)
	return result, err
}

func (r *Runner) location() *time.Location {
	if r.locFn != nil {
		if loc := r.locFn(); loc != nil {
			return loc
		}
	}
	return time.UTC
}

func enqueueChainedBucket(ctx context.Context, enq JobEnqueuer, jobType string, start, end time.Time) (uuid.UUID, error) {
	payload, err := BuildPayload(start, end)
	if err != nil {
		return uuid.Nil, err
	}
	bucketStart := start
	bucketEnd := end
	return enq.Insert(ctx, asyncjob.InsertRequest{
		JobType:     jobType,
		ScheduledAt: time.Now(),
		BucketStart: &bucketStart,
		BucketEnd:   &bucketEnd,
		Payload:     payload,
	})
}

// BuildPayload 是 cron 触发器装 async_jobs.payload 时的助手；保证字段名与 runPayload 对齐。
func BuildPayload(start, end time.Time) (json.RawMessage, error) {
	return json.Marshal(runPayload{Start: start, End: end})
}
