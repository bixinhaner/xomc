import React, { useCallback, useMemo, useState } from 'react';
import { Button, Modal, Space, Tag, Typography, message } from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import { useTriggerAlarmSync } from '@/hooks/api/useAlarms';

const { Text } = Typography;

interface AlarmSyncTask {
  id: string;
  syncId: string;
  deviceSn: string;
  deviceName: string;
  syncType: 'full' | 'incremental' | 'realtime';
  startTime: string;
  endTime: string | null;
  status: 'running' | 'success' | 'failed' | 'cancelled';
  syncCount: number;
  failCount: number;
  operator: string;
}

const MOCK_SYNC_TASKS: AlarmSyncTask[] = [
  {
    id: '1', syncId: 'SYNC-2024-001', deviceSn: 'SN-BJ001', deviceName: '北京基站-001',
    syncType: 'full', startTime: '2024-03-01 09:00:00', endTime: '2024-03-01 09:05:30',
    status: 'success', syncCount: 125, failCount: 0, operator: '系统',
  },
  {
    id: '2', syncId: 'SYNC-2024-002', deviceSn: 'SN-SH002', deviceName: '上海基站-002',
    syncType: 'incremental', startTime: '2024-03-01 09:30:00', endTime: '2024-03-01 09:30:45',
    status: 'success', syncCount: 23, failCount: 0, operator: '系统',
  },
  {
    id: '3', syncId: 'SYNC-2024-003', deviceSn: 'SN-GZ003', deviceName: '广州基站-003',
    syncType: 'realtime', startTime: '2024-03-01 08:00:00', endTime: null,
    status: 'running', syncCount: 456, failCount: 2, operator: '系统',
  },
  {
    id: '4', syncId: 'SYNC-2024-004', deviceSn: 'SN-CD005', deviceName: '成都基站-005',
    syncType: 'full', startTime: '2024-03-01 07:30:00', endTime: '2024-03-01 07:38:12',
    status: 'failed', syncCount: 89, failCount: 15, operator: '张工',
  },
  {
    id: '5', syncId: 'SYNC-2024-005', deviceSn: 'SN-XA007', deviceName: '西安基站-007',
    syncType: 'incremental', startTime: '2024-03-01 10:00:00', endTime: '2024-03-01 10:00:18',
    status: 'success', syncCount: 7, failCount: 0, operator: '系统',
  },
  {
    id: '6', syncId: 'SYNC-2024-006', deviceSn: 'SN-NJ008', deviceName: '南京基站-008',
    syncType: 'full', startTime: '2024-03-01 06:00:00', endTime: '2024-03-01 06:02:45',
    status: 'cancelled', syncCount: 0, failCount: 0, operator: '李工',
  },
];

const SYNC_TYPE_COLOR: Record<AlarmSyncTask['syncType'], string> = {
  full: 'blue',
  incremental: 'green',
  realtime: 'purple',
};

