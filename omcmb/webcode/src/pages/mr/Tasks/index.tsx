import { useState, useMemo } from 'react';
import { Button, Card, Dropdown, Tag, Space, Progress, Modal, Form, Input, Select, message, DatePicker } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, PlayCircleOutlined, PauseCircleOutlined, DeleteOutlined, EyeOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type TaskStatus = 'running' | 'paused' | 'completed' | 'failed' | 'pending';

interface MRTask {
  id: string;
  taskName: string;
  collectType: string;
  deviceRange: number;
  schedule: string;
  status: TaskStatus;
  progress?: number;
  lastRun?: string;
  nextRun?: string;
  creator: string;
}

const mockTasks: MRTask[] = [
  { id: 'mrt-001', taskName: '全网eNB MRO周期采集', collectType: 'MRO', deviceRange: 150, schedule: '每15分钟', status: 'running', progress: 72, lastRun: '2024-06-01T08:00:00.000Z', nextRun: '2024-06-01T08:15:00.000Z', creator: 'admin' },
  { id: 'mrt-002', taskName: '北京eNB MRE切换分析', collectType: 'MRE', deviceRange: 35, schedule: '每小时', status: 'running', progress: 45, lastRun: '2024-06-01T08:00:00.000Z', nextRun: '2024-06-01T09:00:00.000Z', creator: 'admin' },
  { id: 'mrt-003', taskName: 'gNB MRS统计采集', collectType: 'MRS', deviceRange: 18, schedule: '每天01:00', status: 'paused', lastRun: '2024-05-31T01:00:00.000Z', creator: 'operator1' },
  { id: 'mrt-004', taskName: '全网MRO日报汇总', collectType: 'MRO', deviceRange: 215, schedule: '每天02:00', status: 'completed', lastRun: '2024-06-01T02:00:00.000Z', creator: 'admin' },
  { id: 'mrt-005', taskName: '上海eNB覆���MR专项', collectType: 'MRO', deviceRange: 28, schedule: '一次性', status: 'failed', lastRun: '2024-06-01T06:00:00.000Z', creator: 'operator2' },
  { id: 'mrt-006', taskName: '新增gNB MRE采集任务', collectType: 'MRE', deviceRange: 5, schedule: '待配置', status: 'pending', creator: 'admin' },
];

const statusColorMap: Record<TaskStatus, string> = {
  running: 'processing',
  paused: 'warning',
  completed: 'green',
  failed: 'red',
  pending: 'default',
};

const mrTypeColorMap: Record<string, string> = { MRO: 'blue', MRE: 'green', MRS: 'orange' };

