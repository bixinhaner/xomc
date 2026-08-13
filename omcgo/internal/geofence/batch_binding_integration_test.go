package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

func TestBuildListPendingManualBindItemIDsQueryRequiresEarlierAttempt(
	t *testing.T,
) {
	jobID := uuid.New()

	query, args, err := buildListPendingManualBindItemIDsQuery(jobID, 2, 100)

	require.NoError(t, err)
	require.Contains(t, query, "status = $2")
	require.Contains(t, query, "attempt < $3")
	require.Contains(t, query, "ORDER BY device_id, id")
	require.Equal(t, []any{jobID.String(), BatchItemPending, 2, 100}, args)
}

func TestPgRepositoryProcessManualBindItemCreatesBindingStateAndUnknownWithoutLocationIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)

	result, err := repo.ProcessManualBindItem(
		context.Background(),
		processManualBindRequest(fixture, jobID, itemID, 1, 3),
	)

	require.NoError(t, err)
	require.Equal(t, BatchItemSucceeded, result.Status)
	require.NotNil(t, result.BindingID)
	var (
		bindingCount, stateCount, effectiveCount, replayCount int
		itemStatus                                            BatchItemStatus
		itemBindingID                                         *uuid.UUID
	)
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM device_geofence_bindings
		  WHERE id = $1 AND status = 'active'`,
		*result.BindingID,
	).Scan(&bindingCount))
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM device_geofence_states
		  WHERE binding_id = $1 AND confirmed_state = 'unknown'`,
		*result.BindingID,
	).Scan(&stateCount))
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM device_geofence_effective_states
		  WHERE device_id = $1 AND effective_state = 'unknown'`,
		fixture.deviceIDs[0],
	).Scan(&effectiveCount))
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT status, binding_id FROM geofence_batch_items WHERE id = $1`,
		itemID,
	).Scan(&itemStatus, &itemBindingID))
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM event_outbox
		  WHERE dedupe_key LIKE $1`,
		fmt.Sprintf("geofence.binding.evaluate:%s:%%", *result.BindingID),
	).Scan(&replayCount))
	require.Equal(t, 1, bindingCount)
	require.Equal(t, 1, stateCount)
	require.Equal(t, 1, effectiveCount)
	require.Equal(t, BatchItemSucceeded, itemStatus)
	require.Equal(t, result.BindingID, itemBindingID)
	require.Zero(t, replayCount)
}

func TestPgRepositoryProcessManualBindItemReplaysOnlyOriginalLocationPayloadIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
	payload := insertOriginalLocationObservation(t, fixture, true)
	t.Cleanup(func() {
		cleanupManualBindReplayFixture(
			context.Background(),
			fixture,
			fixture.deviceIDs[0],
		)
	})

	result, err := repo.ProcessManualBindItem(
		context.Background(),
		processManualBindRequest(fixture, jobID, itemID, 1, 3),
	)

	require.NoError(t, err)
	require.NotNil(t, result.BindingID)
	var replayPayload json.RawMessage
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT payload FROM event_outbox WHERE dedupe_key = $1`,
		fmt.Sprintf(
			"geofence.binding.evaluate:%s:%d",
			*result.BindingID,
			payload.ObservationVersion,
		),
	).Scan(&replayPayload))
	require.JSONEq(t, mustJSON(t, payload), string(replayPayload))
}

