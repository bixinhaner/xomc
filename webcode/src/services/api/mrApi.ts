import http from '../http';
import type { PageRequest, PageResponse } from '@/types/pagination';
import type { MRIndicator, MRDeviceMapping } from '@/mock/data/mr';

// --- Backend response types ---

interface BackendMRFileInfo {
  id: string;
  device_id: string;
  device_sn: string;
  carrier: string;
  mr_type: string;
  file_name: string;
  file_size: number;
  collect_time: string;
  minio_path: string;
  parsed: boolean;
  parsed_at?: string;
  record_count: number;
  created_at: string;
}

interface BackendMRRecordEntry {
  time: string;
  file_id: string;
  device_id: string;
  cell_id: string;
  mr_type: string;
  measurement_data: Record<string, number>;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface BackendMRIndicator {
  id: string;
  indicator_name: string;
  indicator_code: string;
  description?: string;
  unit?: string;
  category?: string;
  value_range_min?: number;
  value_range_max?: number;
  created_at: string;
}

interface BackendMRDeviceMapping {
  id: string;
  device_sn: string;
  device_name?: string;
  cell_id: string;
  cell_name?: string;
  enabled: boolean;
  sampling_interval: number;
  last_collect_time?: string;
  total_records: number;
  created_at: string;
  updated_at: string;
}

// --- Frontend types for MR files ---

export interface MRFileItem {
  id: string;
  deviceSn: string;
  carrier: string;
  mrType: string;
  fileName: string;
  fileSize: number;
  collectTime: string;
  parsed: boolean;
  recordCount: number;
  createdAt: string;
}

// MR data record (matches mock MRRecord shape)
export interface MRDataRecord {
  id: string;
  deviceSn: string;
  cellId: string;
  timestamp: string;
  indicators: Record<string, number>;
}

// --- Mapping functions ---

function mapMRFile(f: BackendMRFileInfo): MRFileItem {
  return {
    id: f.id,
    deviceSn: f.device_sn,
    carrier: f.carrier,
    mrType: f.mr_type,
    fileName: f.file_name,
    fileSize: f.file_size,
    collectTime: f.collect_time,
    parsed: f.parsed,
    recordCount: f.record_count,
    createdAt: f.created_at,
  };
}

function mapMRRecord(r: BackendMRRecordEntry): MRDataRecord {
  return {
    id: `${r.file_id}-${r.cell_id}-${r.time}`,
    deviceSn: '',
    cellId: r.cell_id,
    timestamp: r.time,
    indicators: r.measurement_data || {},
  };
}

function mapBackendIndicator(bi: BackendMRIndicator): MRIndicator {
  return {
    id: bi.id,
    indicatorName: bi.indicator_name,
    indicatorCode: bi.indicator_code,
    description: bi.description || '',
    unit: bi.unit || '',
    category: bi.category || '',
    valueRange: [bi.value_range_min ?? 0, bi.value_range_max ?? 0],
  };
}

function mapBackendMapping(bm: BackendMRDeviceMapping): MRDeviceMapping {
  return {
    id: bm.id,
    deviceSn: bm.device_sn,
    deviceName: bm.device_name || '',
    cellId: bm.cell_id,
    cellName: bm.cell_name || '',
    enabled: bm.enabled,
    samplingInterval: bm.sampling_interval,
    lastCollectTime: bm.last_collect_time,
    totalRecords: bm.total_records,
  };
}

// --- Exported service ---

export const mrApi = {
  // MR file listing
  async getFiles(
    params: {
      deviceSn?: string;
      mrType?: string;
      timeRange?: [string, string];
    } & PageRequest
  ): Promise<PageResponse<MRFileItem>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.mrType) query.mr_type = params.mrType;
    if (params.timeRange) {
      query.start_time = params.timeRange[0];
      query.end_time = params.timeRange[1];
    }

