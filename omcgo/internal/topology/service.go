package topology

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// DeviceGroupService provides business logic for device group management.
type DeviceGroupService struct {
	repo   DeviceGroupRepository
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewDeviceGroupService creates a new DeviceGroupService.
func NewDeviceGroupService(repo DeviceGroupRepository, pool *pgxpool.Pool, logger *zap.Logger) *DeviceGroupService {
	return &DeviceGroupService{repo: repo, pool: pool, logger: logger}
}

// GetTree returns the full group tree with children nested under parents.
func (s *DeviceGroupService) GetTree(ctx context.Context) ([]DeviceGroup, error) {
	flat, err := s.repo.GetTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("get group tree: %w", err)
	}
	return buildTree(flat), nil
}

// GetTreeWithCounts returns the tree with device counts and stats.
func (s *DeviceGroupService) GetTreeWithCounts(ctx context.Context) ([]DeviceGroup, *GroupStats, error) {
	flat, err := s.repo.GetTreeWithCounts(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get tree with counts: %w", err)
	}

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get group stats: %w", err)
	}

	tree := buildTree(flat)
	// Propagate L2 device counts up to L1 parents.
	for i := range tree {
		var childTotal int
		for _, child := range tree[i].Children {
			childTotal += child.DeviceCount
		}
		tree[i].DeviceCount += childTotal
	}

	return tree, stats, nil
}

// CreateGroup creates a device group with optional sub-groups.
func (s *DeviceGroupService) CreateGroup(ctx context.Context, req CreateGroupRequest, operator string) (*DeviceGroup, error) {
	var parentID *uuid.UUID
	var parentLevel int

	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "invalid parent_id", err)
		}
		parent, err := s.repo.GetByID(ctx, pid)
		if err != nil {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "parent group not found", err)
		}
		if parent.Level != 1 {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupLevelInvalid, "parent must be a level-1 group", nil)
		}
		parentID = &pid
		parentLevel = parent.Level
	}

	level := 1
	if parentID != nil {
		level = parentLevel + 1
	}

	// Name uniqueness check.
	exists, err := s.repo.ExistsByParentAndName(ctx, parentID, req.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("check name uniqueness: %w", err)
	}
	if exists {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupNameDuplicate, "group name already exists under this parent", nil)
	}

	group := &DeviceGroup{
		Name:         req.Name,
		ParentID:     parentID,
		Carrier:      carrier(req.Carrier),
		Remark:       req.Remark,
		SortOrder:    req.SortOrder,
		Level:        level,
		Status:       string(global.GroupStatusActive),
		CreatedBy:    operator,
		MatchingMode: MatchingMode(req.MatchingMode),
		NameRuleList: req.NameRuleList,
		LACList:      req.LACList,
		TACList:      req.TACList,
	}

	if err := s.repo.Create(ctx, group); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	// Create sub-groups if this is a L1 group.
	if level == 1 && len(req.SubGroups) > 0 {
		for i, sub := range req.SubGroups {
			childGroup := &DeviceGroup{
				Name:      sub.Name,
				ParentID:  &group.ID,
				Carrier:   carrier(req.Carrier),
				Remark:    sub.Remark,
				SortOrder: i + 1,
				Level:     2,
				Status:    string(global.GroupStatusActive),
				CreatedBy: operator,
			}
			if err := s.repo.Create(ctx, childGroup); err != nil {
				return nil, fmt.Errorf("create sub-group %q: %w", sub.Name, err)
			}

			// Assign devices to sub-group.
			if len(sub.DeviceIDs) > 0 {
				deviceIDs, err := parseUUIDs(sub.DeviceIDs)
				if err != nil {
					return nil, commonerrors.NewBusinessError(global.ErrCodeDeviceInvalidInput, "invalid device_id in sub_group", err)
				}
				if _, err := s.repo.BatchAddDevices(ctx, childGroup.ID, deviceIDs); err != nil {
					return nil, fmt.Errorf("assign devices to sub-group: %w", err)
				}
			}

			group.Children = append(group.Children, *childGroup)
		}
	}

	return group, nil
}