func TestPgRepositoryProcessManualBindItemDoesNotSynthesizeReplayWhenOriginalPayloadMissingIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
	insertOriginalLocationObservation(t, fixture, false)
	t.Cleanup(func() {
		cleanupManualBindReplayFixture(
			context.Background(),
			fixture,
			fixture.deviceIDs[0],
		)
	})

	result, err := repo.ProcessManualBindItem(
		context.Background(),
		processManualBindRequest(fixture, jobID, itemID, 1, 3),
	)

	require.NoError(t, err)
	require.NotNil(t, result.BindingID)
	var replayCount int
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM event_outbox WHERE dedupe_key LIKE $1`,
		fmt.Sprintf("geofence.binding.evaluate:%s:%%", *result.BindingID),
	).Scan(&replayCount))
	require.Zero(t, replayCount)
}

func TestPgRepositoryProcessManualBindItemRejectsUntrustedOriginalReplayPayloadIntegration(
	t *testing.T,
) {
	tests := []struct {
		name       string
		rawPayload func(event.DeviceLocationObservedPayload) json.RawMessage
	}{
		{
			name: "invalid typed JSON",
			rawPayload: func(event.DeviceLocationObservedPayload) json.RawMessage {
				return json.RawMessage(
					`{"device_id":17,"observation_version":7}`,
				)
			},
		},
		{
			name: "device mismatch",
			rawPayload: func(
				payload event.DeviceLocationObservedPayload,
			) json.RawMessage {
				payload.DeviceID = uuid.New()
				return json.RawMessage(mustJSON(t, payload))
			},
		},
		{
			name: "version mismatch",
			rawPayload: func(
				payload event.DeviceLocationObservedPayload,
			) json.RawMessage {
				payload.ObservationVersion++
				return json.RawMessage(mustJSON(t, payload))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, fixture := newBatchRepositoryFixture(t)
			jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
			payload := insertOriginalLocationObservation(t, fixture, false)
			insertOriginalLocationOutboxPayload(
				t,
				fixture,
				payload.ObservationVersion,
				test.rawPayload(payload),
			)
			t.Cleanup(func() {
				cleanupManualBindReplayFixture(
					context.Background(),
					fixture,
					fixture.deviceIDs[0],
				)
			})

			result, err := repo.ProcessManualBindItem(
				context.Background(),
				processManualBindRequest(fixture, jobID, itemID, 1, 3),
			)

			require.NoError(t, err)
			require.Equal(t, BatchItemSucceeded, result.Status)
			require.NotNil(t, result.BindingID)
			var replayCount int
			var confirmedState ConfirmedState
			require.NoError(t, fixture.pool.QueryRow(
				context.Background(),
				`SELECT COUNT(*) FROM event_outbox
				  WHERE dedupe_key LIKE $1`,
				fmt.Sprintf(
					"geofence.binding.evaluate:%s:%%",
					*result.BindingID,
				),
			).Scan(&replayCount))
			require.NoError(t, fixture.pool.QueryRow(
				context.Background(),
				`SELECT confirmed_state FROM device_geofence_states
				  WHERE binding_id = $1`,
				*result.BindingID,
			).Scan(&confirmedState))
			require.Zero(t, replayCount)
			require.Equal(t, ConfirmedStateUnknown, confirmedState)
		})
	}
}

func TestPgRepositoryProcessManualBindItemReplayAndNormalEventShareEvaluationIdentityIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
	payload := insertOriginalLocationObservation(t, fixture, true)
	t.Cleanup(func() {
		cleanupManualBindReplayFixture(
			context.Background(),
			fixture,
			fixture.deviceIDs[0],
		)
	})
	result, err := repo.ProcessManualBindItem(
		context.Background(),
		processManualBindRequest(fixture, jobID, itemID, 1, 3),
	)
	require.NoError(t, err)
	require.NotNil(t, result.BindingID)
	enableCoordinatorForBatchFixture(t, fixture)

	coordinator := NewPgCoordinatorRepository(storage.NewPoolDB(fixture.pool))
	normal, err := coordinator.EvaluateLocation(
		context.Background(),
		payload,
		fixture.now.Add(2*time.Minute),
	)
	require.NoError(t, err)
	require.True(t, normal.Processed)
	replay, err := coordinator.EvaluateLocation(
		context.Background(),
		payload,
		fixture.now.Add(3*time.Minute),
	)
	require.NoError(t, err)
	require.True(t, replay.NoOp)
	require.Equal(t, "duplicate_observation", replay.Reason)

	var evaluationCount int
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM geofence_evaluations
		  WHERE binding_id = $1 AND observation_version = $2`,
		*result.BindingID,
		payload.ObservationVersion,
	).Scan(&evaluationCount))
	require.Equal(t, 1, evaluationCount)
}

