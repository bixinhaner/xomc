import http from '../http';
import type { ConfigTemplate, ConfigParam } from '../../types/config';
import type { PageRequest, PageResponse } from '../../types/pagination';

// Backend config template model
interface BackendConfigTemplate {
  id: string;
  name: string;
  carrier: string;
  technology: string;
  product_class: string;
  template_type: string;
  parameters: unknown; // json.RawMessage — could be ConfigParam[] or raw JSON
  priority: number;
  version: number;
  active: boolean;
  description: string;
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

function parseParameters(raw: unknown): ConfigParam[] {
  if (Array.isArray(raw)) return raw as ConfigParam[];
  if (typeof raw === 'object' && raw !== null) {
    return Object.entries(raw).map(([key, val], idx) => ({
      id: `param-${idx}`,
      paramName: key,
      paramCode: key,
      paramValue: val as string | number | boolean,
      defaultValue: val as string | number | boolean,
      paramType: 'string' as const,
      category: 'general',
      description: '',
      readonly: false,
    }));
  }
  return [];
}

function mapBackendTemplate(bt: BackendConfigTemplate): ConfigTemplate {
  return {
    id: bt.id,
    templateName: bt.name,
    description: bt.description || '',
    params: parseParameters(bt.parameters),
    createTime: bt.created_at,
    creator: '', // backend has no creator field
  };
}

function mapListResponse(
  resp: BackendListResponse<BackendConfigTemplate>
): PageResponse<ConfigTemplate> {
  return {
    items: (resp.items || []).map(mapBackendTemplate),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

export const templateApi = {
  async getTemplates(params: PageRequest): Promise<PageResponse<ConfigTemplate>> {
    const { data } = await http.get<BackendListResponse<BackendConfigTemplate>>(
      '/templates',
      {
        params: {
          page: params.page,
          pageSize: params.pageSize,
          sortField: params.sortField,
          sortOrder: params.sortOrder,
        },
      }
    );
    return mapListResponse(data);
  },

  async getTemplateById(id: string): Promise<ConfigTemplate | null> {
    try {
      const { data } = await http.get<BackendConfigTemplate>(`/templates/${id}`);
      return mapBackendTemplate(data);
    } catch {
      return null;
    }
  },

  async createTemplate(
    data: Omit<ConfigTemplate, 'id' | 'createTime'>
  ): Promise<ConfigTemplate> {
    const { data: bt } = await http.post<BackendConfigTemplate>('/templates', {
      name: data.templateName,
      carrier: 'cmcc',
      technology: 'lte',
      template_type: 'batch_config',
      parameters: data.params,
      description: data.description,
      priority: 0,
    });
    return mapBackendTemplate(bt);
  },

  async updateTemplate(
    id: string,
    data: Partial<ConfigTemplate>
  ): Promise<ConfigTemplate> {
    // Backend PUT requires full object; fetch current then merge
    const current = await templateApi.getTemplateById(id);
    const merged = { ...current, ...data };

    const { data: bt } = await http.put<BackendConfigTemplate>(
      `/templates/${id}`,
      {
        name: merged.templateName,
        carrier: 'cmcc',
        technology: 'lte',
        template_type: 'batch_config',
        parameters: merged.params,
        description: merged.description,
        priority: 0,
      }
    );
    return mapBackendTemplate(bt);
  },

  async deleteTemplates(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/templates/${id}`);
    }
  },

  /**
   * T-0120 / T-0120-b: 显式下发模板到指定设备。
   * 后端 POST /api/v1/templates/:id/dispatch；body { device_ids: string[] }。
   * 响应：{ template_id, dispatched: [...], failed: [...], total_devices }。
   * HTTP 语义：200 全成功 / 207 部分 / 422 全败 / 503 dispatcher 未配置。
   */
  async dispatchTemplate(
    templateId: string,
    deviceIds: string[]
  ): Promise<DispatchTemplateResponse> {
    const { data } = await http.post<BackendDispatchTemplateResponse>(
      `/templates/${templateId}/dispatch`,
      { device_ids: deviceIds }
    );
    return mapDispatchResponse(data);
  },
};

// --- T-0120-b dispatch types ---

interface BackendDispatchResult {
  device_id: string;
  task_id?: string;
  error?: string;
}

interface BackendDispatchTemplateResponse {
  template_id: string;
  dispatched: BackendDispatchResult[];
  failed: BackendDispatchResult[];
  total_devices: number;
}

export interface DispatchResult {
  deviceId: string;
  taskId?: string;
  error?: string;
}

export interface DispatchTemplateResponse {
  templateId: string;
  dispatched: DispatchResult[];
  failed: DispatchResult[];
  totalDevices: number;
}

function mapDispatchResult(r: BackendDispatchResult): DispatchResult {
  return { deviceId: r.device_id, taskId: r.task_id, error: r.error };
}

function mapDispatchResponse(
  resp: BackendDispatchTemplateResponse
): DispatchTemplateResponse {
  return {
    templateId: resp.template_id,
    dispatched: (resp.dispatched || []).map(mapDispatchResult),
    failed: (resp.failed || []).map(mapDispatchResult),
    totalDevices: resp.total_devices,
  };
}
