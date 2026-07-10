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

	// RemoveDevicesFromAllGroups 删除给定设备的**全部**归属记录（不限分组），
	// 等价"移出分组" —— 设备回到 NOT EXISTS device_group_members 的未分组态。
	// 用于「移动/添加到『未分组设备』内置节点(DefaultLevel2GroupID)」的服务层兜底，
	// 避免再产生指向内置节点的归属行（issue #478）。
	RemoveDevicesFromAllGroups(ctx context.Context, deviceIDs []uuid.UUID) (int64, error)

	// AddDeviceWithSource UPSERT 设备到指定 group 并标注来源（T-0027 D5.B）。
	// SQL 层 A4 守护：WHERE source_type != 'manual' 拒绝覆盖手工置位行。
	// 返回 rowsAffected：1 = inserted/updated；0 = manual override 跳过。
	// sourceType 应为 "rule"；sourceRuleID 非 nil 时记录归属规则。
	AddDeviceWithSource(ctx context.Context, groupID, deviceID uuid.UUID, sourceType string, sourceRuleID *uuid.UUID) (int64, error)

	// MoveDeviceAutoMatched 仅当设备当前仍属于 sourceGroupID 时移动到 targetGroupID。
	// DefaultLevel2GroupID 表示未分组设备，此时仅在不存在归属记录时插入。
	// 返回实际影响行数，0 表示设备已离开源组或发生并发竞争。
	MoveDeviceAutoMatched(ctx context.Context, sourceGroupID, targetGroupID, deviceID uuid.UUID) (int64, error)
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
