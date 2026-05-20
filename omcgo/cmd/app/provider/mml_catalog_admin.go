package provider

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/response"
)

// registerMMLCatalogAdminRoutes 在 superAdminGroup 下挂 mml-catalog 元信息端点。
//
// 路由：
//
//	GET  /api/v1/admin/mml-catalog/info
//
// 行为：返回 catalogloader.Loader 当前已加载的所有 spec catalog 元信息：
// specVersion / carrier / tech / sourceDocSha256 / generatedAt / loadedAt /
// 行数统计。供 admin UI 展示 catalog 加载状态、与 spec MD sha256 对账。
//
// 注：热重载入口已通过通用 POST /admin/dictload/reload?name=mml-catalog 提供
// （dictload_admin.go 已注册）。本文件仅加 info 查询端点。
//
// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.7.5
func registerMMLCatalogAdminRoutes(c *Container, superAdmin *gin.RouterGroup) {
	if c.MMLCatalogLoader == nil {
		c.Logger.Warn("mml-catalog admin info route skipped: MMLCatalogLoader is nil")
		return
	}

	// @Summary      查询 mml-catalog 已加载 spec 列表
	// @Description  返回 catalog Loader 当前已加载的 spec catalog 元信息；用于 UI 展示 + sha256 对账
	// @Tags         Admin, MML
	// @Security     BearerAuth
	// @Success      200  {object}  map[string]any  "{loaded: [...]}"
	// @Failure      401  {string}  string  "未授权"
	// @Failure      403  {string}  string  "需 superAdmin"
	// @Router       /api/v1/admin/mml-catalog/info [get]
	superAdmin.GET("/admin/mml-catalog/info", func(ctx *gin.Context) {
		_ = http.StatusOK // referenced for godoc parity
		response.OK(ctx, gin.H{
			"loaded": c.MMLCatalogLoader.Info(),
		})
	})
}
