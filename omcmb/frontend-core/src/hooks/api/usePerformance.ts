import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PerformanceThreshold, AggregatedCounterQuery, KPICalculationRequest } from '../../types/performance';
import type { PageRequest } from '../../types/pagination';
import type { Locale } from '../../types/common';
import { performanceService } from '../../mock/services/performanceService';
import { pmApi } from '../../services/api/pmApi';
import { useMock } from '../../services/apiSwitch';
import { useAppStore } from '../../store/appStore';

type KPIListParams = { keyword?: string } & PageRequest;

export const performanceKpiQueryKeys = {
  list: (params: KPIListParams, locale: Locale) =>
    ['performance', 'kpis', params, locale] as const,
  all: (locale: Locale) => ['performance', 'kpis', 'all', locale] as const,
  candidates: (deviceType: string | undefined, includeCounters: boolean, locale: Locale) =>
    ['performance', 'indicator-candidates', deviceType, { includeCounters }, locale] as const,
};

export function useKPIList(params: KPIListParams) {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: performanceKpiQueryKeys.list(params, locale),
    queryFn: () =>
      useMock ? performanceService.getKPIs(params) : pmApi.getKPIs(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAllKPIs() {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: performanceKpiQueryKeys.all(locale),
    queryFn: () =>
      useMock ? performanceService.getAllKPIs() : pmApi.getAllKPIs(),
    staleTime: 10 * 60 * 1000,
  });
}

// 向导第③步穿梭框：取指标候选清单（按制式，含计数器，带编号/类型）。
// 走 pm 权限的 /pm/kpi/definitions（运维可访问、全量无截断），不再用 super_admin 的 /indicators。
export function useIndicatorCandidates(
  deviceType: string | undefined,
  opts?: { includeCounters?: boolean }
) {
  const includeCounters = opts?.includeCounters ?? true;
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: performanceKpiQueryKeys.candidates(deviceType, includeCounters, locale),
    queryFn: async () => {
      if (useMock) {
        // Mock 模式退化：把 mock KPI 列表映射为候选（无 counter 区分，统一当 KPI）。
        const kpis = await performanceService.getAllKPIs();
        return kpis.map((k) => ({
          id: k.kpiCode,
          name: k.kpiName,
          cnName: k.kpiName,
          enName: k.kpiCode,
          isCounter: false,
        }));
      }
      return pmApi.getIndicatorCandidates(deviceType as string, { includeCounters });
    },
    enabled: Boolean(deviceType),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCounters(params: PageRequest) {
  return useQuery({
    queryKey: ['performance', 'counters', params],
    queryFn: () =>
      useMock
        ? (performanceService.getCounters(params) as unknown as ReturnType<typeof pmApi.getCounters>)
        : pmApi.getCounters(params),
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

export function useAggregatedCounters(params?: AggregatedCounterQuery) {
  return useQuery({
    queryKey: ['performance', 'counters', 'aggregated', params],
    queryFn: () => pmApi.getAggregatedCounters(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCalculateKPI() {
  return useMutation({
    mutationFn: (params: KPICalculationRequest) =>
      pmApi.calculateKPI(params),
  });
}

// --- PM Files (File Management → PM Tab) ---

export function usePMFiles(
  params: { deviceSn?: string; timeRange?: [string, string] } & PageRequest,
) {
  return useQuery({
    queryKey: ['pm', 'files', params],
    queryFn: () => pmApi.getFiles(params),
    enabled: !useMock,
  });
}

// 按设备聚合的 PM 文件视图 —— File Management → PM Tab 主列表。
// 10s 轮询：reporting 字段依赖 last_collect_time，文件数也会随设备上传变化。
export function usePMFileDevices(
  params: { keyword?: string; siteName?: string; productClass?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['pm', 'files', 'devices', params],
    queryFn: () => pmApi.getFileDevices(params),
    enabled: !useMock,
    refetchInterval: 10_000,
  });
}

export function useDownloadPMFile() {
  return useMutation({
    mutationFn: (fileId: string) => pmApi.downloadFile(fileId),
  });
}

// 按设备 SN 批量删除 PM 文件（PG + MinIO）。成功后 invalidate 设备聚合列表。
export function useBatchDeletePMFiles() {
  const qc = useQueryClient();
  return useMutation<{ succeeded: string[]; failed: string[] }, Error, string[]>({
    mutationFn: (sns: string[]) => pmApi.batchDeleteFiles(sns),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['pm', 'files'] });
    },
  });
}
