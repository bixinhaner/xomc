import { useState } from 'react';
import {
  Button,
  Card,
  Form,
  message,
  Tabs,
  Space,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import BasicSettings from './BasicSettings';
import SecuritySettings from './SecuritySettings';
import DeviceSettings from './DeviceSettings';
import NotificationSettings from './NotificationSettings';
import StorageSettings from './StorageSettings';
import OmcSettings from './OmcSettings';
import NorthboundSettings from './NorthboundSettings';
import SasSettings from './SasSettings';
import LdapSettings from './LdapSettings';

// 设置子页签类型
type SettingsTab = 'basic' | 'security' | 'device' | 'notify' | 'storage' | 'omc' | 'northbound' | 'sas' | 'ldap';

// 设置子页签配置
const settingsTabs: { key: SettingsTab; label: string }[] = [
  { key: 'basic', label: '基本设置' },
  { key: 'security', label: '安全设置' },
  { key: 'device', label: '设备设置' },
  { key: 'notify', label: '通知设置' },
  { key: 'storage', label: '存储设置' },
  { key: 'omc', label: '网管设置' },
  { key: 'northbound', label: '北向接口设置' },
  { key: 'sas', label: 'SAS设置' },
  { key: 'ldap', label: 'LDAP协议' },
];

export default function SystemConfig() {
  const t = useT();
  const [activeTab, setActiveTab] = useState<SettingsTab>('basic');
  const [saving, setSaving] = useState(false);

  // 各设置模块的表单实例
  const [basicForm] = Form.useForm();
  const [securityForm] = Form.useForm();
  const [deviceForm] = Form.useForm();
  const [notifyForm] = Form.useForm();
  const [storageForm] = Form.useForm();
  const [omcForm] = Form.useForm();
  const [northboundForm] = Form.useForm();
  const [sasForm] = Form.useForm();
  const [ldapForm] = Form.useForm();

  // 获取当前设置页签对应的表单
  const getCurrentForm = () => {
    const formMap: Record<SettingsTab, ReturnType<typeof Form.useForm>[0]> = {
      basic: basicForm,
      security: securityForm,
      device: deviceForm,
      notify: notifyForm,
      storage: storageForm,
      omc: omcForm,
      northbound: northboundForm,
      sas: sasForm,
      ldap: ldapForm,
    };
    return formMap[activeTab];
  };

  // 保存当前设置
  const handleSave = () => {
    getCurrentForm().validateFields().then(() => {
      setSaving(true);
      setTimeout(() => {
        setSaving(false);
        void message.success(t('common.save'));
      }, 800);
    }).catch(() => {
      void message.error('请检查表单填写是否正确');
    });
  };

  // 渲染设置内容
  const renderSettingsContent = () => {
    switch (activeTab) {
      case 'basic':
        return <BasicSettings form={basicForm} />;
      case 'security':
        return <SecuritySettings form={securityForm} />;
      case 'device':
        return <DeviceSettings form={deviceForm} />;
      case 'notify':
        return <NotificationSettings form={notifyForm} />;
      case 'storage':
        return <StorageSettings form={storageForm} />;
      case 'omc':
        return <OmcSettings form={omcForm} />;
      case 'northbound':
        return <NorthboundSettings form={northboundForm} />;
      case 'sas':
        return <SasSettings form={sasForm} />;
      case 'ldap':
        return <LdapSettings form={ldapForm} />;
      default:
        return null;
    }
  };

  // Tabs 配置
  const tabItems = settingsTabs.map((tab) => ({
    key: tab.key,
    label: tab.label,
  }));

  return (
    <ListPageLayout title={t('nav.system.config')}>
      {/* 页签切换 - 放在 Card 外部 */}
      <Tabs
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as SettingsTab)}
        items={tabItems}
      />
      {/* 设置内容 - 放在 Card 内部 */}
      <Card bordered={false}>
        {renderSettingsContent()}
      </Card>
      {/* 底部保存按钮 */}
      <div style={{ marginTop: 16, textAlign: 'center' }}>
        <Space>
          <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSave}>
            {t('common.save')}
          </Button>
        </Space>
      </div>
    </ListPageLayout>
  );
}
