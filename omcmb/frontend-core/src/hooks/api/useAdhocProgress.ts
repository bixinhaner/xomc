/**
 * useAdhocProgressStream — PM 自定义聚合任务（adhoc）进度实时订阅（SSE）。
 *
 * issue #399：后端进度端点 `GET /pm/adhoc/tasks/:id/progress` 早已就绪，按 task_id
 * 推 `event: progress`（{task_id, progress, granularity, rows}）与 `event: completed`
 * （{task_id, status, rows_total, error}）两具名事件，30s 一个 keep-alive comment。
 * 前端此前只靠 5s/10s 轮询拿 task.progress，进度跳变粗糙、刷新滞后；本 hook 接通 SSE，
 * 让运行中任务的进度由事件实时驱动，轮询降为低频兜底（≥30s）。
 *
 * 与 MML/Trace 的共享 hub（一条 `/events/stream` 过滤多 task）不同，adhoc 端点是
 * **单任务流**，故为传入的每个运行中 task id 各建一条 EventSource。
 *
 * 行为：
 *   - 仅为调用方传入的「运行中」task id 建连（已终态/scheduled 不传 → 不建连）。
 *   - id 集合变化：为新增 id 建连、为移除 id 关连接（按稳定 key diff）。
 *   - progress 事件 → 更新该 id 的 {progress, rows}。
 *   - completed 事件 → 落终态 {status, rowsTotal, error, progress:100}，主动关该条连接，
 *     并 invalidate ['pm-adhoc-tasks'] 让列表/run 立即重取拿到终态行（不必等降频轮询）。
 *   - onerror → 关该条连接、不抛错（优雅降级，进度回退到轮询兜底，不让进度条永久卡住）。
 *   - useEffect cleanup → 关全部连接。
 *
 * 注意：EventSource 绕过 axios 的 camelCase 拦截器，原始 SSE payload 就是 snake_case，
 * 故直接消费孤儿类型 AdhocProgressEvent / AdhocCompletedEvent 解析原始 JSON。
 */

import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useUserStore } from '../../store/userStore';
import type {
  AdhocProgressEvent,
  AdhocCompletedEvent,
  AdhocStatus,
} from '../../types/pmAdhoc';

/** 单个运行中任务的实时进度快照（live 优先于轮询拿到的 task.progress）。 */
export interface AdhocLiveProgress {
  /** 0–100 当前进度（progress 事件刷新；completed 落 100）。 */
  progress: number;
  /** progress 事件携带的累计行数。 */
  rows?: number;
  /** completed 事件落的终态（succeeded/failed/canceled）。 */
  status?: AdhocStatus;
  /** completed 事件携带的总行数。 */
  rowsTotal?: number;
  /** completed 事件携带的错误（failed 时）。 */
  error?: string;
  /** 是否已收到 completed（连接已关、可作终态判定）。 */
  done?: boolean;
}

export const ADHOC_TASKS_QUERY_KEY = ['pm-adhoc-tasks'] as const;

