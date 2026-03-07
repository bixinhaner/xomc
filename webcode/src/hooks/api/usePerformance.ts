import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PerformanceThreshold } from '@/types/performance';
import type { PageRequest } from '@/types/pagination';
import { performanceService } from '@/mock/services/performanceService';
import { pmApi } from '@/services/api/pmApi';
import { useMock } from '@/services/apiSwitch';

export function useKPIList(params: { keyword?: string } & PageRequest) {
  return useQuery({
    queryKey: ['performance', 'kpis', params],
    queryFn: () =>
      useMock ? performanceService.getKPIs(params) : pmApi.getKPIs(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAllKPIs() {
  return useQuery({
    queryKey: ['performance', 'kpis', 'all'],
    queryFn: () =>
      useMock ? performanceService.getAllKPIs() : pmApi.getAllKPIs(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useCounters(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'counters', params],
    queryFn: () =>
      useMock ? performanceService.getCounters(params) : pmApi.getCounters(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useMeasurements(
  params: { deviceSn?: string; kpiCode?: string; granularity?: string; timeRange?: [string, string] } & PageRequest
) {
  return useQuery({
    queryKey: ['performance', 'measurements', params],
    queryFn: () =>
      useMock
        ? performanceService.getMeasurements(params)
        : pmApi.getMeasurements(params),
  });
}

export function useKPISeries(kpiCode: string, deviceSn?: string, days = 7) {
  return useQuery({
    queryKey: ['performance', 'series', kpiCode, deviceSn, days],
    queryFn: () =>
      useMock
        ? performanceService.getKPISeries(kpiCode, deviceSn, days)
        : pmApi.getKPISeries(kpiCode, deviceSn, days),
    enabled: Boolean(kpiCode),
  });
}

export function useMultipleKPISeries(kpiCodes: string[], deviceSn?: string) {
  return useQuery({
    queryKey: ['performance', 'multi-series', kpiCodes, deviceSn],
    queryFn: () =>
      useMock
        ? performanceService.getMultipleKPISeries(kpiCodes, deviceSn)
        : pmApi.getMultipleKPISeries(kpiCodes, deviceSn),
    enabled: kpiCodes.length > 0,
  });
}

export function useThresholds(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'thresholds', params],
    queryFn: () =>
      useMock ? performanceService.getThresholds(params) : pmApi.getThresholds(params),
  });
}

export function useCreateThreshold() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<PerformanceThreshold, 'id' | 'createTime' | 'updateTime'>) =>
      useMock ? performanceService.createThreshold(data) : pmApi.createThreshold(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'thresholds'] });
    },
  });
}

export function useUpdateThreshold() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<PerformanceThreshold> }) =>
      useMock ? performanceService.updateThreshold(id, data) : pmApi.updateThreshold(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'thresholds'] });
    },
  });
}

export function useDeleteThresholds() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? performanceService.deleteThresholds(ids) : pmApi.deleteThresholds(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'thresholds'] });
    },
  });
}

export function usePerformanceTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'tasks', params],
    queryFn: () =>
      useMock ? performanceService.getTasks(params) : pmApi.getTasks(params),
  });
}

export function useCreatePerformanceTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof performanceService.createTask>[0]) =>
      useMock ? performanceService.createTask(data) : pmApi.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['performance', 'tasks'] });
    },
  });
}

export function useAggregatedCounters(params?: any) {
  return useQuery({
    queryKey: ['performance', 'counters', 'aggregated', params],
    queryFn: () => pmApi.getAggregatedCounters(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCalculateKPI() {
  return useMutation({
    mutationFn: (params: { kpi_name: string; device_ids?: string[]; start_time?: string; end_time?: string }) =>
      pmApi.calculateKPI(params),
  });
}
