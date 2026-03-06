import http from '../http';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mrService } from '@/mock/services/mrService';

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

  // --- Delegated to mock (no backend endpoint) ---

  getIndicators: mrService.getIndicators.bind(mrService),
  getAllIndicators: mrService.getAllIndicators.bind(mrService),
  getMappings: mrService.getMappings.bind(mrService),
  updateMapping: mrService.updateMapping.bind(mrService),
  toggleMapping: mrService.toggleMapping.bind(mrService),
  exportMRData: mrService.exportMRData.bind(mrService),
  getIndicatorStats: mrService.getIndicatorStats.bind(mrService),
};
