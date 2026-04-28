import { Drawer, Descriptions, Tag, Typography, Space } from 'antd';
import type {
  NotificationHistory,
  NotificationChannel,
  NotificationHistoryStatus,
} from '@core/types/notification';
import { useT } from '@/hooks/useT';

interface HistoryDetailProps {
  open: boolean;
  history: NotificationHistory | null;
  onClose: () => void;
}

const channelColorMap: Record<NotificationChannel, string> = {
  email: 'blue',
  sms: 'green',
  webhook: 'purple',
};

const statusColorMap: Record<NotificationHistoryStatus, string> = {
  pending: 'orange',
  sent: 'green',
  failed: 'red',
  dead_letter: 'volcano',
};

export default function HistoryDetail({
  open,
  history,
  onClose,
}: HistoryDetailProps) {
  const t = useT();

  return (
    <Drawer
      title={t('notification.history.detail')}
      open={open}
      width={640}
      onClose={onClose}
      destroyOnClose
    >
      {history ? (
        <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="ID">
            <Typography.Text copyable code>
              {history.id}
            </Typography.Text>
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.template.channel')}>
            <Tag color={channelColorMap[history.channel]}>{history.channel}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.history.status')}>
            <Tag color={statusColorMap[history.status]}>
              {t(`notification.status.${history.status}`)}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.history.recipients')}>
            <Space wrap>
              {history.recipients.length === 0
                ? '-'
                : history.recipients.map((r, i) => (
                    <Tag key={i}>{r}</Tag>
                  ))}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.history.subject')}>
            {history.subject || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.history.body')}>
            <pre
              style={{
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                margin: 0,
                fontFamily: 'inherit',
              }}
            >
              {history.body || '-'}
            </pre>
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.history.retryCount')}>
            {history.retryCount}
          </Descriptions.Item>
          {history.errorMessage && (
            <Descriptions.Item label={t('notification.history.errorMessage')}>
              <Typography.Text type="danger">{history.errorMessage}</Typography.Text>
            </Descriptions.Item>
          )}
          {history.alarmId && (
            <Descriptions.Item label={t('notification.history.alarmId')}>
              <Typography.Text code>{history.alarmId}</Typography.Text>
            </Descriptions.Item>
          )}
          {history.templateId && (
            <Descriptions.Item label={t('notification.history.templateId')}>
              <Typography.Text code>{history.templateId}</Typography.Text>
            </Descriptions.Item>
          )}
          <Descriptions.Item label={t('notification.history.sentAt')}>
            {history.sentAt
              ? new Date(history.sentAt).toLocaleString()
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('notification.history.createdAt')}>
            {new Date(history.createdAt).toLocaleString()}
          </Descriptions.Item>
        </Descriptions>
      ) : null}
    </Drawer>
  );
}
