package paramsync

import (
	"context"
	"fmt"
	"testing"

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

func TestRecoverMissingResultsWithoutProcessorOrBusReturnsError(t *testing.T) {
	_, err := NewReconciler(nil, nil, nil).RecoverMissingResults(context.Background(), 20, 200, 200)

	require.ErrorContains(t, err, "requires result processor or event bus")
}
