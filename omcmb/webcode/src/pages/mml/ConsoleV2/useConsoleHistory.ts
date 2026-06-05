import { useCallback, useMemo, useState } from 'react';
import type { ExecRecord } from './types';
import { MOCK_HISTORY } from './mock';

/**
 * 命令记录数据层（设计 §3.10.4-5 + §3.11.4；2026-06-05 决策）。
 *
 * ── 持久化策略（用户决策 2026-06-05）：**localStorage 只存「命令 ID 数组」，凭命令 ID 现拉数据**。
 *   不缓存命令名/设备数等摘要，更不缓存结果行快照（天然规避容量；任务被清理→现拉 404 自动剔除）。
 *
 * ── 当前（mock）：
 *   - `recordStore`（模块级 Map<commandId, ExecRecord>）= mock「后端」，初始化注入 MOCK_HISTORY 种子；
 *   - localStorage 仅存命令 ID 数组（有序，最近在前）；进入页面凭 ID 从 recordStore 取记录；
 *   - 会话内 append 的记录写入 recordStore + ID 数组；整页刷新后内存 Map 丢失会话记录属预期，
 *     仅种子可解析（模拟「凭 ID 向后端取数」链路）。
 *
 * ── 接后端 TODO（仅改本 hook 内部，组件层契约不变）：
 *   1. recordStore.get(id) → GET /mml/tasks/:id（摘要）+ 惰性 GET …/results-schema、…/results（结果）。
 *   2. append：执行改走真实 execute-statements 返回 mml_tasks.id 后，把该 ID push 进数组即可。
 */
export interface ConsoleHistory {
  records: ExecRecord[];
  activeId: string | null;
  activeRecord: ExecRecord | null;
  select: (id: string) => void;
  /** 追加一条新执行记录并置为当前（mock 阶段用；真实接入后由 refetch 取代）。 */
  append: (rec: ExecRecord) => void;
  /** 清空命令记录（内存列表 + localStorage 的命令 ID 数组）。 */
  clear: () => void;
}

const STORAGE_KEY = 'mml-console-v2-history';
const MAX_IDS = 50;

// mock「后端」：commandId → ExecRecord。模块初始化注入种子，模拟已持久化的历史任务。
const recordStore = new Map<string, ExecRecord>(MOCK_HISTORY.map((r) => [r.commandId, r]));

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

/** 首次进入：无持久化则以种子命令 ID 初始化并落盘（模拟「上次执行记录」）。 */
function initialIds(): string[] {
  const persisted = readIds();
  if (persisted.length > 0) return persisted;
  const seedIds = MOCK_HISTORY.map((r) => r.commandId);
  writeIds(seedIds);
  return seedIds;
}

export function useConsoleHistory(): ConsoleHistory {
  const [ids, setIds] = useState<string[]>(initialIds);
  const [activeId, setActiveId] = useState<string | null>(null);

  // 凭命令 ID 现拉记录（mock：recordStore；真实：GET /mml/tasks/:id）。无法解析的 ID 剔除。
  const records = useMemo(
    () => ids.map((id) => recordStore.get(id)).filter((r): r is ExecRecord => !!r),
    [ids],
  );

  // 默认选中最新一条（render 阶段派生，避免 effect 改 state）。
  const resolvedActiveId =
    activeId && records.some((r) => r.id === activeId) ? activeId : (records[0]?.id ?? null);

  const activeRecord = useMemo(
    () => records.find((r) => r.id === resolvedActiveId) ?? null,
    [records, resolvedActiveId],
  );

  const append = useCallback((rec: ExecRecord) => {
    recordStore.set(rec.commandId, rec);
    setIds((prev) => {
      const next = [rec.commandId, ...prev.filter((x) => x !== rec.commandId)].slice(0, MAX_IDS);
      writeIds(next);
      return next;
    });
    setActiveId(rec.id);
  }, []);

  const select = useCallback((id: string) => setActiveId(id), []);

  const clear = useCallback(() => {
    setIds([]);
    writeIds([]);
    setActiveId(null);
  }, []);

  return { records, activeId: resolvedActiveId, activeRecord, select, append, clear };
}
