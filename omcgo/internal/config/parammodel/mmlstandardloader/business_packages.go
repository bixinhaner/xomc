package mmlstandardloader

// 业务套餐（Business Package）— F-B (T-0119 follow-up)：
//
// Sprint A grouper 按 leaf parent path 切组生成 260 groups × 4 ops = 831 commands，
// "小区管理" 类别下塞 528 条，UI 命令树爆炸。普通运维场景下用户其实只想"看
// 这台基站的全部小区配置"或"读 NR 全部射频参数"，根本不需要 528 个独立的
// LST 命令。
//
// 业务套餐：人工选 ~15 个高频 LST 命令，target_paths 用 TR-069 partial path
// (§A.3.2.7) 一次抓整棵子树。这些命令独立于 standard-model.xml 派生的
// 831 条，写到同一 mml_commands 表，code 用 `LST_PKG_*` 前缀以便区分。
//
// 后续调整：用户提到"参考老的 OMC 再次调整"，这份是 v1，欢迎按真实运维需求
// 增删 — 改 PackageSpecs() 切片即可，Loader 自动 UPSERT。

// PackageSpec 一条业务套餐 LST 命令的完整定义。
//
// 与 GenerateCommands 走的 CommandSpec 同结构，但额外约束：
//   - Code 必须以 "LST_PKG_" 开头（schema 层面无 enforce；约定）
//   - OperationType 固定 LST（套餐目前只覆盖只读场景；MOD/ADD/RMV 风险大需
//     精确知道 path）
//   - TargetPaths 必须是 partial path（以 "." 结尾）或精确 path
type PackageSpec struct {
	Code        string
	NameZh      string
	NameEn      string
	Category    string
	Description string
	TargetPaths []string
}

