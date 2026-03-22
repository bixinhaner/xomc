import http from '../http';
import type { ReportDefinition, ReportRecord, ReportType, ReportStatus, ReportFormat } from '@/mock/data/reports';
import type { PageRequest, PageResponse } from '@/types/pagination';

// Backend report definition model (snake_case from Go)
interface BackendReportDefinition {
  id: string;
  report_name: string;
  report_type: ReportType;
  description: string;
  format: string[];
  period: string;
  kpi_codes: string[];
  device_groups: string[];
  auto_generate: boolean;
  cron_expression: string;
  status: ReportStatus;
  creator: string;
  last_gen_time: string | null;
  created_at: string;
  updated_at: string;
}

// Backend report record model (snake_case from Go)
interface BackendReportRecord {
  id: string;
  report_definition_id: string;
  report_name: string;
  period: string;
  generate_time: string;
  file_size: number;
  download_url: string;
  format: string;
  status: 'generating' | 'ready' | 'failed';
  minio_path: string;
  created_at: string;
}

function mapDefinition(bd: BackendReportDefinition): ReportDefinition {
  return {
    id: bd.id,
    reportName: bd.report_name,
    reportType: bd.report_type,
    description: bd.description ?? '',
    format: (bd.format ?? ['pdf']) as ReportFormat[],
    period: bd.period as ReportDefinition['period'],
    kpiCodes: bd.kpi_codes ?? [],
    deviceGroups: bd.device_groups ?? [],
    autoGenerate: bd.auto_generate,
    cronExpression: bd.cron_expression || undefined,
    status: bd.status,
    createTime: bd.created_at,
    creator: bd.creator ?? '',
    lastGenTime: bd.last_gen_time ?? undefined,
  };
}

function mapRecord(br: BackendReportRecord): ReportRecord {
  return {
    id: br.id,
    reportDefinitionId: br.report_definition_id,
    reportName: br.report_name,
    period: br.period ?? '',
    generateTime: br.generate_time,
    fileSize: br.file_size,
    downloadUrl: br.download_url ?? '',
    format: br.format as ReportFormat,
    status: br.status,
  };
}

export const reportsApi = {
  async getDefinitions(
    params: { reportType?: ReportType; status?: string } & PageRequest
  ): Promise<PageResponse<ReportDefinition>> {
    const { data } = await http.get<{
      items: BackendReportDefinition[];
      total: number;
      page: number;
      page_size: number;
    }>('/reports/definitions', {
      params: {
        report_type: params.reportType,
        status: params.status,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return {
      items: (data.items || []).map(mapDefinition),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getDefinitionById(id: string): Promise<ReportDefinition | null> {
    try {
      const { data } = await http.get<BackendReportDefinition>(`/reports/definitions/${id}`);
      return mapDefinition(data);
    } catch {
      return null;
    }
  },

  async createDefinition(
    input: Omit<ReportDefinition, 'id' | 'createTime'>
  ): Promise<ReportDefinition> {
    const { data } = await http.post<BackendReportDefinition>('/reports/definitions', {
      report_name: input.reportName,
      report_type: input.reportType,
      description: input.description,
      format: input.format,
      period: input.period,
      kpi_codes: input.kpiCodes ?? [],
      device_groups: input.deviceGroups ?? [],
      auto_generate: input.autoGenerate,
      cron_expression: input.cronExpression ?? '',
      status: input.status,
      creator: input.creator,
    });
    return mapDefinition(data);
  },

  async updateDefinition(id: string, input: Partial<ReportDefinition>): Promise<ReportDefinition> {
    const { data } = await http.put<BackendReportDefinition>(`/reports/definitions/${id}`, {
      report_name: input.reportName,
      report_type: input.reportType,
      description: input.description,
      format: input.format,
      period: input.period,
      kpi_codes: input.kpiCodes ?? [],
      device_groups: input.deviceGroups ?? [],
      auto_generate: input.autoGenerate,
      cron_expression: input.cronExpression ?? '',
      status: input.status,
      creator: input.creator,
    });
    return mapDefinition(data);
  },

  async deleteDefinitions(ids: string[]): Promise<void> {
    // Backend supports single delete; loop for batch
    for (const id of ids) {
      await http.delete(`/reports/definitions/${id}`);
    }
  },

  async getRecords(
    params: { reportDefinitionId?: string; format?: string } & PageRequest
  ): Promise<PageResponse<ReportRecord>> {
    const { data } = await http.get<{
      items: BackendReportRecord[];
      total: number;
      page: number;
      page_size: number;
    }>('/reports/records', {
      params: {
        definition_id: params.reportDefinitionId,
        format: params.format,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return {
      items: (data.items || []).map(mapRecord),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async generateReport(
    definitionId: string,
    period?: string,
    _reportType?: string
  ): Promise<ReportRecord> {
    const { data } = await http.post<BackendReportRecord>('/reports/generate', {
      definition_id: definitionId,
      period: period ?? new Date().toISOString().slice(0, 10),
    });
    return mapRecord(data);
  },

  async downloadRecord(id: string): Promise<{ url: string; fileName: string }> {
    const { data } = await http.get<{ url: string; file_name: string }>(
      `/reports/records/${id}/download`
    );
    return { url: data.url, fileName: data.file_name };
  },

  async getSampleData(): Promise<Record<string, unknown>> {
    const { data } = await http.get<Record<string, unknown>>('/reports/sample-data');
    return data;
  },
};
