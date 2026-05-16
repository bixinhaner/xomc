// T-0071 / R-102 followup: BackupPolicy 接入真后端
//
// 历史: 此页 305 行纯本地 form + console.log 假保存（CLAUDE.md 禁止 production
// console.log），后端无 endpoint。
//
// 现在: 消费 useBackupPolicy + useUpdateBackupPolicy 真实接到 GET/PUT /backup/policy。
// 删除 `<input>` 直接 DOM 标签换为 AntD Input；删除 console.log；删除 setTimeout 假延迟。
//
// ⚠ Enforcement boundary（PRD §2）: 本页面下 4 个 Collapse panel（自动清理 / 压缩 /
// 加密 / 告警）头部加 "尚未生效" Tag — 字段持久化到 DB，但 backup executor 还未
// 集成对应能力。executor 集成是 follow-up（T-0073 / T-0074 / T-0075）。

import React, { useEffect } from 'react';
import {
  Alert,
  Button,
  Card,
  Collapse,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import { InfoCircleOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useBackupPolicy,
  useUpdateBackupPolicy,
  useFTPConfigs,
} from '@core/hooks/api/useBackup';
import { DEFAULT_BACKUP_POLICY } from '@core/mock/data/backup';
import type { BackupPolicy, FTPConfig } from '@core/mock/data/backup';
import { useT } from '@/hooks/useT';

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message);
  }
  return '';
}

// PersistedOnlyTag — UI marker added to Collapse panel headers for categories
// that the executor doesn't yet enforce (per T-0071 PRD §2). The encryption
// panel passes `severity="warning"` to make the security false-trust risk
// visually distinct from the neutral cleanup/compression panels.
function PersistedOnlyTag({ severity = 'info' }: { severity?: 'info' | 'warning' }): React.JSX.Element {
  const t = useT();
  const color = severity === 'warning' ? 'orange' : 'default';
  return (
    <Tooltip title={t('backup.policy.persistedNotEnforcedTooltip')}>
      <Tag color={color} icon={<InfoCircleOutlined />} style={{ marginLeft: 8 }}>
        {t('backup.policy.persistedNotEnforced')}
      </Tag>
    </Tooltip>
  );
}

// EncryptionStatusTag — 3-state badge replacing PersistedOnlyTag on the
// encryption Collapse header (T-0088). State derives FE-only from form
// values; KEK availability is enforced by backend at PUT time (returns 400
// with a clear message when AES-256-GCM is enabled but no key is configured).
//
//   active            — green   — EnableEncryption=true && algorithm=AES-256-GCM
//   notYetEffective   — orange  — EnableEncryption=true && other algorithm
//                                 (CBC / ChaCha20 are persisted but executor
//                                 only acts on GCM until T-0085 lands)
//   disabled          — default — EnableEncryption=false
function EncryptionStatusTag({
  enabled,
  algorithm,
}: {
  enabled: boolean;
  algorithm: string;
}): React.JSX.Element {
  const t = useT();
  if (!enabled) {
    return (
      <Tag color="default" icon={<InfoCircleOutlined />} style={{ marginLeft: 8 }}>
        {t('backup.policy.encryptionStatus.disabled')}
      </Tag>
    );
  }
  if (algorithm === 'AES-256-GCM') {
    return (
      <Tooltip title={t('backup.policy.encryptionStatus.activeHelp')}>
        <Tag color="green" icon={<InfoCircleOutlined />} style={{ marginLeft: 8 }}>
          {t('backup.policy.encryptionStatus.active')}
        </Tag>
      </Tooltip>
    );
  }
  return (
    <Tooltip title={t('backup.policy.persistedNotEnforcedTooltip')}>
      <Tag color="orange" icon={<InfoCircleOutlined />} style={{ marginLeft: 8 }}>
        {t('backup.policy.encryptionStatus.notYetEffective')}
      </Tag>
    </Tooltip>
  );
}

