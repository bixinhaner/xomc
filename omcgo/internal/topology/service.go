package topology

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

// DeviceGroupService provides business logic for device group management.
type DeviceGroupService struct {
	repo         DeviceGroupRepository
	topoNodeRepo TopoNodeRepository
	pool         *pgxpool.Pool
	logger       *zap.Logger

	// PRD users.md §11.7 决议③ / roles.md §11.4：删除设备分组联动钩子。
	// 两者均可为 nil（测试场景），DeleteGroup 退化为旧行为。
	roleQuery       RoleAffectedQuery
	roleCacheBuster RoleCachePurger

	// groupMatchEngine：分组新增/编辑后异步回灌匹配设备。可为 nil（测试/未接线
	// 时退化为不触发）。
	groupMatchEngine groupMatcher
}

// groupMatcher 是 DeviceGroupService 消费的窄接口：分组增改后按其匹配规则
// 异步回灌设备。由 *GroupMatchEngine 实现。
type groupMatcher interface {
	MatchGroup(ctx context.Context, groupID uuid.UUID) error
}

type deviceGroupTreeCacheInvalidator interface {
	InvalidateDeviceGroupCounts()
}

// NewDeviceGroupService creates a new DeviceGroupService.
func NewDeviceGroupService(repo DeviceGroupRepository, topoNodeRepo TopoNodeRepository, pool *pgxpool.Pool, logger *zap.Logger) *DeviceGroupService {
	return &DeviceGroupService{repo: repo, topoNodeRepo: topoNodeRepo, pool: pool, logger: logger}
}

// SetGroupMatchEngine 注入分组匹配引擎，使分组新增/编辑后异步触发设备回灌。
func (s *DeviceGroupService) SetGroupMatchEngine(m groupMatcher) {
	s.groupMatchEngine = m
}

// fireGroupMatch 在分组新增/编辑成功后异步回灌匹配设备。对传入分组及其子分组
// 逐个调 MatchGroup（非 L2 / 未配匹配规则的分组会在 MatchGroup 内安全 no-op）。
func (s *DeviceGroupService) fireGroupMatch(g *DeviceGroup) {
	if s.groupMatchEngine == nil || g == nil {
		return
	}
	ids := []uuid.UUID{g.ID}
	for i := range g.Children {
		ids = append(ids, g.Children[i].ID)
	}
	go func() {
		ctx := context.Background()
		for _, id := range ids {
			if err := s.groupMatchEngine.MatchGroup(ctx, id); err != nil {
				s.logger.Warn("group match after group CRUD failed",
					zap.String("group_id", id.String()), zap.Error(err))
			}
		}
	}()
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

func (s *DeviceGroupService) validateRuleSource(ctx context.Context, targetID uuid.UUID, level int, mode MatchingMode, sourceRaw string) (*uuid.UUID, error) {
	if mode == "" && sourceRaw == "" {
		return nil, nil
	}
	if level != 2 {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupLevelInvalid, "matching rules are only supported for level-2 groups", nil)
	}
	if mode != "" && sourceRaw == "" {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "source_group_id is required for matching rules", nil)
	}
	sourceID, err := uuid.Parse(sourceRaw)
	if err != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "invalid source_group_id", err)
	}
	if targetID != uuid.Nil && sourceID == targetID {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "source group cannot equal target group", nil)
	}
	source, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "source group not found", err)
	}
	if source.Level != 2 {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupLevelInvalid, "source group must be a level-2 group", nil)
	}
	return &sourceID, nil
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
			return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "parent group not found",
				fmt.Errorf("get parent group: %w", err))
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

	sourceGroupID, err := s.validateRuleSource(ctx, uuid.Nil, level, MatchingMode(req.MatchingMode), req.SourceGroupID)
	if err != nil {
		return nil, err
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
		Name:             req.Name,
		NameI18n:         req.NameI18n,
		DescriptionI18n:  req.DescriptionI18n,
		RemarkI18n:       req.RemarkI18n,
		ParentID:         parentID,
		Carrier:          carrier(req.Carrier),
		Remark:           req.Remark,
		SortOrder:        req.SortOrder,
		Level:            level,
		Status:           string(global.GroupStatusActive),
		CreatedBy:        operator,
		SourceGroupID:    sourceGroupID,
		MatchingMode:     MatchingMode(req.MatchingMode),
		NameRuleList:     req.NameRuleList,
		LACList:          req.LACList,
		TACList:          req.TACList,
		SerialNumberList: req.SerialNumberList,
	}

	if err := s.repo.Create(ctx, group); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	s.invalidateTreeCache()

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
			s.invalidateTreeCache()

			// Assign devices to sub-group.
			if len(sub.DeviceIDs) > 0 {
				deviceIDs, err := parseUUIDs(sub.DeviceIDs)
				if err != nil {
					return nil, commonerrors.NewBusinessError(global.ErrCodeDeviceInvalidInput, "invalid device_id in sub_group", err)
				}
				if _, err := s.repo.BatchAddDevices(ctx, childGroup.ID, deviceIDs); err != nil {
					return nil, fmt.Errorf("assign devices to sub-group: %w", err)
				}
				s.invalidateTreeCache()
			}

			group.Children = append(group.Children, *childGroup)
		}
	}

	// 异步回灌：按新分组（及其子分组）的匹配规则把命中设备归入。
	s.fireGroupMatch(group)
	return group, nil
}

