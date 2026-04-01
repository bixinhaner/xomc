package carrier

import "github.com/omcgo/omcgo/internal/core/model"

// OUIProductClassInfo 描述一个已知的厂商/产品小单元。
// 用于设备首次 Inform 时根据 OUI 和 ProductClass 自动识别所属运营商（见 CarrierRegistry.ResolveByOUI）。
type OUIProductClassInfo struct {
	OUI              string
	ProductClass     string
	ManufacturerName string
	Description      string
	Technology       model.Technology
}

// ProvisionTemplate 描述运营商针对某种技术（LTE/NR）的开站模板。
// 包含需要下发的默认参数和必填参数列表，由 provision.Engine 在自动开站容窗期内应用。
type ProvisionTemplate struct {
	Name       string
	Technology model.Technology
	Parameters map[string]interface{} // parameter name -> default value
	Required   []string               // required parameter paths
}

// KPIDefinition 描述一个 KPI 计算公式。
// 由运营商适配器按技术返回，由 PM 模块在 PM 文件入库后计算 KPI 指标时引用。
type KPIDefinition struct {
	Name        string   // unique KPI identifier
	DisplayName string   // human-readable name
	Formula     string   // calculation formula expression
	Unit        string   // "%", "ms", "Mbps", etc.
	Counters    []string // dependent counter names
	Category    string   // "accessibility", "retainability", "mobility", "throughput", "utilization"
}
