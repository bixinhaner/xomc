import { useState, useMemo, useCallback, useRef } from 'react';
import { Tabs, Empty, Modal, App } from 'antd';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import { useExecuteStatementsStructured } from '@core/hooks/api/useMmlConsole';
import { useExecuteMMLCommand } from '@core/hooks/api/useMML';
import { useMmlTaskStream } from '@core/hooks/api/useMmlTaskStream';
import { statementToStructured } from '@core/types/mmlConsole';
import type { Statement } from '@core/types/mmlConsole';
import type { MMLTask } from '@core/types/mml';
// v2.4 D37：MmlEditor (拼接命令 textarea + DO 按钮) 已下线，结构化通道由
// ConsoleActionBar 承担；不再 import。
import SubFieldChecklist from './SubFieldChecklist';
import SubFieldInputList from './SubFieldInputList';
import InstancePicker from './InstancePicker';
import TerminalPanel from './TerminalPanel';
import ParamPathExpert, { type ParamPathChangePayload } from './ParameterPathCommand';
import { ConsoleActionBar, type ExecutionMode } from './ConsoleActionBar';
import { InstanceArityInput } from './InstanceArityInput';
import { deriveArityFromSubFields } from './instanceArity';
import { useT } from '@/hooks/useT';

type RightTab = 'control' | 'paramPath';

export interface RightPanelProps {
  onExecuted?: (task: MMLTask) => void;
}

// 操作面板(控制面板 / 参数路径指定)固定高度 = 8 行 PATH 显示高度(用户决策 2026-06-02)。
// 该高度对"未选命令的空态"与"已选命令"一致生效——空态不再塌缩成一小块。
const OP_PANEL_HEIGHT = 320;

function ActiveSubView({ statement }: { statement: Statement }) {
  switch (statement.operationType) {
    case 'LST':
      return <SubFieldChecklist statement={statement} />;
    case 'MOD':
    case 'ADD':
      return <SubFieldInputList statement={statement} />;
    case 'RMV':
      return <InstancePicker statement={statement} />;
    default:
      return null;
  }
}

function hasOnRebootHits(statements: Statement[]): boolean {
  return statements.some((stmt) =>
    stmt.subFields.some(
      (sf) =>
        sf.changeApplies === 'OnReboot' &&
        (stmt.selectedSubFieldIds.includes(sf.id) || stmt.values[sf.mmlCode] !== undefined),
    ),
  );
}

// 任务名命名规则（用户决策）：命令名 + 设备 SN。
// 业务约束：一次执行只允许 1 个命令（多命令不在 /mml/console 范畴；
// 多语句场景走 ScriptTask 单独入口）。多设备允许。
// 输出格式：
//   - 1 设备："{op} {logicalName} {sn}"
//   - N 设备："{op} {logicalName} {sn1} 等N台"
// 后端兜底逻辑：taskName 为空时回落到 "MML console (X statements × Y devices)"。
function buildTaskName(
  stmts: Statement[],
  sns: string[],
  lang: 'zh-CN' | 'en-US' = 'zh-CN',
): string {
  if (stmts.length === 0 || sns.length === 0) return '';
  const first = stmts[0];
  const cmdName =
    first.logicalNameI18n?.[lang] ?? first.logicalNameI18n?.['zh'] ?? first.logicalCode;
  const opVerbZh: Record<string, string> = {
    LST: '查询',
    MOD: '修改',
    ADD: '添加',
    RMV: '删除',
  };
  const opVerb =
    lang === 'zh-CN' ? (opVerbZh[first.operationType] ?? first.operationType) : first.operationType;
  // 防御性 dedup：CommandTree 历史上曾把后端 displayName（已带 op 动词，如「查询 设备
  // 基本信息」）塞进 logicalNameI18n。若 cmdName 已以 opVerb 开头，跳过 prepend，
  // 避免「查询 查询 设备基本信息」体感 bug 复发。
  const cmdPart = cmdName.startsWith(opVerb) ? cmdName : `${opVerb} ${cmdName}`.trim();
  const sn = sns[0];
  const snPart = sns.length === 1 ? sn : `${sn} 等${sns.length}台`;
  return `${cmdPart} ${snPart}`;
}

