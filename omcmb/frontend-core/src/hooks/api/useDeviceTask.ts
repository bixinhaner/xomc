import { useQuery } from '@tanstack/react-query';
import { deviceTaskApi } from '../../services/api/deviceTaskApi';
import { isDeviceTaskTerminal } from '../../types/deviceTask';

/**
 * 轮询设备任务状态(T-0146)。
 *
 * 输入:taskId(由 SetParameters/AddObject/DeleteObject 调用后从响应拿到)。
 * 输出:DeviceTask 含 status / sentAt / completedAt / errorMessage。
 *
 * 轮询策略:
 *  - 未到终态前持续轮询(默认 2 秒间隔)
 *  - status 进入 `completed/failed/expired/cancelled` 后停止轮询
 *  - 4xx 错误不重试(任务不存在等场景)
 *
 * 调用方典型用法:
 *   const { data: taskId, mutateAsync } = useUpdateParameters();
 *   await mutateAsync({...}); // 拿到 taskId
 *   const { data: task } = useDeviceTaskStatus(taskId);
 *   // task.status 进入终态后 React Query 自动停轮询
 */
export function useDeviceTaskStatus(
  taskId: string | undefined,
  options?: { intervalMs?: number },
) {
  const intervalMs = options?.intervalMs ?? 2000;
  return useQuery({
    queryKey: ['device-task', 'status', taskId],
    queryFn: () => deviceTaskApi.getTask(taskId as string),
    enabled: Boolean(taskId),
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      // 到达终态后停止轮询(返 false 关闭定时器)
      if (isDeviceTaskTerminal(status)) return false;
      return intervalMs;
    },
    // 后台标签页时也保持轮询(用户可能切到通知中心查任务)
    refetchIntervalInBackground: true,
  });
}
