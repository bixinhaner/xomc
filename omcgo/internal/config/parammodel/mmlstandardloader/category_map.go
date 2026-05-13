package mmlstandardloader

import "strings"

// Category 既有 7 类 code（来自 sys_dictionaries.mml_command_category）。
// 这些是字符串 '1'..'7'，FE 通过 mml_command_category dict 解析为中文名。
const (
	CategoryCellMgmt      = "1" // 小区管理
	CategoryNeighborMgmt  = "2" // 邻区管理
	CategoryBaseStation   = "3" // 基站管理
	CategoryAlarmQuery    = "4" // 告警查询
	CategoryPerfMgmt      = "5" // 性能采集
	CategoryTransportMgmt = "6" // 传输管理
	CategoryVersionMgmt   = "7" // 版本管理
)

// CategorizePath 把一条 TR-069 standard path 映射到既有 7 类 category。
//
// 规则**自顶向下匹配，前序优先**（docs/design/mml-rebuild-plan-20260513.md §7.2 26 条）。
// 任何不匹配兜底归 3 基站管理。
//
// 入参建议传 group path（如 "Device.DeviceInfo.AntennaInfo"）而非叶子 param path —
// group 视角粒度更稳定。
func CategorizePath(path string) string {
	// 规则 1-4：告警/性能（优先级最高，避免被下面 FaultMgmt 之外的规则误吞）
	if strings.HasPrefix(path, "Device.FaultMgmt.") || path == "Device.FaultMgmt" {
		return CategoryAlarmQuery
	}
	if strings.HasPrefix(path, "Device.FAP.PerfMgmt") {
		return CategoryPerfMgmt
	}
	if strings.Contains(path, "FAPControl.") && strings.Contains(path, "SelfConfig.") && strings.Contains(path, "KPI") {
		return CategoryPerfMgmt
	}
	if strings.HasPrefix(path, "Device.KeepalivedMgmt.Counter") {
		return CategoryPerfMgmt
	}

	// 规则 5-7：邻区（在小区配置规则之前）
	if strings.Contains(path, "CellConfig.") && strings.Contains(path, "NeighborList") {
		return CategoryNeighborMgmt
	}
	if strings.Contains(path, "DeviceGSM.Bts.") && strings.Contains(path, "NeighborList") {
		return CategoryNeighborMgmt
	}
	if strings.Contains(path, "Si2quaterNeighborList") {
		return CategoryNeighborMgmt
	}

	// 规则 15：版本管理（在 DeviceInfo 兜底前抢）
	if isVersionPath(path) {
		return CategoryVersionMgmt
	}

	// 规则 8-10：小区管理
	if strings.Contains(path, "CellConfig.") {
		return CategoryCellMgmt
	}
	if strings.Contains(path, "FAPControl.") {
		return CategoryCellMgmt
	}
	if strings.HasPrefix(path, "Device.Services.GsmBTSCellDT") {
		return CategoryCellMgmt
	}

	// 规则 11-14, 23：传输管理
	if strings.Contains(path, "FAPService") && strings.Contains(path, ".Transport") {
		return CategoryTransportMgmt
	}
	if strings.Contains(path, "FAPService") && strings.Contains(path, ".Ipsec") {
		return CategoryTransportMgmt
	}
	if strings.Contains(path, "FAPService") && strings.Contains(path, "MmePoolConfigParam") {
		return CategoryTransportMgmt
	}
	if strings.HasPrefix(path, "Device.Ethernet") {
		return CategoryTransportMgmt
	}
	if strings.HasPrefix(path, "Device.IP") {
		return CategoryTransportMgmt
	}
	if strings.HasPrefix(path, "Device.ManagementServer") {
		return CategoryTransportMgmt
	}

	// 规则 16-22, 24-26：基站管理（兜底）
	// Device.DeviceInfo.* / Device.FAP.* (除 PerfMgmt) /
	// Device.KeepalivedMgmt.* / Device.Time.* / Device.LAN_HostConfigManagement /
	// Device.LogMgmt / Device.Nr / Device.Services.FAPService.{i}.Capabilities |
	// X_COM | AccessMgmt | EMBEDDED_* / DeviceGSM (剩余) / boardconf / InternetGatewayDevice
	return CategoryBaseStation
}

// isVersionPath 识别"软件版本相关"路径（§7.2 规则 15）。
//   - Device.DeviceInfo.*Software* / *Firmware*
//   - Device.RemoteDeviceList.{i}.Software*
//   - Device.Software*
//   - Device.SoftwareCtrl.*
func isVersionPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "Device.SoftwareCtrl"),
		strings.HasPrefix(path, "Device.Software"):
		return true
	case strings.HasPrefix(path, "Device.DeviceInfo") && (strings.Contains(path, "Software") || strings.Contains(path, "Firmware")):
		return true
	case strings.HasPrefix(path, "Device.RemoteDeviceList") && strings.Contains(path, "Software"):
		return true
	}
	return false
}
