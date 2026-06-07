import { useEffect, useMemo, useRef, useState } from 'react';
import { message, Space } from 'antd';
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
import { isReadOp, opLabel } from './constants';
import { useConsoleHistory } from './useConsoleHistory';
import { useExecStream } from './useExecStream';
import {
  applyFrameToRow,
  buildColumns,
  buildColumnsFromRawPaths,
  buildDeviceRows,
  buildRawExecutePayload,
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

/** 进行中的一次执行（SSE 帧实时回填 rows，完成后落入命令记录）。 */
interface LiveExec {
  taskId: string;
  meta: ExecMeta;
  columns: ResultColumn[];
  rows: ResultRow[];
  deviceCount: number;
}

/**
 * MML 控制台 V2（设计 docs/design/mml-console-redesign-20260603.md，§3.12 后端对接）。
 *
 * 顶部条 ①设备 ②命令 ③配置参数(弹框) + 执行;主舞台「命令记录(可收缩) ｜ 执行结果」。
 * 执行走真实后端：标准命令 → POST …/execute-statements-structured；裸路径 → POST /mml/execute；
 * 结果经 SSE（mml_device_frame）就地回填表格行，整体完成后落入命令记录。
 */
export default function MMLConsoleV2() {
  const [deviceModalOpen, setDeviceModalOpen] = useState(false);
  const [commandModalOpen, setCommandModalOpen] = useState(false);
  const [configModalOpen, setConfigModalOpen] = useState(false);

  const [selectedSns, setSelectedSns] = useState<string[]>([]);
  // 所选产品 ID（设备弹框强制同一产品）：用于「选择命令 / 配置参数」按产品拉不支持 path 过滤。
  const [selectedProductId, setSelectedProductId] = useState<string>('');
  const [command, setCommand] = useState<CommandItem | null>(null);
  const [config, setConfig] = useState<ExecRequest | null>(null);
  const [configTouched, setConfigTouched] = useState(false);
  // 配置参数弹框打开时激活的标签：命令参数(standard) / 指定参数(raw)。
  const [configMode, setConfigMode] = useState<OperationMode>('standard');

  // 命令记录数据层（P3 接后端；当前凭内存 + localStorage 命令 ID，详见 useConsoleHistory）。
  const { records, activeId, activeRecord, select, append, clear } = useConsoleHistory();
  const [running, setRunning] = useState(false);
  const [historyCollapsed, setHistoryCollapsed] = useState(true); // 默认收缩(§3.10.4)

  // 进行中的执行（实时结果）。用 ref 让 SSE 完成回调读到最新值。
  const [liveExec, setLiveExec] = useState<LiveExec | null>(null);
  const liveExecRef = useRef<LiveExec | null>(null);
  // 已收口的 taskId（防 SSE 完成与轮询兜底双路径重复落记录）。
  const finalizedRef = useRef<Set<string>>(new Set());
  useEffect(() => {
    liveExecRef.current = liveExec;
  }, [liveExec]);

  // 统一收口：拉 /results → buildDeviceRows（逐 PATH 合并）→ 落命令记录。
  // SSE 完成帧（handleCompleted）与轮询兜底都走这里，确保结果不取自被逐帧覆盖的 SSE 行。
  const finalizeFromResults = async (taskId: string): Promise<void> => {
    if (finalizedRef.current.has(taskId)) return;
    const le = liveExecRef.current;
    if (!le || le.taskId !== taskId) return;
    let rows: ResultRow[] | null = null;
    try {
      const pageSize = Math.max(le.deviceCount * Math.max(le.columns.length, 1), 50);
      const resp = await mmlApi.getTaskResults(taskId, 1, pageSize);
      if (resp.items.length > 0) rows = buildDeviceRows(resp.items, le.columns, le.meta.read);
    } catch {
      /* 拉取失败：交给轮询下个 tick 重试 */
    }
    // 拉不到结果不收口，交给轮询下个 tick 重试，避免落入被 SSE 逐帧覆盖的空行。
    if (!rows) return;
    if (finalizedRef.current.has(taskId)) return;
    finalizedRef.current.add(taskId);
    append({
      id: `rec-${recordSeq++}`,
      commandId: le.taskId,
      time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      commandName: le.meta.commandName ?? `裸路径 ${opLabel(le.meta.operationType)}`,
      operationType: le.meta.operationType,
      deviceCount: le.deviceCount,
      execMeta: le.meta,
      columns: le.columns,
      rows,
    });
    setRunning(false);
    setLiveExec(null);
  };

  const structuredMutation = useExecuteStatementsStructured();
  const rawMutation = useExecuteMMLCommand();

  // ── SSE 实时回填（设计 §3.12.2）──────────────────────────────────────────────
  const handleFrame = (frame: DeviceFramePayload): void => {
    setLiveExec((prev) =>
      prev
        ? {
            ...prev,
            rows: prev.rows.map((r) =>
              r.deviceSn === frame.device_sn
                ? applyFrameToRow(r, frame, prev.columns, prev.meta.read)
                : r,
            ),
          }
        : prev,
    );
  };
  const handleCompleted = (_frame: TaskCompletedPayload): void => {
    const le = liveExecRef.current;
    setRunning(false);
    if (!le) return;
    // 收口走 /results（逐 PATH 合并），不取被逐帧覆盖的 SSE 行。
    void finalizeFromResults(le.taskId);
  };
  useExecStream(liveExec?.taskId ?? null, { onFrame: handleFrame, onCompleted: handleCompleted });

  // 轮询兜底（健壮性）：SSE 帧可能因同用户多会话被踢/网络抖动/重连而丢失，导致结果
  // 长期停在「执行中」。运行期周期性查后端任务状态，到达终态时直接拉 /results 重建
  // 结果行并落入命令记录——使结果呈现不依赖 SSE 实时帧（与 SSE 完成路径等价）。
  useEffect(() => {
    const taskId = liveExec?.taskId;
    if (!taskId) return;
    let cancelled = false;
    const tick = async (): Promise<void> => {
      try {
        const task = await mmlApi.getTaskById(taskId);
        if (cancelled || !task || !TERMINAL_TASK_STATUS.has(task.status)) return;
        // 与 SSE 完成路径共用收口（/results → buildDeviceRows），由 finalizedRef 去重。
        await finalizeFromResults(taskId);
      } catch {
        /* 忽略，下个 tick 再试 */
      }
    };
    const timer = setInterval(() => void tick(), 4000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
    // finalizeFromResults 仅读 ref + 稳定 setter，无需进依赖（进依赖会每渲染重启轮询）。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [liveExec?.taskId, append]);

  // 配置摘要(顶部条③显示)。未手动配置时不显示(走默认全部)。
  const configSummary = useMemo(() => {
    if (!configTouched || !config) return undefined;
    if (config.mode === 'standard') return `命令参数 · ${config.checkedPaths.length} 路径`;
    const n = config.rows.filter((r) => r.path.trim()).length;
    return `指定参数 · ${n} PATH · ${config.operationType}`;
  }, [config, configTouched]);

  const canExecute = useMemo(() => {
    if (running || selectedSns.length === 0) return false;
    if (config?.mode === 'raw') return config.rows.some((r) => r.path.trim() !== '');
    return !!command; // standard(含默认配置)需有命令
  }, [running, selectedSns.length, config, command]);

  // targetSns 默认全部所选设备；「重新执行」时传 [单个设备 SN] 仅对该设备重跑同一命令。
  const runExecute = async (req: ExecRequest, targetSns: string[] = selectedSns): Promise<void> => {
    if (targetSns.length === 0) return;
    const deviceCount = targetSns.length;
    // 任务名称 = 命令名称 + 设备SN（单设备拼 SN；多设备拼首个 SN + 等N台），便于任务记录区分。
    const snSuffix =
      targetSns.length === 1
        ? `_${targetSns[0]}`
        : `_${targetSns[0]}等${targetSns.length}台`;
    const taskNameWithSn = (base: string): string => `${base}${snSuffix}`;

    let columns: ResultColumn[];
    let meta: ExecMeta;
    let taskId: string;

    try {
      setRunning(true);
      if (req.mode === 'standard') {
        if (!command) {
          setRunning(false);
          return;
        }
        columns = buildColumns(command, req.checkedPaths);
        meta = {
          operationType: command.operationType,
          read: isReadOp(command.operationType),
          label: command.commandCode,
          commandName: command.commandName,
        };
        if (isStructuredOp(command.operationType)) {
          // 逐 PATH：LST/MOD 多 path 时按列序拆成每 path 一条 statement（后端每 statement 一条
          // command→一个 device_task/RPC，path 级成败独立）。命令序与结果列序一致，便于合并。
          const splittable = command.operationType === 'LST' || command.operationType === 'MOD';
          const perPath =
            req.execMode === 'single-path' && splittable && req.checkedPaths.length > 1;
          let statements;
          if (perPath) {
            const checkedSet = new Set(req.checkedPaths);
            const orderedPaths = command.paramPaths
              .filter((p) => checkedSet.has(p.path))
              .map((p) => p.path);
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
          // 非结构化操作（DSP/ACT…）走 legacy 裸路径通道。task_name 用命令名（req4）。
          const payload = buildRawExecutePayload(
            command.operationType,
            req.checkedPaths.map((p) => ({ path: p, value: req.values?.[p] ?? '' })),
            targetSns,
            taskNameWithSn(command.commandName),
          );
          const task = await rawMutation.mutateAsync({ payload });
          taskId = task.id;
        }
      } else {
        const paths = req.rows.map((r) => r.path.trim()).filter(Boolean);
        if (paths.length === 0) {
          setRunning(false);
          return;
        }
        columns = buildColumnsFromRawPaths(paths);
        // 命令记录命名：用执行的 path → 设备模型 path 字典里的友好名（缺省回退叶子名）。
        const nameMap = await mmlApi.resolveParamNames(paths);
        const cmdName = rawCommandName(req.operationType, paths, nameMap);
        meta = {
          operationType: req.operationType,
          read: isReadOp(req.operationType),
          label: `${req.operationType} (裸路径)`,
          commandName: cmdName,
        };
        // 用同一名称作为后端 task_name，使「命令记录」与「任务记录」名称对应（req4）。
        const payload = buildRawExecutePayload(
          req.operationType,
          req.rows,
          targetSns,
          taskNameWithSn(cmdName),
          req.execMode,
        );
        const task = await rawMutation.mutateAsync({ payload });
        taskId = task.id;
      }
    } catch (e) {
      setRunning(false);
      message.error(e instanceof Error ? e.message : '执行下发失败');
      return;
    }

    // 下发成功：建进行中记录，等 SSE 帧回填。
    setLiveExec({
      taskId,
      meta,
      columns,
      rows: initialPendingRows(targetSns),
      deviceCount,
    });
    message.success(`已下发执行（任务 ${taskId}）`);
  };

  const handleExecute = (): void => {
    // 顶部「执行」按钮:用已保存配置;未手动配置时用默认(标准模式全部路径)
    const req: ExecRequest =
      config ?? { mode: 'standard', checkedPaths: command?.paramPaths.map((p) => p.path) ?? [] };
    void runExecute(req);
  };

  // 展示数据：进行中显示 liveExec，否则显示命令记录选中项。
  const dispExecMeta = liveExec ? liveExec.meta : (activeRecord?.execMeta ?? null);
  const dispCommandId = liveExec ? liveExec.taskId : (activeRecord?.commandId ?? null);
  const dispColumns = liveExec ? liveExec.columns : (activeRecord?.columns ?? []);
  const dispRows = liveExec ? liveExec.rows : (activeRecord?.rows ?? []);

  // 「重新执行」（结果列表逐设备）：仅对该设备重跑同一命令。
  // 优先用当前命令+配置（刚执行完，含正确写入值）；回看历史记录（无当前命令）时按展示的
  // 操作类型 + PATH 重建 RAW 执行——读类（LST）适用，写类需重新配置（避免丢失下发值误写）。
  const handleReexecute = (deviceSn: string): void => {
    if (running) return;
    if (command || config) {
      const req: ExecRequest =
        config ?? { mode: 'standard', checkedPaths: command?.paramPaths.map((p) => p.path) ?? [] };
      void runExecute(req, [deviceSn]);
      return;
    }
    const op = dispExecMeta?.operationType;
    const paths = dispColumns.map((c) => c.path);
    if (!op || paths.length === 0) {
      message.warning('无可重新执行的命令');
      return;
    }
    if (!dispExecMeta?.read) {
      message.warning('写类命令请重新选择命令并配置参数后执行');
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
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <SelectionBar
        deviceCount={selectedSns.length}
        command={command}
        configSummary={configSummary}
        running={running}
        canExecute={canExecute}
        onPickDevices={() => setDeviceModalOpen(true)}
        onPickCommand={() => setCommandModalOpen(true)}
        onConfigParams={() => setConfigModalOpen(true)}
        onExecute={handleExecute}
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
          <div style={{ flex: '0 0 24%', minWidth: 0 }}>
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
            running={running}
            hasExecuted={records.length > 0 || running || !!liveExec}
            onReexecute={handleReexecute}
          />
        </div>
      </div>

      <DeviceSelectModal
        open={deviceModalOpen}
        value={selectedSns}
        onCancel={() => setDeviceModalOpen(false)}
        onConfirm={(sns, productId) => {
          setSelectedSns(sns);
          setSelectedProductId(productId);
          setDeviceModalOpen(false);
          if (!command) setCommandModalOpen(true);
        }}
      />

      <CommandSelectModal
        open={commandModalOpen}
        value={command}
        deviceSn={selectedSns[0]}
        productId={selectedProductId}
        onCancel={() => setCommandModalOpen(false)}
        onConfirm={(cmd) => {
          setCommand(cmd);
          // 选命令后置默认配置(标准模式全部路径),让「执行」无需先开③
          setConfig({ mode: 'standard', checkedPaths: cmd.paramPaths.map((p) => p.path) });
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
        deviceCount={selectedSns.length}
        initialMode={configMode}
        onGotoCommand={() => {
          // 「选择命令」入口：关闭配置参数弹框，打开命令选择弹框（与「指定参数」形成双向切换）。
          setConfigModalOpen(false);
          setCommandModalOpen(true);
        }}
        onCancel={() => setConfigModalOpen(false)}
        onConfirm={(req) => {
          setConfig(req);
          setConfigTouched(true);
          setConfigModalOpen(false);
        }}
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
