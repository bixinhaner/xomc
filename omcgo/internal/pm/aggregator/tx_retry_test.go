package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTransactionWithRetryRetriesRetriableSQLStateWithFreshTransaction(t *testing.T) {
	withRetryTiming(t, func(min, max time.Duration) time.Duration { return min }, sleepImmediately)

	first := &retryTx{}
	second := &retryTx{}
	transactions := []pgx.Tx{first, second}
	var begins, runs int

	err := runTransactionWithRetry(context.Background(), func(context.Context) (pgx.Tx, error) {
		tx := transactions[begins]
		begins++
		return tx, nil
	}, 3, func(_ context.Context, _ pgx.Tx) error {
		runs++
		if runs == 1 {
			return &pgconn.PgError{Code: "40P01"}
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, begins)
	assert.Equal(t, 2, runs)
	assert.Equal(t, 1, first.rollbacks)
	assert.Equal(t, 0, first.commits)
	assert.Equal(t, 0, second.rollbacks)
	assert.Equal(t, 1, second.commits)
}

func TestRunTransactionWithRetryReturnsNonRetriableErrorImmediately(t *testing.T) {
	first := &retryTx{}
	nonRetriable := &pgconn.PgError{Code: "23505"}
	var begins int

	err := runTransactionWithRetry(context.Background(), func(context.Context) (pgx.Tx, error) {
		begins++
		return first, nil
	}, 3, func(context.Context, pgx.Tx) error {
		return nonRetriable
	})

	require.ErrorIs(t, err, nonRetriable)
	assert.Equal(t, 1, begins)
	assert.Equal(t, 1, first.rollbacks)
	assert.Equal(t, 0, first.commits)
}

func TestRunTransactionWithRetryStopsAfterThreeRetriableFailures(t *testing.T) {
	withRetryTiming(t, func(min, max time.Duration) time.Duration { return min }, sleepImmediately)

	transactions := []*retryTx{{}, {}, {}}
	var begins int
	retryable := &pgconn.PgError{Code: "40001"}

	err := runTransactionWithRetry(context.Background(), func(context.Context) (pgx.Tx, error) {
		tx := transactions[begins]
		begins++
		return tx, nil
	}, maxHourlyTxRetryAttempts+1, func(context.Context, pgx.Tx) error {
		return retryable
	})

	require.ErrorIs(t, err, retryable)
	assert.Equal(t, 3, begins)
	for _, tx := range transactions {
		assert.Equal(t, 1, tx.rollbacks)
		assert.Equal(t, 0, tx.commits)
	}
}

func TestRunTransactionWithRetryStopsWhenContextCancelsDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	withRetryTiming(t, func(min, max time.Duration) time.Duration { return min }, sleepUntilContextCanceled)

	tx := &retryTx{}
	var begins int
	err := runTransactionWithRetry(ctx, func(context.Context) (pgx.Tx, error) {
		begins++
		return tx, nil
	}, 3, func(context.Context, pgx.Tx) error {
		cancel()
		return &pgconn.PgError{Code: "40P01"}
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, begins)
	assert.Equal(t, 1, tx.rollbacks)
}

func TestRunTransactionWithRetryJitterStaysWithinBackoffBound(t *testing.T) {
	var bounds [][2]time.Duration
	withRetryTiming(t, func(min, max time.Duration) time.Duration {
		bounds = append(bounds, [2]time.Duration{min, max})
		return max
	}, sleepImmediately)

	transactions := []*retryTx{{}, {}, {}}
	var begins int
	err := runTransactionWithRetry(context.Background(), func(context.Context) (pgx.Tx, error) {
		tx := transactions[begins]
		begins++
		return tx, nil
	}, 3, func(context.Context, pgx.Tx) error {
		return &pgconn.PgError{Code: "40001"}
	})

	require.Error(t, err)
	assert.Equal(t, [][2]time.Duration{
		{hourlyTxRetryMinDelay, 2 * hourlyTxRetryMinDelay},
		{hourlyTxRetryMinDelay, hourlyTxRetryMaxDelay},
	}, bounds)
}

func TestRunTransactionWithRetryReportsRetryAndExhaustionSQLStates(t *testing.T) {
	withRetryTiming(t, func(min, max time.Duration) time.Duration { return min }, sleepImmediately)

	transactions := []*retryTx{{}, {}, {}}
	var begins int
	var retries, exhausted []string
	err := runTransactionWithRetryObserved(context.Background(), func(context.Context) (pgx.Tx, error) {
		tx := transactions[begins]
		begins++
		return tx, nil
	}, 3, func(context.Context, pgx.Tx) error {
		return &pgconn.PgError{Code: "40001"}
	}, func(sqlState string) {
		retries = append(retries, sqlState)
	}, func(sqlState string) {
		exhausted = append(exhausted, sqlState)
	})

	require.Error(t, err)
	assert.Equal(t, []string{"40001", "40001"}, retries)
	assert.Equal(t, []string{"40001"}, exhausted)
}

func TestMetricsRecordsHourlyTransactionRetryOutcomes(t *testing.T) {
	m := NewMetrics(nil)
	m.IncHourlyTxRetry("40P01")
	m.IncHourlyTxRetryExhausted("40001")

	assert.Equal(t, float64(1), testutil.ToFloat64(m.HourlyTxRetries.WithLabelValues("40P01")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.HourlyTxRetryExhausted.WithLabelValues("40001")))
}

type retryTx struct {
	pgx.Tx
	commits   int
	rollbacks int
}

func (tx *retryTx) Commit(context.Context) error {
	tx.commits++
	return nil
}

func (tx *retryTx) Rollback(context.Context) error {
	tx.rollbacks++
	return nil
}

func withRetryTiming(
	t *testing.T,
	jitter func(time.Duration, time.Duration) time.Duration,
	sleep func(context.Context, time.Duration) error,
) {
	t.Helper()
	previousJitter := hourlyTxRetryJitter
	previousSleep := hourlyTxRetrySleep
	hourlyTxRetryJitter = jitter
	hourlyTxRetrySleep = sleep
	t.Cleanup(func() {
		hourlyTxRetryJitter = previousJitter
		hourlyTxRetrySleep = previousSleep
	})
}

func sleepImmediately(context.Context, time.Duration) error { return nil }

func sleepUntilContextCanceled(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
