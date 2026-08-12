package northbound

import (
	"context"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/paramsync"
)

type northboundParamSyncService interface {
	GetRequest(context.Context, uuid.UUID) (*paramsync.SyncRequest, error)
	GetRun(context.Context, uuid.UUID) (*paramsync.SyncRun, error)
	FindRequestByIdempotency(context.Context, string, string) (*paramsync.SyncRequest, error)
}

// SetParamSyncService wires durable parameter-sync status lookup into the
// legacy-shaped northbound job result endpoint.
func (r *Router) SetParamSyncService(svc northboundParamSyncService) {
	r.paramSyncService = svc
}
