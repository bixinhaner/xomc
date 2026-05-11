import http from '../http';
import type { License } from '../../mock/data/license';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---------------------------------------------------------------------------
// License audit logs (T-0100-P1)
// 与后端 internal/license/license_log_model.go 保持一致：9 种 log_type + 4 种 result。
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// T-0100-P3: Activate 同维度冲突响应 + Import 签名状态
// 后端契约：handler.ActivateConflictResponse / handler.ImportResponse
// ---------------------------------------------------------------------------

/** 后端 409 同维度冲突响应（snake_case，对应 ActivateConflictResponse）。 */
export interface ActivateConflictBody {
  conflict: 'same_dimension_active';
  device_type: string | null;
  region: string | null;
  conflicting_active: BackendLicense[];
  hint: string;
}

/** 前端 camelCase 形式（mapper 输出）。 */
export interface ActivateConflict {
  conflict: 'same_dimension_active';
  deviceType: string | null;
  region: string | null;
  conflictingActive: License[];
  hint: string;
}

export type LicenseSignatureStatus = 'verified' | 'unverified' | 'invalid';

/** Import 响应（POST /licenses/import 的 envelope.data）。 */
export interface ImportLicenseResult {
  license: License;
  signatureStatus: LicenseSignatureStatus;
  signatureNote: string;
}

interface BackendImportResponse {
  license: BackendLicense;
  signature_status: LicenseSignatureStatus;
  signature_note?: string;
}

export type LicenseLogType =
  | 'import'
  | 'activate'
  | 'revoke'
  | 'query_detail'
  | 'enforcement_capacity'
  | 'enforcement_expiry'
  | 'capacity_alert'
  | 'expiry_alert'
  | 'auto_expire';

export type LicenseLogResult = 'success' | 'failed' | 'denied' | 'warning';

/** 单条审计日志（前端类型，camelCase）。 */
export interface LicenseLog {
  id: string;
  licenseId: string | null;
  logType: LicenseLogType;
  actorUserId: string | null;
  result: LicenseLogResult;
  details: Record<string, unknown>;
  clientIp: string | null;
  userAgent: string | null;
  createdAt: string;
}

interface BackendLicenseLog {
  id: string;
  license_id: string | null;
  log_type: LicenseLogType;
  actor_user_id: string | null;
  result: LicenseLogResult;
  details: Record<string, unknown>;
  client_ip: string | null;
  user_agent: string | null;
  created_at: string;
}

function mapBackendLog(b: BackendLicenseLog): LicenseLog {
  return {
    id: b.id,
    licenseId: b.license_id,
    logType: b.log_type,
    actorUserId: b.actor_user_id,
    result: b.result,
    details: b.details ?? {},
    clientIp: b.client_ip,
    userAgent: b.user_agent,
    createdAt: b.created_at,
  };
}

/** 全量审计日志查询参数（与后端 ListLogs handler 接受的 query 对齐）。 */
export interface LicenseLogQuery extends PageRequest {
  licenseId?: string;
  actorUserId?: string;
  logTypes?: LicenseLogType[]; // 多值 → 后端 ?log_type=A&log_type=B
  results?: LicenseLogResult[];
  startTime?: string; // RFC3339
  endTime?: string;
  search?: string;
}

// ---------------------------------------------------------------------------
// Backend response types
// ---------------------------------------------------------------------------

