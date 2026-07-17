import { useEffect, useMemo, useState } from 'react';
import { App, Alert, Button, Form, Input, Radio, Space } from 'antd';
import { ReloadOutlined, SendOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useParameterSchema, useSearchParameters } from '@core/hooks/api/useDeviceParameters';
import { useSecuritySettings } from '@core/hooks/api/useSecuritySettings';
import { deviceTaskApi } from '@core/services/api/deviceTaskApi';

const LMT_PASSWORD_STANDARD_PATH = 'Device.Services.lmt.userConfig.1.pass';
const LEGACY_LMT_USERNAME_PATH = 'Device.DeviceInfo.X_COM_Localweb_username';
const LEGACY_LMT_PASSWORD_PATH = 'Device.DeviceInfo.X_COM_Localweb_password';
const SET_PARAMETER_VALUES_METHOD = 'SetParameterValues';
const BAICELLS_PASSWORD_RESET_METHOD = 'X_BAICELLS_COM_PasswordReset';
const DEFAULT_RESET_PASSWORD_VALUE = 'OMC@123456';

type OperationType = 'update' | 'reset';

interface PasswordManagementTabProps {
  deviceId: string;
  deviceSn: string;
  productClass?: string;
  deviceModel?: string;
  networkType?: string;
  onTaskSubmitted?: (taskId: string) => void;
  onTaskCleared?: () => void;
}

interface PasswordFormValues {
  operation: OperationType;
  username?: string;
  newPassword?: string;
  confirmPassword?: string;
}

interface PasswordFlow {
  operations: OperationType[];
  requiresUsername: boolean;
  usernamePaths: string[];
  passwordPaths: string[];
  noticeTitle: string;
  noticeDesc: string;
}

function resolvePasswordFlow(productClass?: string, deviceModel?: string, networkType?: string): PasswordFlow {
  const productValues = [productClass, deviceModel]
    .map((value) => String(value ?? '').trim().toUpperCase())
    .filter(Boolean);
  const normalizedNetworkType = String(networkType ?? '').trim().toLowerCase();
  const isNrStation = normalizedNetworkType === 'gnb'
    || normalizedNetworkType === 'nr'
    || productValues.some((value) => value.includes('GNB') || value.includes('NR'));
  const hasFamily = (families: string[]) => productValues.some((value) => families.some((family) => {
    if (value === family) return true;
    if (value.startsWith(`${family}-`) || value.startsWith(`${family}_`) || value.startsWith(`${family}/`) || value.startsWith(`${family}.`)) {
      return true;
    }
    return family.length >= 3 && value.startsWith(family);
  }));

  if (isNrStation) {
    return {
      operations: ['update'],
      requiresUsername: true,
      usernamePaths: [LEGACY_LMT_USERNAME_PATH],
      passwordPaths: [LEGACY_LMT_PASSWORD_PATH],
      noticeTitle: 'device.password.nrNoticeTitle',
      noticeDesc: 'device.password.nrNoticeDesc',
    };
  }

  let operations: OperationType[] = ['update'];
  if (hasFamily(['BLQ', 'MLQ'])) {
    operations = ['update', 'reset'];
  } else if (hasFamily(['BSC', 'BTS', 'BM'])) {
    operations = ['reset'];
  } else if (hasFamily(['BLN', 'MLN'])) {
    operations = ['update'];
  }

  return {
    operations,
    requiresUsername: false,
    usernamePaths: [],
    passwordPaths: [LEGACY_LMT_PASSWORD_PATH, LMT_PASSWORD_STANDARD_PATH],
    noticeTitle: 'device.password.noticeTitle',
    noticeDesc: 'device.password.noticeDesc',
  };
}

function pickAvailablePath(candidates: string[], availablePathSet: Set<string>): string {
  return candidates.find((path) => availablePathSet.has(path)) ?? candidates[0] ?? '';
}

