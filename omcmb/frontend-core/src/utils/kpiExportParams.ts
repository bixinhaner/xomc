/**
 * KPI-EXPORT（T4）导出入参组装纯函数。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §3.2 / §6.3
 * 实施：~/Documents/notes/PM功能设计/kpi-export-impl-plan-20260604.md T4。
 *
 * 仪表盘「导出」与 adhoc 任务详情「导出」各组一份 params(jsonb)，键名 snake_case，
 * 与后端 export 模块的 DashboardParams / AdhocParams 字段一一对齐（internal/pm/export/params.go）：
 *   - dashboard：granularity / dimension / device_sns / metric_paths /
 *                technologies / start_time / end_time / object_ldns。
 *   - adhoc：task_id（+ 可选 start_time / end_time 二次时窗）。
 *
 * 时间统一用 RFC3339（ISO8601），与后端 time.Parse(time.RFC3339) 对齐。
 *
 * A3：不发送 metric_type——仪表盘出图本就不限指标类型（counter 与 ratio/kpi 混选混画），
 * 导出与出图同口径，由所选 metric_paths 隐含类型，后端 metric_type 为空即不过滤。
 *
 * A1：小区/PLMN（object_ldn）下钻白名单——下钻定格某小区/PLMN 后导出时回填 object_ldns，
 * 后端按白名单只导命中行（空则导该设备全部小区/PLMN，向后兼容）。
 *
 * 消费方：仪表盘页（DeviceListPane/TaskDashboardPane）、adhoc 结果页（AdhocResultPanel）、
 * 指标查询页（KPIQuery，复用 kpiQueryToDashboardSelection）。放 frontend-core 供多页面共享。
 */

import type { QueryTemplatePayload } from '../types/pmQuery';
import { deviceTypeToNetworkTech } from '../types/indicatorLibrary';

/** 仪表盘导出的当前筛选快照（设备列表 Pane 的提交态）。 */
export interface DashboardExportSelection {
  /** 制式：lte / nr / gsm（小写，对齐 devices.technology）。 */
  technology: string;
  /** 选中设备序列号（仪表盘按 SN 过滤，OUI 可省）。 */
  deviceSns: string[];
  /** 选中指标编号。 */
  metricPaths: string[];
  /** 粒度：15min / hourly / daily / weekly / monthly。 */
  granularity: string;
  /** 大时间段起（ISO8601）。 */
  startTime: string;
  /** 大时间段止（ISO8601）。 */
  endTime: string;
  /** A1：小区/PLMN 下钻白名单（空/缺席=不过滤，导全部小区/PLMN）。 */
  objectLdns?: string[];
  /** #599：星期过滤（0=周日..6=周六）。全选/缺席 = 不过滤。 */
  weekdays?: number[];
  /** #599：小时段过滤（0..23）。全选/缺席 = 不过滤。 */
  hours?: number[];
}

/** adhoc 导出的当前筛选快照。 */
export interface AdhocExportSelection {
  taskId: string;
  /** 可选大时间段起（ISO8601）。 */
  startTime?: string;
  /** 可选大时间段止（ISO8601）。 */
  endTime?: string;
}

/**
 * 组装仪表盘来源的 params(jsonb)。
 * - 维度固定 device（设备列表 Pane 是按设备出图）。
 * - A3：不发送 metric_type（与出图同口径，counter/kpi 都导，类型由 metric_paths 隐含）。
 * - A1：object_ldns 仅在非空时写入（下钻定格的小区/PLMN 白名单；空=导全部）。
 */
export function buildDashboardExportParams(
  sel: DashboardExportSelection,
): Record<string, unknown> {
  const params: Record<string, unknown> = {
    granularity: sel.granularity,
    dimension: 'device',
    device_sns: sel.deviceSns,
    metric_paths: sel.metricPaths,
    technologies: sel.technology ? [sel.technology] : [],
    start_time: sel.startTime,
    end_time: sel.endTime,
  };
  if (sel.objectLdns && sel.objectLdns.length > 0) {
    params.object_ldns = sel.objectLdns;
  }
  // #599：导出与出图同口径——星期/小时段传入导出 params。
  if (sel.weekdays && sel.weekdays.length > 0 && sel.weekdays.length < 7) {
    params.weekdays = sel.weekdays;
  }
  if (sel.hours && sel.hours.length > 0 && sel.hours.length < 24) {
    params.hours = sel.hours;
  }
  return params;
}

