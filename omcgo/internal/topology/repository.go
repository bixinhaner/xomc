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
	GetTreeWithCounts(ctx context.Context) ([]DeviceGroup, error)
	ExistsByParentAndName(ctx context.Context, parentID *uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
	GetStats(ctx context.Context) (*GroupStats, error)
	CountDevicesByGroup(ctx context.Context, groupID uuid.UUID) (int, error)
	ListChildIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error)
}

// GroupWriter provides write operations for device groups.
type GroupWriter interface {
	Create(ctx context.Context, group *DeviceGroup) error
	Update(ctx context.Context, group *DeviceGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	// 规则绑定管理
	UpdateBoundRule(ctx context.Context, groupID, ruleID uuid.UUID) error
	ClearBoundRule(ctx context.Context, groupID uuid.UUID) error
}

// GroupMembership manages device-group associations.
type GroupMembership interface {
	AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error
	RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error
	ListDeviceIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
	BatchRemoveDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
	MoveDevices(ctx context.Context, deviceIDs []uuid.UUID, targetGroupID uuid.UUID) (int64, error)
	MoveGroupDevicesToDefault(ctx context.Context, groupIDs []uuid.UUID) (int64, error)
}

// DeviceGroupRepository defines the full persistence interface for device groups.
// It composes smaller interfaces for backward compatibility.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_device_group_repository_test.go -package=topology . DeviceGroupRepository
type DeviceGroupRepository interface {
	GroupReader
	GroupWriter
	GroupMembership
}
