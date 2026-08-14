/**
 * T-0174 指标查询页类型定义。
 *
 * - 复用 pmDashboard 的 Granularity / AggregatedRow / AggregatedQueryParams（不再重复定义）。
 * - 模板 payload 是后端不感知的 JSON，前端按自己需要扩展字段。
 *
 * 后端：omcgo/internal/pm/querytemplate/{model,handler}.go
 *   GET    /api/v1/pm/query-templates           列表
 *   GET    /api/v1/pm/query-templates/:id       详情
 *   POST   /api/v1/pm/query-templates           创建
 *   PATCH  /api/v1/pm/query-templates/:id       更新
 *   DELETE /api/v1/pm/query-templates/:id       删除
 */

import type { Granularity } from './pmDashboard';

export type TemplateVisibility = 'public' | 'private';

// 时间窗预设：前端选预设后转 RFC3339 区间下发后端，预设原值仍保留方便回显。
export type TimeRangePreset = 'last_1h' | 'last_3h' | 'last_24h' | 'last_7d' | 'last_30d' | 'last_6m' | 'custom';

// 模板 payload — 前端定义的 JSON 结构，后端 JSONB 原样存储。
export interface QueryTemplatePayload {
  deviceSns: string[];
  metricPaths: string[];
  granularity: Granularity;
  timeRangePreset: TimeRangePreset;
  // custom 模式下保留绝对时间（RFC3339）；预设模式下下次查询会按当前时间重算。
  absoluteStart?: string;
  absoluteEnd?: string;
  // 设备类型（影响 indicator picker 筛选）
  deviceType?: 'ENB' | 'GNB' | 'GSM';
}

// 前端域模型
export interface QueryTemplate {
  id: string;
  name: string;
  visibility: TemplateVisibility;
  creatorId: string;
  description?: string;
  payload: QueryTemplatePayload;
  createdAt: string;
  updatedAt: string;
}

export interface ListTemplateParams {
  visibility?: TemplateVisibility;
  search?: string;
  page?: number;
  pageSize?: number;
}

export interface CreateTemplateInput {
  name: string;
  visibility: TemplateVisibility;
  description?: string;
  payload: QueryTemplatePayload;
}

export interface UpdateTemplateInput {
  name?: string;
  description?: string;
  payload?: QueryTemplatePayload;
  visibility?: TemplateVisibility;
}

// 后端 wire 类型
export interface BackendQueryTemplate {
  id: string;
  name: string;
  visibility: TemplateVisibility;
  creator_id: string;
  description?: string;
  payload: Record<string, unknown>; // JSONB 原样
  created_at: string;
  updated_at: string;
}

export type ReportPeriod = '15min' | 'hourly' | 'daily';

export interface QueryReportSubscription {
  id: string;
  queryTemplateId: string;
  enabled: boolean;
  period: ReportPeriod;
  sendTimes: string[];
  recipients: string[];
  timezoneName: string;
  nextRunAt?: string;
  lastRunAt?: string;
  lastStatus?: 'pending' | 'running' | 'sent' | 'export_failed' | 'delivery_failed';
  lastError?: string;
}

export interface UpsertQueryReportSubscriptionInput {
  enabled: boolean;
  period: ReportPeriod;
  sendTimes: string[];
  recipients: string[];
}

export interface BackendQueryReportSubscription {
  id: string;
  query_template_id: string;
  enabled: boolean;
  period: ReportPeriod;
  send_times: string[];
  recipients?: string[];
  timezone_name: string;
  next_run_at?: string;
  last_run_at?: string;
  last_status?: 'pending' | 'running' | 'sent' | 'export_failed' | 'delivery_failed';
  last_error?: string;
}

export function mapBackendQueryReportSubscription(
  value: BackendQueryReportSubscription,
): QueryReportSubscription {
  return {
    id: value.id,
    queryTemplateId: value.query_template_id,
    enabled: value.enabled,
    period: value.period,
    sendTimes: value.send_times ?? [],
    recipients: value.recipients ?? [],
    timezoneName: value.timezone_name,
    nextRunAt: value.next_run_at,
    lastRunAt: value.last_run_at,
    lastStatus: value.last_status,
    lastError: value.last_error,
  };
}

// payload 的后端 JSONB 字段名（snake_case 由前端写入）— 与上面 QueryTemplatePayload 一一对应。
interface BackendPayload {
  device_sns?: string[];
  metric_paths?: string[];
  granularity?: string;
  time_range_preset?: string;
  absolute_start?: string;
  absolute_end?: string;
  device_type?: string;
}

function mapPayload(p: Record<string, unknown> | null | undefined): QueryTemplatePayload {
  const b = (p ?? {}) as BackendPayload;
  return {
    deviceSns: b.device_sns ?? [],
    metricPaths: b.metric_paths ?? [],
    granularity: (b.granularity as Granularity) ?? '15min',
    timeRangePreset: (b.time_range_preset as TimeRangePreset) ?? 'last_1h',
    absoluteStart: b.absolute_start,
    absoluteEnd: b.absolute_end,
    deviceType: b.device_type as 'ENB' | 'GNB' | 'GSM' | undefined,
  };
}

function payloadToBackend(p: QueryTemplatePayload): BackendPayload {
  return {
    device_sns: p.deviceSns,
    metric_paths: p.metricPaths,
    granularity: p.granularity,
    time_range_preset: p.timeRangePreset,
    absolute_start: p.absoluteStart,
    absolute_end: p.absoluteEnd,
    device_type: p.deviceType,
  };
}

export function mapBackendQueryTemplate(b: BackendQueryTemplate): QueryTemplate {
  return {
    id: b.id,
    name: b.name,
    visibility: b.visibility,
    creatorId: b.creator_id,
    description: b.description,
    payload: mapPayload(b.payload),
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

export function templatePayloadToBackend(p: QueryTemplatePayload): Record<string, unknown> {
  return payloadToBackend(p) as Record<string, unknown>;
}
