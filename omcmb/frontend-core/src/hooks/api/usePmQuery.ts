/**
 * T-0174 指标查询页 React Query Hooks。
 *
 * - 模板 CRUD 走 pmQueryApi。
 * - 聚合数据查询走 pmDashboardApi.queryAggregated（已存在，多设备由组件层多次 query 合并）。
 */

import { useQuery, useMutation, useQueryClient, useQueries } from '@tanstack/react-query';
import { pmQueryApi } from '../../services/api/pmQueryApi';
import { pmDashboardApi } from '../../services/api/pmDashboardApi';
import type {
  ListTemplateParams,
  CreateTemplateInput,
  UpdateTemplateInput,
} from '../../types/pmQuery';
import type { AggregatedQueryParams, AggregatedRow } from '../../types/pmDashboard';

const KEY = ['pm-query-templates'] as const;

export function useQueryTemplates(params?: ListTemplateParams) {
  return useQuery({
    queryKey: [...KEY, 'list', params ?? {}],
    queryFn: () => pmQueryApi.list(params),
  });
}

export function useQueryTemplate(id: string | undefined) {
  return useQuery({
    queryKey: [...KEY, 'detail', id],
    queryFn: () => pmQueryApi.get(id as string),
    enabled: Boolean(id),
  });
}

export function useCreateQueryTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTemplateInput) => pmQueryApi.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...KEY, 'list'] });
    },
  });
}

export function useUpdateQueryTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateTemplateInput }) =>
      pmQueryApi.update(id, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...KEY, 'list'] });
      void qc.invalidateQueries({ queryKey: [...KEY, 'detail', vars.id] });
    },
  });
}

export function useDeleteQueryTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => pmQueryApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...KEY, 'list'] });
    },
  });
}

// 多设备聚合查询：后端 /pm/metrics/aggregated 仅支持单 device_sn 过滤，
// 这里用 useQueries 并发 N 次再合并 — N <= 用户选中设备数（通常 1~10）。
export function useAggregatedMetricsByDevices(
  baseParams: Omit<AggregatedQueryParams, 'deviceSn'>,
  deviceSns: string[],
  enabled: boolean,
) {
  const queries = useQueries({
    queries: deviceSns.map((sn) => ({
      queryKey: ['pm-aggregated', { ...baseParams, deviceSn: sn }],
      queryFn: () => pmDashboardApi.queryAggregated({ ...baseParams, deviceSn: sn }),
      enabled,
      staleTime: 30_000,
    })),
  });
  const isLoading = queries.some((q) => q.isLoading);
  const isFetching = queries.some((q) => q.isFetching);
  const isError = queries.some((q) => q.isError);
  const errors = queries.map((q) => q.error).filter(Boolean);
  const data: AggregatedRow[] = queries.flatMap((q) => q.data ?? []);
  const refetch = () => queries.forEach((q) => void q.refetch());
  return { data, isLoading, isFetching, isError, errors, refetch };
}
