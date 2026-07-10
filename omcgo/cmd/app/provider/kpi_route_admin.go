package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	coremiddleware "github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/core/response"
	kpirouter "github.com/omcgo/omcgo/internal/pm/kpi/router"
)

type kpiRouteInvalidator interface {
	Invalidate(context.Context, kpirouter.InvalidationTrigger) (kpirouter.InvalidationResult, error)
}

// registerKPIRouteAdminRoutes 注册仅供 super_admin 使用的 KPI 路由人工恢复入口。
func registerKPIRouteAdminRoutes(invalidator kpiRouteInvalidator, logger *zap.Logger, superAdmin *gin.RouterGroup) {
	if invalidator == nil {
		logger.Warn("KPI route admin refresh route skipped: invalidator is nil")
		return
	}

	// @Summary      人工刷新 KPI 路由缓存
	// @Description  清理当前 app 的 KPI 路由 L1，并最多三次推进 Redis 全局版本；仅 super_admin 可调用
	// @Tags         Admin, PM
	// @Security     BearerAuth
	// @Success      200  {object}  map[string]any  "{cache_version, attempts, scope, multi_process_sync}"
	// @Failure      401  {string}  string  "未授权"
	// @Failure      403  {string}  string  "需 super_admin"
	// @Failure      503  {string}  string  "Redis 全局版本推进最终失败"
	// @Router       /api/v1/admin/kpi-routes/refresh [post]
	superAdmin.POST("/admin/kpi-routes/refresh", func(c *gin.Context) {
		requestLogger := logger.With(zap.String("request_id", coremiddleware.GetRequestID(c)))
		result, err := invalidator.Invalidate(c.Request.Context(), kpirouter.InvalidationTriggerManual)
		if err != nil {
			requestLogger.Error("manual KPI route refresh failed", zap.Error(err))
			response.Fail(c, http.StatusServiceUnavailable, fmt.Sprintf(
				"KPI route refresh failed after %d attempts; local L1 was cleared but the global version was not updated",
				result.Attempts))
			return
		}
		requestLogger.Info("manual KPI route refresh completed",
			zap.Int64("cache_version", result.CacheVersion),
			zap.Int("attempts", result.Attempts),
			zap.String("scope", string(result.Scope)),
			zap.Bool("multi_process_sync", result.MultiProcessSync))
		response.OK(c, result)
	})
}
