package device

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceFilter specifies criteria for listing devices.
type DeviceFilter struct {
	Carrier    *model.CarrierCode
	Technology *model.Technology
	Status     *model.DeviceStatus
	OUI        *string
	SN         *string // exact match on serial_number
	Search     *string // fuzzy search across serial_number/site_name/manufacturer/device_name/address

	// Extended filters (device_info / devices additional fields)
	Manufacturer  *string // devices.manufacturer exact match
	ProductClass  *string // devices.product_class exact match
	RFStatus      *string // device_info.rf_status exact match
	CellStatus    *string // device_info.cell_status exact match
	ProjectStatus *string // device_info.project_status exact match
	GPSStatus     *string // device_info.gps_status exact match
	AlarmSeverity *string // device_info.alarm_severity exact match
	LicenseStatus *string // device_info.license_status exact match

	model.ListRequest
}

// DeviceReader provides read-only access to devices.
type DeviceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error)
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
	List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
	// ListActiveByLastInform returns active devices ordered by last_inform_at ASC
	// using keyset (cursor) pagination to avoid the sliding-window problem caused
	// by OFFSET-based pagination when rows are mutated during iteration.
	// Pass cursorTime=nil for the first batch. Subsequent calls should pass the
	// last_inform_at of the last device returned, along with its ID as cursorID.
	ListActiveByLastInform(ctx context.Context, cursorTime *time.Time, cursorID *uuid.UUID, limit int) ([]model.Device, error)
}

// DeviceWriter provides write operations for devices.
type DeviceWriter interface {
	Create(ctx context.Context, device *model.Device) error
	Update(ctx context.Context, device *model.Device) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error
}

// DeviceRepository defines the full interface for device persistence.
// It composes smaller interfaces for backward compatibility.
type DeviceRepository interface {
	DeviceReader
	DeviceWriter
}
