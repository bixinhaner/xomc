package alarm

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

type localizedAlarmReader interface {
	GetActiveByIDLocalized(ctx context.Context, id uuid.UUID) (*model.Alarm, error)
	GetHistoryByIDLocalized(ctx context.Context, id uuid.UUID) (*model.Alarm, error)
}

// Handler provides REST API endpoints for alarm management.
type Handler struct {
	engine      *AlarmEngine
	store       AlarmStore
	syncService *AlarmSyncService
	resolver    *authz.Resolver
	logger      *zap.Logger
}

// NewHandler creates a new alarm handler.
func NewHandler(engine *AlarmEngine, store AlarmStore, syncService *AlarmSyncService, logger *zap.Logger) *Handler {
	return &Handler{engine: engine, store: store, syncService: syncService, logger: logger}
}

// SetPermissionService 注入数据权限解析器（#64 统一强制层），使告警读链路按调用者
// 可见设备组过滤。未注入时退化为不过滤（dev/test），与 device 模块语义一致。
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

// applyVisibleGroups 解析调用者可见设备组并写入 filter.VisibleGroups。
// 返回 false 表示解析失败已 abort（403/500），调用方应立即 return。
// h.resolver 为 nil 时 FromContext 走 nil-safe 退化路径，返回 (nil, true) 不过滤。
func (h *Handler) applyVisibleGroups(c *gin.Context, filter *AlarmFilter) bool {
	groups, ok := h.resolver.FromContext(c)
	if !ok {
		return false
	}
	filter.VisibleGroups = groups
	return true
}

// RegisterRoutes registers alarm API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	alarms := rg.Group("/alarms")
	{
		alarms.GET("/active", h.ListActive)
		alarms.GET("/history", h.ListHistory)
		alarms.GET("/statistics", h.Statistics)
		alarms.GET("/history/statistics", h.HistoryStatistics)
		alarms.POST("/history/batch/acknowledge", h.BatchHistoryAcknowledge)
		alarms.POST("/history/batch/unacknowledge", h.BatchHistoryUnacknowledge)
		alarms.POST("/history/batch/delete", h.BatchHistoryDelete)
		alarms.GET("/:id", h.GetByID)
		alarms.POST("/:id/acknowledge", h.Acknowledge)
		alarms.POST("/:id/clear", h.ClearAlarm)
		// 增强接口
		alarms.POST("/active/batch/acknowledge", h.BatchAcknowledge)
		alarms.POST("/active/batch/clear", h.BatchClear)
		alarms.POST("/active/batch/unacknowledge", h.BatchUnacknowledge)
		alarms.POST("/active/:id/read", h.MarkRead)
		// 告警同步
		alarms.POST("/sync/:device_sn", h.TriggerSync)
	}
}

