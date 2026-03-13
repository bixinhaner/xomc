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
func (c *CMCCCarrier) Name() string             { return "中国移动" }

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

func (c *CMCCCarrier) KPIDefinitions(tech model.Technology) []*carrier.KPIDefinition {
	return kpiDefinitions(tech)
}

func (c *CMCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	if sev, ok := alarmSeverityMap[carrierAlarmCode]; ok {
		return sev
	}
	return model.AlarmWarning
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
