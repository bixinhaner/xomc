import { useState, useMemo } from 'react';
import { Button, Descriptions, Popconfirm, Progress, Space, Tag, Timeline, Typography, message } from 'antd';
import { DownloadOutlined, DeleteOutlined, StopOutlined } from '@ant-design/icons';
import SplitPanelLayout from '@/components/Layout/SplitPanelLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useBackupTasks, useDeleteBackupTasks, useCancelBackupTask } from '@/hooks/api/useBackup';
import type { BackupTask } from '@/mock/data/backup';
import { useT } from '@/hooks/useT';

interface BackupTaskRow extends Record<string, unknown> {
  id: string;
  taskName: string;
  taskType: 'manual' | 'scheduled';
  backupType: 'full' | 'incremental' | 'config-only';
  deviceRange: string;
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
  progress: number;
  startTime: string;
  endTime: string;
  fileSize: number;
  creator: string;
}

const STATUS_MAP_KEYS: Record<string, { color: string; key: string }> = {
  pending: { color: 'default', key: 'status.pending' },
  running: { color: 'processing', key: 'status.running' },
  success: { color: 'success', key: 'status.success' },
  failed: { color: 'error', key: 'status.failed' },
  cancelled: { color: 'warning', key: 'status.cancelled' },
};

const BACKUP_TYPE_MAP: Record<string, string> = {
  full: '全量备份',
  incremental: '增量备份',
  'config-only': '配置备份',
};

const TASK_TYPE_MAP: Record<string, string> = {
  manual: '手动',
  scheduled: '计划',
};

