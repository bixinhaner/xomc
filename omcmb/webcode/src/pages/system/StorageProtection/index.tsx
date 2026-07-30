import { useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Form,
  InputNumber,
  Popconfirm,
  Progress,
  Row,
  Select,
  Space,
  Switch,
  Tag,
  message,
} from 'antd';
import {
  DatabaseOutlined,
  ReloadOutlined,
  SaveOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useSystemInfo } from '@core/hooks/api/useSystem';
import type { SystemInfo } from '@core/services/api/systemApi';
import {
  useSaveStorageProtectionPolicy,
  useStorageProtectionPolicies,
  useUpdateStorageProtectionPolicy,
} from '@core/hooks/api/useStorageProtection';
import { UNIFIED_STORAGE_TARGET } from '@core/services/api/storageProtectionApi';
import type {
  StorageProtectionPolicy,
  StorageProtectionPolicyPayload,
  StorageProtectionState,
  StorageUnknownBehavior,
} from '@core/services/api/storageProtectionApi';

type PolicyFormValues = Omit<StorageProtectionPolicyPayload, 'targetType' | 'targetId' | 'writeScope'>;

const stateColors: Record<StorageProtectionState, string> = {
  normal: 'green',
  warning: 'orange',
  blocked: 'red',
  unknown: 'default',
};

const defaultFormValues: PolicyFormValues = {
  enabled: true,
  warnUsedPercent: 80,
  recoverUsedPercent: 85,
  blockUsedPercent: 90,
  checkIntervalSeconds: 30,
  unknownBehavior: 'allow_with_alarm',
};

function formatBytes(value: number | undefined, unavailable: string) {
  if (value === undefined) return unavailable;
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let amount = value;
  let index = 0;
  while (amount >= 1024 && index < units.length - 1) {
    amount /= 1024;
    index += 1;
  }
  return `${Number.isInteger(amount) ? amount : amount.toFixed(1)} ${units[index]}`;
}

