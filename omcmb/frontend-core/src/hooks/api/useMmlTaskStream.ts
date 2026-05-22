/**
 * useMmlTaskStream — MML 任务执行结果实时订阅（SSE）。
 *
 * 历史：T-0123-P2-c 重构（commit da6c1c68，2026-05-14）下线旧 polling 路径
 * （useCommandExecution.ts 445 行），承诺 P4 接 SSE 但未落地，RightPanel 长期
 * 硬编码 <TerminalPanel lines={[]} />。本 hook 在 2026-05-22 补齐 P4 缺口。
 *
 * 后端事件：`omcgo/internal/mml/result_aggregator.go` 已经在跑——
 *   - mml_task_status   ← service.publishTaskStatus（pending/running 切换）
 *   - mml_device_frame  ← ResultAggregator.publishDeviceFrame（每 device_task 终态）
 *   - mml_task_completed ← ResultAggregator.finalizeIfComplete（整体收口）
 * 全部 publishSimple(executor, ...) 到 per-user channel，前端订阅
 * /api/v1/events/stream?token=<jwt> 即可。
 *
 * 行为：
 *   - mount 立刻建立 SSE（不等 execute），避免 task 创建瞬间事件丢失
 *   - taskId 切换时 setLines([]) 复位，旧任务历史不残留
 *   - taskId 为 null 时仍保持连接，但不写 lines（防御调用方早 mount）
 *   - SSE 全用户 channel 共享，事件按 frame.task_id === taskId 二次过滤
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useUserStore } from '@core/store/userStore';

export type MmlTerminalLineType = 'stdout' | 'stderr' | 'info' | 'success';

export interface MmlTerminalLine {
  text: string;
  type?: MmlTerminalLineType;
  timestamp?: string;
}

export type MmlTaskStreamStatus =
  | 'idle'
  | 'dispatched'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface UseMmlTaskStreamReturn {
  /** 累积的终端行；taskId 切换后会复位为空 */
  lines: MmlTerminalLine[];
  /** 任务整体状态推断 */
  status: MmlTaskStreamStatus;
  /** 主动追加一行（如调用方在 execute 成功后立刻种"已派发"行）*/
  appendLine: (line: MmlTerminalLine) => void;
  /** 清空 lines 并把 status 回到 idle */
  clear: () => void;
}

interface MmlDeviceFramePayload {
  task_id: string;
  device_task_id?: string;
  device_sn: string;
  method?: string;
  status: string;
  command_index?: number;
  device_index?: number;
  result?: unknown;
  error_message?: string;
  completed_at?: string;
}

interface MmlTaskStatusPayload {
  task_id: string;
  old_status: string;
  new_status: string;
  executor?: string;
}

interface MmlTaskCompletedPayload {
  task_id: string;
  status: string;
  result: string;
  success_count: number;
  failed_count: number;
}

function formatTimestamp(): string {
  return new Date().toLocaleTimeString();
}

function deviceFrameToLines(frame: MmlDeviceFramePayload): MmlTerminalLine[] {
  const ts = frame.completed_at
    ? new Date(frame.completed_at).toLocaleTimeString()
    : formatTimestamp();
  const out: MmlTerminalLine[] = [];
  const header = `[${frame.device_sn}] ${frame.method ?? ''} → ${frame.status}`.trimEnd();
  out.push({ type: 'info', text: header, timestamp: ts });
  if (frame.error_message) {
    out.push({ type: 'stderr', text: frame.error_message, timestamp: ts });
  }
  if (frame.result !== undefined && frame.result !== null) {
    const body =
      typeof frame.result === 'string'
        ? frame.result
        : JSON.stringify(frame.result, null, 2);
    if (body.length > 0) {
      out.push({
        type: frame.status === 'completed' ? 'stdout' : 'stderr',
        text: body,
        timestamp: ts,
      });
    }
  }
  return out;
}

