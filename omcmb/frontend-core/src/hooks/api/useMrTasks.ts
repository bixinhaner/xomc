/**
 * F05 MR 测量任务 React Query Hooks。
 *
 * 走 useMock 切换：开发期 Mock 模式不依赖后端就能跑通 UI 流程；
 * 联调期切回 real API 验证完整链路。
 *
 * 查询键约定：['mrTask', '<action>', ...params]，层级与其他模块（MR / 设备等）一致。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { useMock } from '../../services/apiSwitch';
import { mrTaskApi } from '../../services/api/mrTaskApi';
import { mrTaskService } from '../../mock/services/mrTaskService';
import type {
  CreateMRTaskRequest,
  MRTaskListFilter,
  MRTaskProgressFilter,
} from '../../types/mrTask';

export function useMRTasks(filter: MRTaskListFilter) {
  return useQuery({
    queryKey: ['mrTask', 'list', filter],
    queryFn: () => (useMock ? mrTaskService.list(filter) : mrTaskApi.list(filter)),
    // 列表轮询：scheduler 30s 周期推 waitting→on / on→off，progress 由 SPV / 心跳推。
    // 10s 与 useMRTaskProgress 同步节拍，UI 不需要用户手动刷新就能看到状态变更。
    refetchInterval: 10_000,
  });
}

export function useMRTask(taskId: string | undefined) {
  return useQuery({
    queryKey: ['mrTask', 'detail', taskId],
    queryFn: () => {
      if (!taskId) throw new Error('useMRTask: taskId is required');
      return useMock ? mrTaskService.get(taskId) : mrTaskApi.get(taskId);
    },
    enabled: Boolean(taskId),
    // 详情抽屉也加上 10s 轮询，跟列表 + 进度表节拍一致。
    refetchInterval: 10_000,
  });
}

export function useMRTaskProgress(taskId: string | undefined, filter: MRTaskProgressFilter) {
  return useQuery({
    queryKey: ['mrTask', 'progress', taskId, filter],
    queryFn: () => {
      if (!taskId) throw new Error('useMRTaskProgress: taskId is required');
      return useMock
        ? mrTaskService.listProgress(taskId, filter)
        : mrTaskApi.listProgress(taskId, filter);
    },
    enabled: Boolean(taskId),
    // 进度抽屉打开时按 10s 轮询，避免用户手动刷新
    refetchInterval: 10_000,
  });
}

export function useCreateMRTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateMRTaskRequest) =>
      useMock ? mrTaskService.create(req) : mrTaskApi.create(req),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['mrTask', 'list'] });
    },
  });
}

export function useStopMRTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: string) =>
      useMock ? mrTaskService.stop(taskId) : mrTaskApi.stop(taskId),
    onSuccess: (_data, taskId) => {
      void qc.invalidateQueries({ queryKey: ['mrTask', 'list'] });
      void qc.invalidateQueries({ queryKey: ['mrTask', 'detail', taskId] });
    },
  });
}

export function useDeleteMRTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: string) =>
      useMock ? mrTaskService.delete(taskId) : mrTaskApi.delete(taskId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['mrTask', 'list'] });
    },
  });
}
