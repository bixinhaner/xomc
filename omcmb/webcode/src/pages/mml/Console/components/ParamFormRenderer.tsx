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
      const allCodes = command.paramRefs.map((p) => p.paramCode);
      setSelectedFields([]);
      setSelectedParams(allCodes);
      setFormValues({});
      onChangeRef.current?.({ selectedParams: allCodes });
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
    onChangeRef.current?.({ parameters: nextValues });
  };

  const handleFieldsChange = (nextFields: Array<string | number>) => {
    const values = nextFields.map(String);
    setSelectedFields(values);
    onChangeRef.current?.({ selectedFields: values });
  };

  const handleParamsChange = (nextFields: Array<string | number>) => {
    const values = nextFields.map(String);
    setSelectedParams(values);
    onChangeRef.current?.({ selectedParams: values });
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

  if (!command) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.console.selectCommandFirst')} />;
  }

  if (command.paramRefs && command.paramRefs.length > 0) {
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
            {command.paramRefs.map((p: MMLParamRef) => (
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
