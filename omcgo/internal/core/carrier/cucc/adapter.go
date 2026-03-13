package cucc

import (
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

// CUCCCarrier implements the Carrier interface for China Unicom (中国联通).
// This is a skeleton implementation; full adapter will be completed in Phase 3.
// CUCC only supports NR (5G), no LTE small cells.
type CUCCCarrier struct {
	paramMapping   map[string]string
	reverseMapping map[string]string
}

// New creates a new CUCC carrier adapter.
func New() *CUCCCarrier {
	c := &CUCCCarrier{
		paramMapping:   make(map[string]string),
		reverseMapping: make(map[string]string),
	}
	return c
}

func (c *CUCCCarrier) Code() model.CarrierCode { return model.CarrierCUCC }
func (c *CUCCCarrier) Name() string             { return "中国联通" }

func (c *CUCCCarrier) SupportedTechnologies() []model.Technology {
	return []model.Technology{model.TechNR} // NR only
}

func (c *CUCCCarrier) DefaultDataModelVersions(tech model.Technology) []string {
	if tech == model.TechNR {
		return []string{"V1.0"}
	}
	return nil
}

func (c *CUCCCarrier) KnownOUIProductClasses(tech model.Technology) []carrier.OUIProductClassInfo {
	return nil
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
	return nil
}

func (c *CUCCCarrier) KPIDefinitions(tech model.Technology) []*carrier.KPIDefinition {
	return nil
}

func (c *CUCCCarrier) AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity {
	return model.AlarmWarning
}

func (c *CUCCCarrier) ValidateParameter(path string, value string) error {
	return nil
}
