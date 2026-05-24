package mml

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ============================================================
// console_service.go — T-0123-P1 Console 后端服务编排
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §N.2 / §N.5
//
// 职责（5 个 Console 端点的 service 入口）：
//   1. BuildGroupTree — 透传 GroupTreeRepository.BuildTree
//   2. GetCommandSubFields — 透传 SubFieldRepository.ListEnrichedByCommand + lang 派生
//   3. RenderMML — 调 mml_renderer 把 statement → MML 字符串（service 层加载 sub_fields）
//   4. ParseMML — 调 mml_parser 把 MML 字符串 → statements（lookup 注入自身）
//   5. ExecuteStatements — N 设备 × M statements fanout（T-0123-P1 D2 下）
// ============================================================

// ConsoleService 是 Console 后端的业务编排层。
type ConsoleService struct {
	treeRepo     GroupTreeRepository
	subFieldRepo SubFieldRepository
	commandRepo  CommandRepository
	// cmdParamRepo 用于 cmd.Params enrichment（commandRepo.GetByID 不查 params 列）。
	// 未装配时 attachParams 短路（保留旧行为 / 兼容测试 fixture 直接预填 cmd.Params）。
	cmdParamRepo CommandParamRepository
	// flatTreeRepo 是 Task #4 新增的扁平命令树仓储，nil 表示未装配
	// （老测试以及 v1 部署路径保持原戉行为）。BuildFlatGroupTree 未装配时
	// 返回明确错误，handler 映射为 503。
	flatTreeRepo FlatGroupTreeRepository
	// searchRepo 是 Bundle C 新增的命令搜索仓储；nil 表示未装配（503）。
	searchRepo SearchRepository
	// T-0170: 注入"device key (SN 或 UUID) → paramModelID"反查闭包；nil 时退化为全集（admin 视图）。
	// 设计哲学：param_mappings 是 product 实际支持 path 的真值源；缺映射 = 不支持。
	resolveParamModelByDevice func(ctx context.Context, deviceKey string) (*uuid.UUID, error)
	logger                    *zap.Logger
}

// NewConsoleService 构造 ConsoleService。
func NewConsoleService(
	treeRepo GroupTreeRepository,
	subFieldRepo SubFieldRepository,
	commandRepo CommandRepository,
	logger *zap.Logger,
) *ConsoleService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ConsoleService{
		treeRepo:     treeRepo,
		subFieldRepo: subFieldRepo,
		commandRepo:  commandRepo,
		logger:       logger.Named("mml-console-service"),
	}
}

// BuildGroupTree 透传 repo BuildTree。
func (s *ConsoleService) BuildGroupTree(ctx context.Context, rootCode, lang string) ([]GroupTreeNode, error) {
	return s.treeRepo.BuildTree(ctx, rootCode, lang)
}

// SetFlatTreeRepo 装配 Task #4 扁平命令树仓储。不走构造函数以避免贩及
// 现有 ~18 个测试点 NewConsoleService 的签名。provider 装配时调用一次。
func (s *ConsoleService) SetFlatTreeRepo(repo FlatGroupTreeRepository) {
	s.flatTreeRepo = repo
}

// SetSearchRepo 装配 Bundle C 命令搜索仓储；理由同 SetFlatTreeRepo。
func (s *ConsoleService) SetSearchRepo(repo SearchRepository) {
	s.searchRepo = repo
}

// SetParamModelByDeviceResolver 注入 deviceKey → paramModelID 反查（T-0170）。
// deviceKey 可以是 serial_number 或 UUID（resolver 自行判定）。
// 不注入时 sub_field 端点不按设备过滤（admin 视图等价）。
func (s *ConsoleService) SetParamModelByDeviceResolver(fn func(ctx context.Context, deviceKey string) (*uuid.UUID, error)) {
	s.resolveParamModelByDevice = fn
}

// SetCmdParamRepo 装配命令参数 enrichment 仓储；理由同 SetFlatTreeRepo。
//
// 修复 2026-05-22：旧版 ConsoleService 直接调 commandRepo.GetByID 拿 cmd，
// 但 PgCommandRepository.GetByID 不查 params 列，cmd.Params 永远是 nil；
// 下游 StructuredToStatement / buildLSTParamRefs / buildMODParamRefs 全部依赖
// cmd.Params → standardPath 反查全失败 → 误报 R-9.2 unknown_paths。
// 装配本 repo 后，attachParams 在 GetByID 之后填上 cmd.Params。
func (s *ConsoleService) SetCmdParamRepo(repo CommandParamRepository) {
	s.cmdParamRepo = repo
}

