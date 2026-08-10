package cucc

import (
	"fmt"

	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

// CUCCCarrier implements the Carrier interface for China Unicom (中国联通).
// CUCC only supports NR (5G), no LTE small cells.
type CUCCCarrier struct {
	paramMapping   map[string]string
	reverseMapping map[string]string
}

// New creates a new CUCC carrier adapter.
func New() *CUCCCarrier {
	c := &CUCCCarrier{}
	c.paramMapping, c.reverseMapping = buildParamMappings()
	return c
}

func (c *CUCCCarrier) Code() model.CarrierCode { return model.CarrierCUCC }
func (c *CUCCCarrier) Name() string            { return "中国联通" }

func (c *CUCCCarrier) SupportedTechnologies() []model.Technology {
	return []model.Technology{model.TechNR} // NR only
}

func (c *CUCCCarrier) DefaultDataModelVersions(tech model.Technology) []string {
	if tech == model.TechNR {
		return []string{"V1.2", "V1.0"}
	}
	return nil
}

func (c *CUCCCarrier) KnownOUIProductClasses(tech model.Technology) []carrier.OUIProductClassInfo {
	return knownOUIProducts(tech)
}

func (c *CUCCCarrier) MapParameterToUnified(carrierPath string) string {
	if unified, ok := c.paramMapping[carrierPath]; ok {
		return unified
	}
	return carrierPath
}

func (c *CUCCCarrier) MapUnifiedToParameter(unifiedName string) string {
	if carrierPath, ok := c.reverseMapping[unifiedName]; ok {
		return carrierPath
	}
	return unifiedName
}

func (c *CUCCCarrier) ProvisioningTemplates(tech model.Technology) []*carrier.ProvisionTemplate {
	return provisioningTemplates(tech)
}

func (c *CUCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	if sev, ok := alarmSeverityMap[carrierAlarmCode]; ok {
		return sev
	}
	return 0
}

func (c *CUCCCarrier) ValidateParameter(path string, value string) error {
	if validator, ok := paramValidators[path]; ok {
		return validator(value)
	}
	return nil
}

// GetInfoParamMapping returns TR069 parameter path → device_info column mapping for CUCC.
// CUCC only supports NR (5G).
func (c *CUCCCarrier) GetInfoParamMapping(tech model.Technology) map[string]string {
	if tech != model.TechNR {
		return nil
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
// (T-0029 / R-201). CUCC supports NR only — LTE returns "" so the device
// layer surfaces an "unsupported tech" error rather than queuing an
// FAPControl.LTE.AdminState command that the CPE will reject.
func (c *CUCCCarrier) RFControlPath(tech model.Technology) string {
	if tech == model.TechNR {
		return "Device.Services.FAPService.1.FAPControl.NR.AdminState"
	}
	return ""
}

func (c *CUCCCarrier) GeofenceControlParametersForInstances(
	productClass string, tech model.Technology, enabled bool, instances []int,
) ([]carrier.GeofenceControlParameter, error) {
	return carrier.BuildGeofenceControlParametersForInstances(productClass, tech, enabled, instances)
}

// SupportsMRType reports whether CUCC collects the given MR type. CUCC (China
// Unicom) does NOT collect MRE (UE capability) reports, so MRE returns false;
// MRO / MRS are supported. This replaces the "if carrier == cucc" branch
// previously hardcoded in mr/parser.MREParser (#17).
func (c *CUCCCarrier) SupportsMRType(mrType model.MRType) bool {
	switch mrType {
	case model.MRTypeMRO, model.MRTypeMRS:
		return true
	default:
		// MRE and any unknown type are unsupported for CUCC.
		return false
	}
}

// alarmSeverityMap maps CUCC alarm codes to standard severity levels.
var alarmSeverityMap = map[string]model.AlarmSeverity{
	// Critical alarms
	"CELL_UNAVAILABLE":   model.AlarmCritical,
	"NG_LINK_FAILURE":    model.AlarmCritical,
	"SCTP_LINK_FAILURE":  model.AlarmCritical,
	"SITE_POWER_FAILURE": model.AlarmCritical,

	// Major alarms
	"XN_LINK_FAILURE":    model.AlarmMajor,
	"RADIO_FAILURE":      model.AlarmMajor,
	"GPS_FAILURE":        model.AlarmMajor,
	"CLOCK_SYNC_FAILURE": model.AlarmMajor,
	"BACKHAUL_DEGRADED":  model.AlarmMajor,
	"RF_TX_FAILURE":      model.AlarmMajor,

	// Minor alarms
	"TEMP_HIGH":      model.AlarmMinor,
	"TEMP_LOW":       model.AlarmMinor,
	"VSWR_HIGH":      model.AlarmMinor,
	"CPU_OVERLOAD":   model.AlarmMinor,
	"POWER_DEGRADED": model.AlarmMinor,

	// Warning alarms
	"MEM_OVERLOAD":        model.AlarmWarning,
	"CONFIG_MISMATCH":     model.AlarmWarning,
	"SW_VERSION_MISMATCH": model.AlarmWarning,
	"LICENSE_EXPIRING":    model.AlarmWarning,
}

// paramValidators maps parameter paths to validation functions.
var paramValidators = map[string]func(string) error{
	"Device.Services.FAPService.1.CellConfig.NR.Core.PLMNList.1.PLMNID": func(v string) error {
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
	"Device.Services.FAPService.1.CellConfig.NR.Core.SNSSAI.1.SST": func(v string) error {
		if len(v) == 0 {
			return fmt.Errorf("SST must not be empty")
		}
		for _, ch := range v {
			if ch < '0' || ch > '9' {
				return fmt.Errorf("SST must contain only digits")
			}
		}
		return nil
	},
}
