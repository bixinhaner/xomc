import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Checkbox,
  DatePicker,
  Divider,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Space,
  Tag,
  Upload,
  message,
} from 'antd';
import {
  PlusOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
  UploadOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import type { MenuProps, UploadFile } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import type { MMLTask, MMLTaskStatus, MMLExecuteType, MMLTaskResult } from '@/types/mml';
import { useMMLTasks, useCreateMMLTask, useStartMMLTask, usePauseMMLTask, useCancelMMLTask, useDeleteMMLTask } from '@/hooks/api/useMML';
import { useT } from '@/hooks/useT';

// 任务类型映射 - use i18n keys
const CREATE_STATUS_KEYS: Record<MMLExecuteType, { color: string; key: string }> = {
  immediate: { color: 'green', key: 'mml.immediateExecute' },
  suspended: { color: 'orange', key: 'mml.suspended' },
  scheduled: { color: 'blue', key: 'mml.scheduledExecute' },
  periodic: { color: 'purple', key: 'mml.periodicTask' },
};

// 任务状态映射 - use i18n keys
const TASK_STATUS_KEYS: Record<MMLTaskStatus, { color: string; key: string }> = {
  pending: { color: 'default', key: 'mml.pendingStatus' },
  running: { color: 'processing', key: 'mml.runningStatus' },
  paused: { color: 'warning', key: 'mml.pausedStatus' },
  completed: { color: 'success', key: 'mml.completedStatus' },
  cancelled: { color: 'error', key: 'mml.cancelledStatus' },
  failed: { color: 'error', key: 'mml.failedStatus' },
};

// 任务结果映射 - use i18n keys
const TASK_RESULT_KEYS: Record<MMLTaskResult, { color: string; key: string }> = {
  success: { color: 'success', key: 'status.success' },
  partial: { color: 'warning', key: 'mml.partialSuccess' },
  failed: { color: 'error', key: 'status.failed' },
};

// 添加任务表单接口
interface AddTaskForm {
  taskName: string;
  fileName: string;
  executeType: MMLExecuteType;
  time: Dayjs | null;
  periodStartTime: Dayjs | null;
  periodEndTime: Dayjs | null;
  periodTime: Dayjs | null;
  offlineRetryEnable: boolean;
  offlineRetryWaitTime: number;
  failedRetryEnable: boolean;
  failedRetryCount: number;
  failedRetryWaitTime: number;
}

function formatTime(iso?: string): string {
  if (!iso) return '';
  return dayjs(iso).format('YYYY-MM-DD HH:mm:ss');
}

function computeProgress(task: MMLTask): string {
  if (task.totalDevices === 0) return '0%';
  return `${Math.round(((task.successCount + task.failedCount) / task.totalDevices) * 100)}%`;
}

export default function ScriptTask() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<MMLTask | null>(null);
  const [resultModalVisible, setResultModalVisible] = useState(false);
  const [resultTaskName, setResultTaskName] = useState<string>('');
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [addForm] = Form.useForm<AddTaskForm>();
  const executeType = Form.useWatch('executeType', addForm);

  const { data, isLoading, refetch } = useMMLTasks({ page, pageSize });
  const createTaskMutation = useCreateMMLTask();
  const startTaskMutation = useStartMMLTask();
  const pauseTaskMutation = usePauseMMLTask();
  const cancelTaskMutation = useCancelMMLTask();
  const deleteTaskMutation = useDeleteMMLTask();

  // Map API tasks to display rows with client-side filtering
  const tasks = useMemo(() => data?.items ?? [], [data?.items]);

  const filteredData = useMemo(() => {
    return tasks.filter((task) => {
      if (filterParams.taskName && typeof filterParams.taskName === 'string') {
        if (!task.taskName.toLowerCase().includes(filterParams.taskName.toLowerCase())) return false;
      }
      if (filterParams.startTime && Array.isArray(filterParams.startTime) && filterParams.startTime.length === 2) {
        const [start, end] = filterParams.startTime as [string, string];
        if (task.startedAt) {
          if (start && task.startedAt < start) return false;
          if (end && task.startedAt > end) return false;
        }
      }
      if (filterParams.executeType && filterParams.executeType !== 'all') {
        if (task.executeType !== filterParams.executeType) return false;
      }
      if (filterParams.taskStatus && filterParams.taskStatus !== 'all') {
        if (task.status !== filterParams.taskStatus) return false;
      }
      if (filterParams.taskResult && filterParams.taskResult !== 'all') {
        if (task.result !== filterParams.taskResult) return false;
      }
      return true;
    });
  }, [tasks, filterParams]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'taskName', label: t('mml.taskName'), type: 'input', placeholder: t('mml.inputTaskNameRequired') },
    { name: 'startTime', label: t('mml.startTime'), type: 'date-range' },
    { name: 'executeType', label: t('mml.type'), type: 'select', placeholder: t('mml.type'), options: [
      { label: t('common.all'), value: 'all' },
      { label: t('mml.immediateExecute'), value: 'immediate' },
      { label: t('mml.suspended'), value: 'suspended' },
      { label: t('mml.scheduledExecute'), value: 'scheduled' },
      { label: t('mml.periodicTask'), value: 'periodic' },
    ] },
    { name: 'taskStatus', label: t('mml.status'), type: 'select', placeholder: t('mml.status'), options: [
      { label: t('common.all'), value: 'all' },
      { label: t('mml.pendingStatus'), value: 'pending' },
      { label: t('mml.runningStatus'), value: 'running' },
      { label: t('mml.pausedStatus'), value: 'paused' },
      { label: t('mml.completedStatus'), value: 'completed' },
      { label: t('mml.cancelledStatus'), value: 'cancelled' },
      { label: t('mml.failedStatus'), value: 'failed' },
    ] },
    { name: 'taskResult', label: t('mml.result'), type: 'select', placeholder: t('mml.result'), options: [
      { label: t('common.all'), value: 'all' },
      { label: t('status.success'), value: 'success' },
      { label: t('mml.partialSuccess'), value: 'partial' },
      { label: t('status.failed'), value: 'failed' },
    ] },
  ], [t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setPage(1);
  }, []);

  const showResult = (task: MMLTask) => {
    setResultTaskName(task.taskName);
    setResultModalVisible(true);
  };

  const handleStartTask = (task: MMLTask) => {
    startTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(t('mml.taskStarted', { name: task.taskName })),
      onError: (err) => void message.error(t('mml.startFailed', { error: err instanceof Error ? err.message : 'Unknown' })),
    });
  };

  const handlePauseTask = (task: MMLTask) => {
    pauseTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(t('mml.taskPaused', { name: task.taskName })),
      onError: (err) => void message.error(t('mml.pauseFailed', { error: err instanceof Error ? err.message : 'Unknown' })),
    });
  };

  const handleCancelTask = (task: MMLTask) => {
    cancelTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(t('mml.taskCancelled', { name: task.taskName })),
      onError: (err) => void message.error(t('mml.cancelFailed', { error: err instanceof Error ? err.message : 'Unknown' })),
    });
  };

  const handleDeleteTask = (task: MMLTask) => {
    deleteTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(t('mml.taskDeleted', { name: task.taskName })),
      onError: (err) => void message.error(t('mml.deleteFailed', { error: err instanceof Error ? err.message : 'Unknown' })),
    });
  };

  const viewTaskInfo = (task: MMLTask) => {
    setEditingTask(task);
    setModalVisible(true);
  };

  const openAddModal = () => {
    addForm.resetFields();
    addForm.setFieldsValue({
      executeType: 'immediate',
      offlineRetryEnable: false,
      offlineRetryWaitTime: 60,
      failedRetryEnable: false,
      failedRetryCount: 3,
      failedRetryWaitTime: 5,
    });
    setFileList([]);
    setAddModalVisible(true);
  };

  const handleAddTask = () => {
    addForm.validateFields().then((values) => {
      const payload = {
        taskName: values.taskName,
        deviceSns: [] as string[],
        commands: [] as string[],
        creator: '',
        executeType: values.executeType,
        offlineRetry: values.offlineRetryEnable,
        offlineRetryWait: values.offlineRetryWaitTime,
        failedRetry: values.failedRetryEnable,
        failedRetryCount: values.failedRetryCount,
        failedRetryInterval: values.failedRetryWaitTime,
        scheduledAt: values.executeType === 'scheduled' && values.time ? values.time.toISOString() : undefined,
        periodStart: values.executeType === 'periodic' && values.periodStartTime ? values.periodStartTime.toISOString() : undefined,
        periodEnd: values.executeType === 'periodic' && values.periodEndTime ? values.periodEndTime.toISOString() : undefined,
        periodTime: values.executeType === 'periodic' && values.periodTime ? values.periodTime.format('HH:mm:ss') : undefined,
      };
      createTaskMutation.mutate(payload, {
        onSuccess: () => {
          void message.success(t('mml.taskCreated'));
          setAddModalVisible(false);
        },
        onError: (err) => void message.error(t('mml.taskCreateFailed', { error: err instanceof Error ? err.message : 'Unknown' })),
      });
    }).catch(() => undefined);
  };

  const handleDownloadTemplate = () => {
    void message.info(t('mml.templateDownloading'));
  };

  const getActionMenu = (task: MMLTask): MenuProps['items'] => {
    const status = task.status;
    return [
      { key: 'info', icon: <InfoCircleOutlined />, label: t('mml.info'), onClick: () => viewTaskInfo(task) },
      { key: 'start', icon: <PlayCircleOutlined />, label: t('common.start'), disabled: status !== 'paused' && status !== 'pending', onClick: () => handleStartTask(task) },
      { key: 'wait', icon: <PauseCircleOutlined />, label: t('common.pause'), disabled: status !== 'running', onClick: () => handlePauseTask(task) },
      { key: 'end', icon: <StopOutlined />, label: t('mml.terminateTask'), disabled: !['running', 'pending', 'paused'].includes(status), onClick: () => handleCancelTask(task) },
      { key: 'del', icon: <DeleteOutlined />, label: t('common.delete'), danger: true, disabled: status === 'running', onClick: () => handleDeleteTask(task) },
    ];
  };

  const columns: DataTableColumn<MMLTask>[] = useMemo(() => [
    {
      key: 'operation', title: t('table.operation'), dataIndex: 'id', width: 100, fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => showResult(record)}>{t('common.view')}</Button>
          <Dropdown menu={{ items: getActionMenu(record) }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      ),
    },
    { key: 'taskName', title: t('mml.taskName'), dataIndex: 'taskName', ellipsis: true },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'createdAt', title: t('mml.createTime'), dataIndex: 'createdAt', width: 160, render: (val: string) => formatTime(val) },
    { key: 'executeType', title: t('mml.type'), dataIndex: 'executeType', width: 100, render: (val: MMLExecuteType) => <Tag color={CREATE_STATUS_KEYS[val]?.color}>{t(CREATE_STATUS_KEYS[val]?.key)}</Tag> },
    { key: 'status', title: t('mml.status'), dataIndex: 'status', width: 100, render: (val: MMLTaskStatus) => <Tag color={TASK_STATUS_KEYS[val]?.color}>{t(TASK_STATUS_KEYS[val]?.key)}</Tag> },
    { key: 'progress', title: t('mml.progress'), width: 80, render: (_: unknown, record: MMLTask) => computeProgress(record) },
    { key: 'result', title: t('mml.result'), dataIndex: 'result', width: 100, render: (val?: MMLTaskResult) => val ? <Tag color={TASK_RESULT_KEYS[val]?.color}>{t(TASK_RESULT_KEYS[val]?.key)}</Tag> : '-' },
    { key: 'startedAt', title: t('mml.startTime'), dataIndex: 'startedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
    { key: 'finishedAt', title: t('mml.endTime'), dataIndex: 'finishedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={openAddModal}>{t('common.add')}</Button>
        </Space>
      }
    >
      <FilterBar filterId="mml-script-task" fields={filterFields} onSearch={handleSearch} onReset={handleReset} />

      <DataTable<MMLTask>
        tableId="script-task" columns={columns} dataSource={filteredData} loading={isLoading} rowKey="id"
        total={data?.total ?? 0} currentPage={page} pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }} onRefresh={() => void refetch()} scroll={{ x: 1400 }}
      />

      {/* 任务详情弹窗 */}
      <Modal title={t('mml.taskDetail')} open={modalVisible} onCancel={() => setModalVisible(false)} footer={null} width={680}>
        {editingTask && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.taskNameLabel')}</strong>{editingTask.taskName}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{editingTask.creator}</p>
            <p><strong>{t('mml.createTimeLabel')}</strong>{formatTime(editingTask.createdAt)}</p>
            <p><strong>{t('mml.typeLabel')}</strong>{t(CREATE_STATUS_KEYS[editingTask.executeType]?.key)}</p>
            <p><strong>{t('mml.statusLabel')}</strong>{t(TASK_STATUS_KEYS[editingTask.status]?.key)}</p>
            <p><strong>{t('mml.progressLabel')}</strong>{computeProgress(editingTask)}</p>
            <p><strong>{t('mml.resultLabel')}</strong>{editingTask.result ? t(TASK_RESULT_KEYS[editingTask.result]?.key) : '-'}</p>
            <p><strong>{t('mml.deviceCountLabel')}</strong>{editingTask.totalDevices}</p>
            <p><strong>{t('mml.successCountLabel')}</strong>{editingTask.successCount} / <strong>{t('mml.failedCountLabel')}</strong>{editingTask.failedCount}</p>
            <p><strong>{t('mml.startTimeLabel')}</strong>{formatTime(editingTask.startedAt) || '-'}</p>
            <p><strong>{t('mml.endTimeLabel')}</strong>{formatTime(editingTask.finishedAt) || '-'}</p>
          </div>
        )}
      </Modal>

      {/* 查看结果弹窗 */}
      <Modal title={t('mml.executionResult', { name: resultTaskName })} open={resultModalVisible} onCancel={() => setResultModalVisible(false)} footer={null} width={900}>
        <div style={{ padding: '16px 0' }}>
          <p style={{ color: '#999' }}>{t('mml.resultPlaceholder')}</p>
        </div>
      </Modal>

      {/* 新建任务抽屉 */}
      <Drawer
        title={t('mml.newMmlTask')}
        open={addModalVisible}
        onClose={() => setAddModalVisible(false)}
        width={560}
        destroyOnClose
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => setAddModalVisible(false)} style={{ marginRight: 8 }}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleAddTask} loading={createTaskMutation.isPending}>{t('common.confirm')}</Button>
          </div>
        }
      >
        <Form form={addForm} layout="vertical">
          {/* 基本信息 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            {t('mml.basicInfo')}
          </div>
          <Form.Item label={t('mml.taskName')} name="taskName" rules={[{ required: true, message: t('mml.inputTaskNameRequired') }]} style={{ marginLeft: 12 }}>
            <Input maxLength={50} placeholder={t('mml.inputTaskName')} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('mml.selectScript')} name="fileName" rules={[{ required: true, message: t('mml.selectFileFirst') }]} style={{ marginLeft: 12 }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                <Upload
                  accept=".txt"
                  fileList={fileList}
                  beforeUpload={(file) => {
                    setFileList([file as unknown as UploadFile]);
                    addForm.setFieldValue('fileName', file.name);
                    return false;
                  }}
                  onRemove={() => {
                    setFileList([]);
                    addForm.setFieldValue('fileName', '');
                  }}
                  maxCount={1}
                >
                  <Button icon={<UploadOutlined />}>{t('mml.selectFile')}</Button>
                </Upload>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.onlyTxtFormat')}</span>
              </Space>
              <div>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.templateImportTip')}</span>
                <Button type="link" size="small" icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>{t('mml.exportTemplate')}</Button>
              </div>
            </Space>
          </Form.Item>

          <Divider />

          {/* 选择执行方式 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            {t('mml.selectExecuteMethod')}
          </div>
          <Form.Item name="executeType" style={{ marginBottom: 8, marginLeft: 12 }}>
            <Radio.Group>
              <Radio value="immediate">{t('mml.immediateExecute')}</Radio>
              <Radio value="suspended">{t('mml.suspended')}</Radio>
              <Space>
                <Radio value="scheduled">{t('mml.scheduledExecute')}</Radio>
                {executeType === 'scheduled' && (
                  <Form.Item name="time" noStyle rules={[{ required: executeType === 'scheduled', message: t('mml.selectScheduledTime') }]}>
                    <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" disabledDate={(current) => current && current < dayjs().startOf('day')} style={{ width: 185 }} />
                  </Form.Item>
                )}
              </Space>
            </Radio.Group>
          </Form.Item>
          <Form.Item style={{ marginBottom: 8, marginLeft: 12 }}>
            <Space align="start">
              <Radio value="periodic" checked={executeType === 'periodic'} onChange={() => addForm.setFieldValue('executeType', 'periodic')}>{t('mml.periodicTask')}</Radio>
              {executeType === 'periodic' && (
                <>
                  <Form.Item name="periodDateRange" noStyle rules={[{ required: executeType === 'periodic', message: t('mml.selectDateRange') }]}>
                    <DatePicker.RangePicker disabledDate={(current) => current && current < dayjs().startOf('day')} style={{ width: 240 }} />
                  </Form.Item>
                  <span>:</span>
                  <Form.Item name="periodTime" noStyle rules={[{ required: executeType === 'periodic', message: t('mml.selectTime') }]}>
                    <DatePicker.TimePicker format="HH:mm:ss" style={{ width: 110 }} />
                  </Form.Item>
                </>
              )}
            </Space>
          </Form.Item>

          <Divider />

          {/* 执行策略 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            {t('mml.executionStrategy')}
          </div>
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            {t('mml.offlineDevice')}
            <Form.Item name="offlineRetryEnable" valuePropName="checked" noStyle>
              <Checkbox style={{ marginLeft: 8 }}>{t('mml.waitOnlineRetry')}</Checkbox>
            </Form.Item>
            <Form.Item name="offlineRetryWaitTime" noStyle>
              <InputNumber min={20} max={10080} style={{ width: 80, margin: '0 8px' }} />
            </Form.Item>
            {t('mml.minutes')}
          </div>
          <div style={{ marginLeft: 12, marginBottom: 8 }}>
            {t('mml.onlineDevice')}
            <Form.Item name="failedRetryEnable" valuePropName="checked" noStyle>
              <Checkbox style={{ marginLeft: 8 }}>{t('mml.failedRetry')}</Checkbox>
            </Form.Item>
            <Form.Item name="failedRetryCount" noStyle>
              <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
            </Form.Item>
            {t('mml.intervalRetry')}
            <Form.Item name="failedRetryWaitTime" noStyle>
              <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
            </Form.Item>
            {t('mml.minutes')}
          </div>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
