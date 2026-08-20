package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// SysConfigHandler provides HTTP endpoints for system configuration.
type SysConfigHandler struct {
	service *SysConfigService
}

// NewSysConfigHandler creates a new SysConfigHandler.
func NewSysConfigHandler(service *SysConfigService) *SysConfigHandler {
	return &SysConfigHandler{service: service}
}

// RegisterRoutes registers config routes on the given router group.
func (h *SysConfigHandler) RegisterRoutes(rg *gin.RouterGroup) {
	configs := rg.Group("/sysConfig")
	{
		configs.GET("", h.List)
		configs.GET("/apply-batches/:id", h.GetApplyBatch)
		configs.GET("/:id", h.Get)
		configs.POST("", h.Create)
		configs.POST("/batch", h.BatchUpdate)
		configs.PUT("/:id", h.Update)
		configs.DELETE("/:id", h.Delete)
	}
}

// GetApplyBatch 返回一次配置保存对应的运行态应用状态；目标的期望/实际值不会出现在响应中。
func (h *SysConfigHandler) GetApplyBatch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}
	batch, err := h.service.GetApplyBatch(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, batch, "查询成功")
}

// RegisterPublicRoutes 注册无需鉴权即可访问的子集端点。
// 仅返回代码白名单允许的配置项，供认证前页面消费。
// 参 docs/prd/system/ui-customization.md §6。
func (h *SysConfigHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/admin/public/configs", h.ListPublic)
}

// ListPublic 返回 (可选 category 过滤后) 可公开读取的配置。
// 不携带敏感字段，无需登录态。
func (h *SysConfigHandler) ListPublic(c *gin.Context) {
	category := c.Query("category")
	result, err := h.service.ListPublic(c.Request.Context(), category)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, toSysConfigResponses(result), "查询成功")
}

func (h *SysConfigHandler) Create(c *gin.Context) {
	var req CreateSysConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, toSysConfigResponse(*result), "创建成功")
}

func (h *SysConfigHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !allowSysConfigCategory(c, result.Category) {
		return
	}
	response.OKWithMsg(c, toSysConfigResponse(*result), "查询成功")
}

func (h *SysConfigHandler) List(c *gin.Context) {
	category := c.Query("category")
	if !allowSysConfigCategory(c, category) {
		return
	}
	publicOnly := c.Query("public") == "true"

	result, err := h.service.List(c.Request.Context(), category, publicOnly)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, toSysConfigResponses(result), "查询成功")
}

func (h *SysConfigHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}
	var req UpdateSysConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, toSysConfigResponse(*result), "更新成功")
}

func (h *SysConfigHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "删除成功")
}

// BatchUpdate 按 category 批量 upsert 一组 (key,value) 配置项 — system/config 页面"保存"用。
// 参 docs/prd/system/config.md §5.2。
//
// validator 校验失败的错误已被 SysConfigService.BatchUpsert 包 ErrInvalidInput，
// 走 HTTPStatusFromError 自动映射成 HTTP 400（issue #548 切片 2）。
func (h *SysConfigHandler) BatchUpdate(c *gin.Context) {
	var req BatchUpdateSysConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if !allowSysConfigCategory(c, req.Category) {
		return
	}
	if err := validateGenericBatchWrite(req); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	result, err := h.service.BatchUpsertWithResult(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, gin.H{"updated": result.Updated, "batch": result.Batch}, "保存成功")
}

func allowSysConfigCategory(c *gin.Context, category string) bool {
	if category != "notification.email" {
		return true
	}
	if isSuper, _ := c.Get(CtxKeyIsSuperAdmin); isSuper == true {
		return true
	}
	commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
	return false
}
