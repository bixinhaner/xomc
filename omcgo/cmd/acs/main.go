package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omcgo/omcgo/internal/acs"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/event"
	"github.com/omcgo/omcgo/internal/appconfig"
	"github.com/omcgo/omcgo/internal/components"
	logpkg "github.com/omcgo/omcgo/internal/components/logger"
	natscomp "github.com/omcgo/omcgo/internal/components/nats"
	rediscomp "github.com/omcgo/omcgo/internal/components/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-acs",
		Short: "OMC TR069 ACS Engine",
		Long:  "TR069/CWMP Auto Configuration Server for small cell management",
		RunE:  runACS,
	}

	rootCmd.Flags().String("config", "cmd/acs/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runACS(cmd *cobra.Command, args []string) error {
	// 1. Load config
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.ACSConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	logger, err := logpkg.NewLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("omcgo-acs starting", zap.String("config", cfgPath))

	// 3. Graceful shutdown setup
	gs := components.NewGracefulShutdown(30*time.Second, logger)

	// 4. Connect to Redis
	redisClient, err := rediscomp.NewRedisClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	gs.Register("redis", 3, func(ctx context.Context) error { return redisClient.Close() })

	// 5. Connect to NATS
	natsClient, err := natscomp.NewNATSClient(cfg.NATS, logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	gs.Register("nats", 2, func(ctx context.Context) error { natsClient.Close(); return nil })

	if err := natsClient.EnsureStreams(context.Background()); err != nil {
		logger.Warn("ensure NATS streams", zap.Error(err))
	}

	// 6. Create EventBus
	eventBus := event.NewNATSEventBus(natsClient.Conn, natsClient.JS, logger)
	gs.Register("eventbus", 2, func(ctx context.Context) error { return eventBus.Close() })

	// 7. Create ACS components
	sessionStore := acs.NewRedisSessionStore(redisClient, cfg.Session.Timeout)
	cmdQueue := cmdqueue.NewRedisCommandQueue(redisClient)

	// 8. Prometheus metrics
	metricsReg := prometheus.NewRegistry()
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
		Handler: metricsMux,
	}

	// 9. Create ACS server
	deps := acs.NewDefaultDeps(
		sessionStore,
		cmdQueue,
		eventBus,
		cfg.Auth.Mode, cfg.Auth.Username, cfg.Auth.Password,
		cfg.RateLimit.PerDevice,
		cfg.Session.MaxConcurrent,
		metricsReg,
		logger,
	)

	acsServer := acs.NewACSServer(cfg, deps)
	gs.Register("acs-http", 1, func(ctx context.Context) error { return acsServer.Shutdown(ctx) })
	gs.Register("metrics-http", 1, func(ctx context.Context) error { return metricsServer.Shutdown(ctx) })

	// 10. Start metrics server
	go func() {
		logger.Info("metrics server starting", zap.Int("port", cfg.Metrics.Port))
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	// 11. Start ACS server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- acsServer.Start()
	}()

	// 12. Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			logger.Error("ACS server error", zap.Error(err))
		}
	}

	// 13. Graceful shutdown
	if err := gs.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("omcgo-acs stopped")
	return nil
}
