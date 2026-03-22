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
func (c *CTCCCarrier) Name() string             { return "中国电信" }

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

func (c *CTCCCarrier) KPIDefinitions(tech model.Technology) []*carrier.KPIDefinition {
	return kpiDefinitions(tech)
}

func (c *CTCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	if sev, ok := alarmSeverityMap[carrierAlarmCode]; ok {
		return sev
	}
	return model.AlarmWarning
}

func (c *CTCCCarrier) ValidateParameter(path string, value string) error {
	if validator, ok := paramValidators[path]; ok {
		return validator(value)
	}
	return nil
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
