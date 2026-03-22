package topology

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DeviceGroupService provides business logic for device group management.
type DeviceGroupService struct {
	repo   GroupReader
	logger *zap.Logger
}

// NewDeviceGroupService creates a new DeviceGroupService.
func NewDeviceGroupService(repo GroupReader, logger *zap.Logger) *DeviceGroupService {
	return &DeviceGroupService{repo: repo, logger: logger}
}

// GetTree returns the full group tree with children nested under parents.
func (s *DeviceGroupService) GetTree(ctx context.Context) ([]DeviceGroup, error) {
	flat, err := s.repo.GetTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("get group tree: %w", err)
	}
	return buildTree(flat), nil
}

// buildTree assembles a flat list of groups into a tree structure.
func buildTree(flat []DeviceGroup) []DeviceGroup {
	byID := make(map[uuid.UUID]*DeviceGroup)
	var roots []DeviceGroup

	// Index all groups by ID.
	for i := range flat {
		flat[i].Children = nil
		byID[flat[i].ID] = &flat[i]
	}

	// Build parent-child relationships.
	for i := range flat {
		g := &flat[i]
		if g.ParentID == nil {
			roots = append(roots, *g)
		} else if parent, ok := byID[*g.ParentID]; ok {
			parent.Children = append(parent.Children, *g)
		}
	}

	// Copy children from indexed groups to roots.
	for i := range roots {
		if indexed, ok := byID[roots[i].ID]; ok {
			roots[i].Children = indexed.Children
		}
	}

	return roots
}
