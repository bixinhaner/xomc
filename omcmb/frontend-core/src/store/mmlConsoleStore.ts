/**
 * T-0123-P2-a Console Zustand 状态机 — frontend-core
 *
 * PRD: docs/design/mml-restore-old-interaction-plan-20260514.md §7.4
 *
 * 核心职责：维护三栏交互的运行期状态，双向同步 UI ↔ mmlText。
 *
 * 双向同步流：
 *   [UI 勾选/输入] → setStatementsFromUI → 重算 mmlText（本地 render）
 *   [textbox 改]   → setMmlText (debounce 300ms) → 服务端 parse → 写 statements
 *
 * 防 sync 循环：用 `syncSource` 字段标记本次更新的源（'ui' / 'text' / 'none'），
 * 订阅 effect 检查 syncSource ≠ 'self' 才回流，避免 ping-pong。
 *
 * 本地渲染 vs 服务端 render：
 *   - P2-a 走本地简化渲染（renderStatementLocal）— 纯格式串接，零网络
 *     语法遵循 PRD §1.5：LST `:lstId={K1,K2}` / MOD `:K=V,K=V` / ADD 同 MOD
 *     / RMV `:Index=N`；空字段返裸 op
 *   - 服务端 /render 兜底通道：组件层判定本地不够时调 hooks/useRenderMML
 *
 * 本地解析 vs 服务端 parse：
 *   - 解析需要命令字典查找（command_id 反查），本地无法做
 *   - 一律走服务端 POST /parse via useParseMML
 *   - debounce 在 store 层 setTimeout 调度，组件层只是 setMmlText
 */

import { create } from 'zustand';

import type {
  Statement,
  ParseError,
  ConsoleSupportedOp,
} from '../types/mmlConsole';

// ============================================================
// 本地渲染器（PRD §1.5 语法）
// ============================================================

/**
 * 把单条 Statement 渲染为 MML 字符串片段（不含末尾 `;`）。
 *
 * 与后端 `internal/mml/mml_renderer.go::RenderStatement` 行为一致：
 *   - 空字段返裸 op："LST DEVICE_INFO"
 *   - 含字段时按 sort_order 排序：sub_fields 已在 store 内按 sort_order 存放
 *   - 特殊字符（, ; : = { } 空格 "）双引号包裹 + 内部 \" escape
 */
export function renderStatementLocal(stmt: Statement): string {
  const op = stmt.operationType.toUpperCase() as ConsoleSupportedOp;
  // 用户决策 2026-05-18 实测 bug：部分命令 backend 返回 logical_code 为空字符串
  // （TypeScript 标的是 required string 但运行时仍可能空），导致 TextArea 显示空。
  // fallback 到 commandCode（mml_commands.command_code 是 NOT NULL，必定非空）。
  const code = stmt.logicalCode || stmt.commandCode;
  if (!code) return '';

  switch (op) {
    case 'LST': {
      // 顺序：selectedSubFieldIds → 在 subFields 中找对应 mml_code，按 sortOrder 排序
      const idSet = new Set(stmt.selectedSubFieldIds);
      const codes = stmt.subFields
        .filter((sf) => idSet.has(sf.id))
        .sort((a, b) =>
          a.sortOrder !== b.sortOrder
            ? a.sortOrder - b.sortOrder
            : a.mmlCode.localeCompare(b.mmlCode)
        )
        .map((sf) => sf.mmlCode);
      if (codes.length === 0) return `LST ${code}`;
      return `LST ${code}:lstId={${codes.join(',')}}`;
    }

    case 'MOD':
    case 'ADD': {
      const sorted = [...stmt.subFields].sort((a, b) =>
        a.sortOrder !== b.sortOrder
          ? a.sortOrder - b.sortOrder
          : a.mmlCode.localeCompare(b.mmlCode)
      );
      const seen = new Set<string>();
      const pairs: string[] = [];
      // sub_fields 顺序优先
      for (const sf of sorted) {
        if (stmt.values[sf.mmlCode] !== undefined) {
          pairs.push(`${sf.mmlCode}=${quoteValueIfNeeded(stmt.values[sf.mmlCode])}`);
          seen.add(sf.mmlCode);
        }
      }
      // Values 中未命中 sub_fields 的 key 按字典序追加
      const unknownKeys = Object.keys(stmt.values).filter((k) => !seen.has(k));
      unknownKeys.sort();
      for (const k of unknownKeys) {
        pairs.push(`${k}=${quoteValueIfNeeded(stmt.values[k])}`);
      }
      if (pairs.length === 0) return `${op} ${code}`;
      return `${op} ${code}:${pairs.join(',')}`;
    }

    case 'RMV': {
      if (stmt.rmvInstanceIndex !== undefined && stmt.rmvInstanceIndex !== null) {
        return `RMV ${code}:Index=${stmt.rmvInstanceIndex}`;
      }
      return `RMV ${code}`;
    }

    default:
      return '';
  }
}