// UpdateGroup updates a device group (default groups cannot be modified).
func (s *DeviceGroupService) UpdateGroup(ctx context.Context, id uuid.UUID, req UpdateGroupRequest, operator string) (*DeviceGroup, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if group.IsDefault {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupIsDefault, "cannot modify default group", nil)
	}

	// 处理 ParentID 变更（L1 ↔ L2 转换）
	if req.ParentID != nil {
		newParentID := *req.ParentID
		// 不能将自己设为父级
		if newParentID == id.String() {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "cannot set self as parent", nil)
		}
		// 如果新父级不为空，需要验证父级存在且不能是自己的子级
		if newParentID != "" {
			parentUUID, err := uuid.Parse(newParentID)
			if err != nil {
				return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "invalid parent_id format", nil)
			}
			// 检查父级是否存在
			parent, err := s.repo.GetByID(ctx, parentUUID)
			if err != nil {
				return nil, commonerrors.NewBusinessError(global.ErrCodeGroupNotFound, "parent group not found", nil)
			}
			// 检查是否会形成循环（父级不能是当前分组的子级）
			if parent.ParentID != nil && *parent.ParentID == id {
				return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "cannot set a child group as parent", nil)
			}
			group.ParentID = &parentUUID
		} else {
			// 清空 parent_id，变为 L1 分组
			group.ParentID = nil
		}
	}

	// 确定用于名称唯一性检查的父级 ID
	checkParentID := group.ParentID
	if req.ParentID != nil && *req.ParentID != "" {
		if parsed, err := uuid.Parse(*req.ParentID); err == nil {
			checkParentID = &parsed
		}
	}

	if req.Name != nil {
		exists, err := s.repo.ExistsByParentAndName(ctx, checkParentID, *req.Name, &id)
		if err != nil {
			return nil, fmt.Errorf("check name uniqueness: %w", err)
		}
		if exists {
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupNameDuplicate, "group name already exists under this parent", nil)
		}
		group.Name = *req.Name
	}
	if req.Remark != nil {
		group.Remark = *req.Remark
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}
	// 更新匹配规则（仅 L2 分组）
	if req.MatchingMode != nil {
		group.MatchingMode = MatchingMode(*req.MatchingMode)
	}
	if req.NameRuleList != nil {
		group.NameRuleList = req.NameRuleList
	}
	if req.LACList != nil {
		group.LACList = req.LACList
	}
	if req.TACList != nil {
		group.TACList = req.TACList
	}
	group.UpdatedBy = operator

	if err := s.repo.Update(ctx, group); err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	return group, nil
}

// DeleteGroup deletes a group, moving its devices to the default L2 group.
// All steps run inside a single transaction when a pool is available.
func (s *DeviceGroupService) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if group.IsDefault {
		return commonerrors.NewBusinessError(global.ErrCodeGroupIsDefault, "cannot delete default group", nil)
	}

	// Collect IDs to delete (parent + all children for L1).
	groupIDs := []uuid.UUID{id}
	if group.Level == 1 {
		childIDs, err := s.repo.ListChildIDs(ctx, id)
		if err != nil {
			return fmt.Errorf("list children for cascade delete: %w", err)
		}
		groupIDs = append(groupIDs, childIDs...)
	}

	// When pool is available (production), use a transaction.
	if s.pool != nil {
		return s.deleteGroupTx(ctx, id, group, groupIDs)
	}

	// Fallback (e.g. unit tests with mock repo): non-transactional path.
	if _, err := s.repo.MoveGroupDevicesToDefault(ctx, groupIDs); err != nil {
		return fmt.Errorf("move devices to default: %w", err)
	}
	if group.Level == 1 {
		childIDs, _ := s.repo.ListChildIDs(ctx, id)
		for _, cid := range childIDs {
			if delErr := s.repo.Delete(ctx, cid); delErr != nil {
				s.logger.Warn("failed to delete child group", zap.String("child_id", cid.String()), zap.Error(delErr))
			}
		}
	}
	return s.repo.Delete(ctx, id)
}

// deleteGroupTx performs the delete group operation inside a pgx transaction.
func (s *DeviceGroupService) deleteGroupTx(ctx context.Context, id uuid.UUID, group *DeviceGroup, groupIDs []uuid.UUID) (err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete group transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Step 1: Move all devices from these groups to the default group.
	if _, err = moveGroupDevicesToDefaultTx(ctx, tx, groupIDs); err != nil {
		return fmt.Errorf("move devices to default: %w", err)
	}

	// Step 2: Delete children first (for L1 groups).
	if group.Level == 1 {
		childIDs, _ := listChildIDsTx(ctx, tx, id)
		for _, cid := range childIDs {
			if delErr := deleteGroupTx(ctx, tx, cid); delErr != nil {
				s.logger.Warn("failed to delete child group", zap.String("child_id", cid.String()), zap.Error(delErr))
			}
		}
	}

	// Step 3: Delete the parent (or L2) group itself.
	if err = deleteGroupTx(ctx, tx, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete group transaction: %w", err)
	}
	return nil
}

