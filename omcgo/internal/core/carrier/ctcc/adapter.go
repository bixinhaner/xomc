package ctcc

import (
	"fmt"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

// CTCCCarrier implements the Carrier interface for China Telecom (中国电信).
type CTCCCarrier struct {
	paramMapping   map[string]string // carrier path -> unified name
	reverseMapping map[string]string // unified name -> carrier path
}

// New creates a new CTCC carrier adapter.
func New() *CTCCCarrier {
	c := &CTCCCarrier{}
	c.paramMapping, c.reverseMapping = buildParamMappings()
	return c
}

func (c *CTCCCarrier) Code() model.CarrierCode { return model.CarrierCTCC }
func (c *CTCCCarrier) Name() string            { return "中国电信" }

func (c *CTCCCarrier) SupportedTechnologies() []model.Technology {
	return []model.Technology{model.TechLTE, model.TechNR}
}

func (c *CTCCCarrier) DefaultDataModelVersions(tech model.Technology) []string {
	switch tech {
	case model.TechLTE:
		return []string{"V1.0.2"}
	case model.TechNR:
		return []string{"V2.8.7"}
	default:
		return nil
	}
}

func (c *CTCCCarrier) KnownOUIProductClasses(tech model.Technology) []carrier.OUIProductClassInfo {
	return knownOUIProducts(tech)
}

func (c *CTCCCarrier) MapParameterToUnified(carrierPath string) string {
	if unified, ok := c.paramMapping[carrierPath]; ok {
		return unified
	}
	return carrierPath
}

func (c *CTCCCarrier) MapUnifiedToParameter(unifiedName string) string {
	if carrierPath, ok := c.reverseMapping[unifiedName]; ok {
		return carrierPath
	}
	return unifiedName
}

func (c *CTCCCarrier) ProvisioningTemplates(tech model.Technology) []*carrier.ProvisionTemplate {
	return provisioningTemplates(tech)
}

func (c *CTCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	if sev, ok := alarmSeverityMap[carrierAlarmCode]; ok {
		return sev
	}
	return 0
}

func (c *CTCCCarrier) ValidateParameter(path string, value string) error {
	if validator, ok := paramValidators[path]; ok {
		return validator(value)
	}
	return nil
}

// GetInfoParamMapping returns TR069 parameter path → device_info column mapping for CTCC.
func (c *CTCCCarrier) GetInfoParamMapping(tech model.Technology) map[string]string {
	if tech == model.TechLTE {
		return map[string]string{
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity":     "eci",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID":            "pci",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL":             "freq_point",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.EARFCNDL":         "freq_point",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth":          "bandwidth",
			"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower": "transmit_power",
			"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID":       "plmn",
			"Device.DeviceInfo.HardwareVersion":                                       "hardware_version",
		}
	}
	// 双路径覆盖：未带 cell index 的 TR-181 标准 path 与 Baicells/Dengyo 等设备实际上报
	// 带 cell index 的路径并列。详见 cmcc/adapter.go GetInfoParamMapping 中的同款注释。
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
		"Device.DeviceInfo.HardwareVersion":                            "hardware_version",
	}
}

// RFControlPath returns the TR069 path for radio control by technology
// (T-0029 / R-201). CTCC follows the same TR-181 FAPControl pattern as
// CMCC; if vendor-specific extensions appear (X_CTCC_*) this method is
// the seam to localize them without leaking into device-layer code.
func (c *CTCCCarrier) RFControlPath(tech model.Technology) string {
	switch tech {
	case model.TechLTE:
		return "Device.Services.FAPService.1.FAPControl.LTE.AdminState"
	case model.TechNR:
		return "Device.Services.FAPService.1.FAPControl.NR.AdminState"
	default:
		return ""
	}
}

func (c *CTCCCarrier) GeofenceControlParametersForInstances(
	productClass string, tech model.Technology, enabled bool, instances []int,
) ([]carrier.GeofenceControlParameter, error) {
	return carrier.BuildGeofenceControlParametersForInstances(productClass, tech, enabled, instances)
}

// SupportsMRType reports whether CTCC collects the given MR type. CTCC supports
// all three measurement-report types (MRO / MRS / MRE) — #17.
func (c *CTCCCarrier) SupportsMRType(mrType model.MRType) bool {
	switch mrType {
	case model.MRTypeMRO, model.MRTypeMRS, model.MRTypeMRE:
		return true
	default:
		return false
	}
}

// alarmSeverityMap maps CTCC alarm codes to standard severity levels.
// CTCC uses its own alarm code system but maps to the same standard severities.
var alarmSeverityMap = map[string]model.AlarmSeverity{
	"CELL_UNAVAILABLE":   model.AlarmCritical,
	"S1_LINK_FAILURE":    model.AlarmCritical,
	"SCTP_LINK_FAILURE":  model.AlarmCritical,
	"X2_LINK_FAILURE":    model.AlarmMajor,
	"RADIO_FAILURE":      model.AlarmMajor,
	"GPS_FAILURE":        model.AlarmMajor,
	"CLOCK_SYNC_FAILURE": model.AlarmMajor,
	"BACKHAUL_FAILURE":   model.AlarmMajor,
	"TEMP_HIGH":          model.AlarmMinor,
	"TEMP_LOW":           model.AlarmMinor,
	"VSWR_HIGH":          model.AlarmMinor,
	"CPU_OVERLOAD":       model.AlarmMinor,
	"POWER_DEGRADED":     model.AlarmMinor,
	"MEM_OVERLOAD":       model.AlarmWarning,
	"CONFIG_MISMATCH":    model.AlarmWarning,
	"LICENSE_EXPIRING":   model.AlarmWarning,
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
