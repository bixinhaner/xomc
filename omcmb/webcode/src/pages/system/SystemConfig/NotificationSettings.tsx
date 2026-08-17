import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  message,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
} from 'antd';
import { MailOutlined, SaveOutlined, SendOutlined } from '@ant-design/icons';
import {
  useBatchUpdateSysConfigs,
  useSendTestEmail,
  useSysConfigsByCategory,
} from '@core/hooks/api/useSystem';
import { useT } from '@/hooks/useT';
import { buildBatchItems } from './sysConfigSerialize';

const EMAIL_CATEGORY = 'notification.email';

interface EmailSettingsFormValues {
  enabled: boolean;
  host: string;
  port: number;
  security_mode: 'none' | 'starttls' | 'implicit_tls';
  auth_enabled: boolean;
  username: string;
  password?: string;
  from_address: string;
  from_name: string;
  timeout_seconds: number;
}

const defaultValues: EmailSettingsFormValues = {
  enabled: false,
  host: '',
  port: 587,
  security_mode: 'starttls',
  auth_enabled: true,
  username: '',
  password: '',
  from_address: '',
  from_name: '',
  timeout_seconds: 10,
};

export default function NotificationSettings() {
  const t = useT();
  const [form] = Form.useForm<EmailSettingsFormValues>();
  const [dirty, setDirty] = useState(false);
  const [testRecipient, setTestRecipient] = useState('');
  const {
    data: configs,
    isLoading,
    isFetching,
    isError,
    isSuccess,
    refetch,
  } = useSysConfigsByCategory(EMAIL_CATEGORY);
  const batchUpdate = useBatchUpdateSysConfigs();
  const sendTestEmail = useSendTestEmail();
  const enabled = Form.useWatch('enabled', form) ?? false;
  const authEnabled = Form.useWatch('auth_enabled', form) ?? false;
  const securityMode = Form.useWatch('security_mode', form);

  const passwordConfigured = useMemo(
    () => configs?.some((item) => item.key === 'password' && item.isSecret && item.isConfigured) ?? false,
    [configs],
  );

  useEffect(() => {
    if (!isSuccess) return;
    const values: Record<string, unknown> = { ...defaultValues };
    for (const item of configs ?? []) {
      if (item.isSecret) continue;
      if (item.valueType === 'bool') values[item.key] = item.value === 'true' || item.value === '1';
      else if (item.valueType === 'int') values[item.key] = Number.parseInt(item.value, 10);
      else values[item.key] = item.value;
    }
    form.setFieldsValue({ ...values, password: '' } as EmailSettingsFormValues);
    setDirty(false);
  }, [configs, form, isSuccess]);

  const handleSave = async () => {
    if (!isSuccess || isFetching) {
      void message.error(t('empty.loadFailed'));
      return;
    }
    try {
      const values = await form.validateFields();
      const items = buildBatchItems(values as unknown as Record<string, unknown>, configs);
      await batchUpdate.mutateAsync({ category: EMAIL_CATEGORY, items });
      await refetch();
      form.setFieldValue('password', '');
      setDirty(false);
      void message.success(t('system.notification.saveSuccess'));
    } catch (error) {
      if (error instanceof Error) void message.error(error.message);
    }
  };

  const handleTest = async () => {
    const recipient = testRecipient.trim();
    if (!recipient) {
      void message.warning(t('system.notification.testRecipientRequired'));
      return;
    }
    try {
      await sendTestEmail.mutateAsync(recipient);
      void message.success(t('system.notification.testEmailSent'));
    } catch (error) {
      const text = error instanceof Error ? error.message : t('system.notification.testEmailFailed');
      void message.error(text);
    }
  };

  return (
    <Spin spinning={isLoading}>
      {isError && (
        <Alert
          type="error"
          showIcon
          title={t('empty.loadFailed')}
          action={<Button onClick={() => void refetch()}>{t('common.retry')}</Button>}
          style={{ marginBottom: 16 }}
        />
      )}
      <Form<EmailSettingsFormValues>
        form={form}
        layout="vertical"
        initialValues={defaultValues}
        onValuesChange={() => setDirty(true)}
      >
        <Card
          size="small"
          title={t('system.notification.emailService')}
          extra={enabled ? <Tag color="success">{t('common.enabled')}</Tag> : <Tag>{t('common.disabled')}</Tag>}
        >
          <Alert
            type="info"
            showIcon
            title={t('system.notification.scopeHint')}
            style={{ marginBottom: 16 }}
          />

          <Form.Item name="enabled" label={t('system.notification.channelEnabled')} valuePropName="checked">
            <Switch />
          </Form.Item>

          <Row gutter={[24, 0]}>
            <Col xs={24} md={12} xl={8}>
              <Form.Item
                name="host"
                label={t('system.notification.smtpServer')}
                rules={[{ required: enabled, message: t('system.notification.smtpServerRequired') }]}
              >
                <Input placeholder="smtp.example.com" />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12} md={6} xl={4}>
              <Form.Item
                name="port"
                label={t('common.port')}
                rules={[{ required: true }]}
              >
                <InputNumber min={1} max={65535} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12} md={6} xl={6}>
              <Form.Item name="security_mode" label={t('system.notification.securityMode')} rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: 'starttls', label: 'STARTTLS' },
                    { value: 'implicit_tls', label: t('system.notification.implicitTls') },
                    { value: 'none', label: t('system.notification.noTls') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12} md={6} xl={4}>
              <Form.Item name="timeout_seconds" label={t('system.notification.timeoutSeconds')} rules={[{ required: true }]}>
                <InputNumber min={1} max={120} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          {securityMode === 'none' && (
            <Alert
              type="warning"
              showIcon
              title={t('system.notification.noTlsWarning')}
              style={{ marginBottom: 16 }}
            />
          )}

          <Form.Item name="auth_enabled" label={t('system.notification.authEnabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Row gutter={[24, 0]}>
            <Col xs={24} md={12} xl={8}>
              <Form.Item
                name="username"
                label={t('system.notification.username')}
                rules={[{ required: enabled && authEnabled, message: t('system.notification.usernameRequired') }]}
              >
                <Input autoComplete="off" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12} xl={8}>
              <Form.Item
                name="password"
                label={t('common.password')}
                rules={[{
                  validator: async (_, value: string | undefined) => {
                    if (enabled && authEnabled && !passwordConfigured && !value) {
                      throw new Error(t('system.notification.passwordRequired'));
                    }
                  },
                }]}
              >
                <Input.Password
                  autoComplete="new-password"
                  visibilityToggle={false}
                  placeholder={t(passwordConfigured
                    ? 'system.notification.passwordConfiguredPlaceholder'
                    : 'system.notification.passwordUnsetPlaceholder')}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={[24, 0]}>
            <Col xs={24} md={12} xl={8}>
              <Form.Item
                name="from_address"
                label={t('system.notification.fromAddress')}
                rules={[
                  { required: enabled, message: t('system.notification.fromAddressRequired') },
                  { type: 'email', message: t('system.notification.emailInvalid') },
                ]}
              >
                <Input prefix={<MailOutlined />} placeholder="omc@example.com" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12} xl={8}>
              <Form.Item name="from_name" label={t('system.notification.fromName')}>
                <Input placeholder={t('system.notification.fromNamePlaceholder')} />
              </Form.Item>
            </Col>
          </Row>

          <div
            data-testid="email-settings-actions"
            style={{ display: 'flex', justifyContent: 'flex-end', paddingTop: 8 }}
          >
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={batchUpdate.isPending}
              disabled={!isSuccess || isFetching}
              onClick={() => void handleSave()}
            >
              {t('common.save')}
            </Button>
          </div>
        </Card>

        <Card size="small" title={t('system.notification.testEmail')} style={{ marginTop: 16 }}>
          <Alert
            type={dirty ? 'warning' : 'info'}
            showIcon
            title={t(dirty
              ? 'system.notification.testSavedOnlyDirty'
              : 'system.notification.testSavedOnly')}
            style={{ marginBottom: 16 }}
          />
          <Space wrap align="start">
            <Input
              value={testRecipient}
              onChange={(event) => setTestRecipient(event.target.value)}
              prefix={<MailOutlined />}
              placeholder={t('system.notification.testRecipientPlaceholder')}
              style={{ width: 340 }}
            />
            <Button
              icon={<SendOutlined />}
              loading={sendTestEmail.isPending}
              disabled={!enabled || dirty || !testRecipient.trim()}
              onClick={() => void handleTest()}
            >
              {t('common.test')}
            </Button>
          </Space>
        </Card>
      </Form>
    </Spin>
  );
}
