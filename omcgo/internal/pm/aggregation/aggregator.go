package aggregation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Aggregator manages TimescaleDB continuous aggregate operations.
type Aggregator struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewAggregator creates a new aggregator.
func NewAggregator(pool *pgxpool.Pool, logger *zap.Logger) *Aggregator {
	return &Aggregator{pool: pool, logger: logger}
}

// RefreshContinuousAggregates manually refreshes the hourly PM counter aggregate.
func (a *Aggregator) RefreshContinuousAggregates(ctx context.Context) error {
	_, err := a.pool.Exec(ctx,
		"CALL refresh_continuous_aggregate('pm_counters_hourly', now() - interval '2 hours', now())")
	if err != nil {
		return fmt.Errorf("refresh pm_counters_hourly: %w", err)
	}
	a.logger.Info("refreshed pm_counters_hourly continuous aggregate")
	return nil
}
