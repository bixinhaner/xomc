import React, { useCallback, useMemo, useState } from 'react';
import { App, Button, Dropdown, Modal, Space, Steps, Tag, Typography } from 'antd';
import type { MenuProps } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  LoadingOutlined,
  MoreOutlined,
  PlusOutlined,
  RedoOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import AddDrawer from './AddDrawer';

const { Text } = Typography;

type StationType = 'eNB' | 'gNB' | 'GSM';
type TaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled';

interface CommissioningTask {
  id: string;
  stationCode: string;
  stationName: string;
  stationType: StationType;
  status: TaskStatus;
  currentStep: string;
  progress: number;
  operator: string;
  createTime: string;
  operateTime: string;
}

const MOCK_TASKS: CommissioningTask[] = [
  {
    id: '1', stationCode: 'BJ-CY-001', stationName: '北京朝阳站-001', stationType: 'eNB',
    currentStep: '参数配置', status: 'running', createTime: '2024-03-01 09:00:00',
    operateTime: '2024-03-01 09:32:00', operator: '张工', progress: 60,
  },
  {
    id: '2', stationCode: 'SH-PD-002', stationName: '上海浦东站-002', stationType: 'gNB',
    currentStep: '射频调试', status: 'success', createTime: '2024-03-01 08:00:00',
    operateTime: '2024-03-01 10:15:00', operator: '李工', progress: 100,
  },
  {
    id: '3', stationCode: 'GZ-TH-003', stationName: '广州天河站-003', stationType: 'eNB',
    currentStep: '设备注册', status: 'failed', createTime: '2024-03-01 07:30:00',
    operateTime: '2024-03-01 08:45:00', operator: '王工', progress: 20,
  },
  {
    id: '4', stationCode: 'SZ-NS-004', stationName: '深圳南山站-004', stationType: 'GSM',
    currentStep: '待开始', status: 'pending', createTime: '2024-03-01 10:00:00',
    operateTime: '2024-03-01 10:00:00', operator: '赵工', progress: 0,
  },
  {
    id: '5', stationCode: 'CD-WH-005', stationName: '成都武侯站-005', stationType: 'gNB',
    currentStep: '网络测试', status: 'running', createTime: '2024-03-01 08:30:00',
    operateTime: '2024-03-01 09:55:00', operator: '陈工', progress: 80,
  },
];

