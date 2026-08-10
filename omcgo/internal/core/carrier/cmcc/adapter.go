package cmcc

import (
	"fmt"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

// CMCCCarrier implements the Carrier interface for China Mobile (中国移动).
type CMCCCarrier struct {
	paramMapping   map[string]string // carrier path -> unified name
	reverseMapping map[string]string // unified name -> carrier path
}

// New creates a new CMCC carrier adapter.
func New() *CMCCCarrier {
	c := &CMCCCarrier{}
	c.paramMapping, c.reverseMapping = buildParamMappings()
	return c
}

func (c *CMCCCarrier) Code() model.CarrierCode { return model.CarrierCMCC }
func (c *CMCCCarrier) Name() string            { return "中国移动" }

func (c *CMCCCarrier) SupportedTechnologies() []model.Technology {
	return []model.Technology{model.TechLTE, model.TechNR}
}

func (c *CMCCCarrier) DefaultDataModelVersions(tech model.Technology) []string {
	switch tech {
	case model.TechLTE:
		return []string{"V2.3", "V2.1"}
	case model.TechNR:
		return []string{"V1.9.4", "V1.7"}
	default:
		return nil
	}
}

func (c *CMCCCarrier) KnownOUIProductClasses(tech model.Technology) []carrier.OUIProductClassInfo {
	return knownOUIProducts(tech)
}

func (c *CMCCCarrier) MapParameterToUnified(carrierPath string) string {
	if unified, ok := c.paramMapping[carrierPath]; ok {
		return unified
	}
	return carrierPath
}

func (c *CMCCCarrier) MapUnifiedToParameter(unifiedName string) string {
	if carrierPath, ok := c.reverseMapping[unifiedName]; ok {
		return carrierPath
	}
	return unifiedName
}

func (c *CMCCCarrier) ProvisioningTemplates(tech model.Technology) []*carrier.ProvisionTemplate {
	return provisioningTemplates(tech)
}

func (c *CMCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	if sev, ok := alarmSeverityMap[carrierAlarmCode]; ok {
		return sev
	}
	return 0
}

func (c *CMCCCarrier) ValidateParameter(path string, value string) error {
	// Validate CMCC-specific parameter constraints
	if validator, ok := paramValidators[path]; ok {
		return validator(value)
	}
	return nil
}

// SupportsDirectConnection returns true as CMCC supports NE direct connection (网元直连).
func (c *CMCCCarrier) SupportsDirectConnection() bool { return true }

// GetInfoParamMapping returns TR069 parameter path → device_info column mapping for CMCC.
func (c *CMCCCarrier) GetInfoParamMapping(tech model.Technology) map[string]string {
	if tech == model.TechLTE {
		return map[string]string{
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity":     "eci",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID":            "pci",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL":             "freq_point",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL":         "freq_point",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth":          "bandwidth",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower": "transmit_power",
			"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID":       "plmn",
			"Device.DeviceInfo.X_CMCC_MACAddress":                                     "mac",
			"Device.DeviceInfo.HardwareVersion":                                       "hardware_version",
		}
	}
	// NR mapping
	//
	// 双路径覆盖：未带 cell index 的 TR-181 标准 path（Device.Services.FAPService.1.CellConfig.NR.*）
	// 与 Baicells/Dengyo 等设备实际上报的带 cell index 路径
	// （Device.Services.FAPService.1.CellConfig.{cellIdx}.NR.*）并列。InfoSyncer 跑 map
	// 时遍历参数：哪条 path 在 device_parameters 里存在，哪条就胜出。详见 detail_assembler.go
	// AssembleCells 中对相同两套 NR 路径的兼容。
	return map[string]string{
		// 标准（未索引）路径
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.CellLocalId":  "cell_id",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.NRPCI":            "pci",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.NRARFCN":      "freq_point",
		"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.ChannelBandwidth": "bandwidth",
		"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID":  "plmn",
		// Baicells/Dengyo 带 cell index 路径（实际上报）
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.Common.CellLocalId": "cell_id",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID":       "pci",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNDL":       "freq_point",
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.DLBandwidth":     "bandwidth",
		"Device.Services.FAPService.1.CellConfig.1.NR.Core.PLMNList.1.PLMNID": "plmn",
		// NR 发射功率：与 #362 LTE 设计一致 —— 取 RW、表征「小区实际工作功率」的字段
		// （PowerModify，对应快速设置面板「功率调整」），而非只读硬件能力上限 MaxTxPower。
		"Device.Services.FAPService.1.CellConfig.1.NR.RAN.PowerModify": "transmit_power",
		// Carrier-specific / 通用
		"Device.DeviceInfo.X_CMCC_MACAddress": "mac",
		"Device.DeviceInfo.HardwareVersion":   "hardware_version",
	}
}

// RFControlPath returns the TR069 path for enabling / disabling the device's
// radio frequency output for the given technology. CMCC uses the standard
// TR-181 FAPControl branch keyed by technology. Returns "" when the
// technology is not recognized so the caller can surface a clear error
// rather than send an unkeyed SetParameterValues (T-0029 / R-201).
func (c *CMCCCarrier) RFControlPath(tech model.Technology) string {
	switch tech {
	case model.TechLTE:
		return "Device.Services.FAPService.1.FAPControl.LTE.AdminState"
	case model.TechNR:
		return "Device.Services.FAPService.1.FAPControl.NR.AdminState"
	default:
		return ""
	}
}

func (c *CMCCCarrier) GeofenceControlParametersForInstances(
	productClass string, tech model.Technology, enabled bool, instances []int,
) ([]carrier.GeofenceControlParameter, error) {
	return carrier.BuildGeofenceControlParametersForInstances(productClass, tech, enabled, instances)
}

// SupportsMRType reports whether CMCC collects the given MR type. CMCC supports
// all three measurement-report types (MRO / MRS / MRE) — #17.
func (c *CMCCCarrier) SupportsMRType(mrType model.MRType) bool {
	switch mrType {
	case model.MRTypeMRO, model.MRTypeMRS, model.MRTypeMRE:
		return true
	default:
		return false
	}
}

// alarmSeverityMap maps CMCC alarm codes to standard severity levels.
var alarmSeverityMap = map[string]model.AlarmSeverity{
	"CELL_UNAVAILABLE":   model.AlarmCritical,
	"S1_LINK_FAILURE":    model.AlarmCritical,
	"X2_LINK_FAILURE":    model.AlarmMajor,
	"RADIO_FAILURE":      model.AlarmMajor,
	"GPS_FAILURE":        model.AlarmMajor,
	"CLOCK_SYNC_FAILURE": model.AlarmMajor,
	"TEMP_HIGH":          model.AlarmMinor,
	"TEMP_LOW":           model.AlarmMinor,
	"VSWR_HIGH":          model.AlarmMinor,
	"CPU_OVERLOAD":       model.AlarmMinor,
	"MEM_OVERLOAD":       model.AlarmWarning,
	"CONFIG_MISMATCH":    model.AlarmWarning,
}

// paramValidators maps parameter paths to validation functions.
var paramValidators = map[string]func(string) error{
	"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID": func(v string) error {
		if len(v) < 5 || len(v) > 6 {
			return fmt.Errorf("PLMNID must be 5-6 digits, got %d", len(v))
		}
		for _, ch := range v {
			if ch < '0' || ch > '9' {
				return fmt.Errorf("PLMNID must contain only digits")
			}
		}
		return nil
	},
}
