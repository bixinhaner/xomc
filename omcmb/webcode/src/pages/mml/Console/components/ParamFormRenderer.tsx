import { useEffect, useMemo, useState } from 'react';
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
import type { MMLCommand, MMLParam, MMLOperationType } from '@/types/mml';

export type ParamFormPrimitive = string | number | boolean;

export interface ParamFormChangePayload {
  selectedFields?: string[];
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

function resolveOperationType(command: MMLCommand | null): MMLOperationType {
  const operationType = command?.operationType?.trim().toUpperCase();
  if (operationType) {
    return operationType as MMLOperationType;
  }

  const prefix = command?.commandCode?.trim().split(/\s+/)[0]?.toUpperCase();
  return (prefix || 'LST') as MMLOperationType;
}

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
  const [formValues, setFormValues] = useState<Record<string, ParamFormPrimitive>>({});

  useEffect(() => {
    if (!command) {
      setSelectedFields([]);
      setFormValues({});
      onChange?.({});
      return;
    }

    if (QUERY_OPERATIONS.has(operationType)) {
      setSelectedFields([]);
      setFormValues({});
      onChange?.({ selectedFields: [] });
      return;
    }

    const initialValues = buildInitialValues(visibleParams, value);
    setSelectedFields([]);
    setFormValues(initialValues);

    if (ACTION_OPERATIONS.has(operationType) && Object.keys(initialValues).length === 0) {
      onChange?.({});
      return;
    }

    onChange?.({ parameters: initialValues });
  }, [command?.id, operationType]);

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
    onChange?.({ parameters: nextValues });
  };

  const handleFieldsChange = (nextFields: Array<string | number>) => {
    const values = nextFields.map(String);
    setSelectedFields(values);
    onChange?.({ selectedFields: values });
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
            placeholder={required ? '请输入数值' : '选填'}
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
            placeholder={required ? '请选择' : '选填'}
            onChange={(nextValue) => updateParameters(param.name, nextValue)}
            onClear={() => updateParameters(param.name, undefined)}
          />
        );
      default:
        return (
          <Input
            value={formValues[param.name] as string | undefined}
            placeholder={required ? '请输入' : '选填'}
            onChange={(event) => updateParameters(param.name, event.target.value)}
          />
        );
    }
  };

  if (!command) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请先从左侧选择命令" />;
  }

  if (QUERY_OPERATIONS.has(operationType)) {
    if (visibleParams.length === 0) {
      return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="该命令无需配置参数" />;
    }

    return (
      <div>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          勾选需要查询的字段，未勾选时由后续执行逻辑决定默认查询范围。
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
          message="该命令将直接执行，请确认"
          description={command.description || command.helpDoc || '该操作会立即向目标设备下发执行。'}
        />
        <Typography.Text type="secondary">该命令无需配置参数</Typography.Text>
      </Space>
    );
  }

  if (visibleParams.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="该命令无需配置参数" />;
  }

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {ACTION_OPERATIONS.has(operationType) ? (
        <Alert
          type="warning"
          showIcon
          message="该命令将直接执行，请确认"
          description={command.description || command.helpDoc || '请确认操作对象与参数无误后再执行。'}
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
              extra={param.unit ? `单位: ${param.unit}` : undefined}
            >
              {renderControl(param)}
            </Form.Item>
          );
        })}
      </Form>
    </Space>
  );
}
