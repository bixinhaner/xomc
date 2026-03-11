package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/cmd/app/router"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-app",
		Short: "OMC Main Application",
		Long:  "OMC main application server providing REST API, device management, and all F02-F10 modules",
		RunE:  runApp,
	}

	rootCmd.Flags().String("config", "cmd/app/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runApp(cmd *cobra.Command, args []string) error {
	// 1. Load config
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.AppConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	logger, err := logpkg.NewLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("omcgo-app starting", zap.String("config", cfgPath))

	// 3. Graceful shutdown setup
	gs := components.NewGracefulShutdown(30*time.Second, logger)
	ctx := context.Background()

	// 4. Connect to PostgreSQL
	pgPool, err := postgres.NewPostgresPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	gs.Register("postgres", 4, func(ctx context.Context) error { pgPool.Close(); return nil })

	// 5. Connect to Redis
	redisClient, err := rediscomp.NewRedisClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	gs.Register("redis", 3, func(ctx context.Context) error { return redisClient.Close() })

	// 6. Connect to TimescaleDB
	tsPool, err := postgres.NewTimescalePool(ctx, cfg.TSDB)
	if err != nil {
		return fmt.Errorf("connect to TimescaleDB: %w", err)
	}
	gs.Register("timescale", 4, func(ctx context.Context) error { tsPool.Close(); return nil })

	// 7. Connect to MinIO
	minioClient, err := miniocomp.NewMinIOClient(cfg.MinIO)
	if err != nil {
		return fmt.Errorf("connect to MinIO: %w", err)
	}

	// 8. Connect to NATS
	natsClient, err := natscomp.NewNATSClient(cfg.NATS, logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	gs.Register("nats", 2, func(ctx context.Context) error { natsClient.Close(); return nil })

	if err := natsClient.EnsureStreams(ctx); err != nil {
		logger.Warn("ensure NATS streams", zap.Error(err))
	}

	// 9. Create EventBus
	eventBus := event.NewNATSEventBus(natsClient.Conn, natsClient.JS, logger)
	gs.Register("eventbus", 2, func(ctx context.Context) error { return eventBus.Close() })

	// 10. Create carrier registry
	carrierRegistry := carrier.NewRegistry()
	carrierRegistry.Register(cmcc.New())
	carrierRegistry.Register(ctcc.New())
	carrierRegistry.Register(cucc.New())
	logger.Info("carrier registry initialized", zap.Int("carriers", len(carrierRegistry.All())))

	// 11. Create command queue (shared with ACS)
	cmdQueue := cmdqueue.NewRedisCommandQueue(redisClient)

	// 12. Setup Gin engine
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	ginEngine := gin.New()
	ginEngine.Use(gin.Recovery())

	metricsReg := prometheus.NewRegistry()

	// 13. Setup all modules and routes
	router.Setup(ginEngine, &router.Deps{
		PgPool:          pgPool,
		TsPool:          tsPool,
		Redis:           redisClient,
		MinIO:           minioClient,
		EventBus:        eventBus,
		CmdQueue:        cmdQueue,
		CarrierRegistry: carrierRegistry,
		Cfg:             &cfg,
		Logger:          logger,
		GS:              gs,
		MetricsReg:      metricsReg,
	})

	// 14. Prometheus metrics server
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

	// 15. Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      ginEngine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	gs.Register("app-http", 1, func(ctx context.Context) error { return httpServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		logger.Info("app server starting", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 16. Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			logger.Error("app server error", zap.Error(err))
		}
	}

	// 17. Graceful shutdown
	if err := gs.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("omcgo-app stopped")
	return nil
}
