package ctcc

import (
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/common/model"
)

// CTCCCarrier implements the Carrier interface for China Telecom (中国电信).
// This is a skeleton implementation; full adapter will be completed in Phase 3.
type CTCCCarrier struct {
	paramMapping   map[string]string
	reverseMapping map[string]string
}

// New creates a new CTCC carrier adapter.
func New() *CTCCCarrier {
	c := &CTCCCarrier{
		paramMapping:   make(map[string]string),
		reverseMapping: make(map[string]string),
	}
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
	// CTCC uses same vendor hardware; no CTCC-specific OUI mappings yet
	return nil
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
	// Skeleton: will be populated in Phase 3
	return nil
}

func (c *CTCCCarrier) KPIDefinitions(tech model.Technology) []*carrier.KPIDefinition {
	// Skeleton: will be populated in Phase 3
	return nil
}

func (c *CTCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	return model.AlarmWarning
}

func (c *CTCCCarrier) ValidateParameter(path string, value string) error {
	return nil
}