function formatBytes(bytes?: number): string {
  if (!bytes || bytes === 0) return '-';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

const mockTasks: BackupTaskRow[] = [
  { id: 'bkp-001', taskName: '全量备份-北京站点', taskType: 'manual', backupType: 'full', deviceRange: 'ENB00001, ENB00002, ENB00003, GNB00001', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:45:32', fileSize: 1024 * 1024 * 256, creator: 'admin' },
  { id: 'bkp-002', taskName: '计划备份-全网每日', taskType: 'scheduled', backupType: 'full', deviceRange: '全部设备 (18台)', status: 'running', progress: 65, startTime: '2026-03-02 02:00:00', endTime: '-', fileSize: 0, creator: 'system' },
  { id: 'bkp-003', taskName: '配置备份-5G基站', taskType: 'manual', backupType: 'config-only', deviceRange: 'GNB00001, GNB00002', status: 'failed', progress: 40, startTime: '2026-03-01 15:30:00', endTime: '2026-03-01 15:38:12', fileSize: 0, creator: 'operator1' },
  { id: 'bkp-004', taskName: '增量备份-上海站点', taskType: 'scheduled', backupType: 'incremental', deviceRange: 'ENB00003', status: 'success', progress: 100, startTime: '2026-03-01 04:00:00', endTime: '2026-03-01 04:12:05', fileSize: 1024 * 1024 * 32, creator: 'system' },
  { id: 'bkp-005', taskName: '手动备份-单台设备', taskType: 'manual', backupType: 'config-only', deviceRange: 'ENB00001', status: 'cancelled', progress: 20, startTime: '2026-02-28 09:00:00', endTime: '2026-02-28 09:05:00', fileSize: 0, creator: 'operator2' },
];

export default function BackupTasks() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedTask, setSelectedTask] = useState<BackupTaskRow | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'taskName', label: t('table.name'), type: 'input' },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'running' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.cancelled'), value: 'cancelled' },
      ],
    },
    {
      name: 'taskType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: t('common.execute'), value: 'manual' },
        { label: t('common.deploy'), value: 'scheduled' },
      ],
    },
  ], [t]);

  const { data, isLoading, refetch } = useBackupTasks({ page, pageSize });
  const deleteTasksMut = useDeleteBackupTasks();
  const cancelTask = useCancelBackupTask();

  void (null as unknown as BackupTask);

  const tableSource = (data?.items ?? mockTasks) as unknown as BackupTaskRow[];

  const filteredSource = tableSource.filter((row) => {
    if (filters.taskName && !row.taskName.includes(filters.taskName as string)) return false;
    if (filters.status && row.status !== filters.status) return false;
    if (filters.taskType && row.taskType !== filters.taskType) return false;
    return true;
  });

  const columns: DataTableColumn<BackupTaskRow>[] = useMemo(() => [
    { key: 'taskName', title: t('table.name'), dataIndex: 'taskName', width: 220, ellipsis: true },
    {
      key: 'taskType',
      title: t('table.type'),
      dataIndex: 'taskType',
      width: 90,
      render: (val) => <Tag>{TASK_TYPE_MAP[val as string] ?? (val as string)}</Tag>,
    },
    {
      key: 'backupType',
      title: t('table.type'),
      dataIndex: 'backupType',
      width: 110,
      render: (val) => <Tag color="blue">{BACKUP_TYPE_MAP[val as string] ?? (val as string)}</Tag>,
    },
    { key: 'deviceRange', title: t('table.description'), dataIndex: 'deviceRange', width: 200, ellipsis: true },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP_KEYS[val as string] ?? STATUS_MAP_KEYS.pending;
        return <Tag color={cfg.color}>{t(cfg.key)}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('table.result'),
      dataIndex: 'progress',
      width: 120,
      render: (val, record) =>
        record.status === 'running' ? (
          <Progress percent={val as number} size="small" />
        ) : (
          <span>{val as number}%</span>
        ),
    },
    { key: 'startTime', title: t('table.createTime'), dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: t('table.updateTime'), dataIndex: 'endTime', width: 160 },
    {
      key: 'fileSize',
      title: t('table.description'),
      dataIndex: 'fileSize',
      width: 100,
      render: (val) => formatBytes(val as number),
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          {record.status === 'success' && (
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              onClick={() => void message.success(`开始下载备份文件: ${record.taskName as string}`)}
            >
              {t('common.download')}
            </Button>
          )}
          {record.status === 'running' && (
            <Button
              type="link"
              size="small"
              icon={<StopOutlined />}
              danger
              onClick={() => cancelTask.mutate(record.id)}
            >
              {t('common.cancel')}
            </Button>
          )}
          <Popconfirm
            title={t('common.confirmDelete')}
            onConfirm={() => deleteTasksMut.mutate([record.id])}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [t]);

  const upperPanel = (
    <div style={{ padding: 12 }}>
      <FilterBar
        filterId="backup-tasks"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<BackupTaskRow>
        tableId="backup-tasks"
        columns={columns}
        dataSource={filteredSource}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => {
          setSelectedKeys(keys);
          if (keys.length > 0) {
            const task = filteredSource.find((t) => t.id === keys[0]);
            if (task) setSelectedTask(task);
          }
        }}
        total={filteredSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        onExport={() => void message.info(t('common.exportInProgress'))}
        batchActions={[
          {
            key: 'delete',
            label: t('common.batchDelete'),
            danger: true,
            onClick: (keys) => {
              deleteTasksMut.mutate(keys as string[], {
                onSuccess: () => { void message.success(t('common.deleteSuccess')); setSelectedKeys([]); },
              });
            },
          },
        ]}
        scroll={{ x: 1600 }}
      />
    </div>
  );

  const lowerPanel = (
    <div style={{ padding: '12px 16px' }}>
      {selectedTask ? (
        <>
          <Typography.Text strong style={{ fontSize: 13, display: 'block', marginBottom: 12 }}>
            {t('common.detail')} — {selectedTask.taskName}
          </Typography.Text>
          <Descriptions bordered size="small" column={3}>
            <Descriptions.Item label={t('table.index')}>{selectedTask.id}</Descriptions.Item>
            <Descriptions.Item label={t('table.type')}>{TASK_TYPE_MAP[selectedTask.taskType]}</Descriptions.Item>
            <Descriptions.Item label={t('table.type')}>{BACKUP_TYPE_MAP[selectedTask.backupType]}</Descriptions.Item>
            <Descriptions.Item label={t('table.description')} span={2}>{selectedTask.deviceRange}</Descriptions.Item>
            <Descriptions.Item label={t('table.operator')}>{selectedTask.creator}</Descriptions.Item>
            <Descriptions.Item label={t('table.createTime')}>{selectedTask.startTime}</Descriptions.Item>
            <Descriptions.Item label={t('table.updateTime')}>{selectedTask.endTime}</Descriptions.Item>
            <Descriptions.Item label={t('table.description')}>{formatBytes(selectedTask.fileSize)}</Descriptions.Item>
          </Descriptions>

          <Typography.Text strong style={{ fontSize: 12, display: 'block', marginTop: 16, marginBottom: 8 }}>
            {t('table.result')}
          </Typography.Text>
          <Timeline
            items={[
              { color: 'blue', children: <span style={{ fontSize: 12 }}>[{selectedTask.startTime}] 备份任务开始执行</span> },
              { color: 'blue', children: <span style={{ fontSize: 12 }}>正在连接目标设备...</span> },
              { color: selectedTask.status === 'success' ? 'green' : selectedTask.status === 'failed' ? 'red' : 'gray',
                children: <span style={{ fontSize: 12 }}>
                  {selectedTask.status === 'success' ? `[${selectedTask.endTime}] 备份完成，文件大小: ${formatBytes(selectedTask.fileSize)}`
                    : selectedTask.status === 'failed' ? `[${selectedTask.endTime}] 备份失败，请检查设备连接状态`
                    : selectedTask.status === 'cancelled' ? `[${selectedTask.endTime}] 备份已取消`
                    : '执行中...'}
                </span> },
            ]}
          />
        </>
      ) : (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 100 }}>
          <Typography.Text type="secondary">{t('common.pleaseSelect')}</Typography.Text>
        </div>
      )}
    </div>
  );

  return (
    <SplitPanelLayout
      upper={upperPanel}
      lower={lowerPanel}
      upperTitle={t('nav.backup.tasks')}
      lowerTitle={t('common.detail')}
      defaultSplitRatio={0.65}
    />
  );
}
