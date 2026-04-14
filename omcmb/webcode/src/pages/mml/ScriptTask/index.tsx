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
  EyeOutlined,
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

// 任务类型映射
const CREATE_STATUS_MAP: Record<MMLExecuteType, { color: string; text: string }> = {
  immediate: { color: 'green', text: '立即执行' },
  suspended: { color: 'orange', text: '挂起' },
  scheduled: { color: 'blue', text: '定时执行' },
  periodic: { color: 'purple', text: '周期任务' },
};

// 任务状态映射
const TASK_STATUS_MAP: Record<MMLTaskStatus, { color: string; text: string }> = {
  pending: { color: 'default', text: '等待中' },
  running: { color: 'processing', text: '执行中' },
  paused: { color: 'warning', text: '已暂停' },
  completed: { color: 'success', text: '已完成' },
  cancelled: { color: 'error', text: '已终止' },
  failed: { color: 'error', text: '失败' },
};

// 任务结果映射
const TASK_RESULT_MAP: Record<MMLTaskResult, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  partial: { color: 'warning', text: '部分成功' },
  failed: { color: 'error', text: '失败' },
};

// 任务类型选项
const CREATE_STATUS_OPTIONS = [
  { label: '立即执行', value: 'immediate' },
  { label: '挂起', value: 'suspended' },
  { label: '定时执行', value: 'scheduled' },
  { label: '周期任务', value: 'periodic' },
];

// 任务状态选项
const TASK_STATUS_OPTIONS = [
  { label: '等待中', value: 'pending' },
  { label: '执行中', value: 'running' },
  { label: '已暂停', value: 'paused' },
  { label: '已完成', value: 'completed' },
  { label: '已终止', value: 'cancelled' },
  { label: '失败', value: 'failed' },
];

