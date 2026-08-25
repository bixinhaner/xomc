import http from '../http';
import type {
  DeviceParameter,
  ParameterTreeNode,
  ParameterSyncStatus,
  ParameterFilter,
  ParameterUpdateRequest,
  ParameterSchemaResponse,
  ParameterUpdateResponse,
  ParameterConstraints,
  ChildParameter,
  DirectChildrenResponse,
  ParameterSyncRequest,
  ParameterSyncRun,
} from '../../types/deviceParameter';
import type { PageRequest, PageResponse } from '../../types/pagination';

// Backend response types (snake_case)
interface BackendDeviceParameter {
  id: string;
  device_id: string;
  parameter_path: string;
  parameter_value: string;
  parameter_type: string;
  writable: boolean;
  last_updated_at: string;
}

interface BackendConstraints {
  min_value?: number;
  max_value?: number;
  enum_values?: string[];
  enum_labels?: string[]; // T-0158
  pattern?: string;
  max_length?: number;
  min_length?: number;
  mirror_with?: string; // T-0159: 交叉镜像目标 standardPath
}

interface BackendParameterTreeNode {
  name: string;
  full_path: string;
  is_object?: boolean;
  is_leaf?: boolean;
  parameter_type?: string;
  type?: string;
  parameter_value?: string;
  value?: string;
  writable?: boolean;
  last_updated_at?: string;
  children?: BackendParameterTreeNode[];
  // Model metadata
  description?: string;
  multi_instance?: boolean;
  max_instances?: number;
  min_instances?: number;
  instance_count?: number;
  can_add?: boolean;
  can_delete?: boolean;
  change_applies?: string;
  default_value?: string;
  constraints?: BackendConstraints;
}

interface BackendSyncStatus {
  device_id?: string;
  status: string;
  total_parameters?: number;
  pending_commands?: number;
  last_sync_gpv?: {
    source_id?: string;
    task_count?: number;
    successful_commands?: number;
    failed_commands?: number;
    requested_path_count?: number;
    successful_path_count?: number;
    failed_path_count?: number;
    failed_paths?: Array<{
      path?: string;
      fault_code?: number;
      fault_text?: string;
      command_key?: string;
      status?: string;
    }>;
    first_created_at?: string;
    last_completed_at?: string;
    wall_clock_seconds?: number;
  };
  last_param_sync_at?: string;
  last_param_sync_failed_at?: string;
  last_param_sync_error?: string;
}