func (s *DeviceGroupService) invalidateTreeCache() {
	if invalidator, ok := s.repo.(deviceGroupTreeCacheInvalidator); ok {
		invalidator.InvalidateDeviceGroupCounts()
	}
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
			group.Level = 2 // 有父级，变为 L2 分组
		} else {
			// 清空 parent_id，变为 L1 分组
			group.ParentID = nil
			group.Level = 1 // 无父级，变为 L1 分组
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
	if req.NameI18n != nil {
		group.NameI18n = req.NameI18n
	}
	if req.DescriptionI18n != nil {
		group.DescriptionI18n = req.DescriptionI18n
	}
	if req.RemarkI18n != nil {
		group.RemarkI18n = req.RemarkI18n
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
	if req.SerialNumberList != nil {
		group.SerialNumberList = req.SerialNumberList
	}
	if req.SourceGroupID != nil {
		if *req.SourceGroupID == "" {
			group.SourceGroupID = nil
		} else {
			sourceID, err := s.validateRuleSource(ctx, group.ID, group.Level, group.MatchingMode, *req.SourceGroupID)
			if err != nil {
				return nil, err
			}
			group.SourceGroupID = sourceID
		}
	}
	if req.SourceGroupID != nil && *req.SourceGroupID == "" && group.MatchingMode != "" {
		return nil, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "source_group_id is required for matching rules", nil)
	}
	group.UpdatedBy = operator

	if err := s.repo.Update(ctx, group); err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	s.invalidateTreeCache()

	// 异步回灌：按编辑后的匹配规则把命中设备归入本分组。
	s.fireGroupMatch(group)
	return group, nil
}

// DeleteGroup deletes a group, moving its devices to the default L2 group.
// All steps run inside a single transaction when a pool is available.
//
// PRD users.md §11.7 决议③ / roles.md §11.4：删除前查 role_device_groups 取得受
// 影响角色集 → 删除完成后写审计日志 + 失效这些角色下所有用户的可见域缓存。
// 当钩子未注入（roleQuery == nil）时退化为旧行为（仅删库）。
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

	// 1) 删前查受影响角色（删后 role_device_groups ON DELETE CASCADE 会清掉关联）。
	var affected []AffectedRole
	if s.roleQuery != nil {
		var qerr error
		affected, qerr = s.roleQuery.ListRolesByGroupIDs(ctx, groupIDs)
		if qerr != nil {
			s.logger.Warn("query affected roles before group delete failed",
				zap.String("group_id", id.String()), zap.Error(qerr))
		}
	}

	// 2) 实际删除（事务 / 非事务两路径不变）。
	if s.pool != nil {
		if err := s.deleteGroupTx(ctx, id, group, groupIDs); err != nil {
			return err
		}
	} else {
		if _, err := s.repo.MoveGroupDevicesToDefault(ctx, groupIDs); err != nil {
			return fmt.Errorf("move devices to default: %w", err)
		}
		if group.Level == 1 {
			childIDs, _ := s.repo.ListChildIDs(ctx, id)
			for _, cid := range childIDs {
				if delErr := s.repo.Delete(ctx, cid); delErr != nil {
					s.logger.Warn("failed to delete child group",
						zap.String("child_id", cid.String()), zap.Error(delErr))
				}
			}
		}
		if err := s.repo.Delete(ctx, id); err != nil {
			return err
		}
	}
	s.invalidateTreeCache()

	// 3) 删后处理：写审计 + 失效缓存（任一失败仅日志，不阻塞删除结果）。
	s.afterGroupDeleted(ctx, id, group, affected)
	return nil
}

