package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
)

// startPMAggregatorPipeline 仅保留 KPI 导出和通用异步任务维护。
// 自然桶、组聚合、补跑、水位和 adhoc worker 已由在线流式聚合替代，不再装配。
func startPMAggregatorPipeline(
	ctx context.Context,
	w *workerInfra,
	kpiRouter *router.Router,
	tz *tzManager,
	exportBucket string,
	emailSender *notification.EmailSender,
	zedSummaryConfig notification.StatusSummaryConfigRepository,
) {
	startPMExportOnly(ctx, w, kpiRouter, tz, exportBucket, emailSender, zedSummaryConfig)
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

func exportTimezoneProvider(tz *tzManager) pmexport.TimezoneProvider {
	if tz == nil {
		return nil
	}
	return tz.provider
}

// cronEntry 是单条 cron 调度配置（含启动补跑用的 advance）。
type cronEntry struct {
	spec    string
	jobType string
	window  func(now time.Time) (start, end time.Time)
	advance asyncjob.BucketAdvance
}

// pmAggregatorCronEntries 返回 PM 聚合独立 cron 配置。
//
// 设备组级聚合不再有独立 cron——改为设备级该桶聚合成功后由 Runner 确定性 chain
// （见 internal/pm/aggregator GroupJobTypeFor / Runner.SetGroupChain）。
// 旧的"设备组级 cron 晚 10 分钟错峰"已废弃（脆弱、两套游标漂移、白等延迟）。
//
//	只保留设备级 hourly :05；daily/weekly/monthly 由上游 bucket 成功后 chain。
//
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

// pmAggregatorCronEntriesFn 构造 cron entries（#458 保留 locFn 签名供 tzManager 重建 cron）。
// 当前只剩 hourly 独立 cron，窗口本身与业务时区无关；daily/weekly/monthly 的业务时区边界
// 在 Runner chain 时通过 tz.Current() 计算。
func pmAggregatorCronEntriesFn(locFn func() *time.Location) []cronEntry {
	_ = locFn
	hourlyAdvance := func(prev time.Time) time.Time { return prev.Add(time.Hour) }

	// hourly 整点对齐与时区无关（整点 UTC = 整点北京同一瞬间），不需 loc。
	hourlyWindow := func(now time.Time) (time.Time, time.Time) {
		end := now.Truncate(time.Hour)
		return end.Add(-time.Hour), end
	}

	// 独立 cron 只负责 hourly。hourly catchup 补出漏桶后，Runner 成功事件继续 chain
	// daily/weekly/monthly 和同粒度 group，避免下游早于最后源桶成功提交。
	return []cronEntry{
		{spec: "5 * * * *", jobType: aggregator.JobTypeHourly, window: hourlyWindow, advance: hourlyAdvance},
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
			"hourly @:05",
		}),
		zap.String("schedules_rollup", "daily/weekly/monthly chained from upstream bucket success"),
		zap.String("schedules_group", "chained from same-granularity device-level success (#479; no separate cron)"))
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
		if _, err := enqueueAggregationJob(ctx, jobRepo, entry.jobType, b.Start, b.End, logger); err != nil {
			logger.Warn("catchup enqueue failed; stop advancing cron state",
				zap.String("job_type", entry.jobType),
				zap.Time("bucket_start", b.Start),
				zap.Time("bucket_end", b.End),
				zap.Error(err))
			break
		}
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
	if _, err := enqueueAggregationJob(context.Background(), jobRepo, entry.jobType, start, end, logger); err != nil {
		logger.Warn("skip cron state update after enqueue failure",
			zap.String("job_type", entry.jobType),
			zap.Time("bucket_start", start),
			zap.Time("bucket_end", end),
			zap.Error(err))
		return
	}

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
func enqueueAggregationJob(ctx context.Context, repo asyncjob.Repository, jobType string, start, end time.Time, logger *zap.Logger) (uuid.UUID, error) {
	payload, err := aggregator.BuildPayload(start, end)
	if err != nil {
		logger.Error("build payload failed",
			zap.String("job_type", jobType), zap.Error(err))
		return uuid.Nil, err
	}
	jobID, err := repo.Insert(ctx, asyncjob.InsertRequest{
		JobType:     jobType,
		ScheduledAt: time.Now(),
		BucketStart: &start,
		BucketEnd:   &end,
		Payload:     payload,
	})
	if err != nil {
		logger.Error("enqueue aggregation job failed",
			zap.String("job_type", jobType),
			zap.Time("bucket_start", start),
			zap.Time("bucket_end", end),
			zap.Error(err))
		return uuid.Nil, err
	}
	logger.Info("aggregation job enqueued",
		zap.String("job_type", jobType),
		zap.String("job_id", jobID.String()),
		zap.Time("bucket_start", start),
		zap.Time("bucket_end", end))
	return jobID, nil
}

// ── time helper ───────────────────────────────────────────────────────────

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func dailyAggregationWindow(now time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	today := truncateDay(now.In(loc))
	return today.AddDate(0, 0, -1), today
}

func weeklyAggregationWindow(now time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	thisMon := truncateWeekISO(now.In(loc))
	return thisMon.AddDate(0, 0, -7), thisMon
}

func monthlyAggregationWindow(now time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	n := now.In(loc)
	thisMonth := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, n.Location())
	return thisMonth.AddDate(0, -1, 0), thisMonth
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
