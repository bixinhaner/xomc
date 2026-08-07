import { Button, Descriptions, Divider, Drawer, Input, Modal, Space, Table, Tag, Typography, message } from 'antd';
import {
  useNotificationDelivery,
  useNotificationDeliveryAttempts,
  useRetryNotificationDelivery,
} from '@core/hooks/api/useNotifications';
import type { NotificationDelivery } from '@core/types/notification';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';

interface HistoryDetailProps {
  open: boolean;
  delivery: NotificationDelivery | null;
  onClose: () => void;
}

export default function HistoryDetail({ open, delivery, onClose }: HistoryDetailProps) {
  const t = useT();
  const detail = useNotificationDelivery(delivery?.id ?? '');
  const attempts = useNotificationDeliveryAttempts(delivery?.id ?? '');
  const retryMutation = useRetryNotificationDelivery();
  const current = detail.data ?? delivery;

  const retry = () => {
    if (!current) return;
    let reason = '';
    Modal.confirm({
      title: t('notification.history.retryConfirm'),
      content: (
        <Input.TextArea
          rows={3}
          maxLength={500}
          placeholder={t('notification.history.retryReason')}
          onChange={(event) => { reason = event.target.value; }}
        />
      ),
      onOk: async () => {
        if (!reason.trim()) {
          void message.warning(t('notification.history.retryReasonRequired'));
          return Promise.reject();
        }
        try {
          await retryMutation.mutateAsync({ id: current.id, reason: reason.trim() });
          void message.success(t('notification.history.retryQueued'));
        } catch (error) {
          void message.error(t('notification.operationFailed', {
            error: error instanceof Error ? error.message : String(error),
          }));
          throw error;
        }
      },
    });
  };

  return (
    <Drawer
      title={t('notification.history.detail')}
      open={open}
      size={760}
      onClose={onClose}
      destroyOnHidden
      extra={current?.flowState === 'dead_letter' ? (
        <Button danger loading={retryMutation.isPending} onClick={retry}>
          {t('notification.history.manualRetry')}
        </Button>
      ) : undefined}
    >
      {current && (
        <>
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="ID">
              <Typography.Text copyable code>{current.id}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.device')}>
              {current.deviceSn || current.deviceId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.channel')}>
              <Tag>{current.channel}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.flowState')}>
              <Tag>{t(`notification.delivery.flow.${current.flowState}`)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.deliveryResult')}>
              <Tag>{t(`notification.delivery.result.${current.deliveryResult}`)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.recipient')}>
              {current.maskedAddress || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.dispatchKind')}>
              {current.dispatchKind || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.ruleVersion')}>
              <Typography.Text copyable code>{current.ruleVersionId || '-'}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.templateVersion')}>
              <Typography.Text copyable code>{current.templateVersionId || '-'}</Typography.Text>
            </Descriptions.Item>
            {current.suppressionReason && (
              <Descriptions.Item label={t('notification.history.suppressionReason')}>
                {current.suppressionReason}
              </Descriptions.Item>
            )}
            {current.failureReason && (
              <Descriptions.Item label={t('notification.history.failureReason')}>
                <Typography.Text type="danger">{current.failureReason}</Typography.Text>
              </Descriptions.Item>
            )}
            {current.originDeliveryId && (
              <Descriptions.Item label={t('notification.history.originDelivery')}>
                <Typography.Text copyable code>{current.originDeliveryId}</Typography.Text>
              </Descriptions.Item>
            )}
            <Descriptions.Item label={t('notification.history.createdAt')}>
              {formatSystemTime(current.createdAt)}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.history.updatedAt')}>
              {formatSystemTime(current.updatedAt)}
            </Descriptions.Item>
          </Descriptions>
          <Divider titlePlacement="start">{t('notification.history.attempts')}</Divider>
          <Table
            size="small"
            rowKey="id"
            loading={attempts.isLoading}
            pagination={false}
            dataSource={attempts.data ?? []}
            columns={[
              { title: t('notification.history.attemptNo'), dataIndex: 'attemptNo', width: 90 },
              {
                title: t('notification.history.startedAt'),
                dataIndex: 'startedAt',
                width: 180,
                render: (value: string) => value ? formatSystemTime(value) : '-',
              },
              { title: t('notification.history.attemptResult'), dataIndex: 'result', width: 120 },
              { title: t('notification.history.statusSummary'), dataIndex: 'statusSummary' },
              {
                title: t('notification.history.nextRetryAt'),
                dataIndex: 'nextRetryAt',
                width: 180,
                render: (value: string) => value ? formatSystemTime(value) : '-',
              },
            ]}
          />
          {attempts.data?.length === 0 && (
            <Space style={{ marginTop: 12 }}>
              <Typography.Text type="secondary">{t('notification.history.noAttempts')}</Typography.Text>
            </Space>
          )}
        </>
      )}
    </Drawer>
  );
}
