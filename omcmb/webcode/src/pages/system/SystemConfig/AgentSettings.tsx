import { useEffect } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  Segmented,
  Select,
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
import { AddonInput, AddonInputNumber } from '@/components/common/InputAddon';

type AgentFormValues = AgentAdminConfigUpdate & {
  agentStudioServiceToken?: string;
};

const METHOD_OPTIONS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];
const READ_ONLY_METHODS = ['GET'];
const WRITE_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

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
  const allowedMethods = Form.useWatch('allowedMethods', form) ?? READ_ONLY_METHODS;
  const executionMode = allowedMethods.some((method) => method !== 'GET') ? 'write' : 'read';

  useEffect(() => {
    if (!config) return;
    form.setFieldsValue({
      enabled: config.enabled,
      agentStudioBaseUrl: config.agentStudioBaseUrl,
      agentStudioServiceToken: '',
      connectorSlug: config.connectorSlug,
      allowedMethods: config.policy.allowedMethods,
      blockedPathPrefixes: config.policy.blockedPathPrefixes,
      toolTimeoutSeconds: config.policy.toolTimeoutSeconds,
      maxResponseBytes: config.policy.maxResponseBytes,
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
        connectorSlug: values.connectorSlug?.trim() || undefined,
        allowedMethods: values.allowedMethods?.length ? values.allowedMethods : READ_ONLY_METHODS,
        blockedPathPrefixes: values.blockedPathPrefixes ?? [],
        toolTimeoutSeconds: values.toolTimeoutSeconds,
        maxResponseBytes: values.maxResponseBytes,
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
            title={t('system.agent.lastError')}
            description={config.lastError}
          />
        ) : null}

        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.agent.section.connection')}</span>}
          style={{ marginBottom: 16 }}
        >
          <Form.Item
            name="agentStudioBaseUrl"
            label={t('system.agent.agentStudioBaseUrl')}
            rules={[urlRule]}
          >
            <Input placeholder="https://agent.example.com" autoComplete="off" />
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
          <Form.Item name="connectorSlug" label={t('system.agent.connectorSlug')}>
            <Input placeholder="external-agent-..." autoComplete="off" />
          </Form.Item>
        </Card>

        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.agent.section.runtime')}</span>}
          style={{ marginBottom: 16 }}
        >
          <Space orientation="vertical" style={{ width: '100%' }} size={12}>
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
            <AddonInput
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
            <AddonInput
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

        <Card
          size="small"
          title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.agent.section.security')}</span>}
        >
          <Space orientation="vertical" style={{ width: '100%' }} size={14}>
            <Form.Item name="enabled" label={t('system.agent.visible')} valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch />
            </Form.Item>
            <Form.Item label={t('system.agent.executionMode')} style={{ marginBottom: 0 }}>
              <Segmented
                size="small"
                value={executionMode}
                options={[
                  { label: t('system.agent.readOnlyMode'), value: 'read' },
                  { label: t('system.agent.writeMode'), value: 'write' },
                ]}
                onChange={(value) => {
                  form.setFieldValue('allowedMethods', value === 'write' ? WRITE_METHODS : READ_ONLY_METHODS);
                }}
              />
            </Form.Item>
            <Form.Item
              name="allowedMethods"
              label={t('system.agent.allowedMethods')}
              rules={[
                {
                  validator: (_: unknown, value?: string[]) => {
                    if (!value?.length) return Promise.reject(new Error(t('system.agent.required')));
                    return Promise.resolve();
                  },
                },
              ]}
            >
              <Checkbox.Group options={METHOD_OPTIONS} />
            </Form.Item>
            <Space wrap size={16}>
              <Form.Item
                name="maxResponseBytes"
                label={t('system.agent.maxResponseBytes')}
                rules={[{ type: 'number', min: 4096, max: 4194304 }]}
              >
                <AddonInputNumber min={4096} max={4194304} step={4096} compactStyle={{ width: 180 }} />
              </Form.Item>
              <Form.Item
                name="toolTimeoutSeconds"
                label={t('system.agent.toolTimeoutSeconds')}
                rules={[{ type: 'number', min: 1, max: 300 }]}
              >
                <AddonInputNumber
                  min={1}
                  max={300}
                  compactStyle={{ width: 160 }}
                  addonAfter={t('common.seconds')}
                />
              </Form.Item>
            </Space>
            <Form.Item name="blockedPathPrefixes" label={t('system.agent.blockedPathPrefixes')}>
              <Select
                mode="tags"
                tokenSeparators={[',', '\n']}
                placeholder="/api/v1/auth/*"
                options={(config?.policy.blockedPathPrefixes ?? []).map((value) => ({ label: value, value }))}
              />
            </Form.Item>
            <Alert type="info" showIcon title={t('system.agent.securityHint')} />
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
