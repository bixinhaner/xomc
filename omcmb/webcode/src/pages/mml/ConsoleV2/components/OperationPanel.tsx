import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Divider,
  Empty,
  Input,
  InputNumber,
  Radio,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { ClearOutlined, PlayCircleOutlined } from '@ant-design/icons';
import type { CommandItem, ExecMode, ExecRequest, OperationMode, RawPathPayload } from '../types';
import { isReadOp, opColor, opLabel } from '../constants';
import RawPathPanel from './RawPathPanel';
import { newRawPathRow } from '../rawPathRow';

const { Text } = Typography;

interface OperationPanelProps {
  command: CommandItem | null;
  deviceCount: number;
  running: boolean;
  onExecute: (req: ExecRequest) => void;
  onClear: () => void;
}

/**
 * 左侧操作 & 参数区（设计 §3.2 ② + §3.9）—— 顶部 Segmented 切换两种模式：
 *  · 标准参数：操作类型随命令派生 + 参数路径勾选(读)/填值(写)/实例号(RMV)
 *  · 参数路径指定：裸路径专家模式（操作类型下拉 + 任意 TR-069 路径行）
 * 执行模式 + 执行/清空按钮两模式共用。当前为 mock，执行请求经 onExecute 上抛驱动右侧结果表格。
 */
