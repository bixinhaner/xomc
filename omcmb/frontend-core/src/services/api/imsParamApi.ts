/**
 * imsParamApi — 核心网（IMS Core）参数文件库 API。
 *
 * 参数文件库承载运营者上传的 FT1~FT12 参数配置文件；IMS_PARAM_DISTRIBUTE 任务
 * 从这里选文件下发。参数类型注册表（P1~P12）也经本 API 暴露给任务创建抽屉。
 *
 * 后端端点：/api/v1/imsparam/*（docs/design/imscore-file-transfer.md）
 */
import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { filenameFromContentDisposition, saveBlob } from '../../utils/saveBlob';

// ─────────────────────────────────────────────────────────────────────────
// Backend shapes
// ─────────────────────────────────────────────────────────────────────────

export interface ImsParamTypeOption {
  code: string; // FT_ImsCore_User_Setting_UD ...
  name: string; // 功能名称（中文）
  uploadSupported: boolean;
  downloadSupported: boolean;
}

interface BackendImsParamFile {
  id: string;
  param_type: string;
  file_name: string;
  object_path: string;
  md5?: string | null;
  file_size: number;
  description?: string | null;
  uploaded_by?: string | null;
  device_sn?: string | null;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse {
  items: BackendImsParamFile[];
  total: number;
  page: number;
  page_size: number;
}

export interface ImsParamImportFailure {
  file_name: string;
  error_code: string;
  message: string;
}

export interface ImsParamImportResult {
  succeeded: string[];
  failed: ImsParamImportFailure[];
}

// ─────────────────────────────────────────────────────────────────────────
// Frontend shapes
// ─────────────────────────────────────────────────────────────────────────

export interface ImsParamFile {
  id: string;
  paramType: string;
  fileName: string;
  objectPath: string;
  md5?: string;
  fileSize: number;
  description?: string;
  uploadedBy?: string;
  deviceSn?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ImsParamListParams extends PageRequest {
  paramType?: string;
  fileName?: string;
  uploadedBy?: string;
  deviceSn?: string;
}

// ─────────────────────────────────────────────────────────────────────────
// API
// ─────────────────────────────────────────────────────────────────────────

function mapBackendFile(b: BackendImsParamFile): ImsParamFile {
  return {
    id: b.id,
    paramType: b.param_type,
    fileName: b.file_name,
    objectPath: b.object_path,
    md5: b.md5 ?? undefined,
    fileSize: b.file_size,
    description: b.description ?? undefined,
    uploadedBy: b.uploaded_by ?? undefined,
    deviceSn: b.device_sn ?? undefined,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

export const imsParamApi = {
  async getParamTypes(): Promise<ImsParamTypeOption[]> {
    const { data } = await http.get<ImsParamTypeOption[]>('/imsparam/param-types');
    return data ?? [];
  },

  async list(params: ImsParamListParams): Promise<PageResponse<ImsParamFile>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.paramType) query.param_type = params.paramType;
    if (params.fileName) query.file_name = params.fileName;
    if (params.uploadedBy) query.uploaded_by = params.uploadedBy;
    if (params.deviceSn) query.device_sn = params.deviceSn;
    const { data } = await http.get<BackendListResponse>('/imsparam/param-files', { params: query });
    return {
      items: (data.items || []).map(mapBackendFile),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async import(paramType: string, files: File[], description?: string): Promise<ImsParamImportResult> {
    const fd = new FormData();
    fd.append('param_type', paramType);
    if (description) fd.append('description', description);
    for (const f of files) {
      fd.append('files', f, f.name);
    }
    const { data } = await http.post<ImsParamImportResult>(
      '/imsparam/param-files/import',
      fd,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return {
      succeeded: data.succeeded || [],
      failed: data.failed || [],
    };
  },

  async download(id: string, fallbackName: string): Promise<void> {
    const response = await http.get(`/imsparam/param-files/${encodeURIComponent(id)}/download`, {
      responseType: 'blob',
    });
    const filename = filenameFromContentDisposition(
      response.headers['content-disposition'] as string | undefined,
      fallbackName,
    );
    saveBlob(response.data as BlobPart, filename);
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/imsparam/param-files/${encodeURIComponent(id)}`);
  },

  async batchDelete(ids: string[]): Promise<{ succeeded: string[]; failed: string[] }> {
    const { data } = await http.post<{ succeeded: string[]; failed: string[] }>(
      '/imsparam/param-files/batch-delete',
      { ids },
    );
    return { succeeded: data.succeeded || [], failed: data.failed || [] };
  },
};
