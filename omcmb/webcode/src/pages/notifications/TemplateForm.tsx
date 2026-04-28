import { useEffect } from 'react';
import {
  Drawer,
  Form,
  Input,
  Select,
  Switch,
  Button,
  Space,
  message,
} from 'antd';
import {
  useCreateNotificationTemplate,
  useUpdateNotificationTemplate,
} from '@core/hooks/api/useNotifications';
import type {
  NotificationTemplate,
  NotificationTemplateCreatePayload,
  NotificationChannel,
  NotificationLanguage,
} from '@core/types/notification';
import { useT } from '@/hooks/useT';

interface FormValues {
  name: string;
  channel: NotificationChannel;
  language: NotificationLanguage;
  subject: string;
  body: string;
  variables: string;
  enabled: boolean;
}

interface TemplateFormProps {
  open: boolean;
  initial?: NotificationTemplate | null;
  onClose: () => void;
  onSaved?: () => void;
}

export default function TemplateForm({
  open,
  initial,
  onClose,
  onSaved,
}: TemplateFormProps) {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const createMut = useCreateNotificationTemplate();
  const updateMut = useUpdateNotificationTemplate();

  const isEdit = Boolean(initial?.id);
  const submitting = createMut.isPending || updateMut.isPending;

  useEffect(() => {
    if (!open) return;
    if (initial) {
      form.setFieldsValue({
        name: initial.name,
        channel: initial.channel,
        language: initial.language,
        subject: initial.subject,
        body: initial.body,
        variables: (initial.variables ?? []).join(','),
        enabled: initial.enabled,
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        channel: 'email',
        language: 'zh-CN',
        enabled: true,
      });
    }
  }, [open, initial, form]);

  const handleSubmit = () => {
    void form
      .validateFields()
      .then((vals) => {
        const variables = (vals.variables ?? '')
          .split(',')
          .map((v) => v.trim())
          .filter(Boolean);
        const payload: NotificationTemplateCreatePayload = {
          name: vals.name,
          channel: vals.channel,
          language: vals.language,
          subject: vals.subject,
          body: vals.body,
          variables,
          enabled: vals.enabled,
        };

        if (isEdit && initial) {
          updateMut.mutate(
            { id: initial.id, payload },
            {
              onSuccess: () => {
                void message.success(t('common.save') + ' ✓');
                onSaved?.();
                onClose();
              },
              onError: (err) => {
                void message.error(
                  err instanceof Error ? err.message : 'Save failed'
                );
              },
            }
          );
        } else {
          createMut.mutate(payload, {
            onSuccess: () => {
              void message.success(t('common.save') + ' ✓');
              onSaved?.();
              onClose();
            },
            onError: (err) => {
              void message.error(
                err instanceof Error ? err.message : 'Save failed'
              );
            },
          });
        }
      })
      .catch(() => {
        // 表单校验失败：保留 antd 自身错误提示
      });
  };

  return (
    <Drawer
      title={
        isEdit ? t('notification.template.edit') : t('notification.template.create')
      }
      open={open}
      width={600}
      onClose={onClose}
      destroyOnClose
      footer={
        <div style={{ textAlign: 'right' }}>
          <Space>
            <Button onClick={onClose}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleSubmit} loading={submitting}>
              {t('common.save')}
            </Button>
          </Space>
        </div>
      }
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label={t('notification.template.name')}
          rules={[{ required: true }]}
        >
          <Input placeholder={t('common.pleaseInput')} />
        </Form.Item>

        <Form.Item
          name="channel"
          label={t('notification.template.channel')}
          rules={[{ required: true }]}
        >
          <Select
            options={[
              { label: t('notification.channel.email'), value: 'email' },
              { label: t('notification.channel.sms'), value: 'sms' },
              { label: t('notification.channel.webhook'), value: 'webhook' },
            ]}
          />
        </Form.Item>

        <Form.Item
          name="language"
          label={t('notification.template.language')}
          rules={[{ required: true }]}
        >
          <Select
            options={[
              { label: '简体中文', value: 'zh-CN' },
              { label: 'English', value: 'en-US' },
            ]}
          />
        </Form.Item>

        <Form.Item
          name="subject"
          label={t('notification.template.subject')}
          rules={[{ required: true }]}
        >
          <Input placeholder={t('common.pleaseInput')} />
        </Form.Item>

        <Form.Item
          name="body"
          label={t('notification.template.body')}
          rules={[{ required: true }]}
        >
          <Input.TextArea
            rows={6}
            placeholder={t('common.pleaseInput')}
            autoSize={{ minRows: 4, maxRows: 12 }}
          />
        </Form.Item>

        <Form.Item
          name="variables"
          label={t('notification.template.variables')}
          tooltip={t('notification.template.variablesHint')}
        >
          <Input placeholder={t('notification.template.variablesHint')} />
        </Form.Item>

        <Form.Item
          name="enabled"
          label={t('notification.template.enabled')}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
