import { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, Typography, Alert, Space, Tag } from 'antd';
import type { ParameterType, ParameterConstraints } from '@core/types/deviceParameter';
import { useUpdateParameters } from '@core/hooks/api/useDeviceParameters';

const { Text } = Typography;

interface ParameterEditModalProps {
  open: boolean;
  deviceId: string;
  parameterPath: string;
  currentValue: string;
  parameterType: ParameterType;
  constraints?: ParameterConstraints;
  description?: string;
  changeApplies?: string;
  defaultValue?: string;
  onClose: () => void;
}

const BOOLEAN_OPTIONS = [
  { label: 'true', value: 'true' },
  { label: 'false', value: 'false' },
];

function validateValue(
  value: string,
  parameterType: ParameterType,
  constraints?: ParameterConstraints
): string | null {
  if (!value && parameterType !== 'string') {
    return '请输入值';
  }

  if (parameterType === 'int') {
    const num = Number(value);
    if (!Number.isInteger(num)) return '请输入整数';
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return `最小值为 ${constraints.minValue}`;
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return `最大值为 ${constraints.maxValue}`;
    }
  }

  if (parameterType === 'unsignedInt') {
    const num = Number(value);
    if (!Number.isInteger(num) || num < 0) return '请输入非负整数';
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return `最小值为 ${constraints.minValue}`;
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return `最大值为 ${constraints.maxValue}`;
    }
  }

  if (parameterType === 'string' && constraints) {
    if (constraints.maxLength && value.length > constraints.maxLength) {
      return `最大长度为 ${constraints.maxLength}`;
    }
    if (constraints.minLength && value.length < constraints.minLength) {
      return `最小长度为 ${constraints.minLength}`;
    }
    if (constraints.pattern) {
      try {
        const re = new RegExp(constraints.pattern);
        if (!re.test(value)) return `不匹配模式: ${constraints.pattern}`;
      } catch {
        // ignore invalid regex
      }
    }
  }

  if (constraints?.enumValues && constraints.enumValues.length > 0) {
    if (!constraints.enumValues.includes(value)) {
      return `允许的值: ${constraints.enumValues.join(', ')}`;
    }
  }

  return null;
}

export default function ParameterEditModal({
  open,
  deviceId,
  parameterPath,
  currentValue,
  parameterType,
  constraints,
  description,
  changeApplies,
  defaultValue,
  onClose,
}: ParameterEditModalProps) {
  const [newValue, setNewValue] = useState(currentValue);
  const [validationError, setValidationError] = useState<string | null>(null);
  const updateMutation = useUpdateParameters();

  useEffect(() => {
    if (open) {
      setNewValue(currentValue);
      setValidationError(null);
    }
  }, [open, currentValue]);

  const handleValueChange = (value: string) => {
    setNewValue(value);
    setValidationError(validateValue(value, parameterType, constraints));
  };

  const handleOk = () => {
    const error = validateValue(newValue, parameterType, constraints);
    if (error) {
      setValidationError(error);
      return;
    }

    updateMutation.mutate(
      {
        deviceId,
        parameters: [
          {
            parameterPath,
            parameterValue: newValue,
            parameterType,
          },
        ],
      },
      {
        onSuccess: () => {
          onClose();
        },
      }
    );
  };

  const renderValueInput = () => {
    if (parameterType === 'boolean') {
      return (
        <Select
          value={newValue}
          onChange={handleValueChange}
          options={BOOLEAN_OPTIONS}
          style={{ width: '100%' }}
        />
      );
    }

    if (constraints?.enumValues && constraints.enumValues.length > 0) {
      return (
        <Select
          value={newValue}
          onChange={handleValueChange}
          options={constraints.enumValues.map((v, i) => ({ label: constraints.enumLabels?.[i] ?? v, value: v }))}
          style={{ width: '100%' }}
          showSearch
        />
      );
    }

    return (
      <Input
        value={newValue}
        onChange={(e) => handleValueChange(e.target.value)}
        placeholder="请输入新值"
        status={validationError ? 'error' : undefined}
      />
    );
  };

  const hasConstraints = constraints && (
    constraints.minValue !== undefined ||
    constraints.maxValue !== undefined ||
    constraints.maxLength !== undefined ||
    constraints.minLength !== undefined ||
    constraints.pattern ||
    (constraints.enumValues && constraints.enumValues.length > 0)
  );

  return (
    <Modal
      title="修改参数值"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={updateMutation.isPending}
      okText="确认下发"
      cancelText="取消"
      okButtonProps={{ disabled: Boolean(validationError) }}
      destroyOnHidden
    >
      <Space direction="vertical" style={{ width: '100%' }} size={16}>
        {changeApplies === 'RebootRequired' && (
          <Alert
            type="warning"
            showIcon
            message="此参数修改后需要设备重启才能生效。"
          />
        )}

        <Alert
          type="info"
          showIcon
          message="参数修改将异步下发到设备，可能需要等待设备下次 Inform 后生效。"
        />

        <Form layout="vertical">
          <Form.Item label="参数路径">
            <Text
              code
              copyable
              style={{ wordBreak: 'break-all' }}
            >
              {parameterPath}
            </Text>
          </Form.Item>

          {description && (
            <Form.Item label="描述">
              <Text type="secondary">{description}</Text>
            </Form.Item>
          )}

          <Form.Item label="参数类型">
            <Space>
              <Text>{parameterType}</Text>
              {changeApplies && (
                <Tag color={changeApplies === 'RebootRequired' ? 'warning' : 'default'}>
                  {changeApplies}
                </Tag>
              )}
            </Space>
          </Form.Item>

          <Form.Item label="当前值">
            <Text type="secondary">{currentValue || '(空)'}</Text>
          </Form.Item>

          {defaultValue && (
            <Form.Item label="默认值">
              <Text type="secondary">{defaultValue}</Text>
            </Form.Item>
          )}

          {hasConstraints && (
            <Form.Item label="约束">
              <Space wrap>
                {constraints!.minValue !== undefined && (
                  <Tag>min: {constraints!.minValue}</Tag>
                )}
                {constraints!.maxValue !== undefined && (
                  <Tag>max: {constraints!.maxValue}</Tag>
                )}
                {constraints!.maxLength !== undefined && (
                  <Tag>maxLen: {constraints!.maxLength}</Tag>
                )}
                {constraints!.minLength !== undefined && constraints!.minLength > 0 && (
                  <Tag>minLen: {constraints!.minLength}</Tag>
                )}
                {constraints!.pattern && (
                  <Tag>pattern: {constraints!.pattern}</Tag>
                )}
                {constraints!.enumValues && constraints!.enumValues.length > 0 && (
                  <Tag>enum: [{constraints!.enumValues.join(', ')}]</Tag>
                )}
              </Space>
            </Form.Item>
          )}

          <Form.Item
            label="新值"
            required
            validateStatus={validationError ? 'error' : undefined}
            help={validationError}
          >
            {renderValueInput()}
          </Form.Item>
        </Form>
      </Space>
    </Modal>
  );
}
