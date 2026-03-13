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
	engine *AlarmEngine
	store  AlarmStore
	logger *zap.Logger
}

// NewHandler creates a new alarm handler.
func NewHandler(engine *AlarmEngine, store AlarmStore, logger *zap.Logger) *Handler {
	return &Handler{engine: engine, store: store, logger: logger}
}

// RegisterRoutes registers alarm API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	alarms := rg.Group("/alarms")
	{
		alarms.GET("/active", h.ListActive)
		alarms.GET("/history", h.ListHistory)
		alarms.GET("/statistics", h.Statistics)
		alarms.GET("/:id", h.GetByID)
		alarms.POST("/:id/acknowledge", h.Acknowledge)
		alarms.POST("/:id/clear", h.ClearAlarm)
	}
}

type alarmQuery struct {
	DeviceID  string `form:"device_id"`
	DeviceSN  string `form:"device_sn"`
	Carrier   string `form:"carrier"`
	Severity  string `form:"severity"`
	Status    string `form:"status"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
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
		id, _ := uuid.Parse(q.DeviceID)
		filter.DeviceID = &id
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
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
