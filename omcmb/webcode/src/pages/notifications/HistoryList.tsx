import { useMemo, useState } from 'react';
import { Alert, Button, Space, Tag, Typography } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useNotificationDeliveries } from '@core/hooks/api/useNotifications';
import type {
  NotificationDelivery,
  NotificationDeliveryFlowState,
} from '@core/types/notification';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';
import HistoryDetail from './HistoryDetail';

export default function HistoryList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const [detail, setDetail] = useState<NotificationDelivery | null>(null);
  const deliveries = useNotificationDeliveries({
    channel: 'email',
    flowState: filters.flowState as NotificationDeliveryFlowState | undefined,
    limit: pageSize + 1,
    offset: (page - 1) * pageSize,
  });

  const filterFields = useMemo<FilterField[]>(() => [
    {
      name: 'flowState',
      label: t('notification.history.flowState'),
      type: 'select',
      options: ['queued', 'sending', 'retry_wait', 'awaiting_receipt', 'completed', 'dead_letter', 'suppressed', 'cancelled']
        .map((value) => ({ label: t(`notification.delivery.flow.${value}`), value })),
    },
  ], [t]);

  const columns = useMemo<DataTableColumn<NotificationDelivery>[]>(() => [
    {
      key: 'createdAt',
      title: t('notification.history.createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      render: (value) => value ? formatSystemTime(String(value)) : '-',
    },
    { key: 'deviceSn', title: t('notification.history.device'), dataIndex: 'deviceSn', width: 160 },
    {
      key: 'channel',
      title: t('notification.history.channel'),
      dataIndex: 'channel',
      width: 120,
      render: (value) => <Tag>{String(value)}</Tag>,
    },
    { key: 'dispatchKind', title: t('notification.history.dispatchKind'), dataIndex: 'dispatchKind', width: 120 },
    { key: 'maskedAddress', title: t('notification.history.recipient'), dataIndex: 'maskedAddress', width: 180 },
    {
      key: 'flowState',
      title: t('notification.history.flowState'),
      dataIndex: 'flowState',
      width: 140,
      render: (value) => <Tag>{t(`notification.delivery.flow.${String(value)}`)}</Tag>,
    },
    {
      key: 'deliveryResult',
      title: t('notification.history.deliveryResult'),
      dataIndex: 'deliveryResult',
      width: 130,
      render: (value) => <Tag>{t(`notification.delivery.result.${String(value)}`)}</Tag>,
    },
    {
      key: 'actions',
      title: t('common.operation'),
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => setDetail(record)}>
          {t('common.detail')}
        </Button>
      ),
    },
  ], [t]);

  const fetchedItems = deliveries.data ?? [];
  const pageItems = fetchedItems.slice(0, pageSize);
  const hasNextPage = fetchedItems.length > pageSize;

  return (
    <>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        title={t('notification.history.perRecipientHint')}
      />
      <FilterBar
        filterId="notification-delivery-filter"
        fields={filterFields}
        onSearch={(values) => {
          setFilters(values);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <DataTable
        tableId="notification-deliveries"
        columns={columns}
        dataSource={pageItems}
        loading={deliveries.isLoading}
        rowKey="id"
        showPagination={false}
        onRefresh={() => void deliveries.refetch()}
        scroll={{ x: 1250 }}
      />
      <Space style={{ justifyContent: 'flex-end', width: '100%', marginTop: 12 }}>
        <Typography.Text type="secondary">
          {t('notification.history.page', { page })}
        </Typography.Text>
        <Button disabled={page === 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>
          {t('notification.history.previousPage')}
        </Button>
        <Button disabled={!hasNextPage} onClick={() => setPage((value) => value + 1)}>
          {t('notification.history.nextPage')}
        </Button>
      </Space>
      <HistoryDetail open={Boolean(detail)} delivery={detail} onClose={() => setDetail(null)} />
    </>
  );
}
