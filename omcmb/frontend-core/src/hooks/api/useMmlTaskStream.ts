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
 *   - lines 由 useMmlConsoleTerminalStore（localStorage 持久化）托管：刷新 / 切路由 /
 *     执行新命令都不会清空历史；用户主动点"清空"按钮才 reset（用户决策 2026-05-24）
 *   - taskId 为 null 时仍保持连接，但不写 lines（防御调用方早 mount）
 *   - SSE 全用户 channel 共享，事件按 frame.task_id === taskId 二次过滤
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useUserStore } from '@core/store/userStore';
import { useMmlConsoleTerminalStore } from '@core/store/mmlConsoleTerminalStore';
import { parseMmlDeviceTaskResult } from '@core/utils/mmlResultParser';
import type { ParsedMmlResult, ParsedParamValue } from '@core/utils/mmlResultParser';

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

/**
 * 可注入的本地化文案，供 hook 在状态切换时写入对应 line 文案。
 * 调用方（RightPanel）传 t() 渲染后的中文/英文字串，hook 内部不依赖 i18n。
 */
export interface MmlTaskStreamMessages {
  /** mml_task_status running → 写一行（"任务开始执行..."）*/
  running?: string;
  /** mml_task_status cancelled → 写一行（"任务已取消"）*/
  cancelled?: string;
  /** mml_task_completed → 写一行最终统计，调用方按需用模板拼成最终字串 */
  completedSummary?: (args: {
    status: string;
    result: string;
    successCount: number;
    failedCount: number;
  }) => string;
  /**
   * 设备 device_task 终态 status 标签翻译：
   * { completed: '任务完成', failed: '任务失败', expired: '任务超时', cancelled: '任务已取消' }
   * 取代过去 header 里直显英文 raw status；未传 / 无对应 key 时回落 raw 值。
   */
  deviceStatusLabels?: Record<string, string>;
}

