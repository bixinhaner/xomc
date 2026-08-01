package paramsync

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestReconcileRunCountsUsesIncrementalCountersBeforeBoundary(t *testing.T) {
	run := &SyncRun{ExpectedTaskCount: 3}
	loads := 0

	authoritative, err := reconcileRunCountsForInsertedResult(
		context.Background(), nil, run, false,
		func(context.Context, pgx.Tx, SyncRun) (authoritativeRunCounts, error) {
			loads++
			return authoritativeRunCounts{}, nil
		},
	)

	require.NoError(t, err)
	require.False(t, authoritative)
	require.Zero(t, loads)
	require.Equal(t, 1, run.TerminalTaskCount)
	require.Equal(t, 1, run.ProcessedTaskCount)
	require.Zero(t, run.FailedTaskCount)
}

func TestReconcileRunCountsLoadsAuthoritativeCountersAtFinalBoundary(t *testing.T) {
	run := &SyncRun{ExpectedTaskCount: 1}
	loads := 0

	authoritative, err := reconcileRunCountsForInsertedResult(
		context.Background(), nil, run, false,
		func(context.Context, pgx.Tx, SyncRun) (authoritativeRunCounts, error) {
			loads++
			return authoritativeRunCounts{expected: 1, terminal: 1, processed: 1}, nil
		},
	)

	require.NoError(t, err)
	require.True(t, authoritative)
	require.Equal(t, 1, loads)
	require.Equal(t, 1, run.ExpectedTaskCount)
	require.Equal(t, 1, run.TerminalTaskCount)
	require.Equal(t, 1, run.ProcessedTaskCount)
}

func TestReconcileRunCountsLoadsAuthoritativeCountersOnFailure(t *testing.T) {
	run := &SyncRun{ExpectedTaskCount: 3}
	loads := 0

	authoritative, err := reconcileRunCountsForInsertedResult(
		context.Background(), nil, run, true,
		func(context.Context, pgx.Tx, SyncRun) (authoritativeRunCounts, error) {
			loads++
			return authoritativeRunCounts{expected: 3, terminal: 2, processed: 2, failed: 1}, nil
		},
	)

	require.NoError(t, err)
	require.True(t, authoritative)
	require.Equal(t, 1, loads)
	require.Equal(t, 2, run.TerminalTaskCount)
	require.Equal(t, 2, run.ProcessedTaskCount)
	require.Equal(t, 1, run.FailedTaskCount)
}

func TestReconcileRunCountsRepairsInvalidIncrementalState(t *testing.T) {
	run := &SyncRun{ExpectedTaskCount: 3, TerminalTaskCount: 3, ProcessedTaskCount: 3}
	loads := 0

	authoritative, err := reconcileRunCountsForInsertedResult(
		context.Background(), nil, run, false,
		func(context.Context, pgx.Tx, SyncRun) (authoritativeRunCounts, error) {
			loads++
			return authoritativeRunCounts{expected: 3, terminal: 2, processed: 2}, nil
		},
	)

	require.NoError(t, err)
	require.True(t, authoritative)
	require.Equal(t, 1, loads)
	require.Equal(t, 2, run.TerminalTaskCount)
	require.Equal(t, 2, run.ProcessedTaskCount)
}
