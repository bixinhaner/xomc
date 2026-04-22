import { useState, useMemo, useCallback, useRef } from 'react';
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
  Select,
  Space,
  Tabs,
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
  EditOutlined,
  InfoCircleOutlined,
  UploadOutlined,
  DownloadOutlined,
  FileTextOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import type { MenuProps, UploadFile } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import type { MMLTask, MMLTaskStatus, MMLExecuteType, MMLTaskResult, DeviceTaskResultItem, MMLScript } from '@core/types/mml';
import { useMMLTasks, useCreateMMLTask, useStartMMLTask, usePauseMMLTask, useCancelMMLTask, useDeleteMMLTask, useMMLScripts, useDeleteMMLScripts } from '@core/hooks/api/useMML';
import { mmlApi } from '@core/services/api/mmlApi';
import { useDictionary } from '@core/hooks/api/useSystem';
import { useT } from '@/hooks/useT';
import { useUserStore } from '@core/store/userStore';

const EXECUTE_TYPE_KEYS: Record<MMLExecuteType, { color: string; key: string }> = {
  immediate: { color: 'green', key: 'mml.immediateExecute' },
  suspended: { color: 'orange', key: 'mml.suspended' },
  scheduled: { color: 'blue', key: 'mml.scheduledExecute' },
  periodic: { color: 'purple', key: 'mml.periodicTask' },
};

const TASK_STATUS_KEYS: Record<MMLTaskStatus, { color: string; key: string }> = {
  pending: { color: 'default', key: 'mml.pendingStatus' },
  running: { color: 'processing', key: 'mml.runningStatus' },
  paused: { color: 'warning', key: 'mml.pausedStatus' },
  completed: { color: 'success', key: 'mml.completedStatus' },
  cancelled: { color: 'error', key: 'mml.cancelledStatus' },
  failed: { color: 'error', key: 'mml.failedStatus' },
};

const TASK_RESULT_KEYS: Record<MMLTaskResult, { color: string; key: string }> = {
  success: { color: 'success', key: 'status.success' },
  partial: { color: 'warning', key: 'mml.partialSuccess' },
  failed: { color: 'error', key: 'status.failed' },
};

interface TaskForm {
  taskName: string;
  fileName: string;
  productType: string;
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
  const done = task.successCount + task.failedCount;
  return `${done}/${task.totalDevices}`;
}

