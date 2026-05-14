/**
 * T-0123-P2-a Console 业务层类型 — frontend-core
 *
 * PRD: docs/design/mml-restore-old-interaction-plan-20260514.md §7.4
 * 后端契约：omcgo/internal/mml/mml_renderer.go Statement struct json tags
 *
 * 与 types/mml.ts 区别：mml.ts 包含历史 MML 命令/脚本/任务等模型；
 * 本文件专注 Console 三栏交互的运行期状态（statements、sub_field 元数据、
 * 命令树节点等）。后续 P2-b/c/d 组件层消费。
 */

import type { MMLOperationType } from './mml';

// ============================================================
// 后端 snake_case 类型（与 Go 后端 json tags 一一对应）
// ============================================================

/** 后端 `mml_renderer.go` Statement struct 的 wire 格式。 */
export interface BackendStatement {
  command_id?: string;                 // *uuid.UUID → string|undefined
  logical_code: string;
  operation_type: string;              // LST / MOD / ADD / RMV
  selected_sub_field_ids?: string[];   // LST 用：已选 sub_field id 列表
  selected_mml_codes?: string[];       // parser 内部 — UI round-trip 可见
  values?: Record<string, string>;     // MOD/ADD 用：mml_code → value 字符串
  rmv_instance_index?: number;
  unknown_codes?: string[];            // lookup 后未命中的 mml_code（FE toast）
}

/** GET /mml/group-tree 响应的子节点（递归结构）。 */
export interface BackendGroupTreeNode {
  id: string;
  group_code: string;
  path: string;
  display_name: string;                // "设备信息(LST DEVICE_INFO)" 老 OMC 格式
  display_order: number;
  commands: BackendGroupTreeCommand[];
  children: BackendGroupTreeNode[];
}

export interface BackendGroupTreeCommand {
  id: string;
  command_code: string;
  logical_code: string;
  operation_type: string;
  display_name: string;
  target_object?: string;
  require_confirm: boolean;
}

/** GET /mml/commands/:id/sub-fields 响应的单条 sub-field。 */
export interface BackendSubField {
  id: string;
  command_id: string;
  param_id: string;
  mml_code: string;
  label: string;                       // lang 派生
  label_i18n: Record<string, string>;
  tr069_path: string;
  value_type: string;
  access_type: string;                 // READ_ONLY / READ_WRITE
  is_object: boolean;
  supports_add: boolean;
  supports_delete: boolean;
  change_applies: string;              // OnReboot / Immediate / OnSession
  constraint_text: string;             // lang 派生
  constraint_text_i18n: Record<string, string>;
  default_value?: string;
  js_regex?: string;
  default_selected: boolean;
  is_required: boolean;
  sort_order: number;
}

/** POST /mml/parse 响应中的 parse_error 单条。 */
export interface BackendParseError {
  statement_index: number;
  raw: string;
  reason: string;
}

// ============================================================
// 前端 camelCase 类型（消费侧）
// ============================================================

/**
 * 命令树节点（递归）。
 * `commands` 是该 group 直接挂的命令（不含子 group 内的命令）。
 * `children` 是子 group 节点。
 */
export interface GroupTreeNode {
  id: string;
  groupCode: string;
  path: string;                        // ltree path "BSC_CONFIGURATION.BASIC_INFO"
  displayName: string;                 // "设备信息(LST DEVICE_INFO)"
  displayOrder: number;
  commands: GroupTreeCommand[];
  children: GroupTreeNode[];
}

export interface GroupTreeCommand {
  id: string;
  commandCode: string;
  logicalCode: string;
  operationType: MMLOperationType;
  displayName: string;
  targetObject?: string;
  requireConfirm: boolean;
}

/**
 * sub-field 元数据（命令上下文）。
 * 用于 LST Checklist / MOD InputList / ADD InputList 的渲染依据。
 * access_type / change_applies / is_object 等元数据驱动 UI 差异化（PRD §7.3）。
 */
export interface SubFieldDef {
  id: string;
  commandId: string;
  paramId: string;
  mmlCode: string;
  label: string;                       // 当前 lang 派生
  labelI18n: Record<string, string>;
  tr069Path: string;
  valueType: string;
  accessType: 'READ_ONLY' | 'READ_WRITE' | string;
  isObject: boolean;
  supportsAdd: boolean;
  supportsDelete: boolean;
  changeApplies: 'OnReboot' | 'Immediate' | 'OnSession' | string;
  constraintText: string;
  constraintTextI18n: Record<string, string>;
  defaultValue?: string;
  jsRegex?: string;
  defaultSelected: boolean;
  isRequired: boolean;
  sortOrder: number;
}

/**
 * Console 中单条 statement 的客户端状态。
 *
 * 注：uid 是客户端生成（crypto.randomUUID），用于 React key 与 store 内部
 * 寻址；commandId 是后端 mml_commands.id（uuid）。统计 / 提交 / round-trip 用
 * commandId，本地 UI 操作用 uid。
 */
export interface Statement {
  uid: string;                         // 客户端 nanoid 风格的 UUID
  commandId?: string;                  // 服务端解析后填；新增前可暂缺
  commandCode: string;                 // 例 "LST_DEVICE_INFO"
  logicalCode: string;                 // 例 "DEVICE_INFO"
  operationType: MMLOperationType;
  logicalNameI18n: Record<string, string>;
  /** 命令完整子字段（来自 GET /sub-fields），保留全集以便 toggle */
  subFields: SubFieldDef[];
  selectedSubFieldIds: string[];       // LST：勾选的 sub_field id
  values: Record<string, string>;      // MOD/ADD：mml_code → value
  rmvInstanceIndex?: number;           // RMV
  unknownCodes: string[];              // parser/lookup 未命中
}

/** parse 错误（不阻塞 statement 显示，前端 toast 提示） */
export interface ParseError {
  statementIndex: number;
  raw: string;
  reason: string;
}

// ============================================================
// 请求 / 响应（与 P1 backend 5 endpoint 一一对应）
// ============================================================

/** POST /mml/render 请求（UI → mml_string）。 */
export interface RenderRequest {
  commandId: string;
  operationType: MMLOperationType;
  selectedSubFieldIds?: string[];
  values?: Record<string, string>;
  rmvInstanceIndex?: number;
}

/** POST /mml/parse 请求（mml_string → statements）。 */
export interface ParseRequest {
  mmlString: string;
  lang?: string;
}

/** POST /mml/parse 响应。 */
export interface ParseResponse {
  statements: Statement[];
  parseErrors: ParseError[];
}

/** POST /mml/execute-statements 请求。 */
export interface ExecuteStatementsRequest {
  statements: Statement[];
  deviceSns: string[];
  taskName?: string;
  creator?: string;
  executor?: string;
  executeType?: 'immediate' | 'scheduled' | 'periodic' | 'suspended';
}

/** P2-a 简化版本地渲染器支持的操作类型。 */
export type ConsoleSupportedOp = 'LST' | 'MOD' | 'ADD' | 'RMV';
