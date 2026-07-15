package paramsync

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

type VisibleDeviceGrantsResolver interface {
	GetUserVisibleDeviceGrants(context.Context, uuid.UUID, bool) ([]model.DeviceVisibilityGrant, error)
}

type DeviceAccessAuthorizer interface {
	AuthorizeDeviceGroupAccessByGrants(context.Context, uuid.UUID, []model.DeviceVisibilityGrant) error
}

type Handler struct {
	service          *Service
	ops              *Operations
	permission       VisibleDeviceGrantsResolver
	deviceAuthorizer DeviceAccessAuthorizer
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) WithOperations(ops *Operations) *Handler { h.ops = ops; return h }

func (h *Handler) WithAuthorization(permission VisibleDeviceGrantsResolver, authorizer DeviceAccessAuthorizer) *Handler {
	h.permission = permission
	h.deviceAuthorizer = authorizer
	return h
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/parameter-sync/requests/:request_id", h.GetRequest)
	r.GET("/devices/:id/parameter-sync/active", h.GetActiveRun)
	r.GET("/devices/:id/parameter-sync/history", h.ListHistory)
	if h.ops != nil {
		r.POST("/parameter-sync/requests/:request_id/cancel", h.CancelRequest)
		r.POST("/parameter-sync/requests/:request_id/retry", h.RetryRequest)
		r.GET("/parameter-sync/outbox/dead", h.ListDeadOutbox)
		r.POST("/parameter-sync/outbox/:id/retry", h.RetryOutbox)
	}
}

func (h *Handler) CancelRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("request_id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	req, err := h.service.GetRequest(c.Request.Context(), id)
	if err != nil {
		h.abortLookupError(c, err)
		return
	}
	if !h.authorizeDevice(c, req.DeviceID) {
		return
	}
	if err := h.ops.CancelRequest(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	response.OK(c, gin.H{"status": "cancelled"})
}

func (h *Handler) RetryRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("request_id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	req, err := h.service.GetRequest(c.Request.Context(), id)
	if err != nil {
		h.abortLookupError(c, err)
		return
	}
	if !h.authorizeDevice(c, req.DeviceID) {
		return
	}
	result, err := h.ops.RetryRequest(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, result)
}

func (h *Handler) ListDeadOutbox(c *gin.Context) {
	if !requireParamSyncAdmin(c) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.ops.ListDeadOutbox(c.Request.Context(), limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) RetryOutbox(c *gin.Context) {
	if !requireParamSyncAdmin(c) {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.ops.RetryOutbox(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, err)
		return
	}
	response.OK(c, gin.H{"status": "pending"})
}

func (h *Handler) GetRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("request_id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	req, err := h.service.GetRequest(c.Request.Context(), id)
	if err != nil {
		h.abortLookupError(c, err)
		return
	}
	if !h.authorizeDevice(c, req.DeviceID) {
		return
	}
	response.OK(c, req)
}

func (h *Handler) GetActiveRun(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if !h.authorizeDevice(c, deviceID) {
		return
	}
	run, err := h.service.GetActiveRun(c.Request.Context(), deviceID)
	if err != nil {
		if IsNotFound(err) {
			response.OK(c, gin.H{"active_run": nil})
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"active_run": run})
}

func (h *Handler) ListHistory(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if !h.authorizeDevice(c, deviceID) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.service.ListHistory(c.Request.Context(), deviceID, limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) abortLookupError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if IsNotFound(err) {
		status = http.StatusNotFound
	}
	commonerrors.AbortWithError(c, status, err)
}

func (h *Handler) authorizeDevice(c *gin.Context, deviceID uuid.UUID) bool {
	if h.permission == nil || h.deviceAuthorizer == nil {
		return true
	}
	userID, ok := c.Get(admin.CtxKeyUserID)
	uid, valid := userID.(uuid.UUID)
	if !ok || !valid {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return false
	}
	isSuper, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	super, _ := isSuper.(bool)
	grants, err := h.permission.GetUserVisibleDeviceGrants(c.Request.Context(), uid, super)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return false
	}
	if err := h.deviceAuthorizer.AuthorizeDeviceGroupAccessByGrants(c.Request.Context(), deviceID, grants); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return false
	}
	return true
}

func requireParamSyncAdmin(c *gin.Context) bool {
	isSuper, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	if super, _ := isSuper.(bool); super {
		return true
	}
	commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
	return false
}
