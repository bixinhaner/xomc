package northbound

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// VisibleGroupsResolver 解析用户可见的设备组集合（与 device 包同名接口语义一致）。
// 超管返回 nil 切片（看见全部）；非超管返回其可见 L2 组 ID 列表。
type VisibleGroupsResolver interface {
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error)
}

// DeviceAccessAuthorizer 校验某设备是否在给定可见组集合内（device.DeviceService 已实现）。
type DeviceAccessAuthorizer interface {
	AuthorizeDeviceGroupAccess(ctx context.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) error
}

// DeviceSNResolver 按 SN 解析设备（device.DeviceService 已实现 GetBySerialNumber）。
// 用于把告警导出里的 device_sn 解析成 deviceID 后再做 IDOR 校验。
type DeviceSNResolver interface {
	ResolveDeviceID(ctx context.Context, sn string) (uuid.UUID, bool, error)
}

// Scoper 为北向/OSS 导出端点提供多租户数据隔离。
//
// 北向接口面向上游 OSS，但发起请求的仍是一个 OMC 用户（JWT/API-Key）。
// 数据权限语义与设备域完全对齐（避免另造一套）：
//   - 超管（source='builtIn'，visibleGroups==nil）→ 不限制，可导出全量；
//   - 非超管 → 只能导出其可见设备组内的设备数据。
//
// 由于北向导出多为「按设备聚合」或「全量同步」，本 Scoper 提供两类守卫：
//   - AuthorizeDevice：按 deviceID 直读类端点（ExportConfig）的 IDOR 守卫；
//   - RequireDeviceScope：列表/同步类端点对非超管要求显式 device 维度过滤，
//     否则拒绝（不允许非超管跨租户枚举全量数据）。
//
// permService / deviceAuthz 任一为 nil（dev/test 未注入）→ 退化为放行，与 device
// 域的 nil-safe 语义一致，不降低生产安全（生产装配始终注入）。
type Scoper struct {
	permService VisibleGroupsResolver
	deviceAuthz DeviceAccessAuthorizer
	snResolver  DeviceSNResolver // 可选；按 SN 校验告警导出归属
}

// NewScoper 构造北向数据权限 Scoper。snResolver 可为 nil（仅按 ID 的端点不需要）。
func NewScoper(permService VisibleGroupsResolver, deviceAuthz DeviceAccessAuthorizer, snResolver DeviceSNResolver) *Scoper {
	return &Scoper{permService: permService, deviceAuthz: deviceAuthz, snResolver: snResolver}
}

// principal 从 gin ctx 取认证身份。第二个返回值表示 user_id 是否存在。
func (s *Scoper) principal(c *gin.Context) (uuid.UUID, bool, bool) {
	uidVal, _ := c.Get(admin.CtxKeyUserID)
	uid, ok := uidVal.(uuid.UUID)
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)
	return uid, isSuper, ok
}

func isNorthboundAPIUser(c *gin.Context) bool {
	_, ok := c.Get("northbound_api_user")
	return ok
}

// IsSuperAdmin 返回当前请求是否为超管（用于端点内决定是否放行全量）。
func (s *Scoper) IsSuperAdmin(c *gin.Context) bool {
	if isNorthboundAPIUser(c) {
		return true
	}
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)
	return isSuper
}

// AuthorizeDevice 校验当前请求方是否有权访问指定设备（IDOR 守卫）。
// 返回 true 放行；返回 false 表示已写 403/500 响应并 abort，caller 应直接 return。
//
// 跨租户访问（CTCC 用户 → CMCC 设备）→ AuthorizeDeviceGroupAccess 返 ErrForbidden → 403。
func (s *Scoper) AuthorizeDevice(c *gin.Context, deviceID uuid.UUID) bool {
	if s.permService == nil || s.deviceAuthz == nil {
		return true
	}
	if isNorthboundAPIUser(c) {
		return true
	}
	uid, isSuper, ok := s.principal(c)
	if !ok {
		// 已过鉴权中间件却拿不到 user_id：拒绝，不泄露设备数据。
		response.Fail(c, http.StatusForbidden, "forbidden")
		return false
	}
	visibleGroups, err := s.permService.GetUserVisibleGroupIDs(c.Request.Context(), uid, isSuper)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "permission resolve failed")
		return false
	}
	if authErr := s.deviceAuthz.AuthorizeDeviceGroupAccess(c.Request.Context(), deviceID, visibleGroups); authErr != nil {
		response.Fail(c, commonerrors.HTTPStatusFromError(authErr), "forbidden")
		return false
	}
	return true
}

// AuthorizeDeviceBySN 按设备 SN 做 IDOR 守卫（先解析 SN→ID 再走 AuthorizeDevice）。
// snResolver 未注入时退化为放行。设备不存在 → 404（不泄露存在性差异，统一 not found）。
// 返回 true 放行；false 表示已写响应并 abort。
func (s *Scoper) AuthorizeDeviceBySN(c *gin.Context, sn string) bool {
	if s.permService == nil || s.deviceAuthz == nil || s.snResolver == nil {
		return true
	}
	if s.IsSuperAdmin(c) {
		return true
	}
	deviceID, found, err := s.snResolver.ResolveDeviceID(c.Request.Context(), sn)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "device lookup failed")
		return false
	}
	if !found {
		response.Fail(c, http.StatusNotFound, "device not found")
		return false
	}
	return s.AuthorizeDevice(c, deviceID)
}

// RequireDeviceScope 用于列表/同步/导出类端点：非超管必须把请求限定到一个具体设备
// （hasDeviceScope=true），否则拒绝（403）——杜绝非超管跨租户枚举全量数据。
// 超管放行（可全量）。permService 未注入（dev/test）→ 放行。
//
// 返回 true 放行；false 表示已写响应并 abort。
func (s *Scoper) RequireDeviceScope(c *gin.Context, hasDeviceScope bool) bool {
	if s.permService == nil {
		return true
	}
	if s.IsSuperAdmin(c) {
		return true
	}
	if !hasDeviceScope {
		response.Fail(c, http.StatusForbidden,
			"non-superadmin must scope export/sync to a specific device")
		return false
	}
	return true
}
