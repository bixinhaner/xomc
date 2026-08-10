package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const manualBindRunnerPageSize = 100

type ManualBindRunnerRepository interface {
	ListPendingManualBindItemIDs(
		context.Context,
		uuid.UUID,
		int,
		int,
	) ([]uuid.UUID, error)
	ProcessManualBindItem(
		context.Context,
		ProcessManualBindItemRequest,
	) (BatchItemResult, error)
	GetBatchProgress(context.Context, uuid.UUID) (BatchProgress, error)
}

type ProcessManualBindItemRequest struct {
	JobID                     uuid.UUID
	ItemID                    uuid.UUID
	GeofenceID                uuid.UUID
	ExpectedGeofenceVersionID uuid.UUID
	ActorID                   uuid.UUID
	JobAttempt                int
	MaxAttempts               int
	ProcessedAt               time.Time
}

type BatchItemResult struct {
	Status     BatchItemStatus
	ReasonCode string
	BindingID  *uuid.UUID
}

type ManualBindRunner struct {
	repository ManualBindRunnerRepository
	metrics    *BatchMetrics
	now        func() time.Time
}

type manualBindRunError struct {
	cause error
}

func (e *manualBindRunError) Error() string {
	return manualBindPublicErrorMessage()
}

func (e *manualBindRunError) Unwrap() error {
	return e.cause
}

func newManualBindRunError(cause error) error {
	if cause == nil {
		return nil
	}
	return &manualBindRunError{cause: cause}
}

func NewManualBindRunner(
	repository ManualBindRunnerRepository,
	metrics *BatchMetrics,
) *ManualBindRunner {
	return &ManualBindRunner{
		repository: repository,
		metrics:    metrics,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (r *ManualBindRunner) JobType() string {
	return ManualBindJobType
}

func (r *ManualBindRunner) Run(
	ctx context.Context,
	job *asyncjob.Job,
) (json.RawMessage, error) {
	startedAt := r.now()
	processedCount := 0
	defer func() {
		r.metrics.observeBind(r.now().Sub(startedAt), processedCount)
	}()

	if job == nil {
		return nil, fmt.Errorf("manual bind job is required: %w", commonerrors.ErrInvalidInput)
	}
	payload, err := decodeManualBindJobPayload(job.Payload)
	if err != nil {
		return nil, newManualBindRunError(err)
	}

	var processingErr error
	for {
		ids, err := r.repository.ListPendingManualBindItemIDs(
			ctx,
			job.ID,
			job.Attempt,
			manualBindRunnerPageSize,
		)
		if err != nil {
			return nil, newManualBindRunError(
				errors.Join(
					processingErr,
					fmt.Errorf(
						"list pending manual bind items: %w",
						err,
					),
				),
			)
		}
		if len(ids) == 0 {
			break
		}
		stopPagingAfterPage := false
		for _, itemID := range ids {
			result, err := r.repository.ProcessManualBindItem(
				ctx,
				ProcessManualBindItemRequest{
					JobID:                     job.ID,
					ItemID:                    itemID,
					GeofenceID:                payload.GeofenceID,
					ExpectedGeofenceVersionID: payload.GeofenceVersionID,
					ActorID:                   payload.RequestedBy,
					JobAttempt:                job.Attempt,
					MaxAttempts:               job.MaxAttempts,
					ProcessedAt:               r.now(),
				},
			)
			processedCount++
			if result.Status != "" {
				r.metrics.observeItem(result.Status, result.ReasonCode)
			}
			if err != nil {
				processingErr = errors.Join(
					processingErr,
					fmt.Errorf(
						"process manual bind item %s: %w",
						itemID,
						err,
					),
				)
				if result.Status == "" {
					stopPagingAfterPage = true
				}
			}
		}
		if stopPagingAfterPage {
			return nil, newManualBindRunError(processingErr)
		}
	}

	progress, err := r.repository.GetBatchProgress(ctx, job.ID)
	if err != nil {
		return nil, newManualBindRunError(
			errors.Join(
				processingErr,
				fmt.Errorf("get manual bind progress: %w", err),
			),
		)
	}
	if processingErr != nil {
		return nil, newManualBindRunError(processingErr)
	}
	result, err := json.Marshal(progress)
	if err != nil {
		return nil, newManualBindRunError(
			fmt.Errorf("marshal manual bind progress: %w", err),
		)
	}
	return result, nil
}

func decodeManualBindJobPayload(raw json.RawMessage) (ManualBindJobPayload, error) {
	var payload ManualBindJobPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ManualBindJobPayload{}, fmt.Errorf(
			"decode manual bind job payload: %w",
			err,
		)
	}
	if payload.SchemaVersion != ManualBindPayloadVersion ||
		payload.GeofenceID == uuid.Nil ||
		payload.GeofenceVersionID == uuid.Nil ||
		payload.RequestedBy == uuid.Nil {
		return ManualBindJobPayload{}, fmt.Errorf(
			"invalid manual bind job payload: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	return payload, nil
}

var _ asyncjob.JobRunner = (*ManualBindRunner)(nil)
var _ ManualBindRunnerRepository = (*PgRepository)(nil)
