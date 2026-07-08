/**
 * KPI-EXPORT（KPI 数据导出）前端类型 + Backend wire types。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §6.1
 * 后端：internal/pm/export（一张表 pm_kpi_export_tasks 喂两个视图——任务管理看状态、文件管理下载）。
 *
 * http.ts 响应拦截器只拆 envelope（{ret,msg,data}）不做命名转换，故后端响应保持 snake_case；
 * 这里按 BackendXxx → mapBackendXxx → 前端 camelCase 的项目惯例落地。
 */

/** 导出来源：dashboard 仪表盘曲线 / kpi_query 指标查询页 / adhoc 聚合任务结果。 */
export type KpiExportSource = 'dashboard' | 'kpi_query' | 'adhoc';

/** 导出任务生命周期状态。 */
export type KpiExportStatus = 'pending' | 'running' | 'succeeded' | 'failed';

/**
 * 导出任务（任务管理 Tab 视图）。
 * 文件管理 Tab 复用同一类型（只列 status=succeeded 且文件就绪的行）。
 */
export interface KpiExportTask {
  id: string;
  taskName: string;
  sourceType: KpiExportSource;
  /** 导出范围参数（JSON 原文，前端不解析，重试时原样回传）。 */
  params: Record<string, unknown>;
  format: string;
  status: KpiExportStatus;
  rowCount: number;
  fileSize: number;
  error?: string;
  createUser: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

/** 建导出任务入参。 */
export interface CreateKpiExportInput {
  sourceType: KpiExportSource;
  /** 导出范围参数（dashboard 维度/指标/时窗/粒度，adhoc task_id 等）。 */
  params: Record<string, unknown>;
  /** 可选任务名；不传后端自动生成 KPI导出_{来源}_{时间戳}。 */
  taskName?: string;
}

// ── Backend wire types（snake_case，对齐 handler.go taskResponseDTO）─────────

export interface BackendKpiExportTask {
  id: string;
  task_name: string;
  source_type: string;
  params?: Record<string, unknown>;
  format: string;
  status: string;
  row_count: number;
  file_size: number;
  error?: string;
  create_user: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
}

export function mapBackendKpiExportTask(b: BackendKpiExportTask): KpiExportTask {
  return {
    id: b.id,
    taskName: b.task_name,
    sourceType: (b.source_type as KpiExportSource) ?? 'dashboard',
    params: b.params ?? {},
    format: b.format,
    status: (b.status as KpiExportStatus) ?? 'pending',
    rowCount: b.row_count ?? 0,
    fileSize: b.file_size ?? 0,
    error: b.error || undefined,
    createUser: b.create_user,
    createdAt: b.created_at,
    startedAt: b.started_at || undefined,
    finishedAt: b.finished_at || undefined,
  };
}
