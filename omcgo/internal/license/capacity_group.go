// capacity_group.go — 网元类型容量分组（issue #318）。
//
// license 文件（TrueLicense）只携带 eNB/gNB/EPC/CPE/EGW/UPS 容量字段，没有 GSM
// 独立容量。现网语义：GSM 网元（2G BSC/BTS，products.alarm_ne_type='GSM'）不单独
// 控制容量，与 eNB 共用——只授权 eNB 的 license 同样允许 GSM 设备接入，用量按
// eNB+GSM（BTS 的网元类型同为 GSM）合计对 eNB 配额控制。
//
// 分组只影响容量口径（授权判断 + 用量合计 + 降容踢线 + 容量满告警）；特性授权
// （feature_code）与菜单闸门不受影响。
package license

// neTypeCapacityGroups 设备网元类型（大写）→ 共用的容量组 key（大写，与
// DevicesSupport 的 license key 大小写不敏感匹配）。未列出的类型独立占自己的
// 容量。
//
// 注意：device_repository.OfflineExcessByTypeCapacity 的 SQL 内嵌了同一规则的
// CASE 归一（GSM→ENB），修改本表时必须同步修改。
var neTypeCapacityGroups = map[string]string{
	"GSM": "ENB",
}

// capacityGroupKey 把设备网元类型归一到其所属容量组的 key（大写）。
// GSM → ENB；其余类型原样大写返回。
func capacityGroupKey(neType string) string {
	u := upperKey(neType)
	if group, ok := neTypeCapacityGroups[u]; ok {
		return group
	}
	return u
}

// capacityGroupMembers 返回消耗某容量组 key 的全部设备网元类型（含 key 本身，
// 均大写）。容量组的用量 = 各成员用量之和。
func capacityGroupMembers(quotaKey string) []string {
	u := upperKey(quotaKey)
	members := []string{u}
	for neType, group := range neTypeCapacityGroups {
		if group == u && neType != u {
			members = append(members, neType)
		}
	}
	return members
}

// capacityGroupUsage 返回容量组的合计用量。usedByType 是 CountDevicesByType 的
// 原始 per-ne_type 在线计数，组内成员（如 ENB 与 GSM）用量相加。
func capacityGroupUsage(usedByType map[string]int, quotaKey string) int {
	total := 0
	for _, member := range capacityGroupMembers(quotaKey) {
		total += usedByType[member]
	}
	return total
}
