/**
 * Dashboard 类型定义
 * 包含仪表板相关的所有 TypeScript 类型
 *
 * 注意：这些类型与后端 API 响应结构保持一致
 * Mock 数据使用相同的类型定义，确保类型安全
 */


// ============================================================================
// 从 mock/data/dashboard.ts 导入的核心类型
// ============================================================================

/**
 * 设备状态统计（Dashboard 概览卡片用）。
 * 注：与 types/device.ts 的 DeviceStats（{ counts: Record<string, number> }）形状不同、
 * 用途不同，故用 DashboardDeviceStats 命名避免 types barrel 重复导出歧义。
 */
export interface DashboardDeviceStats {
  total: number;
  online: number;
  offline: number;
  alarm: number;
}

/**
 * 告警状态统计
 */
export interface AlarmStats {
  critical: number;
  major: number;
  minor: number;
  warning: number;
  total: number;
}

/**
 * KPI 概览（支持动态字段）
 * 后端返回的 kpi_overview 是 map[string]float64，支持自定义 KPI
 */
export interface KPIOverview {
  [key: string]: number | undefined;
}

/**
 * KPI 趋势增量数据
 * 用于 KPI 卡片显示趋势（上升/下降/稳定）
 *
 * 字段采用前端 camelCase；后端响应 snake_case 形态见 [`BackendKPIDelta`]，
 * 由 `mapBackendSummary` 做一次性转换收敛。
 */
export interface KPIDelta {
  /** 当前值 */
  currentValue: number;
  /** 对比周期的值 */
  previousValue: number;
  /** 变化百分比（正数表示增长） */
  changePercent: number;
  /** 趋势方向: "up" | "down" | "stable" */
  trend: 'up' | 'down' | 'stable';
  /** 对比类型: "yesterday" | "last_week" */
  compareType: 'yesterday' | 'last_week';
  /** 当前值与历史基线是否都有效，可用于百分比对比 */
  hasComparison: boolean;
}

/**
 * 任务状态统计
 */
export interface TaskStats {
  running: number;
  pending: number;
  success: number;
  failed: number;
}

export interface PMSlotHealth {
  slotEnd: string;
  technology: string;
  carrier: string;
  expectedDevices: number;
  receivedDevices: number;
  coverageRatio: number;
  status: 'complete' | 'partial' | 'missing' | 'bootstrap_ignored';
  evaluatedAt: string;
}

/**
 * Dashboard 汇总数据
 */
export interface DashboardSummary {
  deviceCounts: DashboardDeviceStats;
  alarmCounts: AlarmStats;
  kpiSummary: KPIOverview;
  kpiDeltas: Record<string, KPIDelta>;  // KPI趋势数据（新增）
  taskSummary: TaskStats;
  pmSlotHealth: PMSlotHealth[];
}

/**
 * 告警趋势数据点
 */
