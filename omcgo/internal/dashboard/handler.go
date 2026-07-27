package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// Handler provides REST API endpoints for dashboard.
type Handler struct {
	service  *Service
	resolver *authz.Resolver
	// permChecker 用于存全局布局接口在 handler 层再校验一次管理员身份（issue #213 S1）。
	// 复用 admin 的 PermissionChecker（Casbin 端点级权限点）+ 超管 IsSuperAdmin 旁路，
	// 不发明新机制。可能为 nil（测试 / 旧 wiring）：此时存盘只认 super_admin。
	permChecker admin.PermissionChecker
}

// NewHandler creates a new dashboard handler.
//
// permChecker 用于存全局 KPI 布局接口的管理员二次校验（可为 nil，仅认 super_admin）。
func NewHandler(service *Service, permChecker admin.PermissionChecker) *Handler {
	return &Handler{service: service, permChecker: permChecker}
}

// SetPermissionService injects the device-group visibility resolver used by
// active alarm inventory charts.
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

// RegisterRoutes registers dashboard routes on the given router group.
//
// Authentication & Authorization:
// - The router group 'rg' should have authentication middleware applied (e.g., JWT verification)
// - All endpoints below are automatically protected once registered with this group
// - For additional permission checks, implement them in individual handlers
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dashboard := rg.Group("/dashboard")
	{
		dashboard.GET("/summary", h.GetSummary)
		dashboard.GET("/alarm-trend", h.GetAlarmTrend)
		dashboard.GET("/device-status", h.GetDeviceStatus)
		dashboard.GET("/device-status-by-type", h.GetDeviceStatusByType)
		dashboard.GET("/kpi-trend", h.GetKPITrend)
		dashboard.GET("/region-stats", h.GetRegionStats)
		dashboard.GET("/widgets", h.GetWidgets)
		dashboard.PUT("/widgets", h.SaveWidgets)
		dashboard.GET("/alarm-type-pie", h.GetAlarmTypePie)
		dashboard.GET("/kpi-time-series", h.GetKPITimeSeries)
		// issue #213 Phase1：KPI 定义动态化（symbolic key + K 编号 + 中文名 + 单位，按制式/Panel 分组）
		dashboard.GET("/kpi/definitions", h.GetKPIDefinitions)
		// issue #213 S1：全局首页 KPI 布局（全局单套，按制式带参）。
		// GET 所有登录用户可读；PUT 仅管理员（handler 层再校验，super_admin 旁路 + Casbin 权限点）。
		dashboard.GET("/kpi-layout", h.GetKPILayout)
		dashboard.PUT("/kpi-layout", h.SaveKPILayout)
		// 告警统计新增端点
		dashboard.GET("/alarm-efficiency", h.GetAlarmEfficiency)
		dashboard.GET("/alarm-heatmap", h.GetAlarmHeatmap)
		dashboard.GET("/alarm-heatmap-by-severity", h.GetAlarmHeatmapBySeverity)
	}
}

// GetSummary handles GET /api/v1/dashboard/summary.
func (h *Handler) GetSummary(c *gin.Context) {
	summary, err := h.service.GetSummary(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, summary)
}

// GetAlarmTrend handles GET /api/v1/dashboard/alarm-trend?days=7&metric=raised.
func (h *Handler) GetAlarmTrend(c *gin.Context) {
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || parsed < 1 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid days parameter: %s", daysStr))
			return
		}
		days = parsed
	}

	metric := c.DefaultQuery("metric", "raised")
	if metric != "raised" && metric != "active" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid metric parameter: %s", metric))
		return
	}

	var (
		entries []AlarmTrendEntry
		err     error
	)
	if metric == "active" {
		visibleGroups, ok := h.resolver.FromContext(c)
		if !ok {
			return
		}
		entries, err = h.service.GetActiveAlarmTrend(c.Request.Context(), days, visibleGroups)
	} else {
		entries, err = h.service.GetAlarmTrend(c.Request.Context(), days)
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, entries)
}

// GetDeviceStatus handles GET /api/v1/dashboard/device-status.
func (h *Handler) GetDeviceStatus(c *gin.Context) {
	counts, err := h.service.GetDeviceStatus(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, counts)
}

