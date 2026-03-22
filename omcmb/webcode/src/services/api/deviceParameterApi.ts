import http from '../http';
import type {
  DeviceParameter,
  ParameterTreeNode,
  ParameterSyncStatus,
  ParameterFilter,
  ParameterUpdateRequest,
  ParameterSyncOptions,
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

interface BackendParameterTreeNode {
  name: string;
  full_path: string;
  is_object: boolean;
  parameter_type?: string;
  parameter_value?: string;
  writable?: boolean;
  last_updated_at?: string;
  children?: BackendParameterTreeNode[];
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

// Mappers
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
  return {
    name: bn.name,
    fullPath: bn.full_path,
    isObject: bn.is_object,
    parameterType: bn.parameter_type as ParameterTreeNode['parameterType'],
    parameterValue: bn.parameter_value,
    writable: bn.writable,
    lastUpdatedAt: bn.last_updated_at,
    children: bn.children?.map(mapBackendTreeNode),
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
  ): Promise<void> {
    await http.put(`/devices/${deviceId}/parameters`, { parameters });
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
};
