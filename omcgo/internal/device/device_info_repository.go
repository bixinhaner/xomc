package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceInfoRepository provides persistence for the device_info table.
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
}
