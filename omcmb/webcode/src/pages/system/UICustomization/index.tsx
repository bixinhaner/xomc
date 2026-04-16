import { useState } from 'react';
import {
  Button,
  Form,
  message,
  Space,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import UICustomSettings from '../SystemConfig/UICustomSettings';

export default function UICustomization() {
  const t = useT();
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  // 保存设置
  const handleSave = () => {
    form.validateFields().then(() => {
      setSaving(true);
      setTimeout(() => {
        setSaving(false);
        void message.success(t('common.save'));
      }, 800);
    }).catch(() => {
      void message.error(t('common.formValidationFailed'));
    });
  };

  return (
    <ListPageLayout title={t('nav.system.uiCustom')}>
      <UICustomSettings form={form} />
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
