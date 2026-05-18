/**
 * 设备任务类型(T-0146)。
 *
 * 后端 internal/task/model.go::Task 的前端镜像;
 * 用于 useDeviceTaskStatus Hook 轮询 GET /api/v1/devices/tasks/:task_id 显示真实 CPE 应答状态。
 */

/** 任务状态机(后端 internal/task/model.go TaskStatus)。 */
export type DeviceTaskStatus =
  | 'pending'   // 等待下发(已入队 ACS)
  | 'sent'      // 已发送给 CPE(等待应答)
  | 'completed' // CPE 执行成功
  | 'failed'    // CPE 执行失败
  | 'expired'   // 任务超时
  | 'cancelled'; // 任务取消

/** 终态:不再变化,前端可停止轮询。 */
export const DEVICE_TASK_TERMINAL_STATUSES: ReadonlyArray<DeviceTaskStatus> = [
  'completed',
  'failed',
  'expired',
  'cancelled',
];

export function isDeviceTaskTerminal(status: DeviceTaskStatus | undefined): boolean {
  if (!status) return false;
  return DEVICE_TASK_TERMINAL_STATUSES.includes(status);
}

/** 设备任务实体(后端 Task 关键字段镜像)。 */
export interface DeviceTask {
  id: string;
  deviceSn: string;
  method: string;
  status: DeviceTaskStatus;
  retryCount: number;
  maxRetries: number;
  createdAt: string;
  sentAt?: string;
  completedAt?: string;
  expiresAt?: string;
  errorCode?: number;
  errorMessage?: string;
}
