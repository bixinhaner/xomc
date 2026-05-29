import http from '../http';
import type { KPI, Measurement, KPISeries, PerformanceThreshold, PerformanceTask, AggregatedCounter, AggregatedCounterQuery, KPICalculationRequest, KPICalculationResult } from '../../types/performance';
import type { PageRequest, PageResponse } from '../../types/pagination';

// --- Backend response types ---

interface BackendKPIDefinition {
  name: string;
  display_name: string;
  formula: string;
  unit: string;
  carrier: string;
  technology: string;
  counters: string[];
}

interface BackendKPIValue {
  time: string;
  device_id: string;
  cell_id: string;
  kpi_name: string;
  kpi_value: number;
  carrier: string;
  technology: string;
}

interface BackendPMCounter {
  time: string;
  device_id: string;
  cell_id: string;
  counter_group: string;
  counter_name: string;
  counter_value: number;
  granularity: number;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendDefinitionsResponse {
  items: BackendKPIDefinition[];
  total: number;
}

// --- Mapping functions ---

function mapBackendKPIDefinition(d: BackendKPIDefinition): KPI {
  return {
    id: d.name,
    kpiName: d.display_name,
    kpiCode: d.name,
    unit: d.unit,
    description: d.formula,
    category: '',
  };
}

function granularityLabel(minutes: number): '15min' | '30min' | '1h' | '1d' {
  if (minutes <= 15) return '15min';
  if (minutes <= 30) return '30min';
  if (minutes <= 60) return '1h';
  return '1d';
}

function mapBackendCounter(c: BackendPMCounter): Measurement {
  return {
    id: `${c.device_id}-${c.counter_name}-${c.time}`,
    measurementName: c.counter_name,
    measurementCode: c.counter_name,
    deviceSn: '',
    deviceName: '',
    kpiCode: c.counter_name,
    value: c.counter_value,
    unit: '',
    timestamp: c.time,
    granularity: granularityLabel(c.granularity),
  };
}

// --- Backend threshold types & mapping ---

interface BackendKPIThreshold {
  id: string;
  kpi_name: string;
  carrier: string;
  technology: string;
  warning_threshold: number | null;
  minor_threshold: number | null;
  major_threshold: number | null;
  critical_threshold: number | null;
  comparison: string;
  enabled: boolean;
  description: string;
  created_at: string;
  updated_at: string;
}

function mapComparisonToOperator(comp: string): PerformanceThreshold['operator'] {
  const map: Record<string, PerformanceThreshold['operator']> = {
    '>': 'gt', '<': 'lt', '>=': 'gte', '<=': 'lte', '=': 'eq', '!=': 'ne',
    'gt': 'gt', 'lt': 'lt', 'gte': 'gte', 'lte': 'lte', 'eq': 'eq', 'ne': 'ne',
  };
  return map[comp] || 'gt';
}

function mapOperatorToComparison(op: string): string {
  const map: Record<string, string> = {
    'gt': '>', 'lt': '<', 'gte': '>=', 'lte': '<=', 'eq': '=', 'ne': '!=',
  };
  return map[op] || '>';
}

function mapBackendThreshold(t: BackendKPIThreshold): PerformanceThreshold {
  return {
    id: t.id,
    thresholdName: t.description || t.kpi_name,
    kpiCode: t.kpi_name,
    kpiName: t.kpi_name,
    operator: mapComparisonToOperator(t.comparison),
    warningValue: t.warning_threshold ?? 0,
    criticalValue: t.critical_threshold ?? 0,
    unit: '',
    enabled: t.enabled,
    deviceGroups: [],
    createTime: t.created_at || '',
    updateTime: t.updated_at || '',
  };
}

// --- Backend PM Task types & mapping ---

interface BackendPMTask {
  id: string;
  task_name: string;
  task_type: string;
  device_sns: string[];
  kpi_codes: string[];
  granularity: string;
  time_range: [string, string] | null;
  status: string;
  progress: number;
  creator: string;
  created_at: string;
  updated_at: string;
}

function mapBackendPMTask(b: BackendPMTask) {
  return {
    id: b.id,
    taskName: b.task_name,
    taskType: b.task_type,
    deviceSns: b.device_sns || [],
    kpiCodes: b.kpi_codes || [],
    granularity: b.granularity,
    timeRange: b.time_range,
    status: b.status,
    progress: b.progress,
    creator: b.creator,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapToBackendPMTask(t: Partial<PerformanceTask>) {
  return {
    task_name: t.taskName,
    task_type: t.taskType,
    device_sns: t.deviceSns,
    kpi_codes: t.kpiCodes,
    granularity: t.granularity,
    time_range: t.timeRange,
  };
}

// --- Exported service ---

export const pmApi = {
  // KPI definitions → KPI list
  async getKPIs(
    params: { keyword?: string } & PageRequest
  ): Promise<PageResponse<KPI>> {
    const query: Record<string, unknown> = {};
    if (params.keyword) query.carrier = params.keyword;

    const { data } = await http.get<BackendDefinitionsResponse>(
      '/pm/kpi/definitions',
      { params: query }
    );

    const allItems = (data.items || []).map(mapBackendKPIDefinition);

    // Client-side pagination (definitions endpoint returns all)
    const page = params.page || 1;
    const pageSize = params.pageSize || 20;
    let filtered = allItems;
    if (params.keyword) {
      const kw = params.keyword.toLowerCase();
      filtered = allItems.filter(
        (k) =>
          k.kpiName.toLowerCase().includes(kw) ||
          k.kpiCode.toLowerCase().includes(kw)
      );
    }
    const start = (page - 1) * pageSize;
    return {
      items: filtered.slice(start, start + pageSize),
      total: filtered.length,
      page,
      pageSize,
    };
  },

  async getAllKPIs(): Promise<KPI[]> {
    const { data } = await http.get<BackendDefinitionsResponse>(
      '/pm/kpi/definitions'
    );
    return (data.items || []).map(mapBackendKPIDefinition);
  },

  // PM counter records → Measurement list
  async getMeasurements(
    params: {
      deviceSn?: string;
      kpiCode?: string;
      granularity?: string;
      timeRange?: [string, string];
    } & PageRequest
  ): Promise<PageResponse<Measurement>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.kpiCode) query.counter_name = params.kpiCode;
    if (params.timeRange) {
      query.start_time = params.timeRange[0];
      query.end_time = params.timeRange[1];
    }

    const { data } = await http.get<BackendListResponse<BackendPMCounter>>(
      '/pm/counters',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendCounter),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // KPI time series for a single KPI
  async getKPISeries(
    kpiCode: string,
    _deviceSn?: string,
    days = 7
  ): Promise<KPISeries> {
    const now = new Date();
    const start = new Date(now.getTime() - days * 24 * 60 * 60 * 1000);
    const query: Record<string, unknown> = {
      kpi_name: kpiCode,
      start_time: start.toISOString(),
      end_time: now.toISOString(),
      page: 1,
      pageSize: 100,
    };

    const { data } = await http.get<BackendListResponse<BackendKPIValue>>(
      '/pm/kpi',
      { params: query }
    );

    const values = (data.items || [])
      .filter((v) => v.kpi_name === kpiCode)
      .sort((a, b) => a.time.localeCompare(b.time));

    return {
      kpiName: kpiCode,
      unit: '',
      data: values.map((v) => ({ timestamp: v.time, value: v.kpi_value })),
    };
  },

  // Multiple KPI series
  async getMultipleKPISeries(
    kpiCodes: string[],
    deviceSn?: string
  ): Promise<KPISeries[]> {
    const results = await Promise.all(
      kpiCodes.map((code) => pmApi.getKPISeries(code, deviceSn))
    );
    return results;
  },

  // PM counters
  async getCounters(params: PageRequest): Promise<PageResponse<Measurement>> {
    const { data } = await http.get<BackendListResponse<BackendPMCounter>>(
      '/pm/counters',
      { params }
    );
    return {
      items: (data.items || []).map(mapBackendCounter),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // Threshold CRUD
  async getThresholds(params?: PageRequest): Promise<PageResponse<PerformanceThreshold>> {
    const { data } = await http.get<BackendListResponse<BackendKPIThreshold>>('/pm/thresholds', { params });
    const items = (data.items || []).map((t: BackendKPIThreshold) => mapBackendThreshold(t));
    return {
      items,
      total: data.total || items.length,
      page: data.page || 1,
      pageSize: data.page_size || 20,
    };
  },

  async createThreshold(data: Partial<PerformanceThreshold>): Promise<PerformanceThreshold> {
    const payload = {
      kpi_name: data.kpiCode || data.kpiName,
      warning_threshold: data.warningValue,
      critical_threshold: data.criticalValue,
      comparison: mapOperatorToComparison(data.operator || 'gt'),
      enabled: data.enabled ?? true,
      description: data.thresholdName || '',
    };
    const { data: result } = await http.post<BackendKPIThreshold>('/pm/thresholds', payload);
    return mapBackendThreshold(result);
  },

  async updateThreshold(id: string, data: Partial<PerformanceThreshold>): Promise<PerformanceThreshold> {
    const payload: Record<string, unknown> = {};
    if (data.kpiCode !== undefined || data.kpiName !== undefined) payload.kpi_name = data.kpiCode || data.kpiName;
    if (data.warningValue !== undefined) payload.warning_threshold = data.warningValue;
    if (data.criticalValue !== undefined) payload.critical_threshold = data.criticalValue;
    if (data.operator !== undefined) payload.comparison = mapOperatorToComparison(data.operator);
    if (data.enabled !== undefined) payload.enabled = data.enabled;
    if (data.thresholdName !== undefined) payload.description = data.thresholdName;
    const { data: result } = await http.put<BackendKPIThreshold>(`/pm/thresholds/${id}`, payload);
    return mapBackendThreshold(result);
  },

  async deleteThresholds(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/pm/thresholds/${id}`);
    }
  },

  // Aggregated counters & KPI calculation
  async getAggregatedCounters(params?: AggregatedCounterQuery): Promise<{ items: AggregatedCounter[] }> {
    const { data } = await http.get<{ items: AggregatedCounter[] }>('/pm/counters/aggregated', { params });
    return data;
  },

  async calculateKPI(params: KPICalculationRequest): Promise<{ items: KPICalculationResult[]; total: number }> {
    const { data } = await http.post<{ items: KPICalculationResult[]; total: number }>('/pm/kpi/calculate', params);
    return data;
  },

  // PM Tasks
  async getTasks(params: PageRequest): Promise<PageResponse<ReturnType<typeof mapBackendPMTask>>> {
    const { data } = await http.get<BackendListResponse<BackendPMTask>>('/pm/tasks', { params });
    return { items: (data.items || []).map(mapBackendPMTask), total: data.total, page: data.page, pageSize: data.page_size };
  },
  async createTask(taskData: Partial<PerformanceTask>): Promise<ReturnType<typeof mapBackendPMTask>> {
    const { data } = await http.post<BackendPMTask>('/pm/tasks', mapToBackendPMTask(taskData));
    return mapBackendPMTask(data);
  },

  // --- PM Files (File Management → PM Tab) ---

  // 单个设备的 PM 文件列表（DeviceFilesDrawer 用）
  async getFiles(
    params: {
      deviceSn?: string;
      timeRange?: [string, string];
    } & PageRequest,
  ): Promise<PageResponse<PMFileItem>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.timeRange) {
      query.start_time = params.timeRange[0];
      query.end_time = params.timeRange[1];
    }
    const { data } = await http.get<BackendListResponse<BackendPMFileInfo>>(
      '/pm/files',
      { params: query },
    );
    return {
      items: (data.items || []).map(mapPMFile),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // 按设备聚合（File Management → PM Tab 主列表）
  async getFileDevices(
    params: { keyword?: string; siteName?: string; productClass?: string } & PageRequest,
  ): Promise<PageResponse<PMFileDeviceItem>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.keyword) query.keyword = params.keyword;
    if (params.siteName) query.site_name = params.siteName;
    if (params.productClass) query.product_class = params.productClass;
    const { data } = await http.get<BackendListResponse<BackendPMFileDevice>>(
      '/pm/files/devices',
      { params: query },
    );
    return {
      items: (data.items || []).map((d) => ({
        deviceSn: d.device_sn,
        siteName: d.site_name || '',
        productClass: d.product_class || '',
        firstCollectTime: d.first_collect_time,
        lastCollectTime: d.last_collect_time,
        fileCount: d.file_count,
        reporting: !!d.reporting,
      })),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // 按设备 SN 批量删除（PG + MinIO），后端逐 SN 处理，返回成功/失败列表。
  async batchDeleteFiles(serialNumbers: string[]): Promise<{ succeeded: string[]; failed: string[] }> {
    const { data } = await http.post<{ succeeded: string[]; failed: string[] }>(
      '/pm/files/batch-delete',
      { serial_numbers: serialNumbers },
    );
    return { succeeded: data.succeeded || [], failed: data.failed || [] };
  },

  async downloadFile(fileId: string): Promise<void> {
    const response = await http.get(`/pm/files/${fileId}/download`, {
      responseType: 'blob',
    });
    const disposition = response.headers['content-disposition'] as string | undefined;
    const filename = disposition
      ? disposition.split('filename=')[1]?.replace(/['"]/g, '')
      : `pm_${fileId}.xml`;
    // Blob 必须显式带 type，否则 Chrome 会按 text/html 推断给 link.download 加 .html 后缀。
    const url = window.URL.createObjectURL(
      new Blob([response.data as BlobPart], { type: 'application/octet-stream' }),
    );
    const link = document.createElement('a');
    link.href = url;
    link.download = filename || `pm_${fileId}.xml`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  },
};

// --- PM file types ---

interface BackendPMFileInfo {
  id: string;
  device_id: string;
  device_sn: string;
  carrier: string;
  technology: string;
  file_name: string;
  file_size: number;
  collect_time: string;
  minio_path: string;
  parsed: boolean;
  parsed_at?: string;
  counter_count: number;
  created_at: string;
}

interface BackendPMFileDevice {
  device_sn: string;
  site_name: string;
  product_class: string;
  first_collect_time: string;
  last_collect_time: string;
  file_count: number;
  reporting: boolean;
}

export interface PMFileItem {
  id: string;
  deviceSn: string;
  carrier: string;
  technology: string;
  fileName: string;
  fileSize: number;
  collectTime: string;
  parsed: boolean;
  counterCount: number;
  createdAt: string;
}

/** PM 文件按设备聚合视图（GET /pm/files/devices）。 */
export interface PMFileDeviceItem {
  deviceSn: string;
  /** 基站名称（来自 devices.site_name，允许空） */
  siteName: string;
  /** 产品类（来自 devices.product_class，允许空） */
  productClass: string;
  firstCollectTime: string;
  lastCollectTime: string;
  fileCount: number;
  /** 最近 2 小时内有新 PM 文件视为"上报中"，由后端用 last_collect_time 判定。 */
  reporting: boolean;
}

function mapPMFile(f: BackendPMFileInfo): PMFileItem {
  return {
    id: f.id,
    deviceSn: f.device_sn,
    carrier: f.carrier,
    technology: f.technology,
    fileName: f.file_name,
    fileSize: f.file_size,
    collectTime: f.collect_time,
    parsed: f.parsed,
    counterCount: f.counter_count,
    createdAt: f.created_at,
  };
}
