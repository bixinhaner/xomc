// Package device — T-0162 过渡兼容 helper。
//
// 老 `DeviceStatus` 字段 / 类型在 P2 标记 DEPRECATED 但保留过渡期；本文件提供
// 双向翻译函数，让 Repository 在 SELECT 后能为老调用方派生 Status 字段，
// INSERT/UPDATE 前能从老的 device.Status 派生新列。
//
// P3 完成后端调用方完整迁到 LifecycleState + IsOnline 后，本文件随
// `DeviceStatus` 类型整体删除。
package device

import "github.com/omcgo/omcgo/internal/core/model"

// DeriveLifecycleFromStatus 把老 DeviceStatus 翻译成新二元组 (lifecycle, is_online)。
//
// 翻译表：
//
//	DeviceActive          → (Commissioned, true)   // 已入网 + 在线
//	DeviceOffline         → (Commissioned, false)  // 已入网 + 暂离线
//	DeviceMaintenance     → (Maintenance, false)   // 维护中（is_online 保守归 false；
//	                                               //   实际维护中是否在线由 HeartbeatMonitor 写）
//	DeviceDecommissioned  → (Decommissioned, false)
//	DeviceDiscovered      → (Discovered, false)
//	DeviceRegistered      → (Registered, false)
//	DeviceProvisioning    → (Provisioning, false)
//	""（空字符串）          → (Registered, false)   // 默认安全值
func DeriveLifecycleFromStatus(s model.DeviceStatus) (model.DeviceLifecycle, bool) {
	switch s {
	case model.DeviceActive:
		return model.LifecycleCommissioned, true
	case model.DeviceOffline:
		return model.LifecycleCommissioned, false
	case model.DeviceMaintenance:
		return model.LifecycleMaintenance, false
	case model.DeviceDecommissioned:
		return model.LifecycleDecommissioned, false
	case model.DeviceDiscovered:
		return model.LifecycleDiscovered, false
	case model.DeviceRegistered:
		return model.LifecycleRegistered, false
	case model.DeviceProvisioning:
		return model.LifecycleProvisioning, false
	}
	return model.LifecycleRegistered, false
}

// DeriveStatusFromLifecycle 把新 (lifecycle, is_online) 二元组反推成老 DeviceStatus。
//
// 翻译表：
//
//	Commissioned + online    → Active
//	Commissioned + offline   → Offline
//	Maintenance              → Maintenance
//	Decommissioned           → Decommissioned
//	Discovered/Registered/Provisioning → 同名
//
// 用于 Repository SELECT 后填充 device.Status 兼容老读侧代码（在 P3 阶段尚未
// 迁移到 LifecycleState/IsOnline 的调用方仍可读 device.Status 拿到一致语义）。
func DeriveStatusFromLifecycle(l model.DeviceLifecycle, isOnline bool) model.DeviceStatus {
	switch l {
	case model.LifecycleCommissioned:
		if isOnline {
			return model.DeviceActive
		}
		return model.DeviceOffline
	case model.LifecycleMaintenance:
		return model.DeviceMaintenance
	case model.LifecycleDecommissioned:
		return model.DeviceDecommissioned
	case model.LifecycleDiscovered:
		return model.DeviceDiscovered
	case model.LifecycleRegistered:
		return model.DeviceRegistered
	case model.LifecycleProvisioning:
		return model.DeviceProvisioning
	}
	return ""
}

// normalizeDeviceForPersist 在 Create/Update 写入前确保 device.LifecycleState
// 与 device.IsOnline 字段就位。老调用方只设了 Status 字段时，从 Status 派生
// 新字段；新调用方两边都设了则尊重新字段（Status 被忽略）。
//
// 调用约定：Repository.Create / Repository.Update 在构造 SQL 前调用本函数。
func normalizeDeviceForPersist(device *model.Device) {
	if device.LifecycleState == "" && device.Status != "" {
		l, online := DeriveLifecycleFromStatus(device.Status)
		device.LifecycleState = l
		device.IsOnline = online
	}
	if device.LifecycleState == "" {
		// 兜底：没有任何线索时默认 registered + offline，避免插入空字符串触发
		// 000137 migration 的 chk_devices_lifecycle_state CHECK 约束失败
		device.LifecycleState = model.LifecycleRegistered
	}
	if !device.LocationSourceMode.Valid() {
		device.LocationSourceMode = model.LocationSourceTR069
	}
}

// populateDeviceCompat 在 SELECT scan 后用 LifecycleState + IsOnline 反推 Status
// 字段，给尚未迁移到新字段的读侧代码一个一致值。同时回填 OpState（已激活态，
// 这部分逻辑与 P2 之前一致：从 Status 推导）。
//
// 调用约定：所有 Repository scan 函数（scanDeviceFromRow / scanDeviceRow /
// scanDeviceInfoFromRow / scanDeviceWithInfoRow）在 row.Scan() 成功后调用本函数。
func populateDeviceCompat(d *model.Device) {
	d.Status = DeriveStatusFromLifecycle(d.LifecycleState, d.IsOnline)
	d.OpState = model.DeriveOpState(d.Status)
}
