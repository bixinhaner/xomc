package admin

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin/dictsource"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DictionaryService provides business logic for dictionary management.
//
// T-0182:增加 sourceRegistry / syncEngine 两个可选依赖。
//   - sourceRegistry 不为 nil 时,Create/Update 启用 source_* 字段校验,以及 ListDictionarySources / Preview / Refresh / SyncAll 4 个新接口。
//   - 为 nil 时所有 source 相关方法返 BusinessError "dictionary source feature disabled" — 保留运行时回退能力。
type DictionaryService struct {
	dictRepo       DictionaryRepository
	detailRepo     DictionaryDetailRepository
	sourceRegistry *dictsource.Registry
	syncEngine     *dictsource.SyncEngine
	log            *zap.Logger
}

// NewDictionaryService creates a new DictionaryService.
func NewDictionaryService(dictRepo DictionaryRepository, detailRepo DictionaryDetailRepository) *DictionaryService {
	return &DictionaryService{dictRepo: dictRepo, detailRepo: detailRepo, log: zap.NewNop()}
}

// SetSourceWiring 注入数据源 Registry + SyncEngine + logger。
// provider 启动期调用一次;v1 nil 检查保留运行时退化为"无数据源功能"模式。
func (s *DictionaryService) SetSourceWiring(reg *dictsource.Registry, engine *dictsource.SyncEngine, log *zap.Logger) {
	s.sourceRegistry = reg
	s.syncEngine = engine
	if log != nil {
		s.log = log
	}
}

// CreateDictionary creates a new dictionary entry.
//
// T-0182:如 source_* 三字段填写,校验白名单后写入并立即触发首次同步。
// 首次同步失败不阻塞创建 — 字典已落库,前端可看到 last_refresh_error,
// 用户可点"刷新"按钮重试。
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

	// T-0182 — 数据源校验(可选,需启用 sourceRegistry)。
	if s.sourceRegistry != nil {
		t, l, v, bound, err := validateSourceFields(s.sourceRegistry, req.SourceTable, req.SourceLabelField, req.SourceValueField, false)
		if err != nil {
			return nil, err
		}
		if bound {
			dict.SourceTable = &t
			dict.SourceLabelField = &l
			dict.SourceValueField = &v
		}
	}

	if err := s.dictRepo.Create(ctx, dict); err != nil {
		return nil, fmt.Errorf("create dictionary: %w", err)
	}

	// T-0182 — 首次同步(异步? 同步? v1 决策:同步)。
	// 同步语义让前端立刻看到 auto 项,体验最直接;失败也不影响字典创建本身。
	if dict.SourceTable != nil && s.syncEngine != nil {
		s.runSyncOne(ctx, dict)
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
//
// T-0182 source 字段语义(参 model.go UpdateDictionaryRequest 注释):
//   - 三个字段都不传 → 不动数据源绑定
//   - 三个字段都传非空 → 绑定/切换;切换时清空旧 auto 项 + 立即触发同步
//   - 三个字段都传空字符串 → 解绑,删所有 auto 项,保留 manual 项
//   - 部分填写 → 400
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

	// T-0182 — 数据源 update 路径。
	sourceChanged := false
	sourceCleared := false
	if s.sourceRegistry != nil && (req.SourceTable != nil || req.SourceLabelField != nil || req.SourceValueField != nil) {
		t, l, v, bound, err := validateSourceFields(s.sourceRegistry, req.SourceTable, req.SourceLabelField, req.SourceValueField, true)
		if err != nil {
			return nil, err
		}
		if bound {
			// 绑定 / 切换
			oldTable := derefString(dict.SourceTable)
			if oldTable != t || derefString(dict.SourceLabelField) != l || derefString(dict.SourceValueField) != v {
				sourceChanged = true
			}
			dict.SourceTable = &t
			dict.SourceLabelField = &l
			dict.SourceValueField = &v
		} else {
			// 解绑(三字段都空字符串)
			if dict.SourceTable != nil {
				sourceCleared = true
			}
			dict.SourceTable = nil
			dict.SourceLabelField = nil
			dict.SourceValueField = nil
		}
	}

	if err := s.dictRepo.Update(ctx, dict); err != nil {
		return nil, fmt.Errorf("update dictionary: %w", err)
	}

	// 解绑 → 清空 auto 项(manual 项保留)。
	if sourceCleared {
		if _, err := s.detailRepo.DeleteAutoNotIn(ctx, dict.ID, nil); err != nil {
			// 解绑过程已经写库,删 auto 失败只记日志不阻塞返回
			s.log.Warn("dict_source_unbind_cleanup_failed",
				zap.Int64("dict_id", dict.ID), zap.Error(err))
		}
	}

	// 绑定/切换 → 删除旧 auto 项 + 触发同步。
	if sourceChanged && s.syncEngine != nil {
		// 切换源表:旧 auto 项与新源 value 大概率全不匹配,DeleteAutoNotIn(keep=nil)
		// 直接全软删,后续 SyncOne 重建。
		if _, err := s.detailRepo.DeleteAutoNotIn(ctx, dict.ID, nil); err != nil {
			s.log.Warn("dict_source_switch_cleanup_failed",
				zap.Int64("dict_id", dict.ID), zap.Error(err))
		}
		s.runSyncOne(ctx, dict)
	}

	return dict, nil
}

