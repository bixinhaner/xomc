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
  /** Catalog 册版本(对应后端 param_version,目前只有 cmcc-td-lte-v2.3)。
   * 不传则由 api 层用 catalogParamVersion 默认值。 */
  paramVersion?: string;
}

export interface UpdateGroupRequest {
  /** 注意:后端 PATCH 端点不接受 groupCode / parentId,字段保留是为了向前兼容
   * 调用者(传了会被 api 层忽略,只把 displayNameI18n / displayOrder 转换后发送)。 */
  groupCode?: string;
  parentId?: string | null;
  displayNameI18n?: I18nMap;
  displayOrder?: number;
}

// ============================================================
// 写端点 toBackend transform —— 2026-05-27 修复前后端契约错位
// 后端 admin_service.go 用 snake_case + 平铺 zh/en 字段(非 i18n map);
// HTTP 拦截器仅转 query params 不转 body,所以这里手动 map。
// ============================================================

/** 默认 catalog 册版本。当前数据库唯一一册为 cmcc-td-lte-v2.3。
 * 后续 multi-catalog 时改为从 UI 选 / 后端 list-versions 拉。 */
export const DEFAULT_CATALOG_PARAM_VERSION = 'cmcc-td-lte-v2.3';

export interface BackendCreateGroupBody {
  group_code: string;
  group_name_zh: string;
  group_name_en: string;
  param_version: string;
  display_order?: number;
}

export interface BackendUpdateGroupBody {
  group_name_zh?: string;
  group_name_en?: string;
  display_order?: number;
}

export function toBackendCreateGroup(req: CreateGroupRequest): BackendCreateGroupBody {
  const i18n = req.displayNameI18n ?? {};
  const zh = i18n['zh-CN'] || i18n.zh || '';
  const en = i18n['en-US'] || i18n.en || '';
  return {
    group_code: req.groupCode,
    group_name_zh: zh,
    group_name_en: en || zh,
    param_version: req.paramVersion || DEFAULT_CATALOG_PARAM_VERSION,
    display_order: req.displayOrder ?? 100,
  };
}

export function toBackendUpdateGroup(req: UpdateGroupRequest): BackendUpdateGroupBody {
  const out: BackendUpdateGroupBody = {};
  if (req.displayNameI18n) {
    const zh = req.displayNameI18n['zh-CN'] || req.displayNameI18n.zh;
    const en = req.displayNameI18n['en-US'] || req.displayNameI18n.en;
    if (zh !== undefined) out.group_name_zh = zh;
    if (en !== undefined) out.group_name_en = en;
  }
  if (req.displayOrder !== undefined) out.display_order = req.displayOrder;
  return out;
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
  /** 后端 PATCH 不接受 commandCode / operationType(创建后不可改),传了会被 api 层忽略 */
  commandCode?: string;
  operationType?: MMLOperationType;
  logicalCode?: string;
  commandNameI18n?: I18nMap;
  groupId?: string;
  targetObject?: string | null;
  requireConfirm?: boolean;
}

// ---- Command toBackend transforms ----

/** 后端 binding 允许的 operation_type 子集。前端 select 应收窄。 */
export const BACKEND_ALLOWED_OPERATION_TYPES = ['LST', 'MOD', 'ADD', 'RMV'] as const;
export type BackendOperationType = (typeof BACKEND_ALLOWED_OPERATION_TYPES)[number];

export interface BackendCreateCommandBody {
  command_name: string;
  command_code: string;
  operation_type: string;
  group_id?: string;
  command_name_i18n?: Record<string, string>;
  target_object?: string;
  require_confirm?: boolean;
  logical_code?: string;
  logical_name_i18n?: Record<string, string>;
}

export interface BackendUpdateCommandBody {
  command_name?: string;
  group_id?: string;
  command_name_i18n?: Record<string, string>;
  target_object?: string;
  require_confirm?: boolean;
  logical_code?: string;
  logical_name_i18n?: Record<string, string>;
}

function deriveCommandName(i18n: I18nMap | undefined, code: string): string {
  if (!i18n) return code;
  return i18n['zh-CN'] || i18n['en-US'] || i18n.zh || i18n.en || code;
}

export function toBackendCreateCommand(req: CreateCommandRequest): BackendCreateCommandBody {
  return {
    command_name: deriveCommandName(req.commandNameI18n, req.commandCode),
    command_code: req.commandCode,
    operation_type: req.operationType,
    group_id: req.groupId,
    command_name_i18n: req.commandNameI18n,
    target_object: req.targetObject || undefined,
    require_confirm: req.requireConfirm ?? false,
    logical_code: req.logicalCode,
    logical_name_i18n: req.commandNameI18n,
  };
}

export function toBackendUpdateCommand(req: UpdateCommandRequest): BackendUpdateCommandBody {
  const out: BackendUpdateCommandBody = {};
  if (req.commandNameI18n) {
    out.command_name_i18n = req.commandNameI18n;
    out.logical_name_i18n = req.commandNameI18n;
    const name = deriveCommandName(req.commandNameI18n, '');
    if (name) out.command_name = name;
  }
  if (req.groupId !== undefined) out.group_id = req.groupId;
  if (req.targetObject !== undefined && req.targetObject !== null) {
    out.target_object = req.targetObject;
  }
  if (req.requireConfirm !== undefined) out.require_confirm = req.requireConfirm;
  if (req.logicalCode !== undefined) out.logical_code = req.logicalCode;
  return out;
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

// ---- SubField toBackend transforms ----

export interface BackendCreateSubFieldBody {
  param_id: string;
  mml_code: string;
  label_i18n?: Record<string, string>;
  default_selected?: boolean;
  is_required?: boolean;
  sort_order?: number;
}

export interface BackendUpdateSubFieldBody {
  mml_code?: string;
  label_i18n?: Record<string, string>;
  default_selected?: boolean;
  is_required?: boolean;
  sort_order?: number;
}

export function toBackendCreateSubField(req: CreateSubFieldRequest): BackendCreateSubFieldBody {
  return {
    param_id: req.paramId,
    mml_code: req.mmlCode,
    label_i18n: req.labelI18n,
    default_selected: req.defaultSelected,
    is_required: req.isRequired,
    sort_order: req.sortOrder ?? 100,
  };
}

export function toBackendUpdateSubField(req: UpdateSubFieldRequest): BackendUpdateSubFieldBody {
  const out: BackendUpdateSubFieldBody = {};
  if (req.mmlCode !== undefined) out.mml_code = req.mmlCode;
  if (req.labelI18n !== undefined) out.label_i18n = req.labelI18n;
  if (req.defaultSelected !== undefined) out.default_selected = req.defaultSelected;
  if (req.isRequired !== undefined) out.is_required = req.isRequired;
  if (req.sortOrder !== undefined) out.sort_order = req.sortOrder;
  return out;
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

/** 后端 GET /admin/standard-params 单条原始字段(snake_case)。 */
export interface BackendStandardParam {
  id: string;
  standard_path: string;
  entry_type: string;
  access: string;
  data_type: string;
  change_applies: string;
  min_value?: number | null;
  max_value?: number | null;
  description: string;
}

export function mapBackendStandardParam(b: BackendStandardParam): StandardParamView {
  return {
    id: b.id,
    standardPath: b.standard_path,
    entryType: b.entry_type,
    access: b.access,
    dataType: b.data_type,
    changeApplies: b.change_applies,
    minValue: b.min_value ?? null,
    maxValue: b.max_value ?? null,
    description: b.description ?? '',
  };
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
