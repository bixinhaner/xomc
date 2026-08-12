package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

type ManualParamSyncStart struct {
	Used       bool
	TaskCount  int
	RequestID  uuid.UUID
	RunID      *uuid.UUID
	Status     string
	ResultCode string
}

// ParamSyncStarter exposes the durable request/run path to the device package
// without creating a device -> paramsync package dependency cycle.
type ParamSyncStarter interface {
	StartManualSyncDetailed(ctx context.Context, dev *model.Device, sourceID string, parameterPaths []string) (*ManualParamSyncStart, error)
}
