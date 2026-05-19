// deviceLicenseParamApi.ts — DeviceDetail "License 参数" tab API service。
//
// 后端契约：omcgo/internal/device/license_params_handler.go
//
//   GET  /api/v1/devices/{id}/license-params          → 列表
//   POST /api/v1/devices/{id}/license-params/refresh  → 下发 GPV 刷新（202）
//
// 设计：响应仅含 standardPath，不暴露 privatePath（厂商实现细节，用户决策）。
import http from '../http';

export interface DeviceLicenseParam {
  standardPath: string;
  value: string;
  dataType?: string;
  access?: string;
  lastUpdatedAt: string; // ISO8601
}

export interface DeviceLicenseParamListResponse {
  items: DeviceLicenseParam[];
  total: number;
}

export interface DeviceLicenseRefreshResult {
  /** GPV 任务批数 */
  taskCount: number;
  /** 去重后 partial path 数量（GPV target object 数） */
  prefixCount: number;
  /** sourceID 标签，固定 "manual_license_refresh" */
  reason: string;
}

// Backend mirrors（http interceptor 不做 snake↔camel 响应转换，需手工 mapper）

interface BackendDeviceLicenseParam {
  standardPath: string;
  value: string;
  dataType?: string;
  access?: string;
  lastUpdatedAt: string;
}

interface BackendDeviceLicenseParamListResp {
  items: BackendDeviceLicenseParam[];
  total: number;
}

interface BackendDeviceLicenseRefreshResp {
  taskCount: number;
  prefixCount: number;
  reason: string;
}

function mapItem(b: BackendDeviceLicenseParam): DeviceLicenseParam {
  return {
    standardPath: b.standardPath,
    value: b.value,
    dataType: b.dataType,
    access: b.access,
    lastUpdatedAt: b.lastUpdatedAt,
  };
}

export const deviceLicenseParamApi = {
  /**
   * 拉取设备 license 参数列表（standardPath + value）。
   *
   * 后端走 device_parameters ILIKE %LICENSE% + ParamRegistry Translator 反向翻译。
   * 设备从未上报 license 参数时返回空 items（前端展示空态 + 引导用户点刷新）。
   */
  async list(deviceId: string): Promise<DeviceLicenseParamListResponse> {
    const { data } = await http.get<BackendDeviceLicenseParamListResp>(
      `/devices/${deviceId}/license-params`,
    );
    return {
      items: (data.items ?? []).map(mapItem),
      total: data.total ?? 0,
    };
  },

  /**
   * 下发 GPV 任务，让 CPE 返回 license 子树最新值。
   *
   * 后端 30s Redis 防抖锁：连点 30s 内会返 409 + bizCode 1205（ErrCodeRuleTaskRunning）。
   * 接口立即返回 202（不等 CPE 响应），用户需"再次点刷新"或重新进入页面看新值。
   */
  async refresh(deviceId: string): Promise<DeviceLicenseRefreshResult> {
    const { data } = await http.post<BackendDeviceLicenseRefreshResp>(
      `/devices/${deviceId}/license-params/refresh`,
    );
    return {
      taskCount: data.taskCount ?? 0,
      prefixCount: data.prefixCount ?? 0,
      reason: data.reason ?? 'manual_license_refresh',
    };
  },
};
