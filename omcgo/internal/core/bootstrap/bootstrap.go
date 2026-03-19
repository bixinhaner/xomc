package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/components"
	logpkg "github.com/omcgo/omcgo/internal/core/components/logger"
	miniocomp "github.com/omcgo/omcgo/internal/core/components/minio"
	natscomp "github.com/omcgo/omcgo/internal/core/components/nats"
	"github.com/omcgo/omcgo/internal/core/components/postgres"
	rediscomp "github.com/omcgo/omcgo/internal/core/components/redis"
	"github.com/omcgo/omcgo/internal/core/event"
)

// App holds all infrastructure connections initialized during startup.
type App struct {
	Logger     *zap.Logger
	GS         *components.GracefulShutdown
	PgPool     *pgxpool.Pool
	TsPool     *pgxpool.Pool
	Redis      redis.UniversalClient
	MinIO      *minio.Client
	NATS       *natscomp.NATSClient
	EventBus   event.EventBus
	Carriers   *carrier.CarrierRegistry
	CmdQueue   *cmdqueue.RedisCommandQueue
	MetricsReg *prometheus.Registry

	metricsPort int
}

func newBase(logCfg appconfig.LogConfig, metricsPort int) (*App, error) {
	logger, err := logpkg.NewLogger(logCfg)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	return &App{
		Logger:      logger,
		GS:          components.NewGracefulShutdown(30*time.Second, logger),
		MetricsReg:  prometheus.NewRegistry(),
		metricsPort: metricsPort,
	}, nil
}

