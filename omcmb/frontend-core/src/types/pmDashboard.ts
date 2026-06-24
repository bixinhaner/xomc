/**
 * PM 聚合指标查询类型定义。
 *
 * 旧拖拽仪表盘（Dashboard / Panel / 用户偏好）模型已随后端模块下线（ISSUE-403），
 * 本文件仅保留 /pm/metrics/aggregated 聚合查询所需类型，被指标查询页 / 设备详情 KPI 等复用。
 */

// 数据维度
export type Dimension = 'device' | 'device_group';

// 粒度（与后端 metrics.Granularity 一致）
export type Granularity = '15min' | 'hourly' | 'daily' | 'weekly' | 'monthly';

// ── G6 Phase 4: /pm/metrics/aggregated 真 API 数据形态 ─────────────────

export interface AggregatedQueryParams {
  granularity: Granularity;
  dimension?: Dimension;
  // 后端 v1 handler 只支持单 OUI+SN 过滤；多设备走多次查询或 group dimension。
  deviceOui?: string;
  deviceSn?: string;
  deviceGroupId?: string;
  metricPaths?: string[];
  metricType?: 'counter' | 'kpi';
  // ISO RFC3339 字符串。
  startTime?: string;
  endTime?: string;
  limit?: number;
  offset?: number;
  // 后端按 (时间桶 × 指标) 补齐占位行（filled=true，metric_value 前端 mapper 设 null）。
  // 适用 device 维度单设备查询；未启用时后端不补行。
  fillEmpty?: boolean;
  // #599：星期过滤（0=周日..6=周六）。全选/空 = 不过滤。
  weekdays?: number[];
  // #599：小时段过滤（0..23）。全选/空 = 不过滤。
  hours?: number[];
}

// 后端 aggregator.Row JSON（已加 snake_case json tag）。
export interface BackendAggregatedRow {
  device_oui?: string;
  device_sn?: string;
  // device 维度查询时是全零 UUID；分组维度才有实际值。前端在 mapper 里把全零规整成 undefined。
  device_group_id?: string;
  metric_path: string;
  // KPI 行 metric_path 是 K 编号；display_name 为后端按编号回填的友好名（PLMN 级带「（PLMN级）」标记）。
  // counter 行 display_name = metric_path（本身可读）。
  display_name?: string;
  metric_type: string;
  metric_value: number;
  statis_type?: string;
  granularity: string;
  time: string;
  start_time: string;
  end_time: string;
  ingest_time: string;
  object_ldn?: string | null;
  extra?: Record<string, unknown>;
  // 后端 fill_empty 占位行（DB 无样本时补的空桶），前端 mapper 见 filled=true 把
  // metricValue 设 null 用于"-"渲染。
  filled?: boolean;
}

export interface AggregatedRow {
  deviceOui?: string;
  deviceSn?: string;
  deviceGroupId?: string;
  metricPath: string;
  // KPI 行 metricPath 是 K 编号；displayName 为后端回填的友好名（展示层用 displayName，不露编号）。
  displayName?: string;
  metricType: 'counter' | 'kpi';
  // null 表示该 (时间桶 × 指标) 占位（fill_empty 补行 / 兼容旧缺采渲染）。
  metricValue: number | null;
  statisType?: string;
  granularity: Granularity;
  time: string;
  startTime: string;
  endTime: string;
  ingestTime: string;
  objectLdn?: string | null;
  extra?: Record<string, unknown>;
  filled?: boolean;
}

const ZERO_UUID = '00000000-0000-0000-0000-000000000000';

export function mapBackendAggregatedRow(b: BackendAggregatedRow): AggregatedRow {
  return {
    deviceOui: b.device_oui || undefined,
    deviceSn: b.device_sn || undefined,
    deviceGroupId: !b.device_group_id || b.device_group_id === ZERO_UUID ? undefined : b.device_group_id,
    metricPath: b.metric_path,
    displayName: b.display_name || undefined,
    metricType: (b.metric_type as 'counter' | 'kpi') ?? 'counter',
    // filled=true 是 fill_empty 占位行，后端 metric_value 字段无意义，前端统一显示 "-"。
    metricValue: b.filled ? null : b.metric_value,
    statisType: b.statis_type,
    granularity: b.granularity as Granularity,
    time: b.time,
    startTime: b.start_time,
    endTime: b.end_time,
    ingestTime: b.ingest_time,
    objectLdn: b.object_ldn ?? null,
    extra: b.extra ?? {},
    filled: b.filled,
  };
}
