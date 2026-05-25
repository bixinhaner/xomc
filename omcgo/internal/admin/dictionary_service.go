package admin

import (
	"context"
	"fmt"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DictionaryService provides business logic for dictionary management.
type DictionaryService struct {
	dictRepo   DictionaryRepository
	detailRepo DictionaryDetailRepository
}

// NewDictionaryService creates a new DictionaryService.
func NewDictionaryService(dictRepo DictionaryRepository, detailRepo DictionaryDetailRepository) *DictionaryService {
	return &DictionaryService{dictRepo: dictRepo, detailRepo: detailRepo}
}

// CreateDictionary creates a new dictionary entry.
func (s *DictionaryService) CreateDictionary(ctx context.Context, req CreateDictionaryRequest) (*Dictionary, error) {
	dict := &Dictionary{
		Name:        req.Name,
		Type:        req.Type,
		Status:      true,
		Description: req.Description,
	}
	if req.Status != nil {
		dict.Status = *req.Status
	}

	if err := s.dictRepo.Create(ctx, dict); err != nil {
		return nil, fmt.Errorf("create dictionary: %w", err)
	}
	return dict, nil
}

// GetDictionaryByType returns a dictionary with its active details by type.
func (s *DictionaryService) GetDictionaryByType(ctx context.Context, dictType string) (*Dictionary, error) {
	return s.dictRepo.GetByType(ctx, dictType)
}

// ListDictionaries returns all dictionaries (without details).
func (s *DictionaryService) ListDictionaries(ctx context.Context) ([]Dictionary, error) {
	return s.dictRepo.List(ctx)
}

// UpdateDictionary updates an existing dictionary.
func (s *DictionaryService) UpdateDictionary(ctx context.Context, req UpdateDictionaryRequest) (*Dictionary, error) {
	dict, err := s.dictRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get dictionary for update: %w", err)
	}

	if req.Name != nil {
		dict.Name = *req.Name
	}
	if req.Type != nil {
		dict.Type = *req.Type
	}
	if req.Status != nil {
		dict.Status = *req.Status
	}
	if req.Description != nil {
		dict.Description = *req.Description
	}

	if err := s.dictRepo.Update(ctx, dict); err != nil {
		return nil, fmt.Errorf("update dictionary: %w", err)
	}
	return dict, nil
}

// DeleteDictionary soft-deletes a dictionary and its details.
func (s *DictionaryService) DeleteDictionary(ctx context.Context, id int64) error {
	return s.dictRepo.Delete(ctx, id)
}

// CreateDictionaryDetail creates a new dictionary detail entry.
//
// PRD §10：支持 ParentID 创建子项，校验：
//   - 父明细存在且属于同一 sys_dictionary
//   - 新明细 level = parent.level + 1，且 level < MaxDictionaryDetailDepth (3)
func (s *DictionaryService) CreateDictionaryDetail(ctx context.Context, req CreateDictionaryDetailRequest) (*DictionaryDetail, error) {
	// Verify parent dictionary exists
	if _, err := s.dictRepo.GetByID(ctx, req.SysDictionaryID); err != nil {
		return nil, commonerrors.NewBusinessError(7003, "parent dictionary not found", err)
	}

	detail := &DictionaryDetail{
		Label:           req.Label,
		Value:           req.Value,
		Extend:          req.Extend,
		Status:          true,
		Sort:            0,
		SysDictionaryID: req.SysDictionaryID,
		Level:           0,
	}
	if req.Status != nil {
		detail.Status = *req.Status
	}
	if req.Sort != nil {
		detail.Sort = *req.Sort
	}

	// PRD §10：父明细解析 + 深度校验
	if req.ParentID != nil {
		parent, err := s.detailRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, commonerrors.NewBusinessError(7011, "parent detail not found", err)
		}
		if parent.SysDictionaryID != req.SysDictionaryID {
			return nil, commonerrors.NewBusinessError(7012, "parent detail belongs to a different dictionary", nil)
		}
		if parent.Level+1 >= MaxDictionaryDetailDepth {
			return nil, commonerrors.NewBusinessError(7013,
				fmt.Sprintf("max dictionary depth %d exceeded", MaxDictionaryDetailDepth), nil)
		}
		detail.ParentID = req.ParentID
		detail.Level = parent.Level + 1
	}

	if err := s.detailRepo.Create(ctx, detail); err != nil {
		return nil, fmt.Errorf("create dictionary detail: %w", err)
	}
	return detail, nil
}

