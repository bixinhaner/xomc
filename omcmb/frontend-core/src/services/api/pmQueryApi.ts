/**
 * T-0174 指标查询页 REST API 客户端。
 *
 * 模板 CRUD 5 端点 + 复用 pmDashboardApi.queryAggregated 做 long-format 查询。
 */

import http from '../http';
import type {
  BackendQueryTemplate,
  QueryTemplate,
  ListTemplateParams,
  CreateTemplateInput,
  UpdateTemplateInput,
  QueryTemplateRegularReport,
  QueryTemplateRegularReportInput,
} from '../../types/pmQuery';
import { mapBackendQueryTemplate, templatePayloadToBackend } from '../../types/pmQuery';

interface ListResponse {
  items: BackendQueryTemplate[];
  total: number;
}

interface BackendRegularReport {
  template_id: string;
  enabled: boolean;
  send_time: string;
  period: QueryTemplateRegularReport['period'];
  recipients: string[];
  revision: number;
  next_run_at?: string;
}

function mapRegularReport(value: BackendRegularReport, etag?: string): QueryTemplateRegularReport {
  const headerRevision = Number(etag?.replace(/^W\//, '').replaceAll('"', ''));
  return {
    templateId: value.template_id,
    enabled: value.enabled,
    sendTime: value.send_time,
    period: value.period,
    recipients: value.recipients ?? [],
    revision: Number.isInteger(headerRevision) && headerRevision > 0 ? headerRevision : value.revision,
    nextRunAt: value.next_run_at,
  };
}

export const pmQueryApi = {
  async list(params?: ListTemplateParams): Promise<{ items: QueryTemplate[]; total: number }> {
    const qp: Record<string, string | number | undefined> = {
      visibility: params?.visibility,
      search: params?.search,
      page: params?.page,
      page_size: params?.pageSize,
    };
    const { data } = await http.get<ListResponse>('/pm/query-templates', { params: qp });
    return {
      items: (data.items ?? []).map(mapBackendQueryTemplate),
      total: data.total ?? 0,
    };
  },

  async get(id: string): Promise<QueryTemplate> {
    const { data } = await http.get<BackendQueryTemplate>(`/pm/query-templates/${id}`);
    return mapBackendQueryTemplate(data);
  },

  async create(input: CreateTemplateInput): Promise<QueryTemplate> {
    const body = {
      name: input.name,
      visibility: input.visibility,
      description: input.description ?? '',
      payload: templatePayloadToBackend(input.payload),
    };
    const { data } = await http.post<BackendQueryTemplate>('/pm/query-templates', body);
    return mapBackendQueryTemplate(data);
  },

  async update(id: string, input: UpdateTemplateInput): Promise<QueryTemplate> {
    const body: Record<string, unknown> = {};
    if (input.name !== undefined) body.name = input.name;
    if (input.description !== undefined) body.description = input.description;
    if (input.visibility !== undefined) body.visibility = input.visibility;
    if (input.payload !== undefined) body.payload = templatePayloadToBackend(input.payload);
    const { data } = await http.patch<BackendQueryTemplate>(`/pm/query-templates/${id}`, body);
    return mapBackendQueryTemplate(data);
  },

  async remove(id: string): Promise<void> {
    await http.delete(`/pm/query-templates/${id}`);
  },

  async getRegularReport(id: string): Promise<QueryTemplateRegularReport> {
    const response = await http.get<BackendRegularReport>(`/pm/query-templates/${id}/regular-report`);
    return mapRegularReport(response.data, response.headers.etag as string | undefined);
  },

  async updateRegularReport(
    id: string,
    revision: number,
    input: QueryTemplateRegularReportInput,
  ): Promise<QueryTemplateRegularReport> {
    const response = await http.patch<BackendRegularReport>(
      `/pm/query-templates/${id}/regular-report`,
      {
        enabled: input.enabled,
        send_time: input.sendTime,
        period: input.period,
        recipients: input.recipients,
      },
      { headers: { 'If-Match': `"${revision}"` } },
    );
    return mapRegularReport(response.data, response.headers.etag as string | undefined);
  },
};
