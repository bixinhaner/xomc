package topology

import (
	"context"

	"github.com/google/uuid"
)

// GroupReader provides read-only access to device groups.
type GroupReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*DeviceGroup, error)
	ListRoots(ctx context.Context) ([]DeviceGroup, error)
	ListChildren(ctx context.Context, parentID uuid.UUID) ([]DeviceGroup, error)
	GetTree(ctx context.Context) ([]DeviceGroup, error)
}

// GroupWriter provides write operations for device groups.
type GroupWriter interface {
	Create(ctx context.Context, group *DeviceGroup) error
	Update(ctx context.Context, group *DeviceGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// GroupMembership manages device-group associations.
type GroupMembership interface {
	AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error
	RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error
	ListDeviceIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
}

// DeviceGroupRepository defines the full persistence interface for device groups.
// It composes smaller interfaces for backward compatibility.
type DeviceGroupRepository interface {
	GroupReader
	GroupWriter
	GroupMembership
}
