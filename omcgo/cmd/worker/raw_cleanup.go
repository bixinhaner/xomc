package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
	gonats "github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/rawcleanup"
	"go.uber.org/zap"
)

type queueStatsProvider interface {
	QueueStats(context.Context, string, string) (event.QueueStats, error)
}

func startRawObjectCleanup(ctx context.Context, w *workerInfra, cfg *appconfig.WorkerConfig) {
	cleanupCfg := cfg.RawCleanup.Defaults()
	if !cleanupCfg.Enabled {
		w.Logger.Info("precise raw object cleanup disabled")
		return
	}
	days := minioinfra.DefaultRawFileRetentionDays
	sysRepo := admin.NewPgSysConfigRepository(w.PgPool)
	if row, err := sysRepo.GetByKey(ctx, "minio.retention", "raw_object_days"); err == nil && row != nil {
		if parsed, parseErr := strconv.Atoi(row.Value); parseErr == nil && parsed > 0 {
			days = parsed
		}
	}
	if row, err := sysRepo.GetByKey(ctx, "minio.retention", "cleanup_mode"); err == nil && row != nil && row.Value != "" {
		cleanupCfg.Mode = row.Value
	}
	repo := rawcleanup.NewPGRepository(w.TsPool)
	deleter := rawcleanup.NewDeleter(
		rawcleanup.MinIOStore{Client: w.MinIO},
		cfg.MinIO.Buckets.PMFiles,
		cfg.MinIO.Buckets.MRFiles,
	)
	period := cfg.PM.PeriodicUploadInterval
	if period <= 0 {
		period = 900
	}
	runner := rawcleanup.NewRunner(repo, deleter, rawcleanup.Config{
		Mode:                cleanupCfg.Mode,
		BatchSize:           cleanupCfg.BatchSize,
		RetentionDays:       days,
		MinRate:             cleanupCfg.MinRate,
		MaxRate:             cleanupCfg.MaxRate,
		Headroom:            cleanupCfg.RateHeadroom,
		RecalculateInterval: cleanupCfg.RecalculateInterval,
		ObjectTimeout:       cleanupCfg.ObjectTimeout,
		ModeLookup: func(modeCtx context.Context) string {
			row, err := sysRepo.GetByKey(modeCtx, "minio.retention", "cleanup_mode")
			if err != nil || row == nil {
				return ""
			}
			return row.Value
		},
		RetentionDaysLookup: func(retentionCtx context.Context) int {
			row, err := sysRepo.GetByKey(retentionCtx, "minio.retention", "raw_object_days")
			if err != nil || row == nil {
				return 0
			}
			parsed, err := strconv.Atoi(row.Value)
			if err != nil || parsed <= 0 || parsed > 3650 {
				return 0
			}
			return parsed
		},
		PredictedRate: func(rateCtx context.Context) float64 {
			query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
				Select("COUNT(*)").From("devices").
				Where(sq.Eq{"is_online": true, "deleted_at": nil}).ToSql()
			if err != nil {
				return 0
			}
			var count int64
			if err := w.PgPool.QueryRow(rateCtx, query, args...).Scan(&count); err != nil {
				return 0
			}
			return float64(count) / float64(period)
		},
	})
	runner.SetLogger(w.Logger)
	runner.SetMetrics(rawcleanup.NewMetrics(w.MetricsReg))
	var queueProbe rawcleanup.QueueProbe = func(context.Context) (rawcleanup.QueueSample, error) {
		return nil, fmt.Errorf("NATS queue stats provider unavailable")
	}
	if provider, ok := w.EventBus.(queueStatsProvider); ok {
		queueProbe = func(queueCtx context.Context) (rawcleanup.QueueSample, error) {
			stats, err := provider.QueueStats(queueCtx, event.SubjectPMFileReceived, "pm-workers")
			if err != nil {
				return nil, err
			}
			sample := rawcleanup.QueueSample{
				"pm-workers": {
					Pending: stats.Pending, AckPending: uint64(max(stats.AckPending, 0)),
				},
			}
			protected := []struct{ stream, durable string }{
				{"PM_AGG_15M", "pm-aggregation-workers-pull"},
				{"PM_AGG_HOURLY", "pm-aggregation-hourly-rollup-pull"},
				{"PM_AGG_DAILY", "pm-aggregation-daily-rollup-pull"},
			}
			for _, target := range protected {
				consumer, err := w.NATS.JS.ConsumerInfo(
					target.stream, target.durable, gonats.Context(queueCtx),
				)
				if err != nil {
					return nil, fmt.Errorf("read %s/%s pressure: %w", target.stream, target.durable, err)
				}
				sample[target.durable] = rawcleanup.QueueDepth{
					Pending:    consumer.NumPending,
					AckPending: uint64(max(consumer.NumAckPending, 0)),
				}
			}
			return sample, nil
		}
	}
	runner.SetPressureProbe(rawcleanup.NewPrometheusPressureProbe(
		cleanupCfg.PrometheusURL, 2*time.Second, queueProbe,
	))
	go runner.Run(ctx)
	w.Logger.Info("precise raw object cleanup started",
		zap.String("mode", cleanupCfg.Mode),
		zap.Int("retention_days", days),
		zap.Int("batch_size", cleanupCfg.BatchSize),
		zap.Float64("max_rate", cleanupCfg.MaxRate))
}
