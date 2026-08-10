package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestManualBindRunnerProcessesAllPendingItemsAndReturnsSummary(t *testing.T) {
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}
	repo := &fakeManualBindRunnerRepository{
		pendingPages: [][]uuid.UUID{itemIDs, nil},
		results: []BatchItemResult{
			{Status: BatchItemSucceeded},
			{Status: BatchItemSkipped, ReasonCode: ReasonActiveRuleConflict},
		},
		progress: BatchProgress{Total: 2, Succeeded: 1, Skipped: 1},
	}
	runner := NewManualBindRunner(repo, nil)

	result, err := runner.Run(context.Background(), manualBindAsyncJob(t, 1, 3))

	require.NoError(t, err)
	require.JSONEq(
		t,
		`{"total":2,"pending":0,"succeeded":1,"skipped":1,"failed":0}`,
		string(result),
	)
	require.Len(t, repo.processRequests, 2)
	require.Equal(t, itemIDs[0], repo.processRequests[0].ItemID)
	require.Equal(t, itemIDs[1], repo.processRequests[1].ItemID)
	require.Equal(t, 1, repo.processRequests[0].JobAttempt)
	require.Equal(t, 3, repo.processRequests[0].MaxAttempts)
}

func TestManualBindRunnerLeavesTransientItemForNextAttemptAndContinues(t *testing.T) {
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}
	transientErr := errors.New("temporary database failure")
	repo := &fakeManualBindRunnerRepository{
		pendingPages: [][]uuid.UUID{itemIDs, nil},
		results: []BatchItemResult{
			{Status: BatchItemPending},
			{Status: BatchItemSucceeded},
		},
		processErrors: []error{transientErr, nil},
		progress:      BatchProgress{Total: 2, Pending: 1, Succeeded: 1},
	}
	runner := NewManualBindRunner(repo, nil)

	_, err := runner.Run(context.Background(), manualBindAsyncJob(t, 1, 3))

	require.EqualError(t, err, commonerrors.ErrInternal.Error())
	require.ErrorIs(t, err, transientErr)
	require.Len(t, repo.processRequests, 2)
	require.Equal(t, itemIDs[1], repo.processRequests[1].ItemID)
	require.Equal(t, 2, repo.listCalls)
}

func TestManualBindRunnerStopsPagingAfterUnpersistedFailuresButFinishesCurrentPage(
	t *testing.T,
) {
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}
	persistenceErr := errors.New("persist item attempt failed")
	unexpectedListErr := errors.New("runner listed the same page again")
	repo := &fakeManualBindRunnerRepository{
		repeatedPendingPage: itemIDs,
		repeatedProcessErr:  persistenceErr,
		maxRepeatedLists:    2,
		repeatedListErr:     unexpectedListErr,
	}
	runner := NewManualBindRunner(repo, nil)

	_, err := runner.Run(context.Background(), manualBindAsyncJob(t, 1, 3))

	require.ErrorIs(t, err, persistenceErr)
	require.NotErrorIs(t, err, unexpectedListErr)
	require.Equal(t, 1, repo.listCalls)
	require.Len(t, repo.processRequests, len(itemIDs))
	require.Equal(t, itemIDs[0], repo.processRequests[0].ItemID)
	require.Equal(t, itemIDs[1], repo.processRequests[1].ItemID)
}

func TestManualBindRunnerPersistedErrorStatusContinuesPagingWithoutSamePageRetry(
	t *testing.T,
) {
	tests := []struct {
		name   string
		status BatchItemStatus
	}{
		{name: "pending retry persisted", status: BatchItemPending},
		{name: "final failure persisted", status: BatchItemFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transientID := uuid.New()
			laterSamePageID := uuid.New()
			nextPageID := uuid.New()
			transientErr := errors.New("temporary database failure")
			repo := &fakeManualBindRunnerRepository{
				pendingPages: [][]uuid.UUID{
					{transientID, laterSamePageID},
					{nextPageID},
					nil,
				},
				results: []BatchItemResult{
					{
						Status:     test.status,
						ReasonCode: ReasonProcessingError,
					},
					{Status: BatchItemSucceeded},
					{Status: BatchItemSucceeded},
				},
				processErrors: []error{transientErr, nil, nil},
				progress: BatchProgress{
					Total: 3, Pending: 1, Succeeded: 2,
				},
			}
			runner := NewManualBindRunner(repo, nil)

			_, err := runner.Run(
				context.Background(),
				manualBindAsyncJob(t, 1, 3),
			)

			require.ErrorIs(t, err, transientErr)
			require.Equal(t, 3, repo.listCalls)
			require.Len(t, repo.processRequests, 3)
			require.Equal(t, transientID, repo.processRequests[0].ItemID)
			require.Equal(t, laterSamePageID, repo.processRequests[1].ItemID)
			require.Equal(t, nextPageID, repo.processRequests[2].ItemID)
		})
	}
}

