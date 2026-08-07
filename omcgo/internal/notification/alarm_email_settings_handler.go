package notification

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type AlarmEmailSettingsHandler struct {
	service    *AlarmEmailSettingsService
	authorizer *RuleScopeAuthorizer
}

func NewAlarmEmailSettingsHandler(service *AlarmEmailSettingsService, authorizer *RuleScopeAuthorizer) *AlarmEmailSettingsHandler {
	return &AlarmEmailSettingsHandler{service: service, authorizer: authorizer}
}

func (h *AlarmEmailSettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/alarm-email-settings")
	g.GET("", h.List)
	g.GET("/default-recipients", h.GetDefaults)
	g.PATCH("/default-recipients", h.UpdateDefaults)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Archive)
}

func (h *AlarmEmailSettingsHandler) GetDefaults(c *gin.Context) {
	if !requireAlarmEmailDefaultsAdmin(c) {
		return
	}
	item, err := h.service.GetDefaults(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}

func (h *AlarmEmailSettingsHandler) UpdateDefaults(c *gin.Context) {
	if !requireAlarmEmailDefaultsAdmin(c) {
		return
	}
	revision, err := parseIfMatchRevision(c.GetHeader("If-Match"))
	if errors.Is(err, ErrPreconditionRequired) {
		commonerrors.AbortWithError(c, http.StatusPreconditionRequired, err)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var input AlarmEmailDefaultsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.UpdateDefaults(c.Request.Context(), revision, input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}

func (h *AlarmEmailSettingsHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	visible := make([]AlarmEmailSetting, 0, len(items))
	for i := range items {
		err := h.authorizeSetting(c, &items[i])
		if errors.Is(err, ErrRuleScopeDenied) {
			continue
		}
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		visible = append(visible, items[i])
	}
	response.OK(c, visible)
}

func (h *AlarmEmailSettingsHandler) Get(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	if !h.writeAuthorizationError(c, h.authorizeSetting(c, item)) {
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}

func (h *AlarmEmailSettingsHandler) Create(c *gin.Context) {
	var input AlarmEmailSettingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if !h.authorize(c, input) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OKWithStatus(c, http.StatusCreated, item)
}

func (h *AlarmEmailSettingsHandler) Update(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	var input AlarmEmailSettingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	existing, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	if !h.writeAuthorizationError(c, h.authorizeSetting(c, existing)) {
		return
	}
	if !h.authorize(c, input) {
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, revision, input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}

func (h *AlarmEmailSettingsHandler) Archive(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	existing, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	if !h.writeAuthorizationError(c, h.authorizeSetting(c, existing)) {
		return
	}
	if err := h.service.Archive(c.Request.Context(), id, revision); err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

func (h *AlarmEmailSettingsHandler) authorize(c *gin.Context, input AlarmEmailSettingInput) bool {
	input, _, err := validateAlarmEmailSettingInput(input)
	if err == nil {
		err = h.authorizeScope(c, RuleDraftInput{
			MatchConditions: RuleMatchConditions{
				AlarmIdentifiers: input.AlarmIdentifiers, Severities: input.Severities,
				DeviceIDs: input.DeviceIDs, DeviceGroupIDs: input.DeviceGroupIDs,
				Technologies: input.Technologies,
			},
		})
	}
	return h.writeAuthorizationError(c, err)
}

func (h *AlarmEmailSettingsHandler) authorizeSetting(c *gin.Context, setting *AlarmEmailSetting) error {
	if setting == nil {
		return commonerrors.ErrNotFound
	}
	return h.authorizeScope(c, RuleDraftInput{MatchConditions: RuleMatchConditions{
		AlarmIdentifiers: setting.AlarmIdentifiers,
		Severities:       setting.Severities,
		DeviceIDs:        setting.DeviceIDs,
		DeviceGroupIDs:   setting.DeviceGroupIDs,
		Technologies:     setting.Technologies,
	}})
}

func (h *AlarmEmailSettingsHandler) authorizeScope(c *gin.Context, input RuleDraftInput) error {
	userValue, _ := c.Get(admin.CtxKeyUserID)
	userID, _ := userValue.(uuid.UUID)
	superValue, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	superAdmin, _ := superValue.(bool)
	return h.authorizer.Authorize(c.Request.Context(), userID, superAdmin, input)
}

func (h *AlarmEmailSettingsHandler) writeAuthorizationError(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	status := http.StatusBadRequest
	if errors.Is(err, ErrRuleScopeDenied) {
		status = http.StatusForbidden
	} else {
		status = http.StatusInternalServerError
	}
	commonerrors.AbortWithError(c, status, err)
	return false
}

func requireAlarmEmailDefaultsAdmin(c *gin.Context) bool {
	superValue, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	superAdmin, _ := superValue.(bool)
	if superAdmin {
		return true
	}
	commonerrors.AbortWithError(c, http.StatusForbidden, errors.New("alarm email default recipients require super administrator permission"))
	return false
}
