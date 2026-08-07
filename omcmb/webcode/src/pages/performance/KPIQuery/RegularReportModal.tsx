import { useEffect } from 'react';
import { Form, Input, Modal, Select, Switch, TimePicker, message } from 'antd';
import dayjs from 'dayjs';
import {
  useQueryTemplateRegularReport,
  useUpdateQueryTemplateRegularReport,
} from '@core/hooks/api/usePmQuery';
import type { QueryTemplate, RegularReportPeriod } from '@core/types/pmQuery';
import { useT } from '@/hooks/useT';

interface FormValues {
  enabled: boolean;
  sendTime: dayjs.Dayjs;
  period: RegularReportPeriod;
  recipients: string;
}

export default function RegularReportModal({ template, onClose }: { template: QueryTemplate | null; onClose: () => void }) {
  const t = useT();
  const report = useQueryTemplateRegularReport(template?.id);
  const update = useUpdateQueryTemplateRegularReport();
  const [form] = Form.useForm<FormValues>();

  useEffect(() => {
    if (!report.data) return;
    form.setFieldsValue({
      enabled: report.data.enabled,
      sendTime: dayjs(`2000-01-01T${report.data.sendTime}:00`),
      period: report.data.period,
      recipients: report.data.recipients.join(';'),
    });
  }, [form, report.data]);

  const save = async () => {
    if (!template || !report.data) return;
    try {
      const values = await form.validateFields();
      const recipients = values.recipients.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean);
      if (values.enabled && recipients.length === 0) {
        form.setFields([{ name: 'recipients', errors: [t('perf.kpiQuery.regularReport.recipientRequired')] }]);
        return;
      }
      await update.mutateAsync({
        id: template.id,
        revision: report.data.revision,
        input: {
          enabled: values.enabled,
          sendTime: values.sendTime.format('HH:mm'),
          period: values.period,
          recipients,
        },
      });
      void message.success(t('common.saveSuccess'));
      onClose();
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return;
      void message.error(error instanceof Error ? error.message : String(error));
    }
  };

  return (
    <Modal
      title={t('perf.kpiQuery.regularReport.title', { name: template?.name ?? '' })}
      open={Boolean(template)}
      onCancel={onClose}
      onOk={() => void save()}
      confirmLoading={update.isPending}
      okButtonProps={{ disabled: report.isLoading || !report.data }}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item name="enabled" label={t('common.enabled')} valuePropName="checked">
          <Switch disabled={report.isLoading || !report.data} />
        </Form.Item>
        <Form.Item name="sendTime" label={t('perf.kpiQuery.regularReport.sendTime')} rules={[{ required: true }]}> 
          <TimePicker format="HH:mm" minuteStep={15} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="period" label={t('perf.kpiQuery.regularReport.period')} rules={[{ required: true }]}> 
          <Select options={[
            { value: '15min', label: t('perf.kpiQuery.regularReport.period15min') },
            { value: 'hour', label: t('perf.kpiQuery.regularReport.periodHour') },
            { value: 'day', label: t('perf.kpiQuery.regularReport.periodDay') },
          ]} />
        </Form.Item>
        <Form.Item name="recipients" label={t('perf.kpiQuery.regularReport.recipients')}>
          <Input.TextArea rows={4} placeholder={t('perf.kpiQuery.regularReport.recipientsHint')} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
