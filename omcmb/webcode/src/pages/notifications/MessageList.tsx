import { useMemo, useState } from 'react';
import { Button, Descriptions, Drawer, Space, Tag, Typography, message } from 'antd';
import { CheckCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import AutoRefreshDropdown from '@/pages/alarm/components/AutoRefreshDropdown';
import { useT } from '@/hooks/useT';
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotificationCenter,
} from '@core/hooks/api/useNotificationCenter';
import type {
  NotificationCenterItem,
  NotificationCenterStatus,
  NotificationCenterType,
  NotificationCenterUiState,
} from '@core/types/notificationCenter';
import { toUiState } from '@core/types/notificationCenter';
import { formatSystemTime } from '@core/utils/systemTime';

const { Text } = Typography;

const uiStateColorMap: Record<NotificationCenterUiState, string> = {
  in_progress: 'processing',
  completed: 'success',
  failed: 'error',
  expired: 'warning',
  cancelled: 'default',
};

export default function MessageList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [detail, setDetail] = useState<NotificationCenterItem | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30);

  const isReadFilter =
    filters.isRead === 'true' ? true : filters.isRead === 'false' ? false : undefined;

  const listQuery = useNotificationCenter({
    page,
    pageSize,
    filter: {
      type: filters.type as NotificationCenterType | undefined,
      isRead: isReadFilter,
    },
    refetchIntervalMs: autoRefresh ? refreshInterval * 1000 : undefined,
  });
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  const typeLabelMap: Record<NotificationCenterType, string> = useMemo(
    () => ({
      alarm: t('notificationCenter.type.alarm'),
      task_complete: t('notificationCenter.type.taskComplete'),
      system: t('notificationCenter.type.system'),
      approval: t('notificationCenter.type.approval'),
      device_status: t('notificationCenter.type.deviceStatus'),
    }),
    [t]
  );

  const statusLabelMap: Record<NotificationCenterUiState, string> = useMemo(
    () => ({
      in_progress: t('notificationCenter.status.inProgress'),
      completed: t('notificationCenter.status.completed'),
      failed: t('notificationCenter.status.failed'),
      expired: t('notificationCenter.status.expired'),
      cancelled: t('notificationCenter.status.cancelled'),
    }),
    [t]
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'type',
        label: t('notificationCenter.messageType'),
        type: 'select',
        options: [
          { label: t('notificationCenter.type.alarm'), value: 'alarm' },
          { label: t('notificationCenter.type.taskComplete'), value: 'task_complete' },
          { label: t('notificationCenter.type.system'), value: 'system' },
          { label: t('notificationCenter.type.approval'), value: 'approval' },
          { label: t('notificationCenter.type.deviceStatus'), value: 'device_status' },
        ],
      },
      {
        name: 'isRead',
        label: t('notificationCenter.readState'),
        type: 'select',
        options: [
          { label: t('notificationCenter.readState.unread'), value: 'false' },
          { label: t('notificationCenter.readState.read'), value: 'true' },
        ],
      },
    ],
    [t]
  );

  const handleOpenMessage = async (item: NotificationCenterItem) => {
    if (!item.isRead) {
      try {
        await markRead.mutateAsync(item.id);
      } catch {
        // Mark-read failure should not block viewing the message detail.
      }
    }
    setDetail({ ...item, isRead: true });
  };

  const handleMarkAllRead = async () => {
    try {
      await markAllRead.mutateAsync();
      void message.success(t('notificationCenter.markAllReadSuccess'));
    } catch (e) {
      void message.error(t('notificationCenter.markAllReadFailed', { error: String(e) }));
    }
  };

  const columns: DataTableColumn<NotificationCenterItem & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'createdAt',
        title: t('notificationCenter.createdAt'),
        dataIndex: 'createdAt',
        width: 170,
        render: (val) => (val ? formatSystemTime(String(val)) : '-'),
      },
      {
        key: 'isRead',
        title: t('notificationCenter.readState'),
        dataIndex: 'isRead',
        width: 90,
        render: (val) =>
          val ? (
            <Tag>{t('notificationCenter.readState.read')}</Tag>
          ) : (
            <Tag color="blue">{t('notificationCenter.readState.unread')}</Tag>
          ),
      },
      {
        key: 'type',
        title: t('notificationCenter.messageType'),
        dataIndex: 'type',
        width: 130,
        render: (val) => {
          const type = val as NotificationCenterType;
          return <Tag color="geekblue">{typeLabelMap[type] ?? String(val)}</Tag>;
        },
      },
      {
        key: 'status',
        title: t('notificationCenter.messageStatus'),
        dataIndex: 'status',
        width: 120,
        render: (val) => {
          const uiState = toUiState(val as NotificationCenterStatus);
          return (
            <Tag color={uiStateColorMap[uiState]}>
              {statusLabelMap[uiState] ?? String(val)}
            </Tag>
          );
        },
      },
      {
        key: 'title',
        title: t('notificationCenter.messageTitle'),
        dataIndex: 'title',
        width: 260,
        render: (val, record) => (
          <Text strong={!record.isRead} style={{ whiteSpace: 'normal', wordBreak: 'break-all' }}>
            {String(val ?? '-')}
          </Text>
        ),
      },
      {
        key: 'content',
        title: t('notificationCenter.messageContent'),
        dataIndex: 'content',
        width: 420,
        render: (val) => (
          <Text type="secondary" style={{ whiteSpace: 'normal', wordBreak: 'break-all' }}>
            {String(val ?? '-')}
          </Text>
        ),
      },
      {
        key: 'sender',
        title: t('notificationCenter.sender'),
        dataIndex: 'sender',
        width: 120,
      },
      {
        key: 'actions',
        title: t('common.operation'),
        dataIndex: 'id',
        width: 110,
        fixed: 'right',
        render: (_, record) => (
          <Button
            type="link"
            size="small"
            onClick={() => void handleOpenMessage(record)}
          >
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    // handleOpenMessage closes over mutation state; columns do not need to rebuild on every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, typeLabelMap, statusLabelMap]
  );

  return (
    <div>
      <FilterBar
        filterId="notification-center-message-filter"
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
        tableId="notification-center-messages"
        columns={columns}
        dataSource={(listQuery.data?.items ?? []) as (NotificationCenterItem & Record<string, unknown>)[]}
        loading={listQuery.isLoading}
        rowKey="id"
        total={listQuery.data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void listQuery.refetch()}
        hideRealtime
        hideRefresh
        scroll={{ x: 1450 }}
        extraToolbarLeft={
          <Space size={8}>
            <Button
              icon={<CheckCircleOutlined />}
              onClick={() => void handleMarkAllRead()}
              disabled={(listQuery.data?.total ?? 0) === 0 || markAllRead.isPending}
            >
              {t('notificationCenter.markAllRead')}
            </Button>
          </Space>
        }
        extraToolbarRight={
          <Space size={8}>
            <Button
              size="small"
              icon={<ReloadOutlined />}
              loading={listQuery.isFetching}
              onClick={() => void listQuery.refetch()}
            >
              {t('common.refresh')}
            </Button>
            <AutoRefreshDropdown
              enabled={autoRefresh}
              intervalSeconds={refreshInterval}
              onEnabledChange={setAutoRefresh}
              onIntervalChange={setRefreshInterval}
              spinning={autoRefresh && listQuery.isFetching}
              size="small"
            />
          </Space>
        }
      />
      <MessageDetailDrawer
        item={detail}
        typeLabelMap={typeLabelMap}
        statusLabelMap={statusLabelMap}
        onClose={() => setDetail(null)}
      />
    </div>
  );
}

