package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
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
//
// G8-Gap-1（启动补跑）：cronScheduler 启动前先扫 async_jobs_cron_state 表，对每个 job_type
// 算"上次成功触发到现在之间所有应触发但漏掉的 bucket"逐个 enqueue，保证 worker 停机期间
// 的 cron 触发不丢失。补跑后才启动 cron 走正常调度。
func startPMAggregatorPipeline(
	ctx context.Context,
	w *workerInfra,
	kpiRouter *router.Router,
	tz *tzManager,
	exportBucket string,
) {
	logger := w.Logger.Named("pm-aggregator")

	// 1) 构造 aggregator + asyncjob 基础设施
	aggr := aggregator.NewWithPool(w.TsPool, kpiRouter, logger)
	jobRepo := asyncjob.NewPgRepository(w.PgPool)
	cronStateRepo := asyncjob.NewPgCronStateRepository(w.PgPool)
	lockOwner := buildLockOwner()
	registry := asyncjob.NewRegistry(jobRepo, lockOwner, logger)

	// 1b) Prometheus 指标注册 + 注入（T-0164 P1 收尾：aggregator + asyncjob hooks 全注入）
	aggregatorMetrics := aggregator.NewMetrics(w.MetricsReg)
	asyncMetrics := asyncjob.NewMetrics(w.MetricsReg)
	registry.SetMetrics(asyncMetrics)

	// 2) 注册 4 个 G5 设备级 cron runner
	runners := []*aggregator.Runner{
		aggregator.NewHourlyRunner(aggr),
		aggregator.NewDailyRunner(aggr),
		aggregator.NewWeeklyRunner(aggr),
		aggregator.NewMonthlyRunner(aggr),
	}
	for _, r := range runners {
		r.SetMetrics(aggregatorMetrics)
		// #479 改动三：设备级该桶聚合成功后，确定性串联对应粒度的设备组聚合任务
		// （同一 [Start,End)，读到的必是已提交的完整设备级数据）。取代旧组级独立 cron + 10 分钟错峰。
		if gjt := aggregator.GroupJobTypeFor(r.JobType()); gjt != "" {
			r.SetGroupChain(gjt, jobRepo)
		}
		// #528 P1：设备级该桶聚合成功后推进设备级完成水位（水位表在主库，故传 PgPool）。
		r.SetWatermarkExec(w.PgPool)
		registry.Register(r)
		logger.Info("registered pm aggregator runner (device)",
			zap.String("job_type", r.JobType()),
			zap.String("source", r.Source()),
			zap.String("target", r.Target()),
			zap.String("chained_group_job_type", aggregator.GroupJobTypeFor(r.JobType())),
			zap.String("granularity", string(r.Granularity())))
	}

	// 2b) 注册 4 个 G5 设备组级 cron runner（T-0164 收尾 G5-Gap-1）
	groupRunners := []*aggregator.GroupRunner{
		aggregator.NewHourlyGroupRunner(aggr),
		aggregator.NewDailyGroupRunner(aggr),
		aggregator.NewWeeklyGroupRunner(aggr),
		aggregator.NewMonthlyGroupRunner(aggr),
	}
	for _, r := range groupRunners {
		r.SetMetrics(aggregatorMetrics)
		// #528 P1：组级该桶聚合成功后推进组级完成水位（水位表在主库，故传 PgPool）。
		r.SetWatermarkExec(w.PgPool)
		registry.Register(r)
		logger.Info("registered pm aggregator runner (group)",
			zap.String("job_type", r.JobType()),
			zap.String("device_target", r.DeviceTarget()),
			zap.String("group_target", r.GroupTarget()),
			zap.String("granularity", string(r.Granularity())))
	}

	// 2c) 注册 KPI-EXPORT 导出处理器（job_type=pm_kpi_export）。
	// T2 真生成：载任务 → running → 按 source_type 取数 → 流式写 CSV 直传对象存储 → 回填 succeeded。
	// KPI/时序库物理分离后池路由：
	//   - metricDB   = TsPool（PM 指标超表 pm_metrics*，dashboard device 维度直查 + 指标名解析）
	//   - adhocDB    = TsPool（pm_adhoc_aggregation_results 已迁时序库，直查 + 指标名解析）
	//   - taskMetaDB = PgPool（pm_tasks 留主库，loadAdhocDimension 取 adhoc 任务维度/设备数）
	//   - aggr 复用上面的 device 级聚合查询入口（dashboard 聚合维度 + KPI 反算）
	//   - bucket 复用报表桶（设计 §5.6）
	exportRunner := pmexport.NewRunner(pmexport.RunnerDeps{
		Repo:       pmexport.NewPgRepository(w.PgPool),
		Aggr:       aggr,
		MetricDB:   w.TsPool,
		AdhocDB:    w.TsPool,
		TaskMetaDB: w.PgPool,
		Uploader:   w.MinIO,
		Bucket:     exportBucket,
		Logger:     logger,
	})
	registry.Register(exportRunner)
	logger.Info("registered pm kpi export runner (T2)",
		zap.String("job_type", exportRunner.JobType()),
		zap.String("bucket", exportBucket))

	// 3) 启动 Sweeper（zombie reset + 心跳监控）
	// T-0164 收尾 G8-Gap-3：从 sys_configs 读 sweeper_interval / zombie_threshold / heartbeat_interval；
	// 缺失 / 解析失败 fallback 走 asyncjob 包默认值（与原行为一致）。
	// heartbeat_interval 写回 asyncjob.HeartbeatInterval（var），后续 Runner.runOnce 启 ticker 时读取。
	sweeperInterval, zombieThreshold := loadAsyncJobThresholds(ctx, w.PgPool, logger)
	sweeper := asyncjob.NewSweeper(jobRepo, sweeperInterval, zombieThreshold, logger)
	sweeper.SetMetrics(asyncMetrics)
	go sweeper.Run(ctx)
	logger.Info("asyncjob sweeper started",
		zap.Duration("interval", sweeperInterval),
		zap.Duration("zombie_threshold", zombieThreshold),
		zap.Duration("heartbeat_interval", asyncjob.HeartbeatInterval))

	// 3b) 启动 QueueDepthSampler — 每 30s 扫 async_jobs 表更新 omc_async_jobs_queue_depth gauge
	go asyncjob.RunQueueDepthSampler(ctx, jobRepo, asyncMetrics, 30*time.Second, &queueDepthSamplerLogger{logger: logger})
	logger.Info("asyncjob queue depth sampler started", zap.Duration("interval", 30*time.Second))

	// 4) 每个 JobType 开一个 worker goroutine（设备级 4 + 设备组级 4 = 8 个）
	for _, r := range runners {
		jt := r.JobType()
		go runJobTypeWorker(ctx, registry, jt, logger)
	}
	for _, r := range groupRunners {
		jt := r.JobType()
		go runJobTypeWorker(ctx, registry, jt, logger)
	}
	// KPI 导出处理器单独一个 worker goroutine（按需触发，无 cron）。
	go runJobTypeWorker(ctx, registry, exportRunner.JobType(), logger)

	// 5) 启动 cron 调度器（含启动补跑）。注册到 tzManager：改时区时随其余 cron 一并用新 loc 重建。
	startCronScheduler(ctx, jobRepo, cronStateRepo, logger, asyncMetrics, tz)

	// 6) PM retention cleanup（T-0164 收尾 G2-Gap-2）— 共享 jobRepo / cronStateRepo / registry / asyncMetrics
	startPMRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)

	// 7) 基站日志按时间保留清理（#320）— 复用同一 jobRepo / cronStateRepo / registry / asyncMetrics
	startStationLogRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)

	// 8) 审计/业务日志按时间保留清理（log.retention）— 复用同一 jobRepo / cronStateRepo / registry / asyncMetrics
	startLogRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)

	logger.Info("PM aggregator pipeline ready (8 aggregator runners + 3 retention runners + sweeper + cron triggers + catchup)")
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

