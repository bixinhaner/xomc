import type { PageRequest, PageResponse } from '../../types/pagination';
import type { MRIndicator, MRDeviceMapping, MRRecord } from '../data/mr';
import { mockMRIndicators, mockMRDeviceMappings, mockMRRecords } from '../data/mr';
import { delay, paginate, generateId } from '../utils';

const mappings = [...mockMRDeviceMappings];

export const mrService = {
  async getIndicators(params: PageRequest): Promise<PageResponse<MRIndicator>> {
    await delay(80, 150);
    return paginate(mockMRIndicators, params.page, params.pageSize);
  },

  async getAllIndicators(): Promise<MRIndicator[]> {
    await delay(80, 150);
    return mockMRIndicators;
  },

  async getMappings(
    params: { deviceSn?: string; enabled?: boolean } & PageRequest
  ): Promise<PageResponse<MRDeviceMapping>> {
    await delay(100, 200);
    let filtered = [...mappings];
    if (params.deviceSn) filtered = filtered.filter((m) => m.deviceSn === params.deviceSn);
    if (params.enabled !== undefined) filtered = filtered.filter((m) => m.enabled === params.enabled);
    return paginate(filtered, params.page, params.pageSize);
  },

  async updateMapping(id: string, data: Partial<MRDeviceMapping>): Promise<MRDeviceMapping> {
    await delay(150, 300);
    const idx = mappings.findIndex((m) => m.id === id);
    if (idx === -1) throw new Error(`MR mapping ${id} not found`);
    mappings[idx] = { ...mappings[idx], ...data };
    return mappings[idx];
  },

  async toggleMapping(id: string, enabled: boolean): Promise<MRDeviceMapping> {
    await delay(150, 300);
    const idx = mappings.findIndex((m) => m.id === id);
    if (idx === -1) throw new Error(`MR mapping ${id} not found`);
    mappings[idx] = { ...mappings[idx], enabled };
    return mappings[idx];
  },

  async getRecords(
    params: { deviceSn?: string; cellId?: string; timeRange?: [string, string] } & PageRequest
  ): Promise<PageResponse<MRRecord>> {
    await delay(100, 250);
    let filtered = [...mockMRRecords];
    if (params.deviceSn) filtered = filtered.filter((r) => r.deviceSn === params.deviceSn);
    if (params.cellId) filtered = filtered.filter((r) => r.cellId === params.cellId);
    if (params.timeRange) {
      const [start, end] = params.timeRange;
      filtered = filtered.filter((r) => r.timestamp >= start && r.timestamp <= end);
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async exportMRData(params: { deviceSns: string[]; timeRange: [string, string] }): Promise<{ taskId: string }> {
    await delay(300, 600);
    void params;
    return { taskId: generateId('mr-export') };
  },

  async getIndicatorStats(indicatorCode: string, deviceSn?: string): Promise<{
    avg: number;
    max: number;
    min: number;
    p50: number;
    p95: number;
    sampleCount: number;
  }> {
    await delay(200, 400);
    void deviceSn;
    const indicator = mockMRIndicators.find((i) => i.indicatorCode === indicatorCode);
    const [min, max] = indicator?.valueRange ?? [0, 100];
    const avg = (min + max) / 2 + (Math.random() - 0.5) * (max - min) * 0.3;
    return {
      avg: parseFloat(avg.toFixed(2)),
      max: parseFloat((avg + Math.random() * (max - avg) * 0.5).toFixed(2)),
      min: parseFloat((avg - Math.random() * (avg - min) * 0.5).toFixed(2)),
      p50: parseFloat((avg + (Math.random() - 0.5) * 2).toFixed(2)),
      p95: parseFloat((avg + Math.random() * (max - avg) * 0.3).toFixed(2)),
      sampleCount: Math.floor(Math.random() * 50000 + 10000),
    };
  },
};
