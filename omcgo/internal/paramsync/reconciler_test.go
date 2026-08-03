package paramsync

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

type nullableErrorCodeRow struct {
	runID     uuid.UUID
	requestID uuid.UUID
}

func (r nullableErrorCodeRow) Scan(dest ...any) error {
	if len(dest) != 7 {
		return fmt.Errorf("unexpected destination count: %d", len(dest))
	}
	*(dest[0].(*string)) = uuid.NewString()
	*(dest[1].(*string)) = "SN-NULL-ERROR-CODE"
	*(dest[2].(*task.TaskStatus)) = task.TaskStatusCancelled
	*(dest[3].(*pgtype.Int4)) = pgtype.Int4{Valid: false}
	*(dest[4].(*string)) = "parameter sync run failed"
	*(dest[5].(*uuid.UUID)) = r.runID
	*(dest[6].(*uuid.UUID)) = r.requestID
	return nil
}

func TestScanMissingTaskResultAcceptsNullErrorCode(t *testing.T) {
	runID, requestID := uuid.New(), uuid.New()

	result, err := scanMissingTaskResult(nullableErrorCodeRow{runID: runID, requestID: requestID})

	require.NoError(t, err)
	assert.Equal(t, task.TaskStatusCancelled, result.status)
	assert.Empty(t, result.errorCode)
	assert.Equal(t, runID, result.runID)
	assert.Equal(t, requestID, result.requestID)
}

func TestApplyAuthoritativeRunCountsDoesNotDoubleCountReconciledTerminalTask(t *testing.T) {
	run := &SyncRun{
		ExpectedTaskCount:  1,
		TerminalTaskCount:  1,
		ProcessedTaskCount: 0,
	}

	applyAuthoritativeRunCounts(run, authoritativeRunCounts{
		expected: 1, terminal: 1, processed: 1, failed: 1,
	})

	assert.Equal(t, 1, run.ExpectedTaskCount)
	assert.Equal(t, 1, run.TerminalTaskCount)
	assert.Equal(t, 1, run.ProcessedTaskCount)
	assert.Equal(t, 1, run.FailedTaskCount)
}

func TestApplyAuthoritativeRunCountsNeverShrinksPlannedTaskCount(t *testing.T) {
	run := &SyncRun{ExpectedTaskCount: 20}

	applyAuthoritativeRunCounts(run, authoritativeRunCounts{
		expected: 10, terminal: 10, processed: 10,
	})

	assert.Equal(t, 20, run.ExpectedTaskCount)
	assert.Equal(t, 10, run.TerminalTaskCount)
	assert.Equal(t, 10, run.ProcessedTaskCount)
}

func TestRecoverMissingResultsWithoutProcessorOrBusReturnsError(t *testing.T) {
	_, err := NewReconciler(nil, nil, nil).RecoverMissingResults(context.Background(), 20, 200, 200)

	require.ErrorContains(t, err, "requires result processor or event bus")
}

func TestMissingResultRecoverySelectsDurableTerminalTasksDirectly(t *testing.T) {
	query := missingResultRunCandidatesSQL

	require.Contains(t, query, "EXISTS")
	require.Contains(t, query, "t.status IN ('completed','failed','expired','cancelled')")
	require.Contains(t, query, "res.task_id IS NULL")
	require.NotContains(t, query, "terminal_task_count = expected_task_count")
	require.NotContains(t, query, "processed_task_count < expected_task_count")
}

func TestProcessMissingTaskResultsContinuesAfterOneMalformedResult(t *testing.T) {
	processor := &failFirstRecoveryProcessor{}
	firstRun, secondRun := uuid.New(), uuid.New()
	processed, err := processMissingTaskResults(context.Background(), processor, []missingTaskResult{
		{taskID: uuid.NewString(), runID: firstRun, requestID: uuid.New(), status: task.TaskStatusCompleted},
		{taskID: uuid.NewString(), runID: secondRun, requestID: uuid.New(), status: task.TaskStatusCompleted},
	})

	require.ErrorContains(t, err, "permanent malformed result")
	assert.Equal(t, 1, processed)
	assert.Equal(t, 2, processor.calls)
	require.Len(t, processor.payloads, 1)
	assert.Equal(t, secondRun, processor.payloads[0].RunID)
}

func TestRunCountReconciliationRestrictsAggregationToCandidateBatch(t *testing.T) {
	first, second := uuid.New(), uuid.New()

	query, args, err := buildRunCountReconciliationSQL([]uuid.UUID{first, second})

	require.NoError(t, err)
	require.Contains(t, query, "source_id=ANY($1)")
	require.Contains(t, query, "WITH batch AS")
	require.Contains(t, query, "task_rows AS MATERIALIZED")
	require.Contains(t, query, "JOIN parameter_sync_task_results res ON res.run_id=t.run_id AND res.task_id=t.id")
	require.NotContains(t, query, "FROM parameter_sync_runs run")
	require.Contains(t, query, "GREATEST(run.expected_task_count, actual.expected)")
	require.Len(t, args, 1)
	require.Equal(t, []uuid.UUID{first, second}, args[0])
}

