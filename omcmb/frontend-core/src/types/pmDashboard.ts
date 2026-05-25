/**
 * T-0164-P6 / G6 PM 性能查看仪表盘类型定义。
 *
 * 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.6
 * 实施 plan：docs/project/plan-T-0164-P6-frontend-dashboard.md
 *
 * 与现有 src/types/performance.ts 关系：
 *   - performance.ts 承载老三 tab（概览/趋势探查/报表）的类型；G6 上线后这些类型逐步归档
 *   - pmDashboard.ts 是新单 tab "性能查看"的 Dashboard / Panel / 用户偏好模型
 */

// 顶层制式切换
export type Technology = 'lte' | 'nr' | 'gsm';

// 7 种 Panel 类型（T-0164 收尾 G6-Gap-5 加 topn + big_number）
export type PanelType = 'kpi_card' | 'line_chart' | 'bar_chart' | 'table' | 'gauge' | 'topn' | 'big_number';

// 数据维度
export type Dimension = 'device' | 'device_group';

// 对比双模式
export type CompareMode = 'same_window_other_devices' | 'previous_window';

// 粒度（与后端 metrics.Granularity 一致）
export type Granularity = '15min' | 'hourly' | 'daily' | 'weekly' | 'monthly';

// 时间范围：相对（offset）或绝对（absolute）。前端编辑器二选一。
export type PanelTimeRange =
  | { start_offset: string; end_offset?: string } // 相对：'-1h' / '-7d' / '-30d'
  | { absolute_start: string; absolute_end: string }; // ISO RFC3339

// react-grid-layout item（layout JSON 透传，不在 TS 里强类型化）
export type PanelGridItem = {
  i: string; // panel id
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
  static?: boolean;
};

export interface DashboardLayout {
  panels: PanelGridItem[];
}

// ── 前端域模型 ────────────────────────────────────────────────────────

export interface Dashboard {
  id: string;
  name: string;
  description?: string;
  ownerId: string;
  sharedWith: string[];
  parentDashboardId?: string;
  technology: Technology;
  layout: DashboardLayout;
  isBuiltin: boolean; // G6-Gap-4: 系统内置 readonly 标记
  createdAt: string;
  updatedAt: string;
}

export interface Panel {
  id: string;
  dashboardId: string;
  panelType: PanelType;
  title: string;
  metricPaths: string[];
  // G6-Gap-6：多粒度数组，前端 PanelHeader 用 Tab 切换浏览，不重新请求 CRUD。
  granularities: Granularity[];
  dimension: Dimension;
  deviceSns?: string[];
  deviceGroupIds?: string[];
  timeRange: PanelTimeRange;
  compareMode?: CompareMode;
  adhocTaskId?: string;
  config: Record<string, unknown>;
}

export interface UserDashboardPreferences {
  userId: string;
  technology: Technology;
  kpiCardLayout: Record<string, unknown>;
  currentDashboardId?: string;
  sharedFilters: Record<string, unknown>;
}

// T-0164 收尾 G6-Gap-3：upsert user preferences 时必须显式指定 technology。
export interface UpsertUserPreferencesInput {
  technology: Technology;
  kpiCardLayout?: Record<string, unknown>;
  currentDashboardId?: string;
  sharedFilters?: Record<string, unknown>;
}

// ── Input types（hooks / handler 入参）────────────────────────────────

export interface CreateDashboardInput {
  name: string;
  description?: string;
  technology: Technology;
  layout?: DashboardLayout;
}

export interface UpdateDashboardInput {
  name?: string;
  description?: string;
  technology?: Technology;
  layout?: DashboardLayout;
}

export interface CreatePanelInput {
  dashboardId: string;
  panelType: PanelType;
  title: string;
  metricPaths: string[];
  granularities: Granularity[];
  dimension: Dimension;
  deviceSns?: string[];
  deviceGroupIds?: string[];
  timeRange: PanelTimeRange;
  compareMode?: CompareMode;
  adhocTaskId?: string;
  config?: Record<string, unknown>;
}

export interface ForkInput {
  sourceId: string;
  newName: string;
}

export interface ShareInput {
  dashboardId: string;
  userIds: string[];
}

