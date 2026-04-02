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

	// Group filters
	GroupID       *uuid.UUID  // filter by specific device group
	VisibleGroups []uuid.UUID // data permission: restrict to these groups (nil = no restriction)

	// Extended filters (device_info / devices additional fields)
	Manufacturer  *string // devices.manufacturer exact match
	ProductClass  *string // devices.product_class exact match
	RFStatus      *string // device_info.rf_status exact match
	CellStatus    *string // device_info.cell_status exact match
	ProjectStatus *string // device_info.project_status exact match
	GPSStatus     *string // device_info.gps_status exact match
	AlarmSeverity *string // device_info.alarm_severity exact match
	LicenseStatus *string // device_info.license_status exact match
	OpState       *string // "1" = active (status='active'), "0" = not active (status!='active')

	model.ListRequest
}

// GeoDeviceFilter specifies criteria for listing devices with geo data.
type GeoDeviceFilter struct {
	GroupIDs []string
	Status   []model.DeviceStatus
	Keyword  string
	Bounds   *GeoBounds
	Page     int
	PageSize int
}

// GeoBounds defines a geographic bounding box.
type GeoBounds struct {
	MinLng float64
	MaxLng float64
	MinLat float64
	MaxLat float64
}

// GeoDevice represents device data for map display.
type GeoDevice struct {
	ID           uuid.UUID          `json:"id"`
	SerialNumber string             `json:"sn"`
	Name         string             `json:"name"`
	Status       model.DeviceStatus `json:"status"`
	Latitude     float64            `json:"latitude"`
	Longitude    float64            `json:"longitude"`
	GroupID      *uuid.UUID         `json:"group_id,omitempty"`
	GroupName    string             `json:"group_name,omitempty"`
	Address      string             `json:"address,omitempty"`
	AlarmCount   int                `json:"alarm_count"`
	Type         string             `json:"type,omitempty"`
}

// GeoStats represents device statistics for map display.
type GeoStats struct {
	Total       int64                        `json:"total"`
	StatusCount map[model.DeviceStatus]int64 `json:"status_count"`
	AlarmCount  int64                        `json:"alarm_count"`
	Center      *GeoCenter                   `json:"center,omitempty"` // 平均经纬度中心点
}

// GeoCenter represents the geographic center point of all devices.
type GeoCenter struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
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
	// ListGeo returns devices with geographic coordinates for map display.
	ListGeo(ctx context.Context, filter GeoDeviceFilter) ([]GeoDevice, int64, error)
	// GetGeoStats returns device statistics for map display.
	GetGeoStats(ctx context.Context, groupIDs []string) (*GeoStats, error)
	// SearchDevices searches devices by keyword for map display.
	SearchDevices(ctx context.Context, keyword string, limit int) ([]GeoDevice, error)
}

// DeviceWriter provides write operations for devices.
type DeviceWriter interface {
	Create(ctx context.Context, device *model.Device) error
	Update(ctx context.Context, device *model.Device) error
	Delete(ctx context.Context, id uuid.UUID) error
	// BatchDelete soft-deletes multiple devices and removes their group memberships
	// and device_info records within a transaction. Returns the number of deleted devices.
	BatchDelete(ctx context.Context, ids []uuid.UUID) (int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error
}

// DeviceRepository defines the full interface for device persistence.
// It composes smaller interfaces for backward compatibility.
type DeviceRepository interface {
	DeviceReader
	DeviceWriter
}