// cronEntry 是单条 cron 调度配置（含启动补跑用的 advance）。
type cronEntry struct {
	spec    string
	jobType string
	window  func(now time.Time) (start, end time.Time)
	advance asyncjob.BucketAdvance
}

// pmAggregatorCronEntries 4 个 G5 设备级 cron 配置（#479 改动三后只剩设备级）。
//
// 设备组级聚合不再有独立 cron——改为设备级该桶聚合成功后由 Runner 确定性 chain
// （见 internal/pm/aggregator GroupJobTypeFor / Runner.SetGroupChain）。
// 旧的"设备组级 cron 晚 10 分钟错峰"已废弃（脆弱、两套游标漂移、白等延迟）。
//
//	设备级 hourly :05 / daily 00:05 / weekly Mon 00:10 / monthly 1日 00:15
//	→ 各自成功后立即 chain 出对应粒度的设备组聚合任务（同一 [Start,End)）。
// pmAggregatorCronEntries 接受固定 loc，等价 pmAggregatorCronEntriesFn(func() loc)。
// 保留此签名供既有单测直接断言某时区下的窗口边界。
func pmAggregatorCronEntries(loc *time.Location) []cronEntry {
	return pmAggregatorCronEntriesFn(func() *time.Location {
		if loc != nil {
			return loc
		}
		return time.UTC
	})
}

