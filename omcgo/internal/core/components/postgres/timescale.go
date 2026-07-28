package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// NewTimescalePool creates a new connection pool for TimescaleDB.
// It shares the same config structure as PostgreSQL but connects to the
// TimescaleDB-enabled database.
// If logger is provided and LogSQL is enabled, SQL queries will be logged.
func NewTimescalePool(ctx context.Context, cfg appconfig.PostgresConfig, log *zap.Logger) (*pgxpool.Pool, error) {
	return NewTimescalePoolWithRegisterer(ctx, cfg, log, nil)
}

// NewTimescalePoolWithRegisterer creates a TimescaleDB pool and exposes the
// same slow-query metrics as the primary PostgreSQL pool.
func NewTimescalePoolWithRegisterer(
	ctx context.Context,
	cfg appconfig.PostgresConfig,
	log *zap.Logger,
	reg prometheus.Registerer,
) (*pgxpool.Pool, error) {
	pool, err := NewPostgresPoolWithRegisterer(ctx, cfg, log, reg)
	if err != nil {
		return nil, fmt.Errorf("create timescale pool: %w", err)
	}
	return pool, nil
}

// EnsureTimescaleExtension verifies that the TimescaleDB extension is installed.
func EnsureTimescaleExtension(ctx context.Context, pool *pgxpool.Pool) error {
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").
		Scan(&exists)
	if err != nil {
		return fmt.Errorf("check timescaledb extension: %w", err)
	}
	if !exists {
		return fmt.Errorf("timescaledb extension is not installed")
	}
	return nil
}
