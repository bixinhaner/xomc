import { useState, useMemo } from 'react';
import { Button, Tag, Space } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useNotificationHistory } from '@core/hooks/api/useNotifications';
import type {
  NotificationHistory,
  NotificationChannel,
  NotificationHistoryStatus,
} from '@core/types/notification';
import { useT } from '@/hooks/useT';
import HistoryDetail from './HistoryDetail';
import { formatSystemTime } from '@core/utils/systemTime';

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

export default function HistoryList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [detail, setDetail] = useState<NotificationHistory | null>(null);

  const { data, isLoading, refetch } = useNotificationHistory({
    channel: filters.channel as NotificationChannel | undefined,
    status: filters.status as NotificationHistoryStatus | undefined,
    page,
    pageSize,
  });

  const statusLabelMap: Record<NotificationHistoryStatus, string> = useMemo(
    () => ({
      pending: t('notification.status.pending'),
      sent: t('notification.status.sent'),
      failed: t('notification.status.failed'),
      dead_letter: t('notification.status.dead_letter'),
    }),
    [t]
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'channel',
        label: t('notification.template.channel'),
        type: 'select',
        options: [
          { label: t('notification.channel.email'), value: 'email' },
          { label: t('notification.channel.sms'), value: 'sms' },
          { label: t('notification.channel.webhook'), value: 'webhook' },
        ],
      },
      {
        name: 'status',
        label: t('notification.history.status'),
        type: 'select',
        options: [
          { label: t('notification.status.pending'), value: 'pending' },
          { label: t('notification.status.sent'), value: 'sent' },
          { label: t('notification.status.failed'), value: 'failed' },
          { label: t('notification.status.dead_letter'), value: 'dead_letter' },
        ],
      },
    ],
    [t]
  );

  const columns: DataTableColumn<NotificationHistory & Record<string, unknown>>[] =
    useMemo(
      () => [
        {
          key: 'createdAt',
          title: t('notification.history.createdAt'),
          dataIndex: 'createdAt',
          width: 170,
          render: (val) =>
            val ? formatSystemTime(String(val)) : '-',
        },
        {
          key: 'channel',
          title: t('notification.template.channel'),
          dataIndex: 'channel',
          width: 100,
          render: (val) => {
            const ch = val as NotificationChannel;
            return <Tag color={channelColorMap[ch]}>{ch}</Tag>;
          },
        },
        {
          key: 'status',
          title: t('notification.history.status'),
          dataIndex: 'status',
          width: 110,
          render: (val) => {
            const s = val as NotificationHistoryStatus;
            return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
          },
        },
        {
          key: 'subject',
          title: t('notification.history.subject'),
          dataIndex: 'subject',
          width: 240,
          ellipsis: true,
        },
        {
          key: 'recipients',
          title: t('notification.history.recipients'),
          dataIndex: 'recipients',
          width: 240,
          ellipsis: true,
          render: (val) => {
            const list = (val as string[]) ?? [];
            if (list.length === 0) return '-';
            return list.join(', ');
          },
        },
        {
          key: 'retryCount',
          title: t('notification.history.retryCount'),
          dataIndex: 'retryCount',
          width: 90,
        },
        {
          key: 'sentAt',
          title: t('notification.history.sentAt'),
          dataIndex: 'sentAt',
          width: 170,
          render: (val) =>
            val ? formatSystemTime(String(val)) : '-',
        },
        {
          key: 'actions',
          title: t('common.operation'),
          dataIndex: 'id',
          width: 100,
          fixed: 'right',
          render: (_, record) => {
            const h = record as NotificationHistory;
            return (
              <Space size={4}>
                <Button
                  type="link"
                  size="small"
                  icon={<EyeOutlined />}
                  onClick={() => setDetail(h)}
                >
                  {t('common.detail')}
                </Button>
              </Space>
            );
          },
        },
      ],
      [t, statusLabelMap]
    );

  return (
    <div>
      <FilterBar
        filterId="notification-history-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setFilters(vals);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <DataTable
        tableId="notification-history"
        columns={columns}
        dataSource={
          (data?.items ?? []) as (NotificationHistory & Record<string, unknown>)[]
        }
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1200 }}
      />

      <HistoryDetail
        open={detail !== null}
        history={detail}
        onClose={() => setDetail(null)}
      />
    </div>
  );
}
