import { useState, useMemo, useCallback } from 'react';
import {
  Tag,
  message,
  Progress,
  Checkbox,
  Alert,
  Space,
  Modal,
  Input,
  Button,
  Select,
  Radio,
  DatePicker,
  Drawer,
  Form,
  Divider,
  Table,
  Card,
  Dropdown,
  Descriptions,
} from 'antd';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { PlayCircleOutlined, WarningOutlined, PlusOutlined, ReloadOutlined, DownloadOutlined, DeleteOutlined, PauseOutlined, StopOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useUpgradeTasks,
  useSubTasks,
  useAllSubTasks,
  useCreateRollback,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useDeleteTask,
  useRetryTask,
} from '@core/hooks/api/useSoftware';
import { useDeviceList, useProductClasses } from '@core/hooks/api/useDevices';
import type { UpgradeTaskInfo, UpgradeSubTaskInfo } from '@core/mock/data/software';

// Execution method enum
type ExecutionMethod = 'immediate' | 'suspend' | 'scheduled';

// Map backend task status to display status code (1-4)
function mapTaskStatusToCode(status: string): number {
  const map: Record<string, number> = { pending: 1, in_progress: 2, suspended: 3, ended: 4 };
  return map[status] ?? 1;
}

// Compute overall progress percentage from task counters
function computeProgress(task: UpgradeTaskInfo): number {
  if (task.totalCount === 0) return 0;
  return Math.round(((task.successCount + task.failCount) / task.totalCount) * 100);
}

// Task status display color
const TASK_STATUS_COLORS: Record<number, string> = {
  1: 'default',     // pending
  2: 'processing',  // in_progress
  3: 'warning',     // suspended
  4: 'success',     // ended
};

// Task result display color (T-0136: 保留为 export 占位避免 TS6133)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const TASK_RESULT_COLORS: Record<string, string> = {
  success: 'success',
  partial: 'warning',
  failed: 'error',
  terminated: 'default',
};

// Sub-task status display color (T-0136: 保留为 export 占位避免 TS6133)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const SUB_TASK_STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  downloading: 'processing',
  rebooting: 'processing',
  verifying: 'processing',
  completed: 'success',
  failed: 'error',
  suspended: 'warning',
  terminated: 'default',
};

