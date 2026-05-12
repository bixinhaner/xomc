import http from '../http';
import type { MMLCommand, MMLScript, MMLTask, MMLParam, MMLCustomCommand, ParamPath, MMLOperationType, DeviceTaskResultItem, MMLParamRef } from '../../types/mml';
import type { PageRequest, PageResponse } from '../../types/pagination';

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
  param_template: Record<string, unknown> | null;
  product_types: string[] | null;
  created_at: string;
  // Extended fields (may be absent in older backend versions)
  operation_type?: string;
  param_paths?: Array<string | { path: string; label?: string; writable?: boolean }> | null;
  supported_operations?: string[] | null;
  help_doc?: string;
  notes?: string;
  params?: BackendMMLParamRef[] | null;
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

function mapBackendCommand(bc: BackendMMLCommand): MMLCommand {
  let paramPaths: ParamPath[] | undefined;
  if (Array.isArray(bc.param_paths)) {
    paramPaths = bc.param_paths
      .map((item) => {
        if (typeof item === 'string') {
          const path = item.trim();
          if (!path) return null;
          return { path, label: path, writable: true } as ParamPath;
        }
        if (!item.path) return null;
        return {
          path: item.path,
          label: item.label || item.path,
          writable: item.writable ?? true,
        } as ParamPath;
      })
      .filter((v): v is ParamPath => v !== null);
  }

  return {
    id: bc.id,
    commandName: bc.command_name,
    commandCode: bc.command_code,
    category: bc.category,
    description: bc.description,
    params: mapParamTemplate(bc.param_template),
    productTypes: bc.product_types || [],
    operationType: bc.operation_type as MMLOperationType | undefined,
    paramPaths,
    supportedOperations: bc.supported_operations || undefined,
    helpDoc: bc.help_doc || undefined,
    notes: bc.notes || undefined,
    paramRefs: dedupeParamRefs(bc.params?.map(mapBackendParamRef)),
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
    // Backend query param name stays template_scope for backward compat
    if (params?.templateScope) query.template_scope = params.templateScope;

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
};
