package dashboard

import "github.com/omcgo/omcgo/internal/pm/indicator"

// kpi_panel_registry.go —— Dashboard 首页 KPI 面板注册表。
//
// 每条记录定义一个指标编号（K/C 编号）在首页属于哪个制式/Panel，
// 供 GetKPIDefinitions 接口按制式返回完整指标列表。
// 前端直接传 K/C 编号查询，此处无需 symbolic→K 的别名映射层。

// kpiPanelEntry 描述一条指标的 Panel 归属与制式元数据。
type kpiPanelEntry struct {
	// KCode 是指标库编号（pm_metrics.metric_path 落库值），同时作为接口 key 返回给前端。
	KCode string
	// Tech 是制式：lte / nr / gsm。
	Tech string
	// DeviceType 是指标库设备类型（ENB / GNB / GSM），用于反查 cnName / unit。
	DeviceType indicator.DeviceType
	// Panel 是 Dashboard Panel 归类（traffic / availability / ...）。
	Panel string
}

// dashboardKPIAliases 保持变量名兼容 kpi_definitions.go 的遍历逻辑，
// 内容已由 symbolic key 切换为直接用 K/C 编号。
var dashboardKPIAliases = []struct {
	Symbolic   string
	KCode      string
	Tech       string
	DeviceType indicator.DeviceType
	Panel      string
	NeedsReview bool
}{
	// ── LTE / Traffic ────────────────────────────────────────────────────────
	{Symbolic: "K900010015", KCode: "K900010015", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	{Symbolic: "K900010016", KCode: "K900010016", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	{Symbolic: "K900010040", KCode: "K900010040", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	{Symbolic: "K900010041", KCode: "K900010041", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	// ── LTE / Availability ───────────────────────────────────────────────────
	{Symbolic: "K900010076", KCode: "K900010076", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "availability"},
	// ── LTE / Utilization ────────────────────────────────────────────────────
	{Symbolic: "K900010014", KCode: "K900010014", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "utilization"},
	{Symbolic: "K900010013", KCode: "K900010013", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "utilization"},
	// ── LTE / Accessibility ──────────────────────────────────────────────────
	{Symbolic: "K900010006", KCode: "K900010006", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},
	{Symbolic: "K900010002", KCode: "K900010002", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},
	{Symbolic: "K900010005", KCode: "K900010005", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},
	{Symbolic: "K900010029", KCode: "K900010029", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},
	// ── LTE / Retainability ──────────────────────────────────────────────────
	{Symbolic: "K900010027", KCode: "K900010027", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "retainability"},
	// ── LTE / Mobility ───────────────────────────────────────────────────────
	{Symbolic: "K900010017", KCode: "K900010017", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility"},
	{Symbolic: "K900010022", KCode: "K900010022", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility"},
	{Symbolic: "K900010021", KCode: "K900010021", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility"},
	{Symbolic: "K900010026", KCode: "K900010026", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility"},
	// ── NR / Traffic ─────────────────────────────────────────────────────────
	{Symbolic: "KGNB0511", KCode: "KGNB0511", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic"},
	{Symbolic: "KGNB0510", KCode: "KGNB0510", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic"},
	{Symbolic: "KGNB0517", KCode: "KGNB0517", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic"},
	{Symbolic: "KGNB0516", KCode: "KGNB0516", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic"},
	// ── NR / Utilization ─────────────────────────────────────────────────────
	{Symbolic: "KGNB0506", KCode: "KGNB0506", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "utilization"},
	{Symbolic: "KGNB0505", KCode: "KGNB0505", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "utilization"},
	// ── GSM / Accessibility ──────────────────────────────────────────────────
	{Symbolic: "KGSM0102", KCode: "KGSM0102", Tech: "gsm", DeviceType: indicator.DeviceTypeGSM, Panel: "accessibility"},
	// ── GSM / Retainability ──────────────────────────────────────────────────
	{Symbolic: "KGSM0103", KCode: "KGSM0103", Tech: "gsm", DeviceType: indicator.DeviceTypeGSM, Panel: "retainability"},
	// ── GSM / Mobility ───────────────────────────────────────────────────────
	{Symbolic: "KGSM0101", KCode: "KGSM0101", Tech: "gsm", DeviceType: indicator.DeviceTypeGSM, Panel: "mobility"},
}