export default function StorageProtection() {
  const t = useT();
  const [form] = Form.useForm<PolicyFormValues>();
  const [editingId, setEditingId] = useState<string>();
  const { data: policies = [], isFetching: policiesFetching, refetch } = useStorageProtectionPolicies();
  const { data: systemInfo } = useSystemInfo();
  const runtimeInfo = systemInfo as SystemInfo | undefined;
  const savePolicy = useSaveStorageProtectionPolicy();
  const updatePolicy = useUpdateStorageProtectionPolicy();
  const [editorOpen, setEditorOpen] = useState(false);
  const unavailable = t('common.notAvailable');
  const policy = policies[0];

  const openCreate = () => {
    setEditingId(undefined);
    form.setFieldsValue(defaultFormValues);
    setEditorOpen(true);
  };

  const closeEditor = () => {
    setEditorOpen(false);
    setEditingId(undefined);
    form.resetFields();
  };

  const startEdit = (policy: StorageProtectionPolicy) => {
    setEditingId(policy.id);
    form.setFieldsValue({
      enabled: policy.enabled,
      warnUsedPercent: policy.warnUsedPercent,
      blockUsedPercent: policy.blockUsedPercent,
      recoverUsedPercent: policy.recoverUsedPercent,
      checkIntervalSeconds: policy.checkIntervalSeconds,
      unknownBehavior: policy.unknownBehavior,
    });
    setEditorOpen(true);
  };

  const handleSave = async (values: PolicyFormValues) => {
    const payload: StorageProtectionPolicyPayload = {
      ...values,
      targetType: UNIFIED_STORAGE_TARGET.targetType,
      targetId: UNIFIED_STORAGE_TARGET.targetId,
      writeScope: UNIFIED_STORAGE_TARGET.writeScope,
    };
    try {
      if (editingId) {
        await updatePolicy.mutateAsync({ id: editingId, payload });
      } else {
        await savePolicy.mutateAsync(payload);
      }
      message.success(t('common.saveSuccess'));
      closeEditor();
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.saveFailed'));
    }
  };

  const togglePolicy = async (enabled: boolean) => {
    if (!policy) return;
    try {
      await updatePolicy.mutateAsync({
        id: policy.id,
        payload: {
          enabled,
          warnUsedPercent: policy.warnUsedPercent,
          blockUsedPercent: policy.blockUsedPercent,
          recoverUsedPercent: policy.recoverUsedPercent,
          checkIntervalSeconds: policy.checkIntervalSeconds,
          unknownBehavior: policy.unknownBehavior,
          targetType: UNIFIED_STORAGE_TARGET.targetType,
          targetId: UNIFIED_STORAGE_TARGET.targetId,
          writeScope: UNIFIED_STORAGE_TARGET.writeScope,
        },
      });
      message.success(t(enabled ? 'system.storageProtection.enableSuccess' : 'system.storageProtection.disableSuccess'));
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.saveFailed'));
    }
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Alert
        type="info"
        showIcon
        message={t('system.storageProtection.description')}
        description={t('system.storageProtection.unifiedTargetDescription')}
      />

      <Card
        title={<Space><DatabaseOutlined />{t('system.storageProtection.capacityOverview')}</Space>}
        extra={<Button icon={<ReloadOutlined />} loading={policiesFetching} onClick={() => void refetch()}>{t('common.refresh')}</Button>}
      >
        <Row gutter={[16, 16]}>
          {(runtimeInfo?.storage ?? []).filter((metric) => metric.kind === 'host_filesystem').map((metric) => {
            const ratio = metric.status === 'available' ? metric.usedPercent : undefined;
            const state = policy?.currentState;
            return (
              <Col xs={24} md={12} xl={8} key={metric.id}>
                <Card size="small" title={metric.label}>
                  <Space direction="vertical" style={{ width: '100%' }} size={4}>
                    <span style={{ color: '#888', wordBreak: 'break-all' }}>{metric.mountPath ?? metric.mountpoint ?? metric.instance ?? metric.id}</span>
                    {ratio !== undefined ? (
                      <Progress percent={ratio} status={ratio >= 90 ? 'exception' : ratio >= 80 ? 'active' : 'normal'} />
                    ) : <span>{metric.status === 'stale' ? t('system.dashboard.storage.stale') : unavailable}</span>}
                    <span>{t('system.storageProtection.used')}: {formatBytes(metric.usedBytes, unavailable)} / {formatBytes(metric.totalBytes, unavailable)}</span>
                    {state && <Tag color={stateColors[state]}>{t(`system.storageProtection.state.${state}`)}</Tag>}
                  </Space>
                </Card>
              </Col>
            );
          })}
          {(runtimeInfo?.storage ?? []).filter((metric) => metric.kind === 'host_filesystem').length === 0 && <Col span={24}><span>{unavailable}</span></Col>}
        </Row>
      </Card>

      <Row gutter={16} align="top">
        <Col xs={24}>
          <Card
            loading={policiesFetching}
            title={<Space><SettingOutlined />{t('system.storageProtection.policySummary')}</Space>}
            extra={policy && (
              <Space wrap>
                <Popconfirm
                  title={t(policy.enabled ? 'system.storageProtection.disableConfirm' : 'system.storageProtection.enableConfirm')}
                  onConfirm={() => void togglePolicy(!policy.enabled)}
                  okText={t('common.confirm')}
                  cancelText={t('common.cancel')}
                >
                  <Button loading={updatePolicy.isPending}>
                    {t(policy.enabled ? 'system.storageProtection.disablePolicy' : 'system.storageProtection.enablePolicy')}
                  </Button>
                </Popconfirm>
                <Button type="primary" onClick={() => startEdit(policy)}>{t('common.edit')}</Button>
              </Space>
            )}
          >
            {policy ? (
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label={t('system.storageProtection.target')}>
                  <Space direction="vertical" size={0}>
                    <span>{t('system.storageProtection.unifiedTarget')}</span>
                    <span style={{ color: '#888' }}>/</span>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label={t('system.storageProtection.enabled')}>
                  <Tag color={policy.enabled ? 'green' : 'default'}>
                    {t(policy.enabled ? 'common.enabled' : 'common.disabled')}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('system.storageProtection.thresholds')}>
                  {`${policy.warnUsedPercent}% / ${policy.blockUsedPercent}% / ${policy.recoverUsedPercent}%`}
                </Descriptions.Item>
                <Descriptions.Item label={t('system.storageProtection.checkInterval')}>
                  {`${policy.checkIntervalSeconds} ${t('system.storageProtection.seconds')}`}
                </Descriptions.Item>
                <Descriptions.Item label={t('system.storageProtection.unknownBehavior')}>
                  {t(`system.storageProtection.unknownBehavior.${policy.unknownBehavior}`)}
                </Descriptions.Item>
                <Descriptions.Item label={t('system.storageProtection.state')}>
                  <Tag color={stateColors[policy.currentState]}>
                    {t(`system.storageProtection.state.${policy.currentState}`)}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('system.storageProtection.currentBlocks')} span={2}>
                  {policy.currentState === 'blocked' ? (
                    <Space direction="vertical" size={0}>
                      <Tag color="red">{t('system.storageProtection.state.blocked')}</Tag>
                      <span style={{ color: '#888' }}>
                        {t('system.storageProtection.lastObserved')}: {policy.lastObservedRatio === undefined ? unavailable : `${(policy.lastObservedRatio * 100).toFixed(1)}%`}
                      </span>
                    </Space>
                  ) : <Tag>{t('system.storageProtection.notBlocked')}</Tag>}
                </Descriptions.Item>
              </Descriptions>
            ) : (
              <Space direction="vertical" align="center" style={{ width: '100%', padding: '24px 0' }}>
                <span>{t('system.storageProtection.noPolicyDescription')}</span>
                <Button type="primary" icon={<SettingOutlined />} onClick={openCreate}>
                  {t('system.storageProtection.configurePolicy')}
                </Button>
              </Space>
            )}
          </Card>
        </Col>
      </Row>

      <Drawer
        title={t(editingId ? 'system.storageProtection.editPolicy' : 'system.storageProtection.configurePolicy')}
        open={editorOpen}
        onClose={closeEditor}
        destroyOnClose
        width={480}
      >
        <Form<PolicyFormValues> form={form} layout="vertical" onFinish={(values) => void handleSave(values)}>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="warnUsedPercent" label={t('system.storageProtection.warnThreshold')} rules={[{ required: true }]}>
                <InputNumber min={0} max={98} addonAfter="%" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="blockUsedPercent" label={t('system.storageProtection.blockThreshold')} rules={[{ required: true }]}>
                <InputNumber min={1} max={100} addonAfter="%" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="recoverUsedPercent" label={t('system.storageProtection.recoverThreshold')} rules={[{ required: true }]}>
                <InputNumber min={0} max={99} addonAfter="%" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="checkIntervalSeconds" label={t('system.storageProtection.checkInterval')} rules={[{ required: true }]}>
            <InputNumber min={1} max={86400} addonAfter={t('system.storageProtection.seconds')} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="unknownBehavior" label={t('system.storageProtection.unknownBehavior')} rules={[{ required: true }]}>
            <Select options={(['allow_with_alarm', 'block_new_uploads'] as StorageUnknownBehavior[]).map((value) => ({ value, label: t(`system.storageProtection.unknownBehavior.${value}`) }))} />
          </Form.Item>
          <Form.Item name="enabled" label={t('system.storageProtection.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={savePolicy.isPending || updatePolicy.isPending}>
              {t('common.save')}
            </Button>
            <Button onClick={closeEditor}>{t('common.cancel')}</Button>
          </Space>
        </Form>
      </Drawer>

      <Card title={t('system.storageProtection.retentionTitle')}>
        <Alert type="warning" showIcon message={t('system.storageProtection.retentionPending')} />
      </Card>

    </div>
  );
}