/** 安全解析 SSE event.data（JSON 字符串）为类型化 payload；失败返 null。 */
function parseEventData<T>(raw: string): T | null {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

/**
 * 纯 reducer：把一条 progress 事件应用到当前 live map，返回新 map。
 * 抽成纯函数便于单测（不依赖 EventSource / DOM）。
 */
export function applyProgressEvent(
  prev: ReadonlyMap<string, AdhocLiveProgress>,
  ev: AdhocProgressEvent,
): Map<string, AdhocLiveProgress> {
  const next = new Map(prev);
  const cur = next.get(ev.task_id);
  // 已 done 的任务不再被 progress 事件回退（终态优先）。
  if (cur?.done) return next;
  next.set(ev.task_id, {
    ...cur,
    progress: ev.progress,
    rows: ev.rows,
  });
  return next;
}

/**
 * 纯 reducer：把一条 completed 事件应用到当前 live map，返回新 map。
 * 落终态 + progress=100 + done 标记。
 */
export function applyCompletedEvent(
  prev: ReadonlyMap<string, AdhocLiveProgress>,
  ev: AdhocCompletedEvent,
): Map<string, AdhocLiveProgress> {
  const next = new Map(prev);
  const cur = next.get(ev.task_id);
  next.set(ev.task_id, {
    ...cur,
    progress: 100,
    status: ev.status,
    rowsTotal: ev.rows_total,
    error: ev.error,
    done: true,
  });
  return next;
}

/** 解析 baseURL（与既有 SSE hook 同口径）。 */
function resolveBaseURL(): string {
  return (
    (typeof import.meta !== 'undefined' &&
      (import.meta as { env?: { VITE_API_BASE_URL?: string } }).env?.VITE_API_BASE_URL) ||
    '/api/v1'
  );
}

/**
 * useAdhocProgressStream —— 为每个运行中 task id 建一条 SSE 连接，返回实时进度 map。
 *
 * @param runningTaskIds 运行中（含 pending）任务 id 列表；调用方负责只传非终态/非 scheduled。
 * @returns Map<taskId, AdhocLiveProgress>；渲染时 `live.get(id)?.progress ?? task.progress`。
 */
export function useAdhocProgressStream(
  runningTaskIds: string[],
): ReadonlyMap<string, AdhocLiveProgress> {
  const accessToken = useUserStore((s) => s.accessToken);
  const qc = useQueryClient();
  const [live, setLive] = useState<ReadonlyMap<string, AdhocLiveProgress>>(
    () => new Map(),
  );

  // 活跃连接表：taskId → EventSource，跨 render 持有以便增量增删连接。
  const sourcesRef = useRef<Map<string, EventSource>>(new Map());

  // 用稳定 key（去重+排序后 join）跟踪 id 集合变化，避免数组引用变化触发不必要的 effect。
  const uniqueIds = Array.from(new Set(runningTaskIds.filter(Boolean)));
  const idsKey = [...uniqueIds].sort().join(',');

  useEffect(() => {
    if (!accessToken) {
      // 未登录：关掉所有现存连接，避免 401 噪音。
      sourcesRef.current.forEach((s) => s.close());
      sourcesRef.current.clear();
      return;
    }

    const baseURL = resolveBaseURL();
    const desired = new Set(idsKey ? idsKey.split(',') : []);
    const sources = sourcesRef.current;

    // 1) 关闭并移除不再 desired 的连接（任务已终态/被移出运行集合）。
    for (const [id, src] of sources) {
      if (!desired.has(id)) {
        src.close();
        sources.delete(id);
      }
    }

    // 2) 为新增 desired id 建连。
    for (const id of desired) {
      if (sources.has(id)) continue;
      const url = `${baseURL}/pm/adhoc/tasks/${encodeURIComponent(
        id,
      )}/progress?token=${encodeURIComponent(accessToken)}`;
      let source: EventSource;
      try {
        source = new EventSource(url, { withCredentials: true });
      } catch {
        // 建连失败：优雅降级，不抛错；该任务进度回退轮询兜底。
        continue;
      }

      const onProgress = (e: MessageEvent<string>): void => {
        const data = parseEventData<AdhocProgressEvent>(e.data);
        if (!data || data.task_id !== id) return;
        setLive((prev) => applyProgressEvent(prev, data));
      };
      const onCompleted = (e: MessageEvent<string>): void => {
        const data = parseEventData<AdhocCompletedEvent>(e.data);
        if (!data || data.task_id !== id) return;
        setLive((prev) => applyCompletedEvent(prev, data));
        // 终态到达：主动关连接（后端靠 client ctx Done 收尾）。
        const s = sources.get(id);
        if (s) {
          s.close();
          sources.delete(id);
        }
        // 让列表/run 立即重取拿到终态行，不必等降频轮询。
        void qc.invalidateQueries({ queryKey: ADHOC_TASKS_QUERY_KEY });
      };
      const onError = (): void => {
        // 优雅降级：关该条连接、不抛错。进度回退轮询兜底，不让进度条永久卡住。
        const s = sources.get(id);
        if (s) {
          s.close();
          sources.delete(id);
        }
      };

      source.addEventListener('progress', onProgress as EventListener);
      source.addEventListener('completed', onCompleted as EventListener);
      source.addEventListener('error', onError as EventListener);
      sources.set(id, source);
    }

    // 注意：本 effect 内 sources 的增删是增量的；cleanup 只在 unmount / token 变化时关全部。
  }, [accessToken, idsKey, qc]);

  // unmount 关全部连接（token 变化时上面分支已先清；此处兜底卸载场景）。
  useEffect(() => {
    const sources = sourcesRef.current;
    return () => {
      sources.forEach((s) => s.close());
      sources.clear();
    };
  }, []);

  return live;
}
