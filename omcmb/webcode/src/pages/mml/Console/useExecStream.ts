import { useEffect, useRef } from 'react';
import { useUserStore } from '@core/store/userStore';
import type { DeviceFramePayload, TaskCompletedPayload } from './adapters';

// MML 控制台 V2 —— 执行结果实时订阅（SSE，设计 §3.12.2）。
//
// 复用老 console 的 SSE 机制（同一 /events/stream 全用户 channel + 同一帧契约），
// 但**面向结果表格**而非终端文本：只把 mml_device_frame / mml_task_completed 帧透传给
// 调用方回调，由调用方就地更新 ResultRow。与 useMmlTaskStream（写终端 store）解耦。

interface ExecStreamHandlers {
  /** 每个设备 device_task 终态一帧（status + result/error）。 */
  onFrame: (frame: DeviceFramePayload) => void;
  /** 整体收口（全部设备完成）。 */
  onCompleted: (frame: TaskCompletedPayload) => void;
}

function parseEventData<T>(raw: string): T | null {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

/**
 * 订阅一组在途 taskId 的执行结果帧（#217 多条命令并发在途）。空集合时保持连接但不分发。
 * 用 ref 保存最新 taskId 集合 / handlers，避免它们变化时重建 EventSource（频繁重连会丢事件）。
 * 单一全用户 channel 透传所有任务的帧，listener 内按 frame.task_id 是否在订阅集合内决定分发。
 */
export function useExecStream(taskIds: ReadonlySet<string>, handlers: ExecStreamHandlers): void {
  const accessToken = useUserStore((s) => s.accessToken);
  // ref 让长连接 listener 读到最新 taskId 集合 / handlers，而不重建 EventSource。
  const taskIdsRef = useRef(taskIds);
  const handlersRef = useRef(handlers);
  useEffect(() => {
    taskIdsRef.current = taskIds;
  }, [taskIds]);
  useEffect(() => {
    handlersRef.current = handlers;
  }, [handlers]);

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
      const frame = parseEventData<DeviceFramePayload>(ev.data);
      if (!frame || !taskIdsRef.current.has(frame.task_id)) return;
      handlersRef.current.onFrame(frame);
    };
    const onTaskCompleted = (ev: MessageEvent<string>): void => {
      const frame = parseEventData<TaskCompletedPayload>(ev.data);
      if (!frame || !taskIdsRef.current.has(frame.task_id)) return;
      handlersRef.current.onCompleted(frame);
    };

    source.addEventListener('mml_device_frame', onDeviceFrame as EventListener);
    source.addEventListener('mml_task_completed', onTaskCompleted as EventListener);

    return () => {
      source?.removeEventListener('mml_device_frame', onDeviceFrame as EventListener);
      source?.removeEventListener('mml_task_completed', onTaskCompleted as EventListener);
      source?.close();
    };
  }, [accessToken]);
}