export interface UseMmlTaskStreamReturn {
  /** 累积的终端行；由持久化 store 托管，跨刷新 / 跨命令保留 */
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

/** 把 ParsedMmlResult 翻译为多行可读输出（每 path 独立一行 / 状态摘要一行）。
 *
 * 用户决策（2026-05-26）：
 *   1) 保留整段原始 response 输出（位置在解析行之后）
 *   2) 增加按 path 拆分的解析行，每条 path 占一行，方便对照 LST 结果
 */
function parsedResultToLines(
  parsed: ParsedMmlResult,
  ts: string,
  frameStatus: string,
): MmlTerminalLine[] {
  const out: MmlTerminalLine[] = [];
  const lineType: MmlTerminalLineType = frameStatus === 'completed' ? 'stdout' : 'stderr';

  if (parsed.kind === 'gpv' && parsed.params && parsed.params.length > 0) {
    // GPV: 每个 path 一行；值为空显示 (empty)；类型作为括号注解（如有）
    for (const p of parsed.params) {
      out.push({
        type: lineType,
        text: formatGPVParamLine(p),
        timestamp: ts,
      });
    }
    return out;
  }
  if (parsed.kind === 'spv') {
    // SPV 状态：0=立即生效；1=需重启生效；其它=原值
    const tag =
      parsed.status === 0 ? 'applied (immediate)'
      : parsed.status === 1 ? 'applied (requires reboot)'
      : parsed.status === undefined ? 'no status'
      : `status=${parsed.status}`;
    out.push({ type: lineType, text: `  → ${tag}`, timestamp: ts });
    return out;
  }
  if (parsed.kind === 'add') {
    const inst = parsed.instanceNumber != null ? `InstanceNumber=${parsed.instanceNumber}` : 'no instance';
    const st = parsed.status === 0 ? 'applied' : parsed.status === 1 ? 'requires reboot' : `status=${parsed.status ?? '?'}`;
    out.push({ type: lineType, text: `  → ${inst} (${st})`, timestamp: ts });
    return out;
  }
  if (parsed.kind === 'delete') {
    const st = parsed.status === 0 ? 'deleted (immediate)' : parsed.status === 1 ? 'deleted (requires reboot)' : `status=${parsed.status ?? '?'}`;
    out.push({ type: lineType, text: `  → ${st}`, timestamp: ts });
    return out;
  }
  if (parsed.kind === 'reboot') {
    out.push({ type: lineType, text: `  → accepted`, timestamp: ts });
    return out;
  }
  return out;
}

/** 单条 GPV path/值的可读化：`  Device.X = "foo" (xsd:string)` */
function formatGPVParamLine(p: ParsedParamValue): string {
  const displayValue = p.value === '' ? '(empty)' : JSON.stringify(p.value);
  const suffix = p.type ? `  (${p.type})` : '';
  return `  ${p.name} = ${displayValue}${suffix}`;
}

function deviceFrameToLines(
  frame: MmlDeviceFramePayload,
  statusLabels?: Record<string, string>,
): MmlTerminalLine[] {
  const ts = frame.completed_at
    ? new Date(frame.completed_at).toLocaleTimeString()
    : formatTimestamp();
  const out: MmlTerminalLine[] = [];
  // status 翻译：completed → 任务完成，failed → 任务失败 等；未提供 / 未命中
  // 时落回 raw 字串（兼容老调用方 / 后端新增状态码）。
  const statusLabel = statusLabels?.[frame.status] ?? frame.status;
  const header = `[${frame.device_sn}] ${frame.method ?? ''} → ${statusLabel}`.trimEnd();
  out.push({ type: 'info', text: header, timestamp: ts });
  if (frame.error_message) {
    out.push({ type: 'stderr', text: frame.error_message, timestamp: ts });
  }
  if (frame.result !== undefined && frame.result !== null) {
    // 先尝试结构化解析（GPV 逐 path / SPV/ADD/DEL/Reboot 状态摘要）。
    // 解析成功 → 插入可读行；同时**保留**原始 response body 输出，便于排查协议层细节。
    const parsed = parseMmlDeviceTaskResult(frame.result);
    if (parsed) {
      out.push(...parsedResultToLines(parsed, ts, frame.status));
    }

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

export function useMmlTaskStream(
  taskId: string | null,
  messages?: MmlTaskStreamMessages,
): UseMmlTaskStreamReturn {
  // lines 走持久化 store —— 跨刷新 / 路由 / 新 execute 保留历史；只在用户点"清空"
  // 按钮时 reset。原 taskId-切换-自动-清空 行为已下线（用户决策 2026-05-24）。
  const lines = useMmlConsoleTerminalStore((s) => s.lines);
  const appendLineStore = useMmlConsoleTerminalStore((s) => s.appendLine);
  const appendLinesStore = useMmlConsoleTerminalStore((s) => s.appendLines);
  const clearStore = useMmlConsoleTerminalStore((s) => s.clear);
  const [status, setStatus] = useState<MmlTaskStreamStatus>('idle');
  const accessToken = useUserStore((s) => s.accessToken);
  // 用 ref 保存最新 messages，避免每次 messages 引用变化重建 EventSource
  const messagesRef = useRef<MmlTaskStreamMessages | undefined>(messages);
  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);
  // 用 ref 保留最新 taskId，让 listener 闭包始终读到当前值（避免每次 taskId
  // 变化重建 EventSource — 频繁重连会丢事件 + 增加后端连接开销）。
  //
  // 关键：ref 写在 render body，而不是 useEffect。因为 EventSource 是浏览器侧
  // 持久连接，事件可能在 setCurrentTaskId 之后、useEffect 触发之前就到达（后端
  // CreateAndFanoutTask publishTaskStatus 在 HTTP 响应之前 publish 到 hub，
  // 而 SSE 和 HTTP 走不同 TCP，事件抢跑常见）；放 effect 里会造成首条
  // mml_task_status 事件被 taskIdRef.current === null 过滤丢弃。
  // ref 在 render 期赋值合规：不触发 re-render，对其它 hook 无副作用。
  const taskIdRef = useRef<string | null>(taskId);
  taskIdRef.current = taskId;

  // mml_task_status running 事件去重 ref。
  // 之前在 setStatus((prev) => { ... appendLineStore(...) ... }) 里做 dedup，
  // 但 React 18+ 并发渲染下 setState updater 可能被调用多次（docs 明确要求
  // updater 必须纯函数），导致 appendLineStore 被调多次 → 终端出现两条
  // "任务开始执行" 重复行。改用 ref 在 effect 外部做 dedup，与 React 状态机解耦。
  const runningEmittedRef = useRef(false);

  // taskId 变化只重置会话级 status + dedup 标记；lines 由持久化 store 托管，跨 task 累积。
  useEffect(() => {
    runningEmittedRef.current = false;
    setStatus(taskId ? 'dispatched' : 'idle');
  }, [taskId]);

  const appendLine = useCallback(
    (line: MmlTerminalLine) => {
      appendLineStore(line);
    },
    [appendLineStore],
  );
  const clear = useCallback(() => {
    clearStore();
    setStatus('idle');
  }, [clearStore]);

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
      appendLinesStore(deviceFrameToLines(frame, messagesRef.current?.deviceStatusLabels));
    };
    const onTaskStatus = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<MmlTaskStatusPayload>(ev.data);
      if (!frame || frame.task_id !== taskIdRef.current) return;
      const msgs = messagesRef.current;
      if (frame.new_status === 'running') {
        // running 状态首次到达 → 写一条"开始执行"提示行，让用户感知任务真的进入下发阶段
        // （之前 silent → 用户体感是"派发后死寂"）。重复 running 事件不再写新行。
        // dedup 用 ref 而非 setState updater：state updater 必须纯函数，
        // appendLineStore 是副作用，并发渲染下可能被调多次（实测内网 HTTP 部署复现）。
        if (!runningEmittedRef.current) {
          runningEmittedRef.current = true;
          if (msgs?.running) {
            appendLineStore({
              type: 'info',
              text: msgs.running as string,
              timestamp: formatTimestamp(),
            });
          }
        }
        setStatus('running');
      } else if (frame.new_status === 'cancelled') {
        setStatus('cancelled');
        appendLineStore({
          type: 'info',
          text: msgs?.cancelled ?? 'Task cancelled',
          timestamp: formatTimestamp(),
        });
      }
    };
    const onTaskCompleted = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<MmlTaskCompletedPayload>(ev.data);
      if (!frame || frame.task_id !== taskIdRef.current) return;
      const newStatus = statusFromCompleted(frame);
      setStatus(newStatus);
      const msgs = messagesRef.current;
      const text =
        msgs?.completedSummary?.({
          status: frame.status,
          result: frame.result,
          successCount: frame.success_count,
          failedCount: frame.failed_count,
        }) ??
        `Task ${frame.status} (${frame.result}): success=${frame.success_count}, failed=${frame.failed_count}`;
      appendLineStore({
        type: newStatus === 'completed' ? 'success' : 'stderr',
        text,
        timestamp: formatTimestamp(),
      });
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
  }, [accessToken, appendLineStore, appendLinesStore]);

  return { lines, status, appendLine, clear };
}
