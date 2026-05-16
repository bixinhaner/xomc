// T-0137 / M1: TR069 报文跟踪 API 服务
import http from '../http';
import type {
  TraceTask,
  TraceMessage,
  TraceTaskListParams,
  TraceTaskListResponse,
  TraceMessageListParams,
  TraceMessageListResponse,
  CreateTraceTaskPayload,
  TraceTaskStatus,
  TraceDirection,
  TraceExportJob,
  TraceExportJobStatus,
} from '../../types/trace';

// ---------------------------------------------------------------------------
// Backend models (snake_case)
// ---------------------------------------------------------------------------

interface BackendTraceTask {
  id: string;
  device_sn: string;
  operator_code: string;
  status: string;
  start_time: string;
  expires_at: string;
  stopped_at?: string;
  purged_at?: string;
  created_by: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

interface BackendTraceMessage {
  id: string;
  captured_at: string;
  task_id: string;
  device_sn: string;
  direction: string;
  rpc_method?: string;
  cwmp_id?: string;
  session_id?: string;
  http_status?: number;
  payload_size_bytes: number;
  payload_inline?: string;
  payload_object_key?: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

function asStatus(v: string): TraceTaskStatus {
  if (v === 'running' || v === 'stopped' || v === 'purged') return v;
  return 'stopped';
}

function asDirection(v: string): TraceDirection {
  return v === 'out' ? 'out' : 'in';
}

function mapBackendTask(bt: BackendTraceTask): TraceTask {
  return {
    id: bt.id,
    deviceSn: bt.device_sn,
    operatorCode: bt.operator_code,
    status: asStatus(bt.status),
    startTime: bt.start_time,
    expiresAt: bt.expires_at,
    stoppedAt: bt.stopped_at,
    purgedAt: bt.purged_at,
    createdBy: bt.created_by,
    messageCount: bt.message_count ?? 0,
    createdAt: bt.created_at,
    updatedAt: bt.updated_at,
  };
}

function mapBackendMessage(bm: BackendTraceMessage): TraceMessage {
  return {
    id: bm.id,
    capturedAt: bm.captured_at,
    taskId: bm.task_id,
    deviceSn: bm.device_sn,
    direction: asDirection(bm.direction),
    rpcMethod: bm.rpc_method,
    cwmpId: bm.cwmp_id,
    sessionId: bm.session_id,
    httpStatus: bm.http_status,
    payloadSizeBytes: bm.payload_size_bytes ?? 0,
    payloadInline: bm.payload_inline,
    payloadObjectKey: bm.payload_object_key,
  };
}

// ---------------------------------------------------------------------------
// Trace API service
// ---------------------------------------------------------------------------

export const traceApi = {
  async listTasks(params: TraceTaskListParams = {}): Promise<TraceTaskListResponse> {
    const query: Record<string, unknown> = {};
    if (params.deviceSn) query.device_sn = params.deviceSn;
    if (params.status) query.status = params.status;
    if (params.operatorCode) query.operator_code = params.operatorCode;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.pageSize = params.pageSize;

    const { data } = await http.get<BackendListResponse<BackendTraceTask>>('/trace/tasks', {
      params: query,
    });
    return {
      items: (data.items || []).map(mapBackendTask),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTask(id: string): Promise<TraceTask> {
    const { data } = await http.get<BackendTraceTask>(`/trace/tasks/${id}`);
    return mapBackendTask(data);
  },

  async createTask(payload: CreateTraceTaskPayload): Promise<TraceTask> {
    const body: Record<string, unknown> = {
      device_sn: payload.deviceSn,
    };
    if (payload.durationMinutes !== undefined) {
      body.duration_minutes = payload.durationMinutes;
    }
    const { data } = await http.post<BackendTraceTask>('/trace/tasks', body);
    return mapBackendTask(data);
  },

  async stopTask(id: string, purge = false): Promise<TraceTask> {
    const { data } = await http.post<BackendTraceTask>(`/trace/tasks/${id}/stop`, {
      purge,
    });
    return mapBackendTask(data);
  },

  async listMessages(
    taskId: string,
    params: TraceMessageListParams = {}
  ): Promise<TraceMessageListResponse> {
    const query: Record<string, unknown> = {};
    if (params.direction) query.direction = params.direction;
    if (params.rpcMethod) query.rpc_method = params.rpcMethod;
    if (params.startTime) query.start_time = params.startTime;
    if (params.endTime) query.end_time = params.endTime;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.pageSize = params.pageSize;

    const { data } = await http.get<BackendListResponse<BackendTraceMessage>>(
      `/trace/tasks/${taskId}/messages`,
      { params: query }
    );
    return {
      items: (data.items || []).map(mapBackendMessage),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getActiveTaskBySN(sn: string): Promise<TraceTask | null> {
    const { data } = await http.get<BackendTraceTask | null>(
      `/trace/devices/${encodeURIComponent(sn)}/active-task`
    );
    return data ? mapBackendTask(data) : null;
  },

  // M2 异步导出：POST 触发 → 返回 job_id → 前端轮询 GET /exports/{job_id} 拿预签名 URL
  async requestExport(taskId: string): Promise<TraceExportJob> {
    const { data } = await http.post<BackendTraceExportJob>(`/trace/tasks/${taskId}/export`);
    return mapBackendExportJob(data);
  },

  async getExportJob(jobId: string): Promise<TraceExportJob> {
    const { data } = await http.get<BackendTraceExportJob>(`/trace/exports/${jobId}`);
    return mapBackendExportJob(data);
  },

  // M2-06: 拉取单条报文完整 payload（external 报文走 MinIO 回读）
  async getMessagePayload(taskId: string, msgId: string): Promise<string> {
    const { data } = await http.get<string>(
      `/trace/tasks/${taskId}/messages/${msgId}/payload`,
      { responseType: 'text', transformResponse: (v) => v }
    );
    return data ?? '';
  },
};

// ---------------------------------------------------------------------------
// Export job mapper
// ---------------------------------------------------------------------------

interface BackendTraceExportJob {
  id: string;
  task_id: string;
  requested_by: string;
  status: string;
  object_key?: string;
  object_bucket?: string;
  message_count: number;
  size_bytes: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  download_url?: string;
}

function asExportStatus(v: string): TraceExportJobStatus {
  if (v === 'queued' || v === 'running' || v === 'done' || v === 'failed') return v;
  return 'queued';
}

function mapBackendExportJob(bj: BackendTraceExportJob): TraceExportJob {
  return {
    id: bj.id,
    taskId: bj.task_id,
    requestedBy: bj.requested_by,
    status: asExportStatus(bj.status),
    objectKey: bj.object_key,
    objectBucket: bj.object_bucket,
    messageCount: bj.message_count ?? 0,
    sizeBytes: bj.size_bytes ?? 0,
    errorMessage: bj.error_message,
    startedAt: bj.started_at,
    completedAt: bj.completed_at,
    createdAt: bj.created_at,
    updatedAt: bj.updated_at,
    downloadUrl: bj.download_url,
  };
}
