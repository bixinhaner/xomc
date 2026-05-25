/**
 * T-0164-P7 / G7 adhoc 任务 React Query hooks。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createApiSwitch } from '../../services/apiSwitch';
import { pmAdhocApi, pmAdhocMock } from '../../services/api/pmAdhocApi';
import type { CreateAdhocTaskInput } from '../../types/pmAdhoc';

const api = createApiSwitch(pmAdhocMock, pmAdhocApi);

const ADHOC_KEY = ['pm-adhoc-tasks'] as const;

export function usePmAdhocList(opts?: { refetchInterval?: number }) {
  return useQuery({
    queryKey: [...ADHOC_KEY, 'list'],
    queryFn: () => api.list(),
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

export function usePmAdhocResults(taskId: string | undefined) {
  return useQuery({
    queryKey: [...ADHOC_KEY, 'results', taskId],
    queryFn: () => api.results(taskId as string),
    enabled: Boolean(taskId),
  });
}
