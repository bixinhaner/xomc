import http from '../http';
import type { License } from '../../mock/data/license';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---------------------------------------------------------------------------
// License audit logs (T-0100-P1)
// 与后端 internal/license/license_log_model.go 保持一致：9 种 log_type + 4 种 result。
// ---------------------------------------------------------------------------

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

  async getLicenseSummary(): Promise<{
    total: number;
    active: number;
    expired: number;
    pending: number;
    expiringSoon: number;
  }> {
    const { data } = await http.get<BackendLicenseSummary>('/licenses/summary');
    return {
      total: data.total,
      active: data.active,
      expired: data.expired,
      pending: data.pending,
      expiringSoon: data.expiring_soon,
    };
  },

  async activateLicense(licenseCode: string): Promise<License> {
    const { data } = await http.post<BackendLicense>('/licenses/activate', {
      license_code: licenseCode,
    });
    return mapBackendLicense(data);
  },

  async revokeLicense(id: string): Promise<void> {
    await http.post(`/licenses/${id}/revoke`);
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

  async importLicense(licenseData: Omit<License, 'id'>): Promise<License> {
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
    const { data } = await http.post<BackendLicense>('/licenses/import', payload);
    return mapBackendLicense(data);
  },
};
