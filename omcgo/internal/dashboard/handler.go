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
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler provides REST API endpoints for dashboard.
type Handler struct {
	service *Service
}

// NewHandler creates a new dashboard handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers dashboard routes on the given router group.
// NOTE: rg 应该已经应用了认证中间件（如 JWT 验证），所有端点都会自动受到保护。
// 如需添加额外的权限检查，请在具体 handler 中实现。
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

// GetAlarmTrend handles GET /api/v1/dashboard/alarm-trend?days=7.
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

	entries, err := h.service.GetAlarmTrend(c.Request.Context(), days)
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

// GetKPITimeSeries handles GET /api/v1/dashboard/kpi-time-series?kpi_names=...&start_time=...&end_time=...
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

	result, err := h.service.GetKPITimeSeries(c.Request.Context(), kpiNames, startTime, endTime)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
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
