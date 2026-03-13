package carrier

import "github.com/omcgo/omcgo/internal/core/model"

// Carrier defines the interface for carrier-specific behavior.
// All carrier differences must be implemented through this interface.
// Hard-coding like "if carrier == cmcc" is prohibited.
type Carrier interface {
	// Code returns the carrier code.
	Code() model.CarrierCode

	// Name returns the carrier display name.
	Name() string

	// SupportedTechnologies returns the radio technologies this carrier supports.
	SupportedTechnologies() []model.Technology

	// DefaultDataModelVersions returns default data model version strings for a technology.
	DefaultDataModelVersions(tech model.Technology) []string

	// KnownOUIProductClasses returns known vendor/product combinations for device identification.
	KnownOUIProductClasses(tech model.Technology) []OUIProductClassInfo

	// MapParameterToUnified maps a carrier-specific parameter path to the unified internal name.
	MapParameterToUnified(carrierPath string) string

	// MapUnifiedToParameter maps a unified internal name back to the carrier-specific path.
	MapUnifiedToParameter(unifiedName string) string

	// ProvisioningTemplates returns default provisioning template definitions for a technology.
	ProvisioningTemplates(tech model.Technology) []*ProvisionTemplate

	// KPIDefinitions returns KPI formula definitions for a technology.
	KPIDefinitions(tech model.Technology) []*KPIDefinition

	// AlarmSeverityMapping maps a carrier-specific alarm code to a standard severity level.
	AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity

	// ValidateParameter validates a parameter value against carrier-specific rules.
	ValidateParameter(path string, value string) error
}
