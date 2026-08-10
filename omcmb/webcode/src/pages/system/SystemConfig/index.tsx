import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Alert,
  Form,
  message,
  Tabs,
  Space,
  Spin,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import BasicSettings from './BasicSettings';
import SecuritySettings from './SecuritySettings';
import DeviceSettings from './DeviceSettings';
import StorageSettings from './StorageSettings';
// NorthboundSettings import 已移除（#820：北向功能未完成，tab 已隐藏）
import TransferSettings from './TransferSettings';
import AgentSettings from './AgentSettings';
import PmRetentionSection from './PmRetentionSection';
import RetentionBackpressureSection from './RetentionBackpressureSection';
import GeofenceSystemSettings from './GeofenceSystemSettings';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
  useSysConfigApplyBatch,
} from '@core/hooks/api/useSystem';
import type { SysConfigValueType } from '@core/types/system';
import { buildBatchItems } from './sysConfigSerialize';
import type { ConfigApplyBatch } from '@core/types/system';
import { isApplyBatchForCategory, isEventDeliveryBatch } from './applyStatus';
import { useUserStore } from '@core/store/userStore';
import styles from './SystemConfig.module.css';

// 设置子页签类型（v1.0：移除 sas / ldap，参 omgo/docs/prd/system/config.md）
// notify tab 已隐藏（#781）：邮件/短信后端未真实打通前不展示，避免误导用户
// omc tab 已隐藏（#802）：rsyslog/磁盘告警后端未实现，两个卡片均为空壳
// northbound tab 已隐藏（#820）：北向功能未完成（用户管理 Mock 数据、服务信息无 DB 记录），待完成后恢复
type SettingsTab = 'basic' | 'security' | 'device' | 'storage' | 'acs_transfer' | 'agent' | 'geofence' | 'pm_retention' | 'retention_bp';

// 设置子页签配置
const settingsTabs: { key: SettingsTab; labelKey: string }[] = [
  { key: 'basic', labelKey: 'system.config.basic' },
  { key: 'security', labelKey: 'system.config.security' },
  { key: 'device', labelKey: 'system.config.device' },
  { key: 'storage', labelKey: 'system.config.storage' },
  { key: 'acs_transfer', labelKey: 'system.config.acsTransfer' },
  { key: 'agent', labelKey: 'system.config.agent' },
  { key: 'geofence', labelKey: 'system.config.geofence' },
  // northbound 已隐藏（#820）
  // T-0164 收尾 G2-Gap-1：PM 数据保留策略页签
  { key: 'pm_retention', labelKey: 'system.config.pmRetention' },
  // #318-321：资源保留与上传背压（背压 / 原始件 ILM / 基站日志保留 / 压缩回写）
  { key: 'retention_bp', labelKey: 'system.config.retentionBp' },
];

function coerceSettingsTab(raw: string | null): SettingsTab {
  return settingsTabs.some((tab) => tab.key === raw) ? (raw as SettingsTab) : 'basic';
}

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