// ── Backend wire types（snake_case，与后端 handler.go DTO 对齐）────────

export interface BackendDashboard {
  id: string;
  name: string;
  description?: string;
  owner_id: string;
  shared_with: string[];
  parent_dashboard_id?: string;
  technology: string;
  layout: DashboardLayout;
  is_builtin?: boolean; // G6-Gap-4
  created_at: string;
  updated_at: string;
}

export interface BackendPanel {
  id: string;
  dashboard_id: string;
  panel_type: string;
  title: string;
  metric_paths: string[];
  granularities: string[];
  dimension: string;
  device_sns?: string[];
  device_group_ids?: string[];
  time_range: PanelTimeRange;
  compare_mode?: string;
  adhoc_task_id?: string;
  config: Record<string, unknown>;
}

export interface BackendUserPreferences {
  user_id: string;
  technology: string;
  kpi_card_layout: Record<string, unknown>;
  current_dashboard_id?: string;
  shared_filters: Record<string, unknown>;
}

// ── Mappers（snake_case → camelCase）────────────────────────────────

export function mapBackendDashboard(b: BackendDashboard): Dashboard {
  return {
    id: b.id,
    name: b.name,
    description: b.description,
    ownerId: b.owner_id,
    sharedWith: b.shared_with ?? [],
    parentDashboardId: b.parent_dashboard_id,
    technology: b.technology as Technology,
    layout: b.layout ?? { panels: [] },
    isBuiltin: b.is_builtin ?? false,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

export function mapBackendPanel(b: BackendPanel): Panel {
  return {
    id: b.id,
    dashboardId: b.dashboard_id,
    panelType: b.panel_type as PanelType,
    title: b.title,
    metricPaths: b.metric_paths,
    granularities: (b.granularities ?? []) as Granularity[],
    dimension: b.dimension as Dimension,
    deviceSns: b.device_sns,
    deviceGroupIds: b.device_group_ids,
    timeRange: b.time_range,
    compareMode: b.compare_mode as CompareMode | undefined,
    adhocTaskId: b.adhoc_task_id,
    config: b.config ?? {},
  };
}

export function mapBackendPreferences(b: BackendUserPreferences): UserDashboardPreferences {
  return {
    userId: b.user_id,
    technology: (b.technology as Technology) ?? 'lte',
    kpiCardLayout: b.kpi_card_layout ?? {},
    currentDashboardId: b.current_dashboard_id,
    sharedFilters: b.shared_filters ?? {},
  };
}

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
}

// 后端 aggregator.Row JSON（已加 snake_case json tag）。
export interface BackendAggregatedRow {
  device_oui?: string;
  device_sn?: string;
  // device 维度查询时是全零 UUID；分组维度才有实际值。前端在 mapper 里把全零规整成 undefined。
  device_group_id?: string;
  metric_path: string;
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
}

export interface AggregatedRow {
  deviceOui?: string;
  deviceSn?: string;
  deviceGroupId?: string;
  metricPath: string;
  metricType: 'counter' | 'kpi';
  metricValue: number;
  statisType?: string;
  granularity: Granularity;
  time: string;
  startTime: string;
  endTime: string;
  ingestTime: string;
  objectLdn?: string | null;
  extra?: Record<string, unknown>;
}

const ZERO_UUID = '00000000-0000-0000-0000-000000000000';

export function mapBackendAggregatedRow(b: BackendAggregatedRow): AggregatedRow {
  return {
    deviceOui: b.device_oui || undefined,
    deviceSn: b.device_sn || undefined,
    deviceGroupId: !b.device_group_id || b.device_group_id === ZERO_UUID ? undefined : b.device_group_id,
    metricPath: b.metric_path,
    metricType: (b.metric_type as 'counter' | 'kpi') ?? 'counter',
    metricValue: b.metric_value,
    statisType: b.statis_type,
    granularity: b.granularity as Granularity,
    time: b.time,
    startTime: b.start_time,
    endTime: b.end_time,
    ingestTime: b.ingest_time,
    objectLdn: b.object_ldn ?? null,
    extra: b.extra ?? {},
  };
}
