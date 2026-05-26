import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { MRDeviceMapping } from '../../mock/data/mr';
import type { PageRequest } from '../../types/pagination';
import { mrService } from '../../mock/services/mrService';
import { mrApi } from '../../services/api/mrApi';
import { useMock } from '../../services/apiSwitch';

export function useMRIndicators(params: PageRequest) {
  return useQuery({
    queryKey: ['mr', 'indicators', params],
    queryFn: () =>
      useMock ? mrService.getIndicators(params) : mrApi.getIndicators(params),
    staleTime: 10 * 60 * 1000,
  });
}

export function useAllMRIndicators() {
  return useQuery({
    queryKey: ['mr', 'indicators', 'all'],
    queryFn: () =>
      useMock ? mrService.getAllIndicators() : mrApi.getAllIndicators(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useMRMappings(
  params: { deviceSn?: string; enabled?: boolean } & PageRequest
) {
  return useQuery({
    queryKey: ['mr', 'mappings', params],
    queryFn: () =>
      useMock ? mrService.getMappings(params) : mrApi.getMappings(params),
  });
}

export function useUpdateMRMapping() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<MRDeviceMapping> }) =>
      useMock ? mrService.updateMapping(id, data) : mrApi.updateMapping(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mr', 'mappings'] });
    },
  });
}

export function useToggleMRMapping() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      useMock ? mrService.toggleMapping(id, enabled) : mrApi.toggleMapping(id, enabled),
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
    queryFn: () =>
      useMock ? mrService.getRecords(params) : mrApi.getRecords(params),
  });
}

export function useMRFiles(
  params: { deviceSn?: string; mrType?: string; timeRange?: [string, string] } & PageRequest
) {
  return useQuery({
    queryKey: ['mr', 'files', params],
    queryFn: () => mrApi.getFiles(params),
    enabled: !useMock,
  });
}

export function useDownloadMRFile() {
  return useMutation({
    mutationFn: (fileId: string) => mrApi.downloadFile(fileId),
  });
}

// 按设备聚合的 MR 文件视图 —— File Management → MR Tab 主列表
export function useMRFileDevices(
  params: { keyword?: string } & PageRequest,
) {
  return useQuery({
    queryKey: ['mr', 'files', 'devices', params],
    queryFn: () => mrApi.getFileDevices(params),
    enabled: !useMock,
  });
}

export function useExportMRData() {
  return useMutation({
    mutationFn: (params: { deviceSns: string[]; timeRange: [string, string] }) =>
      useMock ? mrService.exportMRData(params) : mrApi.exportMRData(params),
  });
}

export function useMRIndicatorStats(indicatorCode: string, deviceSn?: string) {
  return useQuery({
    queryKey: ['mr', 'stats', indicatorCode, deviceSn],
    queryFn: () =>
      useMock ? mrService.getIndicatorStats(indicatorCode, deviceSn) : mrApi.getIndicatorStats(indicatorCode, deviceSn),
    enabled: Boolean(indicatorCode),
  });
}
