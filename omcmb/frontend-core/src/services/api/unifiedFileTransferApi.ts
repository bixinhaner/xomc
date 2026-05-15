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

type BackendUnifiedFileTransferDeviceItem = Omit<UnifiedFileTransferDeviceItem, 'deviceSn'> & {
  deviceSn?: string;
  deviceSN?: string;
};

function normalizeDeviceItem(item: BackendUnifiedFileTransferDeviceItem): UnifiedFileTransferDeviceItem {
  return {
    ...item,
    deviceSn: item.deviceSn ?? item.deviceSN ?? '',
  };
}

function normalizeDevicePageResponse(
  data: PageResponse<BackendUnifiedFileTransferDeviceItem> | undefined,
  page: number,
  pageSize: number,
): PageResponse<UnifiedFileTransferDeviceItem> {
  return {
    items: (data?.items ?? []).map(normalizeDeviceItem),
    total: data?.total ?? 0,
    page: data?.page ?? page,
    pageSize: data?.pageSize ?? pageSize,
  };
}

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
    const { data } = await http.get<PageResponse<BackendUnifiedFileTransferDeviceItem>>('/ufte/devices', {
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
    return normalizeDevicePageResponse(data, params.page, params.pageSize);
  },

  async getDeviceCandidates(
    params: { keyword?: string; category?: string; typeCode?: string; productType?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    const { data } = await http.get<PageResponse<BackendUnifiedFileTransferDeviceItem>>('/ufte/device-candidates', {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        category: params.category,
        typeCode: params.typeCode,
        productType: params.productType,
        keyword: params.keyword,
      },
    });
    return normalizeDevicePageResponse(data, params.page, params.pageSize);
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