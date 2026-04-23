import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Checkbox,
  Empty,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Tooltip,
  Typography,
} from 'antd';
import { QuestionCircleOutlined } from '@ant-design/icons';
import type { MMLCommand, MMLParam, MMLOperationType, MMLParamRef } from '@core/types/mml';
import { resolveOperationType } from '../utils/resolveOperationType';
import { useT } from '@/hooks/useT';

export type ParamFormPrimitive = string | number | boolean;

export interface ParamFormChangePayload {
  selectedFields?: string[];
  selectedParams?: string[];
  parameters?: Record<string, ParamFormPrimitive>;
}

interface ParamFormRendererProps {
  command: MMLCommand | null;
  value?: Record<string, ParamFormPrimitive>;
  onChange?: (payload: ParamFormChangePayload) => void;
}

// TR-069 RPC 方法与操作类型对应：
//   QUERY   → GetParameterValues（仅需路径；勾选要 GET 的参数）
//   EDIT    → SetParameterValues（每个参数都要带值；MOD 选中并编辑）
//   CREATE  → AddObject（ADD，通常也需要一组初始值）
//   REMOVE  → DeleteObject（RMV/DEL，仅需对象路径）
//   ACTION  → Reboot / FactoryReset / 以及通过 SetParameterValues 实现的启停开关
// 每类对应控制面板的不同渲染策略。
const QUERY_OPERATIONS = new Set<MMLOperationType>(['LST', 'DSP']);
const EDIT_OPERATIONS = new Set<MMLOperationType>(['MOD', 'ADD']);
const REMOVE_OPERATIONS = new Set<MMLOperationType>(['RMV']);
const ACTION_OPERATIONS = new Set<MMLOperationType>(['ACT', 'DEA', 'RST', 'CLR']);

function sortParams(params: MMLParam[]): MMLParam[] {
  return [...params].sort((a, b) => {
    const orderA = a.order ?? Number.MAX_SAFE_INTEGER;
    const orderB = b.order ?? Number.MAX_SAFE_INTEGER;

    if (orderA !== orderB) {
      return orderA - orderB;
    }

    return a.name.localeCompare(b.name);
  });
}

function buildInitialValues(params: MMLParam[], value?: Record<string, ParamFormPrimitive>) {
  const nextValues: Record<string, ParamFormPrimitive> = {};

  params.forEach((param) => {
    if (value && value[param.name] !== undefined) {
      nextValues[param.name] = value[param.name];
      return;
    }

    if (param.defaultValue !== undefined) {
      nextValues[param.name] = param.defaultValue;
    }
  });

  return nextValues;
}