type alarmQuery struct {
	DeviceID   string `form:"device_id"`
	DeviceSN   string `form:"device_sn"`
	Carrier    string `form:"carrier"`
	Severity   string `form:"severity"`
	Status     string `form:"status"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	AlarmType  string `form:"alarm_type"`
	EventType  string `form:"event_type"`
	NeType     string `form:"ne_type"`
	IsRead     string `form:"is_read"`
	IsUnknown  string `form:"is_unknown"`
	DeviceName string `form:"device_name"`
	Keyword    string `form:"keyword"`
	model.ListRequest
}

func (h *Handler) ListActive(c *gin.Context) {
	var q alarmQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := AlarmFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DeviceID = &id
	}
	if q.DeviceSN != "" {
		filter.DeviceSN = &q.DeviceSN
	}
	if q.Carrier != "" {
		cc := model.CarrierCode(q.Carrier)
		filter.Carrier = &cc
	}
	if q.Severity != "" {
		severities := parseSeverities(q.Severity)
		switch len(severities) {
		case 1:
			filter.Severity = &severities[0]
		case 2, 3, 4:
			filter.Severities = severities
		}
	}
	if q.Status != "" {
		st := model.AlarmStatus(q.Status)
		filter.Status = &st
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}
	if q.AlarmType != "" {
		filter.AlarmType = &q.AlarmType
	}
	if q.EventType != "" {
		filter.EventType = &q.EventType
	}
	if q.NeType != "" {
		filter.Technologies = []string{q.NeType}
	}
	if q.IsRead != "" {
		read := q.IsRead == "true"
		filter.IsRead = &read
	}
	if q.IsUnknown != "" {
		unk := q.IsUnknown == "true"
		filter.IsUnknown = &unk
	}
	if q.DeviceName != "" {
		filter.DeviceName = &q.DeviceName
	}
	if q.Keyword != "" {
		filter.Keyword = &q.Keyword
	}

	if !h.applyVisibleGroups(c, &filter) {
		return
	}
	result, err := h.store.ListActive(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListHistory(c *gin.Context) {
	var q alarmQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := AlarmFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DeviceID = &id
	}
	if q.DeviceSN != "" {
		filter.DeviceSN = &q.DeviceSN
	}
	if q.Carrier != "" {
		cc := model.CarrierCode(q.Carrier)
		filter.Carrier = &cc
	}
	if q.Severity != "" {
		severities := parseSeverities(q.Severity)
		switch len(severities) {
		case 1:
			filter.Severity = &severities[0]
		case 2, 3, 4:
			filter.Severities = severities
		}
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}
	if q.AlarmType != "" {
		filter.AlarmType = &q.AlarmType
	}
	if q.EventType != "" {
		filter.EventType = &q.EventType
	}
	if q.NeType != "" {
		filter.Technologies = []string{q.NeType}
	}
	if q.DeviceName != "" {
		filter.DeviceName = &q.DeviceName
	}
	if q.Keyword != "" {
		filter.Keyword = &q.Keyword
	}

	if !h.applyVisibleGroups(c, &filter) {
		return
	}
	result, err := h.store.ListHistory(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	getActive := h.store.GetActiveByID
	getHistory := h.store.GetHistoryByID
	if reader, ok := h.store.(localizedAlarmReader); ok {
		getActive = reader.GetActiveByIDLocalized
		getHistory = reader.GetHistoryByIDLocalized
	}
	alarm, err := getActive(c.Request.Context(), id)
	if err != nil {
		alarm, err = getHistory(c.Request.Context(), id)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
	}
	response.OK(c, alarm)
}

type acknowledgeRequest struct {
	AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
}

func (h *Handler) Acknowledge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var req acknowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.engine.Acknowledge(c.Request.Context(), id, req.AcknowledgedBy); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "alarm acknowledged")
}

func (h *Handler) ClearAlarm(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.engine.Clear(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "alarm cleared")
}

func (h *Handler) Statistics(c *gin.Context) {
	filter := AlarmFilter{}
	if deviceID := c.Query("device_id"); deviceID != "" {
		id, _ := uuid.Parse(deviceID)
		filter.DeviceID = &id
	}
	if !h.applyVisibleGroups(c, &filter) {
		return
	}
	stats, err := h.store.Statistics(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, stats)
}

func (h *Handler) HistoryStatistics(c *gin.Context) {
	filter := AlarmFilter{}
	if !h.applyVisibleGroups(c, &filter) {
		return
	}
	stats, err := h.store.HistoryStatistics(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, stats)
}

// BatchAcknowledge handles POST /alarms/active/batch/acknowledge.
func (h *Handler) BatchAcknowledge(c *gin.Context) {
	var req struct {
		IDs  []uuid.UUID `json:"ids" binding:"required"`
		Note string      `json:"acknowledged_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	username, _ := c.Get("username")
	if username == nil {
		username = "operator"
	}
	if err := h.store.BatchAcknowledge(c.Request.Context(), req.IDs, username.(string), req.Note); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"count": len(req.IDs)}, "alarms acknowledged")
}

// BatchUnacknowledge handles POST /alarms/active/batch/unacknowledge.
func (h *Handler) BatchUnacknowledge(c *gin.Context) {
	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.store.BatchUnacknowledge(c.Request.Context(), req.IDs); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"count": len(req.IDs)}, "alarms unacknowledged")
}

