import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, Typography, Alert, Space } from 'antd';
import type { ParameterType } from '@/types/deviceParameter';
import { useUpdateParameters } from '@/hooks/api/useDeviceParameters';

const { Text } = Typography;

interface ParameterEditModalProps {
  open: boolean;
  deviceId: string;
  parameterPath: string;
  currentValue: string;
  parameterType: ParameterType;
  onClose: () => void;
}

const BOOLEAN_OPTIONS = [
  { label: 'true', value: 'true' },
  { label: 'false', value: 'false' },
];

export default function ParameterEditModal({
  open,
  deviceId,
  parameterPath,
  currentValue,
  parameterType,
  onClose,
}: ParameterEditModalProps) {
  const [newValue, setNewValue] = useState(currentValue);
  const updateMutation = useUpdateParameters();

  useEffect(() => {
    if (open) {
      setNewValue(currentValue);
    }
  }, [open, currentValue]);

  const handleOk = () => {
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
          onChange={setNewValue}
          options={BOOLEAN_OPTIONS}
          style={{ width: '100%' }}
        />
      );
    }
    return (
      <Input
        value={newValue}
        onChange={(e) => setNewValue(e.target.value)}
        placeholder="请输入新值"
      />
    );
  };

  return (
    <Modal
      title="修改参数值"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={updateMutation.isPending}
      okText="确认下发"
      cancelText="取消"
      destroyOnClose
    >
      <Space direction="vertical" style={{ width: '100%' }} size={16}>
        <Alert
          type="warning"
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

          <Form.Item label="参数类型">
            <Text>{parameterType}</Text>
          </Form.Item>

          <Form.Item label="当前值">
            <Text type="secondary">{currentValue || '(空)'}</Text>
          </Form.Item>

          <Form.Item label="新值" required>
            {renderValueInput()}
          </Form.Item>
        </Form>
      </Space>
    </Modal>
  );
}
