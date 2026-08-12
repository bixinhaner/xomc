package main

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	corelogger "github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/logretention"
	"github.com/omcgo/omcgo/internal/pm/retention"
	"github.com/omcgo/omcgo/internal/stationlog"
)

// startPMRetentionCleanup wires configured cleanup for compact Counter snapshots
// and mixed-granularity aggregation results. pm_files/pm_ingest_batches belong
// to the exact-path raw object cleaner; sparse hypertables use drop-chunk policy.
//
// 装配：
//  1. retention.Service（worker 进程本地实例，复用 app 端 SysConfigRepository → sys_configs 读取）
//  2. CleanupRunner（asyncjob.JobRunner）
//  3. 注册到 registry + 单独 worker goroutine
//  4. cron 每日 03:00 触发 enqueue（用 G8-Gap-1 cron_state 启动补跑机制）
func startPMRetentionCleanup(
	ctx context.Context,
	w *workerInfra,
	jobRepo asyncjob.Repository,
	cronStateRepo asyncjob.CronStateRepository,
	registry *asyncjob.Registry,
	asyncMetrics *asyncjob.Metrics,
	tz *tzManager,
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
	startRetentionCleanupCron(ctx, jobRepo, cronStateRepo, logger, asyncMetrics, tz)

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
	tz *tzManager,
) {
	const spec = "0 3 * * *" // 每日 03:00（业务时区，T-0192）
	jobType := retention.JobTypeCleanup

	entry := cronEntry{
		spec:    spec,
		jobType: jobType,
		window: func(now time.Time) (time.Time, time.Time) {
			// cleanup 不读时间窗 payload（runner 内自己算 cutoff），但保持与 G5 同接口形态。
			// 经 tz.Current() 实时取业务时区（#458 动态感知）。
			today := truncateDay(now.In(tz.Current()))
			return today.AddDate(0, 0, -1), today
		},
		advance: func(prev time.Time) time.Time { return prev.AddDate(0, 0, 1) },
	}

	// 启动补跑（complementary 防丢）
	now := time.Now().In(tz.Current())
	catchupCronEntry(ctx, jobRepo, stateRepo, entry, now, logger, asyncMetrics)

	// 注册到 tzManager：WithLocation 与业务时区一致（T-0192），改时区时随其余 cron 一并重建（#458）。
	tz.registerCron(func(loc *time.Location) (*cron.Cron, error) {
		c := cron.New(cron.WithLocation(loc))
		if _, err := c.AddFunc(spec, func() {
			triggerCron(ctx, jobRepo, stateRepo, entry, logger, tz.Current)
		}); err != nil {
			return nil, err
		}
		return c, nil
	})
	logger.Info("pm retention cleanup cron started",
		zap.String("spec", spec),
		zap.String("job_type", jobType))
}

// startStationLogRetentionCleanup wire 起 #320：基站日志按时间保留（默认 60 天）的定时清理。
//
// 与 PM retention 同范式：retention.Service 角色由 stationlog.RetentionPolicy 承担（从 sys_configs
// 读 stationlog.retention.max_retention_days / max_file_count，TTL 缓存）；CleanupRunner 按时间删
// station_fault_logs / station_running_logs 旧表，以及 backup_restore_file 中运行/故障日志任务文件
// 的过期 MinIO 对象 + PG 软删；调度周期从 stationlog.retention.cleanup_interval_minutes
// 读取，按周期触发 async job。
// 文件数配额仅针对故障日志文件（事件驱动，app 进程）并与本时间清理并存。
func startStationLogRetentionCleanup(
	ctx context.Context,
	w *workerInfra,
	jobRepo asyncjob.Repository,
	cronStateRepo asyncjob.CronStateRepository,
	registry *asyncjob.Registry,
	asyncMetrics *asyncjob.Metrics,
	tz *tzManager,
) {
	logger := w.Logger.Named("stationlog-retention")

	sysConfigRepo := admin.NewPgSysConfigRepository(w.PgPool)
	lookup := func(ctx context.Context, category, key string) (string, bool) {
		row, err := sysConfigRepo.GetByKey(ctx, category, key)
		if err != nil || row == nil {
			return "", false
		}
		return row.Value, true
	}
	policy := stationlog.NewRetentionPolicy(lookup, logger)

	faultRepo := stationlog.NewPgFaultRepository(w.PgPool)
	runningRepo := stationlog.NewPgRunningRepository(w.PgPool)
	runner := stationlog.NewCleanupRunner(faultRepo, runningRepo, w.MinIO, policy, logger)
	runner.SetTaskLogStore(&stationLogTaskFileStore{repo: backup.NewPgFileRepository(w.PgPool)})
	registry.Register(runner)
	logger.Info("registered stationlog retention cleanup runner",
		zap.String("job_type", stationlog.JobTypeStationLogCleanup))

	go runJobTypeWorker(ctx, registry, stationlog.JobTypeStationLogCleanup, logger)

	startStationLogCleanupCron(ctx, jobRepo, cronStateRepo, logger, asyncMetrics, tz, policy)

	logger.Info("stationlog retention cleanup pipeline ready (1 runner + interval scheduler)")
}