/**
 * 把多条 Statement 渲染为完整 MML 字符串（`;` 分隔 + 末尾 `;`）。
 * 空数组返空字符串。
 */
export function renderStatementsLocal(statements: Statement[]): string {
  if (statements.length === 0) return '';
  return statements.map(renderStatementLocal).filter(Boolean).join(';') + ';';
}

/**
 * 值含 MML 特殊字符时双引号包裹 + 内部 `"` 转义。
 * 与后端 `quoteValueIfNeeded` 行为一致。
 */
function quoteValueIfNeeded(v: string): string {
  if (!/[,;:={} "]/.test(v)) return v;
  return `"${v.replace(/"/g, '\\"')}"`;
}

// ============================================================
// Store
// ============================================================

/**
 * 双向同步来源标记：
 *   - 'ui'    : UI 操作触发（勾选/输入）
 *   - 'text'  : textbox 修改触发
 *   - 'none'  : 初始 / 重置
 *
 * 组件层订阅时按需对比，避免回流：
 *   - mmlText 订阅器：syncSource === 'ui' 才更新自身展示
 *   - statements 订阅器：syncSource === 'text' 才重渲染 UI 控件
 */
export type SyncSource = 'ui' | 'text' | 'none';

interface MmlConsoleState {
  // 数据
  selectedDeviceSns: string[];
  statements: Statement[];
  mmlText: string;
  activeStatementUid: string | null;
  lang: 'zh-CN' | 'en-US';
  parseErrors: ParseError[];
  syncSource: SyncSource;

  // 防抖控制
  parsePending: boolean;            // 当前是否有 pending parse 调用
  parseDebounceTimer: ReturnType<typeof setTimeout> | null;

  // 设备选择
  setSelectedDeviceSns(sns: string[]): void;

  // Statement 操作（UI 触发 → 'ui' source）
  appendStatement(stmt: Statement): void;
  /**
   * 用单条 statement 替换全部 statements（覆盖语义）。
   * MML 控制台命令树要求：连续点击命令时，控制面板显示 **最近一条** 而不是
   * 累加列表。多语句脚本场景仍用 textbox 直接编辑或 setMmlText 进入。
   */
  replaceStatement(stmt: Statement): void;
  removeStatement(uid: string): void;
  updateStatement(uid: string, patch: Partial<Statement>): void;
  toggleSubField(uid: string, subFieldId: string): void;
  setValue(uid: string, mmlCode: string, value: string): void;
  setRmvIndex(uid: string, index: number | undefined): void;

  // 文本操作（textbox 触发 → 'text' source）
  setMmlText(text: string): void;
  /** 防抖入口：组件 onChange 调本方法，store 内部 setTimeout 后触发 parse */
  setMmlTextDebounced(
    text: string,
    parseFn: (text: string) => Promise<{ statements: Statement[]; parseErrors: ParseError[] }>
  ): void;

  // 焦点
  setActiveStatement(uid: string | null): void;

  // 语言
  setLang(lang: 'zh-CN' | 'en-US'): void;

  // 重置 / 同步源探测
  reset(): void;
  /** 测试 / 高级用法：手动重算 mmlText */
  recomputeMmlText(): void;
}

const DEBOUNCE_MS = 300;

export const useMmlConsoleStore = create<MmlConsoleState>((set, get) => ({
  selectedDeviceSns: [],
  statements: [],
  mmlText: '',
  activeStatementUid: null,
  lang: 'zh-CN',
  parseErrors: [],
  syncSource: 'none',
  parsePending: false,
  parseDebounceTimer: null,

  setSelectedDeviceSns(sns) {
    set({ selectedDeviceSns: [...sns] });
  },

  appendStatement(stmt) {
    const newStatements = [...get().statements, stmt];
    set({
      statements: newStatements,
      activeStatementUid: stmt.uid,
      mmlText: renderStatementsLocal(newStatements),
      syncSource: 'ui',
    });
  },

  replaceStatement(stmt) {
    set({
      statements: [stmt],
      activeStatementUid: stmt.uid,
      mmlText: renderStatementsLocal([stmt]),
      syncSource: 'ui',
    });
  },

  removeStatement(uid) {
    const newStatements = get().statements.filter((s) => s.uid !== uid);
    const activeUid = get().activeStatementUid;
    set({
      statements: newStatements,
      activeStatementUid: activeUid === uid ? null : activeUid,
      mmlText: renderStatementsLocal(newStatements),
      syncSource: 'ui',
    });
  },

  updateStatement(uid, patch) {
    const newStatements = get().statements.map((s) =>
      s.uid === uid ? { ...s, ...patch } : s
    );
    set({
      statements: newStatements,
      mmlText: renderStatementsLocal(newStatements),
      syncSource: 'ui',
    });
  },

  toggleSubField(uid, subFieldId) {
    const newStatements = get().statements.map((s) => {
      if (s.uid !== uid) return s;
      const selected = new Set(s.selectedSubFieldIds);
      if (selected.has(subFieldId)) {
        selected.delete(subFieldId);
      } else {
        selected.add(subFieldId);
      }
      return { ...s, selectedSubFieldIds: Array.from(selected) };
    });
    set({
      statements: newStatements,
      mmlText: renderStatementsLocal(newStatements),
      syncSource: 'ui',
    });
  },

  setValue(uid, mmlCode, value) {
    const newStatements = get().statements.map((s) => {
      if (s.uid !== uid) return s;
      const values = { ...s.values };
      if (value === '' || value === undefined) {
        delete values[mmlCode];
      } else {
        values[mmlCode] = value;
      }
      return { ...s, values };
    });
    set({
      statements: newStatements,
      mmlText: renderStatementsLocal(newStatements),
      syncSource: 'ui',
    });
  },

  setRmvIndex(uid, index) {
    const newStatements = get().statements.map((s) =>
      s.uid === uid ? { ...s, rmvInstanceIndex: index } : s
    );
    set({
      statements: newStatements,
      mmlText: renderStatementsLocal(newStatements),
      syncSource: 'ui',
    });
  },

  setMmlText(text) {
    set({ mmlText: text, syncSource: 'text' });
  },

  setMmlTextDebounced(text, parseFn) {
    const prev = get().parseDebounceTimer;
    if (prev) clearTimeout(prev);
    // 立即写本地 text 状态（textbox UI 响应不阻塞）；
    // 防抖到点才发 parse 请求并据此刷 statements。
    set({ mmlText: text, syncSource: 'text', parsePending: true });
    const timer = setTimeout(async () => {
      try {
        const resp = await parseFn(text);
        // P2-a 防御：本次 parse 完成时，若用户已又敲了新文本（mmlText !== text），
        // 丢弃本次结果（race-loser 直接退出）。
        if (get().mmlText !== text) {
          set({ parsePending: false, parseDebounceTimer: null });
          return;
        }
        set({
          statements: resp.statements,
          parseErrors: resp.parseErrors,
          parsePending: false,
          parseDebounceTimer: null,
          // 注意：syncSource 保持 'text'，让 UI 订阅者据此重渲染 sub-field 控件
          syncSource: 'text',
        });
      } catch (e) {
        // 网络错误：保留旧 statements，清 pending；UI 层 React Query mutation
        // onError 路径已 toast 提示
        set({ parsePending: false, parseDebounceTimer: null });
      }
    }, DEBOUNCE_MS);
    set({ parseDebounceTimer: timer });
  },

  setActiveStatement(uid) {
    set({ activeStatementUid: uid });
  },

  setLang(lang) {
    set({ lang });
  },

  reset() {
    const t = get().parseDebounceTimer;
    if (t) clearTimeout(t);
    set({
      selectedDeviceSns: [],
      statements: [],
      mmlText: '',
      activeStatementUid: null,
      parseErrors: [],
      syncSource: 'none',
      parsePending: false,
      parseDebounceTimer: null,
    });
  },

  recomputeMmlText() {
    set({ mmlText: renderStatementsLocal(get().statements), syncSource: 'ui' });
  },
}));

/** 默认 debounce 时长（暴露给测试 / 高级配置）。 */
export const MML_CONSOLE_DEBOUNCE_MS = DEBOUNCE_MS;
