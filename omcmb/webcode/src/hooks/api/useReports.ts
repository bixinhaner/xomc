import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ReportDefinition, ReportType, ReportPeriod } from '@/mock/data/reports';
import type { PageRequest } from '@/types/pagination';
import { reportService } from '@/mock/services/reportService';
import { reportsApi } from '@/services/api/reportsApi';
import { useMock } from '@/services/apiSwitch';

export function useReportDefinitions(
  params: { reportType?: ReportType; status?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['reports', 'definitions', params],
    queryFn: () =>
      useMock ? reportService.getDefinitions(params) : reportsApi.getDefinitions(params),
  });
}

export function useReportDefinitionById(id: string) {
  return useQuery({
    queryKey: ['reports', 'definitions', 'detail', id],
    queryFn: () =>
      useMock ? reportService.getDefinitionById(id) : reportsApi.getDefinitionById(id),
    enabled: Boolean(id),
  });
}

export function useCreateReportDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ReportDefinition, 'id' | 'createTime'>) =>
      useMock ? reportService.createDefinition(data) : reportsApi.createDefinition(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useUpdateReportDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ReportDefinition> }) =>
      useMock ? reportService.updateDefinition(id, data) : reportsApi.updateDefinition(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useDeleteReportDefinitions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? reportService.deleteDefinitions(ids) : reportsApi.deleteDefinitions(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useReportRecords(
  params: { reportDefinitionId?: string; format?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['reports', 'records', params],
    queryFn: () =>
      useMock ? reportService.getRecords(params) : reportsApi.getRecords(params),
  });
}

export function useGenerateReport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      definitionId,
      period,
      reportType,
    }: {
      definitionId: string;
      period?: string;
      reportType?: ReportPeriod;
    }) =>
      useMock
        ? reportService.generateReport(definitionId, period, reportType)
        : reportsApi.generateReport(definitionId, period, reportType),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'records'] });
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useDownloadReport() {
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? reportService.downloadRecord(id) : reportsApi.downloadRecord(id),
  });
}

export function useReportSampleData() {
  return useQuery({
    queryKey: ['reports', 'sample-data'],
    queryFn: () =>
      useMock ? reportService.getSampleData() : reportsApi.getSampleData(),
    staleTime: 5 * 60 * 1000,
  });
}
