import { useEffect } from 'react';
import { Alert, App, Button, Form, Modal, Select, Space, Switch, Typography } from 'antd';
import {
  useDeleteQueryReportSubscription,
  useQueryReportSubscription,
  useUpsertQueryReportSubscription,
} from '@core/hooks/api/usePmQuery';
import type {
  QueryTemplate,
  ReportPeriod,
  UpsertQueryReportSubscriptionInput,
} from '@core/types/pmQuery';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  template: QueryTemplate | null;
  onClose: () => void;
}

interface FormValues {
  enabled: boolean;
  period: ReportPeriod;
  sendTimes: string[];
  recipients: string[];
}

const DEFAULT_VALUES: FormValues = {
  enabled: true,
  period: 'daily',
  sendTimes: ['08:00'],
  recipients: [],
};

const MAX_RECIPIENTS = 50;
const EMAIL_ADDRESS_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const SEND_TIME_PATTERN = /^(?:[01]\d|2[0-3]):[0-5]\d$/;

const REPORT_STATUS_KEYS = {
  pending: 'perf.kpiQuery.report.status.pending',
  running: 'perf.kpiQuery.report.status.running',
  sent: 'perf.kpiQuery.report.status.sent',
  export_failed: 'perf.kpiQuery.report.status.exportFailed',
  delivery_failed: 'perf.kpiQuery.report.status.deliveryFailed',
} as const;

