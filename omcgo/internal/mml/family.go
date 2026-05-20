package mml

import "strings"

// ============================================================
// family.go — MML 控制台 v2.4 D32+D38 path-prefix-family 推断
//
// 设计依据：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §15.4
//
// 把多个 spec H4 group（72 个）按公共 path prefix 聚合为 ~9 个 family，让前端
// CommandTree 顶层从冗长的 72 个一级节点收敛到 ~40 个；同 family 的 group 在
// UI 上折叠为一个"家族节点"，原 group 退到二级。
//
// **运行时推断**：family 不入库（不改 DB schema），由 InferFamily 在
// group_tree_repository.BuildTree 后处理阶段按 group_code 机械推断。
// 规则有顺序——更具体的规则在前（hardware_upgrade 在 hardware_units / device_info
// 之前；G-02 `Device.DeviceInfo.SwUpgrade.*` 不属于 MU 升级，所以走 device_info）。
// ============================================================

// familyRule 是单条匹配规则。Match 返回 true 表示命中。
type familyRule struct {
	code   string
	nameZh string
	match  func(groupCode string) bool
}

// familyRules 是有序规则表。InferFamily 线性扫描，第一个命中规则胜出。
//
// 顺序的设计意图（与 §15.4 表对应）：
//  1. hardware_upgrade 必须先于 hardware_units / device_info —— RU/EU/Slot/MU
//     的 SwUpgrade 子树在路径上既匹配 `Device.DeviceInfo.MU.` 也含 `SwUpgrade`，
//     必须先被升级 family 截获。
//  2. hardware_units 先于 device_info —— MU 子树是 DeviceInfo 的子集，需要先
//     从 device_info 里切出去。
//  3. capabilities 用 `.Capabilities.` 作为路径分量匹配，避开 substring 误伤
//     （未来若出现 `Foo.CapabilitiesSomething.` 不会被误判）。
//  4. sctp 用全限定前缀，避免与其他 SCTP-like 命名冲突。
//  5. measure_ctrl 兜底 A1-A5 / B1-B2 / Periodic / IRAT 各 MeasureCtrl 变体。
//  6. ethernet_ip / idle_mode 留到最后做前缀 / 子串兜底。
var familyRules = []familyRule{
	{
		// 规则 1：硬件升级（G-69~G-72） — RU/EU/Slot/MU 的 SwUpgrade 子树
		// 路径形如 Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.*
		// 必须先于 hardware_units / device_info 规则匹配。
		code:   "hardware_upgrade",
		nameZh: "硬件升级",
		match: func(g string) bool {
			return strings.Contains(g, "Device.DeviceInfo.MU.") && strings.Contains(g, "SwUpgrade")
		},
	},
	{
		// 规则 2：硬件单元（G-64~G-68） — MU/Slot/EU/RU/RFChannel
		// 路径形如 Device.DeviceInfo.MU.{i}.* （不含 SwUpgrade，已被规则 1 截获）
		code:   "hardware_units",
		nameZh: "硬件单元",
		match: func(g string) bool {
			return strings.HasPrefix(g, "Device.DeviceInfo.MU.")
		},
	},
	{
		// 规则 3：设备信息（G-01 + G-02） — DeviceInfo 根 + DeviceInfo.SwUpgrade（注意不含 MU）
		// 路径形如 Device.DeviceInfo.* 或 Device.DeviceInfo.SwUpgrade.*
		code:   "device_info",
		nameZh: "设备信息",
		match: func(g string) bool {
			return strings.HasPrefix(g, "Device.DeviceInfo.")
		},
	},
	{
		// 规则 4：告警实例（G-05~G-10） — 故障管理统计 + 当前/实时/历史/队列/支持告警
		code:   "alarm_instances",
		nameZh: "告警实例",
		match: func(g string) bool {
			return strings.HasPrefix(g, "Device.FaultMgmt.")
		},
	},
	{
		// 规则 5：能力集（G-18 + G-19） — FAPService Capabilities + CellConfig Capabilities
		// 用 ".Capabilities." 作为路径分量匹配，避免 substring 误伤（如 "FooCapabilitiesBar"）。
		code:   "capabilities",
		nameZh: "能力集",
		match: func(g string) bool {
			return strings.Contains(g, ".Capabilities.")
		},
	},
	{
		// 规则 6：SCTP（G-23 + G-24） — SCTP 配置 + SCTP Assoc
		code:   "sctp",
		nameZh: "SCTP",
		match: func(g string) bool {
			return strings.HasPrefix(g, "Device.Services.FAPControl.Transport.SCTP")
		},
	},
	{
		// 规则 7：测量控制（G-36~G-44） — A1-A5 / B1-B2 / Periodic / ConnMode IRAT 各 MeasureCtrl 变体
		// 路径片段形如 ConnMode.A3MeasureCtrl.{i}.* / ConnMode.PeriodicMeasureCtrl.{i}.* 等
		code:   "measure_ctrl",
		nameZh: "测量控制",
		match: func(g string) bool {
			return strings.Contains(g, "MeasureCtrl")
		},
	},
	{
		// 规则 8：以太网/IP（G-52~G-58） — Interface / IPv4 / IPv6 / VLAN / IP Route
		code:   "ethernet_ip",
		nameZh: "以太网/IP",
		match: func(g string) bool {
			return strings.HasPrefix(g, "Device.Ethernet.")
		},
	},
	{
		// 规则 9：空闲态移动性（G-45~G-49） — IdleMode 根 + IRAT + GERAN/UTRA 频组 + 异频载波
		// 路径片段形如 ...Mobility.IdleMode.* / ...IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.*
		code:   "idle_mode",
		nameZh: "空闲态移动性",
		match: func(g string) bool {
			return strings.Contains(g, "IdleMode")
		},
	},
}

// InferFamily 按 §15.4 9 条规则推断 group 所属 family。
//
// 返回 (familyCode, familyNameZh)：
//   - 命中任意规则：返回该规则的 code + 中文名
//   - 未命中：返回 ("", "") — 表示该 group 自成一家，前端按 group 名渲染顶层节点
//
// 时间复杂度：O(规则数) = O(9)，线性扫描，无 regex。
//
// 详见设计文档：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §15.4
func InferFamily(groupCode string) (familyCode, familyNameZh string) {
	for _, r := range familyRules {
		if r.match(groupCode) {
			return r.code, r.nameZh
		}
	}
	return "", ""
}