// InitForApp initializes all infrastructure for the main application.
func InitForApp(ctx context.Context, cfg *appconfig.AppConfig) (*App, error) {
	app, err := newBase(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := app.connectPostgres(ctx, cfg.DB); err != nil {
		return nil, err
	}
	if err := app.connectTimescale(ctx, cfg.TSDB); err != nil {
		return nil, err
	}
	if err := app.connectRedis(cfg.Redis); err != nil {
		return nil, err
	}
	if err := app.connectNATS(ctx, cfg.NATS); err != nil {
		return nil, err
	}
	if err := app.connectMinIO(ctx, cfg.MinIO); err != nil {
		return nil, err
	}
	app.createEventBus()
	app.registerCarriers()
	app.createCmdQueue()

	return app, nil
}

// InitForACS initializes infrastructure for the ACS engine (Redis + NATS + MinIO).
func InitForACS(ctx context.Context, cfg *appconfig.ACSConfig) (*App, error) {
	app, err := newBase(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := app.connectRedis(cfg.Redis); err != nil {
		return nil, err
	}
	if err := app.connectNATS(ctx, cfg.NATS); err != nil {
		return nil, err
	}
	// MinIO connection for file upload proxy
	if cfg.MinIO.Endpoint != "" {
		if err := app.connectMinIO(ctx, cfg.MinIO); err != nil {
			return nil, err
		}
	}
	app.createEventBus()

	return app, nil
}

// InitForWorker initializes all infrastructure for the background worker.
func InitForWorker(ctx context.Context, cfg *appconfig.WorkerConfig) (*App, error) {
	app, err := newBase(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := app.connectPostgres(ctx, cfg.DB); err != nil {
		return nil, err
	}
	if err := app.connectTimescale(ctx, cfg.TSDB); err != nil {
		return nil, err
	}
	if err := app.connectRedis(cfg.Redis); err != nil {
		return nil, err
	}
	if err := app.connectNATS(ctx, cfg.NATS); err != nil {
		return nil, err
	}
	if err := app.connectMinIO(ctx, cfg.MinIO); err != nil {
		return nil, err
	}
	app.createEventBus()
	app.registerCarriers()
	app.createCmdQueue()

	return app, nil
}

// --- infrastructure connect methods ---

func (a *App) connectPostgres(ctx context.Context, cfg appconfig.PostgresConfig) error {
	pool, err := postgres.NewPostgresPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	a.PgPool = pool
	a.GS.Register("postgres", 4, func(ctx context.Context) error { pool.Close(); return nil })
	return nil
}

func (a *App) connectTimescale(ctx context.Context, cfg appconfig.PostgresConfig) error {
	pool, err := postgres.NewTimescalePool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to TimescaleDB: %w", err)
	}
	a.TsPool = pool
	a.GS.Register("timescale", 4, func(ctx context.Context) error { pool.Close(); return nil })
	return nil
}

func (a *App) connectRedis(cfg appconfig.RedisConfig) error {
	client, err := rediscomp.NewRedisClient(cfg)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	a.Redis = client
	a.GS.Register("redis", 3, func(ctx context.Context) error { return client.Close() })
	return nil
}

func (a *App) connectNATS(ctx context.Context, cfg appconfig.NATSConfig) error {
	client, err := natscomp.NewNATSClient(cfg, a.Logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	a.NATS = client
	a.GS.Register("nats", 2, func(ctx context.Context) error { client.Close(); return nil })

	if err := client.EnsureStreams(ctx); err != nil {
		a.Logger.Warn("ensure NATS streams", zap.Error(err))
	}
	return nil
}

func (a *App) connectMinIO(ctx context.Context, cfg appconfig.MinIOConfig) error {
	client, err := miniocomp.NewMinIOClient(cfg)
	if err != nil {
		return fmt.Errorf("connect to MinIO: %w", err)
	}
	a.MinIO = client

	if err := miniocomp.EnsureBuckets(ctx, client, cfg.Buckets); err != nil {
		a.Logger.Warn("ensure MinIO buckets", zap.Error(err))
	}
	return nil
}

func (a *App) createEventBus() {
	a.EventBus = event.NewNATSEventBus(a.NATS.Conn, a.NATS.JS, a.Logger)
	a.GS.Register("eventbus", 2, func(ctx context.Context) error { return a.EventBus.Close() })
}

func (a *App) registerCarriers() {
	reg := carrier.NewRegistry()
	reg.Register(cmcc.New())
	reg.Register(ctcc.New())
	reg.Register(cucc.New())
	a.Carriers = reg
	a.Logger.Info("carrier registry initialized", zap.Int("carriers", len(reg.All())))
}

func (a *App) createCmdQueue() {
	a.CmdQueue = cmdqueue.NewRedisCommandQueue(a.Redis)
}

// --- server lifecycle ---

// ListenAndServe starts the HTTP server and metrics server,
// then blocks until a signal is received and performs graceful shutdown.
func (a *App) ListenAndServe(handler http.Handler, addr string) error {
	a.startMetrics()

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	a.GS.Register("app-http", 1, func(ctx context.Context) error { return httpServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		a.Logger.Info("http server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	return a.waitForShutdown(errCh)
}

// WaitAndShutdown starts the metrics server, then blocks until a signal is received
// or an error arrives on errCh. Pass nil if there is no error channel to monitor.
func (a *App) WaitAndShutdown(errCh <-chan error) error {
	a.startMetrics()
	return a.waitForShutdown(errCh)
}

func (a *App) startMetrics() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(a.MetricsReg, promhttp.HandlerOpts{}))
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.metricsPort),
		Handler: metricsMux,
	}
	a.GS.Register("metrics-http", 1, func(ctx context.Context) error { return metricsServer.Shutdown(ctx) })

	go func() {
		a.Logger.Info("metrics server starting", zap.Int("port", a.metricsPort))
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			a.Logger.Error("metrics server error", zap.Error(err))
		}
	}()
}

func (a *App) waitForShutdown(errCh <-chan error) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if errCh != nil {
		select {
		case sig := <-sigCh:
			a.Logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
		case err := <-errCh:
			if err != nil {
				a.Logger.Error("server error", zap.Error(err))
			}
		}
	} else {
		sig := <-sigCh
		a.Logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	}

	if err := a.GS.Shutdown(context.Background()); err != nil {
		a.Logger.Error("shutdown error", zap.Error(err))
	}

	a.Logger.Info("service stopped")
	return nil
}
