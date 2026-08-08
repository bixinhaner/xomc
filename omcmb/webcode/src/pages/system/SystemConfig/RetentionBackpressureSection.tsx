// #318-321 资源保留与上传背压配置（系统配置页 —— 一个 tab，4 张分类卡片）。
//
// 后端 sys_configs 分类（seed migrations/seed/000004，各模块热加载）：
//   acs.backpressure     #318 PM 上传背压 watchdog（ACS 每采样周期轮询刷新）
//   minio.retention      #319 原始件 ILM 保留天数（app RegisterSavedHook 重应用 lifecycle）
//   stationlog.retention #320+#798 基站日志按时间保留 + 全局/每设备文件数配额（worker TTL 缓存）
//   raw_archive          #836 PM/MR 新文件入库后一次性压缩开关（worker TTL 缓存）
//
// 接入方式与 PmRetentionSection 一致：复用 useSysConfigsByCategory + useBatchUpdateSysConfigs，
// 每张卡片自管 form + 保存（一次保存 = 该分类一次 batch upsert）。

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
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
  Spin,
} from 'antd';
import {
  DatabaseOutlined,
  ReloadOutlined,
  SaveOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import {
  useSaveStorageProtectionPolicy,
  useStorageProtectionEvents,
  useStorageProtectionPolicies,
  useStorageProtectionTargets,
  useUpdateStorageProtectionPolicy,
} from '@core/hooks/api/useStorageProtection';
import {
  UNIFIED_STORAGE_TARGET,
  type StorageProtectionPolicy,
  type StorageProtectionPolicyPayload,
  type StorageProtectionEvent,
  type StorageProtectionState,
  type StorageProtectionTarget,
  type StorageUnknownBehavior,
} from '@core/services/api/storageProtectionApi';
import { useT, type TranslateFn } from '@/hooks/useT';

type FieldType = 'int' | 'float' | 'bool';

interface FieldSpec {
  key: string;
  type: FieldType;
  min?: number;
  max?: number;
  step?: number;
}

interface CardSpec {
  category: string;
  titleKey: string;
  descKey: string;
  fields: FieldSpec[];
}

type PolicyFormValues = Omit<StorageProtectionPolicyPayload, 'targetType' | 'targetId' | 'writeScope'>;

const stateColors: Record<StorageProtectionState, string> = {
  normal: 'green',
  warning: 'orange',
  blocked: 'red',
  unknown: 'default',
};

const defaultPolicyFormValues: PolicyFormValues = {
  enabled: true,
  warnUsedPercent: 80,
  recoverUsedPercent: 85,
  blockUsedPercent: 90,
  checkIntervalSeconds: 30,
  unknownBehavior: 'allow_with_alarm',
};

// 字段定义与后端 sys_configs 键 + value_type 严格对齐（seed 000004）。
const CARDS: CardSpec[] = [
  {
    category: 'acs.backpressure',
    titleKey: 'retentionBp.bp.title',
    descKey: 'retentionBp.bp.desc',
    fields: [
      { key: 'enabled', type: 'bool' },
      { key: 'disk_high_pct', type: 'int', min: 1, max: 100 },
      { key: 'disk_low_pct', type: 'int', min: 0, max: 100 },
      { key: 'check_interval_sec', type: 'int', min: 1, max: 3600 },
    ],
  },
  {
    category: 'minio.retention',
    titleKey: 'retentionBp.ilm.title',
    descKey: 'retentionBp.ilm.desc',
    fields: [{ key: 'raw_object_days', type: 'int', min: 1, max: 3650 }],
  },
  {
    category: 'stationlog.retention',
    titleKey: 'retentionBp.stationlog.title',
    descKey: 'retentionBp.stationlog.desc',
    fields: [
      { key: 'max_retention_days', type: 'int', min: 1, max: 3650 },
      { key: 'max_file_count', type: 'int', min: 0, max: 100000 },
      { key: 'max_file_count_per_device', type: 'int', min: 0, max: 100000 },
      { key: 'cleanup_interval_minutes', type: 'int', min: 10, max: 1440 },
    ],
  },
  {
    category: 'raw_archive',
    titleKey: 'retentionBp.archive.title',
    descKey: 'retentionBp.archive.desc',
    fields: [{ key: 'compress_after_ingest', type: 'bool' }],
  },
];

function decodeValue(raw: string, type: FieldType): number | boolean {
  if (type === 'bool') {
    return raw === 'true' || raw === '1';
  }
  const n = type === 'int' ? parseInt(raw, 10) : parseFloat(raw);
  return Number.isFinite(n) ? n : 0;
}

function encodeValue(v: unknown, type: FieldType): string {
  if (type === 'bool') {
    return v ? 'true' : 'false';
  }
  return String(v ?? '');
}

function formatBytes(value: number | undefined, unavailable: string) {
  if (value === undefined || value <= 0) return unavailable;
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let amount = value;
  let index = 0;
  while (amount >= 1024 && index < units.length - 1) {
    amount /= 1024;
    index += 1;
  }
  return `${Number.isInteger(amount) ? amount : amount.toFixed(1)} ${units[index]}`;
}

function ratioToPercent(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return undefined;
  return value <= 1 ? Math.round(value * 1000) / 10 : Math.round(value * 10) / 10;
}

function targetTitle(target: StorageProtectionTarget) {
  return target.label || target.mountpoint || target.mountPath || target.targetId;
}

function targetPath(target: StorageProtectionTarget) {
  return target.mountpoint || target.mountPath || target.targetId;
}

function validateThresholds(values: PolicyFormValues) {
  return (
    values.warnUsedPercent >= 0 &&
    values.warnUsedPercent < values.recoverUsedPercent &&
    values.recoverUsedPercent < values.blockUsedPercent &&
    values.blockUsedPercent <= 100
  );
}

function formatObservedRatio(value: number | undefined, unavailable: string) {
  if (value === undefined || !Number.isFinite(value)) return unavailable;
  const percent = value <= 1 ? value * 100 : value;
  return `${percent.toFixed(1)}%`;
}

const eventReasonRules: Array<[string, string]> = [
  ['usage reached warning threshold for two checks', 'system.storageProtection.event.reason.warningThreshold'],
  ['usage reached block threshold for two checks', 'system.storageProtection.event.reason.blockThreshold'],
  ['usage reached block threshold; confirmation pending', 'system.storageProtection.event.reason.blockPending'],
  ['usage recovered below recovery threshold for two checks', 'system.storageProtection.event.reason.recovered'],
  ['storage usage is unavailable', 'system.storageProtection.event.reason.capacityUnknown'],
  ['host filesystem metrics unavailable', 'system.storageProtection.event.reason.capacityUnknown'],
];

function formatEventReason(event: StorageProtectionEvent, t: TranslateFn, unavailable: string) {
  const reason = event.reason.toLowerCase();
  const key = eventReasonRules.find(([needle]) => reason.includes(needle))?.[1]
    ?? 'system.storageProtection.event.reason.stateChanged';
  return t(key, {
    previous: event.previousState ? t(`system.storageProtection.state.${event.previousState}`) : unavailable,
    current: t(`system.storageProtection.state.${event.newState}`),
    ratio: formatObservedRatio(event.observedRatio, unavailable),
  });
}

function StorageProtectionSection() {
  const t = useT();
  const [form] = Form.useForm<PolicyFormValues>();
  const [editingId, setEditingId] = useState<string>();
  const [editorOpen, setEditorOpen] = useState(false);
  const [manualRefreshing, setManualRefreshing] = useState(false);
  const { data: policies = [], isFetching: policiesFetching, refetch: refetchPolicies } = useStorageProtectionPolicies();
  const { data: targets = [], isFetching: targetsFetching, refetch: refetchTargets } = useStorageProtectionTargets();
  const { data: events = [], isFetching: eventsFetching, refetch: refetchEvents } = useStorageProtectionEvents(5);
  const savePolicy = useSaveStorageProtectionPolicy();
  const updatePolicy = useUpdateStorageProtectionPolicy();
  const unavailable = t('common.notAvailable');
  const policy = policies[0];

  const openCreate = () => {
    setEditingId(undefined);
    form.setFieldsValue(defaultPolicyFormValues);
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
      recoverUsedPercent: policy.recoverUsedPercent,
      blockUsedPercent: policy.blockUsedPercent,
      checkIntervalSeconds: policy.checkIntervalSeconds,
      unknownBehavior: policy.unknownBehavior,
    });
    setEditorOpen(true);
  };

  const handleSavePolicy = async (values: PolicyFormValues) => {
    if (!validateThresholds(values)) {
      message.error(t('system.storageProtection.thresholdValidation'));
      return;
    }
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
          recoverUsedPercent: policy.recoverUsedPercent,
          blockUsedPercent: policy.blockUsedPercent,
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

  const refresh = async () => {
    setManualRefreshing(true);
    try {
      const results = await Promise.all([refetchTargets(), refetchPolicies(), refetchEvents()]);
      if (results.some((result) => result.isError)) {
        throw new Error(t('common.refreshFailed'));
      }
      message.success(t('common.refreshSuccess'));
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.refreshFailed'));
    } finally {
      setManualRefreshing(false);
    }
  };

  return (
    <>
      <Alert
        type="info"
        showIcon
        message={t('system.storageProtection.description')}
        description={t('system.storageProtection.unifiedTargetDescription')}
        style={{ marginBottom: 16 }}
      />

      <Card
        title={<Space><DatabaseOutlined />{t('system.storageProtection.capacityOverview')}</Space>}
        style={{ marginBottom: 16 }}
        extra={
          <Button
            icon={<ReloadOutlined />}
            loading={manualRefreshing || targetsFetching || policiesFetching || eventsFetching}
            onClick={() => void refresh()}
          >
            {t('common.refresh')}
          </Button>
        }
      >
        <Row gutter={[16, 16]}>
          {targets.map((target) => {
            const percent = target.available ? ratioToPercent(target.usedRatio) : undefined;
            const sourcePaths = (target.protectedPaths.length > 0 ? target.protectedPaths : target.sourcePaths).slice(0, 3);
            return (
              <Col xs={24} md={12} xl={8} key={`${target.targetType}-${target.targetId}`}>
                <Card size="small" title={targetTitle(target)}>
                  <Space direction="vertical" style={{ width: '100%' }} size={6}>
                    <span style={{ color: '#888', wordBreak: 'break-all' }}>{targetPath(target)}</span>
                    {percent !== undefined ? (
                      <Progress percent={percent} status={percent >= 90 ? 'exception' : percent >= 80 ? 'active' : 'normal'} />
                    ) : (
                      <span>{t('system.storageProtection.capacityUnavailable')}</span>
                    )}
                    <span>{t('system.storageProtection.used')}: {formatBytes(target.usedBytes, unavailable)} / {formatBytes(target.capacityBytes, unavailable)}</span>
                    <Space size={[4, 4]} wrap>
                      <Tag color={stateColors[target.currentState]}>{t(`system.storageProtection.state.${target.currentState}`)}</Tag>
                      {target.components.map((component) => <Tag key={component}>{component}</Tag>)}
                    </Space>
                    {sourcePaths.length > 0 && (
                      <span style={{ color: '#888', fontSize: 12, wordBreak: 'break-all' }}>
                        {t('system.storageProtection.sourcePaths')}: {sourcePaths.join(', ')}
                      </span>
                    )}
                  </Space>
                </Card>
              </Col>
            );
          })}
          {targets.length === 0 && <Col span={24}><span>{unavailable}</span></Col>}
        </Row>
      </Card>

      <Card
        loading={policiesFetching}
        title={<Space><SettingOutlined />{t('system.storageProtection.policySummary')}</Space>}
        style={{ marginBottom: 16 }}
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
              {t('system.storageProtection.unifiedTarget')}
            </Descriptions.Item>
            <Descriptions.Item label={t('system.storageProtection.enabled')}>
              <Tag color={policy.enabled ? 'green' : 'default'}>
                {t(policy.enabled ? 'common.enabled' : 'common.disabled')}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('system.storageProtection.thresholds')}>
              {`${policy.warnUsedPercent}% / ${policy.recoverUsedPercent}% / ${policy.blockUsedPercent}%`}
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

      <Card title={t('system.storageProtection.audit')} size="small" style={{ marginBottom: 16 }} loading={eventsFetching}>
        {events.length > 0 ? (
          <Space direction="vertical" style={{ width: '100%' }}>
            {events.map((event) => (
              <span key={`${event.policyId}-${event.createdAt}`}>
                <Tag color={stateColors[event.newState]}>{t(`system.storageProtection.state.${event.newState}`)}</Tag>
                {formatEventReason(event, t, unavailable)}
              </span>
            ))}
          </Space>
        ) : (
          <span>{t('system.storageProtection.noEvents')}</span>
        )}
      </Card>

      <Drawer
        title={t(editingId ? 'system.storageProtection.editPolicy' : 'system.storageProtection.configurePolicy')}
        open={editorOpen}
        onClose={closeEditor}
        destroyOnClose
        width={480}
      >
        <Form<PolicyFormValues> form={form} layout="vertical" onFinish={(values) => void handleSavePolicy(values)}>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="warnUsedPercent" label={t('system.storageProtection.warnThreshold')} rules={[{ required: true }]}>
                <InputNumber min={0} max={98} addonAfter="%" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="recoverUsedPercent" label={t('system.storageProtection.recoverThreshold')} rules={[{ required: true }]}>
                <InputNumber min={0} max={99} addonAfter="%" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="blockUsedPercent" label={t('system.storageProtection.blockThreshold')} rules={[{ required: true }]}>
                <InputNumber min={1} max={100} addonAfter="%" style={{ width: '100%' }} />
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
    </>
  );
}

