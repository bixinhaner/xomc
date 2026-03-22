import { useState } from 'react';
import { Button, Card, Collapse, Form, InputNumber, Select, Space, Switch, Typography, message } from 'antd';
import { SaveOutlined, ReloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

interface BackupPolicyValues {
  // 保留策略
  retentionDays: number;
  maxBackupCount: number;
  minBackupCount: number;
  // 自动清理
  autoCleanup: boolean;
  cleanupTime: string;
  cleanupDayOfWeek: number;
  keepLastN: number;
  // 压缩设置
  enableCompression: boolean;
  compressionLevel: number;
  compressionFormat: 'gzip' | 'bzip2' | 'lz4' | 'zstd';
  // 存储设置
  storageBackend: 'local' | 'ftp' | 'sftp' | 'nfs';
  ftpConfigId: string;
  localPath: string;
  maxStorageGB: number;
  // 加密设置
  enableEncryption: boolean;
  encryptionAlgorithm: string;
  // 告警设置
  alertOnFailure: boolean;
  alertEmail: string;
  alertThresholdPercent: number;
}

const DEFAULT_VALUES: BackupPolicyValues = {
  retentionDays: 30,
  maxBackupCount: 100,
  minBackupCount: 3,
  autoCleanup: true,
  cleanupTime: '03:00',
  cleanupDayOfWeek: 0,
  keepLastN: 5,
  enableCompression: true,
  compressionLevel: 6,
  compressionFormat: 'gzip',
  storageBackend: 'local',
  ftpConfigId: '',
  localPath: '/var/backup/omc',
  maxStorageGB: 500,
  enableEncryption: false,
  encryptionAlgorithm: 'AES-256',
  alertOnFailure: true,
  alertEmail: 'admin@example.com',
  alertThresholdPercent: 80,
};

export default function BackupPolicy() {
  const t = useT();
  const [form] = Form.useForm<BackupPolicyValues>();
  const [saving, setSaving] = useState(false);
  const [storageBackend, setStorageBackend] = useState<string>('local');
  const [autoCleanup, setAutoCleanup] = useState(true);
  const [enableCompression, setEnableCompression] = useState(true);
  const [enableEncryption, setEnableEncryption] = useState(false);

  const handleSave = () => {
    form.validateFields().then((vals) => {
      setSaving(true);
      setTimeout(() => {
        setSaving(false);
        void message.success(t('common.save'));
        console.log('backup policy:', vals);
      }, 700);
    }).catch(() => undefined);
  };

  const handleReset = () => {
    form.setFieldsValue(DEFAULT_VALUES);
    setStorageBackend('local');
    setAutoCleanup(true);
    setEnableCompression(true);
    setEnableEncryption(false);
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
      label: t('backup.policy.retention'),
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
      label: t('backup.policy.autoCleanup'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.enableAutoCleanup')} name="autoCleanup" valuePropName="checked">
            <Switch
              checkedChildren={t('common.on')}
              unCheckedChildren={t('common.off')}
              onChange={(val) => setAutoCleanup(val)}
            />
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
      label: t('backup.policy.compression'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.enableCompression')} name="enableCompression" valuePropName="checked">
            <Switch
              checkedChildren={t('common.on')}
              unCheckedChildren={t('common.off')}
              onChange={(val) => setEnableCompression(val)}
            />
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
      label: t('backup.policy.storage'),
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
              onChange={(val) => setStorageBackend(val as string)}
            />
          </Form.Item>
          {storageBackend === 'local' ? (
            <Form.Item label={t('backup.policy.localPath')} name="localPath" rules={[{ required: true }]}>
              <input
                style={{ width: '100%', padding: '4px 8px', border: '1px solid #d9d9d9', borderRadius: 4, fontSize: 14 }}
                placeholder="/var/backup/omc"
              />
            </Form.Item>
          ) : (
            <Form.Item label={t('backup.policy.ftpConfig')} name="ftpConfigId" rules={[{ required: true }]}>
              <Select
                placeholder={t('backup.policy.selectFtpConfig')}
                options={[
                  { label: t('backup.policy.primaryFtp'), value: '1' },
                  { label: t('backup.policy.remoteSftp'), value: '2' },
                ]}
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
      label: t('backup.policy.encryption'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.enableEncryption')} name="enableEncryption" valuePropName="checked">
            <Switch
              checkedChildren={t('common.on')}
              unCheckedChildren={t('common.off')}
              onChange={(val) => setEnableEncryption(val)}
            />
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
      ),
    },
    {
      key: 'alert',
      label: t('backup.policy.alert'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('backup.policy.failureAlert')} name="alertOnFailure" valuePropName="checked">
            <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
          </Form.Item>
          <Form.Item label={t('backup.policy.alertEmail')} name="alertEmail">
            <input
              style={{ width: '100%', padding: '4px 8px', border: '1px solid #d9d9d9', borderRadius: 4, fontSize: 14 }}
              placeholder="admin@example.com"
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
          <Button type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}>
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Card>
        <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 16 }}>
          {t('backup.policy.hint')}
        </Typography.Text>
        <Form
          form={form}
          layout="vertical"
          initialValues={DEFAULT_VALUES}
          style={{ maxWidth: 960 }}
        >
          <Collapse
            defaultActiveKey={['retention', 'cleanup', 'compression', 'storage', 'encryption', 'alert']}
            items={collapseItems}
            style={{ background: 'transparent' }}
          />
        </Form>
      </Card>
    </ListPageLayout>
  );
}
