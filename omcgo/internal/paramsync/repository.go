package paramsync

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrRequestStateConflict = errors.New("parameter sync request state changed")

type Repository interface {
	CreateRequest(ctx context.Context, req *SyncRequest) error
	CreateAutomaticRequest(ctx context.Context, req *SyncRequest, now time.Time) (bool, error)
	GetRequest(ctx context.Context, id uuid.UUID) (*SyncRequest, error)
	FindRequestByIdempotency(ctx context.Context, callerType, key string) (*SyncRequest, error)
	CompleteRequest(ctx context.Context, id uuid.UUID, status RequestStatus, code ResultCode, activeRunID *uuid.UUID, message string) error
	FailRunAndRequest(ctx context.Context, runID, requestID uuid.UUID, code ResultCode, message string) error
	CreateOrDeduplicateRun(ctx context.Context, req *SyncRequest) (*StartResult, error)
	GetRun(ctx context.Context, id uuid.UUID) (*SyncRun, error)
	GetActiveRunByDevice(ctx context.Context, deviceID uuid.UUID) (*SyncRun, error)
	ListRequestsByDevice(ctx context.Context, deviceID uuid.UUID, limit int) ([]*SyncRequest, error)
	InsertTaskResultIfAbsent(ctx context.Context, result *TaskResult) (bool, error)
	ClaimRunForFinalize(ctx context.Context, runID uuid.UUID) (bool, error)
}
