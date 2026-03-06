import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ReportDefinition, ReportType, ReportPeriod } from '@/mock/data/reports';
import type { PageRequest } from '@/types/pagination';
import { reportService } from '@/mock/services/reportService';

export function useReportDefinitions(
  params: { reportType?: ReportType; status?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['reports', 'definitions', params],
    queryFn: () => reportService.getDefinitions(params),
  });
}

export function useReportDefinitionById(id: string) {
  return useQuery({
    queryKey: ['reports', 'definitions', 'detail', id],
    queryFn: () => reportService.getDefinitionById(id),
    enabled: Boolean(id),
  });
}

export function useCreateReportDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ReportDefinition, 'id' | 'createTime'>) =>
      reportService.createDefinition(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useUpdateReportDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ReportDefinition> }) =>
      reportService.updateDefinition(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useDeleteReportDefinitions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => reportService.deleteDefinitions(ids),
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
    queryFn: () => reportService.getRecords(params),
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
    }) => reportService.generateReport(definitionId, period, reportType),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['reports', 'records'] });
      void queryClient.invalidateQueries({ queryKey: ['reports', 'definitions'] });
    },
  });
}

export function useDownloadReport() {
  return useMutation({
    mutationFn: (id: string) => reportService.downloadRecord(id),
  });
}

export function useReportSampleData() {
  return useQuery({
    queryKey: ['reports', 'sample-data'],
    queryFn: () => reportService.getSampleData(),
    staleTime: 5 * 60 * 1000,
  });
}