export default function PasswordManagementTab({ deviceId, deviceSn, productClass, deviceModel, networkType, onTaskSubmitted, onTaskCleared }: PasswordManagementTabProps) {
  const t = useT();
  const [form] = Form.useForm<PasswordFormValues>();
  const { message, modal, notification } = App.useApp();
  const { settings: securitySettings } = useSecuritySettings();
  const [taskSubmitting, setTaskSubmitting] = useState(false);
  const passwordFlow = useMemo(
    () => resolvePasswordFlow(productClass, deviceModel, networkType),
    [deviceModel, networkType, productClass]
  );
  const supportedOperations = passwordFlow.operations;
  const showOperationPicker = supportedOperations.length > 1;

  const searchQuery = useSearchParameters(deviceId, 'password', 80, Boolean(deviceId));
  const lmtSchemaQuery = useParameterSchema(deviceId, 'Device.Services.lmt.userConfig.', Boolean(deviceId));
  const deviceInfoSchemaQuery = useParameterSchema(deviceId, 'Device.DeviceInfo.', Boolean(deviceId));
  const managementServerSchemaQuery = useParameterSchema(deviceId, 'Device.ManagementServer.', Boolean(deviceId));

  const availablePathSet = useMemo(() => {
    const paths = new Set<string>();
    for (const item of searchQuery.data ?? []) {
      paths.add(item.parameterPath);
    }
    for (const schema of [lmtSchemaQuery.data, deviceInfoSchemaQuery.data, managementServerSchemaQuery.data]) {
      for (const item of schema?.parameters ?? []) {
        paths.add(item.path);
      }
    }
    return paths;
  }, [deviceInfoSchemaQuery.data, lmtSchemaQuery.data, managementServerSchemaQuery.data, searchQuery.data]);

  const effectivePasswordPath = useMemo(
    () => pickAvailablePath(passwordFlow.passwordPaths, availablePathSet),
    [availablePathSet, passwordFlow.passwordPaths]
  );
  const effectiveUsernamePath = useMemo(
    () => pickAvailablePath(passwordFlow.usernamePaths, availablePathSet),
    [availablePathSet, passwordFlow.usernamePaths]
  );

  const schemaLoading = searchQuery.isLoading
    || lmtSchemaQuery.isLoading
    || deviceInfoSchemaQuery.isLoading
    || managementServerSchemaQuery.isLoading;

  const refreshSchema = () => {
    void searchQuery.refetch();
    void lmtSchemaQuery.refetch();
    void deviceInfoSchemaQuery.refetch();
    void managementServerSchemaQuery.refetch();
  };

  const watchedOperation = Form.useWatch('operation', form);
  const selectedOperation = watchedOperation && supportedOperations.includes(watchedOperation)
    ? watchedOperation
    : supportedOperations[0] ?? 'update';
  const isResetOperation = selectedOperation === 'reset';
  const resetPasswordValue = securitySettings?.raw.get('defaultPasswd')?.trim() || DEFAULT_RESET_PASSWORD_VALUE;

  useEffect(() => {
    form.setFieldsValue({ operation: selectedOperation });
    form.resetFields(['username', 'newPassword', 'confirmPassword']);
  }, [form, selectedOperation]);

  const handleSubmit = async () => {
    let values: PasswordFormValues;
    try {
      values = isResetOperation
        ? await form.validateFields(['operation'])
        : await form.validateFields();
    } catch {
      return;
    }
    const password = isResetOperation ? resetPasswordValue : values.newPassword ?? '';
    const parameters = [{
      name: effectivePasswordPath,
      value: password,
      type: 'xsd:string',
    }];

    if (!isResetOperation && passwordFlow.requiresUsername && effectiveUsernamePath) {
      parameters.unshift({
        name: effectiveUsernamePath,
        value: values.username ?? '',
        type: 'xsd:string',
      });
    }

    const submit = async () => {
      try {
        onTaskCleared?.();
        setTaskSubmitting(true);
        if (isResetOperation) {
          const resetTask = await deviceTaskApi.createTask(deviceSn, {
            method: BAICELLS_PASSWORD_RESET_METHOD,
            params: {},
            commandKey: `password-reset-${Date.now()}`,
            maxRetries: 0,
            description: 'Reset LMT login password',
          });
          onTaskSubmitted?.(resetTask.id);
        } else {
          const commandKey = `password-update-${Date.now()}`;
          const updateTask = await deviceTaskApi.createTask(deviceSn, {
            method: SET_PARAMETER_VALUES_METHOD,
            params: { values: parameters },
            commandKey,
            maxRetries: 0,
            description: 'Update LMT login password',
          });
          onTaskSubmitted?.(updateTask.id);
        }
        message.success(t('device.password.submitted'));
      } catch (error) {
        const description = error instanceof Error ? error.message : t('device.multi.unknownErrorHint');
        notification.error({
          message: t('device.password.taskFailed'),
          description,
          duration: 8,
        });
      } finally {
        setTaskSubmitting(false);
      }
    };

    if (isResetOperation) {
      modal.confirm({
        title: t('device.password.resetConfirmTitle'),
        content: t('device.password.resetConfirmContent'),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: submit,
      });
      return;
    }
    await submit();
  };

  const currentAction = isResetOperation
    ? t('device.password.actionReset')
    : t('device.password.actionUpdate');

  return (
    <div style={{ padding: '16px 0 24px', maxWidth: 920 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {!isResetOperation && (
          <Alert
            type="info"
            showIcon
            message={t(passwordFlow.noticeTitle)}
            description={t(passwordFlow.noticeDesc)}
          />
        )}

        <Form<PasswordFormValues>
          form={form}
          layout="vertical"
          initialValues={{ operation: 'update' }}
          style={{ maxWidth: 560 }}
        >
          {showOperationPicker && (
            <Form.Item name="operation" label={t('device.password.operation')}>
              <Radio.Group optionType="button" buttonStyle="solid">
                {supportedOperations.includes('update') && (
                  <Radio.Button value="update">{t('device.password.actionUpdate')}</Radio.Button>
                )}
                {supportedOperations.includes('reset') && (
                  <Radio.Button value="reset">{t('device.password.actionReset')}</Radio.Button>
                )}
              </Radio.Group>
            </Form.Item>
          )}

          {!isResetOperation && (
            <>
              {passwordFlow.requiresUsername && (
                <Form.Item
                  name="username"
                  label={t('device.password.username')}
                  rules={[{ required: true, message: t('device.password.usernameRequired') }]}
                >
                  <Input autoComplete="off" />
                </Form.Item>
              )}

              <Form.Item
                name="newPassword"
                label={t('device.password.newPassword')}
                rules={[
                  { required: true, message: t('device.password.passwordRequired') },
                ]}
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
            </>
          )}

          <Space>
            <Button
              type="primary"
              icon={<SendOutlined />}
              loading={taskSubmitting}
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
      </Space>
    </div>
  );
}
