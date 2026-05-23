// Package backup — restore HTTP handlers (T-0072).
//
// Methods extend the existing *Handler in handler.go; routes are registered
// on demand from RegisterRoutes when restoreService is wired (nil-safe).
package backup

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// SetRestoreService wires the RestoreService post-construction (DI hook,
// matches T-0071's SetPolicyService pattern).
func (h *Handler) SetRestoreService(s *RestoreService) {
	h.restoreService = s
}

// CreateRestore handles POST /api/v1/backup/restore.
//
// Body: {"bucket": "config_backup", "object_path": "backup/...xml.gz",
//        "target_device_sns": ["SN001", ...]}
//
// Returns 200 with the persisted RestoreTask. Validation failures map to 400;
// missing source object maps to 404; service unavailability (DI not wired)
// maps to 503.
func (h *Handler) CreateRestore(c *gin.Context) {
	if h.restoreService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("restore service not configured"))
		return
	}
	var req CreateRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	createdByStr := admin.UserIDStringFromCtx(c) // optional middleware-injected actor

	rt, err := h.restoreService.Create(c.Request.Context(), &req, createdByStr)
	if err != nil {
		switch {
		case errors.Is(err, commonerrors.ErrInvalidInput):
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		case errors.Is(err, commonerrors.ErrNotFound):
			commonerrors.AbortWithError(c, http.StatusNotFound, err)
		default:
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		}
		return
	}
	response.OK(c, rt)
}

// ListRestoreTasks handles GET /api/v1/backup/restore-tasks.
func (h *Handler) ListRestoreTasks(c *gin.Context) {
	if h.restoreService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("restore service not configured"))
		return
	}
	filter := RestoreFilter{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if statusStr := c.Query("status"); statusStr != "" {
		s := RestoreStatus(statusStr)
		filter.Status = &s
	}
	resp, err := h.restoreService.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, resp)
}

// CreateRestoreByTaskID handles POST /api/v1/backup/restore/by-task-id (T-0079).
//
// Body: {"backup_task_id": "<uuid>", "target_device_sns": ["SN001", ...]}
//
// Resolves backup_tasks.file_path for the given task and dispatches the same
// fan-out as POST /backup/restore. Multi-device source backups produce a
// human-readable warning in the response body alongside the created
// RestoreTask. Returns 404 when the task hasn't uploaded yet (file_path null).
func (h *Handler) CreateRestoreByTaskID(c *gin.Context) {
	if h.restoreService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("restore service not configured"))
		return
	}
	var req CreateByTaskIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	createdByStr := admin.UserIDStringFromCtx(c)

	result, err := h.restoreService.CreateByTaskID(c.Request.Context(), &req, createdByStr)
	if err != nil {
		switch {
		case errors.Is(err, commonerrors.ErrInvalidInput):
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		case errors.Is(err, commonerrors.ErrNotFound):
			commonerrors.AbortWithError(c, http.StatusNotFound, err)
		default:
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		}
		return
	}
	response.OK(c, result)
}

// CreateRestoreBySnapshot handles POST /api/v1/backup/restore/by-snapshot (T-0164 B5).
//
// Body: {"target_device_sns": ["SN001", "SN002", ...]}
//
// Each target device picks its own latest config_snapshots row as source.
// Integral rejection: if any SN has no snapshot, returns 404 with the list of
// missing SNs in the body so the operator can fix and retry.
func (h *Handler) CreateRestoreBySnapshot(c *gin.Context) {
	if h.restoreService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("restore service not configured"))
		return
	}
	var req CreateBySnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	createdByStr := admin.UserIDStringFromCtx(c)
	result, err := h.restoreService.CreateBySnapshot(c.Request.Context(), &req, createdByStr)
	if err != nil {
		switch {
		case errors.Is(err, commonerrors.ErrInvalidInput):
			// Service returns InvalidInput when by-snapshot mode isn't wired
			// OR when the request body has zero targets. result may carry
			// missing SNs if integral rejection happened.
			if result != nil && len(result.Missing) > 0 {
				response.FailWithData(c, http.StatusBadRequest, err.Error(), result)
				return
			}
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		case errors.Is(err, commonerrors.ErrNotFound):
			// Missing snapshot rows → 404 with body listing missing SNs.
			response.FailWithData(c, http.StatusNotFound, err.Error(), result)
		default:
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		}
		return
	}
	response.OK(c, result)
}

// GetRestoreTask handles GET /api/v1/backup/restore-tasks/:id.
func (h *Handler) GetRestoreTask(c *gin.Context) {
	if h.restoreService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("restore service not configured"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	rt, err := h.restoreService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			commonerrors.AbortWithError(c, http.StatusNotFound, err)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, rt)
}
