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
  Tooltip,
  Typography,
} from 'antd';
import { PlayCircleOutlined, RightOutlined } from '@ant-design/icons';
import type { CommandItem, ExecMode, ExecRequest, OperationMode, RawPathPayload } from '../types';
import { CONFIG_TAB_HEIGHT, isReadOp, opColor, opLabel } from '../constants';
import RawPathPanel from './RawPathPanel';
import { newRawPathRow } from '../rawPathRow';
import { validateRawPath } from '../rawPathValidate';
import { computeInstanceSlots } from '../adapters';

const { Text } = Typography;

/** 写类操作各自的提醒文案（§需求 3）。读类 LST/DSP 无提醒。 */
const WRITE_REMINDERS: Record<string, string> = {
  ADD: 'ADD / RMV 协议规定单次仅作用于一个对象路径，已锁定为单行。',
  RMV: 'ADD / RMV 协议规定单次仅作用于一个对象路径，已锁定为单行。',
  MOD: 'MOD 将修改所有已选设备的参数值，请确认参数值无误。',
};

interface ConfigParamsModalProps {
  open: boolean;
  command: CommandItem | null;
  deviceCount: number;
  /** 打开时初始激活的标签：'standard'(命令参数) / 'raw'(指定参数)。默认 'standard'。 */
  initialMode?: OperationMode;
  /**
   * 跳回「选择命令」。仅当本弹框由「选择命令」流程跳转而来时由父组件注入，
   * 注入则在标题处展示「选择命令」入口，便于回去改命令；其它来源（指定参数 / 直接配置）不注入。
   */
  onGotoCommand?: () => void;
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
  initialMode,
  onGotoCommand,
  onCancel,
  onConfirm,
  onConfirmAndExecute,
}: ConfigParamsModalProps) {
  const [mode, setMode] = useState<OperationMode>('standard');
  const [wasOpen, setWasOpen] = useState(false);

  const read = isReadOp(command?.operationType);
  const [checkedPaths, setCheckedPaths] = useState<string[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [instance, setInstance] = useState<number>(1);
  // 父级 `.{i}.` 实例选择器（key=i01/i02…），默认每个 1。
  const [instanceSelectors, setInstanceSelectors] = useState<Record<string, string>>({});
  const [lastCmdId, setLastCmdId] = useState<string | null>(null);

  // 需要用户填写的实例占位符槽位（ADD/RMV 看 targetObject，LST/MOD 看 paramPaths）。
  const instanceSlots = useMemo(() => (command ? computeInstanceSlots(command) : []), [command]);

  // 每次打开时按 initialMode 切换激活标签（渲染阶段调整 state，避开 set-state-in-effect）。
  // 「指定参数」快捷入口打开时 initialMode='raw'，直接落到裸路径标签。
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setMode(initialMode ?? 'standard');
  }

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
      setValues({});
      setInstanceSelectors({});
    } else {
      if (isReadOp(command.operationType)) {
        setCheckedPaths(command.paramPaths.map((p) => p.path));
      } else {
        setCheckedPaths(command.paramPaths.filter((p) => p.writable).map((p) => p.path));
      }
      // MOD/ADD 标量参数「值」默认取 standard_params.min_value（值默认 min_value）。
      const initVals: Record<string, string> = {};
      if (!isReadOp(command.operationType)) {
        command.paramPaths.forEach((p) => {
          if (p.writable && p.minValue != null) initVals[p.path] = String(p.minValue);
        });
      }
      setValues(initVals);
      // 父级 `.{i}.` 实例选择器默认每个 1（实例默认 1）。
      const initSel: Record<string, string> = {};
      computeInstanceSlots(command).forEach((s) => {
        initSel[s.key] = '1';
      });
      setInstanceSelectors(initSel);
    }
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
  // ADD/RMV 天然单对象操作，不支持「逐 PATH」拆分——硬禁用该选项并强制整体下发。
  const perPathDisabled = currentOp === 'ADD' || currentOp === 'RMV';
  const effectiveExecMode: ExecMode = perPathDisabled ? 'whole' : execMode;
  const rawHasPath = rawPayload.rows.some((r) => r.path.trim() !== '');
  // 裸路径每行须通过 ADD/RMV/{i} 校验，否则禁用执行（plan A）。
  const rawAllValid = rawPayload.rows.every(
    (r) => validateRawPath(rawPayload.operationType, r.path) === null,
  );

  // 写类操作各自的提醒文案（取代原「写操作将对所有已选设备生效」通用提示，§需求 3）。
  const writeReminder = currentOp ? WRITE_REMINDERS[currentOp] : undefined;

  const valid = mode === 'standard' ? !!command : rawHasPath && rawAllValid;

  const buildRequest = (): ExecRequest =>
    mode === 'standard'
      ? {
          mode: 'standard',
          checkedPaths,
          execMode: effectiveExecMode,
          // 父级 `.{i}.` 实例选择器（默认 1）。
          ...(instanceSlots.length > 0 ? { instanceSelectors } : {}),
          // MOD/ADD 携带写入值；RMV 携带实例号。LST 两者均不消费。
          ...(command && !isReadOp(command.operationType) && command.operationType !== 'RMV'
            ? { values }
            : {}),
          ...(command?.operationType === 'RMV' ? { instance } : {}),
        }
      : {
          mode: 'raw',
          operationType: rawPayload.operationType,
          rows: rawPayload.rows,
          execMode: effectiveExecMode,
        };

  const standardBody = !command ? (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      // 未选命令时，「请先选择 MML 命令」做成跳转链接，点击直接去「选择命令」（§需求 2）。
      description={
        deviceCount === 0 ? (
          '请先选择目标设备'
        ) : onGotoCommand ? (
          <Button type="link" onClick={onGotoCommand}>
            请先选择 MML 命令
            <RightOutlined style={{ fontSize: 11 }} />
          </Button>
        ) : (
          '请先选择 MML 命令'
        )
      }
      style={{ marginTop: 32 }}
    />
  ) : (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <div>
        {/* 统一头部（所有操作类型同一布局，参考 LST）：全选(仅读类) + 操作类型 + 命令名称 + 操作提示。 */}
        <Space size={8} wrap style={{ width: '100%' }}>
          {read && (
            <Checkbox
              indeterminate={checkedPaths.length > 0 && checkedPaths.length < command.paramPaths.length}
              checked={command.paramPaths.length > 0 && checkedPaths.length === command.paramPaths.length}
              onChange={(e) =>
                setCheckedPaths(e.target.checked ? command.paramPaths.map((p) => p.path) : [])
              }
            >
              全选
            </Checkbox>
          )}
          <Tag color={opColor(command.operationType)} style={{ marginInlineEnd: 0 }}>
            {command.operationType} · {opLabel(command.operationType)}
          </Tag>
          <Text strong>
            {command.commandName}
            <Text type="secondary" style={{ fontWeight: 400 }}>
              {' '}
              ({checkedPaths.length})
            </Text>
          </Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {read
              ? '勾选要查询的参数'
              : command.operationType === 'RMV'
                ? '指定要删除的实例号'
                : '填写要下发的参数值'}
          </Text>
        </Space>

        {/* 父级 `.{i}.` 实例选择器：对象实例默认 1，按路径中占位符个数渲染。 */}
        {instanceSlots.length > 0 && (
          <div style={{ marginTop: 10 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              对象实例（{'{i}'}，默认 1）：
            </Text>
            <Space wrap style={{ marginTop: 6 }}>
              {instanceSlots.map((s) => (
                <span key={s.key}>
                  <Text type="secondary" style={{ fontSize: 12, marginRight: 4 }}>
                    {s.label}
                  </Text>
                  <InputNumber
                    min={1}
                    size="small"
                    style={{ width: 96 }}
                    value={Number(instanceSelectors[s.key] ?? '1')}
                    onChange={(v) =>
                      setInstanceSelectors((prev) => ({ ...prev, [s.key]: String(v ?? 1) }))
                    }
                  />
                </span>
              ))}
            </Space>
          </div>
        )}

        {/* 主体：读类勾选 PATH；RMV 填实例号；MOD/ADD 逐 PATH 填值。列表统一由标签页容器滚动。 */}
        {read ? (
          <Checkbox.Group
            style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8 }}
            value={checkedPaths}
            onChange={(v) => setCheckedPaths(v as string[])}
          >
            {command.paramPaths.map((p) => (
              <Checkbox key={p.path} value={p.path} style={{ whiteSpace: 'nowrap' }}>
                <Text>{p.label}</Text>{' '}
                <Text type="secondary" code style={{ fontSize: 11 }}>
                  {p.path}
                </Text>
              </Checkbox>
            ))}
          </Checkbox.Group>
        ) : command.operationType === 'RMV' ? (
          <div style={{ marginTop: 8 }}>
            <InputNumber
              min={1}
              value={instance}
              onChange={(v) => setInstance(v ?? 1)}
              addonBefore="实例号"
              style={{ width: 180 }}
            />
          </div>
        ) : (
          <Space direction="vertical" size={10} style={{ width: '100%', marginTop: 8 }}>
            {writablePaths.map((p) => (
              <div key={p.path} style={{ whiteSpace: 'nowrap' }}>
                <Text>{p.label}</Text>{' '}
                <Text type="secondary" code style={{ fontSize: 11 }}>
                  {p.path}
                </Text>
                <Input
                  placeholder={`输入 ${p.label}`}
                  value={values[p.path] ?? ''}
                  onChange={(e) => setValues((prev) => ({ ...prev, [p.path]: e.target.value }))}
                />
              </div>
            ))}
          </Space>
        )}
      </div>
    </Space>
  );

  return (
    <Modal
      title={
        <Space size={12} align="center">
          <span>配置参数</span>
          {/* 「选择命令 ›」与命令选择弹框的「指定参数 ›」形成双向切换（§需求 1） */}
          {onGotoCommand && (
            <Button type="link" size="small" style={{ padding: 0 }} onClick={onGotoCommand}>
              选择命令
              <RightOutlined style={{ fontSize: 11 }} />
            </Button>
          )}
        </Space>
      }
      open={open}
      width={860}
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
      {/* 标签页内容区固定高度，超出竖向滚动 → 弹框总高不随命令/标签页变化（§需求 1/2）。 */}
      <Tabs
        activeKey={mode}
        onChange={(k) => setMode(k as OperationMode)}
        items={[
          {
            key: 'standard',
            label: '命令参数',
            children: (
              <div style={{ height: CONFIG_TAB_HEIGHT, overflowY: 'auto', overflowX: 'auto', paddingRight: 4 }}>
                {standardBody}
              </div>
            ),
          },
          {
            key: 'raw',
            label: '指定参数',
            children: (
              <div style={{ height: CONFIG_TAB_HEIGHT, overflowY: 'auto', overflowX: 'auto', paddingRight: 4 }}>
                <RawPathPanel value={rawPayload} onChange={setRawPayload} suggestions={suggestions} />
              </div>
            ),
          },
        ]}
      />

      <Divider style={{ margin: '4px 0 12px' }} />

      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <div>
          <Text type="secondary">执行模式</Text>
          <div style={{ marginTop: 6 }}>
            <Radio.Group
              value={effectiveExecMode}
              onChange={(e) => setExecMode(e.target.value)}
            >
              <Radio value="whole">整体执行</Radio>
              <Tooltip
                title={perPathDisabled ? 'ADD / RMV 为单对象操作，不支持逐 PATH 拆分' : ''}
              >
                <Radio value="single-path" disabled={perPathDisabled}>
                  逐 PATH
                </Radio>
              </Tooltip>
            </Radio.Group>
          </div>
        </div>
        {/* 预留提醒行高度，读类(无提醒)与写类(有提醒)切换时弹框总高不抖动（§需求 1）。 */}
        <div style={{ minHeight: 40 }}>
          {!currentRead && writeReminder && (
            <Alert type="warning" showIcon message={writeReminder} style={{ padding: '6px 12px' }} />
          )}
        </div>
      </Space>
    </Modal>
  );
}
