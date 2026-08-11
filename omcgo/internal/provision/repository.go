package provision

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProvisioningTaskRepository defines the persistence interface for provisioning tasks.
type ProvisioningTaskRepository interface {
	Create(ctx context.Context, task *ProvisioningTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error)
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error)
	GetByDelegatedTaskID(ctx context.Context, delegatedTaskID uuid.UUID, deviceID uuid.UUID) (*ProvisioningTask, error)
	Update(ctx context.Context, task *ProvisioningTask) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error
	List(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error)
	CountByStatus(ctx context.Context) (map[ProvisioningState]int64, error)
	// ListActivationChecksDue returns non-gNB self-configuration tasks whose
	// five-minute post-reboot-online observation window elapsed.
	ListActivationChecksDue(ctx context.Context, before time.Time, limit int) ([]ProvisioningTask, error)
	// FailStale marks all non-terminal tasks older than maxAge as failed.
	// Returns the number of tasks affected.
	FailStale(ctx context.Context, maxAge time.Duration) (int64, error)
}
