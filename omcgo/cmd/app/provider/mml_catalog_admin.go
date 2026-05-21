package provider

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/mml/catalogloader"
)

// registerMMLCatalogAdminRoutes 在 superAdminGroup 下挂 mml-catalog 元信息端点。
//
// 路由：
//
//	GET  /api/v1/admin/mml-catalog/info
//	GET  /api/v1/admin/mml-catalog/link-health
//	GET  /api/v1/admin/mml-catalog/link-health/summary
//
// 行为：
//   - info：返回 catalogloader.Loader 当前已加载的所有 spec catalog 元信息：
//     specVersion / carrier / tech / sourceDocSha256 / generatedAt / loadedAt /
//     行数统计。
//   - link-health：列出 mml_catalog_link_health 表中 §R-2.5.2 标准参数树关联失败
//     的明细行（默认只返未解决项，支持 ?spec_version= / ?include_resolved=true /
//     ?limit=N 过滤）。
//   - link-health/summary：按 spec_version 聚合的失败统计（卡片用）。
//
// 注：热重载入口已通过通用 POST /admin/dictload/reload?name=mml-catalog 提供
// （dictload_admin.go 已注册）。本文件仅加查询端点。
//
// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.7.5
// 规范：omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md §R-2.5.2
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

	// @Summary      查询 mml-catalog 标准参数树关联失败清单
	// @Description  返回 §R-2.5.2 mml_catalog_link_health 表中"无法在标准参数树命中"的 path 明细。
	// @Description  默认只返未解决项；?include_resolved=true 包含已解决；?spec_version=xxx 限定版本；?limit=N 限制返回行数（默认 500，最大 5000）。
	// @Tags         Admin, MML
	// @Security     BearerAuth
	// @Param        spec_version       query  string  false  "限定 spec 版本"
	// @Param        include_resolved   query  bool    false  "是否包含已解决项（默认 false）"
	// @Param        limit              query  int     false  "返回行数上限（默认 500，最大 5000）"
	// @Success      200  {object}  map[string]any  "{rows: [...]}"
	// @Failure      500  {string}  string  "查询失败"
	// @Router       /api/v1/admin/mml-catalog/link-health [get]
	superAdmin.GET("/admin/mml-catalog/link-health", func(ctx *gin.Context) {
		if c.PgPool == nil {
			response.Fail(ctx, http.StatusInternalServerError, "DB pool unavailable")
			return
		}
		opt := catalogloader.LinkHealthListOptions{
			SpecVersion:     ctx.Query("spec_version"),
			IncludeResolved: ctx.Query("include_resolved") == "true",
		}
		if v := ctx.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				opt.Limit = n
			}
		}
		rows, err := catalogloader.ListLinkHealth(ctx.Request.Context(), c.PgPool, opt)
		if err != nil {
			c.Logger.Error("list link-health failed: " + err.Error())
			response.Fail(ctx, http.StatusInternalServerError, "list link_health: "+err.Error())
			return
		}
		response.OK(ctx, gin.H{
			"rows":  rows,
			"count": len(rows),
		})
	})

	// @Summary      按 spec 聚合的 link-health 统计
	// @Description  返回每个 spec_version 的未解决 / 已解决数 + 按 failure_reason (A/B/C/D) 拆分的未解决数；admin UI 卡片用。
	// @Tags         Admin, MML
	// @Security     BearerAuth
	// @Success      200  {object}  map[string]any  "{summary: [...]}"
	// @Failure      500  {string}  string  "查询失败"
	// @Router       /api/v1/admin/mml-catalog/link-health/summary [get]
	superAdmin.GET("/admin/mml-catalog/link-health/summary", func(ctx *gin.Context) {
		if c.PgPool == nil {
			response.Fail(ctx, http.StatusInternalServerError, "DB pool unavailable")
			return
		}
		summary, err := catalogloader.SummarizeLinkHealth(ctx.Request.Context(), c.PgPool)
		if err != nil {
			c.Logger.Error("summarize link-health failed: " + err.Error())
			response.Fail(ctx, http.StatusInternalServerError, "summarize link_health: "+err.Error())
			return
		}
		response.OK(ctx, gin.H{
			"summary": summary,
		})
	})
}