type stationLogTaskFileStore struct {
	repo backup.LogFileRetentionRepository
}

func (s *stationLogTaskFileStore) ListExpiredTaskLogs(ctx context.Context, cutoff time.Time, limit int) ([]stationlog.TaskLogFile, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	files, err := s.repo.ListExpiredStationLogFiles(ctx, cutoff, limit)
	if err != nil {
		return nil, err
	}
	out := make([]stationlog.TaskLogFile, 0, len(files))
	for _, f := range files {
		taskLog := stationlog.TaskLogFile{ID: f.ID}
		bucket, objectPath, splitErr := backup.SplitBucketAndPath(f.ObjectPath)
		if splitErr == nil {
			taskLog.Bucket = bucket
			taskLog.ObjectPath = objectPath
		}
		out = append(out, taskLog)
	}
	return out, nil
}

func (s *stationLogTaskFileStore) MarkTaskLogDeleted(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.MarkFileDeleted(ctx, id)
}

// startLogRetentionCleanup wire 起「审计/业务日志」按时间保留清理（internal/logretention）。
//
// 与 PM/stationlog retention 同范式：RetentionPolicy 从 sys_configs（category=log.retention）读
// 数据库日志统一保留天数（TTL 缓存）；CleanupRunner 对 9 张日志表（audit/ops_audit/login/oper/task/
// system/ne_message/event/northbound_api）批量 DELETE 过期行；cron 每日 05:00（与 PM 03:00 /
// stationlog 04:00 错峰）触发 + 启动补跑。全部 9 表都在主库（w.PgPool）。
func startLogRetentionCleanup(
	ctx context.Context,
	w *workerInfra,
	jobRepo asyncjob.Repository,
	cronStateRepo asyncjob.CronStateRepository,
	registry *asyncjob.Registry,
	asyncMetrics *asyncjob.Metrics,
	tz *tzManager,
) {
	logger := w.Logger.Named("log-retention")

	sysConfigRepo := admin.NewPgSysConfigRepository(w.PgPool)
	lookup := func(ctx context.Context, category, key string) (string, bool) {
		row, err := sysConfigRepo.GetByKey(ctx, category, key)
		if err != nil || row == nil {
			return "", false
		}
		return row.Value, true
	}
	policy := logretention.NewRetentionPolicy(lookup, logger)
	runner := logretention.NewCleanupRunner(w.PgPool, policy, logger)
	registry.Register(runner)
	logger.Info("registered log retention cleanup runner",
		zap.String("job_type", logretention.JobTypeLogRetentionCleanup))

	// 顺带为 worker 自身日志文件起轮转配置 watcher（log.rotation 热加载，复用同一 lookup）。
	corelogger.StartRotationConfigWatcher(ctx, lookup, logger)

	go runJobTypeWorker(ctx, registry, logretention.JobTypeLogRetentionCleanup, logger)

	startLogCleanupCron(ctx, jobRepo, cronStateRepo, logger, asyncMetrics, tz)

	logger.Info("log retention cleanup pipeline ready (1 runner + 1 cron + catchup)")
}

// startLogCleanupCron 单 cron entry，每日 05:00 触发审计/业务日志时间清理入队
// （与 PM 03:00 / stationlog 04:00 错峰；共享 cron_state 表的不同 job_type 行）。
func startLogCleanupCron(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	logger *zap.Logger,
	asyncMetrics *asyncjob.Metrics,
	tz *tzManager,
) {
	const spec = "0 5 * * *" // 每日 05:00（业务时区，与 PM/stationlog 错峰）
	jobType := logretention.JobTypeLogRetentionCleanup

	entry := cronEntry{
		spec:    spec,
		jobType: jobType,
		window: func(now time.Time) (time.Time, time.Time) {
			today := truncateDay(now.In(tz.Current()))
			return today.AddDate(0, 0, -1), today
		},
		advance: func(prev time.Time) time.Time { return prev.AddDate(0, 0, 1) },
	}

	now := time.Now().In(tz.Current())
	catchupCronEntry(ctx, jobRepo, stateRepo, entry, now, logger, asyncMetrics)

	tz.registerCron(func(loc *time.Location) (*cron.Cron, error) {
		c := cron.New(cron.WithLocation(loc))
		if _, err := c.AddFunc(spec, func() {
			triggerCron(ctx, jobRepo, stateRepo, entry, logger, tz.Current)
		}); err != nil {
			return nil, err
		}
		return c, nil
	})
	logger.Info("log retention cleanup cron started",
		zap.String("spec", spec), zap.String("job_type", jobType))
}

