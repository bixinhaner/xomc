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

export function useActivateLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (licenseCode: string) =>
      useMock
        ? licenseService.activate(licenseCode)
        : licenseApi.activateLicense(licenseCode),
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

export function useImportLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<License, 'id'>) =>
      useMock
        ? licenseService.importLicense(data)
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
