package main

import (
	"context"
	"errors"
	"time"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/geofence"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	pmmetrics "github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/pm/reportsubscription"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"go.uber.org/zap"
)

const pmBuiltinInitialRetryInterval = 2 * time.Second
const pmBuiltinInitialRetryTimeout = time.Minute
const pmRuleCatalogRefreshInterval = 5 * time.Minute
const pmRedisSweepInterval = 30 * time.Second
const pmRedisSweepSafetyThreshold = 30 * time.Minute
const pmRedisSweepScanLimit = 512
const pmRedisSweepUnlinkBatch = 128
const pmPublishedVersionRepairInterval = time.Minute

type pmBuiltinReconcileFunc func(context.Context) (adhoc.BuiltinReconcileResult, error)
type pmSnapshotReloadFunc func(context.Context) error

func runPMBuiltinReconcile(
	ctx context.Context,
	reconcile pmBuiltinReconcileFunc,
	reload pmSnapshotReloadFunc,
) (adhoc.BuiltinReconcileResult, error) {
	result, reconcileErr := reconcile(ctx)
	var reloadErr error
	if result.Changed > 0 {
		reloadErr = reload(ctx)
	}
	return result, errors.Join(reconcileErr, reloadErr)
}

func reconcilePMBuiltinsUntilReady(
	ctx context.Context,
	reconcile pmBuiltinReconcileFunc,
	reload pmSnapshotReloadFunc,
	retryInterval time.Duration,
) (adhoc.BuiltinReconcileResult, error) {
	var lastResult adhoc.BuiltinReconcileResult
	var lastErr error
	for {
		lastResult, lastErr = runPMBuiltinReconcile(ctx, reconcile, reload)
		if lastErr == nil {
			return lastResult, nil
		}
		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastResult, errors.Join(lastErr, ctx.Err())
		case <-timer.C:
		}
	}
}