export default function Tasks() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [tasks, setTasks] = useState<MRTask[]>(mockTasks);
  const [createVisible, setCreateVisible] = useState(false);
  const [form] = Form.useForm();

  const statusLabelMap: Record<TaskStatus, string> = useMemo(() => ({
    running: t('status.running'),
    paused: t('mr.paused'),
    completed: t('mr.completed'),
    failed: t('status.failed'),
    pending: t('status.pending'),
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('mr.taskName'), type: 'input', placeholder: t('mr.taskNamePlaceholder') },
    {
      name: 'collectType',
      label: t('mr.collectType'),
      type: 'select',
      options: [
        { label: 'MRO', value: 'MRO' },
        { label: 'MRE', value: 'MRE' },
        { label: 'MRS', value: 'MRS' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.running'), value: 'running' },
        { label: t('mr.paused'), value: 'paused' },
        { label: t('mr.completed'), value: 'completed' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.pending'), value: 'pending' },
      ],
    },
  ], [t]);

  const filtered = tasks.filter((tk) => {
    if (filters.keyword && !tk.taskName.includes(String(filters.keyword))) return false;
    if (filters.collectType && tk.collectType !== filters.collectType) return false;
    if (filters.status && tk.status !== filters.status) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      const newTask: MRTask = {
        id: `mrt-${Date.now()}`,
        taskName: vals.taskName as string,
        collectType: vals.collectType as string,
        deviceRange: Number(vals.deviceRange ?? 0),
        schedule: vals.schedule as string,
        status: 'pending',
        creator: 'admin',
      };
      setTasks((prev) => [newTask, ...prev]);
      void message.success(t('mr.taskCreateSuccess'));
      setCreateVisible(false);
      form.resetFields();
    });
  };

  const columns: DataTableColumn<MRTask & Record<string, unknown>>[] = useMemo(() => [
    { key: 'taskName', title: t('mr.taskName'), dataIndex: 'taskName', ellipsis: true, width: 220 },
    {
      key: 'collectType', title: t('mr.collectType'), dataIndex: 'collectType', width: 100,
      render: (val) => <Tag color={mrTypeColorMap[String(val)] ?? 'default'}>{String(val)}</Tag>,
    },
    { key: 'deviceRange', title: t('mr.deviceScope'), dataIndex: 'deviceRange', width: 90, render: (val) => String(val) },
    { key: 'schedule', title: t('mr.schedule'), dataIndex: 'schedule', width: 120 },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as TaskStatus;
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'progress', title: t('mr.progress'), dataIndex: 'progress', width: 120,
      render: (val) => val !== undefined ? <Progress percent={Number(val)} size="small" /> : '—',
    },
    {
      key: 'lastRun', title: t('mr.lastRun'), dataIndex: 'lastRun', width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : '—',
    },
    { key: 'creator', title: t('mr.creator'), dataIndex: 'creator', width: 90 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 100, fixed: 'right',
      render: (_, record) => {
        const tk = record as MRTask;
        const moreItems: MenuProps['items'] = [
          ...(tk.status === 'running'
            ? [{ key: 'pause', label: t('mr.pause'), icon: <PauseCircleOutlined />, onClick: () => setTasks((prev) => prev.map((item) => item.id === tk.id ? { ...item, status: 'paused' } : item)) }]
            : []),
          ...(tk.status === 'paused' || tk.status === 'pending'
            ? [{ key: 'resume', label: t('common.execute'), icon: <PlayCircleOutlined />, onClick: () => setTasks((prev) => prev.map((item) => item.id === tk.id ? { ...item, status: 'running' } : item)) }]
            : []),
          { type: 'divider' as const },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: () => { setTasks((prev) => prev.filter((item) => item.id !== tk.id)); void message.success(t('common.deleteSuccess')); } },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />}>{t('common.detail')}</Button>
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
  ], [t, statusLabelMap]);

  return (
    <ListPageLayout
      title={t('nav.mr.tasks')}
      subtitle={t('mr.tasksSubtitle')}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>{t('mr.newTask')}</Button>}
    >
      <FilterBar
        filterId="mr-tasks-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable
          tableId="mr-tasks-list"
          columns={columns}
          dataSource={paginated as (MRTask & Record<string, unknown>)[]}
          loading={false}
          rowKey="id"
          total={filtered.length}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => setTasks([...mockTasks])}
          scroll={{ x: 1100 }}
        />
      </Card>

      <Modal
        title={t('mr.newCollectTask')}
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="taskName" label={t('mr.taskName')} rules={[{ required: true }]}>
            <Input placeholder={t('mr.taskNamePlaceholder')} />
          </Form.Item>
          <Form.Item name="collectType" label={t('mr.collectType')} rules={[{ required: true }]}>
            <Select options={[{ label: 'MRO', value: 'MRO' }, { label: 'MRE', value: 'MRE' }, { label: 'MRS', value: 'MRS' }]} />
          </Form.Item>
          <Form.Item name="deviceRange" label={t('mr.deviceCount')}>
            <Input type="number" placeholder={t('mr.deviceCountPlaceholder')} />
          </Form.Item>
          <Form.Item name="schedule" label={t('mr.collectSchedule')} rules={[{ required: true }]}>
            <Select options={[
              { label: t('mr.every15min'), value: '每15分钟' },
              { label: t('mr.everyHour'), value: '每小时' },
              { label: t('mr.dailyAt01'), value: '每天01:00' },
              { label: t('mr.oneTime'), value: '一次性' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
