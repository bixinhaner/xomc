package mml

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ============================================================
// admin_service.go — T-0123-P0 catalog 管理服务层
//
// 守护逻辑（§M.4.3 + Q2=C 决议）：
//   - Update 时检查 existing.CatalogProtected → 锁定关键字段
//   - Delete 时检查 existing.CatalogProtected → 拒绝
//   - Delete param 时检查 sub_fields 引用 → 0 才允许
//   - Delete group 时检查 commands 是否空 → 空才允许
//   - catalog_protected / source / is_writable / tr069_path 不暴露给 PATCH
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §M.4.3 / §M.5.3
//
// 接口约束：所有写方法签名 ctx context.Context 在前；错误用 fmt.Errorf 包装。
// ============================================================

// AdminAuditWriter 是 audit log 写入的小接口（消费者驱动）。
// service 层在每次写操作后调用 Write(ctx, entry) 落审计；
// 实际实现由 internal/admin/auditlog 提供（cross-module audit pkg from T-0063）。
type AdminAuditWriter interface {
	Write(ctx context.Context, op, resource string, detail map[string]any)
}

// noopAuditWriter 用于 DI 未注入 audit 时的兜底。
type noopAuditWriter struct{}

func (noopAuditWriter) Write(ctx context.Context, op, resource string, detail map[string]any) {}

// AdminService 提供 mml catalog 管理的业务编排（守护 + audit + log）。
type AdminService struct {
	groupRepo    AdminGroupRepository
	commandRepo  AdminCommandRepository
	subFieldRepo SubFieldRepository
	paramRepo    AdminParamRepository
	audit        AdminAuditWriter
	logger       *zap.Logger
}

// NewAdminService 构造 AdminService。
// audit 可为 nil（用 noopAuditWriter 兜底）。
func NewAdminService(
	groupRepo AdminGroupRepository,
	commandRepo AdminCommandRepository,
	subFieldRepo SubFieldRepository,
	paramRepo AdminParamRepository,
	audit AdminAuditWriter,
	logger *zap.Logger,
) *AdminService {
	if audit == nil {
		audit = noopAuditWriter{}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AdminService{
		groupRepo:    groupRepo,
		commandRepo:  commandRepo,
		subFieldRepo: subFieldRepo,
		paramRepo:    paramRepo,
		audit:        audit,
		logger:       logger.Named("mml-admin-service"),
	}
}

// ============================================================
// Group
// ============================================================

// CreateGroupReq 是 POST /admin/groups 的请求体。
type CreateGroupReq struct {
	GroupCode     string `json:"group_code" binding:"required,max=255"`
	GroupNameZh   string `json:"group_name_zh" binding:"max=500"`
	GroupNameEn   string `json:"group_name_en" binding:"max=500"`
	ParamVersion  string `json:"param_version" binding:"required,max=50"`
	DisplayOrder  int    `json:"display_order"`
}

// UpdateGroupReq 是 PATCH /admin/groups/:id 的请求体。
// 注意：catalog_protected / source / param_version 不可改（Q2=C）。
type UpdateGroupReq struct {
	GroupNameZh  *string `json:"group_name_zh,omitempty"`
	GroupNameEn  *string `json:"group_name_en,omitempty"`
	DisplayOrder *int    `json:"display_order,omitempty"`
}

// CreateGroup 新建一个 admin 来源的 group。
func (s *AdminService) CreateGroup(ctx context.Context, req CreateGroupReq) (*ParamGroup, error) {
	g := &ParamGroup{
		GroupCode:        req.GroupCode,
		GroupNameZh:      req.GroupNameZh,
		GroupNameEn:      req.GroupNameEn,
		ParamVersion:     req.ParamVersion,
		DisplayOrder:     req.DisplayOrder,
		Source:           SourceAdmin,
		CatalogProtected: false,
	}
	if err := s.groupRepo.Create(ctx, g); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.group.created", "group:"+g.ID.String(), map[string]any{
		"group_code": g.GroupCode,
		"source":     g.Source,
	})
	return g, nil
}