export default function BackupPolicyPage(): React.JSX.Element {
  const t = useT();
  const [form] = Form.useForm<BackupPolicy>();
  const { data: policy, isLoading, isError, refetch } = useBackupPolicy();
  const { data: ftpData } = useFTPConfigs({ page: 1, pageSize: 100 });
  const updatePolicy = useUpdateBackupPolicy();

  // FTP options come from real backend data (useFTPConfigs) — not hardcoded.
  // Without this, ftp_config_id would be a fake string ID and the backend
  // FK constraint on ftp_configs(id) would reject the upsert. (Review HIGH-3.)
  const ftpOptions = (ftpData?.items ?? []).map((cfg: FTPConfig) => ({
    label: `${cfg.configName} (${cfg.host}:${cfg.port})`,
    value: cfg.id,
  }));

  // Conditional sub-fields read live from the form via useWatch — keeps a
  // single source of truth (the form) and avoids "setState in useEffect"
  // (react-hooks/set-state-in-effect lint rule).
  const storageBackend = Form.useWatch('storageBackend', form) ?? 'local';
  const autoCleanup = Form.useWatch('autoCleanup', form) ?? true;
  const enableCompression = Form.useWatch('enableCompression', form) ?? true;
  const enableEncryption = Form.useWatch('enableEncryption', form) ?? false;
  // T-0088: encryption_ready FE-only derivation. Backend enforces KEK
  // availability at PUT time; UI just reflects current form values.
  const encryptionAlgorithm = Form.useWatch('encryptionAlgorithm', form) ?? 'AES-256-GCM';

  // GET on mount: setFieldsValue once data arrives. setFieldsValue is OK in
  // useEffect because it mutates the form (ref-stable) — not a setState.
  useEffect(() => {
    if (!policy) return;
    form.setFieldsValue(policy);
  }, [policy, form]);

  const handleSave = async (): Promise<void> => {
    let vals: BackupPolicy;
    try {
      vals = await form.validateFields();
    } catch {
      return;
    }
    updatePolicy.mutate(vals, {
      onSuccess: () => {
        void message.success(t('backup.policy.saveSuccess'));
      },
      onError: (e: unknown) => {
        void message.error(t('backup.policy.saveFailed', { error: getErrMsg(e) }));
      },
    });
  };

  const handleReset = (): void => {
    form.setFieldsValue(DEFAULT_BACKUP_POLICY);
    void message.info(t('common.reset'));
  };

  const WEEKDAY_OPTIONS = [
    { label: t('backup.policy.weekday.sun'), value: 0 },
    { label: t('backup.policy.weekday.mon'), value: 1 },
    { label: t('backup.policy.weekday.tue'), value: 2 },
    { label: t('backup.policy.weekday.wed'), value: 3 },
    { label: t('backup.policy.weekday.thu'), value: 4 },
    { label: t('backup.policy.weekday.fri'), value: 5 },
    { label: t('backup.policy.weekday.sat'), value: 6 },
  ];

  const collapseItems = [
    {
      key: 'retention',
      label: <span>{t('backup.policy.retention')}</span>,
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.retentionDays')} name="retentionDays" rules={[{ required: true }]}>
            <Select
              options={[
                { label: `7 ${t('backup.policy.days')}`, value: 7 },
                { label: `14 ${t('backup.policy.days')}`, value: 14 },
                { label: `30 ${t('backup.policy.days')}`, value: 30 },
                { label: `60 ${t('backup.policy.days')}`, value: 60 },
                { label: `90 ${t('backup.policy.days')}`, value: 90 },
                { label: `180 ${t('backup.policy.days')}`, value: 180 },
                { label: `365 ${t('backup.policy.days')}`, value: 365 },
              ]}
            />
          </Form.Item>
          <Form.Item label={t('backup.policy.maxBackupCount')} name="maxBackupCount" rules={[{ required: true }]}>
            <InputNumber min={1} max={10000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('backup.policy.minBackupCount')} name="minBackupCount" rules={[{ required: true }]}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'cleanup',
      label: (
        <span>
          {t('backup.policy.autoCleanup')}
          <PersistedOnlyTag />
        </span>
      ),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.enableAutoCleanup')} name="autoCleanup" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          {autoCleanup && (
            <>
              <Form.Item label={t('backup.policy.cleanupTime')} name="cleanupTime" rules={[{ required: true }]}>
                <Select
                  options={['00:00', '01:00', '02:00', '03:00', '04:00', '05:00'].map((v) => ({ label: v, value: v }))}
                />
              </Form.Item>
              <Form.Item label={t('backup.policy.cleanupDay')} name="cleanupDayOfWeek">
                <Select options={[{ label: t('backup.policy.everyday'), value: -1 }, ...WEEKDAY_OPTIONS]} />
              </Form.Item>
              <Form.Item label={t('backup.policy.keepLastN')} name="keepLastN" rules={[{ required: true }]}>
                <InputNumber min={1} max={50} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'compression',
      label: (
        <span>
          {t('backup.policy.compression')}
          <PersistedOnlyTag />
        </span>
      ),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.enableCompression')} name="enableCompression" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          {enableCompression && (
            <>
              <Form.Item label={t('backup.policy.compressionFormat')} name="compressionFormat">
                <Select
                  options={[
                    { label: t('backup.policy.gzipStandard'), value: 'gzip' },
                    { label: t('backup.policy.bzip2High'), value: 'bzip2' },
                    { label: t('backup.policy.lz4Fast'), value: 'lz4' },
                    { label: t('backup.policy.zstdBalanced'), value: 'zstd' },
                  ]}
                />
              </Form.Item>
              <Form.Item label={t('backup.policy.compressionLevel')} name="compressionLevel">
                <InputNumber min={1} max={9} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'storage',
      label: <span>{t('backup.policy.storage')}</span>,
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.storageBackend')} name="storageBackend" rules={[{ required: true }]}>
            <Select
              options={[
                { label: t('backup.policy.localStorage'), value: 'local' },
                { label: t('backup.policy.ftpServer'), value: 'ftp' },
                { label: t('backup.policy.sftpServer'), value: 'sftp' },
                { label: t('backup.policy.nfsShare'), value: 'nfs' },
              ]}
            />
          </Form.Item>
          {storageBackend === 'local' ? (
            <Form.Item label={t('backup.policy.localPath')} name="localPath" rules={[{ required: true }]}>
              <Input placeholder="/var/backup/omc" />
            </Form.Item>
          ) : (
            <Form.Item label={t('backup.policy.ftpConfig')} name="ftpConfigId" rules={[{ required: true }]}>
              <Select
                placeholder={t('backup.policy.selectFtpConfig')}
                options={ftpOptions}
                showSearch
                optionFilterProp="label"
                notFoundContent={t('common.empty')}
              />
            </Form.Item>
          )}
          <Form.Item label={t('backup.policy.maxStorageGB')} name="maxStorageGB" rules={[{ required: true }]}>
            <InputNumber min={1} max={100000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('backup.policy.alertThreshold')} name="alertThresholdPercent" rules={[{ required: true }]}>
            <InputNumber min={50} max={95} style={{ width: '100%' }} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'encryption',
      label: (
        <span>
          {t('backup.policy.encryption')}
          <EncryptionStatusTag enabled={enableEncryption} algorithm={encryptionAlgorithm} />
        </span>
      ),
      children: (
        <>
          <Alert
            type="warning"
            showIcon
            message={t('backup.policy.encryptionSecurityWarning')}
            style={{ marginBottom: 16 }}
          />
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
            <Form.Item label={t('backup.policy.enableEncryption')} name="enableEncryption" valuePropName="checked">
              <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
            </Form.Item>
            {enableEncryption && (
              <Form.Item label={t('backup.policy.encryptionAlgorithm')} name="encryptionAlgorithm">
                <Select
                  options={[
                    { label: 'AES-256-GCM', value: 'AES-256-GCM' },
                    { label: 'AES-256-CBC', value: 'AES-256-CBC' },
                    { label: 'ChaCha20-Poly1305', value: 'ChaCha20-Poly1305' },
                  ]}
                />
              </Form.Item>
            )}
          </div>
        </>
      ),
    },
    {
      key: 'alert',
      label: (
        <span>
          {t('backup.policy.alert')}
          <PersistedOnlyTag />
        </span>
      ),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.failureAlert')} name="alertOnFailure" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('backup.policy.alertEmail')} name="alertEmail">
            <Input placeholder="admin@example.com" />
          </Form.Item>
          {/* T-0088: alert_severity policy-driven (T-0084 schema). Single
              field controls both backup_task_failed and
              backup_storage_threshold_exceeded alarm severity. */}
          <Form.Item
            label={
              <Tooltip title={t('backup.policy.alertSeverityHelp')}>
                <span>
                  {t('backup.policy.alertSeverity')} <InfoCircleOutlined />
                </span>
              </Tooltip>
            }
            name="alertSeverity"
            rules={[{ required: true }]}
          >
            <Select
              options={[
                { label: t('backup.policy.alertSeverity.warning'), value: 'warning' },
                { label: t('backup.policy.alertSeverity.major'), value: 'major' },
                { label: t('backup.policy.alertSeverity.critical'), value: 'critical' },
              ]}
            />
          </Form.Item>
        </div>
      ),
    },
  ];

  return (
    <ListPageLayout
      title={t('nav.backup.policy')}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>{t('common.reset')}</Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            onClick={() => void handleSave()}
            loading={updatePolicy.isPending}
          >
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Card>
        {isError && (
          <Alert
            type="error"
            showIcon
            message={t('backup.policy.loadFailed')}
            action={<Button onClick={() => void refetch()}>{t('common.retry')}</Button>}
            style={{ marginBottom: 16 }}
          />
        )}
        <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 16 }}>
          {t('backup.policy.hint')}
        </Typography.Text>
        <Spin spinning={isLoading}>
          <Form
            form={form}
            layout="vertical"
            initialValues={DEFAULT_BACKUP_POLICY}
            style={{ maxWidth: 960 }}
          >
            <Collapse
              defaultActiveKey={['retention', 'cleanup', 'compression', 'storage', 'encryption', 'alert']}
              items={collapseItems}
              style={{ background: 'transparent' }}
            />
          </Form>
        </Spin>
      </Card>
    </ListPageLayout>
  );
}
