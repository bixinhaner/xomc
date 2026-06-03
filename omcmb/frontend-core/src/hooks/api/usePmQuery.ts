/**
 * T-0174 指标查询页 React Query Hooks。
 *
 * - 模板 CRUD 走 pmQueryApi。
 * - 聚合数据查询走 pmDashboardApi.queryAggregated（已存在，多设备由组件层多次 query 合并）。
 */

import { useQuery, useMutation, useQueryClient, useQueries } from '@tanstack/react-query';
import { pmQueryApi } from '../../services/api/pmQueryApi';
import { pmDashboardApi } from '../../services/api/pmDashboardApi';
import { pmObjectsApi, pmObjectsMock } from '../../services/api/pmObjectsApi';
import { createApiSwitch } from '../../services/apiSwitch';
import type {
  ListTemplateParams,
  CreateTemplateInput,
  UpdateTemplateInput,
} from '../../types/pmQuery';
import type { AggregatedQueryParams, AggregatedRow } from '../../types/pmDashboard';
import type { MetricObject } from '../../types/pmObject';

const objectsApi = createApiSwitch(pmObjectsMock, pmObjectsApi);

const KEY = ['pm-query-templates'] as const;

export function useQueryTemplates(params?: ListTemplateParams) {
  return useQuery({
    queryKey: [...KEY, 'list', params ?? {}],
    queryFn: () => pmQueryApi.list(params),
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
  const data: AggregatedRow[] = queries.flatMap((q) => q.data?.rows ?? []);
  // T-0194：跨设备汇总真实总数；任一设备查询命中 limit（返回行数 < 该设备真实总数）即判截断。
  const total: number = queries.reduce((sum, q) => sum + (q.data?.total ?? 0), 0);
  const truncated = queries.some(
    (q) => q.data != null && q.data.total > q.data.rows.length,
  );
  const refetch = () => queries.forEach((q) => void q.refetch());
  return { data, total, truncated, isLoading, isFetching, isError, errors, refetch };
}

/**
 * T-0193：列出一批设备实际出现过的「小区+PLMN」清单（下钻选择器用）。
 * - enabled 仅当 deviceSns 非空（空设备不发请求，向后兼容「不下钻」语义）。
 * - 查询键层级式 ['pm','metricObjects',{deviceSns,technology}]——设备/制式变化即重取。
 */
export function useMetricObjects(deviceSns: string[], technology?: string) {
  return useQuery({
    queryKey: ['pm', 'metricObjects', { deviceSns, technology }] as const,
    queryFn: () => objectsApi.listMetricObjects(deviceSns, technology),
    enabled: deviceSns.length > 0,
    staleTime: 30_000,
  });
}

/**
 * T-0193 下钻选择器用：按设备分别取小区清单（列小区接口本身不带 device_sn，
 * 所以两层「设备→小区」必须每设备一查），合并成 Record<deviceSn, MetricObject[]>。
 * 单设备并发 N 次（N=选中设备数，通常 1~10）。
 */
export function useMetricObjectsByDevices(deviceSns: string[], technology?: string) {
  const queries = useQueries({
    queries: deviceSns.map((sn) => ({
      queryKey: ['pm', 'metricObjects', { deviceSns: [sn], technology }] as const,
      queryFn: () => objectsApi.listMetricObjects([sn], technology),
      enabled: deviceSns.length > 0,
      staleTime: 30_000,
    })),
  });
  const isLoading = queries.some((q) => q.isLoading);
  const isFetching = queries.some((q) => q.isFetching);
  const byDevice: Record<string, MetricObject[]> = {};
  deviceSns.forEach((sn, i) => {
    byDevice[sn] = queries[i]?.data ?? [];
  });
  return { byDevice, isLoading, isFetching };
}
