import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { message, Space } from 'antd';
import { useT, type TranslateFn } from '@/hooks/useT';
import { useExecuteStatementsStructured } from '@core/hooks/api/useMmlConsole';
import { useExecuteMMLCommand } from '@core/hooks/api/useMML';
import { mmlApi } from '@core/services/api/mmlApi';
import SelectionBar from './components/SelectionBar';
import DeviceSelectModal from './components/DeviceSelectModal';
import CommandSelectModal from './components/CommandSelectModal';
import ConfigParamsModal from './components/ConfigParamsModal';
import CommandHistoryPanel from './components/CommandHistoryPanel';
import ResultTable from './components/ResultTable';
import type {
  CommandItem,
  ExecMeta,
  ExecRequest,
  OperationMode,
  ResultColumn,
  ResultRow,
} from './types';
import type { MMLTask } from '@core/types/mml';
import { isReadOp, opLabel, opLabelI18nKey } from './constants';
import { commandUsesPathSelection } from './pathSelection';
import { useConsoleHistory } from './useConsoleHistory';
import { useExecStream } from './useExecStream';
import {
  applyFrameToRow,
  buildColumns,
  buildColumnsFromRawPaths,
  buildDeviceRows,
  buildMODReadbackRows,
  buildPerPathStatementPaths,
  buildRawExecutePayload,
  buildStandardQueryColumns,
  buildStandardRawRows,
  buildStructuredStatement,
  initialPendingRows,
  isStructuredOp,
  rawCommandName,
  type DeviceFramePayload,
  type TaskCompletedPayload,
} from './adapters';

/** 任务终态集合（轮询兜底据此判定执行已结束）。 */
const TERMINAL_TASK_STATUS = new Set(['completed', 'failed', 'expired', 'cancelled', 'timeout']);

let recordSeq = 1;

/** 进行中的一次执行（SSE 帧实时回填 rows，完成后原地更新对应命令记录）。 */
interface LiveExec {
  taskId: string;
  /** 点击执行时即插入的命令记录 id，完成收口按它原地更新（非新增）。 */
  recordId: string;
  /** 下发时间 HH:mm:ss，作为命令记录的展示时间（收口时沿用，不被完成时间覆盖）。 */
  startTime: string;
  meta: ExecMeta;
  columns: ResultColumn[];
  rows: ResultRow[];
  deviceCount: number;
  /**
   * #196：MOD 下发值 path→value。MOD 命令后端自动追加回读 LST（compound），
   * 收口时据此走 buildMODReadbackRows 关联「下发 vs 回读」；非 MOD 为空。
   */
  setValues?: Record<string, string>;
}

function buildTerminalFallbackRows(le: LiveExec, task: MMLTask, t: TranslateFn): ResultRow[] {
  const success = task.successCount ?? 0;
  const failed = task.failedCount ?? 0;
  const fallbackFault = t('mml.consoleV2.result.terminalResultUnavailable', { success, failed });
  return le.rows.map((row) => {
    if (!['pending', 'running'].includes(row.status)) return row;
    return {
      ...row,
      status: 'unverified',
      faultCode: row.faultCode ?? fallbackFault,
      unverifiedReason: 'query-failed',
    };
  });
}

/**
 * MML 控制台 V2（设计 docs/design/mml-console-redesign-20260603.md，§3.12 后端对接）。
 *
 * 顶部条 ①设备 ②命令 ③配置参数(弹框) + 执行;主舞台「命令记录(可收缩) ｜ 执行结果」。
 * 执行走真实后端：标准命令 → POST …/execute-statements-structured；裸路径 → POST /mml/execute；
 * 结果经 SSE（mml_device_frame）就地回填表格行，整体完成后落入命令记录。
 */
