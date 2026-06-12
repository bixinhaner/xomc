package dashboard

import "github.com/omcgo/omcgo/internal/pm/indicator"

// kpi_alias.go —— Dashboard 首页 KPI 别名解析层（issue #227 + #213 Phase1）。
//
// 背景：前端 Dashboard 首页用一套可读的 symbolic key（如 LTE_PDCP_VOLUME_DL）渲染
// Panel 并取数，但 pm_metrics.metric_path 存的是指标库里的 K 编号（如 K900010015），
// symbolic key 与 K 编号永不相等 → WHERE metric_path = ANY(symbolic) 恒空 → 首页图表无数据。
//
// 解决：在后端 dashboard 侧维护一张 symbolic→K编号 映射（本文件）。GetKPITimeSeries 入口
// 把前端传入的 symbolic key 翻成 K 编号再查 pm_metrics，返回时按原 symbolic key 回填，
// 前端继续用可读 key、无需感知 K 编号。映射集中在后端一处，便于维护与领域复核。
//
// 数据来源：/tmp/omc-issues/_kpi_key_mapping.md（symbolic→K编号 候选对照表，
// high=17 / medium=5 / none=1）。本文件采用 high+medium 项；medium / none 在下方注释标注
// “待党晓萍复核”，none 项（LTE_CELL_AVAILABLE）暂不解析、返回空序列。
//
// 制式前缀：LTE = K9000100xx，NR = KGNB0xxx，GSM = KGSM0xxx（三库 id 命名实况）。

// kpiAlias 是单条 symbolic→K编号 映射条目，附 Panel / 制式归类与显示元数据，
// 供 GetKPITimeSeries（取 KCode）与 GetKPIDefinitions（出完整定义）共用。
type kpiAlias struct {
	// Symbolic 是前端可读 key（kpi-config.ts 里的 KPIConfig.key）。
	Symbolic string
	// KCode 是该 symbolic 对应的指标库编号（pm_metrics.metric_path 落库值）。
	// 为空表示库内无对应 KPI（none 项），解析时跳过、取数返回空序列。
	KCode string
	// Tech 是制式：lte / nr / gsm（与前端 TechnologyType 对齐）。
	Tech string
	// DeviceType 是该制式在 indicator 库的设备类型表（ENB / GNB / GSM），
	// 用于 GetKPIDefinitions 按 KCode 反查 cnName / unit。
	DeviceType indicator.DeviceType
	// Panel 是 Dashboard Panel 归类（traffic / availability / ...，与前端 PanelType 对齐）。
	Panel string
	// NeedsReview 标记 medium 置信度（命名口径不一致，待党晓萍复核）。
	NeedsReview bool
}