interface BackendLicense {
  id: string;
  license_name: string;
  license_code: string;
  product_name: string;
  license_type: string;
  status: string;
  max_devices: number;
  used_devices: number;
  features: string[];
  issue_date: string;
  expiry_date: string | null;
  licensor: string | null;
  device_type: string | null;
  region: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

interface BackendLicenseSummary {
  total: number;
  active: number;
  expired: number;
  pending: number;
  expiring_soon: number;
  // T-0100-P2：近 7 天 enforcement 拒绝次数（result='denied'）
  // 后端 logRepo 未注入或查询失败时退化为 0
  enforcement_hits_7d?: number;
}

/** Summary 卡片用的统计快照（前端 camelCase）。 */
export interface LicenseSummary {
  total: number;
  active: number;
  expired: number;
  pending: number;
  expiringSoon: number;
  enforcementHits7d: number;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend -> frontend
// ---------------------------------------------------------------------------

function mapBackendLicense(bl: BackendLicense): License {
  return {
    id: bl.id,
    licenseName: bl.license_name,
    licenseCode: bl.license_code,
    productName: bl.product_name,
    licenseType: bl.license_type as License['licenseType'],
    status: bl.status as License['status'],
    maxDevices: bl.max_devices,
    usedDevices: bl.used_devices,
    features: bl.features || [],
    issueDate: bl.issue_date,
    expiryDate: bl.expiry_date,
    licensor: bl.licensor || '',
    deviceType: bl.device_type || '',
    region: bl.region || '',
    notes: bl.notes || undefined,
  };
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export const licenseApi = {
  async getLicenses(
    params: { status?: string; licenseType?: string; deviceType?: string } & PageRequest
  ): Promise<PageResponse<License>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.status) query.status = params.status;
    if (params.licenseType) query.license_type = params.licenseType;
    if (params.deviceType) query.device_type = params.deviceType;

    const { data } = await http.get<BackendListResponse<BackendLicense>>(
      '/licenses',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendLicense),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getLicenseById(id: string): Promise<License | null> {
    try {
      const { data } = await http.get<BackendLicense>(`/licenses/${id}`);
      return mapBackendLicense(data);
    } catch {
      return null;
    }
  },

  async getLicenseSummary(): Promise<LicenseSummary> {
    const { data } = await http.get<BackendLicenseSummary>('/licenses/summary');
    return {
      total: data.total,
      active: data.active,
      expired: data.expired,
      pending: data.pending,
      expiringSoon: data.expiring_soon,
      enforcementHits7d: data.enforcement_hits_7d ?? 0,
    };
  },

  /**
   * 激活 license（T-0100-P3）。
   *
   * - force=false（默认）：检测同 (device_type, region) 维度已有 active license
   *   时后端返 409 + ActivateConflictBody，调用方 catch error.response.data
   *   弹 Modal 二次确认。
   * - force=true：跳过冲突预检，自动 revoke 同维度旧 license 后再激活。
   */
  async activateLicense(licenseCode: string, force = false): Promise<License> {
    const { data } = await http.post<BackendLicense>('/licenses/activate', {
      license_code: licenseCode,
      force,
    });
    return mapBackendLicense(data);
  },

  async revokeLicense(id: string): Promise<void> {
    await http.post(`/licenses/${id}/revoke`);
  },

  /**
   * 单条 license PDF 导出（T-0100-P4-A）。
   *
   * 返回 Blob，调用方用 URL.createObjectURL 触发浏览器下载。后端走
   * GET /licenses/:id/export?format=pdf，Content-Type=application/pdf。
   */
  async exportLicensePDF(id: string): Promise<{ blob: Blob; filename: string }> {
    const resp = await http.get(`/licenses/${id}/export`, {
      params: { format: 'pdf' },
      responseType: 'blob',
    });
    const cd = (resp.headers as Record<string, string | undefined>)['content-disposition'] ?? '';
    return { blob: resp.data as Blob, filename: parseFilename(cd, `license-${id}.pdf`) };
  },

  /**
   * 全量 active license CSV 导出（T-0100-P4-A）。
   *
   * 后端 GET /licenses/export?format=csv，Content-Type=text/csv，UTF-8 BOM
   * 头方便 Excel 直接打开。
   */
  async exportAllActiveCSV(): Promise<{ blob: Blob; filename: string }> {
    const resp = await http.get(`/licenses/export`, {
      params: { format: 'csv' },
      responseType: 'blob',
    });
    const cd = (resp.headers as Record<string, string | undefined>)['content-disposition'] ?? '';
    return { blob: resp.data as Blob, filename: parseFilename(cd, 'licenses-active.csv') };
  },

  // -------------------------------------------------------------------------
  // T-0100-P1 audit logs
  // -------------------------------------------------------------------------

  /**
   * 拉取全量审计日志（分页 + 过滤）。
   *
   * `URLSearchParams` 直接 append 同名 key 实现 ?log_type=A&log_type=B 多值；
   * Axios `params` 默认序列化用逗号分隔，与后端 c.QueryArray 不兼容。
   */
  async getLicenseLogs(params: LicenseLogQuery): Promise<PageResponse<LicenseLog>> {
    const sp = new URLSearchParams();
    if (params.page) sp.set('page', String(params.page));
    if (params.pageSize) sp.set('page_size', String(params.pageSize));
    if (params.licenseId) sp.set('license_id', params.licenseId);
    if (params.actorUserId) sp.set('actor_user_id', params.actorUserId);
    (params.logTypes ?? []).forEach((t) => sp.append('log_type', t));
    (params.results ?? []).forEach((r) => sp.append('result', r));
    if (params.startTime) sp.set('start_time', params.startTime);
    if (params.endTime) sp.set('end_time', params.endTime);
    if (params.search) sp.set('search', params.search);

    const url = sp.toString() ? `/licenses/logs?${sp.toString()}` : '/licenses/logs';
    interface Resp {
      items: BackendLicenseLog[];
      total: number;
      page: number;
      page_size: number;
    }
    const { data } = await http.get<Resp>(url);
    return {
      items: (data.items || []).map(mapBackendLog),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  /** 取单 license 最近 N 条审计日志（详情抽屉 / 跳转过滤用，limit 默认 10，上限 100）。 */
  async getLicenseLogsByLicense(licenseId: string, limit = 10): Promise<LicenseLog[]> {
    interface Resp {
      items: BackendLicenseLog[];
      total: number;
    }
    const { data } = await http.get<Resp>(`/licenses/${licenseId}/logs`, {
      params: { limit },
    });
    return (data.items || []).map(mapBackendLog);
  },

  /**
   * 导入 license（T-0100-P3）。
   *
   * 返回包含签名状态的 ImportLicenseResult；前端在导入成功 Modal 上根据
   * `signatureStatus` 决定是否显示 "未签名校验" warning Tag（Q4=B 决议：
   * MVP 阶段允许 'unverified' 入库，P4-C 收紧为强校验）。
   */
  async importLicense(licenseData: Omit<License, 'id'>): Promise<ImportLicenseResult> {
    const payload = {
      license_name: licenseData.licenseName,
      license_code: licenseData.licenseCode,
      product_name: licenseData.productName,
      license_type: licenseData.licenseType,
      status: licenseData.status,
      max_devices: licenseData.maxDevices,
      used_devices: licenseData.usedDevices,
      features: licenseData.features,
      issue_date: licenseData.issueDate,
      expiry_date: licenseData.expiryDate,
      licensor: licenseData.licensor || null,
      device_type: licenseData.deviceType || null,
      region: licenseData.region || null,
      notes: licenseData.notes || null,
    };
    const { data } = await http.post<BackendImportResponse>('/licenses/import', payload);
    return {
      license: mapBackendLicense(data.license),
      signatureStatus: data.signature_status,
      signatureNote: data.signature_note ?? '',
    };
  },

  /**
   * 上传签名 license 文件并导入（T-0100-P5-c）。
   *
   * 工作流（PRD §5.3.1）：
   *   1. 前端 FileReader 读 .lic / .json 文件原始内容
   *   2. POST /licenses/import 时附带 signed_license_json 字段（后端在 verifier
   *      注入时会做 RSA-PSS 验签 + canonicalize）
   *   3. strict 模式下若验签失败 → 后端返 12109 错误 → 调用方 catch 后解析错误码
   *
   * 文件内容期望是一个完整 JSON 对象，至少含：license_code / signature / key_id
   * 等字段；handler.go::Import 会从 SignedLicenseJSON 解构出 license 元数据。
   */
  async importSignedLicense(signedJSON: string): Promise<ImportLicenseResult> {
    const payload = {
      signed_license_json: signedJSON,
    };
    const { data } = await http.post<BackendImportResponse>('/licenses/import', payload);
    return {
      license: mapBackendLicense(data.license),
      signatureStatus: data.signature_status,
      signatureNote: data.signature_note ?? '',
    };
  },
};

/**
 * License 错误码（与 omcgo/global/errors.go 12100-12109 段保持同步）。
 *
 * 前端用于区分签名校验失败（12109）等专属错误，给出针对性 UI 文案。
 */
export const LicenseErrorCodes = {
  RevokedActivate: 12100,
  QuotaLoad: 12103,
  LogsServiceUnavail: 12104,
  LogsListFailed: 12105,
  LogsByIDFailed: 12106,
  ExportFormatInvalid: 12107,
  BulkExportFormatInvalid: 12108,
  SignatureVerifyFailed: 12109,
} as const;

/**
 * 从 axios error 提取后端 BusinessError.Code（如有）。
 * 后端 response.Fail 输出格式：{ code: <number>, msg: "...", biz_code: <number> }
 * 通过 biz_code 字段提取错误码；提取失败返 0。
 */
export function extractLicenseErrorCode(err: unknown): number {
  if (typeof err !== 'object' || err === null) return 0;
  const e = err as { response?: { data?: { biz_code?: number; code?: number } } };
  return e.response?.data?.biz_code ?? e.response?.data?.code ?? 0;
}

/**
 * 从 Content-Disposition 头解析 filename。
 *
 * 支持两种格式：
 *   - filename="xxx.csv"
 *   - filename*=UTF-8''xxx.csv（RFC 5987）
 * 解析失败用 fallback。
 */
function parseFilename(cd: string, fallback: string): string {
  const m1 = /filename\*=UTF-8''([^;]+)/i.exec(cd);
  if (m1) {
    try { return decodeURIComponent(m1[1]); } catch { /* fall through */ }
  }
  const m2 = /filename="?([^";]+)"?/i.exec(cd);
  if (m2) return m2[1];
  return fallback;
}

/**
 * 把 axios error 解析为同维度冲突信息（若是）。
 *
 * 用于 LicenseOperations 的 Activate Tab：activateLicense 抛 409 时调用方
 * 取 error.response?.data 传给本函数；非冲突错误返 null（继续按通用错误处理）。
 */
export function parseActivateConflict(payload: unknown): ActivateConflict | null {
  if (typeof payload !== 'object' || payload === null) return null;
  const body = payload as Partial<ActivateConflictBody>;
  if (body.conflict !== 'same_dimension_active' || !Array.isArray(body.conflicting_active)) {
    return null;
  }
  return {
    conflict: 'same_dimension_active',
    deviceType: body.device_type ?? null,
    region: body.region ?? null,
    conflictingActive: body.conflicting_active.map(mapBackendLicense),
    hint: body.hint ?? '',
  };
}
