package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/carrier/cucc"
	"github.com/omcgo/omcgo/internal/event"
	"github.com/omcgo/omcgo/internal/appconfig"
	"github.com/omcgo/omcgo/internal/components"
	logpkg "github.com/omcgo/omcgo/internal/components/logger"
	miniocomp "github.com/omcgo/omcgo/internal/components/minio"
	natscomp "github.com/omcgo/omcgo/internal/components/nats"
	"github.com/omcgo/omcgo/internal/components/postgres"
	rediscomp "github.com/omcgo/omcgo/internal/components/redis"
	"github.com/omcgo/omcgo/internal/mr"
	mrcollector "github.com/omcgo/omcgo/internal/mr/collector"
	"github.com/omcgo/omcgo/internal/pm/collector"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-worker",
		Short: "OMC Background Worker",
		Long:  "Background worker process for PM/MR file processing and KPI calculation",
		RunE:  runWorker,
	}

	rootCmd.Flags().String("config", "cmd/worker/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWorker(cmd *cobra.Command, args []string) error {
	// 1. Load config
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.WorkerConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	logger, err := logpkg.NewLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("omcgo-worker starting", zap.String("config", cfgPath))

	// 3. Graceful shutdown setup
	gs := components.NewGracefulShutdown(30*time.Second, logger)
	ctx := context.Background()

	// 4. Connect to PostgreSQL
	pgPool, err := postgres.NewPostgresPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	gs.Register("postgres", 4, func(ctx context.Context) error { pgPool.Close(); return nil })

	// 5. Connect to TimescaleDB
	tsPool, err := postgres.NewTimescalePool(ctx, cfg.TSDB)
	if err != nil {
		return fmt.Errorf("connect to TimescaleDB: %w", err)
	}
	gs.Register("timescale", 4, func(ctx context.Context) error { tsPool.Close(); return nil })

	// 6. Connect to Redis
	redisClient, err := rediscomp.NewRedisClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	gs.Register("redis", 3, func(ctx context.Context) error { return redisClient.Close() })

	// 7. Connect to NATS
	natsClient, err := natscomp.NewNATSClient(cfg.NATS, logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	gs.Register("nats", 2, func(ctx context.Context) error { natsClient.Close(); return nil })

	if err := natsClient.EnsureStreams(ctx); err != nil {
		logger.Warn("ensure NATS streams", zap.Error(err))
	}

	// 8. Connect to MinIO
	minioClient, err := miniocomp.NewMinIOClient(cfg.MinIO)
	if err != nil {
		return fmt.Errorf("connect to MinIO: %w", err)
	}
	if err := miniocomp.EnsureBuckets(ctx, minioClient, cfg.MinIO.Buckets); err != nil {
		logger.Warn("ensure MinIO buckets", zap.Error(err))
	}

	// 9. Create EventBus
	eventBus := event.NewNATSEventBus(natsClient.Conn, natsClient.JS, logger)
	gs.Register("eventbus", 2, func(ctx context.Context) error { return eventBus.Close() })

	// 10. Create CarrierRegistry
	carrierRegistry := carrier.NewRegistry()
	carrierRegistry.Register(cmcc.New())
	carrierRegistry.Register(ctcc.New())
	carrierRegistry.Register(cucc.New())
	logger.Info("carrier registry initialized", zap.Int("carriers", len(carrierRegistry.All())))

	// 11. Create PM Collector + Subscribe
	counterRepo := counter.NewPgCounterRepository(tsPool)
	kpiRepo := kpi.NewPgKPIRepository(tsPool)
	kpiEngine := kpi.NewKPIEngine(counterRepo, kpiRepo, carrierRegistry, logger)
	pmParser := collector.NewPMXMLParser()
	pmCollector := collector.NewPMCollector(minioClient, cfg.MinIO.Buckets.PMFiles, pmParser, counterRepo, kpiEngine, eventBus, logger)
	if err := pmCollector.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe PM collector", zap.Error(err))
	}
	logger.Info("PM collector started")

	// 12. Create Alarm Receiver + Subscribe
	alarmPgStore := alarm.NewPgAlarmStore(pgPool, tsPool)
	alarmRedisStore := alarm.NewRedisAlarmStore(redisClient)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, carrierRegistry, eventBus, logger)
	alarmReceiver := alarm.NewAlarmReceiver(alarmEngine, logger)
	if err := alarmReceiver.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe alarm receiver", zap.Error(err))
	}
	logger.Info("alarm receiver started")

	// 13. Create MR Collector + Subscribe
	mrStore := mr.NewPgMRStore(pgPool, tsPool)
	mrCollector := mrcollector.NewMRCollector(minioClient, cfg.MinIO.Buckets.MRFiles, mrStore, eventBus, logger)
	if err := mrCollector.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe MR collector", zap.Error(err))
	}
	logger.Info("MR collector started")

	// 14. Start Prometheus metrics server
	metricsReg := prometheus.NewRegistry()
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
		Handler: metricsMux,
	}
	gs.Register("metrics-http", 1, func(ctx context.Context) error { return metricsServer.Shutdown(ctx) })

	go func() {
		logger.Info("metrics server starting", zap.Int("port", cfg.Metrics.Port))
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	logger.Info("omcgo-worker ready, waiting for events...")

	// 15. Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received signal, shutting down", zap.String("signal", sig.String()))

	// 16. Graceful shutdown
	if err := gs.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("omcgo-worker stopped")
	return nil
}
