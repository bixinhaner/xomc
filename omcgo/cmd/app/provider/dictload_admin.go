package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// registerDictLoadAdminRoutes 在 superAdminGroup 下挂 dictloader 主动 reload 端点。
//
// 路由：
//
//	POST /api/v1/admin/dictload/reload?name=<loader_name>
//
// 行为：调用 c.DictLoaderRegistry.ReloadOne，等待完成后返回结构化 Report。
// 失败 / 未知 loader 都返 4xx/5xx，方便运维通过 HTTP 状态码即时辨识。
//
// 主要消费场景：MML 命令树重建后，运维不重启 app 即触发 mml-standard Loader。
// 但接口是通用的 — 4 个既有 dictloader（param-model / indicator /
// alarm-definition / product）也能用这个入口手动重载。
//
// 决策依据：docs/design/mml-rebuild-plan-20260513.md §5.5（Q-V3-7 决议）。
func registerDictLoadAdminRoutes(c *Container, superAdmin *gin.RouterGroup) {
	if c.DictLoaderRegistry == nil {
		// 启动期 DictLoaderRegistry 未注册（极少；通常意味着 DictLoader 配置缺失）。
		// 跳过路由注册；后续访问 404，比挂个 panic 入口更安全。
		c.Logger.Warn("dictload admin reload route skipped: DictLoaderRegistry is nil")
		return
	}

	logger := c.Logger.Named("dictload-admin")

	// reloadDictLoader 热重载单 Loader handler — swag-style godoc 供未来 swag init 拾取。
	//
	// @Summary      热重载单个 dictload Loader
	// @Description  调用 dictloader.Registry.ReloadOne；mml-standard 支持 sha256 增量重载毫秒级 skip
	// @Tags         Admin, MML
	// @Security     BearerAuth
	// @Param        name  query  string  true  "Loader name (mml-standard | param-model | indicator | alarm-definition | product)"
	// @Success      200  {object}  map[string]any  "{loader, rows_affected, files_loaded, files_skipped, elapsed_ms, errors}"
	// @Failure      400  {string}  string  "name 参数缺失"
	// @Failure      401  {string}  string  "未授权"
	// @Failure      403  {string}  string  "需 superAdmin"
	// @Router       /api/v1/admin/dictload/reload [post]
	superAdmin.POST("/admin/dictload/reload", func(ctx *gin.Context) {
		name := ctx.Query("name")
		if name == "" {
			response.Fail(ctx, http.StatusBadRequest, "query param 'name' required")
			return
		}

		// 5 分钟超时上限 — 与启动期 LoadAll 一致；单 loader 远低于此（通常 <30s）。
		reloadCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Minute)
		defer cancel()

		start := time.Now()
		report, err := c.DictLoaderRegistry.ReloadOne(reloadCtx, name)
		elapsed := time.Since(start)

		if err != nil {
			logger.Error("dictload admin reload failed",
				zap.String("loader", name),
				zap.Duration("elapsed", elapsed),
				zap.Error(err))
			commonerrors.AbortWithError(ctx,
				commonerrors.HTTPStatusFromError(err), err)
			return
		}

		logger.Info("dictload admin reload completed",
			zap.String("loader", name),
			zap.Duration("elapsed", elapsed),
			zap.Int("rows", report.RowsAffected),
			zap.Int("files_loaded", report.FilesLoaded),
			zap.Int("files_skipped", report.FilesSkipped))

		// T-0183: param-model loader reload 完成后自动清 Redis L2 + bump
		// cache_version,避免运维手工 redis-cli DEL(参考 parammodel iteration guide §2.2)。
		// 其他 loader(mml-standard / indicator / 等)走自己的缓存策略,这里只兜底 param-model。
		cacheKeysCleared := 0
		var cacheVersion int64
		if name == "param-model" && c.Redis != nil {
			if cacheRes, cErr := parammodel.InvalidateCache(reloadCtx, c.Redis, c.Logger); cErr != nil {
				logger.Warn("param-model cache invalidate failed (DB still consistent)",
					zap.Error(cErr))
			} else {
				cacheKeysCleared = cacheRes.KeysCleared
				cacheVersion = cacheRes.CacheVersionAfter
			}
		}

		response.OK(ctx, gin.H{
			"loader":              name,
			"elapsed_ms":          elapsed.Milliseconds(),
			"rows_affected":       report.RowsAffected,
			"files_loaded":        report.FilesLoaded,
			"files_skipped":       report.FilesSkipped,
			"errors":              report.Errors,
			"cache_keys_cleared":  cacheKeysCleared,
			"cache_version_after": cacheVersion,
		})
	})
}
