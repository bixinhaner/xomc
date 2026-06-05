package export

import (
	"fmt"
	"strings"
)

// adhocFirstColHeader 返回导出横表首列的列名（按 adhoc 任务聚合维度）。
// 与网页结果表首列口径一致：设备组 / 产品 / 频段 / 全网 / 聚合组；其余（device/空/未知）退化"设备"。
func adhocFirstColHeader(dim string) string {
	switch dim {
	case "device_group":
		return "设备组"
	case "product":
		return "产品"
	case "band":
		return "频段"
	case "network":
		return "全网"
	case "aggregate_group":
		return "聚合组"
	default: // device / 空 / 未知
		return "设备SN"
	}
}

// adhocIncludesCell 仅 device 维度的导出含"小区/PLMN"列；其余聚合维度小区已被聚掉
// （aggregate_group 依赖任务 A 改为单条聚合后亦无小区），导出表直接不含该列。
func adhocIncludesCell(dim string) bool { return dim == "device" }

// adhocObjectLabel 按维度把分组键解析成可读对象名（与网页 buildResultsQuery + 前端 seriesLabel 同口径）。
// 名缺失（脏数据 / 对象已删）回退 ID 前 8 位；前缀（DeviceGroup= / Band=）按维度剥离。
func adhocObjectLabel(dim, oui, sn, productID, productName, objectLDN, groupName string, deviceCount int) string {
	switch dim {
	case "device_group":
		if groupName != "" {
			return groupName
		}
		return first8(strings.TrimPrefix(objectLDN, "DeviceGroup="))
	case "product":
		if productName != "" {
			return productName
		}
		return first8(productID)
	case "band":
		return strings.TrimPrefix(objectLDN, "Band=")
	case "network":
		return "全网"
	case "aggregate_group":
		return fmt.Sprintf("聚合组(%d个设备)", deviceCount)
	default: // device：只用 SN，与页面表格「设备 SN」列一致
		return deviceSNLabel(oui, sn)
	}
}

// first8 取字符串前 8 位（命名回退用），不足 8 位原样返回。
func first8(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