func startPMAggregationStream(ctx context.Context, w *workerInfra, tz *tzManager) {
	cfg := pmstream.ConfigFromEnv()
	pmmetrics.AggregationOutboxEnabled = cfg.Enabled
	pmmetrics.AggregationMaxEventBytes = cfg.MaxEventBytes
	pmstream.SetMaxRollupEventBytes(cfg.MaxEventBytes)
	if !cfg.Enabled {
		w.Logger.Warn("PM streaming aggregation disabled")
		return
	}
	logger := w.Logger.Named("pm-streaming-aggregation")
	streamMetrics := pmstream.NewMetrics(w.MetricsReg)
	store := newWorkerPMWindowStore(w, cfg.WindowTTL, streamMetrics).
		SetV2WriteEnabled(cfg.RedisV2WriteEnabled)
	if err := store.ValidateConfiguration(ctx); err != nil {
		streamMetrics.Ready.Set(0)
		logger.Error("PM streaming aggregation Redis configuration rejected", zap.Error(err))
		return
	}
	taskRepo := pmstream.NewPgTaskRepository(w.PgPool, w.EventBus)
	adhocRepo := adhoc.NewPgRepository(w.PgPool, w.TsPool).SetStreamingRepository(taskRepo)
	builtinReconciler := adhoc.NewBuiltinReconciler(adhocRepo)
	initialCtx, initialCancel := context.WithTimeout(ctx, pmBuiltinInitialRetryTimeout)
	initialResult, initialErr := reconcilePMBuiltinsUntilReady(
		initialCtx,
		builtinReconciler.Reconcile,
		func(context.Context) error { return nil },
		pmBuiltinInitialRetryInterval,
	)
	initialCancel()
	recordPMBuiltinReconcile(streamMetrics, initialResult)
	if initialErr != nil {
		logger.Error("reconcile initial built-in PM aggregation tasks",
			zap.Int("definitions", initialResult.Definitions),
			zap.Int("failed", initialResult.Failed),
			zap.Error(initialErr))
	}
	snapshot := pmstream.NewSnapshotStore(taskRepo, logger)
	if err := snapshot.Reload(ctx); err != nil {
		logger.Error("load initial PM aggregation task snapshot", zap.Error(err))
		return
	}
	store.SetSnapshot(snapshot)
	locationProvider := tz.Current
	logger.Info("PM streaming aggregation timezone provider ready",
		zap.String("timezone", locationProvider().String()))
	matcher := pmstream.NewMatcherWithLocationProvider(locationProvider)
	windowRepo := pmstream.NewWindowRepository(w.TsPool)
	if err := windowRepo.BackfillVersionMetadata(ctx, snapshot.Current()); err != nil {
		logger.Error("backfill PM aggregation window version metadata", zap.Error(err))
		return
	}
	finalizer := pmstream.NewFinalizer(windowRepo, store, logger).
		SetConcurrency(cfg.FinalizeConcurrency).
		SetSnapshot(snapshot).
		SetLocationProvider(locationProvider).
		SetMetrics(streamMetrics)
	recovery := pmstream.NewRecovery(w.NATS.JS, windowRepo, store, snapshot, matcher, logger)
	if err := recovery.RestoreActiveWindows(ctx); err != nil {
		logger.Error("restore active PM aggregation windows", zap.Error(err))
	}
	outboxRepo := pmstream.NewOutboxRepository(w.TsPool)
	rollupOutboxRepo := pmstream.NewRollupOutboxRepository(w.TsPool)
	if err := rollupOutboxRepo.EnsurePeriodRebuildIndex(ctx); err != nil {
		logger.Error("ensure PM rollup period rebuild index", zap.Error(err))
		return
	}
	if err := rollupOutboxRepo.EnsureRevisionCleanupIndexes(ctx); err != nil {
		logger.Error("ensure PM rollup revision cleanup indexes", zap.Error(err))
		return
	}
	rebuildRepo := pmstream.NewRebuildRepository(w.TsPool)
	consumer := pmstream.NewConsumer(
		w.EventBus, snapshot, matcher, windowRepo, store, finalizer, logger,
	).SetMetrics(streamMetrics).
		SetConsumeBarriers(outboxRepo, rollupOutboxRepo).
		SetRebuildRepository(rebuildRepo)
	if setter, ok := w.EventBus.(interface {
		SetPullTuning(string, event.PullTuning)
	}); ok {
		setter.SetPullTuning(event.SubjectPMAggregationNormalized, event.PullTuning{
			BatchSize:     cfg.OutboxBatch,
			Concurrency:   cfg.ConsumerConcurrency,
			AckWait:       2 * time.Minute,
			MaxAckPending: cfg.ConsumerConcurrency * 4,
		})
		for _, subject := range []string{
			event.SubjectPMAggregationHourlyRollup,
			event.SubjectPMAggregationDailyRollup,
		} {
			setter.SetPullTuning(subject, event.PullTuning{
				BatchSize:     cfg.OutboxBatch,
				Concurrency:   cfg.ConsumerConcurrency,
				AckWait:       2 * time.Minute,
				MaxAckPending: cfg.ConsumerConcurrency * 4,
			})
		}
	}
	if _, err := consumer.Subscribe(); err != nil {
		logger.Error("subscribe PM streaming aggregation", zap.Error(err))
		return
	}
	relay := pmstream.NewOutboxRelay(
		outboxRepo, w.EventBus, logger,
	).SetBatch(cfg.OutboxBatch).
		SetRetention(cfg.OutboxRetention, cfg.ReplayRetention).
		SetMetrics(streamMetrics)
	rollupRelay := pmstream.NewRollupOutboxRelay(
		rollupOutboxRepo, w.EventBus, logger,
	).SetBatch(cfg.OutboxBatch).SetMetrics(streamMetrics)
	scanner := pmstream.NewTimeoutScanner(windowRepo, finalizer, cfg.CloseGrace, logger).
		SetGranularityGrace(cfg.DailyCloseGrace, cfg.WeeklyCloseGrace, cfg.MonthlyCloseGrace).
		SetIncompleteDeviceHourReplay(recovery.ReplayDurableDeviceHour).
		SetMetrics(streamMetrics)
	rebuilder := pmstream.NewRebuilder(
		rebuildRepo, recovery, finalizer, store, rollupOutboxRepo, logger,
	).SetMetrics(streamMetrics)
	redisSweeper := pmstream.NewRedisStateSweeper(
		store, windowRepo, pmRedisSweepSafetyThreshold, streamMetrics, logger,
	)
	publishedVersionRepairer := pmstream.NewPublishedVersionRepairerWithLocationProvider(
		windowRepo, snapshot, locationProvider, logger,
	)
	go func() {
		if err := relay.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("PM aggregation outbox relay stopped", zap.Error(err))
		}
	}()
	go func() {
		if err := rollupRelay.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("PM aggregation rollup outbox relay stopped", zap.Error(err))
		}
	}()
	go runPMRuleCatalogRefreshLoop(
		ctx, builtinReconciler.Reconcile, snapshot.Reload, streamMetrics, logger,
	)
	go adhoc.NewPlannedEndScheduler(adhocRepo, time.Minute, logger).Run(ctx)
	go snapshot.RunRefresh(ctx, time.Minute)
	go recovery.Run(ctx, time.Minute)
	go rebuilder.Run(ctx)
	go scanner.Run(ctx)
	go publishedVersionRepairer.Run(ctx, pmPublishedVersionRepairInterval)
	go redisSweeper.Run(
		ctx, pmRedisSweepInterval, pmRedisSweepScanLimit, pmRedisSweepUnlinkBatch,
	)
	streamMetrics.Ready.Set(1)
	logger.Info("PM streaming aggregation ready",
		zap.String("timezone", locationProvider().String()),
		zap.Duration("close_grace", cfg.CloseGrace),
		zap.Duration("window_ttl", cfg.WindowTTL),
		zap.Bool("redis_v2_write_enabled", cfg.RedisV2WriteEnabled))
}