func TestPgRepositoryProcessManualBindItemConcurrentRunnersCreateOneBindingIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
	req := processManualBindRequest(fixture, jobID, itemID, 1, 3)
	results := make(chan BatchItemResult, 2)
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := repo.ProcessManualBindItem(context.Background(), req)
			results <- result
			errs <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var bindingID uuid.UUID
	for result := range results {
		require.Equal(t, BatchItemSucceeded, result.Status)
		require.NotNil(t, result.BindingID)
		if bindingID == uuid.Nil {
			bindingID = *result.BindingID
		}
		require.Equal(t, bindingID, *result.BindingID)
	}
	var bindingCount int
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM device_geofence_bindings
		  WHERE device_id = $1 AND rule_type = $2 AND status = 'active'`,
		fixture.deviceIDs[0],
		RuleTypePolygonAllowZone,
	).Scan(&bindingCount))
	require.Equal(t, 1, bindingCount)
}

func TestPgRepositoryProcessManualBindItemRevalidatesBindingConflictsIntegration(
	t *testing.T,
) {
	tests := []struct {
		name       string
		insertFact func(*testing.T, *batchRepositoryFixture)
		wantReason string
	}{
		{
			name: "same-geofence suspended binding",
			insertFact: func(t *testing.T, fixture *batchRepositoryFixture) {
				bindingID := uuid.New()
				fixture.otherBindingIDs = append(fixture.otherBindingIDs, bindingID)
				fixture.insertBinding(
					t,
					bindingID,
					fixture.deviceIDs[0],
					BindingStatusSuspended,
				)
			},
			wantReason: ReasonBindingSuspended,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, fixture := newBatchRepositoryFixture(t)
			jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
			test.insertFact(t, fixture)

			result, err := repo.ProcessManualBindItem(
				context.Background(),
				processManualBindRequest(fixture, jobID, itemID, 1, 3),
			)

			require.NoError(t, err)
			require.Equal(t, BatchItemSkipped, result.Status)
			require.Equal(t, test.wantReason, result.ReasonCode)
			require.Nil(t, result.BindingID)
		})
	}
}

func TestPgRepositoryProcessManualBindItemAtomicallyReassignsActiveRuleBindingIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	fixture.insertConflictingActiveBinding(t, fixture.deviceIDs[0])
	sourceBindingID := fixture.otherBindingIDs[len(fixture.otherBindingIDs)-1]
	_, err := fixture.pool.Exec(
		context.Background(),
		`INSERT INTO device_geofence_states
		    (binding_id, device_id, confirmed_state, candidate_count, state_version)
		 VALUES ($1, $2, 'outside', 0, 7)`,
		sourceBindingID,
		fixture.deviceIDs[0],
	)
	require.NoError(t, err)
	_, err = fixture.pool.Exec(
		context.Background(),
		`INSERT INTO device_geofence_effective_states
		    (device_id, effective_state, required_action_level, state_version,
		     evaluation_health, trigger_binding_id)
		 VALUES ($1, 'outside', 'deactivate', 9, 'healthy', $2)`,
		fixture.deviceIDs[0],
		sourceBindingID,
	)
	require.NoError(t, err)
	params := fixture.validCreateJobParams(t)
	accepted, err := repo.CreateManualBindJob(context.Background(), params)
	require.NoError(t, err)
	jobID := accepted.JobID
	var itemID uuid.UUID
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT id FROM geofence_batch_items
		  WHERE job_id = $1 AND device_id = $2`,
		jobID,
		fixture.deviceIDs[0],
	).Scan(&itemID))

	result, err := repo.ProcessManualBindItem(
		context.Background(),
		processManualBindRequest(fixture, jobID, itemID, 1, 3),
	)

	require.NoError(t, err)
	require.Equal(t, BatchItemSucceeded, result.Status)
	require.NotNil(t, result.BindingID)

	var (
		sourceStatus BindingStatus
		removedBy    *uuid.UUID
		removeReason string
	)
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT status, removed_by, COALESCE(remove_reason, '')
		   FROM device_geofence_bindings
		  WHERE id = $1`,
		sourceBindingID,
	).Scan(&sourceStatus, &removedBy, &removeReason))
	require.Equal(t, BindingStatusRemoved, sourceStatus)
	require.Equal(t, fixture.actorID, *removedBy)
	require.Equal(t, ReasonReassigned, removeReason)

	var activeCount int
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		   FROM device_geofence_bindings
		  WHERE device_id = $1 AND rule_type = $2 AND status = 'active'`,
		fixture.deviceIDs[0],
		RuleTypePolygonAllowZone,
	).Scan(&activeCount))
	require.Equal(t, 1, activeCount)

	var (
		newConfirmedState ConfirmedState
		candidateState    CandidateState
		candidateCount    int
		effectiveState    EffectiveState
	)
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT confirmed_state, candidate_state, candidate_count
		   FROM device_geofence_states
		  WHERE binding_id = $1`,
		*result.BindingID,
	).Scan(&newConfirmedState, &candidateState, &candidateCount))
	require.Equal(t, ConfirmedStateOutside, newConfirmedState)
	require.Equal(t, CandidateStateNone, candidateState)
	require.Zero(t, candidateCount)
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT effective_state
		   FROM device_geofence_effective_states
		  WHERE device_id = $1`,
		fixture.deviceIDs[0],
	).Scan(&effectiveState))
	require.Equal(t, EffectiveStateOutside, effectiveState)
}

