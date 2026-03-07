package provision

import (
	"context"

	"github.com/google/uuid"
)

// ProvisioningTaskRepository defines the persistence interface for provisioning tasks.
type ProvisioningTaskRepository interface {
	Create(ctx context.Context, task *ProvisioningTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error)
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error)
	Update(ctx context.Context, task *ProvisioningTask) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error
	List(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error)
	CountByStatus(ctx context.Context) (map[ProvisioningState]int64, error)
}
