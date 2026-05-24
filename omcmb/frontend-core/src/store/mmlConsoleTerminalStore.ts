/**
 * mmlConsoleTerminalStore — MML 控制台"终端输出"持久化状态。
 *
 * 背景：原 useMmlTaskStream 把 lines 放在 hook 内部 useState，刷新页面 / 切换路由
 * 即丢失；执行新命令时也因 taskId 切换被 reset。用户希望终端像真终端一样跨刷新
 * 保留历史，每次执行追加而非覆盖。
 *
 * 设计：
 * - zustand + persist(localStorage) — 项目内已有同款模式（userStore 等）。
 * - 容量上限 MAX_LINES=2000：超出时丢弃最早的行，体感与终端 scrollback buffer 一致，
 *   避免 localStorage 无限膨胀（写入 quota 拒写会让整个 store 失败）。
 * - 只持久化 lines；status 是会话级语义（idle/running/completed 仅对"当前 task"有意义），
 *   留在 useMmlTaskStream 内部即可。
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { MmlTerminalLine } from '../hooks/api/useMmlTaskStream';

export const MAX_TERMINAL_LINES = 2000;

interface MmlConsoleTerminalState {
  lines: MmlTerminalLine[];
  appendLine: (line: MmlTerminalLine) => void;
  appendLines: (newLines: MmlTerminalLine[]) => void;
  clear: () => void;
}

function capLines(lines: MmlTerminalLine[]): MmlTerminalLine[] {
  if (lines.length <= MAX_TERMINAL_LINES) return lines;
  return lines.slice(lines.length - MAX_TERMINAL_LINES);
}

export const useMmlConsoleTerminalStore = create<MmlConsoleTerminalState>()(
  persist(
    (set) => ({
      lines: [],
      appendLine: (line) =>
        set((s) => ({ lines: capLines([...s.lines, line]) })),
      appendLines: (newLines) =>
        set((s) =>
          newLines.length === 0
            ? s
            : { lines: capLines([...s.lines, ...newLines]) },
        ),
      clear: () => set({ lines: [] }),
    }),
    {
      name: 'omc-mml-console-terminal',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ lines: state.lines }),
    },
  ),
);