// PackageSpecs 返回 v1 套餐表（15 条）。
//
// 选择标准：
//   - 实测 mml_params 表里 path 命中数 >0（避免空套餐）
//   - 业务上"日常运维想一次看完"的子树
//   - 覆盖 5 大场景：基站基础 / 网络配置 / 小区配置（LTE+NR）/ 告警 / GSM
//
// 实测命中数（2026-05-14 standard-model.xml v1）：
//
//	DEVICE_BASIC    442  | TR069           24
//	DEVICE_TIME      14  | KEEPALIVED      25
//	DEVICE_NET        4  | FAP_GPS         98
//	DEVICE_ETH       59  | FAP_SERVICE  1067
//	CELL_ALL        780  | CELL_LTE       374
//	CELL_NR         382  | CELL_LTE_RAN   322
//	FAULT_CURRENT   <59  | FAULT_HISTORY  <59
//	GSM_BTS          57
func PackageSpecs() []PackageSpec {
	return []PackageSpec{
		// ===== 基站基础（用户首屏最常看的）=====
		{
			Code:        "LST_PKG_DEVICE_BASIC",
			NameZh:      "查看 设备基础信息（全部）",
			NameEn:      "Query Device Basic Info (full)",
			Category:    CategoryBaseStation,
			Description: "Device.DeviceInfo.* 整棵子树，含型号 / 序列号 / 硬件版本 / 软件版本 / 厂商扩展",
			TargetPaths: []string{"Device.DeviceInfo."},
		},
		{
			Code:        "LST_PKG_DEVICE_TIME",
			NameZh:      "查看 设备时间配置",
			NameEn:      "Query Device Time Config",
			Category:    CategoryBaseStation,
			Description: "Device.Time.* 时区 / NTP server / 时钟同步",
			TargetPaths: []string{"Device.Time."},
		},
		{
			Code:        "LST_PKG_DEVICE_NET",
			NameZh:      "查看 网络 IP 配置",
			NameEn:      "Query Device IP Config",
			Category:    CategoryBaseStation,
			Description: "Device.IP.* 接口 / 路由 / DNS",
			TargetPaths: []string{"Device.IP."},
		},
		{
			Code:        "LST_PKG_DEVICE_ETH",
			NameZh:      "查看 以太网接口",
			NameEn:      "Query Ethernet Interfaces",
			Category:    CategoryBaseStation,
			Description: "Device.Ethernet.* 端口 / VLAN / 链路状态",
			TargetPaths: []string{"Device.Ethernet."},
		},
		{
			Code:        "LST_PKG_TR069",
			NameZh:      "查看 TR-069 接口配置",
			NameEn:      "Query TR-069 Management Server",
			Category:    CategoryBaseStation,
			Description: "Device.ManagementServer.* ACS URL / 心跳 / 鉴权",
			TargetPaths: []string{"Device.ManagementServer."},
		},
		{
			Code:        "LST_PKG_HA",
			NameZh:      "查看 双机热备配置",
			NameEn:      "Query HA / Keepalived",
			Category:    CategoryBaseStation,
			Description: "Device.KeepalivedMgmt.* VIP / 优先级 / 探活",
			TargetPaths: []string{"Device.KeepalivedMgmt."},
		},
		{
			Code:        "LST_PKG_FAP_GPS",
			NameZh:      "查看 FAP GPS 位置",
			NameEn:      "Query FAP GPS",
			Category:    CategoryBaseStation,
			Description: "Device.FAP.GPS.* 经纬度 / 海拔 / 锁星数",
			TargetPaths: []string{"Device.FAP.GPS."},
		},

		// ===== 小区配置（最常用）=====
		{
			Code:     "LST_PKG_CELL_ALL",
			NameZh:   "查看 小区配置（全部）",
			NameEn:   "Query Cell Config (full)",
			Category: CategoryCellMgmt,
			Description: "Device.Services.FAPService.{i}.CellConfig.* 整树（~780 paths），" +
				"含 LTE + NR + 厂商扩展。响应较大，慎用于全网设备",
			TargetPaths: []string{"Device.Services.FAPService.{i}.CellConfig."},
		},
		{
			Code:        "LST_PKG_CELL_LTE",
			NameZh:      "查看 小区配置（LTE）",
			NameEn:      "Query Cell Config (LTE)",
			Category:    CategoryCellMgmt,
			Description: "Device.Services.FAPService.{i}.CellConfig.LTE.* (~374 paths)",
			TargetPaths: []string{"Device.Services.FAPService.{i}.CellConfig.LTE."},
		},
		{
			Code:        "LST_PKG_CELL_LTE_RAN",
			NameZh:      "查看 小区 LTE 无线层",
			NameEn:      "Query Cell LTE RAN",
			Category:    CategoryCellMgmt,
			Description: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.* (~322 paths) RF / 邻区 / 调度",
			TargetPaths: []string{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN."},
		},
		{
			Code:        "LST_PKG_CELL_NR",
			NameZh:      "查看 小区配置（NR/5G）",
			NameEn:      "Query Cell Config (NR/5G)",
			Category:    CategoryCellMgmt,
			Description: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.* (~382 paths)",
			TargetPaths: []string{"Device.Services.FAPService.{i}.CellConfig.{i}.NR."},
		},
		{
			Code:        "LST_PKG_FAP_SERVICE",
			NameZh:      "查看 FAP Service 全树",
			NameEn:      "Query FAP Service (full)",
			Category:    CategoryCellMgmt,
			Description: "Device.Services.FAPService.{i}.* 整棵 (~1067 paths)，含小区 + 传输 + EPC",
			TargetPaths: []string{"Device.Services.FAPService.{i}."},
		},

		// ===== 告警 =====
		{
			Code:        "LST_PKG_FAULT_CURRENT",
			NameZh:      "查看 当前告警",
			NameEn:      "Query Current Alarms",
			Category:    CategoryAlarmQuery,
			Description: "Device.FaultMgmt.CurrentAlarm.* 活动告警列表",
			TargetPaths: []string{"Device.FaultMgmt.CurrentAlarm."},
		},
		{
			Code:        "LST_PKG_FAULT_HISTORY",
			NameZh:      "查看 历史告警",
			NameEn:      "Query Alarm History",
			Category:    CategoryAlarmQuery,
			Description: "Device.FaultMgmt.HistoryEvent.* 历史事件",
			TargetPaths: []string{"Device.FaultMgmt.HistoryEvent."},
		},

		// ===== GSM（移动专有）=====
		{
			Code:        "LST_PKG_GSM_BTS",
			NameZh:      "查看 GSM BTS 配置",
			NameEn:      "Query GSM BTS",
			Category:    CategoryCellMgmt,
			Description: "DeviceGSM.Bts.* 整树 (~57 paths)，GSM 基站配置",
			TargetPaths: []string{"DeviceGSM.Bts."},
		},
	}
}
