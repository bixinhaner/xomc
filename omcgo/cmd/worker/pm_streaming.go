package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/geofence"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	pmmetrics "github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/pm/regularreport"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/omcgo/omcgo/internal/topology"
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
const pmStreamingLifecycleReconcileInterval = time.Minute

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
	lifecycleReconciler := adhoc.NewStreamingLifecycleReconciler(
		adhocRepo, pmStreamingLifecycleReconcileInterval, logger,
	)
	lifecycleCtx, lifecycleCancel := context.WithTimeout(ctx, time.Minute)
	lifecycleResult, lifecycleErr := lifecycleReconciler.Reconcile(lifecycleCtx)
	lifecycleCancel()
	if lifecycleErr != nil {
		logger.Warn("reconcile initial PM streaming task lifecycle",
			zap.Int("candidates", lifecycleResult.Candidates),
			zap.Int("reconciled", lifecycleResult.Reconciled),
			zap.Int("retired", lifecycleResult.Retired),
			zap.Int("failed", lifecycleResult.Failed),
			zap.Error(lifecycleErr))
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
	// 大数据升级时下面四步（元数据回填 / 活跃窗口恢复 / 补建两个索引）是启动期
	// 最重的一段，每步都可能分钟~小时级；逐步打耗时让"还在推进"可见，避免静默
	// 长跑被误判为卡死（且此时 metrics/healthz 已提前启动，探活不受影响）。
	backfillStart := time.Now()
	if err := windowRepo.BackfillVersionMetadata(ctx, snapshot.Current()); err != nil {
		logger.Error("backfill PM aggregation window version metadata",
			zap.Duration("duration", time.Since(backfillStart)), zap.Error(err))
		return
	}
	logger.Info("PM aggregation window version metadata backfill completed",
		zap.Duration("duration", time.Since(backfillStart)))
	finalizer := pmstream.NewFinalizer(windowRepo, store, logger).
		SetConcurrency(cfg.FinalizeConcurrency).
		SetSnapshot(snapshot).
		SetLocationProvider(locationProvider).
		SetMetrics(streamMetrics)
	recovery := pmstream.NewRecovery(w.NATS.JS, windowRepo, store, snapshot, matcher, logger).
		SetReplayRetention(cfg.ReplayRetention)
	restoreStart := time.Now()
	if err := recovery.RestoreActiveWindows(ctx); err != nil {
		logger.Error("restore active PM aggregation windows",
			zap.Duration("duration", time.Since(restoreStart)), zap.Error(err))
	} else {
		logger.Info("active PM aggregation windows restored",
			zap.Duration("duration", time.Since(restoreStart)))
	}
	outboxRepo := pmstream.NewOutboxRepository(w.TsPool)
	rollupOutboxRepo := pmstream.NewRollupOutboxRepository(w.TsPool)
	rebuildIndexStart := time.Now()
	if err := rollupOutboxRepo.EnsurePeriodRebuildIndex(ctx); err != nil {
		logger.Error("ensure PM rollup period rebuild index",
			zap.Duration("duration", time.Since(rebuildIndexStart)), zap.Error(err))
		return
	}
	logger.Info("PM rollup period rebuild index ensured",
		zap.Duration("duration", time.Since(rebuildIndexStart)))
	cleanupIndexStart := time.Now()
	if err := rollupOutboxRepo.EnsureRevisionCleanupIndexes(ctx); err != nil {
		logger.Error("ensure PM rollup revision cleanup indexes",
			zap.Duration("duration", time.Since(cleanupIndexStart)), zap.Error(err))
		return
	}
	logger.Info("PM rollup revision cleanup indexes ensured",
		zap.Duration("duration", time.Since(cleanupIndexStart)))
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
	).SetBatch(cfg.OutboxBatch).SetMetrics(streamMetrics)
	rollupRelay := pmstream.NewRollupOutboxRelay(
		rollupOutboxRepo, w.EventBus, logger,
	).SetBatch(cfg.OutboxBatch).SetMetrics(streamMetrics)
	scanner := pmstream.NewTimeoutScanner(windowRepo, finalizer, cfg.CloseGrace, logger).
		SetGranularityGrace(cfg.DailyCloseGrace, cfg.WeeklyCloseGrace, cfg.MonthlyCloseGrace).
		SetIncompleteDeviceHourVersionReplay(
			recovery.LoadDurableDeviceHourVersions,
			recovery.AccumulateDurableDeviceHourVersion,
		).
		SetMetrics(streamMetrics)
	rebuilder := pmstream.NewRebuilder(
		rebuildRepo, recovery, finalizer, store, rollupOutboxRepo, logger,
	).SetMetrics(streamMetrics)
	redisSweeper := pmstream.NewRedisStateSweeper(
		store, windowRepo, pmRedisSweepSafetyThreshold, streamMetrics, logger,
	)
	runtimeCleaner := pmstream.NewRuntimeCleaner(
		outboxRepo, rollupOutboxRepo, windowRepo, logger,
	).SetConfig(pmstream.RuntimeCleanupConfig{
		BatchSize: cfg.CleanupBatch, Interval: cfg.CleanupInterval,
		MaxDuration:     cfg.CleanupMaxDuration,
		OutboxRetention: cfg.OutboxRetention, ReplayRetention: cfg.ReplayRetention,
		VacuumEnabled: cfg.CleanupVacuum,
	}).SetMetrics(streamMetrics)
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
	go lifecycleReconciler.Run(ctx)
	go adhoc.NewPlannedEndScheduler(adhocRepo, time.Minute, logger).Run(ctx)
	go snapshot.RunRefresh(ctx, time.Minute)
	go recovery.Run(ctx, time.Minute)
	go rebuilder.Run(ctx)
	go scanner.Run(ctx)
	go publishedVersionRepairer.Run(ctx, pmPublishedVersionRepairInterval)
	go redisSweeper.Run(
		ctx, pmRedisSweepInterval, pmRedisSweepScanLimit, pmRedisSweepUnlinkBatch,
	)
	go runtimeCleaner.Run(ctx)
	streamMetrics.Ready.Set(1)
	logger.Info("PM streaming aggregation ready",
		zap.String("timezone", locationProvider().String()),
		zap.Duration("close_grace", cfg.CloseGrace),
		zap.Duration("window_ttl", cfg.WindowTTL),
		zap.Int("cleanup_batch", cfg.CleanupBatch),
		zap.Duration("cleanup_max_duration", cfg.CleanupMaxDuration),
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
	smtpConfig appconfig.SMTPConfig,
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
	exportRepository := pmexport.NewPgRepository(w.PgPool)
	exportRunner := pmexport.NewRunner(pmexport.RunnerDeps{
		Repo: exportRepository, Aggr: aggr,
		MetricDB: w.TsPool, AdhocDB: w.TsPool, TaskMetaDB: w.PgPool,
		Uploader: w.MinIO, Bucket: exportBucket, Logger: logger,
		TimezoneProvider: exportTimezoneProvider(tz),
		StorageAdmission: w.StorageProtection,
	})
	registry.Register(exportRunner)
	go runJobTypeWorker(ctx, registry, exportRunner.JobType(), logger)
	alarmEmailRepository := alarm.NewPgAlarmEmailRepository(w.PgPool)
	omcName := "OMC"
	if row, err := sysConfig.GetByKey(ctx, "basic", "mrOMCName"); err == nil {
		if configuredName := strings.TrimSpace(row.Value); configuredName != "" {
			omcName = configuredName
		}
	} else if !errors.Is(err, commonerrors.ErrNotFound) {
		logger.Warn("load OMC name for alarm email", zap.Error(err))
	}
	emailSender := notification.NewDynamicEmailSender(
		workerNotificationSMTPStore{repo: sysConfig},
		smtpOptionsFromWorkerConfig(smtpConfig),
		logger.Named("email"),
	)
	alarmEmailRunner := alarm.NewAlarmEmailJobRunner(
		alarmEmailRepository,
		alarm.NewPgAlarmEmailReader(w.PgPool, w.TsPool),
		emailSender,
		omcName,
		tz.Current(),
		logger,
	)
	runtimeScopeAuthorizer := authz.NewRuntimeDeviceScopeAuthorizer(
		admin.NewPgUserRepository(w.PgPool),
		admin.NewPermissionService(
			admin.NewPgRoleRepository(w.PgPool),
			topology.NewPgDeviceGroupRepository(w.PgPool),
			w.Redis,
			logger.Named("email-runtime-authz"),
		),
		device.NewPgDeviceGroupReader(w.PgPool),
		device.NewPgDeviceRepository(w.PgPool),
	)
	alarmEmailRunner.SetLocationProvider(tz.Current)
	alarmEmailRunner.SetScopeAuthorizer(runtimeScopeAuthorizer)
	registry.Register(alarmEmailRunner)
	go runJobTypeWorker(ctx, registry, alarmEmailRunner.JobType(), logger.Named("alarm-email"))
	go alarm.NewAlarmEmailScheduler(alarmEmailRepository, logger).Run(ctx, time.Minute)
	if err := alarm.NewAlarmEmailRealtimeSubscriber(alarmEmailRepository, logger).Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe realtime alarm email", zap.Error(err))
	}
	kpiReportRepository := regularreport.NewPgRepository(w.PgPool)
	kpiReportRunner := regularreport.NewRunner(
		kpiReportRepository,
		exportRepository,
		exportRunner,
		regularreport.NewMinIOAttachmentStore(w.MinIO),
		emailSender,
		omcName,
		tz.Current,
		logger,
	)
	kpiReportRunner.SetScopeAuthorizer(runtimeScopeAuthorizer)
	registry.Register(kpiReportRunner)
	go runJobTypeWorker(ctx, registry, kpiReportRunner.JobType(), logger.Named("kpi-regular-report"))
	pmReadiness := pmstream.ConfigFromEnv()
	go regularreport.NewScheduler(kpiReportRepository, tz.Current, logger).
		SetReadinessGrace(pmReadiness.CloseGrace, pmReadiness.DailyCloseGrace).
		Run(ctx, time.Minute)
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
