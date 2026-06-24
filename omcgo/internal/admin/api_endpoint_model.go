package admin

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ApiEndpointDB represents an API endpoint stored in the database.
type ApiEndpointDB struct {
	ID          uuid.UUID `json:"id"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ApiGroup    string    `json:"api_group"`
	IsAuto      bool      `json:"is_auto"`
	// IsUserModified 为 true 表示该端点的 name/description/api_group 被用户手工改过，
	// Sync 扫描时不再用自动推断值覆盖（保护用户已保存的修改）。
	IsUserModified bool      `json:"is_user_modified"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ApiEndpointFilter provides filtering options for listing API endpoints.
type ApiEndpointFilter struct {
	Path     string `form:"path"`
	Method   string `form:"method"`
	ApiGroup string `form:"api_group"`
	Name     string `form:"name"`
	model.ListRequest
}

// CreateApiEndpointRequest is the input for creating a new API endpoint.
type CreateApiEndpointRequest struct {
	Path        string `json:"path" binding:"required"`
	Method      string `json:"method" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ApiGroup    string `json:"api_group"`
}

// UpdateApiEndpointRequest is the input for updating an existing API endpoint.
type UpdateApiEndpointRequest struct {
	Path        *string `json:"path"`
	Method      *string `json:"method"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	ApiGroup    *string `json:"api_group"`
}

// BatchDeleteApiEndpointsRequest is the input for batch-deleting API endpoints.
type BatchDeleteApiEndpointsRequest struct {
	IDs []uuid.UUID `json:"ids" binding:"required,min=1"`
}

// SyncResult holds the statistics of a route sync operation.
type SyncResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Total   int `json:"total"`
}
