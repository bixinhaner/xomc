/**
 * F05 MR 测量任务 — REST API 服务。
 *
 * 后端：`omcgo/internal/mr/task/handler.go` 注册在 `/api/v1/mr/tasks` 下，
 * 资源组 "pm"（与现有 `/mr/files` 等共享权限范围）。
 *
 * 设计要点：
 *   - 业务层从这里拿到 Backend* snake_case → mapBackendXxx → 前端 camelCase
 *   - http.ts 拦截器**不会**自动转换路径参数与 query；只转 body / response key
 *   - 部分接口需要 operator_code 走 query（GET/DELETE）或 body（POST）
 */

import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  CreateMRTaskRequest,
  MRHealthStatus,
  MRProgressStatus,
  MRTask,
  MRTaskListFilter,
  MRTaskProgress,
  MRTaskProgressFilter,
  MRTaskStatus,
} from '../../types/mrTask';

// ---------- Backend snake_case 形状 ----------

interface BackendMRTask {
  task_id: string;
  task_name: string;
  mr_type: string;
  statis_period: string;
  report_period: string;
  start_time: string;
  end_time?: string;
  task_status: MRTaskStatus;
  task_result?: string;
  creator: string;
  target_device_sns: string[];
  created_at: string;
  updated_at: string;
}

interface BackendMRProgress {
  id: string;
  task_id: string;
  small_cell_code: string;
  serial_number: string;
  host_name?: string;
  progress_status: MRProgressStatus;
  health_status: MRHealthStatus;
  fault_code?: string;
  last_heartbeat?: string;
  missed_heartbeat: number;
  created_at: string;
  updated_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ---------- map functions ----------

function mapBackendMRTask(b: BackendMRTask): MRTask {
  return {
    taskId: b.task_id,
    taskName: b.task_name,
    mrType: b.mr_type,
    statisPeriod: b.statis_period as MRTask['statisPeriod'],
    reportPeriod: b.report_period as MRTask['reportPeriod'],
    startTime: b.start_time,
    endTime: b.end_time,
    taskStatus: b.task_status,
    taskResult: b.task_result,
    creator: b.creator,
    targetDeviceSns: b.target_device_sns ?? [],
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapBackendMRProgress(b: BackendMRProgress): MRTaskProgress {
  return {
    id: b.id,
    taskId: b.task_id,
    smallCellCode: b.small_cell_code,
    serialNumber: b.serial_number,
    hostName: b.host_name,
    progressStatus: b.progress_status,
    healthStatus: b.health_status,
    faultCode: b.fault_code,
    lastHeartbeat: b.last_heartbeat,
    missedHeartbeat: b.missed_heartbeat,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

// ---------- exported service ----------

export const mrTaskApi = {
  async create(req: CreateMRTaskRequest): Promise<MRTask> {
    // 后端 handler 的 binding tag 走 snake_case，前端显式构造 body
    const body = {
      task_name: req.taskName,
      mr_type: req.mrType,
      statis_period: req.statisPeriod,
      report_period: req.reportPeriod,
      start_time: req.startTime,
      end_time: req.endTime,
      target_device_sns: req.targetDeviceSns,
    };
    const { data } = await http.post<BackendMRTask>('/mr/tasks', body);
    return mapBackendMRTask(data);
  },

  async list(
    filter: MRTaskListFilter,
  ): Promise<PageResponse<MRTask>> {
    const query: Record<string, unknown> = {
      page: filter.page,
      page_size: filter.pageSize,
    };
    if (filter.status) query.status = filter.status;
    if (filter.keyword) query.keyword = filter.keyword;
    if (filter.sortBy) query.sort_by = filter.sortBy;
    if (filter.sortDir) query.sort_dir = filter.sortDir;

    const { data } = await http.get<BackendListResponse<BackendMRTask>>(
      '/mr/tasks',
      { params: query },
    );
    return {
      items: (data.items || []).map(mapBackendMRTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async get(taskId: string): Promise<MRTask> {
    const { data } = await http.get<BackendMRTask>(`/mr/tasks/${taskId}`);
    return mapBackendMRTask(data);
  },

  async stop(taskId: string): Promise<void> {
    await http.post(`/mr/tasks/${taskId}/stop`);
  },

  async delete(taskId: string): Promise<void> {
    await http.delete(`/mr/tasks/${taskId}`);
  },

  async listProgress(
    taskId: string,
    filter: MRTaskProgressFilter,
  ): Promise<PageResponse<MRTaskProgress>> {
    const query: Record<string, unknown> = {
      page: filter.page,
      page_size: filter.pageSize,
    };
    if (filter.status) query.status = filter.status;
    if (filter.health) query.health = filter.health;

    const { data } = await http.get<BackendListResponse<BackendMRProgress>>(
      `/mr/tasks/${taskId}/progress`,
      { params: query },
    );
    return {
      items: (data.items || []).map(mapBackendMRProgress),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },
};

// PageRequest type is re-exported for hook layer convenience.
export type { PageRequest };
