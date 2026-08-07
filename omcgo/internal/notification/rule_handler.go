package notification

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/response"
)

type RuleHandler struct {
	service    *RuleService
	authorizer *RuleScopeAuthorizer
}

func NewRuleHandler(service *RuleService) *RuleHandler { return &RuleHandler{service: service} }

func (h *RuleHandler) SetScopeAuthorizer(authorizer *RuleScopeAuthorizer) { h.authorizer = authorizer }

func (h *RuleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rules := rg.Group("/notification-rules")
	rules.GET("", h.List)
	rules.POST("", h.Create)
	rules.GET("/:id", h.Get)
	rules.PATCH("/:id", h.UpdateDraft)
	rules.POST("/:id/publish", h.Publish)
	rules.POST("/:id/enable", h.Enable)
	rules.POST("/:id/disable", h.Disable)
	rules.POST("/:id/archive", h.Archive)
	rules.POST("/:id/preview", h.Preview)
}

func (h *RuleHandler) List(c *gin.Context) {
	rules, err := h.service.List(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, rules)
}

func (h *RuleHandler) Get(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	rule, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, rule.Revision)
	response.OK(c, rule)
}

func (h *RuleHandler) Create(c *gin.Context) {
	var input RuleDraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if !h.authorizeInput(c, input) {
		return
	}
	rule, err := h.service.Create(c.Request.Context(), input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, rule.Revision)
	response.OKWithStatus(c, http.StatusCreated, rule)
}

func (h *RuleHandler) UpdateDraft(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	var input RuleDraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if !h.authorizeInput(c, input) {
		return
	}
	rule, err := h.service.UpdateDraft(c.Request.Context(), id, revision, input, requestActor(c))
	writeRuleMutation(c, rule, err)
}

func (h *RuleHandler) Publish(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	rule, err := h.service.Publish(c.Request.Context(), id, revision, requestActor(c))
	writeRuleMutation(c, rule, err)
}

func (h *RuleHandler) Enable(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	var request struct {
		RuleVersionID uuid.UUID `json:"rule_version_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	rule, err := h.service.Enable(c.Request.Context(), id, revision, request.RuleVersionID)
	writeRuleMutation(c, rule, err)
}

func (h *RuleHandler) Disable(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	rule, err := h.service.Disable(c.Request.Context(), id, revision)
	writeRuleMutation(c, rule, err)
}

func (h *RuleHandler) Archive(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	rule, err := h.service.Archive(c.Request.Context(), id, revision)
	writeRuleMutation(c, rule, err)
}

func (h *RuleHandler) Preview(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	var payload event.AlarmLifecyclePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	rule, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	if rule.Draft == nil {
		abortRuleError(c, ErrDraftUnavailable)
		return
	}
	preview := h.service.Preview(payload, []RuleCandidate{{
		ID: rule.ID, Priority: rule.Priority, Conditions: rule.Draft.MatchConditions,
	}})
	response.OK(c, preview)
}

func parseRuleID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return uuid.Nil, false
	}
	return id, true
}

func parseRuleMutation(c *gin.Context) (uuid.UUID, int64, bool) {
	id, ok := parseRuleID(c)
	if !ok {
		return uuid.Nil, 0, false
	}
	revision, err := parseIfMatchRevision(c.GetHeader("If-Match"))
	if errors.Is(err, ErrPreconditionRequired) {
		commonerrors.AbortWithError(c, http.StatusPreconditionRequired, err)
		return uuid.Nil, 0, false
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return uuid.Nil, 0, false
	}
	return id, revision, true
}

var ErrPreconditionRequired = errors.New("If-Match revision is required")

func parseIfMatchRevision(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ErrPreconditionRequired
	}
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, `"`)
	revision, err := strconv.ParseInt(value, 10, 64)
	if err != nil || revision < 1 {
		return 0, commonerrors.ErrInvalidInput
	}
	return revision, nil
}

func writeRuleMutation(c *gin.Context, rule *NotificationRule, err error) {
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, rule.Revision)
	response.OK(c, rule)
}

func abortRuleError(c *gin.Context, err error) {
	status := commonerrors.HTTPStatusFromError(err)
	switch {
	case errors.Is(err, ErrRevisionMismatch):
		status = http.StatusPreconditionFailed
	case errors.Is(err, ErrDraftUnavailable), errors.Is(err, ErrVersionUnpublished),
		errors.Is(err, ErrRuleIncomplete), errors.Is(err, ErrChannelDisabled):
		status = http.StatusConflict
	}
	commonerrors.AbortWithError(c, status, err)
}

func setRevisionETag(c *gin.Context, revision int64) {
	c.Header("ETag", fmt.Sprintf(`"%d"`, revision))
}

func requestActor(c *gin.Context) string {
	actor, _ := c.Get(admin.CtxKeyUsername)
	username, _ := actor.(string)
	return username
}

func (h *RuleHandler) authorizeInput(c *gin.Context, input RuleDraftInput) bool {
	if h.authorizer == nil {
		return true
	}
	value, _ := c.Get(admin.CtxKeyUserID)
	userID, _ := value.(uuid.UUID)
	superAdmin, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuperAdmin, _ := superAdmin.(bool)
	if err := h.authorizer.Authorize(c.Request.Context(), userID, isSuperAdmin, input); err != nil {
		status := http.StatusForbidden
		if !errors.Is(err, ErrRuleScopeDenied) {
			status = http.StatusInternalServerError
		}
		commonerrors.AbortWithError(c, status, err)
		return false
	}
	return true
}
