import http from '../http';
import type { MMLCommand, MMLScript, MMLTask, MMLResult, MMLParam } from '@/types/mml';
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

interface BackendMMLResult {
  success: boolean;
  raw_output: string;
  parsed_data?: Record<string, unknown>;
  execution_time: number;
  timestamp: string;
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
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
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
      pageSize: params.pageSize,
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
    const { data } = await http.get<BackendListResponse<BackendMMLCommand>>(
      '/mml/commands',
      { params: { page: 1, page_size: 1000 } }
    );
    return (data.items || []).map(mapBackendCommand);
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
  ): Promise<Array<{ deviceSn: string; result: MMLResult }>> {
    const payload: Record<string, unknown> = {
      command_code: commandCode,
      device_sns: deviceSns,
    };
    if (params) payload.params = params;

    const { data } = await http.post<BackendMMLTask>('/mml/execute', payload);

    // The execute endpoint creates a task; extract results if available,
    // otherwise return an empty-result per device (task is async).
    if (data.results && data.results.length > 0) {
      return data.results.map(mapBackendResult);
    }
    // Task created but not yet finished — return placeholder results
    return deviceSns.map((sn) => ({
      deviceSn: sn,
      result: {
        success: true,
        rawOutput: '',
        executionTime: 0,
        timestamp: data.created_at || new Date().toISOString(),
      },
    }));
  },

  // --- Scripts ---

  async getScripts(p: PageRequest): Promise<PageResponse<MMLScript>> {
    const query: Record<string, unknown> = {
      page: p.page,
      pageSize: p.pageSize,
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
      pageSize: p.pageSize,
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
    const payload = {
      task_name: data.taskName,
      script_id: data.scriptId,
      device_sns: data.deviceSns,
      commands: data.commands.map((cmd) => ({ command_code: cmd })),
      creator: data.creator,
    };
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
};
