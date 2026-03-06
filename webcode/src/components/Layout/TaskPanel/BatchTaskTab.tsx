import { useMemo } from 'react';
import { Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTaskStore } from '@/store/taskStore';
import type { BatchTask, TaskStatus } from '@/store/taskStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

export default function BatchTaskTab() {
  const batchTasks = useTaskStore((s) => s.batchTasks);
  const t = useT();
  const token = useThemeToken();

  const statusTagMap: Record<TaskStatus, { color: string; text: string }> = useMemo(() => ({
    pending:   { color: 'default',    text: t('status.pending') },
    running:   { color: 'processing', text: t('status.running') },
    success:   { color: 'success',    text: t('status.success') },
    failed:    { color: 'error',      text: t('status.failed') },
    cancelled: { color: 'default',    text: t('status.cancelled') },
    paused:    { color: 'warning',    text: t('status.pending') },
  }), [t]);

  const columns: ColumnsType<BatchTask> = useMemo(() => [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
      ellipsis: true,
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{id.slice(0, 8)}</span>
      ),
    },
    {
      title: t('table.name'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      width: 120,
    },
    {
      title: t('table.type'),
      dataIndex: 'type',
      key: 'type',
      width: 80,
      ellipsis: true,
    },
    {
      title: `${t('table.total')}/${t('table.success')}/${t('table.failed')}`,
      key: 'counts',
      width: 120,
      render: (_: unknown, record: BatchTask) => (
        <span style={{ fontSize: 12 }}>
          <span style={{ color: token.colorText }}>{record.total}</span>
          {' / '}
          <span style={{ color: '#52c41a' }}>{record.success}</span>
          {' / '}
          <span style={{ color: record.failed > 0 ? '#f5222d' : token.colorTextDisabled }}>
            {record.failed}
          </span>
        </span>
      ),
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: TaskStatus) => {
        const cfg = statusTagMap[status];
        return (
          <Tag color={cfg.color} style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
            {cfg.text}
          </Tag>
        );
      },
    },
    {
      title: t('table.createTime'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      ellipsis: true,
      width: 140,
      render: (time: string) => (
        <span style={{ fontSize: 11, color: token.colorTextSecondary }}>{time}</span>
      ),
    },
  ], [t, statusTagMap, token]);

  return (
    <Table<BatchTask>
      dataSource={batchTasks}
      columns={columns}
      rowKey="id"
      size="small"
      pagination={false}
      scroll={{ y: 140 }}
      locale={{ emptyText: t('common.noData') }}
      style={{ fontSize: 12 }}
    />
  );
}
