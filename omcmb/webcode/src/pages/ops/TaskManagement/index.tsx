import { useState, useMemo, useCallback } from 'react';
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
import { useT } from '@/hooks/useT';

// 状态值集合保持为常量（value 是与后端契约，不翻译）；显示标签在 render 时经 t() 派生（#226）。
const STATUS_VALUES: ReadonlyArray<OpsTask['status']> = [
  'pending',
  'running',
  'paused',
  'success',
  'failed',
  'cancelled',
];

const statusColorMap: Record<OpsTask['status'], string> = {
  pending: 'default',
  running: 'processing',
  paused: 'warning',
  success: 'green',
  failed: 'red',
  cancelled: 'default',
};

export default function TaskManagement() {
  const t = useT();
  const canCreate = usePermission('ops:task:create');
  const canCancel = usePermission('ops:task:cancel');
  const canControl = usePermission('ops:task:control');

  // 状态值 → 已翻译标签（切语言即时生效，#226）
  const statusLabel = useCallback((s: OpsTask['status']) => t(`ops.task.status.${s}`), [t]);

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'keyword', label: t('ops.task.colName'), type: 'input', placeholder: t('ops.task.namePlaceholder') },
      {
        name: 'status',
        label: t('ops.task.colStatus'),
        type: 'select',
        options: STATUS_VALUES.map((v) => ({ label: statusLabel(v), value: v })),
      },
      { name: 'creator', label: t('ops.task.colCreator'), type: 'input', placeholder: t('ops.task.creatorPlaceholder') },
    ],
    [t, statusLabel],
  );

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [keyword, setKeyword] = useState<string>();
  const [creator, setCreator] = useState<string>();
  const [status, setStatus] = useState<OpsTask['status'] | undefined>(undefined);
  const [createVisible, setCreateVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<OpsTask | null>(null);
  const [form] = Form.useForm();

  const queryParams = useMemo(
    () => ({ status, keyword, creator, page, pageSize }),
    [creator, keyword, page, pageSize, status],
  );
  const { data: tasksData, isLoading, refetch } = useOpsTasks(queryParams, {
    refetchOnMount: 'always',
  });
  const tasks: OpsTask[] = tasksData?.items ?? [];
  const total = tasksData?.total ?? 0;

  // 模板下拉数据（仅供创建任务时关联）
  const { data: templatesData } = useOpsTemplates(
    { page: 1, pageSize: 200 },
    { refetchOnMount: 'always' },
  );
  const templates = templatesData?.items ?? [];
  const templateNameMap = useMemo(
    () => Object.fromEntries(templates.map((tpl) => [tpl.id, tpl.templateName])),
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
            void message.success(t('ops.task.createSuccess'));
            setCreateVisible(false);
            form.resetFields();
          },
          onError: () => void message.error(t('ops.task.createFailed')),
        },
      );
    });
  };

  const handlePause = (id: string) => {
    pauseTask.mutate(id, {
      onSuccess: () => void message.success(t('ops.task.paused')),
      onError: () => void message.error(t('ops.task.opFailed')),
    });
  };

  const handleResume = (id: string) => {
    resumeTask.mutate(id, {
      onSuccess: () => void message.success(t('ops.task.resumed')),
      onError: () => void message.error(t('ops.task.opFailed')),
    });
  };

  const handleCancel = (id: string) => {
    Modal.confirm({
      title: t('ops.task.cancelConfirmTitle'),
      content: t('ops.task.cancelConfirmContent'),
      okType: 'danger',
      onOk: () =>
        cancelTask.mutateAsync(id).then(
          () => message.success(t('ops.task.cancelled')),
          () => message.error(t('ops.task.opFailed')),
        ),
    });
  };

  const columns: DataTableColumn<OpsTask & Record<string, unknown>>[] = useMemo(
    () => [
      { key: 'taskName', title: t('ops.task.colName'), dataIndex: 'taskName', ellipsis: true, width: 220 },
      {
        key: 'templateId',
        title: t('ops.task.colTemplate'),
        dataIndex: 'templateId',
        width: 160,
        ellipsis: true,
        render: (val) =>
          val ? templateNameMap[String(val)] ?? String(val) : <span style={{ color: '#999' }}>—</span>,
      },
      {
        key: 'totalCount',
        title: t('ops.task.colDeviceCount'),
        dataIndex: 'totalCount',
        width: 80,
        render: (val) => t('ops.task.deviceCountUnit', { count: String(val) }),
      },
      {
        key: 'status',
        title: t('ops.task.colStatus'),
        dataIndex: 'status',
        width: 100,
        render: (val) => {
          const s = val as OpsTask['status'];
          return <Tag color={statusColorMap[s]}>{statusLabel(s)}</Tag>;
        },
      },
      {
        key: 'progress',
        title: t('ops.task.colProgress'),
        dataIndex: 'progress',
        width: 160,
        render: (val, record) => {
          const task = record as OpsTask;
          const pct = Number(val);
          return (
            <div>
              <Progress
                percent={pct}
                size="small"
                status={task.status === 'failed' ? 'exception' : task.status === 'running' ? 'active' : undefined}
              />
              <span style={{ fontSize: 11, color: '#999' }}>
                {t('ops.task.stepProgress', { current: task.currentStep, total: task.totalSteps })}
                {task.status === 'success' &&
                  ` · ${t('ops.task.successFail', { success: task.successCount, fail: task.failCount })}`}
              </span>
            </div>
          );
        },
      },
      {
        key: 'createdAt',
        title: t('ops.task.colCreatedAt'),
        dataIndex: 'createdAt',
        width: 160,
        render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
      },
      { key: 'creator', title: t('ops.task.colCreator'), dataIndex: 'creator', width: 90 },
      {
        key: 'actions',
        title: t('ops.task.colActions'),
        dataIndex: 'id',
        width: 110,
        fixed: 'right',
        render: (_, record) => {
          const task = record as OpsTask;
          const moreItems: MenuProps['items'] = [
            ...(task.status === 'running' && canControl
              ? [{ key: 'pause', label: t('ops.task.pause'), icon: <PauseCircleOutlined />, onClick: () => handlePause(task.id) }]
              : []),
            ...((task.status === 'paused' || task.status === 'pending') && canControl
              ? [{ key: 'resume', label: t('ops.task.resume'), icon: <PlayCircleOutlined />, onClick: () => handleResume(task.id) }]
              : []),
            ...((task.status === 'running' || task.status === 'paused' || task.status === 'pending') && canCancel
              ? [
                  { type: 'divider' as const },
                  {
                    key: 'cancel',
                    label: t('ops.task.cancel'),
                    icon: <StopOutlined />,
                    danger: true,
                    onClick: () => handleCancel(task.id),
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
                  setSelectedTask(task);
                  setDetailVisible(true);
                }}
              >
                {t('ops.task.detail')}
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
    [templateNameMap, canCancel, canControl, t, statusLabel],
  );

  return (
    <ListPageLayout
      title={t('ops.task.title')}
      subtitle={t('ops.task.subtitle')}
      extra={
        <Tooltip title={canCreate ? '' : t('ops.task.createNoPermission')}>
          <Button type="primary" icon={<PlusOutlined />} disabled={!canCreate} onClick={() => setCreateVisible(true)}>
            {t('ops.task.create')}
          </Button>
        </Tooltip>
      }
    >
      <FilterBar
        filterId="ops-task-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setKeyword(((vals.keyword as string) || '').trim() || undefined);
          setCreator(((vals.creator as string) || '').trim() || undefined);
          setStatus(((vals.status as string) || undefined) as OpsTask['status'] | undefined);
          setPage(1);
        }}
        onReset={() => {
          setKeyword(undefined);
          setCreator(undefined);
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
          const task = record as OpsTask;
          return task.status === 'failed' ? 'major' : null;
        }}
      />

      <Modal
        title={t('ops.task.createModalTitle')}
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
          <Form.Item name="taskName" label={t('ops.task.colName')} rules={[{ required: true }]}>
            <Input placeholder={t('ops.task.namePlaceholder')} />
          </Form.Item>
          <Form.Item name="templateId" label={t('ops.task.colTemplate')} rules={[{ required: true }]}>
            <Select
              placeholder={t('ops.task.templatePlaceholder')}
              options={templates.map((tpl) => ({ label: `${tpl.templateName} (${tpl.category})`, value: tpl.id }))}
            />
          </Form.Item>
          <Form.Item name="deviceSns" label={t('ops.task.targetDevices')} rules={[{ required: true }]}>
            <Input.TextArea rows={5} placeholder={'ENB00001\nENB00002\nGNB00001'} style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={selectedTask ? `${t('ops.task.detailTitle')} — ${selectedTask.taskName}` : t('ops.task.detailTitle')}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        size={600}
      >
        {selectedTask && (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label={t('ops.task.colName')} span={2}>
              {selectedTask.taskName}
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.colTemplate')} span={2}>
              {selectedTask.templateId ? templateNameMap[selectedTask.templateId] ?? selectedTask.templateId : '—'}
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.colStatus')}>
              <Tag color={statusColorMap[selectedTask.status]}>{statusLabel(selectedTask.status)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.colProgress')}>
              <Progress
                percent={selectedTask.progress}
                size="small"
                status={selectedTask.status === 'failed' ? 'exception' : selectedTask.status === 'running' ? 'active' : undefined}
              />
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.currentStep')}>
              {selectedTask.currentStep} / {selectedTask.totalSteps}
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.totalCount')}>{t('ops.task.deviceCountUnit', { count: selectedTask.totalCount })}</Descriptions.Item>
            <Descriptions.Item label={t('ops.task.success')}>
              <span style={{ color: '#52c41a', fontWeight: 600 }}>{t('ops.task.deviceCountUnit', { count: selectedTask.successCount })}</span>
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.fail')}>
              <span
                style={{
                  color: selectedTask.failCount > 0 ? '#ff4d4f' : '#999',
                  fontWeight: selectedTask.failCount > 0 ? 600 : 400,
                }}
              >
                {t('ops.task.deviceCountUnit', { count: selectedTask.failCount })}
              </span>
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.task.colCreator')}>{selectedTask.creator}</Descriptions.Item>
            <Descriptions.Item label={t('ops.task.colCreatedAt')}>{new Date(selectedTask.createdAt).toLocaleString('zh-CN')}</Descriptions.Item>
            {selectedTask.startedAt && (
              <Descriptions.Item label={t('ops.task.startedAt')}>
                {new Date(selectedTask.startedAt).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
            {selectedTask.completedAt && (
              <Descriptions.Item label={t('ops.task.completedAt')}>
                {new Date(selectedTask.completedAt).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
            {selectedTask.message && (
              <Descriptions.Item label={t('ops.task.message')} span={2}>
                {selectedTask.message}
              </Descriptions.Item>
            )}
            <Descriptions.Item label={t('ops.task.targetDevices')} span={2}>
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