// GetDictionaryDetail returns a dictionary detail by ID.
func (s *DictionaryService) GetDictionaryDetail(ctx context.Context, id int64) (*DictionaryDetail, error) {
	return s.detailRepo.GetByID(ctx, id)
}

// ListDictionaryDetails returns filtered, paginated dictionary details.
func (s *DictionaryService) ListDictionaryDetails(ctx context.Context, req DictionaryDetailListRequest) (*model.ListResponse[DictionaryDetail], error) {
	items, total, err := s.detailRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []DictionaryDetail{}
	}
	return model.NewListResponse(items, total, req.Page, req.PageSize), nil
}

// UpdateDictionaryDetail updates an existing dictionary detail.
//
// PRD §10：换父逻辑（move-subtree）
//   - 不在请求里出现 parent_id 时：保持原 parent_id 不动（约定：前端始终把当前值
//     回填到 form 一并提交，因此 nil 视为「无意改动」）。语义见 model.go 注释。
//   - 出现 parent_id（含 null）时按以下规则：
//     * 防环：新 parent_id 不能等于 self.id 或 self 的任一后代。
//     * 同字典：新父明细的 sys_dictionary_id 必须与 self 一致。
//     * 深度：max(子树 level + delta) < MaxDictionaryDetailDepth (3)。
//     * 级联：self 自身和所有后代的 level 同步加 delta。
func (s *DictionaryService) UpdateDictionaryDetail(ctx context.Context, req UpdateDictionaryDetailRequest) (*DictionaryDetail, error) {
	detail, err := s.detailRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get detail for update: %w", err)
	}

	if req.Label != nil {
		detail.Label = *req.Label
	}
	if req.Value != nil {
		detail.Value = *req.Value
	}
	if req.Extend != nil {
		detail.Extend = *req.Extend
	}
	if req.Status != nil {
		detail.Status = *req.Status
	}
	if req.Sort != nil {
		detail.Sort = *req.Sort
	}
	if req.SysDictionaryID != nil {
		// Verify new parent dictionary exists
		if _, err := s.dictRepo.GetByID(ctx, *req.SysDictionaryID); err != nil {
			return nil, commonerrors.NewBusinessError(7003, "parent dictionary not found", err)
		}
		detail.SysDictionaryID = *req.SysDictionaryID
	}

	// PRD §10：换父 + 子树级联 level 调整
	if err := s.applyParentChange(ctx, detail, req.ParentID); err != nil {
		return nil, err
	}

	if err := s.detailRepo.Update(ctx, detail); err != nil {
		return nil, fmt.Errorf("update dictionary detail: %w", err)
	}
	return detail, nil
}

