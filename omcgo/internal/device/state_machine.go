package device

import (
	"fmt"

	"github.com/omcgo/omcgo/internal/core/model"
)

// validTransitions defines the allowed state transitions for a device.
//
// DEPRECATED (T-0162): 使用旧 DeviceStatus 类型，混淆了"生命周期"与"在线"。
// 已被 validLifecycleTransitions 替代。保留过渡期供 P3 阶段渐进改造调用方。
var validTransitions = map[model.DeviceStatus][]model.DeviceStatus{
	model.DeviceDiscovered:   {model.DeviceRegistered, model.DeviceActive},
	model.DeviceRegistered:   {model.DeviceProvisioning, model.DeviceActive},
	model.DeviceProvisioning: {model.DeviceActive, model.DeviceRegistered},
	model.DeviceActive:       {model.DeviceMaintenance, model.DeviceOffline, model.DeviceDecommissioned},
	model.DeviceMaintenance:  {model.DeviceActive, model.DeviceDecommissioned},
	model.DeviceOffline:      {model.DeviceActive, model.DeviceDecommissioned},
}

// ValidateTransition checks whether a device state transition is allowed.
//
// DEPRECATED (T-0162): 用 ValidateLifecycleTransition 替代。
func ValidateTransition(current, target model.DeviceStatus) error {
	allowed, ok := validTransitions[current]
	if !ok {
		return fmt.Errorf("no transitions defined from state %q", current)
	}
	for _, s := range allowed {
		if s == target {
			return nil
		}
	}
	return fmt.Errorf("invalid transition from %q to %q", current, target)
}

// ============================================================================
// T-0162 新状态机：业务生命周期（不含 offline，offline 归 is_online）
// ============================================================================

// validLifecycleTransitions defines the allowed lifecycle state transitions.
//
// 关键差异 vs 旧 validTransitions：
//   - 不含 `offline`：offline 移到 devices.is_online BOOLEAN 字段
//   - active → commissioned 重命名（同语义）
//   - decommissioned 是终态，无转出（D3 决策）
//   - registered/provisioning 增加直接到 decommissioned 路径（管理面强制退服）
var validLifecycleTransitions = map[model.DeviceLifecycle][]model.DeviceLifecycle{
	model.LifecycleDiscovered:    {model.LifecycleRegistered, model.LifecycleCommissioned},
	model.LifecycleRegistered:    {model.LifecycleProvisioning, model.LifecycleCommissioned, model.LifecycleDecommissioned},
	model.LifecycleProvisioning:  {model.LifecycleCommissioned, model.LifecycleRegistered, model.LifecycleDecommissioned},
	model.LifecycleCommissioned:  {model.LifecycleMaintenance, model.LifecycleDecommissioned},
	model.LifecycleMaintenance:   {model.LifecycleCommissioned, model.LifecycleDecommissioned},
	// LifecycleDecommissioned 不在此 map 中 — 终态无转出，无论 target 是什么都返回错误
}

// ValidateLifecycleTransition checks whether a lifecycle transition is allowed.
//
// 使用场景：service.TransitionLifecycle 调用前；管理面 maintenance / decommission
// 操作前。is_online 不受本函数管辖（is_online 由 HeartbeatMonitor / Inform 自动维护，
// 无状态机约束，true ↔ false 任意切换合法）。
func ValidateLifecycleTransition(current, target model.DeviceLifecycle) error {
	if !current.IsValid() {
		return fmt.Errorf("invalid current lifecycle %q", current)
	}
	if !target.IsValid() {
		return fmt.Errorf("invalid target lifecycle %q", target)
	}
	allowed, ok := validLifecycleTransitions[current]
	if !ok {
		// 落到这里通常是 decommissioned（终态），或映射中漏配
		return fmt.Errorf("no transitions defined from lifecycle %q (terminal state?)", current)
	}
	for _, s := range allowed {
		if s == target {
			return nil
		}
	}
	return fmt.Errorf("invalid lifecycle transition from %q to %q", current, target)
}
