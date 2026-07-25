package main

import (
	"context"
	"errors"
	"time"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	pmmetrics "github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"go.uber.org/zap"
)

func startPMAggregationStream(ctx context.Context, w *workerInfra, tz *tzManager) {
	cfg := pmstream.ConfigFromEnv()
	pmmetrics.AggregationOutboxEnabled = cfg.Enabled
	pmmetrics.AggregationMaxEventBytes = cfg.MaxEventBytes
	if !cfg.Enabled {
		w.Logger.Warn("PM streaming aggregation disabled")
		return
	}
	logger := w.Logger.Named("pm-streaming-aggregation")
	streamMetrics := pmstream.NewMetrics(w.MetricsReg)
	store := pmstream.NewRedisWindowStore(w.Redis, cfg.WindowTTL)
	if err := store.ValidateConfiguration(ctx); err != nil {
		streamMetrics.Ready.Set(0)
		logger.Error("PM streaming aggregation Redis configuration rejected", zap.Error(err))
		return
	}
	taskRepo := pmstream.NewPgTaskRepository(w.PgPool, w.EventBus)
	snapshot := pmstream.NewSnapshotStore(taskRepo, logger)
	if err := snapshot.Reload(ctx); err != nil {
		logger.Error("load initial PM aggregation task snapshot", zap.Error(err))
		return
	}
	matcher := pmstream.NewMatcher(tz.Current())
	windowRepo := pmstream.NewWindowRepository(w.TsPool)
	finalizer := pmstream.NewFinalizer(windowRepo, store, logger).
		SetConcurrency(cfg.FinalizeConcurrency).
		SetMetrics(streamMetrics)
	recovery := pmstream.NewRecovery(w.NATS.JS, windowRepo, store, snapshot, matcher, logger)
	if err := recovery.RestoreActiveWindows(ctx); err != nil {
		logger.Error("restore active PM aggregation windows", zap.Error(err))
	}
	consumer := pmstream.NewConsumer(
		w.EventBus, snapshot, matcher, windowRepo, store, finalizer, logger,
	).SetMetrics(streamMetrics)
	if setter, ok := w.EventBus.(interface {
		SetPullTuning(string, event.PullTuning)
	}); ok {
		setter.SetPullTuning(event.SubjectPMAggregationNormalized, event.PullTuning{
			BatchSize:     cfg.OutboxBatch,
			Concurrency:   cfg.ConsumerConcurrency,
			AckWait:       2 * time.Minute,
			MaxAckPending: cfg.ConsumerConcurrency * 4,
		})
	}
	if _, err := consumer.Subscribe(); err != nil {
		logger.Error("subscribe PM streaming aggregation", zap.Error(err))
		return
	}
	relay := pmstream.NewOutboxRelay(
		pmstream.NewOutboxRepository(w.TsPool), w.EventBus, logger,
	).SetBatch(cfg.OutboxBatch).SetMetrics(streamMetrics)
	scanner := pmstream.NewTimeoutScanner(windowRepo, finalizer, cfg.CloseGrace, logger)
	go func() {
		if err := relay.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("PM aggregation outbox relay stopped", zap.Error(err))
		}
	}()
	go snapshot.RunRefresh(ctx, time.Minute)
	go recovery.Run(ctx, time.Minute)
	go scanner.Run(ctx)
	streamMetrics.Ready.Set(1)
	logger.Info("PM streaming aggregation ready",
		zap.Duration("close_grace", cfg.CloseGrace),
		zap.Duration("window_ttl", cfg.WindowTTL))
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
	exportRunner := pmexport.NewRunner(pmexport.RunnerDeps{
		Repo: pmexport.NewPgRepository(w.PgPool), Aggr: aggr,
		MetricDB: w.TsPool, AdhocDB: w.TsPool, TaskMetaDB: w.PgPool,
		Uploader: w.MinIO, Bucket: exportBucket, Logger: logger,
		TimezoneProvider: exportTimezoneProvider(tz),
	})
	registry.Register(exportRunner)
	go runJobTypeWorker(ctx, registry, exportRunner.JobType(), logger)
	sweeperInterval, zombieThreshold := loadAsyncJobThresholds(ctx, w.PgPool, logger)
	sweeper := asyncjob.NewSweeper(jobRepo, sweeperInterval, zombieThreshold, logger)
	sweeper.SetMetrics(asyncMetrics)
	go sweeper.Run(ctx)
	go asyncjob.RunQueueDepthSampler(
		ctx, jobRepo, asyncMetrics, 30*time.Second, &queueDepthSamplerLogger{logger: logger},
	)
	cronStateRepo := asyncjob.NewPgCronStateRepository(w.PgPool)
	startStationLogRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)
	startLogRetentionCleanup(ctx, w, jobRepo, cronStateRepo, registry, asyncMetrics, tz)
	logger.Info("PM KPI export worker ready", zap.String("bucket", exportBucket))
}