// UpdateGroup 更新一个 group 的可编辑字段。catalog_protected=true 时仅允许 i18n + display_order。
func (s *AdminService) UpdateGroup(ctx context.Context, id uuid.UUID, req UpdateGroupReq) (*ParamGroup, error) {
	existing, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	// 当前实现 i18n / display_order 都属"非锁定"字段，protected 也可改。
	// 真正的锁定字段（group_code / param_version / source / catalog_protected）本就不在 req 中。
	if req.GroupNameZh != nil {
		existing.GroupNameZh = *req.GroupNameZh
	}
	if req.GroupNameEn != nil {
		existing.GroupNameEn = *req.GroupNameEn
	}
	if req.DisplayOrder != nil {
		existing.DisplayOrder = *req.DisplayOrder
	}
	if err := s.groupRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.group.updated", "group:"+id.String(), map[string]any{
		"group_code": existing.GroupCode,
	})
	return existing, nil
}

// DeleteGroup 删除一个 group。
//   - catalog_protected=true → ErrCatalogProtected
//   - 还含 commands → ErrGroupNotEmpty
func (s *AdminService) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	existing, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if existing.CatalogProtected {
		return ErrCatalogProtected
	}
	count, err := s.groupRepo.CountCommandsByGroup(ctx, id)
	if err != nil {
		return fmt.Errorf("count commands by group: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("group has %d commands: %w", count, ErrGroupNotEmpty)
	}
	if err := s.groupRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.group.deleted", "group:"+id.String(), map[string]any{
		"group_code": existing.GroupCode,
	})
	return nil
}

// ============================================================
// Command
// ============================================================

// CreateCommandReq 是 POST /admin/commands 的请求体。
type CreateCommandReq struct {
	CommandName     string            `json:"command_name" binding:"required,max=500"`
	CommandCode     string            `json:"command_code" binding:"required,max=255"`
	Category        string            `json:"category" binding:"max=10"`
	Description     string            `json:"description"`
	RPCMethod       string            `json:"rpc_method" binding:"max=50"`
	OperationType   string            `json:"operation_type" binding:"required,oneof=LST MOD ADD RMV"`
	HelpDoc         string            `json:"help_doc"`
	Notes           string            `json:"notes"`
	TargetObject    string            `json:"target_object"`
	GroupID         *uuid.UUID        `json:"group_id"`
	CommandNameI18n map[string]string `json:"command_name_i18n"`
	RequireConfirm  bool              `json:"require_confirm"`
	ConfirmMsgI18n  map[string]string `json:"confirm_msg_i18n"`
	LogicalCode     string            `json:"logical_code" binding:"max=100"`
	LogicalNameI18n map[string]string `json:"logical_name_i18n"`
}

// UpdateCommandReq 是 PATCH /admin/commands/:id 的请求体。
// 不可改字段：command_code / operation_type / source / catalog_protected（Q2=C）。
type UpdateCommandReq struct {
	CommandName     *string            `json:"command_name,omitempty"`
	Category        *string            `json:"category,omitempty"`
	Description     *string            `json:"description,omitempty"`
	HelpDoc         *string            `json:"help_doc,omitempty"`
	Notes           *string            `json:"notes,omitempty"`
	TargetObject    *string            `json:"target_object,omitempty"`
	GroupID         *uuid.UUID         `json:"group_id,omitempty"`
	CommandNameI18n *map[string]string `json:"command_name_i18n,omitempty"`
	RequireConfirm  *bool              `json:"require_confirm,omitempty"`
	ConfirmMsgI18n  *map[string]string `json:"confirm_msg_i18n,omitempty"`
	LogicalCode     *string            `json:"logical_code,omitempty"`
	LogicalNameI18n *map[string]string `json:"logical_name_i18n,omitempty"`
}

