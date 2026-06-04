/**
 * KPI-EXPORT（KPI 数据导出）React Query hooks。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §6.1
 *   - useKpiExportTasks：任务管理 Tab 列表，自动轮询——只要还有 pending/running 任务就定时重取，
 *     让用户看到状态由"进行中 → 成功/失败"刷新；全部进终态后停轮询省请求。
 *   - useKpiExportFiles：文件管理 Tab 列表（只列已成功且文件就绪）。
 *   - 建任务 / 删除 / 失败重试 mutation，成功后失效列表缓存。
 */

import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryKey,
} from '@tanstack/react-query';
import { createApiSwitch } from '../../services/apiSwitch';
import { kpiExportApi, kpiExportMock } from '../../services/api/kpiExportApi';
import type { KpiExportListFilter } from '../../services/api/kpiExportApi';
import type { KpiExportTask, CreateKpiExportInput } from '../../types/kpiExport';

const api = createApiSwitch(kpiExportMock, kpiExportApi);

const KPI_EXPORT_KEY = ['pm-kpi-export'] as const;

/** 列表里还有未到终态的任务时，按此间隔轮询（毫秒）。 */
const POLL_INTERVAL_MS = 4000;

/** 任意任务处于 pending/running 即视为"跑动中"，需要继续轮询。 */
function hasInflight(tasks: KpiExportTask[] | undefined): boolean {
  if (!tasks) return false;
  return tasks.some((t) => t.status === 'pending' || t.status === 'running');
}

/** 任务管理 Tab：列导出任务（含各状态），跑动中自动轮询刷新。 */
export function useKpiExportTasks(filter?: KpiExportListFilter) {
  return useQuery({
    queryKey: [...KPI_EXPORT_KEY, 'tasks', filter ?? null],
    queryFn: () => api.listTasks(filter),
    // 列表里有 pending/running 就轮询，全终态后返回 false 停轮询。
    refetchInterval: (query) =>
      hasInflight(query.state.data as KpiExportTask[] | undefined)
        ? POLL_INTERVAL_MS
        : false,
  });
}

/** 文件管理 Tab：列已成功且文件就绪的导出文件。 */
export function useKpiExportFiles(filter?: KpiExportListFilter) {
  return useQuery({
    queryKey: [...KPI_EXPORT_KEY, 'files', filter ?? null],
    queryFn: () => api.listFiles(filter),
  });
}

function useInvalidateAll() {
  const qc = useQueryClient();
  return () =>
    qc.invalidateQueries({
      predicate: (q) => (q.queryKey as QueryKey)[0] === KPI_EXPORT_KEY[0],
    });
}

/** 建导出任务。 */
export function useCreateKpiExport() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (input: CreateKpiExportInput) => api.create(input),
    onSuccess: () => invalidate(),
  });
}

/** 删除导出任务。 */
export function useDeleteKpiExport() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (id: string) => api.remove(id),
    onSuccess: () => invalidate(),
  });
}

/** 失败重试（用原来源 + 参数重建任务）。 */
export function useRetryKpiExport() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (task: KpiExportTask) => api.retry(task),
    onSuccess: () => invalidate(),
  });
}
