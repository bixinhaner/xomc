package notification

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type DeliveryApplication interface {
	List(context.Context, uuid.UUID, bool, DeliveryFilter) ([]DeliveryView, error)
	Get(context.Context, uuid.UUID, bool, uuid.UUID) (DeliveryView, error)
	Attempts(context.Context, uuid.UUID, bool, uuid.UUID) ([]DomainDeliveryAttempt, error)
	Retry(context.Context, uuid.UUID, bool, []uuid.UUID, string, string) (int, error)
}

type DeliveryHandler struct{ service DeliveryApplication }

func NewDeliveryHandler(service DeliveryApplication) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

func (h *DeliveryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	deliveries := rg.Group("/notification-deliveries")
	deliveries.GET("", h.List)
	deliveries.POST("/retry", h.RetryBatch)
	deliveries.GET("/:id", h.Get)
	deliveries.GET("/:id/attempts", h.Attempts)
	deliveries.POST("/:id/retry", h.RetryOne)
}

func (h *DeliveryHandler) List(c *gin.Context) {
	userID, superAdmin, ok := deliveryRequestIdentity(c)
	if !ok {
		return
	}
	filter := DeliveryFilter{Channel: c.Query("channel"), FlowState: c.Query("flow_state")}
	if raw := c.Query("occurrence_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.OccurrenceID = &id
	}
	filter.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	filter.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := h.service.List(c.Request.Context(), userID, superAdmin, filter)
	if err != nil {
		abortDeliveryError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *DeliveryHandler) Get(c *gin.Context) {
	userID, superAdmin, id, ok := deliveryRequest(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), userID, superAdmin, id)
	if err != nil {
		abortDeliveryError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DeliveryHandler) Attempts(c *gin.Context) {
	userID, superAdmin, id, ok := deliveryRequest(c)
	if !ok {
		return
	}
	items, err := h.service.Attempts(c.Request.Context(), userID, superAdmin, id)
	if err != nil {
		abortDeliveryError(c, err)
		return
	}
	views := make([]DeliveryAttemptView, 0, len(items))
	for _, item := range items {
		views = append(views, newDeliveryAttemptView(item))
	}
	response.OK(c, views)
}

func (h *DeliveryHandler) RetryOne(c *gin.Context) {
	userID, superAdmin, id, ok := deliveryRequest(c)
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.retry(c, userID, superAdmin, []uuid.UUID{id}, request.Reason)
}

func (h *DeliveryHandler) RetryBatch(c *gin.Context) {
	userID, superAdmin, ok := deliveryRequestIdentity(c)
	if !ok {
		return
	}
	var request struct {
		DeliveryIDs []uuid.UUID `json:"delivery_ids" binding:"required"`
		Reason      string      `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.retry(c, userID, superAdmin, request.DeliveryIDs, request.Reason)
}

func (h *DeliveryHandler) retry(c *gin.Context, userID uuid.UUID, superAdmin bool, ids []uuid.UUID, reason string) {
	count, err := h.service.Retry(c.Request.Context(), userID, superAdmin, ids, reason, requestActor(c))
	if err != nil {
		abortDeliveryError(c, err)
		return
	}
	response.OK(c, gin.H{"retried": count})
}

type DeliveryAttemptView struct {
	ID                uuid.UUID  `json:"id"`
	DeliveryID        uuid.UUID  `json:"delivery_id"`
	AttemptNo         int        `json:"attempt_no"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	Result            string     `json:"result"`
	ErrorCategory     *string    `json:"error_category,omitempty"`
	StatusSummary     *string    `json:"status_summary,omitempty"`
	ProviderRequestID *string    `json:"provider_request_id,omitempty"`
	NextRetryAt       *time.Time `json:"next_retry_at,omitempty"`
}

func newDeliveryAttemptView(item DomainDeliveryAttempt) DeliveryAttemptView {
	return DeliveryAttemptView{
		ID: item.ID, DeliveryID: item.DeliveryID, AttemptNo: item.AttemptNo,
		StartedAt: item.StartedAt, FinishedAt: item.FinishedAt, Result: item.Result,
		ErrorCategory: item.ErrorCategory, StatusSummary: item.StatusSummary,
		ProviderRequestID: item.ProviderRequestID, NextRetryAt: item.NextRetryAt,
	}
}

func deliveryRequest(c *gin.Context) (uuid.UUID, bool, uuid.UUID, bool) {
	userID, superAdmin, ok := deliveryRequestIdentity(c)
	if !ok {
		return uuid.Nil, false, uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return uuid.Nil, false, uuid.Nil, false
	}
	return userID, superAdmin, id, true
}

func deliveryRequestIdentity(c *gin.Context) (uuid.UUID, bool, bool) {
	value, exists := c.Get(admin.CtxKeyUserID)
	userID, valid := value.(uuid.UUID)
	if !exists || !valid || userID == uuid.Nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return uuid.Nil, false, false
	}
	superAdmin, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuperAdmin, _ := superAdmin.(bool)
	return userID, isSuperAdmin, true
}

func abortDeliveryError(c *gin.Context, err error) {
	status := commonerrors.HTTPStatusFromError(err)
	switch {
	case errors.Is(err, ErrDeliveryNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrDeliveryRetryNotAllowed):
		status = http.StatusConflict
	}
	commonerrors.AbortWithError(c, status, err)
}