function statusFromCompleted(p: MmlTaskCompletedPayload): MmlTaskStreamStatus {
  if (p.status === 'cancelled') return 'cancelled';
  if (p.failed_count === 0) return 'completed';
  if (p.success_count === 0) return 'failed';
  return 'completed';
}

/**
 * 安全解析 SSE event.data（JSON 字符串）为类型化 payload。
 * 失败返 null，调用方按需 ignore。
 */
function parseEventData<T>(raw: string): T | null {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

export function useMmlTaskStream(taskId: string | null): UseMmlTaskStreamReturn {
  const [lines, setLines] = useState<MmlTerminalLine[]>([]);
  const [status, setStatus] = useState<MmlTaskStreamStatus>('idle');
  const accessToken = useUserStore((s) => s.accessToken);
  // 用 ref 保留最新 taskId，让 listener 闭包始终读到当前值（避免每次 taskId
  // 变化重建 EventSource — 频繁重连会丢事件 + 增加后端连接开销）。
  const taskIdRef = useRef<string | null>(taskId);
  useEffect(() => {
    taskIdRef.current = taskId;
  }, [taskId]);

  // taskId 切换时复位 lines / status（每条 task 独立终端会话）
  useEffect(() => {
    setLines([]);
    setStatus(taskId ? 'dispatched' : 'idle');
  }, [taskId]);

  const appendLine = useCallback((line: MmlTerminalLine) => {
    setLines((prev) => [...prev, line]);
  }, []);
  const clear = useCallback(() => {
    setLines([]);
    setStatus('idle');
  }, []);

  useEffect(() => {
    if (!accessToken) return;
    const baseURL =
      (typeof import.meta !== 'undefined' &&
        (import.meta as { env?: { VITE_API_BASE_URL?: string } }).env?.VITE_API_BASE_URL) ||
      '/api/v1';
    const url = `${baseURL}/events/stream?token=${encodeURIComponent(accessToken)}`;
    let source: EventSource | null = null;
    try {
      source = new EventSource(url, { withCredentials: true });
    } catch {
      return;
    }

    const onDeviceFrame = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<MmlDeviceFramePayload>(ev.data);
      if (!frame || frame.task_id !== taskIdRef.current) return;
      setLines((prev) => [...prev, ...deviceFrameToLines(frame)]);
    };
    const onTaskStatus = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<MmlTaskStatusPayload>(ev.data);
      if (!frame || frame.task_id !== taskIdRef.current) return;
      // running 状态切换只更新 status 不写行，避免噪声；只在终态时写
      if (frame.new_status === 'running') {
        setStatus('running');
      } else if (frame.new_status === 'cancelled') {
        setStatus('cancelled');
        setLines((prev) => [
          ...prev,
          { type: 'info', text: `Task cancelled`, timestamp: formatTimestamp() },
        ]);
      }
    };
    const onTaskCompleted = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<MmlTaskCompletedPayload>(ev.data);
      if (!frame || frame.task_id !== taskIdRef.current) return;
      const newStatus = statusFromCompleted(frame);
      setStatus(newStatus);
      setLines((prev) => [
        ...prev,
        {
          type: newStatus === 'completed' ? 'success' : 'stderr',
          text: `Task ${frame.status} (${frame.result}): success=${frame.success_count}, failed=${frame.failed_count}`,
          timestamp: formatTimestamp(),
        },
      ]);
    };

    source.addEventListener('mml_device_frame', onDeviceFrame as EventListener);
    source.addEventListener('mml_task_status', onTaskStatus as EventListener);
    source.addEventListener('mml_task_completed', onTaskCompleted as EventListener);

    return () => {
      source?.removeEventListener('mml_device_frame', onDeviceFrame as EventListener);
      source?.removeEventListener('mml_task_status', onTaskStatus as EventListener);
      source?.removeEventListener('mml_task_completed', onTaskCompleted as EventListener);
      source?.close();
    };
  }, [accessToken]);

  return { lines, status, appendLine, clear };
}