// GetKPITrend handles GET /api/v1/dashboard/kpi-trend?kpi_name=...&days=7 or &compare_with=yesterday.
func (h *Handler) GetKPITrend(c *gin.Context) {
	kpiName := c.Query("kpi_name")
	if kpiName == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("kpi_name is required"))
		return
	}

	// Check if compare_with parameter is present
	compareWith := c.Query("compare_with")
	if compareWith != "" {
		// Use comparison API
		if compareWith != "yesterday" && compareWith != "last_week" {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("compare_with must be 'yesterday' or 'last_week'"))
			return
		}
		result, err := h.service.GetKPITrendComparison(c.Request.Context(), kpiName, compareWith)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		response.OK(c, result)
		return
	}

	// Original behavior: use days parameter
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || parsed < 1 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid days parameter: %s", daysStr))
			return
		}
		days = parsed
	}

	entries, err := h.service.GetKPITrend(c.Request.Context(), kpiName, days)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, entries)
}

// GetRegionStats handles GET /api/v1/dashboard/region-stats.
func (h *Handler) GetRegionStats(c *gin.Context) {
	entries, err := h.service.GetRegionStats(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, entries)
}

// GetWidgets handles GET /api/v1/dashboard/widgets.
func (h *Handler) GetWidgets(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, err)
		return
	}

	layout, err := h.service.GetWidgetLayout(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, layout)
}

// SaveWidgets handles PUT /api/v1/dashboard/widgets.
func (h *Handler) SaveWidgets(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, err)
		return
	}

	var body struct {
		Layout json.RawMessage `json:"layout" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid request body: %w", err))
		return
	}

	layout, err := h.service.SaveWidgetLayout(c.Request.Context(), userID, body.Layout)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, layout)
}

// GetAlarmTypePie handles GET /api/v1/dashboard/alarm-type-pie.
func (h *Handler) GetAlarmTypePie(c *gin.Context) {
	entries, err := h.service.GetAlarmTypePie(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, entries)
}

func parseDashboardKPIGranularity(raw string) (metrics.Granularity, error) {
	switch raw {
	case "", string(metrics.GranularityHourly):
		return metrics.GranularityHourly, nil
	case string(metrics.GranularityDaily):
		return metrics.GranularityDaily, nil
	default:
		return "", fmt.Errorf("invalid granularity %q (allowed: hourly, daily)", raw)
	}
}

// GetKPITimeSeries handles GET /api/v1/dashboard/kpi-time-series?kpi_names=...&start_time=...&end_time=...&granularity=...
func (h *Handler) GetKPITimeSeries(c *gin.Context) {
	kpiNamesRaw := c.Query("kpi_names")
	kpiNames := parseKPINames(kpiNamesRaw)
	if len(kpiNames) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("kpi_names is required (comma-separated)"))
		return
	}

	// Default to last 7 days if not specified
	now := time.Now()
	startTime := now.AddDate(0, 0, -7)
	endTime := now

	if st := c.Query("start_time"); st != "" {
		parsed, err := time.Parse(time.RFC3339, st)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid start_time (use RFC3339 format): %w", err))
			return
		}
		startTime = parsed
	}

	if et := c.Query("end_time"); et != "" {
		parsed, err := time.Parse(time.RFC3339, et)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid end_time (use RFC3339 format): %w", err))
			return
		}
		endTime = parsed
	}
	if !endTime.After(startTime) {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("end_time must be after start_time"))
		return
	}

	granularity, err := parseDashboardKPIGranularity(c.Query("granularity"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	technology, err := parseDashboardKPITechnology(c.Query("technology"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.GetKPITimeSeries(c.Request.Context(), kpiNames, technology, granularity, startTime, endTime)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func parseDashboardKPITechnology(raw string) (model.Technology, error) {
	if raw == "" {
		return "", nil
	}
	technology := model.Technology(raw)
	if !technology.IsValid() {
		return "", fmt.Errorf("invalid technology %q (allowed: lte, nr, gsm)", raw)
	}
	return technology, nil
}

// GetKPIDefinitions handles GET /api/v1/dashboard/kpi/definitions.
//
// issue #213 Phase1：返回 Dashboard 首页全部 KPI 的动态定义（symbolic key + K 编号 +
// 中文名 + 单位，按制式 / Panel 分组），供前端动态加载指标列表。
func (h *Handler) GetKPIDefinitions(c *gin.Context) {
	defs, err := h.service.GetKPIDefinitions(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, defs)
}

// GetKPILayout handles GET /api/v1/dashboard/kpi-layout?tech=lte.
//
// issue #213 S1：读全局首页 KPI 布局（所有登录用户可读，按制式带参）。
// 无配置时由 service 回退内置默认布局，保证永不空白。tech 缺省 lte。
func (h *Handler) GetKPILayout(c *gin.Context) {
	tech := c.Query("tech")
	if tech == "" {
		tech = "lte"
	}
	layout, err := h.service.GetKPILayout(c.Request.Context(), tech)
	if err != nil {
		if err == ErrInvalidTech {
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, layout)
}

// SaveKPILayout handles PUT /api/v1/dashboard/kpi-layout.
//
// issue #213 S1：存全局首页 KPI 布局（仅管理员）。handler 层再校验一次管理员身份
// （super_admin IsSuperAdmin 旁路 + Casbin 端点级权限点），非管理员一律 403。
func (h *Handler) SaveKPILayout(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, err)
		return
	}

	// 管理员二次校验：super_admin 旁路，否则查 Casbin 端点级权限点（与中间件同口径）。
	if !h.isAdmin(c, userID) {
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("insufficient permissions: admin required to save dashboard layout"))
		return
	}

	var body struct {
		Tech   string          `json:"tech" binding:"required"`
		Layout json.RawMessage `json:"layout" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid request body: %w", err))
		return
	}

	layout, err := h.service.SaveKPILayout(c.Request.Context(), body.Tech, body.Layout, userID)
	if err != nil {
		if err == ErrInvalidTech {
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, layout)
}

