/**
 * T-0130 GPV instance probe hook.
 *
 * 给定单设备 SN + 目标对象（如 "Device.IP.Interface."），触发 GPV RPC + 订阅
 * mml_device_frame SSE → 提取实例索引集合返给调用方。
 *
 * 协议链路（与后端 commit 3f802ef7 对齐）：
 *   1. POST /api/v1/ops/commands/rpc {device_sn, action="get_param", params={path: target}}
 *      → 返 task_id（per-device fan-out 单实例）
 *   2. SSE 订阅 GET /api/v1/events/stream（per-user channel）
 *   3. 监听 event="mml_device_frame"，过滤 payload.task_id === 本次 task_id
 *   4. payload.status="completed" 时解析 payload.result.ParameterList → parseGpvInstances → 实例 array
 *   5. payload.status="failed" → setError(error_message)
 *
 * 超时：60s（与 PRD §4.2.3 单命令默认对齐）；超时后 close SSE + setError
 *
 * 状态机：
 *   idle → probing (POST /rpc 触发后) → (success | failed | timeout) → idle (reset 后)
 *
 * 防泄漏：组件 unmount 时 cleanup 立即 close SSE + clearTimeout
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { opsExtApi, subscribeSSE } from '../../services/api/opsExtApi';
import {
  parseGpvInstances,
  type GpvParameter,
} from '../../utils/parseGpvInstances';

const PROBE_TIMEOUT_MS = 60_000;

export type ProbeStatus = 'idle' | 'probing' | 'success' | 'failed' | 'timeout';

export interface UseGPVProbeResult {
  /** 触发探测；deviceSn / targetObject 为空则 no-op。同时多次调用会先 reset。 */
  probe: () => void;
  /** 重置内部状态到 idle（也会断开 SSE 订阅）。 */
  reset: () => void;
  /** 当前状态。 */
  status: ProbeStatus;
  /** 探测出的实例索引集合（success 状态后填充；其他状态保持上一次结果或 []）。 */
  instances: number[];
  /** failed / timeout 时的错误信息。 */
  error?: string;
}

interface MmlDeviceFramePayload {
  task_id?: string;
  device_task_id?: string;
  device_sn?: string;
  method?: string;
  status?: string;
  result?: unknown;
  error_message?: string;
}

interface GpvResultEnvelope {
  ParameterList?: GpvParameter[];
}

/**
 * Pure helper — exported for unit-test friendliness.
 *
 * GPV result 在 backend 里被 `json.RawMessage` 透传，可能是 object 或 already-parsed
 * 嵌套 — defensively narrow before extracting ParameterList.
 */
export function extractParameterList(raw: unknown): GpvParameter[] {
  if (!raw || typeof raw !== 'object') return [];
  const env = raw as GpvResultEnvelope;
  if (!Array.isArray(env.ParameterList)) return [];
  return env.ParameterList as GpvParameter[];
}

/**
 * Probe instance indices for a single device + target object.
 *
 * Multi-device fan-out 不在本期范围 — 上层 UI 应在多设备时禁用 probe 按钮。
 */
export function useGPVProbe(
  deviceSn: string | undefined,
  targetObject: string | undefined,
): UseGPVProbeResult {
  const [status, setStatus] = useState<ProbeStatus>('idle');
  const [instances, setInstances] = useState<number[]>([]);
  const [error, setError] = useState<string | undefined>(undefined);

  // refs for cleanup: SSE source + timeout timer
  const sseCloseRef = useRef<(() => void) | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // 在 SSE 回调里需要稳定的 taskId 与 targetObject 引用（state 异步）
  const taskIdRef = useRef<string | null>(null);
  const targetRef = useRef<string | undefined>(targetObject);
  // mount guard 防止 unmount 后 setState
  const mountedRef = useRef(true);

  // keep targetRef synced with prop without re-creating probe()
  useEffect(() => {
    targetRef.current = targetObject;
  }, [targetObject]);

  const cleanup = useCallback(() => {
    if (sseCloseRef.current) {
      sseCloseRef.current();
      sseCloseRef.current = null;
    }
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    taskIdRef.current = null;
  }, []);

  const reset = useCallback(() => {
    cleanup();
    if (!mountedRef.current) return;
    setStatus('idle');
    setError(undefined);
    setInstances([]);
  }, [cleanup]);

  const probe = useCallback(() => {
    cleanup();
    if (!mountedRef.current) return;

    if (!deviceSn || !targetObject) {
      setStatus('failed');
      setError('missing device or target');
      return;
    }

    setStatus('probing');
    setError(undefined);

    // 先建 SSE 订阅（防止 RPC 完成时 frame 已发出但 listener 未挂上）
    const { close } = subscribeSSE('/events/stream', {
      onEvent: (event, data) => {
        if (event !== 'mml_device_frame') return;
        const payload = data as MmlDeviceFramePayload;
        if (!payload || typeof payload !== 'object') return;
        // 单一 task 过滤（per-call frame may arrive after our taskId assigned）
        if (taskIdRef.current && payload.task_id !== taskIdRef.current) return;
        if (payload.method && payload.method !== 'GetParameterValues') return;

        if (payload.status === 'completed') {
          const params = extractParameterList(payload.result);
          const idxs = parseGpvInstances(params, targetRef.current ?? '');
          if (!mountedRef.current) return;
          setInstances(idxs);
          setStatus('success');
          cleanup();
          return;
        }
        if (payload.status === 'failed' || payload.status === 'expired') {
          if (!mountedRef.current) return;
          setStatus('failed');
          setError(payload.error_message || `GPV ${payload.status}`);
          cleanup();
        }
      },
      onError: () => {
        if (!mountedRef.current) return;
        setStatus('failed');
        setError('SSE connection error');
        cleanup();
      },
    });
    sseCloseRef.current = close;

    // 触发 GPV — get_param action 走 backend rpc_dispatcher → ACS GetParameterValues
    opsExtApi
      .executeRPC({
        device_sn: deviceSn,
        action: 'get_param',
        params: { path: targetObject },
      })
      .then((resp) => {
        if (!mountedRef.current) {
          cleanup();
          return;
        }
        taskIdRef.current = resp.task_id;
      })
      .catch((e: unknown) => {
        if (!mountedRef.current) return;
        const msg = e instanceof Error ? e.message : 'RPC dispatch failed';
        setStatus('failed');
        setError(msg);
        cleanup();
      });

    // 60s 超时
    timerRef.current = setTimeout(() => {
      if (!mountedRef.current) return;
      // 仅在仍在探测时升 timeout（避免覆盖已成功/失败状态）
      setStatus((cur) => (cur === 'probing' ? 'timeout' : cur));
      setError((cur) => cur ?? 'probe timeout (60s)');
      cleanup();
    }, PROBE_TIMEOUT_MS);
  }, [cleanup, deviceSn, targetObject]);

  // unmount cleanup
  useEffect(() => {
    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, [cleanup]);

  return { probe, reset, status, instances, error };
}

/** 暴露给测试 / 调试。 */
export const GPV_PROBE_TIMEOUT_MS = PROBE_TIMEOUT_MS;