func TestManualBindRunnerJoinsProcessingAndLaterListErrors(t *testing.T) {
	itemID := uuid.New()
	processErr := errors.New("persisted item failure")
	listErr := errors.New("list next page failed")
	repo := &fakeManualBindRunnerRepository{
		pendingPages: [][]uuid.UUID{{itemID}},
		listErrors:   []error{nil, listErr},
		results: []BatchItemResult{{
			Status:     BatchItemPending,
			ReasonCode: ReasonProcessingError,
		}},
		processErrors: []error{processErr},
	}
	runner := NewManualBindRunner(repo, nil)

	_, err := runner.Run(context.Background(), manualBindAsyncJob(t, 1, 3))

	require.ErrorIs(t, err, processErr)
	require.ErrorIs(t, err, listErr)
	require.Equal(t, 2, repo.listCalls)
}

func TestManualBindRunnerJoinsProcessingAndProgressErrors(t *testing.T) {
	itemID := uuid.New()
	processErr := errors.New("persisted item failure")
	progressErr := errors.New("load progress failed")
	repo := &fakeManualBindRunnerRepository{
		pendingPages: [][]uuid.UUID{{itemID}, nil},
		results: []BatchItemResult{{
			Status:     BatchItemPending,
			ReasonCode: ReasonProcessingError,
		}},
		processErrors: []error{processErr},
		progressErr:   progressErr,
	}
	runner := NewManualBindRunner(repo, nil)

	_, err := runner.Run(context.Background(), manualBindAsyncJob(t, 1, 3))

	require.ErrorIs(t, err, processErr)
	require.ErrorIs(t, err, progressErr)
}

func TestManualBindRunnerRejectsMalformedPayloadBeforeRepositoryAccess(
	t *testing.T,
) {
	repo := &fakeManualBindRunnerRepository{}
	runner := NewManualBindRunner(repo, nil)

	_, err := runner.Run(context.Background(), &asyncjob.Job{
		ID:          uuid.New(),
		Attempt:     1,
		MaxAttempts: 3,
		Payload:     json.RawMessage(`{"schema_version":99}`),
	})

	require.Error(t, err)
	require.Zero(t, repo.listCalls)
}

func TestManualBindRunErrorExposesPublicSummaryAndPreservesRawCause(t *testing.T) {
	raw := &pgconn.PgError{
		Code:           "23505",
		Message:        "database password=do-not-leak",
		ConstraintName: "uq_secret_manual_binding",
	}

	err := newManualBindRunError(fmt.Errorf("transaction failed: %w", raw))

	require.EqualError(t, err, commonerrors.ErrInternal.Error())
	var unwrapped *pgconn.PgError
	require.ErrorAs(t, err, &unwrapped)
	require.Same(t, raw, unwrapped)
	require.NotContains(t, err.Error(), raw.Message)
	require.NotContains(t, err.Error(), raw.ConstraintName)
	require.NotContains(t, err.Error(), raw.Code)
	require.NotContains(t, err.Error(), "SQLSTATE")
}