    const { data } = await http.get<BackendListResponse<BackendMRFileInfo>>(
      '/mr/files',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapMRFile),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // MR parsed data records
  async getRecords(
    params: {
      deviceSn?: string;
      cellId?: string;
      timeRange?: [string, string];
    } & PageRequest
  ): Promise<PageResponse<MRDataRecord>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.cellId) query.cell_id = params.cellId;
    if (params.timeRange) {
      query.start_time = params.timeRange[0];
      query.end_time = params.timeRange[1];
    }

    const { data } = await http.get<BackendListResponse<BackendMRRecordEntry>>(
      '/mr/data',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapMRRecord),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  // MR file download (blob)
  async downloadFile(fileId: string): Promise<void> {
    const response = await http.get(`/mr/files/${fileId}/download`, {
      responseType: 'blob',
    });

    // Extract filename from Content-Disposition header
    const disposition = response.headers['content-disposition'] as string | undefined;
    const filename = disposition
      ? disposition.split('filename=')[1]?.replace(/['"]/g, '')
      : `mr_${fileId}.xml`;

    // Trigger browser download
    const url = window.URL.createObjectURL(new Blob([response.data as BlobPart]));
    const link = document.createElement('a');
    link.href = url;
    link.download = filename || `mr_${fileId}.xml`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  },

  // --- Indicators ---

  async getIndicators(
    params: PageRequest
  ): Promise<PageResponse<MRIndicator>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };

    const { data } = await http.get<BackendListResponse<BackendMRIndicator>>(
      '/mr/indicators',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendIndicator),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getAllIndicators(): Promise<MRIndicator[]> {
    const { data } = await http.get<BackendMRIndicator[]>(
      '/mr/indicators/all'
    );
    return (data || []).map(mapBackendIndicator);
  },

  // --- Mappings ---

  async getMappings(
    params: { deviceSn?: string; enabled?: boolean } & PageRequest
  ): Promise<PageResponse<MRDeviceMapping>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.enabled !== undefined) query.enabled = params.enabled;

    const { data } = await http.get<BackendListResponse<BackendMRDeviceMapping>>(
      '/mr/mappings',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendMapping),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async updateMapping(
    id: string,
    data: Partial<MRDeviceMapping>
  ): Promise<MRDeviceMapping> {
    const payload: Record<string, unknown> = {};
    if (data.deviceSn !== undefined) payload.device_sn = data.deviceSn;
    if (data.deviceName !== undefined) payload.device_name = data.deviceName;
    if (data.cellId !== undefined) payload.cell_id = data.cellId;
    if (data.cellName !== undefined) payload.cell_name = data.cellName;
    if (data.enabled !== undefined) payload.enabled = data.enabled;
    if (data.samplingInterval !== undefined) payload.sampling_interval = data.samplingInterval;

    const { data: bm } = await http.put<BackendMRDeviceMapping>(
      `/mr/mappings/${id}`,
      payload
    );
    return mapBackendMapping(bm);
  },

  async toggleMapping(
    id: string,
    enabled: boolean
  ): Promise<MRDeviceMapping> {
    const { data } = await http.put<BackendMRDeviceMapping>(
      `/mr/mappings/${id}/toggle`,
      { enabled }
    );
    return mapBackendMapping(data);
  },

  // --- Export ---

  async exportMRData(
    params: { deviceSns: string[]; timeRange: [string, string] }
  ): Promise<{ taskId: string }> {
    const { data } = await http.post<{ task_id: string; status: string }>(
      '/mr/export',
      params
    );
    return { taskId: data.task_id };
  },

  // --- Indicator Stats ---

  async getIndicatorStats(
    indicatorCode: string,
    deviceSn?: string
  ): Promise<{
    avg: number;
    max: number;
    min: number;
    p50: number;
    p95: number;
    sampleCount: number;
  }> {
    const query: Record<string, unknown> = {};
    if (deviceSn) query.device_sn = deviceSn;

    const { data } = await http.get<{
      indicator_code: string;
      avg: number;
      min: number;
      max: number;
      p50: number;
      p95: number;
      sample_count: number;
    }>(`/mr/indicators/${indicatorCode}/stats`, { params: query });

    return {
      avg: data.avg,
      max: data.max,
      min: data.min,
      p50: data.p50,
      p95: data.p95,
      sampleCount: data.sample_count,
    };
  },
};
