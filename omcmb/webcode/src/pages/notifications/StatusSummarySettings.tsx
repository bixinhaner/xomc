import { useEffect } from 'react';
import { Alert, Button, Checkbox, Form, Input, Select, Space, Tag, Typography, message } from 'antd';
import { useStatusSummaryConfig, useUpdateStatusSummaryConfig } from '@core/hooks/api/useNotifications';
import { isNotificationRevisionConflict } from '@core/services/api/notificationApi';
import { useUserStore } from '@core/store/userStore';
import type { StatusSummaryConfigPayload } from '@core/types/notification';
import { useT } from '@/hooks/useT';

interface FormValues extends StatusSummaryConfigPayload {
  recipientsText: string;
}

const timeZones = ['Asia/Shanghai', 'UTC', 'Africa/Maputo', 'Africa/Lusaka'];

export default function StatusSummarySettings() {
  const t = useT();
  const isSuperAdmin = useUserStore((state) => state.currentUser?.isSuperAdmin === true);
  const config = useStatusSummaryConfig(isSuperAdmin);
  const update = useUpdateStatusSummaryConfig();
  const [form] = Form.useForm<FormValues>();

  useEffect(() => {
    if (!config.data) return;
    form.setFieldsValue({
      enabled: config.data.enabled,
      sendTime: config.data.sendTime,
      timeZone: config.data.timeZone,
      recipients: config.data.recipients,
      recipientsText: config.data.recipients.join(';'),
    });
  }, [config.data, form]);

  const save = async () => {
    if (!config.data) return;
    try {
      const values = await form.validateFields();
      const recipients = values.recipientsText
        .split(/[;,\n]/)
        .map((value) => value.trim())
        .filter(Boolean);
      const payload: StatusSummaryConfigPayload = {
        enabled: values.enabled,
        sendTime: values.sendTime,
        timeZone: values.timeZone,
        recipients,
      };
      await update.mutateAsync({ revision: config.data.revision, payload });
      void message.success(t('notification.statusSummary.saved'));
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return;
      void message.error(isNotificationRevisionConflict(error)
        ? t('notification.concurrentConflict')
        : t('notification.operationFailed', { error: error instanceof Error ? error.message : String(error) }));
    }
  };

  if (!isSuperAdmin) return null;

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        title={t('notification.statusSummary.gateTitle')}
        description={t('notification.statusSummary.gateDescription')}
      />
      <Form form={form} layout="vertical" style={{ maxWidth: 720 }}>
        <Form.Item name="enabled" valuePropName="checked">
          <Checkbox>{t('notification.statusSummary.enabled')}</Checkbox>
        </Form.Item>
        <Form.Item name="sendTime" label={t('notification.statusSummary.sendTime')} rules={[{ required: true }]}> 
          <Input type="time" style={{ maxWidth: 180 }} />
        </Form.Item>
        <Form.Item name="timeZone" label={t('notification.statusSummary.timeZone')} rules={[{ required: true }]}> 
          <Select options={timeZones.map((value) => ({ value, label: value }))} style={{ maxWidth: 280 }} />
        </Form.Item>
        <Form.Item
          name="recipientsText"
          label={t('notification.statusSummary.recipients')}
          rules={[{ required: true, message: t('notification.statusSummary.recipientRequired') }]}
        >
          <Input.TextArea rows={4} placeholder={t('notification.statusSummary.recipientsHint')} />
        </Form.Item>
        <Space>
          <Button type="primary" loading={update.isPending} onClick={() => void save()}>
            {t('common.save')}
          </Button>
          {config.data && (
            <Tag color={config.data.enabled ? 'green' : 'default'}>
              {config.data.enabled ? t('common.enabled') : t('common.disabled')}
            </Tag>
          )}
        </Space>
        <Typography.Text type="secondary">
          {t('notification.statusSummary.bodyDescription')}
        </Typography.Text>
      </Form>
    </Space>
  );
}
