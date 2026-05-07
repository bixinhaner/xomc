package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ApiEndpointService provides CRUD and sync operations for API endpoints.
type ApiEndpointService struct {
	repo   ApiEndpointRepository
	logger *zap.Logger
}

// NewApiEndpointService creates a new ApiEndpointService.
func NewApiEndpointService(repo ApiEndpointRepository, logger *zap.Logger) *ApiEndpointService {
	return &ApiEndpointService{
		repo:   repo,
		logger: logger.Named("api-endpoint-service"),
	}
}

// ListApiEndpoints retrieves a paginated, filtered list of API endpoints.
func (s *ApiEndpointService) ListApiEndpoints(ctx context.Context, filter ApiEndpointFilter) (*model.ListResponse[ApiEndpointDB], error) {
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list api endpoints: %w", err)
	}
	return result, nil
}

// CreateApiEndpoint inserts a new API endpoint.
func (s *ApiEndpointService) CreateApiEndpoint(ctx context.Context, req CreateApiEndpointRequest) (*ApiEndpointDB, error) {
	ep := &ApiEndpointDB{
		Path:        req.Path,
		Method:      strings.ToUpper(req.Method),
		Name:        req.Name,
		Description: req.Description,
		ApiGroup:    req.ApiGroup,
		IsAuto:      false,
	}
	if err := s.repo.Create(ctx, ep); err != nil {
		return nil, fmt.Errorf("create api endpoint: %w", err)
	}
	return ep, nil
}

// UpdateApiEndpoint modifies an existing API endpoint.
func (s *ApiEndpointService) UpdateApiEndpoint(ctx context.Context, id uuid.UUID, req UpdateApiEndpointRequest) (*ApiEndpointDB, error) {
	ep, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("update api endpoint: %w", err)
	}
	return ep, nil
}

// DeleteApiEndpoint removes a single API endpoint by ID.
func (s *ApiEndpointService) DeleteApiEndpoint(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete api endpoint: %w", err)
	}
	return nil
}

// DeleteApiEndpointsByIDs removes multiple API endpoints.
func (s *ApiEndpointService) DeleteApiEndpointsByIDs(ctx context.Context, ids []uuid.UUID) error {
	if err := s.repo.DeleteByIDs(ctx, ids); err != nil {
		return fmt.Errorf("batch delete api endpoints: %w", err)
	}
	return nil
}

// GetApiGroups returns the distinct list of api_group values.
func (s *ApiEndpointService) GetApiGroups(ctx context.Context) ([]string, error) {
	groups, err := s.repo.GetGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("get api groups: %w", err)
	}
	return groups, nil
}

// SyncApiEndpoints scans gin.RoutesInfo and upserts each route into the database.
// Routes with is_auto=false (manually created) are preserved.
func (s *ApiEndpointService) SyncApiEndpoints(ctx context.Context, routes gin.RoutesInfo) (SyncResult, error) {
	var result SyncResult
	result.Total = len(routes)

	for _, route := range routes {
		apiGroup := inferApiGroup(route.Path)
		name := inferRouteName(route.Method, route.Path)

		created, err := s.repo.Upsert(ctx, route.Path, route.Method, name, apiGroup)
		if err != nil {
			s.logger.Warn("failed to upsert route",
				zap.String("path", route.Path),
				zap.String("method", route.Method),
				zap.Error(err),
			)
			continue
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	s.logger.Info("synced api endpoints",
		zap.Int("total", result.Total),
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
	)
	return result, nil
}

// inferApiGroup extracts an api_group from a URL path.
//
// 规则：跳过 /api/v{N} 前缀后取第一个有意义的段。/admin/<sub>/... 形态再下钻
// 一级，避免所有 admin 路径都被聚成同一个 "admin" 组（粒度过粗），更贴近
// 业务模块（users/roles/menus/sysConfig/api-endpoints 等）。
//
// Examples:
//
//	/api/v1/admin/users        -> users
//	/api/v1/admin/roles/:id    -> roles
//	/api/v1/admin/sysConfig    -> sysConfig
//	/api/v1/auth/login         -> auth
//	/api/v1/devices            -> devices
//	/api/v1/device-groups/tree -> device-groups
//	/api/v1/admin              -> admin   （兜底，无下钻段）
func inferApiGroup(path string) string {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	skip := map[string]bool{"api": true, "v1": true, "v2": true, "v3": true}
	idx := 0
	for idx < len(parts) && (parts[idx] == "" || skip[parts[idx]]) {
		idx++
	}
	if idx >= len(parts) {
		return ""
	}

	first := parts[idx]
	// /admin/<sub>/... 下钻一级；仅当下一段非空时生效，避免 "/admin" 退化成 ""。
	if first == "admin" && idx+1 < len(parts) && parts[idx+1] != "" {
		return parts[idx+1]
	}
	return first
}

// inferRouteName generates a human-readable name from method and path.
func inferRouteName(method, path string) string {
	return strings.ToUpper(method) + " " + path
}
