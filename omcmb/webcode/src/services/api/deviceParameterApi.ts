import http from '../http';
import type {
  DeviceParameter,
  ParameterTreeNode,
  ParameterSyncStatus,
  ParameterFilter,
  ParameterUpdateRequest,
  ParameterSyncOptions,
  ParameterSchemaResponse,
  ParameterUpdateResponse,
  ParameterConstraints,
} from '@/types/deviceParameter';
import type { PageRequest, PageResponse } from '@/types/pagination';

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
  pattern?: string;
  max_length?: number;
  min_length?: number;
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
  device_id: string;
  status: string;
  total_batches: number;
  completed_batches: number;
  total_parameters: number;
  synced_parameters: number;
  percentage: number;
  started_at?: string;
  completed_at?: string;
  error?: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
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
}

// Mappers
function mapBackendConstraints(bc: BackendConstraints | undefined): ParameterConstraints | undefined {
  if (!bc) return undefined;
  return {
    minValue: bc.min_value,
    maxValue: bc.max_value,
    enumValues: bc.enum_values,
    pattern: bc.pattern,
    maxLength: bc.max_length,
    minLength: bc.min_length,
  };
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

function mapBackendSyncStatus(bs: BackendSyncStatus): ParameterSyncStatus {
  return {
    deviceId: bs.device_id,
    status: bs.status as ParameterSyncStatus['status'],
    totalBatches: bs.total_batches,
    completedBatches: bs.completed_batches,
    totalParameters: bs.total_parameters,
    syncedParameters: bs.synced_parameters,
    percentage: bs.percentage,
    startedAt: bs.started_at,
    completedAt: bs.completed_at,
    error: bs.error,
  };
}

export const deviceParameterApi = {
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

  async getParameterTree(deviceId: string): Promise<ParameterTreeNode[]> {
    const { data } = await http.get<BackendParameterTreeNode[]>(
      `/devices/${deviceId}/parameters/tree`
    );
    return (data || []).map(mapBackendTreeNode);
  },

  async updateParameters(
    deviceId: string,
    parameters: ParameterUpdateRequest[]
  ): Promise<ParameterUpdateResponse> {
    const { data } = await http.put<BackendUpdateResponse>(
      `/devices/${deviceId}/parameters`,
      { parameters }
    );
    return {
      message: data.message,
      parameters: data.parameters,
      rebootRequired: data.reboot_required,
    };
  },

  async syncParameters(
    deviceId: string,
    options?: ParameterSyncOptions
  ): Promise<void> {
    await http.post(`/devices/${deviceId}/parameters/sync`, options ?? {});
  },

  async discoverParameters(deviceId: string): Promise<void> {
    await http.post(`/devices/${deviceId}/parameters/discover`);
  },

  async getSyncStatus(deviceId: string): Promise<ParameterSyncStatus> {
    const { data } = await http.get<BackendSyncStatus>(
      `/devices/${deviceId}/parameters/sync-status`
    );
    return mapBackendSyncStatus(data);
  },

  async getParameterSchema(
    deviceId: string,
    pathPrefix?: string
  ): Promise<ParameterSchemaResponse> {
    const params: Record<string, string> = {};
    if (pathPrefix) params.path_prefix = pathPrefix;

    const { data } = await http.get<BackendSchemaResponse>(
      `/devices/${deviceId}/parameters/schema`,
      { params }
    );
    return {
      parameters: (data.parameters || []).map((p) => ({
        path: p.path,
        type: p.type,
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
  },

  async addObject(deviceId: string, objectPath: string): Promise<void> {
    await http.post(`/devices/${deviceId}/objects/add`, {
      object_path: objectPath,
    });
  },

  async deleteObject(deviceId: string, objectPath: string): Promise<void> {
    await http.post(`/devices/${deviceId}/objects/delete`, {
      object_path: objectPath,
    });
  },
};
