// useDeviceLicenseParams.ts — DeviceDetail "License 参数" tab React Query hooks。
//
// 行为约定（PRD Q3 手动刷新）：
//   - useDeviceLicenseParams(deviceId)：拉列表
//   - useRefreshDeviceLicenseParams()：POST 下发 GPV
//   - 刷新成功后**不自动**重拉列表 — 由调用方在 onSuccess 里手动 refetch（user
//     可选择立即拉一次"当前 DB 值"，但因 CPE 响应有延迟，新值需用户再点一次
//     刷新才会看到，这是 Q3 设计）
//   - 无 mock 模式（VITE_USE_MOCK=true 时 reject，与 useSystemLicense 一致）
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  deviceLicenseParamApi,
  type DeviceLicenseParamListResponse,
  type DeviceLicenseRefreshResult,
} from '../../services/api/deviceLicenseParamApi';
import { useMock } from '../../services/apiSwitch';

const MOCK_NOT_SUPPORTED =
  'device license params API does not support mock mode; toggle VITE_USE_MOCK=false';

/** GET 列表。deviceId 为空时禁用 query。 */
export function useDeviceLicenseParams(deviceId: string) {
  return useQuery<DeviceLicenseParamListResponse, Error>({
    queryKey: ['deviceLicenseParams', deviceId],
    queryFn: () => {
      if (useMock) return Promise.reject(new Error(MOCK_NOT_SUPPORTED));
      return deviceLicenseParamApi.list(deviceId);
    },
    enabled: Boolean(deviceId),
    retry: false, // 404 / 400 不重试（用户感知更快）
  });
}

/**
 * 触发后端刷新（异步 GPV）。
 *
 * Q3 设计：成功后只 invalidate query key 触发 refetch 一次拉当前 DB 值
 * （新值尚未回来，但保证用户立即看到的是"刚下发刷新时的最新 DB 状态"）。
 * 后端 GPV 响应到达 ~ 数秒后才会写入 device_parameters，用户需再点一次刷新
 * 才能看到完整新值（PRD Q3 显式约定，无自动轮询）。
 */
export function useRefreshDeviceLicenseParams(deviceId: string) {
  const queryClient = useQueryClient();
  return useMutation<DeviceLicenseRefreshResult, Error, void>({
    mutationFn: () => {
      if (useMock) return Promise.reject(new Error(MOCK_NOT_SUPPORTED));
      return deviceLicenseParamApi.refresh(deviceId);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['deviceLicenseParams', deviceId],
      });
    },
  });
}
