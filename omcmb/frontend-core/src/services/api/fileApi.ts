import http from '../http';
import type { ManagedFile, FileType, FileStatus } from '../../mock/data/fileManagement';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---------------------------------------------------------------------------
// Backend response types
// ---------------------------------------------------------------------------

interface BackendManagedFile {
  id: string;
  file_name: string;
  file_type: string;   // config, log, firmware, script, other
  file_size: number;
  minio_path: string;
  content_type: string;
  uploader: string;
  device_sn: string;
  status: string;       // uploaded, processing, ready, failed
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

interface BackendDistributeResponse {
  task_id: string;
  file_id: string;
  device_count: number;
  status: string;
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend file_type <-> frontend FileType
// ---------------------------------------------------------------------------

/**
 * Backend uses: config, log, firmware, script, other
 * Frontend uses: config, log, firmware, backup, report, certificate
 *
 * We pass through values that overlap and map the rest as 'other'.
 */
function mapBackendFileType(backendType: string): FileType {
  const known: Record<string, FileType> = {
    config: 'config',
    log: 'log',
    firmware: 'firmware',
  };
  return known[backendType] ?? ('config' as FileType);
}

function mapFrontendFileType(frontendType: FileType): string {
  const known: Record<string, string> = {
    config: 'config',
    log: 'log',
    firmware: 'firmware',
    backup: 'other',
    report: 'other',
    certificate: 'other',
  };
  return known[frontendType] ?? 'other';
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend status <-> frontend FileStatus
// ---------------------------------------------------------------------------

/**
 * Backend uses: uploaded, processing, ready, failed
 * Frontend uses: available, uploading, processing, expired, deleted
 */
function mapBackendStatus(backendStatus: string): FileStatus {
  const map: Record<string, FileStatus> = {
    uploaded: 'available',
    processing: 'processing',
    ready: 'available',
    failed: 'expired',
  };
  return map[backendStatus] ?? ('available' as FileStatus);
}

function mapFrontendStatus(frontendStatus: FileStatus): string {
  const map: Record<string, string> = {
    available: 'ready',
    uploading: 'uploaded',
    processing: 'processing',
    expired: 'failed',
    deleted: 'failed',
  };
  return map[frontendStatus] ?? 'ready';
}

// ---------------------------------------------------------------------------
// Mapping functions: backend -> frontend
// ---------------------------------------------------------------------------

function mapBackendFile(bf: BackendManagedFile): ManagedFile {
  return {
    id: bf.id,
    fileName: bf.file_name,
    fileType: mapBackendFileType(bf.file_type),
    fileSize: bf.file_size,
    mimeType: bf.content_type || 'application/octet-stream',
    status: mapBackendStatus(bf.status),
    deviceSn: bf.device_sn || undefined,
    deviceName: undefined,
    uploadTime: bf.created_at,
    expiryTime: undefined,
    uploader: bf.uploader || '',
    checksum: '',
    downloadUrl: `/files/${bf.id}/download`,
    description: bf.description || undefined,
    tags: [],
  };
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export const fileApi = {
  async getList(
    params: { fileType?: FileType; status?: FileStatus; keyword?: string; deviceSn?: string } & PageRequest
  ): Promise<PageResponse<ManagedFile>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.fileType) query.file_type = mapFrontendFileType(params.fileType);
    if (params.status) query.status = mapFrontendStatus(params.status);
    if (params.keyword) query.search = params.keyword;
    if (params.deviceSn) query.device_sn = params.deviceSn;

    const { data } = await http.get<BackendListResponse<BackendManagedFile>>(
      '/files',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendFile),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getById(id: string): Promise<ManagedFile | null> {
    try {
      const { data } = await http.get<BackendManagedFile>(`/files/${id}`);
      return mapBackendFile(data);
    } catch {
      return null;
    }
  },

  async upload(data: Omit<ManagedFile, 'id' | 'uploadTime'>): Promise<ManagedFile> {
    const formData = new FormData();

    // The real upload expects a File object. When called from the frontend form
    // the caller should have attached a File via (data as any).file.
    // For the API layer we build the FormData from the fields the backend expects.
    const fileObj = (data as unknown as { file?: File }).file;
    if (fileObj) {
      formData.append('file', fileObj);
    }

    formData.append('file_type', mapFrontendFileType(data.fileType));
    if (data.description) formData.append('description', data.description);
    if (data.deviceSn) formData.append('device_sn', data.deviceSn);
    if (data.uploader) formData.append('uploader', data.uploader);

    const { data: bf } = await http.post<BackendManagedFile>('/files', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });

    return mapBackendFile(bf);
  },

  async delete(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/files/${id}`);
    }
  },

  async download(id: string): Promise<{ url: string; fileName: string }> {
    const response = await http.get(`/files/${id}/download`, {
      responseType: 'blob',
    });

    // Extract filename from Content-Disposition header
    const disposition = response.headers['content-disposition'] as string | undefined;
    const filename = disposition
      ? disposition.split('filename=')[1]?.replace(/['"]/g, '')
      : `file_${id}`;

    // Trigger browser download
    const url = window.URL.createObjectURL(new Blob([response.data as BlobPart]));
    const link = document.createElement('a');
    link.href = url;
    link.download = filename || `file_${id}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);

    return { url, fileName: filename || `file_${id}` };
  },

  async distribute(fileId: string, deviceSns: string[]): Promise<{ taskId: string }> {
    const { data } = await http.post<BackendDistributeResponse>(
      `/files/${fileId}/distribute`,
      { device_sns: deviceSns }
    );
    return { taskId: data.task_id };
  },

  /**
   * Storage stats — the backend does not expose a dedicated stats endpoint.
   * We derive basic stats from a single list request (first page only).
   * For a richer implementation this could be replaced with a dedicated
   * backend endpoint in the future.
   */
  async getStorageStats(): Promise<{
    totalSize: number;
    usedSize: number;
    fileCount: number;
    byType: Record<string, number>;
  }> {
    const { data } = await http.get<BackendListResponse<BackendManagedFile>>(
      '/files',
      { params: { page: 1, page_size: 1 } }
    );

    // We only have the total count from the list response.
    // Real size data would require a backend endpoint.
    return {
      totalSize: 1024 * 1024 * 1024 * 10, // 10 GB placeholder
      usedSize: 0,
      fileCount: data.total ?? 0,
      byType: {},
    };
  },
};
