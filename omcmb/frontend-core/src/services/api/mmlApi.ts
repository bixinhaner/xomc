import http from '../http';
import type { MMLCommand, MMLScript, MMLTask, MMLParam, MMLCustomCommand, ParamPath, MMLOperationType, DeviceTaskResultItem, MMLParamRef } from '../../types/mml';
import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  BackendStatement,
  BackendGroupTreeNode,
  BackendSubField,
  BackendParseError,
  GroupTreeNode,
  GroupTreeCommand,
  SubFieldDef,
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
}

interface BackendMMLTask {
  id: string;
  task_name: string;
  script_id: string;
  device_sns: string[] | null;
  commands: Array<Record<string, unknown>> | null;
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
  // P2/P3 Scheduler fields
  next_trigger_at?: string | null;
  parent_task_id?: string | null;
  // 整改方案 Stage 3 — 路径翻译警告（GetTask 聚合 device_tasks 后填充）
  path_translation_warning?: {
    any_miss: boolean;
    device_count: number;
    path_count: number;
  } | null;
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
    productTypes: bc.product_types || [],
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
  };
}

function mapBackendResult(br: Record<string, unknown>): DeviceTaskResultItem {
  return {
    deviceSn: (br.device_sn as string) || '',
    deviceName: (br.device_name as string) || undefined,
    mmlScript: (br.mml_script as string) || (br.command as string) || undefined,
    status: (br.status as DeviceTaskResultItem['status']) || undefined,
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

function mapBackendTask(bt: BackendMMLTask): MMLTask {
  // Sprint B-6：扫 commands 数组中 orphan=true 的条目，提取其 command_code
  // 让 UI 给用户清晰提示"命令已下线"，否则 0 设备派发让人疑惑。
  const orphanCommandCodes: string[] = [];
  for (const c of bt.commands || []) {
    if (c.orphan === true && typeof c.command_code === 'string') {
      orphanCommandCodes.push(c.command_code as string);
    }
  }

  return {
    id: bt.id,
    taskName: bt.task_name,
    scriptId: bt.script_id || undefined,
    deviceSns: bt.device_sns || [],
    commands: (bt.commands || []).map((c) => {
      // Backend stores commands as {command_code: "...", ...params}
      // Frontend expects a flat string array
      if (typeof c.command_code === 'string') return c.command_code as string;
      return JSON.stringify(c);
    }),
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
    // Scheduler fields (P2/P3)
    nextTriggerAt: bt.next_trigger_at || undefined,
    parentTaskId: bt.parent_task_id || undefined,
    pathTranslationWarning: bt.path_translation_warning
      ? {
          anyMiss: bt.path_translation_warning.any_miss,
          deviceCount: bt.path_translation_warning.device_count,
          pathCount: bt.path_translation_warning.path_count,
        }
      : undefined,
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
    let total = 0;
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
   * Sprint B-5：按组批量执行 mml_param_groups 下的全部命令。
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
    p: PageRequest & { status?: string; executeType?: string; result?: string; taskName?: string }
  ): Promise<PageResponse<MMLTask>> {
    const query: Record<string, unknown> = {
      page: p.page,
      page_size: p.pageSize,
    };
    if (p.status) query.status = p.status;
    if (p.executeType) query.execute_type = p.executeType;
    if (p.result) query.result = p.result;
    if (p.taskName) query.task_name = p.taskName;

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

  async createTask(
    data: Partial<Omit<MMLTask, 'id' | 'status' | 'results' | 'createdAt' | 'updatedAt'>> &
    Pick<MMLTask, 'taskName' | 'deviceSns' | 'commands'>
  ): Promise<MMLTask> {
    const payload: Record<string, unknown> = {
      task_name: data.taskName,
      script_id: data.scriptId,
      device_sns: data.deviceSns,
      commands: data.commands.map((cmd) => ({ command_code: cmd })),
      total_devices: data.deviceSns?.length ?? 0,
      creator: data.creator || '',
      execute_type: data.executeType || 'immediate',
      offline_retry: data.offlineRetry || false,
      offline_retry_wait: data.offlineRetryWait || 60,
      failed_retry: data.failedRetry || false,
      failed_retry_count: data.failedRetryCount || 3,
      failed_retry_interval: data.failedRetryInterval || 5,
    };
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

  async pauseTask(id: string): Promise<MMLTask> {
    const { data } = await http.post<BackendMMLTask>(`/mml/tasks/${id}/pause`);
    return mapBackendTask(data);
  },

  async cancelTask(id: string): Promise<MMLTask> {
    const { data } = await http.post<BackendMMLTask>(`/mml/tasks/${id}/cancel`);
    return mapBackendTask(data);
  },

  async deleteTask(id: string): Promise<void> {
    await http.delete(`/mml/tasks/${id}`);
  },

  // --- Task results ---

  async getTaskResults(
    id: string,
    page = 1,
    pageSize = 20
  ): Promise<PageResponse<DeviceTaskResultItem>> {
    const { data } = await http.get<BackendListResponse<Record<string, unknown>>>(
      `/mml/tasks/${id}/results`,
      { params: { page, page_size: pageSize } }
    );
    return {
      items: (data.items || []).map(mapBackendResult),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
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

  /** GET /mml/group-tree?root=&lang= — 命令分组树。 */
  async buildGroupTree(
    root?: string,
    lang: string = 'zh-CN'
  ): Promise<GroupTreeNode[]> {
    const params: Record<string, string> = { lang };
    if (root) params.root = root;
    const { data } = await http.get<{ tree: BackendGroupTreeNode[] } | BackendGroupTreeNode[]>(
      '/mml/group-tree',
      { params }
    );
    // 后端响应可能是 { tree: [...] } 或 [...] 形态；兼容两种
    const arr = Array.isArray(data) ? data : (data?.tree ?? []);
    return arr.map(mapGroupTreeNode);
  },

  /**
   * Task #9: GET /mml/group-tree?format=flat&lang= — 扁平化命令分组树。
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
    lang: string = 'zh-CN'
  ): Promise<FlatGroupTreeResponse> {
    const { data } = await http.get<FlatGroupTreeResponse>('/mml/group-tree', {
      params: { format: 'flat', lang },
    });
    return {
      groups: Array.isArray(data?.groups) ? data.groups : [],
    };
  },

  /** GET /mml/commands/:id/sub-fields?lang= — 命令的 sub-fields。 */
  async getCommandSubFields(
    commandId: string,
    lang: string = 'zh-CN'
  ): Promise<SubFieldDef[]> {
    const { data } = await http.get<{ sub_fields: BackendSubField[] } | BackendSubField[]>(
      `/mml/commands/${commandId}/sub-fields`,
      { params: { lang } }
    );
    const arr = Array.isArray(data) ? data : (data?.sub_fields ?? []);
    return arr.map(mapSubField);
  },

  /**
   * R-8.5 GET /mml/console/command-compatibility — 该 product_class 下不兼容的命令 ID 集合。
   *
   * 404 product_class 无匹配 / 503 service nil — axios 抛出，调用方（React Query）兜底
   * 走 retry / fallback；hook 层把数据转 Set<string> 给 CommandTree 装饰用。
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
    familyCode: n.family_code,
    familyNameZh: n.family_name_zh,
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
    requireConfirm: c.require_confirm,
    source: c.source,
    catalogProtected: c.catalog_protected,
    // R-4.1.1：原样透传 instance_range_meta；后端 omitempty + 前端 helper 已把空数组归一为 undefined
    instanceRangeMeta: c.instance_range_meta,
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
    supportsAdd: s.supports_add,
    supportsDelete: s.supports_delete,
    changeApplies: s.change_applies,
    constraintText: s.constraint_text,
    constraintTextI18n: s.constraint_text_i18n ?? {},
    defaultValue: s.default_value || undefined,
    jsRegex: s.js_regex || undefined,
    defaultSelected: s.default_selected,
    isRequired: s.is_required,
    sortOrder: s.sort_order,
  };
}

function mapBackendStatement(b: BackendStatement): Statement {
  // 注：parser 返回的 Statement 没有 commandCode / logicalNameI18n / subFields。
  // 这些是 P2-a 组件层在 appendStatement 时通过 GET /commands/:id 与
  // GET /commands/:id/sub-fields 补齐。此处保守填空，让 store 决定何时 enrich。
  return {
    uid: typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `stmt-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`,
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
