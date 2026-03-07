import React, { useCallback, useMemo, useState } from 'react';
import { Button, Modal, Space, Steps, Tag, Typography, message } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EyeOutlined,
  LoadingOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface CommissioningTask {
  id: string;
  taskId: string;
  siteName: string;
  deviceSn: string;
  currentStep: string;
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
  createTime: string;
  updateTime: string;
  operator: string;
  progress: number;
}

const MOCK_TASKS: CommissioningTask[] = [
  {
    id: '1', taskId: 'COMM-2024-001', siteName: '北京朝阳站-001', deviceSn: 'SN-BJ001',
    currentStep: '参数配置', status: 'running', createTime: '2024-03-01 09:00:00',
    updateTime: '2024-03-01 09:32:00', operator: '张工', progress: 60,
  },
  {
    id: '2', taskId: 'COMM-2024-002', siteName: '上海浦东站-002', deviceSn: 'SN-SH002',
    currentStep: '射频调试', status: 'success', createTime: '2024-03-01 08:00:00',
    updateTime: '2024-03-01 10:15:00', operator: '李工', progress: 100,
  },
  {
    id: '3', taskId: 'COMM-2024-003', siteName: '广州天河站-003', deviceSn: 'SN-GZ003',
    currentStep: '设备注册', status: 'failed', createTime: '2024-03-01 07:30:00',
    updateTime: '2024-03-01 08:45:00', operator: '王工', progress: 20,
  },
  {
    id: '4', taskId: 'COMM-2024-004', siteName: '深圳南山站-004', deviceSn: 'SN-SZ004',
    currentStep: '待开始', status: 'pending', createTime: '2024-03-01 10:00:00',
    updateTime: '2024-03-01 10:00:00', operator: '赵工', progress: 0,
  },
  {
    id: '5', taskId: 'COMM-2024-005', siteName: '成都武侯站-005', deviceSn: 'SN-CD005',
    currentStep: '网络测试', status: 'running', createTime: '2024-03-01 08:30:00',
    updateTime: '2024-03-01 09:55:00', operator: '陈工', progress: 80,
  },
];

const COMMISSIONING_STEPS = ['Step 1', 'Step 2', 'Step 3', 'Step 4', 'Step 5', 'Step 6'];

