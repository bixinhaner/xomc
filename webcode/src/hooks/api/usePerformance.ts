import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PerformanceThreshold } from '@/types/performance';
import type { PageRequest } from '@/types/pagination';
import { performanceService } from '@/mock/services/performanceService';

export function useKPIList(params: { keyword?: string } & PageRequest) {
  return useQuery({
    queryKey: ['performance', 'kpis', params],
    queryFn: () => performanceService.getKPIs(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAllKPIs() {
  return useQuery({
    queryKey: ['performance', 'kpis', 'all'],
    queryFn: () => performanceService.getAllKPIs(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useCounters(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'counters', params],
    queryFn: () => performanceService.getCounters(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useMeasurements(
  params: { deviceSn?: string; kpiCode?: string; granularity?: string; timeRange?: [string, string] } & PageRequest
) {
  return useQuery({
    queryKey: ['performance', 'measurements', params],
    queryFn: () => performanceService.getMeasurements(params),
  });
}

export function useKPISeries(kpiCode: string, deviceSn?: string, days = 7) {
  return useQuery({
    queryKey: ['performance', 'series', kpiCode, deviceSn, days],
    queryFn: () => performanceService.getKPISeries(kpiCode, deviceSn, days),
    enabled: Boolean(kpiCode),
  });
}

export function useMultipleKPISeries(kpiCodes: string[], deviceSn?: string) {
  return useQuery({
    queryKey: ['performance', 'multi-series', kpiCodes, deviceSn],
    queryFn: () => performanceService.getMultipleKPISeries(kpiCodes, deviceSn),
    enabled: kpiCodes.length > 0,
  });
}

export function useThresholds(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'thresholds', params],
    queryFn: () => performanceService.getThresholds(params),
  });
}

export function useCreateThreshold() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<PerformanceThreshold, 'id' | 'createTime' | 'updateTime'>) =>
      performanceService.createThreshold(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'thresholds'] });
    },
  });
}

export function useUpdateThreshold() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<PerformanceThreshold> }) =>
      performanceService.updateThreshold(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'thresholds'] });
    },
  });
}

export function useDeleteThresholds() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => performanceService.deleteThresholds(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'thresholds'] });
    },
  });
}

export function usePerformanceTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'tasks', params],
    queryFn: () => performanceService.getTasks(params),
  });
}

export function useCreatePerformanceTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof performanceService.createTask>[0]) =>
      performanceService.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'tasks'] });
    },
  });
}