func TestPgRepositoryProcessManualBindItemDoesNotMoveUnconfirmedBindingIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
	fixture.insertConflictingActiveBinding(t, fixture.deviceIDs[0])
	sourceBindingID := fixture.otherBindingIDs[len(fixture.otherBindingIDs)-1]

	result, err := repo.ProcessManualBindItem(
		context.Background(),
		processManualBindRequest(fixture, jobID, itemID, 1, 3),
	)

	require.NoError(t, err)
	require.Equal(t, BatchItemSkipped, result.Status)
	require.Equal(t, ReasonActiveRuleConflict, result.ReasonCode)
	require.Nil(t, result.BindingID)
	var sourceStatus BindingStatus
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT status FROM device_geofence_bindings WHERE id = $1`,
		sourceBindingID,
	).Scan(&sourceStatus))
	require.Equal(t, BindingStatusActive, sourceStatus)
}

func TestPgRepositoryProcessManualBindItemRecordsTransientAttemptAndFinalFailureIntegration(
	t *testing.T,
) {
	tests := []struct {
		name       string
		attempt    int
		wantStatus BatchItemStatus
	}{
		{name: "non-final", attempt: 1, wantStatus: BatchItemPending},
		{name: "final", attempt: 3, wantStatus: BatchItemFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, fixture := newBatchRepositoryFixture(t)
			jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
			injectedErr := &pgconn.PgError{Code: "40001", Message: "retry"}
			repo.hooks = &pgRepositoryHooks{
				afterManualBindItemLock: func(context.Context) error {
					return injectedErr
				},
			}

			result, err := repo.ProcessManualBindItem(
				context.Background(),
				processManualBindRequest(
					fixture,
					jobID,
					itemID,
					test.attempt,
					3,
				),
			)

			require.Error(t, err)
			require.Equal(t, test.wantStatus, result.Status)
			var status BatchItemStatus
			var attempt int
			require.NoError(t, fixture.pool.QueryRow(
				context.Background(),
				`SELECT status, attempt FROM geofence_batch_items WHERE id = $1`,
				itemID,
			).Scan(&status, &attempt))
			require.Equal(t, test.wantStatus, status)
			require.Equal(t, test.attempt, attempt)
		})
	}
}

func TestManualBindFailurePersistenceRedactsWrappedPgErrorIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	params := fixture.validCreateJobParams(t)
	accepted, err := repo.CreateManualBindJob(context.Background(), params)
	require.NoError(t, err)
	_, err = fixture.pool.Exec(
		context.Background(),
		"UPDATE async_jobs SET max_attempts = 1 WHERE id = $1",
		accepted.JobID,
	)
	require.NoError(t, err)

	const (
		secretMessage    = "database password=do-not-leak"
		secretConstraint = "uq_secret_manual_binding"
		secretSQLState   = "23505"
	)
	repo.hooks = &pgRepositoryHooks{
		afterManualBindItemLock: func(context.Context) error {
			return fmt.Errorf(
				"manual bind transaction failed: %w",
				&pgconn.PgError{
					Code:           secretSQLState,
					Message:        secretMessage,
					ConstraintName: secretConstraint,
				},
			)
		},
	}
	registry := asyncjob.NewRegistry(
		asyncjob.NewPgRepository(fixture.pool),
		"manual-bind-error-redaction",
		nil,
	)
	registry.Register(NewManualBindRunner(repo, nil))

	ran, err := registry.RunNext(context.Background(), ManualBindJobType)
	require.NoError(t, err)
	require.True(t, ran)

	job, err := repo.GetManualBindJob(
		context.Background(),
		accepted.JobID,
	)
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, asyncjob.StatusFailed, job.Status)
	require.Equal(t, commonerrors.ErrInternal.Error(), job.ErrorMessage)

	page, err := repo.ListManualBindItems(
		context.Background(),
		BatchItemFilter{
			JobID:         accepted.JobID,
			Page:          1,
			PageSize:      MaxBatchItemPageSize,
			VisibleGroups: nil,
		},
	)
	require.NoError(t, err)
	require.Len(t, page.Items, len(params.Inputs))
	for _, item := range page.Items {
		require.Equal(t, BatchItemFailed, item.Status)
		require.Equal(t, ReasonProcessingError, item.ReasonCode)
		require.Equal(
			t,
			commonerrors.ErrInternal.Error(),
			item.ErrorMessage,
		)
	}
	publicTexts := []string{job.ErrorMessage}
	for _, item := range page.Items {
		publicTexts = append(publicTexts, item.ErrorMessage)
	}
	for _, publicText := range publicTexts {
		require.NotContains(t, publicText, secretMessage)
		require.NotContains(t, publicText, secretConstraint)
		require.NotContains(t, publicText, secretSQLState)
		require.NotContains(t, publicText, "SQLSTATE")
	}
}

func TestPgRepositoryMarkManualBindItemFailureDoesNotRegressNewerAttemptIntegration(
	t *testing.T,
) {
	repo, fixture := newBatchRepositoryFixture(t)
	jobID, itemID := createManualBindProcessFixture(t, repo, fixture, 0)
	newerReq := processManualBindRequest(fixture, jobID, itemID, 2, 3)
	newerReq.ProcessedAt = fixture.now.Add(2 * time.Minute)
	olderReq := processManualBindRequest(fixture, jobID, itemID, 1, 3)
	olderReq.ProcessedAt = fixture.now.Add(time.Minute)

	newerResult, err := repo.markManualBindItemFailure(
		context.Background(),
		newerReq,
	)
	require.NoError(t, err)
	require.Equal(t, BatchItemPending, newerResult.Status)
	olderResult, err := repo.markManualBindItemFailure(
		context.Background(),
		olderReq,
	)
	require.NoError(t, err)
	require.Equal(t, BatchItemPending, olderResult.Status)

	var (
		attempt      int
		errorMessage string
		startedAt    time.Time
		updatedAt    time.Time
		finishedAt   *time.Time
	)
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT attempt, error_message, started_at, updated_at, finished_at
		   FROM geofence_batch_items
		  WHERE id = $1`,
		itemID,
	).Scan(
		&attempt,
		&errorMessage,
		&startedAt,
		&updatedAt,
		&finishedAt,
	))
	require.Equal(t, 2, attempt)
	require.Equal(t, commonerrors.ErrInternal.Error(), errorMessage)
	require.True(t, newerReq.ProcessedAt.Equal(startedAt))
	require.True(t, newerReq.ProcessedAt.Equal(updatedAt))
	require.Nil(t, finishedAt)
}

