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
  Table,
  Tabs,
  Tag,
  Tooltip,
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
  CloseOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { MenuProps, UploadFile } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import type { MMLTask, MMLTaskStatus, MMLExecuteType, MMLTaskResult, MMLScript, DeviceTaskResultItem } from '@/types/mml';
import { useMMLTasks, useCreateMMLTask, useStartMMLTask, usePauseMMLTask, useCancelMMLTask, useDeleteMMLTask, useMMLScripts } from '@/hooks/api/useMML';
import { mmlApi } from '@/services/api/mmlApi';
import { useDictionary } from '@/hooks/api/useSystem';
import { useT } from '@/hooks/useT';
import { useUserStore } from '@/store/userStore';

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
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<MMLTask | null>(null);
  const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([]);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [addForm] = Form.useForm<AddTaskForm>();
  const executeType = Form.useWatch('executeType', addForm);
  const [deviceSns, setDeviceSns] = useState<string[]>([]);
  const [parsedCommands, setParsedCommands] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState('tasks');
  const [scriptPage, setScriptPage] = useState(1);
  const [scriptPageSize, setScriptPageSize] = useState(50);

  const { data: productTypeDict } = useDictionary('product_type');
  const productTypeOptions = useMemo(() => {
    const details = productTypeDict?.sysDictionaryDetails;
    if (details && details.length > 0) {
      return details.map((d) => ({ label: d.label, value: d.value }));
    }
    return [];
  }, [productTypeDict]);

  const { data, isLoading, refetch } = useMMLTasks({
    page,
    pageSize,
    status: filterParams.taskStatus && filterParams.taskStatus !== 'all' ? filterParams.taskStatus as string : undefined,
    executeType: filterParams.executeType && filterParams.executeType !== 'all' ? filterParams.executeType as string : undefined,
    result: filterParams.taskResult && filterParams.taskResult !== 'all' ? filterParams.taskResult as string : undefined,
    taskName: filterParams.taskName ? filterParams.taskName as string : undefined,
  });
  const createTaskMutation = useCreateMMLTask();
  const startTaskMutation = useStartMMLTask();
  const pauseTaskMutation = usePauseMMLTask();
  const cancelTaskMutation = useCancelMMLTask();
  const deleteTaskMutation = useDeleteMMLTask();
  const { data: scriptData, isLoading: scriptLoading, refetch: refetchScripts } = useMMLScripts({ page: scriptPage, pageSize: scriptPageSize });

  // Per-task results cache (keyed by task ID)
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
    const header = '基站编码,基站名称,MML脚本,状态,结果,失败原因,详情,开始时间,结束时间\n';
    const rows = cached.items.map(item =>
      [
        item.deviceSn,
        item.deviceName || '',
        item.mmlScript || '',
        item.status || 'completed',
        item.result.success ? '成功' : '失败',
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
    a.download = `${task.taskName}_结果.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [resultCache]);

  // Map API tasks to display rows (server-side filtering)
  const tasks = useMemo(() => data?.items ?? [], [data?.items]);

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

  const handleExportAllResults = useCallback((task: MMLTask) => {
    handleExportResults(task);
  }, [handleExportResults]);

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
    const userName = useUserStore.getState().currentUser?.userName ?? 'unknown';
    const defaultName = `MML任务_${userName}_${dayjs().format('YYYY-MM-DD HH:mm:ss')}`;
    addForm.setFieldsValue({
      taskName: defaultName,
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
    setAddModalVisible(true);
  };

  const handleAddTask = () => {
    addForm.validateFields().then((values) => {
      if (deviceSns.length === 0) {
        void message.warning(t('mml.selectDeviceFirst') || '请先输入设备SN');
        return;
      }
      if (parsedCommands.length === 0) {
        void message.warning(t('mml.selectFileFirst') || '请先上传脚本文件');
        return;
      }
      const payload = {
        taskName: values.taskName,
        deviceSns,
        commands: parsedCommands,
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
      { key: 'start', icon: <PlayCircleOutlined />, label: t('common.start'), disabled: status !== 'paused' && status !== 'pending', onClick: () => handleStartTask(task) },
      { key: 'wait', icon: <PauseCircleOutlined />, label: t('common.pause'), disabled: status !== 'running', onClick: () => handlePauseTask(task) },
      { key: 'end', icon: <StopOutlined />, label: t('mml.terminateTask'), disabled: !['running', 'pending', 'paused'].includes(status), onClick: () => handleCancelTask(task) },
      { key: 'del', icon: <DeleteOutlined />, label: t('common.delete'), danger: true, disabled: status === 'running', onClick: () => handleDeleteTask(task) },
      { key: 'export', icon: <DownloadOutlined />, label: t('mml.exportResult') || '导出结果', onClick: () => handleExportAllResults(task) },
    ];
  }, [t, viewTaskInfo, handleStartTask, handlePauseTask, handleCancelTask, handleDeleteTask]);

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
    { key: 'executeType', title: t('mml.type'), dataIndex: 'executeType', width: 100, render: (val: MMLExecuteType) => { const e = CREATE_STATUS_KEYS[val]; return e ? <Tag color={e.color}>{t(e.key)}</Tag> : <Tag>{val}</Tag>; } },
    { key: 'status', title: t('mml.status'), dataIndex: 'status', width: 100, render: (val: MMLTaskStatus) => { const e = TASK_STATUS_KEYS[val]; return e ? <Tag color={e.color}>{t(e.key)}</Tag> : <Tag>{val}</Tag>; } },
    { key: 'progress', title: t('mml.progress'), width: 80, render: (_: unknown, record: MMLTask) => computeProgress(record) },
    { key: 'result', title: t('mml.result'), dataIndex: 'result', width: 100, render: (val?: MMLTaskResult) => { if (!val) return '-'; const e = TASK_RESULT_KEYS[val]; return e ? <Tag color={e.color}>{t(e.key)}</Tag> : <Tag>{val}</Tag>; } },
    { key: 'startedAt', title: t('mml.startTime'), dataIndex: 'startedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
    { key: 'finishedAt', title: t('mml.endTime'), dataIndex: 'finishedAt', width: 140, render: (val?: string) => formatTime(val) || '-' },
  ], [t, getActionMenu]);

  const scriptColumns: DataTableColumn<MMLScript>[] = useMemo(() => [
    { key: 'scriptName', title: t('mml.scriptName') || '脚本名称', dataIndex: 'scriptName', ellipsis: true },
    { key: 'description', title: t('common.description') || '描述', dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
    { key: 'deviceType', title: t('mml.productType') || '产品类型', dataIndex: 'deviceType', width: 120, render: (v: string) => v || '-' },
    { key: 'creator', title: t('mml.creator') || '创建者', dataIndex: 'creator', width: 100 },
    {
      key: 'content', title: t('mml.scriptContent') || '脚本内容', dataIndex: 'content', width: 200, ellipsis: true,
      render: (v: string) => v ? <Tooltip title={v}><span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span></Tooltip> : '-',
    },
    { key: 'createTime', title: t('mml.createTime') || '创建时间', dataIndex: 'createTime', width: 160, render: (v: string) => formatTime(v) },
  ], [t]);

  const scripts = useMemo(() => scriptData?.items ?? [], [scriptData?.items]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        <Space>
          {activeTab === 'tasks' && (
            <Button type="primary" icon={<PlusOutlined />} onClick={openAddModal}>{t('common.add')}</Button>
          )}
        </Space>
      }
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'tasks',
            label: t('mml.taskList') || '任务列表',
            children: (
              <>
                <FilterBar filterId="mml-script-task" fields={filterFields} onSearch={handleSearch} onReset={handleReset} />

      <DataTable<MMLTask>
        tableId="script-task" columns={columns} dataSource={tasks} loading={isLoading} rowKey="id"
        total={data?.total ?? 0} currentPage={page} pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }} onRefresh={() => void refetch()} scroll={{ x: 1400 }}
        expandable={{
          expandedRowKeys,
          onExpandedRowsChange: (keys) => setExpandedRowKeys(keys as string[]),
          onExpand: handleExpand,
          expandedRowRender: (record) => {
            const cached = resultCache[record.id];
            const allItems = cached?.items ?? [];
            const search = resultSearch[record.id] || '';
            const currentPage = resultPage[record.id] || 1;
            const pageSize = 10;

            if (!cached) return <div style={{ padding: 16, color: '#999' }}>加载中...</div>;

            const filtered = search
              ? allItems.filter(item =>
                  item.deviceSn.toLowerCase().includes(search.toLowerCase()) ||
                  (item.deviceName || '').toLowerCase().includes(search.toLowerCase())
                )
              : allItems;

            const total = filtered.length;
            const paged = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);

            const resultColumns = [
              { title: '基站编码', dataIndex: 'deviceSn' as const, width: 180, render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v}</span> },
              { title: '基站名称', dataIndex: 'deviceName' as const, width: 140, render: (v?: string) => v || '-' },
              { title: 'MML脚本', dataIndex: 'mmlScript' as const, width: 200, ellipsis: true, render: (v?: string) => v ? <Tooltip title={v}><span>{v}</span></Tooltip> : '-' },
              {
                title: '状态', dataIndex: 'status' as const, width: 80, align: 'center' as const,
                render: (v?: string) => {
                  if (v === 'running') return <Tag color="processing">执行中</Tag>;
                  if (v === 'pending') return <Tag color="default">待执行</Tag>;
                  return <Tag color="success">已结束</Tag>;
                },
              },
              {
                title: '结果', width: 80, align: 'center' as const,
                render: (_: unknown, item: DeviceTaskResultItem) => (
                  <Tag color={item.result.success ? 'success' : 'error'}>{item.result.success ? '成功' : '失败'}</Tag>
                ),
              },
              { title: '失败原因', dataIndex: 'failReason' as const, width: 160, ellipsis: true, render: (v?: string) => v ? <Tooltip title={v}><span style={{ color: '#ff4d4f' }}>{v}</span></Tooltip> : '-' },
              {
                title: '详情', width: 160, ellipsis: true,
                render: (_: unknown, item: DeviceTaskResultItem) => item.result.rawOutput
                  ? <Tooltip title={item.result.rawOutput}><span>{item.result.rawOutput}</span></Tooltip>
                  : '-',
              },
              { title: '开始时间', dataIndex: 'startedAt' as const, width: 150, render: (v?: string) => v ? formatTime(v) : '-' },
              {
                title: '结束时间', width: 150,
                render: (_: unknown, item: DeviceTaskResultItem) => {
                  if (item.finishedAt) return formatTime(item.finishedAt);
                  if (item.result.timestamp) return formatTime(item.result.timestamp);
                  return '-';
                },
              },
            ];

            return (
              <div style={{ padding: '8px 0' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <span style={{ fontWeight: 500, fontSize: 14 }}>
                    {t('mml.result')}（{record.taskName}）
                  </span>
                  <Space>
                    <Input
                      prefix={<SearchOutlined />}
                      placeholder="搜索基站编码/名称"
                      size="small"
                      allowClear
                      value={search}
                      onChange={(e) => {
                        setResultSearch(prev => ({ ...prev, [record.id]: e.target.value }));
                        setResultPage(prev => ({ ...prev, [record.id]: 1 }));
                      }}
                      style={{ width: 220 }}
                    />
                    <Button size="small" icon={<DownloadOutlined />} onClick={() => handleExportResults(record)}>
                      {t('mml.exportResult') || '导出'}
                    </Button>
                    <Button size="small" icon={<CloseOutlined />} onClick={() => {
                      setExpandedRowKeys(prev => prev.filter(k => k !== record.id));
                    }} />
                  </Space>
                </div>

                {total === 0 ? (
                  <div style={{ padding: 16, color: '#999', textAlign: 'center' }}>
                    {search ? '未找到匹配结果' : '暂无执行结果'}
                  </div>
                ) : (
                  <Table
                    columns={resultColumns}
                    dataSource={paged}
                    rowKey={(_, idx) => String(idx)}
                    size="small"
                    pagination={{
                      current: currentPage,
                      pageSize,
                      total,
                      size: 'small',
                      showTotal: (tot) => `共 ${tot} 条`,
                      onChange: (p) => setResultPage(prev => ({ ...prev, [record.id]: p })),
                    }}
                    scroll={{ x: 1200 }}
                  />
                )}
              </div>
            );
          },
        }}
      />
              </>
            ),
          },
          {
            key: 'scripts',
            label: t('mml.scriptLibrary') || '脚本库',
            children: (
              <DataTable<MMLScript>
                tableId="mml-scripts"
                columns={scriptColumns}
                dataSource={scripts}
                loading={scriptLoading}
                rowKey="id"
                total={scriptData?.total ?? 0}
                currentPage={scriptPage}
                pageSize={scriptPageSize}
                onPageChange={(p, s) => { setScriptPage(p); setScriptPageSize(s); }}
                onRefresh={() => void refetchScripts()}
                scroll={{ x: 900 }}
              />
            ),
          },
        ]}
      />

      {/* 任务详情弹窗 */}
      <Modal title={t('mml.taskDetail')} open={modalVisible} onCancel={() => setModalVisible(false)} footer={null} width={680}>
        {editingTask && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>{t('mml.taskNameLabel')}</strong>{editingTask.taskName}</p>
            <p><strong>{t('mml.creatorLabel')}</strong>{editingTask.creator}</p>
            <p><strong>{t('mml.createTimeLabel')}</strong>{formatTime(editingTask.createdAt)}</p>
            <p><strong>{t('mml.typeLabel')}</strong>{CREATE_STATUS_KEYS[editingTask.executeType] ? t(CREATE_STATUS_KEYS[editingTask.executeType].key) : editingTask.executeType}</p>
            <p><strong>{t('mml.statusLabel')}</strong>{TASK_STATUS_KEYS[editingTask.status] ? t(TASK_STATUS_KEYS[editingTask.status].key) : editingTask.status}</p>
            <p><strong>{t('mml.progressLabel')}</strong>{computeProgress(editingTask)}</p>
            <p><strong>{t('mml.resultLabel')}</strong>{editingTask.result ? (TASK_RESULT_KEYS[editingTask.result] ? t(TASK_RESULT_KEYS[editingTask.result].key) : editingTask.result) : '-'}</p>
            <p><strong>{t('mml.deviceCountLabel')}</strong>{editingTask.totalDevices}</p>
            <p><strong>{t('mml.successCountLabel')}</strong>{editingTask.successCount} / <strong>{t('mml.failedCountLabel')}</strong>{editingTask.failedCount}</p>
            <p><strong>{t('mml.startTimeLabel')}</strong>{formatTime(editingTask.startedAt) || '-'}</p>
            <p><strong>{t('mml.endTimeLabel')}</strong>{formatTime(editingTask.finishedAt) || '-'}</p>
          </div>
        )}
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
          <Form.Item label={t('mml.productType') || '产品类型'} name="productType" style={{ marginLeft: 12 }}>
            <Select
              placeholder={t('mml.selectProductType') || '选择产品类型'}
              allowClear
              options={productTypeOptions}
            />
          </Form.Item>
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4, fontSize: 14 }}>
              {t('mml.deviceSn') || '设备SN'} <span style={{ color: '#ff4d4f' }}>*</span>
            </label>
            <Select
              mode="tags"
              value={deviceSns}
              onChange={setDeviceSns}
              placeholder={t('mml.inputDeviceSn') || '输入设备SN，按回车添加'}
              style={{ width: '100%' }}
              tokenSeparators={[',', ';', '\n']}
              open={false}
            />
            <span style={{ color: '#999', fontSize: 12 }}>{t('mml.deviceSnTip') || '输入SN后按回车确认，支持逗号分隔'}</span>
          </div>
          <Form.Item label={t('mml.selectScript')} name="fileName" rules={[{ required: true, message: t('mml.selectFileFirst') }]} style={{ marginLeft: 12 }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                <Upload
                  accept=".txt"
                  fileList={fileList}
                  beforeUpload={(file) => {
                    setFileList([file as unknown as UploadFile]);
                    addForm.setFieldValue('fileName', file.name);
                    parseUploadedFile(file);
                    return false;
                  }}
                  onRemove={() => {
                    setFileList([]);
                    addForm.setFieldValue('fileName', '');
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
                  {t('mml.commandsParsed') || `已解析 ${parsedCommands.length} 条命令`}
                </div>
              )}
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
