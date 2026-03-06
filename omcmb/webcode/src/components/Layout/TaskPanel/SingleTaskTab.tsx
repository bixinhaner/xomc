import { useMemo } from 'react';
import { Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTaskStore } from '@/store/taskStore';
import type { SingleTask, TaskStatus } from '@/store/taskStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

export default function SingleTaskTab() {
  const singleTasks = useTaskStore((s) => s.singleTasks);
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

  const columns: ColumnsType<SingleTask> = useMemo(() => [
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
      title: t('alarm.deviceName'),
      dataIndex: 'deviceName',
      key: 'deviceName',
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
      title: t('task.progress'),
      key: 'progress',
      width: 100,
      render: (_: unknown, record: SingleTask) => (
        <span style={{ fontSize: 12, color: token.colorText }}>
          {record.currentStep}/{record.totalSteps}
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
      title: t('task.message'),
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
      render: (msg: string) => (
        <span style={{ fontSize: 12, color: token.colorText }}>{msg || '—'}</span>
      ),
    },
  ], [t, statusTagMap, token]);

  return (
    <Table<SingleTask>
      dataSource={singleTasks}
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
