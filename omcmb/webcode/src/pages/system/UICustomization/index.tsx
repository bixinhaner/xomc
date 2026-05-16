import { useEffect, useMemo } from 'react';
import { App, Button, Form, Space, Spin } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import type { BatchUpdateSysConfigItem } from '@core/types/system';
import UICustomSettings from '../SystemConfig/UICustomSettings';
import { UI_CUSTOM_DEFAULTS, UI_CUSTOM_KEYS } from '../SystemConfig/uiCustomConstants';

// PRD docs/prd/system/ui-customization.md §5 — UI 定制化页面：
// - 加载：GET /admin/sysConfig?category=ui_custom
// - 保存：POST /admin/sysConfig/batch（一次写所有 5 项）
// - 上传图片：在 UICustomSettings 内部走 useUploadUIAsset → POST /admin/uploads/ui-asset
function UICustomizationContent() {
  const t = useT();
  const { message } = App.useApp();
  const [form] = Form.useForm();

  const { data: configs, isLoading, isError, refetch } = useSysConfigsByCategory('ui_custom');
  const batchUpdate = useBatchUpdateSysConfigs();

  // 把后端 list 拍平成 key → value，并合并默认值（缺失项用默认值兜底）。
  const initialValues = useMemo<Record<string, string>>(() => {
    const merged: Record<string, string> = { ...UI_CUSTOM_DEFAULTS };
    if (configs) {
      for (const c of configs) {
        if ((UI_CUSTOM_KEYS as readonly string[]).includes(c.key)) merged[c.key] = c.value ?? '';
      }
    }
    return merged;
  }, [configs]);

  // 数据到位后回填表单。
  useEffect(() => {
    if (configs) form.setFieldsValue(initialValues);
  }, [configs, initialValues, form]);

  // 加载失败提示。
  useEffect(() => {
    if (isError) void message.error(t('system.ui.loadFailed'));
  }, [isError, message, t]);

  const buildItems = (vals: Record<string, string>): BatchUpdateSysConfigItem[] =>
    UI_CUSTOM_KEYS.map((key) => ({ key, value: vals[key] ?? '', value_type: 'string' as const }));

  const handleSave = () => {
    form
      .validateFields()
      .then((vals) => batchUpdate.mutateAsync({ category: 'ui_custom', items: buildItems(vals) }))
      .then(() => {
        void message.success(t('common.save'));
        void refetch();
      })
      .catch((err) => {
        if (err && typeof err === 'object' && 'errorFields' in err) {
          void message.error(t('common.formValidationFailed'));
        } else {
          void message.error(t('system.ui.saveFailed'));
        }
      });
  };

  // 恢复默认：先把表单字段重置为 UI_CUSTOM_DEFAULTS，再 batch 写后端，使数据库与界面同步。
  const handleRestoreDefaults = () => {
    form.setFieldsValue(UI_CUSTOM_DEFAULTS);
    batchUpdate
      .mutateAsync({ category: 'ui_custom', items: buildItems(UI_CUSTOM_DEFAULTS) })
      .then(() => {
        void message.success(t('system.ui.restoredDefault'));
        void refetch();
      })
      .catch(() => void message.error(t('system.ui.saveFailed')));
  };

  return (
    <ListPageLayout title={t('nav.system.uiCustom')}>
      <Spin spinning={isLoading}>
        <UICustomSettings
          form={form}
          initialValues={initialValues}
          onRestoreDefaults={handleRestoreDefaults}
          restoring={batchUpdate.isPending}
        />
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
      </Spin>
    </ListPageLayout>
  );
}

export default function UICustomization() {
  return (
    <App>
      <UICustomizationContent />
    </App>
  );
}
