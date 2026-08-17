import { useMemo, useState } from 'react';
import { Button, Descriptions, Drawer, Space, Table, Tag, Typography } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import {
  useEmailRunDeliveries,
  useEmailRunHistory,
} from '@core/hooks/api/useNotifications';
import type {
  EmailRunBusinessType,
  EmailRunHistory,
  EmailRunStatus,
} from '@core/types/notification';
import { formatSystemTime } from '@core/utils/systemTime';

const statusColors: Record<EmailRunStatus, string> = {
  pending: 'default',
  processing: 'processing',
  sent: 'success',
  partial_failed: 'warning',
  failed: 'error',
};

export default function EmailRunHistoryList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedRun, setSelectedRun] = useState<EmailRunHistory>();

  const { data, isLoading, refetch } = useEmailRunHistory({
    businessType: filters.businessType as EmailRunBusinessType | undefined,
    status: filters.status as EmailRunStatus | undefined,
    page,
    pageSize,
  });
  const deliveries = useEmailRunDeliveries(
    selectedRun?.businessType,
    selectedRun?.id
  );

  const statusLabels = useMemo<Record<EmailRunStatus, string>>(
    () => ({
      pending: t('notification.emailRun.status.pending'),
      processing: t('notification.emailRun.status.processing'),
      sent: t('notification.emailRun.status.sent'),
      partial_failed: t('notification.emailRun.status.partialFailed'),
      failed: t('notification.emailRun.status.failed'),
    }),
    [t]
  );

  const filterFields = useMemo<FilterField[]>(
    () => [
      {
        name: 'businessType',
        label: t('notification.emailRun.businessType'),
        type: 'select',
        options: [
          { label: t('notification.emailRun.business.alarm'), value: 'alarm' },
          { label: t('notification.emailRun.business.kpi'), value: 'kpi' },
        ],
      },
      {
        name: 'status',
        label: t('notification.emailRun.status'),
        type: 'select',
        options: Object.entries(statusLabels).map(([value, label]) => ({ value, label })),
      },
    ],
    [statusLabels, t]
  );

  const columns = useMemo<DataTableColumn<EmailRunHistory & Record<string, unknown>>[]>(
    () => [
      {
        key: 'scheduledAt',
        title: t('notification.emailRun.scheduledAt'),
        dataIndex: 'scheduledAt',
        width: 170,
        render: (value) => (value ? formatSystemTime(String(value)) : '-'),
      },
      {
        key: 'businessType',
        title: t('notification.emailRun.businessType'),
        dataIndex: 'businessType',
        width: 120,
        render: (value) =>
          t(`notification.emailRun.business.${String(value)}`),
      },
      {
        key: 'templateName',
        title: t('notification.emailRun.template'),
        dataIndex: 'templateName',
        width: 220,
        ellipsis: true,
      },
      {
        key: 'window',
        title: t('notification.emailRun.window'),
        width: 320,
        render: (_, record) =>
          `${formatSystemTime(record.windowStart)} — ${formatSystemTime(record.windowEnd)}`,
      },
      {
        key: 'status',
        title: t('notification.emailRun.status'),
        dataIndex: 'status',
        width: 130,
        render: (value) => {
          const status = value as EmailRunStatus;
          return <Tag color={statusColors[status]}>{statusLabels[status]}</Tag>;
        },
      },
      {
        key: 'summary',
        title: t('notification.emailRun.deliverySummary'),
        width: 200,
        render: (_, record) =>
          t('notification.emailRun.deliverySummaryValue', {
            total: record.recipientCount,
            sent: record.sentCount,
            failed: record.failedCount,
          }),
      },
      {
        key: 'actions',
        title: t('common.operation'),
        width: 100,
        fixed: 'right',
        render: (_, record) => (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setSelectedRun(record)}
          >
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    [statusLabels, t]
  );

  return (
    <>
      <FilterBar
        filterId="email-run-history-filter"
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
        tableId="email-run-history"
        columns={columns}
        dataSource={(data?.items ?? []) as (EmailRunHistory & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey={(record) => `${record.businessType}:${record.id}`}
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(nextPage, nextPageSize) => {
          setPage(nextPage);
          setPageSize(nextPageSize);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1250 }}
      />

      <Drawer
        open={Boolean(selectedRun)}
        title={t('notification.emailRun.deliveryDetail')}
        size={720}
        onClose={() => setSelectedRun(undefined)}
      >
        {selectedRun ? (
          <Descriptions size="small" column={2} style={{ marginBottom: 16 }}>
            <Descriptions.Item label={t('notification.emailRun.scheduledAt')}>
              {formatSystemTime(selectedRun.scheduledAt)}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.emailRun.jobAttempt')}>
              {selectedRun.jobAttempt}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.emailRun.startedAt')}>
              {selectedRun.startedAt ? formatSystemTime(selectedRun.startedAt) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('notification.emailRun.finishedAt')}>
              {selectedRun.finishedAt ? formatSystemTime(selectedRun.finishedAt) : '-'}
            </Descriptions.Item>
          </Descriptions>
        ) : null}
        {selectedRun?.lastError ? (
          <Typography.Paragraph type="danger" copyable>
            {selectedRun.lastError}
          </Typography.Paragraph>
        ) : null}
        <Table
          rowKey="id"
          loading={deliveries.isLoading}
          pagination={false}
          dataSource={deliveries.data ?? []}
          columns={[
            {
              title: t('notification.emailRun.recipient'),
              dataIndex: 'recipient',
              ellipsis: true,
            },
            {
              title: t('notification.emailRun.status'),
              dataIndex: 'status',
              width: 120,
              render: (status: EmailRunStatus) => (
                <Tag color={statusColors[status] ?? 'default'}>
                  {statusLabels[status] ?? status}
                </Tag>
              ),
            },
            {
              title: t('notification.emailRun.attempts'),
              dataIndex: 'attempt',
              width: 90,
            },
            {
              title: t('notification.emailRun.lastError'),
              dataIndex: 'lastError',
              render: (error?: string) =>
                error ? <Typography.Text type="danger">{error}</Typography.Text> : '-',
            },
          ]}
        />
        {deliveries.isError ? (
          <Space style={{ marginTop: 16 }}>
            <Typography.Text type="danger">
              {t('notification.emailRun.loadDetailFailed')}
            </Typography.Text>
            <Button onClick={() => void deliveries.refetch()}>{t('common.retry')}</Button>
          </Space>
        ) : null}
      </Drawer>
    </>
  );
}
