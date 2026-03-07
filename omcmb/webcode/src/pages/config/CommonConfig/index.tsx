import { useState } from 'react';
import { Button, Card, Collapse, Form, Input, InputNumber, Select, Space, Switch, message } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

interface CommonConfigValues {
  // 系统参数
  systemName: string;
  systemVersion: string;
  maxOnlineDevices: number;
  sessionTimeout: number;
  logLevel: string;
  logRetentionDays: number;
  // 网络参数
  managementIp: string;
  managementPort: number;
  ntpServer: string;
  dnsServer: string;
  snmpCommunity: string;
  snmpVersion: string;
  // 安全参数
  passwordMinLength: number;
  passwordExpireDays: number;
  maxLoginAttempts: number;
  enableTwoFactor: boolean;
  enableSslOnly: boolean;
  certExpireWarningDays: number;
}

const DEFAULT_VALUES: CommonConfigValues = {
  systemName: 'OMC网络管理系统',
  systemVersion: '3.2.1',
  maxOnlineDevices: 10000,
  sessionTimeout: 30,
  logLevel: 'INFO',
  logRetentionDays: 90,
  managementIp: '10.0.0.1',
  managementPort: 8443,
  ntpServer: '10.0.0.253',
  dnsServer: '10.0.0.252',
  snmpCommunity: 'public',
  snmpVersion: 'v2c',
  passwordMinLength: 8,
  passwordExpireDays: 90,
  maxLoginAttempts: 5,
  enableTwoFactor: false,
  enableSslOnly: true,
  certExpireWarningDays: 30,
};

export default function CommonConfig() {
  const t = useT();
  const [form] = Form.useForm<CommonConfigValues>();
  const [saving, setSaving] = useState(false);

  const handleSave = () => {
    form.validateFields().then((vals) => {
      setSaving(true);
      setTimeout(() => {
        setSaving(false);
        void message.success(t('common.save'));
        console.log('saved config:', vals);
      }, 800);
    }).catch(() => undefined);
  };

  const handleReset = () => {
    form.setFieldsValue(DEFAULT_VALUES);
    void message.info(t('common.reset'));
  };

  const collapseItems = [
    {
      key: 'system',
      label: t('table.type'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('table.name')} name="systemName" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.version')} name="systemVersion">
            <Input disabled />
          </Form.Item>
          <Form.Item label={t('table.total')} name="maxOnlineDevices" rules={[{ required: true }]}>
            <InputNumber min={100} max={100000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('table.time')} name="sessionTimeout" rules={[{ required: true }]}>
            <InputNumber min={5} max={480} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('table.status')} name="logLevel" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'DEBUG', value: 'DEBUG' },
                { label: 'INFO', value: 'INFO' },
                { label: 'WARN', value: 'WARN' },
                { label: 'ERROR', value: 'ERROR' },
              ]}
            />
          </Form.Item>
          <Form.Item label={t('table.time')} name="logRetentionDays" rules={[{ required: true }]}>
            <InputNumber min={7} max={365} style={{ width: '100%' }} />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'network',
      label: t('device.ipAddress'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('device.ipAddress')} name="managementIp" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.index')} name="managementPort" rules={[{ required: true }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="NTP" name="ntpServer" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label="DNS" name="dnsServer">
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label="SNMP Community" name="snmpCommunity" rules={[{ required: true }]}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.version')} name="snmpVersion" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'v1', value: 'v1' },
                { label: 'v2c', value: 'v2c' },
                { label: 'v3', value: 'v3' },
              ]}
            />
          </Form.Item>
        </div>
      ),
    },
    {
      key: 'security',
      label: t('config.readonly'),
      children: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
          <Form.Item label={t('config.paramValue')} name="passwordMinLength" rules={[{ required: true }]}>
            <InputNumber min={6} max={32} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('table.time')} name="passwordExpireDays" rules={[{ required: true }]}>
            <InputNumber min={0} max={365} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('table.total')} name="maxLoginAttempts" rules={[{ required: true }]}>
            <InputNumber min={3} max={10} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('table.time')} name="certExpireWarningDays" rules={[{ required: true }]}>
            <InputNumber min={7} max={90} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('common.enable')} name="enableTwoFactor" valuePropName="checked">
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>
          <Form.Item label="SSL" name="enableSslOnly" valuePropName="checked">
            <Switch checkedChildren={t('common.enable')} unCheckedChildren={t('common.disable')} />
          </Form.Item>
        </div>
      ),
    },
  ];

  return (
    <ListPageLayout
      title={t('nav.config.common')}
      extra={
        <Space>
          <Button onClick={handleReset}>{t('common.reset')}</Button>
          <Button type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}>
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Card>
        <Form
          form={form}
          layout="vertical"
          initialValues={DEFAULT_VALUES}
          style={{ maxWidth: 1000 }}
        >
          <Collapse
            defaultActiveKey={['system', 'network', 'security']}
            items={collapseItems}
            style={{ background: 'transparent' }}
          />
        </Form>
      </Card>
    </ListPageLayout>
  );
}
