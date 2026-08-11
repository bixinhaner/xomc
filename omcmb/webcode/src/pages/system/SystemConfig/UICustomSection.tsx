import { useEffect, useMemo } from 'react';
import { App, Button, Form, Space, Spin } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import type { BatchUpdateSysConfigItem } from '@core/types/system';
import UICustomSettings from './UICustomSettings';
import { UI_CUSTOM_DEFAULTS, UI_CUSTOM_KEYS } from './uiCustomConstants';

export default function UICustomSection() {
  const t = useT();
  const { message } = App.useApp();
  const [form] = Form.useForm();

  const { data: configs, isLoading, isError, refetch } = useSysConfigsByCategory('ui_custom');
  const batchUpdate = useBatchUpdateSysConfigs();

  const initialValues = useMemo<Record<string, string>>(() => {
    const merged: Record<string, string> = { ...UI_CUSTOM_DEFAULTS };
    if (configs) {
      for (const c of configs) {
        if ((UI_CUSTOM_KEYS as readonly string[]).includes(c.key)) merged[c.key] = c.value ?? '';
      }
    }
    return merged;
  }, [configs]);

  useEffect(() => {
    if (configs) form.setFieldsValue(initialValues);
  }, [configs, initialValues, form]);

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
  );
}
