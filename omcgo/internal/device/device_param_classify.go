package device

import (
	"strconv"
	"strings"
)

// ExtractFAPInstance 从 TR069 路径中提取 FAPService 实例编号。
// 非 FAPService 路径返回 0。
func ExtractFAPInstance(path string) int {
	const marker = "FAPService."
	idx := strings.Index(path, marker)
	if idx < 0 {
		return 0
	}
	rest := path[idx+len(marker):]
	dotIdx := strings.Index(rest, ".")
	if dotIdx < 0 {
		return 0
	}
	n, err := strconv.Atoi(rest[:dotIdx])
	if err != nil {
		return 0 // FAPService.Ipsec 等非数字路径
	}
	return n
}

// ClassifyParamGroup 根据 TR069 路径返回功能分组标签。
// 匹配规则按优先级从高到低排列。
func ClassifyParamGroup(path string) string {
	switch {
	// 多实例对象组（高优先级，路径含特征关键字）
	case strings.Contains(path, "MmePoolConfigParam.") ||
		strings.Contains(path, "X_COM_MmePool."):
		return "mme_pool"
	case strings.Contains(path, "X_COM_LICENSE."):
		return "license"
	case strings.Contains(path, "AntennaInfo."):
		return "antenna"
	case strings.Contains(path, "FaultMgmt.CurrentAlarm."):
		return "alarm"

	// 同步相关（跨多个父路径）
	case strings.Contains(path, "X_COM_GPS") ||
		strings.Contains(path, "X_COM_BDS") ||
		strings.Contains(path, "X_COM_1588") ||
		strings.Contains(path, "X_COM_GLONASS") ||
		strings.HasPrefix(path, "Device.FAP.GPS.") ||
		strings.Contains(path, "tfcsSync") ||
		strings.Contains(path, "tfcsManager"):
		return "sync"

	// 射频/无线配置
	case strings.Contains(path, "CellConfig") &&
		(strings.Contains(path, "RAN.") || strings.Contains(path, "EPC.PLMN")):
		return "radio"

	// FAPService 控制
	case strings.Contains(path, "FAPControl."):
		return "fap_control"

	// 根级对象
	case strings.HasPrefix(path, "Device.DeviceInfo."):
		return "device_info"
	case strings.HasPrefix(path, "Device.ManagementServer."):
		return "management"
	case strings.Contains(path, "Ipsec") || strings.Contains(path, "IPSEC"):
		return "ipsec"
	case strings.HasPrefix(path, "Device.IP."):
		return "network"
	case strings.HasPrefix(path, "Device.SoftwareCtrl."):
		return "software"

	default:
		return "other"
	}
}