// CreateCommand 新建 admin 来源命令。
func (s *AdminService) CreateCommand(ctx context.Context, req CreateCommandReq) (*MMLCommand, error) {
	c := &MMLCommand{
		CommandName:      req.CommandName,
		CommandCode:      req.CommandCode,
		Category:         req.Category,
		Description:      req.Description,
		RPCMethod:        req.RPCMethod,
		OperationType:    req.OperationType,
		HelpDoc:          req.HelpDoc,
		Notes:            req.Notes,
		TargetPaths:      []string{}, // 由 sub_fields trigger 派生
		TargetObject:     req.TargetObject,
		GroupID:          req.GroupID,
		CommandNameI18n:  req.CommandNameI18n,
		RequireConfirm:   req.RequireConfirm,
		ConfirmMsgI18n:   req.ConfirmMsgI18n,
		LogicalCode:      req.LogicalCode,
		LogicalNameI18n:  req.LogicalNameI18n,
		Source:           SourceAdmin,
		CatalogProtected: false,
	}
	if err := s.commandRepo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.command.created", "command:"+c.ID.String(), map[string]any{
		"command_code":   c.CommandCode,
		"operation_type": c.OperationType,
		"logical_code":   c.LogicalCode,
	})
	return c, nil
}

// UpdateCommand 更新命令的可编辑字段。
// catalog_protected=true 时 锁定 logical_code（其余 i18n / help / display 等可改）。
func (s *AdminService) UpdateCommand(ctx context.Context, getByID func(context.Context, uuid.UUID) (*MMLCommand, error), id uuid.UUID, req UpdateCommandReq) (*MMLCommand, error) {
	existing, err := getByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get command: %w", err)
	}
	if existing == nil {
		return nil, ErrCommandNotFound
	}
	if existing.CatalogProtected {
		if req.LogicalCode != nil && *req.LogicalCode != existing.LogicalCode {
			return nil, fmt.Errorf("logical_code is locked: %w", ErrCatalogProtected)
		}
	}
	if req.CommandName != nil {
		existing.CommandName = *req.CommandName
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.HelpDoc != nil {
		existing.HelpDoc = *req.HelpDoc
	}
	if req.Notes != nil {
		existing.Notes = *req.Notes
	}
	if req.TargetObject != nil {
		existing.TargetObject = *req.TargetObject
	}
	if req.GroupID != nil {
		existing.GroupID = req.GroupID
	}
	if req.CommandNameI18n != nil {
		existing.CommandNameI18n = *req.CommandNameI18n
	}
	if req.RequireConfirm != nil {
		existing.RequireConfirm = *req.RequireConfirm
	}
	if req.ConfirmMsgI18n != nil {
		existing.ConfirmMsgI18n = *req.ConfirmMsgI18n
	}
	if req.LogicalCode != nil {
		existing.LogicalCode = *req.LogicalCode
	}
	if req.LogicalNameI18n != nil {
		existing.LogicalNameI18n = *req.LogicalNameI18n
	}
	if err := s.commandRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update command: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.command.updated", "command:"+id.String(), map[string]any{
		"command_code": existing.CommandCode,
	})
	return existing, nil
}

// DeleteCommand 删除命令。catalog_protected=true → 拒绝。
func (s *AdminService) DeleteCommand(ctx context.Context, getByID func(context.Context, uuid.UUID) (*MMLCommand, error), id uuid.UUID) error {
	existing, err := getByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get command: %w", err)
	}
	if existing == nil {
		return ErrCommandNotFound
	}
	if existing.CatalogProtected {
		return ErrCatalogProtected
	}
	if err := s.commandRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete command: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.command.deleted", "command:"+id.String(), map[string]any{
		"command_code": existing.CommandCode,
	})
	return nil
}

// ============================================================
// SubField
// ============================================================

