import { useEffect, useMemo, useRef, useState } from 'react';
import { message, Space } from 'antd';
import { useExecuteStatementsStructured } from '@core/hooks/api/useMmlConsole';
import { useExecuteMMLCommand } from '@core/hooks/api/useMML';
import SelectionBar from './components/SelectionBar';
import DeviceSelectModal from './components/DeviceSelectModal';
import CommandSelectModal from './components/CommandSelectModal';
import ConfigParamsModal from './components/ConfigParamsModal';
import CommandHistoryPanel from './components/CommandHistoryPanel';
import ResultTable from './components/ResultTable';
import type {
  CommandItem,
  ExecMeta,
  ExecRecord,
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
  buildRawExecutePayload,
  buildStructuredStatement,
  initialPendingRows,
  isStructuredOp,
  type DeviceFramePayload,
  type TaskCompletedPayload,
} from './adapters';

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
  const [command, setCommand] = useState<CommandItem | null>(null);
  const [config, setConfig] = useState<ExecRequest | null>(null);
  const [configTouched, setConfigTouched] = useState(false);
  // 配置参数弹框打开时激活的标签：命令参数(standard) / 指定参数(raw)。
  const [configMode, setConfigMode] = useState<OperationMode>('standard');
  // 配置参数弹框是否由「选择命令」流程跳转而来（决定标题是否展示「选择命令」回跳入口）。
  const [configFromCommand, setConfigFromCommand] = useState(false);

  // 命令记录数据层（P3 接后端；当前凭内存 + localStorage 命令 ID，详见 useConsoleHistory）。
  const { records, activeId, activeRecord, select, append, clear } = useConsoleHistory();
  const [running, setRunning] = useState(false);
  const [historyCollapsed, setHistoryCollapsed] = useState(true); // 默认收缩(§3.10.4)

  // 进行中的执行（实时结果）。用 ref 让 SSE 完成回调读到最新值。
  const [liveExec, setLiveExec] = useState<LiveExec | null>(null);
  const liveExecRef = useRef<LiveExec | null>(null);
  useEffect(() => {
    liveExecRef.current = liveExec;
  }, [liveExec]);

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
    // 落入命令记录（commandId = 真实 mml_tasks.id）。
    const rec: ExecRecord = {
      id: `rec-${recordSeq++}`,
      commandId: le.taskId,
      time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      commandName: le.meta.commandName ?? `裸路径 ${opLabel(le.meta.operationType)}`,
      operationType: le.meta.operationType,
      deviceCount: le.deviceCount,
      execMeta: le.meta,
      columns: le.columns,
      rows: le.rows,
    };
    append(rec);
    setLiveExec(null); // 切回命令记录视图（同一份数据，无闪烁）
  };
  useExecStream(liveExec?.taskId ?? null, { onFrame: handleFrame, onCompleted: handleCompleted });

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

  const runExecute = async (req: ExecRequest): Promise<void> => {
    if (selectedSns.length === 0) return;
    const deviceCount = selectedSns.length;

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
          const stmt = buildStructuredStatement(command, req.checkedPaths, req.values, req.instance);
          const task = await structuredMutation.mutateAsync({
            deviceSns: selectedSns,
            statements: [stmt],
            executeType: 'immediate',
          });
          taskId = task.id;
        } else {
          // 非结构化操作（DSP/ACT…）走 legacy 裸路径通道。
          const payload = buildRawExecutePayload(
            command.operationType,
            req.checkedPaths.map((p) => ({ path: p, value: req.values?.[p] ?? '' })),
            selectedSns,
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
        meta = {
          operationType: req.operationType,
          read: isReadOp(req.operationType),
          label: `${req.operationType} (裸路径)`,
        };
        const payload = buildRawExecutePayload(req.operationType, req.rows, selectedSns);
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
      rows: initialPendingRows(selectedSns),
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
          />
        </div>
      </div>

      <DeviceSelectModal
        open={deviceModalOpen}
        value={selectedSns}
        onCancel={() => setDeviceModalOpen(false)}
        onConfirm={(sns) => {
          setSelectedSns(sns);
          setDeviceModalOpen(false);
          if (!command) setCommandModalOpen(true);
        }}
      />

      <CommandSelectModal
        open={commandModalOpen}
        value={command}
        onCancel={() => setCommandModalOpen(false)}
        onConfirm={(cmd) => {
          setCommand(cmd);
          // 选命令后置默认配置(标准模式全部路径),让「执行」无需先开③
          setConfig({ mode: 'standard', checkedPaths: cmd.paramPaths.map((p) => p.path) });
          setConfigTouched(false);
          setConfigMode('standard');
          setConfigFromCommand(true); // 来源=选择命令 → 配置参数标题展示回跳入口
          setCommandModalOpen(false);
          // 选完命令自动进入「配置参数」弹框,保持 ①→②→③ 操作连续性
          setConfigModalOpen(true);
        }}
        onGotoRawParams={() => {
          // 「指定参数」快捷入口：跳过命令选择，直接进「配置参数」弹框的「指定参数」标签。
          setConfigMode('raw');
          setConfigFromCommand(false); // 来源=指定参数(裸路径)，无命令可回跳
          setCommandModalOpen(false);
          setConfigModalOpen(true);
        }}
      />

      <ConfigParamsModal
        open={configModalOpen}
        command={command}
        deviceCount={selectedSns.length}
        initialMode={configMode}
        onGotoCommand={
          configFromCommand
            ? () => {
                // 回跳「选择命令」：关闭配置参数弹框，重开命令选择弹框（保留已选命令）。
                setConfigModalOpen(false);
                setCommandModalOpen(true);
              }
            : undefined
        }
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
