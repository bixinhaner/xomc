/**
 * T-0164-P7 / G7 自定义聚合任务前端类型 + Backend wire types。
 */

export type AdhocMode = 'oneshot' | 'continuous';

/**
 * 聚合维度（T-0185 扩展至 6 维，对齐后端 binding oneof）：
 *   - 'device' 每设备保留一条结果（默认，老任务兼容）
 *   - 'aggregate_group' N 个 SN 临时组聚合成一条（按时间桶 + LDN GROUP BY）
 *   - 'product' 按产品分组（T-0182，每产品一条线，全量聚合）
 *   - 'band' 按频段分组（T-0183，自动分组）
 *   - 'network' 全网汇总一条总线（T-0184）
 *   - 'device_group' 按设备组分组（T-0184，每组一条线，全量聚合）
 */
export type AdhocDimension =
  | 'device'
  | 'aggregate_group'
  | 'product'
  | 'band'
  | 'network'
  | 'device_group';
export type AdhocStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'canceled'
  | 'scheduled';

export interface AdhocTask {
  id: string;
  name: string;
  mode: AdhocMode;
  cronExpr?: string;
  deviceSns: string[];
  metricPaths: string[];
  granularities: string[];
  windowStart: string;
  windowEnd: string;
  dimension: AdhocDimension;
  // 制式 lte/nr/gsm，空=不限（T-0182，建后只读）
  technology?: string;
  // 内置任务标记（T-0182，前端列表分内置/自建区用）
  isBuiltin: boolean;
  // 非持续型过期天数（T-0182，默认 60）
  expireDays: number;
  status: AdhocStatus;
  progress: number;
  creator: string;
  createdAt: string;
  updatedAt: string;
  // T-0193 小区/PLMN 白名单：完整 object_ldn 字符串数组，仅 device/aggregate_group 维度生效；
  // 空/缺 = 不过滤（全小区，向后兼容旧任务）。
  objectLdns?: string[];
}

export interface CreateAdhocTaskInput {
  name: string;
  mode: AdhocMode;
  cronExpr?: string;
  deviceSns: string[];
  metricPaths: string[];
  granularities: string[];
  // T-0185：window 仅 oneshot 必填；continuous 不传 → 后端开窗滚动聚合。
  windowStart?: string;
  windowEnd?: string;
  dimension?: AdhocDimension;
  technology?: string;
  isBuiltin?: boolean;
  expireDays?: number;
  // T-0193 小区/PLMN 白名单（完整 object_ldn 字符串数组）。空/缺 = 不传 → 全小区（现状语义）。
  objectLdns?: string[];
}

export interface AdhocResultRow {
  id: string;
  taskId: string;
  deviceOui: string;
  deviceSn: string;
  metricPath: string;
  // KPI 行 metricPath 是 K 编号；displayName 为后端回填的友好名（PLMN 级带标记）。counter 行 = metricPath。
  displayName?: string;
  metricType: string;
  metricValue: number;
  statisType?: string;
  granularity: string;
  time: string;
  startTime: string;
  endTime: string;
  // T-0187 多线系列键来源：product 维度的产品 id、band/device_group/aggregate_group 维度的 object_ldn。
  productId?: string;
  objectLdn?: string;
}

/**
 * 运行历史一行（T-0186）：每次 worker 执行落一条。
 * run 是"单次执行视角"——continuous 任务整体状态可为 scheduled，但每次 run 终态只 running/succeeded/failed。
 */
export interface AdhocTaskRun {
  id: string;
  taskId: string;
  runSeq: number;
  granularity: string;
  dimension: string;
  windowStart?: string;
  windowEnd?: string;
  status: AdhocStatus;
  queuedAt?: string;
  startedAt: string;
  finishedAt?: string;
  error?: string;
  rowsTotal: number;
}

export interface AdhocProgressEvent {
  task_id: string;
  progress: number;
  granularity: string;
  rows: number;
}

export interface AdhocCompletedEvent {
  task_id: string;
  status: AdhocStatus;
  rows_total: number;
  error?: string;
}

// Backend wire types
export interface BackendAdhocTask {
  id: string;
  name: string;
  mode: string;
  cron_expr?: string;
  device_sns: string[];
  metric_paths: string[];
  granularities: string[];
  window_start: string;
  window_end: string;
  dimension?: string;
  technology?: string;
  is_builtin?: boolean;
  expire_days?: number;
  status: string;
  progress: number;
  creator: string;
  created_at: string;
  updated_at: string;
  // T-0193 任务白名单回吐。
  object_ldns?: string[];
}

export interface BackendAdhocResultRow {
  id: string;
  task_id: string;
  device_oui: string;
  device_sn: string;
  metric_path: string;
  display_name?: string;
  metric_type: string;
  metric_value: number;
  statis_type?: string;
  granularity: string;
  time: string;
  start_time: string;
  end_time: string;
  // T-0187：后端 resultDTO 已吐这两个分组键（handler.go），前端补透传。
  product_id?: string;
  object_ldn?: string;
}

export interface BackendAdhocTaskRun {
  id: string;
  task_id: string;
  run_seq: number;
  granularity: string;
  dimension: string;
  window_start?: string;
  window_end?: string;
  status: string;
  queued_at?: string;
  started_at: string;
  finished_at?: string;
  error?: string;
  rows_total: number;
}

export function mapBackendAdhocTaskRun(b: BackendAdhocTaskRun): AdhocTaskRun {
  return {
    id: b.id,
    taskId: b.task_id,
    runSeq: b.run_seq,
    granularity: b.granularity,
    dimension: b.dimension,
    windowStart: b.window_start || undefined,
    windowEnd: b.window_end || undefined,
    status: b.status as AdhocStatus,
    queuedAt: b.queued_at || undefined,
    startedAt: b.started_at,
    finishedAt: b.finished_at || undefined,
    error: b.error || undefined,
    rowsTotal: b.rows_total,
  };
}

export function mapBackendAdhocTask(b: BackendAdhocTask): AdhocTask {
  return {
    id: b.id,
    name: b.name,
    mode: b.mode as AdhocMode,
    cronExpr: b.cron_expr,
    deviceSns: b.device_sns,
    metricPaths: b.metric_paths,
    granularities: b.granularities,
    windowStart: b.window_start,
    windowEnd: b.window_end,
    dimension: (b.dimension as AdhocDimension) ?? 'device',
    technology: b.technology,
    isBuiltin: b.is_builtin ?? false,
    expireDays: b.expire_days ?? 60,
    status: b.status as AdhocStatus,
    progress: b.progress,
    creator: b.creator,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
    objectLdns: b.object_ldns && b.object_ldns.length > 0 ? b.object_ldns : undefined,
  };
}

export function mapBackendAdhocResult(b: BackendAdhocResultRow): AdhocResultRow {
  return {
    id: b.id,
    taskId: b.task_id,
    deviceOui: b.device_oui,
    deviceSn: b.device_sn,
    metricPath: b.metric_path,
    displayName: b.display_name || undefined,
    metricType: b.metric_type,
    metricValue: b.metric_value,
    statisType: b.statis_type,
    granularity: b.granularity,
    time: b.time,
    startTime: b.start_time,
    endTime: b.end_time,
    productId: b.product_id || undefined,
    objectLdn: b.object_ldn || undefined,
  };
}
