// T-0137 / M2: TR069 报文跟踪 React Query Hooks + SSE 实时订阅
import { useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { traceApi } from '../../services/api/traceApi';
import { useUserStore } from '../../store/userStore';
import type {
  TraceTaskListParams,
  TraceMessageListParams,
  CreateTraceTaskPayload,
  TraceExportJob,
} from '../../types/trace';

const TASK_KEYS = ['trace', 'tasks'] as const;

// ----- Task queries -----

export function useTraceTasks(params: TraceTaskListParams) {
  return useQuery({
    queryKey: [...TASK_KEYS, 'list', params],
    queryFn: () => traceApi.listTasks(params),
    // L-2 修复：running 状态下 message_count 持续增长但不发 SSE 事件
    // （SSE 只发 task.{started,stopped,purged}）。给列表加 3s polling，仅当当前
    // 页存在 running 任务时启用，避免空转。stopped 后下次刷新自然停止 polling。
    refetchInterval: (query) => {
      const items = (query.state.data as { items?: { status?: string }[] } | undefined)?.items;
      const hasRunning = items?.some((t) => t.status === 'running') ?? false;
      return hasRunning ? 3000 : false;
    },
    refetchIntervalInBackground: false,
  });
}

export function useTraceTask(id: string | undefined) {
  return useQuery({
    queryKey: [...TASK_KEYS, 'detail', id],
    queryFn: () => traceApi.getTask(id!),
    enabled: Boolean(id),
  });
}

export function useActiveTraceTaskBySN(sn: string | undefined) {
  return useQuery({
    queryKey: [...TASK_KEYS, 'active', sn],
    queryFn: () => traceApi.getActiveTaskBySN(sn!),
    enabled: Boolean(sn),
  });
}

// ----- Task mutations -----

export function useCreateTraceTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateTraceTaskPayload) => traceApi.createTask(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: TASK_KEYS });
    },
  });
}

export function useStopTraceTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, purge = false }: { id: string; purge?: boolean }) =>
      traceApi.stopTask(id, purge),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: TASK_KEYS });
    },
  });
}

// ----- T-0161: 删除 -----

// useDeleteTraceTask 单条删除任务。任务必须先 stop（后端 service 层会拒绝 running）。
// 成功后失效任务列表与详情 query。
export function useDeleteTraceTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => traceApi.deleteTask(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: TASK_KEYS });
    },
  });
}

// useBatchDeleteTraceTasks 批量删除任务。返回 DeleteResult 汇总。
// 调用方负责弹 Popconfirm 二次确认 + 上限校验。
export function useBatchDeleteTraceTasks() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => traceApi.batchDeleteTasks(ids),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: TASK_KEYS });
    },
  });
}

// ----- Message query -----

export function useTraceMessages(taskId: string | undefined, params: TraceMessageListParams) {
  return useQuery({
    queryKey: ['trace', 'messages', taskId, params],
    queryFn: () => traceApi.listMessages(taskId!, params),
    enabled: Boolean(taskId),
    refetchInterval: 5000, // M1：轮询查看是否有新报文，5s 频率
  });
}

// ----- M2-08: 异步导出 -----

export function useRequestTraceExport() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: string) => traceApi.requestExport(taskId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: TASK_KEYS });
    },
  });
}

// useTraceExportJob 轮询单个导出任务直到 done/failed；done 时停止轮询
export function useTraceExportJob(jobId: string | undefined) {
  return useQuery<TraceExportJob>({
    queryKey: ['trace', 'exports', jobId],
    queryFn: () => traceApi.getExportJob(jobId!),
    enabled: Boolean(jobId),
    refetchInterval: (q) => {
      const data = q.state.data;
      if (!data) return 1500;
      if (data.status === 'done' || data.status === 'failed') return false;
      return 1500;
    },
  });
}

// ----- M2-07: SSE 实时订阅 trace.task.* -----

// useTraceSseRefresh 订阅 /api/v1/events/stream（已有 SSE Hub），
// 收到 trace.task.* 事件后 invalidate React Query 任务列表与详情。
// 单页挂载一次即可；外部不需要返回值。
//
// L-3 修复：EventSource 不支持自定义 header，但后端 SSE 端点支持 ?token= query
// 鉴权（events/handler.go），所以这里把 accessToken 拼到 URL。useEffect 依赖
// accessToken：登录/续 token 时自动重连。
export function useTraceSseRefresh() {
  const qc = useQueryClient();
  const accessToken = useUserStore((s) => s.accessToken);
  useEffect(() => {
    if (!accessToken) return; // 未登录不开 SSE，避免 401 噪音
    const baseURL = (
      (typeof import.meta !== 'undefined' && (import.meta as { env?: { VITE_API_BASE_URL?: string } }).env?.VITE_API_BASE_URL) ||
      '/api/v1'
    );
    const url = `${baseURL}/events/stream?token=${encodeURIComponent(accessToken)}`;
    let source: EventSource | null = null;
    try {
      source = new EventSource(url, { withCredentials: true });
    } catch {
      return;
    }
    const invalidate = () => void qc.invalidateQueries({ queryKey: ['trace'] });
    source.addEventListener('trace.task.started', invalidate);
    source.addEventListener('trace.task.stopped', invalidate);
    source.addEventListener('trace.task.purged', invalidate);
    return () => {
      source?.close();
    };
  }, [qc, accessToken]);
}
