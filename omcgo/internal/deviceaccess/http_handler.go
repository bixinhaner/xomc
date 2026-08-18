package deviceaccess

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type ActorResolver interface {
	ResolveActor(ctx *gin.Context) (PolicyActor, error)
}

type PolicyHTTPHandler struct {
	policies               *PolicyService
	lists                  *ListService
	actors                 ActorResolver
	actions                *ActionService
	actionMutationsEnabled bool
	settings               RuntimeSettingsStore
	manage                 ManagementStore
}

func (h *PolicyHTTPHandler) SetActionReader(actions *ActionService) {
	h.actions = actions
	h.actionMutationsEnabled = false
}

func (h *PolicyHTTPHandler) SetActionService(actions *ActionService) {
	h.actions = actions
	h.actionMutationsEnabled = actions != nil
}

func (h *PolicyHTTPHandler) SetManagementStore(store ManagementStore) {
	h.manage = store
}

func (h *PolicyHTTPHandler) SetRuntimeSettingsStore(store RuntimeSettingsStore) {
	h.settings = store
}

func NewPolicyHTTPHandler(policies *PolicyService, lists *ListService, actors ActorResolver) *PolicyHTTPHandler {
	return &PolicyHTTPHandler{policies: policies, lists: lists, actors: actors}
}

func (h *PolicyHTTPHandler) RegisterRoutes(group *gin.RouterGroup) {
	access := group.Group("/device-access")
	access.POST("/policies/drafts", h.CreateDraft)
	access.GET("/policies/:versionID", h.GetPolicyVersion)
	access.DELETE("/policies/:versionID", h.DeleteDraft)
	access.POST("/policies/:versionID/publish", h.Publish)
	access.POST("/devices/:serialNumber/reevaluate", h.ReevaluateOne)
	access.POST("/access-list", h.UpsertList)
	if h.settings != nil {
		access.GET("/settings", h.GetRuntimeSettings)
		access.PUT("/settings", h.UpdateRuntimeSettings)
	}
	if h.manage != nil {
		access.GET("/states", h.ListStates)
		access.GET("/states/:serialNumber", h.GetStateDetail)
		access.GET("/policies", h.ListPolicyVersions)
		access.GET("/access-list", h.ListAccessEntries)
		access.GET("/candidates", h.ListCandidates)
		access.POST("/candidates/:candidateID/review", h.ReviewCandidate)
	}
	if h.actions != nil {
		access.GET("/actions", h.ListActions)
	}
	if h.actionMutationsEnabled {
		access.POST("/actions/:actionID/retry", h.RetryAction)
	}
}

type updateRuntimeSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *PolicyHTTPHandler) GetRuntimeSettings(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	settings, err := h.settings.GetRuntimeSettings(c.Request.Context(), actor.Carrier)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, settings)
}

func (h *PolicyHTTPHandler) UpdateRuntimeSettings(c *gin.Context) {
	var request updateRuntimeSettingsRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	settings, err := h.settings.UpdateRuntimeSettings(
		c.Request.Context(), actor.Carrier, request.Enabled, actor.SubjectID,
	)
	auditAction := "runtime_switch_disable"
	if request.Enabled {
		auditAction = "runtime_switch_enable"
	}
	h.logAudit(c, actor, auditAction, actor.Carrier, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, settings)
}

func (h *PolicyHTTPHandler) GetPolicyVersion(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	version, err := h.policies.GetVersion(c.Request.Context(), actor, c.Param("versionID"))
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, version)
}

func (h *PolicyHTTPHandler) DeleteDraft(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	err := h.policies.DeleteDraft(c.Request.Context(), actor, c.Param("versionID"))
	h.logAudit(c, actor, "policy_draft_delete", c.Param("versionID"), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, nil)
}

type createDraftRequest struct {
	Name   string         `json:"name" binding:"required"`
	Policy CompiledPolicy `json:"policy" binding:"required"`
	// Carrier is intentionally ignored. The actor scope is authoritative.
	Carrier string `json:"carrier,omitempty"`
}

