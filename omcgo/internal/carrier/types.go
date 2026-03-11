package carrier

import "github.com/omcgo/omcgo/internal/model"

// OUIProductClassInfo describes a known vendor/product combination.
type OUIProductClassInfo struct {
	OUI              string
	ProductClass     string
	ManufacturerName string
	Description      string
	Technology       model.Technology
}

// ProvisionTemplate describes a carrier-specific provisioning template.
type ProvisionTemplate struct {
	Name       string
	Technology model.Technology
	Parameters map[string]interface{} // parameter name -> default value
	Required   []string               // required parameter paths
}

// KPIDefinition describes a KPI calculation formula.
type KPIDefinition struct {
	Name        string   // unique KPI identifier
	DisplayName string   // human-readable name
	Formula     string   // calculation formula expression
	Unit        string   // "%", "ms", "Mbps", etc.
	Counters    []string // dependent counter names
	Category    string   // "accessibility", "retainability", "mobility", "throughput", "utilization"
}
