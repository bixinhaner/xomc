/**
 * KPI-EXPORT（T4）导出入参组装纯函数。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §3.2 / §6.3
 * 实施：~/Documents/notes/PM功能设计/kpi-export-impl-plan-20260604.md T4。
 *
 * 仪表盘「导出」与 adhoc 任务详情「导出」各组一份 params(jsonb)，键名 snake_case，
 * 与后端 export 模块的 DashboardParams / AdhocParams 字段一一对齐（internal/pm/export/params.go）：
 *   - dashboard：granularity / dimension / device_sns / metric_paths / metric_type /
 *                technologies / start_time / end_time。
 *   - adhoc：task_id（+ 可选 start_time / end_time 二次时窗）。
 *
 * 时间统一用 RFC3339（ISO8601），与后端 time.Parse(time.RFC3339) 对齐。
 *
 * 注：小区/PLMN（object_ldn）白名单本期 OUT-OF-SCOPE——后端 DashboardParams 暂无对应字段、
 * applyDeviceExportFilters 也不按 object_ldn 过滤，故前端不发送该键（发了会被静默丢弃，反而误导）。
 * 仪表盘下钻定格某小区/PLMN 后导出，CSV 仍含所选设备的全部小区/PLMN。
 * 若后续要支持，需后端补 DashboardParams.ObjectLDNs 字段 + 过滤逻辑，再在此回填该键。
 */

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
 * - metric_type 固定 counter（设备列表取原始计数器值，与聚合查询同口径）。
 * - 小区/PLMN 白名单本期不发送（后端无对应过滤，见文件头注）。
 */
export function buildDashboardExportParams(
  sel: DashboardExportSelection,
): Record<string, unknown> {
  return {
    granularity: sel.granularity,
    dimension: 'device',
    device_sns: sel.deviceSns,
    metric_paths: sel.metricPaths,
    metric_type: 'counter',
    technologies: sel.technology ? [sel.technology] : [],
    start_time: sel.startTime,
    end_time: sel.endTime,
  };
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

/** 默认导出任务名：KPI导出_{来源}_{时间戳}。后端不传也会自动生成，这里前端给个可读名。 */
export function defaultExportTaskName(
  source: 'dashboard' | 'adhoc',
  now: Date = new Date(),
): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  const ts = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(
    now.getHours(),
  )}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
  const label = source === 'dashboard' ? '仪表盘' : '任务结果';
  return `KPI导出_${label}_${ts}`;
}
