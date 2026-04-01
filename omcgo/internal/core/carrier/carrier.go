// Package carrier 定义运营商适配器接口及公共类型。
// 系统中所有与运营商差异相关的行为（参数映射、KPI 公式、开站模板等）
// 均通过此包的 Carrier 接口来隔离，禁止在业务代码中写 "if carrier == cmcc"。
//
// 目前已注册的适配器：cmcc、ctcc、cucc（位于 cmcc/ctcc/cucc 子包）。
package carrier

import "github.com/omcgo/omcgo/internal/core/model"

// Carrier 定义运营商适配器接口。
// 所有运营商差异行为必须通过实现此接口来完成，禁止硬编码 "if carrier == cmcc"。
// 实现类位于 cmcc/ctcc/cucc 子包，通过 CarrierRegistry 统一注册和查找。
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

	// GetInfoParamMapping returns a mapping from TR069 parameter paths to device_info column names.
	// Used by the info sync mechanism to extract key radio parameters from device_parameters
	// and denormalize them into the device_info table for fast list/filter queries.
	GetInfoParamMapping(tech model.Technology) map[string]string
}