export default function RightPanel({ onExecuted }: RightPanelProps) {
  const t = useT();
  // antd v5 静态 message 在嵌套 portal / Tabs 内会脱 ConfigProvider 上下文导致
  // 不出 toast；项目内 14+ 处统一改走 App.useApp() scoped 实例（参 LoginPage / PrivateCommand）。
  const { message } = App.useApp();
  const [tab, setTab] = useState<RightTab>('control');
  const activeStatementUid = useMmlConsoleStore((s) => s.activeStatementUid);
  const statements = useMmlConsoleStore((s) => s.statements);
  const selectedDeviceSns = useMmlConsoleStore((s) => s.selectedDeviceSns);
  const updateStatement = useMmlConsoleStore((s) => s.updateStatement);
  const setInstanceSelectors = useMmlConsoleStore((s) => s.setInstanceSelectors);
  // R-9.2：ConsoleActionBar 走结构化通道；MmlEditor 仍用旧 useExecuteStatements。
  // 切换边界：activeStatement 由 CommandTree 加载并附带 subFields → 可做 structured 转换。
  const executeMutation = useExecuteStatementsStructured();
  // "参数路径指定" tab 走旧的 POST /api/v1/mml/execute 裸路径模式：
  // service.go ~L735 在 command_code 为空、param_paths 非空时合成 RAW LST/MOD/ADD/RMV 命令。
  // 与结构化通道不同 —— 无 command_id，未经 sub_field 字典校验，让用户随手输入任意
  // TR-069 path 临时下发（适合 sub_field 未维护、catalog 外路径的探测场景）。
  const rawPathMutation = useExecuteMMLCommand();

  // ParamPathExpert 受控状态：onChange 透传，让 RightPanel 这一层能拼装 execute payload。
  const [rawPathPayload, setRawPathPayload] = useState<ParamPathChangePayload>({
    operationType: 'LST',
    paramPaths: [],
    paramValues: [],
  });

  // T-0123-P4 收尾 (2026-05-22)：execute 后用 task.id 订阅 /events/stream 的
  // mml_device_frame / mml_task_status / mml_task_completed 事件，把终端输出
  // 接回来；da6c1c68 重构时这条路径被推到 P4 但从未落地。
  //
  // 2026-05-28 单PATH执行模式扩展:不再是单 task,而是数组 — 一次 per-path 派发
  // 会产生 N 个 task,全部塞进 currentTaskIds 让 hook 监听所有事件。
  const [currentTaskIds, setCurrentTaskIds] = useState<string[]>([]);
  // 执行模式 Radio: 'batch'(默认整体执行) | 'per-path'(单PATH独立任务)
  const [executionMode, setExecutionMode] = useState<ExecutionMode>('batch');
  // 单PATH 模式下 taskId → path 映射,聚合摘要时把"哪些 path 成功/失败"翻译成可读 path 串。
  // ref 在 dispatch 循环里写入,在 streamMessages.aggregateSummary 闭包里只读;
  // 不进入 React 状态避免触发 hook 闭包重建。
  const pathByTaskIdRef = useRef<Map<string, string>>(new Map());
  // 把 i18n 翻译后的文案传给 frontend-core hook —— hook 自身不依赖 i18n 系统。
  const streamMessages = useMemo(
    () => ({
      running: t('mml.console.terminal.running'),
      cancelled: t('mml.console.terminal.cancelled'),
      completedSummary: ({
        status,
        result,
        successCount,
        failedCount,
      }: {
        status: string;
        result: string;
        successCount: number;
        failedCount: number;
      }) =>
        t('mml.console.terminal.completedSummary', {
          status,
          result,
          success: String(successCount),
          failed: String(failedCount),
        }),
      deviceStatusLabels: {
        completed: t('mml.console.terminal.deviceStatus.completed'),
        failed: t('mml.console.terminal.deviceStatus.failed'),
        expired: t('mml.console.terminal.deviceStatus.expired'),
        cancelled: t('mml.console.terminal.deviceStatus.cancelled'),
      },
      // 设备帧 header:`[SN] METHOD → 任务完成  (任务ID: xxx)`
      deviceHeader: ({
        deviceSn,
        method,
        statusLabel,
        taskId,
      }: {
        deviceSn: string;
        method: string;
        statusLabel: string;
        taskId: string;
      }) =>
        t('mml.console.terminal.deviceHeader', {
          sn: deviceSn,
          method,
          statusLabel,
          taskId,
        }),
      // 所有 task 完成后的聚合摘要:仅在 path 映射齐全(per-path 多任务)时输出;
      // 单 task 场景返回 '' 让 hook 抑制,避免与 completedSummary 重复。
      aggregateSummary: ({
        results,
        successCount,
        failedCount,
        total,
      }: {
        results: ReadonlyArray<{
          taskId: string;
          status: string;
          successCount: number;
          failedCount: number;
        }>;
        successCount: number;
        failedCount: number;
        total: number;
      }) => {
        const map = pathByTaskIdRef.current;
        if (total <= 1 || map.size === 0) return '';
        const successPaths: string[] = [];
        const failedPaths: string[] = [];
        for (const r of results) {
          const path = map.get(r.taskId) ?? r.taskId;
          if (r.status === 'completed' && r.failedCount === 0) {
            successPaths.push(path);
          } else {
            failedPaths.push(path);
          }
        }
        const head = t('mml.console.terminal.aggregateSummary', {
          total: String(total),
          success: String(successCount),
          failed: String(failedCount),
        });
        const successLine =
          successPaths.length > 0
            ? '\n' +
              t('mml.console.terminal.aggregateSuccessPaths', {
                paths: successPaths.join(', '),
              })
            : '';
        const failedLine =
          failedPaths.length > 0
            ? '\n' +
              t('mml.console.terminal.aggregateFailedPaths', {
                paths: failedPaths.join(', '),
              })
            : '';
        return `${head}${successLine}${failedLine}`;
      },
    }),
    [t],
  );
  const {
    lines: terminalLines,
    appendLine,
    clear: clearTerminal,
  } = useMmlTaskStream(currentTaskIds, streamMessages);

  const activeStatement = useMemo(
    () => statements.find((s) => s.uid === activeStatementUid) ?? null,
    [statements, activeStatementUid],
  );

  // R-9.1 ConsoleActionBar 摘要文案：按 op_type 显示已勾选/已填项数
  const summaryText = useMemo(() => {
    if (!activeStatement) return undefined;
    switch (activeStatement.operationType) {
      case 'LST':
        return t('mml.console.actionBar.summaryLst', {
          selected: activeStatement.selectedSubFieldIds.length,
          total: activeStatement.subFields.length,
        });
      case 'MOD': {
        // MOD 现在与 LST 一样支持 path 选填（2026-05-23）；摘要同时展示已勾选数
        // 与实际填值数，让用户知道有多少行被勾上 + 多少行有 value 会下发。
        // 注意：subFields 含 READ_ONLY，selectedSubFieldIds 也只来自可见的
        // READ_WRITE 行（CommandTree 初始化按 defaultSelected & isWritable），
        // total 用 READ_WRITE 行数才贴合视觉。
        const writableTotal = activeStatement.subFields.filter(
          (sf) => sf.accessType !== 'READ_ONLY',
        ).length;
        const writableSelected = activeStatement.selectedSubFieldIds.filter((id) => {
          const sf = activeStatement.subFields.find((x) => x.id === id);
          return sf && sf.accessType !== 'READ_ONLY';
        }).length;
        return t('mml.console.actionBar.summaryMod', {
          selected: writableSelected,
          total: writableTotal,
          filled: Object.keys(activeStatement.values).length,
        });
      }
      case 'ADD':
        return t('mml.console.actionBar.summaryMod', {
          selected: activeStatement.selectedSubFieldIds.length,
          total: activeStatement.subFields.length,
          filled: Object.keys(activeStatement.values).length,
        });
      case 'RMV':
        return t('mml.console.actionBar.summaryRmv', {
          index: activeStatement.rmvInstanceIndex ?? '-',
        });
      default:
        return undefined;
    }
  }, [activeStatement, t]);

  const runExecute = useCallback(async () => {
    try {
      // 把 UI 编辑态 Statement[] 翻译为 StructuredStatement[]，路径以 standardPath
      // 形式直传后端；后端按 device.product_class 翻译为 privatePath。
      const structured = statements.map(statementToStructured);

      // ===== 单PATH执行模式(2026-05-28 用户决策):并发派发 N 个独立任务 =====
      // 触发条件:executionMode='per-path' + 仅 LST/MOD(ADD/RMV 单实例语义,
      // 即使用户切到 per-path 也按 batch 处理)。
      const firstOp = statements[0]?.operationType;
      const perPathEligible =
        executionMode === 'per-path' && (firstOp === 'LST' || firstOp === 'MOD');

      if (perPathEligible) {
        // 把每个 statement 按 paths 拆成 N 个单 path 子 statement
        const splits: Array<{ ss: typeof structured[number]; sourceStmt: typeof statements[number]; path: string }> = [];
        for (let i = 0; i < structured.length; i++) {
          const ss = structured[i];
          const srcStmt = statements[i];
          if (ss.paths.length === 0) {
            splits.push({ ss, sourceStmt: srcStmt, path: '' });
            continue;
          }
          for (const p of ss.paths) {
            const splitValues: Record<string, string> | undefined =
              ss.values && ss.values[p] !== undefined ? { [p]: ss.values[p] } : undefined;
            const sub: typeof ss = {
              ...ss,
              paths: [p],
            };
            if (splitValues) sub.values = splitValues;
            else delete sub.values;
            splits.push({ ss: sub, sourceStmt: srcStmt, path: p });
          }
        }

        const total = splits.length;
        if (total === 0) {
          message.warning(t('mml.console.execute.noStatements'));
          return;
        }

        // 终端先打 header
        appendLine({
          type: 'info',
          text: t('mml.console.terminal.perPathHeader', { count: total }),
          timestamp: new Date().toLocaleTimeString(),
        });

        // 并发派发,Promise.allSettled 让部分失败不影响其它派发
        const dispatchResults = await Promise.allSettled(
          splits.map((s, idx) => {
            const leafName = s.path.split('.').pop() || s.path || '';
            const cmdName =
              s.sourceStmt.logicalNameI18n?.['zh-CN'] ??
              s.sourceStmt.logicalNameI18n?.['zh'] ??
              s.sourceStmt.logicalCode ??
              '';
            const sn = selectedDeviceSns[0] ?? '';
            const snPart =
              selectedDeviceSns.length <= 1
                ? sn
                : `${sn} 等${selectedDeviceSns.length}台`;
            // 任务名:{logicalName} [{leafName}] {sn}
            const taskName = leafName
              ? `${cmdName} [${leafName}] ${snPart}`.trim()
              : `${cmdName} ${snPart}`.trim();
            return executeMutation
              .mutateAsync({
                statements: [s.ss],
                deviceSns: selectedDeviceSns,
                executeType: 'immediate',
                taskName,
              })
              .then((task) => ({ idx, task, leafName: leafName || s.path }));
          }),
        );

        const taskIds: string[] = [];
        const newPathByTaskId = new Map<string, string>();
        let failedCount = 0;
        dispatchResults.forEach((r, idx) => {
          const split = splits[idx];
          const leafName = split.path.split('.').pop() || split.path || '';
          const i = idx + 1;
          if (r.status === 'fulfilled') {
            taskIds.push(r.value.task.id);
            newPathByTaskId.set(r.value.task.id, split.path || leafName || '-');
            appendLine({
              type: 'info',
              text: t('mml.console.terminal.perPathDispatched', {
                i: String(i),
                total: String(total),
                taskId: r.value.task.id,
                path: split.path || leafName || '-',
              }),
              timestamp: new Date().toLocaleTimeString(),
            });
            // 注意:仅最后一次 onExecuted 回调(避免 N 次抖动)
            if (idx === dispatchResults.length - 1) onExecuted?.(r.value.task);
          } else {
            failedCount += 1;
            const errMsg = r.reason instanceof Error ? r.reason.message : String(r.reason);
            appendLine({
              type: 'stderr',
              text: `[${i}/${total}] dispatch failed  path=${split.path || '-'}  error=${errMsg}`,
              timestamp: new Date().toLocaleTimeString(),
            });
          }
        });

        if (taskIds.length > 0) {
          // 把所有 task_id 塞给 SSE hook 监听
          // 先写 pathByTaskId 映射(给 aggregateSummary 闭包用),再 setCurrentTaskIds
          // 触发 hook idsKey 变化重置 completedResultsRef + aggregateEmittedRef。
          pathByTaskIdRef.current = newPathByTaskId;
          setCurrentTaskIds(taskIds);
        }
        appendLine({
          type: failedCount > 0 ? 'stderr' : 'info',
          text: t('mml.console.terminal.allPerPathDispatched', { count: taskIds.length }),
          timestamp: new Date().toLocaleTimeString(),
        });
        if (taskIds.length > 0) {
          message.success(
            t('mml.console.execute.success', { taskId: `${taskIds.length} tasks` }),
          );
        }
        return;
      }

      // ===== 整体执行模式(默认):1 个任务包含全部 statements =====
      const task = await executeMutation.mutateAsync({
        statements: structured,
        deviceSns: selectedDeviceSns,
        executeType: 'immediate',
        taskName: buildTaskName(statements, selectedDeviceSns),
      });
      // 接入 SSE:先切 currentTaskIds(hook 会复位 lines)再种一条"已派发"行,
      // 避免 task 创建瞬间到第一台设备完成之间终端是空白的体感。
      setCurrentTaskIds([task.id]);
      appendLine({
        type: 'info',
        text: t('mml.console.terminal.dispatched', {
          taskId: task.id,
          devices: String(selectedDeviceSns.length),
        }),
        timestamp: new Date().toLocaleTimeString(),
      });
      message.success(t('mml.console.execute.success', { taskId: task.id }));
      onExecuted?.(task);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      message.error(t('mml.console.execute.failed', { message: msg }));
    }
  }, [
    appendLine,
    executeMutation,
    executionMode,
    message,
    onExecuted,
    selectedDeviceSns,
    statements,
    t,
  ]);

  const handleExecute = useCallback(() => {
    if (selectedDeviceSns.length === 0) {
      message.warning(t('mml.console.execute.noDevices'));
      return;
    }
    if (statements.length === 0) {
      message.warning(t('mml.console.execute.noStatements'));
      return;
    }
    if (hasOnRebootHits(statements)) {
      Modal.confirm({
        title: t('mml.console.editor.onRebootConfirm'),
        onOk: runExecute,
      });
      return;
    }
    void runExecute();
  }, [runExecute, selectedDeviceSns.length, statements, t]);

  // 裸路径模式执行：构造与 ExecuteHTTPRequest (handler.go) 一致的 snake_case payload，
  // 由 axios 拦截器跳过转换直接送后端。后端走 service.go raw param_paths 分支生成
  // RAW {LST/MOD/ADD/RMV} 命令，fanout 后下发到目标设备。
  const handleRawPathExecute = useCallback(async () => {
    if (selectedDeviceSns.length === 0) {
      message.warning(t('mml.console.execute.noDevices'));
      return;
    }
    const trimmedPaths = rawPathPayload.paramPaths
      .map((p) => p.trim())
      .filter(Boolean);
    if (trimmedPaths.length === 0) {
      message.warning(t('mml.console.execute.noParamPaths'));
      return;
    }
    const op = rawPathPayload.operationType.toUpperCase();
    // MOD 必填 value（后端 service.go ~L781 否则 400）；前端先拦下避免 toast 噪声
    if (op === 'MOD') {
      for (let i = 0; i < trimmedPaths.length; i += 1) {
        const v = rawPathPayload.paramValues[i];
        if (!v || v.trim() === '') {
          message.warning(t('mml.console.execute.modValueRequired', { path: trimmedPaths[i] }));
          return;
        }
      }
    }
    try {
      const task = await rawPathMutation.mutateAsync({
        payload: {
          device_sns: selectedDeviceSns,
          param_paths: trimmedPaths,
          // ADD/RMV/LST 不消费 param_values，但保持下标对齐让后端正确按 i 配对
          param_values: trimmedPaths.map((_, i) => rawPathPayload.paramValues[i] ?? ''),
          operation_type: op,
          execute_type: 'immediate',
          task_name: `${op} ${trimmedPaths[0]}${
            selectedDeviceSns.length === 1 ? ` ${selectedDeviceSns[0]}` : ` 等${selectedDeviceSns.length}台`
          }`,
        },
      });
      setCurrentTaskIds([task.id]);
      appendLine({
        type: 'info',
        text: t('mml.console.terminal.dispatched', {
          taskId: task.id,
          devices: String(selectedDeviceSns.length),
        }),
        timestamp: new Date().toLocaleTimeString(),
      });
      message.success(t('mml.console.execute.success', { taskId: task.id }));
      onExecuted?.(task);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      message.error(t('mml.console.execute.failed', { message: msg }));
    }
  }, [appendLine, message, onExecuted, rawPathMutation, rawPathPayload, selectedDeviceSns, t]);

  // R-4：从当前 activeStatement 的 subFields 推断 instance arity；
  // > 0 时在 ActiveSubView 之前渲染 InstanceArityInput，4 个 op 都适用。
  const instanceArity = useMemo(
    () => (activeStatement ? deriveArityFromSubFields(activeStatement.subFields) : 0),
    [activeStatement],
  );

  const handleSelectorChange = useCallback(
    (key: string, value: string) => {
      if (!activeStatement) return;
      const next: Record<string, string> = {
        ...(activeStatement.instanceSelectors ?? {}),
        [key]: value,
      };
      setInstanceSelectors(activeStatement.uid, next);
    },
    [activeStatement, setInstanceSelectors],
  );

  // v2.4 D36：LST 全选/全部取消 合一为 toggleAll 单按钮，按当前选中比例切换文案。
  // 2026-05-23：MOD path 改为选填后，ToggleAll 同样适用（与 LST 同款语义）；
  // 但 MOD 的可勾选行只含 READ_WRITE（SubFieldInputList 已过滤），allSelected /
  // toggle 也只按 READ_WRITE 集合计数，免得 READ_ONLY 字段被误"全选"。
  const toggleableFieldIds = useMemo<string[]>(() => {
    if (!activeStatement) return [];
    if (activeStatement.operationType === 'LST') {
      return activeStatement.subFields.map((sf) => sf.id);
    }
    if (activeStatement.operationType === 'MOD') {
      return activeStatement.subFields
        .filter((sf) => sf.accessType !== 'READ_ONLY')
        .map((sf) => sf.id);
    }
    return [];
  }, [activeStatement]);
  const supportsToggleAll =
    activeStatement?.operationType === 'LST' ||
    activeStatement?.operationType === 'MOD';

  const allSelected = useMemo(() => {
    if (!activeStatement || !supportsToggleAll) return false;
    if (toggleableFieldIds.length === 0) return false;
    const selectedSet = new Set(activeStatement.selectedSubFieldIds);
    return toggleableFieldIds.every((id) => selectedSet.has(id));
  }, [activeStatement, supportsToggleAll, toggleableFieldIds]);

  const handleToggleAll = useCallback(() => {
    if (!activeStatement || !supportsToggleAll) return;
    // 切换只影响 toggleable 集合；其它行（如 MOD 的 READ_ONLY）的勾选状态保留
    const toggleableSet = new Set(toggleableFieldIds);
    const others = activeStatement.selectedSubFieldIds.filter(
      (id) => !toggleableSet.has(id),
    );
    updateStatement(activeStatement.uid, {
      selectedSubFieldIds: allSelected
        ? others
        : [...others, ...toggleableFieldIds],
    });
  }, [
    activeStatement,
    allSelected,
    supportsToggleAll,
    toggleableFieldIds,
    updateStatement,
  ]);

  return (
    // 视口高度布局(用户决策 2026-06-02):操作面板 / 参数路径指定 固定高度(≈8 行 PATH),
    // 终端输出 flex:1 自适应填充其下方剩余空间,延伸到浏览器最下方。
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
        height: 'calc(100vh - 180px)',
        minHeight: 460,
      }}
    >
      <div style={{ flexShrink: 0 }}>
      <Tabs
        activeKey={tab}
        onChange={(k) => setTab(k as RightTab)}
        items={[
          {
            key: 'control',
            label: t('mml.tabs.controlPanel'),
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {/* v2.4 D37：MmlEditor (拼接命令 textarea + DO 执行按钮) 已下线，
                    结构化通道由下方 ConsoleActionBar 承担；用户在 SubField 区勾选 path
                    → 后端 API 通过结构化 standardPath[] 传，不再生成 MML 文本 */}
                {/* 参数列表（LST/MOD/ADD/RMV 因 op 不同换皮）：内层 SubFieldChecklist /
                    SubFieldInputList 现为 height:100% 撑满外层固定盒并自带 overflowY:auto;
                    外层盒 overflow:auto 仅为 RMV(InstancePicker 无内滚)兜底。内层是 100%
                    高度,外层无可滚内容 → 不会出现双层滚动条。 */}
                {/* 操作面板固定高度盒(用户决策 2026-06-02)：无论是否选中命令,
                    高度恒为 OP_PANEL_HEIGHT(8 行 PATH);空态居中、命令态由内层
                    viewport 撑满并自行滚动。 */}
                <div
                  style={{
                    height: OP_PANEL_HEIGHT,
                    display: 'flex',
                    flexDirection: 'column',
                    minHeight: 0,
                  }}
                >
                  {activeStatement ? (
                    <>
                      {/* R-4：命令含多层 {i} 占位符时显示 instance 索引输入；
                          arity=0 时组件自身返回 null，不留空白。 */}
                      <InstanceArityInput
                        arity={instanceArity}
                        values={activeStatement.instanceSelectors ?? {}}
                        onChange={handleSelectorChange}
                        operationType={activeStatement.operationType}
                        instanceRangeMeta={activeStatement.instanceRangeMeta}
                      />
                      {/* 内层 viewport(SubFieldChecklist/InputList 已 height:100%)填满剩余高度,
                          自身 overflowY:auto 单根滚动条；外层 overflow:auto 兜底 RMV 等无内滚组件。 */}
                      <div style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
                        <ActiveSubView statement={activeStatement} />
                      </div>
                    </>
                  ) : (
                    <div
                      style={{
                        flex: 1,
                        minHeight: 0,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        border: '1px solid #f0f0f0',
                        borderRadius: 4,
                      }}
                    >
                      <Empty description={t('mml.console.stepBar.step2')} />
                    </div>
                  )}
                </div>
                {/* R-9.1 底部固定操作行：执行按钮显式带 op_type，避免随
                    LST/MOD/ADD/RMV 切换位置漂移。仅当有 activeStatement
                    时挂载（无命令选中时不渲染，避免空按钮误导）。 */}
                {activeStatement && (
                  <ConsoleActionBar
                    operationType={activeStatement.operationType}
                    summaryText={summaryText}
                    deviceCount={selectedDeviceSns.length}
                    disabled={selectedDeviceSns.length === 0 || statements.length === 0}
                    loading={executeMutation.isPending}
                    onToggleAll={supportsToggleAll ? handleToggleAll : undefined}
                    allSelected={allSelected}
                    onExecute={handleExecute}
                    executionMode={executionMode}
                    onExecutionModeChange={setExecutionMode}
                  />
                )}
              </div>
            ),
          },
          {
            key: 'paramPath',
            label: t('mml.tabs.parameterPathCommand'),
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {/* 参数路径指定同样固定为 8 行 PATH 高度(用户决策 2026-06-02),与控制面板一致。 */}
                <div style={{ height: OP_PANEL_HEIGHT, overflow: 'auto' }}>
                  <ParamPathExpert
                    command={null}
                    onChange={setRawPathPayload}
                  />
                </div>
                <ConsoleActionBar
                  operationType={rawPathPayload.operationType as Statement['operationType']}
                  deviceCount={selectedDeviceSns.length}
                  disabled={
                    selectedDeviceSns.length === 0 ||
                    rawPathPayload.paramPaths.length === 0
                  }
                  loading={rawPathMutation.isPending}
                  onExecute={handleRawPathExecute}
                />
              </div>
            ),
          },
        ]}
      />
      </div>
      {/* 终端输出:flex:1 自适应填充操作面板下方剩余空间,高度延伸到浏览器最下方。 */}
      <div style={{ flex: 1, minHeight: 160 }}>
        <TerminalPanel
          lines={terminalLines}
          onClear={() => {
            clearTerminal();
            setCurrentTaskIds([]);
          }}
        />
      </div>
    </div>
  );
}
