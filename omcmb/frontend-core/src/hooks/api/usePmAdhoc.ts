/**
 * T-0164-P7 / G7 adhoc 任务 React Query hooks。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createApiSwitch } from '../../services/apiSwitch';
import { pmAdhocApi, pmAdhocMock } from '../../services/api/pmAdhocApi';
import type {
  AdhocDimension,
  CreateAdhocTaskInput,
  UpdateAdhocTaskInput,
} from '../../types/pmAdhoc';
import { useAppStore } from '../../store/appStore';

const api = createApiSwitch(pmAdhocMock, pmAdhocApi);

const ADHOC_KEY = ['pm-adhoc-tasks'] as const;

export function usePmAdhocList(opts?: { refetchInterval?: number; isBuiltin?: boolean }) {
  // locale 并入查询键：切语言后内置任务名随后端本地化重取（pm-name-i18n）。
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    // 查询键带 isBuiltin，内置区/自建区两次调用各自独立缓存（T-0186）
    queryKey: [...ADHOC_KEY, 'list', { isBuiltin: opts?.isBuiltin ?? null, locale }],
    queryFn: () =>
      api.list(opts?.isBuiltin === undefined ? undefined : { isBuiltin: opts.isBuiltin }),
    refetchInterval: opts?.refetchInterval,
  });
}

export function usePmAdhocDetail(id: string | undefined) {
  // locale 并入查询键：切语言后内置任务名随后端本地化重取（pm-name-i18n）。
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: [...ADHOC_KEY, 'detail', id, locale],
    queryFn: () => api.get(id as string),
    enabled: Boolean(id),
  });
}

export function useCreatePmAdhoc() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateAdhocTaskInput) => api.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ADHOC_KEY });
    },
  });
}

export function useUpdatePmAdhoc() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateAdhocTaskInput }) =>
      api.update(id, input),
    onSuccess: () => {
      // 列表 + 详情 + 结果一并失效（编辑后详情/仪表盘要拿新指标集）
      void qc.invalidateQueries({ queryKey: ADHOC_KEY });
    },
  });
}

export function useCancelPmAdhoc() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancel(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ADHOC_KEY });
    },
  });
}

/**
 * issue #392：硬删终态自建任务的定义行。删除成功后失效自定义聚合任务列表使其刷新（任务消失）。
 * 仅对终态(成功/失败/已取消) + 自建任务调用；非终态/内置由 UI 不渲染删除按钮拦在前面，
 * 后端也会以 409/403 二次守门。
 */
export function useDeletePmAdhoc() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTask(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ADHOC_KEY });
    },
  });
}

export function usePmAdhocResults(
  taskId: string | undefined,
  opts?: {
    limit?: number;
    startTime?: string;
    endTime?: string;
    // PM-DASH-DIMFILTER 维度子集过滤：product 维度走 productIds，device_group/band 维度走 objectLdns。
    productIds?: string[];
    objectLdns?: string[];
    // #599：星期/小时段后端过滤（全选/空 = 不传 = 不过滤）。
    weekdays?: number[];
    hours?: number[];
  },
) {
  // T-0187 仪表盘自动出图要取更多结果行（默认 100 保持 AdhocResultPanel 现状）。
  const limit = opts?.limit ?? 100;
  // T-0189 大时间段驱动取数：startTime/endTime 透传后端 + 并入 queryKey（窗口变化即重取）。
  const startTime = opts?.startTime;
  const endTime = opts?.endTime;
  const productIds = opts?.productIds;
  const objectLdns = opts?.objectLdns;
  const weekdays = opts?.weekdays;
  const hours = opts?.hours;
  // locale 并入查询键：切语言后图表标题指标名随后端本地化重取（pm-name-i18n）。
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    // PM-DASH-DIMFILTER：productIds/objectLdns 并入查询键 → 子集变化即触发重查（B 的重查语义落点）。
    queryKey: [
      ...ADHOC_KEY,
      'results',
      taskId,
      { limit, startTime, endTime, productIds, objectLdns, weekdays, hours, locale },
    ],
    queryFn: () =>
      api.results(taskId as string, limit, 0, startTime, endTime, productIds, objectLdns, weekdays, hours),
    enabled: Boolean(taskId),
  });
}

/**
 * PM-DASH-DIMFILTER：列出本任务可筛选的维度子集选项（产品/设备组/频段）。
 * 仅当维度 ∈ {product, device_group, band} 时 enabled——其余维度（device/aggregate_group/network）
 * 不渲染筛选框，也就不发请求。
 */
const FILTERABLE_DIMENSIONS: ReadonlySet<AdhocDimension> = new Set<AdhocDimension>([
  'product',
  'device_group',
  'band',
]);

export function usePmAdhocFilterOptions(
  taskId: string | undefined,
  dimension: AdhocDimension | undefined,
) {
  // locale 并入查询键：切语言后（若后端可读名本地化）随之重取，与其余 hook 同口径。
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: [...ADHOC_KEY, 'filterOptions', taskId, locale],
    queryFn: () => api.filterOptions(taskId as string),
    enabled: Boolean(taskId) && Boolean(dimension) && FILTERABLE_DIMENSIONS.has(dimension!),
  });
}

export function usePmAdhocRuns(
  taskId: string | undefined,
  opts?: { refetchInterval?: number },
) {
  return useQuery({
    queryKey: [...ADHOC_KEY, 'runs', taskId],
    queryFn: () => api.runs(taskId as string),
    enabled: Boolean(taskId),
    refetchInterval: opts?.refetchInterval,
  });
}
