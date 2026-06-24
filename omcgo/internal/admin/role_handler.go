package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

func (h *Handler) ListRoles(c *gin.Context) {
	var filter RoleFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListRolesPaginated(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, result)
}

// ListAllRoles 返回所有角色（不分页，用于下拉框）
func (h *Handler) ListAllRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, roles)
}

func (h *Handler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	role, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OK(c, role)
}

func (h *Handler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	role, err := h.service.CreateRole(userContextWithOperator(c), req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, role)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	role, err := h.service.UpdateRole(userContextWithOperator(c), id, req)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OK(c, role)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteRole(c.Request.Context(), id); err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OK(c, nil)
}

// CopyRole handles POST /roles/:id/copy（roles.md §7 P2 #9）。
// 一键复制角色：name 自动加 _copy / _copy_2... 后缀；permissions / role_menus /
// role_device_groups / role_api_permissions 全部复制；is_system 强制为 false。
func (h *Handler) CopyRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	role, err := h.service.CopyRole(userContextWithOperator(c), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, role)
}

func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.service.ListAllPermissions(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, perms)
}

// GetRoleDeviceGroups handles GET /roles/:id/device-groups.
func (h *Handler) GetRoleDeviceGroups(c *gin.Context) {
	if h.roleGroupRepo == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("role device group service not configured"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	data, err := h.roleGroupRepo.GetDeviceGroupData(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, data)
}

// SetRoleDeviceGroups handles PUT /roles/:id/device-groups.
func (h *Handler) SetRoleDeviceGroups(c *gin.Context) {
	if h.roleGroupRepo == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			errors.New("role device group service not configured"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req RoleDeviceGroupData
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.roleGroupRepo.SetDeviceGroupData(c.Request.Context(), id, req); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// PRD roles.md §10 DoD：设备分组绑定变更后必须失效该角色下所有用户的可见域缓存。
	// 之前只调 InvalidateRoleCache（log-only），无法真正使 perm:visible_groups 失效。
	h.service.InvalidatePermCacheByRole(c.Request.Context(), id)

	response.OKWithMsg(c, nil, "device groups updated")
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	var filter AuditLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.auditRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, result)
}

// ListRoleUsers handles GET /roles/:id/users — returns paginated users in a role.
func (h *Handler) ListRoleUsers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var filter model.ListRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.GetRoleUsers(c.Request.Context(), id, filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, result)
}
