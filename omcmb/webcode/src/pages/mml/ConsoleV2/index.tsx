import { useRef, useState } from 'react';
import { Col, Row, Space } from 'antd';
import SelectionBar from './components/SelectionBar';
import DeviceSelectModal from './components/DeviceSelectModal';
import CommandSelectModal from './components/CommandSelectModal';
import OperationPanel from './components/OperationPanel';
import ResultTable from './components/ResultTable';
import type { CommandItem, ExecMeta, ExecRequest, ResultColumn, ResultRow } from './types';
import { buildColumns, buildColumnsFromRawPaths, buildResultRows } from './mock';
import { isReadOp } from './constants';

/**
 * MML 控制台 V2（设计 docs/design/mml-console-redesign-20260603.md）。
 *
 * 与 V1（/mml/console）并存、互不影响：V2 以「结果表格」为主舞台，设备/命令选择收纳进
 * 顶部选择条 + 弹框，左操作区 + 右结果表格左右分区（设计 §3.2 主推方案）。左操作区支持
 * 「命令参数（结构化）/ 指定参数（裸路径专家）」双模式（§3.9）。
 *
 * 当前阶段为前端 mock 实现，执行结果由 mock.ts 生成，待后端 results-schema / export
 * 端点就绪后再桥接真实 useMML* hooks（P1/P2）。
 */
export default function MMLConsoleV2() {
  const [deviceModalOpen, setDeviceModalOpen] = useState(false);
  const [commandModalOpen, setCommandModalOpen] = useState(false);

  const [selectedSns, setSelectedSns] = useState<string[]>([]);
  const [command, setCommand] = useState<CommandItem | null>(null);

  const [running, setRunning] = useState(false);
  const [hasExecuted, setHasExecuted] = useState(false);
  const [columns, setColumns] = useState<ResultColumn[]>([]);
  const [rows, setRows] = useState<ResultRow[]>([]);
  const [execMeta, setExecMeta] = useState<ExecMeta | null>(null);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleExecute = (req: ExecRequest) => {
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

    setRunning(true);
    setHasExecuted(true);
    setColumns(nextColumns);
    setExecMeta(meta);
    setRows([]);
    // mock 异步：模拟扇出执行耗时后一次性回填（真实接入时改为 SSE 就地更新每行）
    timerRef.current = setTimeout(() => {
      setRows(buildResultRows(nextColumns, selectedSns, meta.read, meta.operationType));
      setRunning(false);
    }, 900);
  };

  const handleClear = () => {
    if (timerRef.current) clearTimeout(timerRef.current);
    setRunning(false);
    setHasExecuted(false);
    setRows([]);
    setColumns([]);
    setExecMeta(null);
  };

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <SelectionBar
        deviceCount={selectedSns.length}
        command={command}
        onPickDevices={() => setDeviceModalOpen(true)}
        onPickCommand={() => setCommandModalOpen(true)}
      />

      {/* 操作区与结果区高度固定、自适应浏览器视口高度（减去顶部 header/tab 栏 + 选择条），
          内部各自滚动，不把整页撑高。 */}
      <Row gutter={12} align="stretch" style={{ height: 'calc(100vh - 220px)', minHeight: 420 }}>
        <Col xs={24} lg={9} style={{ height: '100%' }}>
          <OperationPanel
            command={command}
            deviceCount={selectedSns.length}
            running={running}
            onExecute={handleExecute}
            onClear={handleClear}
          />
        </Col>
        <Col xs={24} lg={15} style={{ height: '100%' }}>
          <ResultTable
            execMeta={execMeta}
            columns={columns}
            rows={rows}
            running={running}
            hasExecuted={hasExecuted}
          />
        </Col>
      </Row>

      <DeviceSelectModal
        open={deviceModalOpen}
        value={selectedSns}
        onCancel={() => setDeviceModalOpen(false)}
        onConfirm={(sns) => {
          setSelectedSns(sns);
          setDeviceModalOpen(false);
          // 选完设备引导第②步选命令（设计 §3.3 两步流程）
          if (!command) setCommandModalOpen(true);
        }}
      />

      <CommandSelectModal
        open={commandModalOpen}
        value={command}
        onCancel={() => setCommandModalOpen(false)}
        onConfirm={(cmd) => {
          setCommand(cmd);
          setCommandModalOpen(false);
        }}
      />
    </Space>
  );
}
