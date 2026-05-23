package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
)

// startPMAggregatorPipeline wire 起 T-0164-P5 / G5 自然桶聚合 + T-0164-P8 / G8 asyncjob 框架。
//
// 三个组件协作：
//  1. asyncjob.Registry — 把 4 个 Runner（hourly/daily/weekly/monthly）注册进框架
//  2. asyncjob.Sweeper  — 后台扫描 zombie 任务，过期 reset 重试
//  3. cron Scheduler    — 整点对齐入队 async_jobs（payload 含本轮聚合时间窗）
//
// 每个 JobType 单独开一个 worker goroutine，每 5 秒尝试抢一个待执行任务。
// 单进程内串行（避免同 JobType 重复跑同一 bucket）；多 worker 跨进程靠
// LockNextPending 的 SKIP LOCKED 自然分配。
func startPMAggregatorPipeline(
	ctx context.Context,
	w *workerInfra,
	kpiRouter *router.Router,
) {
	logger := w.Logger.Named("pm-aggregator")

	// 1) 构造 aggregator + asyncjob 基础设施
	aggr := aggregator.NewWithPool(w.TsPool, kpiRouter, logger)
	jobRepo := asyncjob.NewPgRepository(w.PgPool)
	lockOwner := buildLockOwner()
	registry := asyncjob.NewRegistry(jobRepo, lockOwner, logger)

	// 2) 注册 4 个 G5 cron runner
	runners := []*aggregator.Runner{
		aggregator.NewHourlyRunner(aggr),
		aggregator.NewDailyRunner(aggr),
		aggregator.NewWeeklyRunner(aggr),
		aggregator.NewMonthlyRunner(aggr),
	}
	for _, r := range runners {
		registry.Register(r)
		logger.Info("registered pm aggregator runner",
			zap.String("job_type", r.JobType()),
			zap.String("source", r.Source()),
			zap.String("target", r.Target()),
			zap.String("granularity", string(r.Granularity())))
	}

	// 3) 启动 Sweeper（zombie reset + 心跳监控）
	sweeper := asyncjob.NewSweeper(jobRepo, 0, 0, logger)
	go sweeper.Run(ctx)
	logger.Info("asyncjob sweeper started",
		zap.Duration("interval", asyncjob.SweeperInterval),
		zap.Duration("zombie_threshold", asyncjob.ZombieThreshold))

	// 4) 每个 JobType 开一个 worker goroutine
	for _, r := range runners {
		jt := r.JobType()
		go runJobTypeWorker(ctx, registry, jt, logger)
	}

	// 5) 启动 cron 调度器，按 wall-clock 整点触发 enqueue
	startCronScheduler(ctx, jobRepo, logger)

	logger.Info("PM aggregator pipeline ready (4 runners + sweeper + cron triggers)")
}

// runJobTypeWorker 单 JobType 内串行循环 RunNext。
//
// 5 秒 tick 是个权衡：cron 触发频率最高 1/小时，5 秒延迟无业务影响；
// 比 1 秒减少 90% SELECT 噪音、比 30 秒响应快得多。
func runJobTypeWorker(ctx context.Context, registry *asyncjob.Registry, jobType string, logger *zap.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 一次 tick 内排空当前可用任务（cron 可能批量产了多个 bucket）
			for {
				ran, err := registry.RunNext(ctx, jobType)
				if err != nil {
					logger.Warn("asyncjob run next failed",
						zap.String("job_type", jobType), zap.Error(err))
					break
				}
				if !ran {
					break
				}
			}
		}
	}
}

