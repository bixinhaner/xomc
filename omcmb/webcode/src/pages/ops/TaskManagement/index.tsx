import { useState } from 'react';
import { Button, Dropdown, Tag, Space, Progress, Modal, Form, Input, Select, message, Tabs, Descriptions, Drawer } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, PlayCircleOutlined, PauseCircleOutlined, StopOutlined, EyeOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { mockOpsTasks, mockOpsTemplates } from '@core/mock/data/opsTools';
import type { OpsTask } from '@core/mock/data/opsTools';
import { useCreateOpsTask, usePauseOpsTask, useResumeOpsTask, useCancelOpsTask } from '@core/hooks/api/useOpsTools';

const filterFields: FilterField[] = [
  { name: 'keyword', label: '任务名称', type: 'input', placeholder: '请输入任务名称' },
  {
    name: 'status',
    label: '状态',
    type: 'select',
    options: [
      { label: '待执行', value: 'pending' },
      { label: '运行中', value: 'running' },
      { label: '已暂停', value: 'paused' },
      { label: '已完成', value: 'success' },
      { label: '失败', value: 'failed' },
      { label: '已取消', value: 'cancelled' },
    ],
  },
  { name: 'creator', label: '创建者', type: 'input', placeholder: '请输入创建者' },
];

const statusColorMap: Record<OpsTask['status'], string> = {
  pending: 'default',
  running: 'processing',
  paused: 'warning',
  success: 'green',
  failed: 'red',
  cancelled: 'default',
};

const statusLabelMap: Record<OpsTask['status'], string> = {
  pending: '待执行',
  running: '运行中',
  paused: '已暂停',
  success: '已完成',
  failed: '失败',
  cancelled: '已取消',
};

const templateNameMap = Object.fromEntries(mockOpsTemplates.map((t) => [t.id, t.templateName]));

// Batch param config mock tasks (different scenario)
const mockBatchParamTasks = [
  { id: 'bpt-001', taskName: '华北区eNB切换参数批量调整', deviceCount: 45, paramSet: '切换A3偏置优化', status: 'success' as const, successCount: 43, failCount: 2, createdAt: '2024-06-10T08:00:00.000Z', completedAt: '2024-06-10T08:45:00.000Z', creator: 'operator01' },
  { id: 'bpt-002', taskName: '全网gNB功率参数统一下发', deviceCount: 28, paramSet: '5G功率控制参数集V2', status: 'running' as const, successCount: 12, failCount: 0, createdAt: new Date(Date.now() - 300000).toISOString(), completedAt: undefined, creator: 'admin' },
  { id: 'bpt-003', taskName: '上海eNB邻区关系批量更新', deviceCount: 32, paramSet: '邻区关系优化-上海-20240601', status: 'pending' as const, successCount: 0, failCount: 0, createdAt: new Date(Date.now() - 60000).toISOString(), completedAt: undefined, creator: 'operator01' },
  { id: 'bpt-004', taskName: '广州覆盖参数专项调整', deviceCount: 18, paramSet: '广州覆盖优化参数包', status: 'failed' as const, successCount: 5, failCount: 13, createdAt: '2024-06-05T14:00:00.000Z', completedAt: '2024-06-05T14:30:00.000Z', creator: 'operator02' },
  { id: 'bpt-005', taskName: '深圳gNB调度参数下发', deviceCount: 10, paramSet: '5G调度参数优化包', status: 'paused' as const, successCount: 3, failCount: 0, createdAt: '2024-06-08T10:00:00.000Z', completedAt: undefined, creator: 'admin' },
];