function CategoryConfigCard({ spec }: { spec: CardSpec }) {
  const t = useT();
  const [form] = Form.useForm<Record<string, number | boolean>>();
  const [submitting, setSubmitting] = useState(false);

  const {
    data: configs,
    isLoading,
    isFetching,
    isError,
    isSuccess,
    refetch,
  } = useSysConfigsByCategory(spec.category);
  const { mutateAsync: batchUpdate } = useBatchUpdateSysConfigs();
  const canEdit = isSuccess && !isFetching;
  const canEditRef = useRef(canEdit);
  useLayoutEffect(() => {
    canEditRef.current = canEdit;
  }, [canEdit]);

  // 把后端 sys_configs 行映射为 { key: typed value }；缺失键给安全默认。
  const initialValues = useMemo<Record<string, number | boolean>>(() => {
    const out: Record<string, number | boolean> = {};
    for (const f of spec.fields) {
      out[f.key] = f.type === 'bool' ? false : 0;
    }
    if (configs) {
      for (const cfg of configs) {
        const f = spec.fields.find((x) => x.key === cfg.key);
        if (f) {
          out[cfg.key] = decodeValue(cfg.value, f.type);
        }
      }
    }
    return out;
  }, [configs, spec.fields]);

  useEffect(() => {
    if (!isSuccess) return;
    form.setFields(
      spec.fields.map((field) => ({
        name: field.key,
        value: initialValues[field.key],
        touched: false,
      })),
    );
  }, [form, initialValues, isSuccess, spec.fields]);

  const handleSave = async () => {
    if (!canEditRef.current) {
      message.error(t('empty.loadFailed'));
      return;
    }
    try {
      const values = await form.validateFields();
      if (!canEditRef.current) {
        message.error(t('empty.loadFailed'));
        return;
      }
      const changedFields = spec.fields.filter((field) => form.isFieldTouched(field.key));
      if (changedFields.length === 0) {
        message.warning(t('sysconfig.warn.noSaveable'));
        return;
      }
      setSubmitting(true);
      await batchUpdate({
        category: spec.category,
        items: changedFields.map((f) => ({
          key: f.key,
          value: encodeValue(values[f.key], f.type),
          value_type: f.type,
        })),
      });
      message.success(t('retentionBp.save.success'));
    } catch {
      // validateFields 失败或网络错误：交给 antd 自身 UI 提示
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card
      size="small"
      title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t(spec.titleKey)}</span>}
      style={{ marginBottom: 16 }}
      extra={
        <Button size="small" type="primary" onClick={handleSave} loading={submitting} disabled={!canEdit}>
          {t('retentionBp.save')}
        </Button>
      }
    >
      <div style={{ color: '#888', fontSize: 12, marginBottom: 12 }}>{t(spec.descKey)}</div>
      {isError && (
        <Alert
          type="error"
          showIcon
          title={t('empty.loadFailed')}
          description={t('empty.loadFailedDesc')}
          action={<Button size="small" onClick={() => void refetch()}>{t('common.retry')}</Button>}
          style={{ marginBottom: 12 }}
        />
      )}
      <Spin spinning={isLoading || isFetching}>
        <Form form={form} layout="vertical" size="small">
          {spec.fields.map((f) => (
            <Form.Item
              key={f.key}
              name={f.key}
              label={t(`retentionBp.field.${f.key}`)}
              valuePropName={f.type === 'bool' ? 'checked' : 'value'}
              rules={f.type === 'bool' ? undefined : [
                {
                  type: 'number',
                  min: f.min,
                  max: f.max,
                  message: t('retentionBp.validate.range', {
                    field: t(`retentionBp.field.${f.key}`),
                    min: f.min ?? 0,
                    max: f.max ?? 0,
                  }),
                },
              ]}
              style={{ marginBottom: 12, maxWidth: 360 }}
            >
              {f.type === 'bool' ? (
                <Switch />
              ) : (
                <InputNumber min={f.min} max={f.max} step={f.step ?? 1} style={{ width: 200 }} />
              )}
            </Form.Item>
          ))}
        </Form>
      </Spin>
    </Card>
  );
}

export default function RetentionBackpressureSection() {
  return (
    <>
      <StorageProtectionSection />
      {CARDS.map((spec) => (
        <CategoryConfigCard key={spec.category} spec={spec} />
      ))}
    </>
  );
}
