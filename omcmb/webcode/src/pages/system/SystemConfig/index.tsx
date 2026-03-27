import { useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Switch,
  Select,
  Divider,
  message,
  Tabs,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import LayoutPicker from './LayoutPicker';

interface SystemConfigValues {
  systemName: string;
  systemLogo: string;
  sessionTimeout: number;
  maxLoginAttempts: number;
  lockDuration: number;
  passwordMinLength: number;
  passwordComplexity: boolean;
  passwordExpiry: number;
  enableTwoFactor: boolean;
  smtpHost: string;
  smtpPort: number;
  smtpUser: string;
  smtpPassword: string;
  smtpSsl: boolean;
  senderEmail: string;
  dataRetentionDays: number;
  autoCleanup: boolean;
  backupPath: string;
  maxFileSize: number;
  allowedFileTypes: string[];
}

const defaultValues: SystemConfigValues = {
  systemName: 'OMC 网络管理平台',
  systemLogo: '',
  sessionTimeout: 30,
  maxLoginAttempts: 5,
  lockDuration: 30,
  passwordMinLength: 8,
  passwordComplexity: true,
  passwordExpiry: 90,
  enableTwoFactor: false,
  smtpHost: 'smtp.example.com',
  smtpPort: 587,
  smtpUser: 'noreply@example.com',
  smtpPassword: '',
  smtpSsl: true,
  senderEmail: 'omc-noreply@example.com',
  dataRetentionDays: 365,
  autoCleanup: true,
  backupPath: '/data/backup',
  maxFileSize: 1024,
  allowedFileTypes: ['xml', 'cfg', 'json', 'tar.gz', 'zip'],
};

export default function SystemConfig() {
  const t = useT();
  const [basicForm] = Form.useForm<SystemConfigValues>();
  const [securityForm] = Form.useForm<SystemConfigValues>();
  const [notifyForm] = Form.useForm<SystemConfigValues>();
  const [storageForm] = Form.useForm<SystemConfigValues>();
  const [saving, setSaving] = useState(false);

  const handleSave = (form: ReturnType<typeof Form.useForm>[0], section: string) => {
    form.validateFields().then(() => {
      setSaving(true);
      setTimeout(() => {
        setSaving(false);
        void message.success(t('common.save'));
      }, 800);
    });
  };

  return (
    <ListPageLayout title={t('nav.system.config')}>
      <Tabs
        items={[
          {
            key: 'layout',
            label: t('layout.title'),
            children: (
              <Card>
                <LayoutPicker />
              </Card>
            ),
          },
          {
            key: 'basic',
            label: t('nav.system.config'),
            children: (
              <Card
                extra={
                  <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => handleSave(basicForm, t('common.save'))}>
                    {t('common.save')}
                  </Button>
                }
              >
                <Form
                  form={basicForm}
                  layout="vertical"
                  initialValues={defaultValues}
                  style={{ maxWidth: 600 }}
                >
                  <Divider orientation="left" plain>{t('nav.system.config')}</Divider>
                  <Form.Item name="systemName" label={t('table.name')} rules={[{ required: true }]}>
                    <Input placeholder={t('table.name')} />
                  </Form.Item>
                  <Form.Item name="systemLogo" label="Logo URL">
                    <Input placeholder="Logo URL" />
                  </Form.Item>
                  <Divider orientation="left" plain>{t('nav.system.config')}</Divider>
                  <Form.Item name="sessionTimeout" label={t('perf.timeRange')} rules={[{ required: true }]}>
                    <InputNumber min={5} max={480} style={{ width: '100%' }} addonAfter="min" />
                  </Form.Item>
                </Form>
              </Card>
            ),
          },
          {
            key: 'security',
            label: t('nav.system.config'),
            children: (
              <Card
                extra={
                  <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => handleSave(securityForm, t('common.save'))}>
                    {t('common.save')}
                  </Button>
                }
              >
                <Form
                  form={securityForm}
                  layout="vertical"
                  initialValues={defaultValues}
                  style={{ maxWidth: 600 }}
                >
                  <Form.Item name="maxLoginAttempts" label={t('nav.system.config')} rules={[{ required: true }]}>
                    <InputNumber min={3} max={20} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="lockDuration" label={t('perf.timeRange')}>
                    <InputNumber min={1} max={1440} style={{ width: '100%' }} addonAfter="min" />
                  </Form.Item>
                  <Form.Item name="enableTwoFactor" label={t('common.enable')} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="passwordMinLength" label={t('user.password')} rules={[{ required: true }]}>
                    <InputNumber min={6} max={32} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="passwordComplexity" label={t('user.password')} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="passwordExpiry" label={t('perf.timeRange')}>
                    <InputNumber min={0} max={365} style={{ width: '100%' }} addonAfter="days" />
                  </Form.Item>
                </Form>
              </Card>
            ),
          },
          {
            key: 'notify',
            label: t('nav.system.notifications'),
            children: (
              <Card
                extra={
                  <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => handleSave(notifyForm, t('common.save'))}>
                    {t('common.save')}
                  </Button>
                }
              >
                <Form
                  form={notifyForm}
                  layout="vertical"
                  initialValues={defaultValues}
                  style={{ maxWidth: 600 }}
                >
                  <Form.Item name="smtpHost" label="SMTP Host" rules={[{ required: true }]}>
                    <Input placeholder="smtp.example.com" />
                  </Form.Item>
                  <Form.Item name="smtpPort" label="SMTP Port" rules={[{ required: true }]}>
                    <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="smtpUser" label={t('user.username')} rules={[{ required: true }]}>
                    <Input />
                  </Form.Item>
                  <Form.Item name="smtpPassword" label={t('user.password')}>
                    <Input.Password />
                  </Form.Item>
                  <Form.Item name="smtpSsl" label="SSL/TLS" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="senderEmail" label={t('user.email')} rules={[{ type: 'email' }]}>
                    <Input placeholder="noreply@example.com" />
                  </Form.Item>
                </Form>
              </Card>
            ),
          },
          {
            key: 'storage',
            label: t('nav.system.config'),
            children: (
              <Card
                extra={
                  <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => handleSave(storageForm, t('common.save'))}>
                    {t('common.save')}
                  </Button>
                }
              >
                <Form
                  form={storageForm}
                  layout="vertical"
                  initialValues={defaultValues}
                  style={{ maxWidth: 600 }}
                >
                  <Form.Item name="dataRetentionDays" label={t('perf.timeRange')} rules={[{ required: true }]}>
                    <InputNumber min={30} max={3650} style={{ width: '100%' }} addonAfter="days" />
                  </Form.Item>
                  <Form.Item name="autoCleanup" label={t('common.enable')} valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="backupPath" label={t('table.description')} rules={[{ required: true }]}>
                    <Input placeholder="/data/backup" />
                  </Form.Item>
                  <Form.Item name="maxFileSize" label={t('table.total')} rules={[{ required: true }]}>
                    <InputNumber min={1} max={10240} style={{ width: '100%' }} addonAfter="MB" />
                  </Form.Item>
                  <Form.Item name="allowedFileTypes" label={t('table.type')}>
                    <Select
                      mode="tags"
                      placeholder={t('common.pleaseSelect')}
                      options={[
                        { label: 'xml', value: 'xml' },
                        { label: 'cfg', value: 'cfg' },
                        { label: 'json', value: 'json' },
                        { label: 'tar.gz', value: 'tar.gz' },
                        { label: 'zip', value: 'zip' },
                        { label: 'bin', value: 'bin' },
                      ]}
                    />
                  </Form.Item>
                </Form>
              </Card>
            ),
          },
        ]}
      />
    </ListPageLayout>
  );
}