export default function AlarmSync() {
  const t = useT();
  const triggerSync = useTriggerAlarmSync();
  const [tasks, setTasks] = useState<AlarmSyncTask[]>(MOCK_SYNC_TASKS);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [newSyncSn, setNewSyncSn] = useState('');
  const [newSyncType, setNewSyncType] = useState<AlarmSyncTask['syncType']>('incremental');

  const SYNC_TYPE_LABEL: Record<AlarmSyncTask['syncType'], string> = useMemo(() => ({
    full: 'Full',
    incremental: 'Incremental',
    realtime: 'Realtime',
  }), []);

  const STATUS_CONFIG: Record<AlarmSyncTask['status'], { label: string; color: string }> = useMemo(() => ({
    running: { label: t('status.running'), color: 'processing' },
    success: { label: t('status.success'), color: 'success' },
    failed: { label: t('status.failed'), color: 'error' },
    cancelled: { label: t('status.cancelled'), color: 'default' },
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'syncId', label: t('table.index'), type: 'input' },
    { name: 'deviceSn', label: t('alarm.deviceSn'), type: 'input' },
    {
      name: 'syncType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: 'Full', value: 'full' },
        { label: 'Incremental', value: 'incremental' },
        { label: 'Realtime', value: 'realtime' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.running'), value: 'running' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.cancelled'), value: 'cancelled' },
      ],
    },
    { name: 'timeRange', label: t('table.time'), type: 'date-range' },
  ], [t]);

  const filteredTasks = useMemo(() => {
    return tasks.filter((tk) => {
      if (filterParams.syncId && !tk.syncId.toLowerCase().includes(String(filterParams.syncId).toLowerCase())) return false;
      if (filterParams.deviceSn && !tk.deviceSn.toLowerCase().includes(String(filterParams.deviceSn).toLowerCase())) return false;
      if (filterParams.syncType && tk.syncType !== filterParams.syncType) return false;
      if (filterParams.status && tk.status !== filterParams.status) return false;
      return true;
    });
  }, [tasks, filterParams]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleRetry = useCallback((id: string) => {
    setTasks((prev) =>
      prev.map((tk) =>
        tk.id === id
          ? { ...tk, status: 'running', startTime: new Date().toLocaleString('zh-CN'), endTime: null, syncCount: 0, failCount: 0 }
          : tk
      )
    );
    void message.info(t('status.running'));
    // Simulate completion after 2 seconds
    setTimeout(() => {
      setTasks((prev) =>
        prev.map((tk) =>
          tk.id === id
            ? { ...tk, status: 'success', endTime: new Date().toLocaleString('zh-CN'), syncCount: 45 }
            : tk
        )
      );
      void message.success(t('status.success'));
    }, 2000);
  }, [t]);

  const handleCancel = useCallback((id: string) => {
    setTasks((prev) =>
      prev.map((tk) =>
        tk.id === id
          ? { ...tk, status: 'cancelled', endTime: new Date().toLocaleString('zh-CN') }
          : tk
      )
    );
    void message.warning(t('status.cancelled'));
  }, [t]);

  const handleCreateSync = useCallback(() => {
    if (!newSyncSn.trim()) {
      void message.warning(t('common.placeholder'));
      return;
    }
    const newTask: AlarmSyncTask = {
      id: String(Date.now()),
      syncId: `SYNC-${String(tasks.length + 1).padStart(3, '0')}`,
      deviceSn: newSyncSn,
      deviceName: `Device-${newSyncSn}`,
      syncType: newSyncType,
      startTime: new Date().toLocaleString('zh-CN'),
      endTime: null,
      status: 'running',
      syncCount: 0,
      failCount: 0,
      operator: 'Admin',
    };
    setTasks((prev) => [newTask, ...prev]);
    setCreateModalOpen(false);
    setNewSyncSn('');

    // Call real API to trigger alarm sync
    triggerSync.mutate(newSyncSn, {
      onSuccess: () => {
        void message.success(t('status.success'));
        setTasks((prev) =>
          prev.map((tk) =>
            tk.id === newTask.id
              ? { ...tk, status: 'success', endTime: new Date().toLocaleString('zh-CN'), syncCount: 1 }
              : tk
          )
        );
      },
      onError: () => {
        void message.error(t('status.failed'));
        setTasks((prev) =>
          prev.map((tk) =>
            tk.id === newTask.id
              ? { ...tk, status: 'failed', endTime: new Date().toLocaleString('zh-CN'), failCount: 1 }
              : tk
          )
        );
      },
    });
  }, [newSyncSn, newSyncType, tasks.length, t, triggerSync]);

  const columns = useMemo(
    (): DataTableColumn<AlarmSyncTask>[] => [
      {
        key: 'syncId',
        title: t('table.index'),
        dataIndex: 'syncId',
        width: 150,
        mono: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      {
        key: 'deviceSn',
        title: t('alarm.deviceSn'),
        dataIndex: 'deviceSn',
        width: 150,
        mono: true,
        copyable: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      { key: 'deviceName', title: t('alarm.deviceName'), dataIndex: 'deviceName', width: 150, ellipsis: true },
      {
        key: 'syncType',
        title: t('table.type'),
        dataIndex: 'syncType',
        width: 110,
        render: (_val, record) => (
          <Tag color={SYNC_TYPE_COLOR[record.syncType]}>{SYNC_TYPE_LABEL[record.syncType]}</Tag>
        ),
      },
      { key: 'startTime', title: t('table.time'), dataIndex: 'startTime', width: 160 },
      {
        key: 'endTime',
        title: t('table.time'),
        dataIndex: 'endTime',
        width: 160,
        render: (v) => v ? String(v) : <Text type="secondary">{t('status.running')}</Text>,
      },
      {
        key: 'status',
        title: t('table.status'),
        dataIndex: 'status',
        width: 100,
        render: (_val, record) => {
          const cfg = STATUS_CONFIG[record.status];
          return <Tag color={cfg.color}>{cfg.label}</Tag>;
        },
      },
      {
        key: 'syncCount',
        title: t('table.success'),
        dataIndex: 'syncCount',
        width: 90,
        render: (v) => (
          <Text strong style={{ color: 'var(--color-primary-600)' }}>
            {String(v)}
          </Text>
        ),
      },
      {
        key: 'failCount',
        title: t('table.failed'),
        dataIndex: 'failCount',
        width: 90,
        render: (v) => {
          const num = Number(v);
          return (
            <Text strong style={{ color: num > 0 ? '#F5222D' : '#52C41A' }}>
              {num}
            </Text>
          );
        },
      },
      { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 90 },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            {record.status === 'failed' && (
              <Button
                type="link"
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => handleRetry(record.id)}
              >
                {t('common.refresh')}
              </Button>
            )}
            {record.status === 'running' && (
              <Button
                type="link"
                size="small"
                danger
                onClick={() => handleCancel(record.id)}
              >
                {t('common.cancel')}
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [handleRetry, handleCancel, t, STATUS_CONFIG, SYNC_TYPE_LABEL]
  );

  return (
    <>
      <ListPageLayout
        title={t('nav.alarm.sync')}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />}>{t('common.refresh')}</Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalOpen(true)}
            >
              {t('common.execute')}
            </Button>
          </Space>
        }
      >
        <FilterBar
          filterId="alarm-sync"
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />

        <DataTable<AlarmSyncTask>
          tableId="alarm-sync-table"
          columns={columns}
          dataSource={filteredTasks}
          loading={false}
          rowKey="id"
          total={filteredTasks.length}
          pageSize={20}
          currentPage={currentPage}
          onPageChange={(p) => setCurrentPage(p)}
          defaultDensity="default"
        />
      </ListPageLayout>

      <Modal
        title={t('nav.alarm.sync')}
        open={createModalOpen}
        onOk={handleCreateSync}
        onCancel={() => setCreateModalOpen(false)}
        okText={t('common.execute')}
        width={420}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16, marginTop: 16 }}>
          <div>
            <Text style={{ display: 'block', marginBottom: 6, fontSize: 13 }}>
              {t('alarm.deviceSn')} <Text type="danger">*</Text>
            </Text>
            <input
              value={newSyncSn}
              onChange={(e) => setNewSyncSn(e.target.value)}
              placeholder={t('common.placeholder')}
              style={{
                width: '100%',
                padding: '6px 12px',
                border: '1px solid #d9d9d9',
                borderRadius: 6,
                fontSize: 13,
                fontFamily: 'monospace',
                outline: 'none',
              }}
            />
          </div>
          <div>
            <Text style={{ display: 'block', marginBottom: 6, fontSize: 13 }}>
              {t('table.type')}
            </Text>
            <div style={{ display: 'flex', gap: 8 }}>
              {(['full', 'incremental', 'realtime'] as const).map((type) => (
                <Button
                  key={type}
                  type={newSyncType === type ? 'primary' : 'default'}
                  size="small"
                  onClick={() => setNewSyncType(type)}
                >
                  {SYNC_TYPE_LABEL[type]}
                </Button>
              ))}
            </div>
          </div>
        </div>
      </Modal>
    </>
  );
}
