import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { License } from '../../mock/data/license';
import type { PageRequest } from '../../types/pagination';
import { licenseService } from '../../mock/services/licenseService';
import { licenseApi } from '../../services/api/licenseApi';
import { useMock } from '../../services/apiSwitch';

export function useLicenses(
  params: { status?: string; licenseType?: string; deviceType?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['licenses', 'list', params],
    queryFn: () =>
      useMock
        ? licenseService.getList(params)
        : licenseApi.getLicenses(params),
  });
}

export function useLicenseById(id: string) {
  return useQuery({
    queryKey: ['licenses', 'detail', id],
    queryFn: () =>
      useMock
        ? licenseService.getById(id)
        : licenseApi.getLicenseById(id),
    enabled: Boolean(id),
  });
}

export function useLicenseSummary() {
  return useQuery({
    queryKey: ['licenses', 'summary'],
    queryFn: () =>
      useMock
        ? licenseService.getSummary()
        : licenseApi.getLicenseSummary(),
    refetchInterval: 60000,
  });
}

/**
 * 激活 license（T-0100-P3）。
 *
 * mutate 接受 { licenseCode, force? }：force=false 时若同 (device_type, region)
 * 已有 active license，后端返 409 + ActivateConflictBody，调用方在 onError
 * 里 parseActivateConflict(err.response.data) 弹 Modal 二次确认；用户确认后
 * 用 force=true 再调一次。
 */
export interface ActivateLicenseInput {
  licenseCode: string;
  force?: boolean;
}

export function useActivateLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ licenseCode, force }: ActivateLicenseInput) =>
      useMock
        ? licenseService.activate(licenseCode)
        : licenseApi.activateLicense(licenseCode, force ?? false),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useRevokeLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock
        ? licenseService.revoke(id)
        : licenseApi.revokeLicense(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

/**
 * 导入 license（T-0100-P3）。
 *
 * 真实 API 返回 ImportLicenseResult（含 signatureStatus）；mock 路径用 service
 * 返回 License（向下兼容），调用方需 instanceof check 或检查 license 字段。
 * 推荐：禁用 mock 模式或仅在后端联调时使用本 hook。
 */
export function useImportLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<License, 'id'>) =>
      useMock
        ? licenseService.importLicense(data).then((lic) => ({
            license: lic,
            signatureStatus: 'unverified' as const,
            signatureNote: 'mock mode: signature not verified',
          }))
        : licenseApi.importLicense(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

// ---------------------------------------------------------------------------
// T-0100-P1 audit logs
// 后端 GET /licenses/logs（分页+过滤）+ GET /licenses/:id/logs（单 license 最近 N 条）
// 没有 mock 实现 — 按现状治理层日志只在真后端展示，dev 用 docker compose 起后端跑通
// ---------------------------------------------------------------------------

import type { LicenseLogQuery } from '../../services/api/licenseApi';

/** 全量审计日志（LicenseLogs 主页用）。 */
export function useLicenseLogs(params: LicenseLogQuery) {
  return useQuery({
    queryKey: ['licenses', 'logs', params],
    queryFn: () => licenseApi.getLicenseLogs(params),
    placeholderData: (prev) => prev, // 翻页不闪
  });
}

/** 单 license 最近 N 条日志（详情抽屉用，P2 阶段消费）。 */
export function useLicenseLogsByLicense(licenseId: string, limit = 10) {
  return useQuery({
    queryKey: ['licenses', 'logs', 'by-license', licenseId, limit],
    queryFn: () => licenseApi.getLicenseLogsByLicense(licenseId, limit),
    enabled: Boolean(licenseId),
  });
}