export default function SystemConfig() {
  const t = useT();
  const isSuperAdmin = useUserStore(
    (state) => state.currentUser?.isSuperAdmin === true,
  );
  const [searchParams] = useSearchParams();
  const tabsContainerRef = useRef<HTMLDivElement>(null);
  const requestedTab = coerceSettingsTab(searchParams.get('tab'));
  const [activeTab, setActiveTab] = useState<SettingsTab>(requestedTab);
  const [submittedBatch, setSubmittedBatch] = useState<ConfigApplyBatch | null>(null);
  const { data: refreshedBatch } = useSysConfigApplyBatch(submittedBatch?.id);
  const applyBatch = refreshedBatch ?? submittedBatch;
  const visibleApplyBatch = isApplyBatchForCategory(applyBatch, activeTab) ? applyBatch : null;

  useEffect(() => {
    setActiveTab(requestedTab);
    setSubmittedBatch(null);
  }, [requestedTab]);

  // 各设置模块的表单实例
  const [basicForm] = Form.useForm();
  const [securityForm] = Form.useForm();
  const [deviceForm] = Form.useForm();
  const [storageForm] = Form.useForm();
  const [transferForm] = Form.useForm();
  // tab → form 映射（稳定引用，因为每个 form 都来自 useForm()）。
  // 注意：自管表单的页签（如 pm_retention，PmRetentionSection 内部自带 form +
  // 保存/重置按钮）【不在此映射内】，故用 Partial —— formMap[activeTab] 可能为
  // undefined，下面 useEffect / handleSave / 底部保存按钮均按是否存在 form 做守卫。
  const formMap = useMemo<Partial<Record<SettingsTab, ReturnType<typeof Form.useForm>[0]>>>(
    () => ({
      basic: basicForm,
      security: securityForm,
      device: deviceForm,
      storage: storageForm,
      acs_transfer: transferForm,
    }),
    [basicForm, securityForm, deviceForm, storageForm, transferForm],
  );

  // 拉当前 tab 的所有 KV（按 category）。切 tab 自动重发请求。
  const activeForm = formMap[activeTab];
  const {
    data: configList,
    isFetching,
    isError,
    isSuccess,
    refetch,
  } = useSysConfigsByCategory(activeTab, Boolean(activeForm));
  const canSave = isSuccess && !isFetching;
  const canSaveRef = useRef(canSave);
  useLayoutEffect(() => {
    canSaveRef.current = canSave;
  }, [canSave]);

  // 把后端返回的 KV 灌进对应 tab 的 form。空数据也照样 reset，避免显示其他 tab 的残留值。
  useEffect(() => {
    const form = activeForm;
    // 自管表单页签（pm_retention / agent 等）无对应 form，跳过 —— 否则 form.resetFields()
    // 会抛 "Cannot read properties of undefined (reading 'resetFields')"。
    if (!form || !isSuccess) return;
    form.resetFields();
    if (!configList || configList.length === 0) return;
    const fields: Record<string, unknown> = {};
    for (const item of configList) {
      // write-only secret 的 value 固定为空，不能灌回表单形成“空值覆盖”。
      if (item.isSecret) continue;
      fields[item.key] = decodeValue(item.value, item.valueType);
    }
    if (
      activeTab === 'device' &&
      fields.periodicSyncIntervalMinutes === undefined &&
      typeof fields.periodicSyncIntervalHours === 'number' &&
      fields.periodicSyncIntervalHours > 0
    ) {
      fields.periodicSyncIntervalMinutes = fields.periodicSyncIntervalHours * 60;
    }
    form.setFields(
      Object.entries(fields).map(([name, value]) => ({ name, value, touched: false })),
    );
  }, [activeForm, activeTab, configList, isSuccess]);

  const batchUpdate = useBatchUpdateSysConfigs();

  const defaultPasswordConfigured = useMemo(
    () => configList?.some((item) => item.key === 'defaultPasswd' && item.isSecret && item.isConfigured) ?? false,
    [configList],
  );

  // 保存当前设置
  const handleSave = useCallback(async () => {
    const form = formMap[activeTab];
    // 自管表单页签（pm_retention）由其组件内部按钮保存，不走这里的全局保存。
    if (!form) return;
    if (!canSaveRef.current) {
      void message.error(t('empty.loadFailed'));
      return;
    }
    let values: Record<string, unknown>;
    try {
      values = (await form.validateFields()) as Record<string, unknown>;
    } catch {
      void message.error(t('common.formValidationFailed'));
      return;
    }
    if (!canSaveRef.current) {
      void message.error(t('empty.loadFailed'));
      return;
    }
    const changedValues = Object.fromEntries(
      Object.entries(values).filter(([key]) => form.isFieldTouched(key)),
    );
    const items = buildBatchItems(changedValues, configList);
    if (items.length === 0) {
      void message.warning(t('sysconfig.warn.noSaveable'));
      return;
    }
    try {
      const result = await batchUpdate.mutateAsync({ category: activeTab, items });
      setSubmittedBatch(result.batch);
      void message.success(t('common.save'));
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('sysconfig.error.saveFailed');
      void message.error(msg);
    }
  }, [activeTab, formMap, configList, batchUpdate, t]);

  // 渲染设置内容
  const renderSettingsContent = () => {
    switch (activeTab) {
      case 'basic':
        return <BasicSettings form={basicForm} />;
      case 'security':
        return (
          <SecuritySettings
            form={securityForm}
            defaultPasswordConfigured={defaultPasswordConfigured}
          />
        );
      case 'device':
        return <DeviceSettings form={deviceForm} />;
      case 'storage':
        return <StorageSettings form={storageForm} />;
      case 'acs_transfer':
		return <TransferSettings form={transferForm} />;
      case 'agent':
        return <AgentSettings />;
      case 'geofence':
        return <GeofenceSystemSettings />;
      // northbound case 已移除（#820）
      case 'pm_retention':
        // T-0164 收尾 G2-Gap-1：PM 数据保留独立组件，内部自管 form + state（不需要 form props）
        return <PmRetentionSection />;
      case 'retention_bp':
        // #318-321：资源保留与上传背压，4 张分类卡片各自管 form + 保存
        return <RetentionBackpressureSection />;
      default:
        return null;
    }
  };

  // Tabs 配置
  const tabItems = settingsTabs
    .filter((tab) => tab.key !== 'geofence' || isSuperAdmin)
    .map((tab) => ({
      key: tab.key,
      label: t(tab.labelKey),
    }));

  const handleTabChange = useCallback((key: string) => {
    const scrollContainer = tabsContainerRef.current?.closest('main');
    if (scrollContainer) scrollContainer.scrollTop = 0;
    setSubmittedBatch(null);
    setActiveTab(key as SettingsTab);
  }, []);

  return (
    <ListPageLayout>
      {/* 页签切换 - 放在 Card 外部 */}
      <div ref={tabsContainerRef} className={styles.stickyTabs}>
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          items={tabItems}
        />
      </div>
      {/* 设置内容（含 loading 遮罩） */}
      <Spin spinning={isFetching}>
        {renderSettingsContent()}
      </Spin>
      {activeForm && isError && (
        <Alert
          type="error"
          showIcon
          title={t('empty.loadFailed')}
          description={t('empty.loadFailedDesc')}
          action={<Button onClick={() => void refetch()}>{t('common.retry')}</Button>}
        />
      )}
      {visibleApplyBatch && (
        <Alert
          style={{ marginTop: 12 }}
          type={visibleApplyBatch.status === 'failed' ? 'error' : visibleApplyBatch.status === 'applied' ? 'success' : 'info'}
          showIcon
          message={t(
            visibleApplyBatch.status === 'applied' && isEventDeliveryBatch(visibleApplyBatch)
              ? 'sysconfig.apply.delivered'
              : `sysconfig.apply.${visibleApplyBatch.status}`,
          )}
          description={visibleApplyBatch.status === 'failed'
            ? visibleApplyBatch.targets.find((target) => target.lastError)?.lastError
            : undefined}
        />
      )}
      {/* 底部保存按钮：仅对走全局表单的页签显示；自管表单页签（pm_retention）
          由其组件内部的保存/重置按钮负责，避免重复且避免对 undefined form 操作。 */}
      {formMap[activeTab] && (
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Space>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={batchUpdate.isPending}
              disabled={!canSave}
              onClick={handleSave}
            >
              {t('common.save')}
            </Button>
          </Space>
        </div>
      )}
    </ListPageLayout>
  );
}
