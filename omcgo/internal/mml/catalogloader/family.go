package catalogloader

import "strings"

// ============================================================
// family.go — MML 控制台 v2.4 P3 path-prefix-family 推断（catalogloader 本地副本）
//
// 设计依据：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §15.4 + §15.8
//
// 为何在 catalogloader 包内重复一份 InferFamily？
//   - 父包 internal/mml 拥有原始实现 internal/mml/family.go
//   - 当前 catalogloader 是 internal/mml 的子包，Go 允许子包 import 父包，但反向不行
//   - 为避免未来若父包 import catalogloader 引发循环依赖，本地保留一份独立副本
//   - 9 条规则必须与 internal/mml/family.go 保持完全一致（同时保持与 migration
//     000141 历史回填 CASE WHEN 一致），任何调整需三处同步：
//       1. internal/mml/family.go familyRules
//       2. internal/mml/catalogloader/family.go familyRules（本文件）
//       3. migrations/0001NN_mml_param_groups_family*.sql CASE WHEN
//
// 规则顺序敏感：更具体的规则在前（hardware_upgrade 在 hardware_units / device_info
// 之前；G-02 Device.DeviceInfo.SwUpgrade.* 不属于 MU 升级所以走 device_info）。
// ============================================================

// familyRule 是单条匹配规则。Match 返回 true 表示命中。
type familyRule struct {
	code   string
	nameZh string
	match  func(groupCode string) bool
}

// familyRules 是有序规则表。InferFamily 线性扫描，第一个命中规则胜出。
// 内容须与 internal/mml/family.go familyRules 同步。
var familyRules = []familyRule{
	{
		// 规则 1：硬件升级（G-69~G-72） — RU/EU/Slot/MU 的 SwUpgrade 子树
		code:   "hardware_upgrade",
		nameZh: "硬件升级",
		match: func(g string) bool {
			return strings.Contains(g, "Device.DeviceInfo.MU.") && strings.Contains(g, "SwUpgrade")
		},
	},
	{
		// 规则 2：硬件单元（G-64~G-68） — MU/Slot/EU/RU/RFChannel
		code:   "hardware_units",
		nameZh: "硬件单元",
		match: func(g string) bool {
			return strings.HasPrefix(g, "Device.DeviceInfo.MU.")
		},
	},
	{
		// 规则 3：设备信息（G-01 + G-02） — DeviceInfo 根 + DeviceInfo.SwUpgrade（注意不含 MU）
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
		code:   "idle_mode",
		nameZh: "空闲态移动性",
		match: func(g string) bool {
			return strings.Contains(g, "IdleMode")
		},
	},
}

// inferFamily 按 9 条规则推断 group 所属 family。
//
// 返回 (familyCode, familyNameZh)：
//   - 命中任意规则：返回该规则的 code + 中文名
//   - 未命中：返回 ("", "") — 表示该 group 自成一家
//
// 时间复杂度：O(规则数) = O(9)，线性扫描，无 regex。
//
// 注：未导出（小写开头），仅供 catalogloader 包内使用；外部走 mml.InferFamily。
func inferFamily(groupCode string) (familyCode, familyNameZh string) {
	for _, r := range familyRules {
		if r.match(groupCode) {
			return r.code, r.nameZh
		}
	}
	return "", ""
}
