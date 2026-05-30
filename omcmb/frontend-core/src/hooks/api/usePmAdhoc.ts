/**
 * T-0164-P7 / G7 adhoc 任务 React Query hooks。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createApiSwitch } from '../../services/apiSwitch';
import { pmAdhocApi, pmAdhocMock } from '../../services/api/pmAdhocApi';
import type { CreateAdhocTaskInput } from '../../types/pmAdhoc';

const api = createApiSwitch(pmAdhocMock, pmAdhocApi);

const ADHOC_KEY = ['pm-adhoc-tasks'] as const;

export function usePmAdhocList(opts?: { refetchInterval?: number; isBuiltin?: boolean }) {
  return useQuery({
    // 查询键带 isBuiltin，内置区/自建区两次调用各自独立缓存（T-0186）
    queryKey: [...ADHOC_KEY, 'list', { isBuiltin: opts?.isBuiltin ?? null }],
    queryFn: () =>
      api.list(opts?.isBuiltin === undefined ? undefined : { isBuiltin: opts.isBuiltin }),
    refetchInterval: opts?.refetchInterval,
  });
}

export function usePmAdhocDetail(id: string | undefined) {
  return useQuery({
    queryKey: [...ADHOC_KEY, 'detail', id],
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

export function useCancelPmAdhoc() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancel(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ADHOC_KEY });
    },
  });
}

export function usePmAdhocResults(
  taskId: string | undefined,
  opts?: { limit?: number; startTime?: string; endTime?: string },
) {
  // T-0187 仪表盘自动出图要取更多结果行（默认 100 保持 AdhocResultPanel 现状）。
  const limit = opts?.limit ?? 100;
  // T-0189 大时间段驱动取数：startTime/endTime 透传后端 + 并入 queryKey（窗口变化即重取）。
  const startTime = opts?.startTime;
  const endTime = opts?.endTime;
  return useQuery({
    queryKey: [...ADHOC_KEY, 'results', taskId, { limit, startTime, endTime }],
    queryFn: () => api.results(taskId as string, limit, 0, startTime, endTime),
    enabled: Boolean(taskId),
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
