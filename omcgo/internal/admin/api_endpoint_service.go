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
// Examples:
//
//	/api/v1/admin/users        -> admin
//	/api/v1/devices            -> devices
//	/api/v1/device-groups/tree -> device-groups
func inferApiGroup(path string) string {
	// Strip leading slash and split
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	// Skip well-known prefix segments: "api", "v1" (or any vN), "admin"
	skip := map[string]bool{"api": true, "v1": true, "v2": true, "v3": true}
	for _, p := range parts {
		if p == "" || skip[p] {
			continue
		}
		// Return the first meaningful segment as the group
		return p
	}
	return ""
}

// inferRouteName generates a human-readable name from method and path.
func inferRouteName(method, path string) string {
	return strings.ToUpper(method) + " " + path
}
