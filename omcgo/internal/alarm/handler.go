package alarm

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for alarm management.
type Handler struct {
	engine      *AlarmEngine
	store       AlarmStore
	syncService *AlarmSyncService
	logger      *zap.Logger
}

// NewHandler creates a new alarm handler.
func NewHandler(engine *AlarmEngine, store AlarmStore, syncService *AlarmSyncService, logger *zap.Logger) *Handler {
	return &Handler{engine: engine, store: store, syncService: syncService, logger: logger}
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
	IsRead     string `form:"is_read"`
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
		sev := parseSeverity(q.Severity)
		if sev != 0 {
			filter.Severity = &sev
		}
	}
	if q.Status != "" {
		st := model.AlarmStatus(q.Status)
		filter.Status = &st
	}
	if q.AlarmType != "" {
		filter.AlarmType = &q.AlarmType
	}
	if q.IsRead != "" {
		read := q.IsRead == "true"
		filter.IsRead = &read
	}
	if q.DeviceName != "" {
		filter.DeviceName = &q.DeviceName
	}
	if q.Keyword != "" {
		filter.Keyword = &q.Keyword
	}

	result, err := h.store.ListActive(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
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
	if q.Carrier != "" {
		cc := model.CarrierCode(q.Carrier)
		filter.Carrier = &cc
	}
	if q.Severity != "" {
		sev := parseSeverity(q.Severity)
		if sev != 0 {
			filter.Severity = &sev
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
	if q.DeviceName != "" {
		filter.DeviceName = &q.DeviceName
	}
	if q.Keyword != "" {
		filter.Keyword = &q.Keyword
	}

	result, err := h.store.ListHistory(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	alarm, err := h.store.GetActiveByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	c.JSON(http.StatusOK, alarm)
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
	c.JSON(http.StatusOK, gin.H{"message": "alarm acknowledged"})
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
	c.JSON(http.StatusOK, gin.H{"message": "alarm cleared"})
}

func (h *Handler) Statistics(c *gin.Context) {
	filter := AlarmFilter{}
	if deviceID := c.Query("device_id"); deviceID != "" {
		id, _ := uuid.Parse(deviceID)
		filter.DeviceID = &id
	}
	stats, err := h.store.Statistics(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) HistoryStatistics(c *gin.Context) {
	stats, err := h.store.HistoryStatistics(c.Request.Context(), AlarmFilter{})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// BatchAcknowledge handles POST /alarms/active/batch/acknowledge.
func (h *Handler) BatchAcknowledge(c *gin.Context) {
	var req struct {
		IDs   []uuid.UUID `json:"ids" binding:"required"`
		Note  string      `json:"acknowledged_by"`
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
	c.JSON(http.StatusOK, gin.H{"message": "alarms acknowledged", "count": len(req.IDs)})
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
	c.JSON(http.StatusOK, gin.H{"message": "alarms unacknowledged", "count": len(req.IDs)})
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
	if err := h.store.BatchClear(c.Request.Context(), req.IDs, username.(string), req.Note); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "alarms cleared", "count": len(req.IDs)})
}

// BatchHistoryAcknowledge handles POST /alarms/history/batch/acknowledge.
func (h *Handler) BatchHistoryAcknowledge(c *gin.Context) {
	var req struct {
		IDs   []uuid.UUID `json:"ids" binding:"required"`
		Note  string      `json:"acknowledged_by"`
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
	c.JSON(http.StatusOK, gin.H{"message": "history alarms acknowledged", "count": len(req.IDs)})
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
	c.JSON(http.StatusOK, gin.H{"message": "history alarms unacknowledged", "count": len(req.IDs)})
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
	c.JSON(http.StatusOK, gin.H{"message": "history alarms deleted", "count": len(req.IDs)})
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
	c.JSON(http.StatusOK, gin.H{"message": "alarm marked as read"})
}

func parseSeverity(s string) model.AlarmSeverity {
	switch s {
	case "1":
		return model.AlarmCritical
	case "2":
		return model.AlarmMajor
	case "3":
		return model.AlarmMinor
	case "4":
		return model.AlarmWarning
	default:
		return 0
	}
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
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "alarm sync service not available"})
		return
	}

	if err := h.syncService.TriggerSync(c.Request.Context(), deviceSN); err != nil {
		h.logger.Error("trigger alarm sync", zap.Error(err), zap.String("device_sn", deviceSN))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "alarm sync triggered",
		"device_sn": deviceSN,
	})
}
