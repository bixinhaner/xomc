// systemLicenseApi.ts — F06 System License 重构 Step 4 前端业务层。
//
// 后端 PRD: docs/project/prd/F06-system-license-redesign.md §5 API 契约
// 后端实现: omcgo/internal/license/system_license_handler.go (Step 2 commit 50840622)
//
// 与历史页面接口并存——本文件提供旧项目 License 的当前状态、历史和限制查询：
//
//   GET  /api/v1/system-license          → SystemLicense
//   POST /api/v1/system-license          → UpdateResult{current, replaced}
//   GET  /api/v1/system-license/history  → PageResponse<SystemLicenseHistory>
//
// 老 multi-license API（Import/Activate/Revoke/Logs/Export）仍在 licenseApi.ts
// 上提供给已移除的老 UI 壳，Step 5 一并下线。
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
export interface SystemLicenseFeature {
  featureId?: string;
  featureCode?: string;
  nameZh: string;
  nameEn: string;
  path?: string;
  pathEn?: string;
  source: string;
  recognized: boolean;
  licensed: boolean;
  rawIds?: string[];
  rawCodes?: string[];
}

export type FeatureList = Record<string, unknown> & {
  features?: SystemLicenseFeature[];
  legacy_feature_ids?: string[];
  legacy_feature_codes?: string[];
};

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

export interface SystemLicenseFeatureCheck {
  path: string;
  authorized: boolean;
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

interface BackendSystemLicenseFeature {
  feature_id?: string;
  feature_code?: string;
  name_zh: string;
  name_en: string;
  path?: string;
  path_en?: string;
  source: string;
  recognized: boolean;
  licensed: boolean;
  raw_ids?: string[];
  raw_codes?: string[];
}

function mapFeatureList(value: FeatureList): FeatureList {
  if (!value || typeof value !== 'object') return {};
  const features = Array.isArray(value.features)
    ? value.features.map((feature) => {
        const backend = feature as unknown as BackendSystemLicenseFeature;
        return {
          featureId: backend.feature_id,
          featureCode: backend.feature_code,
          nameZh: backend.name_zh ?? '',
          nameEn: backend.name_en ?? '',
          path: backend.path,
          pathEn: backend.path_en,
          source: backend.source ?? 'unknown',
          recognized: backend.recognized === true,
          licensed: backend.licensed === true,
          rawIds: backend.raw_ids ?? [],
          rawCodes: backend.raw_codes ?? [],
        } satisfies SystemLicenseFeature;
      })
    : undefined;
  return features ? { ...value, features } : value;
}

/** Decode the stored Base64 representation back to the original .lic bytes. */
export function decodeSystemLicenseRawContent(rawContent: string): Uint8Array {
  const binary = atob(rawContent);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
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
    featureList: mapFeatureList(b.feature_list ?? {}),
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
    featureList: mapFeatureList(b.feature_list ?? {}),
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
  * @param rawContent 旧项目 .lic 二进制文件的 Base64 原文
   * 后端业务错误（前端按 biz_code 区分）：
   *   - 12109 签名校验失败（strict 模式）
   *   - 12110 license_id 已存在
   *   - 12111 旧项目 .lic 格式错误 / 解密或验签失败
   */
  async update(rawContent: string): Promise<SystemLicenseUpdateResult> {
    const { data } = await http.post<BackendUpdateResult>('/system-license', {
      raw_content: rawContent,
      raw_content_encoding: 'base64',
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

  /** 查询当前 License 是否授权指定的三级功能路径。 */
  async checkFeature(path: string): Promise<SystemLicenseFeatureCheck> {
    const { data } = await http.get<SystemLicenseFeatureCheck>('/system-license/feature-check', {
      params: { path },
    });
    return data;
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

// ---------------------------------------------------------------------------
// 错误码提取（Step 5 从老 licenseApi.ts 迁入；老文件已删除）
// ---------------------------------------------------------------------------

/**
 * 从 Axios error / Error / 后端业务错误里抽出 biz_code。
 *
 * 兼容：
 *   - Error & { bizCode: number }       (http interceptor 抛出的业务失败)
 *   - Axios error: error.response.data.biz_code
 *   - 其他形式 → 返 0（unknown）
 */
export function extractLicenseErrorCode(err: unknown): number {
  if (!err || typeof err !== 'object') return 0;
  const e = err as Record<string, unknown>;
  if (typeof e.bizCode === 'number') return e.bizCode;
  const resp = e.response as Record<string, unknown> | undefined;
  if (resp && typeof resp === 'object') {
    const data = resp.data as Record<string, unknown> | undefined;
    if (data && typeof data === 'object') {
      if (typeof data.biz_code === 'number') return data.biz_code;
      if (typeof data.bizCode === 'number') return data.bizCode as number;
    }
  }
  return 0;
}