// dashboardKPIAliases 是 Dashboard 首页全部 symbolic key 的别名表。
//
// 顺序按制式 + Panel 归并（LTE 6 Panel / NR 2 Panel / GSM 3 Panel），与 kpi-config.ts 对齐。
// confidence=high 直接采用；confidence=medium 置 NeedsReview=true 并在行尾注释；
// confidence=none（LTE_CELL_AVAILABLE）KCode 留空、单独注释硬缺口。
var dashboardKPIAliases = []kpiAlias{
	// ── LTE / Traffic ────────────────────────────────────────────────────────
	{Symbolic: "LTE_PDCP_VOLUME_DL", KCode: "K900010015", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	{Symbolic: "LTE_PDCP_VOLUME_UL", KCode: "K900010016", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	{Symbolic: "LTE_PDCP_RATE_DL", KCode: "K900010040", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},
	{Symbolic: "LTE_PDCP_RATE_UL", KCode: "K900010041", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "traffic"},

	// ── LTE / Availability ───────────────────────────────────────────────────
	// LTE_CELL_AVAILABLE（小区可用率）：confidence=none —— LTE 指标库（BLQ/ALL/ENB_DEFAULT…
	// 全集）无任何“小区可用率 / Cell Availability”KPI（硬缺口）。暂不解析（KCode 留空 →
	// 取数返回空序列），待党晓萍复核：(a) 库内新增该 KPI；(b) 前端隐藏 Availability Panel；
	// (c) 用其它可用性近似指标替代。
	{Symbolic: "LTE_CELL_AVAILABLE", KCode: "", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "availability", NeedsReview: true},

	// ── LTE / Utilization ────────────────────────────────────────────────────
	{Symbolic: "LTE_PRB_UTIL_DL", KCode: "K900010014", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "utilization"},
	{Symbolic: "LTE_PRB_UTIL_UL", KCode: "K900010013", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "utilization"},

	// ── LTE / Accessibility ──────────────────────────────────────────────────
	// WIRELESS_SETUP_SR → K900010006（无线初始连接成功率）：confidence=medium —— 前端语义
	// “无线接通率”，库内是“初始连接成功率”，口径相近但非同名。待党晓萍复核是否接受近似。
	{Symbolic: "WIRELESS_SETUP_SR", KCode: "K900010006", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility", NeedsReview: true},
	{Symbolic: "RRC_CONN_SETUP_SR", KCode: "K900010002", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},
	{Symbolic: "ERAB_SETUP_SR", KCode: "K900010005", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},
	{Symbolic: "CSFB_SR", KCode: "K900010029", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "accessibility"},

	// ── LTE / Retainability ──────────────────────────────────────────────────
	{Symbolic: "ERAB_DROP_RATE", KCode: "K900010027", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "retainability"},

	// ── LTE / Mobility ───────────────────────────────────────────────────────
	// HO_INTRA_ENB_OUT_SR / HO_INTRA_ENB_IN_SR → K900010017 / K900010022（同频切出/切入）：
	// confidence=medium —— 前端“eNB 内切换”映射库内“同频切换”，库无聚合“eNB 内切换”KPI。
	// 待党晓萍复核是取同频代表还是新增聚合 KPI。
	{Symbolic: "HO_INTRA_ENB_OUT_SR", KCode: "K900010017", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility", NeedsReview: true},
	{Symbolic: "HO_INTRA_ENB_IN_SR", KCode: "K900010022", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility", NeedsReview: true},
	{Symbolic: "HO_INTER_ENB_OUT_SR", KCode: "K900010021", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility"},
	{Symbolic: "HO_INTER_ENB_IN_SR", KCode: "K900010026", Tech: "lte", DeviceType: indicator.DeviceTypeENB, Panel: "mobility"},

	// ── NR / Traffic ─────────────────────────────────────────────────────────
	// NR_PDCP_VOLUME_DL / NR_PDCP_VOLUME_UL → KGNB0511 / KGNB0510（PDCP 上/下行业务字节数）：
	// confidence=medium —— 库 enName PdcpUpOct*（累计字节 MByte），前端期望“总数据量(GB)”，
	// 语义可对齐但换算系数与“总量 vs 周期字节数”口径待党晓萍复核。
	{Symbolic: "NR_PDCP_VOLUME_DL", KCode: "KGNB0511", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic", NeedsReview: true},
	{Symbolic: "NR_PDCP_VOLUME_UL", KCode: "KGNB0510", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic", NeedsReview: true},
	{Symbolic: "NR_PDCP_RATE_DL", KCode: "KGNB0517", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic"},
	{Symbolic: "NR_PDCP_RATE_UL", KCode: "KGNB0516", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "traffic"},

	// ── NR / Utilization ─────────────────────────────────────────────────────
	{Symbolic: "NR_PRB_UTIL_DL", KCode: "KGNB0506", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "utilization"},
	{Symbolic: "NR_PRB_UTIL_UL", KCode: "KGNB0505", Tech: "nr", DeviceType: indicator.DeviceTypeGNB, Panel: "utilization"},

	// ── GSM / Accessibility ──────────────────────────────────────────────────
	{Symbolic: "GSM_CALL_SETUP_SR", KCode: "KGSM0102", Tech: "gsm", DeviceType: indicator.DeviceTypeGSM, Panel: "accessibility"},

	// ── GSM / Retainability ──────────────────────────────────────────────────
	{Symbolic: "GSM_CALL_DROP_RATE", KCode: "KGSM0103", Tech: "gsm", DeviceType: indicator.DeviceTypeGSM, Panel: "retainability"},

	// ── GSM / Mobility ───────────────────────────────────────────────────────
	{Symbolic: "GSM_HO_SR", KCode: "KGSM0101", Tech: "gsm", DeviceType: indicator.DeviceTypeGSM, Panel: "mobility"},
}

// symbolicToKCode 是 dashboardKPIAliases 的 symbolic→KCode 反查索引（仅含 KCode 非空项）。
// 包级初始化一次，GetKPITimeSeries 高频调用直接读 map。
var symbolicToKCode = func() map[string]string {
	m := make(map[string]string, len(dashboardKPIAliases))
	for _, a := range dashboardKPIAliases {
		if a.KCode != "" {
			m[a.Symbolic] = a.KCode
		}
	}
	return m
}()

// resolveKPIAliases 把前端传入的 symbolic key 列表翻成 K 编号，并返回 K编号→symbolic 反查表，
// 供查询命中后按原 symbolic key 回填响应。
//
// 行为：
//   - 命中别名表且 KCode 非空 → 收集 KCode，记录 kcode→symbolic。
//   - 未命中别名表（透传值，可能本就是 K 编号）→ 原样作为 KCode 收集，自映射回原 key，
//     兼容“前端直接传 K 编号”与“非 Dashboard 调用方”，不破坏既有等值查询语义。
//   - 命中别名表但 KCode 为空（none 项，如 LTE_CELL_AVAILABLE）→ 跳过查询，
//     该 symbolic key 由调用方回填为空序列（库内无此 KPI 的硬缺口）。
//
// kcodes 已去重（同一 KCode 不重复进 WHERE ANY()）。reverse 一个 KCode 可能映射回多个
// symbolic（理论上别名表内 KCode 唯一，但透传值与别名值可能撞同一 KCode），故值为 slice。
func resolveKPIAliases(symbolic []string) (kcodes []string, reverse map[string][]string) {
	reverse = make(map[string][]string, len(symbolic))
	seen := make(map[string]struct{}, len(symbolic))
	for _, s := range symbolic {
		kcode, mapped := symbolicToKCode[s]
		if !mapped {
			// 未登记的 symbolic：可能调用方直接传了 K 编号或别的 metric_path，原样透传。
			// 但若它是已知 none 项（别名表登记、KCode 空），不透传、跳过（返回空序列）。
			if isKnownNoneAlias(s) {
				continue
			}
			kcode = s
		}
		reverse[kcode] = append(reverse[kcode], s)
		if _, dup := seen[kcode]; !dup {
			seen[kcode] = struct{}{}
			kcodes = append(kcodes, kcode)
		}
	}
	return kcodes, reverse
}

// noneAliases 是别名表里 KCode 为空（库内无对应 KPI）的 symbolic 集合，
// resolveKPIAliases 据此区分“未登记的透传值”与“已知硬缺口”。
var noneAliases = func() map[string]struct{} {
	m := make(map[string]struct{})
	for _, a := range dashboardKPIAliases {
		if a.KCode == "" {
			m[a.Symbolic] = struct{}{}
		}
	}
	return m
}()

func isKnownNoneAlias(symbolic string) bool {
	_, ok := noneAliases[symbolic]
	return ok
}
