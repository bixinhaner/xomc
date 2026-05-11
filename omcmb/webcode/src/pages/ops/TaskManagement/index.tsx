import { useState, useMemo } from 'react';
import { Button, Dropdown, Tag, Space, Progress, Modal, Form, Input, Select, message, Descriptions, Drawer, Tooltip } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, PlayCircleOutlined, PauseCircleOutlined, StopOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import type { OpsTask } from '@core/mock/data/opsTools';
import {
  useOpsTasks,
  useOpsTemplates,
  useCreateOpsTask,
  usePauseOpsTask,
  useResumeOpsTask,
  useCancelOpsTask,
} from '@core/hooks/api/useOpsTools';
import { usePermission } from '@core/hooks/usePermission';

const STATUS_OPTIONS: ReadonlyArray<{ value: OpsTask['status']; labelZh: string }> = [
  { value: 'pending', labelZh: '待执行' },
  { value: 'running', labelZh: '运行中' },
  { value: 'paused', labelZh: '已暂停' },
  { value: 'success', labelZh: '已完成' },
  { value: 'failed', labelZh: '失败' },
  { value: 'cancelled', labelZh: '已取消' },
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

const filterFields: FilterField[] = [
  { name: 'keyword', label: '任务名称', type: 'input', placeholder: '请输入任务名称' },
  {
    name: 'status',
    label: '状态',
    type: 'select',
    options: STATUS_OPTIONS.map((o) => ({ label: o.labelZh, value: o.value })),
  },
  { name: 'creator', label: '创建者', type: 'input', placeholder: '请输入创建者' },
];

export default function TaskManagement() {
  const canCreate = usePermission('ops:task:create');
  const canCancel = usePermission('ops:task:cancel');
  const canControl = usePermission('ops:task:control');

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [status, setStatus] = useState<OpsTask['status'] | undefined>(undefined);
  const [createVisible, setCreateVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<OpsTask | null>(null);
  const [form] = Form.useForm();

  const queryParams = useMemo(() => ({ status, page, pageSize }), [status, page, pageSize]);
  const { data: tasksData, isLoading, refetch } = useOpsTasks(queryParams);
  const tasks: OpsTask[] = tasksData?.items ?? [];
  const total = tasksData?.total ?? 0;

  // 模板下拉数据（仅供创建任务时关联）
  const { data: templatesData } = useOpsTemplates({ page: 1, pageSize: 200 });
  const templates = templatesData?.items ?? [];
  const templateNameMap = useMemo(
    () => Object.fromEntries(templates.map((t) => [t.id, t.templateName])),
    [templates],
  );

  const pauseTask = usePauseOpsTask();
  const resumeTask = useResumeOpsTask();
  const cancelTask = useCancelOpsTask();
  const createTask = useCreateOpsTask();

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
            void message.success('任务创建成功');
            setCreateVisible(false);
            form.resetFields();
          },
          onError: () => void message.error('任务创建失败'),
        },
      );
    });
  };

  const handlePause = (id: string) => {
    pauseTask.mutate(id, {
      onSuccess: () => void message.success('任务已暂停'),
      onError: () => void message.error('操作失败'),
    });
  };

  const handleResume = (id: string) => {
    resumeTask.mutate(id, {
      onSuccess: () => void message.success('任务继续执行'),
      onError: () => void message.error('操作失败'),
    });
  };

  const handleCancel = (id: string) => {
    Modal.confirm({
      title: '确认取消',
      content: '取消后任务无法恢复，是否确认？',
      okType: 'danger',
      onOk: () =>
        cancelTask.mutateAsync(id).then(
          () => message.success('任务已取消'),
          () => message.error('操作失败'),
        ),
    });
  };

  const columns: DataTableColumn<OpsTask & Record<string, unknown>>[] = useMemo(
    () => [
      { key: 'taskName', title: '任务名称', dataIndex: 'taskName', ellipsis: true, width: 220 },
      {
        key: 'templateId',
        title: '使用模板',
        dataIndex: 'templateId',
        width: 160,
        ellipsis: true,
        render: (val) =>
          val ? templateNameMap[String(val)] ?? String(val) : <span style={{ color: '#999' }}>—</span>,
      },
      {
        key: 'totalCount',
        title: '设备数',
        dataIndex: 'totalCount',
        width: 80,
        render: (val) => `${String(val)} 台`,
      },
      {
        key: 'status',
        title: '状态',
        dataIndex: 'status',
        width: 100,
        render: (val) => {
          const s = val as OpsTask['status'];
          return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
        },
      },
      {
        key: 'progress',
        title: '进度',
        dataIndex: 'progress',
        width: 160,
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
        key: 'createdAt',
        title: '创建时间',
        dataIndex: 'createdAt',
        width: 160,
        render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
      },
      { key: 'creator', title: '创建者', dataIndex: 'creator', width: 90 },
      {
        key: 'actions',
        title: '操作',
        dataIndex: 'id',
        width: 110,
        fixed: 'right',
        render: (_, record) => {
          const t = record as OpsTask;
          const moreItems: MenuProps['items'] = [
            ...(t.status === 'running' && canControl
              ? [{ key: 'pause', label: '暂停', icon: <PauseCircleOutlined />, onClick: () => handlePause(t.id) }]
              : []),
            ...((t.status === 'paused' || t.status === 'pending') && canControl
              ? [{ key: 'resume', label: '继续', icon: <PlayCircleOutlined />, onClick: () => handleResume(t.id) }]
              : []),
            ...((t.status === 'running' || t.status === 'paused' || t.status === 'pending') && canCancel
              ? [
                  { type: 'divider' as const },
                  {
                    key: 'cancel',
                    label: '取消',
                    icon: <StopOutlined />,
                    danger: true,
                    onClick: () => handleCancel(t.id),
                  },
                ]
              : []),
          ];
          return (
            <Space size={4}>
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setSelectedTask(t);
                  setDetailVisible(true);
                }}
              >
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
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [templateNameMap, canCancel, canControl],
  );

  return (
    <ListPageLayout
      title="任务管理"
      subtitle="管理运维自动化任务（注：执行引擎尚未实施，参 PRD F06-ops-management §12 T-0101）"
      extra={
        <Tooltip title={canCreate ? '' : '无权限：请联系管理员申请「任务新建」权限'}>
          <Button type="primary" icon={<PlusOutlined />} disabled={!canCreate} onClick={() => setCreateVisible(true)}>
            新建任务
          </Button>
        </Tooltip>
      }
    >
      <FilterBar
        filterId="ops-task-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setStatus(((vals.status as string) || undefined) as OpsTask['status'] | undefined);
          setPage(1);
        }}
        onReset={() => {
          setStatus(undefined);
          setPage(1);
        }}
      />
      <DataTable
        tableId="ops-task-list"
        columns={columns}
        dataSource={tasks as (OpsTask & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={total}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1200 }}
        alarmRowStyle={(record) => {
          const t = record as OpsTask;
          return t.status === 'failed' ? 'major' : null;
        }}
      />

      <Modal
        title="新建运维任务"
        open={createVisible}
        onOk={handleCreate}
        onCancel={() => {
          setCreateVisible(false);
          form.resetFields();
        }}
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
              options={templates.map((t) => ({ label: `${t.templateName} (${t.category})`, value: t.id }))}
            />
          </Form.Item>
          <Form.Item name="deviceSns" label="目标设备SN（每行一个）" rules={[{ required: true }]}>
            <Input.TextArea rows={5} placeholder={'ENB00001\nENB00002\nGNB00001'} style={{ fontFamily: 'monospace' }} />
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
            <Descriptions.Item label="任务名称" span={2}>
              {selectedTask.taskName}
            </Descriptions.Item>
            <Descriptions.Item label="使用模板" span={2}>
              {selectedTask.templateId ? templateNameMap[selectedTask.templateId] ?? selectedTask.templateId : '—'}
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
            <Descriptions.Item label="当前步骤">
              {selectedTask.currentStep} / {selectedTask.totalSteps}
            </Descriptions.Item>
            <Descriptions.Item label="设备总数">{selectedTask.totalCount} 台</Descriptions.Item>
            <Descriptions.Item label="成功">
              <span style={{ color: '#52c41a', fontWeight: 600 }}>{selectedTask.successCount} 台</span>
            </Descriptions.Item>
            <Descriptions.Item label="失败">
              <span
                style={{
                  color: selectedTask.failCount > 0 ? '#ff4d4f' : '#999',
                  fontWeight: selectedTask.failCount > 0 ? 600 : 400,
                }}
              >
                {selectedTask.failCount} 台
              </span>
            </Descriptions.Item>
            <Descriptions.Item label="创建者">{selectedTask.creator}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{new Date(selectedTask.createdAt).toLocaleString('zh-CN')}</Descriptions.Item>
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
              <Descriptions.Item label="消息" span={2}>
                {selectedTask.message}
              </Descriptions.Item>
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
