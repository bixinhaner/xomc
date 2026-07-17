import { useEffect, useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { App, Alert, Button, Descriptions, Form, Input, Radio, Space, Tag, Typography } from 'antd';
import { KeyOutlined, ReloadOutlined, SendOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { useParameterSchema, useSearchParameters, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import { createPasswordComplexityRule, createPasswordLengthRule } from '@core/utils/passwordValidator';
import { isDeviceTaskTerminal, type DeviceTaskStatus } from '@core/types/deviceTask';

const { Text } = Typography;

const LMT_PASSWORD_STANDARD_PATH = 'Device.Services.lmt.userConfig.1.pass';
const LEGACY_LMT_PASSWORD_PATH = 'Device.DeviceInfo.X_COM_Localweb_password';

type OperationType = 'update' | 'reset';

interface PasswordManagementTabProps {
  deviceId: string;
}

interface PasswordFormValues {
  operation: OperationType;
  newPassword?: string;
  confirmPassword?: string;
}

function taskStatusColor(status: DeviceTaskStatus | undefined): string {
  switch (status) {
    case 'completed':
      return 'success';
    case 'failed':
    case 'expired':
    case 'cancelled':
      return 'error';
    case 'sent':
      return 'processing';
    case 'pending':
      return 'warning';
    default:
      return 'default';
  }
}

function taskStatusLabel(status: DeviceTaskStatus | undefined, t: (key: string) => string): string {
  if (!status) return t('device.password.waiting');
  return t(`device.taskStatus.${status}`);
}

export default function PasswordManagementTab({ deviceId }: PasswordManagementTabProps) {
  const t = useT();
  const [form] = Form.useForm<PasswordFormValues>();
  const { message, modal, notification } = App.useApp();
  const queryClient = useQueryClient();
  const updateMutation = useUpdateParameters();
  const [lastTaskId, setLastTaskId] = useState<string | undefined>();
  const { data: lastTask } = useDeviceTaskStatus(lastTaskId);

  const searchQuery = useSearchParameters(deviceId, 'password', 80, Boolean(deviceId));
  const lmtSchemaQuery = useParameterSchema(deviceId, 'Device.Services.lmt.userConfig.', Boolean(deviceId));
  const deviceInfoSchemaQuery = useParameterSchema(deviceId, 'Device.DeviceInfo.', Boolean(deviceId));

  const availablePathSet = useMemo(() => {
    const paths = new Set<string>();
    for (const item of searchQuery.data ?? []) {
      paths.add(item.parameterPath);
    }
    for (const schema of [lmtSchemaQuery.data, deviceInfoSchemaQuery.data]) {
      for (const item of schema?.parameters ?? []) {
        paths.add(item.path);
      }
    }
    return paths;
  }, [deviceInfoSchemaQuery.data, lmtSchemaQuery.data, searchQuery.data]);

  const effectivePath = useMemo(() => {
    return [LMT_PASSWORD_STANDARD_PATH, LEGACY_LMT_PASSWORD_PATH]
      .find((path) => availablePathSet.has(path)) ?? LMT_PASSWORD_STANDARD_PATH;
  }, [availablePathSet]);

  const lmtPathDetected = availablePathSet.has(effectivePath);
  const lmtPathHint = lmtPathDetected
    ? t('device.password.lmtDetected')
    : t('device.password.lmtFallback');

  const schemaLoading = searchQuery.isLoading || lmtSchemaQuery.isLoading || deviceInfoSchemaQuery.isLoading;

  const refreshSchema = () => {
    void searchQuery.refetch();
    void lmtSchemaQuery.refetch();
    void deviceInfoSchemaQuery.refetch();
  };

  const selectedOperation = Form.useWatch('operation', form) ?? 'update';
  useEffect(() => {
    if (!lastTask || !isDeviceTaskTerminal(lastTask.status)) return;
    if (lastTask.status === 'completed') {
      deviceParameterApi.invalidateParameterSchemaCache(deviceId);
      void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', deviceId] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', 'search', deviceId] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
      message.success(t('device.password.taskCompleted'));
      form.resetFields(['newPassword', 'confirmPassword']);
      return;
    }
    notification.error({
      message: t('device.password.taskFailed'),
      description: lastTask.errorMessage || t('device.multi.unknownErrorHint'),
      duration: 8,
    });
  }, [deviceId, form, lastTask, message, notification, queryClient, t]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    const password = values.newPassword ?? '';

    const submit = async () => {
      const result = await updateMutation.mutateAsync({
        deviceId,
        parameters: [{
          parameterPath: effectivePath,
          parameterValue: password,
          parameterType: 'string',
        }],
      });
      setLastTaskId(result.taskId);
      message.success(t('device.password.submitted'));
    };

    if (values.operation === 'reset') {
      modal.confirm({
        title: t('device.password.resetConfirmTitle'),
        content: t('device.password.resetConfirmContent', { path: effectivePath }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: submit,
      });
      return;
    }
    await submit();
  };

  const currentAction = selectedOperation === 'reset'
    ? t('device.password.actionReset')
    : t('device.password.actionUpdate');

  return (
    <div style={{ padding: '16px 0 24px', maxWidth: 920 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Alert
          type="info"
          showIcon
          message={t('device.password.noticeTitle')}
          description={t('device.password.noticeDesc')}
        />

        <Form<PasswordFormValues>
          form={form}
          layout="vertical"
          initialValues={{ operation: 'update' }}
          style={{ maxWidth: 560 }}
        >
          <Form.Item name="operation" label={t('device.password.operation')}>
            <Radio.Group optionType="button" buttonStyle="solid">
              <Radio.Button value="update">{t('device.password.actionUpdate')}</Radio.Button>
              <Radio.Button value="reset">{t('device.password.actionReset')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            name="newPassword"
            label={selectedOperation === 'reset' ? t('device.password.defaultPassword') : t('device.password.newPassword')}
            rules={[
              { required: true, message: t('device.password.passwordRequired') },
              createPasswordLengthRule(t('system.security.passwordLengthRequirement', { min: 5, max: 64 }), 5, 64),
              createPasswordComplexityRule(t('system.security.passwordComplexityRequirement')),
            ]}
            extra={t('device.password.passwordHint')}
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>

          <Form.Item
            name="confirmPassword"
            label={t('device.password.confirmPassword')}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('device.password.confirmRequired') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error(t('user.passwordMismatch')));
                },
              }),
            ]}
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>

          <Space>
            <Button
              type="primary"
              icon={<SendOutlined />}
              loading={updateMutation.isPending}
              onClick={() => void handleSubmit()}
            >
              {currentAction}
            </Button>
            <Button
              icon={<ReloadOutlined />}
              loading={schemaLoading}
              onClick={refreshSchema}
            >
              {t('common.refresh')}
            </Button>
          </Space>
        </Form>

        <Descriptions
          size="small"
          column={1}
          bordered
          items={[
            {
              key: 'path',
              label: t('device.password.effectivePath'),
              children: (
                <Space>
                  <Text code>{effectivePath || '-'}</Text>
                  <Tag color={lmtPathDetected ? 'success' : 'warning'}>{lmtPathHint}</Tag>
                </Space>
              ),
            },
            {
              key: 'task',
              label: t('device.password.latestTask'),
              children: lastTaskId ? (
                <Space>
                  <KeyOutlined />
                  <Text code>{lastTaskId}</Text>
                  <Tag color={taskStatusColor(lastTask?.status)}>{taskStatusLabel(lastTask?.status, t)}</Tag>
                </Space>
              ) : '-',
            },
          ]}
        />
      </Space>
    </div>
  );
}
