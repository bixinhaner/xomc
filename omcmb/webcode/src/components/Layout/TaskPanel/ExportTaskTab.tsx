import { useMemo } from 'react';
import { Table, Tag, Button } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownloadOutlined } from '@ant-design/icons';
import { useTaskStore } from '@core/store/taskStore';
import type { ExportTask, TaskStatus } from '@core/store/taskStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

export default function ExportTaskTab() {
  const exportTasks = useTaskStore((s) => s.exportTasks);
  const t = useT();
  const token = useThemeToken();

  const statusTagMap: Record<TaskStatus, { color: string; text: string }> = useMemo(() => ({
    pending:   { color: 'default',    text: t('status.pending') },
    running:   { color: 'processing', text: t('common.exportInProgress') },
    success:   { color: 'success',    text: t('status.success') },
    failed:    { color: 'error',      text: t('status.failed') },
    cancelled: { color: 'default',    text: t('status.cancelled') },
    paused:    { color: 'warning',    text: t('status.pending') },
  }), [t]);

  const columns: ColumnsType<ExportTask> = useMemo(() => [
    {
      title: t('table.type'),
      dataIndex: 'exportType',
      key: 'exportType',
      width: 100,
      ellipsis: true,
    },
    {
      title: t('table.name'),
      dataIndex: 'fileName',
      key: 'fileName',
      ellipsis: true,
      render: (name: string) => (
        <span style={{ fontSize: 12, color: token.colorTextHeading }}>{name}</span>
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
      title: t('task.startTime'),
      dataIndex: 'startTime',
      key: 'startTime',
      ellipsis: true,
      width: 140,
      render: (time: string) => (
        <span style={{ fontSize: 11, color: token.colorTextSecondary }}>{time}</span>
      ),
    },
    {
      title: t('table.operation'),
      key: 'action',
      width: 80,
      render: (_: unknown, record: ExportTask) => (
        <Button
          type="link"
          size="small"
          icon={<DownloadOutlined />}
          disabled={record.status !== 'success' || !record.downloadUrl}
          href={record.downloadUrl}
          download={record.fileName}
          style={{ fontSize: 12, padding: '0 4px' }}
        >
          {t('common.download')}
        </Button>
      ),
    },
  ], [t, statusTagMap, token]);

  return (
    <Table<ExportTask>
      dataSource={exportTasks}
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
