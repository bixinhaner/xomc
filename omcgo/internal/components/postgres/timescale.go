package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/appconfig"
)

// NewTimescalePool creates a new connection pool for TimescaleDB.
// It shares the same config structure as PostgreSQL but connects to the
// TimescaleDB-enabled database.
func NewTimescalePool(ctx context.Context, cfg appconfig.PostgresConfig) (*pgxpool.Pool, error) {
	pool, err := NewPostgresPool(ctx, cfg)
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
