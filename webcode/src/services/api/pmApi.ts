import http from '../http';
import type { KPI, Measurement, KPISeries, PerformanceThreshold } from '@/types/performance';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { performanceService } from '@/mock/services/performanceService';

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

function mapBackendKPIValue(v: BackendKPIValue): Measurement {
  return {
    id: `${v.device_id}-${v.kpi_name}-${v.time}`,
    measurementName: v.kpi_name,
    measurementCode: v.kpi_name,
    deviceSn: '',
    deviceName: '',
    kpiCode: v.kpi_name,
    value: v.kpi_value,
    unit: '',
    timestamp: v.time,
    granularity: '15min',
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
      pageSize: 1000,
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
  async getThresholds(params?: any): Promise<PageResponse<PerformanceThreshold>> {
    const { data } = await http.get('/pm/thresholds', { params });
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
    const { data: result } = await http.post('/pm/thresholds', payload);
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
    const { data: result } = await http.put(`/pm/thresholds/${id}`, payload);
    return mapBackendThreshold(result);
  },

  async deleteThresholds(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/pm/thresholds/${id}`);
    }
  },

  // Aggregated counters & KPI calculation
  async getAggregatedCounters(params?: any): Promise<any> {
    const { data } = await http.get('/pm/counters/aggregated', { params });
    return data;
  },

  async calculateKPI(params: { kpi_name: string; device_ids?: string[]; start_time?: string; end_time?: string }): Promise<any> {
    const { data } = await http.post('/pm/kpi/calculate', params);
    return data;
  },

  // --- Delegated to mock (no backend endpoint) ---
  getTasks: performanceService.getTasks.bind(performanceService),
  createTask: performanceService.createTask.bind(performanceService),
};
