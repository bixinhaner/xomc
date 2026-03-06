import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { MRDeviceMapping } from '@/mock/data/mr';
import type { PageRequest } from '@/types/pagination';
import { mrService } from '@/mock/services/mrService';

export function useMRIndicators(params: PageRequest) {
  return useQuery({
    queryKey: ['mr', 'indicators', params],
    queryFn: () => mrService.getIndicators(params),
    staleTime: 10 * 60 * 1000,
  });
}

export function useAllMRIndicators() {
  return useQuery({
    queryKey: ['mr', 'indicators', 'all'],
    queryFn: () => mrService.getAllIndicators(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useMRMappings(
  params: { deviceSn?: string; enabled?: boolean } & PageRequest
) {
  return useQuery({
    queryKey: ['mr', 'mappings', params],
    queryFn: () => mrService.getMappings(params),
  });
}

export function useUpdateMRMapping() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<MRDeviceMapping> }) =>
      mrService.updateMapping(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mr', 'mappings'] });
    },
  });
}

export function useToggleMRMapping() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      mrService.toggleMapping(id, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mr', 'mappings'] });
    },
  });
}

export function useMRRecords(
  params: { deviceSn?: string; cellId?: string; timeRange?: [string, string] } & PageRequest
) {
  return useQuery({
    queryKey: ['mr', 'records', params],
    queryFn: () => mrService.getRecords(params),
  });
}

export function useExportMRData() {
  return useMutation({
    mutationFn: (params: { deviceSns: string[]; timeRange: [string, string] }) =>
      mrService.exportMRData(params),
  });
}

export function useMRIndicatorStats(indicatorCode: string, deviceSn?: string) {
  return useQuery({
    queryKey: ['mr', 'stats', indicatorCode, deviceSn],
    queryFn: () => mrService.getIndicatorStats(indicatorCode, deviceSn),
    enabled: Boolean(indicatorCode),
  });
}
