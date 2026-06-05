import { useMemo, useRef, useState } from 'react';
import { Space } from 'antd';
import SelectionBar from './components/SelectionBar';
import DeviceSelectModal from './components/DeviceSelectModal';
import CommandSelectModal from './components/CommandSelectModal';
import ConfigParamsModal from './components/ConfigParamsModal';
import CommandHistoryPanel from './components/CommandHistoryPanel';
import ResultTable from './components/ResultTable';
import type { CommandItem, ExecMeta, ExecRecord, ExecRequest, ResultColumn } from './types';
import { buildColumns, buildColumnsFromRawPaths, buildResultRows } from './mock';
import { isReadOp, opLabel } from './constants';
import { useConsoleHistory } from './useConsoleHistory';

let recordSeq = 1;

/** 每次批量执行的全局唯一命令 ID（mock 用 crypto.randomUUID；真实由后端 mml_tasks.id 返回）。 */
function makeCommandId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `cmd-${recordSeq}-${Date.now()}`;
}

/**
 * MML 控制台 V2（设计 docs/design/mml-console-redesign-20260603.md，含 §3.10 二轮完善）。
 *
 * 顶部条 ①设备 ②命令 ③配置参数(弹框) + 执行;主舞台「命令记录(可收缩,展开 30%) ｜ 执行结果」
 * 左右分区,结果默认显示最近一次执行,点击记录联动回看。当前为前端 mock 实现。
 */
export default function MMLConsoleV2() {
  const [deviceModalOpen, setDeviceModalOpen] = useState(false);
  const [commandModalOpen, setCommandModalOpen] = useState(false);
  const [configModalOpen, setConfigModalOpen] = useState(false);

  const [selectedSns, setSelectedSns] = useState<string[]>([]);
  const [command, setCommand] = useState<CommandItem | null>(null);
  const [config, setConfig] = useState<ExecRequest | null>(null);
  const [configTouched, setConfigTouched] = useState(false);

  // 命令记录数据层（方案 A：当前 mock 种子+会话追加，后续切 /mml/tasks，详见 useConsoleHistory）。
  const { records, activeId, activeRecord, select, append, clear } = useConsoleHistory();
  const [running, setRunning] = useState(false);
  const [historyCollapsed, setHistoryCollapsed] = useState(true); // 默认收缩(§3.10.4)

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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

  const runExecute = (req: ExecRequest) => {
    if (selectedSns.length === 0) return;

    let nextColumns: ResultColumn[];
    let meta: ExecMeta;
    if (req.mode === 'standard') {
      if (!command) return;
      nextColumns = buildColumns(command, req.checkedPaths);
      meta = {
        operationType: command.operationType,
        read: isReadOp(command.operationType),
        label: command.commandCode,
        commandName: command.commandName,
      };
    } else {
      const paths = req.rows.map((r) => r.path.trim()).filter(Boolean);
      if (paths.length === 0) return;
      nextColumns = buildColumnsFromRawPaths(paths);
      meta = {
        operationType: req.operationType,
        read: isReadOp(req.operationType),
        label: `${req.operationType} (裸路径)`,
      };
    }

    const deviceCount = selectedSns.length;
    setRunning(true);
    // mock 异步:模拟扇出耗时后落一条记录并置为当前(真实接入改 SSE 就地更新)
    timerRef.current = setTimeout(() => {
      const rows = buildResultRows(nextColumns, selectedSns, meta.read, meta.operationType);
      const rec: ExecRecord = {
        id: `rec-${recordSeq++}`,
        commandId: makeCommandId(),
        time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
        commandName: meta.commandName ?? `裸路径 ${opLabel(meta.operationType)}`,
        operationType: meta.operationType,
        deviceCount,
        execMeta: meta,
        columns: nextColumns,
        rows,
      };
      append(rec);
      setRunning(false);
    }, 900);
  };

  const handleExecute = () => {
    // 顶部「执行」按钮:用已保存配置;未手动配置时用默认(标准模式全部路径)
    const req: ExecRequest =
      config ?? { mode: 'standard', checkedPaths: command?.paramPaths.map((p) => p.path) ?? [] };
    runExecute(req);
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
            execMeta={activeRecord?.execMeta ?? null}
            commandId={activeRecord?.commandId ?? null}
            columns={activeRecord?.columns ?? []}
            rows={activeRecord?.rows ?? []}
            running={running}
            hasExecuted={records.length > 0 || running}
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
          setCommandModalOpen(false);
          // 选完命令自动进入「配置参数」弹框,保持 ①→②→③ 操作连续性
          setConfigModalOpen(true);
        }}
      />

      <ConfigParamsModal
        open={configModalOpen}
        command={command}
        deviceCount={selectedSns.length}
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
          runExecute(req);
        }}
      />
    </Space>
  );
}
