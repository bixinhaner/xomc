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

  /**
   * 导出设备列表 CSV（流式，复用 ListDevices 的过滤逻辑，不分页拿全量）。
   * 后端直接写 text/csv 流；axios 用 responseType=blob 接收。
   */
  async exportDevices(params: {
    status?: string;
    typeCode?: string;
    keyword?: string;
    category?: string;
    productType?: string;
    /** #524：按产品名筛选（与列表口径一致），后端 matchesDeviceFilter 按 ProductName 过滤 */
    productName?: string;
    /** "upgrade" = 4G/5G 升级页签列名；其它（默认） = 通用页签列名 */
    view?: 'upgrade' | 'default';
  }): Promise<{ blob: Blob; filename: string }> {
    const resp = await http.get<Blob>('/ufte/devices/export', {
      params: {
        status: params.status,
        typeCode: params.typeCode,
        category: params.category,
        productType: params.productType,
        productName: params.productName,
        keyword: params.keyword,
        view: params.view,
      },
      responseType: 'blob',
    });
    // 优先用 Content-Disposition 里的服务端命名
    const cd = (resp.headers['content-disposition'] || resp.headers['Content-Disposition']) as string | undefined;
    let filename = 'ufte-devices.csv';
    if (cd) {
      const m = /filename=([^;]+)/i.exec(cd);
      if (m) filename = m[1].trim().replace(/^"|"$/g, '');
    }
    return { blob: resp.data, filename };
  },

  async getDevices(
    params: { status?: string; typeCode?: string; keyword?: string; category?: string; productType?: string; productName?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    const { data } = await http.get<PageResponse<BackendUnifiedFileTransferDeviceItem>>('/ufte/devices', {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        status: params.status,
        typeCode: params.typeCode,
        category: params.category,
        productType: params.productType,
        // #524：设备列表「产品名称」筛选改传产品名（后端 matchesDeviceFilter 按 ProductName 过滤）。
        productName: params.productName,
        keyword: params.keyword,
      },
    });
    return normalizeDevicePageResponse(data, params.page, params.pageSize);
  },

  async getDeviceCandidates(
    params: { keyword?: string; category?: string; typeCode?: string; productType?: string; productName?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    const { data } = await http.get<PageResponse<BackendUnifiedFileTransferDeviceItem>>('/ufte/device-candidates', {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        category: params.category,
        typeCode: params.typeCode,
        productType: params.productType,
        // #492：按产品英文名收窄候选（升级抽屉选产品名后传此参数）。
        productName: params.productName,
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

  async deleteTaskType(typeCode: string): Promise<void> {
	await http.delete(`/ufte/task-types/${typeCode}`);
  },

  async startTask(id: string): Promise<void> {
    await http.put(`/ufte/tasks/${id}/start`);
  },

  async suspendTask(id: string): Promise<void> {
    await http.put(`/ufte/tasks/${id}/suspend`);
  },

  async terminateTask(id: string): Promise<void> {
    await http.put(`/ufte/tasks/${id}/terminate`);
  },

  async deleteTask(id: string): Promise<void> {
    await http.delete(`/ufte/tasks/${id}`);
  },

  /**
   * 批量删除任务。单条失败不影响其他；返回 succeeded + failed 列表。
   * 走 POST /tasks/batch-delete 而不是 DELETE+body（DELETE 携带 body 在某些
   * 代理 / WAF 下会被吞，与 backup snapshot/license 批删同款约定）。
   */
  async batchDeleteTasks(taskIds: string[]): Promise<{
    succeeded: string[];
    failed: Array<{ taskId: string; error: string }>;
  }> {
    const { data } = await http.post<{
      succeeded: string[];
      failed: Array<{ task_id: string; error: string }>;
    }>('/ufte/tasks/batch-delete', { task_ids: taskIds });
    return {
      succeeded: data.succeeded ?? [],
      failed: (data.failed ?? []).map((f) => ({ taskId: f.task_id, error: f.error })),
    };
  },

  async retryTask(id: string): Promise<void> {
    await http.post(`/ufte/tasks/${id}/retry`);
  },
};