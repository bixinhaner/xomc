import { useMemo, useState } from 'react';
import { Button, Descriptions, Modal, Table, Tag } from 'antd';
import { EyeOutlined, DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTaskStore } from '@core/store/taskStore';
import type { SingleTask, TaskStatus } from '@core/store/taskStore';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

export default function SingleTaskTab() {
  const singleTasks = useTaskStore((s) => s.singleTasks);
  const t = useT();
  const token = useThemeToken();
  const [viewModalOpen, setViewModalOpen] = useState(false);
  const [currentTask, setCurrentTask] = useState<SingleTask | null>(null);

  const statusTagMap: Record<TaskStatus, { color: string; text: string }> = useMemo(() => ({
    pending:   { color: 'default',    text: t('status.pending') },
    running:   { color: 'processing', text: t('task.status.running') },
    success:   { color: 'success',    text: t('task.status.completed') },
    failed:    { color: 'error',      text: t('task.status.failed') },
    cancelled: { color: 'default',    text: t('status.cancelled') },
    paused:    { color: 'warning',    text: t('status.pending') },
  }), [t]);

  const handleViewTask = (record: SingleTask) => {
    setCurrentTask(record);
    setViewModalOpen(true);
  };

  const handleDownload = () => {
    // TODO: 实现下载功能
    console.log('Download file for task:', currentTask);
  };

  // 根据任务类型生成模拟的文件信息
  const getFileInfo = (task: SingleTask) => {
    const timestamp = new Date(task.updatedAt).toISOString().replace(/[:.]/g, '-').slice(0, 19);
    if (task.type.includes(t('device.action.tr069Collect'))) {
      return {
        fileName: `tr069_${task.neSn}_${timestamp}.xml`,
        fileSize: `${Math.floor(Math.random() * 500 + 100)} KB`,
        fileType: 'XML',
      };
    } else if (task.type.includes(t('device.action.logCollect'))) {
      return {
        fileName: `log_${task.neSn}_${timestamp}.tar.gz`,
        fileSize: `${Math.floor(Math.random() * 2000 + 500)} KB`,
        fileType: 'TAR.GZ',
      };
    }
    return null;
  };

  const columns: ColumnsType<SingleTask> = useMemo(() => [
    {
      title: 'SN',
      dataIndex: 'neSn',
      key: 'neSn',
      width: 140,
      ellipsis: true,
      render: (sn: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{sn}</span>
      ),
    },
    {
      title: t('alarm.deviceName'),
      dataIndex: 'neName',
      key: 'neName',
      ellipsis: true,
      width: 120,
    },
    {
      title: t('table.type'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      ellipsis: true,
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
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
      title: t('table.action'),
      key: 'action',
      width: 60,
      render: (_: unknown, record: SingleTask) => {
        // 只有完成的 TR069 收集和日志收集任务才显示查看按钮
        const isViewable =
          record.status === 'success' &&
          (record.type.includes(t('device.action.tr069Collect')) ||
           record.type.includes(t('device.action.logCollect')));
        if (!isViewable) return null;
        return (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewTask(record)}
            style={{ padding: 0 }}
          >
            {t('task.view')}
          </Button>
        );
      },
    },
  ], [t, statusTagMap]);

  const fileInfo = currentTask ? getFileInfo(currentTask) : null;
  const statusConfig = currentTask ? statusTagMap[currentTask.status] : null;

  return (
    <>
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

      <Modal
        title={t('task.detail')}
        open={viewModalOpen}
        onCancel={() => setViewModalOpen(false)}
        footer={[
          <Button key="download" type="primary" icon={<DownloadOutlined />} onClick={handleDownload}>
            {t('common.download')}
          </Button>,
          <Button key="close" onClick={() => setViewModalOpen(false)}>
            {t('common.close')}
          </Button>,
        ]}
        width={520}
      >
        {currentTask && (
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="SN" span={2}>
              <span style={{ fontFamily: 'monospace' }}>{currentTask.neSn}</span>
            </Descriptions.Item>
            <Descriptions.Item label={t('alarm.deviceName')} span={2}>
              {currentTask.neName}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.type')}>
              {currentTask.type}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.status')}>
              {statusConfig && (
                <Tag color={statusConfig.color}>{statusConfig.text}</Tag>
              )}
            </Descriptions.Item>
            <Descriptions.Item label={t('task.startTime')}>
              {new Date(currentTask.createdAt).toLocaleString('zh-CN')}
            </Descriptions.Item>
            <Descriptions.Item label={t('task.endTime')}>
              {new Date(currentTask.updatedAt).toLocaleString('zh-CN')}
            </Descriptions.Item>
            {fileInfo && (
              <>
                <Descriptions.Item label={t('task.fileName')}>
                  {fileInfo.fileName}
                </Descriptions.Item>
                <Descriptions.Item label={t('task.fileSize')}>
                  {fileInfo.fileSize}
                </Descriptions.Item>
              </>
            )}
          </Descriptions>
        )}
      </Modal>
    </>
  );
}
