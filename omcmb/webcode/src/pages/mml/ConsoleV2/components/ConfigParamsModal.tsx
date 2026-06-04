import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Checkbox,
  Divider,
  Empty,
  Input,
  InputNumber,
  Modal,
  Radio,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import type { CommandItem, ExecMode, ExecRequest, OperationMode, RawPathPayload } from '../types';
import { isReadOp, opColor, opLabel } from '../constants';
import RawPathPanel from './RawPathPanel';
import { newRawPathRow } from '../rawPathRow';

const { Text } = Typography;

interface ConfigParamsModalProps {
  open: boolean;
  command: CommandItem | null;
  deviceCount: number;
  onCancel: () => void;
  /** 确定:保存配置(不执行) */
  onConfirm: (req: ExecRequest) => void;
  /** 确定并执行 */
  onConfirmAndExecute: (req: ExecRequest) => void;
}

/**
 * 配置参数弹框（设计 §3.10.3）—— 由顶部条「③ 配置参数」触发，内含「命令参数 / 指定参数」
 * 两 Tab（原左操作区 OperationPanel 主体搬入）。确定回填顶部条摘要；确定并执行直接下发。
 * 不用 destroyOnHidden,关闭后保留编辑态,再次打开沿用(命令变更时由 lastCmdId 重置)。
 */
export default function ConfigParamsModal({
  open,
  command,
  deviceCount,
  onCancel,
  onConfirm,
  onConfirmAndExecute,
}: ConfigParamsModalProps) {
  const [mode, setMode] = useState<OperationMode>('standard');

  const read = isReadOp(command?.operationType);
  const [checkedPaths, setCheckedPaths] = useState<string[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [instance, setInstance] = useState<number>(1);
  const [lastCmdId, setLastCmdId] = useState<string | null>(null);

  const [rawPayload, setRawPayload] = useState<RawPathPayload>({
    operationType: 'LST',
    rows: [newRawPathRow()],
  });

  const [execMode, setExecMode] = useState<ExecMode>('whole');

  // 命令变更时重置标准模式参数(渲染阶段按 command.id 调整,避开 set-state-in-effect)。
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

  const currentOp = mode === 'standard' ? command?.operationType : rawPayload.operationType;
  const currentRead = isReadOp(currentOp);
  const rawHasPath = rawPayload.rows.some((r) => r.path.trim() !== '');

  const valid = mode === 'standard' ? !!command : rawHasPath;

  const buildRequest = (): ExecRequest =>
    mode === 'standard'
      ? { mode: 'standard', checkedPaths }
      : { mode: 'raw', operationType: rawPayload.operationType, rows: rawPayload.rows };

  const standardBody = !command ? (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={deviceCount === 0 ? '请先选择目标设备' : '请先选择 MML 命令'}
      style={{ marginTop: 32 }}
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
              indeterminate={checkedPaths.length > 0 && checkedPaths.length < command.paramPaths.length}
              checked={command.paramPaths.length > 0 && checkedPaths.length === command.paramPaths.length}
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
          <div>
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
          <div>
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
    <Modal
      title="配置参数"
      open={open}
      width={680}
      onCancel={onCancel}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          取消
        </Button>,
        <Button key="ok" disabled={!valid} onClick={() => onConfirm(buildRequest())}>
          确定
        </Button>,
        <Button
          key="exec"
          type="primary"
          icon={<PlayCircleOutlined />}
          disabled={!valid || deviceCount === 0}
          onClick={() => onConfirmAndExecute(buildRequest())}
        >
          确定并执行（{deviceCount} 台）
        </Button>,
      ]}
    >
      <Tabs
        activeKey={mode}
        onChange={(k) => setMode(k as OperationMode)}
        items={[
          { key: 'standard', label: '命令参数', children: standardBody },
          {
            key: 'raw',
            label: '指定参数',
            children: <RawPathPanel value={rawPayload} onChange={setRawPayload} suggestions={suggestions} />,
          },
        ]}
      />

      <Divider style={{ margin: '4px 0 12px' }} />

      <Space direction="vertical" size={12} style={{ width: '100%' }}>
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
      </Space>
    </Modal>
  );
}
