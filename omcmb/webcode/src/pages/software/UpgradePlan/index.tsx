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
  InputNumber,
  Form,
  Divider,
  Table,
  Card,
  Descriptions,
} from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { PlayCircleOutlined, WarningOutlined, PlusOutlined, ReloadOutlined, DownloadOutlined, DeleteOutlined, PauseOutlined, StopOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useUpgradeTasks,
  useCreateUpgradeTask,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useDeleteTask,
  useRetryTask,
  useSubTasks,
  useAllSubTasks,
  useSoftwareVersions,
  // T-0019: canary stage transitions
  useAdvanceCanary,
  usePauseCanary,
  useResumeCanary,
  useAbortCanary,
} from '@core/hooks/api/useSoftware';
import { useDeviceList, useProductClasses } from '@core/hooks/api/useDevices';
import { useProductList } from '@core/hooks/api/useProducts';
import type { UpgradeTaskInfo, UpgradeSubTaskInfo } from '@core/mock/data/software';
import { formatSystemTime } from '@core/utils/systemTime';

// Upgrade category enum
type UpgradeCategory = 'software' | 'patch' | 'fpga';
// Execution method enum
type ExecutionMethod = 'immediate' | 'suspend' | 'scheduled';

// Map backend task status to display status code (1-4)
function mapTaskStatusToCode(status: string): number {
  const map: Record<string, number> = {
    pending: 1,
    in_progress: 2,
    suspended: 3,
    ended: 4,
  };
  return map[status] ?? 1;
}

// Task status display color
const TASK_STATUS_COLORS: Record<number, string> = {
  1: 'default',     // pending
  2: 'processing',  // in_progress
  3: 'warning',     // suspended
  4: 'success',     // ended
};

// Task type display color
const TASK_TYPE_COLORS: Record<number, string> = {
  1: 'green',   // IMG upgrade
  2: 'orange',  // rollback
  4: 'blue',    // patch
  6: 'purple',  // FPGA
};

// TASK_RESULT_COLORS / SUB_TASK_STATUS_COLORS 占位常量已拆到同级 ./constants.ts
// （react-refresh/only-export-components：页面文件只导出组件）。