export default function ScriptTask() {
  const t = useT();
  const [activeTab, setActiveTab] = useState('scripts');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailTask, setDetailTask] = useState<MMLTask | null>(null);
  const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([]);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<MMLTask | null>(null);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [form] = Form.useForm<TaskForm>();
  const executeType = Form.useWatch('executeType', form);
  const [deviceSns, setDeviceSns] = useState<string[]>([]);
  const [parsedCommands, setParsedCommands] = useState<string[]>([]);

  const [taskSearch, setTaskSearch] = useState('');

  const { data: productTypeDict } = useDictionary('product_type');
  const productTypeOptions = useMemo(() => {
    const details = productTypeDict?.sysDictionaryDetails;
    return details?.length ? details.map((d) => ({ label: d.label, value: d.value })) : [];
  }, [productTypeDict]);

  const { data, isLoading, refetch } = useMMLTasks({
    page,
    pageSize,
    taskName: taskSearch.trim() || undefined,
  });
  const createTaskMutation = useCreateMMLTask();
  const startTaskMutation = useStartMMLTask();
  const pauseTaskMutation = usePauseMMLTask();
  const cancelTaskMutation = useCancelMMLTask();
  const deleteTaskMutation = useDeleteMMLTask();

  // Per-task results cache
  const [resultCache, setResultCache] = useState<Record<string, { items: DeviceTaskResultItem[]; total: number }>>({});
  const [resultSearch, setResultSearch] = useState<Record<string, string>>({});
  const [resultPage, setResultPage] = useState<Record<string, number>>({});
  const fetchedRef = useRef<Set<string>>(new Set());

  const fetchResults = useCallback(async (taskId: string) => {
    if (fetchedRef.current.has(taskId)) return;
    fetchedRef.current.add(taskId);
    try {
      const resp = await mmlApi.getTaskResults(taskId, 1, 999);
      setResultCache(prev => ({ ...prev, [taskId]: resp }));
    } catch {
      fetchedRef.current.delete(taskId);
    }
  }, []);

  const handleExpand = useCallback((expanded: boolean, record: MMLTask) => {
    if (expanded) {
      setExpandedRowKeys(prev => [...prev, record.id]);
      void fetchResults(record.id);
    } else {
      setExpandedRowKeys(prev => prev.filter(k => k !== record.id));
    }
  }, [fetchResults]);

  const handleExportResults = useCallback((task: MMLTask) => {
    const cached = resultCache[task.id];
    if (!cached?.items.length) return;
    const header = `${t('mml.deviceSn')},${t('mml.deviceName')},MML${t('mml.scriptName')},${t('mml.status')},${t('mml.result')},${t('mml.failReason')},${t('mml.detail')},${t('mml.startTime')},${t('mml.endTime')}\n`;
    const rows = cached.items.map(item =>
      [
        item.deviceSn,
        item.deviceName || '',
        item.mmlScript || '',
        item.status || 'completed',
        item.result.success ? t('status.success') : t('status.failed'),
        item.failReason || '',
        `"${(item.result.rawOutput || '').replace(/"/g, '""')}"`,
        item.startedAt ? formatTime(item.startedAt) : '',
        item.finishedAt ? formatTime(item.finishedAt) : item.result.timestamp ? formatTime(item.result.timestamp) : '',
      ].join(',')
    ).join('\n');
    const bom = '\uFEFF';
    const blob = new Blob([bom + header + rows], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${task.taskName}_${t('mml.exportResult')}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [resultCache, t]);

  const tasks = useMemo(() => data?.items ?? [], [data?.items]);

  // ---- Script library state ----
  const [scriptPage, setScriptPage] = useState(1);
  const [scriptPageSize, setScriptPageSize] = useState(50);
  const [scriptSearch, setScriptSearch] = useState('');
  const { data: scriptsData, isLoading: scriptsLoading, refetch: refetchScripts } = useMMLScripts({
    page: scriptPage,
    pageSize: scriptPageSize,
    keyword: scriptSearch.trim() || undefined,
  });
  const deleteScriptsMutation = useDeleteMMLScripts();
  const [scriptViewVisible, setScriptViewVisible] = useState(false);
  const [viewingScript, setViewingScript] = useState<MMLScript | null>(null);

  const scripts = useMemo(() => scriptsData?.items ?? [], [scriptsData]);

  const handleDeleteScripts = useCallback((ids: string[]) => {
    deleteScriptsMutation.mutate(ids, {
      onSuccess: () => void message.success(t('common.deleteSuccess')),
      onError: (err) => void message.error(err instanceof Error ? err.message : 'Unknown'),
    });
  }, [deleteScriptsMutation, t]);

  const scriptColumns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => { setViewingScript(record); setScriptViewVisible(true); }}>{t('mml.info')}</Button>
          <Button type="link" size="small" danger onClick={() => handleDeleteScripts([record.id])}>{t('common.delete')}</Button>
        </Space>
      ),
    },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('mml.description'), dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
    { key: 'deviceType', title: t('mml.deviceType'), dataIndex: 'deviceType', width: 120, render: (v: string) => v || '-' },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'tags', title: t('mml.tags'), dataIndex: 'tags', width: 180, render: (tags: string[]) => tags?.length ? tags.map((tag) => <Tag key={tag}>{tag}</Tag>) : '-' },
    { key: 'updatedAt', title: t('mml.updateTime'), dataIndex: 'updateTime', width: 160, render: (val: string) => formatTime(val) },
  ], [t, handleDeleteScripts]);

  const scriptFilterFields: FilterField[] = useMemo(() => [
    { name: 'scriptName', label: t('mml.scriptName'), type: 'input', placeholder: t('mml.scriptName') },
    { name: 'deviceType', label: t('mml.deviceType'), type: 'select', placeholder: t('mml.deviceType'), options: [
      { label: t('common.all'), value: 'all' },
      ...productTypeOptions,
    ] },
    { name: 'creator', label: t('mml.creator'), type: 'input', placeholder: t('mml.creator') },
  ], [t, productTypeOptions]);

  const handleScriptSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setScriptPage(1);
    const keyword = (values.scriptName as string) || '';
    setScriptSearch(keyword);
  }, []);

  const handleScriptReset = useCallback(() => {
    setFilterParams({});
    setScriptSearch('');
    setScriptPage(1);
  }, []);

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
    setDetailTask(task);
    setDetailVisible(true);
  };

  const openCreateDrawer = () => {
    setEditingTask(null);
    form.resetFields();
    const userName = useUserStore.getState().currentUser?.userName ?? 'unknown';
    form.setFieldsValue({
      taskName: `MML任务_${userName}_${dayjs().format('YYYY-MM-DD HH:mm:ss')}`,
      executeType: 'immediate',
      offlineRetryEnable: false,
      offlineRetryWaitTime: 60,
      failedRetryEnable: false,
      failedRetryCount: 3,
      failedRetryWaitTime: 5,
    });
    setFileList([]);
    setDeviceSns([]);
    setParsedCommands([]);
    setDrawerVisible(true);
  };

  const openEditDrawer = (task: MMLTask) => {
    setEditingTask(task);
    form.resetFields();
    form.setFieldsValue({
      taskName: task.taskName,
      productType: '',
      executeType: task.executeType,
      offlineRetryEnable: task.offlineRetry,
      offlineRetryWaitTime: task.offlineRetryWait,
      failedRetryEnable: task.failedRetry,
      failedRetryCount: task.failedRetryCount,
      failedRetryWaitTime: task.failedRetryInterval,
    });
    setDeviceSns(task.deviceSns);
    setParsedCommands(task.commands);
    setFileList([]);
    setDrawerVisible(true);
  };

  const handleSubmitTask = () => {
    form.validateFields().then((values) => {
      if (deviceSns.length === 0) {
        void message.warning(t('mml.selectDeviceFirst'));
        return;
      }
      if (parsedCommands.length === 0) {
        void message.warning(t('mml.selectFileFirst'));
        return;
      }
      const payload = {
        taskName: values.taskName,
        deviceSns,
        commands: parsedCommands,
        creator: editingTask?.creator ?? '',
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
          void message.success(editingTask ? t('mml.taskUpdated') : t('mml.taskCreated'));
          setDrawerVisible(false);
        },
        onError: (err) => void message.error(t('mml.taskCreateFailed', { error: err instanceof Error ? err.message : 'Unknown' })),
      });
    }).catch(() => undefined);
  };

  const handleDownloadTemplate = () => {
    const templateContent = `# MML Script Template
# One command per line. Lines starting with # are comments.
# Example:
# LST CELL
# DSP Equipment
# MOD CELL:CellId=1,CellName=TestCell
`;
    const blob = new Blob([templateContent], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'mml-script-template.txt';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const parseUploadedFile = useCallback((file: File) => {
    const reader = new FileReader();
    reader.onload = (event) => {
      const content = event.target?.result as string;
      const lines = content
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => line && !line.startsWith('#'));
      setParsedCommands(lines);
    };
    reader.readAsText(file);
  }, []);

  const getActionMenu = useCallback((task: MMLTask): MenuProps['items'] => {
    const status = task.status;
    return [
      { key: 'info', icon: <InfoCircleOutlined />, label: t('mml.info'), onClick: () => viewTaskInfo(task) },
      { key: 'update', icon: <EditOutlined />, label: t('mml.updateTask'), onClick: () => openEditDrawer(task) },
      { key: 'start', icon: <PlayCircleOutlined />, label: t('common.start'), disabled: status !== 'paused' && status !== 'pending', onClick: () => handleStartTask(task) },
      { key: 'pause', icon: <PauseCircleOutlined />, label: t('common.pause'), disabled: status !== 'running', onClick: () => handlePauseTask(task) },
      { key: 'cancel', icon: <StopOutlined />, label: t('mml.terminateTask'), disabled: !['running', 'pending', 'paused'].includes(status), onClick: () => handleCancelTask(task) },
      { key: 'delete', icon: <DeleteOutlined />, label: t('common.delete'), danger: true, disabled: status === 'running', onClick: () => handleDeleteTask(task) },
      { key: 'export', icon: <DownloadOutlined />, label: t('mml.exportResult'), onClick: () => handleExportResults(task) },
    ];
  }, [t, handleExportResults]);

  const columns: DataTableColumn<MMLTask>[] = useMemo(() => [
    {
      key: 'operation', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => (
        <Dropdown menu={{ items: getActionMenu(record) }} trigger={['click']}>
          <Button type="link" size="small" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
    { key: 'taskName', title: t('mml.taskName'), dataIndex: 'taskName', ellipsis: true },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    { key: 'createdAt', title: t('mml.createTime'), dataIndex: 'createdAt', width: 160, render: (val: string) => formatTime(val) },
    { key: 'executeType', title: t('mml.type'), dataIndex: 'executeType', width: 100, render: (val: MMLExecuteType) => { const e = EXECUTE_TYPE_KEYS[val]; return e ? <Tag color={e.color}>{t(e.key)}</Tag> : <Tag>{val}</Tag>; } },
    { key: 'status', title: t('mml.status'), dataIndex: 'status', width: 100, render: (val: MMLTaskStatus) => { const e = TASK_STATUS_KEYS[val]; return e ? <Tag color={e.color}>{t(e.key)}</Tag> : <Tag>{val}</Tag>; } },
    { key: 'progress', title: t('mml.progress'), width: 80, render: (_: unknown, record: MMLTask) => computeProgress(record) },
    { key: 'result', title: t('mml.result'), dataIndex: 'result', width: 100, render: (val?: MMLTaskResult) => { if (!val) return '-'; const e = TASK_RESULT_KEYS[val]; return e ? <Tag color={e.color}>{t(e.key)}</Tag> : <Tag>{val}</Tag>; } },
    { key: 'startedAt', title: t('mml.startTime'), dataIndex: 'startedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
    { key: 'finishedAt', title: t('mml.endTime'), dataIndex: 'finishedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
  ], [t, getActionMenu]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        activeTab === 'tasks' ? (
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateDrawer}>{t('common.add')}</Button>
        ) : null
      }
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'scripts',
            label: (
              <span>
                <FileTextOutlined style={{ marginRight: 6 }} />
                {t('mml.scriptTask')}
              </span>
            ),
            children: (
              <>
                <FilterBar filterId="mml-script-library" fields={scriptFilterFields} onSearch={handleScriptSearch} onReset={handleScriptReset} />
                <DataTable<MMLScript>
                  tableId="mml-scripts" columns={scriptColumns} dataSource={scripts} loading={scriptsLoading} rowKey="id"
                  total={scriptsData?.total ?? 0} currentPage={scriptPage} pageSize={scriptPageSize}
                  onPageChange={(p, s) => { setScriptPage(p); setScriptPageSize(s); }} onRefresh={() => void refetchScripts()} scroll={{ x: 1000 }}
                />
              </>
            ),
          },
          {
            key: 'tasks',
            label: (
              <span>
                <UnorderedListOutlined style={{ marginRight: 6 }} />
                {t('mml.taskRecord')}
              </span>
            ),
            children: (
              <>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Input.Search
                    placeholder={t('mml.taskName')}
                    allowClear
                    style={{ width: 300 }}
                    onSearch={(val) => { setTaskSearch(val); setPage(1); }}
                  />
                  <Button icon={<DownloadOutlined />} onClick={() => void refetch()}>{t('common.refresh')}</Button>
                </div>
                <DataTable<MMLTask>
                  tableId="script-task" columns={columns} dataSource={tasks} loading={isLoading} rowKey="id"
                  total={data?.total ?? 0} currentPage={page} pageSize={pageSize}
                  onPageChange={(p, s) => { setPage(p); setPageSize(s); }} onRefresh={() => void refetch()} scroll={{ x: 1400 }}
                />
              </>
            ),
          },
        ]}
      />

      {/* 任务详情弹窗 */}
      <Modal title={t('mml.taskDetail')} open={detailVisible} onCancel={() => setDetailVisible(false)} footer={null} width={680}>
        {detailTask && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.taskNameLabel')}</strong>{detailTask.taskName}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{detailTask.creator}</p>
            <p><strong>{t('mml.createTimeLabel')}</strong>{formatTime(detailTask.createdAt)}</p>
            <p><strong>{t('mml.typeLabel')}</strong>{EXECUTE_TYPE_KEYS[detailTask.executeType] ? t(EXECUTE_TYPE_KEYS[detailTask.executeType].key) : detailTask.executeType}</p>
            <p><strong>{t('mml.statusLabel')}</strong>{TASK_STATUS_KEYS[detailTask.status] ? t(TASK_STATUS_KEYS[detailTask.status].key) : detailTask.status}</p>
            <p><strong>{t('mml.progressLabel')}</strong>{computeProgress(detailTask)}</p>
            <p><strong>{t('mml.resultLabel')}</strong>{detailTask.result ? (TASK_RESULT_KEYS[detailTask.result] ? t(TASK_RESULT_KEYS[detailTask.result].key) : detailTask.result) : '-'}</p>
            <p><strong>{t('mml.deviceCountLabel')}</strong>{detailTask.totalDevices}</p>
            <p><strong>{t('mml.successCountLabel')}</strong>{detailTask.successCount} / <strong>{t('mml.failedCountLabel')}</strong>{detailTask.failedCount}</p>
            <p><strong>{t('mml.startTimeLabel')}</strong>{formatTime(detailTask.startedAt) || '-'}</p>
            <p><strong>{t('mml.endTimeLabel')}</strong>{formatTime(detailTask.finishedAt) || '-'}</p>
          </div>
        )}
      </Modal>

      {/* 新建/编辑任务抽屉 */}
      <Drawer
        title={editingTask ? t('mml.updateTask') : t('mml.newMmlTask')}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        width={560}
        destroyOnClose
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => setDrawerVisible(false)} style={{ marginRight: 8 }}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleSubmitTask} loading={createTaskMutation.isPending}>{t('common.confirm')}</Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            {t('mml.basicInfo')}
          </div>
          <Form.Item label={t('mml.taskName')} name="taskName" rules={[{ required: true, message: t('mml.inputTaskNameRequired') }]} style={{ marginLeft: 12 }}>
            <Input maxLength={50} placeholder={t('mml.inputTaskName')} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label={t('mml.productType')} name="productType" style={{ marginLeft: 12 }}>
            <Select placeholder={t('mml.selectProductType')} allowClear options={productTypeOptions} />
          </Form.Item>
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4, fontSize: 14 }}>
              {t('mml.deviceSn')} <span style={{ color: '#ff4d4f' }}>*</span>
            </label>
            <Select
              mode="tags"
              value={deviceSns}
              onChange={setDeviceSns}
              placeholder={t('mml.inputDeviceSn')}
              style={{ width: '100%' }}
              tokenSeparators={[',', ';', '\n']}
              open={false}
            />
            <span style={{ color: '#999', fontSize: 12 }}>{t('mml.deviceSnTip')}</span>
          </div>
          <Form.Item label={t('mml.selectScript')} name="fileName" rules={[{ required: true, message: t('mml.selectFileFirst') }]} style={{ marginLeft: 12 }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                <Upload
                  accept=".txt"
                  fileList={fileList}
                  beforeUpload={(file) => {
                    setFileList([file as unknown as UploadFile]);
                    form.setFieldValue('fileName', file.name);
                    parseUploadedFile(file);
                    return false;
                  }}
                  onRemove={() => {
                    setFileList([]);
                    form.setFieldValue('fileName', '');
                    setParsedCommands([]);
                  }}
                  maxCount={1}
                >
                  <Button icon={<UploadOutlined />}>{t('mml.selectFile')}</Button>
                </Upload>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.onlyTxtFormat')}</span>
              </Space>
              {parsedCommands.length > 0 && (
                <div style={{ color: '#52c41a', fontSize: 12 }}>
                  {t('mml.commandsParsed', { count: parsedCommands.length })}
                </div>
              )}
              <div>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.templateImportTip')}</span>
                <Button type="link" size="small" icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>{t('mml.exportTemplate')}</Button>
              </div>
            </Space>
          </Form.Item>

          <Divider />

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
              <Radio value="periodic" checked={executeType === 'periodic'} onChange={() => form.setFieldValue('executeType', 'periodic')}>{t('mml.periodicTask')}</Radio>
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

      <Modal title={t('mml.scriptDetail')} open={scriptViewVisible} onCancel={() => setScriptViewVisible(false)} footer={null} width={600}>
        {viewingScript && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.scriptNameLabel')}</strong>{viewingScript.scriptName}</p>
            <p><strong>{t('mml.description')}</strong>{viewingScript.description || '-'}</p>
            <p><strong>{t('mml.deviceType')}</strong>{viewingScript.deviceType || '-'}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{viewingScript.creator}</p>
            <p><strong>{t('mml.updateTime')}</strong>{formatTime(viewingScript.updateTime)}</p>
            <div style={{ marginTop: 12 }}>
              <strong>{t('mml.scriptContent')}</strong>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, maxHeight: 300, overflow: 'auto', fontSize: 13, fontFamily: 'monospace' }}>
                {viewingScript.content}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </ListPageLayout>
  );
}
