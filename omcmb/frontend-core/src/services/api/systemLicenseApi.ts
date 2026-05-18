// systemLicenseApi.ts — F06 System License 重构 Step 4 前端业务层。
//
// 后端 PRD: docs/project/prd/F06-system-license-redesign.md §5 API 契约
// 后端实现: omcgo/internal/license/system_license_handler.go (Step 2 commit 50840622)
//
// 与 licenseApi.ts 完全并存——本文件只消费新 singleton 接口：
//
//   GET  /api/v1/system-license          → SystemLicense
//   POST /api/v1/system-license          → UpdateResult{current, replaced}
//   GET  /api/v1/system-license/history  → PageResponse<SystemLicenseHistory>
//
// 老 multi-license API（Import/Activate/Revoke/Logs/Export）仍在 licenseApi.ts
// 上提供给 webcode-v2 等老 UI 壳，Step 5 一并下线。
import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';

export type SystemLicenseType = 'Commercial' | 'Trial' | 'Evaluation' | 'Internal';
export type SystemLicenseSignatureStatus = 'verified' | 'unverified' | 'invalid';

/** 设备类型 → 容量配额。后端 JSONB column。 */
export type DevicesSupport = Record<string, number>;

/**
 * feature_list 是后端 json.RawMessage（三级嵌套：string "All" / string[] / Record<string, string[]>）。
 * 前端展示用 `unknown` + 类型保护，不在 API 层强类型化（结构灵活性优先）。
 */
export type FeatureList = Record<string, unknown>;

/** 当前生效 license。 */
export interface SystemLicense {
  id: string;
  licenseId: string;
  licenseType: SystemLicenseType;
  issuer: string | null;
  licensee: string | null;
  issuedAt: string;
  expiryDate: string | null;
  devicesSupport: DevicesSupport;
  featureList: FeatureList;
  rawContent: string;
  signature: string | null;
  signatureKeyId: string | null;
  signatureStatus: SystemLicenseSignatureStatus;
  uploadedAt: string;
  uploadedByUserId: string | null;
  isCurrent: boolean;
  createdAt: string;
  updatedAt: string;
}

/** 历史 license（每次 Update 保留的归档）。 */
export interface SystemLicenseHistory {
  id: string;
  licenseId: string;
  licenseType: SystemLicenseType;
  issuer: string | null;
  licensee: string | null;
  issuedAt: string;
  expiryDate: string | null;
  devicesSupport: DevicesSupport;
  featureList: FeatureList;
  rawContent: string;
  signature: string | null;
  signatureKeyId: string | null;
  signatureStatus: SystemLicenseSignatureStatus;
  uploadedAt: string;
  uploadedByUserId: string | null;
  replacedAt: string;
  replacedById: string | null;
  createdAt: string;
}

/** POST /system-license 响应体（UpdateResult）。 */
export interface SystemLicenseUpdateResult {
  current: SystemLicense;
  replaced: SystemLicenseHistory | null;
}

// ---- Backend snake_case mirrors（response interceptor 不做自动转换，需要手动 map） ----

interface BackendSystemLicense {
  id: string;
  license_id: string;
  license_type: SystemLicenseType;
  issuer: string | null;
  licensee: string | null;
  issued_at: string;
  expiry_date: string | null;
  devices_support: DevicesSupport;
  feature_list: FeatureList;
  raw_content: string;
  signature: string | null;
  signature_key_id: string | null;
  signature_status: SystemLicenseSignatureStatus;
  uploaded_at: string;
  uploaded_by_user_id: string | null;
  is_current: boolean;
  created_at: string;
  updated_at: string;
}

interface BackendSystemLicenseHistory {
  id: string;
  license_id: string;
  license_type: SystemLicenseType;
  issuer: string | null;
  licensee: string | null;
  issued_at: string;
  expiry_date: string | null;
  devices_support: DevicesSupport;
  feature_list: FeatureList;
  raw_content: string;
  signature: string | null;
  signature_key_id: string | null;
  signature_status: SystemLicenseSignatureStatus;
  uploaded_at: string;
  uploaded_by_user_id: string | null;
  replaced_at: string;
  replaced_by_id: string | null;
  created_at: string;
}

