package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// NewPostgresPool creates a new PostgreSQL connection pool.
// If logger is provided and LogSQL is enabled, SQL queries will be logged.
// The tracer also emits OpenTelemetry spans when a TracerProvider is configured.
func NewPostgresPool(ctx context.Context, cfg appconfig.PostgresConfig, log *zap.Logger) (*pgxpool.Pool, error) {
	return NewPostgresPoolWithRegisterer(ctx, cfg, log, nil)
}

// NewPostgresPoolWithRegisterer creates a PostgreSQL pool and registers
// slow-query metrics with reg when slow-query monitoring is configured.
func NewPostgresPoolWithRegisterer(
	ctx context.Context,
	cfg appconfig.PostgresConfig,
	log *zap.Logger,
	reg prometheus.Registerer,
) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckInterval > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckInterval
	}

	poolCfg.ConnConfig.Tracer = buildQueryTracer(cfg, log, reg)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func buildQueryTracer(
	cfg appconfig.PostgresConfig,
	log *zap.Logger,
	reg prometheus.Registerer,
) pgx.QueryTracer {
	var tracer pgx.QueryTracer
	if cfg.LogSQL && log != nil {
		// Slow-query warnings are emitted by the dedicated tracer below, so the
		// SQL/OTEL tracer keeps its slow threshold disabled to avoid duplicates.
		tracer = NewOTELSQLTracer(log, cfg.LogSQLParams, 0)
	}
	if cfg.LogSQLSlowThreshold > 0 && log != nil {
		slowTracer := WithSlowQueryTracer(
			time.Duration(cfg.LogSQLSlowThreshold)*time.Millisecond,
			log,
			reg,
		)
		tracer = ChainTracer(tracer, slowTracer)
	}
	return tracer
}

// PostgresHealthCheck verifies the PostgreSQL connection is alive.
func PostgresHealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return pool.Ping(ctx)
}