/** 组装 adhoc 来源的 params(jsonb)。可选二次时窗仅在非空时写入。 */
export function buildAdhocExportParams(
  sel: AdhocExportSelection,
): Record<string, unknown> {
  const params: Record<string, unknown> = {
    task_id: sel.taskId,
  };
  if (sel.startTime) params.start_time = sel.startTime;
  if (sel.endTime) params.end_time = sel.endTime;
  return params;
}

/**
 * 校验仪表盘导出筛选是否齐备（死判 T4-params-carried 要求设备/指标/时间/粒度均非空）。
 * 返回 null 表示通过；否则返回缺失项的语料键，调用方据此提示。
 */
export function validateDashboardExportSelection(
  sel: DashboardExportSelection,
): string | null {
  if (sel.deviceSns.length === 0) return 'perf.dashboard.selectAtLeastOneDevice';
  if (sel.metricPaths.length === 0) return 'perf.dashboard.selectAtLeastOneMetric';
  if (!sel.granularity) return 'kpiExport.export.missingGranularity';
  if (!sel.startTime || !sel.endTime) return 'kpiExport.export.missingTimeRange';
  return null;
}

interface ExportTaskNameOptions {
  locale?: string;
  /** 当前导出对象的页面可见名称，例如 adhoc 任务 / 聚合模板名称。 */
  subjectName?: string;
}

function safeFilenamePart(value: string): string {
  return value
    .trim()
    .split('')
    .map((ch) => {
      const code = ch.charCodeAt(0);
      return code <= 31 || code === 127 || /[\\/:*?"<>|;]/.test(ch) ? '_' : ch;
    })
    .join('')
    .replace(/\s+/g, '_')
    .replace(/_+/g, '_')
    .replace(/^_+|_+$/g, '');
}

/** 默认导出任务名。adhoc/result 导出会带上当前任务名，便于任务列表和下载文件反查来源。 */
export function defaultExportTaskName(
  source: 'dashboard' | 'device_view' | 'kpi_query' | 'adhoc',
  now: Date = new Date(),
  options: ExportTaskNameOptions = {},
): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  const ts = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(
    now.getHours(),
  )}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
  const isEnglish = options.locale?.toLowerCase().startsWith('en') ?? false;
  const label = isEnglish
    ? source === 'dashboard'
      ? 'Dashboard'
      : source === 'device_view'
        ? 'Device_Performance_View'
      : source === 'kpi_query'
        ? 'KPI_Query'
        : 'Result'
    : source === 'dashboard'
      ? '仪表盘'
      : source === 'device_view'
        ? '设备性能查看'
      : source === 'kpi_query'
        ? '指标查询'
        : '任务结果';
  const subject = options.subjectName ? safeFilenamePart(options.subjectName) : '';
  const parts = isEnglish && source === 'adhoc' && subject
    ? ['KPI', 'Export']
    : isEnglish
      ? ['KPI', 'Export', label]
      : ['KPI导出', label];
  if (subject) parts.push(subject);
  parts.push(ts);
  return parts.join('_');
}

/**
 * 指标查询页"最近一次查询快照"→ dashboard 导出筛选。维度固定 device（页面就是按设备查）。
 * - 指标查询页如做小区下钻，调用方会把 object_ldns 合并进返回的 selection，保证导出与页面一致。
 * - 不发 metric_type（与出图同口径，counter/kpi 由 metric_paths 隐含）。
 *   这两点由 buildDashboardExportParams 天然处理。
 */
export function kpiQueryToDashboardSelection(
  payload: QueryTemplatePayload,
  range: { start: string; end: string },
): DashboardExportSelection {
  return {
    technology: deviceTypeToNetworkTech(payload.deviceType ?? 'ENB'),
    deviceSns: payload.deviceSns,
    metricPaths: payload.metricPaths,
    granularity: payload.granularity,
    startTime: range.start,
    endTime: range.end,
  };
}