func TestHistoricalResultNormalizationRestrictsWorkToCandidateBatch(t *testing.T) {
	query := historicalResultNormalizationSQL

	require.Contains(t, query, "run.id=ANY($2)")
	require.NotContains(t, query, "FROM parameter_sync_runs run, device_tasks t\nWHERE res.status='received' AND run.id=res.run_id\n  AND run.status")
}

func TestRunCountSweepFitsWithinIndependentMaintenanceDeadline(t *testing.T) {
	require.Equal(t, 50, runCountReconcileBatchSize)
}

func TestStagingCleanupScansABoundedKeysetPage(t *testing.T) {
	query := stagingCleanupSQL

	require.Contains(t, query, "ORDER BY run_id, parameter_path")
	require.Contains(t, query, "LIMIT $3")
	require.Contains(t, query, "(run_id, parameter_path) > ($1, $2)")
	require.Contains(t, query, "DELETE FROM parameter_sync_staging_values s USING page, parameter_sync_runs run")
	require.NotContains(t, query, "$4::boolean OR")
	require.NotContains(t, query, "DELETE FROM parameter_sync_staging_values s WHERE (s.run_id, s.parameter_path) IN")
	require.NotContains(t, stagingCleanupInitialSQL, "WHERE $4::boolean OR")
}

func TestParamSyncMetricsAvoidUnboundedExactCounts(t *testing.T) {
	require.Contains(t, outboxBacklogMetricSQL, "status IN ('pending','failed')")
	require.Contains(t, outboxBacklogMetricSQL, "status='delivering'")
	require.Contains(t, stagingRowsMetricSQL, "reltuples")
	require.NotContains(t, stagingRowsMetricSQL, "count(*)")
}

func TestRunConvergenceForwardSelectionUsesBoundedSweepKeyset(t *testing.T) {
	startedAt := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	cursor := runConvergenceCursor{StartedAt: startedAt, ID: uuid.New()}
	sweepEnd := runConvergenceCursor{StartedAt: startedAt.Add(time.Hour), ID: uuid.New()}
	excluded := []uuid.UUID{uuid.New()}

	query, args, err := buildRunConvergenceForwardSelectSQL(&cursor, &sweepEnd, excluded, 40)

	require.NoError(t, err)
	require.Contains(t, query, "status IN")
	require.Contains(t, query, "(run.started_at, run.id) >")
	require.Contains(t, query, "(run.started_at, run.id) <=")
	require.NotContains(t, query, "device_tasks")
	require.NotContains(t, query, "parameter_sync_task_results")
	require.NotContains(t, strings.ToUpper(query), "LATERAL")
	require.Contains(t, query, "LIMIT 40")
	require.Contains(t, query, "FOR UPDATE SKIP LOCKED")
	require.NotContains(t, strings.ToUpper(query), "OFFSET")
	require.Equal(t, []interface{}{startedAt, cursor.ID, sweepEnd.StartedAt, sweepEnd.ID, excluded}, args)
}

func TestRunConvergencePrioritySelectionHasIndependentQuota(t *testing.T) {
	query, args, err := buildRunConvergencePrioritySelectSQL(10)

	require.NoError(t, err)
	require.Contains(t, query, "status IN")
	require.Contains(t, query, "run.expected_task_count>0")
	require.NotContains(t, query, "device_tasks")
	require.Contains(t, query, "LIMIT 10")
	require.Contains(t, query, "FOR UPDATE SKIP LOCKED")
	require.Empty(t, args)
}

func TestRunConvergenceCursorAdvancesOnlyFromForwardCandidates(t *testing.T) {
	base := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	priority := runConvergenceCandidate{id: uuid.New(), startedAt: base.Add(3 * time.Hour)}
	first := runConvergenceCandidate{id: uuid.New(), startedAt: base.Add(time.Hour), advancesCursor: true}
	last := runConvergenceCandidate{id: uuid.New(), startedAt: base.Add(2 * time.Hour), advancesCursor: true}

	cursor, hasForward := nextRunConvergenceCursor([]runConvergenceCandidate{priority, first, last})

	require.True(t, hasForward)
	require.Equal(t, last.startedAt, cursor.StartedAt)
	require.Equal(t, last.id, cursor.ID)
	_, hasForward = nextRunConvergenceCursor([]runConvergenceCandidate{priority})
	require.False(t, hasForward)
}
