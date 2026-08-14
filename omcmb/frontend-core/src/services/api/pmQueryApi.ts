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
  BackendQueryReportSubscription,
  QueryReportSubscription,
  UpsertQueryReportSubscriptionInput,
} from '../../types/pmQuery';
import {
  mapBackendQueryReportSubscription,
  mapBackendQueryTemplate,
  templatePayloadToBackend,
} from '../../types/pmQuery';

interface ListResponse {
  items: BackendQueryTemplate[];
  total: number;
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

  async getReportSubscription(id: string): Promise<QueryReportSubscription> {
    const { data } = await http.get<BackendQueryReportSubscription>(
      `/pm/query-templates/${id}/report-subscription`,
    );
    return mapBackendQueryReportSubscription(data);
  },

  async upsertReportSubscription(
    id: string,
    input: UpsertQueryReportSubscriptionInput,
  ): Promise<QueryReportSubscription> {
    const { data } = await http.put<BackendQueryReportSubscription>(
      `/pm/query-templates/${id}/report-subscription`,
      {
        enabled: input.enabled,
        period: input.period,
        send_times: input.sendTimes,
        recipients: input.recipients,
      },
    );
    return mapBackendQueryReportSubscription(data);
  },

  async deleteReportSubscription(id: string): Promise<void> {
    await http.delete(`/pm/query-templates/${id}/report-subscription`);
  },
};