// 任务结果选项
const TASK_RESULT_OPTIONS = [
  { label: '成功', value: 'success' },
  { label: '部分成功', value: 'partial' },
  { label: '失败', value: 'failed' },
];

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
    { name: 'taskName', label: '任务名称', type: 'input', placeholder: '请输入任务名称' },
    { name: 'startTime', label: '开始时间', type: 'date-range' },
    { name: 'executeType', label: '类型', type: 'select', placeholder: '请选择类型', options: [{ label: '全部', value: 'all' }, ...CREATE_STATUS_OPTIONS] },
    { name: 'taskStatus', label: '状态', type: 'select', placeholder: '请选择状态', options: [{ label: '全部', value: 'all' }, ...TASK_STATUS_OPTIONS] },
    { name: 'taskResult', label: '结果', type: 'select', placeholder: '请选择结果', options: [{ label: '全部', value: 'all' }, ...TASK_RESULT_OPTIONS] },
  ], []);

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
      onSuccess: () => void message.success(`任务 ${task.taskName} 已启动`),
      onError: (err) => void message.error(`启动失败: ${err instanceof Error ? err.message : '未知错误'}`),
    });
  };

  const handlePauseTask = (task: MMLTask) => {
    pauseTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(`任务 ${task.taskName} 已暂停`),
      onError: (err) => void message.error(`暂停失败: ${err instanceof Error ? err.message : '未知错误'}`),
    });
  };

  const handleCancelTask = (task: MMLTask) => {
    cancelTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(`任务 ${task.taskName} 已终止`),
      onError: (err) => void message.error(`终止失败: ${err instanceof Error ? err.message : '未知错误'}`),
    });
  };

  const handleDeleteTask = (task: MMLTask) => {
    deleteTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(`任务 ${task.taskName} 已删除`),
      onError: (err) => void message.error(`删除失败: ${err instanceof Error ? err.message : '未知错误'}`),
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
          void message.success('任务创建成功');
          setAddModalVisible(false);
        },
        onError: (err) => void message.error(`创建失败: ${err instanceof Error ? err.message : '未知错误'}`),
      });
    }).catch(() => undefined);
  };

  const handleDownloadTemplate = () => {
    void message.info('模板下载功能开发中...');
  };

  const getActionMenu = (task: MMLTask): MenuProps['items'] => {
    const status = task.status;
    return [
      { key: 'info', icon: <InfoCircleOutlined />, label: '信息', onClick: () => viewTaskInfo(task) },
      { key: 'start', icon: <PlayCircleOutlined />, label: '开始', disabled: status !== 'paused' && status !== 'pending', onClick: () => handleStartTask(task) },
      { key: 'wait', icon: <PauseCircleOutlined />, label: '暂停', disabled: status !== 'running', onClick: () => handlePauseTask(task) },
      { key: 'end', icon: <StopOutlined />, label: '终止任务', disabled: !['running', 'pending', 'paused'].includes(status), onClick: () => handleCancelTask(task) },
      { key: 'del', icon: <DeleteOutlined />, label: '删除', danger: true, disabled: status === 'running', onClick: () => handleDeleteTask(task) },
    ];
  };

  const columns: DataTableColumn<MMLTask>[] = useMemo(() => [
    {
      key: 'operation', title: '操作', dataIndex: 'id', width: 70, fixed: 'left',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => showResult(record)} title="查看结果" />
          <Dropdown menu={{ items: getActionMenu(record) }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      ),
    },
    { key: 'taskName', title: '任务名称', dataIndex: 'taskName', ellipsis: true },
    { key: 'creator', title: '创建者', dataIndex: 'creator', width: 100 },
    { key: 'createdAt', title: '创建时间', dataIndex: 'createdAt', width: 160, render: (val: string) => formatTime(val) },
    { key: 'executeType', title: '类型', dataIndex: 'executeType', width: 100, render: (val: MMLExecuteType) => <Tag color={CREATE_STATUS_MAP[val]?.color}>{CREATE_STATUS_MAP[val]?.text}</Tag> },
    { key: 'status', title: '状态', dataIndex: 'status', width: 100, render: (val: MMLTaskStatus) => <Tag color={TASK_STATUS_MAP[val]?.color}>{TASK_STATUS_MAP[val]?.text}</Tag> },
    { key: 'progress', title: '进度', width: 80, render: (_: unknown, record: MMLTask) => computeProgress(record) },
    { key: 'result', title: '结果', dataIndex: 'result', width: 100, render: (val?: MMLTaskResult) => val ? <Tag color={TASK_RESULT_MAP[val]?.color}>{TASK_RESULT_MAP[val]?.text}</Tag> : '-' },
    { key: 'startedAt', title: '开始时间', dataIndex: 'startedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
    { key: 'finishedAt', title: '结束时间', dataIndex: 'finishedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
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
      <Modal title="任务详情" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null} width={680}>
        {editingTask && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>任务名称：</strong>{editingTask.taskName}</p>
            <p><strong>创建者：</strong>{editingTask.creator}</p>
            <p><strong>创建时间：</strong>{formatTime(editingTask.createdAt)}</p>
            <p><strong>类型：</strong>{CREATE_STATUS_MAP[editingTask.executeType]?.text}</p>
            <p><strong>状态：</strong>{TASK_STATUS_MAP[editingTask.status]?.text}</p>
            <p><strong>进度：</strong>{computeProgress(editingTask)}</p>
            <p><strong>结果：</strong>{editingTask.result ? TASK_RESULT_MAP[editingTask.result]?.text : '-'}</p>
            <p><strong>设备总数：</strong>{editingTask.totalDevices}</p>
            <p><strong>成功：</strong>{editingTask.successCount} / <strong>失败：</strong>{editingTask.failedCount}</p>
            <p><strong>开始时间：</strong>{formatTime(editingTask.startedAt) || '-'}</p>
            <p><strong>结束时间：</strong>{formatTime(editingTask.finishedAt) || '-'}</p>
          </div>
        )}
      </Modal>

      {/* 查看结果弹窗 */}
      <Modal title={`执行结果 - ${resultTaskName}`} open={resultModalVisible} onCancel={() => setResultModalVisible(false)} footer={null} width={900}>
        <div style={{ padding: '16px 0' }}>
          <p style={{ color: '#999' }}>任务执行结果列表将在此显示...</p>
        </div>
      </Modal>

      {/* 新建任务抽屉 */}
      <Drawer
        title="新建MML脚本任务"
        open={addModalVisible}
        onClose={() => setAddModalVisible(false)}
        width={560}
        destroyOnClose
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => setAddModalVisible(false)} style={{ marginRight: 8 }}>取消</Button>
            <Button type="primary" onClick={handleAddTask} loading={createTaskMutation.isPending}>确定</Button>
          </div>
        }
      >
        <Form form={addForm} layout="vertical">
          {/* 基本信息 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            基本信息
          </div>
          <Form.Item label="任务名称" name="taskName" rules={[{ required: true, message: '请输入任务名称' }]} style={{ marginLeft: 12 }}>
            <Input maxLength={50} placeholder="请输入新建任务名称" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="选择脚本" name="fileName" rules={[{ required: true, message: '请先选择文件' }]} style={{ marginLeft: 12 }}>
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
                  <Button icon={<UploadOutlined />}>选择文件</Button>
                </Upload>
                <span style={{ color: '#999', fontSize: 12 }}>( 仅支持 .txt 格式 )</span>
              </Space>
              <div>
                <span style={{ color: '#999', fontSize: 12 }}>使用模板导入提示：支持使用模板导入</span>
                <Button type="link" size="small" icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>导出模板</Button>
              </div>
            </Space>
          </Form.Item>

          <Divider />

          {/* 选择执行方式 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            选择执行方式
          </div>
          <Form.Item name="executeType" style={{ marginBottom: 8, marginLeft: 12 }}>
            <Radio.Group>
              <Radio value="immediate">立即执行</Radio>
              <Radio value="suspended">挂起</Radio>
              <Space>
                <Radio value="scheduled">定时执行</Radio>
                {executeType === 'scheduled' && (
                  <Form.Item name="time" noStyle rules={[{ required: executeType === 'scheduled', message: '请选择定时执行时间' }]}>
                    <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" disabledDate={(current) => current && current < dayjs().startOf('day')} style={{ width: 185 }} />
                  </Form.Item>
                )}
              </Space>
            </Radio.Group>
          </Form.Item>
          <Form.Item style={{ marginBottom: 8, marginLeft: 12 }}>
            <Space align="start">
              <Radio value="periodic" checked={executeType === 'periodic'} onChange={() => addForm.setFieldValue('executeType', 'periodic')}>周期任务</Radio>
              {executeType === 'periodic' && (
                <>
                  <Form.Item name="periodDateRange" noStyle rules={[{ required: executeType === 'periodic', message: '请选择日期范围' }]}>
                    <DatePicker.RangePicker disabledDate={(current) => current && current < dayjs().startOf('day')} style={{ width: 240 }} />
                  </Form.Item>
                  <span>:</span>
                  <Form.Item name="periodTime" noStyle rules={[{ required: executeType === 'periodic', message: '请选择时间' }]}>
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
            执行策略
          </div>
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            离线设备
            <Form.Item name="offlineRetryEnable" valuePropName="checked" noStyle>
              <Checkbox style={{ marginLeft: 8 }}>等待设备上线重试</Checkbox>
            </Form.Item>
            <Form.Item name="offlineRetryWaitTime" noStyle>
              <InputNumber min={20} max={10080} style={{ width: 80, margin: '0 8px' }} />
            </Form.Item>
            分钟
          </div>
          <div style={{ marginLeft: 12, marginBottom: 8 }}>
            在线设备
            <Form.Item name="failedRetryEnable" valuePropName="checked" noStyle>
              <Checkbox style={{ marginLeft: 8 }}>配置失败重试</Checkbox>
            </Form.Item>
            <Form.Item name="failedRetryCount" noStyle>
              <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
            </Form.Item>
            间隔次数重试
            <Form.Item name="failedRetryWaitTime" noStyle>
              <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
            </Form.Item>
            分钟
          </div>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
