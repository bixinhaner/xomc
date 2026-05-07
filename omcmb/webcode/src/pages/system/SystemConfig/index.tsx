import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Form,
  message,
  Tabs,
  Space,
  Spin,
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
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import type {
  BatchUpdateSysConfigItem,
  SysConfigItem,
  SysConfigValueType,
} from '@core/types/system';

// 设置子页签类型（v1.0：移除 sas / ldap，参 omgo/docs/prd/system/config.md）
type SettingsTab = 'basic' | 'security' | 'device' | 'notify' | 'storage' | 'omc' | 'northbound';

// 设置子页签配置
const settingsTabs: { key: SettingsTab; labelKey: string }[] = [
  { key: 'basic', labelKey: 'system.config.basic' },
  { key: 'security', labelKey: 'system.config.security' },
  { key: 'device', labelKey: 'system.config.device' },
  { key: 'notify', labelKey: 'system.config.notify' },
  { key: 'storage', labelKey: 'system.config.storage' },
  { key: 'omc', labelKey: 'system.config.omc' },
  { key: 'northbound', labelKey: 'system.config.northbound' },
];

// ----- value <-> form value 编解码 -----
// sys_configs.value 列是 TEXT；DDL CHECK value_type IN ('string','int','float','bool','json')。
// 表单各控件期望强类型（Switch=boolean, InputNumber=number, Input/Select=string），
// 这里做 string ↔ typed 双向桥。

function decodeValue(raw: string, type: SysConfigValueType | undefined): unknown {
  switch (type) {
    case 'bool':
      return raw === 'true' || raw === '1';
    case 'int': {
      const n = parseInt(raw, 10);
      return Number.isFinite(n) ? n : 0;
    }
    case 'float': {
      const n = parseFloat(raw);
      return Number.isFinite(n) ? n : 0;
    }
    case 'json':
      try {
        return JSON.parse(raw) as unknown;
      } catch {
        return raw;
      }
    case 'string':
    default:
      return raw;
  }
}

function encodeValue(v: unknown): { value: string; valueType: SysConfigValueType } {
  if (v === null || v === undefined) return { value: '', valueType: 'string' };
  if (typeof v === 'boolean') return { value: v ? 'true' : 'false', valueType: 'bool' };
  if (typeof v === 'number') {
    return { value: String(v), valueType: Number.isInteger(v) ? 'int' : 'float' };
  }
  if (typeof v === 'object') {
    try {
      return { value: JSON.stringify(v), valueType: 'json' };
    } catch {
      return { value: '', valueType: 'string' };
    }
  }
  return { value: String(v), valueType: 'string' };
}

export default function SystemConfig() {
  const t = useT();
  const [activeTab, setActiveTab] = useState<SettingsTab>('basic');

  // 各设置模块的表单实例
  const [basicForm] = Form.useForm();
  const [securityForm] = Form.useForm();
  const [deviceForm] = Form.useForm();
  const [notifyForm] = Form.useForm();
  const [storageForm] = Form.useForm();
  const [omcForm] = Form.useForm();
  const [northboundForm] = Form.useForm();

  // tab → form 映射（稳定引用，因为每个 form 都来自 useForm()）
  const formMap = useMemo<Record<SettingsTab, ReturnType<typeof Form.useForm>[0]>>(
    () => ({
      basic: basicForm,
      security: securityForm,
      device: deviceForm,
      notify: notifyForm,
      storage: storageForm,
      omc: omcForm,
      northbound: northboundForm,
    }),
    [basicForm, securityForm, deviceForm, notifyForm, storageForm, omcForm, northboundForm],
  );

  // 拉当前 tab 的所有 KV（按 category）。切 tab 自动重发请求。
  const { data: configList, isFetching } = useSysConfigsByCategory(activeTab);

  // 把后端返回的 KV 灌进对应 tab 的 form。空数据也照样 reset，避免显示其他 tab 的残留值。
  useEffect(() => {
    const form = formMap[activeTab];
    form.resetFields();
    if (!configList || configList.length === 0) return;
    const fields: Record<string, unknown> = {};
    for (const item of configList) {
      fields[item.key] = decodeValue(item.value, item.valueType);
    }
    form.setFieldsValue(fields);
  }, [activeTab, configList, formMap]);

  // 把 form 里所有字段（含未在 configList 中的新增 key）打包成批量 upsert items。
  // value_type 优先沿用后端已记录的 valueType，否则按 JS 运行期类型推断。
  const buildBatchItems = useCallback(
    (formValues: Record<string, unknown>, existing: SysConfigItem[] | undefined): BatchUpdateSysConfigItem[] => {
      const existingMap = new Map<string, SysConfigItem>();
      for (const it of existing || []) existingMap.set(it.key, it);

      const items: BatchUpdateSysConfigItem[] = [];
      for (const [key, raw] of Object.entries(formValues)) {
        const enc = encodeValue(raw);
        const dbType = existingMap.get(key)?.valueType;
        items.push({
          key,
          value: enc.value,
          // 已存在的 key：保留 DB 中的 value_type；新 key：用编码推断的 type。
          value_type: dbType ?? enc.valueType,
        });
      }
      return items;
    },
    [],
  );

  const batchUpdate = useBatchUpdateSysConfigs();

  // 保存当前设置
  const handleSave = useCallback(async () => {
    const form = formMap[activeTab];
    let values: Record<string, unknown>;
    try {
      values = (await form.validateFields()) as Record<string, unknown>;
    } catch {
      void message.error(t('common.formValidationFailed'));
      return;
    }
    const items = buildBatchItems(values, configList);
    if (items.length === 0) {
      void message.warning('当前页面无可保存字段');
      return;
    }
    try {
      await batchUpdate.mutateAsync({ category: activeTab, items });
      void message.success(t('common.save'));
    } catch (err) {
      const msg = err instanceof Error ? err.message : '保存失败';
      void message.error(msg);
    }
  }, [activeTab, formMap, configList, buildBatchItems, batchUpdate, t]);

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
      default:
        return null;
    }
  };

  // Tabs 配置
  const tabItems = settingsTabs.map((tab) => ({
    key: tab.key,
    label: t(tab.labelKey),
  }));

  return (
    <ListPageLayout title={t('nav.system.config')}>
      {/* 页签切换 - 放在 Card 外部 */}
      <Tabs
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as SettingsTab)}
        items={tabItems}
      />
      {/* 设置内容（含 loading 遮罩） */}
      <Spin spinning={isFetching}>
        {renderSettingsContent()}
      </Spin>
      {/* 底部保存按钮 */}
      <div style={{ marginTop: 16, textAlign: 'center' }}>
        <Space>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={batchUpdate.isPending}
            onClick={handleSave}
          >
            {t('common.save')}
          </Button>
        </Space>
      </div>
    </ListPageLayout>
  );
}
