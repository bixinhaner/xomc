import http from '../http';
import type { PageResponse } from '@/types/pagination';

// Backend model types (snake_case from Go)
interface BackendDeviceRule {
  id: string;
  name: string;
  priority: number;
  target_group_id: string | null;
  target_group_name: string;
  enabled: boolean;
  matching_mode: string;
  name_rule_list: BackendNameRule[] | null;
  lac_list: number[] | null;
  tac_list: number[] | null;
  description: string;
  operators: string;
  created_by: string;
  updated_by: string;
  created_at: string;
  updated_at: string;
}

interface BackendNameRule {
  type: string;
  operator: string;
  value: string;
}

interface BackendRuleTask {
  id: string;
  rule_id: string;
  status: string;
  total_devices: number;
  matched_count: number;
  failed_count: number;
  started_at: string | null;
  completed_at: string | null;
  error_message: string;
  created_by: string;
  created_at: string;
}

// Frontend model types (camelCase)
export interface NameRule {
  type: 'prefix' | 'suffix' | 'contains' | 'regex';
  operator: string;
  value: string;
}

export interface DeviceRule {
  id: string;
  name: string;
  priority: number;
  targetGroupId: string | null;
  targetGroupName: string;
  enabled: boolean;
  matchingMode: 'and' | 'or';
  nameRuleList: NameRule[] | null;
  lacList: number[] | null;
  tacList: number[] | null;
  description: string;
  operators: string;
  createdBy: string;
  updatedBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface RuleTask {
  id: string;
  ruleId: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  totalDevices: number;
  matchedCount: number;
  failedCount: number;
  startedAt: string | null;
  completedAt: string | null;
  errorMessage: string;
  createdBy: string;
  createdAt: string;
}

// Request/Response types
export interface RuleListRequest {
  page?: number;
  pageSize?: number;
  enabled?: boolean;
  name?: string;
}

export interface RuleListResponse extends PageResponse<DeviceRule> {}

export interface RuleTaskListResponse {
  items: RuleTask[];
  total: number;
}

export interface CreateRuleRequest {
  name: string;
  priority: number;
  targetGroupId: string;
  enabled?: boolean;
  matchingMode?: 'and' | 'or';
  nameRuleList?: NameRule[];
  lacList?: number[];
  tacList?: number[];
  description?: string;
}

export interface UpdateRuleRequest {
  name?: string;
  priority?: number;
  targetGroupId?: string;
  enabled?: boolean;
  matchingMode?: 'and' | 'or';
  nameRuleList?: NameRule[];
  lacList?: number[];
  tacList?: number[];
  description?: string;
}

export interface BatchSortItem {
  id: string;
  priority: number;
}

export interface BatchSortRequest {
  items: BatchSortItem[];
}

export interface ApplyRuleRequest {
  dryRun?: boolean;
}

// Mapper functions
function mapBackendNameRule(bnr: BackendNameRule): NameRule {
  return {
    type: bnr.type as NameRule['type'],
    operator: bnr.operator,
    value: bnr.value,
  };
}

function mapBackendRule(br: BackendDeviceRule): DeviceRule {
  return {
    id: br.id,
    name: br.name,
    priority: br.priority,
    targetGroupId: br.target_group_id,
    targetGroupName: br.target_group_name,
    enabled: br.enabled,
    matchingMode: br.matching_mode as DeviceRule['matchingMode'],
    nameRuleList: br.name_rule_list?.map(mapBackendNameRule),
    lacList: br.lac_list,
    tacList: br.tac_list,
    description: br.description || '',
    operators: br.operators || '',
    createdBy: br.created_by || '',
    updatedBy: br.updated_by || '',
    createdAt: br.created_at,
    updatedAt: br.updated_at,
  };
}

function mapBackendTask(bt: BackendRuleTask): RuleTask {
  return {
    id: bt.id,
    ruleId: bt.rule_id,
    status: bt.status as RuleTask['status'],
    totalDevices: bt.total_devices,
    matchedCount: bt.matched_count,
    failedCount: bt.failed_count,
    startedAt: bt.started_at,
    completedAt: bt.completed_at,
    errorMessage: bt.error_message || '',
    createdBy: bt.created_by || '',
    createdAt: bt.created_at,
  };
}

// API Service
const deviceRulesApi = {
  // List rules with pagination
  list: async (params: RuleListRequest): Promise<RuleListResponse> => {
    const response = await http.get<{ items: BackendDeviceRule[]; total: number }>('/device-rules', {
      params,
    });
    return {
      items: response.data.items.map(mapBackendRule),
      total: response.data.total,
      page: params.page || 1,
      pageSize: params.pageSize || 20,
    };
  },

  // Get rule by ID
  get: async (id: string): Promise<DeviceRule> => {
    const response = await http.get<BackendDeviceRule>(`/device-rules/${id}`);
    return mapBackendRule(response.data);
  },

  // Get next available priority
  getNextPriority: async (): Promise<number> => {
    const response = await http.get<{ priority: number }>('/device-rules/next-priority');
    return response.data.priority;
  },

  // Create rule
  create: async (req: CreateRuleRequest): Promise<DeviceRule> => {
    const payload = {
      name: req.name,
      priority: req.priority,
      target_group_id: req.targetGroupId,
      enabled: req.enabled ?? false,
      matching_mode: req.matchingMode || 'and',
      name_rule_list: req.nameRuleList,
      lac_list: req.lacList,
      tac_list: req.tacList,
      description: req.description,
    };
    const response = await http.post<BackendDeviceRule>('/device-rules', payload);
    return mapBackendRule(response.data);
  },

  // Update rule
  update: async (id: string, req: UpdateRuleRequest): Promise<DeviceRule> => {
    const payload: Record<string, unknown> = {};
    if (req.name !== undefined) payload.name = req.name;
    if (req.priority !== undefined) payload.priority = req.priority;
    if (req.targetGroupId !== undefined) payload.target_group_id = req.targetGroupId;
    if (req.enabled !== undefined) payload.enabled = req.enabled;
    if (req.matchingMode !== undefined) payload.matching_mode = req.matchingMode;
    if (req.nameRuleList !== undefined) payload.name_rule_list = req.nameRuleList;
    if (req.lacList !== undefined) payload.lac_list = req.lacList;
    if (req.tacList !== undefined) payload.tac_list = req.tacList;
    if (req.description !== undefined) payload.description = req.description;

    const response = await http.put<BackendDeviceRule>(`/device-rules/${id}`, payload);
    return mapBackendRule(response.data);
  },

  // Delete rule
  delete: async (id: string): Promise<void> => {
    await http.delete(`/device-rules/${id}`);
  },

  // Toggle rule enabled status
  toggle: async (id: string, enabled: boolean): Promise<DeviceRule> => {
    const response = await http.patch<BackendDeviceRule>(`/device-rules/${id}/toggle`, {
      enabled,
    });
    return mapBackendRule(response.data);
  },

  // Batch sort rules
  batchSort: async (items: BatchSortItem[]): Promise<void> => {
    await http.put('/device-rules/batch-sort', {
      items: items.map((item) => ({
        id: item.id,
        priority: item.priority,
      })),
    });
  },

  // Apply rule (creates async task)
  apply: async (id: string, req?: ApplyRuleRequest): Promise<RuleTask> => {
    const response = await http.post<BackendRuleTask>(`/device-rules/${id}/apply`, req || {});
    return mapBackendTask(response.data);
  },

  // Get task details
  getTask: async (ruleId: string, taskId: string): Promise<RuleTask> => {
    const response = await http.get<BackendRuleTask>(
      `/device-rules/${ruleId}/tasks/${taskId}`
    );
    return mapBackendTask(response.data);
  },

  // List tasks for a rule
  listTasks: async (ruleId: string, limit?: number): Promise<RuleTaskListResponse> => {
    const params: Record<string, unknown> = {};
    if (limit !== undefined) {
      params.limit = limit;
    }
    const response = await http.get<{ items: BackendRuleTask[]; total: number }>(
      `/device-rules/${ruleId}/tasks`,
      { params }
    );
    return {
      items: response.data.items.map(mapBackendTask),
      total: response.data.total,
    };
  },
};

export { deviceRulesApi };
