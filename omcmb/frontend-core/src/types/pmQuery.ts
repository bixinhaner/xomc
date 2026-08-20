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
export type RegularReportPeriod = '15min' | 'hourly' | 'daily';

export interface RegularReportConfig {
  enabled: boolean;
  sendTime: string;
  periods: RegularReportPeriod[];
  emailEnabled: boolean;
  recipients: string[];
}

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
  regularReport?: RegularReportConfig;
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

// payload 的后端 JSONB 字段名（snake_case 由前端写入）— 与上面 QueryTemplatePayload 一一对应。
interface BackendPayload {
  device_sns?: string[];
  metric_paths?: string[];
  granularity?: string;
  time_range_preset?: string;
  absolute_start?: string;
  absolute_end?: string;
  device_type?: string;
  regular_report?: {
    enabled?: boolean;
    send_time?: string;
    periods?: string[];
    email_enabled?: boolean;
    recipients?: string[];
  };
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
    regularReport: {
      enabled: b.regular_report?.enabled ?? false,
      sendTime: b.regular_report?.send_time ?? '08:00',
      periods: (b.regular_report?.periods ?? ['daily']) as RegularReportPeriod[],
      emailEnabled: b.regular_report?.email_enabled ?? true,
      recipients: b.regular_report?.recipients ?? [],
    },
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
    regular_report: {
      enabled: p.regularReport?.enabled ?? false,
      send_time: p.regularReport?.sendTime ?? '08:00',
      periods: p.regularReport?.periods ?? ['daily'],
      email_enabled: p.regularReport?.emailEnabled ?? true,
      recipients: p.regularReport?.recipients ?? [],
    },
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