func TestBatchMetricsUseBoundedStatusAndReasonLabels(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewBatchMetrics(registry)

	metrics.observeItem(BatchItemSucceeded, "")
	metrics.observeItem(BatchItemSkipped, ReasonActiveRuleConflict)
	metrics.observeItem(BatchItemStatus("unexpected"), "device-"+uuid.NewString())
	metrics.observeBind(125*time.Millisecond, 3)

	require.Equal(
		t,
		float64(1),
		testutil.ToFloat64(
			metrics.itemsTotal.WithLabelValues(string(BatchItemSucceeded), "none"),
		),
	)
	require.Equal(
		t,
		float64(1),
		testutil.ToFloat64(
			metrics.itemsTotal.WithLabelValues(
				string(BatchItemSkipped),
				ReasonActiveRuleConflict,
			),
		),
	)
	require.Equal(
		t,
		float64(1),
		testutil.ToFloat64(metrics.itemsTotal.WithLabelValues("unknown", "other")),
	)
	require.NoError(t, testutil.GatherAndCompare(
		registry,
		strings.NewReader(`
# HELP omc_geofence_batch_bind_size Number of devices selected in one manual geofence binding run.
# TYPE omc_geofence_batch_bind_size histogram
omc_geofence_batch_bind_size_bucket{le="1"} 0
omc_geofence_batch_bind_size_bucket{le="5"} 1
omc_geofence_batch_bind_size_bucket{le="10"} 1
omc_geofence_batch_bind_size_bucket{le="25"} 1
omc_geofence_batch_bind_size_bucket{le="50"} 1
omc_geofence_batch_bind_size_bucket{le="100"} 1
omc_geofence_batch_bind_size_bucket{le="250"} 1
omc_geofence_batch_bind_size_bucket{le="500"} 1
omc_geofence_batch_bind_size_bucket{le="1000"} 1
omc_geofence_batch_bind_size_bucket{le="+Inf"} 1
omc_geofence_batch_bind_size_sum 3
omc_geofence_batch_bind_size_count 1
`),
		"omc_geofence_batch_bind_size",
	))
}

func manualBindAsyncJob(t *testing.T, attempt, maxAttempts int) *asyncjob.Job {
	t.Helper()
	payload, err := json.Marshal(ManualBindJobPayload{
		SchemaVersion:     ManualBindPayloadVersion,
		GeofenceID:        uuid.New(),
		GeofenceVersionID: uuid.New(),
		RequestedBy:       uuid.New(),
		Reason:            "approved",
	})
	require.NoError(t, err)
	return &asyncjob.Job{
		ID:          uuid.New(),
		JobType:     ManualBindJobType,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		Payload:     payload,
	}
}

type fakeManualBindRunnerRepository struct {
	pendingPages        [][]uuid.UUID
	repeatedPendingPage []uuid.UUID
	maxRepeatedLists    int
	repeatedListErr     error
	listErrors          []error
	results             []BatchItemResult
	processErrors       []error
	repeatedProcessErr  error
	progress            BatchProgress
	progressErr         error
	listErr             error
	listCalls           int
	processRequests     []ProcessManualBindItemRequest
}

func (r *fakeManualBindRunnerRepository) ListPendingManualBindItemIDs(
	_ context.Context,
	_ uuid.UUID,
	_ int,
	_ int,
) ([]uuid.UUID, error) {
	r.listCalls++
	listIndex := r.listCalls - 1
	if listIndex < len(r.listErrors) && r.listErrors[listIndex] != nil {
		return nil, r.listErrors[listIndex]
	}
	if r.listErr != nil {
		return nil, r.listErr
	}
	if r.repeatedPendingPage != nil {
		if r.maxRepeatedLists > 0 && r.listCalls > r.maxRepeatedLists {
			return nil, r.repeatedListErr
		}
		return append([]uuid.UUID(nil), r.repeatedPendingPage...), nil
	}
	if listIndex >= len(r.pendingPages) {
		return nil, nil
	}
	return r.pendingPages[listIndex], nil
}

func (r *fakeManualBindRunnerRepository) ProcessManualBindItem(
	_ context.Context,
	req ProcessManualBindItemRequest,
) (BatchItemResult, error) {
	r.processRequests = append(r.processRequests, req)
	if r.repeatedProcessErr != nil {
		return BatchItemResult{}, r.repeatedProcessErr
	}
	index := len(r.processRequests) - 1
	var result BatchItemResult
	if index < len(r.results) {
		result = r.results[index]
	}
	var err error
	if index < len(r.processErrors) {
		err = r.processErrors[index]
	}
	return result, err
}

func (r *fakeManualBindRunnerRepository) GetBatchProgress(
	_ context.Context,
	_ uuid.UUID,
) (BatchProgress, error) {
	return r.progress, r.progressErr
}

var _ ManualBindRunnerRepository = (*fakeManualBindRunnerRepository)(nil)

func fixedRunnerNow() time.Time {
	return time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
}
