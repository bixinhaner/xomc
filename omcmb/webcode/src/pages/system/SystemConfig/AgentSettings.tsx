import { useEffect } from 'react';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Space,
  Spin,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  ApiOutlined,
  CheckCircleOutlined,
  CopyOutlined,
  ReloadOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  useAdminAgentConfig,
  useSaveAdminAgentConfig,
  useSyncAdminAgentConfig,
  useTestAdminAgentConfig,
} from '@core/hooks/api/useAgentConfig';
import type { AgentAdminConfigUpdate } from '@core/types/agentConfig';
import { useT } from '@/hooks/useT';

type AgentFormValues = AgentAdminConfigUpdate & {
  agentStudioServiceToken?: string;
};

function statusColor(status: string) {
  if (status === 'connected') return 'success';
  if (status === 'error') return 'error';
  if (status === 'disabled') return 'default';
  return 'warning';
}

function isHttpUrl(value: string | undefined) {
  if (!value) return true;
  return /^https?:\/\/[^/]+/i.test(value.trim());
}

async function copyValue(value: string, successText: string) {
  if (!value) return;
  await navigator.clipboard?.writeText(value);
  void message.success(successText);
}

export default function AgentSettings() {
  const t = useT();
  const [form] = Form.useForm<AgentFormValues>();
  const configQ = useAdminAgentConfig();
  const saveM = useSaveAdminAgentConfig();
  const testM = useTestAdminAgentConfig();
  const syncM = useSyncAdminAgentConfig();
  const config = configQ.data;
  const enabled = Form.useWatch('enabled', form);

  useEffect(() => {
    if (!config) return;
    form.setFieldsValue({
      enabled: config.enabled,
      agentStudioBaseUrl: config.agentStudioBaseUrl,
      agentStudioServiceToken: '',
      omcPublicBaseUrl: config.omcPublicBaseUrl,
      connectorSlug: config.connectorSlug,
    });
  }, [config, form]);

  const buildPayload = async (): Promise<AgentAdminConfigUpdate | null> => {
    try {
      const values = await form.validateFields();
      const token = values.agentStudioServiceToken?.trim();
      return {
        enabled: Boolean(values.enabled),
        agentStudioBaseUrl: values.agentStudioBaseUrl?.trim() ?? '',
        agentStudioServiceToken: token || undefined,
        omcPublicBaseUrl: values.omcPublicBaseUrl?.trim() ?? '',
        connectorSlug: values.connectorSlug?.trim() || undefined,
      };
    } catch {
      void message.error(t('common.formValidationFailed'));
      return null;
    }
  };

  const handleSave = async () => {
    const payload = await buildPayload();
    if (!payload) return;
    try {
      await saveM.mutateAsync(payload);
      void message.success(t('system.agent.saveSuccess'));
    } catch (err) {
      void message.error(err instanceof Error ? err.message : t('sysconfig.error.saveFailed'));
    }
  };

  const handleTest = async () => {
    const payload = await buildPayload();
    if (!payload) return;
    try {
      await testM.mutateAsync(payload);
      void message.success(t('system.agent.testSuccess'));
    } catch (err) {
      void message.error(err instanceof Error ? err.message : t('agent.errorPrefix'));
    }
  };

  const handleSync = async () => {
    const payload = await buildPayload();
    if (!payload) return;
    try {
      await syncM.mutateAsync(payload);
      void message.success(t('system.agent.syncSuccess'));
    } catch (err) {
      void message.error(err instanceof Error ? err.message : t('agent.errorPrefix'));
    }
  };

  const urlRule = {
    validator: (_: unknown, value?: string) => {
      if (enabled && !value?.trim()) return Promise.reject(new Error(t('system.agent.required')));
      if (!isHttpUrl(value)) return Promise.reject(new Error(t('system.agent.urlInvalid')));
      return Promise.resolve();
    },
  };

  return (
    <Spin spinning={configQ.isLoading || configQ.isFetching}>
      <Form form={form} layout="vertical" size="small" initialValues={{ enabled: false }}>
        {config?.lastError ? (
          <Alert
            type="error"
            showIcon
            style={{ marginBottom: 16 }}
            message={t('system.agent.lastError')}
            description={config.lastError}
          />
        ) : null}

        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.agent.section.connection')}</span>}
          style={{ marginBottom: 16 }}
        >
          <Form.Item name="enabled" label={t('system.agent.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="agentStudioBaseUrl"
            label={t('system.agent.agentStudioBaseUrl')}
            rules={[urlRule]}
          >
            <Input placeholder="https://agent.example.com" />
          </Form.Item>
          <Form.Item
            name="agentStudioServiceToken"
            label={
              <Space>
                <span>{t('system.agent.serviceToken')}</span>
                <Tag color={config?.serviceTokenConfigured ? 'success' : 'warning'}>
                  {config?.serviceTokenConfigured
                    ? t('system.agent.serviceTokenSet')
                    : t('system.agent.serviceTokenUnset')}
                </Tag>
              </Space>
            }
            rules={[
              {
                validator: (_: unknown, value?: string) => {
                  if (enabled && !config?.serviceTokenConfigured && !value?.trim()) {
                    return Promise.reject(new Error(t('system.agent.required')));
                  }
                  return Promise.resolve();
                },
              },
            ]}
          >
            <Input.Password placeholder={t('system.agent.serviceTokenPlaceholder')} autoComplete="new-password" />
          </Form.Item>
          <Form.Item
            name="omcPublicBaseUrl"
            label={t('system.agent.omcPublicBaseUrl')}
            rules={[urlRule]}
          >
            <Input placeholder="https://ops.example.com" />
          </Form.Item>
          <Form.Item name="connectorSlug" label={t('system.agent.connectorSlug')}>
            <Input placeholder="external-agent-..." />
          </Form.Item>
        </Card>

        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.agent.section.runtime')}</span>}
        >
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            <Space>
              <Typography.Text type="secondary">{t('system.agent.status')}</Typography.Text>
              <Tag color={statusColor(config?.status ?? 'not_configured')}>
                {config?.status ?? 'not_configured'}
              </Tag>
              {config?.lastValidatedAt ? (
                <Typography.Text type="secondary">
                  {t('system.agent.lastValidatedAt')}: {config.lastValidatedAt}
                </Typography.Text>
              ) : null}
            </Space>
            <Input
              readOnly
              addonBefore={t('system.agent.connectorId')}
              value={config?.connectorId || t('system.agent.emptyValue')}
              suffix={
                <Button
                  type="text"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => copyValue(config?.connectorId ?? '', t('system.agent.copied'))}
                />
              }
            />
            <Input
              readOnly
              addonBefore={t('system.agent.runtimeStreamUrl')}
              value={config?.runtimeStreamUrl || t('system.agent.emptyValue')}
              suffix={
                <Button
                  type="text"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => copyValue(config?.runtimeStreamUrl ?? '', t('system.agent.copied'))}
                />
              }
            />
          </Space>
        </Card>

        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => configQ.refetch()}>
              {t('system.agent.reset')}
            </Button>
            <Button icon={<ApiOutlined />} loading={testM.isPending} onClick={handleTest}>
              {t('system.agent.test')}
            </Button>
            <Button icon={<SaveOutlined />} loading={saveM.isPending} onClick={handleSave}>
              {t('system.agent.save')}
            </Button>
            <Button type="primary" icon={<CheckCircleOutlined />} loading={syncM.isPending} onClick={handleSync}>
              {t('system.agent.sync')}
            </Button>
          </Space>
        </div>
      </Form>
    </Spin>
  );
}
