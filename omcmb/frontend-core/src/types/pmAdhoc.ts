/**
 * T-0164-P7 / G7 自定义聚合任务前端类型 + Backend wire types。
 */

export type AdhocMode = 'oneshot' | 'continuous';
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
  status: AdhocStatus;
  progress: number;
  creator: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAdhocTaskInput {
  name: string;
  mode: AdhocMode;
  cronExpr?: string;
  deviceSns: string[];
  metricPaths: string[];
  granularities: string[];
  windowStart: string;
  windowEnd: string;
}

export interface AdhocResultRow {
  id: string;
  taskId: string;
  deviceOui: string;
  deviceSn: string;
  metricPath: string;
  metricType: string;
  metricValue: number;
  statisType?: string;
  granularity: string;
  time: string;
  startTime: string;
  endTime: string;
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
  status: string;
  progress: number;
  creator: string;
  created_at: string;
  updated_at: string;
}

export interface BackendAdhocResultRow {
  id: string;
  task_id: string;
  device_oui: string;
  device_sn: string;
  metric_path: string;
  metric_type: string;
  metric_value: number;
  statis_type?: string;
  granularity: string;
  time: string;
  start_time: string;
  end_time: string;
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
    status: b.status as AdhocStatus,
    progress: b.progress,
    creator: b.creator,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

export function mapBackendAdhocResult(b: BackendAdhocResultRow): AdhocResultRow {
  return {
    id: b.id,
    taskId: b.task_id,
    deviceOui: b.device_oui,
    deviceSn: b.device_sn,
    metricPath: b.metric_path,
    metricType: b.metric_type,
    metricValue: b.metric_value,
    statisType: b.statis_type,
    granularity: b.granularity,
    time: b.time,
    startTime: b.start_time,
    endTime: b.end_time,
  };
}
