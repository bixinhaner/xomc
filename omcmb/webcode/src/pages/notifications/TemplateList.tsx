import { useState, useMemo } from 'react';
import { Button, Tag, Space, Modal, message } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import {
  useNotificationTemplates,
  useDeleteNotificationTemplate,
} from '@core/hooks/api/useNotifications';
import type {
  NotificationTemplate,
  NotificationChannel,
  NotificationLanguage,
} from '@core/types/notification';
import { useT } from '@/hooks/useT';
import TemplateForm from './TemplateForm';
import { formatSystemTime } from '@core/utils/systemTime';

const channelColorMap: Record<NotificationChannel, string> = {
  email: 'blue',
  sms: 'green',
  webhook: 'purple',
};

export default function TemplateList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<NotificationTemplate | null>(null);

  const { data, isLoading, refetch } = useNotificationTemplates({
    channel: filters.channel as NotificationChannel | undefined,
    language: filters.language as NotificationLanguage | undefined,
    enabled:
      typeof filters.enabled === 'boolean'
        ? (filters.enabled as boolean)
        : undefined,
    page,
    pageSize,
  });

  const deleteMut = useDeleteNotificationTemplate();

  const channelLabelMap: Record<NotificationChannel, string> = useMemo(
    () => ({
      email: t('notification.channel.email'),
      sms: t('notification.channel.sms'),
      webhook: t('notification.channel.webhook'),
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
        name: 'language',
        label: t('notification.template.language'),
        type: 'select',
        options: [
          { label: '简体中文', value: 'zh-CN' },
          { label: 'English', value: 'en-US' },
        ],
      },
    ],
    [t]
  );

  const handleDelete = (record: NotificationTemplate) => {
    Modal.confirm({
      title: t('notification.template.deleteConfirm'),
      content: record.name,
      okType: 'danger',
      onOk: () => {
        return new Promise<void>((resolve) => {
          deleteMut.mutate(record.id, {
            onSuccess: () => {
              void message.success(t('common.delete') + ' ✓');
              resolve();
            },
            onError: (err) => {
              void message.error(
                err instanceof Error ? err.message : 'Delete failed'
              );
              resolve();
            },
          });
        });
      },
    });
  };

  const columns: DataTableColumn<NotificationTemplate & Record<string, unknown>>[] =
    useMemo(
      () => [
        {
          key: 'name',
          title: t('notification.template.name'),
          dataIndex: 'name',
          width: 200,
        },
        {
          key: 'channel',
          title: t('notification.template.channel'),
          dataIndex: 'channel',
          width: 100,
          render: (val) => {
            const ch = val as NotificationChannel;
            return (
              <Tag color={channelColorMap[ch]}>{channelLabelMap[ch] ?? String(val)}</Tag>
            );
          },
        },
        {
          key: 'language',
          title: t('notification.template.language'),
          dataIndex: 'language',
          width: 110,
        },
        {
          key: 'subject',
          title: t('notification.template.subject'),
          dataIndex: 'subject',
          width: 240,
          ellipsis: true,
        },
        {
          key: 'enabled',
          title: t('notification.template.enabled'),
          dataIndex: 'enabled',
          width: 90,
          render: (val) =>
            val ? (
              <Tag color="green">{t('common.enabled')}</Tag>
            ) : (
              <Tag>{t('common.disabled')}</Tag>
            ),
        },
        {
          key: 'updatedAt',
          title: t('notification.template.updatedAt'),
          dataIndex: 'updatedAt',
          width: 170,
          render: (val) =>
            val ? formatSystemTime(String(val)) : '-',
        },
        {
          key: 'actions',
          title: t('common.operation'),
          dataIndex: 'id',
          width: 160,
          fixed: 'right',
          render: (_, record) => {
            const tpl = record as NotificationTemplate;
            return (
              <Space size={4}>
                <Button
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => {
                    setEditing(tpl);
                    setFormOpen(true);
                  }}
                >
                  {t('common.edit')}
                </Button>
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => handleDelete(tpl)}
                >
                  {t('common.delete')}
                </Button>
              </Space>
            );
          },
        },
      ],
      // handleDelete is stable enough — derived solely from t and deleteMut closures
      // eslint-disable-next-line react-hooks/exhaustive-deps
      [t, channelLabelMap]
    );

  return (
    <div>
      <FilterBar
        filterId="notification-template-filter"
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
        tableId="notification-templates"
        columns={columns}
        dataSource={
          (data?.items ?? []) as (NotificationTemplate & Record<string, unknown>)[]
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
        scroll={{ x: 1100 }}
        extraToolbarLeft={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditing(null);
              setFormOpen(true);
            }}
          >
            {t('notification.template.create')}
          </Button>
        }
      />

      <TemplateForm
        open={formOpen}
        initial={editing}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        onSaved={() => void refetch()}
      />
    </div>
  );
}
