package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceInfoRepository provides persistence for the device_info table.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_device_info_repository_test.go -package=device . DeviceInfoRepository
type DeviceInfoRepository interface {
	// GetByDeviceID returns the extended info for a device.
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error)

	// Create inserts a new device_info row.
	Create(ctx context.Context, info *DeviceInfo) error

	// UpdateManualFields updates only the operator-editable fields.
	UpdateManualFields(ctx context.Context, deviceID uuid.UUID, req UpdateDeviceInfoRequest, updater string) error

	// UpdateSyncFields updates fields synced from TR069 parameters or events.
	UpdateSyncFields(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) error

	// ListDevicesWithInfo returns a paginated list of devices joined with device_info.
	ListDevicesWithInfo(ctx context.Context, filter DeviceFilter) (*model.ListResponse[DeviceWithInfo], error)

	// GetByIDWithInfo returns a single device joined with device_info (and group).
	// 与 ListDevicesWithInfo 同样的 LEFT JOIN，供设备详情页使用——使详情与列表
	// 的 op_state（激活状态）及 device_info 扩展字段口径完全一致。
	// 设备不存在或已软删返回 (nil, nil)。
	GetByIDWithInfo(ctx context.Context, deviceID uuid.UUID) (*DeviceWithInfo, error)
}
