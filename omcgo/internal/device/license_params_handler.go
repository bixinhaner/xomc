// license_params_handler.go — DeviceDetail "License 参数" tab 后端 REST。
//
// 2 个端点（PRD §2.2）：
//
//	GET  /api/v1/devices/{id}/license-params          → ListLicenseParams
//	POST /api/v1/devices/{id}/license-params/refresh  → 下发 GPV 拉新值
//
// 路由前缀 /devices/:id 与现有 device handler 共享，刻意在 device handler
// RegisterRoutes 之后单独 RegisterRoutes，避免改动既有 routes 大组。
package device

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// LicenseParamHandler — REST handler 薄壳，业务逻辑全在 LicenseParamService。
type LicenseParamHandler struct {
	service *LicenseParamService
	logger  *zap.Logger
}

// NewLicenseParamHandler 构造 handler。
func NewLicenseParamHandler(service *LicenseParamService, logger *zap.Logger) *LicenseParamHandler {
	return &LicenseParamHandler{
		service: service,
		logger:  logger.Named("license-params-handler"),
	}
}

// RegisterRoutes 注册路由到给定的 rg（通常是 v1 受保护组）。
//
// 注意：路径前缀 `/devices/:id/license-params`，不在 device.Handler.RegisterRoutes
// 里以避免污染主 handler；在 modules.go 单独调用本方法。
func (h *LicenseParamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/devices/:id/license-params")
	g.GET("", h.ListLicenseParams)
	g.POST("/refresh", h.RefreshLicenseParams)
}

// ListLicenseParams — GET /api/v1/devices/:id/license-params
//
// 200 + { items: [...], total: N }；空列表（设备未上报或无 license 模型）
// 也返 200 + items=[]，前端展示空态。
func (h *LicenseParamHandler) ListLicenseParams(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, err := h.service.ListLicenseParams(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

// RefreshLicenseParams — POST /api/v1/devices/:id/license-params/refresh
//
// 触发异步 GPV 任务。立即返回（202 接受语义），不等 CPE 响应。
// 30s 内重复触发会撞 Redis 锁，返 409 + 业务码。
func (h *LicenseParamHandler) RefreshLicenseParams(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	operator := actorUsername(c)
	result, err := h.service.TriggerLicenseRefresh(c.Request.Context(), id, operator)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, result)
}

// actorUsername 从 gin.Context 取当前用户名（admin middleware 设置）；
// 不存在返 "system"。仅用作 sourceID 标签 / Redis 锁 value，不参与鉴权。
func actorUsername(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			return s
		}
	}
	return "system"
}