func newWorkerPMWindowStore(
	w *workerInfra,
	windowTTL time.Duration,
	metrics *pmstream.Metrics,
) *pmstream.RedisWindowStore {
	return pmstream.NewRedisWindowStore(w.PMRedis, windowTTL).SetMetrics(metrics)
}

// runPMRuleCatalogRefreshLoop refreshes immutable membership definitions only.
// It does not scan PM data, open aggregation jobs, or calculate any KPI.
func runPMRuleCatalogRefreshLoop(
	ctx context.Context,
	refresh pmBuiltinReconcileFunc,
	reload pmSnapshotReloadFunc,
	metrics *pmstream.Metrics,
	logger *zap.Logger,
) {
	ticker := time.NewTicker(pmRuleCatalogRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := runPMBuiltinReconcile(ctx, refresh, reload)
			recordPMBuiltinReconcile(metrics, result)
			if err != nil {
				logger.Error("refresh PM aggregation rule catalog",
					zap.Int("definitions", result.Definitions),
					zap.Int("failed", result.Failed),
					zap.Error(err))
				continue
			}
			if result.Changed > 0 {
				logger.Info("refreshed PM aggregation rule catalog",
					zap.Int("definitions", result.Definitions),
					zap.Int("changed", result.Changed),
					zap.Int("empty", result.Empty))
			}
		}
	}
}

func recordPMBuiltinReconcile(metrics *pmstream.Metrics, result adhoc.BuiltinReconcileResult) {
	metrics.BuiltinReconcileRunsTotal.Inc()
	metrics.BuiltinReconcileErrorsTotal.Add(float64(result.Failed))
	metrics.BuiltinVersionsChangedTotal.Add(float64(result.Changed))
	metrics.BuiltinDefinitionsEmpty.Set(float64(result.Empty))
}

