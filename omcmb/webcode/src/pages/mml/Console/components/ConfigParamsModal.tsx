import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Divider,
  Empty,
  Input,
  InputNumber,
  Modal,
  Radio,
  Select,
  Space,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import { PlayCircleOutlined, RightOutlined } from '@ant-design/icons';
import { InputAddon } from '@/components/common/InputAddon';
import type {
  CommandItem,
  CommandParamPath,
  ExecMode,
  ExecRequest,
  OperationMode,
  RawPathPayload,
} from '../types';
import { CONFIG_TAB_HEIGHT, isReadOp, opColor, opLabelI18nKey } from '../constants';
import RawPathPanel from './RawPathPanel';
import { newRawPathRow } from '../rawPathRow';
import { validateRawPath } from '../rawPathValidate';
import {
  buildDefaultInstanceSelectors,
  computeInstanceSlots,
  resolveObjectPath,
} from '../adapters';
import {
  commandUsesPathSelection,
  getOrderedSelectedCommandPaths,
} from '../pathSelection';
import { getModParamValidationErrors } from '../modParamValidation';
import { useT } from '@/hooks/useT';
import { usePermission } from '@core/hooks/usePermission';
import { dataTypeRangeKind } from '@core/types/paramModel';

const { Text } = Typography;

/** 写类操作各自的提醒文案（§需求 3）。读类 LST/DSP 无提醒。 */
const WRITE_REMINDER_KEYS: Record<string, string> = {
  ADD: 'mml.consoleV2.config.writeReminder.addRmv',
  RMV: 'mml.consoleV2.config.writeReminder.addRmv',
  MOD: 'mml.consoleV2.config.writeReminder.mod',
};

const BOOLEAN_OPTIONS = [
  { label: 'true', value: 'true' },
  { label: 'false', value: 'false' },
];

const PARAM_CONTROL_STYLE = {
  width: '100%',
  marginTop: 6,
};

const INPUT_CONTROL_STYLE = {
  ...PARAM_CONTROL_STYLE,
  display: 'block',
};

function isBooleanValueType(valueType?: string): boolean {
  const normalized = valueType?.trim().toLowerCase();
  return normalized === 'boolean' || normalized === 'bool';
}

function normalizeBooleanValue(value: unknown): 'true' | 'false' | undefined {
  const normalized = String(value ?? '').trim().toLowerCase();
  if (normalized === 'true' || normalized === '1') return 'true';
  if (normalized === 'false' || normalized === '0') return 'false';
  return undefined;
}

function paramModelKey(path: CommandParamPath): string {
  return [
    path.path,
    path.writable,
    path.isObject,
    path.minValue ?? '',
    path.maxValue ?? '',
    path.valueType ?? '',
    path.defaultValue ?? '',
    path.validationPattern ?? '',
    JSON.stringify(path.enumOptions ?? []),
  ].join('\u001f');
}