// attachParams 把 cmd.Params 从 cmdParamRepo 填上（enrichment）。
//
// 短路条件（任一满足即 no-op）：
//   - cmd 为 nil
//   - cmd.Params 已挂载（测试 fixture 直接预填的兼容路径）
//   - cmdParamRepo 未装配（保留 ConsoleService 老部署的退化兜底行为）
func (s *ConsoleService) attachParams(ctx context.Context, cmd *MMLCommand) error {
	if cmd == nil || len(cmd.Params) > 0 || s.cmdParamRepo == nil {
		return nil
	}
	paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, []uuid.UUID{cmd.ID})
	if err != nil {
		return fmt.Errorf("attach command %s params: %w", cmd.ID, err)
	}
	if params, ok := paramMap[cmd.ID]; ok {
		cmd.Params = params
	}
	return nil
}

// SearchCommandDTO 是 GET /mml/commands/search 响应单元（lang 派生 + 中文 i18n 解析）。
type SearchCommandDTO struct {
	CommandID     uuid.UUID `json:"command_id"`
	CommandCode   string    `json:"command_code"`
	LogicalCode   string    `json:"logical_code"`
	OperationType string    `json:"operation_type"`
	DisplayName   string    `json:"display_name"`
	LogicalName   string    `json:"logical_name"`
	GroupID       uuid.UUID `json:"group_id"`
	GroupCode     string    `json:"group_code"`
	GroupName     string    `json:"group_name"`
	ChapterCode   string    `json:"chapter_code"`
	MatchedPaths  []string  `json:"matched_paths"`
	MatchReasons  []string  `json:"match_reasons"`
}

// ErrSearchNotConfigured handler 据此返 503（searchRepo 未装配时）。
var ErrSearchNotConfigured = fmt.Errorf("search repo not configured")

// SearchCommands 透传 SearchRepository.SearchCommands + lang 派生 display_name / group_name。
func (s *ConsoleService) SearchCommands(ctx context.Context, query, lang string, limit int) ([]SearchCommandDTO, error) {
	if s.searchRepo == nil {
		return nil, ErrSearchNotConfigured
	}
	if lang == "" {
		lang = "zh-CN"
	}
	rows, err := s.searchRepo.SearchCommands(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]SearchCommandDTO, 0, len(rows))
	for _, r := range rows {
		logicalName := pickI18n(r.LogicalI18n, lang, "", "", r.LogicalCode)
		out = append(out, SearchCommandDTO{
			CommandID:     r.CommandID,
			CommandCode:   r.CommandCode,
			LogicalCode:   r.LogicalCode,
			OperationType: r.OperationType,
			DisplayName:   buildDisplayName(logicalName, r.OperationType, r.LogicalCode, lang),
			LogicalName:   logicalName,
			GroupID:       r.GroupID,
			GroupCode:     r.GroupCode,
			GroupName:     pickI18n(r.GroupNameI18n, lang, "", "", r.GroupCode),
			ChapterCode:   r.ChapterCode,
			MatchedPaths:  r.MatchedPaths,
			MatchReasons:  r.MatchReasons,
		})
	}
	return out, nil
}

// BuildFlatGroupTree 透传 FlatGroupTreeRepository.BuildFlatTree。未装配时返
// ErrFlatTreeNotConfigured，handler 映射为 503。
func (s *ConsoleService) BuildFlatGroupTree(ctx context.Context) ([]FlatGroup, error) {
	if s.flatTreeRepo == nil {
		return nil, ErrFlatTreeNotConfigured
	}
	return s.flatTreeRepo.BuildFlatTree(ctx)
}