export default function ReportSubscriptionModal({ open, template, onClose }: Props) {
  const t = useT();
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<FormValues>();
  const subscriptionQuery = useQueryReportSubscription(open ? template?.id : undefined);
  const upsertMutation = useUpsertQueryReportSubscription();
  const deleteMutation = useDeleteQueryReportSubscription();
  const notFound = (subscriptionQuery.error as { response?: { status?: number } } | null)?.response?.status === 404;

  useEffect(() => {
    if (!open) return;
    const value = subscriptionQuery.data;
    form.setFieldsValue(value ? {
      enabled: value.enabled,
      period: value.period,
      sendTimes: value.sendTimes,
      recipients: value.recipients,
    } : DEFAULT_VALUES);
  }, [form, open, subscriptionQuery.data, notFound, template?.id]);

  const handleSave = async () => {
    if (!template) return;
    const values = await form.validateFields();
    const input: UpsertQueryReportSubscriptionInput = {
      enabled: values.enabled,
      period: values.period,
      sendTimes: Array.from(new Set(values.sendTimes.map((item) => item.trim()))),
      recipients: Array.from(new Set(values.recipients.map((item) => item.trim().toLowerCase()))),
    };
    try {
      await upsertMutation.mutateAsync({ templateId: template.id, input });
      message.success(t('perf.kpiQuery.report.saved'));
      onClose();
    } catch {
      message.error(t('common.operationFailed'));
    }
  };

  const deleteSubscription = async () => {
    if (!template) return;
    try {
      await deleteMutation.mutateAsync(template.id);
      message.success(t('perf.kpiQuery.report.deleted'));
      onClose();
    } catch {
      message.error(t('common.operationFailed'));
    }
  };

  const handleDelete = () => {
    modal.confirm({
      title: t('perf.kpiQuery.report.deleteConfirmTitle'),
      content: t('perf.kpiQuery.report.deleteConfirmContent'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: deleteSubscription,
    });
  };

  return (
    <Modal
      open={open}
      title={t('perf.kpiQuery.report.title', { name: template?.name ?? '' })}
      onCancel={onClose}
      onOk={() => void handleSave()}
      confirmLoading={upsertMutation.isPending}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      destroyOnHidden
      footer={(_, { OkBtn, CancelBtn }) => (
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Button
            danger
            disabled={!subscriptionQuery.data}
            loading={deleteMutation.isPending}
            onClick={handleDelete}
          >
            {t('perf.kpiQuery.report.delete')}
          </Button>
          <Space><CancelBtn /><OkBtn /></Space>
        </Space>
      )}
    >
      {subscriptionQuery.isError && !notFound ? (
        <Alert type="error" showIcon message={t('perf.kpiQuery.report.loadFailed')} style={{ marginBottom: 16 }} />
      ) : null}
      <Form form={form} layout="vertical" initialValues={DEFAULT_VALUES}>
        <Form.Item name="enabled" label={t('perf.kpiQuery.report.enabled')} valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item name="period" label={t('perf.kpiQuery.report.period')} rules={[{ required: true }]}>
          <Select options={[
            { value: '15min', label: t('perf.dashboard.granular15min') },
            { value: 'hourly', label: t('perf.dashboard.granularHourly') },
            { value: 'daily', label: t('perf.dashboard.granularDaily') },
          ]} />
        </Form.Item>
        <Form.Item
          name="sendTimes"
          label={t('perf.kpiQuery.report.sendTimes')}
          extra={t('perf.kpiQuery.report.sendTimesHint')}
          rules={[
            { required: true, message: t('perf.kpiQuery.report.sendTimesRequired') },
            {
              validator: (_, values: string[] | undefined) =>
                (values || []).every((value) => SEND_TIME_PATTERN.test(value.trim()))
                  ? Promise.resolve()
                  : Promise.reject(new Error(t('perf.kpiQuery.report.sendTimesInvalid'))),
            },
          ]}
        >
          <Select
            mode="tags"
            tokenSeparators={[',', ';', ' ']}
            maxCount={8}
            placeholder={t('perf.kpiQuery.report.sendTimesPlaceholder')}
          />
        </Form.Item>
        <Form.Item
          name="recipients"
          label={t('perf.kpiQuery.report.recipients')}
          extra={t('perf.kpiQuery.report.recipientsHint')}
          rules={[
            { required: true, message: t('perf.kpiQuery.report.recipientsRequired') },
            {
              validator: (_, values: string[] | undefined) => {
                const recipients = (values || []).map((value) => value.trim()).filter(Boolean);
                if (recipients.length > MAX_RECIPIENTS) {
                  return Promise.reject(new Error(t('perf.kpiQuery.report.recipientsMax', { max: MAX_RECIPIENTS })));
                }
                if (recipients.some((recipient) => !EMAIL_ADDRESS_PATTERN.test(recipient))) {
                  return Promise.reject(new Error(t('perf.kpiQuery.report.recipientsInvalid')));
                }
                return Promise.resolve();
              },
            },
          ]}
        >
          <Select
            mode="tags"
            tokenSeparators={[',', ';', ' ']}
            maxCount={MAX_RECIPIENTS}
            maxTagCount="responsive"
            placeholder={t('perf.kpiQuery.report.recipientsPlaceholder')}
          />
        </Form.Item>
        {subscriptionQuery.data ? (
          <Space direction="vertical" size={4}>
            {subscriptionQuery.data.nextRunAt ? (
              <Typography.Text type="secondary">
                {t('perf.kpiQuery.report.nextRun', {
                  time: formatSystemTime(subscriptionQuery.data.nextRunAt),
                })}
              </Typography.Text>
            ) : null}
            {subscriptionQuery.data.lastRunAt ? (
              <Typography.Text type="secondary">
                {t('perf.kpiQuery.report.lastRun', {
                  time: formatSystemTime(subscriptionQuery.data.lastRunAt),
                })}
              </Typography.Text>
            ) : null}
            {subscriptionQuery.data.lastStatus ? (
              <Typography.Text type={subscriptionQuery.data.lastStatus.endsWith('_failed') ? 'danger' : 'secondary'}>
                {t('perf.kpiQuery.report.lastStatus', {
                  status: t(REPORT_STATUS_KEYS[subscriptionQuery.data.lastStatus]),
                })}
              </Typography.Text>
            ) : null}
            {subscriptionQuery.data.lastError ? (
              <Alert
                type="error"
                showIcon
                message={t('perf.kpiQuery.report.lastError')}
                description={subscriptionQuery.data.lastError}
              />
            ) : null}
          </Space>
        ) : null}
      </Form>
    </Modal>
  );
}
