// system_license_handler.go — F06 System License 重构 P1 Step 2。
//
// 4 个 REST 端点（与 PRD §5 API 契约一致）：
//
//	GET  /api/v1/system-license              GetCurrent
//	POST /api/v1/system-license              Update（上传新文件）
//	GET  /api/v1/system-license/history      ListHistory（分页）
//	GET  /api/v1/system-license/feature-check CheckFeature
//
// 老 /api/v1/licenses/* 路由完全保留（Step 5 才下线），两套并存便于灰度。
//
// feature-check 端点（PRD §5）依赖三级嵌套 feature_list 的查询器，按 PRD §7
// 拆到 Phase 7 RBAC 联动 sprint；本 step 不实现。
package license

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// actorIDFromContext 从 gin.Context 取当前用户 UUID（admin middleware 设置）。
// 找不到 / 类型错误时返 nil（视为 system 操作）。
func actorIDFromContext(c *gin.Context) *uuid.UUID {
	v, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}

// SystemLicenseHandler — gin 层 thin wrapper，仅做绑定 / 转发，业务逻辑在 service。
type SystemLicenseHandler struct {
	service *SystemLicenseService
	logger  *zap.Logger
}

// NewSystemLicenseHandler 构造 handler。
func NewSystemLicenseHandler(svc *SystemLicenseService, logger *zap.Logger) *SystemLicenseHandler {
	return &SystemLicenseHandler{
		service: svc,
		logger:  logger.Named("system-license-handler"),
	}
}

// RegisterRoutes 在给定路由组下挂 3 个端点。
//
// 路径前缀 /system-license（与老 /licenses 完全不冲突）；权限继承调用方
// router 组的中间件（当前走 permGroup("devices") 的端点级 RequireAPIPermission）。
func (h *SystemLicenseHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/system-license")
	g.GET("", h.GetCurrent)
	g.POST("", h.Update)
	g.GET("/history", h.ListHistory)
	g.GET("/feature-check", h.CheckFeature)
}

// GetCurrent — GET /api/v1/system-license。
//
// 200 + SystemLicense（含 devices_support / feature_list / signature_status）
// 404 + 12113 当 system_license 表空。
func (h *SystemLicenseHandler) GetCurrent(c *gin.Context) {
	lic, err := h.service.GetCurrent(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, lic)
}

// Update — POST /api/v1/system-license。
//
// Body：{"raw_content": "<旧项目 .lic 二进制的 Base64>", "raw_content_encoding": "base64"}
//
// 成功：201 + UpdateResult{Current, Replaced}
//   - Current  = 刚生效的 license
//   - Replaced = 被替换的旧 license history 行（首次上传时为 null）
//
// 错误：
//   - .lic 解密 / 字段 / 验签失败 → 400 + 12111
//   - license_id 已存在     → 409 + 12110
func (h *SystemLicenseHandler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.UploadedByUserID = actorIDFromContext(c)

	result, err := h.service.Update(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, result)
}

// ListHistory — GET /api/v1/system-license/history。
//
// Query：page / page_size（model.ListRequest 标准），可选 license_id 过滤。
// 响应：标准 envelope + ListResponse[SystemLicenseHistory]，按 replaced_at DESC。
func (h *SystemLicenseHandler) ListHistory(c *gin.Context) {
	filter := SystemLicenseHistoryFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if v := c.Query("license_id"); v != "" {
		filter.LicenseID = &v
	}

	resp, err := h.service.ListHistory(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list system license history failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, resp)
}

// CheckFeature — GET /api/v1/system-license/feature-check?path=eNB.Monitor.Settings.
func (h *SystemLicenseHandler) CheckFeature(c *gin.Context) {
	path := c.Query("path")
	authorized, err := h.service.CheckFeature(c.Request.Context(), path)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"path": path, "authorized": authorized})
}