// pmAggregatorCronEntriesFn 用「实时取业务时区」的取值器构造 cron entries（#458 动态感知）。
//
// window 闭包每次触发都经 locFn() 取当前时区切桶——管理员改时区后，
// 即便 cron 还没重建完，下一次 window 计算也已用新时区（不重启即生效的核心保证）。
func pmAggregatorCronEntriesFn(locFn func() *time.Location) []cronEntry {
	loc := func() *time.Location {
		if l := locFn(); l != nil {
			return l
		}
		return time.UTC
	}
	hourlyAdvance := func(prev time.Time) time.Time { return prev.Add(time.Hour) }
	dailyAdvance := func(prev time.Time) time.Time { return prev.AddDate(0, 0, 1) }
	weeklyAdvance := func(prev time.Time) time.Time { return prev.AddDate(0, 0, 7) }
	monthlyAdvance := func(prev time.Time) time.Time { return prev.AddDate(0, 1, 0) }

	// hourly 整点对齐与时区无关（整点 UTC = 整点北京同一瞬间），不需 loc。
	hourlyWindow := func(now time.Time) (time.Time, time.Time) {
		end := now.Truncate(time.Hour)
		return end.Add(-time.Hour), end
	}
	// daily/weekly/monthly 把 now 归一到业务时区后再截零点，
	// 使 truncateDay/truncateWeekISO/monthlyWindow 经 t.Location() 自然产本地零点（T-0192）。
	dailyWindow := func(now time.Time) (time.Time, time.Time) {
		today := truncateDay(now.In(loc()))
		return today.AddDate(0, 0, -1), today
	}
	weeklyWindow := func(now time.Time) (time.Time, time.Time) {
		thisMon := truncateWeekISO(now.In(loc()))
		return thisMon.AddDate(0, 0, -7), thisMon
	}
	monthlyWindow := func(now time.Time) (time.Time, time.Time) {
		n := now.In(loc())
		thisMonth := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, n.Location())
		return thisMonth.AddDate(0, -1, 0), thisMonth
	}

	// #479 改动三：只保留设备级 cron。设备组聚合不再有独立 cron / 固定错峰——
	// 改为设备级该桶聚合成功后由 Runner.SetGroupChain 确定性 chain（同一 [Start,End)）。
	// 单一触发链：设备级 cron → 设备级任务成功 → chain 组任务；无两套游标漂移、无 10 分钟空等。
	// 启动补跑同理：设备级漏桶被 catchup 重跑时会重新 chain 出对应组任务，组级无需独立 catchup。
	return []cronEntry{
		// 设备级
		{spec: "5 * * * *", jobType: aggregator.JobTypeHourly, window: hourlyWindow, advance: hourlyAdvance},
		{spec: "5 0 * * *", jobType: aggregator.JobTypeDaily, window: dailyWindow, advance: dailyAdvance},
		{spec: "10 0 * * 1", jobType: aggregator.JobTypeWeekly, window: weeklyWindow, advance: weeklyAdvance},
		{spec: "15 0 1 * *", jobType: aggregator.JobTypeMonthly, window: monthlyWindow, advance: monthlyAdvance},
	}
}