// afterGroupDeleted 实现 PRD §11.7 决议③ 后续动作：审计 + 通知（占位） + 缓存失效。
// audit + cache invalidate 单独 goroutine 执行可减少延迟，但当前规模下顺序调用足够，
// 且能让单元测试稳定断言。
func (s *DeviceGroupService) afterGroupDeleted(ctx context.Context, groupID uuid.UUID, group *DeviceGroup, affected []AffectedRole) {
	if len(affected) == 0 {
		return
	}

	// 写审计日志（失败仅日志）。
	rolesPayload := make([]map[string]any, 0, len(affected))
	for _, r := range affected {
		rolesPayload = append(rolesPayload, map[string]any{
			"role_id":   r.ID.String(),
			"role_name": r.Name,
		})
	}
	audit.Log(ctx, audit.Entry{
		Action:       "role_device_group_revoked_by_group_delete",
		ResourceType: "device_group",
		ResourceID:   groupID.String(),
		Details: map[string]interface{}{
			"group_name":     group.Name,
			"affected_roles": rolesPayload,
			"deleted_at":     time.Now().UTC().Format(time.RFC3339),
		},
		Success: true,
	})

	// 失效每个受影响角色下所有用户的可见域缓存。
	if s.roleCacheBuster != nil {
		for _, r := range affected {
			s.roleCacheBuster.InvalidatePermCacheByRole(ctx, r.ID)
		}
	}
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
		// 参数校验类错误（非法 UUID）须映射 400：包 ErrInvalidInput sentinel，
		// 使 HTTPStatusFromError 经 errors.Is 落到 StatusBadRequest 而非 default 500。
		return 0, commonerrors.NewBusinessError(global.ErrCodeGroupParentInvalid, "invalid target_group_id",
			fmt.Errorf("%w: %v", commonerrors.ErrInvalidInput, err))
	}

	deviceIDs, err := parseUUIDs(req.DeviceIDs)
	if err != nil {
		// 同上：非法 device_id UUID 是参数校验类错误，须映射 400。
		return 0, commonerrors.NewBusinessError(global.ErrCodeDeviceInvalidInput, "invalid device_id",
			fmt.Errorf("%w: %v", commonerrors.ErrInvalidInput, err))
	}

	// 验证目标分组存在
	_, err = s.repo.GetByID(ctx, targetID)
	if err != nil {
		return 0, fmt.Errorf("get target group: %w", err)
	}

	affected, err := s.repo.MoveDevices(ctx, deviceIDs, targetID)
	if err != nil {
		return 0, fmt.Errorf("move devices: %w", err)
	}
	if affected > 0 {
		s.invalidateTreeCache()
	}
	return affected, nil
}

// BatchAddDevices 把多台设备加入目标分组（服务层统一入口）。
func (s *DeviceGroupService) BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	affected, err := s.repo.BatchAddDevices(ctx, groupID, deviceIDs)
	if err != nil {
		return 0, fmt.Errorf("batch add devices to group: %w", err)
	}
	if affected > 0 {
		s.invalidateTreeCache()
	}
	return affected, nil
}

// AddDevice 把单台设备加入目标分组（服务层统一入口，legacy）。
func (s *DeviceGroupService) AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	if err := s.repo.AddDevice(ctx, groupID, deviceID); err != nil {
		return fmt.Errorf("add device to group: %w", err)
	}
	s.invalidateTreeCache()
	return nil
}

