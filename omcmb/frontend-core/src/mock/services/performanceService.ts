import type { KPI, Counter, Measurement, PerformanceTask, PerformanceThreshold, KPISeries } from '../../types/performance';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mockKPIs, mockCounters, mockMeasurements, mockThresholds, mockKPISeries } from '../data/performance';
import { delay, paginate, generateId, generateTimeSeries } from '../utils';

let thresholds = [...mockThresholds];

export const performanceService = {
  async getKPIs(params: { keyword?: string } & PageRequest): Promise<PageResponse<KPI>> {
    await delay(80, 150);
    let filtered = [...mockKPIs];
    if (params.keyword) {
      filtered = filtered.filter(
        (k) => k.kpiName.includes(params.keyword!) || k.kpiCode.includes(params.keyword!)
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getAllKPIs(): Promise<KPI[]> {
    await delay(80, 150);
    return mockKPIs;
  },

  async getCounters(params: PageRequest): Promise<PageResponse<Counter>> {
    await delay(80, 150);
    return paginate(mockCounters, params.page, params.pageSize);
  },

  async getMeasurements(
    params: { deviceSn?: string; kpiCode?: string; granularity?: string; timeRange?: [string, string] } & PageRequest
  ): Promise<PageResponse<Measurement>> {
    await delay(100, 250);
    let filtered = [...mockMeasurements];
    if (params.deviceSn) filtered = filtered.filter((m) => m.deviceSn === params.deviceSn);
    if (params.kpiCode) filtered = filtered.filter((m) => m.kpiCode === params.kpiCode);
    if (params.granularity) filtered = filtered.filter((m) => m.granularity === params.granularity);
    if (params.timeRange) {
      const [start, end] = params.timeRange;
      filtered = filtered.filter((m) => m.timestamp >= start && m.timestamp <= end);
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getKPISeries(kpiCode: string, deviceSn?: string, days = 7): Promise<KPISeries> {
    await delay(150, 300);
    const kpi = mockKPIs.find((k) => k.kpiCode === kpiCode);
    const existing = mockKPISeries.find((s) => {
      const k = mockKPIs.find((kk) => kk.kpiName === s.kpiName);
      return k?.kpiCode === kpiCode;
    });

    void deviceSn;

    if (existing) return existing;

    const rawSeries = generateTimeSeries(days, 60, 95, 5);
    return {
      kpiName: kpi?.kpiName ?? kpiCode,
      unit: kpi?.unit ?? '',
      data: rawSeries.map(([timestamp, value]) => ({ timestamp, value })),
    };
  },

  async getMultipleKPISeries(kpiCodes: string[], deviceSn?: string): Promise<KPISeries[]> {
    await delay(200, 400);
    return Promise.all(kpiCodes.map((code) => performanceService.getKPISeries(code, deviceSn)));
  },

  async getThresholds(params: PageRequest): Promise<PageResponse<PerformanceThreshold>> {
    await delay(80, 150);
    return paginate(thresholds, params.page, params.pageSize);
  },

  async createThreshold(data: Omit<PerformanceThreshold, 'id' | 'createTime' | 'updateTime'>): Promise<PerformanceThreshold> {
    await delay(200, 400);
    const newItem: PerformanceThreshold = {
      ...data,
      id: generateId('thr'),
      createTime: new Date().toISOString(),
      updateTime: new Date().toISOString(),
    };
    thresholds.push(newItem);
    return newItem;
  },

  async updateThreshold(id: string, data: Partial<PerformanceThreshold>): Promise<PerformanceThreshold> {
    await delay(150, 300);
    const idx = thresholds.findIndex((t) => t.id === id);
    if (idx === -1) throw new Error(`Threshold ${id} not found`);
    thresholds[idx] = { ...thresholds[idx], ...data, updateTime: new Date().toISOString() };
    return thresholds[idx];
  },

  async deleteThresholds(ids: string[]): Promise<void> {
    await delay(150, 300);
    thresholds = thresholds.filter((t) => !ids.includes(t.id));
  },

  async getTasks(params: PageRequest): Promise<PageResponse<PerformanceTask>> {
    await delay(100, 200);
    const tasks: PerformanceTask[] = [
      {
        id: 'ptask-001',
        taskName: '全网KPI提取-日报',
        taskType: 'extraction',
        deviceSns: ['ENB00001', 'ENB00002', 'GNB00001'],
        kpiCodes: ['RRC_SUCC_RATE', 'ERAB_SUCC_RATE'],
        granularity: '1h',
        timeRange: [
          new Date(Date.now() - 86400000).toISOString(),
          new Date().toISOString(),
        ],
        status: 'success',
        progress: 100,
        createdAt: new Date(Date.now() - 3600000).toISOString(),
        updatedAt: new Date(Date.now() - 1800000).toISOString(),
        creator: 'admin',
      },
      {
        id: 'ptask-002',
        taskName: '门限检查-华北区',
        taskType: 'threshold-check',
        deviceSns: ['ENB00001', 'ENB00002', 'ENB00003'],
        kpiCodes: ['RRC_SUCC_RATE', 'RADIO_DROP_RATE'],
        granularity: '15min',
        timeRange: [
          new Date(Date.now() - 3600000).toISOString(),
          new Date().toISOString(),
        ],
        status: 'running',
        progress: 60,
        createdAt: new Date(Date.now() - 600000).toISOString(),
        updatedAt: new Date(Date.now() - 60000).toISOString(),
        creator: 'system',
      },
    ];
    return paginate(tasks, params.page, params.pageSize);
  },

  async createTask(data: Omit<PerformanceTask, 'id' | 'status' | 'progress' | 'createdAt' | 'updatedAt'>): Promise<PerformanceTask> {
    await delay(200, 400);
    return {
      ...data,
      id: generateId('ptask'),
      status: 'pending',
      progress: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
  },
};