// startCronScheduler 启动 robfig/cron/v3，按 wall-clock 时刻触发 enqueue。
//
// G8-Gap-1 实施：
//  1. 先做启动补跑（catchup）— 按 cron_state.last_bucket_end 算漏桶逐个 enqueue
//  2. 再启动正常 cron 调度
//
// 触发器只入队 async_jobs + Upsert cron_state，不直接调 runner
// （解耦：worker 重启 / 多 worker 时同样安全）。
func startCronScheduler(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	logger *zap.Logger,
	asyncMetrics *asyncjob.Metrics,
	tz *tzManager,
) {
	// window 闭包经 tz.Current() 实时取业务时区——改时区后下次触发即用新 loc 切桶。
	entries := pmAggregatorCronEntriesFn(tz.Current)

	// === 1. 启动补跑 ===（用启动当下的业务时区算漏桶）
	now := time.Now().In(tz.Current())
	for _, e := range entries {
		catchupCronEntry(ctx, jobRepo, stateRepo, e, now, logger, asyncMetrics)
	}

	// === 2. 注册 cron 构造器到 tzManager ===
	// WithLocation(loc)：cron spec 的 wall-clock 时刻按业务时区解释，不依赖容器 TZ env（T-0192）。
	// 改系统时区时，tzManager 会停掉本 cron 并用新 loc 重新调用本构造器重建（#458 动态感知）。
	tz.registerCron(func(loc *time.Location) (*cron.Cron, error) {
		c := cron.New(cron.WithLocation(loc))
		for _, e := range entries {
			entry := e // 闭包变量捕获
			if _, err := c.AddFunc(entry.spec, func() {
				triggerCron(ctx, jobRepo, stateRepo, entry, logger, tz.Current)
			}); err != nil {
				logger.Error("cron AddFunc failed",
					zap.String("spec", entry.spec),
					zap.String("job_type", entry.jobType),
					zap.Error(err))
			}
		}
		return c, nil
	})

	logger.Info("pm aggregator cron scheduler started",
		zap.String("timezone", tz.Current().String()),
		zap.Strings("schedules_device", []string{
			"hourly @:05", "daily 00:05", "weekly Mon 00:10", "monthly 1日 00:15",
		}),
		zap.String("schedules_group", "chained from device-level success (#479; no separate cron)"))
}

// catchupCronEntry 启动时补跑单个 cron job_type 的所有漏桶。
//
// 流程：
//  1. 查 cron_state — 没有则跳过（首次启动，等下次正常 cron 触发即可）
//  2. CatchupMissedBuckets 算 (last_bucket_end, now] 区间所有应触发但漏掉的 bucket
//  3. 逐个 enqueue（最多 10000 个上限保护）
//  4. 更新 cron_state.last_bucket_end 到最后一个补跑的 bucket 的 end
func catchupCronEntry(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	entry cronEntry,
	now time.Time,
	logger *zap.Logger,
	asyncMetrics *asyncjob.Metrics,
) {
	state, err := stateRepo.Get(ctx, entry.jobType)
	if err != nil {
		if err == asyncjob.ErrNoCronState {
			logger.Info("no cron state; skip catchup (first run)",
				zap.String("job_type", entry.jobType))
			return
		}
		logger.Warn("load cron state failed; skip catchup",
			zap.String("job_type", entry.jobType), zap.Error(err))
		return
	}
	if state.LastBucketEnd == nil {
		logger.Info("cron state has nil last_bucket_end; skip catchup",
			zap.String("job_type", entry.jobType))
		return
	}

	missed := asyncjob.CatchupMissedBuckets(*state.LastBucketEnd, now, entry.advance)
	if len(missed) == 0 {
		logger.Debug("no missed buckets",
			zap.String("job_type", entry.jobType),
			zap.Timep("last_bucket_end", state.LastBucketEnd))
		return
	}

	logger.Info("catching up missed cron buckets",
		zap.String("job_type", entry.jobType),
		zap.Int("count", len(missed)),
		zap.Time("first_start", missed[0].Start),
		zap.Time("last_end", missed[len(missed)-1].End))

	var lastEnd time.Time
	for _, b := range missed {
		enqueueAggregationJob(ctx, jobRepo, entry.jobType, b.Start, b.End, logger)
		asyncMetrics.IncCatchup(entry.jobType) // G8-Gap-4
		lastEnd = b.End
	}

	if !lastEnd.IsZero() {
		if err := stateRepo.Upsert(ctx, entry.jobType, entry.spec, now, lastEnd); err != nil {
			logger.Warn("upsert cron state after catchup failed",
				zap.String("job_type", entry.jobType), zap.Error(err))
		}
	}
}