func createManualBindProcessFixture(
	t *testing.T,
	repo *PgRepository,
	fixture *batchRepositoryFixture,
	itemIndex int,
) (uuid.UUID, uuid.UUID) {
	t.Helper()
	params := fixture.validCreateJobParams(t)
	accepted, err := repo.CreateManualBindJob(context.Background(), params)
	require.NoError(t, err)
	var itemID uuid.UUID
	require.NoError(t, fixture.pool.QueryRow(
		context.Background(),
		`SELECT id FROM geofence_batch_items
		  WHERE job_id = $1 AND device_id = $2`,
		accepted.JobID,
		fixture.deviceIDs[itemIndex],
	).Scan(&itemID))
	return accepted.JobID, itemID
}

func processManualBindRequest(
	fixture *batchRepositoryFixture,
	jobID uuid.UUID,
	itemID uuid.UUID,
	attempt int,
	maxAttempts int,
) ProcessManualBindItemRequest {
	return ProcessManualBindItemRequest{
		JobID:                     jobID,
		ItemID:                    itemID,
		GeofenceID:                fixture.geofenceID,
		ExpectedGeofenceVersionID: fixture.versionID,
		ActorID:                   fixture.actorID,
		JobAttempt:                attempt,
		MaxAttempts:               maxAttempts,
		ProcessedAt:               fixture.now.Add(time.Minute),
	}
}

