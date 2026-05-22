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
