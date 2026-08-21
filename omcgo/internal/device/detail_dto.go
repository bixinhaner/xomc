package device

import (
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceDetailComposite aggregates device data from multiple sources
// for the device detail page. Quick-query columns come from device_info;
// complex/multi-instance data is assembled from device_parameters prefix queries.
type DeviceDetailComposite struct {
	Device   *model.Device  `json:"device"`
	Info     *DeviceInfo    `json:"info"`
	UPS      *UPSDetail     `json:"ups,omitempty"`
	MMEPool  []MMEEntry     `json:"mme_pool"`
	License  *LicenseDetail `json:"license,omitempty"`
	Antenna  *AntennaInfo   `json:"antenna,omitempty"`
	Cells    []CellInfo     `json:"cells"`
	GSMCells []CellInfo     `json:"gsm_cells,omitempty"`
}

// UPSDetail mirrors the legacy UPS detail blocks while keeping the fields in
// the unified device detail endpoint.
type UPSDetail struct {
	PowerSystem UPSPowerSystemParam      `json:"power_system_param"`
	PowerRun    UPSPowerRunParam         `json:"power_run_param"`
	BatteryRun  []UPSBatteryRuntimeParam `json:"battery_run_param"`
}

type UPSPowerSystemParam struct {
	Manufacturer    *string `json:"manufacturer,omitempty"`
	ManufacturerOUI *string `json:"manufacturer_oui,omitempty"`
	SerialNumber    string  `json:"serial_number"`
	HardwareVersion *string `json:"hardware_version,omitempty"`
	SoftwareVersion *string `json:"software_version,omitempty"`
	UpTimeSeconds   *int64  `json:"up_time_seconds,omitempty"`
	ProductClass    string  `json:"product_class"`
}

type UPSPowerRunParam struct {
	ExternalIP       *string    `json:"external_ip,omitempty"`
	TotalVoltage     *string    `json:"total_voltage,omitempty"`
	TotalTemperature *string    `json:"total_temperature,omitempty"`
	TotalCurrent     *string    `json:"total_current,omitempty"`
	BMSCharging      *string    `json:"bms_charging,omitempty"`
	ACPower          *string    `json:"ac_power,omitempty"`
	ACVoltage        *string    `json:"ac_voltage,omitempty"`
	DCVoltage        *string    `json:"dc_voltage,omitempty"`
	DCCurrent        *string    `json:"dc_current,omitempty"`
	BoardTemperature *string    `json:"board_temperature,omitempty"`
	SFPState         *string    `json:"sfp_state,omitempty"`
	Port0State       *string    `json:"port0_state,omitempty"`
	Port1State       *string    `json:"port1_state,omitempty"`
	Port2State       *string    `json:"port2_state,omitempty"`
	Port3State       *string    `json:"port3_state,omitempty"`
	AverageSOC       *int       `json:"average_soc,omitempty"`
	AverageSOCValues *string    `json:"average_soc_values,omitempty"`
	PackCounts       *int       `json:"pack_counts,omitempty"`
	LastInformAt     *time.Time `json:"last_inform_at,omitempty"`
}

type UPSBatteryRuntimeParam struct {
	PackIndex       int     `json:"pack_index"`
	SerialNumber    *string `json:"serial_number,omitempty"`
	SOC             *int    `json:"soc,omitempty"`
	SOCValues       *string `json:"soc_values,omitempty"`
	Voltage         *string `json:"voltage,omitempty"`
	Temperature     *string `json:"temperature,omitempty"`
	Current         *string `json:"current,omitempty"`
	Status          *string `json:"status,omitempty"`
	RecycleCount    *int    `json:"recycle_count,omitempty"`
	Charging        *string `json:"charging,omitempty"`
	Model           *string `json:"model,omitempty"`
	SoftwareVersion *string `json:"software_version,omitempty"`
}

// MMEEntry represents a single MME connection in the MME pool.
type MMEEntry struct {
	Index  int    `json:"index"`
	IP     string `json:"ip"`
	Status string `json:"status"`
	PLMNID string `json:"plmn_id"`
}

// LicenseDetail holds the device license information.
type LicenseDetail struct {
	Code         string            `json:"code"`
	GenerateDate string            `json:"generate_date"`
	Capacities   []LicenseCapacity `json:"capacities"`
}

// LicenseCapacity represents a single license capacity entry.
//
// 设计文档 §3.3 / §4.3：
//   - State 字段在大多数 CPE（如 Baicells BaiBLQ）上**不上报**，恒为空
//   - Value 字段是 CPE 实际上报的容量计数（例如 Max UE = 100）
//   - 前端可优先展示 Value（容量值）；如需"状态文案"由后端按 RemainingPeriod 派生（>0 active / =0 expired）
type LicenseCapacity struct {
	Index           int    `json:"index"`
	Description     string `json:"description"`
	State           string `json:"state"`
	Value           string `json:"value,omitempty"`
	ValidPeriod     int    `json:"valid_period"`
	RemainingPeriod int    `json:"remaining_period"`
}

// AntennaInfo holds antenna configuration parameters.
type AntennaInfo struct {
	Azimuth    string `json:"azimuth"`
	Beamwidth  string `json:"beamwidth"`
	Downtilt   string `json:"downtilt"`
	Gain       string `json:"gain"`
	Height     string `json:"height"`
	HeightType string `json:"height_type"`
}

// CellInfo represents configuration for a single cell/carrier.
type CellInfo struct {
	Index      int    `json:"index"`
	CellID     string `json:"cell_id"`
	ECI        string `json:"eci"`
	PCI        string `json:"pci"`
	FreqPoint  string `json:"freq_point"`
	Bandwidth  string `json:"bandwidth"`
	Band       string `json:"band"`
	OpState    string `json:"op_state"`
	RFTxStatus string `json:"rf_tx_status"`
	AdminState string `json:"admin_state"`
	LAC        string `json:"lac,omitempty"`
	ARFCN      string `json:"arfcn,omitempty"`
	BTSNum     int    `json:"bts_num,omitempty"`
}
