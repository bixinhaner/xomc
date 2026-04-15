import http from '../http';
import type { MMLCommand, MMLScript, MMLTask, MMLResult, MMLParam, MMLTemplate } from '@/types/mml';
import type { PageRequest, PageResponse } from '@/types/pagination';

// ---------------------------------------------------------------------------
// Backend response types  (snake_case, matching omcgo/internal/omcr/mml/model.go)
// ---------------------------------------------------------------------------

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
}

interface BackendMMLScript {
  id: string;
  script_name: string;
  description: string;
  content: string;
  device_type: string;
  creator: string;
  tags: string[] | null;
  created_at: string;
  updated_at: string;
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
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendMMLTemplate {
  id: string;
  template_name: string;
  command_code: string;
  operation_type: string;
  template_scope: string;
  parameters: Record<string, unknown> | null;
  param_paths: string[] | null;
  description: string;
  product_types: string[] | null;
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

function mapBackendCommand(bc: BackendMMLCommand): MMLCommand {
  return {
    id: bc.id,
    commandName: bc.command_name,
    commandCode: bc.command_code,
    category: bc.category,
    description: bc.description,
    params: mapParamTemplate(bc.param_template),
    productTypes: bc.product_types || [],
  };
}

function mapBackendScript(bs: BackendMMLScript): MMLScript {
  return {
    id: bs.id,
    scriptName: bs.script_name,
    description: bs.description,
    content: bs.content,
    deviceType: bs.device_type,
    creator: bs.creator,
    tags: bs.tags || [],
    createTime: bs.created_at,
    updateTime: bs.updated_at,
  };
}

function mapBackendResult(br: Record<string, unknown>): {
  deviceSn: string;
  result: MMLResult;
} {
  return {
    deviceSn: (br.device_sn as string) || '',
    result: {
      success: Boolean(br.success),
      rawOutput: (br.raw_output as string) || '',
      parsedData: br.parsed_data as Record<string, unknown> | undefined,
      executionTime: (br.execution_time as number) || 0,
      timestamp: (br.timestamp as string) || '',
    },
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
  };
}

function mapBackendTemplate(bt: BackendMMLTemplate): MMLTemplate {
  return {
    id: bt.id,
    templateName: bt.template_name,
    commandCode: bt.command_code,
    operationType: bt.operation_type as MMLTemplate['operationType'],
    templateScope: bt.template_scope as MMLTemplate['templateScope'],
    parameters: (bt.parameters as Record<string, string | number | boolean>) || {},
    paramPaths: bt.param_paths || [],
    description: bt.description || '',
    productTypes: bt.product_types || [],
    creator: bt.creator,
    createdAt: bt.created_at,
    updatedAt: bt.updated_at,
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
    commandCode: string,
    deviceSns: string[],
    params?: Record<string, string | number | boolean>
  ): Promise<MMLTask> {
    const payload: Record<string, unknown> = {
      command_code: commandCode,
      device_sns: deviceSns,
    };
    if (params) payload.parameters = params;

    const { data } = await http.post<BackendMMLTask>('/mml/execute', payload);
    return mapBackendTask(data);
  },

  // --- Scripts ---

  async getScripts(p: PageRequest): Promise<PageResponse<MMLScript>> {
    const query: Record<string, unknown> = {
      page: p.page,
      page_size: p.pageSize,
    };

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
      device_type: data.deviceType,
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
    if (data.deviceType !== undefined) payload.device_type = data.deviceType;
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

  // --- Tasks ---

  async getTasks(p: PageRequest): Promise<PageResponse<MMLTask>> {
    const query: Record<string, unknown> = {
      page: p.page,
      page_size: p.pageSize,
    };

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

  async createTask(
    data: Omit<MMLTask, 'id' | 'status' | 'results' | 'createdAt' | 'updatedAt'>
  ): Promise<MMLTask> {
    const payload: Record<string, unknown> = {
      task_name: data.taskName,
      script_id: data.scriptId,
      device_sns: data.deviceSns,
      commands: data.commands.map((cmd) => ({ command_code: cmd })),
      creator: data.creator,
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

    const { data: bt } = await http.post<BackendMMLTask>(
      '/mml/execute',
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
  ): Promise<PageResponse<{ deviceSn: string; result: MMLResult }>> {
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

  // --- Templates ---

  async getTemplates(
    params?: {
      commandCode?: string;
      operationType?: string;
      templateScope?: string;
    } & PageRequest
  ): Promise<PageResponse<MMLTemplate>> {
    const query: Record<string, unknown> = {
      page: params?.page ?? 1,
      page_size: params?.pageSize ?? 20,
    };
    if (params?.commandCode) query.command_code = params.commandCode;
    if (params?.operationType) query.operation_type = params.operationType;
    if (params?.templateScope) query.template_scope = params.templateScope;

    const { data } = await http.get<BackendListResponse<BackendMMLTemplate>>(
      '/mml/templates',
      { params: query }
    );
    return {
      items: (data.items || []).map(mapBackendTemplate),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createTemplate(
    tmpl: Omit<MMLTemplate, 'id' | 'creator' | 'createdAt' | 'updatedAt'>
  ): Promise<MMLTemplate> {
    const payload = {
      template_name: tmpl.templateName,
      command_code: tmpl.commandCode,
      operation_type: tmpl.operationType,
      template_scope: tmpl.templateScope,
      parameters: tmpl.parameters,
      param_paths: tmpl.paramPaths,
      description: tmpl.description,
      product_types: tmpl.productTypes,
    };
    const { data } = await http.post<BackendMMLTemplate>(
      '/mml/templates',
      payload
    );
    return mapBackendTemplate(data);
  },

  async updateTemplate(
    id: string,
    tmpl: Partial<MMLTemplate>
  ): Promise<MMLTemplate> {
    const payload: Record<string, unknown> = {};
    if (tmpl.templateName !== undefined) payload.template_name = tmpl.templateName;
    if (tmpl.commandCode !== undefined) payload.command_code = tmpl.commandCode;
    if (tmpl.operationType !== undefined) payload.operation_type = tmpl.operationType;
    if (tmpl.templateScope !== undefined) payload.template_scope = tmpl.templateScope;
    if (tmpl.parameters !== undefined) payload.parameters = tmpl.parameters;
    if (tmpl.paramPaths !== undefined) payload.param_paths = tmpl.paramPaths;
    if (tmpl.description !== undefined) payload.description = tmpl.description;
    if (tmpl.productTypes !== undefined) payload.product_types = tmpl.productTypes;

    const { data } = await http.put<BackendMMLTemplate>(
      `/mml/templates/${id}`,
      payload
    );
    return mapBackendTemplate(data);
  },

  async deleteTemplate(id: string): Promise<void> {
    await http.delete(`/mml/templates/${id}`);
  },

  async cloneTemplate(id: string): Promise<MMLTemplate> {
    const { data } = await http.post<BackendMMLTemplate>(
      `/mml/templates/${id}/clone`
    );
    return mapBackendTemplate(data);
  },
};
