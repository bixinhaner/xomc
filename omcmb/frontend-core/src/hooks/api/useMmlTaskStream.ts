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
  /**
   * mml_device_frame header 行的可注入模板：
   * 默认 `[{deviceSn}] {method} → {statusLabel}`；
   * 调用方传入后可附带任务 ID 等信息（例如 `[SN] GPV → 任务完成  (任务ID: xxx)`）。
   */
  deviceHeader?: (args: {
    deviceSn: string;
    method: string;
    statusLabel: string;
    taskId: string;
  }) => string;
  /**
   * 多 task 全部完成后的聚合摘要文案（仅在 taskIds 全部 mml_task_completed 后触发一次）。
   * 返回空串则不写行（让调用方在单任务场景下抑制聚合输出）。
   * results 顺序与 taskIds 一致；调用方可结合自身 pathByTaskId 映射打印"哪些 path 成功 / 失败"。
   */
  aggregateSummary?: (args: {
    results: ReadonlyArray<{
      taskId: string;
      status: string;
      successCount: number;
      failedCount: number;
    }>;
    successCount: number;
    failedCount: number;
    total: number;
  }) => string;
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
  deviceHeader?: MmlTaskStreamMessages['deviceHeader'],
): MmlTerminalLine[] {
  const ts = frame.completed_at
    ? new Date(frame.completed_at).toLocaleTimeString()
    : formatTimestamp();
  const out: MmlTerminalLine[] = [];
  // status 翻译：completed → 任务完成，failed → 任务失败 等；未提供 / 未命中
  // 时落回 raw 字串（兼容老调用方 / 后端新增状态码）。
  const statusLabel = statusLabels?.[frame.status] ?? frame.status;
  const header = deviceHeader
    ? deviceHeader({
        deviceSn: frame.device_sn,
        method: frame.method ?? '',
        statusLabel,
        taskId: frame.task_id,
      })
    : `[${frame.device_sn}] ${frame.method ?? ''} → ${statusLabel}`.trimEnd();
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

/**
 * 2026-05-28 用户决策"单PATH执行":hook 同时监听多个 task_id。
 *
 * 兼容性:
 *   - 传 string | null → 单 task,完全等价于老行为
 *   - 传 string[] → 多 task,所有事件按 frame.task_id ∈ Set 过滤
 *   - taskIds 引用变化时,Set 重建;EventSource 连接不变(避免丢事件)
 *
 * status 单值的多 task 推断:
 *   - 任一 task running → status = 'running'
 *   - 全部 task completed → status = 'completed'
 *   - 已 completed 数 < total 时保持 running
 *   - 任一 task failed 且未有成功 → 'failed';mixed 维持 'running'/'completed'
 */
export function useMmlTaskStream(
  taskIds: string | string[] | null,
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
  // 用 ref 保留最新 taskId 集合,让 listener 闭包始终读到当前值(避免每次 taskIds
  // 变化重建 EventSource — 频繁重连会丢事件 + 增加后端连接开销)。
  //
  // 关键:ref 写在 render body 而非 useEffect。EventSource 是浏览器持久连接,
  // 事件可能在 setTaskIds 之后、useEffect 触发之前就到达(后端 publishTaskStatus
  // 在 HTTP 响应之前已 publish 到 hub);放 effect 里会让首条 mml_task_status
  // 被 set 过滤丢弃。ref 在 render 期赋值合规:不触发 re-render,对其它 hook 无副作用。
  const normalizedIds = Array.isArray(taskIds)
    ? taskIds
    : taskIds
      ? [taskIds]
      : [];
  const taskIdsRef = useRef<Set<string>>(new Set(normalizedIds));
  taskIdsRef.current = new Set(normalizedIds);
  const hasAnyTask = normalizedIds.length > 0;
  // 用稳定 key (排序后 join) 跟踪 taskIds 变化,避免数组引用变化触发不必要的 effect
  const idsKey = [...normalizedIds].sort().join(',');

  // mml_task_status running 事件按 task_id 去重(每个 task 第一次 running 才打印)。
  // 改用 ref Set 而非单 bool,支持多 task 各自一次打印。
  const runningEmittedSetRef = useRef<Set<string>>(new Set());

  // 单PATH执行模式:聚合摘要需要等所有 task 都 completed 后再写一行。
  // Map<task_id, {status, successCount, failedCount}>;size 等于 taskIdsRef.current.size 时触发。
  type CompletedRecord = {
    status: string;
    successCount: number;
    failedCount: number;
  };
  const completedResultsRef = useRef<Map<string, CompletedRecord>>(new Map());
  const aggregateEmittedRef = useRef<boolean>(false);

  // taskIds 变化只重置会话级 status + dedup 标记;lines 由持久化 store 托管,跨 task 累积。
  useEffect(() => {
    runningEmittedSetRef.current = new Set();
    completedResultsRef.current = new Map();
    aggregateEmittedRef.current = false;
    setStatus(hasAnyTask ? 'dispatched' : 'idle');
  }, [idsKey, hasAnyTask]);

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
      if (!frame || !taskIdsRef.current.has(frame.task_id)) return;
      const msgs = messagesRef.current;
      appendLinesStore(
        deviceFrameToLines(frame, msgs?.deviceStatusLabels, msgs?.deviceHeader),
      );
    };
    const onTaskStatus = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<MmlTaskStatusPayload>(ev.data);
      if (!frame || !taskIdsRef.current.has(frame.task_id)) return;
      const msgs = messagesRef.current;
      if (frame.new_status === 'running') {
        // running 状态首次到达 → 写一条"开始执行"提示行(每个 task 各自打印 1 次)。
        // 多 task 场景下用 Set<task_id> 去重,避免重复行。
        if (!runningEmittedSetRef.current.has(frame.task_id)) {
          runningEmittedSetRef.current.add(frame.task_id);
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
      if (!frame || !taskIdsRef.current.has(frame.task_id)) return;
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

      // 聚合摘要:登记本次结果,所有 task 都到齐后写一行总览(只触发一次)。
      completedResultsRef.current.set(frame.task_id, {
        status: frame.status,
        successCount: frame.success_count,
        failedCount: frame.failed_count,
      });
      const subscribed = taskIdsRef.current;
      const completedMap = completedResultsRef.current;
      const allDone =
        !aggregateEmittedRef.current &&
        subscribed.size > 0 &&
        completedMap.size >= subscribed.size &&
        [...subscribed].every((id) => completedMap.has(id));
      if (allDone) {
        aggregateEmittedRef.current = true;
        // 保持订阅顺序输出 results,便于调用方按 dispatch 顺序映射 path
        const orderedIds = [...subscribed];
        const results = orderedIds.map((id) => {
          const r = completedMap.get(id)!;
          return {
            taskId: id,
            status: r.status,
            successCount: r.successCount,
            failedCount: r.failedCount,
          };
        });
        // task 级 "成功" 判定:status='completed' 且 failed_count==0
        const failedTasks = results.filter(
          (r) => !(r.status === 'completed' && r.failedCount === 0),
        );
        const successCount = results.length - failedTasks.length;
        const failedCount = failedTasks.length;
        const aggText = msgs?.aggregateSummary?.({
          results,
          successCount,
          failedCount,
          total: results.length,
        });
        // formatter 返回空串 → 抑制聚合行(单任务场景不需要)
        if (aggText) {
          appendLineStore({
            type: failedCount === 0 ? 'success' : 'stderr',
            text: aggText,
            timestamp: formatTimestamp(),
          });
        }
      }
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
