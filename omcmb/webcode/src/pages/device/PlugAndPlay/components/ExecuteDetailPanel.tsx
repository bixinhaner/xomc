import React, { useMemo } from 'react';
import { Drawer, Table, Tag } from 'antd';
import type { TableColumnsType } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  ForwardOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface Props {
  taskId: string;
  onClose: () => void;
}

interface TaskRecord {
  id: string;
  progress: string;
  progressLabel: string;
  status: '0' | '1' | '2' | '3' | '4';
  startTime: string;
  endTime: string;
  failureReason: string;
}

// Mock data for task records
const MOCK_RECORDS: TaskRecord[] = [
  { id: '1', progress: '1', progressLabel: '软件升级', status: '0', startTime: '2026-04-07 10:00:00', endTime: '2026-04-07 10:05:00', failureReason: '' },
  { id: '2', progress: '2', progressLabel: 'License', status: '0', startTime: '2026-04-07 10:05:00', endTime: '2026-04-07 10:08:00', failureReason: '' },
  { id: '3', progress: '3', progressLabel: '参数自配置', status: '0', startTime: '2026-04-07 10:08:00', endTime: '2026-04-07 10:15:00', failureReason: '' },
];

const PROGRESS_MAP: Record<string, string> = {
  '1': '软件升级',
  '2': 'License',
  '3': '参数自配置',
  '4': 'Cell active',
};

const STATUS_CONFIG: Record<string, { color: string; icon: React.ReactNode }> = {
  '0': { color: 'success', icon: <CheckCircleOutlined /> },
  '1': { color: 'error', icon: <CloseCircleOutlined /> },
  '2': { color: 'processing', icon: <LoadingOutlined /> },
  '3': { color: 'default', icon: <ClockCircleOutlined /> },
  '4': { color: 'warning', icon: <ForwardOutlined /> },
};

export default function ExecuteDetailPanel({ taskId, onClose }: Props) {
  const t = useT();

  // Get records based on taskId
  // In real implementation, fetch from API based on taskId
  const records = useMemo(() => {
    return MOCK_RECORDS;
  }, [taskId]);

  // Get status text
  const getStatusText = (status: string) => {
    const map: Record<string, string> = {
      '0': t('status.success'),
      '1': t('status.failed'),
      '2': t('status.running'),
      '3': t('provision.pending'),
      '4': t('provision.skipped'),
    };
    return map[status] || status;
  };

  // Table columns
  const columns: TableColumnsType<TaskRecord> = useMemo(() => [
    {
      title: t('provision.progress'),
      dataIndex: 'progress',
      key: 'progress',
      width: 150,
      render: (_, record) => PROGRESS_MAP[record.progress] || record.progress,
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status) => {
        const cfg = STATUS_CONFIG[status] || STATUS_CONFIG['3'];
        return (
          <Tag color={cfg.color} icon={cfg.icon}>
            {getStatusText(status)}
          </Tag>
        );
      },
    },
    {
      title: t('provision.startTime'),
      dataIndex: 'startTime',
      key: 'startTime',
      width: 160,
    },
    {
      title: t('provision.endTime'),
      dataIndex: 'endTime',
      key: 'endTime',
      width: 160,
    },
    {
      title: t('provision.failureReason'),
      dataIndex: 'failureReason',
      key: 'failureReason',
      ellipsis: true,
    },
  ], [t]);

  return (
    <Drawer
      title={t('provision.executeDetail')}
      placement="right"
      width={600}
      open={true}
      onClose={onClose}
      styles={{ body: { padding: 16 } }}
    >
      <Table
        columns={columns}
        dataSource={records}
        rowKey="id"
        size="small"
        pagination={false}
      />
    </Drawer>
  );
}
