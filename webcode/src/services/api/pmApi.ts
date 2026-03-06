import http from '../http';
import type { KPI, Measurement, KPISeries } from '@/types/performance';
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

  // --- Delegated to mock (no backend endpoint) ---

  getCounters: performanceService.getCounters.bind(performanceService),
  getThresholds: performanceService.getThresholds.bind(performanceService),
  createThreshold: performanceService.createThreshold.bind(performanceService),
  updateThreshold: performanceService.updateThreshold.bind(performanceService),
  deleteThresholds: performanceService.deleteThresholds.bind(performanceService),
  getTasks: performanceService.getTasks.bind(performanceService),
  createTask: performanceService.createTask.bind(performanceService),
};
