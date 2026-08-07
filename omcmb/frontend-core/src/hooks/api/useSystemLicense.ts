// useSystemLicense.ts — F06 System License 重构 Step 4 React Query Hooks。
//
// 与老 useLicense.ts 完全并存。本文件只暴露 singleton 模型的 3 个 hook：
//   - useSystemLicense()             当前 license
//   - useSystemLicenseHistory(p)     历史分页
//   - useUpdateSystemLicense()       上传新文件 mutation
//
// Mock 模式（VITE_USE_MOCK=true）当前不支持 — singleton API 是 P5 后才有的
// 后端能力，开发期请用 docker compose 起后端真接口（reject 让开发者立即看到原因）。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  systemLicenseApi,
  type SystemLicenseHistoryQuery,
  type SystemLicenseUpdateResult,
} from '../../services/api/systemLicenseApi';
import { useMock } from '../../services/apiSwitch';

const MOCK_NOT_SUPPORTED =
  'system_license API does not support mock mode; toggle VITE_USE_MOCK=false';

/** 拉取当前生效 license。404（biz_code 12113）需调用方自己识别空态。 */
export function useSystemLicense() {
  return useQuery({
    queryKey: ['systemLicense', 'current'],
    queryFn: () => {
      if (useMock) return Promise.reject(new Error(MOCK_NOT_SUPPORTED));
      return systemLicenseApi.getCurrent();
    },
    // 空态（404 12113）不要走 React Query 内置 retry，避免开发者等 3 次
    retry: false,
  });
}

/** 历史 license 分页。无 license_id 过滤 = 全部。 */
export function useSystemLicenseHistory(params: SystemLicenseHistoryQuery) {
  return useQuery({
    queryKey: ['systemLicense', 'history', params],
    queryFn: () => {
      if (useMock) return Promise.reject(new Error(MOCK_NOT_SUPPORTED));
      return systemLicenseApi.listHistory(params);
    },
    placeholderData: (prev) => prev, // 翻页不闪
  });
}

/**
 * 上传新 license 文件覆盖当前。
 *
 * 入参是旧项目 .lic 二进制文件的 Base64 字符串。成功后自动 invalidate
 * current + history 查询，让页面拉到新数据。错误由调用方在 onError 里按
 * SystemLicenseErrorCodes 区分（12109 签名失败 / 12110 ID 已存在 / 12111 格式错）。
 */
export interface SystemLicenseUpload {
  rawContent: string;
}

export function useUpdateSystemLicense() {
  const queryClient = useQueryClient();
  return useMutation<SystemLicenseUpdateResult, Error, SystemLicenseUpload>({
    mutationFn: (upload) => {
      if (useMock) return Promise.reject(new Error(MOCK_NOT_SUPPORTED));
      return systemLicenseApi.update(upload.rawContent);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['systemLicense'] });
    },
  });
}
