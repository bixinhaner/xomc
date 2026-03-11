package software

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// FirmwareRepository provides persistence for firmware versions.
type FirmwareRepository interface {
	Create(ctx context.Context, fw *FirmwareVersion) error
	GetByID(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error)
	List(ctx context.Context, filter FirmwareFilter) (*model.ListResponse[FirmwareVersion], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// UpgradeTaskRepository provides persistence for upgrade tasks.
type UpgradeTaskRepository interface {
	Create(ctx context.Context, task *UpgradeTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error
	List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error)
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeTask, error)
	CountByBatchStatus(ctx context.Context, batchID uuid.UUID) (map[UpgradeState]int64, error)
}