// CreateSubFieldReq 是 POST /admin/commands/:cid/sub-fields 的请求体。
type CreateSubFieldReq struct {
	ParamID         uuid.UUID         `json:"param_id" binding:"required"`
	MMLCode         string            `json:"mml_code" binding:"required,max=100"`
	LabelI18n       map[string]string `json:"label_i18n"`
	DefaultSelected *bool             `json:"default_selected,omitempty"`
	IsRequired      *bool             `json:"is_required,omitempty"`
	SortOrder       int               `json:"sort_order"`
}

// UpdateSubFieldReq 是 PATCH /admin/commands/:cid/sub-fields/:sid 的请求体。
type UpdateSubFieldReq struct {
	MMLCode         *string            `json:"mml_code,omitempty"`
	LabelI18n       *map[string]string `json:"label_i18n,omitempty"`
	DefaultSelected *bool              `json:"default_selected,omitempty"`
	IsRequired      *bool              `json:"is_required,omitempty"`
	SortOrder       *int               `json:"sort_order,omitempty"`
}

// CreateSubField 给命令添加一个 sub-field 绑定。
func (s *AdminService) CreateSubField(ctx context.Context, commandID uuid.UUID, req CreateSubFieldReq) (*MMLCommandSubField, error) {
	defaultSelected := true
	if req.DefaultSelected != nil {
		defaultSelected = *req.DefaultSelected
	}
	required := false
	if req.IsRequired != nil {
		required = *req.IsRequired
	}
	sf := &MMLCommandSubField{
		CommandID:       commandID,
		ParamID:         req.ParamID,
		MMLCode:         req.MMLCode,
		LabelI18n:       req.LabelI18n,
		DefaultSelected: defaultSelected,
		IsRequired:      required,
		SortOrder:       req.SortOrder,
	}
	if err := s.subFieldRepo.Create(ctx, sf); err != nil {
		return nil, fmt.Errorf("create sub_field: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.sub_field.created", "sub_field:"+sf.ID.String(), map[string]any{
		"command_id": commandID.String(),
		"param_id":   req.ParamID.String(),
		"mml_code":   sf.MMLCode,
	})
	return sf, nil
}

// UpdateSubField 更新一个 sub-field 绑定的 mml_code / label / sort / default / required。
func (s *AdminService) UpdateSubField(ctx context.Context, id uuid.UUID, req UpdateSubFieldReq) (*MMLCommandSubField, error) {
	existing, err := s.subFieldRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sub_field: %w", err)
	}
	if req.MMLCode != nil {
		existing.MMLCode = *req.MMLCode
	}
	if req.LabelI18n != nil {
		existing.LabelI18n = *req.LabelI18n
	}
	if req.DefaultSelected != nil {
		existing.DefaultSelected = *req.DefaultSelected
	}
	if req.IsRequired != nil {
		existing.IsRequired = *req.IsRequired
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}
	if err := s.subFieldRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update sub_field: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.sub_field.updated", "sub_field:"+id.String(), map[string]any{
		"command_id": existing.CommandID.String(),
		"mml_code":   existing.MMLCode,
	})
	return existing, nil
}

// DeleteSubField 删除 sub-field 绑定（mml_command_sub_fields trigger 自动回填 target_paths）。
func (s *AdminService) DeleteSubField(ctx context.Context, id uuid.UUID) error {
	existing, err := s.subFieldRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get sub_field: %w", err)
	}
	if err := s.subFieldRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete sub_field: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.sub_field.deleted", "sub_field:"+id.String(), map[string]any{
		"command_id": existing.CommandID.String(),
		"mml_code":   existing.MMLCode,
	})
	return nil
}

// ============================================================
// Param
// ============================================================