func insertOriginalLocationObservation(
	t *testing.T,
	fixture *batchRepositoryFixture,
	withOutbox bool,
) event.DeviceLocationObservedPayload {
	t.Helper()
	payload := event.DeviceLocationObservedPayload{
		DeviceID:           fixture.deviceIDs[0],
		SerialNumber:       "BATCH-" + fixture.deviceIDs[0].String(),
		Carrier:            "cmcc",
		ObservationVersion: 7,
		Latitude:           30.5,
		Longitude:          120.5,
		ObservedAt:         fixture.now,
		ReceivedAt:         fixture.now,
		SourcePath:         "integration",
	}
	query, args, err := storage.Psql.
		Insert("device_location_observations").
		Columns(
			"device_id", "latitude", "longitude", "observed_at",
			"received_at", "version", "source_path",
		).
		Values(
			payload.DeviceID,
			payload.Latitude,
			payload.Longitude,
			payload.ObservedAt,
			payload.ReceivedAt,
			payload.ObservationVersion,
			payload.SourcePath,
		).
		ToSql()
	require.NoError(t, err)
	_, err = fixture.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
	if withOutbox {
		query, args, err = storage.Psql.
			Insert("event_outbox").
			Columns(
				"aggregate_type", "aggregate_id", "subject", "payload", "dedupe_key",
			).
			Values(
				"device",
				payload.DeviceID.String(),
				event.SubjectDeviceLocationObserved,
				json.RawMessage(mustJSON(t, payload)),
				fmt.Sprintf(
					"%s:%s:%d",
					event.SubjectDeviceLocationObserved,
					payload.DeviceID,
					payload.ObservationVersion,
				),
			).
			ToSql()
		require.NoError(t, err)
		_, err = fixture.pool.Exec(context.Background(), query, args...)
		require.NoError(t, err)
	}
	return payload
}

func insertOriginalLocationOutboxPayload(
	t *testing.T,
	fixture *batchRepositoryFixture,
	observationVersion int64,
	rawPayload json.RawMessage,
) {
	t.Helper()
	query, args, err := storage.Psql.
		Insert("event_outbox").
		Columns(
			"aggregate_type", "aggregate_id", "subject", "payload", "dedupe_key",
		).
		Values(
			"device",
			fixture.deviceIDs[0].String(),
			event.SubjectDeviceLocationObserved,
			rawPayload,
			fmt.Sprintf(
				"%s:%s:%d",
				event.SubjectDeviceLocationObserved,
				fixture.deviceIDs[0],
				observationVersion,
			),
		).
		ToSql()
	require.NoError(t, err)
	_, err = fixture.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
}

func cleanupManualBindReplayFixture(
	ctx context.Context,
	fixture *batchRepositoryFixture,
	deviceID uuid.UUID,
) {
	_, _ = fixture.pool.Exec(
		ctx,
		`UPDATE device_geofence_states
		    SET last_evaluation_id = NULL
		  WHERE device_id = $1`,
		deviceID,
	)
	_, _ = fixture.pool.Exec(
		ctx,
		"DELETE FROM geofence_evaluations WHERE device_id = $1",
		deviceID,
	)
	_, _ = fixture.pool.Exec(
		ctx,
		`DELETE FROM event_outbox
		  WHERE aggregate_id = $1 OR payload->>'device_id' = $1`,
		deviceID.String(),
	)
	_, _ = fixture.pool.Exec(
		ctx,
		"DELETE FROM device_location_observations WHERE device_id = $1",
		deviceID,
	)
}

func enableCoordinatorForBatchFixture(
	t *testing.T,
	fixture *batchRepositoryFixture,
) {
	t.Helper()
	setSystemMode(
		t,
		context.Background(),
		fixture.pool,
		RuntimeModeObserve,
	)
	query, args, err := storage.Psql.
		Insert("geofence_carrier_settings").
		Columns("carrier", "mode", "updated_by", "updated_at").
		Values(
			"cmcc",
			RuntimeModeObserve,
			fixture.actorID,
			fixture.now,
		).
		Suffix(
			"ON CONFLICT (carrier) DO UPDATE SET mode = EXCLUDED.mode, " +
				"updated_at = EXCLUDED.updated_at",
		).
		ToSql()
	require.NoError(t, err)
	_, err = fixture.pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = fixture.pool.Exec(
			context.Background(),
			"DELETE FROM geofence_carrier_settings WHERE carrier = $1",
			"cmcc",
		)
		_ = setSystemModeValue(
			context.Background(),
			fixture.pool,
			RuntimeModeOff,
		)
	})
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return string(raw)
}
