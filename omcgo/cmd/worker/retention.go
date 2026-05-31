package main

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/retention"
)

// startPMRetentionCleanup wire 起 T-0164 收尾 G2-Gap-2：普通表（pm_metrics_{daily,weekly,monthly}
// + pm_group_metrics_{daily,weekly,monthly}）的定时清理。
//
// hypertable（pm_metrics 15min + pm_metrics_hourly + pm_group_metrics_hourly）走
// TimescaleDB add_retention_policy 自动 drop_chunks（已在 migration 内挂）；本函数仅负责
// 普通表 6 张的 DELETE 清理。
//
// 装配：
//   1. retention.Service（worker 进程本地实例，复用 app 端 SysConfigRepository → sys_configs 读取）
//   2. CleanupRunner（asyncjob.JobRunner）
//   3. 注册到 registry + 单独 worker goroutine
//   4. cron 每日 03:00 触发 enqueue（用 G8-Gap-1 cron_state 启动补跑机制）
func startPMRetentionCleanup(
	ctx context.Context,
	w *workerInfra,
	jobRepo asyncjob.Repository,
	cronStateRepo asyncjob.CronStateRepository,
	registry *asyncjob.Registry,
	asyncMetrics *asyncjob.Metrics,
	loc *time.Location,
) {
	logger := w.Logger.Named("pm-retention")

	// retention.Service 复用 G2 已建逻辑（与 app 端独立实例，缓存内存隔离）。
	// 每次 Run 时 service.Reload 拉最新 sys_configs 值，24h tick 一次成本可忽略。
	sysConfigRepo := admin.NewPgSysConfigRepository(w.PgPool)
	reader := &workerSysConfigReader{inner: sysConfigRepo}
	svc := retention.NewService(reader, logger)

	cleanupRunner := retention.NewCleanupRunner(w.TsPool, svc, logger)
	registry.Register(cleanupRunner)
	logger.Info("registered pm retention cleanup runner",
		zap.String("job_type", retention.JobTypeCleanup))

	// 单独 worker goroutine 抢 cleanup 任务
	go runJobTypeWorker(ctx, registry, retention.JobTypeCleanup, w.Logger.Named("pm-retention"))

	// cron 触发器（每日 03:00）+ 启动补跑
	startRetentionCleanupCron(ctx, jobRepo, cronStateRepo, logger, asyncMetrics, loc)

	logger.Info("PM retention cleanup pipeline ready (1 runner + 1 cron + catchup)")
}

// workerSysConfigReader 是 worker 端 retention.SysConfigReader 适配器
// （与 cmd/app/provider/pm_retention.go::adminSysConfigReader 结构一致，独立避免跨 cmd 引用）。
type workerSysConfigReader struct {
	inner admin.SysConfigRepository
}

func (a *workerSysConfigReader) GetByKey(ctx context.Context, category, key string) (*retention.SysConfigRow, error) {
	row, err := a.inner.GetByKey(ctx, category, key)
	if err != nil || row == nil {
		return nil, err
	}
	return &retention.SysConfigRow{Value: row.Value, ValueType: row.ValueType}, nil
}

// startRetentionCleanupCron 单 cron entry，每日 03:00 触发清理任务入队。
//
// 与 G5 cron 共享 cron_state 表（不同 job_type）+ 启动补跑机制（worker 停机几天再起来时
// 自动补跑漏掉的清理日，避免数据堆积）。
//
// 注意 retention cleanup 的"bucket"概念不是数据时间窗，而是"哪天该跑清理"。
// last_bucket_end 表示"上次清理完成时的截止日期 00:00"；补跑算法用 dailyAdvance。
func startRetentionCleanupCron(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	logger *zap.Logger,
	asyncMetrics *asyncjob.Metrics,
	loc *time.Location,
) {
	const spec = "0 3 * * *" // 每日 03:00（业务时区，T-0192）
	jobType := retention.JobTypeCleanup

	entry := cronEntry{
		spec:    spec,
		jobType: jobType,
		window: func(now time.Time) (time.Time, time.Time) {
			// cleanup 不读时间窗 payload（runner 内自己算 cutoff），但保持与 G5 同接口形态
			today := truncateDay(now.In(loc))
			return today.AddDate(0, 0, -1), today
		},
		advance: func(prev time.Time) time.Time { return prev.AddDate(0, 0, 1) },
	}

	// 启动补跑（complementary 防丢）
	now := time.Now().In(loc)
	catchupCronEntry(ctx, jobRepo, stateRepo, entry, now, logger, asyncMetrics)

	// 正常 cron（WithLocation 与业务时区一致，T-0192）
	c := cron.New(cron.WithLocation(loc))
	_, err := c.AddFunc(spec, func() {
		triggerCron(ctx, jobRepo, stateRepo, entry, logger, loc)
	})
	if err != nil {
		logger.Error("retention cleanup cron AddFunc failed", zap.Error(err))
		return
	}
	c.Start()
	logger.Info("pm retention cleanup cron started",
		zap.String("spec", spec),
		zap.String("job_type", jobType))

	go func() {
		<-ctx.Done()
		stopCtx := c.Stop()
		<-stopCtx.Done()
		logger.Info("pm retention cleanup cron stopped")
	}()
}
