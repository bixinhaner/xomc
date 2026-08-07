import http from '../http';
import { generateUid } from '../../utils/uid';
import type { MMLCommand, MMLScript, MMLTask, MMLTaskCommandDetail, MMLTaskCommandInput, MMLParam, MMLCustomCommand, MMLCustomCommandPathDef, ParamPath, MMLOperationType, DeviceTaskResultItem, MMLParamRef, MMLTaskResultsStats, MMLPathTranslationView, PathTranslationSource, MMLTaskPlanItem, MMLTaskCreateInput, MMLScriptImportValidation, MMLScriptValidationSummary, MMLScriptIssue, MMLImportedScriptCreateInput, MMLImportedScriptReplaceInput, MMLScriptExecutionInput, MMLScriptImportTemplate } from '../../types/mml';
import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  BackendStatement,
  BackendGroupTreeNode,
  BackendSubField,
  BackendParseError,
  BackendSearchCommand,
  GroupTreeNode,
  GroupTreeCommand,
  SubFieldDef,
  SearchCommand,
  Statement,
  ParseError,
  RenderRequest,
  ParseRequest,
  ParseResponse,
  ExecuteStatementsRequest,
  StructuredExecuteRequest,
  StructuredStatement,
  CommandCompatibility,
  FlatGroupTreeResponse,
} from '../../types/mmlConsole';

// ---------------------------------------------------------------------------
// Backend response types  (snake_case, matching omcgo/internal/omcr/mml/model.go)
// ---------------------------------------------------------------------------

interface BackendMMLParamRef {
  id: string;
  param_code: string;
  param_name_zh: string;
  tr069_path: string;
  value_type: string;
  is_writable: boolean;
  default_value?: string;
  js_regex?: string;
  value_constraint: Record<string, unknown> | null;
}

interface BackendMMLCommand {
  id: string;
  command_name: string;
  command_code: string;
  category: string;
  description: string;
  rpc_method: string;
  created_at: string;
  operation_type?: string;
  help_doc?: string;
  notes?: string;
  params?: BackendMMLParamRef[] | null;
  // standard-model 重建后由 mmlstandardloader 写入的新字段（migration 000090）
  target_paths?: string[] | null;
  target_object?: string | null;
  group_id?: string | null;
  command_name_i18n?: Record<string, string> | null;
  require_confirm?: boolean;
  confirm_msg_i18n?: Record<string, string> | null;
  // 老字段，向后兼容已死无返回；保留 type 防 axios 误转
  param_template?: Record<string, unknown> | null;
  product_types?: string[] | null;
  param_paths?: Array<string | { path: string; label?: string; writable?: boolean }> | null;
  supported_operations?: string[] | null;
}

interface BackendMMLScript {
  id: string;
  script_name: string;
  description: string;
  content: string;
  creator: string;
  tags: string[] | null;
  created_at: string;
  updated_at: string;
  // New fields from mml_scripts restructure
  status?: string;
  start_time?: string | null;
  end_time?: string | null;
  type?: string;
  progress?: number;
  result?: Record<string, unknown> | null;
  // P1 last_run snapshot
  last_run_status?: string | null;
  last_run_at?: string | null;
  original_filename?: string;
  content_sha256?: string;
  validation_version?: string;
  validated_at?: string | null;
  plan_items?: BackendMMLPlanItem[] | null;
  validation_summary?: BackendPersistedMMLScriptValidation | null;
}

interface BackendMMLScriptIssue {
  code: string;
  severity: string;
  line_no?: number;
  raw_line?: string;
  field?: string;
  message?: string;
  display_message?: string;
}

interface BackendMMLScriptValidationSummary {
  total_lines?: number;
  valid_lines?: number;
  /** Compatibility with the original frontend import proposal. */
  effective_lines?: number;
  device_count?: number;
  error_count?: number;
  warning_count?: number;
}

/**
 * Imported scripts persist both the validator summary and line issues inside
 * validation_summary. Older rows may contain the summary fields directly.
 */
interface BackendPersistedMMLScriptValidation extends BackendMMLScriptValidationSummary {
  summary?: BackendMMLScriptValidationSummary | null;
  issues?: BackendMMLScriptIssue[] | null;
}

interface BackendMMLScriptImportValidation {
  validation_token?: string;
  original_filename?: string;
  normalized_content?: string;
  content_sha256?: string;
  validation_version?: string;
  validated_at?: string | null;
  plan_items?: BackendMMLPlanItem[] | null;
  summary?: BackendMMLScriptValidationSummary | null;
  issues?: BackendMMLScriptIssue[] | null;
}

export class MMLScriptImportApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly validation?: MMLScriptImportValidation;

  constructor({
    status,
    code,
    message,
    validation,
  }: {
    status: number;
    code?: string;
    message: string;
    validation?: MMLScriptImportValidation;
  }) {
    super(message);
    this.name = 'MMLScriptImportApiError';
    this.status = status;
    this.code = code;
    this.validation = validation;
  }
}

interface BackendMMLTask {
  id: string;
  task_name: string;
  script_id: string;
  script_name?: string | null;
  task_origin?: string;
  device_sns: string[] | null;
  commands: Array<Record<string, unknown>> | null;
  command_count?: number;
  execute_mode?: string | null;
  plan_items?: BackendMMLPlanItem[] | null;
  plan_item_count?: number;
  status: string;
  results: Array<Record<string, unknown>> | null;
  creator: string;
  created_at: string;
  updated_at: string;
  // Scheduling
  execute_type: string;
  scheduled_at: string | null;
  period_start: string | null;
  period_end: string | null;
  period_time: string | null;
  // Retry strategy
  offline_retry: boolean;
  offline_retry_wait: number;
  failed_retry: boolean;
  failed_retry_count: number;
  failed_retry_interval: number;
  // Execution timestamps
  started_at: string | null;
  finished_at: string | null;
  // Statistics
  total_devices: number;
  success_count: number;
  failed_count: number;
  result: string | null;
  latest_run?: BackendMMLTaskRun | null;
  // P2/P3 Scheduler fields
  next_trigger_at?: string | null;
  parent_task_id?: string | null;
  // 整改方案 Stage 3 — 路径翻译警告（GetTask 聚合 device_tasks 后填充）
  path_translation_warning?: {
    any_miss: boolean;
    device_count: number;
    path_count: number;
  } | null;

  // T-0168: 翻译审计 4 列（migration 000171 持久化到 mml_tasks 表）
  product_resolved?: boolean;
  matched_product_id?: string | null;
  matched_product_class?: string | null;
  path_translation_source?: string | null;
}

interface BackendMMLTaskRun {
  id: string;
  execute_type: string;
  execute_mode?: string | null;
  status: string;
  result?: string | null;
  total_devices?: number;
  success_count?: number;
  failed_count?: number;
  command_count?: number;
  plan_item_count?: number;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  updated_at: string;
}

interface BackendMMLPlanItem {
  line_no?: number;
  device_sn?: string;
  order?: number;
  raw_line?: string;
  command?: Record<string, unknown> | null;
  command_code?: string;
  operation_type?: string;
  parameters?: Record<string, unknown> | null;
}

// T-0168: GET /mml/tasks/{id}/results 响应 stats 字段（后端 TaskResultsStats）
interface BackendMMLPathTranslation {
  standard_path: string;
  private_path: string;
  translation_source: string;
  translated: boolean;
}

/** 后端 /mml/unsupported-paths 返回项（axios 可能 camelCase，两种 key 都容忍）。 */
interface BackendUnsupportedPath {
  path: string;
  read_unsupported?: boolean;
  write_unsupported?: boolean;
  readUnsupported?: boolean;
  writeUnsupported?: boolean;
}

/** 产品某 standardPath 的读 / 写不支持标记（自学习表）。 */
export interface UnsupportedPathInfo {
  path: string;
  readUnsupported: boolean;
  writeUnsupported: boolean;
}

