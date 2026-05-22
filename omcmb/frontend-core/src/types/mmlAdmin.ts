/**
 * T-0123-P3 admin Catalog 管理 UI 类型 — frontend-core
 *
 * PRD: docs/design/mml-restore-old-interaction-plan-20260514.md §7.6 + §Q
 * 后端契约：omcgo/internal/mml/admin_handler.go 13 endpoints
 *
 * 4 资源（Group / Command / SubField / Param）× CRUD = 13 write endpoints
 * read 端点复用 P1 /mml/group-tree（GroupsTab + CommandsTab 数据源）。
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
// Param (mml_params)
// ============================================================

export type AccessType = 'READ_ONLY' | 'READ_WRITE' | string;
export type ChangeApplies = 'OnReboot' | 'Immediate' | 'OnSession' | string;

export interface BackendParamAdmin {
  id: string;
  param_code: string;
  param_name_zh: string;
  param_name_en: string;
  tr069_path: string;
  value_type: string;
  access_type: string;
  is_object: boolean;
  supports_add: boolean;
  supports_delete: boolean;
  change_applies: string;
  constraint_text_i18n: Record<string, string>;
  default_value?: string | null;
  js_regex?: string | null;
  catalog_protected: boolean;
}

export interface ParamAdmin {
  id: string;
  paramCode: string;
  paramNameZh: string;
  paramNameEn: string;
  tr069Path: string;
  valueType: string;
  accessType: AccessType;
  isObject: boolean;
  supportsAdd: boolean;
  supportsDelete: boolean;
  changeApplies: ChangeApplies;
  constraintTextI18n: I18nMap;
  defaultValue?: string | null;
  jsRegex?: string | null;
  catalogProtected: boolean;
}

export interface CreateParamRequest {
  paramCode: string;
  paramNameZh: string;
  paramNameEn: string;
  tr069Path: string;
  valueType: string;
  accessType?: AccessType;
  isObject?: boolean;
  supportsAdd?: boolean;
  supportsDelete?: boolean;
  changeApplies?: ChangeApplies;
  constraintTextI18n?: I18nMap;
  defaultValue?: string;
  jsRegex?: string;
}

export interface UpdateParamRequest {
  paramCode?: string;
  paramNameZh?: string;
  paramNameEn?: string;
  tr069Path?: string;
  valueType?: string;
  accessType?: AccessType;
  isObject?: boolean;
  supportsAdd?: boolean;
  supportsDelete?: boolean;
  changeApplies?: ChangeApplies;
  constraintTextI18n?: I18nMap;
  defaultValue?: string | null;
  jsRegex?: string | null;
}

export interface ListParamsRequest {
  search?: string;
  accessType?: string;
  isObject?: boolean;
  page?: number;
  pageSize?: number;
}

export interface ListParamsResponse {
  items: ParamAdmin[];
  total: number;
  page: number;
  pageSize: number;
}

// ============================================================
// ParamReference (T-0131 admin Tab 3 反向查)
// ============================================================

export interface BackendParamReference {
  command_id: string;
  command_code: string;
  logical_code: string;
  operation_type: string;
  command_name_i18n: Record<string, string>;
  group_id: string;
  group_path: string;
}

export interface ParamReference {
  commandId: string;
  commandCode: string;
  logicalCode: string;
  operationType: string;
  commandNameI18n: I18nMap;
  groupId: string;
  groupPath: string;
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

export function mapBackendParamReference(b: BackendParamReference): ParamReference {
  return {
    commandId: b.command_id,
    commandCode: b.command_code,
    logicalCode: b.logical_code,
    operationType: b.operation_type,
    commandNameI18n: b.command_name_i18n ?? {},
    groupId: b.group_id,
    groupPath: b.group_path,
  };
}

export function mapBackendParam(b: BackendParamAdmin): ParamAdmin {
  return {
    id: b.id,
    paramCode: b.param_code,
    paramNameZh: b.param_name_zh,
    paramNameEn: b.param_name_en,
    tr069Path: b.tr069_path,
    valueType: b.value_type,
    accessType: b.access_type,
    isObject: b.is_object,
    supportsAdd: b.supports_add,
    supportsDelete: b.supports_delete,
    changeApplies: b.change_applies,
    constraintTextI18n: b.constraint_text_i18n ?? {},
    defaultValue: b.default_value ?? null,
    jsRegex: b.js_regex ?? null,
    catalogProtected: b.catalog_protected,
  };
}

// ============================================================
// T-0132 admin Tab 4 XML 导入：preview + apply
// ============================================================

/** 单行 diff 桶：add (新增) / modify (覆盖 standard 行) / skipped (admin 改过 被守护跳过)。 */
export type ImportBucket = 'add' | 'modify' | 'skipped';

export interface BackendImportDiff {
  bucket: string;
  param_code: string;
  tr069_path: string;
  value_type: string;
  access_type: string;
  is_object: boolean;
}

export interface BackendImportSummary {
  add: number;
  modify: number;
  skipped: number;
  total: number;
}

export interface BackendImportPreviewResp {
  version_code: string;
  summary: BackendImportSummary;
  diffs: BackendImportDiff[];
  truncated: boolean;
  total: number;
}

export interface BackendImportApplyResp {
  version_code: string;
  rows_affected: number;
  total: number;
}

export interface ImportDiff {
  bucket: ImportBucket;
  paramCode: string;
  tr069Path: string;
  valueType: string;
  accessType: string;
  isObject: boolean;
}

export interface ImportSummary {
  add: number;
  modify: number;
  skipped: number;
  total: number;
}

export interface ImportPreviewResp {
  versionCode: string;
  summary: ImportSummary;
  diffs: ImportDiff[];
  truncated: boolean;
  total: number;
}

export interface ImportApplyResp {
  versionCode: string;
  rowsAffected: number;
  total: number;
}

export function mapBackendImportPreview(b: BackendImportPreviewResp): ImportPreviewResp {
  return {
    versionCode: b.version_code,
    summary: {
      add: b.summary.add,
      modify: b.summary.modify,
      skipped: b.summary.skipped,
      total: b.summary.total,
    },
    diffs: (b.diffs || []).map((d) => ({
      bucket: d.bucket as ImportBucket,
      paramCode: d.param_code,
      tr069Path: d.tr069_path,
      valueType: d.value_type,
      accessType: d.access_type,
      isObject: d.is_object,
    })),
    truncated: b.truncated,
    total: b.total,
  };
}

export function mapBackendImportApply(b: BackendImportApplyResp): ImportApplyResp {
  return {
    versionCode: b.version_code,
    rowsAffected: b.rows_affected,
    total: b.total,
  };
}