// applyParentChange 处理换父：解析新 level + 校验防环/深度，然后级联调整子树 level。
// detail 的 ParentID/Level 字段会被原地更新；调用者负责把更新写回数据库。
func (s *DictionaryService) applyParentChange(ctx context.Context, detail *DictionaryDetail, newParentID *int64) error {
	// 比较「新 parent_id」与「当前 parent_id」是否一致；一致则不动。
	currentParentID := detail.ParentID
	if eqOptInt64(newParentID, currentParentID) {
		return nil
	}

	// 计算 new level
	var newLevel int
	if newParentID == nil {
		// 切顶层
		newLevel = 0
	} else {
		// 设新父
		parent, err := s.detailRepo.GetByID(ctx, *newParentID)
		if err != nil {
			return commonerrors.NewBusinessError(7011, "parent detail not found", err)
		}
		if parent.SysDictionaryID != detail.SysDictionaryID {
			return commonerrors.NewBusinessError(7012, "parent detail belongs to a different dictionary", nil)
		}
		if *newParentID == detail.ID {
			return commonerrors.NewBusinessError(7014, "cannot set self as parent", nil)
		}
		newLevel = parent.Level + 1
	}

	delta := newLevel - detail.Level

	// 取整子树（含自身）
	subtreeIDs, err := s.detailRepo.ListSubtreeIDs(ctx, detail.ID)
	if err != nil {
		return fmt.Errorf("list subtree for parent change: %w", err)
	}

	// 防环：新父不能在自己的子树（含自身）中
	if newParentID != nil {
		for _, id := range subtreeIDs {
			if id == *newParentID {
				return commonerrors.NewBusinessError(7015, "cannot move subtree under its own descendant", nil)
			}
		}
	}

	// 深度上限：移动后任何节点 level 必须 < MaxDictionaryDetailDepth (3)。
	// 当前最大相对 level（相对 self）= subtreeMax - detail.Level；新位置后绝对 level = newLevel + 相对，必须 < 3。
	// 简化：取 subtreeIDs 当前最大 level，加 delta 即移动后最大 level。
	if delta != 0 {
		// 加载子树 level 列表来算 max（小批量；MaxDepth=3 的约束保证子树规模可控）。
		maxLevel := detail.Level
		for _, id := range subtreeIDs {
			if id == detail.ID {
				continue
			}
			d, err := s.detailRepo.GetByID(ctx, id)
			if err != nil {
				return fmt.Errorf("load subtree level: %w", err)
			}
			if d.Level > maxLevel {
				maxLevel = d.Level
			}
		}
		if maxLevel+delta >= MaxDictionaryDetailDepth {
			return commonerrors.NewBusinessError(7013,
				fmt.Sprintf("move would exceed max depth %d", MaxDictionaryDetailDepth), nil)
		}
	}

	// 给除 self 外的子孙 ID 集合先平移 level（self 由 detailRepo.Update 一起写）。
	descendantIDs := make([]int64, 0, len(subtreeIDs))
	for _, id := range subtreeIDs {
		if id != detail.ID {
			descendantIDs = append(descendantIDs, id)
		}
	}
	if err := s.detailRepo.ShiftLevelDelta(ctx, descendantIDs, delta); err != nil {
		return fmt.Errorf("cascade level shift: %w", err)
	}

	// 更新 self 的 parent_id + level（写回由调用者 detailRepo.Update 完成）
	detail.ParentID = newParentID
	detail.Level = newLevel
	return nil
}

// eqOptInt64 比较两个 *int64：nil == nil 返 true；nil 与非 nil 返 false；都非 nil 比较解引用值。
func eqOptInt64(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// DeleteDictionaryDetail soft-deletes a dictionary detail.
func (s *DictionaryService) DeleteDictionaryDetail(ctx context.Context, id int64) error {
	return s.detailRepo.Delete(ctx, id)
}

// BatchGetDicts retrieves multiple dictionaries by their types in a single call.
// Returns a map where the key is the dictionary type and the value is the dictionary with its details.
func (s *DictionaryService) BatchGetDicts(ctx context.Context, dictTypes []string) (map[string]*Dictionary, error) {
	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	uniqueTypes := make([]string, 0, len(dictTypes))
	for _, t := range dictTypes {
		t = trimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			uniqueTypes = append(uniqueTypes, t)
		}
	}

	if len(uniqueTypes) == 0 {
		return nil, commonerrors.NewBusinessError(7001, "at least one dictionary type is required", nil)
	}

	result := make(map[string]*Dictionary, len(uniqueTypes))

	// Fetch each dictionary - repository is expected to have efficient caching
	for _, dictType := range uniqueTypes {
		dict, err := s.dictRepo.GetByType(ctx, dictType)
		if err == nil && dict != nil {
			result[dictType] = dict
		}
		// Silently skip not-found dictionaries to allow partial results
	}

	return result, nil
}

func trimSpace(s string) string {
	// Simple trimSpace implementation to avoid importing strings
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