export default function MMLConsole() {
  const t = useT();
  const [deviceModalOpen, setDeviceModalOpen] = useState(false);
  const [commandModalOpen, setCommandModalOpen] = useState(false);
  const [configModalOpen, setConfigModalOpen] = useState(false);

  const [selectedSns, setSelectedSns] = useState<string[]>([]);
  // 所选产品 ID（设备弹框强制同一产品）：用于「选择命令 / 配置参数」按产品拉不支持 path 过滤。
  const [selectedProductId, setSelectedProductId] = useState<string>('');
  // 所选产品类型：命令树 / 参数列表按 param_mappings 权威支持集合过滤。
  const [selectedProductClass, setSelectedProductClass] = useState<string>('');
  const [command, setCommand] = useState<CommandItem | null>(null);
  const [selectedPathKeys, setSelectedPathKeys] = useState<string[]>([]);
  const [config, setConfig] = useState<ExecRequest | null>(null);
  const [configTouched, setConfigTouched] = useState(false);
  // 配置参数弹框打开时激活的标签：命令参数(standard) / 指定参数(raw)。
  const [configMode, setConfigMode] = useState<OperationMode>('standard');

  // 命令记录数据层（P3 接后端；当前凭内存 + localStorage 命令 ID，详见 useConsoleHistory）。
  const { records, activeId, activeRecord, select, append, update, clear } = useConsoleHistory();
  // #217：dispatching 仅遮挡「下发 mutateAsync」那几百毫秒的 HTTP 往返（避免重复点击同一次下发），
  // 不再绑死整个任务执行期——下发成功插入命令记录后立即解锁，允许并发发起第二条命令。
  const [dispatching, setDispatching] = useState(false);
  const [historyCollapsed, setHistoryCollapsed] = useState(true); // 默认收缩(§3.10.4)

  // #217：多条命令并发在途。taskId → 进行中的执行（SSE 帧实时回填各自记录行）。
  // 用 ref 让 SSE 帧/完成回调与轮询兜底读到最新 Map，而不进 effect 依赖（避免重建订阅/重启轮询）。
  const [liveExecs, setLiveExecs] = useState<Map<string, LiveExec>>(() => new Map());
  const liveExecsRef = useRef<Map<string, LiveExec>>(liveExecs);
  // 已收口的 taskId（防 SSE 完成与轮询兜底双路径重复落记录）。
  const finalizedRef = useRef<Set<string>>(new Set());
  const localizedOpLabel = useCallback(
    (op: string | undefined) => {
      const key = opLabelI18nKey(op);
      return key ? t(key) : opLabel(op);
    },
    [t],
  );

  useEffect(() => {
    liveExecsRef.current = liveExecs;
  }, [liveExecs]);

  // 统一收口：拉 /results → buildDeviceRows（逐 PATH 合并）→ 落命令记录。
  // SSE 完成帧（handleCompleted）与轮询兜底都走这里，确保结果不取自被逐帧覆盖的 SSE 行。
  const finalizeFromResults = async (taskId: string, fallbackRows?: ResultRow[]): Promise<boolean> => {
    if (finalizedRef.current.has(taskId)) return true;
    const le = liveExecsRef.current.get(taskId);
    if (!le) return false;
    let rows: ResultRow[] | null = null;
    try {
      // MOD 复合（下发 + 回读 LST）每设备 2 条 device_task，pageSize 预留回读条目空间。
      const perDevice = Math.max(le.columns.length, 1) + (le.setValues ? 1 : 0);
      const pageSize = Math.max(le.deviceCount * perDevice, 50);
      const resp = await mmlApi.getTaskResults(taskId, 1, pageSize);
      if (resp.items.length > 0) {
        // #196：MOD 命令后端自动追加回读 LST → 走「下发 vs 回读」关联视图（操作类型 / 前后对比 /
        // 双报文页签）；其余命令按逐 PATH 合并。
        rows =
          le.meta.operationType === 'MOD' && le.setValues
            ? buildMODReadbackRows(resp.items, le.setValues)
            : buildDeviceRows(resp.items, le.columns, le.meta.read);
      }
    } catch {
      /* 拉取失败：交给轮询下个 tick 重试 */
    }
    // SSE 完成帧拉不到结果时不收口，交给轮询下个 tick 重试；轮询已确认后端终态时传入
    // fallbackRows，避免后端已完成但前端 liveExec 长期残留，结果区一直 loading。
    rows ??= fallbackRows ?? null;
    if (!rows) return false;
    if (finalizedRef.current.has(taskId)) return true;
    finalizedRef.current.add(taskId);
    // 原地更新点击执行时插入的「执行中」记录（同 recordId/commandId）：沿用下发时间、补齐
    // 结果行、记录态置 done。#217：用 update 不抢占用户当前选中焦点——后台并发任务收口时，
    // 若用户正看着另一条记录，不被强行切走（与下发时 append 主动选中新命令的语义区分）。
    update({
      id: le.recordId,
      status: 'done',
      commandId: le.taskId,
      time: le.startTime,
      commandName: le.meta.commandName ?? t('mml.consoleV2.rawPathCommand', { op: localizedOpLabel(le.meta.operationType) }),
      operationType: le.meta.operationType,
      deviceCount: le.deviceCount,
      execMeta: le.meta,
      columns: le.columns,
      rows,
      setValues: le.setValues,
    });
    // 该任务收口完成：从在途 Map 移除（其余并发任务不受影响）。
    setLiveExecs((prev) => {
      if (!prev.has(taskId)) return prev;
      const next = new Map(prev);
      next.delete(taskId);
      return next;
    });
    return true;
  };

  const structuredMutation = useExecuteStatementsStructured();
  const rawMutation = useExecuteMMLCommand();

  // ── SSE 实时回填（设计 §3.12.2 + #217 多任务并发）────────────────────────────
  // 按 frame.task_id 路由到对应在途执行：更新其 rows，并把进行中的命令记录原地 upsert
  // （同 recordId/commandId），使「命令记录里选中哪条就看哪条」的结果区实时回填该记录行。
  const handleFrame = (frame: DeviceFramePayload): void => {
    const le = liveExecsRef.current.get(frame.task_id);
    if (!le) return;
    const rows = le.rows.map((r) =>
      r.deviceSn === frame.device_sn ? applyFrameToRow(r, frame, le.columns, le.meta.read) : r,
    );
    const updated: LiveExec = { ...le, rows };
    setLiveExecs((prev) => {
      if (!prev.has(frame.task_id)) return prev;
      const next = new Map(prev);
      next.set(frame.task_id, updated);
      return next;
    });
    // 把实时帧回填到对应命令记录行（仍为 running，收口前不落最终结果）。用 update 不抢占
    // 用户当前选中记录的焦点——并发任务的帧只刷新各自记录行，选中哪条看哪条。
    update({
      id: le.recordId,
      status: 'running',
      commandId: le.taskId,
      time: le.startTime,
      commandName: le.meta.commandName ?? t('mml.consoleV2.rawPathCommand', { op: localizedOpLabel(le.meta.operationType) }),
      operationType: le.meta.operationType,
      deviceCount: le.deviceCount,
      execMeta: le.meta,
      columns: le.columns,
      rows,
    });
  };
  const handleCompleted = (frame: TaskCompletedPayload): void => {
    // 收口走 /results（逐 PATH 合并），不取被逐帧覆盖的 SSE 行；按帧 task_id 各自收口。
    void finalizeFromResults(frame.task_id);
  };
  const liveTaskIds = useMemo(() => new Set(liveExecs.keys()), [liveExecs]);
  useExecStream(liveTaskIds, { onFrame: handleFrame, onCompleted: handleCompleted });

  // 轮询兜底（健壮性）：SSE 帧可能因同用户多会话被踢/网络抖动/重连而丢失，导致结果
  // 长期停在「执行中」。运行期周期性查后端任务状态，到达终态时直接拉 /results 重建
  // 结果行并落入命令记录——使结果呈现不依赖 SSE 实时帧（与 SSE 完成路径等价）。
  // #217：遍历所有在途 taskId 各查一次，并发任务各自独立收口。
  const hasLive = liveExecs.size > 0;
  useEffect(() => {
    if (!hasLive) return;
    let cancelled = false;
    const tick = async (): Promise<void> => {
      const taskIds = Array.from(liveExecsRef.current.keys());
      await Promise.all(
        taskIds.map(async (taskId) => {
          if (finalizedRef.current.has(taskId)) return;
          try {
            const task = await mmlApi.getTaskById(taskId);
            if (cancelled || !task || !TERMINAL_TASK_STATUS.has(task.status)) return;
            // 与 SSE 完成路径共用收口（/results → buildDeviceRows），由 finalizedRef 去重。
            const le = liveExecsRef.current.get(taskId);
            await finalizeFromResults(taskId, le ? buildTerminalFallbackRows(le, task, t) : undefined);
          } catch {
            /* 忽略，下个 tick 再试 */
          }
        }),
      );
    };
    const timer = setInterval(() => void tick(), 4000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
    // finalizeFromResults 仅读 ref + 稳定 setter，无需进依赖（进依赖会每渲染重启轮询）。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasLive]);

  // 配置摘要(顶部条③显示)。未手动配置时不显示(走默认全部)。
  const configSummary = useMemo(() => {
    if (!configTouched || !config) return undefined;
    if (config.mode === 'standard') return t('mml.consoleV2.configSummary.standard', { count: config.checkedPaths.length });
    const n = config.rows.filter((r) => r.path.trim()).length;
    return t('mml.consoleV2.configSummary.raw', { count: n, op: config.operationType });
  }, [config, configTouched, t]);

  // targetSns 默认全部所选设备；「重新执行」时传 [单个设备 SN] 仅对该设备重跑同一命令。
  const runExecute = async (req: ExecRequest, targetSns: string[] = selectedSns): Promise<void> => {
    if (targetSns.length === 0) return;
    const deviceCount = targetSns.length;
    // 任务名称 = 命令名称 + 设备SN（单设备拼 SN；多设备拼首个 SN + 等N台），便于任务记录区分。
    const snSuffix =
      targetSns.length === 1
        ? `_${targetSns[0]}`
        : t('mml.consoleV2.snSuffixMulti', { sn: targetSns[0], count: targetSns.length });
    const taskNameWithSn = (base: string): string => `${base}${snSuffix}`;

    let columns: ResultColumn[];
    let meta: ExecMeta;
    let taskId: string;
    // #196：MOD 下发值 path→value（收口时关联回读 LST）；非 MOD 保持 undefined。
    let setValues: Record<string, string> | undefined;

    try {
      setDispatching(true);
      if (req.mode === 'standard') {
        if (!command) {
          setDispatching(false);
          return;
        }
        columns = isReadOp(command.operationType)
          ? buildStandardQueryColumns(command, req.checkedPaths, req.instanceSelectors)
          : buildColumns(command, req.checkedPaths);
        meta = {
          operationType: command.operationType,
          read: isReadOp(command.operationType),
          label: command.commandCode,
          commandName: command.commandName,
        };
        if (command.operationType === 'MOD') {
          const checkedSet = new Set(req.checkedPaths);
          const picked: Record<string, string> = {};
          command.paramPaths
            .filter((p) => p.writable && checkedSet.has(p.path))
            .forEach((p) => {
              const v = req.values?.[p.path];
              if (v != null && v !== '') picked[p.path] = v;
            });
          setValues = picked;
        }
        if (!command.isCustom && isStructuredOp(command.operationType)) {
          // 逐 PATH：LST/MOD 多 path 时按列序拆成每 path 一条 statement（后端每 statement 一条
          // command→一个 device_task/RPC，path 级成败独立）。命令序与结果列序一致，便于合并。
          const splittable = command.operationType === 'LST' || command.operationType === 'MOD';
          const perPath =
            req.execMode === 'single-path' && splittable && req.checkedPaths.length > 1;
          let statements;
          if (perPath) {
            const orderedPaths = buildPerPathStatementPaths(
              command,
              req.checkedPaths,
              req.instanceSelectors,
            );
            statements = orderedPaths.map((p) =>
              buildStructuredStatement(command, [p], req.values, req.instance, req.instanceSelectors),
            );
          } else {
            statements = [
              buildStructuredStatement(
                command,
                req.checkedPaths,
                req.values,
                req.instance,
                req.instanceSelectors,
              ),
            ];
          }
          const task = await structuredMutation.mutateAsync({
            deviceSns: targetSns,
            statements,
            executeType: 'immediate',
            // task_name = 命令名称 + 设备SN（命令记录仍按命令名展示）。
            taskName: taskNameWithSn(command.commandName),
          });
          taskId = task.id;
        } else {
          // 非结构化操作（DSP/ACT…）与自定义命令（无 command_id）走 legacy 裸路径通道。
          // 按勾选 path + 用户填值下发；task_name 用命令名（req4）。
          const payload = buildRawExecutePayload(
            command.operationType,
            buildStandardRawRows(
              command.operationType,
              req.checkedPaths,
              req.values,
              req.instanceSelectors,
            ),
            targetSns,
            taskNameWithSn(command.commandName),
            'whole',
            command.commandName,
          );
          const task = await rawMutation.mutateAsync({ payload });
          taskId = task.id;
        }
      } else {
        const paths = req.rows.map((r) => r.path.trim()).filter(Boolean);
        if (paths.length === 0) {
          setDispatching(false);
          return;
        }
        columns = buildColumnsFromRawPaths(paths);
        if (req.operationType === 'MOD') {
          const picked: Record<string, string> = {};
          for (const r of req.rows) {
            const p = r.path.trim();
            if (p) picked[p] = r.value ?? '';
          }
          setValues = picked;
        }
        // 命令记录命名：用执行的 path → 设备模型 path 字典里的友好名（缺省回退叶子名）。
        const nameMap = await mmlApi.resolveParamNames(paths);
        const multiPathSuffix = paths.length > 1
          ? t('mml.consoleV2.rawPath.multiSuffix', { count: paths.length })
          : undefined;
        const cmdName = rawCommandName(
          req.operationType,
          paths,
          nameMap,
          localizedOpLabel(req.operationType),
          multiPathSuffix,
        );
        meta = {
          operationType: req.operationType,
          read: isReadOp(req.operationType),
          label: t('mml.consoleV2.rawPathLabel', { op: req.operationType }),
          commandName: cmdName,
        };
        // 用同一名称作为后端 task_name，使「命令记录」与「任务记录」名称对应（req4）。
        const payload = buildRawExecutePayload(
          req.operationType,
          req.rows,
          targetSns,
          taskNameWithSn(cmdName),
          req.execMode,
          cmdName,
        );
        const task = await rawMutation.mutateAsync({ payload });
        taskId = task.id;
      }
    } catch (e) {
      setDispatching(false);
      message.error(e instanceof Error ? e.message : t('mml.consoleV2.msg.dispatchFailed'));
      return;
    }

    // 下发成功：立即插入一条「执行中」命令记录（点击执行即可见，不必等收口），
    // SSE 帧实时回填右侧结果；完成后 finalizeFromResults 按同一 recordId/commandId
    // 原地更新为「已完成」（append 按 commandId upsert，不会重复）。append 会把该记录置为当前
    // （activeId），结果区随之切到这条新下发的命令。
    const recordId = `rec-${recordSeq++}`;
    const startTime = new Date().toLocaleTimeString('zh-CN', { hour12: false });
    const pendingRows = initialPendingRows(targetSns);
    append({
      id: recordId,
      status: 'running',
      commandId: taskId,
      time: startTime,
      commandName: meta.commandName ?? t('mml.consoleV2.rawPathCommand', { op: localizedOpLabel(meta.operationType) }),
      operationType: meta.operationType,
      deviceCount,
      execMeta: meta,
      columns,
      rows: pendingRows,
    });
    // #217：加入在途 Map（不覆盖其它并发任务），SSE 帧按 task_id 路由回填各自记录行。
    setLiveExecs((prev) => {
      const next = new Map(prev);
      next.set(taskId, {
        taskId,
        recordId,
        startTime,
        meta,
        columns,
        rows: pendingRows,
        deviceCount,
        setValues,
      });
      return next;
    });
    // #217：下发完成即解锁执行按钮，允许在本条仍「执行中」时并发发起下一条命令。
    setDispatching(false);
    message.success(t('mml.consoleV2.msg.dispatched', { taskId }));
  };

  // #217：结果区以命令记录选中项（activeRecord）为唯一展示入口——并发多任务在途时，
  // 各 liveExec 的 SSE 帧只把实时结果回填到各自的命令记录行（handleFrame 内 append upsert），
  // 不再由「最近一次 liveExec」抢占结果区，避免多任务下结果跳变；点哪条记录看哪条的结果。
  const dispExecMeta = activeRecord?.execMeta ?? null;
  const dispCommandId = activeRecord?.commandId ?? null;
  const dispColumns = activeRecord?.columns ?? [];
  const dispRows = activeRecord?.rows ?? [];
  // 选中记录是否仍在途（其 taskId 仍在在途 Map）：驱动结果表格 loading 与「重新执行」禁用。
  const activeRunning = !!activeRecord && liveExecs.has(activeRecord.commandId);

  // 「重新执行」（结果列表逐设备）：仅对该设备重跑同一命令。
  // 优先用当前命令+配置（刚执行完，含正确写入值）；回看历史记录（无当前命令）时按展示的
  // 操作类型 + PATH 重建 RAW 执行——读类（LST）适用，写类需重新配置（避免丢失下发值误写）。
  const handleReexecute = (deviceSn: string): void => {
    // 选中记录仍在途时不重发（避免对同一在途任务重复下发）；其它命令在途不影响本条重发。
    if (activeRunning) return;
    if (command || config) {
      const req: ExecRequest =
        config ?? { mode: 'standard', checkedPaths: command?.paramPaths.map((p) => p.path) ?? [] };
      void runExecute(req, [deviceSn]);
      return;
    }
    const op = dispExecMeta?.operationType;
    const paths = dispColumns.map((c) => c.path);
    if (!op || paths.length === 0) {
      message.warning(t('mml.consoleV2.msg.noReexecutable'));
      return;
    }
    if (!dispExecMeta?.read) {
      message.warning(t('mml.consoleV2.msg.writeReconfig'));
      return;
    }
    void runExecute(
      {
        mode: 'raw',
        operationType: op,
        rows: paths.map((p, i) => ({ id: i, path: p, value: '' })),
        execMode: 'whole',
      },
      [deviceSn],
    );
  };

  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <SelectionBar
        deviceCount={selectedSns.length}
        command={command}
        configSummary={configSummary}
        onPickDevices={() => setDeviceModalOpen(true)}
        onPickCommand={() => setCommandModalOpen(true)}
        onConfigParams={() => setConfigModalOpen(true)}
      />

      <div style={{ display: 'flex', gap: 12, height: 'calc(100vh - 220px)', minHeight: 420 }}>
        {historyCollapsed ? (
          <CommandHistoryPanel
            records={records}
            activeId={activeId}
            collapsed
            onSelect={select}
            onClear={clear}
            onToggleCollapsed={() => setHistoryCollapsed(false)}
          />
        ) : (
          <div style={{ flex: '0 0 21.6%', minWidth: 0 }}>
            <CommandHistoryPanel
              records={records}
              activeId={activeId}
              collapsed={false}
              onSelect={select}
              onClear={clear}
              onToggleCollapsed={() => setHistoryCollapsed(true)}
            />
          </div>
        )}

        <div style={{ flex: 1, minWidth: 0 }}>
          <ResultTable
            execMeta={dispExecMeta}
            commandId={dispCommandId}
            columns={dispColumns}
            rows={dispRows}
            running={activeRunning}
            hasExecuted={records.length > 0 || dispatching}
            onReexecute={handleReexecute}
          />
        </div>
      </div>

      <DeviceSelectModal
        open={deviceModalOpen}
        value={selectedSns}
        onCancel={() => setDeviceModalOpen(false)}
        onConfirm={(sns, productId, productClass) => {
          const sameDevices =
            sns.length === selectedSns.length &&
            sns.every((sn, index) => sn === selectedSns[index]);
          const sameProduct = productId === selectedProductId;
          const sameProductClass = productClass === selectedProductClass;
          const contextChanged = !sameDevices || !sameProduct || !sameProductClass;
          setSelectedSns(sns);
          setSelectedProductId(productId);
          setSelectedProductClass(productClass);
          if (contextChanged) {
            setCommand(null);
            setSelectedPathKeys([]);
            setConfig(null);
            setConfigTouched(false);
            setConfigMode('standard');
          }
          setDeviceModalOpen(false);
          if (!command || contextChanged) setCommandModalOpen(true);
        }}
      />

      <CommandSelectModal
        open={commandModalOpen}
        value={command}
        selectedPathKeys={selectedPathKeys}
        deviceSn={selectedSns[0]}
        productClass={selectedProductClass}
        productId={selectedProductId}
        onCancel={() => setCommandModalOpen(false)}
        onConfirm={(cmd, pathKeys) => {
          setCommand(cmd);
          setSelectedPathKeys(pathKeys);
          // 选命令后置默认配置，让「执行」无需先开③。
          setConfig({
            mode: 'standard',
            checkedPaths: commandUsesPathSelection(cmd.operationType)
              ? pathKeys
              : cmd.paramPaths.map((p) => p.path),
          });
          setConfigTouched(false);
          setConfigMode('standard');
          setCommandModalOpen(false);
          // 选完命令自动进入「配置参数」弹框,保持 ①→②→③ 操作连续性
          setConfigModalOpen(true);
        }}
        onGotoRawParams={() => {
          // 「指定参数」快捷入口：跳过命令选择，直接进「配置参数」弹框的「指定参数」标签。
          setConfigMode('raw');
          setCommandModalOpen(false);
          setConfigModalOpen(true);
        }}
      />

      <ConfigParamsModal
        open={configModalOpen}
        command={command}
        selectedPathKeys={selectedPathKeys}
        deviceCount={selectedSns.length}
        initialMode={configMode}
        onGotoCommand={() => {
          // 「选择命令」入口：关闭配置参数弹框，打开命令选择弹框（与「指定参数」形成双向切换）。
          setConfigModalOpen(false);
          setCommandModalOpen(true);
        }}
        onCancel={() => setConfigModalOpen(false)}
        onConfirmAndExecute={(req) => {
          setConfig(req);
          setConfigTouched(true);
          setConfigModalOpen(false);
          void runExecute(req);
        }}
      />
    </Space>
  );
}