export interface AlarmTrendItem {
  date: string;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

/**
 * 设备状态分布（用于饼图）
 */
export interface DeviceStatusItem {
  name: string;
  value: number;
}

/**
 * KPI 时序数据点
 * 支持两种格式：
 * - Mock/内部: [string, number] (元组格式，节省空间)
 * - API: { time: string, value: number } (对象格式，更清晰)
 */
export type KPITimeSeriesPoint = { time: string; value: number } | [string, number];

/**
 * 告警类型分布
 */
export interface AlarmTypeItem {
  name: string;
  value: number;
}

/**
 * 区域设备统计
 */
export interface RegionDeviceStats {
  region: string;
  total: number;
  online: number;
  offline: number;
}

/**
 * Top 告警设备
 */
export interface TopAlarmDevice {
  deviceSN: string;
  technology: string;
  alarmCount: number;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

/**
 * Dashboard 图表数据
 */
export interface DashboardChartData {
  alarmTrend: AlarmTrendItem[];
  deviceStatusPie: DeviceStatusItem[];
  kpiTimeSeries: Record<string, KPITimeSeriesPoint[]>;
  alarmTypePie: AlarmTypeItem[];
  deviceByRegion: RegionDeviceStats[];
  topAlarmDevices: TopAlarmDevice[];
}

// ============================================================================
// 后端 API 响应类型（用于映射和类型安全）
// ============================================================================

/**
 * 后端设备状态统计
 */
export interface BackendDeviceStats {
  total: number;
  online: number;
  offline: number;
  alarm: number;
}

/**
 * 后端告警状态统计
 */
export interface BackendAlarmStats {
  critical: number;
  major: number;
  minor: number;
  warning: number;
  total: number;
}

/**
 * 后端 KPI 概览（动态字段）
 */
export interface BackendKPIOverview {
  [key: string]: number | undefined;
}

/**
 * 后端近期告警
 */
export interface BackendRecentAlarm {
  device_sn: string;
  technology: string;
  device_name: string;
  alarm_count: number;
  severity: string;
}

/**
 * 后端 Dashboard 汇总响应
 */
export interface BackendDashboardSummary {
  device_stats: BackendDeviceStats;
  alarm_stats: BackendAlarmStats;
  kpi_overview: BackendKPIOverview;
  kpi_deltas: Record<string, BackendKPIDelta>;  // KPI趋势数据（新增）
  recent_alarms: BackendRecentAlarm[];
	pm_slot_health: BackendPMSlotHealth[];
  timestamp: string;
}

export interface BackendPMSlotHealth {
  slot_end: string;
  technology: string;
  carrier: string;
  expected_devices: number;
  received_devices: number;
  coverage_ratio: number;
  status: PMSlotHealth['status'];
  evaluated_at: string;
}

/**
 * 后端 KPI 趋势增量数据
 */
export interface BackendKPIDelta {
  current_value: number;
  previous_value: number;
  change_percent: number;
  trend: string;  // "up" | "down" | "stable"
  compare_type: string;  // "yesterday" | "last_week"
  has_comparison: boolean;
}

/**
 * 后端告警趋势项
 */
export interface BackendAlarmTrendItem {
  date: string;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

/**
 * 后端设备状态映射（用于饼图）
 */
export type BackendDeviceStatusMap = Record<string, number>;

/**
 * 单个制式的设备状态计数
 */
export interface BackendDeviceStatusCounts {
  online: number;
  offline: number;
  alarm: number;
}

/**
 * 后端按制式分组的设备状态
 */
export type BackendDeviceStatusByType = Record<string, BackendDeviceStatusCounts>;

/**
 * 后端 KPI 趋势项
 */
export interface BackendKPITrendItem {
  time: string;
  value: number;
}

/**
 * 后端 KPI 趋势对比响应
 */
export interface BackendKPITrendComparison {
  current: BackendKPITrendItem[];
  compare: BackendKPITrendItem[];
  metadata: {
    kpi_name: string;
    compare_type: string; // 后端返回普通字符串，需要断言为字面量类型
    change_percent?: number;
  };
}

/**
 * 前端使用的 KPI 趋势对比类型（正确的字面量类型）
 */
export interface KPITrendComparison {
  current: Array<{ time: string; value: number }>;
  compare: Array<{ time: string; value: number }>;
  metadata: {
    kpi_name: string;
    compare_type: 'yesterday' | 'last_week';
    change_percent?: number;
  };
}

/**
 * 后端区域统计项
 */
export interface BackendRegionStatsItem {
  region: string;
  device_count: number;
  online_count: number;
  alarm_count: number;
}

/**
 * 后端 Widget 布局
 */
export interface BackendWidgetLayout {
  id: string;
  user_id: string;
  layout: unknown;
  created_at: string;
  updated_at: string;
}

/**
 * 后端告警类型饼图项
 */
export interface BackendAlarmTypePieItem {
  name: string;
  value: number;
}

/**
 * 后端 KPI 时序数据条目
 */
export interface BackendKPITimeSeriesEntry {
  time: string;
  value: number;
}

/**
 * 后端 KPI 时序响应
 */
export type BackendKPITimeSeriesResponse = Record<string, BackendKPITimeSeriesEntry[]>;

// ============================================================================
// KPI 时序数据类型
// ============================================================================

/**
 * KPI 时序请求参数
 */
export interface KPITimeSeriesRequest {
  kpi_names?: string[];
  start_time?: string;
  end_time?: string;
  granularity?: '15min' | 'hourly' | 'daily';
  device_type?: string;
}

/**
 * KPI 时序数据点（对象格式）
 */
export interface KPITimeSeriesDataPoint {
  time: string;
  values: Record<string, number>;
}

/**
 * KPI 时序响应
 */
export interface KPITimeSeriesResponse {
  data: KPITimeSeriesDataPoint[];
  metadata: {
    start_time: string;
    end_time: string;
    granularity: string;
    kpi_names: string[];
  };
}

// ============================================================================
// KPI 趋势对比类型（v3.5 新增）
// ============================================================================

/**
 * KPI 趋势对比数据
 * 用于表示"今日 vs 昨日"或"本周 vs 上周"的对比结果
 */
export interface TrendComparisonData {
  /** 当前时段数据 */
  current: Array<{ time: string; value: number }>;
  /** 对比时段数据 */
  compare: Array<{ time: string; value: number }>;
  /** 元数据 */
  metadata: {
    kpi_name: string;
    compare_type: 'yesterday' | 'last_week';
    /** 变化百分比 (正数表示上升，负数表示下降) */
    change_percent?: number;
  };
}

/**
 * 多 KPI 趋势对比数据
 * key 为 KPI 名称，value 为对应的对比数据
 */
export interface MultiTrendComparisonData {
  [kpiName: string]: TrendComparisonData;
}

/**
 * KPI 趋势对比请求参数
 */
export interface KPITrendComparisonRequest {
  kpi_name: string;
  start_time?: string;
  end_time?: string;
  compare_with?: 'yesterday' | 'last_week';
}

/**
 * KPI 趋势对比响应（后端返回格式）
 */
export interface KPITrendComparisonResponse {
  current: Array<{ time: string; value: number }>;
  compare: Array<{ time: string; value: number }>;
  metadata: {
    kpi_name: string;
    compare_type: string;
    change_percent?: number;
  };
}

// ============================================================================
// Hook 返回类型
// ============================================================================

/**
 * useKPITrendComparisonV2 Hook 返回类型
 */
export interface UseKPITrendComparisonV2Result {
  /** 对比数据 */
  data: TrendComparisonData | undefined;
  /** 是否加载中 */
  isLoading: boolean;
  /** 错误数组（可能包含多个请求的错误） */
  errors: Array<Error | unknown>;
}

/**
 * useMultiKPITrendComparison Hook 返回类型
 */
export interface UseMultiKPITrendComparisonResult {
  /** 多 KPI 对比数据字典 */
  data: MultiTrendComparisonData | undefined;
  /** 是否加载中 */
  isLoading: boolean;
  /** 错误数组 */
  errors: Array<Error | unknown>;
}

// ============================================================================
// 时间范围参数类型（内部使用）
// ============================================================================

/**
 * KPI 时序查询参数（内部使用）
 */
export interface KPITimeSeriesParams {
  kpi_names: string[];
  start_time: string;
  end_time: string;
  granularity?: DashboardKPIGranularity;
}

export type DashboardKPIGranularity = 'hourly' | 'daily' | 'weekly';

export type DashboardProgressState = 'available' | 'unavailable' | 'not_applicable';

export interface DashboardPeriodProgress {
  taskId: string;
  taskVersionId: string;
  granularity: DashboardKPIGranularity;
  windowStart: string;
  windowEnd: string;
  entityKey: string;
  revision: number;
  versionEffectiveFrom: string;
  versionEffectiveTo: string | null;
  receivedSlots: number;
  expectedSlots: number;
  versionExpectedSlots: number;
  coverageRatio: number;
  versionSliceComplete: boolean;
  periodComplete: boolean;
  state: 'partial';
}

export interface DashboardKPITimeSeriesSnapshot {
  series: Record<string, Array<[string, number]>>;
  periodProgress: DashboardPeriodProgress[];
  progressState: DashboardProgressState;
}

/**
 * 时间范围计算结果（内部使用）
 */
export interface TimeRangesResult {
  currentParams: KPITimeSeriesParams;
  compareParams: KPITimeSeriesParams;
}

// ============================================================================
// Dashboard Widget 类型
// ============================================================================

/**
 * Dashboard Widget 项
 */
export interface DashboardWidgetItem {
  id: string;
  type: string;
  title: string;
  config?: Record<string, unknown>;
}

/**
 * Widget 布局配置
 */
export interface WidgetLayout {
  widgets: DashboardWidgetItem[];
  layout: {
    columns: number;
    rows: number;
  };
}

// ============================================================================
// 告警效率指标类型（Phase 2 新增）
// ============================================================================

/**
 * 每日效率趋势数据点
 */
export interface DailyEfficiencyTrend {
  /** 日期 */
  date: string;
  /** 当日平均确认时间（分钟）- 可能为空（当日无告警时） */
  avg_acknowledge_minutes: number | null;
  /** 当日平均解决时间（分钟）- 可能为空（当日无告警时） */
  avg_resolve_minutes: number | null;
}

/**
 * 告警处理效率指标
 * 包含 MTTA（平均确认时间）、MTTR（平均解决时间）、确认率、清除率等关键运维指标
 */
export interface EfficiencyMetrics {
  /** 告警级别 */
  severity: string;
  /** 已确认告警数量 */
  acknowledged_count: number;
  /** 已清除告警数量 */
  cleared_count: number;
  /** 总告警数量 */
  total_count: number;
  /** 平均确认时间（MTTA，分钟）- 可能为空（无告警数据时） */
  avg_acknowledge_minutes: number | null;
  /** 平均解决时间（MTTR，分钟）- 可能为空（无告警数据时） */
  avg_resolve_minutes: number | null;
  /** 确认率（百分比）- 可能为空（无告警数据时） */
  acknowledge_rate: number | null;
  /** 清除率（百分比）- 可能为空（无告警数据时） */
  clear_rate: number | null;
  /** 近7天趋势数据 */
  daily_trend: DailyEfficiencyTrend[];
}

/**
 * 告警热度图数据
 * 按星期几和小时统计告警数量
 */
export interface HeatmapData {
  /** 7天数据，0=周一, 6=周日 */
  days_of_week: DayOfWeekData[];
  /** 最大告警数（用于热力图颜色范围） */
  max_count: number;
}

/**
 * 一天24小时的告警数量分布
 */
export interface DayOfWeekData {
  /** 星期几，0=周一, 6=周日 */
  day: number;
  /** 24小时告警数量，索引0=00:00-00:59, 23=23:00-23:59 */
  hours: number[];
}

/**
 * 按严重程度分组的告警热度图数据
 */
export interface AlarmHeatmapBySeverity {
  /** 告警级别 */
  severity: string;
  /** 热度图数据 */
  data: HeatmapData;
}

// ============================================================================
// Dashboard KPI 动态定义（issue #213 Phase1 / #227）
//
// 后端 GET /dashboard/kpi/definitions 返回首页全部 KPI 的定义：可读 symbolic key +
// 后端 K 编号 + 中文名 + 单位，按制式/Panel 分组。前端据此动态加载指标列表；取数仍用
// symbolic key（后端别名层翻成 K 编号查 pm_metrics、按 symbolic key 回填）。
//
// 字段为 snake_case：HTTP 响应拦截器只解信封、不做 camelCase 转换，故与后端 JSON 一致。
// ============================================================================

/**
 * 单个 KPI 的完整定义（一条 symbolic key）。
 */
export interface KPIDefinitionItem {
  /** 前端可读 symbolic key（取数时传给后端，经别名层翻成 K 编号）。 */
  key: string;
  /** 指标库编号（pm_metrics.metric_path 落库值）；none 项为空字符串。 */
  k_code: string;
  /** 中文名（来自 indicator 库；缺则空）。 */
  cn_name: string;
  /** 单位（如 % / Mbps / MByte；缺则空）。 */
  unit: string;
  /** Dashboard Panel 归类（traffic / availability / utilization / accessibility / retainability / mobility）。 */
  panel: string;
  /** 该映射待领域复核（medium 置信度或 none 硬缺口）。 */
  needs_review: boolean;
  /** 库内是否有对应 KPI（false = none 项，前端可置灰/隐藏）。 */
  available: boolean;
}

/**
 * 单个制式的 KPI 定义集合。
 */
export interface KPITechDefinitions {
  /** 制式：lte / nr / gsm。 */
  tech: string;
  /** 该制式全部 KPI 定义（按 Panel 归并）。 */
  items: KPIDefinitionItem[];
}

/**
 * GET /dashboard/kpi/definitions 的响应体。
 */
export interface KPIDefinitionsResponse {
  /** 按制式分组的 KPI 定义。 */
  technologies: KPITechDefinitions[];
  /** KPI 定义总条数。 */
  total: number;
}

// ============================================================================
// Dashboard KPI 全局布局（issue #213 S2）
//
// 首页 KPI 折线图区从「写死的 kpi-config.ts」改为读后端全局布局接口
// （GET /dashboard/kpi-layout?tech=lte|nr|gsm）。后端按制式各存一行，
// 布局体（panels 数组）原样透传。读不到/为空/出错时前端回退内置默认。
//
// 后端 JSON 为 snake_case（updated_at / updated_by）；panel 内字段 chartType 为
// camelCase（seed 即如此），HTTP 响应拦截器不做 camelCase 转换，故 BackendXxx 与
// 后端 JSON 字段一致。
// ============================================================================

/**
 * 单张图的布局（后端 panels 数组的一项，原始 JSON 形状）。
 */
export interface BackendKPILayoutPanel {
  /** 图标题（i18n key，与前端 PANEL_LABELS 对齐）。 */
  title: string;
  /** 该图要画的指标 symbolic key 列表。 */
  metrics: string[];
  /** 网格 X 坐标（12 列网格）。 */
  x: number;
  /** 网格 Y 坐标。 */
  y: number;
  /** 网格宽度（半宽 6 / 满宽 12）。 */
  w: number;
  /** 网格高度。 */
  h: number;
  /** 图类型（本版恒为 line）。 */
  chartType: string;
}

/**
 * 单个制式的全局布局（GET /dashboard/kpi-layout 响应，原始 JSON 形状）。
 */
export interface BackendKPILayout {
  /** 制式：lte / nr / gsm。 */
  tech: string;
  /** 布局体：panels 数组。 */
  layout: {
    panels: BackendKPILayoutPanel[];
  };
  /** 最近保存时间（RFC3339 串）。 */
  updated_at: string;
  /** 最近保存的管理员用户 ID（seed 灌入的初始行为空）。 */
  updated_by?: string | null;
}

/**
 * 单张图的布局（前端使用形状，与后端 panel 等价，统一 camelCase）。
 */
export interface KPILayoutPanel {
  /** 图标题（i18n key）。 */
  title: string;
  /** 该图要画的指标 symbolic key 列表。 */
  metrics: string[];
  /** 网格 X 坐标。 */
  x: number;
  /** 网格 Y 坐标。 */
  y: number;
  /** 网格宽度。 */
  w: number;
  /** 网格高度。 */
  h: number;
  /** 图类型（本版恒为 line）。 */
  chartType: string;
}

/**
 * 单个制式的全局布局（前端使用形状）。
 */
export interface KPILayout {
  /** 制式：lte / nr / gsm。 */
  tech: string;
  /** 该制式的图列表。 */
  panels: KPILayoutPanel[];
  /** 最近保存时间（RFC3339 串）。 */
  updatedAt: string;
}
