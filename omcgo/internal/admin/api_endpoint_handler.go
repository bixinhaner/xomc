package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// ListApiEndpoints returns API endpoints from the database with pagination and filtering.
func (h *Handler) ListApiEndpoints(c *gin.Context) {
	var filter ApiEndpointFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.apiEndpointService.ListApiEndpoints(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateApiEndpoint creates a new API endpoint.
func (h *Handler) CreateApiEndpoint(c *gin.Context) {
	var req CreateApiEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ep, err := h.apiEndpointService.CreateApiEndpoint(c.Request.Context(), req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusCreated, ep)
}

// UpdateApiEndpoint modifies an existing API endpoint.
func (h *Handler) UpdateApiEndpoint(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateApiEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ep, err := h.apiEndpointService.UpdateApiEndpoint(c.Request.Context(), id, req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, ep)
}

// DeleteApiEndpoint removes a single API endpoint by ID.
func (h *Handler) DeleteApiEndpoint(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.apiEndpointService.DeleteApiEndpoint(c.Request.Context(), id); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// BatchDeleteApiEndpoints removes multiple API endpoints by IDs.
func (h *Handler) BatchDeleteApiEndpoints(c *gin.Context) {
	var req BatchDeleteApiEndpointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.apiEndpointService.DeleteApiEndpointsByIDs(c.Request.Context(), req.IDs); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetApiGroups returns the list of distinct api_group values.
func (h *Handler) GetApiGroups(c *gin.Context) {
	groups, err := h.apiEndpointService.GetApiGroups(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": groups})
}

// SyncApiEndpoints triggers a sync of Gin routes into the database.
func (h *Handler) SyncApiEndpoints(c *gin.Context) {
	if h.ginRoutes == nil {
		c.JSON(http.StatusOK, SyncResult{})
		return
	}

	result, err := h.apiEndpointService.SyncApiEndpoints(c.Request.Context(), h.ginRoutes)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ginRoutesContextKey is the context key used to pass gin.RoutesInfo into handlers.
type ginRoutesKeyType struct{}

var ginRoutesContextKey = ginRoutesKeyType{}

// GetRoleApiPermissions returns API endpoint IDs granted to a role.
func (h *Handler) GetRoleApiPermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if h.apiPermRepo == nil {
		c.JSON(http.StatusOK, gin.H{"endpoint_ids": []uuid.UUID{}})
		return
	}

	ids, err := h.apiPermRepo.GetRoleApiEndpointIDs(c.Request.Context(), roleID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}

	c.JSON(http.StatusOK, gin.H{"endpoint_ids": ids})
}

// SetRoleApiPermissions sets API endpoint permissions for a role.
func (h *Handler) SetRoleApiPermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req struct {
		EndpointIDs []uuid.UUID `json:"endpoint_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if h.apiPermRepo == nil {
		c.JSON(http.StatusOK, gin.H{"message": "api permission service not configured"})
		return
	}

	if err := h.apiPermRepo.SetRoleApiEndpoints(c.Request.Context(), roleID, req.EndpointIDs); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// PRD roles.md §10 DoD：API 权限变更后失效该角色下所有用户的可见域缓存。
	h.service.InvalidatePermCacheByRole(c.Request.Context(), roleID)

	c.JSON(http.StatusOK, gin.H{"message": "api permissions updated", "count": len(req.EndpointIDs)})
}