export default function ParamFormRenderer({
  command,
  value,
  onChange,
}: ParamFormRendererProps) {
  const t = useT();
  const operationType = useMemo(() => resolveOperationType(command), [command]);
  const sortedParams = useMemo(() => sortParams(command?.params ?? []), [command]);
  const visibleParams = useMemo(() => {
    if (QUERY_OPERATIONS.has(operationType)) {
      return sortedParams;
    }

    if (EDIT_OPERATIONS.has(operationType)) {
      return sortedParams;
    }

    if (REMOVE_OPERATIONS.has(operationType)) {
      return sortedParams.filter((param) => param.required);
    }

    if (ACTION_OPERATIONS.has(operationType)) {
      return sortedParams;
    }

    return sortedParams;
  }, [operationType, sortedParams]);

  const [selectedFields, setSelectedFields] = useState<string[]>([]);
  const [selectedParams, setSelectedParams] = useState<string[]>([]);
  const [formValues, setFormValues] = useState<Record<string, ParamFormPrimitive>>({});

  // Stabilize onChange/value with refs to avoid stale closure in useEffect
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const valueRef = useRef(value);
  valueRef.current = value;

  useEffect(() => {
    if (!command) {
      setSelectedFields([]);
      setSelectedParams([]);
      setFormValues({});
      onChangeRef.current?.({});
      return;
    }

    if (command.paramRefs && command.paramRefs.length > 0) {
      // 对可写操作，默认只勾选"可写"的参数，便于用户编辑值；
      // 查询操作则勾选全部，方便一次性 GET。
      const writable = command.paramRefs.filter((p) => p.isWritable).map((p) => p.paramCode);
      const all = command.paramRefs.map((p) => p.paramCode);
      const nextSelected = EDIT_OPERATIONS.has(operationType) && writable.length > 0 ? writable : all;
      setSelectedFields([]);
      setSelectedParams(nextSelected);
      setFormValues({});
      onChangeRef.current?.({ selectedParams: nextSelected, parameters: {} });
      return;
    }

    if (QUERY_OPERATIONS.has(operationType)) {
      setSelectedFields([]);
      setSelectedParams([]);
      setFormValues({});
      onChangeRef.current?.({ selectedFields: [] });
      return;
    }

    const initialValues = buildInitialValues(visibleParams, valueRef.current);
    setSelectedFields([]);
    setSelectedParams([]);
    setFormValues(initialValues);

    if (ACTION_OPERATIONS.has(operationType) && Object.keys(initialValues).length === 0) {
      onChangeRef.current?.({});
      return;
    }

    onChangeRef.current?.({ parameters: initialValues });
  }, [command?.id, operationType, visibleParams]);

  useEffect(() => {
    if (!value || QUERY_OPERATIONS.has(operationType)) {
      return;
    }

    setFormValues(value);
  }, [operationType, value]);

  const updateParameters = (name: string, nextValue: ParamFormPrimitive | undefined) => {
    const nextValues = { ...formValues };

    if (nextValue === undefined || nextValue === '') {
      delete nextValues[name];
    } else {
      nextValues[name] = nextValue;
    }

    setFormValues(nextValues);
    onChangeRef.current?.({ parameters: nextValues, selectedParams });
  };

  const handleFieldsChange = (nextFields: Array<string | number>) => {
    const values = nextFields.map(String);
    setSelectedFields(values);
    onChangeRef.current?.({ selectedFields: values });
  };

  const handleParamsChange = (nextFields: Array<string | number>) => {
    const values = nextFields.map(String);
    setSelectedParams(values);
    onChangeRef.current?.({ selectedParams: values, parameters: formValues });
  };

  const renderControl = (param: MMLParam) => {
    const required = operationType === 'ADD' ? true : param.required;
    const enumOptions = param.options?.length
      ? param.options
      : (param.enumValues ?? []).map((item) => ({ label: item, value: item }));

    switch (param.type) {
      case 'number':
      case 'unsignedInt':
        return (
          <InputNumber
            style={{ width: '100%' }}
            min={param.minValue ?? (param.type === 'unsignedInt' ? 0 : undefined)}
            max={param.maxValue}
            step={1}
            value={typeof formValues[param.name] === 'number' ? (formValues[param.name] as number) : undefined}
            placeholder={required ? t('mml.console.inputNumber') : t('mml.console.optional')}
            onChange={(nextValue) => updateParameters(param.name, nextValue ?? undefined)}
          />
        );
      case 'boolean':
        return (
          <Switch
            checked={Boolean(formValues[param.name])}
            onChange={(checked) => updateParameters(param.name, checked)}
          />
        );
      case 'enum':
        return (
          <Select
            allowClear={!required}
            value={formValues[param.name] as string | number | undefined}
            options={enumOptions}
            placeholder={required ? t('common.pleaseSelect') : t('mml.console.optional')}
            onChange={(nextValue) => updateParameters(param.name, nextValue)}
            onClear={() => updateParameters(param.name, undefined)}
          />
        );
      default:
        return (
          <Input
            value={formValues[param.name] as string | undefined}
            placeholder={required ? t('common.pleaseInput') : t('mml.console.optional')}
            onChange={(event) => updateParameters(param.name, event.target.value)}
          />
        );
    }
  };

  // 单个 paramRef 按 valueType 渲染输入控件；checked 决定是否纳入命令串。
  // 未勾选时输入仍允许编辑（便于用户先填值再勾选），但不加入最终命令字符串。
  const renderParamRefControl = (ref: MMLParamRef) => {
    const valueType = ref.valueType;
    const isLocked = !ref.isWritable;
    const placeholder = isLocked ? t('mml.console.readOnlyParam') : t('mml.console.inputValue');
    const currentVal = formValues[ref.paramCode];

    if (valueType === 'boolean') {
      return (
        <Switch
          disabled={isLocked}
          checked={Boolean(currentVal)}
          onChange={(checked) => updateParameters(ref.paramCode, checked)}
        />
      );
    }

    if (valueType === 'number') {
      return (
        <InputNumber
          style={{ width: '100%' }}
          disabled={isLocked}
          value={typeof currentVal === 'number' ? currentVal : undefined}
          placeholder={placeholder}
          onChange={(nextValue) => updateParameters(ref.paramCode, nextValue ?? undefined)}
        />
      );
    }

    if (valueType === 'enum') {
      const constraintOptions = Array.isArray(ref.valueConstraint?.options)
        ? (ref.valueConstraint.options as Array<{ label: string; value: string | number }>)
        : Array.isArray(ref.valueConstraint?.values)
        ? (ref.valueConstraint.values as Array<string | number>).map((v) => ({ label: String(v), value: v }))
        : [];
      return (
        <Select
          disabled={isLocked}
          allowClear
          value={currentVal as string | number | undefined}
          options={constraintOptions}
          placeholder={placeholder}
          onChange={(nextValue) => updateParameters(ref.paramCode, nextValue)}
          onClear={() => updateParameters(ref.paramCode, undefined)}
          style={{ width: '100%' }}
        />
      );
    }

    return (
      <Input
        disabled={isLocked}
        value={typeof currentVal === 'string' ? currentVal : currentVal != null ? String(currentVal) : ''}
        placeholder={placeholder}
        onChange={(event) => updateParameters(ref.paramCode, event.target.value)}
      />
    );
  };

  if (!command) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.console.selectCommandFirst')} />;
  }

  // --- 基于 paramRefs 的渲染路径：按操作类型走不同策略 ---------------------
  if (command.paramRefs && command.paramRefs.length > 0) {
    const refs = command.paramRefs;

    // 纯查询：勾选即可，不需要值输入。
    if (QUERY_OPERATIONS.has(operationType)) {
      return (
        <div>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('mml.console.queryFieldsHint')}
          </Typography.Text>
          <Checkbox.Group
            value={selectedParams}
            onChange={handleParamsChange}
            style={{ width: '100%', marginTop: 12 }}
          >
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              {refs.map((p) => (
                <Checkbox key={p.paramCode} value={p.paramCode}>
                  <span style={{ fontWeight: 500 }}>{p.paramNameZh}</span>
                  <span style={{ color: 'rgba(0,0,0,0.35)', fontFamily: 'monospace', fontSize: 11, marginLeft: 6 }}>
                    {p.paramCode}
                  </span>
                  {p.tr069Path ? (
                    <span style={{ color: 'rgba(0,0,0,0.25)', fontSize: 10, marginLeft: 4 }}>
                      ({p.tr069Path})
                    </span>
                  ) : null}
                </Checkbox>
              ))}
            </Space>
          </Checkbox.Group>
        </div>
      );
    }

    // 编辑类（MOD / ADD）：每个参数 checkbox + 输入框；TR-069 SetParameterValues 语义。
    if (EDIT_OPERATIONS.has(operationType)) {
      const selectedSet = new Set(selectedParams);
      return (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Alert
            type="info"
            showIcon
            message={t('mml.console.editParamsHint')}
            description={t('mml.console.editParamsDesc')}
          />
          <div>
            {refs.map((p) => {
              const checked = selectedSet.has(p.paramCode);
              return (
                <div
                  key={p.paramCode}
                  style={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: 8,
                    padding: '8px 0',
                    borderBottom: '1px dashed rgba(0,0,0,0.06)',
                  }}
                >
                  <Checkbox
                    checked={checked}
                    disabled={!p.isWritable}
                    onChange={(e) => {
                      const next = e.target.checked
                        ? Array.from(new Set([...selectedParams, p.paramCode]))
                        : selectedParams.filter((c) => c !== p.paramCode);
                      handleParamsChange(next);
                    }}
                    style={{ marginTop: 6, flexShrink: 0 }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ marginBottom: 4 }}>
                      <span style={{ fontWeight: 500 }}>{p.paramNameZh}</span>
                      <span
                        style={{
                          color: 'rgba(0,0,0,0.35)',
                          fontFamily: 'monospace',
                          fontSize: 11,
                          marginLeft: 6,
                        }}
                      >
                        {p.paramCode}
                      </span>
                      {!p.isWritable && (
                        <Tooltip title={t('mml.console.readOnlyParamHint')}>
                          <Typography.Text type="warning" style={{ marginLeft: 6, fontSize: 11 }}>
                            {t('mml.console.readOnly')}
                          </Typography.Text>
                        </Tooltip>
                      )}
                      {p.tr069Path && (
                        <div
                          style={{
                            color: 'rgba(0,0,0,0.25)',
                            fontSize: 10,
                            fontFamily: 'monospace',
                            marginTop: 2,
                          }}
                        >
                          {p.tr069Path}
                        </div>
                      )}
                    </div>
                    {renderParamRefControl(p)}
                  </div>
                </div>
              );
            })}
          </div>
        </Space>
      );
    }

    // 删除 / 执行类：只展示警示 + 勾选关键参数。不提供值编辑（TR-069 DeleteObject /
    // Reboot / FactoryReset 均不接受自由参数值）。
    if (REMOVE_OPERATIONS.has(operationType)) {
      return (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Alert
            type="warning"
            showIcon
            message={t('mml.console.removeParamsHint')}
            description={command.description || t('mml.console.confirmBeforeExecute')}
          />
          <Checkbox.Group
            value={selectedParams}
            onChange={handleParamsChange}
            style={{ width: '100%' }}
          >
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              {refs.map((p) => (
                <Checkbox key={p.paramCode} value={p.paramCode}>
                  <span style={{ fontWeight: 500 }}>{p.paramNameZh}</span>
                  <span
                    style={{
                      color: 'rgba(0,0,0,0.35)',
                      fontFamily: 'monospace',
                      fontSize: 11,
                      marginLeft: 6,
                    }}
                  >
                    {p.paramCode}
                  </span>
                </Checkbox>
              ))}
            </Space>
          </Checkbox.Group>
        </Space>
      );
    }

    // ACT / DEA / RST / CLR：直接执行操作，通常不需要选择参数；仅保留兜底勾选框
    // 以防后端把必要参数（如重启模式 rebootMode）放在 paramRefs 中。
    if (ACTION_OPERATIONS.has(operationType)) {
      return (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Alert
            type="warning"
            showIcon
            message={t('mml.console.directExecuteWarning')}
            description={command.description || command.helpDoc || t('mml.console.directExecuteDesc')}
          />
          {refs.length > 0 && (
            <Checkbox.Group
              value={selectedParams}
              onChange={handleParamsChange}
              style={{ width: '100%' }}
            >
              <Space direction="vertical" size={10} style={{ width: '100%' }}>
                {refs.map((p) => (
                  <Checkbox key={p.paramCode} value={p.paramCode}>
                    <span style={{ fontWeight: 500 }}>{p.paramNameZh}</span>
                    <span
                      style={{
                        color: 'rgba(0,0,0,0.35)',
                        fontFamily: 'monospace',
                        fontSize: 11,
                        marginLeft: 6,
                      }}
                    >
                      {p.paramCode}
                    </span>
                  </Checkbox>
                ))}
              </Space>
            </Checkbox.Group>
          )}
        </Space>
      );
    }

    // UPG 或未列明的操作：fallback 走勾选视图。
    return (
      <Checkbox.Group
        value={selectedParams}
        onChange={handleParamsChange}
        style={{ width: '100%' }}
      >
        <Space direction="vertical" size={10} style={{ width: '100%' }}>
          {refs.map((p) => (
            <Checkbox key={p.paramCode} value={p.paramCode}>
              <span style={{ fontWeight: 500 }}>{p.paramNameZh}</span>
              <span style={{ color: 'rgba(0,0,0,0.35)', fontFamily: 'monospace', fontSize: 11, marginLeft: 6 }}>
                {p.paramCode}
              </span>
            </Checkbox>
          ))}
        </Space>
      </Checkbox.Group>
    );
  }

  // --- 基于 params（legacy）的渲染路径 ------------------------------------
  if (QUERY_OPERATIONS.has(operationType)) {
    if (visibleParams.length === 0) {
      return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.console.noParamsNeeded')} />;
    }

    return (
      <div>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('mml.console.queryFieldsHint')}
        </Typography.Text>
        <Checkbox.Group value={selectedFields} onChange={handleFieldsChange} style={{ width: '100%', marginTop: 12 }}>
          <Space direction="vertical" size={10} style={{ width: '100%' }}>
            {visibleParams.map((param) => (
              <Checkbox key={param.name} value={param.name}>
                <span style={{ fontWeight: 500 }}>{param.name}</span>
                {param.description ? (
                  <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}> - {param.description}</span>
                ) : null}
              </Checkbox>
            ))}
          </Space>
        </Checkbox.Group>
      </div>
    );
  }

  if (ACTION_OPERATIONS.has(operationType) && visibleParams.length === 0) {
    return (
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Alert
          type="warning"
          showIcon
          message={t('mml.console.directExecuteWarning')}
          description={command.description || command.helpDoc || t('mml.console.directExecuteDesc')}
        />
        <Typography.Text type="secondary">{t('mml.console.noParamsNeeded')}</Typography.Text>
      </Space>
    );
  }

  if (visibleParams.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.console.noParamsNeeded')} />;
  }

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {ACTION_OPERATIONS.has(operationType) ? (
        <Alert
          type="warning"
          showIcon
          message={t('mml.console.directExecuteWarning')}
          description={command.description || command.helpDoc || t('mml.console.confirmBeforeExecute')}
        />
      ) : null}

      <Form layout="vertical" size="small">
        {visibleParams.map((param) => {
          const required = operationType === 'ADD' ? true : param.required;
          const labelNode = (
            <Space size={4}>
              <span>{param.name}</span>
              {param.helpText ? (
                <Tooltip title={param.helpText}>
                  <QuestionCircleOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />
                </Tooltip>
              ) : null}
            </Space>
          );

          return (
            <Form.Item
              key={param.name}
              label={labelNode}
              required={required}
              tooltip={param.description || undefined}
              extra={param.unit ? t('mml.console.unitLabel', { unit: param.unit }) : undefined}
            >
              {renderControl(param)}
            </Form.Item>
          );
        })}
      </Form>
    </Space>
  );
}
