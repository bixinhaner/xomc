package alarm

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type AlarmEmailManagementRepository interface {
	GetGlobalSetting(ctx context.Context) (*AlarmEmailGlobalSetting, error)
	UpdateGlobalSetting(ctx context.Context, setting *AlarmEmailGlobalSetting) error
	ListSubscriptions(ctx context.Context) ([]AlarmEmailSubscription, error)
	GetSubscription(ctx context.Context, id uuid.UUID) (*AlarmEmailSubscription, error)
	CreateSubscription(ctx context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string) error
	UpdateSubscription(ctx context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error
}

var _ AlarmEmailManagementRepository = (*PgAlarmEmailRepository)(nil)

type AlarmEmailSubscriptionHandler struct {
	repository  AlarmEmailManagementRepository
	authz       *authz.Resolver
	groupReader authz.GroupReader
}

func NewAlarmEmailSubscriptionHandler(repository AlarmEmailManagementRepository, permissionService authz.VisibleGroupsResolver, groupReader authz.GroupReader) *AlarmEmailSubscriptionHandler {
	return &AlarmEmailSubscriptionHandler{
		repository:  repository,
		authz:       authz.NewResolver(permissionService),
		groupReader: groupReader,
	}
}

func (h *AlarmEmailSubscriptionHandler) RegisterRoutes(group *gin.RouterGroup) {
	settings := group.Group("/alarms/email-settings")
	settings.GET("", h.GetGlobalSetting)
	settings.PUT("", h.UpdateGlobalSetting)

	subscriptions := group.Group("/alarms/email-subscriptions")
	subscriptions.GET("", h.ListSubscriptions)
	subscriptions.POST("", h.CreateSubscription)
	subscriptions.PUT("/:id", h.UpdateSubscription)
	subscriptions.DELETE("/:id", h.DeleteSubscription)
}

func (h *AlarmEmailSubscriptionHandler) GetGlobalSetting(c *gin.Context) {
	setting, err := h.repository.GetGlobalSetting(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, setting)
}

type updateAlarmEmailGlobalSettingRequest struct {
	Enabled           bool     `json:"enabled"`
	DefaultRecipients []string `json:"default_recipients"`
}

func (h *AlarmEmailSubscriptionHandler) UpdateGlobalSetting(c *gin.Context) {
	if !h.requireSuperAdmin(c) {
		return
	}
	var request updateAlarmEmailGlobalSettingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	recipients, err := NormalizeEmailRecipients(request.DefaultRecipients)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	setting := &AlarmEmailGlobalSetting{
		Enabled: request.Enabled, DefaultRecipients: recipients, UpdatedBy: alarmEmailActorID(c),
	}
	if err := h.repository.UpdateGlobalSetting(c.Request.Context(), setting); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, setting)
}

func (h *AlarmEmailSubscriptionHandler) ListSubscriptions(c *gin.Context) {
	items, err := h.repository.ListSubscriptions(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !alarmEmailCallerIsSuperAdmin(c) && h.authz.Enabled() {
		callerID, ok := alarmEmailCallerID(c)
		if !ok {
			commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
			return
		}
		owned := items[:0]
		for i := range items {
			if items[i].CreatedBy != nil && *items[i].CreatedBy == callerID {
				owned = append(owned, items[i])
			}
		}
		items = owned
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *AlarmEmailSubscriptionHandler) CreateSubscription(c *gin.Context) {
	var subscription AlarmEmailSubscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if !h.authorizeScope(c, &subscription) {
		return
	}
	setting, err := h.repository.GetGlobalSetting(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	subscription.ID = uuid.New()
	subscription.CreatedBy = alarmEmailActorID(c)
	subscription.UpdatedBy = subscription.CreatedBy
	if err := h.repository.CreateSubscription(c.Request.Context(), &subscription, setting.DefaultRecipients); err != nil {
		commonerrors.AbortWithError(c, alarmEmailWriteStatus(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, subscription)
}

func (h *AlarmEmailSubscriptionHandler) UpdateSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var subscription AlarmEmailSubscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	existing, err := h.repository.GetSubscription(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, alarmEmailWriteStatus(err), err)
		return
	}
	if !h.authorizeOwner(c, existing) {
		return
	}
	if !h.authorizeScope(c, &subscription) {
		return
	}
	setting, err := h.repository.GetGlobalSetting(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	subscription.ID = id
	subscription.CreatedBy = existing.CreatedBy
	subscription.UpdatedBy = alarmEmailActorID(c)
	if err := h.repository.UpdateSubscription(c.Request.Context(), &subscription, setting.DefaultRecipients); err != nil {
		commonerrors.AbortWithError(c, alarmEmailWriteStatus(err), err)
		return
	}
	response.OK(c, subscription)
}

func (h *AlarmEmailSubscriptionHandler) DeleteSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	existing, err := h.repository.GetSubscription(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, alarmEmailWriteStatus(err), err)
		return
	}
	if !h.authorizeOwner(c, existing) {
		return
	}
	if err := h.repository.DeleteSubscription(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, alarmEmailWriteStatus(err), err)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": id})
}

func (h *AlarmEmailSubscriptionHandler) authorizeScope(c *gin.Context, subscription *AlarmEmailSubscription) bool {
	visibleGroups, ok := h.authz.FromContext(c)
	if !ok || visibleGroups == nil {
		return ok
	}
	if len(subscription.DeviceIDs) == 0 && len(subscription.DeviceGroupIDs) == 0 {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return false
	}
	for _, selected := range subscription.DeviceGroupIDs {
		if !slices.Contains(visibleGroups, selected) {
			commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
			return false
		}
	}
	for _, deviceID := range subscription.DeviceIDs {
		if err := authz.AuthorizeDeviceAccess(c.Request.Context(), h.groupReader, deviceID, visibleGroups); err != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
			return false
		}
	}
	return true
}

func (h *AlarmEmailSubscriptionHandler) authorizeOwner(c *gin.Context, subscription *AlarmEmailSubscription) bool {
	if !h.authz.Enabled() || alarmEmailCallerIsSuperAdmin(c) {
		return true
	}
	callerID, ok := alarmEmailCallerID(c)
	if !ok || subscription.CreatedBy == nil || *subscription.CreatedBy != callerID {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return false
	}
	return true
}

func (h *AlarmEmailSubscriptionHandler) requireSuperAdmin(c *gin.Context) bool {
	if !h.authz.Enabled() || alarmEmailCallerIsSuperAdmin(c) {
		return true
	}
	commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
	return false
}

func alarmEmailCallerIsSuperAdmin(c *gin.Context) bool {
	value, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := value.(bool)
	return isSuper
}

func alarmEmailCallerID(c *gin.Context) (uuid.UUID, bool) {
	return admin.UserIDFromCtx(c)
}

func alarmEmailActorID(c *gin.Context) *uuid.UUID {
	id, ok := alarmEmailCallerID(c)
	if !ok {
		return nil
	}
	return &id
}

func alarmEmailWriteStatus(err error) int {
	if errors.Is(err, commonerrors.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrAlarmEmailInvalidInterval) ||
		errors.Is(err, ErrAlarmEmailInvalidTolerance) ||
		errors.Is(err, ErrAlarmEmailNoRecipients) ||
		errors.Is(err, ErrAlarmEmailInvalidRecipient) ||
		errors.Is(err, ErrAlarmEmailSubscriptionName) ||
		errors.Is(err, commonerrors.ErrInvalidInput) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
