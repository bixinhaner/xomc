package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for the notification REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new notification Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("notification-handler"),
	}
}

// RegisterRoutes registers notification routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	notifs := rg.Group("/notifications")

	notifs.GET("", h.List)
	notifs.GET("/unread-count", h.GetUnreadCount)
	notifs.PUT("/:id/read", h.MarkRead)
	notifs.PUT("/read-all", h.MarkAllRead)
	notifs.DELETE("/:id", h.Delete)
}

// ---- Request types ----

// CreateNotificationRequest defines the request body for creating a notification.
type CreateNotificationRequest struct {
	UserID   string               `json:"user_id" binding:"required"`
	Type     NotificationType     `json:"type" binding:"required"`
	Priority NotificationPriority `json:"priority"`
	Title    string               `json:"title" binding:"required"`
	Content  string               `json:"content"`
	Link     string               `json:"link"`
	Sender   string               `json:"sender"`
}

// ---- Handlers ----

// List handles GET /notifications.
func (h *Handler) List(c *gin.Context) {
	username, _ := c.Get("username")
	userID, _ := username.(string)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	filter := NotificationFilter{
		UserID:      userID,
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if t := c.Query("type"); t != "" {
		nt := NotificationType(t)
		filter.Type = &nt
	}
	if r := c.Query("is_read"); r == "true" {
		val := true
		filter.IsRead = &val
	} else if r == "false" {
		val := false
		filter.IsRead = &val
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUnreadCount handles GET /notifications/unread-count.
func (h *Handler) GetUnreadCount(c *gin.Context) {
	username, _ := c.Get("username")
	userID, _ := username.(string)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkRead handles PUT /notifications/:id/read.
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	username, _ := c.Get("username")
	userID, _ := username.(string)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	if err := h.service.MarkRead(c.Request.Context(), id, userID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// MarkAllRead handles PUT /notifications/read-all.
func (h *Handler) MarkAllRead(c *gin.Context) {
	username, _ := c.Get("username")
	userID, _ := username.(string)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	if err := h.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Delete handles DELETE /notifications/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	username, _ := c.Get("username")
	userID, _ := username.(string)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, userID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}
