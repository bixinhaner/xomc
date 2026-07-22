package carrier

import "github.com/omcgo/omcgo/internal/core/model"

// OUIProductClassInfo 描述一个已知的厂商/产品小单元。
// 用于设备首次 Inform 时根据 OUI 和 ProductClass 自动识别所属运营商（见 CarrierRegistry.ResolveByIdentity）。
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

// KPIDefinition 类型已废弃（T-0164-P1）。
//
// 旧实现下，每个 carrier 适配器返回一份硬编码的 KPI 列表，KPIEngine 按 carrier+tech 过滤
// 决定算哪些公式。新模型下 KPI 来源全部走 DB（perf_indicators_* + rela_platform_indicator_formula_*），
// 由 pm/kpi/router 按 device → product → 平台路由解析。
//
// 文件保留 ProvisioningTemplates / OUIProductClassInfo 用于其它运营商差异化能力。
