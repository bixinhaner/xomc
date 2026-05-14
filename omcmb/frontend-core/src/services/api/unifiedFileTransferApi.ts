import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  CreateUnifiedFileTransferTaskInput,
  CreateUnifiedFileTransferTypeInput,
  UpdateUnifiedFileTransferTaskTypeInput,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferOverview,
  UnifiedFileTransferTask,
  UnifiedFileTransferTaskType,
} from '../../types/unifiedFileTransfer';

export const unifiedFileTransferApi = {
  async getOverview(): Promise<UnifiedFileTransferOverview> {
    const { data } = await http.get<UnifiedFileTransferOverview>('/ufte/overview');
    return data;
  },

  async getTaskTypes(): Promise<UnifiedFileTransferTaskType[]> {
    const { data } = await http.get<UnifiedFileTransferTaskType[]>('/ufte/task-types');
    return data ?? [];
  },

  async getTasks(
    params: { status?: string; typeCode?: string; keyword?: string; category?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferTask>> {
    const { data } = await http.get<PageResponse<UnifiedFileTransferTask>>('/ufte/tasks', {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        status: params.status,
        typeCode: params.typeCode,
        category: params.category,
        keyword: params.keyword,
      },
    });
    return data;
  },

  async getDevices(
    params: { status?: string; typeCode?: string; keyword?: string; category?: string; productType?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    const { data } = await http.get<PageResponse<UnifiedFileTransferDeviceItem>>('/ufte/devices', {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        status: params.status,
        typeCode: params.typeCode,
        category: params.category,
        productType: params.productType,
        keyword: params.keyword,
      },
    });
    return data;
  },

  async getDeviceCandidates(
    params: { keyword?: string; category?: string; productType?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    const { data } = await http.get<PageResponse<UnifiedFileTransferDeviceItem>>('/ufte/device-candidates', {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        category: params.category,
        productType: params.productType,
        keyword: params.keyword,
      },
    });
    return data;
  },

  async createTask(input: CreateUnifiedFileTransferTaskInput): Promise<UnifiedFileTransferTask> {
    const { data } = await http.post<UnifiedFileTransferTask>('/ufte/tasks', input);
    return data;
  },

  async createTaskType(input: CreateUnifiedFileTransferTypeInput): Promise<UnifiedFileTransferTaskType> {
    const { data } = await http.post<UnifiedFileTransferTaskType>('/ufte/task-types', input);
    return data;
  },

  async updateTaskType(input: UpdateUnifiedFileTransferTaskTypeInput): Promise<UnifiedFileTransferTaskType> {
    const { data } = await http.put<UnifiedFileTransferTaskType>(`/ufte/task-types/${input.typeCode}`, input);
    return data;
  },
};