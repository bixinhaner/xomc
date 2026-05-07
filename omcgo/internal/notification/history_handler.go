package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// HistoryHandler exposes read-only HTTP endpoints for notification history.
// Writes go through the internal dispatcher (Service.Insert), not HTTP.
type HistoryHandler struct {
	service *HistoryService
	logger  *zap.Logger
}

// NewHistoryHandler creates a HistoryHandler.
func NewHistoryHandler(service *HistoryService, logger *zap.Logger) *HistoryHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HistoryHandler{
		service: service,
		logger:  logger.Named("notification-history-handler"),
	}
}

// RegisterRoutes mounts /history under the supplied router group.
// Caller is expected to mount the group at /api/v1/notifications, so the final
// paths become /api/v1/notifications/history[...].
func (h *HistoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	history := rg.Group("/history")
	history.GET("", h.List)
	history.GET("/:id", h.GetByID)
}

// historyListQuery is the binding struct for GET /history query params.
type historyListQuery struct {
	Channel    string `form:"channel"`
	Status     string `form:"status"`
	TemplateID string `form:"template_id"`
	AlarmID    string `form:"alarm_id"`
	model.ListRequest
}

// List handles GET /history.
func (h *HistoryHandler) List(c *gin.Context) {
	q := historyListQuery{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := NotificationHistoryFilter{ListRequest: q.ListRequest}
	if q.Channel != "" {
		ch := q.Channel
		filter.Channel = &ch
	}
	if q.Status != "" {
		st := q.Status
		filter.Status = &st
	}
	if q.TemplateID != "" {
		tid, err := uuid.Parse(q.TemplateID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.TemplateID = &tid
	}
	if q.AlarmID != "" {
		aid, err := uuid.Parse(q.AlarmID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.AlarmID = &aid
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// GetByID handles GET /history/:id.
func (h *HistoryHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	entry, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, entry)
}
