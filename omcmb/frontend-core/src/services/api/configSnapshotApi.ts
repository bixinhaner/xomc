/**
 * configSnapshotApi (T-0164 / F1) — 配置文件快照表 API。
 *
 * 一台设备一份最新配置快照（serial_number 主键），独立于 backup_tasks
 * 的逐任务文件归档。写入：① 备份任务完成后自动 promote；② 用户手动
 * 上传导入。读取：① 配置快照库页面；② Restore 创建页"按设备快照" Tab。
 *
 * 历史数据不回填（用户确认 2026-05-22）—— 表上线后从首次新备份 / 导入累积。
 *
 * 后端端点：/api/v1/backup/config-snapshots/*
 */
import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ─────────────────────────────────────────────────────────────────────────
// Backend shapes (snake_case before axios camelCase conversion)
// ─────────────────────────────────────────────────────────────────────────

export type SnapshotSource = 'backup' | 'manual_upload';
export type SnapshotFileExt = 'xml' | 'nv';

interface BackendConfigSnapshot {
  serial_number: string;
  enb_name?: string | null;
  product_type?: string | null;
  file_name: string;
  file_ext: SnapshotFileExt;
  object_bucket: string;
  object_path: string;
  md5?: string | null;
  file_size: number;
  source: SnapshotSource;
  source_task_id?: string | null;
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

interface BackendSnapshotImportFailure {
  file_name: string;
  serial_number?: string;
  error_code: string;
  message: string;
}

interface BackendSnapshotImportResult {
  succeeded: string[];
  failed: BackendSnapshotImportFailure[];
}

interface BackendBatchGetResponse {
  found: Record<string, BackendConfigSnapshot>;
  missing: string[];
}

// ─────────────────────────────────────────────────────────────────────────
// Frontend shapes
// ─────────────────────────────────────────────────────────────────────────

export interface ConfigSnapshot {
  serialNumber: string;
  enbName?: string;
  productType?: string;
  fileName: string;
  fileExt: SnapshotFileExt;
  objectBucket: string;
  objectPath: string;
  md5?: string;
  fileSize: number;
  source: SnapshotSource;
  sourceTaskId?: string;
  updateBy?: string;
  updateTime: string;
  createdAt: string;
  updatedAt: string;
}

export interface SnapshotImportFailure {
  fileName: string;
  serialNumber?: string;
  errorCode: string;
  message: string;
}

export interface SnapshotImportResult {
  succeeded: string[];
  failed: SnapshotImportFailure[];
}

export interface BatchGetSnapshotsResult {
  found: Record<string, ConfigSnapshot>;
  missing: string[];
}

export interface SnapshotListParams extends PageRequest {
  serialNumber?: string;
  enbName?: string;
  productType?: string;
  source?: SnapshotSource;
  updatedAfter?: string;
  updatedBefore?: string;
}

// ─────────────────────────────────────────────────────────────────────────
// Mappers (snake↔camel 手动版，因为可空字段需要 undefined 归一)
// ─────────────────────────────────────────────────────────────────────────

function mapBackendSnapshot(b: BackendConfigSnapshot): ConfigSnapshot {
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
    sourceTaskId: b.source_task_id ?? undefined,
    updateBy: b.update_by ?? undefined,
    updateTime: b.update_time,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapImportFailure(b: BackendSnapshotImportFailure): SnapshotImportFailure {
  return {
    fileName: b.file_name,
    serialNumber: b.serial_number,
    errorCode: b.error_code,
    message: b.message,
  };
}

// ─────────────────────────────────────────────────────────────────────────
// Client-side validation: import file naming
// ─────────────────────────────────────────────────────────────────────────

/**
 * 与后端 ValidateImportFileName 一致的正则。允许 SN 字符：字母/数字/下划线/短横线。
 * 扩展名忽略大小写，但落库时统一小写。
 */
const SNAPSHOT_NAME_PATTERN = /^([A-Za-z0-9_-]+)_CFG\.(xml|nv)$/i;

export function validateSnapshotFileName(fileName: string): {
  valid: boolean;
  serialNumber?: string;
  ext?: SnapshotFileExt;
  message?: string;
} {
  if (!fileName) {
    return { valid: false, message: '文件名不能为空' };
  }
  const m = SNAPSHOT_NAME_PATTERN.exec(fileName);
  if (!m) {
    return {
      valid: false,
      message: `文件名不符合规范，期望 <serialNumber>_CFG.xml 或 <serialNumber>_CFG.nv，实际收到 ${fileName}`,
    };
  }
  return {
    valid: true,
    serialNumber: m[1],
    ext: m[2].toLowerCase() as SnapshotFileExt,
  };
}

// ─────────────────────────────────────────────────────────────────────────
// API
// ─────────────────────────────────────────────────────────────────────────

export const configSnapshotApi = {
  async list(params: SnapshotListParams): Promise<PageResponse<ConfigSnapshot>> {
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
    };
    if (params.serialNumber) query.serial_number = params.serialNumber;
    if (params.enbName) query.enb_name = params.enbName;
    if (params.productType) query.product_type = params.productType;
    if (params.source) query.source = params.source;
    if (params.updatedAfter) query.updated_after = params.updatedAfter;
    if (params.updatedBefore) query.updated_before = params.updatedBefore;

    const { data } = await http.get<BackendListResponse<BackendConfigSnapshot>>(
      '/backup/config-snapshots',
      { params: query },
    );
    return {
      items: (data.items || []).map(mapBackendSnapshot),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getBySerialNumber(sn: string): Promise<ConfigSnapshot | null> {
    try {
      const { data } = await http.get<BackendConfigSnapshot>(
        `/backup/config-snapshots/${encodeURIComponent(sn)}`,
      );
      return mapBackendSnapshot(data);
    } catch {
      return null;
    }
  },

  async batchGet(serialNumbers: string[]): Promise<BatchGetSnapshotsResult> {
    const { data } = await http.post<BackendBatchGetResponse>(
      '/backup/config-snapshots/batch-get',
      { serial_numbers: serialNumbers },
    );
    const found: Record<string, ConfigSnapshot> = {};
    for (const [sn, snap] of Object.entries(data.found || {})) {
      found[sn] = mapBackendSnapshot(snap);
    }
    return { found, missing: data.missing || [] };
  },

  /**
   * 校验 SN 在 devices 表是否存在（导入文件前的预检查）。
   * 返回 existing/missing 两份列表 —— 给 ImportDrawer 把"SN 不存在"
   * 的文件项标红。
   */
  async validateSNs(serialNumbers: string[]): Promise<{ existing: string[]; missing: string[] }> {
    if (serialNumbers.length === 0) {
      return { existing: [], missing: [] };
    }
    const { data } = await http.post<{ existing: string[]; missing: string[] }>(
      '/backup/config-snapshots/validate-sns',
      { serial_numbers: serialNumbers },
    );
    return { existing: data.existing || [], missing: data.missing || [] };
  },

  /**
   * 上传多个文件到后端 import 端点（multipart/form-data）。
   * 前端层不再校验文件名 —— 由后端做权威校验，这里只构建 FormData。
   * 调用方可在 UI 层先跑 validateSnapshotFileName 给出即时反馈。
   */
  async import(files: File[]): Promise<SnapshotImportResult> {
    const fd = new FormData();
    for (const f of files) {
      fd.append('files', f, f.name);
    }
    const { data } = await http.post<BackendSnapshotImportResult>(
      '/backup/config-snapshots/import',
      fd,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return {
      succeeded: data.succeeded || [],
      failed: (data.failed || []).map(mapImportFailure),
    };
  },

  /**
   * 拿 1h 有效 presigned 下载 URL 后**自动触发浏览器下载**。
   *
   * 不能用 `window.open(/.../download)` —— 那个端点要 Bearer Token，浏览器
   * 新 tab 不会带；改成 axios 调拿 URL，再用 <a download> 触发下载。
   */
  async download(sn: string): Promise<void> {
    // 后端响应是 snake_case（snapshot_handler.go SnapshotDownloadResponse 的 json tag），
    // http.ts 响应拦截器只拆 envelope 不做命名转换 → 这里必须按 snake_case 读，
    // 否则 a.href = undefined → 浏览器下载当前页 HTML，a.download = undefined →
    // 字符串 "undefined" + 自动 .html 后缀 = "undefined.html"。
    const { data } = await http.get<{
      serial_number: string;
      file_name: string;
      download_url: string;
      expires_in_seconds: number;
    }>(`/backup/config-snapshots/${encodeURIComponent(sn)}/download`);
    // MinIO presigned URL 本身已含临时凭据，直接打开新 tab 即可。
    // 用 <a> + click 触发，比 window.open 在某些浏览器更稳定（不被 popup blocker）。
    const a = document.createElement('a');
    a.href = data.download_url;
    a.target = '_blank';
    a.rel = 'noopener';
    a.download = data.file_name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  },

  async delete(sn: string): Promise<void> {
    await http.delete(`/backup/config-snapshots/${encodeURIComponent(sn)}`);
  },

  async batchDelete(serialNumbers: string[]): Promise<{ succeeded: string[]; failed: string[] }> {
    // 走 POST /batch-delete（不是 DELETE+body）—— 后者在某些代理 / WAF 下
    // body 会被吞，后端 ShouldBindJSON 失败。详见 snapshot_handler.go 路由注释。
    const { data } = await http.post<{ succeeded: string[]; failed: string[] }>(
      '/backup/config-snapshots/batch-delete',
      { serial_numbers: serialNumbers },
    );
    return { succeeded: data.succeeded || [], failed: data.failed || [] };
  },
};