interface BackendParamSyncRun {
  id: string;
  request_id: string;
  status: ParameterSyncRun['status'];
  sync_scope: ParameterSyncRun['syncScope'];
  expected_task_count: number;
  terminal_task_count: number;
  processed_task_count: number;
  failed_task_count: number;
  started_at: string;
  completed_at?: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendChildParameter {
  parameter_path: string;
  parameter_value: string;
  parameter_type: string;
  writable: boolean;
  last_updated_at: string;
  description?: string;
  default_value?: string;
  change_applies?: string;
  constraints?: BackendConstraints;
}

interface BackendSchemaItem {
  path: string;
  type: string;
  writable: boolean;
  description?: string;
  default_value?: string;
  notify?: string;
  forced_inform?: boolean;
  change_applies?: string;
  category?: string;
  is_list?: boolean;
  constraints?: BackendConstraints;
  current_value?: string | null;
  last_synced_at?: string;
}

interface BackendObjectSchemaItem {
  path: string;
  access: string;
  max_instances: number;
  min_instances: number;
  current_instances: number[];
  can_add: boolean;
  can_delete_any: boolean;
  is_list: boolean;
}

interface BackendSchemaResponse {
  parameters: BackendSchemaItem[];
  objects: BackendObjectSchemaItem[];
  total: number;
}

interface BackendUpdateResponse {
  message: string;
  parameters: number;
  reboot_required: boolean;
  reboot_target?: number;
  task_id?: string; // T-0146:后端任务 ID,前端用 useTaskStatus 轮询真实 CPE 应答状态
}

const PARAMETER_SCHEMA_CACHE_TTL_MS = 5 * 60 * 1000;
const parameterSchemaInflight = new Map<string, Promise<ParameterSchemaResponse>>();
const parameterSchemaCache = new Map<string, { expiresAt: number; value: ParameterSchemaResponse }>();

function parameterSchemaCacheKey(deviceId: string, pathPrefix?: string) {
  return `${deviceId}::${pathPrefix ?? ''}`;
}

// Mappers
function mapBackendConstraints(bc: BackendConstraints | undefined): ParameterConstraints | undefined {
  if (!bc) return undefined;
  return {
    minValue: bc.min_value,
    maxValue: bc.max_value,
    enumValues: bc.enum_values,
    enumLabels: bc.enum_labels, // T-0158
    pattern: bc.pattern,
    maxLength: bc.max_length,
    minLength: bc.min_length,
    mirrorWith: bc.mirror_with, // T-0159
  };
}

// 后端 type 字段实际返回多种形态：'STRING' / 'string' / 'BOOLEAN' / 'U_INT' / 'INT' 等。
// 前端 ParameterType union 只接受归一化后的小写 camelCase，validateValue 才能 match 分支。
// 这里统一在 mapper 标准化，确保所有 type 进入应用层都是规范值。
function normalizeParameterType(t: unknown): import('../../types/deviceParameter').ParameterType {
  if (typeof t !== 'string') return 'string';
  const s = t.toUpperCase();
  switch (s) {
    case 'INT':
      return 'int';
    case 'U_INT':
    case 'UINT':
    case 'UNSIGNED_INT':
    case 'UNSIGNEDINT':
      return 'unsignedInt';
    case 'BOOL':
    case 'BOOLEAN':
      return 'boolean';
    case 'DATETIME':
    case 'DATE_TIME':
      return 'dateTime';
    case 'BASE64':
      return 'base64';
    case 'HEXBINARY':
    case 'HEX_BINARY':
      return 'hexBinary';
    case 'OBJECT':
      return 'object';
    default:
      return 'string';
  }
}

function mapBackendParameter(bp: BackendDeviceParameter): DeviceParameter {
  return {
    id: bp.id,
    deviceId: bp.device_id,
    parameterPath: bp.parameter_path,
    parameterValue: bp.parameter_value,
    parameterType: bp.parameter_type as DeviceParameter['parameterType'],
    writable: bp.writable,
    lastUpdatedAt: bp.last_updated_at,
  };
}

function mapBackendTreeNode(bn: BackendParameterTreeNode): ParameterTreeNode {
  // Backend may use is_leaf/value/type or is_object/parameter_value/parameter_type
  const isObject = bn.is_object ?? (bn.is_leaf !== undefined ? !bn.is_leaf : !!(bn.children && bn.children.length > 0));
  const paramType = bn.parameter_type ?? bn.type;
  const paramValue = bn.parameter_value ?? bn.value;

  return {
    name: bn.name,
    fullPath: bn.full_path,
    isObject,
    parameterType: paramType as ParameterTreeNode['parameterType'],
    parameterValue: paramValue,
    writable: bn.writable,
    lastUpdatedAt: bn.last_updated_at,
    children: bn.children?.map(mapBackendTreeNode),
    // Model metadata
    description: bn.description,
    multiInstance: bn.multi_instance,
    maxInstances: bn.max_instances,
    minInstances: bn.min_instances,
    instanceCount: bn.instance_count,
    canAdd: bn.can_add,
    canDelete: bn.can_delete,
    changeApplies: bn.change_applies,
    defaultValue: bn.default_value,
    constraints: mapBackendConstraints(bn.constraints),
  };
}

function mapBackendChildParameter(bp: BackendChildParameter): ChildParameter {
  return {
    parameterPath: bp.parameter_path,
    parameterValue: bp.parameter_value,
    parameterType: bp.parameter_type as ChildParameter['parameterType'],
    writable: bp.writable,
    lastUpdatedAt: bp.last_updated_at,
    description: bp.description,
    defaultValue: bp.default_value,
    changeApplies: bp.change_applies,
    constraints: mapBackendConstraints(bp.constraints),
  };
}

function mapBackendSyncStatus(bs: BackendSyncStatus): ParameterSyncStatus {
  return {
    deviceId: bs.device_id ?? '',
    status: bs.status === 'syncing' ? 'syncing' : 'idle',
    totalParameters: bs.total_parameters ?? 0,
    pendingCommands: bs.pending_commands ?? 0,
    lastSyncGpv: bs.last_sync_gpv ? {
      sourceId: bs.last_sync_gpv.source_id ?? '',
      taskCount: bs.last_sync_gpv.task_count ?? 0,
      successfulCommands: bs.last_sync_gpv.successful_commands ?? 0,
      failedCommands: bs.last_sync_gpv.failed_commands ?? 0,
      requestedPathCount: bs.last_sync_gpv.requested_path_count ?? 0,
      successfulPathCount: bs.last_sync_gpv.successful_path_count ?? 0,
      failedPathCount: bs.last_sync_gpv.failed_path_count ?? 0,
      failedPaths: bs.last_sync_gpv.failed_paths?.map((item) => ({
        path: item.path ?? '',
        faultCode: item.fault_code,
        faultText: item.fault_text,
        commandKey: item.command_key,
        status: item.status,
      })),
      firstCreatedAt: bs.last_sync_gpv.first_created_at,
      lastCompletedAt: bs.last_sync_gpv.last_completed_at,
      wallClockSeconds: bs.last_sync_gpv.wall_clock_seconds,
    } : undefined,
    lastParamSyncAt: bs.last_param_sync_at,
    lastParamSyncFailedAt: bs.last_param_sync_failed_at,
    lastParamSyncError: bs.last_param_sync_error,
  };
}

export const deviceParameterApi = {
  invalidateParameterSchemaCache(deviceId?: string, pathPrefix?: string) {
    if (!deviceId) {
      parameterSchemaInflight.clear();
      parameterSchemaCache.clear();
      return;
    }

    if (pathPrefix !== undefined) {
      const key = parameterSchemaCacheKey(deviceId, pathPrefix);
      parameterSchemaInflight.delete(key);
      parameterSchemaCache.delete(key);
      return;
    }

    const deviceKeyPrefix = `${deviceId}::`;
    for (const key of parameterSchemaInflight.keys()) {
      if (key.startsWith(deviceKeyPrefix)) {
        parameterSchemaInflight.delete(key);
      }
    }
    for (const key of parameterSchemaCache.keys()) {
      if (key.startsWith(deviceKeyPrefix)) {
        parameterSchemaCache.delete(key);
      }
    }
  },

  async getParameters(
    deviceId: string,
    params?: ParameterFilter & PageRequest
  ): Promise<PageResponse<DeviceParameter>> {
    const query: Record<string, unknown> = {
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 50,
      sortField: params?.sortField,
      sortOrder: params?.sortOrder,
    };
    if (params?.search) query.search = params.search;
    if (params?.writable !== undefined) query.writable = params.writable;
    if (params?.parameterType) query.parameterType = params.parameterType;

    const { data } = await http.get<BackendListResponse<BackendDeviceParameter>>(
      `/devices/${deviceId}/parameters`,
      { params: query }
    );
    return {
      items: (data.items || []).map(mapBackendParameter),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async searchParameters(deviceId: string, query: string, limit = 100): Promise<DeviceParameter[]> {
    if (!query.trim()) {
      return [];
    }
    const { data } = await http.get<{ items: BackendDeviceParameter[]; total: number }>(
      `/devices/${deviceId}/parameters/search`,
      { params: { q: query, limit } }
    );
    return (data.items || []).map(mapBackendParameter);
  },

  async getParameterTree(deviceId: string): Promise<ParameterTreeNode[]> {
    const { data } = await http.get<{ tree: BackendParameterTreeNode[]; total: number }>(
      `/devices/${deviceId}/parameters/tree`
    );
    const nodes = data.tree ?? data as unknown as BackendParameterTreeNode[];
    return (Array.isArray(nodes) ? nodes : []).map(mapBackendTreeNode);
  },

  async updateParameters(
    deviceId: string,
    parameters: ParameterUpdateRequest[]
  ): Promise<ParameterUpdateResponse> {
    // 后端 ParameterValueItem JSON 字段是 `path/value/type`(见 device_service.go:374);
    // 前端 ParameterUpdateRequest 字段是 `parameterPath/parameterValue/parameterType`。
    // axios 通用 camelCase→snake_case 会把它们转成 `parameter_path/...`,后端 unmarshal 不上
    // 导致 `item.Path = ""` → MappingValidator.LookupParam("") 必然返 not_found
    // → 全部 PUT 都返 400 "parameter not found in mapping"。
    // 显式映射成后端 JSON 形态,绕开 axios 通用转换。
    const backendParameters = parameters.map((p) => ({
      path: p.parameterPath,
      value: p.parameterValue,
      type: p.parameterType,
    }));
    const { data } = await http.put<BackendUpdateResponse>(
      `/devices/${deviceId}/parameters`,
      { parameters: backendParameters }
    );
    return {
      message: data.message,
      parameters: data.parameters,
      rebootRequired: data.reboot_required,
      rebootTarget: data.reboot_target,
      taskId: data.task_id,
    };
  },

  async resetLMTPassword(deviceId: string, password?: string): Promise<ParameterUpdateResponse> {
    const body = password ? { password } : {};
    const { data } = await http.post<Partial<BackendUpdateResponse>>(
      `/devices/${deviceId}/password/reset`,
      body
    );
    return {
      message: data.message ?? '',
      parameters: data.parameters ?? 1,
      rebootRequired: data.reboot_required ?? false,
      taskId: data.task_id,
    };
  },

  // T-0126: syncParameters (Path A) 已下线，迁移到 deviceApi.syncDeviceParams (Path B + reason="manual")。
  // discoverParameters 保留 — discovery flow 与 Path B 全量同步并存。

  async discoverParameters(deviceId: string): Promise<void> {
    await http.post(`/devices/${deviceId}/parameters/discover`);
  },

  async getSyncStatus(deviceId: string): Promise<ParameterSyncStatus> {
    const { data } = await http.get<BackendSyncStatus>(
      `/devices/${deviceId}/parameters/sync-status`
    );
    const status = mapBackendSyncStatus(data);
    try {
      const { data: activeData } = await http.get<{
        active_run?: BackendParamSyncRun;
        stalled_finalizing?: boolean;
        reason?: string;
      }>(
        `/devices/${deviceId}/parameter-sync/active`
      );
      const run = activeData.active_run;
      if (run) {
        status.status = 'syncing';
        status.pendingCommands = Math.max(run.expected_task_count - run.terminal_task_count, 0);
        status.stalledFinalizing = !!activeData.stalled_finalizing;
        status.stalledReason = activeData.reason;
        status.activeRun = {
          id: run.id,
          requestId: run.request_id,
          status: run.status,
          syncScope: run.sync_scope,
          expectedTaskCount: run.expected_task_count,
          terminalTaskCount: run.terminal_task_count,
          processedTaskCount: run.processed_task_count,
          failedTaskCount: run.failed_task_count,
          startedAt: run.started_at,
          completedAt: run.completed_at,
        };
      }
    } catch {
      // Compatible with a backend that has not enabled the durable data plane yet.
    }
    return status;
  },

  async getParameterSyncRequest(requestId: string): Promise<ParameterSyncRequest> {
    const { data } = await http.get<{
      id: string;
      run_id?: string;
      active_run_id?: string;
      status: ParameterSyncRequest['status'];
      result_code?: string;
      error_message?: string;
      created_at: string;
      completed_at?: string;
    }>(`/parameter-sync/requests/${requestId}`);
    return {
      id: data.id,
      runId: data.run_id,
      activeRunId: data.active_run_id,
      status: data.status,
      resultCode: data.result_code,
      errorMessage: data.error_message,
      createdAt: data.created_at,
      completedAt: data.completed_at,
    };
  },

  async getParameterSchema(
    deviceId: string,
    pathPrefix?: string
  ): Promise<ParameterSchemaResponse> {
    const inflightKey = parameterSchemaCacheKey(deviceId, pathPrefix);
    const cached = parameterSchemaCache.get(inflightKey);
    if (cached && cached.expiresAt > Date.now()) {
      return cached.value;
    }
    if (cached) {
      parameterSchemaCache.delete(inflightKey);
    }

    const existing = parameterSchemaInflight.get(inflightKey);
    if (existing) {
      return existing;
    }

    const params: Record<string, string> = {};
    if (pathPrefix) params.path_prefix = pathPrefix;

    const request = http.get<BackendSchemaResponse>(
      `/devices/${deviceId}/parameters/schema`,
      { params }
    ).then(({ data }) => {
      const mapped = {
        parameters: (data.parameters || []).map((p) => ({
        path: p.path,
        type: normalizeParameterType(p.type),
        writable: p.writable,
        description: p.description,
        defaultValue: p.default_value,
        notify: p.notify,
        forcedInform: p.forced_inform,
        changeApplies: p.change_applies,
        category: p.category,
        isList: p.is_list,
        constraints: mapBackendConstraints(p.constraints),
        currentValue: p.current_value,
        lastSyncedAt: p.last_synced_at,
      })),
        objects: (data.objects || []).map((o) => ({
        path: o.path,
        access: o.access,
        maxInstances: o.max_instances,
        minInstances: o.min_instances,
        currentInstances: o.current_instances || [],
        canAdd: o.can_add,
        canDeleteAny: o.can_delete_any,
        isList: o.is_list,
      })),
        total: data.total,
      };
      parameterSchemaCache.set(inflightKey, {
        expiresAt: Date.now() + PARAMETER_SCHEMA_CACHE_TTL_MS,
        value: mapped,
      });
      return mapped;
    }).finally(() => {
      parameterSchemaInflight.delete(inflightKey);
    });

    parameterSchemaInflight.set(inflightKey, request);
    return request;
  },

  async getObjectTree(deviceId: string): Promise<ParameterTreeNode[]> {
    const { data } = await http.get<{ tree: BackendParameterTreeNode[]; total: number }>(
      `/devices/${deviceId}/parameters/tree`,
      { params: { objects_only: true } }
    );
    const nodes = data.tree ?? data as unknown as BackendParameterTreeNode[];
    return (Array.isArray(nodes) ? nodes : []).map(mapBackendTreeNode);
  },

  async getDirectChildren(
    deviceId: string,
    pathPrefix: string,
    params?: { page?: number; pageSize?: number }
  ): Promise<DirectChildrenResponse> {
    const { data } = await http.get<{
      items: BackendChildParameter[];
      sub_objects: { name: string; full_path: string; child_count: number }[];
      total: number;
      page: number;
      page_size: number;
    }>(
      `/devices/${deviceId}/parameters/children`,
      { params: { path_prefix: pathPrefix, page: params?.page ?? 1, page_size: params?.pageSize ?? 50 } }
    );
    return {
      items: (data.items || []).map(mapBackendChildParameter),
      subObjects: (data.sub_objects || []).map((o) => ({
        name: o.name,
        fullPath: o.full_path,
        childCount: o.child_count,
      })),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // T-0157 C7: 后端改为返回 task_id，前端透传供 useDeviceTaskStatus 轮询状态机使用
  async addObject(deviceId: string, objectPath: string): Promise<{ taskId: string; rebootRequired?: boolean; rebootTarget?: number }> {
    const { data } = await http.post<{ task_id: string; message: string; reboot_required?: boolean; reboot_target?: number }>(
      `/devices/${deviceId}/objects/add`,
      { object_path: objectPath },
    );
    return { taskId: data.task_id, rebootRequired: data.reboot_required, rebootTarget: data.reboot_target };
  },

  async deleteObject(deviceId: string, objectPath: string): Promise<{ taskId: string; rebootRequired?: boolean; rebootTarget?: number }> {
    const { data } = await http.post<{ task_id: string; message: string; reboot_required?: boolean; reboot_target?: number }>(
      `/devices/${deviceId}/objects/delete`,
      { object_path: objectPath },
    );
    return { taskId: data.task_id, rebootRequired: data.reboot_required, rebootTarget: data.reboot_target };
  },

  // 同步配置文件 - 创建 filetype=11 的 Upload RPC 任务
  async syncConfigFile(deviceId: string): Promise<{ message: string; command_id: string; file_type: string }> {
    const { data } = await http.post<{ message: string; command_id: string; file_type: string }>(
      `/devices/${deviceId}/config-file/sync`
    );
    return data;
  },
};