interface BackendMMLTaskResultsStats {
  path_translations?: BackendMMLPathTranslation[];
  product_resolved: boolean;
  matched_product_class?: string;
  path_translation_source?: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendMMLCustomCommand {
  id: string;
  command_name: string;
  command_code: string;
  operation_type: string;
  command_scope: string;
  category_group?: string;
  parameters: Record<string, unknown> | null;
  param_paths: string[] | null;
  description: string;
  creator: string;
  created_at: string;
  updated_at: string;
}

interface BackendMMLCustomCommandPath {
  id: string;
  command_id: string;
  standard_path_id: string;
  standard_path: string;
  entry_type: string;
  access: string;
  data_type: string;
  description: string;
  min_value?: number | null;
  max_value?: number | null;
  default_selected: boolean;
  sort_order: number;
  mutable: boolean;
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend -> frontend
// ---------------------------------------------------------------------------

/**
 * Convert backend param_template (a flat map) into the frontend MMLParam[]
 * structure.  The param_template from the backend is a generic JSON object
 * where each key is the parameter name and the value can be a primitive
 * (default value hint) or an object with richer metadata.
 */
function mapParamTemplate(
  template: Record<string, unknown> | null | undefined
): MMLParam[] {
  if (!template) return [];
  return Object.entries(template).map(([name, value]) => {
    // If the value is a structured object with metadata, unpack it
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      const obj = value as Record<string, unknown>;
      return {
        name,
        type: (obj.type as MMLParam['type']) || 'string',
        required: Boolean(obj.required),
        defaultValue: obj.default_value as MMLParam['defaultValue'],
        description: (obj.description as string) || '',
        options: obj.options as MMLParam['options'],
        minValue: obj.min_value as number | undefined,
        maxValue: obj.max_value as number | undefined,
        pattern: obj.pattern as string | undefined,
        suggestedValue: obj.suggested_value as MMLParam['suggestedValue'],
        unit: obj.unit as string | undefined,
        restartRequired: obj.restart_required as boolean | undefined,
        helpText: obj.help_text as string | undefined,
        order: obj.order as number | undefined,
        enumValues: obj.enum_values as string[] | undefined,
      };
    }
    // Simple scalar — treat as a string param with a default
    return {
      name,
      type: 'string' as const,
      required: false,
      description: '',
      defaultValue: value as string | number | boolean | undefined,
    };
  });
}

function mapBackendParamRef(bp: BackendMMLParamRef): MMLParamRef {
  return {
    id: bp.id,
    paramCode: bp.param_code,
    paramNameZh: bp.param_name_zh,
    tr069Path: bp.tr069_path,
    valueType: bp.value_type as MMLParamRef['valueType'],
    isWritable: bp.is_writable,
    defaultValue: bp.default_value || undefined,
    jsRegex: bp.js_regex || undefined,
    valueConstraint: bp.value_constraint ?? {},
  };
}

// 由 backend.operation_type 派生 writable —— 与后端 isWritableOperation 对齐。
function deriveWritableFromOp(op: string | undefined): boolean {
  if (!op) return false;
  const upper = op.toUpperCase();
  return upper === 'MOD' || upper === 'ADD' || upper === 'RMV'
      || upper === 'ACT' || upper === 'DEA' || upper === 'RST'
      || upper === 'CLR' || upper === 'UPG';
}

function mapBackendCommand(bc: BackendMMLCommand): MMLCommand {
  const op = bc.operation_type;
  const writable = deriveWritableFromOp(op);

  // 优先 target_paths（新 schema）；老 param_paths 形态降级兼容
  let paramPaths: ParamPath[] | undefined;
  if (Array.isArray(bc.target_paths) && bc.target_paths.length > 0) {
    paramPaths = bc.target_paths
      .map((p) => (typeof p === 'string' && p.trim() ? { path: p, label: p, writable } : null))
      .filter((v): v is ParamPath => v !== null);
  } else if (Array.isArray(bc.param_paths)) {
    paramPaths = bc.param_paths
      .map((item) => {
        if (typeof item === 'string') {
          const path = item.trim();
          if (!path) return null;
          return { path, label: path, writable } as ParamPath;
        }
        if (!item.path) return null;
        return {
          path: item.path,
          label: item.label || item.path,
          writable: item.writable ?? writable,
        } as ParamPath;
      })
      .filter((v): v is ParamPath => v !== null);
  }

  // standard-model 重建后单命令绑单 operation_type；supportedOperations 派生单值
  // 以兼容 ParamPathPanel 等老消费方
  let supportedOperations: string[] | undefined = bc.supported_operations || undefined;
  if (!supportedOperations && op) {
    supportedOperations = [op];
  }

  return {
    id: bc.id,
    commandName: bc.command_name,
    commandCode: bc.command_code,
    category: bc.category,
    description: bc.description,
    params: mapParamTemplate(bc.param_template),
    operationType: op as MMLOperationType | undefined,
    paramPaths,
    supportedOperations,
    helpDoc: bc.help_doc || undefined,
    notes: bc.notes || undefined,
    paramRefs: dedupeParamRefs(bc.params?.map(mapBackendParamRef)),
    targetObject: bc.target_object || undefined,
    groupId: bc.group_id || undefined,
    commandNameI18n: bc.command_name_i18n || undefined,
    requireConfirm: bc.require_confirm,
    confirmMsgI18n: bc.confirm_msg_i18n || undefined,
    productClasses: bc.product_types || [],
  };
}

// dedupeParamRefs 按 tr069Path（兜底 paramCode）去重，保留首次出现项。
// 后端在数据层已做去重，此处是防御性保护：兼容历史脏数据 / 老缓存场景。
function dedupeParamRefs(refs: MMLParamRef[] | undefined): MMLParamRef[] | undefined {
  if (!refs || refs.length === 0) return refs;
  const seen = new Set<string>();
  return refs.filter((ref) => {
    const key = ref.tr069Path || ref.paramCode;
    if (!key) return true;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function mapBackendScript(bs: BackendMMLScript): MMLScript {
  const persistedValidation = mapPersistedScriptValidation(bs.validation_summary);
  return {
    id: bs.id,
    scriptName: bs.script_name,
    description: bs.description,
    content: bs.content,
    creator: bs.creator,
    tags: bs.tags || [],
    createTime: bs.created_at,
    updateTime: bs.updated_at,
    // New fields from mml_scripts restructure
    status: (bs.status || 'active') as MMLScript['status'],
    startTime: bs.start_time || undefined,
    endTime: bs.end_time || undefined,
    type: (bs.type || 'manual') as MMLScript['type'],
    progress: bs.progress ?? 0,
    result: bs.result ?? undefined,
    lastRunStatus: bs.last_run_status || undefined,
    lastRunAt: bs.last_run_at || undefined,
    originalFilename: bs.original_filename || undefined,
    contentSha256: bs.content_sha256 || undefined,
    validationVersion: bs.validation_version || undefined,
    validatedAt: bs.validated_at || undefined,
    planItems: bs.plan_items?.map(mapBackendPlanItem),
    validationSummary: persistedValidation.validationSummary,
    validationIssues: persistedValidation.validationIssues,
  };
}

function mapPersistedScriptValidation(
  validation: BackendPersistedMMLScriptValidation | null | undefined,
): Pick<MMLScript, 'validationSummary' | 'validationIssues'> {
  if (!validation) return {};
  const summary = validation.summary ?? validation;
  return {
    validationSummary: mapBackendScriptValidationSummary(summary),
    validationIssues: validation.issues?.map(mapBackendScriptIssue),
  };
}

function mapBackendScriptIssue(issue: BackendMMLScriptIssue): MMLScriptIssue {
  return {
    code: issue.code,
    severity: issue.severity === 'warning' ? 'warning' : 'error',
    lineNo: issue.line_no || undefined,
    rawLine: issue.raw_line || undefined,
    field: issue.field || undefined,
    message: issue.message || undefined,
    displayMessage: issue.display_message || undefined,
  };
}

function mapBackendScriptValidationSummary(
  summary: BackendMMLScriptValidationSummary | null | undefined,
): MMLScriptValidationSummary {
  const validLines = summary?.valid_lines ?? summary?.effective_lines ?? 0;
  return {
    totalLines: summary?.total_lines ?? validLines,
    validLines,
    effectiveLines: summary?.effective_lines ?? validLines,
    deviceCount: summary?.device_count ?? 0,
    errorCount: summary?.error_count ?? 0,
    warningCount: summary?.warning_count ?? 0,
  };
}

function mapBackendScriptImportValidation(
  validation: BackendMMLScriptImportValidation,
): MMLScriptImportValidation {
  return {
    validationToken: validation.validation_token || undefined,
    originalFilename: validation.original_filename || undefined,
    normalizedContent: validation.normalized_content || undefined,
    contentSha256: validation.content_sha256 || undefined,
    validationVersion: validation.validation_version || undefined,
    validatedAt: validation.validated_at || undefined,
    planItems: (validation.plan_items || []).map(mapBackendPlanItem),
    summary: mapBackendScriptValidationSummary(validation.summary),
    issues: (validation.issues || []).map(mapBackendScriptIssue),
  };
}

function objectValue(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : undefined;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function validationPayloadFromError(
  body: Record<string, unknown> | undefined,
): BackendMMLScriptImportValidation | undefined {
  const data = objectValue(body?.data);
  const candidate = data ?? body;
  if (!candidate) return undefined;
  const issues = candidate.issues ?? body?.issues;
  const hasValidationFields = candidate.summary !== undefined
    || candidate.plan_items !== undefined
    || issues !== undefined;
  if (!hasValidationFields) return undefined;
  return {
    validation_token: stringValue(candidate.validation_token),
    original_filename: stringValue(candidate.original_filename),
    normalized_content: stringValue(candidate.normalized_content),
    content_sha256: stringValue(candidate.content_sha256),
    validation_version: stringValue(candidate.validation_version),
    validated_at: stringValue(candidate.validated_at),
    plan_items: Array.isArray(candidate.plan_items)
      ? candidate.plan_items as BackendMMLPlanItem[]
      : [],
    summary: objectValue(candidate.summary) as BackendMMLScriptValidationSummary | undefined,
    issues: Array.isArray(issues) ? issues as BackendMMLScriptIssue[] : [],
  };
}

/**
 * Converts Axios-shaped import/execution failures into the shared contract so
 * skin UIs can display 422 errors and 409 warning confirmations consistently.
 */
export function normalizeMMLScriptImportApiError(error: unknown): MMLScriptImportApiError {
  if (error instanceof MMLScriptImportApiError) return error;
  const source = objectValue(error);
  const response = objectValue(source?.response);
  const body = objectValue(response?.data);
  const data = objectValue(body?.data);
  const status = typeof response?.status === 'number' ? response.status : 0;
  const code = stringValue(body?.code) || stringValue(data?.code);
  const message = stringValue(body?.message)
    || stringValue(body?.msg)
    || stringValue(data?.message)
    || stringValue(source?.message)
    || 'MML script import request failed';
  const validationPayload = (status === 422 || status === 409)
    ? validationPayloadFromError(body)
    : undefined;
  return new MMLScriptImportApiError({
    status,
    code,
    message,
    validation: validationPayload ? mapBackendScriptImportValidation(validationPayload) : undefined,
  });
}

function filenameFromContentDisposition(value: unknown, fallback: string): string {
  if (typeof value !== 'string') return fallback;
  const encoded = /filename\*=UTF-8''([^;]+)/i.exec(value)?.[1];
  if (encoded) {
    try {
      return decodeURIComponent(encoded);
    } catch {
      return fallback;
    }
  }
  const quoted = /filename="([^"]+)"/i.exec(value)?.[1];
  return quoted || fallback;
}

function mapTaskResultsStats(
  bs: BackendMMLTaskResultsStats | undefined
): MMLTaskResultsStats | undefined {
  if (!bs) return undefined;
  const pathTranslations: MMLPathTranslationView[] = (bs.path_translations || []).map((p) => ({
    standardPath: p.standard_path,
    privatePath: p.private_path,
    translationSource: p.translation_source as PathTranslationSource,
    translated: p.translated,
  }));
  return {
    pathTranslations: pathTranslations.length > 0 ? pathTranslations : undefined,
    productResolved: bs.product_resolved,
    matchedProductClass: bs.matched_product_class || undefined,
    pathTranslationSource: (bs.path_translation_source || undefined) as PathTranslationSource | undefined,
  };
}

function mapBackendResult(br: Record<string, unknown>): DeviceTaskResultItem {
  const requestMethod = (br.request_method as string) || '';
  return {
    deviceSn: (br.device_sn as string) || '',
    deviceTaskId: (br.device_task_id as string) || undefined,
    commandIndex: typeof br.command_index === 'number' ? (br.command_index as number) : undefined,
    planLineNo: typeof br.plan_line_no === 'number' ? (br.plan_line_no as number) : undefined,
    planDeviceSn: (br.plan_device_sn as string) || undefined,
    planOrder: typeof br.plan_order === 'number' ? (br.plan_order as number) : undefined,
    planRawLine: (br.plan_raw_line as string) || undefined,
    commandCode: (br.command_code as string) || undefined,
    commandName: (br.command_name as string) || undefined,
    operationType: (br.operation_type as string) || undefined,
    deviceName: (br.device_name as string) || undefined,
    mmlScript: (br.mml_script as string) || (br.command as string) || undefined,
    status: (br.status as DeviceTaskResultItem['status']) || undefined,
    request: requestMethod
      ? {
          method: requestMethod,
          payload: br.request_payload,
          rawRequest: (br.raw_request as string) || undefined,
          cwmpId: (br.request_cwmp_id as string) || undefined,
          commandKey: (br.request_command_key as string) || undefined,
        }
      : undefined,
    result: {
      success: Boolean(br.success),
      rawOutput: (br.raw_output as string) || '',
      parsedData: br.parsed_data as Record<string, unknown> | undefined,
      executionTime: (br.execution_time as number) || 0,
      timestamp: (br.timestamp as string) || '',
    },
    failReason: (br.fail_reason as string) || (br.error_message as string) || undefined,
    startedAt: (br.started_at as string) || undefined,
    finishedAt: (br.finished_at as string) || undefined,
  };
}

function mapBackendCommandInput(c: Record<string, unknown> | null | undefined): MMLTaskCommandInput {
  const src = c ?? {};
  return {
    commandCode: typeof src.command_code === 'string' ? src.command_code : JSON.stringify(src),
    operationType: typeof src.operation_type === 'string' ? src.operation_type : undefined,
    paramPaths: Array.isArray(src.param_paths)
      ? (src.param_paths as unknown[]).filter((p): p is string => typeof p === 'string')
      : undefined,
    parameters:
      src.parameters && typeof src.parameters === 'object'
        ? (src.parameters as Record<string, unknown>)
        : undefined,
    rawPathMode: typeof src.raw_path_mode === 'string' ? src.raw_path_mode : undefined,
  };
}

function mapBackendPlanItem(item: BackendMMLPlanItem): MMLTaskPlanItem {
  const commandSource = item.command ?? {
    command_code: item.command_code,
    operation_type: item.operation_type,
    parameters: item.parameters,
  };
  return {
    lineNo: item.line_no ?? 0,
    deviceSn: item.device_sn ?? '',
    order: item.order ?? 0,
    rawLine: item.raw_line || undefined,
    command: mapBackendCommandInput(commandSource),
  };
}

function mapCommandToBackend(cmd: string | MMLTaskCommandInput | MMLTaskCommandDetail): Record<string, unknown> {
  if (typeof cmd === 'string') return { command_code: cmd };
  const detail = cmd as MMLTaskCommandInput & MMLTaskCommandDetail;
  const entry: Record<string, unknown> = {
    command_code: detail.commandCode,
  };
  if (detail.operationType) entry.operation_type = detail.operationType;
  if (detail.paramPaths) entry.param_paths = detail.paramPaths;
  if (detail.paramValues) entry.param_values = detail.paramValues;
  if (detail.parameters) entry.parameters = detail.parameters;
  if (detail.rawPathMode) entry.raw_path_mode = detail.rawPathMode;
  return entry;
}

function mapPlanItemToBackend(item: MMLTaskPlanItem): Record<string, unknown> {
  return {
    line_no: item.lineNo,
    device_sn: item.deviceSn,
    order: item.order,
    raw_line: item.rawLine,
    command: mapCommandToBackend(item.command),
  };
}

function mapBackendTask(bt: BackendMMLTask): MMLTask {
  // Sprint B-6：扫 commands 数组中 orphan=true 的条目，提取其 command_code
  // 让 UI 给用户清晰提示"命令已下线"，否则 0 设备派发让人疑惑。
  const orphanCommandCodes: string[] = [];
  for (const c of bt.commands || []) {
    if (c.orphan === true && typeof c.command_code === 'string') {
      orphanCommandCodes.push(c.command_code as string);
    }
  }

  const planItems = (bt.plan_items || []).map(mapBackendPlanItem);

  // commandsDetail：保留 operation_type + param_paths + param_values，供"任务记录-查看"
  // 页展示用户当时勾选了哪些 path。
  //
  // 路径来源两种形态（按时间顺序）：
  //   1) c.param_refs[].tr069_path —— 当前主流形态：后端把 mml_command_sub_fields 直接
  //      关联进来，每条 ref 含 param_code / param_name_zh / is_writable 元信息
  //   2) c.param_paths[] —— 老形态（raw param_paths 模式 / 部分历史任务），纯字符串数组
  // 同时支持两种，优先取 param_refs，没有再退回 param_paths；都空则 undefined。
  const commandsDetail: MMLTaskCommandDetail[] = (bt.commands || []).map((c) => {
    const refsRaw = Array.isArray(c.param_refs) ? (c.param_refs as unknown[]) : [];
    const pathsFromRefs = refsRaw
      .map((r) => {
        if (r && typeof r === 'object' && typeof (r as { tr069_path?: unknown }).tr069_path === 'string') {
          return (r as { tr069_path: string }).tr069_path;
        }
        return null;
      })
      .filter((p): p is string => Boolean(p));
    const pathsFromLegacy = Array.isArray(c.param_paths)
      ? (c.param_paths as unknown[]).filter((p): p is string => typeof p === 'string')
      : [];
    const paramPaths = pathsFromRefs.length > 0 ? pathsFromRefs : pathsFromLegacy;
    const paramRefs = refsRaw.flatMap((r) => {
      if (!r || typeof r !== 'object') return [];
      const ref = r as { param_code?: unknown; tr069_path?: unknown };
      if (typeof ref.tr069_path !== 'string' && typeof ref.param_code !== 'string') return [];
      return [{
        paramCode: typeof ref.param_code === 'string' ? ref.param_code : undefined,
        tr069Path: typeof ref.tr069_path === 'string' ? ref.tr069_path : undefined,
      }];
    });
    return {
      commandCode: typeof c.command_code === 'string' ? c.command_code : JSON.stringify(c),
      // 后端 GetTask 注入的友好命令名（命令记录 / 执行结果显示「列出 设备基本信息」而非 command_code）。
      commandName: typeof c.command_name === 'string' ? c.command_name : undefined,
      operationType:
        typeof c.operation_type === 'string'
          ? (c.operation_type as MMLTaskCommandDetail['operationType'])
          : undefined,
      paramPaths: paramPaths.length > 0 ? paramPaths : undefined,
      paramRefs: paramRefs.length > 0 ? paramRefs : undefined,
      paramValues: Array.isArray(c.param_values) ? (c.param_values as unknown[]) : undefined,
      // 裸路径/自定义 MOD 把下发值存于 parameters（path→value map），非 param_values 数组。
      parameters:
        c.parameters && typeof c.parameters === 'object'
          ? (c.parameters as Record<string, unknown>)
          : undefined,
      planLineNo: typeof c.plan_line_no === 'number' ? (c.plan_line_no as number) : undefined,
      planDeviceSn: typeof c.plan_device_sn === 'string' ? c.plan_device_sn : undefined,
      planOrder: typeof c.plan_order === 'number' ? (c.plan_order as number) : undefined,
      planRawLine: typeof c.plan_raw_line === 'string' ? c.plan_raw_line : undefined,
    };
  });

  return {
    id: bt.id,
    taskName: bt.task_name,
    scriptId: bt.script_id || undefined,
    scriptName: bt.script_name || undefined,
    taskOrigin: (bt.task_origin || (bt.script_id ? 'script' : 'console')) as MMLTask['taskOrigin'],
    deviceSns: bt.device_sns || [],
    commands: (bt.commands || []).map((c) => {
      // Backend stores commands as {command_code: "...", ...params}
      // Frontend expects a flat string array
      if (typeof c.command_code === 'string') return c.command_code as string;
      return JSON.stringify(c);
    }),
    commandCount: bt.command_count ?? (bt.commands || []).length,
    executeMode: (bt.execute_mode || 'common') as MMLTask['executeMode'],
    planItems: planItems.length > 0 ? planItems : undefined,
    planItemCount: bt.plan_item_count ?? planItems.length,
    planStats:
      planItems.length > 0
        ? {
            totalPlanItems: planItems.length,
            totalDevices: new Set(planItems.map((p) => p.deviceSn).filter(Boolean)).size,
          }
        : undefined,
    commandsDetail: commandsDetail.length > 0 ? commandsDetail : undefined,
    orphanCommandCodes: orphanCommandCodes.length > 0 ? orphanCommandCodes : undefined,
    status: bt.status as MMLTask['status'],
    results: (bt.results || []).map(mapBackendResult),
    creator: bt.creator,
    createdAt: bt.created_at,
    updatedAt: bt.updated_at,
    // Scheduling
    executeType: (bt.execute_type ?? 'immediate') as MMLTask['executeType'],
    scheduledAt: bt.scheduled_at || undefined,
    periodStart: bt.period_start || undefined,
    periodEnd: bt.period_end || undefined,
    periodTime: bt.period_time || undefined,
    // Retry strategy
    offlineRetry: bt.offline_retry ?? false,
    offlineRetryWait: bt.offline_retry_wait ?? 60,
    failedRetry: bt.failed_retry ?? false,
    failedRetryCount: bt.failed_retry_count ?? 3,
    failedRetryInterval: bt.failed_retry_interval ?? 5,
    // Execution timestamps
    startedAt: bt.started_at || undefined,
    finishedAt: bt.finished_at || undefined,
    // Statistics
    totalDevices: bt.total_devices ?? 0,
    successCount: bt.success_count ?? 0,
    failedCount: bt.failed_count ?? 0,
    result: (bt.result || undefined) as MMLTask['result'],
    latestRun: mapBackendTaskRun(bt.latest_run),
    // Scheduler fields (P2/P3)
    nextTriggerAt: bt.next_trigger_at || undefined,
    parentTaskId: bt.parent_task_id || undefined,
    // T-0168: 翻译审计列映射；后端默认值 product_resolved=true（migration 000171）
    productResolved: bt.product_resolved ?? true,
    matchedProductId: bt.matched_product_id || undefined,
    matchedProductClass: bt.matched_product_class || undefined,
    pathTranslationSource: (bt.path_translation_source || undefined) as MMLTask['pathTranslationSource'],
    pathTranslationWarning: bt.path_translation_warning
      ? {
          anyMiss: bt.path_translation_warning.any_miss,
          deviceCount: bt.path_translation_warning.device_count,
          pathCount: bt.path_translation_warning.path_count,
        }
      : undefined,
  };
}

function mapBackendTaskRun(run?: BackendMMLTaskRun | null): MMLTask['latestRun'] {
  if (!run) return undefined;
  return {
    id: run.id,
    executeType: (run.execute_type ?? 'immediate') as MMLTask['executeType'],
    executeMode: (run.execute_mode || 'common') as MMLTask['executeMode'],
    status: run.status as MMLTask['status'],
    result: (run.result || undefined) as MMLTask['result'],
    totalDevices: run.total_devices ?? 0,
    successCount: run.success_count ?? 0,
    failedCount: run.failed_count ?? 0,
    commandCount: run.command_count ?? 0,
    planItemCount: run.plan_item_count ?? 0,
    startedAt: run.started_at || undefined,
    finishedAt: run.finished_at || undefined,
    createdAt: run.created_at,
    updatedAt: run.updated_at,
  };
}

function mapBackendCustomCommand(bc: BackendMMLCustomCommand): MMLCustomCommand {
  return {
    id: bc.id,
    commandName: bc.command_name,
    commandCode: bc.command_code,
    operationType: bc.operation_type as MMLCustomCommand['operationType'],
    commandScope: bc.command_scope as MMLCustomCommand['commandScope'],
    categoryGroup: bc.category_group || '',
    parameters: (bc.parameters as Record<string, string | number | boolean>) || {},
    paramPaths: bc.param_paths || [],
    description: bc.description || '',
    creator: bc.creator,
    createdAt: bc.created_at,
    updatedAt: bc.updated_at,
  };
}

function mapBackendCustomCommandPath(
  path: BackendMMLCustomCommandPath,
): MMLCustomCommandPathDef {
  return {
    id: path.id,
    commandId: path.command_id,
    standardPathId: path.standard_path_id,
    standardPath: path.standard_path,
    entryType: path.entry_type,
    access: path.access,
    dataType: path.data_type,
    description: path.description,
    minValue: path.min_value ?? undefined,
    maxValue: path.max_value ?? undefined,
    defaultSelected: path.default_selected,
    sortOrder: path.sort_order,
    mutable: path.mutable,
  };
}

// ---------------------------------------------------------------------------
// Public API  (matches the mmlService mock interface 1-to-1)
// ---------------------------------------------------------------------------

export const mmlApi = {
  // --- Commands ---

  async getCommands(
    params: { keyword?: string; category?: string } & PageRequest
  ): Promise<PageResponse<MMLCommand>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.keyword) query.search = params.keyword;
    if (params.category) query.category = params.category;

    const { data } = await http.get<BackendListResponse<BackendMMLCommand>>(
      '/mml/commands',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendCommand),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getAllCommands(): Promise<MMLCommand[]> {
    const allCommands: MMLCommand[] = [];
    let page = 1;
    const pageSize = 100;
    let total: number;
    do {
      const { data } = await http.get<BackendListResponse<BackendMMLCommand>>(
        '/mml/commands',
        { params: { page, page_size: pageSize } }
      );
      const items = (data.items || []).map(mapBackendCommand);
      allCommands.push(...items);
      total = data.total || 0;
      page++;
    } while (allCommands.length < total);
    return allCommands;
  },

  async getCommandById(id: string): Promise<MMLCommand | null> {
    try {
      const { data } = await http.get<BackendMMLCommand>(
        `/mml/commands/${id}`
      );
      return mapBackendCommand(data);
    } catch {
      return null;
    }
  },

  // --- Execute ---

  async executeCommand(
    commandCodeOrPayload: string | Record<string, unknown>,
    deviceSns?: string[],
    params?: Record<string, unknown>
  ): Promise<MMLTask> {
    const payload: Record<string, unknown> =
      typeof commandCodeOrPayload === 'string'
        ? {
            command_code: commandCodeOrPayload,
            device_sns: deviceSns ?? [],
            ...(params ? { parameters: params } : {}),
          }
        : commandCodeOrPayload;

    const { data } = await http.post<BackendMMLTask>('/mml/execute', payload);
    return mapBackendTask(data);
  },

  /**
   * Sprint B-5：按组批量执行 mml_command_groups 下的全部命令。
   * 后端会展开 group 下 N 条 mml_commands 为单 mml_task.commands[]，
   * Fanouter 串行下发；operation_filter 可挑 ["LST"] / ["MOD"] 等。
   */
  async executeGroup(
    groupId: string,
    payload: {
      deviceSns: string[];
      parameters?: Record<string, unknown>;
      taskName?: string;
      operationFilter?: string[];
      executeType?: string;
      scheduledAt?: string;
    }
  ): Promise<MMLTask> {
    const body: Record<string, unknown> = {
      device_sns: payload.deviceSns,
    };
    if (payload.parameters) body.parameters = payload.parameters;
    if (payload.taskName) body.task_name = payload.taskName;
    if (payload.operationFilter && payload.operationFilter.length > 0) {
      body.operation_filter = payload.operationFilter;
    }
    if (payload.executeType) body.execute_type = payload.executeType;
    if (payload.scheduledAt) body.scheduled_at = payload.scheduledAt;

    const { data } = await http.post<BackendMMLTask>(
      `/mml/groups/${groupId}/execute`,
      body
    );
    return mapBackendTask(data);
  },

  // --- Scripts ---

  async validateScriptImport(file: File): Promise<MMLScriptImportValidation> {
    const body = new FormData();
    body.append('file', file);
    try {
      const { data } = await http.post<BackendMMLScriptImportValidation>(
        '/mml/scripts/import/validate',
        body,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      );
      return mapBackendScriptImportValidation(data);
    } catch (error) {
      throw normalizeMMLScriptImportApiError(error);
    }
  },

  async validateScriptReplacement(
    id: string,
    file: File,
  ): Promise<MMLScriptImportValidation> {
    const body = new FormData();
    body.append('file', file);
    try {
      const { data } = await http.post<BackendMMLScriptImportValidation>(
        `/mml/scripts/${id}/import/validate`,
        body,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      );
      return mapBackendScriptImportValidation(data);
    } catch (error) {
      throw normalizeMMLScriptImportApiError(error);
    }
  },

  async createImportedScript(input: MMLImportedScriptCreateInput): Promise<MMLScript> {
    const payload: Record<string, unknown> = {
      validation_token: input.validationToken,
      script_name: input.scriptName,
      description: input.description,
      tags: input.tags,
    };
    if (input.requestId) payload.request_id = input.requestId;
    const { data } = await http.post<BackendMMLScript>('/mml/scripts/import', payload);
    return mapBackendScript(data);
  },

  async replaceImportedScript(
    id: string,
    input: MMLImportedScriptReplaceInput,
  ): Promise<MMLScript> {
    const payload: Record<string, unknown> = {
      validation_token: input.validationToken,
      script_name: input.scriptName,
      description: input.description,
      tags: input.tags,
      expected_updated_at: input.expectedUpdatedAt,
    };
    if (input.requestId) payload.request_id = input.requestId;
    const { data } = await http.put<BackendMMLScript>(`/mml/scripts/${id}/import`, payload);
    return mapBackendScript(data);
  },

  async createScriptExecution(
    id: string,
    input: MMLScriptExecutionInput,
  ): Promise<{ task: MMLTask; validation: MMLScriptImportValidation }> {
    const payload: Record<string, unknown> = {
      task_name: input.taskName,
      execute_type: input.executeType || 'immediate',
      offline_retry: input.offlineRetry ?? false,
      offline_retry_wait: input.offlineRetryWait ?? 60,
      failed_retry: input.failedRetry ?? false,
      failed_retry_count: input.failedRetryCount ?? 3,
      failed_retry_interval: input.failedRetryInterval ?? 5,
      confirm_warnings: input.confirmWarnings ?? false,
    };
    if (input.scheduledAt) payload.scheduled_at = input.scheduledAt;
    if (input.periodStart) payload.period_start = input.periodStart;
    if (input.periodEnd) payload.period_end = input.periodEnd;
    if (input.periodTime) payload.period_time = input.periodTime;
    if (input.requestId) payload.request_id = input.requestId;
    try {
      const { data } = await http.post<{
        task: BackendMMLTask;
        validation: BackendMMLScriptImportValidation;
      }>(`/mml/scripts/${id}/executions`, payload);
      return {
        task: mapBackendTask(data.task),
        validation: mapBackendScriptImportValidation(data.validation),
      };
    } catch (error) {
      throw normalizeMMLScriptImportApiError(error);
    }
  },

  async downloadScriptImportTemplate(): Promise<MMLScriptImportTemplate> {
    const response = await http.get<Blob>('/mml/scripts/import/template', {
      responseType: 'blob',
    });
    return {
      blob: response.data,
      filename: filenameFromContentDisposition(
        response.headers?.['content-disposition'],
        'MMLTemplate.txt',
      ),
    };
  },

  async getScripts(
    p: PageRequest & { search?: string; creator?: string }
  ): Promise<PageResponse<MMLScript>> {
    const query: Record<string, unknown> = {
      page: p.page,
      page_size: p.pageSize,
    };
    if (p.search) query.search = p.search;
    if (p.creator) query.creator = p.creator;

    const { data } = await http.get<BackendListResponse<BackendMMLScript>>(
      '/mml/scripts',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendScript),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getScriptById(id: string): Promise<MMLScript | null> {
    try {
      const { data } = await http.get<BackendMMLScript>(
        `/mml/scripts/${id}`
      );
      return mapBackendScript(data);
    } catch {
      return null;
    }
  },

  async createScript(
    data: Omit<MMLScript, 'id' | 'createTime' | 'updateTime'>
  ): Promise<MMLScript> {
    const payload = {
      script_name: data.scriptName,
      description: data.description,
      content: data.content,
      creator: data.creator,
      tags: data.tags,
    };
    const { data: bs } = await http.post<BackendMMLScript>(
      '/mml/scripts',
      payload
    );
    return mapBackendScript(bs);
  },

  async updateScript(id: string, data: Partial<MMLScript>): Promise<MMLScript> {
    const payload: Record<string, unknown> = {};
    if (data.scriptName !== undefined) payload.script_name = data.scriptName;
    if (data.description !== undefined) payload.description = data.description;
    if (data.content !== undefined) payload.content = data.content;
    if (data.creator !== undefined) payload.creator = data.creator;
    if (data.tags !== undefined) payload.tags = data.tags;

    const { data: bs } = await http.put<BackendMMLScript>(
      `/mml/scripts/${id}`,
      payload
    );
    return mapBackendScript(bs);
  },

  async deleteScripts(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/mml/scripts/${id}`);
    }
  },

  // --- Script lifecycle ---

  async startScript(id: string): Promise<MMLScript> {
    const { data } = await http.post<BackendMMLScript>(`/mml/scripts/${id}/start`);
    return mapBackendScript(data);
  },

  async pauseScript(id: string): Promise<MMLScript> {
    const { data } = await http.post<BackendMMLScript>(`/mml/scripts/${id}/pause`);
    return mapBackendScript(data);
  },

  async cancelScript(id: string): Promise<MMLScript> {
    const { data } = await http.post<BackendMMLScript>(`/mml/scripts/${id}/cancel`);
    return mapBackendScript(data);
  },

  // --- Tasks ---

  async getTasks(
    p: PageRequest & { status?: string; executeType?: string; result?: string; taskName?: string; scriptName?: string; taskOrigin?: string }
  ): Promise<PageResponse<MMLTask>> {
    const query: Record<string, unknown> = {
      page: p.page,
      page_size: p.pageSize,
    };
    if (p.status) query.status = p.status;
    if (p.executeType) query.execute_type = p.executeType;
    if (p.result) query.result = p.result;
    if (p.taskName) query.task_name = p.taskName;
    if (p.scriptName) query.script_name = p.scriptName;
    if (p.taskOrigin) query.task_origin = p.taskOrigin;

    const { data } = await http.get<BackendListResponse<BackendMMLTask>>(
      '/mml/tasks',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTaskById(id: string): Promise<MMLTask | null> {
    try {
      const { data } = await http.get<BackendMMLTask>(`/mml/tasks/${id}`);
      return mapBackendTask(data);
    } catch {
      return null;
    }
  },

  // P4 C11：历史执行列表（脚本详情页"历史执行"tab 消费）。
  // 返回同一 script_id 下全部 mml_tasks（模板行 + periodic 子实例），倒序分页。
  async getScriptRuns(
    scriptId: string,
    p: PageRequest
  ): Promise<PageResponse<MMLTask>> {
    const { data } = await http.get<BackendListResponse<BackendMMLTask>>(
      `/mml/scripts/${scriptId}/runs`,
      { params: { page: p.page, page_size: p.pageSize } }
    );
    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createTask(data: MMLTaskCreateInput): Promise<MMLTask> {
    const commands = (data.commands || []).map(mapCommandToBackend);
    const planItems = (data.planItems || []).map(mapPlanItemToBackend);
    const deviceSns = data.deviceSns?.length
      ? data.deviceSns
      : Array.from(new Set((data.planItems || []).map((p) => p.deviceSn).filter(Boolean)));
    const executeMode = data.executeMode || (planItems.length > 0 ? 'device_bound' : 'common');
    const payload: Record<string, unknown> = {
      task_name: data.taskName,
      script_id: data.scriptId,
      device_sns: deviceSns,
      execute_mode: executeMode,
      total_devices: deviceSns.length,
      creator: data.creator || '',
      execute_type: data.executeType || 'immediate',
      offline_retry: data.offlineRetry || false,
      offline_retry_wait: data.offlineRetryWait || 60,
      failed_retry: data.failedRetry || false,
      failed_retry_count: data.failedRetryCount || 3,
      failed_retry_interval: data.failedRetryInterval || 5,
    };
    if (commands.length > 0) payload.commands = commands;
    if (planItems.length > 0) payload.plan_items = planItems;
    if (data.scheduledAt) payload.scheduled_at = data.scheduledAt;
    if (data.periodStart) payload.period_start = data.periodStart;
    if (data.periodEnd) payload.period_end = data.periodEnd;
    if (data.periodTime) payload.period_time = data.periodTime;

    // 新建脚本任务走 /mml/tasks（to-do-list #7），与"临时执行命令"
    // 的 /mml/execute 区分。两者后端共享实现。
    const { data: bt } = await http.post<BackendMMLTask>(
      '/mml/tasks',
      payload
    );
    return mapBackendTask(bt);
  },

  async executeScript(
    scriptId: string,
    deviceSns: string[]
  ): Promise<MMLTask> {
    const payload = {
      script_id: scriptId,
      device_sns: deviceSns,
    };
    const { data: bt } = await http.post<BackendMMLTask>(
      '/mml/execute',
      payload
    );
    return mapBackendTask(bt);
  },

  // --- Task control ---

  async startTask(id: string): Promise<MMLTask> {
    const { data } = await http.post<BackendMMLTask>(`/mml/tasks/${id}/start`);
    return mapBackendTask(data);
  },

  async startTasks(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.post(`/mml/tasks/${id}/start`);
    }
  },

  async pauseTask(id: string): Promise<MMLTask> {
    const { data } = await http.post<BackendMMLTask>(`/mml/tasks/${id}/pause`);
    return mapBackendTask(data);
  },

  async cancelTask(id: string): Promise<MMLTask> {
    const { data } = await http.post<BackendMMLTask>(`/mml/tasks/${id}/cancel`);
    return mapBackendTask(data);
  },

  async cancelTasks(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.post(`/mml/tasks/${id}/cancel`);
    }
  },

  async deleteTask(id: string): Promise<void> {
    await http.delete(`/mml/tasks/${id}`);
  },

  async deleteTasks(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/mml/tasks/${id}`);
    }
  },

  // --- Task results ---

  async getTaskResults(
    id: string,
    page = 1,
    pageSize = 20
  ): Promise<PageResponse<DeviceTaskResultItem, MMLTaskResultsStats>> {
    // T-0168: 响应 stats 字段携带任务级翻译审计 + per-path 翻译详情
    const { data } = await http.get<
      BackendListResponse<Record<string, unknown>> & { stats?: BackendMMLTaskResultsStats }
    >(`/mml/tasks/${id}/results`, { params: { page, page_size: pageSize } });
    return {
      items: (data.items || []).map(mapBackendResult),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
      stats: mapTaskResultsStats(data.stats),
    };
  },

  /**
   * 解析 TR-069 path → 友好名（standard_params.description，即「设备模型 path 字典」里的对应名称）。
   * 供「指定参数」(裸路径)执行的命令记录命名。按 path 精确匹配 standard-params 字典；
   * 无权限/未命中时该 path 缺省，调用方回退路径叶子名。
   */
  async resolveParamNames(paths: string[]): Promise<Record<string, string>> {
    const uniq = Array.from(new Set(paths.map((p) => p.trim()).filter(Boolean)));
    const out: Record<string, string> = {};
    await Promise.all(
      uniq.map(async (p) => {
        try {
          // Search 走 URL 直拼（绕过 http 拦截器的 camelCase→snake_case 改名：后端字段名为
          // Search，被转成 search 会失效）；page/page_size 已是 snake，可走 params。
          const { data } = await http.get<
            BackendListResponse<{ standard_path: string; description: string }>
          >(`/mml/admin/standard-params?Search=${encodeURIComponent(p)}`, {
            params: { page: 1, page_size: 20 },
          });
          const hit = (data.items || []).find((it) => it.standard_path === p);
          if (hit?.description) out[p] = hit.description;
        } catch {
          /* 无权限/失败 → 跳过；调用方回退叶子名 */
        }
      }),
    );
    return out;
  },

  // --- Result CSV export (MinIO) ---

  /**
   * 生成「全设备汇总」结果 CSV，落 MinIO（reports/mml-results/ 目录），地址记入
   * mml_tasks.export_object，返回 object key 与浏览器可下载的预签名 URL。
   */
  async exportTaskCSV(taskId: string): Promise<{ object: string; downloadUrl: string }> {
    const { data } = await http.post<{ object: string; download_url: string }>(
      `/mml/tasks/${taskId}/export`
    );
    // http 拦截器已 snake→camel：download_url → downloadUrl
    const d = data as unknown as { object: string; downloadUrl: string };
    return { object: d.object, downloadUrl: d.downloadUrl };
  },

  /** 生成「单设备」结果 CSV，落 MinIO 并记入 mml_tasks.device_export_objects[sn]。 */
  async exportTaskDeviceCSV(
    taskId: string,
    deviceSn: string
  ): Promise<{ object: string; downloadUrl: string }> {
    const { data } = await http.post<{ object: string; download_url: string }>(
      `/mml/tasks/${taskId}/devices/${encodeURIComponent(deviceSn)}/export`
    );
    const d = data as unknown as { object: string; downloadUrl: string };
    return { object: d.object, downloadUrl: d.downloadUrl };
  },

  /**
   * 同源流式下载「全设备汇总」CSV（GET，responseType=blob）。
   * 替代预签名 MinIO URL：浏览器从 app 同源（经 /api 代理）拿数据，跨主机/反代访问也可靠。
   */
  async downloadTaskCsv(taskId: string): Promise<Blob> {
    const { data } = await http.get(`/mml/tasks/${taskId}/export/download`, {
      responseType: 'blob',
    });
    return data as Blob;
  },

  /** 同源流式下载「单设备」CSV（GET，responseType=blob）。 */
  async downloadTaskDeviceCsv(taskId: string, deviceSn: string): Promise<Blob> {
    const { data } = await http.get(
      `/mml/tasks/${taskId}/devices/${encodeURIComponent(deviceSn)}/export/download`,
      { responseType: 'blob' }
    );
    return data as Blob;
  },

  // --- Dangerous command check ---

  async checkDangerous(
    commandCode: string
  ): Promise<{ dangerous: boolean; info: { Name: string; Desc: string } | null }> {
    const { data } = await http.get('/mml/dangerous-check', {
      params: { command_code: commandCode },
    });
    return data;
  },

  // --- Custom Commands (templates) ---
  // Note: Backend routes still use /mml/templates path for backward compat.

  async getTemplates(
    params?: {
      commandCode?: string;
      operationType?: string;
      templateScope?: string;
      productId?: string;
    } & PageRequest
  ): Promise<PageResponse<MMLCustomCommand>> {
    const query: Record<string, unknown> = {
      page: params?.page ?? 1,
      page_size: params?.pageSize ?? 20,
    };
    if (params?.commandCode) query.command_code = params.commandCode;
    if (params?.operationType) query.operation_type = params.operationType;
    // T-0090-d：后端 handler.go L722 实际读 `command_scope`，本端旧 `template_scope`
    // 与之不一致导致 scope 过滤被静默忽略；按后端契约对齐。
    if (params?.templateScope) query.command_scope = params.templateScope;
    if (params?.productId) query.product_id = params.productId;

    const { data } = await http.get<BackendListResponse<BackendMMLCustomCommand>>(
      '/mml/templates',
      { params: query }
    );
    return {
      items: (data.items || []).map(mapBackendCustomCommand),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTemplatePaths(commandId: string): Promise<MMLCustomCommandPathDef[]> {
    const { data } = await http.get<{ items: BackendMMLCustomCommandPath[] }>(
      `/mml/templates/${commandId}/paths`,
    );
    return (data.items ?? []).map(mapBackendCustomCommandPath);
  },

  async createTemplate(
    tmpl: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'>
  ): Promise<MMLCustomCommand> {
    const payload = {
      command_name: tmpl.commandName,
      command_code: tmpl.commandCode,
      operation_type: tmpl.operationType,
      command_scope: tmpl.commandScope,
      category_group: tmpl.categoryGroup,
      parameters: tmpl.parameters,
      param_paths: tmpl.paramPaths,
      description: tmpl.description,
    };
    const { data } = await http.post<BackendMMLCustomCommand>(
      '/mml/templates',
      payload
    );
    return mapBackendCustomCommand(data);
  },

  async updateTemplate(
    id: string,
    tmpl: Partial<MMLCustomCommand>
  ): Promise<MMLCustomCommand> {
    const payload: Record<string, unknown> = {};
    if (tmpl.commandName !== undefined) payload.command_name = tmpl.commandName;
    if (tmpl.commandCode !== undefined) payload.command_code = tmpl.commandCode;
    if (tmpl.operationType !== undefined) payload.operation_type = tmpl.operationType;
    if (tmpl.commandScope !== undefined) payload.command_scope = tmpl.commandScope;
    if (tmpl.parameters !== undefined) payload.parameters = tmpl.parameters;
    if (tmpl.paramPaths !== undefined) payload.param_paths = tmpl.paramPaths;
    if (tmpl.description !== undefined) payload.description = tmpl.description;

    const { data } = await http.put<BackendMMLCustomCommand>(
      `/mml/templates/${id}`,
      payload
    );
    return mapBackendCustomCommand(data);
  },

  async deleteTemplate(id: string): Promise<void> {
    await http.delete(`/mml/templates/${id}`);
  },

  async cloneTemplate(id: string): Promise<MMLCustomCommand> {
    const { data } = await http.post<BackendMMLCustomCommand>(
      `/mml/templates/${id}/clone`
    );
    return mapBackendCustomCommand(data);
  },

  // --- T-0123-P2 Console 5 端点 ---

  /**
   * GET /mml/group-tree?root=&lang=&product_class=&device_sn= — 命令分组树。
   *
   * productClass（T-0172）非空时后端按"该产品族 default param_mappings"过滤命令：
   *   - 命令的 target_paths 至少 1 条在 supported set → 显示
   *   - ADD/RMV 的 target_object 是 supported set 中 path 前缀 → 显示
   *   - 孤儿设备（productClass 未匹配产品）→ 显示全部命令，每条 product_resolved=false
   *   - 每条返回命令带 supported_path_count / unsupported_paths / product_resolved 注解
   *   - 空 group（含 chapter）被剔除
   *
   * deviceKey 非空时后端按设备对应 ParamModel 的支持集合过滤，和 sub-fields 使用同一口径。
   * productClass 缺省 / 空串、deviceKey 缺省 / 空串 → 不做过滤（向后兼容旧调用）。
   */
  async buildGroupTree(
    root?: string,
    lang: string = 'zh-CN',
    productClass?: string,
    deviceKey?: string,
  ): Promise<GroupTreeNode[]> {
    const params: Record<string, string> = { lang };
    if (root) params.root = root;
    if (productClass) params.product_class = productClass;
    if (deviceKey) params.device_sn = deviceKey;
    const { data } = await http.get<{ tree: BackendGroupTreeNode[] } | BackendGroupTreeNode[]>(
      '/mml/group-tree',
      { params }
    );
    // 后端响应可能是 { tree: [...] } 或 [...] 形态；兼容两种
    const arr = Array.isArray(data) ? data : (data?.tree ?? []);
    return arr.map(mapGroupTreeNode);
  },

  /**
   * Task #9: GET /mml/group-tree?format=flat&lang=&product_class=&device_sn= — 扁平化命令分组树。
   *
   * 与 buildGroupTree 区别：
   *   - 后端预聚合为「分组 → 命令叶子」两层；不返回 ltree children/sub_fields
   *   - 命令直接携带 object_path（LST=string[]、MOD=ModParamPath[]、ADD/RMV=string）
   *   - command.name 已含操作前缀（"LST/MOD/ADD/RMV <中文名>"）
   *
   * 注意：axios 全局拦截器会做 snake→camel；为保持后端契约（path/type/min/max/max_length
   * 等关键字段在 ModParamPath 内是 spec 定义的 wire 名），此方法使用泛型断言
   * 避免拦截器对 object_path 内嵌字段的转换破坏类型。如果业务后续依赖 max_length
   * 等下划线字段被转 camel，调用方应在消费层做兼容；本层保持透传。
   */
  async buildGroupTreeFlat(
    lang: string = 'zh-CN',
    productClass?: string,
    deviceKey?: string,
  ): Promise<FlatGroupTreeResponse> {
    const params: Record<string, string> = { format: 'flat', lang };
    if (productClass) params.product_class = productClass;
    if (deviceKey) params.device_sn = deviceKey;
    const { data } = await http.get<FlatGroupTreeResponse>('/mml/group-tree', {
      params,
    });
    return {
      groups: Array.isArray(data?.groups) ? data.groups : [],
    };
  },

  /**
   * GET /mml/commands/:id/sub-fields?lang=&product_class=&device_sn= — 命令的 sub-fields。
   *
   * 后端 paramModelID 解析优先级:
   *   1. product_class 非空 → 与命令树命令名计数同源(supportedPathsRepo)
   *   2. device_sn / device_id 非空 → 老 T-0170 路径(admin 工具兼容)
   *   3. 都不传 → admin 视图全集
   *
   * 控制台可传 deviceKey 或 productClass；传入后返回的 path 是
   * 当前 ParamModel 支持集合与当前命令 MML sub-fields 的交集。
   */
  async getCommandSubFields(
    commandId: string,
    lang: string = 'zh-CN',
    deviceKey?: string,
    productClass?: string,
  ): Promise<SubFieldDef[]> {
    const params: Record<string, string> = { lang };
    if (productClass) params.product_class = productClass;
    if (deviceKey) params.device_sn = deviceKey;
    const { data } = await http.get<{ sub_fields: BackendSubField[] } | BackendSubField[]>(
      `/mml/commands/${commandId}/sub-fields`,
      { params }
    );
    const arr = Array.isArray(data) ? data : (data?.sub_fields ?? []);
    return arr.map(mapSubField);
  },

  /**
   * GET /mml/unsupported-paths?product_id= —— 该产品已记录的不支持参数 PATH（含读/写标记，
   * source：MML 执行 path 不支持类故障自学习表）。前端「选择命令 / 配置参数」按命令读/写类型过滤。
   * 主用 product_id（产品下拉直给）；deviceSn 为兼容兜底（后端反算 product_id）。
   */
  async getUnsupportedPaths(productId?: string, deviceSn?: string): Promise<UnsupportedPathInfo[]> {
    if (!productId && !deviceSn) return [];
    const params: Record<string, string> = {};
    if (productId) params.product_id = productId;
    if (deviceSn) params.device_sn = deviceSn;
    const { data } = await http.get<{ paths: BackendUnsupportedPath[] } | BackendUnsupportedPath[]>(
      '/mml/unsupported-paths',
      { params },
    );
    const arr = Array.isArray(data) ? data : (data?.paths ?? []);
    return arr.map((p) => ({
      path: p.path,
      readUnsupported: Boolean(p.read_unsupported ?? p.readUnsupported),
      writeUnsupported: Boolean(p.write_unsupported ?? p.writeUnsupported),
    }));
  },

  /**
   * Bundle C: GET /mml/commands/search?q=&lang=&limit= — 命令搜索。
   *
   * 后端 ILIKE 联合搜索 command_code / logical_name / standardPath / description。
   * q 空 → 返空数组（后端不消耗 CPU 做"全表 LIMIT 50"）。
   * 调用方应做 300ms debounce 避免每键击都触发请求。
   */
  async searchCommands(
    q: string,
    lang: string = 'zh-CN',
    limit: number = 50,
  ): Promise<SearchCommand[]> {
    const trimmed = q.trim();
    if (!trimmed) return [];
    const { data } = await http.get<{ items: BackendSearchCommand[] } | BackendSearchCommand[]>(
      '/mml/commands/search',
      { params: { q: trimmed, lang, limit } },
    );
    const arr = Array.isArray(data) ? data : (data?.items ?? []);
    // axios 拦截器已 snake → camel；matched_paths / match_reasons 是 string[]，
    // null 兜底成空数组让调用端不必再判空。
    return arr.map((r) => {
      const item = r as unknown as Partial<SearchCommand> & {
        matched_paths?: string[] | null;
        match_reasons?: string[] | null;
      };
      return {
        commandId: item.commandId ?? '',
        commandCode: item.commandCode ?? '',
        logicalCode: item.logicalCode ?? '',
        operationType: item.operationType ?? '',
        displayName: item.displayName ?? '',
        logicalName: item.logicalName ?? '',
        groupId: item.groupId ?? '',
        groupCode: item.groupCode ?? '',
        groupName: item.groupName ?? '',
        chapterCode: item.chapterCode ?? '',
        matchedPaths: item.matchedPaths ?? item.matched_paths ?? [],
        matchReasons: item.matchReasons ?? item.match_reasons ?? [],
      };
    });
  },

  /**
   * R-8.5 GET /mml/console/command-compatibility — 该 product_class 下不兼容的命令 ID 集合。
   *
   * T-0177 起孤儿 productClass 后端返 200 + `param_model_id` 零值 + 空 unsupported 数组
   * （不再 404），useCommandCompatibility 的 select 把空数组转空 Set，UI 视所有命令为
   * "已知支持"。503 service nil 仍走 axios 异常 — React Query 兜底 retry / fallback。
   */
  async getCommandCompatibility(productClass: string): Promise<CommandCompatibility> {
    const { data } = await http.get<CommandCompatibility>(
      '/mml/console/command-compatibility',
      { params: { product_class: productClass } }
    );
    return data;
  },

  /** POST /mml/render — Statement → mml 字符串片段。 */
  async renderMML(req: RenderRequest): Promise<string> {
    const payload = {
      command_id: req.commandId,
      operation_type: req.operationType,
      selected_sub_field_ids: req.selectedSubFieldIds,
      values: req.values,
      rmv_instance_index: req.rmvInstanceIndex,
    };
    const { data } = await http.post<{ mml_string: string }>('/mml/render', payload);
    return data.mml_string;
  },

  /** POST /mml/parse — mml 字符串 → Statement[]。始终 200，parse_errors 在 body。 */
  async parseMML(req: ParseRequest): Promise<ParseResponse> {
    const payload = {
      mml_string: req.mmlString,
      lang: req.lang ?? 'zh-CN',
    };
    const { data } = await http.post<{
      statements: BackendStatement[];
      parse_errors: BackendParseError[];
    }>('/mml/parse', payload);
    return {
      statements: (data.statements ?? []).map(mapBackendStatement),
      parseErrors: (data.parse_errors ?? []).map(mapBackendParseError),
    };
  },

  /** POST /mml/execute-statements — N 设备 × M statements 扇出执行。 */
  async executeStatements(req: ExecuteStatementsRequest): Promise<MMLTask> {
    const payload = {
      statements: req.statements.map(stmtToBackend),
      device_sns: req.deviceSns,
      task_name: req.taskName,
      creator: req.creator,
      executor: req.executor,
      execute_type: req.executeType,
    };
    const { data } = await http.post<BackendMMLTask>('/mml/execute-statements', payload);
    return mapBackendTask(data);
  },

  /**
   * POST /mml/console/execute-statements-structured (R-9.2)
   * 结构化通道：用户直接传 standardPath + value，后端按设备 product_class 翻译为
   * privatePath 并下发。与 executeStatements（MML 文本 round-trip）的主要差异：
   *   - StructuredStatement.paths 取 sub_field.tr069Path 列表（前端 statementToStructured 转出）
   *   - values key 同样是 standardPath
   *   - 错误体含 unknown_paths 元数据 → 调用方据此呈现 422 报错精确位置
   */
  async executeStatementsStructured(req: StructuredExecuteRequest): Promise<MMLTask> {
    const payload = {
      statements: req.statements.map(structuredStmtToBackend),
      device_sns: req.deviceSns,
      task_name: req.taskName,
      creator: req.creator,
      executor: req.executor,
      execute_type: req.executeType,
    };
    const { data } = await http.post<BackendMMLTask>(
      '/mml/console/execute-statements-structured',
      payload,
    );
    return mapBackendTask(data);
  },
};

// ---------------------------------------------------------------------------
// T-0123-P2-a Console mappers — backend snake_case ↔ frontend camelCase
// ---------------------------------------------------------------------------

function mapGroupTreeNode(n: BackendGroupTreeNode): GroupTreeNode {
  return {
    id: n.id,
    groupCode: n.code,
    path: n.path,
    displayName: n.name,
    displayNameI18n: n.name_i18n,
    displayOrder: n.display_order,
    chapterCode: n.chapter_code,
    source: n.source,
    catalogProtected: n.catalog_protected,
    commands: (n.commands ?? []).map(mapGroupTreeCommand),
    children: (n.children ?? []).map(mapGroupTreeNode),
  };
}

function mapGroupTreeCommand(c: BackendGroupTreeNode['commands'][number]): GroupTreeCommand {
  return {
    id: c.id,
    commandCode: c.command_code,
    logicalCode: c.logical_code,
    logicalName: c.logical_name,
    logicalNameI18n: c.logical_name_i18n,
    operationType: c.operation_type as MMLOperationType,
    displayName: c.display_name,
    rpcMethod: c.rpc_method,
    targetObject: c.target_object || undefined,
    targetPaths: c.target_paths,
    requireConfirm: c.require_confirm,
    source: c.source,
    catalogProtected: c.catalog_protected,
    // R-4.1.1：原样透传 instance_range_meta；后端 omitempty + 前端 helper 已把空数组归一为 undefined
    instanceRangeMeta: c.instance_range_meta,
    // T-0172 catalog filter annotations (后端 omitempty 时 c.* 为 undefined)
    supportedPathCount: c.supported_path_count,
    unsupportedPaths: c.unsupported_paths,
    productResolved: c.product_resolved,
  };
}

function mapSubField(s: BackendSubField): SubFieldDef {
  return {
    id: s.id,
    commandId: s.command_id,
    paramId: s.param_id,
    mmlCode: s.mml_code,
    label: s.label,
    labelI18n: s.label_i18n ?? {},
    tr069Path: s.tr069_path,
    valueType: s.value_type,
    accessType: s.access_type,
    isObject: s.is_object,
    minValue: s.min_value ?? undefined,
    maxValue: s.max_value ?? undefined,
    supportsAdd: s.supports_add,
    supportsDelete: s.supports_delete,
    changeApplies: s.change_applies,
    constraintText: s.constraint_text,
    constraintTextI18n: s.constraint_text_i18n ?? {},
    defaultValue: s.default_value || undefined,
    jsRegex: s.js_regex || undefined,
    validationPattern: s.validation_pattern || undefined,
    enumOptions: s.enum_options?.length ? s.enum_options : undefined,
    defaultSelected: s.default_selected,
    isRequired: s.is_required,
    sortOrder: s.sort_order,
    isSupported: s.is_supported,
  };
}

function mapBackendStatement(b: BackendStatement): Statement {
  // 注：parser 返回的 Statement 没有 commandCode / logicalNameI18n / subFields。
  // 这些是 P2-a 组件层在 appendStatement 时通过 GET /commands/:id 与
  // GET /commands/:id/sub-fields 补齐。此处保守填空，让 store 决定何时 enrich。
  return {
    uid: generateUid('stmt'),
    commandId: b.command_id,
    commandCode: '', // 待 store 层 enrich
    logicalCode: b.logical_code,
    operationType: b.operation_type as MMLOperationType,
    logicalNameI18n: {}, // 待 store 层 enrich
    subFields: [],       // 待 store 层 enrich
    selectedSubFieldIds: b.selected_sub_field_ids ?? [],
    values: b.values ?? {},
    rmvInstanceIndex: b.rmv_instance_index,
    unknownCodes: b.unknown_codes ?? [],
  };
}

function mapBackendParseError(e: BackendParseError): ParseError {
  return {
    statementIndex: e.statement_index,
    raw: e.raw,
    reason: e.reason,
  };
}

/** 把前端 Statement 序列化为后端 BackendStatement wire 格式。 */
function stmtToBackend(s: Statement): BackendStatement {
  return {
    command_id: s.commandId,
    logical_code: s.logicalCode,
    operation_type: s.operationType,
    selected_sub_field_ids: s.selectedSubFieldIds.length > 0 ? s.selectedSubFieldIds : undefined,
    values: Object.keys(s.values).length > 0 ? s.values : undefined,
    rmv_instance_index: s.rmvInstanceIndex,
  };
}

/**
 * R-9.2 结构化 statement → backend wire 格式（snake_case）。
 * Axios 拦截器虽然会做通用 camelCase ↔ snake_case，但显式 mapper 保留
 * 字段意图（避免 instanceIndices 被错误改名）。
 */
function structuredStmtToBackend(s: StructuredStatement): Record<string, unknown> {
  const out: Record<string, unknown> = {
    command_id: s.commandId,
    operation_type: s.operationType,
    paths: s.paths,
  };
  if (s.commandCode) out.command_code = s.commandCode;
  if (s.values && Object.keys(s.values).length > 0) out.values = s.values;
  if (s.instanceSelectors && Object.keys(s.instanceSelectors).length > 0) {
    out.instance_selectors = s.instanceSelectors;
  }
  if (s.instanceIndices && s.instanceIndices.length > 0) {
    out.instance_indices = s.instanceIndices;
  }
  return out;
}