// CreateParamReq 是 POST /admin/params 的请求体。
type CreateParamReq struct {
	ParamCode          string            `json:"param_code" binding:"required,max=255"`
	Tr069Path          string            `json:"tr069_path" binding:"required,max=500"`
	ValueType          string            `json:"value_type" binding:"required,max=30"`
	DefaultValue       *string           `json:"default_value,omitempty"`
	JsRegex            *string           `json:"js_regex,omitempty"`
	ParamVersion       string            `json:"param_version" binding:"required,max=50"`
	NameI18n           map[string]string `json:"name_i18n"`
	ExplanationI18n    map[string]string `json:"explanation_i18n"`
	ConstraintTextI18n map[string]string `json:"constraint_text_i18n"`
	AccessType         string            `json:"access_type" binding:"omitempty,oneof=READ_ONLY READ_WRITE WRITE_ONLY"`
	IsObject           bool              `json:"is_object"`
	SupportsAdd        bool              `json:"supports_add"`
	SupportsDelete     bool              `json:"supports_delete"`
	ChangeApplies      string            `json:"change_applies" binding:"omitempty,oneof=Immediate OnReboot"`
}

// UpdateParamReq 是 PATCH /admin/params/:id 的请求体。
// 不可改字段（Q2=C 决议）：tr069_path / param_code / catalog_protected / source / is_writable
// 对 catalog_protected=true 行额外锁定：access_type / is_object / supports_add / supports_delete / change_applies
type UpdateParamReq struct {
	DefaultValue       *string            `json:"default_value,omitempty"`
	JsRegex            *string            `json:"js_regex,omitempty"`
	NameI18n           *map[string]string `json:"name_i18n,omitempty"`
	ExplanationI18n    *map[string]string `json:"explanation_i18n,omitempty"`
	ConstraintTextI18n *map[string]string `json:"constraint_text_i18n,omitempty"`
	AccessType         *string            `json:"access_type,omitempty" binding:"omitempty,oneof=READ_ONLY READ_WRITE WRITE_ONLY"`
	IsObject           *bool              `json:"is_object,omitempty"`
	SupportsAdd        *bool              `json:"supports_add,omitempty"`
	SupportsDelete     *bool              `json:"supports_delete,omitempty"`
	ChangeApplies      *string            `json:"change_applies,omitempty" binding:"omitempty,oneof=Immediate OnReboot"`
}

// CreateParam 新建 admin 来源的 param。
func (s *AdminService) CreateParam(ctx context.Context, req CreateParamReq) (*Param, error) {
	if req.AccessType == "" {
		req.AccessType = AccessTypeReadOnly
	}
	if req.ChangeApplies == "" {
		req.ChangeApplies = ChangeAppliesImmediate
	}
	p := &Param{
		ParamCode:          req.ParamCode,
		Tr069Path:          req.Tr069Path,
		ValueType:          req.ValueType,
		DefaultValue:       req.DefaultValue,
		JsRegex:            req.JsRegex,
		ParamVersion:       req.ParamVersion,
		AccessType:         req.AccessType,
		IsObject:           req.IsObject,
		SupportsAdd:        req.SupportsAdd,
		SupportsDelete:     req.SupportsDelete,
		ChangeApplies:      req.ChangeApplies,
		ConstraintTextI18n: req.ConstraintTextI18n,
		CatalogProtected:   false,
		Source:             SourceAdmin,
	}
	if err := s.paramRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create param: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.param.created", "param:"+p.ID.String(), map[string]any{
		"param_code":  p.ParamCode,
		"tr069_path":  p.Tr069Path,
		"access_type": p.AccessType,
	})
	return p, nil
}

