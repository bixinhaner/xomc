import http from '../http';
import type { DeviceTask } from '../../types/deviceTask';

/** 后端 GET /devices/tasks/:task_id 响应(信封拆出 data 后的形态)。 */
interface BackendDeviceTask {
  id: string;
  device_sn: string;
  method: string;
  status: string;
  retry_count: number;
  max_retries: number;
  created_at: string;
  sent_at?: string;
  completed_at?: string;
  expires_at?: string;
  error_code?: number;
  error_message?: string;
}

function mapBackendDeviceTask(bt: BackendDeviceTask): DeviceTask {
  return {
    id: bt.id,
    deviceSn: bt.device_sn,
    method: bt.method,
    status: bt.status as DeviceTask['status'],
    retryCount: bt.retry_count ?? 0,
    maxRetries: bt.max_retries ?? 0,
    createdAt: bt.created_at,
    sentAt: bt.sent_at,
    completedAt: bt.completed_at,
    expiresAt: bt.expires_at,
    errorCode: bt.error_code,
    errorMessage: bt.error_message,
  };
}

/**
 * 设备任务 API(T-0146)。
 *
 * 后端 internal/task/handler.go::GetTask;路由 GET /api/v1/devices/tasks/:task_id。
 * 前端 useDeviceTaskStatus Hook 用它轮询任务状态,直到终态后停止。
 */
export const deviceTaskApi = {
  async getTask(taskId: string): Promise<DeviceTask> {
    const { data } = await http.get<BackendDeviceTask>(`/devices/tasks/${taskId}`);
    return mapBackendDeviceTask(data);
  },
};
