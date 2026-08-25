package deviceaccess

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	imports                ImportPreviewService
	importMutations        ImportBatchMutationService
	importReads            ImportBatchReadService
	ruleDimensions         RuleDimensionGovernanceService
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

func (h *PolicyHTTPHandler) SetImportPreviewService(service ImportPreviewService) {
	h.imports = service
}

func (h *PolicyHTTPHandler) SetImportBatchMutationService(service ImportBatchMutationService) {
	h.importMutations = service
}

func (h *PolicyHTTPHandler) SetImportBatchReadService(service ImportBatchReadService) {
	h.importReads = service
}

func (h *PolicyHTTPHandler) SetRuleDimensionGovernanceService(service RuleDimensionGovernanceService) {
	h.ruleDimensions = service
}

func NewPolicyHTTPHandler(policies *PolicyService, lists *ListService, actors ActorResolver) *PolicyHTTPHandler {
	return &PolicyHTTPHandler{policies: policies, lists: lists, actors: actors}
}

func (h *PolicyHTTPHandler) RegisterRoutes(group *gin.RouterGroup) {
	access := group.Group("/device-access")
	access.POST("/policies/drafts", h.CreateDraft)
	access.GET("/policies/:versionID", h.GetPolicyVersion)
	access.GET("/policies/:versionID/difference", h.GetPolicyDifference)
	access.PUT("/policies/:versionID", h.UpdateDraft)
	access.DELETE("/policies/:versionID", h.DeleteDraft)
	access.POST("/policies/:versionID/publish", h.Publish)
	access.POST("/policies/:versionID/rollback", h.RollbackPolicy)
	access.POST("/devices/:serialNumber/reevaluate", h.ReevaluateOne)
	access.POST("/access-list", h.UpsertList)
	access.POST("/access-list/batch-disable", h.DisableListBatch)
	access.GET("/access-list/template", h.DownloadListTemplate)
	access.GET("/import-templates/:importType", h.DownloadImportTemplate)
	if h.imports != nil {
		access.POST("/imports/preview", h.PreviewImport)
	}
	if h.importMutations != nil {
		access.POST("/imports/:batchID/commit", h.CommitImport)
		access.POST("/imports/:batchID/rollback", h.RollbackImport)
	}
	if h.importReads != nil {
		access.GET("/imports", h.ListImportBatches)
		access.GET("/imports/:batchID", h.GetImportBatch)
		access.GET("/imports/:batchID/errors", h.ListImportErrors)
	}
	if h.ruleDimensions != nil {
		access.GET("/policies/:versionID/rules/:ruleID/dimensions/:dimension/export", h.ExportRuleDimension)
		access.DELETE("/policies/:versionID/rules/:ruleID/dimensions/:dimension", h.ClearRuleDimension)
	}
	if h.settings != nil {
		access.GET("/settings", h.GetRuntimeSettings)
		access.PUT("/settings", h.UpdateRuntimeSettings)
	}
	if h.manage != nil {
		access.GET("/states", h.ListStates)
		access.GET("/states/summary", h.SummarizeStates)
		access.GET("/states/:serialNumber/decisions", h.ListDecisions)
		access.GET("/states/:serialNumber/identity-snapshots", h.ListIdentitySnapshots)
		access.GET("/states/:serialNumber/evidence", h.ListEvidence)
		access.GET("/states/:serialNumber/notifications", h.ListNotifications)
		access.GET("/states/:serialNumber/manual-operations", h.ListManualOperations)
		access.GET("/states/:serialNumber", h.GetStateDetail)
		access.POST("/decisions/:decisionID/archive", h.ArchiveDecision)
		access.POST("/decisions/:decisionID/restore", h.RestoreDecision)
		access.GET("/policies", h.ListPolicyVersions)
		access.GET("/access-list", h.ListAccessEntries)
		access.GET("/candidates", h.ListCandidates)
		access.POST("/candidates/:candidateID/review", h.ReviewCandidate)
	}
	if h.actions != nil {
		access.GET("/actions", h.ListActions)
		access.GET("/actions/:actionID/attempts", h.ListActionAttempts)
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

func (h *PolicyHTTPHandler) GetPolicyDifference(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	difference, err := h.policies.Difference(c.Request.Context(), actor, c.Param("versionID"), c.Query("base_version_id"))
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, difference)
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

type updateDraftRequest struct {
	Policy CompiledPolicy `json:"policy" binding:"required"`
}

type listMutationRequest struct {
	Entry   *CompiledListEntry  `json:"entry,omitempty"`
	Entries []CompiledListEntry `json:"entries,omitempty"`
}

type candidateReviewRequest struct {
	Outcome ListEntryType `json:"outcome" binding:"required"`
	Reason  string        `json:"reason" binding:"required"`
}

type listBatchDisableRequest struct {
	EntryType     ListEntryType `json:"entry_type" binding:"required"`
	SerialNumbers []string      `json:"serial_numbers" binding:"required"`
	Reason        string        `json:"reason" binding:"required"`
	Carrier       string        `json:"carrier,omitempty"`
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

func (h *PolicyHTTPHandler) UpdateDraft(c *gin.Context) {
	var request updateDraftRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	version, err := h.policies.UpdateDraft(c.Request.Context(), actor, c.Param("versionID"), request.Policy)
	h.logAudit(c, actor, "policy_draft_update", c.Param("versionID"), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, version)
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

func (h *PolicyHTTPHandler) RollbackPolicy(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	version, err := h.policies.Rollback(c.Request.Context(), actor, c.Param("versionID"))
	h.logAudit(c, actor, "policy_rollback", c.Param("versionID"), err)
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
	entries := request.Entries
	if request.Entry != nil {
		if len(entries) > 0 {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		entries = []CompiledListEntry{*request.Entry}
	}
	if len(entries) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var err error
	if len(entries) == 1 {
		err = h.lists.UpsertAndReevaluate(c.Request.Context(), actor, entries[0])
	} else {
		err = h.lists.UpsertManyAndReevaluate(c.Request.Context(), actor, entries)
	}
	resourceID := entries[0].IdentityValue
	if len(entries) > 1 {
		resourceID = fmt.Sprintf("batch:%d", len(entries))
	}
	auditReason := strings.TrimSpace(entries[0].Reason)
	if auditReason == "" {
		auditReason = "manual_list_change"
	}
	auditDetails := map[string]interface{}{"reason": auditReason}
	if len(entries) > 1 {
		auditDetails["entry_count"] = len(entries)
		serialNumbers := make([]string, 0, len(entries))
		for _, entry := range entries {
			serialNumbers = append(serialNumbers, entry.IdentityValue)
		}
		auditDetails["serial_numbers"] = serialNumbers
	}
	h.logAuditWithDetails(c, actor, "access_list_upsert", resourceID, auditDetails, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func (h *PolicyHTTPHandler) DisableListBatch(c *gin.Context) {
	var request listBatchDisableRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	err := h.lists.DisableManyAndReevaluate(
		c.Request.Context(), actor, request.EntryType, request.SerialNumbers, request.Reason,
	)
	h.logAuditWithDetails(c, actor, "access_list_batch_disable", fmt.Sprintf("batch:%d", len(request.SerialNumbers)), map[string]interface{}{
		"reason": strings.TrimSpace(request.Reason), "entry_count": len(request.SerialNumbers),
		"serial_numbers": request.SerialNumbers,
	}, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func (h *PolicyHTTPHandler) DownloadListTemplate(c *gin.Context) {
	if _, ok := h.actor(c); !ok {
		return
	}
	content, err := GenerateAccessListXLSXTemplate()
	if err != nil {
		writePolicyError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="device-access-list-template.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content)
}

const accessListCSVTemplate = "\ufeffSerial Number,List Type,Reason,Valid From,Valid Until\r\n"

func (h *PolicyHTTPHandler) DownloadImportTemplate(c *gin.Context) {
	if _, ok := h.actor(c); !ok {
		return
	}
	switch ImportType(c.Param("importType")) {
	case ImportTypeAccessList:
		content, err := GenerateAccessListXLSXTemplate()
		if err != nil {
			writePolicyError(c, err)
			return
		}
		c.Header("Content-Disposition", `attachment; filename="device-access-list-template.xlsx"`)
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content)
	case ImportTypeRuleDimension:
		dimension := ImportDimension(c.Query("dimension"))
		content, err := GenerateRuleDimensionXLSXTemplate(dimension)
		if err != nil {
			writePolicyError(c, err)
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="device-access-%s-template.xlsx"`, dimension))
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content)
	default:
		writePolicyError(c, fmt.Errorf("%w: unsupported import type", ErrImportTemplateInvalid))
	}
}

func (h *PolicyHTTPHandler) PreviewImport(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(defaultImportMaxBytes)+64*1024)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("read import file: %w", err))
		return
	}
	defer file.Close()
	if header.Size > defaultImportMaxBytes {
		writePolicyError(c, ErrImportFileTooLarge)
		return
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(defaultImportMaxBytes)+1))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("read import file content: %w", err))
		return
	}
	if len(content) > defaultImportMaxBytes {
		writePolicyError(c, ErrImportFileTooLarge)
		return
	}
	request := ImportPreviewRequest{
		Type: ImportType(c.PostForm("import_type")), EntryType: ListEntryType(c.PostForm("entry_type")),
		Mode: ImportMode(c.PostForm("mode")), FailurePolicy: ImportFailurePolicy(c.PostForm("failure_policy")),
		SourceFilename: filepath.Base(strings.TrimSpace(header.Filename)), Content: content,
		IdempotencyKey:        strings.TrimSpace(c.GetHeader("Idempotency-Key")),
		TargetPolicyVersionID: strings.TrimSpace(c.PostForm("target_policy_version_id")),
		TargetRuleID:          strings.TrimSpace(c.PostForm("target_rule_id")), Dimension: ImportDimension(c.PostForm("dimension")),
	}
	if request.Type == ImportTypeRuleDimension {
		result, previewErr := h.imports.PreviewRuleDimension(c.Request.Context(), actor, request)
		h.logAudit(c, actor, "device_access_rule_dimension_import_preview", result.Batch.ID, previewErr)
		if previewErr != nil {
			writePolicyError(c, previewErr)
			return
		}
		response.OK(c, result)
		return
	}
	result, err := h.imports.PreviewAccessList(c.Request.Context(), actor, request)
	h.logAudit(c, actor, "device_access_import_preview", result.Batch.ID, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *PolicyHTTPHandler) ExportRuleDimension(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	dimension := ImportDimension(c.Param("dimension"))
	content, err := h.ruleDimensions.ExportRuleDimension(c.Request.Context(), actor, c.Param("versionID"), c.Param("ruleID"), dimension)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="device-access-%s.csv"`, dimension))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", content)
}

func (h *PolicyHTTPHandler) ClearRuleDimension(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	dimension := ImportDimension(c.Param("dimension"))
	batch, err := h.ruleDimensions.ClearRuleDimension(c.Request.Context(), actor, c.Param("versionID"), c.Param("ruleID"), dimension)
	h.logAudit(c, actor, "device_access_rule_dimension_clear", c.Param("ruleID")+":"+string(dimension), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, batch)
}

func (h *PolicyHTTPHandler) ListImportBatches(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = normalizePage(page, pageSize)
	items, total, err := h.importReads.ListImportBatches(c.Request.Context(), actor, ImportBatchListFilter{
		Type: ImportType(c.Query("import_type")), TargetPolicyVersionID: c.Query("target_policy_version_id"),
		TargetRuleID: c.Query("target_rule_id"), Dimension: ImportDimension(c.Query("dimension")), Page: page, PageSize: pageSize,
	})
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *PolicyHTTPHandler) GetImportBatch(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	result, err := h.importReads.GetImportBatch(c.Request.Context(), actor, c.Param("batchID"))
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *PolicyHTTPHandler) ListImportErrors(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	rows, err := h.importReads.ListImportErrors(c.Request.Context(), actor, c.Param("batchID"))
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{"rows": rows})
}

func (h *PolicyHTTPHandler) CommitImport(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	var options ImportCommitOptions
	if c.Request.ContentLength != 0 {
		if !bindJSON(c, &options) {
			return
		}
	}
	var result ImportBatch
	var err error
	if confirmed, ok := h.importMutations.(ConfirmedImportBatchMutationService); ok {
		result, err = confirmed.CommitImportConfirmed(c.Request.Context(), actor, c.Param("batchID"), options)
	} else {
		result, err = h.importMutations.CommitImport(c.Request.Context(), actor, c.Param("batchID"))
	}
	h.logAudit(c, actor, "device_access_import_commit", c.Param("batchID"), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *PolicyHTTPHandler) RollbackImport(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	result, err := h.importMutations.RollbackImport(c.Request.Context(), actor, c.Param("batchID"))
	h.logAudit(c, actor, "device_access_import_rollback", c.Param("batchID"), err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *PolicyHTTPHandler) ListActions(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	filter := ActionListFilter{
		Carrier: actor.Carrier, SerialNumber: c.Query("serial_number"),
		Status: ActionStatus(c.Query("status")), Page: page, PageSize: pageSize,
		VisibleGroups: actor.VisibleGroups,
	}
	if err := applyActionAuditQuery(c, &filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, total, err := h.actions.List(c.Request.Context(), filter)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	page, pageSize = normalizePage(page, pageSize)
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *PolicyHTTPHandler) RetryAction(c *gin.Context) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	actionID, err := uuid.Parse(c.Param("actionID"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	err = h.actions.RetryForActor(c.Request.Context(), actor, actionID, request.Reason)
	h.logAuditWithDetails(c, actor, "device_access_action_retry", actionID.String(), map[string]interface{}{
		"reason": strings.TrimSpace(request.Reason),
	}, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, nil)
}

func (h *PolicyHTTPHandler) ListActionAttempts(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	actionID, err := uuid.Parse(c.Param("actionID"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, err := h.actions.ListAttemptsForActor(c.Request.Context(), actor, actionID)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *PolicyHTTPHandler) ListStates(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.State = AccessState(c.Query("state"))
	if err := applyManagementAuditQuery(c, &filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, total, err := h.manage.ListStates(c.Request.Context(), filter)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{
		"items": items, "total": total, "page": filter.Page, "page_size": filter.PageSize,
	})
}

func (h *PolicyHTTPHandler) SummarizeStates(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.State = AccessState(c.Query("state"))
	if err := applyManagementAuditQuery(c, &filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	summary, err := h.manage.SummarizeStates(c.Request.Context(), filter)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, summary)
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

func (h *PolicyHTTPHandler) ListDecisions(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.SerialNumber = strings.TrimSpace(c.Param("serialNumber"))
	if err := applyManagementAuditQuery(c, &filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, total, err := h.manage.ListDecisions(c.Request.Context(), filter)
	writeManagementList(c, items, total, filter, err)
}

func (h *PolicyHTTPHandler) ListIdentitySnapshots(c *gin.Context) {
	h.listStateDetailSection(c, func(ctx context.Context, filter ManagementFilter) (any, int64, error) {
		items, total, err := h.manage.ListIdentitySnapshots(ctx, filter)
		return items, total, err
	})
}

func (h *PolicyHTTPHandler) ListEvidence(c *gin.Context) {
	h.listStateDetailSection(c, func(ctx context.Context, filter ManagementFilter) (any, int64, error) {
		items, total, err := h.manage.ListEvidence(ctx, filter)
		return items, total, err
	})
}

func (h *PolicyHTTPHandler) ListNotifications(c *gin.Context) {
	h.listStateDetailSection(c, func(ctx context.Context, filter ManagementFilter) (any, int64, error) {
		items, total, err := h.manage.ListNotifications(ctx, filter)
		return items, total, err
	})
}

func (h *PolicyHTTPHandler) ListManualOperations(c *gin.Context) {
	h.listStateDetailSection(c, func(ctx context.Context, filter ManagementFilter) (any, int64, error) {
		items, total, err := h.manage.ListManualOperations(ctx, filter)
		return items, total, err
	})
}

func (h *PolicyHTTPHandler) listStateDetailSection(
	c *gin.Context,
	load func(context.Context, ManagementFilter) (any, int64, error),
) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := managementFilter(c, actor)
	filter.SerialNumber = strings.TrimSpace(c.Param("serialNumber"))
	if err := applyManagementAuditQuery(c, &filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, total, err := load(c.Request.Context(), filter)
	writeManagementList(c, items, total, filter, err)
}

func (h *PolicyHTTPHandler) ArchiveDecision(c *gin.Context) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if !bindJSON(c, &request) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	decisionID, actorID, ok := parseDecisionMutationIdentity(c, actor)
	if !ok {
		return
	}
	err := h.manage.ArchiveDecision(
		c.Request.Context(), actor.Carrier, decisionID, actorID, actor.VisibleGroups, request.Reason,
	)
	h.logAuditWithDetails(c, actor, "device_access_decision_archive", decisionID.String(), map[string]interface{}{
		"reason": strings.TrimSpace(request.Reason),
	}, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *PolicyHTTPHandler) RestoreDecision(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	decisionID, actorID, ok := parseDecisionMutationIdentity(c, actor)
	if !ok {
		return
	}
	err := h.manage.RestoreDecision(
		c.Request.Context(), actor.Carrier, decisionID, actorID, actor.VisibleGroups,
	)
	h.logAuditWithDetails(c, actor, "device_access_decision_restore", decisionID.String(), map[string]interface{}{
		"reason": "manual_restore",
	}, err)
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, nil)
}

func parseDecisionMutationIdentity(c *gin.Context, actor PolicyActor) (uuid.UUID, uuid.UUID, bool) {
	decisionID, err := uuid.Parse(c.Param("decisionID"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return uuid.Nil, uuid.Nil, false
	}
	actorID, err := uuid.Parse(actor.SubjectID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return uuid.Nil, uuid.Nil, false
	}
	return decisionID, actorID, true
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
	if rawID := strings.TrimSpace(c.Query("candidate_id")); rawID != "" {
		candidateID, err := uuid.Parse(rawID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.CandidateID = &candidateID
	}
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
	h.logAuditWithDetails(c, actor, "device_access_candidate_review", candidateID.String(), map[string]interface{}{
		"outcome": request.Outcome, "reason": strings.TrimSpace(request.Reason),
	}, err)
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
		Carrier: actor.Carrier, SerialNumber: c.Query("serial_number"), ProductName: c.Query("product_name"),
		State: AccessState(c.Query("state")), Status: c.Query("status"), EntryType: ListEntryType(c.Query("entry_type")),
		Page: page, PageSize: pageSize,
		VisibleGroups: actor.VisibleGroups,
	}
}

func applyManagementAuditQuery(c *gin.Context, filter *ManagementFilter) error {
	filter.Decision = EffectiveAction(strings.TrimSpace(c.Query("decision")))
	filter.ReasonCode = ReasonCode(strings.TrimSpace(c.Query("reason_code")))
	filter.ActionStatus = ActionStatus(strings.TrimSpace(c.Query("action_status")))
	filter.ArchiveStatus = strings.TrimSpace(c.Query("archive_status"))
	dimension := strings.TrimSpace(c.Query("dimension"))
	switch dimension {
	case "sn":
		dimension = string(ConditionTypeIdentity)
	case "ip":
		dimension = string(ConditionTypeObservedIP)
	}
	filter.Dimension = ConditionType(dimension)
	var err error
	filter.PolicyVersionID, err = parseOptionalUUIDQuery(c.Query("policy_version_id"))
	if err != nil {
		return err
	}
	filter.MatchedRuleID, err = parseOptionalUUIDQuery(c.Query("matched_rule_id"))
	if err != nil {
		return err
	}
	filter.StartedAt, err = parseOptionalTimeQuery(c.Query("started_at"))
	if err != nil {
		return err
	}
	filter.EndedAt, err = parseOptionalTimeQuery(c.Query("ended_at"))
	return err
}

func applyActionAuditQuery(c *gin.Context, filter *ActionListFilter) error {
	filter.Decision = EffectiveAction(strings.TrimSpace(c.Query("decision")))
	filter.ReasonCode = ReasonCode(strings.TrimSpace(c.Query("reason_code")))
	var err error
	filter.PolicyVersionID, err = parseOptionalUUIDQuery(c.Query("policy_version_id"))
	if err != nil {
		return err
	}
	filter.StartedAt, err = parseOptionalTimeQuery(c.Query("started_at"))
	if err != nil {
		return err
	}
	filter.EndedAt, err = parseOptionalTimeQuery(c.Query("ended_at"))
	return err
}

func parseOptionalUUIDQuery(raw string) (*uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse UUID query: %w", err)
	}
	return &parsed, nil
}

func parseOptionalTimeQuery(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("parse time query: %w", err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func writeManagementList(c *gin.Context, items any, total int64, filter ManagementFilter, err error) {
	if err != nil {
		writePolicyError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": filter.Page, "page_size": filter.PageSize})
}

func (h *PolicyHTTPHandler) logAudit(c *gin.Context, actor PolicyActor, action, resourceID string, err error) {
	h.logAuditWithDetails(c, actor, action, resourceID, nil, err)
}

func (h *PolicyHTTPHandler) logAuditWithDetails(
	c *gin.Context,
	actor PolicyActor,
	action, resourceID string,
	details map[string]interface{},
	err error,
) {
	var userID *uuid.UUID
	if parsed, parseErr := uuid.Parse(actor.SubjectID); parseErr == nil {
		userID = &parsed
	}
	auditDetails := map[string]interface{}{
		"operation": action,
		"carrier":   actor.Carrier,
	}
	for key, value := range details {
		auditDetails[key] = value
	}
	audit.Log(context.WithoutCancel(c.Request.Context()), audit.Entry{
		UserID:       userID,
		Username:     strings.TrimSpace(actor.Username),
		Action:       audit.ActionConfig,
		ResourceType: "device_access",
		ResourceID:   resourceID,
		Details:      auditDetails,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Success:      err == nil,
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
	case errors.Is(err, ErrListBatchConflict):
		status = http.StatusConflict
	case errors.Is(err, ErrAssetOwnershipConflict):
		status = http.StatusConflict
	case errors.Is(err, ErrActionNotRetryable), errors.Is(err, ErrActionRetryExhausted), errors.Is(err, ErrActionStateChanged):
		status = http.StatusConflict
	case errors.Is(err, ErrCurrentDecisionArchive), errors.Is(err, ErrDecisionArchiveConflict):
		status = http.StatusConflict
	case errors.Is(err, ErrAccessControlDisabled):
		status = http.StatusConflict
	case errors.Is(err, ErrImportFileTooLarge):
		status = http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrImportTemplateInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, ErrImportBatchConflict):
		status = http.StatusConflict
	case errors.Is(err, ErrImportReplaceConfirmationRequired):
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
