package provision

import "strings"

// ── TR-069 object 类型参数路径判定与 GPV batch 划分（修复"object 单 path 9005 拖整批"） ──
//
// 背景：
//   Path B 全量同步把所有 storable path 按 batchSize（默认 50）打包成一个 GPV。
//   若批中任一 path 指向"对象表当前 0 实例"（如 BLQ LteCell 表空），CPE 会以 SOAP Fault
//   9005 (Invalid parameter name) 拒绝整个 GPV —— **同批其它 scalar / 有实例的 object
//   全都拿不到回值**。即使 path 实际被支持，仅"无实例"也会触发，且这是 TR-069 协议正常
//   行为而非"不支持"。
//
// 修复策略：
//   - scalar 参数（如 Device.DeviceInfo.SerialNumber）— 永远存在，可继续批量发
//   - object 类型参数（path 末尾 "." 的对象前缀，会让 CPE 枚举实例）— 一个 path = 一个
//     GPV，单独成 batch；失败只损失自己，不污染其它 path
//
// 判定依据（"最可靠的信号"评估）：
//   1. 末尾 "." 是 TR-069 协议级 object path 规范（GPV "Foo.Bar." 表示让 CPE 枚举所有
//      Foo.Bar.N 实例）。**采用此判定**。
//   2. {i} 占位符 — 在 basePrefix() 阶段已被截掉（"Foo.{i}.X" → "Foo."），到达本函数时
//      不存在。所以选 1 而非 2。
//   3. param_mappings.entry_type = 'object' — 是 schema 字段，但要求改函数签名把 mapping
//      传到这里。本函数纯字符串判定，无需多余耦合；末尾 "." 已捕获 100% 的 object 前缀。
//
// 反例不变：
//   - "Device.DeviceInfo.SerialNumber" → scalar（无尾点）
//   - "Device.X_LteCell."               → object（CPE 应枚举实例；可能 0 实例 → 9005）
//   - "Device.WiFi.SSID."               → object
//   - "NoDot"                            → scalar（basePrefix 兜底输出，含义未明，按 scalar 处理）

// isObjectPath 判定一条 GPV 下发路径是否为 object 类型（CPE 会枚举实例的对象前缀）。
//
// 规则：path 末尾为 "." 即视为 object。这与 TR-069 GPV "Foo.Bar." 语义一致 —— 凡向 CPE
// 发送以 "." 结尾的 path，CPE 都会展开该对象的所有当前实例并返回；该对象 0 实例时返回
// SOAP Fault 9005。空串返回 false（无法判定，按 scalar 处理避免影响 batch 划分）。
func isObjectPath(path string) bool {
	if path == "" {
		return false
	}
	return strings.HasSuffix(path, ".")
}

// classifyPrefixes 把扁平的 GPV path 列表按 object / scalar 分两组，输出顺序稳定（与
// 输入相对顺序一致）。供 enqueueGPVPrefixes 据此划分 batch 用：scalars 走原 batchSize
// 批量，objects 一个一个单独发。
func classifyPrefixes(prefixes []string) (scalars, objects []string) {
	for _, p := range prefixes {
		if isObjectPath(p) {
			objects = append(objects, p)
		} else {
			scalars = append(scalars, p)
		}
	}
	return scalars, objects
}

func isExpandedInstanceObjectPath(path string) bool {
	if !isObjectPath(path) {
		return false
	}
	trimmed := strings.TrimSuffix(path, ".")
	lastDot := strings.LastIndex(trimmed, ".")
	if lastDot < 0 || lastDot == len(trimmed)-1 {
		return false
	}
	for _, ch := range trimmed[lastDot+1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