export default function UpgradePlan() {
  const t = useT();

  // ---- Dynamic product type options from API ----
  const { data: productClassesData } = useProductClasses();
  // #638：用产品中心-产品列表渲染"产品名"选项；同 FirmwareUpload 取数源，保证名称一致。
  const { data: productsData } = useProductList();
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
  // #638：filters 需被读取才能传入 useUpgradeTasks（之前 setFilters 写了不读走丢了过滤语义，顺便修复）。
  const [filters, setFilters] = useState<Record<string, unknown>>({});

  // ---- Tab state ----
  const [activeTab, setActiveTab] = useState<'task' | 'device'>('task');

  // ---- Fetch main upgrade tasks from backend ----
  const { data: tasksData, isLoading: tasksLoading, refetch: refetchTasks } = useUpgradeTasks({
    page,
    pageSize,
    taskType: 1, // Only show upgrade tasks (not rollback)
    // #638：按“产品名”过滤。后端以任务选中固件的 product_ids 包含该 pid 为判据，不需修改 upgrade_tasks 表。
    productId: (filters.productId as string) || undefined,
  });

  // ---- Device list tab: pagination & filter ----
  const [devicePage, setDevicePage] = useState(1);
  const [devicePageSize, setDevicePageSize] = useState(20);
  const [deviceFilters, setDeviceFilters] = useState<Record<string, unknown>>({});
  const { data: deviceSubTasksData, isLoading: deviceSubTasksLoading, refetch: refetchDeviceSubTasks } = useAllSubTasks({
    page: devicePage,
    pageSize: devicePageSize,
    taskName: (deviceFilters.keyword as string) || undefined,
    deviceSn: (deviceFilters.stationCode as string) || undefined,
    status: (deviceFilters.status as string) || undefined,
    taskType: 1,
  });

  // ---- Fetch sub-tasks for the detail drawer ----
  const [selectedTaskId, setSelectedTaskId] = useState<string>('');
  const { data: subTasksData, isLoading: subTasksLoading } = useSubTasks(selectedTaskId, { page: 1, pageSize: 100 });

  // ---- Fetch firmware versions for upgrade file selection ----
  const { data: firmwareData } = useSoftwareVersions({ page: 1, pageSize: 100 });

  // ---- Mutations ----
  const createTaskMutation = useCreateUpgradeTask();
  const suspendMutation = useSuspendTask();
  const resumeMutation = useResumeTask();
  const terminateMutation = useTerminateTask();
  const deleteMutation = useDeleteTask();
  const retryMutation = useRetryTask();

  // T-0019: canary mutations
  const advanceCanaryMutation = useAdvanceCanary();
  const pauseCanaryMutation = usePauseCanary();
  const resumeCanaryMutation = useResumeCanary();
  const abortCanaryMutation = useAbortCanary();

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

  // Task type config with i18n
  const TASK_TYPE_MAP = useMemo((): Record<number, { color: string; text: string }> => ({
    1: { color: TASK_TYPE_COLORS[1], text: t('software.upgrade.softwareUpgrade') },
    2: { color: TASK_TYPE_COLORS[2], text: t('software.rollback.rollback') },
    4: { color: TASK_TYPE_COLORS[4], text: t('software.upgrade.patchUpgrade') },
    6: { color: TASK_TYPE_COLORS[6], text: t('software.upgrade.fpgaUpgrade') },
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

  // ---- Create upgrade drawer state ----
  const [upgradeDrawerVisible, setUpgradeDrawerVisible] = useState(false);
  const [taskName, setTaskName] = useState('');
  const [upgradeCategory, setUpgradeCategory] = useState<UpgradeCategory>('software');
  const [executionMethod, setExecutionMethod] = useState<ExecutionMethod>('immediate');
  const [scheduledTime, setScheduledTime] = useState<Dayjs | null>(null);
  const [upgradeFile, setUpgradeFile] = useState<string | undefined>(undefined);
  const [drawerKeepConfig, setDrawerKeepConfig] = useState(true);
  const [retryOffline, setRetryOffline] = useState(true);
  const [batchSize, setBatchSize] = useState(20);
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

  // ---- Derived data ----

  // Task list from API
  const taskList = useMemo(() => tasksData?.items ?? [], [tasksData]);
  const taskTotal = tasksData?.total ?? 0;

  // Sub-tasks for detail drawer
  const subTaskList = useMemo(() => subTasksData?.items ?? [], [subTasksData]);

  // Sub-tasks for device list tab
  const deviceSubTaskList = useMemo(() => deviceSubTasksData?.items ?? [], [deviceSubTasksData]);
  const deviceSubTaskTotal = deviceSubTasksData?.total ?? 0;

  // Firmware files for upgrade file selection
  const firmwareList = useMemo(() => {
    const allFiles = firmwareData?.items ?? [];
    return allFiles.filter((f) => {
      // Support comma-separated multi-product-type: match if selected type is in the list
      if (drawerProductClass && f.deviceType) {
        const supportedTypes = f.deviceType.split(',').map((s) => s.trim()).filter(Boolean);
        if (!supportedTypes.includes(drawerProductClass)) return false;
      }
      if (upgradeCategory === 'software') return f.fileType === 0 || f.fileType === undefined;
      if (upgradeCategory === 'patch') return f.fileType === 1;
      if (upgradeCategory === 'fpga') return f.fileType === 6;
      return true;
    });
  }, [firmwareData, drawerProductClass, upgradeCategory]);

  // Filtered firmware options for select
  const filteredFiles = useMemo(() =>
    firmwareList.map((f) => ({
      label: `${f.versionCode} [${f.deviceType}]`,
      value: f.id,
    })),
  [firmwareList]);

  // ---- Helpers ----

  function computeProgress(task: UpgradeTaskInfo): number {
    if (task.totalCount === 0) return 0;
    return Math.round(((task.successCount + task.failCount) / task.totalCount) * 100);
  }

  // ---- Export ----
  const handleExport = useCallback(() => {
    if (taskList.length === 0) {
      void message.warning(t('software.upgrade.exportNoData'));
      return;
    }

    const headers = [t('software.taskName'), t('table.operator'), t('software.taskStatus'), t('software.upgrade.productClass'), t('software.upgrade.targetVersion'), t('software.upgrade.upgradeProgress'), t('table.result'), t('software.startTime'), t('software.endTime')];
    const rows = taskList.map((task) => [
      task.taskName,
      task.createUser,
      TASK_STATUS_CONFIG[mapTaskStatusToCode(task.status)]?.text ?? task.status,
      task.productClass,
      task.fileName ?? '',
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
    link.download = `${t('software.upgrade.exportFileName')}_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);

    void message.success(t('software.upgrade.exportSuccess', { count: taskList.length }));
  }, [taskList, TASK_STATUS_CONFIG, TASK_RESULT_MAP, t]);

  // ---- Filter definitions ----

  // Task list filter fields (only task name and time)
  // #638：增加“产品名”选项，选产品名 → 传 product_id → 后端以固件适用产品集合包含为准过滤。
  const productNameOptions = useMemo(
    () =>
      (productsData?.items ?? [])
        .map((p) => ({ label: p.name, value: p.id }))
        .sort((a, b) => a.label.localeCompare(b.label)),
    [productsData],
  );
  const taskFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.taskName'), type: 'input', placeholder: t('software.upgrade.inputTaskName') },
    {
      name: 'productId',
      label: t('software.firmware.productClass'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: [{ label: t('common.all'), value: '' }, ...productNameOptions],
    },
    {
      name: 'timeRange',
      label: t('common.timeRange') ?? '时间范围',
      type: 'date-range',
      placeholder: t('common.selectTimeRange') ?? '请选择时间范围',
    },
  ], [t, productNameOptions]);

  // Device list filter fields
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

  // ---- Open upgrade drawer ----
  const handleOpenUpgradeDrawer = () => {
    setTaskName('');
    setDrawerDevices([]);
    setDrawerProductClass('');
    setUpgradeCategory('software');
    setExecutionMethod('immediate');
    setScheduledTime(null);
    setUpgradeFile(undefined);
    setDrawerKeepConfig(true);
    setRetryOffline(true);
    setBatchSize(20);
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

  // Fetch real devices from API by product_class
  const { data: deviceListData } = useDeviceList(
    {
      productClass: drawerProductClass,
      page: 1,
      pageSize: 500,
    },
    { refetchInterval: 0 },
  );

  // Available devices: real data from API, excluding already added ones
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

  // Total devices count for selected product type (from API)
  const allDevicesCountOfType = deviceListData?.total ?? 0;

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
    // Only add devices matching the selected product type
    const toAdd = matched
      .filter((d) => d.productClass === drawerProductClass && !drawerDevices.some((existing) => existing.id === d.id))
      .map((d) => ({ id: d.id, deviceSn: d.deviceSn, deviceName: d.deviceName, sourceVersion: d.sourceVersion, productClass: d.productClass }));

    setDrawerDevices((prev) => [...prev, ...toAdd]);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
    void message.success(t('software.upgrade.addedDevices', { count: toAdd.length }));
  };

  // ---- Submit upgrade task ----
  const handleSubmitUpgrade = () => {
    // Validate task name
    if (!taskName.trim()) {
      void message.warning(t('software.upgrade.inputTaskNameWarning'));
      return;
    }
    // Validate device selection
    if (!selectAllOfType && drawerDevices.length === 0) {
      void message.warning(t('software.upgrade.selectDeviceOrAll'));
      return;
    }
    if (selectAllOfType && allDevicesCountOfType === 0) {
      void message.warning(t('software.upgrade.noDevicesOfType'));
      return;
    }
    if (!upgradeFile) {
      void message.warning(t('software.upgrade.selectUpgradeFileWarn'));
      return;
    }
    if (executionMethod === 'scheduled' && !scheduledTime) {
      void message.warning(t('software.upgrade.selectScheduleTime'));
      return;
    }

    const deviceIds = selectAllOfType
      ? (deviceListData?.items ?? []).map((d) => d.id)
      : drawerDevices.map((d) => d.id);

    createTaskMutation.mutate(
      {
        deviceIds,
        firmwareId: upgradeFile,
        taskName,
        taskType: upgradeCategory === 'software' ? 1 : upgradeCategory === 'patch' ? 4 : 6,
        isKeepConfig: drawerKeepConfig,
        concurrency: batchSize,
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

          void message.success(t('software.upgrade.createTaskSuccess', { name: taskName, deviceInfo, execMethod: execMethodText }));
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
      deleteMutation.mutate(deleteTaskRecord.id, {
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
      width: 220,
      fixed: 'right',
      render: (_: unknown, record: UpgradeTaskInfo) => {
        const status = mapTaskStatusToCode(record.status);

        const showStart = status === 1 || status === 3;
        const showPause = status === 2;
        const showTerminate = status === 1 || status === 2 || status === 3;
        const showDelete = status !== 2;

        // T-0019: canary stage transition controls. Only available when the
        // task was created with strategy='canary' and the stage is in a
        // non-terminal state.
        const isCanary = record.strategy === 'canary';
        const stageStatus = record.stageStatus ?? 'pending';
        const canaryRunning = isCanary && stageStatus === 'running';
        const canaryPaused = isCanary && stageStatus === 'paused';

        return (
          <Space size={4} wrap={false}>
            <Button type="link" size="small" onClick={() => handleViewTaskDetail(record)}>{t('common.details')}</Button>
            {showStart ? (
              <Button type="link" size="small" icon={<PlayCircleOutlined />} loading={resumeMutation.isPending} onClick={() => handleStartTask(record)}>
                {t('common.start')}
              </Button>
            ) : null}
            {showPause ? (
              <Button type="link" size="small" icon={<PauseOutlined />} loading={suspendMutation.isPending} onClick={() => handlePauseTask(record)}>
                {t('common.pause')}
              </Button>
            ) : null}
            {showTerminate ? (
              <Button type="link" size="small" danger icon={<StopOutlined />} loading={terminateMutation.isPending} onClick={() => handleTerminateTask(record)}>
                {t('common.terminate')}
              </Button>
            ) : null}
            {isCanary && canaryRunning ? (
              <Button type="link" size="small" icon={<PlayCircleOutlined />} loading={advanceCanaryMutation.isPending} onClick={() => advanceCanaryMutation.mutate(record.id)}>
                {t('software.canary.advance') || '推进下一阶段'}
              </Button>
            ) : null}
            {isCanary && canaryRunning ? (
              <Button type="link" size="small" icon={<PauseOutlined />} loading={pauseCanaryMutation.isPending} onClick={() => pauseCanaryMutation.mutate(record.id)}>
                {t('software.canary.pause') || '暂停灰度'}
              </Button>
            ) : null}
            {isCanary && canaryPaused ? (
              <Button type="link" size="small" icon={<PlayCircleOutlined />} loading={resumeCanaryMutation.isPending} onClick={() => resumeCanaryMutation.mutate(record.id)}>
                {t('software.canary.resume') || '恢复灰度'}
              </Button>
            ) : null}
            {isCanary && (canaryRunning || canaryPaused) ? (
              <Button type="link" size="small" danger icon={<StopOutlined />} loading={abortCanaryMutation.isPending} onClick={() => abortCanaryMutation.mutate(record.id)}>
                {t('software.canary.abort') || '终止灰度'}
              </Button>
            ) : null}
            {showDelete ? (
              <Button type="link" size="small" danger icon={<DeleteOutlined />} loading={deleteMutation.isPending} onClick={() => setDeleteTaskRecord(record)}>
                {t('common.delete')}
              </Button>
            ) : null}
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
    { key: 'operateTime', title: t('software.operateTime'), dataIndex: 'createdAt', width: 160, render: (val: unknown) => val ? formatSystemTime(val as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-' },
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
    { key: 'targetVersion', title: t('software.upgrade.targetVersion'), dataIndex: 'fileName', width: 120, render: (val: unknown) => (val as string) || '-' },
    {
      key: 'canaryStage',
      title: t('software.canary.stage') || '灰度阶段',
      width: 160,
      render: (_unused: unknown, record: UpgradeTaskInfo) => {
        if (record.strategy !== 'canary') {
          return <span style={{ color: '#999' }}>-</span>;
        }
        const cur = record.currentStage ?? 0;
        const total = record.canaryStages?.length ?? 0;
        const pct = record.canaryStages?.[cur - 1]?.percent;
        const status = record.stageStatus ?? 'pending';
        const statusColorMap: Record<string, string> = {
          pending: 'default',
          running: 'processing',
          paused: 'warning',
          aborted: 'error',
          completed: 'success',
        };
        return (
          <Space size={4}>
            <span style={{ fontFamily: 'monospace' }}>
              {cur > 0 && total > 0 ? `${cur}/${total}` : '-'}
              {pct ? ` (${pct}%)` : ''}
            </span>
            <Tag color={statusColorMap[status] ?? 'default'}>{status}</Tag>
          </Space>
        );
      },
    },
    {
      key: 'upgradeType',
      title: t('software.upgrade.upgradeType'),
      dataIndex: 'taskType',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as number;
        const cfg = TASK_TYPE_MAP[val as keyof typeof TASK_TYPE_MAP] ?? { color: 'default', text: String(val) };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productClass', title: t('software.upgrade.productClass'), dataIndex: 'productClass', width: 100 },
    {
      key: 'progress',
      title: t('software.upgrade.upgradeProgress'),
      width: 160,
      render: (_: unknown, record: UpgradeTaskInfo) => {
        const val = computeProgress(record);
        const statusCode = mapTaskStatusToCode(record.status);
        // Pending (1) or suspended (3): show normal style, not animated
        // Ended (4): green success or red exception based on result
        // In progress (2): animated
        let progressStatus: 'success' | 'exception' | 'active' | 'normal' = 'normal';
        if (statusCode === 4) {
          progressStatus = (record.result === 'failed' || record.result === 'terminated') ? 'exception' : 'success';
        } else if (statusCode === 2) {
          progressStatus = 'active';
        }
        // #626：进度条下方露出绝对数（已完成/总数 · 失败 N），
        // 测试/运维一眼能看出任务升了多少台、完成多少、失败多少。
        const done = record.successCount + record.failCount;
        return (
          <div>
            <Progress percent={val} size="small" status={progressStatus} />
            <div style={{ fontSize: 11, color: '#8c8c8c', fontFamily: 'monospace', marginTop: 2 }}>
              {t('software.upgrade.progressDetail', { done, total: record.totalCount, fail: record.failCount })}
            </div>
          </div>
        );
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
    { key: 'startTime', title: t('software.startTime'), dataIndex: 'startedAt', width: 160, render: (val: unknown) => val ? formatSystemTime(val as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-' },
    { key: 'endTime', title: t('software.endTime'), dataIndex: 'endedAt', width: 160, render: (val: unknown) => val ? formatSystemTime(val as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-' },
  ], [t, TASK_STATUS_CONFIG, TASK_TYPE_MAP, TASK_RESULT_MAP, resumeMutation, suspendMutation, terminateMutation, deleteMutation, advanceCanaryMutation, pauseCanaryMutation, resumeCanaryMutation, abortCanaryMutation]);

  // ---- Device list tab columns (sub-tasks for selected main task) ----
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
    { key: 'taskName', title: t('software.taskName'), dataIndex: 'taskName', width: 150, ellipsis: true, render: (val: unknown) => (val as string) || '-' },
    { key: 'sourceVersion', title: t('software.upgrade.sourceVersion'), dataIndex: 'oriVersion', width: 100, render: (val: unknown) => (val as string) || '-' },
    { key: 'targetVersion', title: t('software.upgrade.targetVersion'), dataIndex: 'destVersion', width: 100, render: (val: unknown) => (val as string) || '-' },
    {
      key: 'upgradeType',
      title: t('software.upgrade.upgradeType'),
      width: 100,
      render: () => {
        // Sub-tasks inherit the parent task type; show from context or default
        const cfg = TASK_TYPE_MAP[1];
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productClass', title: t('software.upgrade.productClass'), width: 100, render: () => '-' },
    {
      key: 'keepConfig',
      title: t('software.upgrade.keepConfig'),
      width: 90,
      render: () => <Checkbox checked={true} disabled />,
    },
    {
      key: 'progress',
      title: t('software.upgrade.upgradeProgress'),
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
        let progressStatus: 'success' | 'exception' | 'active' | 'normal' = 'normal';
        if (record.status === 'completed') progressStatus = 'success';
        else if (record.status === 'failed') progressStatus = 'exception';
        else if (record.status === 'pending' || record.status === 'suspended') progressStatus = 'normal';
        else progressStatus = 'active';
        return <Progress percent={val} size="small" status={progressStatus} />;
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
    { key: 'failureReason', title: t('software.failureReason'), dataIndex: 'failureReason', width: 200, render: (_: unknown, record: UpgradeSubTaskInfo) => renderFailureReason(record) },
    { key: 'operator', title: t('table.operator'), dataIndex: 'taskId', width: 100, render: () => '-' },
    { key: 'operateTime', title: t('software.operateTime'), dataIndex: 'createdAt', width: 160, render: (val: unknown) => val ? formatSystemTime(val as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-' },
    { key: 'startTime', title: t('software.startTime'), dataIndex: 'startedAt', width: 160, render: (val: unknown) => val ? formatSystemTime(val as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-' },
    { key: 'endTime', title: t('software.endTime'), dataIndex: 'completedAt', width: 160, render: (val: unknown) => val ? formatSystemTime(val as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-' },
  ], [t, SUB_TASK_STATUS_MAP, TASK_TYPE_MAP]);

  // Failure reason renderer with i18n and source classification
  const DEVICE_CODES = new Set(['DOWNLOAD_FAULT', 'TC_FAULT', 'UPGRADE_5G_FAILED', 'VERSION_MISMATCH']);
  const TIMEOUT_CODES = new Set(['DOWNLOAD_TIMEOUT', 'TASK_TIMEOUT']);
  // These codes carry dynamic details (FaultCode/FaultString etc.) worth showing
  const DYNAMIC_DETAIL_CODES = new Set(['DOWNLOAD_FAULT', 'TC_FAULT', 'UPGRADE_5G_FAILED', 'VERSION_MISMATCH', 'INTERNAL_ERROR', 'FIRMWARE_NOT_FOUND']);

  function renderFailureReason(record: UpgradeSubTaskInfo) {
    const code: string | undefined = record.failureReason;
    const detail: string | undefined = record.errorMessage;
    if (!code && !detail) return '-';

    let sourceTag = '';
    if (code) {
      if (DEVICE_CODES.has(code)) {
        sourceTag = t('software.failureSource.device');
      } else if (TIMEOUT_CODES.has(code)) {
        sourceTag = t('software.failureSource.timeout');
      } else {
        sourceTag = t('software.failureSource.system');
      }
    }

    const i18nLabel = code ? t(`software.failureCode.${code}` as Parameters<typeof t>[0]) : '';

    if (i18nLabel && i18nLabel !== `software.failureCode.${code}`) {
      const showDetail = detail && detail !== code && code && DYNAMIC_DETAIL_CODES.has(code);
      return (
        <div style={{ color: '#ff4d4f', lineHeight: '20px' }}>
          {sourceTag && <Tag color="default" style={{ marginRight: 4, fontSize: 11 }}>{sourceTag}</Tag>}
          <span>{i18nLabel}</span>
          {showDetail && (
            <div style={{ fontSize: 11, color: '#999', marginTop: 2 }} title={detail}>{detail}</div>
          )}
        </div>
      );
    }

    return <span style={{ color: '#ff4d4f' }}>{detail || code}</span>;
  }

  // ---- Header buttons ----
  const headerExtra = (
    <Space>
      <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleOpenUpgradeDrawer}>
        {t('software.upgrade.upgrade')}
      </Button>
      <Button icon={<DownloadOutlined />} onClick={handleExport}>
        {t('common.export')}
      </Button>
    </Space>
  );

  return (
    <ListPageLayout title={t('nav.software.versionUpgrade')} extra={headerExtra}>
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
        filterId={`upgrade-plan-filter-${activeTab}`}
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
          variant="outlined"
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
          <DataTable<UpgradeTaskInfo>
            tableId="upgrade-plan-list-task"
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
        // Device list tab - all sub-tasks across all tasks
        <Card
          size="small"
          variant="outlined"
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
          <DataTable<UpgradeSubTaskInfo>
            tableId="upgrade-plan-list-device"
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
                <Space orientation="vertical" size="small">
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
                {t('software.upgrade.confirmRerunMsg')}
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
        title={t('software.firmware.confirmDelete')}
        open={!!deleteTaskRecord}
        onCancel={() => setDeleteTaskRecord(null)}
        onOk={handleDeleteTaskConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true, loading: deleteMutation.isPending }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                {t('software.upgrade.confirmDeleteTask')}
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>{t('software.upgrade.taskNameLabel')}</strong>{deleteTaskRecord?.taskName}<br />
                <strong>{t('software.upgrade.operatorLabel')}</strong>{deleteTaskRecord?.createUser}<br />
                <strong>{t('software.upgrade.upgradeVersionLabel')}</strong>{deleteTaskRecord?.fileName ?? '-'}
              </p>
            </div>
          }
        />
      </Modal>

      {/* Create upgrade drawer */}
      <Drawer
        title={t('software.upgrade.batchUpgrade')}
        placement="right"
        size={600}
        open={upgradeDrawerVisible}
        onClose={() => setUpgradeDrawerVisible(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setUpgradeDrawerVisible(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              onClick={handleSubmitUpgrade}
              disabled={(!selectAllOfType && drawerDevices.length === 0) || !upgradeFile || createTaskMutation.isPending}
              loading={createTaskMutation.isPending}
            >
              {t('software.upgrade.confirmUpgrade')}
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
                setUpgradeFile(undefined);
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
              {t('software.upgrade.upgradeAllOfType')}
              {drawerProductClass && (
                <Tag color="blue" style={{ marginLeft: 8 }}>{t('software.upgrade.totalDevices', { count: allDevicesCountOfType })}</Tag>
              )}
            </Checkbox>
          </Form.Item>

          {/* Upgrade category */}
          <Form.Item label={t('software.upgrade.upgradeCategory')} required>
            <Radio.Group
              value={upgradeCategory}
              onChange={(e) => {
                setUpgradeCategory(e.target.value);
                setUpgradeFile(undefined);
              }}
            >
              <Radio value="software">{t('software.upgrade.softwareUpgrade')}</Radio>
              <Radio value="patch">{t('software.upgrade.patchUpgrade')}</Radio>
              <Radio value="fpga">{t('software.upgrade.fpgaUpgrade')}</Radio>
            </Radio.Group>
          </Form.Item>

          {/* Selected upgrade devices */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
              <span>
                {t('software.upgrade.selectedDevices') ?? '已选升级设备'}
                {' '}
                <Tag color="blue">{selectAllOfType ? allDevicesCountOfType : drawerDevices.length} {t('software.upgrade.units') ?? '台'}</Tag>
              </span>
              {!selectAllOfType && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleOpenBatchInput}
                  disabled={!drawerProductClass}
                >
                  {t('software.upgrade.batchInput') ?? '批量输入'}
                </Button>
              )}
            </div>
          }>
            {selectAllOfType ? (
              <Alert
                type="info"
                showIcon
                message={`${t('software.upgrade.selectedAllOfType') ?? '已选择产品类型'}「${drawerProductClass}」${t('software.upgrade.allDevices') ?? '的全部设备'}，${t('software.upgrade.total') ?? '共'} ${allDevicesCountOfType} ${t('software.upgrade.units') ?? '台'}`}
                description={t('software.upgrade.autoQueryDesc') ?? '执行时将自动查询该产品类型的所有设备进行升级'}
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
                      { title: t('software.upgrade.sourceVersion'), dataIndex: 'sourceVersion', width: 80 },
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
                    {t('software.upgrade.addDevice') ?? '添加设备'}
                  </Button>
                </div>
              </>
            )}
          </Form.Item>

          <Divider />

          {/* Upgrade file */}
          <Form.Item label={t('software.upgrade.upgradeFile') ?? '升级文件'} required>
            <Select
              value={upgradeFile}
              onChange={setUpgradeFile}
              placeholder={t('software.upgrade.selectUpgradeFile') ?? '请选择升级文件'}
              style={{ width: '100%' }}
              options={filteredFiles}
            />
          </Form.Item>

          {/* Keep config */}
          <Form.Item>
            <Checkbox
              checked={drawerKeepConfig}
              onChange={(e) => setDrawerKeepConfig(e.target.checked)}
            >
              {t('software.upgrade.keepConfig')}
            </Checkbox>
          </Form.Item>

          <Divider />

          {/* Execution method */}
          <Form.Item label={t('software.upgrade.executionMethod') ?? '执行方式'} required>
            <Radio.Group value={executionMethod} onChange={(e) => setExecutionMethod(e.target.value)}>
              <Radio value="immediate">{t('software.upgrade.immediateExec') ?? '立即执行'}</Radio>
              <Radio value="suspend">{t('software.upgrade.suspendExec') ?? '挂起'}</Radio>
              <Radio value="scheduled">{t('software.upgrade.scheduledExec') ?? '定时执行'}</Radio>
            </Radio.Group>
          </Form.Item>

          {/* Scheduled time */}
          {executionMethod === 'scheduled' && (
            <Form.Item label={t('software.upgrade.execTime') ?? '执行时间'} required>
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm"
                value={scheduledTime}
                onChange={setScheduledTime}
                placeholder={t('software.upgrade.selectScheduleTime') ?? '请选择执行时间'}
                style={{ width: '100%' }}
                disabledDate={(current) => current && current < dayjs().startOf('day')}
              />
            </Form.Item>
          )}

          <Divider />

          {/* Task config */}
          <Form.Item label={t('software.upgrade.taskConfig') ?? '任务配置'}>
            <Space orientation="vertical" style={{ width: '100%' }}>
              <Checkbox
                checked={retryOffline}
                onChange={(e) => setRetryOffline(e.target.checked)}
              >
                {t('software.upgrade.retryOffline') ?? '离线设备等上线后重试'}
              </Checkbox>
              <Space>
                <span>{t('software.upgrade.batchExecSize') ?? '每次批量执行设备数'}：</span>
                <InputNumber
                  min={1}
                  max={100}
                  value={batchSize}
                  onChange={(val) => setBatchSize(val ?? 20)}
                  style={{ width: 80 }}
                />
                <span>{t('software.upgrade.units') ?? '台'}</span>
              </Space>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>

      {/* Task detail drawer */}
      <Drawer
        title={t('software.upgrade.taskDetail') ?? '任务详情'}
        placement="right"
        size={720}
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
                <Descriptions.Item label={t('software.upgrade.upgradeType')}>
                  <Tag color={TASK_TYPE_MAP[taskDetailRecord.taskType as keyof typeof TASK_TYPE_MAP]?.color}>
                    {TASK_TYPE_MAP[taskDetailRecord.taskType as keyof typeof TASK_TYPE_MAP]?.text || taskDetailRecord.taskType}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('software.upgrade.targetVersion')}>{taskDetailRecord.fileName ?? '-'}</Descriptions.Item>
                <Descriptions.Item label={t('software.upgrade.keepConfig')}>
                  <Checkbox checked={taskDetailRecord.isKeepConfig} disabled />
                </Descriptions.Item>
                <Descriptions.Item label={t('table.operator')}>{taskDetailRecord.createUser}</Descriptions.Item>
                <Descriptions.Item label={t('software.operateTime')}>{taskDetailRecord.createdAt ? formatSystemTime(taskDetailRecord.createdAt, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-'}</Descriptions.Item>
                <Descriptions.Item label={t('software.taskStatus')}>
                  <Tag color={TASK_STATUS_CONFIG[resultCode]?.color}>{TASK_STATUS_CONFIG[resultCode]?.text}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('software.startTime')}>{taskDetailRecord.startedAt ? formatSystemTime(taskDetailRecord.startedAt, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-'}</Descriptions.Item>
                <Descriptions.Item label={t('software.endTime')}>{taskDetailRecord.endedAt ? formatSystemTime(taskDetailRecord.endedAt, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '-'}</Descriptions.Item>
              </Descriptions>

              {/* Progress overview */}
              <Card title={t('software.upgrade.progressOverview') ?? '执行进度概览'} size="small" style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                  <div style={{ flex: '0 0 120px', textAlign: 'center' }}>
                    <Progress
                      type="circle"
                      percent={progress}
                      size={80}
                      status={resultCode === 1 || resultCode === 3 ? 'normal' : progress === 100 ? 'success' : 'active'}
                    />
                    <div style={{ marginTop: 8, color: '#666' }}>{t('software.upgrade.totalProgress') ?? '总体进度'}</div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <Space orientation="vertical" style={{ width: '100%' }}>
                      <div>
                        <Tag color="success">{t('status.success')}</Tag>
                        <span>{resultStats.completed} {t('software.upgrade.units') ?? '台'}</span>
                      </div>
                      <div>
                        <Tag color="error">{t('status.failed')}</Tag>
                        <span>{resultStats.failed} {t('software.upgrade.units') ?? '台'}</span>
                      </div>
                      <div>
                        <Tag color="processing">{t('software.status.upgrading') ?? '升级中'}</Tag>
                        <span>{resultStats.downloading} {t('software.upgrade.units') ?? '台'}</span>
                      </div>
                      <div>
                        <Tag color="warning">{t('software.status.paused')}</Tag>
                        <span>{resultStats.suspended} {t('software.upgrade.units') ?? '台'}</span>
                      </div>
                      <div>
                        <Tag color="default">{t('software.status.waiting')}</Tag>
                        <span>{resultStats.pending} {t('software.upgrade.units') ?? '台'}</span>
                      </div>
                    </Space>
                  </div>
                  <Divider orientation="vertical" style={{ height: 120 }} />
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 32, fontWeight: 'bold', color: '#1890ff' }}>{totalDevices}</div>
                    <div style={{ color: '#666' }}>{t('software.upgrade.deviceTotal')}</div>
                  </div>
                </div>
              </Card>

              {/* Device list (sub-tasks) */}
              <Card title={`${t('software.upgrade.deviceList') ?? '设备列表'} (${subTaskList.length} ${t('software.upgrade.units') ?? '台'})`} size="small">
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
                      title: t('software.upgrade.targetVersion'),
                      dataIndex: 'destVersion',
                      width: 100,
                      render: (val: unknown) => (val as string) || '-',
                    },
                    {
                      title: t('software.upgrade.sourceVersion'),
                      dataIndex: 'oriVersion',
                      width: 100,
                      render: (val: unknown) => (val as string) || '-',
                    },
                    {
                      title: t('software.upgrade.upgradeProgress'),
                      width: 120,
                      render: (_: unknown, record: UpgradeSubTaskInfo) => {
                        const statusProgress: Record<string, number> = {
                          pending: 0, downloading: 25, rebooting: 60, verifying: 85,
                          completed: 100, failed: 100, suspended: 0, terminated: 100,
                        };
                        const val = statusProgress[record.status] ?? 0;
                        let progressStatus: 'success' | 'exception' | 'active' | 'normal' = 'normal';
                        if (record.status === 'completed') progressStatus = 'success';
                        else if (record.status === 'failed') progressStatus = 'exception';
                        else if (record.status === 'pending' || record.status === 'suspended') progressStatus = 'normal';
                        else progressStatus = 'active';
                        return <Progress percent={val} size="small" status={progressStatus} />;
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
                      dataIndex: 'failureReason',
                      width: 200,
                      render: (_: unknown, record: UpgradeSubTaskInfo) => renderFailureReason(record),
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
        title={`${t('software.upgrade.addDevice') ?? '添加设备'} - ${drawerProductClass}`}
        open={addDeviceModalVisible}
        onCancel={() => setAddDeviceModalVisible(false)}
        onOk={handleConfirmAddDevices}
        okText={t('software.upgrade.confirmAdd') ?? '确认添加'}
        cancelText={t('common.cancel')}
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message={t('software.upgrade.noDevicesToAdd') ?? '没有可添加的设备'}
            description={`${t('software.upgrade.productClass') ?? '产品类型'} ${drawerProductClass} ${t('software.upgrade.allDevicesAdded') ?? '的设备已全部在列表中'}`}
          />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={`${t('software.upgrade.availableCount') ?? '共'} ${availableDevices.length} ${t('software.upgrade.devicesAvailable') ?? '台设备可选'}，${t('software.upgrade.currentDisplay') ?? '当前显示'} ${filteredAvailableDevices.length} ${t('software.upgrade.units') ?? '台'}，${t('software.upgrade.selectedCount') ?? '已选择'} ${selectedNewDevices.length} ${t('software.upgrade.units') ?? '台'}`}
              style={{ marginBottom: 16 }}
            />

            {/* Search and select all */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input.Search
                placeholder={t('software.upgrade.searchDevice') ?? '搜索基站编码或名称'}
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
                {t('common.selectAll') ?? '全选'} ({filteredAvailableDevices.length} {t('software.upgrade.units') ?? '台'})
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
                { title: t('software.upgrade.sourceVersion'), dataIndex: 'sourceVersion', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>
    </ListPageLayout>
  );
}
