package aggregator

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	maxHourlyTxRetryAttempts = 3
	hourlyTxRetryMinDelay    = 25 * time.Millisecond
	hourlyTxRetryMaxDelay    = 100 * time.Millisecond
)

var (
	hourlyTxRetryJitter = func(min, max time.Duration) time.Duration {
		return min + time.Duration(rand.Int64N(int64(max-min)+1))
	}
	hourlyTxRetrySleep = sleepWithContext
)

// runTransactionWithRetry retries only PostgreSQL deadlocks and serialization
// failures. Every retry begins a fresh transaction because PostgreSQL aborts a
// transaction after either error.
func runTransactionWithRetry(
	ctx context.Context,
	begin func(context.Context) (pgx.Tx, error),
	maxAttempts int,
	run func(context.Context, pgx.Tx) error,
) error {
	return runTransactionWithRetryObserved(ctx, begin, maxAttempts, run, nil, nil)
}

func runTransactionWithRetryObserved(
	ctx context.Context,
	begin func(context.Context) (pgx.Tx, error),
	maxAttempts int,
	run func(context.Context, pgx.Tx) error,
	onRetry func(sqlState string),
	onExhausted func(sqlState string),
) error {
	if maxAttempts <= 0 {
		return fmt.Errorf("hourly transaction retry attempts must be positive")
	}
	if maxAttempts > maxHourlyTxRetryAttempts {
		maxAttempts = maxHourlyTxRetryAttempts
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		tx, err := begin(ctx)
		if err == nil {
			err = run(ctx, tx)
			if err == nil {
				err = tx.Commit(ctx)
				if err == nil {
					return nil
				}
			}
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}

		sqlState, retriable := retriableHourlyTxSQLState(err)
		if !retriable || attempt == maxAttempts-1 {
			if retriable && onExhausted != nil {
				onExhausted(sqlState)
			}
			return fmt.Errorf("hourly transaction attempt %d: %w", attempt+1, err)
		}
		if onRetry != nil {
			onRetry(sqlState)
		}
		if err := hourlyTxRetrySleep(ctx, hourlyTxRetryDelay(attempt+1)); err != nil {
			return fmt.Errorf("wait to retry hourly transaction: %w", err)
		}
	}

	return fmt.Errorf("hourly transaction retry exhausted")
}

func retriableHourlyTxSQLState(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || (pgErr.Code != "40P01" && pgErr.Code != "40001") {
		return "", false
	}
	return pgErr.Code, true
}

func hourlyTxRetryDelay(retry int) time.Duration {
	max := hourlyTxRetryMinDelay << retry
	if max > hourlyTxRetryMaxDelay {
		max = hourlyTxRetryMaxDelay
	}
	return hourlyTxRetryJitter(hourlyTxRetryMinDelay, max)
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
