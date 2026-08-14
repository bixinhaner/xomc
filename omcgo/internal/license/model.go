// model.go — F06 System License 重构 Step 5 之后的最小化模型层。
//
// 老 multi-license 类型（License/LicenseStatus/LicenseType/LicenseFilter/
// LicenseSummary）已在 Step 5 完整删除。本文件现仅保留 enforcer/handler 仍要
// 用的 Quota 类型 — 它是 GET /quota 端点的响应体（system_license 模型下
// 由 enforcer 填出），与 system_license_model.go 上的 SystemLicense 是两回事
// （SystemLicense 是 DB 行；Quota 是 enforcement 状态视图）。
package license

// Quota describes the current license enforcement state, returned by
// GET /api/v1/licenses/quota (老路由仍可保留，handler 在 Step 5 已删，路由
// 同时下线；前端不再消费此端点，但保留类型给 Phase 7 RBAC 改造期复用)。
//
// 字段语义（system_license singleton 模型下）：
//   - MaxDevices / UsedDevices / UsageRatio 反映 **所有 device_type 容量
//     之和 / 总设备数**
//   - PerType 列出每个 device_type 子配额；前端展示 / 精细化 enforcement 走它
//   - DaysRemaining = -1 表示 perpetual（NULL expiry_date）
//   - GracePeriodDays 新模型无此概念，恒为 0（字段保留为防止旧调用方 panic）
type Quota struct {
	HasActiveLicense bool                     `json:"has_active_license"`
	MaxDevices       int                      `json:"max_devices"`
	UsedDevices      int                      `json:"used_devices"`
	UsageRatio       float64                  `json:"usage_ratio"`
	DaysRemaining    int                      `json:"days_remaining"` // -1 = perpetual / no expiry
	LicenseType      string                   `json:"license_type"`
	GracePeriodDays  int                      `json:"grace_period_days"`
	PerType          map[string]TypeQuotaItem `json:"per_type,omitempty"`
}

// TypeQuotaItem 是 device_type 维度配额条目。
//
// Max 来自 SystemLicense.DevicesSupport[type]；Used 来自 CountDevicesByType
// 按 alarm_ne_type 分组的设备计数（大小写归一化匹配）。
type TypeQuotaItem struct {
	Max   int     `json:"max"`
	Used  int     `json:"used"`
	Ratio float64 `json:"ratio"`
}
