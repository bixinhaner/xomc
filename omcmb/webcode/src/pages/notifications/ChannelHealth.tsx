import { useCallback, useMemo } from 'react';
import { Alert, Button, Space, Tag, Typography, message } from 'antd';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useNotificationChannelHealth,
  useNotificationChannels,
  useVerifyNotificationChannel,
} from '@core/hooks/api/useNotifications';
import type { NotificationChannelConfig } from '@core/types/notification';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';

function HealthSummary({ id }: { id: string }) {
  const t = useT();
  const health = useNotificationChannelHealth(id);
  if (health.isLoading) return <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>;
  if (!health.data) return <Typography.Text type="secondary">-</Typography.Text>;
  const colors = { closed: 'green', open: 'red', half_open: 'orange' } as const;
  return (
    <Space orientation="vertical" size={0}>
      <Tag color={colors[health.data.circuitState]}>
        {t(`notification.channelHealth.circuit.${health.data.circuitState}`)}
      </Tag>
      {health.data.lastVerifiedAt && (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {formatSystemTime(health.data.lastVerifiedAt)}
        </Typography.Text>
      )}
      {health.data.lastErrorSummary && (
        <Typography.Text type="danger" ellipsis style={{ maxWidth: 260, fontSize: 12 }}>
          {health.data.lastErrorSummary}
        </Typography.Text>
      )}
    </Space>
  );
}

export default function ChannelHealth() {
  const t = useT();
  const channels = useNotificationChannels();
  const verifyMutation = useVerifyNotificationChannel();

  const showError = useCallback((error: unknown) => {
    void message.error(
      t('notification.operationFailed', {
        error: error instanceof Error ? error.message : String(error),
      }),
    );
  }, [t]);

  const verify = useCallback(async (record: NotificationChannelConfig) => {
    try {
      await verifyMutation.mutateAsync(record.id);
      void message.success(t('notification.channelHealth.verifyQueued'));
    } catch (error) {
      showError(error);
    }
  }, [showError, t, verifyMutation]);

  const columns = useMemo<DataTableColumn<NotificationChannelConfig>[]>(() => [
    { key: 'name', title: t('notification.channelHealth.name'), dataIndex: 'name', width: 220 },
    {
      key: 'channel',
      title: t('notification.template.channel'),
      dataIndex: 'channel',
      width: 120,
      render: () => <Tag color="blue">{t('notification.channelHealth.email')}</Tag>,
    },
    {
      key: 'enabled',
      title: t('notification.channelHealth.enabled'),
      dataIndex: 'enabled',
      width: 100,
      render: (value) => value
        ? <Tag color="green">{t('common.enabled')}</Tag>
        : <Tag>{t('common.disabled')}</Tag>,
    },
    {
      key: 'secretConfigured',
      title: t('notification.channelHealth.secret'),
      dataIndex: 'secretConfigured',
      width: 120,
      render: (value) => value
        ? <Tag color="green">{t('notification.channelHealth.configured')}</Tag>
        : <Tag>{t('notification.channelHealth.notConfigured')}</Tag>,
    },
    {
      key: 'health',
      title: t('notification.channelHealth.health'),
      width: 280,
      render: (_, record) => <HealthSummary id={record.id} />,
    },
    {
      key: 'actions',
      title: t('common.operation'),
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            loading={verifyMutation.isPending && verifyMutation.variables === record.id}
            onClick={() => void verify(record)}
          >
            {t('notification.channelHealth.verify')}
          </Button>
        </Space>
      ),
    },
  ], [t, verify, verifyMutation.isPending, verifyMutation.variables]);

  return (
    <>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        title={t('notification.channelHealth.productionGate')}
      />
      <DataTable
        tableId="notification-channels"
        columns={columns}
        dataSource={(channels.data ?? []).filter((item) => item.channel === 'email')}
        loading={channels.isLoading}
        rowKey="id"
        showPagination={false}
        onRefresh={() => void channels.refetch()}
        scroll={{ x: 1000 }}
      />
    </>
  );
}