// isAdmin 判定当前请求者是否有权存全局布局。
//
// 复用 admin 现成机制，不发明新身份体系：
//  1. super_admin（builtIn 用户，IsSuperAdmin）直接放行——与 RequireAPIPermission 旁路一致。
//  2. 否则查 Casbin 端点级权限点（path+method），命中即放行；未授任何角色该端点 → 拒绝。
func (h *Handler) isAdmin(c *gin.Context, userID uuid.UUID) bool {
	if isSuper, _ := c.Get(admin.CtxKeyIsSuperAdmin); isSuper == true {
		return true
	}
	if h.permChecker == nil {
		// 无权限检查器时保守拒绝（只认 super_admin），不放行普通用户。
		return false
	}
	allowed, err := h.permChecker.CheckPermission(
		c.Request.Context(), userID, c.Request.URL.Path, c.Request.Method)
	if err != nil {
		return false
	}
	return allowed
}

// getUserID extracts the authenticated user's UUID from the Gin context.
func getUserID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get(admin.CtxKeyUserID)
	if !exists {
		return uuid.UUID{}, fmt.Errorf("user not authenticated")
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("invalid user ID in context")
	}
	return userID, nil
}

// GetDeviceStatusByType handles GET /api/v1/dashboard/device-status-by-type.
func (h *Handler) GetDeviceStatusByType(c *gin.Context) {
	result, err := h.service.GetDeviceStatusByType(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// GetAlarmEfficiency handles GET /api/v1/dashboard/alarm-efficiency.
// 获取告警处理效率指标（MTTA、MTTR、确认率、清除率）
func (h *Handler) GetAlarmEfficiency(c *gin.Context) {
	metrics, err := h.service.GetOverallEfficiencyMetrics(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, metrics)
}

// GetAlarmHeatmap handles GET /api/v1/dashboard/alarm-heatmap?days=30.
// 获取告警热度图数据（按星期几和小时统计）
func (h *Handler) GetAlarmHeatmap(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || parsed < 1 || parsed > 365 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid days parameter: must be between 1 and 365"))
			return
		}
		days = parsed
	}

	heatmap, err := h.service.GetAlarmHeatmap(c.Request.Context(), days)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, heatmap)
}

// GetAlarmHeatmapBySeverity handles GET /api/v1/dashboard/alarm-heatmap-by-severity?days=30&severity=critical.
// 获取按严重程度分组的告警热度图数据
func (h *Handler) GetAlarmHeatmapBySeverity(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || parsed < 1 || parsed > 365 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid days parameter: must be between 1 and 365"))
			return
		}
		days = parsed
	}

	severity := c.Query("severity") // 可选：critical, major, minor, warning

	heatmap, err := h.service.GetAlarmHeatmapBySeverity(c.Request.Context(), days, severity)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, heatmap)
}