// derefString 解引用 *string;nil 视为空串。
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
//
// T-0182:托管字典(source_table != NULL)只接受 origin='manual' 项,且 auto 项不能
// 通过前端入口创建 — handler 不向用户开放写 origin,service 强制为 'manual'。
// (托管字典 v1 是否允许补录 manual 项?方案 A 接受 — UI 隐藏 + 按钮,但 service
// 不直接禁绝,留作未来"运营临时补录"的口子。)
func (s *DictionaryService) CreateDictionaryDetail(ctx context.Context, req CreateDictionaryDetailRequest) (*DictionaryDetail, error) {
	// Verify parent dictionary exists
	parentDict, err := s.dictRepo.GetByID(ctx, req.SysDictionaryID)
	if err != nil {
		return nil, commonerrors.NewBusinessError(7003, "parent dictionary not found", err)
	}
	if err := validateNetworkTypeDictionaryValue(parentDict.Type, req.Value); err != nil {
		return nil, err
	}

	detail := &DictionaryDetail{
		Label:           req.Label,
		LabelI18n:       req.LabelI18n,
		Value:           req.Value,
		Extend:          req.Extend,
		Status:          true,
		Sort:            0,
		SysDictionaryID: req.SysDictionaryID,
		Level:           0,
		Origin:          OriginManual, // 前端入口只能创建 manual 项,auto 项由 SyncEngine 创建
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
//   - 防环：新 parent_id 不能等于 self.id 或 self 的任一后代。
//   - 同字典：新父明细的 sys_dictionary_id 必须与 self 一致。
//   - 深度：max(子树 level + delta) < MaxDictionaryDetailDepth (3)。
//   - 级联：self 自身和所有后代的 level 同步加 delta。
func (s *DictionaryService) UpdateDictionaryDetail(ctx context.Context, req UpdateDictionaryDetailRequest) (*DictionaryDetail, error) {
	detail, err := s.detailRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get detail for update: %w", err)
	}

	if req.Label != nil {
		detail.Label = *req.Label
	}
	if req.LabelI18n != nil {
		detail.LabelI18n = req.LabelI18n
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
	dictionary, err := s.dictRepo.GetByID(ctx, detail.SysDictionaryID)
	if err != nil {
		return nil, commonerrors.NewBusinessError(7003, "parent dictionary not found", err)
	}
	if err := validateNetworkTypeDictionaryValue(dictionary.Type, detail.Value); err != nil {
		return nil, err
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

func validateNetworkTypeDictionaryValue(dictionaryType, value string) error {
	if dictionaryType != "network_type" {
		return nil
	}
	switch value {
	case "lte", "nr", "gsm":
		return nil
	default:
		return commonerrors.NewBusinessError(7014,
			"network_type value must be one of: lte, nr, gsm", nil)
	}
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

// ====================================================================
// T-0182 数据源相关方法
// ====================================================================

// ListDictionarySources 返白名单全部表 + 字段(GET /sysDictionary/sources)。
func (s *DictionaryService) ListDictionarySources(ctx context.Context) ([]dictsource.TableSpec, error) {
	if s.sourceRegistry == nil {
		return nil, commonerrors.NewBusinessError(errCodeSourceSyncFailed,
			"dictionary source feature disabled", nil)
	}
	return s.sourceRegistry.ListTables(), nil
}

// PreviewDictionarySource 调用 SyncEngine.Preview,返前 N 条 + 总数。
func (s *DictionaryService) PreviewDictionarySource(ctx context.Context, table, labelField, valueField string, limit int) (*PreviewSourceResponse, error) {
	if s.syncEngine == nil {
		return nil, commonerrors.NewBusinessError(errCodeSourceSyncFailed,
			"dictionary source feature disabled", nil)
	}
	pairs, total, err := s.syncEngine.Preview(ctx, table, labelField, valueField, limit)
	if err != nil {
		return nil, translateEngineError(err)
	}
	out := &PreviewSourceResponse{
		Rows:  make([]PreviewSourceRow, len(pairs)),
		Total: total,
	}
	for i, p := range pairs {
		out.Rows[i] = PreviewSourceRow{Label: p.Label, Value: p.Value}
	}
	return out, nil
}

// RefreshDictionarySource 手动触发单字典同步。
// 字典必须已绑定数据源,否则返 400。
func (s *DictionaryService) RefreshDictionarySource(ctx context.Context, dictID int64) (*RefreshSourceResponse, error) {
	if s.syncEngine == nil {
		return nil, commonerrors.NewBusinessError(errCodeSourceSyncFailed,
			"dictionary source feature disabled", nil)
	}
	dict, err := s.dictRepo.GetByID(ctx, dictID)
	if err != nil {
		return nil, fmt.Errorf("get dictionary for refresh: %w", err)
	}
	if dict.SourceTable == nil {
		// 用户对未绑源字典点刷新属参数类业务错——包 ErrInvalidInput 使 handler 映射 400 而非 500。
		return nil, commonerrors.NewBusinessError(errCodeSourcePartialFields,
			"dictionary has no source binding", commonerrors.ErrInvalidInput)
	}
	res, err := s.runSyncOneStrict(ctx, dict)
	if err != nil {
		return nil, translateEngineError(err)
	}
	return &RefreshSourceResponse{
		Inserted:   res.Inserted,
		Updated:    res.Updated,
		Deleted:    res.Deleted,
		Total:      res.TotalAuto,
		DurationMS: res.DurationMS,
	}, nil
}

// SyncSourceBoundAll 是 worker daily cron 入口(P2 接入)。
// 遍历所有 source_table != NULL AND status=true 的字典,调 SyncEngine.SyncAll。
// 返 (ok, failed)。单字典失败不中断,失败原因写 last_refresh_error。
func (s *DictionaryService) SyncSourceBoundAll(ctx context.Context) (ok, failed int) {
	if s.syncEngine == nil {
		s.log.Warn("dict_source_daily_skip_engine_nil")
		return 0, 0
	}
	return s.syncEngine.SyncAll(ctx, s.listSourceBoundForSync)
}

// RefreshSourceBoundByTable 是「三库导入 XML 后自动刷新」入口(T-0182 / #241)。
// 找到所有绑定 sourceTable 的源绑定字典,逐个宽容同步(runSyncOne:失败只写
// last_refresh_* + 日志,不返错)——导入主流程不应因字典刷新失败而失败,daily cron 兜底。
// 返回尝试刷新的字典数;引擎未启用(无白名单)或该表无绑定字典时返 (0, nil),均非错误。
func (s *DictionaryService) RefreshSourceBoundByTable(ctx context.Context, sourceTable string) (int, error) {
	if s.syncEngine == nil {
		return 0, nil
	}
	dicts, err := s.dictRepo.ListSourceBound(ctx)
	if err != nil {
		return 0, fmt.Errorf("list source-bound dicts for %q: %w", sourceTable, err)
	}
	matched := dictsMatchingSourceTable(dicts, sourceTable)
	for i := range matched {
		s.runSyncOne(ctx, &matched[i])
	}
	return len(matched), nil
}

// dictsMatchingSourceTable 从源绑定字典集合中筛出绑定 sourceTable 的字典。
// 抽成纯函数便于单测筛选逻辑(忽略未绑定的手工字典与异表项)。
func dictsMatchingSourceTable(dicts []Dictionary, sourceTable string) []Dictionary {
	var out []Dictionary
	for i := range dicts {
		if dicts[i].SourceTable != nil && *dicts[i].SourceTable == sourceTable {
			out = append(out, dicts[i])
		}
	}
	return out
}

// listSourceBoundForSync 把 DB Dictionary 行映射为 dictsource.SyncDict(SyncEngine 接口形状)。
func (s *DictionaryService) listSourceBoundForSync(ctx context.Context) ([]dictsource.SyncDict, error) {
	dicts, err := s.dictRepo.ListSourceBound(ctx)
	if err != nil {
		return nil, fmt.Errorf("list source-bound dicts: %w", err)
	}
	out := make([]dictsource.SyncDict, 0, len(dicts))
	for _, d := range dicts {
		// 防御:DB 应保证三字段同时非空(由 service 写入逻辑保证),稳妥仍判一下。
		if d.SourceTable == nil || d.SourceLabelField == nil || d.SourceValueField == nil {
			continue
		}
		out = append(out, dictsource.SyncDict{
			ID:          d.ID,
			Name:        d.Name,
			SourceTable: *d.SourceTable,
			LabelField:  *d.SourceLabelField,
			ValueField:  *d.SourceValueField,
		})
	}
	return out, nil
}

// runSyncOne 是 Create/Update 路径用的"宽容同步":失败只记日志 + 写 last_refresh_*。
// 不影响 Create/Update 的成功返回。
func (s *DictionaryService) runSyncOne(ctx context.Context, dict *Dictionary) {
	if s.syncEngine == nil || dict.SourceTable == nil {
		return
	}
	sd := dictsource.SyncDict{
		ID:          dict.ID,
		Name:        dict.Name,
		SourceTable: *dict.SourceTable,
		LabelField:  derefString(dict.SourceLabelField),
		ValueField:  derefString(dict.SourceValueField),
	}
	res, err := s.syncEngine.SyncOne(ctx, sd)
	if err != nil {
		s.log.Warn("dict_source_initial_sync_failed",
			zap.Int64("dict_id", dict.ID), zap.Error(err))
		_ = s.dictRepo.UpdateRefreshMetadata(ctx, dict.ID, "failed", truncateErrMsg(err.Error()), 0)
		return
	}
	_ = s.dictRepo.UpdateRefreshMetadata(ctx, dict.ID, "ok", "", res.TotalAuto)
}

// runSyncOneStrict 是 RefreshSource(用户手动)路径用的"严格同步":失败返 err 让前端
// 看到 toast;同时也写 last_refresh_*。
func (s *DictionaryService) runSyncOneStrict(ctx context.Context, dict *Dictionary) (*dictsource.SyncResult, error) {
	sd := dictsource.SyncDict{
		ID:          dict.ID,
		Name:        dict.Name,
		SourceTable: derefString(dict.SourceTable),
		LabelField:  derefString(dict.SourceLabelField),
		ValueField:  derefString(dict.SourceValueField),
	}
	res, err := s.syncEngine.SyncOne(ctx, sd)
	if err != nil {
		_ = s.dictRepo.UpdateRefreshMetadata(ctx, dict.ID, "failed", truncateErrMsg(err.Error()), 0)
		return nil, err
	}
	_ = s.dictRepo.UpdateRefreshMetadata(ctx, dict.ID, "ok", "", res.TotalAuto)
	return &res, nil
}

// truncateErrMsg 把错误摘要截到 500 字符(避免 last_refresh_error 列被超长字符串撑爆)。
func truncateErrMsg(msg string) string {
	const maxLen = 500
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen]
}
