package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// CreateMenu handles POST /menus.
func (h *Handler) CreateMenu(c *gin.Context) {
	var req CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operatorID := getUserID(c)

	menu, err := h.service.CreateMenu(c.Request.Context(), req, operatorID)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusCreated, menu)
}

// GetMenu handles GET /menus/:id.
func (h *Handler) GetMenu(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	menu, err := h.service.GetMenu(c.Request.Context(), id)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, menu)
}

// ListMenus handles GET /menus.
func (h *Handler) ListMenus(c *gin.Context) {
	var filter MenuFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListMenus(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateMenu handles PUT /menus/:id.
func (h *Handler) UpdateMenu(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operatorID := getUserID(c)

	if err := h.service.UpdateMenu(c.Request.Context(), id, &req, operatorID); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteMenus handles DELETE /menus.
func (h *Handler) DeleteMenus(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	ids := make([]uuid.UUID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		ids[i] = id
	}

	if err := h.service.DeleteMenus(c.Request.Context(), ids); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMenuTree handles GET /menus/tree.
func (h *Handler) GetMenuTree(c *gin.Context) {
	statusStr := c.Query("status")
	var status *MenuStatus
	if statusStr != "" {
		s := MenuStatus(statusStr)
		status = &s
	}

	menus, err := h.service.GetMenuTree(c.Request.Context(), status)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menus})
}

// GetUserMenuTree handles GET /menus/user.
func (h *Handler) GetUserMenuTree(c *gin.Context) {
	userID := getUserID(c)

	menus, err := h.service.GetUserMenuTree(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menus})
}

// SetRoleMenus handles PUT /roles/:id/menus.
func (h *Handler) SetRoleMenus(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req SetRoleMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operatorID := getUserID(c)

	if err := h.service.SetRoleMenus(c.Request.Context(), roleID, req.MenuIDs, operatorID); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetRoleMenus handles GET /roles/:id/menus.
func (h *Handler) GetRoleMenus(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	menus, err := h.service.GetRoleMenus(c.Request.Context(), roleID)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menus})
}