interface MessageDetailDrawerProps {
  item: NotificationCenterItem | null;
  typeLabelMap: Record<NotificationCenterType, string>;
  statusLabelMap: Record<NotificationCenterUiState, string>;
  onClose: () => void;
}

function MessageDetailDrawer({
  item,
  typeLabelMap,
  statusLabelMap,
  onClose,
}: MessageDetailDrawerProps) {
  const t = useT();
  const uiState = item ? toUiState(item.status) : 'in_progress';

  return (
    <Drawer
      title={t('notificationCenter.detailTitle')}
      open={Boolean(item)}
      onClose={onClose}
      width={640}
      destroyOnHidden
    >
      {item ? (
        <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="ID">{item.id}</Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.messageTitle')}>
            <Text strong style={{ whiteSpace: 'normal', wordBreak: 'break-all' }}>
              {item.title || '-'}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.messageContent')}>
            <Text style={{ whiteSpace: 'pre-line', wordBreak: 'break-all' }}>
              {item.content || '-'}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.messageType')}>
            <Tag color="geekblue">{typeLabelMap[item.type] ?? item.type}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.messageStatus')}>
            <Tag color={uiStateColorMap[uiState]}>
              {statusLabelMap[uiState] ?? item.status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.readState')}>
            <Tag>{t('notificationCenter.readState.read')}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.sender')}>
            {item.sender || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.createdAt')}>
            {item.createdAt ? formatSystemTime(item.createdAt) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.readAt')}>
            {item.readAt ? formatSystemTime(item.readAt) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('notificationCenter.relatedLink')}>
            {item.link ? (
              <Text copyable style={{ whiteSpace: 'normal', wordBreak: 'break-all' }}>
                {item.link}
              </Text>
            ) : (
              '-'
            )}
          </Descriptions.Item>
        </Descriptions>
      ) : null}
    </Drawer>
  );
}
