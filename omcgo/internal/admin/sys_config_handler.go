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
		configs.GET("/:id", h.Get)
		configs.POST("", h.Create)
		configs.POST("/batch", h.BatchUpdate)
		configs.PUT("/:id", h.Update)
		configs.DELETE("/:id", h.Delete)
	}
}

// RegisterPublicRoutes 注册无需鉴权即可访问的子集端点。
// 仅返回 is_public=true 的配置项，供登录页消费品牌化资产
// （产品名、Logo、登录背景图等）。参 docs/prd/system/ui-customization.md §6。
func (h *SysConfigHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/admin/public/configs", h.ListPublic)
}

// ListPublic 返回 (可选 category 过滤后) is_public=true 的全部配置。
// 不携带敏感字段，无需登录态。
func (h *SysConfigHandler) ListPublic(c *gin.Context) {
	category := c.Query("category")
	result, err := h.service.List(c.Request.Context(), category, true)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if result == nil {
		result = []SysConfig{}
	}
	response.OKWithMsg(c, result, "查询成功")
}

func (h *SysConfigHandler) Create(c *gin.Context) {
	var req CreateSysConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, result, "创建成功")
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
	response.OKWithMsg(c, result, "查询成功")
}

func (h *SysConfigHandler) List(c *gin.Context) {
	category := c.Query("category")
	publicOnly := c.Query("public") == "true"

	result, err := h.service.List(c.Request.Context(), category, publicOnly)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if result == nil {
		result = []SysConfig{}
	}
	response.OKWithMsg(c, result, "查询成功")
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, result, "更新成功")
}

func (h *SysConfigHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.NewBusinessError(7, "invalid id", err))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, nil, "删除成功")
}

// BatchUpdate 按 category 批量 upsert 一组 (key,value) 配置项 — system/config 页面"保存"用。
// 参 docs/prd/system/config.md §5.2。
func (h *SysConfigHandler) BatchUpdate(c *gin.Context) {
	var req BatchUpdateSysConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	count, err := h.service.BatchUpsert(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"updated": count}, "保存成功")
}