// queueDepthSamplerLogger 把 asyncjob.Logger 适配到 zap.Logger（避免 asyncjob 包反向依赖 zap）。
type queueDepthSamplerLogger struct {
	logger *zap.Logger
}

func (q *queueDepthSamplerLogger) Warn(msg string, err error) {
	q.logger.Warn(msg, zap.Error(err))
}

// triggerCron 正常 cron 触发：入队 1 个 bucket + 更新 cron_state。
// locFn 实时取业务时区（#458），保证触发时刻用的是当前时区而非启动时的旧时区。
func triggerCron(
	ctx context.Context,
	jobRepo asyncjob.Repository,
	stateRepo asyncjob.CronStateRepository,
	entry cronEntry,
	logger *zap.Logger,
	locFn func() *time.Location,
) {
	loc := locFn()
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	start, end := entry.window(now)
	enqueueAggregationJob(context.Background(), jobRepo, entry.jobType, start, end, logger)

	// 用 background ctx 防 ctx 取消时丢状态更新（与 enqueue 一致）
	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := stateRepo.Upsert(bg, entry.jobType, entry.spec, now, end); err != nil {
		logger.Warn("upsert cron state after trigger failed",
			zap.String("job_type", entry.jobType), zap.Error(err))
	}
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

// loadAsyncJobThresholds 启动期从 sys_configs 读 sweeper_interval / zombie_threshold / heartbeat_interval。
//
// T-0164 收尾 G8-Gap-3：之前 asyncjob.SweeperInterval / ZombieThreshold / HeartbeatInterval 是包级常量；
// 现在让运维可改（启动生效，重启 worker 拾取新值；运行期热重载未做，按设计简化要求保留启动期注入）。
// heartbeat_interval 直接写回 asyncjob.HeartbeatInterval（var），其它两项作为返回值供 Sweeper 构造。
func loadAsyncJobThresholds(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) (time.Duration, time.Duration) {
	sweeperInterval := asyncjob.SweeperInterval
	zombieThreshold := asyncjob.ZombieThreshold

	if pool == nil {
		return sweeperInterval, zombieThreshold
	}
	repo := admin.NewPgSysConfigRepository(pool)
	if v := readSysConfigInt(ctx, repo, "asyncjob", "sweeper_interval_seconds"); v > 0 {
		sweeperInterval = time.Duration(v) * time.Second
	}
	if v := readSysConfigInt(ctx, repo, "asyncjob", "zombie_threshold_seconds"); v > 0 {
		zombieThreshold = time.Duration(v) * time.Second
	}
	if v := readSysConfigInt(ctx, repo, "asyncjob", "heartbeat_interval_seconds"); v > 0 {
		asyncjob.HeartbeatInterval = time.Duration(v) * time.Second
	}
	logger.Info("asyncjob thresholds loaded from sys_configs",
		zap.Duration("sweeper_interval", sweeperInterval),
		zap.Duration("zombie_threshold", zombieThreshold),
		zap.Duration("heartbeat_interval", asyncjob.HeartbeatInterval))
	return sweeperInterval, zombieThreshold
}

// readSysConfigInt 读单个 sys_configs (category, key) int 值；失败返 0。
func readSysConfigInt(ctx context.Context, repo *admin.PgSysConfigRepository, category, key string) int {
	if repo == nil {
		return 0
	}
	row, err := repo.GetByKey(ctx, category, key)
	if err != nil || row == nil || row.ValueType != "int" {
		return 0
	}
	v := 0
	for _, c := range row.Value {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + int(c-'0')
	}
	return v
}

// buildLockOwner 生成 hostname-pid 形式的锁所有者标识，便于运维排查"哪个 worker 抢到任务"。
func buildLockOwner() string {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
