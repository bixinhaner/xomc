package device

import "github.com/omcgo/omcgo/internal/core/model"

// DeviceDetailComposite aggregates device data from multiple sources
// for the device detail page. Quick-query columns come from device_info;
// complex/multi-instance data is assembled from device_parameters prefix queries.
type DeviceDetailComposite struct {
	Device  *model.Device `json:"device"`
	Info    *DeviceInfo   `json:"info"`
	MMEPool []MMEEntry    `json:"mme_pool"`
	License *LicenseDetail `json:"license,omitempty"`
	Antenna *AntennaInfo   `json:"antenna,omitempty"`
	Cells   []CellInfo     `json:"cells"`
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
}