// startCronScheduler 启动 robfig/cron/v3，按 wall-clock 时刻触发 enqueue。
//
// schedule（5 字段 cron 表达式：m h dom mon dow）：
//   - hourly  '5 * * * *'  每小时第 5 分
//   - daily   '5 0 * * *'  每日 00:05
//   - weekly  '10 0 * * 1' 每周一 00:10（dow=1 ISO Monday）
//   - monthly '15 0 1 * *' 每月 1 日 00:15
//
// 触发器只入队 async_jobs，不直接调 runner（解耦：worker 重启 / 多 worker 时同样安全）。
func startCronScheduler(ctx context.Context, repo asyncjob.Repository, logger *zap.Logger) {
	c := cron.New()
	mustAdd := func(spec, jobType string, window func(now time.Time) (time.Time, time.Time)) {
		_, err := c.AddFunc(spec, func() {
			now := time.Now().UTC()
			start, end := window(now)
			enqueueAggregationJob(context.Background(), repo, jobType, start, end, logger)
		})
		if err != nil {
			logger.Error("cron AddFunc failed",
				zap.String("spec", spec), zap.String("job_type", jobType), zap.Error(err))
		}
	}

	mustAdd("5 * * * *", aggregator.JobTypeHourly, func(now time.Time) (time.Time, time.Time) {
		// 触发时 now ≈ HH:05 — 处理 [HH-1:00, HH:00) 桶
		end := now.Truncate(time.Hour)
		start := end.Add(-time.Hour)
		return start, end
	})
	mustAdd("5 0 * * *", aggregator.JobTypeDaily, func(now time.Time) (time.Time, time.Time) {
		// 触发时 now ≈ 今日 00:05 — 处理 [昨日 00:00, 今日 00:00) 桶
		today := truncateDay(now)
		yesterday := today.AddDate(0, 0, -1)
		return yesterday, today
	})
	mustAdd("10 0 * * 1", aggregator.JobTypeWeekly, func(now time.Time) (time.Time, time.Time) {
		// 触发时 now ≈ 周一 00:10 — 处理 [上周一 00:00, 本周一 00:00) 桶
		thisMon := truncateWeekISO(now)
		lastMon := thisMon.AddDate(0, 0, -7)
		return lastMon, thisMon
	})
	mustAdd("15 0 1 * *", aggregator.JobTypeMonthly, func(now time.Time) (time.Time, time.Time) {
		// 触发时 now ≈ 本月 1 日 00:15 — 处理 [上月 1 日 00:00, 本月 1 日 00:00) 桶
		thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		lastMonth := thisMonth.AddDate(0, -1, 0)
		return lastMonth, thisMonth
	})

	c.Start()
	logger.Info("pm aggregator cron scheduler started",
		zap.Strings("schedules", []string{
			"hourly @:05", "daily 00:05", "weekly Mon 00:10", "monthly 1日 00:15",
		}))

	// ctx 取消时优雅停 cron（不阻塞 worker shutdown）
	go func() {
		<-ctx.Done()
		stopCtx := c.Stop()
		<-stopCtx.Done()
		logger.Info("pm aggregator cron scheduler stopped")
	}()
}

// enqueueAggregationJob 把 (jobType, start, end) 序列化成 async_jobs.payload 入队。
//
// 即使本 worker 没注册对应 runner（运维误配），Insert 仍会成功；其它 worker 实例
// 注册了对应 runner 的话会消费它。多 worker 跨进程跑同一 jobType 时 LockNextPending
// 的 SKIP LOCKED 保证一个 bucket 只被一个 worker 处理。
func enqueueAggregationJob(ctx context.Context, repo asyncjob.Repository, jobType string, start, end time.Time, logger *zap.Logger) {
	payload, err := aggregator.BuildPayload(start, end)
	if err != nil {
		logger.Error("build payload failed",
			zap.String("job_type", jobType), zap.Error(err))
		return
	}
	jobID, err := repo.Insert(ctx, asyncjob.InsertRequest{
		JobType:     jobType,
		ScheduledAt: time.Now(),
		Payload:     payload,
	})
	if err != nil {
		logger.Error("enqueue aggregation job failed",
			zap.String("job_type", jobType),
			zap.Time("bucket_start", start),
			zap.Time("bucket_end", end),
			zap.Error(err))
		return
	}
	logger.Info("aggregation job enqueued",
		zap.String("job_type", jobType),
		zap.String("job_id", jobID.String()),
		zap.Time("bucket_start", start),
		zap.Time("bucket_end", end))
}

// ── time helper ───────────────────────────────────────────────────────────

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// truncateWeekISO 把 t 截到本周周一 00:00（ISO 周历，周一为一周开始）。
//
// time.Weekday() 中 Sunday=0, Monday=1...Saturday=6 — 不是 ISO（ISO Monday=1...Sunday=7）。
// 需要 (wd+6) % 7 来算"距本周一的天数"。
func truncateWeekISO(t time.Time) time.Time {
	wd := int(t.Weekday())
	daysSinceMonday := (wd + 6) % 7 // Sun=6, Mon=0, Tue=1, ...
	monday := time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, t.Location())
	return monday
}

// buildLockOwner 生成 hostname-pid 形式的锁所有者标识，便于运维排查"哪个 worker 抢到任务"。
func buildLockOwner() string {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