export default function OperationPanel({
  command,
  deviceCount,
  running,
  onExecute,
  onClear,
}: OperationPanelProps) {
  const [mode, setMode] = useState<OperationMode>('standard');

  // 标准模式状态
  const read = isReadOp(command?.operationType);
  const [checkedPaths, setCheckedPaths] = useState<string[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [instance, setInstance] = useState<number>(1);
  const [lastCmdId, setLastCmdId] = useState<string | null>(null);

  // 裸路径模式状态
  const [rawPayload, setRawPayload] = useState<RawPathPayload>({
    operationType: 'LST',
    rows: [newRawPathRow()],
  });

  const [execMode, setExecMode] = useState<ExecMode>('whole');

  // 切换命令时重置标准模式参数勾选 / 填值（渲染阶段按 command.id 变化调整，避开 set-state-in-effect）。
  const cmdId = command?.id ?? null;
  if (cmdId !== lastCmdId) {
    setLastCmdId(cmdId);
    if (!command) {
      setCheckedPaths([]);
    } else if (isReadOp(command.operationType)) {
      setCheckedPaths(command.paramPaths.map((p) => p.path));
    } else {
      setCheckedPaths(command.paramPaths.filter((p) => p.writable).map((p) => p.path));
    }
    setValues({});
    setInstance(1);
  }

  const writablePaths = command?.paramPaths.filter((p) => p.writable) ?? [];

  const suggestions = useMemo(
    () =>
      command?.paramPaths.map((p) => ({
        value: p.path,
        label: p.label ? `${p.label} (${p.path})` : p.path,
      })),
    [command],
  );

  // 当前用于页脚展示/告警的操作类型与读写判定
  const currentOp = mode === 'standard' ? command?.operationType : rawPayload.operationType;
  const currentRead = isReadOp(currentOp);

  const rawHasPath = rawPayload.rows.some((r) => r.path.trim() !== '');

  const canExecute = useMemo(() => {
    if (running || deviceCount === 0) return false;
    return mode === 'standard' ? !!command : rawHasPath;
  }, [running, deviceCount, mode, command, rawHasPath]);

  const handleExecute = () => {
    if (mode === 'standard') {
      onExecute({ mode: 'standard', checkedPaths });
    } else {
      onExecute({ mode: 'raw', operationType: rawPayload.operationType, rows: rawPayload.rows });
    }
  };

  const showFooter = mode === 'raw' || !!command;

  // 命令参数（结构化）模式主体
  const standardBody = !command ? (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={deviceCount === 0 ? '请先在顶部选择目标设备' : '请在顶部选择 MML 命令'}
      style={{ marginTop: 48 }}
    />
  ) : (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <div>
        <Text type="secondary">操作类型</Text>
        <div style={{ marginTop: 6 }}>
          <Tag color={opColor(command.operationType)} style={{ fontSize: 13, padding: '2px 10px' }}>
            {command.operationType} · {opLabel(command.operationType)}
          </Tag>
          <Text type="secondary" style={{ marginLeft: 8 }} code>
            {command.commandCode}
          </Text>
        </div>
      </div>

      <div>
        {read ? (
          <div>
            <Checkbox
              indeterminate={
                checkedPaths.length > 0 && checkedPaths.length < command.paramPaths.length
              }
              checked={
                command.paramPaths.length > 0 && checkedPaths.length === command.paramPaths.length
              }
              onChange={(e) =>
                setCheckedPaths(e.target.checked ? command.paramPaths.map((p) => p.path) : [])
              }
            >
              全选
            </Checkbox>
            <Text style={{ display: 'block', marginTop: 8, fontSize: 12 }}>勾选要查询的参数：</Text>
            <Checkbox.Group
              style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8 }}
              value={checkedPaths}
              onChange={(v) => setCheckedPaths(v as string[])}
            >
              {command.paramPaths.map((p) => (
                <Checkbox key={p.path} value={p.path}>
                  <Text>{p.label}</Text>{' '}
                  <Text type="secondary" code style={{ fontSize: 11 }}>
                    {p.path}
                  </Text>
                </Checkbox>
              ))}
            </Checkbox.Group>
          </div>
        ) : command.operationType === 'RMV' ? (
          <div style={{ marginTop: 8 }}>
            <Text style={{ fontSize: 12 }}>指定要删除的实例号（{'{i}'}）：</Text>
            <div style={{ marginTop: 8 }}>
              <InputNumber
                min={1}
                value={instance}
                onChange={(v) => setInstance(v ?? 1)}
                addonBefore="实例号"
                style={{ width: 180 }}
              />
            </div>
          </div>
        ) : (
          <div style={{ marginTop: 8 }}>
            <Text style={{ fontSize: 12 }}>填写要下发的参数值：</Text>
            <Space direction="vertical" size={10} style={{ width: '100%', marginTop: 8 }}>
              {writablePaths.map((p) => (
                <div key={p.path}>
                  <Text style={{ fontSize: 12 }}>
                    {p.label}{' '}
                    <Text type="secondary" code style={{ fontSize: 11 }}>
                      {p.path}
                    </Text>
                  </Text>
                  <Input
                    placeholder={`输入 ${p.label}`}
                    value={values[p.path] ?? ''}
                    onChange={(e) => setValues((prev) => ({ ...prev, [p.path]: e.target.value }))}
                  />
                </div>
              ))}
            </Space>
          </div>
        )}
      </div>
    </Space>
  );

  return (
    <Card
      variant="borderless"
      style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { padding: 16, flex: 1, minHeight: 0, overflow: 'auto' } }}
    >
      <Tabs
        activeKey={mode}
        onChange={(k) => setMode(k as OperationMode)}
        items={[
          { key: 'standard', label: '命令参数', children: standardBody },
          {
            key: 'raw',
            label: '指定参数',
            children: (
              <RawPathPanel value={rawPayload} onChange={setRawPayload} suggestions={suggestions} />
            ),
          },
        ]}
      />

      {showFooter && (
        <Space direction="vertical" size={12} style={{ width: '100%', marginTop: 4 }}>
          <Divider style={{ margin: '4px 0' }} />

          <div>
            <Text type="secondary">执行模式</Text>
            <div style={{ marginTop: 6 }}>
              <Radio.Group value={execMode} onChange={(e) => setExecMode(e.target.value)}>
                <Radio value="whole">整体下发</Radio>
                <Radio value="single-path">逐 PATH</Radio>
              </Radio.Group>
            </div>
          </div>

          {!currentRead && (
            <Alert
              type="warning"
              showIcon
              message="写操作将对所有已选设备生效，请确认参数值无误。"
              style={{ padding: '6px 12px' }}
            />
          )}

          <Space>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              loading={running}
              disabled={!canExecute}
              onClick={handleExecute}
            >
              执行（{deviceCount} 台）
            </Button>
            <Button icon={<ClearOutlined />} onClick={onClear} disabled={running}>
              清空结果
            </Button>
          </Space>
        </Space>
      )}
    </Card>
  );
}