export default function Commissioning() {
  const t = useT();
  const [tasks, setTasks] = useState<CommissioningTask[]>(MOCK_TASKS);
  const [isLoading] = useState(false);
  const [detailTask, setDetailTask] = useState<CommissioningTask | null>(null);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);

  const STATUS_CONFIG: Record<CommissioningTask['status'], { label: string; color: string; icon: React.ReactNode }> = useMemo(() => ({
    pending: { label: t('status.pending'), color: 'default', icon: null },
    running: { label: t('status.running'), color: 'processing', icon: <LoadingOutlined /> },
    success: { label: t('status.success'), color: 'success', icon: <CheckCircleOutlined /> },
    failed: { label: t('status.failed'), color: 'error', icon: <CloseCircleOutlined /> },
    cancelled: { label: t('status.cancelled'), color: 'default', icon: null },
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'taskId', label: t('table.index'), type: 'input' },
    { name: 'siteName', label: t('table.site'), type: 'input' },
    { name: 'deviceSn', label: t('device.sn'), type: 'input' },
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
    { name: 'timeRange', label: t('table.time'), type: 'date-range' },
  ], [t]);

  const filteredTasks = useMemo(() => {
    return tasks.filter((t) => {
      if (filterParams.status && t.status !== filterParams.status) return false;
      if (filterParams.taskId && !t.taskId.toLowerCase().includes(String(filterParams.taskId).toLowerCase())) return false;
      if (filterParams.siteName && !t.siteName.toLowerCase().includes(String(filterParams.siteName).toLowerCase())) return false;
      if (filterParams.deviceSn && !t.deviceSn.toLowerCase().includes(String(filterParams.deviceSn).toLowerCase())) return false;
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

  const columns = useMemo(
    (): DataTableColumn<CommissioningTask>[] => [
      {
        key: 'taskId',
        title: t('table.index'),
        dataIndex: 'taskId',
        width: 150,
        mono: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      { key: 'siteName', title: t('table.site'), dataIndex: 'siteName', width: 180, ellipsis: true },
      {
        key: 'deviceSn',
        title: t('device.sn'),
        dataIndex: 'deviceSn',
        width: 150,
        mono: true,
        copyable: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</Text>,
      },
      { key: 'currentStep', title: t('table.status'), dataIndex: 'currentStep', width: 120 },
      {
        key: 'status',
        title: t('table.result'),
        dataIndex: 'status',
        width: 100,
        render: (_val, record) => {
          const cfg = STATUS_CONFIG[record.status];
          return (
            <Tag color={cfg.color} icon={cfg.icon}>
              {cfg.label}
            </Tag>
          );
        },
      },
      {
        key: 'createTime',
        title: t('table.createTime'),
        dataIndex: 'createTime',
        width: 160,
      },
      { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 90 },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 160,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setDetailTask(record)}
            >
              {t('common.detail')}
            </Button>
            {record.status === 'pending' && (
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setTasks((prev) =>
                    prev.map((tk) =>
                      tk.id === record.id ? { ...tk, status: 'running', currentStep: 'Step 1' } : tk
                    )
                  );
                  void message.success(t('common.execute'));
                }}
              >
                {t('common.execute')}
              </Button>
            )}
            {record.status === 'failed' && (
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setTasks((prev) =>
                    prev.map((tk) =>
                      tk.id === record.id ? { ...tk, status: 'running', progress: 0 } : tk
                    )
                  );
                  void message.info(t('common.refresh'));
                }}
              >
                {t('common.refresh')}
              </Button>
            )}
            {record.status === 'running' && (
              <Button
                type="link"
                size="small"
                danger
                onClick={() => {
                  setTasks((prev) =>
                    prev.map((tk) =>
                      tk.id === record.id ? { ...tk, status: 'cancelled' } : tk
                    )
                  );
                  void message.warning(t('common.cancel'));
                }}
              >
                {t('common.cancel')}
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [t, STATUS_CONFIG]
  );

  const currentStepIndex = detailTask
    ? COMMISSIONING_STEPS.indexOf(detailTask.currentStep)
    : -1;

  return (
    <>
      <ListPageLayout
        title={t('nav.device.commission')}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />}>{t('common.refresh')}</Button>
            <Button type="primary" icon={<PlusOutlined />}>
              {t('common.add')}
            </Button>
          </Space>
        }
      >
        <FilterBar
          filterId="commissioning"
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />

        <DataTable<CommissioningTask>
          tableId="commissioning-table"
          columns={columns}
          dataSource={filteredTasks}
          loading={isLoading}
          rowKey="id"
          total={filteredTasks.length}
          pageSize={pageSize}
          currentPage={currentPage}
          onPageChange={(page) => setCurrentPage(page)}
          defaultDensity="compact"
        />
      </ListPageLayout>

      {/* Detail Modal */}
      <Modal
        title={`${t('common.detail')} - ${detailTask?.taskId ?? ''}`}
        open={Boolean(detailTask)}
        onCancel={() => setDetailTask(null)}
        footer={<Button onClick={() => setDetailTask(null)}>{t('common.close')}</Button>}
        width={640}
      >
        {detailTask && (
          <div style={{ padding: '16px 0' }}>
            <Steps
              direction="vertical"
              size="small"
              current={currentStepIndex >= 0 ? currentStepIndex : 0}
              status={
                detailTask.status === 'failed'
                  ? 'error'
                  : detailTask.status === 'success'
                  ? 'finish'
                  : 'process'
              }
              items={COMMISSIONING_STEPS.map((step, idx) => ({
                title: step,
                description:
                  idx < currentStepIndex
                    ? t('status.success')
                    : idx === currentStepIndex
                    ? detailTask.status === 'failed'
                      ? t('status.failed')
                      : t('status.running')
                    : t('status.pending'),
              }))}
            />
            <div style={{ marginTop: 16, padding: '12px 16px', background: '#fafafa', borderRadius: 6 }}>
              <table style={{ width: '100%', fontSize: 13 }}>
                <tbody>
                  {[
                    { label: t('table.index'), value: detailTask.taskId },
                    { label: t('table.site'), value: detailTask.siteName },
                    { label: t('device.sn'), value: detailTask.deviceSn },
                    { label: t('table.operator'), value: detailTask.operator },
                    { label: t('table.createTime'), value: detailTask.createTime },
                    { label: t('table.updateTime'), value: detailTask.updateTime },
                  ].map(({ label, value }) => (
                    <tr key={label}>
                      <td style={{ padding: '4px 0', color: '#8c8c8c', width: 100 }}>{label}</td>
                      <td style={{ padding: '4px 0', fontWeight: 500 }}>{value}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </Modal>
    </>
  );
}
