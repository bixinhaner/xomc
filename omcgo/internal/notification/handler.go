package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
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
	notifs.DELETE("", h.DeleteAll)        // T-0157 C4: 一键清空当前用户全部消息
	notifs.POST("/sync-stale", h.SyncStale) // T-0157 stale sync: 反查 task 状态修正卡死消息
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
	// T-0157 C5/C10: user_id 统一用 admin 中间件 c.Set(CtxKeyUserID, uuid) 的 uuid string，
	// 与 task subscriber (写入侧) 一致；旧逻辑用 c.Get("username") 拿到的是 username 字符串，
	// 与 task.CreatorID = admin.UserIDStringFromCtx 写入的 uuid 字段错配 → 查不到消息。
	userID := admin.UserIDStringFromCtx(c)
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

	response.OK(c, result)
}

// GetUnreadCount handles GET /notifications/unread-count.
func (h *Handler) GetUnreadCount(c *gin.Context) {
	// T-0157 C5/C10: user_id 统一用 admin 中间件 c.Set(CtxKeyUserID, uuid) 的 uuid string，
	// 与 task subscriber (写入侧) 一致；旧逻辑用 c.Get("username") 拿到的是 username 字符串，
	// 与 task.CreatorID = admin.UserIDStringFromCtx 写入的 uuid 字段错配 → 查不到消息。
	userID := admin.UserIDStringFromCtx(c)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, gin.H{"count": count})
}

// MarkRead handles PUT /notifications/:id/read.
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// T-0157 C5/C10: user_id 统一用 admin 中间件 c.Set(CtxKeyUserID, uuid) 的 uuid string，
	// 与 task subscriber (写入侧) 一致；旧逻辑用 c.Get("username") 拿到的是 username 字符串，
	// 与 task.CreatorID = admin.UserIDStringFromCtx 写入的 uuid 字段错配 → 查不到消息。
	userID := admin.UserIDStringFromCtx(c)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	if err := h.service.MarkRead(c.Request.Context(), id, userID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
}

// MarkAllRead handles PUT /notifications/read-all.
func (h *Handler) MarkAllRead(c *gin.Context) {
	// T-0157 C5/C10: user_id 统一用 admin 中间件 c.Set(CtxKeyUserID, uuid) 的 uuid string，
	// 与 task subscriber (写入侧) 一致；旧逻辑用 c.Get("username") 拿到的是 username 字符串，
	// 与 task.CreatorID = admin.UserIDStringFromCtx 写入的 uuid 字段错配 → 查不到消息。
	userID := admin.UserIDStringFromCtx(c)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	if err := h.service.MarkAllRead(c.Request.Context(), userID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
}

// DeleteAll handles DELETE /notifications — 一键清空当前用户全部消息 (T-0157 C4)。
// 返回 { "deleted": <int64> } 给前端"清空"按钮显示删除数。
func (h *Handler) DeleteAll(c *gin.Context) {
	// T-0157 C5/C10: user_id 统一用 admin 中间件 c.Set(CtxKeyUserID, uuid) 的 uuid string，
	// 与 task subscriber (写入侧) 一致；旧逻辑用 c.Get("username") 拿到的是 username 字符串，
	// 与 task.CreatorID = admin.UserIDStringFromCtx 写入的 uuid 字段错配 → 查不到消息。
	userID := admin.UserIDStringFromCtx(c)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	deleted, err := h.service.DeleteAllByUser(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, gin.H{"deleted": deleted})
}

// SyncStale handles POST /notifications/sync-stale (T-0157 stale sync)。
// 用于 Popover 打开 / 用户主动刷新场景：把 queued/sent 但 task 已终态的消息修正状态。
// 返回 { "updated": N }。
func (h *Handler) SyncStale(c *gin.Context) {
	userID := admin.UserIDStringFromCtx(c)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}
	updated, err := h.service.SyncStaleByUser(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"updated": updated})
}

// Delete handles DELETE /notifications/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// T-0157 C5/C10: user_id 统一用 admin 中间件 c.Set(CtxKeyUserID, uuid) 的 uuid string，
	// 与 task subscriber (写入侧) 一致；旧逻辑用 c.Get("username") 拿到的是 username 字符串，
	// 与 task.CreatorID = admin.UserIDStringFromCtx 写入的 uuid 字段错配 → 查不到消息。
	userID := admin.UserIDStringFromCtx(c)
	if userID == "" {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, userID); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
}