export default function TaskManagement() {
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [batchFilters, setBatchFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [tasks, setTasks] = useState<OpsTask[]>(mockOpsTasks);
  const [createVisible, setCreateVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<OpsTask | null>(null);
  const [form] = Form.useForm();

  const pauseTask = usePauseOpsTask();
  const resumeTask = useResumeOpsTask();
  const cancelTask = useCancelOpsTask();
  const createTask = useCreateOpsTask();

  const filtered = tasks.filter((t) => {
    if (filters.keyword && !t.taskName.includes(String(filters.keyword))) return false;
    if (filters.status && t.status !== filters.status) return false;
    if (filters.creator && !t.creator.includes(String(filters.creator))) return false;
    return true;
  });

  const filteredBatch = mockBatchParamTasks.filter((t) => {
    if (batchFilters.keyword && !t.taskName.includes(String(batchFilters.keyword))) return false;
    if (batchFilters.status && t.status !== batchFilters.status) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const handleCreate = () => {
    form.validateFields().then((vals) => {
      const deviceSns = (vals.deviceSns as string)
        .split('\n')
        .map((s: string) => s.trim())
        .filter(Boolean);
      createTask.mutate(
        {
          taskName: vals.taskName as string,
          templateId: vals.templateId as string | undefined,
          deviceSns,
          totalSteps: 5,
          totalCount: deviceSns.length,
          creator: 'admin',
        },
        {
          onSuccess: () => {
            const newTask: OpsTask = {
              id: `opstask-${Date.now()}`,
              taskName: vals.taskName as string,
              templateId: vals.templateId as string | undefined,
              deviceSns,
              status: 'pending',
              currentStep: 0,
              totalSteps: 5,
              progress: 0,
              successCount: 0,
              failCount: 0,
              totalCount: deviceSns.length,
              createdAt: new Date().toISOString(),
              creator: 'admin',
            };
            setTasks((prev) => [newTask, ...prev]);
            void message.success('任务创建成功');
            setCreateVisible(false);
            form.resetFields();
          },
        }
      );
    });
  };

  const handlePause = (id: string) => {
    pauseTask.mutate(id, {
      onSuccess: () => {
        setTasks((prev) => prev.map((t) => t.id === id ? { ...t, status: 'paused' } : t));
        void message.success('任务已暂停');
      },
    });
  };

  const handleResume = (id: string) => {
    resumeTask.mutate(id, {
      onSuccess: () => {
        setTasks((prev) => prev.map((t) => t.id === id ? { ...t, status: 'running' } : t));
        void message.success('任务继续执行');
      },
    });
  };

  const handleCancel = (id: string) => {
    Modal.confirm({
      title: '确认取消',
      content: '取消后任务无法恢复，是否确认？',
      okType: 'danger',
      onOk: () => {
        cancelTask.mutate(id, {
          onSuccess: () => {
            setTasks((prev) => prev.map((t) => t.id === id ? { ...t, status: 'cancelled' } : t));
            void message.success('任务已取消');
          },
        });
      },
    });
  };

  const commonTaskColumns: DataTableColumn<OpsTask & Record<string, unknown>>[] = [
    { key: 'taskName', title: '任务名称', dataIndex: 'taskName', ellipsis: true, width: 220 },
    {
      key: 'templateId', title: '使用模板', dataIndex: 'templateId', width: 160, ellipsis: true,
      render: (val) => val ? (templateNameMap[String(val)] ?? String(val)) : <span style={{ color: '#999' }}>—</span>,
    },
    {
      key: 'totalCount', title: '设备数', dataIndex: 'totalCount', width: 80,
      render: (val) => `${String(val)} 台`,
    },
    {
      key: 'status', title: '状态', dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as OpsTask['status'];
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'progress', title: '进度', dataIndex: 'progress', width: 160,
      render: (val, record) => {
        const t = record as OpsTask;
        const pct = Number(val);
        return (
          <div>
            <Progress
              percent={pct}
              size="small"
              status={t.status === 'failed' ? 'exception' : t.status === 'running' ? 'active' : undefined}
            />
            <span style={{ fontSize: 11, color: '#999' }}>
              步骤 {t.currentStep}/{t.totalSteps}
              {t.status === 'success' && ` · 成功 ${t.successCount} 失败 ${t.failCount}`}
            </span>
          </div>
        );
      },
    },
    {
      key: 'createdAt', title: '创建时间', dataIndex: 'createdAt', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    { key: 'creator', title: '创建者', dataIndex: 'creator', width: 90 },
    {
      key: 'actions', title: '操作', dataIndex: 'id', width: 100, fixed: 'right',
      render: (_, record) => {
        const t = record as OpsTask;
        const moreItems: MenuProps['items'] = [
          ...(t.status === 'running'
            ? [{ key: 'pause', label: '暂停', icon: <PauseCircleOutlined />, onClick: () => handlePause(t.id) }]
            : []),
          ...((t.status === 'paused' || t.status === 'pending')
            ? [{ key: 'resume', label: '继续', icon: <PlayCircleOutlined />, onClick: () => handleResume(t.id) }]
            : []),
          ...((t.status === 'running' || t.status === 'paused' || t.status === 'pending')
            ? [{ type: 'divider' as const }, { key: 'cancel', label: '取消', icon: <StopOutlined />, danger: true, onClick: () => handleCancel(t.id) }]
            : []),
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => { setSelectedTask(t); setDetailVisible(true); }}>
              详情
            </Button>
            {moreItems.length > 0 && (
              <Dropdown menu={{ items: moreItems }} trigger={['click']}>
                <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
              </Dropdown>
            )}
          </Space>
        );
      },
    },
  ];

  const batchColumns: DataTableColumn<(typeof mockBatchParamTasks)[0] & Record<string, unknown>>[] = [
    { key: 'taskName', title: '任务名称', dataIndex: 'taskName', ellipsis: true, width: 240 },
    { key: 'paramSet', title: '参数集', dataIndex: 'paramSet', ellipsis: true, width: 200 },
    {
      key: 'deviceCount', title: '设备数', dataIndex: 'deviceCount', width: 80,
      render: (val) => `${String(val)} 台`,
    },
    {
      key: 'status', title: '状态', dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as OpsTask['status'];
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'result', title: '执行结果', dataIndex: 'successCount', width: 140,
      render: (val, record) => {
        const r = record as (typeof mockBatchParamTasks)[0];
        if (r.status === 'pending' || r.status === 'running') {
          return <Progress percent={Math.round((r.successCount / r.deviceCount) * 100)} size="small" status="active" />;
        }
        return (
          <span style={{ fontSize: 12 }}>
            <span style={{ color: '#52c41a' }}>成功 {r.successCount}</span>
            {r.failCount > 0 && <span style={{ color: '#ff4d4f', marginLeft: 8 }}>失败 {r.failCount}</span>}
          </span>
        );
      },
    },
    {
      key: 'createdAt', title: '创建时间', dataIndex: 'createdAt', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    { key: 'creator', title: '创建者', dataIndex: 'creator', width: 90 },
    {
      key: 'actions', title: '操作', dataIndex: 'id', width: 80, fixed: 'right',
      render: () => (
        <Button type="link" size="small" icon={<EyeOutlined />}>详情</Button>
      ),
    },
  ];

  return (
    <ListPageLayout
      title="任务管理"
      subtitle="管理运维自动化任务和批量参数配置任务"
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>新建任务</Button>}
    >
      <Tabs
        items={[
          {
            key: 'common',
            label: '通用配置任务',
            children: (
              <>
                <FilterBar
                  filterId="ops-task-common-filter"
                  fields={filterFields}
                  onSearch={(vals) => { setFilters(vals); setPage(1); }}
                  onReset={() => { setFilters({}); setPage(1); }}
                />
                <DataTable
                  tableId="ops-task-common-list"
                  columns={commonTaskColumns}
                  dataSource={paginated as (OpsTask & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filtered.length}
                  pageSize={pageSize}
                  currentPage={page}
                  onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
                  onRefresh={() => setTasks([...mockOpsTasks])}
                  scroll={{ x: 1200 }}
                  alarmRowStyle={(record) => {
                    const t = record as OpsTask;
                    return t.status === 'failed' ? 'major' : null;
                  }}
                />
              </>
            ),
          },
          {
            key: 'batch',
            label: '批量参数配置任务',
            children: (
              <>
                <FilterBar
                  filterId="ops-task-batch-filter"
                  fields={filterFields}
                  onSearch={(vals) => setBatchFilters(vals)}
                  onReset={() => setBatchFilters({})}
                />
                <DataTable
                  tableId="ops-task-batch-list"
                  columns={batchColumns}
                  dataSource={filteredBatch as ((typeof mockBatchParamTasks)[0] & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filteredBatch.length}
                  pageSize={10}
                  currentPage={1}
                  scroll={{ x: 1000 }}
                />
              </>
            ),
          },
        ]}
      />

      <Modal
        title="新建运维任务"
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        confirmLoading={createTask.isPending}
        width={540}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="taskName" label="任务名称" rules={[{ required: true }]}>
            <Input placeholder="请输入任务名称" />
          </Form.Item>
          <Form.Item name="templateId" label="使用模板" rules={[{ required: true }]}>
            <Select
              placeholder="请选择运维模板"
              options={mockOpsTemplates.map((t) => ({ label: `${t.templateName} (${t.category})`, value: t.id }))}
            />
          </Form.Item>
          <Form.Item name="deviceSns" label="目标设备SN（每行一个）" rules={[{ required: true }]}>
            <Input.TextArea
              rows={5}
              placeholder={'ENB00001\nENB00002\nGNB00001'}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={selectedTask ? `任务详情 — ${selectedTask.taskName}` : '任务详情'}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        width={600}
      >
        {selectedTask && (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="任务名称" span={2}>{selectedTask.taskName}</Descriptions.Item>
            <Descriptions.Item label="使用模板" span={2}>
              {selectedTask.templateId ? (templateNameMap[selectedTask.templateId] ?? selectedTask.templateId) : '—'}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColorMap[selectedTask.status]}>{statusLabelMap[selectedTask.status]}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="进度">
              <Progress
                percent={selectedTask.progress}
                size="small"
                status={selectedTask.status === 'failed' ? 'exception' : selectedTask.status === 'running' ? 'active' : undefined}
              />
            </Descriptions.Item>
            <Descriptions.Item label="当前步骤">{selectedTask.currentStep} / {selectedTask.totalSteps}</Descriptions.Item>
            <Descriptions.Item label="设备总数">{selectedTask.totalCount} 台</Descriptions.Item>
            <Descriptions.Item label="成功">
              <span style={{ color: '#52c41a', fontWeight: 600 }}>{selectedTask.successCount} 台</span>
            </Descriptions.Item>
            <Descriptions.Item label="失败">
              <span style={{ color: selectedTask.failCount > 0 ? '#ff4d4f' : '#999', fontWeight: selectedTask.failCount > 0 ? 600 : 400 }}>
                {selectedTask.failCount} 台
              </span>
            </Descriptions.Item>
            <Descriptions.Item label="创建者">{selectedTask.creator}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {new Date(selectedTask.createdAt).toLocaleString('zh-CN')}
            </Descriptions.Item>
            {selectedTask.startedAt && (
              <Descriptions.Item label="开始时间">
                {new Date(selectedTask.startedAt).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
            {selectedTask.completedAt && (
              <Descriptions.Item label="完成时间">
                {new Date(selectedTask.completedAt).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
            {selectedTask.message && (
              <Descriptions.Item label="消息" span={2}>{selectedTask.message}</Descriptions.Item>
            )}
            <Descriptions.Item label="目标设备SN" span={2}>
              <div style={{ fontFamily: 'monospace', fontSize: 12, maxHeight: 120, overflowY: 'auto' }}>
                {selectedTask.deviceSns.join(', ')}
              </div>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </ListPageLayout>
  );
}
