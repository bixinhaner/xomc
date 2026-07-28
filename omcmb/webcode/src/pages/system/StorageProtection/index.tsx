import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  List,
  Form,
  Input,
  InputNumber,
  Progress,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
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
  useStorageProtectionEvents,
  useStorageProtectionPolicies,
  useUpdateStorageProtectionPolicy,
} from '@core/hooks/api/useStorageProtection';
import type {
  StorageProtectionPolicy,
  StorageProtectionPolicyPayload,
  StorageProtectionState,
  StorageTargetType,
  StorageUnknownBehavior,
  StorageWriteScope,
} from '@core/services/api/storageProtectionApi';

type PolicyFormValues = StorageProtectionPolicyPayload;

const targetTypes: StorageTargetType[] = [
  'filesystem',
  'minio',
  'database',
  'redis',
  'nats',
  'monitoring',
  'application',
];

const writeScopes: StorageWriteScope[] = [
  'upload',
  'log',
  'backup',
  'report',
  'trace',
  'pm',
  'mr',
  'all',
];

const stateColors: Record<StorageProtectionState, string> = {
  normal: 'green',
  warning: 'orange',
  blocked: 'red',
  unknown: 'default',
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
  const { data: events = [], isFetching: eventsFetching } = useStorageProtectionEvents();
  const { data: systemInfo } = useSystemInfo();
  const runtimeInfo = systemInfo as SystemInfo | undefined;
  const savePolicy = useSaveStorageProtectionPolicy();
  const updatePolicy = useUpdateStorageProtectionPolicy();
  const unavailable = t('common.notAvailable');

  useEffect(() => {
    if (!editingId) {
      form.resetFields();
      form.setFieldsValue({
        enabled: true,
        warnUsedPercent: 80,
        recoverUsedPercent: 85,
        blockUsedPercent: 90,
        checkIntervalSeconds: 30,
        unknownBehavior: 'allow_with_alarm',
        writeScope: 'all',
        targetType: 'filesystem',
      });
    }
  }, [editingId, form]);

  const startEdit = (policy: StorageProtectionPolicy) => {
    setEditingId(policy.id);
    form.setFieldsValue({
      targetType: policy.targetType,
      targetId: policy.targetId,
      writeScope: policy.writeScope,
      enabled: policy.enabled,
      warnUsedPercent: policy.warnUsedPercent,
      blockUsedPercent: policy.blockUsedPercent,
      recoverUsedPercent: policy.recoverUsedPercent,
      checkIntervalSeconds: policy.checkIntervalSeconds,
      unknownBehavior: policy.unknownBehavior,
    });
  };

  const handleSave = async (values: PolicyFormValues) => {
    try {
      if (editingId) {
        await updatePolicy.mutateAsync({ id: editingId, payload: values });
      } else {
        await savePolicy.mutateAsync(values);
      }
      message.success(t('common.saveSuccess'));
      setEditingId(undefined);
      form.resetFields();
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.saveFailed'));
    }
  };

  const columns: ColumnsType<StorageProtectionPolicy> = [
    {
      title: t('system.storageProtection.target'),
      key: 'target',
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{t(`system.storageProtection.targetType.${record.targetType}`)}</span>
          <span style={{ color: '#888', wordBreak: 'break-all' }}>{record.targetId}</span>
        </Space>
      ),
    },
    {
      title: t('system.storageProtection.scope'),
      dataIndex: 'writeScope',
      render: (scope: StorageWriteScope) => t(`system.storageProtection.scope.${scope}`),
    },
    {
      title: t('system.storageProtection.thresholds'),
      key: 'thresholds',
      render: (_, record) => `${record.warnUsedPercent}% / ${record.blockUsedPercent}% / ${record.recoverUsedPercent}%`,
    },
    {
      title: t('system.storageProtection.state'),
      dataIndex: 'currentState',
      render: (state: StorageProtectionState) => (
        <Tag color={stateColors[state]}>{t(`system.storageProtection.state.${state}`)}</Tag>
      ),
    },
    {
      title: t('system.storageProtection.enabled'),
      dataIndex: 'enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'default'}>
          {t(enabled ? 'common.enabled' : 'common.disabled')}
        </Tag>
      ),
    },
    {
      title: t('common.operation'),
      key: 'operation',
      render: (_, record) => <Button type="link" onClick={() => startEdit(record)}>{t('common.edit')}</Button>,
    },
  ];

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Alert
        type="info"
        showIcon
        message={t('system.storageProtection.description')}
        description={t('system.storageProtection.retentionPending')}
      />

      <Card
        title={<Space><DatabaseOutlined />{t('system.storageProtection.capacityOverview')}</Space>}
        extra={<Button icon={<ReloadOutlined />} loading={policiesFetching} onClick={() => void refetch()}>{t('common.refresh')}</Button>}
      >
        <Row gutter={[16, 16]}>
          {(runtimeInfo?.storage ?? []).map((metric) => {
            const ratio = metric.status === 'available' ? metric.usedPercent : undefined;
            const matchingPolicy = policies.find((policy) => policy.targetId === (metric.targetId ?? metric.id));
            const state = matchingPolicy?.currentState;
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
          {(runtimeInfo?.storage ?? []).length === 0 && <Col span={24}><span>{unavailable}</span></Col>}
        </Row>
      </Card>

      <Row gutter={16} align="top">
        <Col xs={24} xl={15}>
          <Card title={<Space><SettingOutlined />{t('system.storageProtection.policyList')}</Space>}>
            <Table<StorageProtectionPolicy>
              rowKey="id"
              loading={policiesFetching}
              columns={columns}
              dataSource={policies}
              pagination={{ pageSize: 10 }}
              locale={{ emptyText: t('system.storageProtection.noPolicies') }}
            />
          </Card>
        </Col>
        <Col xs={24} xl={9}>
          <Card title={t(editingId ? 'system.storageProtection.editPolicy' : 'system.storageProtection.newPolicy')}>
            <Form<PolicyFormValues> form={form} layout="vertical" onFinish={(values) => void handleSave(values)}>
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="targetType" label={t('system.storageProtection.targetType')} rules={[{ required: true }]}>
                    <Select options={targetTypes.map((value) => ({ value, label: t(`system.storageProtection.targetType.${value}`) }))} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="writeScope" label={t('system.storageProtection.scope')} rules={[{ required: true }]}>
                    <Select options={writeScopes.map((value) => ({ value, label: t(`system.storageProtection.scope.${value}`) }))} />
                  </Form.Item>
                </Col>
              </Row>
              <Form.Item name="targetId" label={t('system.storageProtection.targetId')} rules={[{ required: true }]}>
                <Input placeholder={t('system.storageProtection.targetIdPlaceholder')} />
              </Form.Item>
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
                {editingId && <Button onClick={() => setEditingId(undefined)}>{t('common.cancel')}</Button>}
              </Space>
            </Form>
          </Card>
        </Col>
      </Row>

      <Card title={t('system.storageProtection.retentionTitle')}>
        <Alert type="warning" showIcon message={t('system.storageProtection.retentionPending')} />
      </Card>

      <Row gutter={16} align="top">
        <Col xs={24} xl={10}>
          <Card title={t('system.storageProtection.currentBlocks')}>
            <List
              loading={policiesFetching}
              dataSource={policies.filter((policy) => policy.currentState === 'blocked')}
              locale={{ emptyText: t('system.storageProtection.noBlocks') }}
              renderItem={(policy) => (
                <List.Item>
                  <Space direction="vertical" size={0}>
                    <Tag color="red">{t('system.storageProtection.state.blocked')}</Tag>
                    <span>{policy.targetId} · {t(`system.storageProtection.scope.${policy.writeScope}`)}</span>
                    <span style={{ color: '#888' }}>{t('system.storageProtection.lastObserved')}: {policy.lastObservedRatio === undefined ? unavailable : `${(policy.lastObservedRatio * 100).toFixed(1)}%`}</span>
                  </Space>
                </List.Item>
              )}
            />
          </Card>
        </Col>
        <Col xs={24} xl={14}>
          <Card title={t('system.storageProtection.audit')}>
            <List
              loading={eventsFetching}
              dataSource={events}
              locale={{ emptyText: t('system.storageProtection.noEvents') }}
              renderItem={(event) => (
                <List.Item>
                  <List.Item.Meta
                    title={`${event.targetId} · ${t(`system.storageProtection.scope.${event.writeScope}`)}`}
                    description={`${event.reason} · ${new Date(event.createdAt).toLocaleString()}`}
                  />
                  <Tag color={stateColors[event.newState]}>{t(`system.storageProtection.state.${event.newState}`)}</Tag>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
