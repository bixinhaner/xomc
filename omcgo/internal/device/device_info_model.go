package device

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceInfo stores extended operational and management information for a device.
// The devices table holds core identity fields written by ACS/Inform.
// device_info holds operator-managed and parameter-synced fields.
type DeviceInfo struct {
	DeviceID uuid.UUID `json:"device_id"`

	// 运维标识（手动填写）
	DeviceName    string   `json:"device_name"`
	Address       string   `json:"address"`
	Remark        string   `json:"remark"`
	ProjectStatus string   `json:"project_status"`
	Height        *float64 `json:"height"`

	// 无线参数（从 TR069 参数同步）
	ECI           string   `json:"eci"`
	PCI           string   `json:"pci"`
	CellID        string   `json:"cell_id"`
	FreqPoint     string   `json:"freq_point"`
	Bandwidth     *float64 `json:"bandwidth"`
	TransmitPower *float64 `json:"transmit_power"`
	PLMN          string   `json:"plmn"`

	// 状态
	RFStatus      string `json:"rf_status"`
	CellStatus    string `json:"cell_status"`
	MMEStatus     string `json:"mme_status"`
	SyncStatus    string `json:"sync_status"`
	KPIStatus     string `json:"kpi_status"`
	NumOfCells    int    `json:"num_of_cells"`
	GPSStatus     string `json:"gps_status"`
	AlarmSeverity string `json:"alarm_severity"`
	LicenseStatus string `json:"license_status"`

	// 硬件
	MAC             string `json:"mac"`
	HardwareVersion string `json:"hardware_version"`

	// 时间
	FirstOnlineTime *time.Time `json:"first_online_time"`
	LastOfflineTime *time.Time `json:"last_offline_time"`
	RunTime         int64      `json:"run_time"`

	// 审计
	Creator string `json:"creator"`
	Updater string `json:"updater"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateDeviceInfoRequest defines the manual-only fields that operators can update.
type UpdateDeviceInfoRequest struct {
	DeviceName    *string  `json:"device_name"`
	Address       *string  `json:"address"`
	Remark        *string  `json:"remark"`
	ProjectStatus *string  `json:"project_status"`
	Height        *float64 `json:"height"`
}

// DeviceWithInfo combines core device data with extended info for list views.
// Fields from device_info are flattened into the response so the frontend
// does not need to be aware of the dual-table architecture.
type DeviceWithInfo struct {
	model.Device

	// device_info fields (may be nil if no device_info row exists)
	DeviceName    *string  `json:"device_name"`
	InfoAddress   *string  `json:"device_address"`
	Remark        *string  `json:"remark"`
	ProjectStatus *string  `json:"project_status"`
	Height        *float64 `json:"height"`

	ECI           *string  `json:"eci"`
	PCI           *string  `json:"pci"`
	CellID        *string  `json:"cell_id"`
	FreqPoint     *string  `json:"freq_point"`
	Bandwidth     *float64 `json:"bandwidth"`
	TransmitPower *float64 `json:"transmit_power"`
	PLMN          *string  `json:"plmn"`

	RFStatus      *string `json:"rf_status"`
	CellStatus    *string `json:"cell_status"`
	MMEStatus     *string `json:"mme_status"`
	SyncStatus    *string `json:"sync_status"`
	KPIStatus     *string `json:"kpi_status"`
	NumOfCells    *int    `json:"num_of_cells"`
	GPSStatus     *string `json:"gps_status"`
	AlarmSeverity *string `json:"alarm_severity"`
	LicenseStatus *string `json:"license_status"`

	MAC             *string `json:"mac"`
	HardwareVersion *string `json:"hardware_version"`

	FirstOnlineTime *time.Time `json:"first_online_time"`
	LastOfflineTime *time.Time `json:"last_offline_time"`
	RunTime         *int64     `json:"run_time"`
}

// DeviceInfoSyncUpdate holds fields to update during parameter auto-sync.
type DeviceInfoSyncUpdate struct {
	DeviceID uuid.UUID
	Fields   map[string]interface{}
}
