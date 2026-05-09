package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
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

	response.OKWithStatus(c, http.StatusCreated, menu)
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

	response.OK(c, menu)
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

	response.OK(c, result)
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

	response.OK(c, gin.H{"success": true})
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

	response.OK(c, gin.H{"success": true})
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

	response.OK(c, gin.H{"data": menus})
}

// GetUserMenuTree handles GET /menus/user.
func (h *Handler) GetUserMenuTree(c *gin.Context) {
	userID := getUserID(c)

	menus, err := h.service.GetUserMenuTree(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{"data": menus})
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

	response.OK(c, gin.H{"success": true})
}

// GetRoleMenus handles GET /roles/:id/menus.
//
// 返回角色当前绑定的菜单 ID 列表（仅 UUID 字符串，不返回完整 Menu 对象）。
// 响应契约：{ menu_ids: [string, ...] } —— 与前端 fetchRoleMenuIds + RolePermission
// 编辑页 setCheckedPermissionKeys 严格对齐。历史上曾返回 {data:[Menu, ...]} 与前端
// 契约不符，导致角色编辑保存后重开菜单权限树全部空回显。
func (h *Handler) GetRoleMenus(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	menuIDs, err := h.service.GetRoleMenuIDs(c.Request.Context(), roleID)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	// 显式空切片避免 JSON marshal 出 null（前端 `data?.menu_ids ?? []` 兜底也行，
	// 但显式空数组语义更清晰）。
	if menuIDs == nil {
		menuIDs = []uuid.UUID{}
	}
	response.OK(c, gin.H{"menu_ids": menuIDs})
}
