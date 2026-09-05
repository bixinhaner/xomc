package agentbridge

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type VisibleGroupsResolver interface {
	ResolveFromContext(*gin.Context) ([]uuid.UUID, error)
}

type FindingHandler struct {
	repo     *FindingRepository
	resolver VisibleGroupsResolver
}

func NewFindingHandler(repo *FindingRepository, resolver VisibleGroupsResolver) *FindingHandler {
	return &FindingHandler{repo: repo, resolver: resolver}
}

func (h *FindingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	findings := rg.Group("/agent/findings")
	findings.GET("", h.List)
	findings.GET("/:id", h.Get)
	findings.POST("/:id/read", h.Read)
	findings.POST("/:id/dismiss", h.Dismiss)
	findings.POST("/:id/restore", h.Restore)
	findings.POST("/:id/continue", h.Continue)
}

func (h *FindingHandler) List(c *gin.Context) {
	page, ok := positiveIntQuery(c, "page", 1, 100000)
	if !ok {
		return
	}
	pageSize, ok := positiveIntQuery(c, "page_size", 20, 100)
	if !ok {
		return
	}
	options, ok := h.options(c)
	if !ok {
		return
	}
	options.ResourceType = c.Query("resource_type")
	options.ResourceID = c.Query("resource_id")
	options.IncludeDismissed = c.Query("include_dismissed") == "true"
	options.Limit = pageSize
	options.Offset = (page - 1) * pageSize
	items, total, err := h.repo.List(c.Request.Context(), options)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func (h *FindingHandler) Get(c *gin.Context) {
	options, ok := h.options(c)
	if !ok {
		return
	}
	options.FindingID = c.Param("id")
	options.Limit = 1
	items, _, err := h.repo.List(c.Request.Context(), options)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if len(items) == 0 {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, items[0])
}

func (h *FindingHandler) Read(c *gin.Context)    { h.setState(c, "read") }
func (h *FindingHandler) Dismiss(c *gin.Context) { h.setState(c, "dismiss") }
func (h *FindingHandler) Restore(c *gin.Context) { h.setState(c, "restore") }

func (h *FindingHandler) setState(c *gin.Context, action string) {
	options, ok := h.options(c)
	if !ok {
		return
	}
	options.FindingID = c.Param("id")
	options.Limit = 1
	items, _, err := h.repo.List(c.Request.Context(), options)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if len(items) == 0 {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err := h.repo.SetUserState(c.Request.Context(), c.Param("id"), options.UserID, action); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *FindingHandler) Continue(c *gin.Context) {
	options, ok := h.options(c)
	if !ok {
		return
	}
	options.FindingID = c.Param("id")
	options.Limit = 1
	items, _, err := h.repo.List(c.Request.Context(), options)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if len(items) == 0 {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	finding := items[0]
	response.OK(c, gin.H{
		"context": gin.H{"agentFindingId": finding.ID},
		"message": fmt.Sprintf("请基于系统生成的主动分析继续调查：%s", finding.Title),
	})
}

func (h *FindingHandler) options(c *gin.Context) (FindingListOptions, bool) {
	userID, ok := admin.UserIDFromCtx(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return FindingListOptions{}, false
	}
	groups := []uuid.UUID(nil)
	if h.resolver != nil {
		resolved, err := h.resolver.ResolveFromContext(c)
		if err != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
			return FindingListOptions{}, false
		}
		groups = resolved
	}
	value, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	superAdmin, _ := value.(bool)
	return FindingListOptions{UserID: userID, SuperAdmin: superAdmin, VisibleGroups: groups}, true
}

func positiveIntQuery(c *gin.Context, key string, fallback, maximum int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maximum {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid %s: %w", key, commonerrors.ErrInvalidInput))
		return 0, false
	}
	return value, true
}