// BatchClear handles POST /alarms/active/batch/clear.
func (h *Handler) BatchClear(c *gin.Context) {
	var req struct {
		IDs  []uuid.UUID `json:"ids" binding:"required"`
		Note string      `json:"clear_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	username, _ := c.Get("username")
	if username == nil {
		username = "operator"
	}
	for _, id := range req.IDs {
		if err := h.engine.clearByOperator(c.Request.Context(), id, username.(string), req.Note); err != nil {
			// Preserve the previous batch-clear behavior: alarms that were
			// already removed by another operation do not fail the whole batch.
			if errors.Is(err, commonerrors.ErrNotFound) {
				continue
			}
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
	}
	response.OKWithMsg(c, gin.H{"count": len(req.IDs)}, "alarms cleared")
}

// BatchHistoryAcknowledge handles POST /alarms/history/batch/acknowledge.
func (h *Handler) BatchHistoryAcknowledge(c *gin.Context) {
	var req struct {
		IDs  []uuid.UUID `json:"ids" binding:"required"`
		Note string      `json:"acknowledged_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	username, _ := c.Get("username")
	if username == nil {
		username = "operator"
	}
	if err := h.store.BatchHistoryAcknowledge(c.Request.Context(), req.IDs, username.(string), req.Note); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"count": len(req.IDs)}, "history alarms acknowledged")
}

// BatchHistoryUnacknowledge handles POST /alarms/history/batch/unacknowledge.
func (h *Handler) BatchHistoryUnacknowledge(c *gin.Context) {
	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.store.BatchHistoryUnacknowledge(c.Request.Context(), req.IDs); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"count": len(req.IDs)}, "history alarms unacknowledged")
}

// BatchHistoryDelete handles POST /alarms/history/batch/delete.
func (h *Handler) BatchHistoryDelete(c *gin.Context) {
	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.store.BatchHistoryDelete(c.Request.Context(), req.IDs); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{"count": len(req.IDs)}, "history alarms deleted")
}

// MarkRead handles POST /alarms/active/:id/read.
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.store.MarkRead(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, nil, "alarm marked as read")
}

func parseSeverity(s string) model.AlarmSeverity {
	switch s {
	case "1":
		return model.AlarmCritical
	case "31001":
		return model.AlarmCritical
	case "2":
		return model.AlarmMajor
	case "31002":
		return model.AlarmMajor
	case "3":
		return model.AlarmMinor
	case "31003":
		return model.AlarmMinor
	case "4":
		return model.AlarmWarning
	case "31004":
		return model.AlarmWarning
	default:
		return 0
	}
}

func parseSeverities(raw string) []model.AlarmSeverity {
	parts := strings.Split(raw, ",")
	severities := make([]model.AlarmSeverity, 0, len(parts))
	seen := make(map[model.AlarmSeverity]struct{}, len(parts))
	for _, part := range parts {
		severity := parseSeverity(strings.TrimSpace(part))
		if severity == 0 {
			continue
		}
		if _, exists := seen[severity]; exists {
			continue
		}
		seen[severity] = struct{}{}
		severities = append(severities, severity)
	}
	return severities
}

// TriggerSync handles POST /alarms/sync/:device_sn.
// It triggers an alarm synchronization for the specified device.
func (h *Handler) TriggerSync(c *gin.Context) {
	deviceSN := c.Param("device_sn")
	if deviceSN == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if h.syncService == nil {
		response.Fail(c, http.StatusServiceUnavailable, "alarm sync service not available")
		return
	}

	syncTask, err := h.syncService.TriggerSync(c.Request.Context(), deviceSN)
	if err != nil {
		h.logger.Error("trigger alarm sync", zap.Error(err), zap.String("device_sn", deviceSN))
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	resp := gin.H{"device_sn": deviceSN}
	if syncTask != nil {
		resp["task_id"] = syncTask.ID
	}
	response.OKWithMsg(c, resp, "alarm sync triggered")
}
