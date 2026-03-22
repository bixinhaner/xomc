package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
)

// NewPostgresPool creates a new PostgreSQL connection pool.
// If logger is provided and LogSQL is enabled, SQL queries will be logged.
// The tracer also emits OpenTelemetry spans when a TracerProvider is configured.
func NewPostgresPool(ctx context.Context, cfg appconfig.PostgresConfig, log *zap.Logger) (*pgxpool.Pool, error) {
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

	// Attach SQL tracer if logging is enabled.
	// OTELSQLTracer wraps the base SQLTracer and adds OpenTelemetry spans.
	// When tracing is disabled (no-op TracerProvider), OTEL overhead is negligible.
	if cfg.LogSQL && log != nil {
		poolCfg.ConnConfig.Tracer = NewOTELSQLTracer(log, cfg.LogSQLParams, cfg.LogSQLSlowThreshold)
	}

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

// PostgresHealthCheck verifies the PostgreSQL connection is alive.
func PostgresHealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return pool.Ping(ctx)
}