// CheckDelete returns the impact of deleting a group.
func (s *DeviceGroupService) CheckDelete(ctx context.Context, id uuid.UUID) (*CheckDeleteResponse, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &CheckDeleteResponse{CanDelete: true}

	if group.IsDefault {
		resp.CanDelete = false
		resp.Message = "默认设备组不可删除"
		return resp, nil
	}

	deviceCount, err := s.repo.CountDevicesByGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("count devices: %w", err)
	}
	resp.DeviceCount = deviceCount
	resp.HasDevices = deviceCount > 0

	if group.Level == 1 {
		childIDs, err := s.repo.ListChildIDs(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list children: %w", err)
		}
		resp.ChildCount = len(childIDs)

		// Count devices in children too.
		for _, cid := range childIDs {
			count, _ := s.repo.CountDevicesByGroup(ctx, cid)
			resp.DeviceCount += count
		}
		resp.HasDevices = resp.DeviceCount > 0
	}

	if resp.HasDevices {
		resp.Message = fmt.Sprintf("删除后 %d 台设备将自动归入默认组", resp.DeviceCount)
	}
	if resp.ChildCount > 0 {
		resp.Message = fmt.Sprintf("删除后 %d 个子组和 %d 台设备将自动归入默认组", resp.ChildCount, resp.DeviceCount)
	}

	return resp, nil
}

// MoveDevices moves devices to a target group.
func (s *DeviceGroupService) MoveDevices(ctx context.Context, req MoveDevicesRequest) (int64, error) {
	targetID, err := uuid.Parse(req.TargetGroupID)
	if err != nil {
		return 0, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "invalid target_group_id", err)
	}

	// 验证目标分组存在
	_, err = s.repo.GetByID(ctx, targetID)
	if err != nil {
		return 0, err
	}

	deviceIDs, err := parseUUIDs(req.DeviceIDs)
	if err != nil {
		return 0, commonerrors.NewBusinessError(global.ErrCodeDeviceInvalidInput, "invalid device_id", err)
	}

	return s.repo.MoveDevices(ctx, deviceIDs, targetID)
}

// BatchSort updates sort orders for multiple groups using a single CASE WHEN SQL.
func (s *DeviceGroupService) BatchSort(ctx context.Context, items []SortItem) error {
	if len(items) == 0 {
		return nil
	}

	repo, ok := s.repo.(*PgDeviceGroupRepository)
	if !ok {
		// Fallback for non-PG implementations (e.g. tests with mock repos).
		for _, item := range items {
			id, err := uuid.Parse(item.ID)
			if err != nil {
				continue
			}
			group, err := s.repo.GetByID(ctx, id)
			if err != nil {
				continue
			}
			group.SortOrder = item.SortOrder
			if err := s.repo.Update(ctx, group); err != nil {
				s.logger.Warn("batch sort update failed", zap.String("group_id", id.String()), zap.Error(err))
			}
		}
		return nil
	}

	batchItems := make([]BatchSortItem, 0, len(items))
	for _, item := range items {
		id, err := uuid.Parse(item.ID)
		if err != nil {
			s.logger.Warn("batch sort: invalid group_id", zap.String("id", item.ID), zap.Error(err))
			continue
		}
		batchItems = append(batchItems, BatchSortItem{ID: id, SortOrder: item.SortOrder})
	}

	if _, err := repo.BatchSort(ctx, batchItems); err != nil {
		return fmt.Errorf("batch sort groups: %w", err)
	}
	return nil
}

// GetStats returns group statistics.
func (s *DeviceGroupService) GetStats(ctx context.Context) (*GroupStats, error) {
	return s.repo.GetStats(ctx)
}

// --- Helpers ---

// buildTree assembles a flat list of groups into a tree structure.
func buildTree(flat []DeviceGroup) []DeviceGroup {
	byID := make(map[uuid.UUID]*DeviceGroup)
	var roots []DeviceGroup

	for i := range flat {
		flat[i].Children = nil
		byID[flat[i].ID] = &flat[i]
	}

	for i := range flat {
		g := &flat[i]
		if g.ParentID == nil {
			roots = append(roots, *g)
		} else if parent, ok := byID[*g.ParentID]; ok {
			parent.Children = append(parent.Children, *g)
		}
	}

	for i := range roots {
		if indexed, ok := byID[roots[i].ID]; ok {
			roots[i].Children = indexed.Children
		}
	}

	return roots
}

func parseUUIDs(strs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(strs))
	for _, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID %q: %w", s, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func carrier(s string) carrierCode {
	return carrierCode(s)
}

type carrierCode = model.CarrierCode