// UpdateParam 更新 param。
// 对 catalog_protected=true 行额外锁定 access_type / is_object / supports_add / supports_delete / change_applies；
// i18n + default_value + js_regex + constraint_text_i18n 仍可改。
func (s *AdminService) UpdateParam(ctx context.Context, id uuid.UUID, req UpdateParamReq) (*Param, error) {
	existing, err := s.paramRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get param: %w", err)
	}
	if existing.CatalogProtected {
		if req.AccessType != nil && *req.AccessType != existing.AccessType {
			return nil, fmt.Errorf("access_type is locked: %w", ErrCatalogProtected)
		}
		if req.IsObject != nil && *req.IsObject != existing.IsObject {
			return nil, fmt.Errorf("is_object is locked: %w", ErrCatalogProtected)
		}
		if req.SupportsAdd != nil && *req.SupportsAdd != existing.SupportsAdd {
			return nil, fmt.Errorf("supports_add is locked: %w", ErrCatalogProtected)
		}
		if req.SupportsDelete != nil && *req.SupportsDelete != existing.SupportsDelete {
			return nil, fmt.Errorf("supports_delete is locked: %w", ErrCatalogProtected)
		}
		if req.ChangeApplies != nil && *req.ChangeApplies != existing.ChangeApplies {
			return nil, fmt.Errorf("change_applies is locked: %w", ErrCatalogProtected)
		}
	}
	if req.DefaultValue != nil {
		existing.DefaultValue = req.DefaultValue
	}
	if req.JsRegex != nil {
		existing.JsRegex = req.JsRegex
	}
	if req.ConstraintTextI18n != nil {
		existing.ConstraintTextI18n = *req.ConstraintTextI18n
	}
	if req.AccessType != nil {
		existing.AccessType = *req.AccessType
	}
	if req.IsObject != nil {
		existing.IsObject = *req.IsObject
	}
	if req.SupportsAdd != nil {
		existing.SupportsAdd = *req.SupportsAdd
	}
	if req.SupportsDelete != nil {
		existing.SupportsDelete = *req.SupportsDelete
	}
	if req.ChangeApplies != nil {
		existing.ChangeApplies = *req.ChangeApplies
	}
	if err := s.paramRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update param: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.param.updated", "param:"+id.String(), map[string]any{
		"param_code":        existing.ParamCode,
		"catalog_protected": existing.CatalogProtected,
	})
	return existing, nil
}

// DeleteParam 删除 param。
//   - catalog_protected=true → 拒绝
//   - 被 sub_fields 引用 → 拒绝
func (s *AdminService) DeleteParam(ctx context.Context, id uuid.UUID) error {
	existing, err := s.paramRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get param: %w", err)
	}
	if existing.CatalogProtected {
		return ErrCatalogProtected
	}
	refs, err := s.subFieldRepo.CountByParam(ctx, id)
	if err != nil {
		return fmt.Errorf("count sub_fields by param: %w", err)
	}
	if refs > 0 {
		return fmt.Errorf("param referenced by %d sub_fields: %w", refs, ErrParamInUse)
	}
	if err := s.paramRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete param: %w", err)
	}
	s.audit.Write(ctx, "mml.catalog.param.deleted", "param:"+id.String(), map[string]any{
		"param_code": existing.ParamCode,
		"tr069_path": existing.Tr069Path,
	})
	return nil
}

// ListParams 透传到 repo（不需守护逻辑，仅读取）。
func (s *AdminService) ListParams(ctx context.Context, f AdminParamFilter) ([]Param, int64, error) {
	return s.paramRepo.List(ctx, f)
}

// ListParamReferences 反向查：返回引用该 param 的命令列表（T-0131 admin Tab 3 抽屉用）。
func (s *AdminService) ListParamReferences(ctx context.Context, paramID uuid.UUID) ([]ParamReference, error) {
	return s.paramRepo.ListReferences(ctx, paramID)
}

// ============================================================
// Helpers
// ============================================================

// IsErrNotFound 工具方法：判定 err 是否任一 admin sentinel "not found"。
// handler 据此翻 HTTP 404。
func IsErrNotFound(err error) bool {
	return errors.Is(err, ErrSubFieldNotFound) ||
		errors.Is(err, ErrGroupNotFound) ||
		errors.Is(err, ErrCommandNotFound) ||
		errors.Is(err, ErrParamNotFound)
}