interface ConfigParamsModalProps {
  open: boolean;
  command: CommandItem | null;
  selectedPathKeys: string[];
  deviceCount: number;
  /** 打开时初始激活的标签：'standard'(命令参数) / 'raw'(指定参数)。默认 'standard'。 */
  initialMode?: OperationMode;
  /**
   * 跳回「选择命令」。仅当本弹框由「选择命令」流程跳转而来时由父组件注入，
   * 注入则在标题处展示「选择命令」入口，便于回去改命令；其它来源（指定参数 / 直接配置）不注入。
   */
  onGotoCommand?: () => void;
  onCancel: () => void;
  /** 执行:保存配置 + 直接下发（原「确定并执行」，#470 后唯一执行入口） */
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
  selectedPathKeys,
  deviceCount,
  initialMode,
  onGotoCommand,
  onCancel,
  onConfirmAndExecute,
}: ConfigParamsModalProps) {
  const t = useT();
  // issue #409：弹框内「确定并执行」= 下发，受 execute 权限管控（无权限禁用 + 提示）。
  const canExecutePerm = usePermission('mml:console:execute');
  const [mode, setMode] = useState<OperationMode>('standard');
  const [wasOpen, setWasOpen] = useState(false);

  const read = isReadOp(command?.operationType);
  const usesPathSelection = commandUsesPathSelection(command?.operationType);
  const selectedParamPaths = useMemo(
    () => getOrderedSelectedCommandPaths(command?.paramPaths ?? [], selectedPathKeys),
    [command, selectedPathKeys],
  );
  const standardParamPaths = useMemo(
    () => (usesPathSelection ? selectedParamPaths : (command?.paramPaths ?? [])),
    [command, selectedParamPaths, usesPathSelection],
  );
  const scopedCommand = useMemo(
    () => (command && usesPathSelection ? { ...command, paramPaths: standardParamPaths } : command),
    [command, standardParamPaths, usesPathSelection],
  );
  const [checkedPaths, setCheckedPaths] = useState<string[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [showModValidationErrors, setShowModValidationErrors] = useState(false);
  const [instance, setInstance] = useState<number>(1);
  // 父级 `.{i}.` 实例选择器（key=i01/i02…），默认每个 1。
  const [instanceSelectors, setInstanceSelectors] = useState<Record<string, string>>({});
  const [lastConfigKey, setLastConfigKey] = useState('');

  // 需要用户填写的实例占位符槽位（ADD/RMV 看 targetObject，LST/MOD 看 paramPaths）。
  const instanceSlots = useMemo(
    () => (scopedCommand ? computeInstanceSlots(scopedCommand) : []),
    [scopedCommand],
  );

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

  // 命令或其确认的 PATH 变更时重置标准模式参数（渲染阶段调整，避开 set-state-in-effect）。
  const configKey = command
    ? `${command.id}:${command.operationType}:${standardParamPaths.map(paramModelKey).join('\u0000')}`
    : '';
  if (configKey !== lastConfigKey) {
    setLastConfigKey(configKey);
    setShowModValidationErrors(false);
    if (!command) {
      setCheckedPaths([]);
      setValues({});
      setInstanceSelectors({});
    } else {
      if (isReadOp(command.operationType)) {
        setCheckedPaths(standardParamPaths.map((p) => p.path));
      } else {
        setCheckedPaths(standardParamPaths.filter((p) => p.writable).map((p) => p.path));
      }
      // MOD/ADD 标量参数「值」默认取 standard_params.min_value（值默认 min_value）。
      const initVals: Record<string, string> = {};
      if (!isReadOp(command.operationType)) {
        standardParamPaths.forEach((p) => {
          if (!p.writable) return;
          if (isBooleanValueType(p.valueType)) {
            initVals[p.path] = normalizeBooleanValue(p.defaultValue)
              ?? normalizeBooleanValue(p.minValue)
              ?? 'false';
          } else if (p.defaultValue != null && p.defaultValue !== '') {
            initVals[p.path] = p.defaultValue;
          } else if (p.minValue != null && dataTypeRangeKind(p.valueType) !== 'length') {
            initVals[p.path] = String(p.minValue);
          }
        });
      }
      setValues(initVals);
      const slots = computeInstanceSlots(scopedCommand ?? command);
      setInstanceSelectors(buildDefaultInstanceSelectors(slots, command.operationType));
    }
    setInstance(1);
  }

  const writablePaths = standardParamPaths.filter((p) => p.writable);
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
  const writeValidationErrors = useMemo(
    () => command?.operationType === 'MOD'
      || command?.operationType === 'ADD'
      ? getModParamValidationErrors(writablePaths, values)
      : {},
    [command?.operationType, values, writablePaths],
  );

  // 写类操作各自的提醒文案（取代原「写操作将对所有已选设备生效」通用提示，§需求 3）。
  const writeReminderKey = currentOp ? WRITE_REMINDER_KEYS[currentOp] : undefined;
  const writeReminder = writeReminderKey ? t(writeReminderKey) : undefined;

  // 标准模式可执行性判定（§需求 3）：
  //   - ADD/RMV 以「目标对象路径」(target_object)下发 RPC，不依赖参数 PATH → 有 target_object 即可执行；
  //   - LST/MOD 依赖勾选/可写参数 PATH → checkedPaths 为空时置灰不可执行（保持原行为）。
  const isAddRmvCmd = command?.operationType === 'ADD' || command?.operationType === 'RMV';
  const standardValid =
    !!command &&
    (isAddRmvCmd
      ? !!(command.targetObject && command.targetObject.trim())
      : checkedPaths.length > 0);
  const valid = mode === 'standard' ? standardValid : rawHasPath && rawAllValid;

  const checkedPathSet = new Set(checkedPaths);
  const selectedValues = Object.fromEntries(
    Object.entries(values).filter(([path, value]) => (
      checkedPathSet.has(path) && value.trim() !== ''
    )),
  );

  const buildRequest = (): ExecRequest =>
    mode === 'standard'
      ? {
          mode: 'standard',
          checkedPaths,
          execMode: effectiveExecMode,
          // 父级 `.{i}.` 实例选择器：查询类允许留空，写类默认 1。
          ...(instanceSlots.length > 0 ? { instanceSelectors } : {}),
          // MOD/ADD 携带写入值；RMV 携带实例号。LST 两者均不消费。
          ...(command && !isReadOp(command.operationType) && command.operationType !== 'RMV'
            ? { values: selectedValues }
            : {}),
          ...(command?.operationType === 'RMV' ? { instance } : {}),
        }
      : {
          mode: 'raw',
          operationType: rawPayload.operationType,
          rows: rawPayload.rows,
          execMode: effectiveExecMode,
        };

  const handleConfirmAndExecute = () => {
    if ((command?.operationType === 'MOD' || command?.operationType === 'ADD')
      && Object.keys(writeValidationErrors).length > 0) {
      setShowModValidationErrors(true);
      return;
    }
    onConfirmAndExecute(buildRequest());
  };

  const handleModeChange = (nextMode: OperationMode): void => {
    if (nextMode === 'standard' && !command && onGotoCommand) {
      onGotoCommand();
      return;
    }
    setMode(nextMode);
  };

  const standardBody = !command ? (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      // 未选命令时，「请先选择 MML 命令」做成跳转链接，点击直接去「选择命令」（§需求 2）。
      description={
        deviceCount === 0 ? (
          t('mml.consoleV2.config.pickDeviceFirst')
        ) : onGotoCommand ? (
          <Button type="link" onClick={onGotoCommand}>
            {t('mml.consoleV2.config.pickCommandFirst')}
            <RightOutlined style={{ fontSize: 11 }} />
          </Button>
        ) : (
          t('mml.consoleV2.config.pickCommandFirst')
        )
      }
      style={{ marginTop: 32 }}
    />
  ) : (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <div>
        {/* 统一头部（所有操作类型同一布局，参考 LST）：操作类型 + 命令名称 + 操作提示。 */}
        <Space size={8} wrap style={{ width: '100%' }}>
          <Tag color={opColor(command.operationType)} style={{ marginInlineEnd: 0 }}>
            {command.operationType} · {t(opLabelI18nKey(command.operationType) ?? '')}
          </Tag>
          <Text strong>
            {command.commandName}
            <Text type="secondary" style={{ fontWeight: 400 }}>
              {' '}
              ({checkedPaths.length})
            </Text>
          </Text>
          {!read && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {command.operationType === 'RMV'
                ? t('mml.consoleV2.config.hintRmv')
                : t('mml.consoleV2.config.hintWrite')}
            </Text>
          )}
        </Space>

        {/* §需求 2：ADD/RMV 无参数 PATH，展示执行 RPC 的「目标对象路径」（{i} 由下方实例号实时替换）。 */}
        {isAddRmvCmd && command.targetObject && (
          <div style={{ marginTop: 10 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('mml.consoleV2.config.targetObjectHint', {
                op: command.operationType,
                rpc: command.operationType === 'ADD' ? 'AddObject' : 'DeleteObject',
                slots: instanceSlots.length > 0
                  ? t('mml.consoleV2.config.slotReplaced')
                  : t('mml.consoleV2.config.noSlot'),
              })}
            </Text>
            <div style={{ marginTop: 4 }}>
              <Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>
                {resolveObjectPath(command.targetObject, instanceSelectors)}
              </Text>
            </div>
          </div>
        )}

        {/* 父级 `.{i}.` 实例选择器：查询前层默认 1、最后一层留空；写类默认 1。 */}
        {instanceSlots.length > 0 && (
          <div style={{ marginTop: 10 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t(
                read
                  ? 'mml.consoleV2.config.objectInstanceRead'
                  : 'mml.consoleV2.config.objectInstance',
              )}
            </Text>
            <Space wrap style={{ marginTop: 6 }}>
              {instanceSlots.map((s) => {
                const rawValue = instanceSelectors[s.key] ?? (read ? '' : '1');
                return (
                  <span key={s.key}>
                    <Text type="secondary" style={{ fontSize: 12, marginRight: 4 }}>
                      {s.label}
                    </Text>
                    <InputNumber
                      min={1}
                      precision={0}
                      size="small"
                      style={{ width: 96 }}
                      value={rawValue === '' ? null : Number(rawValue)}
                      onChange={(v) =>
                        setInstanceSelectors((prev) => ({
                          ...prev,
                          [s.key]: String(v ?? (read ? '' : 1)),
                        }))
                      }
                    />
                  </span>
                );
              })}
            </Space>
          </div>
        )}

        {/* 主体：读类确认 PATH；RMV 填实例号；MOD/ADD 逐 PATH 填值。列表统一由标签页容器滚动。 */}
        {read ? (
          <Space orientation="vertical" size={6} style={{ width: '100%', marginTop: 8 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('mml.consoleV2.config.confirmSelectedPaths')}
            </Text>
            {selectedParamPaths.map((p) => (
              <div key={p.path} style={{ minWidth: 0 }}>
                <Text>{p.label}</Text>{' '}
                <Text
                  type="secondary"
                  code
                  style={{
                    fontSize: 11,
                    whiteSpace: 'normal',
                    overflowWrap: 'anywhere',
                    wordBreak: 'break-word',
                  }}
                >
                  {p.path}
                </Text>
              </div>
            ))}
          </Space>
        ) : command.operationType === 'RMV' ? (
          // antd6 addonBefore 已废弃：改用 Space.Compact + InputAddon 复刻前缀盒子。
          <div style={{ marginTop: 8 }}>
            <Space.Compact style={{ width: 180 }}>
              <InputAddon>{t('mml.consoleV2.config.instanceNo')}</InputAddon>
              <InputNumber
                min={1}
                value={instance}
                onChange={(v) => setInstance(v ?? 1)}
                style={{ width: '100%' }}
              />
            </Space.Compact>
          </div>
        ) : (
          <Space orientation="vertical" size={10} style={{ width: '100%', marginTop: 8 }}>
            {writablePaths.map((p) => {
              const error = (command.operationType === 'MOD' || command.operationType === 'ADD')
                && showModValidationErrors
                ? writeValidationErrors[p.path]
                : undefined;
              const isBoolean = isBooleanValueType(p.valueType);
              const enumOptions = isBoolean ? BOOLEAN_OPTIONS : (p.enumOptions ?? []);

              return (
                <div key={p.path} style={{ minWidth: 0 }}>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'baseline',
                      flexWrap: 'wrap',
                      gap: 6,
                      minWidth: 0,
                    }}
                  >
                    <Text style={{ flexShrink: 0 }}>{p.label}</Text>
                    <Text
                      type="secondary"
                      code
                      style={{
                        flex: '1 1 320px',
                        minWidth: 0,
                        fontSize: 11,
                        whiteSpace: 'normal',
                        overflowWrap: 'anywhere',
                        wordBreak: 'break-word',
                      }}
                    >
                      {p.path}
                    </Text>
                    {command.operationType === 'MOD' && p.valueType && (
                      <Tag style={{ marginInlineStart: 0, flexShrink: 0 }}>{p.valueType}</Tag>
                    )}
                  </div>
                  {enumOptions.length > 0 ? (
                    <Select
                      className="mml-config-param-select mml-config-param-control"
                      aria-label={p.label}
                      status={error ? 'error' : undefined}
                      value={values[p.path] ?? (isBoolean ? 'false' : undefined)}
                      options={enumOptions}
                      onChange={(value) => setValues((prev) => ({ ...prev, [p.path]: value }))}
                      style={PARAM_CONTROL_STYLE}
                      styles={{
                        root: { display: 'inline-flex', width: '100%' },
                        content: {
                          flex: '1 1 auto',
                          minWidth: 0,
                          paddingInlineEnd: 32,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        },
                        suffix: { marginInlineStart: 'auto' },
                      }}
                    />
                  ) : (
                    <Input
                      className="mml-config-param-control"
                      status={error ? 'error' : undefined}
                      aria-invalid={Boolean(error)}
                      placeholder={t('mml.consoleV2.config.inputFieldPlaceholder', { label: p.label })}
                      value={values[p.path] ?? ''}
                      onChange={(e) => setValues((prev) => ({ ...prev, [p.path]: e.target.value }))}
                      style={INPUT_CONTROL_STYLE}
                    />
                  )}
                  {error && (
                    <Text type="danger" style={{ display: 'block', fontSize: 12, marginTop: 4 }}>
                      {t(`mml.consoleV2.config.validation.${error.code}`, { bound: error.bound ?? '' })}
                    </Text>
                  )}
                </div>
              );
            })}
          </Space>
        )}
      </div>
    </Space>
  );

  return (
    <Modal
      title={
        <Space size={12} align="center">
          <span>{t('mml.consoleV2.config.title')}</span>
          {/* 「选择命令 ›」与命令选择弹框的「指定参数 ›」形成双向切换（§需求 1） */}
          {onGotoCommand && (
            <Button type="link" size="small" style={{ padding: 0 }} onClick={onGotoCommand}>
              {t('mml.consoleV2.config.gotoCommand')}
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
          {t('common.cancel')}
        </Button>,
        <Tooltip key="exec" title={canExecutePerm ? undefined : t('common.noPermission')}>
          <Button
            type="primary"
            icon={<PlayCircleOutlined />}
            disabled={!valid || deviceCount === 0 || !canExecutePerm}
            onClick={handleConfirmAndExecute}
          >
            {t('mml.consoleV2.config.confirmAndExecute', { count: deviceCount })}
          </Button>
        </Tooltip>,
      ]}
    >
      {/* 标签页内容区固定高度，超出竖向滚动 → 弹框总高不随命令/标签页变化（§需求 1/2）。 */}
      <Tabs
        activeKey={mode}
        onChange={(k) => handleModeChange(k as OperationMode)}
        items={[
          {
            key: 'standard',
            label: t('mml.consoleV2.config.tabStandard'),
            children: (
              <div
                className="mml-config-tab-scroll"
                style={{ height: CONFIG_TAB_HEIGHT, overflowY: 'auto', overflowX: 'hidden', paddingRight: 4 }}
              >
                {standardBody}
              </div>
            ),
          },
          {
            key: 'raw',
            label: t('mml.consoleV2.config.tabRaw'),
            children: (
              <div
                className="mml-config-tab-scroll"
                style={{ height: CONFIG_TAB_HEIGHT, overflowY: 'auto', overflowX: 'hidden', paddingRight: 4 }}
              >
                <RawPathPanel value={rawPayload} onChange={setRawPayload} suggestions={suggestions} />
              </div>
            ),
          },
        ]}
      />

      <Divider style={{ margin: '4px 0 12px' }} />

      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <div>
          <Text type="secondary">{t('mml.consoleV2.config.execMode')}</Text>
          <div style={{ marginTop: 6 }}>
            <Radio.Group
              value={effectiveExecMode}
              onChange={(e) => setExecMode(e.target.value)}
            >
              <Radio value="whole">{t('mml.consoleV2.config.execWhole')}</Radio>
              <Tooltip
                title={perPathDisabled ? t('mml.consoleV2.config.perPathDisabledTip') : ''}
              >
                <Radio value="single-path" disabled={perPathDisabled}>
                  {t('mml.consoleV2.config.execPerPath')}
                </Radio>
              </Tooltip>
            </Radio.Group>
          </div>
        </div>
        {/* 预留提醒行高度，读类(无提醒)与写类(有提醒)切换时弹框总高不抖动（§需求 1）。 */}
        <div style={{ minHeight: 40 }}>
          {!currentRead && writeReminder && (
            <Alert type="warning" showIcon title={writeReminder} style={{ padding: '6px 12px' }} />
          )}
        </div>
      </Space>
    </Modal>
  );
}
