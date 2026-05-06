package alarm

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// DataPermissionChecker 告警数据权限检查器。
type DataPermissionChecker struct {
	logger *zap.Logger
}

// NewDataPermissionChecker 创建数据权限检查器。
func NewDataPermissionChecker(logger *zap.Logger) *DataPermissionChecker {
	return &DataPermissionChecker{logger: logger}
}

// PermissionContext 权限上下文。
//
// v1.0：移除 Carrier 字段（users.carrier 已删除）；多 carrier 隔离改由 DeviceGroupIDs 通过设备分组层实现。
// 详见 docs/prd/system/users.md §11.11。
type PermissionContext struct {
	UserID         string
	Role           string
	IsSuperAdmin   bool
	DeviceGroupIDs []string
}

// ApplyDataPermission 根据用户权限修改告警查询过滤器。
// v1.0：超管直接放行；非超管按 DeviceGroupIDs 收敛（filter.Carrier 仍是设备视角的 carrier，由调用方决定）。
func (c *DataPermissionChecker) ApplyDataPermission(_ context.Context, permCtx *PermissionContext, _ *AlarmFilter) {
	if permCtx == nil || permCtx.IsSuperAdmin {
		return
	}

	if len(permCtx.DeviceGroupIDs) > 0 {
		c.logger.Debug("data permission: device group filter applied",
			zap.Strings("device_groups", permCtx.DeviceGroupIDs))
	}
}

// CanAcknowledge 检查用户是否有权限确认告警。
func (c *DataPermissionChecker) CanAcknowledge(permCtx *PermissionContext) bool {
	if permCtx == nil {
		return false
	}
	if permCtx.IsSuperAdmin {
		return true
	}
	return permCtx.Role == "admin" || permCtx.Role == "operator"
}

// CanClear 检查用户是否有权限清除告警。
func (c *DataPermissionChecker) CanClear(permCtx *PermissionContext) bool {
	if permCtx == nil {
		return false
	}
	if permCtx.IsSuperAdmin {
		return true
	}
	return permCtx.Role == "admin"
}

// CanManageFilterRules 检查用户是否有权限管理过滤规则。
func (c *DataPermissionChecker) CanManageFilterRules(permCtx *PermissionContext) bool {
	if permCtx == nil {
		return false
	}
	if permCtx.IsSuperAdmin {
		return true
	}
	return permCtx.Role == "admin"
}

// BuildPermissionContext 从 JWT claims 构建权限上下文。
func (c *DataPermissionChecker) BuildPermissionContext(claims map[string]interface{}) *PermissionContext {
	if claims == nil {
		return nil
	}
	ctx := &PermissionContext{}
	if userID, ok := claims["user_id"].(string); ok {
		ctx.UserID = userID
	}
	if role, ok := claims["role"].(string); ok {
		ctx.Role = role
	}
	if isSuper, ok := claims["is_super_admin"].(bool); ok {
		ctx.IsSuperAdmin = isSuper
	}
	// v1.0：移除 claims["carrier"] 解析（users.carrier 已删除，JWT 也不再签发该 claim）
	if groups, ok := claims["device_group_ids"].([]interface{}); ok {
		for _, g := range groups {
			if gid, ok := g.(string); ok {
				ctx.DeviceGroupIDs = append(ctx.DeviceGroupIDs, gid)
			}
		}
	}
	return ctx
}

// ValidatePermission 验证权限上下文是否有效。
func (c *DataPermissionChecker) ValidatePermission(permCtx *PermissionContext) error {
	if permCtx == nil {
		return fmt.Errorf("permission context is nil")
	}
	if permCtx.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	return nil
}
