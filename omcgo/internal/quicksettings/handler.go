package quicksettings

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler 暴露「快速设置」分组元数据 REST 端点。
type Handler struct {
	registry *Registry
}

// NewHandler 构造 Handler；registry 由 Loader 提供同一实例。
func NewHandler(registry *Registry) *Handler {
	return &Handler{registry: registry}
}

// RegisterRoutes 注册路由到给定的路由组（已含 /api/v1 前缀）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/quicksettings/groups", h.GetGroups)
}

// GetGroups handles GET /api/v1/quicksettings/groups?tech={lte|nr}.
//
// 响应：{"groups": [Group, ...]}；未知 tech 返回 400。
func (h *Handler) GetGroups(c *gin.Context) {
	tech := c.Query("tech")
	switch TechCode(tech) {
	case TechLTE, TechNR:
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid tech, expected one of: lte, nr",
			"tech":  tech,
		})
		return
	}
	groups := h.registry.GetByTech(TechCode(tech))
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}
