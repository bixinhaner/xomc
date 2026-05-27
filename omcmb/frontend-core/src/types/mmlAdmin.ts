/**
 * T-0123-P3 admin Catalog 管理 UI 类型 — frontend-core
 *
 * PRD: docs/design/mml-restore-old-interaction-plan-20260514.md §7.6 + §Q
 * 后端契约：omcgo/internal/mml/admin_handler.go 9 endpoints（Group / Command / SubField CRUD）
 *
 * 3 资源（Group / Command / SubField）× CRUD = 9 write endpoints
 * read 端点复用 P1 /mml/group-tree（GroupsTab + CommandsTab 数据源）。
 *
 * 备注：mml_params 表 + XML 导入端点已下线，相关类型（ParamAdmin /
 * ParamReference / ImportPreview / ImportApply 等）同步移除。
 */

import type { MMLOperationType } from './mml';

export type I18nMap = Record<string, string>;
export type Source = 'standard' | 'admin';

// ============================================================
// Group (mml_command_groups)
// ============================================================

export interface BackendGroupAdmin {
  id: string;
  group_code: string;
  path: string;
  parent_id?: string | null;
  display_name_i18n: Record<string, string>;
  display_order: number;
  source: string;
  catalog_protected: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface GroupAdmin {
  id: string;
  groupCode: string;
  path: string;
  parentId?: string | null;
  displayNameI18n: I18nMap;
  displayOrder: number;
  source: Source;
  catalogProtected: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateGroupRequest {
  groupCode: string;
  parentId?: string;
  displayNameI18n: I18nMap;
  displayOrder?: number;
}

export interface UpdateGroupRequest {
  groupCode?: string;
  parentId?: string | null;
  displayNameI18n?: I18nMap;
  displayOrder?: number;
}

// ============================================================
// Command (mml_commands)
// ============================================================

export interface BackendCommandAdmin {
  id: string;
  command_code: string;
  logical_code: string;
  operation_type: string;
  command_name_i18n: Record<string, string>;
  group_id: string;
  target_object?: string | null;
  require_confirm: boolean;
  source: string;
  catalog_protected: boolean;
}

export interface CommandAdmin {
  id: string;
  commandCode: string;
  logicalCode: string;
  operationType: MMLOperationType;
  commandNameI18n: I18nMap;
  groupId: string;
  targetObject?: string | null;
  requireConfirm: boolean;
  source: Source;
  catalogProtected: boolean;
}

export interface CreateCommandRequest {
  commandCode: string;
  logicalCode: string;
  operationType: MMLOperationType;
  commandNameI18n: I18nMap;
  groupId: string;
  targetObject?: string;
  requireConfirm?: boolean;
}

export interface UpdateCommandRequest {
  commandCode?: string;
  logicalCode?: string;
  operationType?: MMLOperationType;
  commandNameI18n?: I18nMap;
  groupId?: string;
  targetObject?: string | null;
  requireConfirm?: boolean;
}

// ============================================================
// SubField (mml_command_sub_fields)
// ============================================================

export interface BackendSubFieldAdmin {
  id: string;
  command_id: string;
  param_id: string;
  mml_code: string;
  label_i18n: Record<string, string>;
  default_selected: boolean;
  is_required: boolean;
  sort_order: number;
}

export interface SubFieldAdmin {
  id: string;
  commandId: string;
  paramId: string;
  mmlCode: string;
  labelI18n: I18nMap;
  defaultSelected: boolean;
  isRequired: boolean;
  sortOrder: number;
}

export interface CreateSubFieldRequest {
  paramId: string;
  mmlCode: string;
  labelI18n?: I18nMap;
  defaultSelected?: boolean;
  isRequired?: boolean;
  sortOrder?: number;
}

export interface UpdateSubFieldRequest {
  mmlCode?: string;
  labelI18n?: I18nMap;
  defaultSelected?: boolean;
  isRequired?: boolean;
  sortOrder?: number;
}

// ============================================================
// Mappers (Backend snake_case → FE camelCase)
// ============================================================

function asSource(s: string): Source {
  return s === 'admin' ? 'admin' : 'standard';
}

export function mapBackendGroup(b: BackendGroupAdmin): GroupAdmin {
  return {
    id: b.id,
    groupCode: b.group_code,
    path: b.path,
    parentId: b.parent_id ?? null,
    displayNameI18n: b.display_name_i18n ?? {},
    displayOrder: b.display_order,
    source: asSource(b.source),
    catalogProtected: b.catalog_protected,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

export function mapBackendCommand(b: BackendCommandAdmin): CommandAdmin {
  return {
    id: b.id,
    commandCode: b.command_code,
    logicalCode: b.logical_code,
    operationType: b.operation_type as MMLOperationType,
    commandNameI18n: b.command_name_i18n ?? {},
    groupId: b.group_id,
    targetObject: b.target_object ?? null,
    requireConfirm: b.require_confirm,
    source: asSource(b.source),
    catalogProtected: b.catalog_protected,
  };
}

export function mapBackendSubField(b: BackendSubFieldAdmin): SubFieldAdmin {
  return {
    id: b.id,
    commandId: b.command_id,
    paramId: b.param_id,
    mmlCode: b.mml_code,
    labelI18n: b.label_i18n ?? {},
    defaultSelected: b.default_selected,
    isRequired: b.is_required,
    sortOrder: b.sort_order,
  };
}

export function mapBackendAdminSubFieldEnriched(
  b: BackendAdminSubFieldEnriched,
): AdminSubFieldEnriched {
  const labelI18n = b.label_i18n ?? {};
  const constraintTextI18n = b.constraint_text_i18n ?? {};
  return {
    id: b.id,
    commandId: b.command_id,
    paramId: b.param_id,
    mmlCode: b.mml_code,
    label:
      labelI18n['zh-CN'] ||
      labelI18n['en-US'] ||
      labelI18n.zh ||
      labelI18n.en ||
      b.mml_code,
    labelI18n,
    tr069Path: b.tr069_path,
    valueType: b.value_type,
    accessType: b.access_type,
    isObject: b.is_object,
    changeApplies: b.change_applies,
    constraintText:
      constraintTextI18n['zh-CN'] ||
      constraintTextI18n['en-US'] ||
      constraintTextI18n.zh ||
      constraintTextI18n.en ||
      '',
    constraintTextI18n,
    description: b.description ?? '',
    defaultSelected: b.default_selected,
    isRequired: b.is_required,
    sortOrder: b.sort_order,
    isSupported: b.is_supported,
    supportedModelCount: b.supported_model_count,
    totalModelCount: b.total_model_count,
  };
}

// ============================================================
// T-Mml-Admin: standard_params 下拉 + admin List 返回值
// ============================================================

/** standard_params 下拉项（GET /admin/standard-params 单条）。 */
export interface StandardParamView {
  id: string;
  standardPath: string;
  entryType: string;
  access: string;
  dataType: string;
  changeApplies: string;
  minValue?: number | null;
  maxValue?: number | null;
  description: string;
}

/** 后端 MMLCommandSubFieldEnriched 的 JSON 形态（snake_case）。
 * 与 sub_field_model.go MMLCommandSubFieldEnriched 一一对应。
 * 2026-05-27 修复:原先 listSubFields 直返 data.items 把它当 AdminSubFieldEnriched
 * 使用,导致 mml_code/label_i18n/tr069_path/is_supported 等字段全为 undefined。
 */
export interface BackendAdminSubFieldEnriched {
  id: string;
  command_id: string;
  param_id: string;
  mml_code: string;
  label_i18n: Record<string, string> | null;
  default_selected: boolean;
  is_required: boolean;
  sort_order: number;
  tr069_path: string;
  value_type: string;
  access_type: string;
  is_object: boolean;
  supports_add?: boolean;
  supports_delete?: boolean;
  change_applies: string;
  constraint_text_i18n: Record<string, string> | null;
  param_name_i18n?: Record<string, string> | null;
  description?: string;
  is_supported: boolean;
  supported_model_count: number;
  total_model_count: number;
}

/** Admin SubField enriched 列表行（GET /admin/commands/:id/sub-fields）。
 * 与 console SubFieldDef 类似但多 isSupported 字段，便于管理员看到被 auto-learn 关闭的行。
 */
export interface AdminSubFieldEnriched {
  id: string;
  commandId: string;
  paramId: string;
  mmlCode: string;
  label: string;
  labelI18n: I18nMap;
  tr069Path: string;
  valueType: string;
  accessType: string;
  isObject: boolean;
  changeApplies: string;
  constraintText: string;
  constraintTextI18n: I18nMap;
  description: string;
  defaultSelected: boolean;
  isRequired: boolean;
  sortOrder: number;
  /** 是否被 active param_mappings 中至少一个 paramModel 标为支持。
   * 2026-05-27 后真值源已从 mml_command_sub_fields.is_supported 改为
   * param_mappings.is_supported（T-0176-PR-A 单一真值源策略）。
   * 无映射时兜底 true。 */
  isSupported: boolean;
  /** 标该 path supported=true 的 active paramModel 数量。 */
  supportedModelCount: number;
  /** 注册该 path 的 active paramModel 总数。
   * = 0 表示该 path 没在任何 paramModel 注册（兜底 isSupported=true，但用户应当留意）。 */
  totalModelCount: number;
}

/** GET /admin/groups 返回的"全集" group。
 * 与 GroupAdmin 区别：backend 直接返 group_name_zh / group_name_en 而非 display_name_i18n。
 */
export interface AdminGroup {
  id: string;
  groupCode: string;
  groupNameZh: string;
  groupNameEn: string;
  paramVersion: string;
  displayOrder: number;
  source: string;
  catalogProtected: boolean;
}

/** GET /admin/commands 返回的命令行（含富字段）。 */
export interface AdminCommand {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  rpcMethod: string;
  operationType: MMLOperationType;
  targetObject?: string | null;
  targetPaths: string[];
  groupId?: string | null;
  commandNameI18n: I18nMap;
  logicalCode: string;
  logicalNameI18n: I18nMap;
  requireConfirm: boolean;
  source: string;
  catalogProtected: boolean;
}

/** 批量创建 sub-fields 请求 — 用户规则 #4 的入口。 */
export interface BatchCreateSubFieldsRequest {
  standardPathIds: string[];
}
