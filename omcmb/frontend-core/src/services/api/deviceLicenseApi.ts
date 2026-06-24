/**
 * deviceLicenseApi (T-0165) — 设备 license 文件库 API。
 *
 * 一台设备一份最新 license（serial_number 主键），独立 bucket=device-licenses。
 * 写入只来自手动导入；读取入口：① license 文件库页面；② LICENSE_UPGRADE 任务派发。
 *
 * 后端端点：/api/v1/backup/device-licenses/*
 */
import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { filenameFromContentDisposition, saveBlob } from '../../utils/saveBlob';

// ─────────────────────────────────────────────────────────────────────────
// Backend shapes
// ─────────────────────────────────────────────────────────────────────────

export type LicenseSource = 'manual_upload';
export type LicenseFileExt = 'lic';

interface BackendDeviceLicense {
  serial_number: string;
  enb_name?: string | null;
  product_type?: string | null;
  file_name: string;
  file_ext: LicenseFileExt;
  object_bucket: string;
  object_path: string;
  md5?: string | null;
  file_size: number;
  source: LicenseSource;
  description?: string | null;
  update_by?: string | null;
  update_time: string;
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

interface BackendLicenseImportFailure {
  file_name: string;
  serial_number?: string;
  error_code: string;
  message: string;
}

interface BackendLicenseImportResult {
  succeeded: string[];
  failed: BackendLicenseImportFailure[];
}

interface BackendBatchGetResponse {
  found: Record<string, BackendDeviceLicense>;
  missing: string[];
}

// ─────────────────────────────────────────────────────────────────────────
// Frontend shapes
// ─────────────────────────────────────────────────────────────────────────

export interface DeviceLicense {
  serialNumber: string;
  enbName?: string;
  productType?: string;
  fileName: string;
  fileExt: LicenseFileExt;
  objectBucket: string;
  objectPath: string;
  md5?: string;
  fileSize: number;
  source: LicenseSource;
  description?: string;
  updateBy?: string;
  updateTime: string;
  createdAt: string;
  updatedAt: string;
}

export interface LicenseImportFailure {
  fileName: string;
  serialNumber?: string;
  errorCode: string;
  message: string;
}

export interface LicenseImportResult {
  succeeded: string[];
  failed: LicenseImportFailure[];
}

export interface BatchGetLicensesResult {
  found: Record<string, DeviceLicense>;
  missing: string[];
}

export interface LicenseListParams extends PageRequest {
  serialNumber?: string;
  enbName?: string;
  productType?: string;
  /** #602：按产品名称下拉过滤，传 product.id。 */
  productId?: string;
  updatedAfter?: string;
  updatedBefore?: string;
}

// ─────────────────────────────────────────────────────────────────────────
// Mappers
// ─────────────────────────────────────────────────────────────────────────

function mapBackendLicense(b: BackendDeviceLicense): DeviceLicense {
  return {
    serialNumber: b.serial_number,
    enbName: b.enb_name ?? undefined,
    productType: b.product_type ?? undefined,
    fileName: b.file_name,
    fileExt: b.file_ext,
    objectBucket: b.object_bucket,
    objectPath: b.object_path,
    md5: b.md5 ?? undefined,
    fileSize: b.file_size,
    source: b.source,
    description: b.description ?? undefined,
    updateBy: b.update_by ?? undefined,
    updateTime: b.update_time,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapImportFailure(b: BackendLicenseImportFailure): LicenseImportFailure {
  return {
    fileName: b.file_name,
    serialNumber: b.serial_number,
    errorCode: b.error_code,
    message: b.message,
  };
}

// ─────────────────────────────────────────────────────────────────────────
// 客户端校验：文件名 <SN>.lic，允许后面带任何尾缀（macOS Safari 下载 .lic
// 时会自动加 .html；Windows 邮件客户端可能加 .txt）。落库统一 <SN>.lic。
// ─────────────────────────────────────────────────────────────────────────

const LICENSE_NAME_PATTERN = /^([A-Za-z0-9_-]+)\.lic(?:\.[A-Za-z0-9]+)*$/i;

export function validateLicenseFileName(fileName: string): {
  valid: boolean;
  serialNumber?: string;
  ext?: LicenseFileExt;
  message?: string;
} {
  if (!fileName) {
    return { valid: false, message: '文件名不能为空' };
  }
  const m = LICENSE_NAME_PATTERN.exec(fileName);
  if (!m) {
    return {
      valid: false,
      message: `文件名不符合规范，期望 <serialNumber>.lic，实际收到 ${fileName}`,
    };
  }
  return {
    valid: true,
    serialNumber: m[1],
    ext: 'lic',
  };
}

// ─────────────────────────────────────────────────────────────────────────
// API
// ─────────────────────────────────────────────────────────────────────────

export const deviceLicenseApi = {
  async list(params: LicenseListParams): Promise<PageResponse<DeviceLicense>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.serialNumber) query.serial_number = params.serialNumber;
    if (params.enbName) query.enb_name = params.enbName;
    if (params.productType) query.product_type = params.productType;
    if (params.productId) query.product_id = params.productId;
    if (params.updatedAfter) query.updated_after = params.updatedAfter;
    if (params.updatedBefore) query.updated_before = params.updatedBefore;

    const { data } = await http.get<BackendListResponse<BackendDeviceLicense>>(
      '/backup/device-licenses',
      { params: query },
    );
    return {
      items: (data.items || []).map(mapBackendLicense),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getBySerialNumber(sn: string): Promise<DeviceLicense | null> {
    try {
      const { data } = await http.get<BackendDeviceLicense>(
        `/backup/device-licenses/${encodeURIComponent(sn)}`,
      );
      return mapBackendLicense(data);
    } catch {
      return null;
    }
  },

  async batchGet(serialNumbers: string[]): Promise<BatchGetLicensesResult> {
    const { data } = await http.post<BackendBatchGetResponse>(
      '/backup/device-licenses/batch-get',
      { serial_numbers: serialNumbers },
    );
    const found: Record<string, DeviceLicense> = {};
    for (const [sn, lic] of Object.entries(data.found || {})) {
      found[sn] = mapBackendLicense(lic);
    }
    return { found, missing: data.missing || [] };
  },

  async validateSNs(serialNumbers: string[]): Promise<{ existing: string[]; missing: string[] }> {
    if (serialNumbers.length === 0) {
      return { existing: [], missing: [] };
    }
    const { data } = await http.post<{ existing: string[]; missing: string[] }>(
      '/backup/device-licenses/validate-sns',
      { serial_numbers: serialNumbers },
    );
    return { existing: data.existing || [], missing: data.missing || [] };
  },

  async import(files: File[]): Promise<LicenseImportResult> {
    const fd = new FormData();
    for (const f of files) {
      fd.append('files', f, f.name);
    }
    const { data } = await http.post<BackendLicenseImportResult>(
      '/backup/device-licenses/import',
      fd,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return {
      succeeded: data.succeeded || [],
      failed: (data.failed || []).map(mapImportFailure),
    };
  },

  async download(sn: string): Promise<void> {
    const response = await http.get(`/backup/device-licenses/${encodeURIComponent(sn)}/download`, {
      responseType: 'blob',
    });
    const filename = filenameFromContentDisposition(
      response.headers['content-disposition'] as string | undefined,
      `${sn}.lic`,
    );
    saveBlob(response.data as BlobPart, filename);
  },

  async delete(sn: string): Promise<void> {
    await http.delete(`/backup/device-licenses/${encodeURIComponent(sn)}`);
  },

  async batchDelete(serialNumbers: string[]): Promise<{ succeeded: string[]; failed: string[] }> {
    const { data } = await http.post<{ succeeded: string[]; failed: string[] }>(
      '/backup/device-licenses/batch-delete',
      { serial_numbers: serialNumbers },
    );
    return { succeeded: data.succeeded || [], failed: data.failed || [] };
  },
};
