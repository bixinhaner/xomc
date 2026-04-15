package alarm

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/model"
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
type PermissionContext struct {
	UserID         string
	Role           string
	IsSuperAdmin   bool
	Carrier        model.CarrierCode
	DeviceGroupIDs []string
}

// ApplyDataPermission 根据用户权限修改告警查询过滤器。
func (c *DataPermissionChecker) ApplyDataPermission(ctx context.Context, permCtx *PermissionContext, filter *AlarmFilter) {
	if permCtx == nil || permCtx.IsSuperAdmin {
		return
	}

	if permCtx.Carrier != "" {
		filter.Carrier = &permCtx.Carrier
		c.logger.Debug("data permission: carrier filter applied",
			zap.String("carrier", string(permCtx.Carrier)))
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
	if carrier, ok := claims["carrier"].(string); ok {
		ctx.Carrier = model.CarrierCode(carrier)
	}
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
