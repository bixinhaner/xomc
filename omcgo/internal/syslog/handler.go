package syslog

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for system logs and NE message logs.
type Handler struct {
	repo   SyslogRepository
	logger *zap.Logger
}

// NewHandler creates a new syslog handler.
func NewHandler(repo SyslogRepository, logger *zap.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

// RegisterRoutes registers syslog API routes under the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	logs := rg.Group("/logs")
	{
		logs.GET("/system", h.ListSystemLogs)
		logs.GET("/ne-messages", h.ListNEMessageLogs)
	}
}

type systemLogQuery struct {
	Level     string `form:"level"`
	Source    string `form:"source"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

// ListSystemLogs handles GET /logs/system.
func (h *Handler) ListSystemLogs(c *gin.Context) {
	var q systemLogQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := SystemLogFilter{ListRequest: q.ListRequest}
	if q.Level != "" {
		filter.Level = &q.Level
	}
	if q.Source != "" {
		filter.Source = &q.Source
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

	result, err := h.repo.ListSystemLogs(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

type neMessageLogQuery struct {
	DeviceSN    string `form:"device_sn"`
	DeviceID    string `form:"device_id"`
	MessageType string `form:"message_type"`
	Direction   string `form:"direction"`
	StartTime   string `form:"start_time"`
	EndTime     string `form:"end_time"`
	model.ListRequest
}

// ListNEMessageLogs handles GET /logs/ne-messages.
func (h *Handler) ListNEMessageLogs(c *gin.Context) {
	var q neMessageLogQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := NEMessageLogFilter{ListRequest: q.ListRequest}
	if q.DeviceSN != "" {
		filter.DeviceSN = &q.DeviceSN
	}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DeviceID = &id
	}
	if q.MessageType != "" {
		filter.MessageType = &q.MessageType
	}
	if q.Direction != "" {
		filter.Direction = &q.Direction
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

	result, err := h.repo.ListNEMessageLogs(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}