// BatchRemoveDevices removes multiple devices from a group.
func (s *DeviceGroupService) BatchRemoveDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	affected, err := s.repo.BatchRemoveDevices(ctx, groupID, deviceIDs)
	if err != nil {
		return 0, fmt.Errorf("batch remove devices from group: %w", err)
	}
	if affected > 0 {
		s.invalidateTreeCache()
	}
	return affected, nil
}

// RemoveDevice removes one device from a group.
func (s *DeviceGroupService) RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	if err := s.repo.RemoveDevice(ctx, groupID, deviceID); err != nil {
		return fmt.Errorf("remove device from group: %w", err)
	}
	s.invalidateTreeCache()
	return nil
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

// CreateTopoNodesFromDevices creates topology nodes from devices table.
// Filters by technology (node_types) and site/domain IDs, with optional limit.
func (s *DeviceGroupService) CreateTopoNodesFromDevices(ctx context.Context, siteID, domainID string, nodeTypes []string, limit int) ([]TopoNode, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool not available")
	}

	// Build query to select devices
	base := storage.Psql.
		Select("serial_number", "technology", "status", "latitude", "longitude").
		From("devices").
		Where(sq.Eq{"deleted_at": nil})

	// Filter by technology if specified
	if len(nodeTypes) > 0 {
		var techFilter []interface{}
		for _, nt := range nodeTypes {
			// Map eNB/gNB to lte/nr
			if nt == "eNB" {
				techFilter = append(techFilter, "lte")
			} else if nt == "gNB" {
				techFilter = append(techFilter, "nr")
			} else {
				// For other types, try direct mapping
				techFilter = append(techFilter, nt)
			}
		}
		if len(techFilter) > 0 {
			base = base.Where(sq.Eq{"technology": techFilter})
		}
	}

	// Apply limit
	maxLimit := 100
	if limit > 0 && limit < maxLimit {
		base = base.Limit(uint64(limit))
	} else {
		base = base.Limit(uint64(maxLimit))
	}

	sql, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select devices SQL: %w", err)
	}

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	var nodes []TopoNode
	var x, y float64 = 50, 50
	gridCols := 8
	col := 0

	for rows.Next() {
		var serialNumber, technology, status string
		var latitude, longitude *float64

		if err := rows.Scan(&serialNumber, &technology, &status, &latitude, &longitude); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}

		// Determine node type based on technology
		nodeType := "eGW" // default
		if technology == "lte" {
			nodeType = "eNB"
		} else if technology == "nr" {
			nodeType = "gNB"
		} else if technology == "wifi" || technology == "cpe" {
			nodeType = "CPE"
		}

		// Determine node status based on device status
		nodeStatus := NodeOffline
		if status == "registered" || status == "active" {
			nodeStatus = NodeOnline
		}

		// Generate coordinates (simple grid layout)
		x = 50 + float64(col%gridCols)*80
		y = 50 + float64(col/gridCols)*80
		col++

		node := TopoNode{
			ID:       uuid.New(),
			Label:    serialNumber,
			NodeType: nodeType,
			X:        x,
			Y:        y,
			Status:   nodeStatus,
			DeviceSN: serialNumber,
		}

		// Set site_id if provided
		if siteID != "" {
			sid, err := uuid.Parse(siteID)
			if err == nil {
				node.SiteID = &sid
			}
		}

		// Set domain_id if provided
		if domainID != "" {
			did, err := uuid.Parse(domainID)
			if err == nil {
				node.DomainID = &did
			}
		}

		// Use geo coordinates if available
		if latitude != nil && longitude != nil {
			// Scale geo coords to canvas (rough approximation)
			node.X = *longitude
			node.Y = *latitude
		}

		nodes = append(nodes, node)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate device rows: %w", rows.Err())
	}

	// Batch insert nodes
	for i := range nodes {
		if err := s.topoNodeRepo.Create(ctx, &nodes[i]); err != nil {
			return nil, fmt.Errorf("create topo node %s: %w", nodes[i].Label, err)
		}
	}

	return nodes, nil
}