type listMutationRequest struct {
	Entry CompiledListEntry `json:"entry" binding:"required"`
}

type candidateReviewRequest struct {
	Outcome ListEntryType `json:"outcome" binding:"required"`
	Reason  string        `json:"reason" binding:"required"`
}

func (h *PolicyHTTPHandler) CreateDraft(c *gin.Context) {
	var request createDraftRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	version, err := h.policies.CreateDraft(c.Request.Context(), actor, request.Name, request.Policy)
	h.logAudit(c, actor, "policy_draft_create", request.Name, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, version)
}

func (h *PolicyHTTPHandler) Publish(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	version, err := h.policies.Publish(c.Request.Context(), actor, c.Param("versionID"))
	h.logAudit(c, actor, "policy_publish", c.Param("versionID"), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, version)
}

func (h *PolicyHTTPHandler) ReevaluateOne(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	serialNumber := strings.TrimSpace(c.Param("serialNumber"))
	if h.settings != nil {
		settings, err := h.settings.GetRuntimeSettings(c.Request.Context(), actor.Carrier)
		if err != nil {
			h.logAudit(c, actor, "device_access_reevaluate", serialNumber, err)
			writePolicyError(c, err)
			return
		}
		if !settings.Enabled {
			h.logAudit(c, actor, "device_access_reevaluate", serialNumber, ErrAccessControlDisabled)
			writePolicyError(c, ErrAccessControlDisabled)
			return
		}
	}
	err := h.policies.ReevaluateOne(c.Request.Context(), actor, serialNumber)
	h.logAudit(c, actor, "device_access_reevaluate", serialNumber, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func (h *PolicyHTTPHandler) UpsertList(c *gin.Context) {
	var request listMutationRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	err := h.lists.UpsertAndReevaluate(c.Request.Context(), actor, request.Entry)
	h.logAudit(c, actor, "access_list_upsert", request.Entry.IdentityValue, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func (h *PolicyHTTPHandler) ListActions(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.actions.List(c.Request.Context(), ActionListFilter{
		Carrier: actor.Carrier, SerialNumber: c.Query("serial_number"),
		Status: ActionStatus(c.Query("status")), Page: page, PageSize: pageSize,
		VisibleGroups: actor.VisibleGroups,
	})
	if err != nil {
		writePolicyError(c, err)
		return
	}
	page, pageSize = normalizePage(page, pageSize)
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *PolicyHTTPHandler) RetryAction(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	actionID, err := uuid.Parse(c.Param("actionID"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	err = h.actions.RetryForActor(c.Request.Context(), actor, actionID)
	h.logAudit(c, actor, "device_access_action_retry", actionID.String(), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func (h *PolicyHTTPHandler) ListStates(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.State = AccessState(c.Query("state"))
	items, total, err := h.manage.ListStates(c.Request.Context(), filter)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{
		"items": items, "total": total, "page": filter.Page, "page_size": filter.PageSize,
	})
}

func (h *PolicyHTTPHandler) GetStateDetail(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := ManagementFilter{
		Carrier: actor.Carrier, SerialNumber: strings.TrimSpace(c.Param("serialNumber")),
		VisibleGroups: actor.VisibleGroups,
	}
	detail, err := h.manage.GetDetail(c.Request.Context(), filter)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *PolicyHTTPHandler) ListPolicyVersions(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	items, total, err := h.manage.ListPolicyVersions(c.Request.Context(), filter)
	writeManagementList(c, items, total, filter, err)
}

func (h *PolicyHTTPHandler) ListAccessEntries(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.EntryType = ListEntryType(c.Query("entry_type"))
	items, total, err := h.manage.ListEntries(c.Request.Context(), filter)
	writeManagementList(c, items, total, filter, err)
}

func (h *PolicyHTTPHandler) ListCandidates(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.Status = c.Query("review_status")
	items, total, err := h.manage.ListCandidates(c.Request.Context(), filter)
	writeManagementList(c, items, total, filter, err)
}

func (h *PolicyHTTPHandler) ReviewCandidate(c *gin.Context) {
	var request candidateReviewRequest
	if !bindJSON(c, &request) {
		return
	}
	if strings.TrimSpace(request.Reason) == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	reviewerID, err := uuid.Parse(actor.SubjectID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return
	}
	candidateID, err := uuid.Parse(c.Param("candidateID"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	err = h.manage.ReviewCandidate(c.Request.Context(), actor.Carrier, candidateID, reviewerID, actor.VisibleGroups, request.Outcome, request.Reason)
	h.logAudit(c, actor, "device_access_candidate_review", candidateID.String(), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func managementFilter(c *gin.Context, actor PolicyActor) ManagementFilter {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = normalizePage(page, pageSize)
	return ManagementFilter{
		Carrier: actor.Carrier, SerialNumber: c.Query("serial_number"), Page: page, PageSize: pageSize,
		VisibleGroups: actor.VisibleGroups,
	}
}

func writeManagementList(c *gin.Context, items any, total int64, filter ManagementFilter, err error) {
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": filter.Page, "page_size": filter.PageSize})
}

func (h *PolicyHTTPHandler) logAudit(c *gin.Context, actor PolicyActor, action, resourceID string, err error) {
	var userID *uuid.UUID
	if parsed, parseErr := uuid.Parse(actor.SubjectID); parseErr == nil {
		userID = &parsed
	}
	audit.Log(context.WithoutCancel(c.Request.Context()), audit.Entry{
		UserID:       userID,
		Action:       audit.ActionConfig,
		ResourceType: "device_access",
		ResourceID:   resourceID,
		Details: map[string]interface{}{
			"operation": action,
			"carrier":   actor.Carrier,
		},
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Success:   err == nil,
		ErrorMessage: func() string {
			if err == nil {
				return ""
			}
			return err.Error()
		}(),
	})
}

func (h *PolicyHTTPHandler) actor(c *gin.Context) (PolicyActor, bool) {
	if h == nil || h.actors == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, commonerrors.ErrInternal)
		return PolicyActor{}, false
	}
	actor, err := h.actors.ResolveActor(c)
	if err != nil {
		status := http.StatusUnauthorized
		switch {
		case errors.Is(err, commonerrors.ErrForbidden):
			status = http.StatusForbidden
		case errors.Is(err, commonerrors.ErrInternal):
			status = http.StatusInternalServerError
		case errors.Is(err, ErrAccessGateDependencyMissing):
			status = http.StatusServiceUnavailable
		}
		commonerrors.AbortWithError(c, status, err)
		return PolicyActor{}, false
	}
	return actor, true
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return false
	}
	return true
}

func writePolicyError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrPolicyCarrierScope):
		status = http.StatusForbidden
	case errors.Is(err, commonerrors.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, ErrPolicyVersionImmutable):
		status = http.StatusConflict
	case errors.Is(err, ErrAssetOwnershipConflict):
		status = http.StatusConflict
	case errors.Is(err, ErrActionNotRetryable), errors.Is(err, ErrActionRetryExhausted), errors.Is(err, ErrActionStateChanged):
		status = http.StatusConflict
	case errors.Is(err, ErrAccessControlDisabled):
		status = http.StatusConflict
	case errors.Is(err, ErrAccessGateDependencyMissing):
		status = http.StatusServiceUnavailable
	case errors.Is(err, pgx.ErrNoRows):
		status = http.StatusNotFound
	case errors.Is(err, ErrCarrierRequired), errors.Is(err, ErrSerialNumberRequired):
		status = http.StatusBadRequest
	case errors.Is(err, ErrInvalidAccessInput):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		commonerrors.AbortWithError(c, status, commonerrors.ErrInternal)
		return
	}
	commonerrors.AbortWithError(c, status, err)
}
