package topology

import (
	"context"

	"github.com/google/uuid"
)

// DeviceGroupRepository defines the persistence interface for device groups.
type DeviceGroupRepository interface {
	Create(ctx context.Context, group *DeviceGroup) error
	GetByID(ctx context.Context, id uuid.UUID) (*DeviceGroup, error)
	Update(ctx context.Context, group *DeviceGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListRoots(ctx context.Context) ([]DeviceGroup, error)
	ListChildren(ctx context.Context, parentID uuid.UUID) ([]DeviceGroup, error)
	GetTree(ctx context.Context) ([]DeviceGroup, error)
	AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error
	RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error
	ListDeviceIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
}