export default function Commissioning() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [tasks, setTasks] = useState<CommissioningTask[]>(MOCK_TASKS);
  const [isLoading] = useState(false);
  const [detailTask, setDetailTask] = useState<CommissioningTask | null>(null);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [addDrawerOpen, setAddDrawerOpen] = useState(false);
  const [editDrawerOpen, setEditDrawerOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState<CommissioningTask | null>(null);
  const [submitLoading, setSubmitLoading] = useState(false);

  const STATUS_CONFIG: Record<TaskStatus, { label: string; color: string; icon: React.ReactNode }> = useMemo(() => ({
    pending: { label: t('status.pending'), color: 'default', icon: null },
    running: { label: t('status.running'), color: 'processing', icon: <LoadingOutlined /> },
    success: { label: t('status.success'), color: 'success', icon: <CheckCircleOutlined /> },
    failed: { label: t('status.failed'), color: 'error', icon: <CloseCircleOutlined /> },
    cancelled: { label: t('status.cancelled'), color: 'default', icon: null },
  }), [t]);

  const STATION_TYPE_MAP: Record<StationType, { label: string; color: string }> = {
    eNB: { label: '4G (eNB)', color: 'blue' },
    gNB: { label: '5G (gNB)', color: 'green' },
    GSM: { label: '2G (GSM)', color: 'orange' },
  };

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'stationCode', label: '基站编码', type: 'input' },
    { name: 'stationName', label: '基站名称', type: 'input' },
    {
      name: 'stationType',
      label: '基站类型',
      type: 'select',
      options: [
        { label: '4G (eNB)', value: 'eNB' },
        { label: '5G (gNB)', value: 'gNB' },
        { label: '2G (GSM)', value: 'GSM' },
      ],
    },
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
    return tasks.filter((task) => {
      if (filterParams.status && task.status !== filterParams.status) return false;
      if (filterParams.stationType && task.stationType !== filterParams.stationType) return false;
      if (filterParams.stationCode && !task.stationCode.toLowerCase().includes(String(filterParams.stationCode).toLowerCase())) return false;
      if (filterParams.stationName && !task.stationName.toLowerCase().includes(String(filterParams.stationName).toLowerCase())) return false;
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

  // 处理新增提交
  const handleAddSubmit = useCallback((values: Record<string, unknown>) => {
    setSubmitLoading(true);
    console.log('新增开通任务:', values);
    // 模拟API调用
    setTimeout(() => {
      message.success(t('common.success'));
      setSubmitLoading(false);
      setAddDrawerOpen(false);
    }, 1000);
  }, [t]);

  // 处理编辑
  const handleEdit = useCallback((record: CommissioningTask) => {
    setSelectedTask(record);
    setEditDrawerOpen(true);
  }, []);

  // 处理编辑提交
  const handleEditSubmit = useCallback((values: Record<string, unknown>) => {
    setSubmitLoading(true);
    console.log('编辑开通任务:', values);
    setTimeout(() => {
      message.success(t('common.success'));
      setSubmitLoading(false);
      setEditDrawerOpen(false);
      setSelectedTask(null);
    }, 1000);
  }, [t, message]);

  // 处理重新执行（仅失败状态）
  const handleRetry = useCallback((record: CommissioningTask) => {
    modal.confirm({
      title: '确认重新执行',
      content: `确定要重新执行开通任务 "${record.stationName}" 吗？`,
      onOk: () => {
        setTasks((prev) =>
          prev.map((tk) =>
            tk.id === record.id ? { ...tk, status: 'running', progress: 0, currentStep: 'Step 1' } : tk
          )
        );
        message.success('任务已重新执行');
      },
    });
  }, [modal, message]);

  // 处理删除
  const handleDelete = useCallback((record: CommissioningTask) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: `确定要删除开通任务 "${record.stationName}" 吗？`,
      onOk: () => {
        setTasks((prev) => prev.filter((tk) => tk.id !== record.id));
        message.success(t('common.deleteSuccess'));
      },
    });
  }, [modal, message, t]);

  const columns = useMemo(
    (): DataTableColumn<CommissioningTask>[] => [
      // 操作列放在最前面
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 100,
        fixed: 'right',
        render: (_val, record) => {
          const moreItems: MenuProps['items'] = [
            ...(record.status === 'failed'
              ? [{ key: 'retry', label: '重新执行', icon: <RedoOutlined />, onClick: () => handleRetry(record) }]
              : []),
            { type: 'divider' as const },
            { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, disabled: record.status === 'running', onClick: () => handleDelete(record) },
          ];
          return (
            <Space size={4}>
              <Button
                type="link"
                size="small"
                onClick={() => handleEdit(record)}
                disabled={record.status === 'running'}
              >
                {t('common.edit')}
              </Button>
              <Dropdown
                menu={{ items: moreItems }}
                trigger={['click']}
              >
                <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
              </Dropdown>
            </Space>
          );
        },
      },
      {
        key: 'stationCode',
        title: '基站编码',
        dataIndex: 'stationCode',
        width: 140,
        mono: true,
        render: (v, record) => (
          <Button
            type="link"
            size="small"
            style={{ fontFamily: 'monospace', fontSize: 12, padding: 0 }}
            onClick={() => setDetailTask(record)}
          >
            {String(v)}
          </Button>
        ),
      },
      {
        key: 'stationName',
        title: '基站名称',
        dataIndex: 'stationName',
        width: 180,
        ellipsis: true,
      },
      {
        key: 'stationType',
        title: '基站类型',
        dataIndex: 'stationType',
        width: 110,
        render: (v) => {
          const typeConfig = STATION_TYPE_MAP[v as StationType];
          return <Tag color={typeConfig?.color}>{typeConfig?.label || v}</Tag>;
        },
      },
      {
        key: 'status',
        title: t('table.status'),
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
        key: 'operator',
        title: t('table.operator'),
        dataIndex: 'operator',
        width: 90,
      },
      {
        key: 'createTime',
        title: t('table.createTime'),
        dataIndex: 'createTime',
        width: 160,
      },
      {
        key: 'operateTime',
        title: '操作时间',
        dataIndex: 'operateTime',
        width: 160,
      },
    ],
    [t, STATUS_CONFIG, STATION_TYPE_MAP, handleEdit, handleRetry, handleDelete]
  );

  return (
    <>
      <ListPageLayout
        title={t('nav.device.commission')}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddDrawerOpen(true)}>
            {t('common.add')}
          </Button>
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
          defaultDensity="default"
        />
      </ListPageLayout>

      {/* Detail Modal */}
      <Modal
        title={`开站详情 - ${detailTask?.stationName ?? ''}`}
        open={Boolean(detailTask)}
        onCancel={() => setDetailTask(null)}
        footer={<Button onClick={() => setDetailTask(null)}>{t('common.close')}</Button>}
        width={640}
      >
        {detailTask && (
          <div style={{ padding: '16px 0' }}>
            <div style={{ padding: '12px 16px', background: '#fafafa', borderRadius: 6 }}>
              <table style={{ width: '100%', fontSize: 13 }}>
                <tbody>
                  {[
                    { label: '基站编码', value: detailTask.stationCode },
                    { label: '基站名称', value: detailTask.stationName },
                    { label: '基站类型', value: STATION_TYPE_MAP[detailTask.stationType]?.label },
                    { label: t('table.status'), value: STATUS_CONFIG[detailTask.status]?.label },
                    { label: t('table.operator'), value: detailTask.operator },
                    { label: t('table.createTime'), value: detailTask.createTime },
                    { label: '操作时间', value: detailTask.operateTime },
                  ].map(({ label, value }) => (
                    <tr key={label}>
                      <td style={{ padding: '8px 0', color: '#8c8c8c', width: 100 }}>{label}</td>
                      <td style={{ padding: '8px 0', fontWeight: 500 }}>{value}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </Modal>

      {/* Add Drawer */}
      <AddDrawer
        open={addDrawerOpen}
        onClose={() => setAddDrawerOpen(false)}
        onSubmit={handleAddSubmit}
        loading={submitLoading}
      />

      {/* Edit Drawer */}
      <AddDrawer
        open={editDrawerOpen}
        onClose={() => {
          setEditDrawerOpen(false);
          setSelectedTask(null);
        }}
        onSubmit={handleEditSubmit}
        loading={submitLoading}
        initialValues={selectedTask ? {
          stationCode: selectedTask.stationCode,
          stationName: selectedTask.stationName,
          stationType: selectedTask.stationType,
        } : undefined}
        mode="edit"
      />
    </>
  );
}
