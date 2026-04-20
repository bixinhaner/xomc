package mml

import (
	"context"

	"github.com/google/uuid"
)

// ParamService provides business logic for the parameter library.
type ParamService struct {
	repo ParamRepository
}

// NewParamService creates a new ParamService.
func NewParamService(repo ParamRepository) *ParamService {
	return &ParamService{repo: repo}
}

// ListVersions returns all parameter versions.
func (s *ParamService) ListVersions(ctx context.Context) ([]ParamVersion, error) {
	return s.repo.ListVersions(ctx)
}

// GetGroupTree returns a hierarchical group tree for a given version.
func (s *ParamService) GetGroupTree(ctx context.Context, versionCode string) ([]ParamGroup, error) {
	groups, err := s.repo.ListGroupsByVersion(ctx, versionCode)
	if err != nil {
		return nil, err
	}
	return buildGroupTree(groups), nil
}

// GetGroupParams returns all parameters belonging to a group.
func (s *ParamService) GetGroupParams(ctx context.Context, groupID uuid.UUID) ([]Param, error) {
	return s.repo.ListParamsByGroup(ctx, groupID)
}

// SearchParams searches parameters within a version.
func (s *ParamService) SearchParams(ctx context.Context, filter ParamFilter) ([]Param, error) {
	return s.repo.SearchParams(ctx, filter)
}

// buildGroupTree converts a flat list of groups into a tree structure.
func buildGroupTree(groups []ParamGroup) []ParamGroup {
	index := make(map[uuid.UUID]*ParamGroup, len(groups))
	for i := range groups {
		index[groups[i].ID] = &groups[i]
	}

	var roots []ParamGroup
	for i := range groups {
		g := &groups[i]
		if g.ParentID == nil {
			roots = append(roots, *g)
		} else {
			if parent, ok := index[*g.ParentID]; ok {
				parent.Children = append(parent.Children, *g)
			}
		}
	}
	return roots
}
