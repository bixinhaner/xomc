package components

import (
	"context"
	"encoding/json"
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

	"github.com/omcgo/omcgo/internal/core/appconfig"
	logpkg "github.com/omcgo/omcgo/internal/core/components/logger"
	miniocomp "github.com/omcgo/omcgo/internal/core/components/minio"
	natscomp "github.com/omcgo/omcgo/internal/core/components/nats"
	"github.com/omcgo/omcgo/internal/core/components/postgres"
	rediscomp "github.com/omcgo/omcgo/internal/core/components/redis"
	"github.com/omcgo/omcgo/internal/core/event"
)

// Infra holds all infrastructure connections initialized during startup.
// Each cmd entry point selects which components to initialize.
type Infra struct {
	Logger     *zap.Logger
	GS         *GracefulShutdown
	PgPool     *pgxpool.Pool
	TsPool     *pgxpool.Pool
	Redis      redis.UniversalClient
	MinIO      *minio.Client
	NATS       *natscomp.NATSClient
	EventBus   event.EventBus
	MetricsReg *prometheus.Registry
	Health     *HealthChecker

	metricsPort int
}

// NewInfra creates a base Infra with logger, graceful shutdown, metrics registry, and health checker.
func NewInfra(logCfg appconfig.LogConfig, metricsPort int) (*Infra, error) {
	logger, err := logpkg.NewLogger(logCfg)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	// Set as global logger so logger.L(ctx) can access it via zap.L()
	zap.ReplaceGlobals(logger)
	return &Infra{
		Logger:      logger,
		GS:          NewGracefulShutdown(30*time.Second, logger),
		MetricsReg:  prometheus.NewRegistry(),
		Health:      NewHealthChecker(),
		metricsPort: metricsPort,
	}, nil
}

// InitTracer initializes the OpenTelemetry TracerProvider.
// When tracing is disabled, a no-op provider is set (zero overhead).
// The TracerProvider is registered for graceful shutdown.
func (inf *Infra) InitTracer(ctx context.Context, cfg appconfig.TracerConfig, serviceName string) error {
	tp, err := NewTracerProvider(ctx, cfg, serviceName)
	if err != nil {
		return fmt.Errorf("init tracer provider: %w", err)
	}
	inf.GS.Register("tracer", 5, func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})
	if cfg.Enabled {
		inf.Logger.Info("tracer initialized",
			zap.String("endpoint", cfg.Endpoint),
			zap.Float64("sample_rate", cfg.SampleRate))
	} else {
		inf.Logger.Info("tracer disabled (no-op provider)")
	}
	return nil
}

// ConnectPostgres initializes a PostgreSQL connection pool.
func (inf *Infra) ConnectPostgres(ctx context.Context, cfg appconfig.PostgresConfig) error {
	pool, err := postgres.NewPostgresPool(ctx, cfg, inf.Logger)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	inf.PgPool = pool
	inf.GS.Register("postgres", 4, func(ctx context.Context) error { pool.Close(); return nil })
	inf.Health.Register("postgres", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	return nil
}

// ConnectTimescale initializes a TimescaleDB connection pool.
func (inf *Infra) ConnectTimescale(ctx context.Context, cfg appconfig.PostgresConfig) error {
	pool, err := postgres.NewTimescalePool(ctx, cfg, inf.Logger)
	if err != nil {
		return fmt.Errorf("connect to TimescaleDB: %w", err)
	}
	inf.TsPool = pool
	inf.GS.Register("timescale", 4, func(ctx context.Context) error { pool.Close(); return nil })
	inf.Health.Register("timescale", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	return nil
}

// ConnectRedis initializes a Redis client.
func (inf *Infra) ConnectRedis(cfg appconfig.RedisConfig) error {
	client, err := rediscomp.NewRedisClient(cfg)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	inf.Redis = client
	inf.GS.Register("redis", 3, func(ctx context.Context) error { return client.Close() })
	inf.Health.Register("redis", func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	})
	return nil
}

// ConnectNATS initializes a NATS client and ensures streams exist.
func (inf *Infra) ConnectNATS(ctx context.Context, cfg appconfig.NATSConfig) error {
	client, err := natscomp.NewNATSClient(cfg, inf.Logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	inf.NATS = client
	inf.GS.Register("nats", 2, func(ctx context.Context) error { client.Close(); return nil })

	if err := client.EnsureStreams(ctx); err != nil {
		inf.Logger.Warn("ensure NATS streams", zap.Error(err))
	}
	return nil
}

// ConnectMinIO initializes a MinIO client and ensures buckets exist.
func (inf *Infra) ConnectMinIO(ctx context.Context, cfg appconfig.MinIOConfig) error {
	client, err := miniocomp.NewMinIOClient(cfg)
	if err != nil {
		return fmt.Errorf("connect to MinIO: %w", err)
	}
	inf.MinIO = client

	if err := miniocomp.EnsureBuckets(ctx, client, cfg.Buckets); err != nil {
		inf.Logger.Warn("ensure MinIO buckets", zap.Error(err))
	}
	return nil
}

// CreateEventBus creates a NATS-backed EventBus and registers it for graceful shutdown.
func (inf *Infra) CreateEventBus() {
	inf.EventBus = event.NewNATSEventBus(inf.NATS.Conn, inf.NATS.JS, inf.Logger)
	inf.GS.Register("eventbus", 2, func(ctx context.Context) error { return inf.EventBus.Close() })
}

// --- server lifecycle ---

// ListenAndServe starts the HTTP server and metrics server,
// then blocks until a signal is received and performs graceful shutdown.
func (inf *Infra) ListenAndServe(handler http.Handler, addr string) error {
	inf.startMetrics()

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	inf.GS.Register("app-http", 1, func(ctx context.Context) error { return httpServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		inf.Logger.Info("http server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	return inf.waitForShutdown(errCh)
}

// WaitAndShutdown starts the metrics server, then blocks until a signal is received
// or an error arrives on errCh. Pass nil if there is no error channel to monitor.
func (inf *Infra) WaitAndShutdown(errCh <-chan error) error {
	inf.startMetrics()
	return inf.waitForShutdown(errCh)
}

func (inf *Infra) startMetrics() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(inf.MetricsReg, promhttp.HandlerOpts{}))
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		results := inf.Health.CheckAll(r.Context())
		w.Header().Set("Content-Type", "application/json")
		code := 200
		for _, result := range results {
			if result.Status != "healthy" {
				code = 503
				break
			}
		}
		w.WriteHeader(code)
		data, _ := json.Marshal(map[string]interface{}{
			"status":     map[bool]string{true: "ok", false: "degraded"}[code == 200],
			"components": results,
		})
		w.Write(data)
	})
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", inf.metricsPort),
		Handler: metricsMux,
	}
	inf.GS.Register("metrics-http", 1, func(ctx context.Context) error { return metricsServer.Shutdown(ctx) })

	go func() {
		inf.Logger.Info("metrics server starting", zap.Int("port", inf.metricsPort))
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			inf.Logger.Error("metrics server error", zap.Error(err))
		}
	}()
}

func (inf *Infra) waitForShutdown(errCh <-chan error) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if errCh != nil {
		select {
		case sig := <-sigCh:
			inf.Logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
		case err := <-errCh:
			if err != nil {
				inf.Logger.Error("server error", zap.Error(err))
			}
		}
	} else {
		sig := <-sigCh
		inf.Logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	}

	if err := inf.GS.Shutdown(context.Background()); err != nil {
		inf.Logger.Error("shutdown error", zap.Error(err))
	}

	inf.Logger.Info("service stopped")
	return nil
}