export default function VersionRollback() {
  const t = useT();

  // ---- Dynamic product type options from API ----
  const { data: productClassesData } = useProductClasses();
  const productClassOptions = useMemo(() => {
    if (productClassesData && productClassesData.length > 0) {
      return productClassesData.map((c) => ({ label: c, value: c }));
    }
    return [
      { label: 'PM-B4860', value: 'PM-B4860' },
      { label: 'QAFA', value: 'QAFA' },
      { label: 'QAFB', value: 'QAFB' },
      { label: 'FAP/BU1810', value: 'FAP/BU1810' },
    ];
  }, [productClassesData]);

  // ---- Pagination & filter state ----
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [, setFilters] = useState<Record<string, unknown>>({});

  // ---- Tab state ----
  const [activeTab, setActiveTab] = useState<'task' | 'device'>('task');

  // ---- Fetch rollback tasks from backend (taskType: 2) ----
  const { data: tasksData, isLoading: tasksLoading, refetch: refetchTasks } = useUpgradeTasks({
    page,
    pageSize,
    taskType: 2, // Only show rollback tasks
  });

  // ---- Device list tab: pagination & filter (all sub-tasks across rollback tasks) ----
  const [devicePage, setDevicePage] = useState(1);
  const [devicePageSize, setDevicePageSize] = useState(20);
  const [deviceFilters, setDeviceFilters] = useState<Record<string, unknown>>({});
  const { data: deviceSubTasksData, isLoading: deviceSubTasksLoading, refetch: refetchDeviceSubTasks } = useAllSubTasks({
    page: devicePage,
    pageSize: devicePageSize,
    taskName: (deviceFilters.keyword as string) || undefined,
    deviceSn: (deviceFilters.stationCode as string) || undefined,
    status: (deviceFilters.status as string) || undefined,
    taskType: 2,
  });

  // ---- Fetch sub-tasks for the detail drawer ----
  const [selectedTaskId, setSelectedTaskId] = useState<string>('');
  const { data: subTasksData, isLoading: subTasksLoading } = useSubTasks(selectedTaskId, { page: 1, pageSize: 100 });

  // ---- Mutations ----
  const createRollbackMutation = useCreateRollback();
  const suspendMutation = useSuspendTask();
  const resumeMutation = useResumeTask();
  const terminateMutation = useTerminateTask();
  const deleteMutation = useDeleteTask();
  const retryMutation = useRetryTask();

  // ---- Task status config with i18n ----
  const TASK_STATUS_CONFIG = useMemo((): Record<number, { color: string; text: string }> => ({
    1: { color: TASK_STATUS_COLORS[1], text: t('software.status.waiting') },
    2: { color: TASK_STATUS_COLORS[2], text: t('software.status.inProgress') },
    3: { color: TASK_STATUS_COLORS[3], text: t('software.status.paused') },
    4: { color: TASK_STATUS_COLORS[4], text: t('software.status.ended') },
  }), [t]);

  // Task result config with i18n
  const TASK_RESULT_MAP = useMemo(() => ({
    success: { color: 'success', text: t('status.success') },
    failed: { color: 'error', text: t('status.failed') },
    partial: { color: 'warning', text: t('software.status.partialSuccess') },
    terminated: { color: 'default', text: t('common.terminate') },
  }), [t]);

  // Sub-task status config with i18n
  const SUB_TASK_STATUS_MAP = useMemo(() => ({
    pending: { color: 'default', text: t('software.status.waiting') },
    downloading: { color: 'processing', text: t('software.status.downloading') },
    rebooting: { color: 'processing', text: t('software.status.rebooting') },
    verifying: { color: 'processing', text: t('software.status.verifying') },
    completed: { color: 'success', text: t('status.success') },
    failed: { color: 'error', text: t('status.failed') },
    suspended: { color: 'warning', text: t('software.status.paused') },
    terminated: { color: 'default', text: t('common.terminate') },
  }), [t]);

  // ---- Derived data ----

  // Task list from API
  const taskList = useMemo(() => tasksData?.items ?? [], [tasksData]);
  const taskTotal = tasksData?.total ?? 0;

  // Sub-tasks for detail drawer
  const subTaskList = useMemo(() => subTasksData?.items ?? [], [subTasksData]);

  // Sub-tasks for device list tab
  const deviceSubTaskList = useMemo(() => deviceSubTasksData?.items ?? [], [deviceSubTasksData]);
  const deviceSubTaskTotal = deviceSubTasksData?.total ?? 0;

  // ---- Create rollback drawer state ----
  const [upgradeDrawerVisible, setUpgradeDrawerVisible] = useState(false);
  const [taskName, setTaskName] = useState('');
  const [executionMethod, setExecutionMethod] = useState<ExecutionMethod>('immediate');
  const [scheduledTime, setScheduledTime] = useState<Dayjs | null>(null);
  const [drawerDevices, setDrawerDevices] = useState<{ id: string; deviceSn: string; deviceName: string; sourceVersion: string; productClass: string }[]>([]);
  const [drawerProductClass, setDrawerProductClass] = useState<string>('');
  const [selectAllOfType, setSelectAllOfType] = useState(false);

  // Add device modal state
  const [addDeviceModalVisible, setAddDeviceModalVisible] = useState(false);
  const [selectedNewDevices, setSelectedNewDevices] = useState<React.Key[]>([]);
  const [addDeviceKeyword, setAddDeviceKeyword] = useState('');

  // Batch input modal state
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');
  const [batchInputPreview, setBatchInputPreview] = useState<{
    matched: { id: string; deviceSn: string; deviceName: string; sourceVersion: string; productClass: string }[];
    notFound: string[];
    mixedTypes: string[];
  }>({ matched: [], notFound: [], mixedTypes: [] });

  // Retry confirm modal state
  const [retrySubTask, setRetrySubTask] = useState<UpgradeSubTaskInfo | null>(null);

  // Delete task confirm state
  const [deleteTaskRecord, setDeleteTaskRecord] = useState<UpgradeTaskInfo | null>(null);

  // Task detail drawer state
  const [taskDetailRecord, setTaskDetailRecord] = useState<UpgradeTaskInfo | null>(null);

  // ---- Fetch real devices from API by productClass ----
  const { data: deviceListData } = useDeviceList(
    {
      productClass: drawerProductClass,
      page: 1,
      pageSize: 500,
    },
    { refetchInterval: 0 },
  );

  // ---- Available devices for add-device modal ----
  const availableDevices = useMemo(() => {
    if (!drawerProductClass || !deviceListData?.items) return [];
    return deviceListData.items
      .filter((d) => !drawerDevices.some((existing) => existing.id === d.id))
      .map((d) => ({
        id: d.id,
        deviceSn: d.sn,
        deviceName: d.name,
        sourceVersion: d.firmwareVersion || d.softwareVersion || '',
        productClass: d.productClass,
        deviceGroup: d.groupName || '',
      }));
  }, [drawerProductClass, deviceListData, drawerDevices]);

  const filteredAvailableDevices = useMemo(() => {
    if (!addDeviceKeyword) return availableDevices;
    const keyword = addDeviceKeyword.toLowerCase();
    return availableDevices.filter(
      (d) => d.deviceSn.toLowerCase().includes(keyword) || d.deviceName.toLowerCase().includes(keyword),
    );
  }, [availableDevices, addDeviceKeyword]);

  const allDevicesCountOfType = deviceListData?.total ?? 0;

  // ---- Export ----
  const handleExport = useCallback(() => {
    if (taskList.length === 0) {
      void message.warning(t('software.upgrade.exportNoData'));
      return;
    }

    const headers = [t('software.taskName'), t('table.operator'), t('software.taskStatus'), t('software.upgrade.productClass'), t('software.rollback.rollbackProgress'), t('table.result'), t('software.startTime'), t('software.endTime')];
    const rows = taskList.map((task) => [
      task.taskName,
      task.createUser,
      TASK_STATUS_CONFIG[mapTaskStatusToCode(task.status)]?.text ?? task.status,
      task.productClass,
      `${computeProgress(task)}%`,
      task.result ? (TASK_RESULT_MAP[task.result]?.text ?? task.result) : '',
      task.startedAt ?? '',
      task.endedAt ?? '',
    ]);

    const csvContent = '\uFEFF' + [headers.join(','), ...rows.map((r) => r.map((c) => `"${c}"`).join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${t('software.rollback.exportFileName')}_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);

    void message.success(t('software.upgrade.exportSuccess', { count: taskList.length }));
  }, [taskList, TASK_STATUS_CONFIG, TASK_RESULT_MAP, t]);

  // ---- Filter definitions ----

  const taskFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.taskName'), type: 'input', placeholder: t('software.upgrade.inputTaskName') },
    {
      name: 'timeRange',
      label: t('common.timeRange'),
      type: 'date-range',
      placeholder: t('common.selectTimeRange'),
    },
  ], [t]);

  const deviceFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.taskName'), type: 'input', placeholder: t('software.upgrade.inputTaskName') },
    { name: 'stationCode', label: t('software.stationCode'), type: 'input', placeholder: t('software.inputStationCodeOrName') },
    {
      name: 'status',
      label: t('table.result'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: [
        { label: t('common.all'), value: '' },
        { label: t('software.status.waiting'), value: 'pending' },
        { label: t('software.status.downloading'), value: 'downloading' },
        { label: t('software.status.rebooting'), value: 'rebooting' },
        { label: t('software.status.verifying'), value: 'verifying' },
        { label: t('status.success'), value: 'completed' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('software.status.paused'), value: 'suspended' },
        { label: t('common.terminate'), value: 'terminated' },
      ],
    },
  ], [t]);

  // ---- Open rollback drawer ----
  const handleOpenUpgradeDrawer = () => {
    setTaskName('');
    setDrawerDevices([]);
    setDrawerProductClass('');
    setExecutionMethod('immediate');
    setScheduledTime(null);
    setSelectAllOfType(false);
    setUpgradeDrawerVisible(true);
  };

  // ---- Remove device from drawer list ----
  const handleRemoveDevice = (deviceId: string) => {
    setDrawerDevices((prev) => prev.filter((d) => d.id !== deviceId));
  };

  // ---- Add device modal ----
  const handleOpenAddDeviceModal = () => {
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    setAddDeviceModalVisible(true);
  };

  // Select all filtered devices
  const handleSelectAllDevices = (checked: boolean) => {
    if (checked) {
      setSelectedNewDevices(filteredAvailableDevices.map((d) => d.id));
    } else {
      setSelectedNewDevices([]);
    }
  };

  // Confirm add selected devices
  const handleConfirmAddDevices = () => {
    if (selectedNewDevices.length === 0) {
      void message.warning(t('software.upgrade.selectDeviceToAdd'));
      return;
    }

    const newDevices = availableDevices
      .filter((d) => selectedNewDevices.includes(d.id))
      .map((d) => ({ id: d.id, deviceSn: d.deviceSn, deviceName: d.deviceName, sourceVersion: d.sourceVersion, productClass: d.productClass }));

    setDrawerDevices((prev) => [...prev, ...newDevices]);
    setAddDeviceModalVisible(false);
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    void message.success(t('software.upgrade.addedDevices', { count: newDevices.length }));
  };

  // ---- Batch input ----
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
    setBatchInputVisible(true);
  };

  const handleBatchInputPreview = () => {
    if (!batchInputValue.trim() || !drawerProductClass) {
      setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
      return;
    }
    const sns = batchInputValue
      .split(/[\n,;，；\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);

    const matched = availableDevices.filter((d) => sns.includes(d.deviceSn));
    const matchedSns = new Set(matched.map((d) => d.deviceSn));
    const notFound = sns.filter((sn) => !matchedSns.has(sn));
    const mixedTypes = [...new Set(matched.map((d) => d.productClass))];

    setBatchInputPreview({ matched, notFound, mixedTypes });
  };

  const handleBatchInputConfirm = () => {
    const { matched } = batchInputPreview;
    if (matched.length === 0) {
      void message.warning(t('software.upgrade.selectDeviceToAdd'));
      return;
    }
    const toAdd = matched
      .filter((d) => d.productClass === drawerProductClass && !drawerDevices.some((existing) => existing.id === d.id))
      .map((d) => ({ id: d.id, deviceSn: d.deviceSn, deviceName: d.deviceName, sourceVersion: d.sourceVersion, productClass: d.productClass }));

    setDrawerDevices((prev) => [...prev, ...toAdd]);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
    void message.success(t('software.upgrade.addedDevices', { count: toAdd.length }));
  };

  // ---- Submit rollback task ----
  const handleSubmitUpgrade = () => {
    if (!taskName.trim()) {
      void message.warning(t('software.upgrade.inputTaskNameWarning'));
      return;
    }
    if (!selectAllOfType && drawerDevices.length === 0) {
      void message.warning(t('software.rollback.selectDeviceOrAll'));
      return;
    }
    if (selectAllOfType && allDevicesCountOfType === 0) {
      void message.warning(t('software.upgrade.noDevicesOfType'));
      return;
    }
    if (executionMethod === 'scheduled' && !scheduledTime) {
      void message.warning(t('software.upgrade.selectScheduleTime'));
      return;
    }

    const deviceIds = selectAllOfType
      ? (deviceListData?.items ?? []).map((d) => d.id)
      : drawerDevices.map((d) => d.id);

    createRollbackMutation.mutate(
      {
        deviceIds,
        taskName,
        createUser: 'admin',
        createSuspended: executionMethod === 'suspend',
      },
      {
        onSuccess: () => {
          const execMethodText = executionMethod === 'immediate' ? t('software.upgrade.immediateExecText') :
                                executionMethod === 'suspend' ? t('software.upgrade.suspendExecText') : t('software.upgrade.scheduledExecText', { time: scheduledTime?.format('YYYY-MM-DD HH:mm') ?? '' });
          const deviceCount = selectAllOfType ? allDevicesCountOfType : drawerDevices.length;
          const deviceInfo = selectAllOfType
            ? t('software.upgrade.productClassAll', { type: drawerProductClass, count: deviceCount })
            : t('software.upgrade.deviceCountInfo', { count: deviceCount });

          void message.success(t('software.rollback.createRollbackSuccess', { name: taskName, deviceInfo, execMethod: execMethodText }));
          setUpgradeDrawerVisible(false);
          setTaskName('');
          setSelectAllOfType(false);
        },
        onError: (err) => {
          void message.error(t('common.operationFailed') + ': ' + String(err));
        },
      },
    );
  };

  // ---- Task action handlers ----
  const handleStartTask = (record: UpgradeTaskInfo) => {
    resumeMutation.mutate(record.id, {
      onSuccess: () => {
        void message.success(t('software.upgrade.startedTask', { name: record.taskName }));
      },
      onError: (err) => {
        void message.error(t('common.operationFailed') + ': ' + String(err));
      },
    });
  };

  const handlePauseTask = (record: UpgradeTaskInfo) => {
    suspendMutation.mutate(record.id, {
      onSuccess: () => {
        void message.success(t('software.upgrade.pausedTask', { name: record.taskName }));
      },
      onError: (err) => {
        void message.error(t('common.operationFailed') + ': ' + String(err));
      },
    });
  };

  const handleTerminateTask = (record: UpgradeTaskInfo) => {
    terminateMutation.mutate(record.id, {
      onSuccess: () => {
        void message.success(t('software.upgrade.stoppedTask', { name: record.taskName }));
      },
      onError: (err) => {
        void message.error(t('common.operationFailed') + ': ' + String(err));
      },
    });
  };

  const handleViewTaskDetail = (record: UpgradeTaskInfo) => {
    setTaskDetailRecord(record);
    setSelectedTaskId(record.id);
  };

  const handleDeleteTaskConfirm = () => {
    if (deleteTaskRecord) {
      const statusCode = mapTaskStatusToCode(deleteTaskRecord.status);
      const mutation = statusCode === 4 ? deleteMutation : terminateMutation;
      mutation.mutate(deleteTaskRecord.id, {
        onSuccess: () => {
          void message.success(t('software.upgrade.deletedTask', { name: deleteTaskRecord.taskName }));
          setDeleteTaskRecord(null);
        },
        onError: (err) => {
          void message.error(t('common.operationFailed') + ': ' + String(err));
        },
      });
    }
  };

  // ---- Sub-task retry ----
  const handleRetrySubTask = (record: UpgradeSubTaskInfo) => {
    setRetrySubTask(record);
  };

  const handleRetryConfirm = () => {
    if (!retrySubTask) return;
    retryMutation.mutate(retrySubTask.taskId, {
      onSuccess: () => {
        void message.success(t('software.upgrade.startedTask', { name: retrySubTask.deviceSn ?? retrySubTask.id }));
        setRetrySubTask(null);
      },
      onError: (err) => {
        void message.error(t('common.operationFailed') + ': ' + String(err));
      },
    });
  };

  // ---- Task list columns ----
  // Status operation matrix:
  // | Backend status | Start | Pause | Terminate | Delete |
  // |----------------|-------|-------|-----------|--------|
  // | pending        | Y     |       | Y         | Y      |
  // | in_progress    |       | Y     | Y         |        |
  // | suspended      | Y     |       | Y         | Y      |
  // | ended          |       |       |           | Y      |
  const taskColumns: DataTableColumn<UpgradeTaskInfo>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: UpgradeTaskInfo) => {
        const status = mapTaskStatusToCode(record.status);

        const showStart = status === 1 || status === 3;
        const showPause = status === 2;
        const showTerminate = status === 1 || status === 2 || status === 3;
        const showDelete = status !== 2;

        const items: MenuProps['items'] = [
          showStart ? {
            key: 'start',
            label: t('common.start'),
            icon: <PlayCircleOutlined />,
            onClick: () => handleStartTask(record),
          } : null,
          showPause ? {
            key: 'pause',
            label: t('common.pause'),
            icon: <PauseOutlined />,
            onClick: () => handlePauseTask(record),
          } : null,
          showTerminate ? {
            key: 'terminate',
            label: t('common.terminate'),
            icon: <StopOutlined />,
            danger: true,
            onClick: () => handleTerminateTask(record),
          } : null,
          (showStart || showPause || showTerminate) && showDelete ? { type: 'divider' } : null,
          showDelete ? {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => setDeleteTaskRecord(record),
          } : null,
        ].filter(Boolean) as MenuProps['items'];

        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleViewTaskDetail(record)}>{t('common.details')}</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'taskName',
      title: t('software.taskName'),
      dataIndex: 'taskName',
      width: 150,
      ellipsis: true,
      render: (val: unknown, record: UpgradeTaskInfo) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleViewTaskDetail(record)}
          style={{ padding: 0 }}
        >
          {(val as string) || '-'}
        </Button>
      ),
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'createUser', width: 100 },
    { key: 'operateTime', title: t('software.operateTime'), dataIndex: 'createdAt', width: 160, render: (val: unknown) => val ? dayjs(val as string).format('YYYY-MM-DD HH:mm:ss') : '-' },
    {
      key: 'status',
      title: t('software.taskStatus'),
      dataIndex: 'status',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as string;
        const code = mapTaskStatusToCode(val);
        const cfg = TASK_STATUS_CONFIG[code] ?? { color: 'default', text: String(val) };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productClass', title: t('software.upgrade.productClass'), dataIndex: 'productClass', width: 100 },
    {
      key: 'progress',
      title: t('software.rollback.rollbackProgress'),
      width: 120,
      render: (_: unknown, record: UpgradeTaskInfo) => {
        const val = computeProgress(record);
        return <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />;
      },
    },
    {
      key: 'result',
      title: t('table.result'),
      dataIndex: 'result',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as string | undefined;
        if (!val) return '-';
        const cfg = TASK_RESULT_MAP[val as keyof typeof TASK_RESULT_MAP] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'startTime', title: t('software.startTime'), dataIndex: 'startedAt', width: 160, render: (val: unknown) => val ? dayjs(val as string).format('YYYY-MM-DD HH:mm:ss') : '-' },
    { key: 'endTime', title: t('software.endTime'), dataIndex: 'endedAt', width: 160, render: (val: unknown) => val ? dayjs(val as string).format('YYYY-MM-DD HH:mm:ss') : '-' },
  ], [t, TASK_STATUS_CONFIG, TASK_RESULT_MAP, resumeMutation, suspendMutation, terminateMutation]);

  // ---- Device list tab columns (sub-tasks for selected rollback task) ----
  const deviceColumns: DataTableColumn<UpgradeSubTaskInfo>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: UpgradeSubTaskInfo) => {
        if (record.status === 'failed') {
          return (
            <Button
              type="link"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => handleRetrySubTask(record)}
            >
              {t('software.upgrade.rerun')}
            </Button>
          );
        }
        return null;
      },
    },
    { key: 'deviceSn', title: t('software.stationCode'), dataIndex: 'deviceSn', width: 120, render: (val: unknown) => (val as string) || '-' },
    { key: 'deviceName', title: t('software.stationName'), dataIndex: 'deviceSn', width: 150, render: (val: unknown) => (val as string) || '-' },
    { key: 'targetVersion', title: t('software.rollback.originalVersion'), dataIndex: 'destVersion', width: 100, render: (val: unknown) => (val as string) || '-' },
    {
      key: 'progress',
      title: t('software.rollback.rollbackProgress'),
      width: 120,
      render: (_: unknown, record: UpgradeSubTaskInfo) => {
        const statusProgress: Record<string, number> = {
          pending: 0,
          downloading: 25,
          rebooting: 60,
          verifying: 85,
          completed: 100,
          failed: 100,
          suspended: 0,
          terminated: 100,
        };
        const val = statusProgress[record.status] ?? 0;
        return <Progress percent={val} size="small" status={record.status === 'completed' ? 'success' : record.status === 'failed' ? 'exception' : 'active'} />;
      },
    },
    {
      key: 'result',
      title: t('table.result'),
      dataIndex: 'status',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as string;
        const cfg = SUB_TASK_STATUS_MAP[val as keyof typeof SUB_TASK_STATUS_MAP] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'failureReason', title: t('software.failureReason'), dataIndex: 'errorMessage', width: 150, ellipsis: true, render: (val: unknown, record: UpgradeSubTaskInfo) => {
      const text = record.failureReason || (val as string);
      return text ? <span style={{ color: '#ff4d4f' }}>{text}</span> : '-';
    }},
    { key: 'operator', title: t('table.operator'), width: 100, render: () => '-' },
    { key: 'operateTime', title: t('software.operateTime'), dataIndex: 'createdAt', width: 160, render: (val: unknown) => val ? dayjs(val as string).format('YYYY-MM-DD HH:mm:ss') : '-' },
    { key: 'startTime', title: t('software.startTime'), dataIndex: 'startedAt', width: 160, render: (val: unknown) => val ? dayjs(val as string).format('YYYY-MM-DD HH:mm:ss') : '-' },
    { key: 'endTime', title: t('software.endTime'), dataIndex: 'completedAt', width: 160, render: (val: unknown) => val ? dayjs(val as string).format('YYYY-MM-DD HH:mm:ss') : '-' },
  ], [t, SUB_TASK_STATUS_MAP]);

  // ---- Header buttons ----
  const headerExtra = useMemo(() => (
    <Space>
      <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleOpenUpgradeDrawer}>
        {t('software.rollback.rollback')}
      </Button>
      <Button icon={<DownloadOutlined />} onClick={handleExport}>
        {t('common.export')}
      </Button>
    </Space>
  ), [t, handleExport]);

  return (
    <ListPageLayout title={t('nav.software.versionRollback')} extra={headerExtra}>
      {/* Tab selector */}
      <Radio.Group
        value={activeTab}
        onChange={(e) => {
          const nextTab = e.target.value;
          setActiveTab(nextTab);
          setFilters({});
          setPage(1);
          setDeviceFilters({});
          setDevicePage(1);
          if (nextTab === 'task') void refetchTasks();
          else void refetchDeviceSubTasks();
        }}
        optionType="button"
        buttonStyle="solid"
        style={{ marginBottom: 12 }}
      >
        <Radio.Button value="task">{t('software.upgrade.taskList')}</Radio.Button>
        <Radio.Button value="device">{t('software.upgrade.deviceList')}</Radio.Button>
      </Radio.Group>

      {/* Search form */}
      <FilterBar
        filterId={`rollback-filter-${activeTab}`}
        fields={activeTab === 'task' ? taskFilterFields : deviceFilterFields}
        onSearch={(vals) => {
          if (activeTab === 'task') { setFilters(vals); setPage(1); }
          else { setDeviceFilters(vals); setDevicePage(1); }
        }}
        onReset={() => {
          if (activeTab === 'task') { setFilters({}); setPage(1); }
          else { setDeviceFilters({}); setDevicePage(1); }
        }}
      />

      {/* Task list tab */}
      {activeTab === 'task' ? (
        <Card
          size="small"
          bordered
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
          <DataTable<UpgradeTaskInfo>
            tableId="rollback-list-task"
            columns={taskColumns}
            dataSource={taskList}
            rowKey="id"
            total={taskTotal}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            loading={tasksLoading}
            scroll={{ x: 'max-content', y: 'calc(100vh - 540px)' }}
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
          />
        </Card>
      ) : (
        // Device list tab - all sub-tasks across all rollback tasks
        <Card
          size="small"
          bordered
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
          <DataTable<UpgradeSubTaskInfo>
            tableId="rollback-list-device"
            columns={deviceColumns}
            dataSource={deviceSubTaskList}
            rowKey="id"
            total={deviceSubTaskTotal}
            currentPage={devicePage}
            pageSize={devicePageSize}
            onPageChange={(p, s) => { setDevicePage(p); setDevicePageSize(s); }}
            loading={deviceSubTasksLoading}
            scroll={{ x: 'max-content', y: 'calc(100vh - 540px)' }}
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
          />
        </Card>
      )}

      {/* Batch input modal */}
      <Modal
        title={t('software.upgrade.batchInputTitle', { type: drawerProductClass || t('software.upgrade.selectProductClassFirst') })}
        open={batchInputVisible}
        onCancel={() => setBatchInputVisible(false)}
        onOk={handleBatchInputConfirm}
        okText={t('software.upgrade.confirmAdd')}
        cancelText={t('common.cancel')}
        width={600}
        okButtonProps={{
          disabled: batchInputPreview.matched.length === 0 || !drawerProductClass,
        }}
      >
        {!drawerProductClass && (
          <Alert
            type="warning"
            showIcon
            message={t('software.upgrade.selectProductClassFirst')}
            style={{ marginBottom: 16 }}
          />
        )}

        <div style={{ marginBottom: 16 }}>
          <Input.TextArea
            placeholder={`${t('software.upgrade.batchInputPlaceholder')}\n${t('software.upgrade.batchInputExample') ?? '例如：\nENB00001\nENB00002, ENB00003; GNB00001'}`}
            rows={6}
            value={batchInputValue}
            onChange={(e) => setBatchInputValue(e.target.value)}
            onBlur={handleBatchInputPreview}
          />
        </div>

        {/* Preview results */}
        {batchInputPreview.matched.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Alert
              type={batchInputPreview.mixedTypes.includes(drawerProductClass) ? 'success' : 'warning'}
              showIcon
              icon={!batchInputPreview.mixedTypes.includes(drawerProductClass) ? <WarningOutlined /> : undefined}
              message={
                <Space direction="vertical" size="small">
                  <span>
                    {t('software.upgrade.matchedDevices', { count: batchInputPreview.matched.length })}
                    {batchInputPreview.mixedTypes.includes(drawerProductClass) && (
                      <Tag color="blue" style={{ marginLeft: 8 }}>{t('software.upgrade.addableCount', { count: batchInputPreview.matched.filter(d => d.productClass === drawerProductClass).length })}</Tag>
                    )}
                  </span>
                  {!batchInputPreview.mixedTypes.includes(drawerProductClass) && drawerProductClass && (
                    <span style={{ color: '#faad14' }}>
                      <WarningOutlined style={{ marginRight: 4 }} />
                      {t('software.upgrade.noMatchedType', { type: drawerProductClass })}
                    </span>
                  )}
                </Space>
              }
              style={{ marginBottom: 8 }}
            />
          </div>
        )}

        {/* Not found devices */}
        {batchInputPreview.notFound.length > 0 && (
          <Alert
            type="warning"
            showIcon
            message={
              <div>
                <div>{t('software.upgrade.notFoundSns', { count: batchInputPreview.notFound.length })}</div>
                <div style={{ maxHeight: 80, overflow: 'auto', marginTop: 4 }}>
                  {batchInputPreview.notFound.map((sn) => (
                    <Tag key={sn} style={{ margin: '2px' }}>{sn}</Tag>
                  ))}
                </div>
              </div>
            }
            style={{ marginBottom: 8 }}
          />
        )}
      </Modal>

      {/* Retry confirm modal */}
      <Modal
        title={t('software.upgrade.confirmRerun')}
        open={!!retrySubTask}
        onCancel={() => setRetrySubTask(null)}
        onOk={handleRetryConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                {t('software.rollback.confirmRerunRollback')}
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>{t('software.upgrade.stationCodeLabel')}</strong>{retrySubTask?.deviceSn ?? '-'}<br />
                <strong>{t('software.upgrade.targetVersionLabel')}</strong>{retrySubTask?.destVersion ?? '-'}
              </p>
            </div>
          }
        />
      </Modal>

      {/* Delete task confirm modal */}
      <Modal
        title={t('common.confirmDelete')}
        open={!!deleteTaskRecord}
        onCancel={() => setDeleteTaskRecord(null)}
        onOk={handleDeleteTaskConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                {t('software.rollback.confirmDeleteRollback')}
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>{t('software.upgrade.taskNameLabel')}</strong>{deleteTaskRecord?.taskName}<br />
                <strong>{t('software.upgrade.operatorLabel')}</strong>{deleteTaskRecord?.createUser}<br />
                <strong>{t('software.upgrade.rollbackVersionLabel')}</strong>{deleteTaskRecord?.fileName ?? '-'}
              </p>
            </div>
          }
        />
      </Modal>

      {/* Batch rollback drawer */}
      <Drawer
        title={t('software.rollback.batchRollback')}
        placement="right"
        width={600}
        open={upgradeDrawerVisible}
        onClose={() => setUpgradeDrawerVisible(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setUpgradeDrawerVisible(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              onClick={handleSubmitUpgrade}
              disabled={(!selectAllOfType && drawerDevices.length === 0) || createRollbackMutation.isPending}
              loading={createRollbackMutation.isPending}
            >
              {t('software.rollback.confirmRollback')}
            </Button>
          </Space>
        }
      >
        <Form layout="vertical" size="small">
          {/* Task name */}
          <Form.Item label={t('software.upgrade.taskName')} required>
            <Input
              value={taskName}
              onChange={(e) => setTaskName(e.target.value)}
              placeholder={t('software.upgrade.inputTaskName')}
              maxLength={100}
              showCount
            />
          </Form.Item>

          {/* Product type */}
          <Form.Item label={t('software.upgrade.productClass')} required>
            <Select
              value={drawerProductClass}
              onChange={(val) => {
                setDrawerProductClass(val);
                setSelectAllOfType(false);
                setDrawerDevices((prev) => prev.filter((d) => d.productClass === val));
              }}
              options={productClassOptions}
              style={{ width: '100%' }}
            />
          </Form.Item>

          {/* Select all of product type */}
          <Form.Item>
            <Checkbox
              checked={selectAllOfType}
              onChange={(e) => setSelectAllOfType(e.target.checked)}
              disabled={!drawerProductClass}
            >
              {t('software.rollback.rollbackAllOfType')}
              {drawerProductClass && (
                <Tag color="blue" style={{ marginLeft: 8 }}>{t('software.upgrade.totalDevices', { count: allDevicesCountOfType })}</Tag>
              )}
            </Checkbox>
          </Form.Item>

          {/* Selected rollback devices */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
              <span>
                {t('software.rollback.selectedRollbackDevices')}
                {' '}
                <Tag color="blue">{selectAllOfType ? allDevicesCountOfType : drawerDevices.length} {t('common.devices')}</Tag>
              </span>
              {!selectAllOfType && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleOpenBatchInput}
                  disabled={!drawerProductClass}
                >
                  {t('software.upgrade.batchInput')}
                </Button>
              )}
            </div>
          }>
            {selectAllOfType ? (
              <Alert
                type="info"
                showIcon
                message={t('software.rollback.selectedAllRollbackInfo', { type: drawerProductClass, count: allDevicesCountOfType })}
                description={t('software.rollback.selectedAllRollbackDesc')}
              />
            ) : (
              <>
                <div style={{ maxHeight: 200, overflow: 'auto', border: '1px solid #d9d9d9', borderRadius: 6 }}>
                  <Table
                    size="small"
                    dataSource={drawerDevices}
                    rowKey="id"
                    pagination={false}
                    columns={[
                      { title: t('software.stationCode'), dataIndex: 'deviceSn', width: 100 },
                      { title: t('software.stationName'), dataIndex: 'deviceName', ellipsis: true },
                      { title: t('software.upgrade.currentVersion'), dataIndex: 'sourceVersion', width: 80 },
                      {
                        title: '',
                        width: 40,
                        render: (_: unknown, record: { id: string }) => (
                          <Button
                            type="text"
                            size="small"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => handleRemoveDevice(record.id)}
                          />
                        ),
                      },
                    ]}
                  />
                </div>
                <div style={{ marginTop: 8 }}>
                  <Button
                    type="dashed"
                    icon={<PlusOutlined />}
                    onClick={handleOpenAddDeviceModal}
                    style={{ width: '100%' }}
                    disabled={!drawerProductClass}
                  >
                    {t('software.upgrade.addDevice')}
                  </Button>
                </div>
              </>
            )}
          </Form.Item>

          <Divider />

          {/* Execution method */}
          <Form.Item label={t('software.upgrade.executionMethod')} required>
            <Radio.Group value={executionMethod} onChange={(e) => setExecutionMethod(e.target.value)}>
              <Radio value="immediate">{t('software.upgrade.immediateExecute')}</Radio>
              <Radio value="suspend">{t('software.upgrade.suspendExecute')}</Radio>
              <Radio value="scheduled">{t('software.upgrade.scheduledExecute')}</Radio>
            </Radio.Group>
          </Form.Item>

          {/* Scheduled time */}
          {executionMethod === 'scheduled' && (
            <Form.Item label={t('software.upgrade.executeTime')} required>
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm"
                value={scheduledTime}
                onChange={setScheduledTime}
                placeholder={t('software.upgrade.selectExecuteTime')}
                style={{ width: '100%' }}
                disabledDate={(current) => current && current < dayjs().startOf('day')}
              />
            </Form.Item>
          )}
        </Form>
      </Drawer>

      {/* Task detail drawer */}
      <Drawer
        title={t('software.upgrade.taskDetail')}
        placement="right"
        width={720}
        open={!!taskDetailRecord}
        onClose={() => { setTaskDetailRecord(null); setSelectedTaskId(''); }}
        footer={null}
      >
        {taskDetailRecord && (() => {
          const totalDevices = taskDetailRecord.totalCount;
          const resultCode = mapTaskStatusToCode(taskDetailRecord.status);
          const progress = computeProgress(taskDetailRecord);

          // Compute result stats from sub-tasks
          const resultStats = {
            completed: subTaskList.filter((d) => d.status === 'completed').length,
            failed: subTaskList.filter((d) => d.status === 'failed').length,
            downloading: subTaskList.filter((d) => d.status === 'downloading' || d.status === 'rebooting' || d.status === 'verifying').length,
            pending: subTaskList.filter((d) => d.status === 'pending').length,
            suspended: subTaskList.filter((d) => d.status === 'suspended').length,
            terminated: subTaskList.filter((d) => d.status === 'terminated').length,
          };

          return (
            <>
              {/* Task basic info */}
              <Descriptions column={2} bordered size="small" style={{ marginBottom: 16 }}>
                <Descriptions.Item label={t('software.taskName')} span={2}>{taskDetailRecord.taskName}</Descriptions.Item>
                <Descriptions.Item label={t('software.upgrade.productClass')}>{taskDetailRecord.productClass}</Descriptions.Item>
                <Descriptions.Item label={t('software.rollback.rollbackType')}>
                  <Tag color="orange">{t('software.rollback.rollback')}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('software.rollback.originalVersion')}>{taskDetailRecord.fileName ?? '-'}</Descriptions.Item>
                <Descriptions.Item label={t('software.upgrade.keepConfig')}>
                  <Checkbox checked={taskDetailRecord.isKeepConfig} disabled />
                </Descriptions.Item>
                <Descriptions.Item label={t('table.operator')}>{taskDetailRecord.createUser}</Descriptions.Item>
                <Descriptions.Item label={t('software.operateTime')}>{taskDetailRecord.createdAt ? dayjs(taskDetailRecord.createdAt).format('YYYY-MM-DD HH:mm:ss') : '-'}</Descriptions.Item>
                <Descriptions.Item label={t('software.taskStatus')}>
                  <Tag color={TASK_STATUS_CONFIG[resultCode]?.color}>{TASK_STATUS_CONFIG[resultCode]?.text}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('software.startTime')}>{taskDetailRecord.startedAt ? dayjs(taskDetailRecord.startedAt).format('YYYY-MM-DD HH:mm:ss') : '-'}</Descriptions.Item>
                <Descriptions.Item label={t('software.endTime')}>{taskDetailRecord.endedAt ? dayjs(taskDetailRecord.endedAt).format('YYYY-MM-DD HH:mm:ss') : '-'}</Descriptions.Item>
              </Descriptions>

              {/* Progress overview */}
              <Card title={t('software.upgrade.progressOverview')} size="small" style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                  <div style={{ flex: '0 0 120px', textAlign: 'center' }}>
                    <Progress
                      type="circle"
                      percent={progress}
                      size={80}
                      status={progress === 100 ? 'success' : 'active'}
                    />
                    <div style={{ marginTop: 8, color: '#666' }}>{t('software.upgrade.overallProgress')}</div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <div>
                        <Tag color="success">{t('status.success')}</Tag>
                        <span>{resultStats.completed} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="error">{t('status.failed')}</Tag>
                        <span>{resultStats.failed} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="processing">{t('software.status.rollingBack')}</Tag>
                        <span>{resultStats.downloading} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="warning">{t('software.status.paused')}</Tag>
                        <span>{resultStats.suspended} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="default">{t('software.status.waiting')}</Tag>
                        <span>{resultStats.pending} {t('common.devices')}</span>
                      </div>
                    </Space>
                  </div>
                  <Divider type="vertical" style={{ height: 120 }} />
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 32, fontWeight: 'bold', color: '#1890ff' }}>{totalDevices}</div>
                    <div style={{ color: '#666' }}>{t('software.upgrade.deviceTotal')}</div>
                  </div>
                </div>
              </Card>

              {/* Device list (sub-tasks) */}
              <Card title={`${t('software.upgrade.deviceListTitle', { count: subTaskList.length })}`} size="small">
                <Table
                  size="small"
                  dataSource={subTaskList}
                  rowKey="id"
                  loading={subTasksLoading}
                  pagination={subTaskList.length > 10 ? { pageSize: 10 } : false}
                  locale={{ emptyText: subTasksLoading ? undefined : t('software.upgrade.noSubTasks') ?? '暂无设备数据' }}
                  scroll={{ y: 300 }}
                  columns={[
                    {
                      title: t('software.stationCode'),
                      dataIndex: 'deviceSn',
                      width: 100,
                      render: (val: unknown) => (val as string) || '-',
                    },
                    {
                      title: t('software.stationName'),
                      dataIndex: 'deviceSn',
                      ellipsis: true,
                      render: (val: unknown) => (val as string) || '-',
                    },
                    {
                      title: t('software.upgrade.currentVersion'),
                      dataIndex: 'oriVersion',
                      width: 80,
                      render: (val: unknown) => (val as string) || '-',
                    },
                    {
                      title: t('software.progress'),
                      width: 100,
                      render: (_: unknown, record: UpgradeSubTaskInfo) => {
                        const statusProgress: Record<string, number> = {
                          pending: 0, downloading: 25, rebooting: 60, verifying: 85,
                          completed: 100, failed: 100, suspended: 0, terminated: 100,
                        };
                        const val = statusProgress[record.status] ?? 0;
                        return <Progress percent={val} size="small" status={record.status === 'completed' ? 'success' : record.status === 'failed' ? 'exception' : 'active'} />;
                      },
                    },
                    {
                      title: t('table.result'),
                      dataIndex: 'status',
                      width: 80,
                      render: (val: string) => {
                        const cfg = SUB_TASK_STATUS_MAP[val as keyof typeof SUB_TASK_STATUS_MAP] ?? { color: 'default', text: val };
                        return <Tag color={cfg.color}>{cfg.text}</Tag>;
                      },
                    },
                    {
                      title: t('software.failureReason'),
                      dataIndex: 'errorMessage',
                      width: 120,
                      ellipsis: true,
                      render: (val: string, record: UpgradeSubTaskInfo) => {
                        const text = record.failureReason || val;
                        return text ? <span style={{ color: '#ff4d4f' }}>{text}</span> : '-';
                      },
                    },
                  ]}
                />
              </Card>
            </>
          );
        })()}
      </Drawer>

      {/* Add device modal */}
      <Modal
        title={`${t('software.upgrade.addDevice')} - ${drawerProductClass}`}
        open={addDeviceModalVisible}
        onCancel={() => setAddDeviceModalVisible(false)}
        onOk={handleConfirmAddDevices}
        okText={t('software.upgrade.confirmAdd')}
        cancelText={t('common.cancel')}
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message={t('software.upgrade.noAvailableDevices')}
            description={t('software.upgrade.allDevicesInList', { type: drawerProductClass })}
          />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={t('software.upgrade.deviceSelectable', { total: availableDevices.length, shown: filteredAvailableDevices.length, selected: selectedNewDevices.length })}
              style={{ marginBottom: 16 }}
            />

            {/* Search and select all */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input.Search
                placeholder={t('software.upgrade.searchStationCodeOrName')}
                value={addDeviceKeyword}
                onChange={(e) => setAddDeviceKeyword(e.target.value)}
                style={{ width: 250 }}
                allowClear
              />
              <Checkbox
                checked={selectedNewDevices.length === filteredAvailableDevices.length && filteredAvailableDevices.length > 0}
                indeterminate={selectedNewDevices.length > 0 && selectedNewDevices.length < filteredAvailableDevices.length}
                onChange={(e) => handleSelectAllDevices(e.target.checked)}
              >
                {t('common.selectAll')} ({filteredAvailableDevices.length} {t('common.devices')})
              </Checkbox>
            </div>

            <Table
              size="small"
              dataSource={filteredAvailableDevices}
              rowKey="id"
              pagination={false}
              scroll={{ y: 250 }}
              rowSelection={{
                selectedRowKeys: selectedNewDevices,
                onChange: (keys) => setSelectedNewDevices(keys),
              }}
              columns={[
                { title: t('software.stationCode'), dataIndex: 'deviceSn', width: 120 },
                { title: t('software.stationName'), dataIndex: 'deviceName', ellipsis: true },
                { title: t('software.deviceGroup'), dataIndex: 'deviceGroup', width: 100 },
                { title: t('software.upgrade.currentVersion'), dataIndex: 'sourceVersion', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>
    </ListPageLayout>
  );
}