const (
	stationLogCleanupPollInterval       = time.Minute
	stationLogMinCleanupIntervalMinutes = 10
	stationLogMaxCleanupIntervalMinutes = 1440
)

// startStationLogCleanupCron 按 stationlog.retention.cleanup_interval_minutes 配置周期触发
// 基站日志时间清理入队。函数名保留 cron 语义：仍复用 async_jobs_cron_state 记录最近触发桶，
// 并复用 async_jobs 的 bucket 去重；worker 每轮调度前重新读取 RetentionPolicy（带 TTL）。
func startStationLogCleanupCron(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	logger *zap.Logger,
	_ *asyncjob.Metrics,
	tz *tzManager,
	policy *stationlog.RetentionPolicy,
) {
	if policy == nil {
		policy = stationlog.NewRetentionPolicy(nil, logger)
	}
	if tz == nil {
		tz = &tzManager{}
		tz.cur.Store(time.UTC)
	}

	go func() {
		triggerStationLogCleanupIfDue(ctx, jobRepo, stateRepo, policy.CleanupIntervalMinutes(ctx), time.Now().In(tz.Current()), logger, tz.Current)

		ticker := time.NewTicker(stationLogCleanupPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				triggerStationLogCleanupIfDue(ctx, jobRepo, stateRepo, policy.CleanupIntervalMinutes(ctx), now.In(tz.Current()), logger, tz.Current)
			}
		}
	}()

	logger.Info("stationlog retention cleanup interval scheduler started",
		zap.Duration("poll_interval", stationLogCleanupPollInterval),
		zap.String("job_type", stationlog.JobTypeStationLogCleanup))
}

func triggerStationLogCleanupIfDue(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	intervalMinutes int,
	now time.Time,
	logger *zap.Logger,
	locFn func() *time.Location,
) bool {
	entry := stationLogCleanupCronEntry(intervalMinutes, locFn)
	start, end := entry.window(now)

	state, err := stateRepo.Get(ctx, entry.jobType)
	switch {
	case err == nil && state.LastBucketEnd != nil && !state.LastBucketEnd.Before(end):
		return false
	case err == nil:
	case err == asyncjob.ErrNoCronState:
	default:
		logger.Warn("load stationlog cleanup cron state failed; skip trigger",
			zap.String("job_type", entry.jobType), zap.Error(err))
		return false
	}

	if _, err := enqueueAggregationJob(context.Background(), jobRepo, entry.jobType, start, end, logger); err != nil {
		logger.Warn("stationlog cleanup enqueue failed",
			zap.String("job_type", entry.jobType),
			zap.Time("bucket_start", start),
			zap.Time("bucket_end", end),
			zap.Error(err))
		return false
	}

	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := stateRepo.Upsert(bg, entry.jobType, entry.spec, now, end); err != nil {
		logger.Warn("upsert stationlog cleanup cron state failed",
			zap.String("job_type", entry.jobType), zap.Error(err))
	}
	return true
}

func stationLogCleanupCronEntry(intervalMinutes int, locFn func() *time.Location) cronEntry {
	if intervalMinutes < stationLogMinCleanupIntervalMinutes || intervalMinutes > stationLogMaxCleanupIntervalMinutes {
		intervalMinutes = stationlog.DefaultCleanupIntervalMinutes
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	return cronEntry{
		spec:    fmt.Sprintf("@every %dm", intervalMinutes),
		jobType: stationlog.JobTypeStationLogCleanup,
		window: func(now time.Time) (time.Time, time.Time) {
			loc := time.UTC
			if locFn != nil {
				loc = locFn()
			}
			if loc == nil {
				loc = time.UTC
			}
			end := truncateToContinuousInterval(now.In(loc), interval)
			return end.Add(-interval), end
		},
		advance: func(prev time.Time) time.Time { return prev.Add(interval) },
	}
}

func truncateToContinuousInterval(t time.Time, interval time.Duration) time.Time {
	if interval <= 0 {
		interval = time.Duration(stationlog.DefaultCleanupIntervalMinutes) * time.Minute
	}
	epoch := time.Unix(0, 0).UTC()
	elapsed := t.UTC().Sub(epoch)
	return epoch.Add((elapsed / interval) * interval)
}