// SubFieldDTO 是 GET /mml/commands/:id/sub-fields 端点的响应单元。
// 包装 MMLCommandSubFieldEnriched 加 lang 派生顶级 label / constraint_text。
type SubFieldDTO struct {
	ID                 uuid.UUID         `json:"id"`
	CommandID          uuid.UUID         `json:"command_id"`
	ParamID            uuid.UUID         `json:"param_id"`
	MMLCode            string            `json:"mml_code"`
	Label              string            `json:"label"` // lang 派生
	LabelI18n          map[string]string `json:"label_i18n"`
	Tr069Path          string            `json:"tr069_path"`
	ValueType          string            `json:"value_type"`
	AccessType         string            `json:"access_type"`
	IsObject           bool              `json:"is_object"`
	SupportsAdd        bool              `json:"supports_add"`
	SupportsDelete     bool              `json:"supports_delete"`
	ChangeApplies      string            `json:"change_applies"`
	ConstraintText     string            `json:"constraint_text"`
	ConstraintTextI18n map[string]string `json:"constraint_text_i18n"`
	DefaultValue       *string           `json:"default_value,omitempty"`
	JsRegex            *string           `json:"js_regex,omitempty"`
	DefaultSelected    bool              `json:"default_selected"`
	IsRequired         bool              `json:"is_required"`
	SortOrder          int               `json:"sort_order"`
	// Description 是 TR-181 path 的中文含义说明（来自 standard_params.description）。
	// 前端 MML 控制台 path 行 tooltip / 行内提示用；可为空。
	Description string `json:"description,omitempty"`
}

// GetCommandSubFields 加载命令的 sub_fields（含 JOIN standard_params 元数据），按 lang 派生
// 顶级 label / constraint_text。
func (s *ConsoleService) GetCommandSubFields(ctx context.Context, commandID uuid.UUID, deviceKey string, lang string) ([]SubFieldDTO, error) {
	if lang == "" {
		lang = "zh-CN"
	}
	// T-0170: deviceKey 非空时反查 paramModelID → repo SQL 叠加 param_mappings 过滤。
	// deviceKey 可以是 SN 或 UUID。反查闭包未注入 / 反查失败 → 退化为全集（不阻塞 admin 视图）。
	var paramModelID *uuid.UUID
	if deviceKey != "" && s.resolveParamModelByDevice != nil {
		pmID, err := s.resolveParamModelByDevice(ctx, deviceKey)
		if err != nil {
			s.logger.Warn("resolve param_model by device failed, fallback to unfiltered",
				zap.String("device_key", deviceKey), zap.Error(err))
		}
		paramModelID = pmID
	}
	enriched, err := s.subFieldRepo.ListEnrichedByCommand(ctx, commandID, paramModelID)
	if err != nil {
		return nil, fmt.Errorf("list enriched sub_fields: %w", err)
	}
	out := make([]SubFieldDTO, 0, len(enriched))
	for _, e := range enriched {
		out = append(out, SubFieldDTO{
			ID:                 e.ID,
			CommandID:          e.CommandID,
			ParamID:            e.ParamID,
			MMLCode:            e.MMLCode,
			LabelI18n:          e.LabelI18n,
			Label:              pickI18n(e.LabelI18n, lang, "", "", e.MMLCode),
			Tr069Path:          e.Tr069Path,
			ValueType:          e.ValueType,
			AccessType:         e.AccessType,
			IsObject:           e.IsObject,
			SupportsAdd:        e.SupportsAdd,
			SupportsDelete:     e.SupportsDelete,
			ChangeApplies:      e.ChangeApplies,
			ConstraintTextI18n: e.ConstraintTextI18n,
			ConstraintText:     pickI18n(e.ConstraintTextI18n, lang, "", "", ""),
			DefaultValue:       e.DefaultValue,
			JsRegex:            e.JsRegex,
			DefaultSelected:    e.DefaultSelected,
			IsRequired:         e.IsRequired,
			SortOrder:          e.SortOrder,
			Description:        e.Description,
		})
	}
	return out, nil
}

// RenderRequest 是 POST /mml/render 的请求体。
type RenderRequest struct {
	CommandID           uuid.UUID         `json:"command_id" binding:"required"`
	OperationType       string            `json:"operation_type" binding:"required,oneof=LST MOD ADD RMV"`
	SelectedSubFieldIDs []uuid.UUID       `json:"selected_sub_field_ids"`
	Values              map[string]string `json:"values"`
	RmvInstanceIndex    *int              `json:"rmv_instance_index,omitempty"`
}

