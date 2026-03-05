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
	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/omcgo/omcgo/internal/common/middleware"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/infra"
	"github.com/omcgo/omcgo/internal/infra/cache"
	"github.com/omcgo/omcgo/internal/infra/db"
	"github.com/omcgo/omcgo/internal/infra/mq"
	"github.com/omcgo/omcgo/internal/omcr/device"
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

	rootCmd.Flags().String("config", "configs/app.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runApp(cmd *cobra.Command, args []string) error {
	// 1. Load config
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg config.AppConfig
	if err := config.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	logger, err := infra.NewLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("omcgo-app starting", zap.String("config", cfgPath))

	// 3. Graceful shutdown setup
	gs := infra.NewGracefulShutdown(30*time.Second, logger)
	ctx := context.Background()

	// 4. Connect to PostgreSQL
	pgPool, err := db.NewPostgresPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	gs.Register("postgres", 4, func(ctx context.Context) error { pgPool.Close(); return nil })

	// 5. Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	gs.Register("redis", 3, func(ctx context.Context) error { return redisClient.Close() })

	// 6. Connect to NATS
	natsClient, err := mq.NewNATSClient(cfg.NATS, logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	gs.Register("nats", 2, func(ctx context.Context) error { natsClient.Close(); return nil })

	if err := natsClient.EnsureStreams(ctx); err != nil {
		logger.Warn("ensure NATS streams", zap.Error(err))
	}

	// 7. Create EventBus
	eventBus := event.NewNATSEventBus(natsClient.Conn, natsClient.JS, logger)
	gs.Register("eventbus", 2, func(ctx context.Context) error { return eventBus.Close() })

	// 8. Create repositories
	deviceRepo := device.NewPgDeviceRepository(pgPool)
	paramRepo := device.NewPgDeviceParameterRepository(pgPool)

	// 9. Create HeartbeatMonitor
	heartbeatMonitor := device.NewHeartbeatMonitor(redisClient, deviceRepo, logger)
	heartbeatMonitor.Start()
	gs.Register("heartbeat", 1, func(ctx context.Context) error { heartbeatMonitor.Stop(); return nil })

	// 10. Create services
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, heartbeatMonitor, logger)

	// 11. Subscribe InformHandler to events
	informHandler := device.NewInformHandler(deviceService, model.CarrierCMCC, logger)
	if err := informHandler.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe inform handler", zap.Error(err))
	}

	// 12. Setup Gin router
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(logger))

	metricsReg := prometheus.NewRegistry()
	router.Use(middleware.PrometheusMetrics(metricsReg))

	// Health check
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	deviceHandler := device.NewHandler(deviceService)
	deviceHandler.RegisterRoutes(v1)

	// 13. Prometheus metrics server
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

	// 14. Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
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

	// 15. Wait for signal
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

	// 16. Graceful shutdown
	if err := gs.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("omcgo-app stopped")
	return nil
}
