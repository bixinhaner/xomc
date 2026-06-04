import { useMemo, useState } from 'react';
import type { ExecRecord } from './types';
import { MOCK_HISTORY } from './mock';

/**
 * 命令记录数据层（设计 §3.10.4-5；2026-06-04 决策=方案 A：接 /mml/tasks 持久化）。
 *
 * 阶段策略（用户决策）：**先阶段先 mock 数据，对接后端 API 时换真实数据**。
 *
 * ── 当前（mock）：
 *   列表来源 = MOCK_HISTORY 种子 + 会话内 append 的执行记录；结果快照内嵌在 ExecRecord。
 *
 * ── 接后端 TODO（仅改本 hook 内部，组件层 records/activeId/select/append 契约不变，UI 零改动）：
 *   1. 列表：useMMLTasks({ source: 'console', pageSize })  → 映射为 ExecRecord 摘要
 *      （id=task.id, time=createdAt, commandName=commandsDetail[0], deviceCount=totalDevices）。
 *   2. 结果：select(id) 时惰性 useMMLTaskResults(id) + GET /mml/tasks/:id/results-schema
 *      → 派生 columns/rows（替换内嵌快照）。
 *   3. append：执行改走真实 execute-statements 后，靠 useMMLTasks 的 refetch/SSE 自动入列，
 *      不再前端手动 append（mock 阶段保留 append 以即时反馈）。
 */
export interface ConsoleHistory {
  records: ExecRecord[];
  activeId: string | null;
  activeRecord: ExecRecord | null;
  select: (id: string) => void;
  /** 追加一条新执行记录并置为当前（mock 阶段用；真实接入后由 refetch 取代）。 */
  append: (rec: ExecRecord) => void;
}

export function useConsoleHistory(): ConsoleHistory {
  const [records, setRecords] = useState<ExecRecord[]>(MOCK_HISTORY);
  const [activeId, setActiveId] = useState<string | null>(MOCK_HISTORY[0]?.id ?? null);

  const activeRecord = useMemo(
    () => records.find((r) => r.id === activeId) ?? null,
    [records, activeId],
  );

  const append = (rec: ExecRecord) => {
    setRecords((prev) => [rec, ...prev]);
    setActiveId(rec.id);
  };

  return { records, activeId, activeRecord, select: setActiveId, append };
}