// RenderMML 把 RenderRequest 渲染为 MML 字符串片段（不含末尾 ;，由前端按需追加）。
func (s *ConsoleService) RenderMML(ctx context.Context, req RenderRequest) (string, error) {
	cmd, err := s.commandRepo.GetByID(ctx, req.CommandID)
	if err != nil {
		return "", fmt.Errorf("get command %s: %w", req.CommandID, err)
	}
	if cmd == nil {
		return "", ErrCommandNotFound
	}

	subFields, err := s.subFieldRepo.ListByCommand(ctx, req.CommandID)
	if err != nil {
		return "", fmt.Errorf("list sub_fields: %w", err)
	}

	stmt := Statement{
		CommandID:           &req.CommandID,
		LogicalCode:         cmd.LogicalCode,
		OperationType:       req.OperationType,
		SelectedSubFieldIDs: req.SelectedSubFieldIDs,
		Values:              req.Values,
		RmvInstanceIndex:    req.RmvInstanceIndex,
	}

	// 命令的 logical_code 可能为空（admin 未填）— 兜底用 command_code 去 op 前缀
	if stmt.LogicalCode == "" {
		stmt.LogicalCode = deriveLogicalCodeFromCommandCode(cmd.CommandCode, cmd.OperationType)
	}

	return RenderStatement(stmt, subFields)
}

// ParseRequest 是 POST /mml/parse 的请求体。
type ParseRequest struct {
	MMLString string `json:"mml_string" binding:"required"`
	Lang      string `json:"lang"`
}

// ParseResponse 是 POST /mml/parse 的响应体。
type ParseResponse struct {
	Statements []Statement  `json:"statements"`
	ParseErrors []ParseError `json:"parse_errors"`
}

// ParseMML 调用 mml_parser，注入自身作 CommandLookup。
func (s *ConsoleService) ParseMML(ctx context.Context, req ParseRequest) (ParseResponse, error) {
	stmts, errs := ParseMMLString(ctx, req.MMLString, s)
	return ParseResponse{
		Statements:  stmts,
		ParseErrors: errs,
	}, nil
}

// LookupByLogicalCode 实现 CommandLookup 接口（让 parser 通过 service 反查命令）。
//
// 行为：
//   - 按 (op, logical_code) 查 mml_commands
//   - 命中 → 返 *MMLCommand + 该命令的 sub_fields
//   - 未命中 → ErrCommandNotFound
//   - 多命中 → ErrAmbiguousCommand（数据不一致信号）
func (s *ConsoleService) LookupByLogicalCode(ctx context.Context, op, logicalCode string) (*MMLCommand, []MMLCommandSubField, error) {
	// CommandRepository 没有 ListByLogicalCode；用 GetByCode 退化 — 但 GetByCode 用
	// command_code（含 op 前缀）。这里需补：先尝试 GetByCode(op + "_" + logical_code)
	// 作启发式（mmlstandardloader 命名约定）；将来加 ListByLogicalCode 时切过去。
	candidate := op + "_" + logicalCode
	cmd, err := s.commandRepo.GetByCode(ctx, candidate)
	if err != nil {
		// not found → 退化尝试 logical_code 直接做 command_code（admin 创建的命令可能这样）
		if errors.Is(err, ErrCommandNotFound) {
			cmd, err = s.commandRepo.GetByCode(ctx, logicalCode)
			if err != nil {
				return nil, nil, ErrCommandNotFound
			}
		} else {
			return nil, nil, fmt.Errorf("get command by code: %w", err)
		}
	}
	if cmd == nil {
		return nil, nil, ErrCommandNotFound
	}
	if cmd.OperationType != op {
		return nil, nil, ErrCommandNotFound
	}
	// 见 attachParams 文档：parser → resolveStatement → buildLSTParamRefs 整条链
	// 都依赖 cmd.Params，缺失会让 BuildTR069Params 拿到空 Tr069Path 失败。
	if err := s.attachParams(ctx, cmd); err != nil {
		return nil, nil, fmt.Errorf("attach params: %w", err)
	}

	subFields, err := s.subFieldRepo.ListByCommand(ctx, cmd.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list sub_fields: %w", err)
	}
	return cmd, subFields, nil
}

// deriveLogicalCodeFromCommandCode 从 command_code 派生 logical_code（去 op 前缀）。
//   "LST_DEVICE_INFO" → "DEVICE_INFO"
//   "DEVICE_INFO" (无前缀) → "DEVICE_INFO"
func deriveLogicalCodeFromCommandCode(commandCode, op string) string {
	prefix := op + "_"
	if len(commandCode) > len(prefix) && commandCode[:len(prefix)] == prefix {
		return commandCode[len(prefix):]
	}
	return commandCode
}