interface BackendUpdateResult {
  current: BackendSystemLicense;
  replaced: BackendSystemLicenseHistory | null;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page?: number;
  page_size?: number;
}

function mapSystemLicense(b: BackendSystemLicense): SystemLicense {
  return {
    id: b.id,
    licenseId: b.license_id,
    licenseType: b.license_type,
    issuer: b.issuer,
    licensee: b.licensee,
    issuedAt: b.issued_at,
    expiryDate: b.expiry_date,
    devicesSupport: b.devices_support ?? {},
    featureList: b.feature_list ?? {},
    rawContent: b.raw_content,
    signature: b.signature,
    signatureKeyId: b.signature_key_id,
    signatureStatus: b.signature_status,
    uploadedAt: b.uploaded_at,
    uploadedByUserId: b.uploaded_by_user_id,
    isCurrent: b.is_current,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapSystemLicenseHistory(b: BackendSystemLicenseHistory): SystemLicenseHistory {
  return {
    id: b.id,
    licenseId: b.license_id,
    licenseType: b.license_type,
    issuer: b.issuer,
    licensee: b.licensee,
    issuedAt: b.issued_at,
    expiryDate: b.expiry_date,
    devicesSupport: b.devices_support ?? {},
    featureList: b.feature_list ?? {},
    rawContent: b.raw_content,
    signature: b.signature,
    signatureKeyId: b.signature_key_id,
    signatureStatus: b.signature_status,
    uploadedAt: b.uploaded_at,
    uploadedByUserId: b.uploaded_by_user_id,
    replacedAt: b.replaced_at,
    replacedById: b.replaced_by_id,
    createdAt: b.created_at,
  };
}

/** 历史列表查询参数（可选 license_id 过滤）。 */
export interface SystemLicenseHistoryQuery extends PageRequest {
  licenseId?: string;
}

export const systemLicenseApi = {
  /**
   * 获取当前生效 license。
   * 后端 404 + 业务码 12113 表示 system_license 表空——前端可据此显示空态。
   */
  async getCurrent(): Promise<SystemLicense> {
    const { data } = await http.get<BackendSystemLicense>('/system-license');
    return mapSystemLicense(data);
  },

  /**
   * 上传新 license 覆盖当前。
   *
   * @param rawContent license JSON 文件原始字符串（前端 FileReader 读出）
   * 后端业务错误（前端按 biz_code 区分）：
   *   - 12109 签名校验失败（strict 模式）
   *   - 12110 license_id 已存在
   *   - 12111 JSON 格式错误 / 必填字段缺失
   */
  async update(rawContent: string): Promise<SystemLicenseUpdateResult> {
    const { data } = await http.post<BackendUpdateResult>('/system-license', {
      raw_content: rawContent,
    });
    return {
      current: mapSystemLicense(data.current),
      replaced: data.replaced ? mapSystemLicenseHistory(data.replaced) : null,
    };
  },

  /** 历史列表（按 replaced_at DESC 排序）。 */
  async listHistory(params: SystemLicenseHistoryQuery): Promise<PageResponse<SystemLicenseHistory>> {
    const { data } = await http.get<BackendListResponse<BackendSystemLicenseHistory>>(
      '/system-license/history',
      { params },
    );
    return {
      items: (data.items ?? []).map(mapSystemLicenseHistory),
      total: data.total ?? 0,
      page: data.page ?? params.page,
      pageSize: data.page_size ?? params.pageSize,
    };
  },
};

/** 后端业务错误码（与 omcgo/global/errors.go 12110-12113 段对齐）。 */
export const SystemLicenseErrorCodes = {
  SignatureVerifyFailed: 12109,
  IDExists: 12110,
  InvalidFormat: 12111,
  Downgrade: 12112,
  NotConfigured: 12113,
} as const;
