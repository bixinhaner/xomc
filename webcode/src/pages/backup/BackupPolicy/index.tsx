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
    { label: '周日', value: 0 },
    { label: '周一', value: 1 },
    { label: '周二', value: 2 },
    { label: '周三', value: 3 },
    { label: '周四', value: 4 },
    { label: '周五', value: 5 },
    { label: '周六', value: 6 },
  ];

  const collapseItems = [
    {
      key: 'retention',
      label: '保留策略',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="保留天数" name="retentionDays" rules={[{ required: true }]}>
            <Select
              options={[
                { label: '7天', value: 7 },
                { label: '14天', value: 14 },
                { label: '30天', value: 30 },
                { label: '60天', value: 60 },
                { label: '90天', value: 90 },
                { label: '180天', value: 180 },
                { label: '365天', value: 365 },
              ]}
            />
          </Form.Item>
          <Form.Item label="最大备份数量" name="maxBackupCount" rules={[{ required: true }]}>
            <InputNumber min={1} max={10000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="最少保留备份数" name="minBackupCount" rules={[{ required: true }]}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'cleanup',
      label: '自动清理',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="启用自动清理" name="autoCleanup" valuePropName="checked">
            <Switch
              checkedChildren="开"
              unCheckedChildren="关"
              onChange={(val) => setAutoCleanup(val)}
            />
          </Form.Item>
          {autoCleanup && (
            <>
              <Form.Item label="清理执行时间" name="cleanupTime" rules={[{ required: true }]}>
                <Select
                  options={['00:00', '01:00', '02:00', '03:00', '04:00', '05:00'].map((t) => ({ label: t, value: t }))}
                />
              </Form.Item>
              <Form.Item label="清理执行日期" name="cleanupDayOfWeek">
                <Select options={[{ label: '每天', value: -1 }, ...WEEKDAY_OPTIONS]} />
              </Form.Item>
              <Form.Item label="至少保留最近N份" name="keepLastN" rules={[{ required: true }]}>
                <InputNumber min={1} max={50} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'compression',
      label: '压缩设置',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="启用压缩" name="enableCompression" valuePropName="checked">
            <Switch
              checkedChildren="开"
              unCheckedChildren="关"
              onChange={(val) => setEnableCompression(val)}
            />
          </Form.Item>
          {enableCompression && (
            <>
              <Form.Item label="压缩格式" name="compressionFormat">
                <Select
                  options={[
                    { label: 'GZIP (标准)', value: 'gzip' },
                    { label: 'BZIP2 (高压缩)', value: 'bzip2' },
                    { label: 'LZ4 (快速)', value: 'lz4' },
                    { label: 'ZSTD (均衡)', value: 'zstd' },
                  ]}
                />
              </Form.Item>
              <Form.Item label="压缩级别 (1-9)" name="compressionLevel">
                <InputNumber min={1} max={9} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'storage',
      label: '存储设置',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="存储后端" name="storageBackend" rules={[{ required: true }]}>
            <Select
              options={[
                { label: '本地存储', value: 'local' },
                { label: 'FTP服务器', value: 'ftp' },
                { label: 'SFTP服务器', value: 'sftp' },
                { label: 'NFS共享', value: 'nfs' },
              ]}
              onChange={(val) => setStorageBackend(val as string)}
            />
          </Form.Item>
          {storageBackend === 'local' ? (
            <Form.Item label="本地路径" name="localPath" rules={[{ required: true }]}>
              <input
                style={{ width: '100%', padding: '4px 8px', border: '1px solid #d9d9d9', borderRadius: 4, fontSize: 14 }}
                placeholder="/var/backup/omc"
              />
            </Form.Item>
          ) : (
            <Form.Item label="FTP配置" name="ftpConfigId" rules={[{ required: true }]}>
              <Select
                placeholder="请选择FTP配置"
                options={[
                  { label: '主备份FTP服务器', value: '1' },
                  { label: '异地备份SFTP', value: '2' },
                ]}
              />
            </Form.Item>
          )}
          <Form.Item label="最大存储空间 (GB)" name="maxStorageGB" rules={[{ required: true }]}>
            <InputNumber min={1} max={100000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="存储空间告警阈值 (%)" name="alertThresholdPercent" rules={[{ required: true }]}>
            <InputNumber min={50} max={95} style={{ width: '100%' }} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'encryption',
      label: '加密设置',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="启用加密" name="enableEncryption" valuePropName="checked">
            <Switch
              checkedChildren="开"
              unCheckedChildren="关"
              onChange={(val) => setEnableEncryption(val)}
            />
          </Form.Item>
          {enableEncryption && (
            <Form.Item label="加密算法" name="encryptionAlgorithm">
              <Select
                options={[
                  { label: 'AES-256-GCM (推荐)', value: 'AES-256-GCM' },
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
      label: '告警设置',
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label="失败告警" name="alertOnFailure" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
          <Form.Item label="告警邮箱" name="alertEmail">
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
          备份策略配置影响所有备份任务的默认行为。修改后需保存才能生效。
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
