package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ApiEndpoint represents an API endpoint
type ApiEndpoint struct {
	ID          uuid.UUID `json:"id"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Description string    `json:"description"`
}

// ListApiEndpoints returns all registered API endpoints
func (h *Handler) ListApiEndpoints(c *gin.Context) {
	// TODO: 从数据库或配置文件中读取API端点列表
	// 这里返回一个示例数据
	endpoints := []ApiEndpoint{
		{
			ID:          uuid.New(),
			Path:        "/api/v1/users",
			Method:      "GET",
			Name:        "查询用户列表",
			Module:      "user",
			Description: "获取所有用户列表",
		},
		{
			ID:          uuid.New(),
			Path:        "/api/v1/users",
			Method:      "POST",
			Name:        "创建用户",
			Module:      "user",
			Description: "创建新用户",
		},
		{
			ID:          uuid.New(),
			Path:        "/api/v1/users/:id",
			Method:      "PUT",
			Name:        "更新用户",
			Module:      "user",
			Description: "更新用户信息",
		},
		{
			ID:          uuid.New(),
			Path:        "/api/v1/users/:id",
			Method:      "DELETE",
			Name:        "删除用户",
			Module:      "user",
			Description: "删除用户",
		},
		{
			ID:          uuid.New(),
			Path:        "/api/v1/roles",
			Method:      "GET",
			Name:        "查询角色列表",
			Module:      "role",
			Description: "获取所有角色列表",
		},
		{
			ID:          uuid.New(),
			Path:        "/api/v1/roles",
			Method:      "POST",
			Name:        "创建角色",
			Module:      "role",
			Description: "创建新角色",
		},
	}

	c.JSON(http.StatusOK, gin.H{"data": endpoints})
}

// GetRoleApiPermissions returns API permissions for a role
func (h *Handler) GetRoleApiPermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	// TODO: 从数据库查询角色的API权限
	// 这里返回示例数据
	permissions := []map[string]string{
		{"path": "/api/v1/users", "method": "GET"},
		{"path": "/api/v1/roles", "method": "GET"},
	}

	c.JSON(http.StatusOK, gin.H{"data": permissions, "role_id": roleID})
}

// SetRoleApiPermissions sets API permissions for a role
func (h *Handler) SetRoleApiPermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	var req struct {
		Permissions []map[string]string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 保存角色的API权限到数据库
	// 这里只是返回成功

	c.JSON(http.StatusOK, gin.H{"success": true, "role_id": roleID, "count": len(req.Permissions)})
}
