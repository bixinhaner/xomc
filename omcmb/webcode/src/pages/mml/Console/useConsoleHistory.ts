import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQueries, useQuery } from '@tanstack/react-query';
import { mmlApi } from '@core/services/api/mmlApi';
import { useAppStore } from '@core/store/appStore';
import type { ExecRecord } from './types';
import { buildDeviceRows, buildMODReadbackRows, mapTaskToRecord } from './adapters';

/**
 * 命令记录数据层（设计 §3.10.4-5 + §3.11.4 + §3.12）。
 *
 * ── 持久化策略：**localStorage 只存「命令 ID 数组」，凭命令 ID 现拉数据**。
 *   不缓存命令名/设备数等摘要，更不缓存结果行快照（天然规避容量；任务被清理→现拉 404 自动剔除）。
 *
 * ── 当前（P2 已接真实执行）：
 *   - `recordStore`（模块级 Map<commandId, ExecRecord>）承载**本会话**已执行命令的完整记录（含结果行）；
 *     append 时 commandId = 真实 `mml_tasks.id`，同步写入 localStorage 的 ID 数组（最近在前）。
 *   - 本会话内点击记录可回看完整结果；整页刷新后内存 Map 丢失，记录从列表消失（ID 仍在 localStorage）。
 *
 * ── P3 待办（仅改本 hook 内部，组件层契约不变）：
 *   挂载时对 localStorage 中、不在 recordStore 的命令 ID 调 `GET /mml/tasks/:id` 重建列表摘要，
 *   `select` 时惰性 `GET /mml/tasks/:id/results` 重建结果行——实现跨刷新历史恢复。
 *   （需对照运行时任务/结果数据形态校验 taskName 解析与 parsedData 键，故拆为后置任务。）
 */
export interface ConsoleHistory {
  records: ExecRecord[];
  activeId: string | null;
  activeRecord: ExecRecord | null;
  select: (id: string) => void;
  /** 追加一条新执行记录并置为当前（mock 阶段用；真实接入后由 refetch 取代）。 */
  append: (rec: ExecRecord) => void;
  /**
   * #217：原地更新已存在记录的内容（不改 activeId、不重排列表）。
   * SSE 帧实时回填各在途任务记录行时用——无论该记录是否当前选中，都不抢占用户的选中焦点。
   * 记录不存在（已被清空/未 append）时静默忽略。
   */
  update: (rec: ExecRecord) => void;
  /** 清空命令记录（内存列表 + localStorage 的命令 ID 数组）。 */
  clear: () => void;
}

const STORAGE_KEY = 'mml-console-history';
const MAX_IDS = 50;

// 本会话已执行命令：commandId(= mml_tasks.id) → ExecRecord（含结果行）。跨刷新丢失（P3 重建）。
const recordStore = new Map<string, ExecRecord>();

function readIds(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const ids: unknown = JSON.parse(raw);
    return Array.isArray(ids) ? ids.filter((x): x is string => typeof x === 'string') : [];
  } catch {
    return [];
  }
}

function writeIds(ids: string[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
  } catch {
    /* localStorage 不可用时静默降级为会话内 */
  }
}

export function isMissingTaskError(error: unknown): boolean {
  const status = (error as { response?: { status?: unknown } } | null)?.response?.status;
  return status === 404 || status === 410;
}

