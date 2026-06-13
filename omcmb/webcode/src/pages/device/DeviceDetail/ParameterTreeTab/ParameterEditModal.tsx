import { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, Typography, Alert, Space, Tag } from 'antd';
import type { ParameterType, ParameterConstraints } from '@core/types/deviceParameter';
import { useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { useT } from '@/hooks/useT';

type TFn = (id: string, values?: Record<string, string | number>) => string;

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
  t: TFn,
  constraints?: ParameterConstraints
): string | null {
  if (!value && parameterType !== 'string') {
    return t('device.paramEdit.errRequired');
  }

  if (parameterType === 'int') {
    const num = Number(value);
    if (!Number.isInteger(num)) return t('device.paramEdit.errInteger');
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return t('device.paramEdit.errMinValue', { min: constraints.minValue });
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return t('device.paramEdit.errMaxValue', { max: constraints.maxValue });
    }
  }

  if (parameterType === 'unsignedInt') {
    const num = Number(value);
    if (!Number.isInteger(num) || num < 0) return t('device.paramEdit.errNonNegative');
    if (constraints?.minValue !== undefined && num < constraints.minValue) {
      return t('device.paramEdit.errMinValue', { min: constraints.minValue });
    }
    if (constraints?.maxValue !== undefined && num > constraints.maxValue) {
      return t('device.paramEdit.errMaxValue', { max: constraints.maxValue });
    }
  }

  if (parameterType === 'string' && constraints) {
    if (constraints.maxLength && value.length > constraints.maxLength) {
      return t('device.paramEdit.errMaxLength', { max: constraints.maxLength });
    }
    if (constraints.minLength && value.length < constraints.minLength) {
      return t('device.paramEdit.errMinLength', { min: constraints.minLength });
    }
    if (constraints.pattern) {
      try {
        const re = new RegExp(constraints.pattern);
        if (!re.test(value)) return t('device.paramEdit.errPattern', { pattern: constraints.pattern });
      } catch {
        // ignore invalid regex
      }
    }
  }

  if (constraints?.enumValues && constraints.enumValues.length > 0) {
    if (!constraints.enumValues.includes(value)) {
      return t('device.paramEdit.errEnum', { values: constraints.enumValues.join(', ') });
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
  const t = useT();
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
    setValidationError(validateValue(value, parameterType, t, constraints));
  };

  const handleOk = () => {
    const error = validateValue(newValue, parameterType, t, constraints);
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
        placeholder={t('device.paramEdit.newValuePlaceholder')}
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
      title={t('device.paramEdit.title')}
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={updateMutation.isPending}
      okText={t('device.paramEdit.confirmDispatch')}
      cancelText={t('common.cancel')}
      okButtonProps={{ disabled: Boolean(validationError) }}
      destroyOnHidden
    >
      <Space orientation="vertical" style={{ width: '100%' }} size={16}>
        {changeApplies === 'RebootRequired' && (
          <Alert
            type="warning"
            showIcon
            message={t('device.paramEdit.rebootWarning')}
          />
        )}

        <Alert
          type="info"
          showIcon
          message={t('device.paramEdit.asyncInfo')}
        />

        <Form layout="vertical">
          <Form.Item label={t('device.paramTree.colPath')}>
            <Text
              code
              copyable
              style={{ wordBreak: 'break-all' }}
            >
              {parameterPath}
            </Text>
          </Form.Item>

          {description && (
            <Form.Item label={t('device.paramEdit.description')}>
              <Text type="secondary">{description}</Text>
            </Form.Item>
          )}

          <Form.Item label={t('device.paramEdit.paramType')}>
            <Space>
              <Text>{parameterType}</Text>
              {changeApplies && (
                <Tag color={changeApplies === 'RebootRequired' ? 'warning' : 'default'}>
                  {changeApplies}
                </Tag>
              )}
            </Space>
          </Form.Item>

          <Form.Item label={t('device.paramEdit.currentValue')}>
            <Text type="secondary">{currentValue || t('device.paramTree.emptyValue')}</Text>
          </Form.Item>

          {defaultValue && (
            <Form.Item label={t('device.paramEdit.defaultValue')}>
              <Text type="secondary">{defaultValue}</Text>
            </Form.Item>
          )}

          {hasConstraints && (
            <Form.Item label={t('device.paramEdit.constraints')}>
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
            label={t('device.paramEdit.newValue')}
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