func startPMExportOnly(
	ctx context.Context,
	w *workerInfra,
	kpiRouter *router.Router,
	tz *tzManager,
	exportBucket string,
) {
	logger := w.Logger.Named("pm-export")
	aggr := aggregator.NewWithPool(w.TsPool, kpiRouter, logger)
	sysConfig := admin.NewPgSysConfigRepository(w.PgPool)
	aggr.SetNumberProcessLookup(func(ctx context.Context) (string, error) {
		row, err := sysConfig.GetByKey(ctx, resultnorm.ConfigCategory, resultnorm.ConfigKey)
		if err != nil {
			if errors.Is(err, commonerrors.ErrNotFound) {
				return "", nil
			}
			return "", err
		}
		return row.Value, nil
	})
	jobRepo := asyncjob.NewPgRepository(w.PgPool)
	registry := asyncjob.NewRegistry(jobRepo, buildLockOwner(), logger)
	asyncMetrics := asyncjob.NewMetrics(w.MetricsReg)
	registry.SetMetrics(asyncMetrics)
	exportRepo := pmexport.NewPgRepository(w.PgPool)
	exportRunner := pmexport.NewRunner(pmexport.RunnerDeps{
		Repo: exportRepo, Aggr: aggr,
		MetricDB: w.TsPool, AdhocDB: w.TsPool, TaskMetaDB: w.PgPool,
		Uploader: w.MinIO, Bucket: exportBucket, Logger: logger,
		TimezoneProvider: exportTimezoneProvider(tz),
		StorageAdmission: w.StorageProtection,
	})
	registry.Register(exportRunner)
	go runJobTypeWorker(ctx, registry, exportRunner.JobType(), logger)

	// KPI 查询模板定时报表：调度器创建自然窗口 run/job，runner 复用现有 KPI CSV
	// 导出任务，成功后从对象存储流式读取为附件并通过共享 SMTP 发送。
	historyService := notification.NewHistoryService(notification.NewPgHistoryRepository(w.PgPool), logger)
	mailer := notification.NewMailer(nil, historyService, w.EmailTransport, logger)
	reportRepo := reportsubscription.NewPgRepository(w.PgPool)
	reportRunner := reportsubscription.NewRunner(reportsubscription.RunnerDeps{
		Repository:   reportRepo,
		Exports:      pmexport.NewService(exportRepo, jobRepo),
		ExportReader: exportRepo,
		Objects:      w.MinIO,
		Mailer:       mailer,
		History:      historyService,
		Logger:       logger,
	})
	registry.Register(reportRunner)
	go runJobTypeWorker(ctx, registry, reportRunner.JobType(), logger.Named("report-email"))
	reportLocation := func() *time.Location {
		if tz == nil {
			return time.UTC
		}
		return tz.Current()
	}
	reportScheduler := reportsubscription.NewScheduler(w.PgPool, reportLocation, logger)
	go reportScheduler.Run(ctx, 30*time.Second)
	alarmEmailRunner := alarm.NewEmailJobRunner(mailer, logger)
	registry.Register(alarmEmailRunner)
	go runJobTypeWorker(ctx, registry, alarmEmailRunner.JobType(), logger.Named("alarm-email"))
	geofenceRepository := geofence.NewPgRepository(w.PgPool)
	geofenceMetrics := geofence.NewBatchMetrics(w.MetricsReg)
	geofenceRunner := registerGeofenceManualBindRunner(
		registry,
		geofenceRepository,
		geofenceMetrics,
	)
	go runJobTypeWorker(
		ctx,
		registry,
		geofenceRunner.JobType(),
		logger.Named("geofence-batch"),
	)
	sweeperInterval, zombieThreshold := loadAsyncJobThresholds(ctx, w.PgPool, logger)
	sweeper := asyncjob.NewSweeper(jobRepo, sweeperInterval, zombieThreshold, logger)
	sweeper.SetMetrics(asyncMetrics)
	go sweeper.Run(ctx)
	go asyncjob.RunQueueDepthSampler(
		ctx, jobRepo, asyncMetrics, 30*time.Second, &queueDepthSamplerLogger{logger: logger},
	)
	cronStateRepo := asyncjob.NewPgCronStateRepository(w.PgPool)
	startPMRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)
	startStationLogRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)
	startLogRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)
	logger.Info("PM KPI export worker ready", zap.String("bucket", exportBucket))
}

func registerGeofenceManualBindRunner(
	registry *asyncjob.Registry,
	repository geofence.ManualBindRunnerRepository,
	metrics *geofence.BatchMetrics,
) *geofence.ManualBindRunner {
	runner := geofence.NewManualBindRunner(repository, metrics)
	registry.Register(runner)
	return runner
}