export function useConsoleHistory(): ConsoleHistory {
  const [ids, setIds] = useState<string[]>(readIds);
  const [activeId, setActiveId] = useState<string | null>(null);
  const locale = useAppStore((s) => s.locale);

  // 跨刷新恢复（§3.12.4）：不在本会话 recordStore 的命令 ID 凭 GET /mml/tasks/:id 重建。
  // 解析不到（404/已清理）的 ID 在 records 合并阶段自动剔除。retry:false 避免删除任务反复重试。
  const missingIds = useMemo(() => ids.filter((id) => !recordStore.has(id)), [ids]);
  const queries = useQueries({
    queries: missingIds.map((id) => ({
      queryKey: ['mml', 'console', 'task', id, locale],
      queryFn: () => mmlApi.getTaskById(id),
      staleTime: 60 * 1000,
      retry: false,
    })),
  });
  const fetchedById = new Map<string, ExecRecord>();
  queries.forEach((q, i) => {
    if (q.data) fetchedById.set(missingIds[i], { ...mapTaskToRecord(q.data), locale });
  });
  const missingErrorKey = queries
    .map((q, i) => (q.isError && isMissingTaskError(q.error) ? missingIds[i] : ''))
    .filter(Boolean)
    .join('|');

  useEffect(() => {
    if (!missingErrorKey) return;
    const deadIds = new Set(missingErrorKey.split('|'));
    setIds((prev) => {
      const next = prev.filter((id) => !deadIds.has(id));
      if (next.length === prev.length) return prev;
      writeIds(next);
      return next;
    });
  }, [missingErrorKey]);

  // 合并：本会话完整记录优先，其次拉取重建的记录；解析不到的 ID 剔除。保持 localStorage 顺序。
  const records = ids
    .map((id) => recordStore.get(id) ?? fetchedById.get(id) ?? null)
    .filter((r): r is ExecRecord => !!r);

  // 默认选中最新一条（render 阶段派生，避免 effect 改 state）。
  const resolvedActiveId =
    activeId && records.some((r) => r.id === activeId) ? activeId : (records[0]?.id ?? null);

  const baseActiveRecord = useMemo(
    () => records.find((r) => r.id === resolvedActiveId) ?? null,
    [records, resolvedActiveId],
  );

  // 惰性补结果行（§3.12.4 P3）：跨刷新重建的记录只有列、无结果行（getTaskById 不含 per-device
  // 结果）。当前选中记录若无行，按 commandId 拉 /results 并 buildDeviceRows 合并，使"进入页面默认
  // 展示最后一条命令的执行结果"成立；本会话已执行的记录（recordStore 有行）不触发。
  const needResults = !!baseActiveRecord && baseActiveRecord.rows.length === 0;
  const activeCommandId = baseActiveRecord?.commandId ?? null;
  const resultsQuery = useQuery({
    queryKey: ['mml', 'console', 'results', activeCommandId, locale],
    queryFn: () => mmlApi.getTaskResults(activeCommandId as string, 1, 200),
    enabled: (needResults || (!!baseActiveRecord && baseActiveRecord.locale !== locale)) && !!activeCommandId,
    staleTime: 60 * 1000,
    retry: false,
  });

  const activeRecord = useMemo<ExecRecord | null>(() => {
    if (!baseActiveRecord) return null;
    if (baseActiveRecord.rows.length > 0 && baseActiveRecord.locale === locale) return baseActiveRecord;
    const items = resultsQuery.data?.items;
    if (!items || items.length === 0) return { ...baseActiveRecord, locale };
    // #196：MOD 自动回读复合 → 走「下发 vs 回读」关联视图（操作类型 / 前后对比 / 双报文），
    // 与实时收口、历史摘要同一构建函数；其余命令按逐 PATH 合并。
    const rows =
      baseActiveRecord.execMeta.operationType === 'MOD' && baseActiveRecord.setValues
        ? buildMODReadbackRows(items, baseActiveRecord.setValues)
        : buildDeviceRows(items, baseActiveRecord.columns, baseActiveRecord.execMeta.read);
    const localizedName = items.find((item) => item.commandName?.trim())?.commandName?.trim();
    const commandName = localizedName ?? baseActiveRecord.commandName;
    return {
      ...baseActiveRecord,
      commandName,
      execMeta: { ...baseActiveRecord.execMeta, commandName },
      rows,
      locale,
    };
  }, [baseActiveRecord, locale, resultsQuery.data]);

  const append = useCallback((rec: ExecRecord) => {
    const localizedRecord = { ...rec, locale: rec.locale ?? locale };
    recordStore.set(localizedRecord.commandId, localizedRecord);
    setIds((prev) => {
      const next = [localizedRecord.commandId, ...prev.filter((x) => x !== localizedRecord.commandId)].slice(0, MAX_IDS);
      writeIds(next);
      return next;
    });
    setActiveId(localizedRecord.id);
  }, [locale]);

  // #217：原地更新已 append 记录的内容（不改 activeId / 不重排）。SSE 帧回填各在途任务行时用，
  // 避免非选中任务的帧 setActiveId 抢占用户当前查看的记录。需触发渲染让选中记录的结果区刷新。
  const [, forceRerender] = useState(0);
  const update = useCallback((rec: ExecRecord) => {
    if (!recordStore.has(rec.commandId)) return;
    recordStore.set(rec.commandId, { ...rec, locale: rec.locale ?? locale });
    forceRerender((n) => n + 1);
  }, [locale]);

  const select = useCallback((id: string) => setActiveId(id), []);

  const clear = useCallback(() => {
    setIds([]);
    writeIds([]);
    setActiveId(null);
  }, []);

  return { records, activeId: resolvedActiveId, activeRecord, select, append, update, clear };
}
