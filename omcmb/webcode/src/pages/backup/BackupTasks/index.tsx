import { useState, useMemo } from 'react';
import {
  Button,
  Card,
  Tag,
  message,
  Progress,
  Space,
  Radio,
  Descriptions,
  Typography,
  Timeline,
  Modal,
  Dropdown,
  Drawer,
  Form,
  Input,
  Checkbox,
  Select,
  Switch,
  DatePicker,
  Table,
  Alert,
  Divider,
} from 'antd';
import type { MenuProps } from 'antd';
import type { Dayjs } from 'dayjs';
import {
  DownloadOutlined,
  PlayCircleOutlined,
  MoreOutlined,
  StopOutlined,
  DeleteOutlined,
  ClockCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import {
  useBackupTasks,
  useCreateBackupTask,
  useCancelBackupTask,
  useDeleteBackupTasks,
} from '@core/hooks/api/useBackup';
import type { BackupTask } from '@core/mock/data/backup';
import { useT } from '@/hooks/useT';

// 备份任务状态
type BackupStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
// 备份类型
type BackupType = 'full' | 'incremental' | 'config-only';
// 任务类型
type TaskType = 'manual' | 'scheduled';
// 任务级别状态（1=等待, 2=进行中, 3=暂停, 4=已结束, 5=终止中）
type TaskStatus = 1 | 2 | 3 | 4 | 5;
// 备份抽屉模式
type BackupDrawerMode = 'manual' | 'scheduled' | null;

// 任务级别状态配置 - use i18n keys
const TASK_STATUS_KEYS: Record<TaskStatus, { color: string; key: string }> = {
  1: { color: 'default', key: 'status.waiting' },
  2: { color: 'processing', key: 'status.inProgress' },
  3: { color: 'warning', key: 'status.paused' },
  4: { color: 'success', key: 'status.ended' },
  5: { color: 'error', key: 'status.terminating' },
};

const STATUS_MAP_KEYS: Record<string, { color: string; key: string }> = {
  pending: { color: 'default', key: 'status.pending' },
  running: { color: 'processing', key: 'status.running' },
  success: { color: 'success', key: 'status.success' },
  failed: { color: 'error', key: 'status.failed' },
  cancelled: { color: 'warning', key: 'status.cancelled' },
};

// 设备级别状态配置 - use i18n keys
const DEVICE_STATUS_KEYS: Record<string, { color: string; key: string }> = {
  pending: { color: 'default', key: 'status.pending' },
  running: { color: 'processing', key: 'status.running' },
  success: { color: 'success', key: 'status.success' },
  failed: { color: 'error', key: 'status.failed' },
  cancelled: { color: 'warning', key: 'status.cancelled' },
};

const BACKUP_TYPE_KEYS: Record<string, string> = {
  full: 'backup.fullBackup',
  incremental: 'backup.incrementalBackup',
  'config-only': 'backup.configBackup',
};

const TASK_TYPE_KEYS: Record<string, string> = {
  manual: 'status.waiting',
  scheduled: 'status.inProgress',
};

function formatBytes(bytes?: number): string {
  if (!bytes || bytes === 0) return '-';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

// 设备级别行数据
interface BackupDeviceRow extends Record<string, unknown> {
  id: string;
  taskId: string;
  taskName: string;
  deviceSn: string;
  deviceName: string;
  deviceGroup: string;
  productClass: string;
  backupType: BackupType;
  status: BackupStatus;
  progress: number;
  startTime: string;
  endTime: string;
  fileSize: number;
  failureReason: string;
}

// 任务级别行数据
interface BackupTaskRow extends Record<string, unknown> {
  id: string;
  taskName: string;
  taskType: TaskType;
  backupType: BackupType;
  deviceRange: string;
  deviceCount: number;
  status: TaskStatus;
  progress: number;
  startTime: string;
  endTime: string;
  fileSize: number;
  creator: string;
  operateTime: string;
  successCount: number;
  failedCount: number;
  runningCount: number;
  pendingCount: number;
}

// Mock 设备级别数据
const mockDeviceData: BackupDeviceRow[] = [
  { id: 'd1', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', productClass: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:15:10', fileSize: 1024 * 1024 * 64, failureReason: '' },
  { id: 'd2', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', productClass: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:18:30', fileSize: 1024 * 1024 * 58, failureReason: '' },
  { id: 'd3', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', productClass: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:25:20', fileSize: 1024 * 1024 * 72, failureReason: '' },
  { id: 'd4', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', productClass: 'AAU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:45:32', fileSize: 1024 * 1024 * 62, failureReason: '' },
  { id: 'd5', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', productClass: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-02 02:00:00', endTime: '2026-03-02 02:12:05', fileSize: 1024 * 1024 * 48, failureReason: '' },
  { id: 'd6', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', productClass: 'BBU', backupType: 'full', status: 'running', progress: 80, startTime: '2026-03-02 02:00:00', endTime: '', fileSize: 0, failureReason: '' },
  { id: 'd7', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', productClass: 'BBU', backupType: 'full', status: 'running', progress: 55, startTime: '2026-03-02 02:00:00', endTime: '', fileSize: 0, failureReason: '' },
  { id: 'd8', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', productClass: 'AAU', backupType: 'full', status: 'pending', progress: 0, startTime: '', endTime: '', fileSize: 0, failureReason: '' },
  { id: 'd9', taskId: 'bkp-003', taskName: '配置备份-5G基站', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', productClass: 'AAU', backupType: 'config-only', status: 'failed', progress: 40, startTime: '2026-03-01 15:30:00', endTime: '2026-03-01 15:35:12', fileSize: 0, failureReason: '设备连接超时' },
  { id: 'd10', taskId: 'bkp-003', taskName: '配置备份-5G基站', deviceSn: 'GNB00002', deviceName: '上海5G基站01', deviceGroup: '上海移动', productClass: 'AAU', backupType: 'config-only', status: 'failed', progress: 60, startTime: '2026-03-01 15:30:00', endTime: '2026-03-01 15:38:12', fileSize: 0, failureReason: '认证失败' },
  { id: 'd11', taskId: 'bkp-004', taskName: '增量备份-上海站点', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', productClass: 'BBU', backupType: 'incremental', status: 'success', progress: 100, startTime: '2026-03-01 04:00:00', endTime: '2026-03-01 04:12:05', fileSize: 1024 * 1024 * 32, failureReason: '' },
  { id: 'd12', taskId: 'bkp-005', taskName: '手动备份-单台设备', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', productClass: 'BBU', backupType: 'config-only', status: 'cancelled', progress: 20, startTime: '2026-02-28 09:00:00', endTime: '2026-02-28 09:05:00', fileSize: 0, failureReason: '' },
];

// Map frontend-core BackupTask (camelCase, backend-derived) → UI BackupTaskRow.
// Backend currently lacks a few UI columns (creator/startTime/endTime/runningCount/pendingCount);
// derive what we can and use empty / zero fallbacks for the rest. T-0016 / R-102.
const TASK_STATUS_TO_NUM: Record<BackupTask['status'], TaskStatus> = {
  pending: 1,
  running: 2,
  success: 4,
  failed: 4,
  cancelled: 4,
  partial: 4,
};

function mapTaskToRow(t: BackupTask): BackupTaskRow {
  const total = t.totalCount ?? 0;
  const ok = t.successCount ?? 0;
  const failed = t.failCount ?? 0;
  const remaining = Math.max(total - ok - failed, 0);
  const isRunning = t.status === 'running';
  return {
    id: t.id,
    taskName: t.taskName ?? '',
    taskType: t.taskType ?? 'manual',
    backupType: t.backupType,
    deviceRange: (t.deviceSns ?? []).join(', '),
    deviceCount: total,
    status: TASK_STATUS_TO_NUM[t.status] ?? 1,
    progress: t.progress ?? 0,
    startTime: t.createdAt ?? '',
    endTime: t.updatedAt && t.status !== 'running' && t.status !== 'pending' ? t.updatedAt : '',
    fileSize: t.fileSize ?? 0,
    creator: t.creator ?? '',
    operateTime: t.createdAt ?? '',
    successCount: ok,
    failedCount: failed,
    runningCount: isRunning ? remaining : 0,
    pendingCount: !isRunning ? remaining : 0,
  };
}

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message);
  }
  return '';
}

// Numeric TaskStatus → backend status string. 3 (paused) and 5 (terminating)
// have no backend equivalent yet — those filter selections fall through to a
// no-server-filter request and rely on client-side filtering.
const STATUS_NUM_TO_BACKEND: Record<number, string> = {
  1: 'pending',
  2: 'running',
  4: 'completed',
};

export default function BackupTasks() {
  const t = useT();
  // 页签状态
  const [activeTab, setActiveTab] = useState<'task' | 'device'>('task');
  // 任务列表状态
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  // 设备列表状态
  const [deviceFilters, setDeviceFilters] = useState<Record<string, unknown>>({});
  const [devicePage, setDevicePage] = useState(1);
  const [devicePageSize, setDevicePageSize] = useState(20);
  // 任务详情
  const [taskDetailId, setTaskDetailId] = useState<string | null>(null);
  // 删除确认
  const [deleteRecord, setDeleteRecord] = useState<BackupTaskRow | null>(null);
  // 备份抽屉状态
  const [backupDrawerMode, setBackupDrawerMode] = useState<BackupDrawerMode>(null);
  const [drawerTaskName, setDrawerTaskName] = useState('');
  const [selectAllDevices, setSelectAllDevices] = useState(false);
  const [drawerDevices, setDrawerDevices] = useState<BackupDeviceRow[]>([]);
  const [drawerExecutionMethod, setDrawerExecutionMethod] = useState<'immediate' | 'scheduled'>('immediate');
  const [drawerScheduledTime, setDrawerScheduledTime] = useState<Dayjs | null>(null);
  // 周期配置状态
  const [drawerCycleType, setDrawerCycleType] = useState<'daily' | 'weekly' | 'monthly'>('daily');
  const [drawerCycleTime, setDrawerCycleTime] = useState<Dayjs | null>(null);
  const [drawerCycleWeekDays, setDrawerCycleWeekDays] = useState<number[]>([1]);
  const [drawerCycleMonths, setDrawerCycleMonths] = useState<number[]>([1]);
  const [drawerCycleMonthDays, setDrawerCycleMonthDays] = useState<number[]>([1]);
  const [drawerCycleEnabled, setDrawerCycleEnabled] = useState(true);
  // FTP Server 配置状态
  const [ftpEnabled, setFtpEnabled] = useState(false);
  const [ftpProtocol, setFtpProtocol] = useState<'FTP' | 'SFTP'>('FTP');
  const [ftpHost, setFtpHost] = useState('');
  const [ftpPort, setFtpPort] = useState<string>('21');
  const [ftpPath, setFtpPath] = useState('');
  const [ftpUser, setFtpUser] = useState('');
  const [ftpPassword, setFtpPassword] = useState('');
  // 添加设备弹窗
  const [addDeviceVisible, setAddDeviceVisible] = useState(false);
  const [addDeviceSearch, setAddDeviceSearch] = useState('');
  const [addDeviceProductClasses, setAddDeviceProductClasses] = useState<string[]>([]);
  const [selectedNewDevices, setSelectedNewDevices] = useState<React.Key[]>([]);
  // 批量输入弹窗
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');
  const [batchInputPreview, setBatchInputPreview] = useState<{
    matched: BackupDeviceRow[];
    notFound: string[];
  }>({ matched: [], notFound: [] });

  // T-0016 / R-102: only `status` is pushed to the backend list endpoint.
  // `taskType` (manual/scheduled) and `keyword` / `timeRange` filter
  // client-side — backend backupApi maps task_type to full/incremental/
  // config_only which is a different axis, so the manual/scheduled axis can't
  // round-trip yet. Once backend exposes a "creation_kind" filter, push it here.
  const taskQueryParams = useMemo(() => {
    const q: { page: number; pageSize: number; status?: string } = { page, pageSize };
    if (typeof filters.status === 'number' && STATUS_NUM_TO_BACKEND[filters.status]) {
      q.status = STATUS_NUM_TO_BACKEND[filters.status];
    }
    return q;
  }, [page, pageSize, filters.status]);

  const { data: tasksResp, isLoading, refetch } = useBackupTasks(taskQueryParams);
  const createTask = useCreateBackupTask();
  const cancelTask = useCancelBackupTask();
  const deleteTasks = useDeleteBackupTasks();

  // T-0016 / R-102: replace inline mock with real backend data.
  // Empty list (no items / network error) renders DataTable's empty state — do
  // NOT fall back to fake rows in production.
  const realTaskData: BackupTaskRow[] = useMemo(
    () => (tasksResp?.items ?? []).map(mapTaskToRow),
    [tasksResp?.items]
  );

  // ========== 导出 ==========
  const handleExport = () => {
    const data = activeTab === 'task' ? filteredTaskData : filteredDeviceData;
    if (data.length === 0) {
      void message.warning(t('common.noDataToExport'));
      return;
    }

    let headers: string[];
    let rows: string[][];

    if (activeTab === 'task') {
      headers = [t('table.taskName'), t('table.operator'), t('table.operateTime'), t('backup.taskType'), t('backup.backupType'), t('common.status'), t('backup.taskProgress'), t('table.success'), t('table.failed'), t('status.running'), t('status.waiting'), t('table.startTime'), t('table.endTime')];
      rows = (data as BackupTaskRow[]).map((row) => [
        row.taskName,
        row.creator,
        row.operateTime,
        t(TASK_TYPE_KEYS[row.taskType] ?? ''),
        t(BACKUP_TYPE_KEYS[row.backupType] ?? ''),
        TASK_STATUS_KEYS[row.status] ? t(TASK_STATUS_KEYS[row.status].key) : String(row.status),
        `${row.progress}%`,
        String(row.successCount),
        String(row.failedCount),
        String(row.runningCount),
        String(row.pendingCount),
        row.startTime,
        row.endTime,
      ]);
    } else {
      headers = [t('device.stationCode'), t('device.stationName'), t('common.deviceGroup'), t('backup.backupType'), t('table.taskName'), t('common.status'), t('table.progress'), t('table.failureReason'), t('table.startTime'), t('table.endTime'), t('table.fileSize')];
      rows = (data as BackupDeviceRow[]).map((row) => [
        row.deviceSn,
        row.deviceName,
        row.deviceGroup,
        t(BACKUP_TYPE_KEYS[row.backupType] ?? ''),
        row.taskName,
        STATUS_MAP_KEYS[row.status]?.key ? t(STATUS_MAP_KEYS[row.status].key) : row.status,
        `${row.progress}%`,
        row.failureReason,
        row.startTime,
        row.endTime,
        formatBytes(row.fileSize),
      ]);
    }

    const csvContent = '\uFEFF' + [headers.join(','), ...rows.map((r) => r.map((c) => `"${c}"`).join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${t(activeTab === 'task' ? 'backup.csvTaskFileName' : 'backup.csvDeviceFileName')}_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);

    void message.success(t('backup.exportSuccess', { count: data.length }));
  };

  // ========== 任务操作处理 ==========
  // "Start" remains advisory: backend has no /tasks/:id/start (creation is the
  // only entry point; existing pending tasks transition automatically). T-0016
  // keeps the toast for now — runtime control deferred to a later task.
  const handleStartTask = (record: BackupTaskRow) => {
    void message.success(t('backup.startedTask', { name: record.taskName }));
  };

  const handleStopTask = (record: BackupTaskRow) => {
    // useCancelBackupTask invalidates ['backup', 'tasks'] on success; explicit
    // refetch() would double-fetch.
    cancelTask.mutate(record.id, {
      onSuccess: () => {
        void message.success(t('backup.cancelSuccess', { name: record.taskName }));
      },
      onError: (e: unknown) => {
        void message.error(t('backup.cancelFailed', { error: getErrMsg(e) }));
      },
    });
  };

  const handleDeleteTask = (record: BackupTaskRow) => {
    setDeleteRecord(record);
  };

  // ========== 备份抽屉处理 ==========
  const openBackupDrawer = (mode: 'manual' | 'scheduled') => {
    setBackupDrawerMode(mode);
    setDrawerTaskName('');
    setSelectAllDevices(false);
    setDrawerDevices([]);
    setDrawerExecutionMethod('immediate');
    setDrawerScheduledTime(null);
    setDrawerCycleType('daily');
    setDrawerCycleTime(null);
    setDrawerCycleWeekDays([1]);
    setDrawerCycleMonths([1]);
    setDrawerCycleMonthDays([1]);
    setDrawerCycleEnabled(true);
    setFtpEnabled(false);
    setFtpProtocol('FTP');
    setFtpHost('');
    setFtpPort('21');
    setFtpPath('');
    setFtpUser('');
    setFtpPassword('');
  };

  const closeBackupDrawer = () => {
    setBackupDrawerMode(null);
  };

  // 可添加设备列表
  const availableDevices = useMemo(() => {
    const existingIds = new Set(drawerDevices.map((d) => d.id));
    return mockDeviceData.filter((d) => !existingIds.has(d.id));
  }, [drawerDevices]);

  const filteredAvailableDevices = useMemo(() => {
    let result = availableDevices;
    if (addDeviceProductClasses.length > 0) {
      result = result.filter((d) => addDeviceProductClasses.includes(d.productClass));
    }
    if (addDeviceSearch.trim()) {
      const keyword = addDeviceSearch.toLowerCase();
      result = result.filter(
        (d) => d.deviceSn.toLowerCase().includes(keyword) || d.deviceName.toLowerCase().includes(keyword)
      );
    }
    return result;
  }, [availableDevices, addDeviceSearch, addDeviceProductClasses]);

  // ========== 批量输入处理 ==========
  const parseBatchInput = (input: string): string[] => {
    return input.split(/[\n,;\s]+/).map((s) => s.trim().toUpperCase()).filter((s) => s.length > 0);
  };

  const handleBatchInputPreview = () => {
    const sns = parseBatchInput(batchInputValue);
    if (sns.length === 0) {
      setBatchInputPreview({ matched: [], notFound: [] });
      return;
    }
    const matched: BackupDeviceRow[] = [];
    const notFound: string[] = [];
    const existingIds = new Set(drawerDevices.map((d) => d.id));
    for (const sn of sns) {
      const row = mockDeviceData.find((r) => r.deviceSn.toUpperCase() === sn && !existingIds.has(r.id));
      if (row) {
        matched.push(row);
      } else {
        notFound.push(sn);
      }
    }
    setBatchInputPreview({ matched, notFound });
  };

  const handleBatchInputConfirm = () => {
    const { matched } = batchInputPreview;
    if (matched.length === 0) {
      void message.warning(t('backup.noMatchedDevice'));
      return;
    }
    setDrawerDevices((prev) => [...prev, ...matched]);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [] });
    void message.success(t('backup.deviceCountUnit', { count: matched.length }));
  };

  // ========== 表单校验 ==========
  const canSubmit = useMemo(() => {
    // 任务名称必填
    if (!drawerTaskName.trim()) return false;
    // 设备必选
    if (!selectAllDevices && drawerDevices.length === 0) return false;
    // 新建备份：定时执行时时间必填
    if (backupDrawerMode === 'manual' && drawerExecutionMethod === 'scheduled' && !drawerScheduledTime) return false;
    // 周期备份：启用时周期配置必填
    if (backupDrawerMode === 'scheduled' && drawerCycleEnabled) {
      if (!drawerCycleTime) return false;
      if (drawerCycleType === 'weekly' && drawerCycleWeekDays.length === 0) return false;
      if (drawerCycleType === 'monthly') {
        if (drawerCycleMonths.length === 0) return false;
        if (drawerCycleMonthDays.length === 0) return false;
      }
    }
    // FTP 开启时必填
    if (ftpEnabled) {
      if (!ftpHost.trim() || !ftpPort.trim() || !ftpPath.trim() || !ftpUser.trim() || !ftpPassword.trim()) return false;
    }
    return true;
  }, [drawerTaskName, selectAllDevices, drawerDevices, backupDrawerMode, drawerExecutionMethod, drawerScheduledTime, drawerCycleEnabled, drawerCycleTime, drawerCycleType, drawerCycleWeekDays, drawerCycleMonths, drawerCycleMonthDays, ftpEnabled, ftpHost, ftpPort, ftpPath, ftpUser, ftpPassword]);

  const handleSubmitBackup = () => {
    if (!drawerTaskName.trim()) {
      void message.warning(t('backup.pleaseInputTaskName'));
      return;
    }
    if (!selectAllDevices && drawerDevices.length === 0) {
      void message.warning(t('backup.pleaseSelectDevices'));
      return;
    }
    // T-0016 / R-102: "全选所有设备" path is gated until a real device-list
    // hook (useDevices) replaces mockDeviceData here. POSTing mock SNs to the
    // real /backup/tasks endpoint would create a backend task with fake target_ids.
    if (selectAllDevices) {
      void message.warning(t('backup.selectAllNotSupportedYet'));
      return;
    }

    // T-0016 / R-102: real POST /backup/tasks via useCreateBackupTask.
    // Backend currently lacks scheduled / FTP fields on the create payload —
    // those Drawer states are captured but not yet sent (out of scope for M).
    const deviceSns = drawerDevices.map((d) => d.deviceSn);
    const mode = backupDrawerMode === 'scheduled' ? t('backup.cycleBackupMode') : t('backup.newBackupMode');

    createTask.mutate(
      {
        taskName: drawerTaskName,
        taskType: backupDrawerMode === 'scheduled' ? 'scheduled' : 'manual',
        deviceSns,
        backupType: 'full',
        storageLocation: '',
        creator: '',
      } as Parameters<typeof createTask.mutate>[0],
      {
        onSuccess: () => {
          const deviceInfo = t('backup.deviceCountUnit', { count: drawerDevices.length });
          void message.success(t('backup.createdBackupTask', { mode, name: drawerTaskName, deviceInfo }));
          closeBackupDrawer();
        },
        onError: (e: unknown) => {
          void message.error(t('backup.createFailed', { error: getErrMsg(e) }));
        },
      }
    );
  };

  // ========== 任务列表筛选条件 ==========
  const taskFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('table.taskName'), type: 'input', placeholder: t('backup.inputTaskName') },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('status.waiting'), value: 1 },
        { label: t('status.inProgress'), value: 2 },
        { label: t('status.paused'), value: 3 },
        { label: t('status.ended'), value: 4 },
        { label: t('status.terminating'), value: 5 },
      ],
    },
    {
      name: 'backupType',
      label: t('backup.backupType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('backup.fullBackup'), value: 'full' },
        { label: t('backup.incrementalBackup'), value: 'incremental' },
        { label: t('backup.configBackup'), value: 'config-only' },
      ],
    },
    {
      name: 'taskType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('status.waiting'), value: 'manual' },
        { label: t('status.inProgress'), value: 'scheduled' },
      ],
    },
    {
      name: 'timeRange',
      label: t('common.timeRange'),
      type: 'date-range',
      placeholder: t('common.selectTimeRange'),
    },
  ], [t]);

  // ========== 设备列表筛选条件 ==========
  const deviceFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('device.stationCodeOrName'), type: 'input', placeholder: t('device.searchStationCodeOrName') },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'running' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.cancelled'), value: 'cancelled' },
      ],
    },
    {
      name: 'backupType',
      label: t('backup.backupType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('backup.fullBackup'), value: 'full' },
        { label: t('backup.incrementalBackup'), value: 'incremental' },
        { label: t('backup.configBackup'), value: 'config-only' },
      ],
    },
    {
      name: 'taskName',
      label: t('table.taskName'),
      type: 'input',
      placeholder: t('backup.inputTaskName'),
    },
  ], [t]);

  // ========== 任务列表过滤 ==========
  const filteredTaskData = useMemo(() => {
    return realTaskData.filter((row) => {
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.taskName.toLowerCase().includes(keyword)) return false;
      }
      if (filters.status && filters.status !== 'all') {
        if (row.status !== filters.status) return false;
      }
      if (filters.backupType && filters.backupType !== 'all') {
        if (row.backupType !== filters.backupType) return false;
      }
      if (filters.taskType && filters.taskType !== 'all') {
        if (row.taskType !== filters.taskType) return false;
      }
      if (filters.timeRange && Array.isArray(filters.timeRange)) {
        const [start, end] = filters.timeRange as [import('dayjs').Dayjs, import('dayjs').Dayjs];
        if (start && end) {
          const startDate = start.toDate();
          const endDate = end.endOf('day').toDate();
          const rowStart = row.startTime ? new Date(row.startTime) : null;
          if (!rowStart) return false;
          if (rowStart < startDate || rowStart > endDate) return false;
        }
      }
      return true;
    });
  }, [filters, realTaskData]);

  // ========== 设备列表过滤 ==========
  const filteredDeviceData = useMemo(() => {
    return mockDeviceData.filter((row) => {
      if (deviceFilters.keyword && typeof deviceFilters.keyword === 'string') {
        const keyword = deviceFilters.keyword.toLowerCase();
        if (!row.deviceSn.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      if (deviceFilters.status && deviceFilters.status !== 'all') {
        if (row.status !== deviceFilters.status) return false;
      }
      if (deviceFilters.backupType && deviceFilters.backupType !== 'all') {
        if (row.backupType !== deviceFilters.backupType) return false;
      }
      if (deviceFilters.taskName && typeof deviceFilters.taskName === 'string') {
        if (!row.taskName.toLowerCase().includes(deviceFilters.taskName.toLowerCase())) return false;
      }
      return true;
    });
  }, [deviceFilters]);

  // ========== 设备列表分页 ==========
  const paginatedDeviceData = useMemo(() => {
    const start = (devicePage - 1) * devicePageSize;
    return filteredDeviceData.slice(start, start + devicePageSize);
  }, [filteredDeviceData, devicePage, devicePageSize]);

  // ========== 任务列表列定义 ==========
  const taskColumns: DataTableColumn<BackupTaskRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: BackupTaskRow) => {
        const status = record.status;
        const showStart = status === 1 || status === 3;
        const showTerminate = status === 1 || status === 2 || status === 3;
        const showDelete = status === 1 || status === 4 || status === 5;

        const items: MenuProps['items'] = [
          showStart ? {
            key: 'start',
            label: t('common.start'),
            icon: <PlayCircleOutlined />,
            onClick: () => handleStartTask(record),
          } : null,
          showTerminate ? {
            key: 'terminate',
            label: t('common.terminate'),
            icon: <StopOutlined />,
            danger: true,
            onClick: () => handleStopTask(record),
          } : null,
          (showStart || showTerminate) && showDelete ? { type: 'divider' } : null,
          showDelete ? {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => handleDeleteTask(record),
          } : null,
        ].filter(Boolean) as MenuProps['items'];

        if (!items || items.length === 0) return null;

        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => setTaskDetailId(record.id)}>{t('common.details')}</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'taskName',
      title: t('table.taskName'),
      dataIndex: 'taskName',
      width: 180,
      ellipsis: true,
      render: (val: unknown, record: BackupTaskRow) => (
        <Button type="link" size="small" onClick={() => setTaskDetailId(record.id)} style={{ padding: 0 }}>
          {(val as string) || '-'}
        </Button>
      ),
    },
    { key: 'creator', title: t('table.operator'), dataIndex: 'creator', width: 100 },
    { key: 'operateTime', title: t('table.operateTime'), dataIndex: 'operateTime', width: 160 },
    {
      key: 'taskType',
      title: t('table.type'),
      dataIndex: 'backupType',
      width: 100,
      render: (val) => <Tag color="blue">{t(BACKUP_TYPE_KEYS[String(val)] ?? '') || String(val)}</Tag>,
    },
    {
      key: 'status',
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (raw: unknown) => {
        const val = raw as TaskStatus;
        const cfg = TASK_STATUS_KEYS[val] ?? { color: 'default', key: '' };
        return <Tag color={cfg.color}>{t(cfg.key) || String(val)}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('backup.taskProgress'),
      dataIndex: 'progress',
      width: 120,
      render: (raw: unknown) => { const val = raw as number; return <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />; },
    },
    {
      key: 'result',
      title: t('backup.executionResult'),
      width: 200,
      render: (_: unknown, record: BackupTaskRow) => (
        <Space size={4}>
          {record.successCount > 0 && <Tag color="success">{t('status.success')} {record.successCount}</Tag>}
          {record.failedCount > 0 && <Tag color="error">{t('status.failed')} {record.failedCount}</Tag>}
          {record.runningCount > 0 && <Tag color="processing">{t('status.inProgress')} {record.runningCount}</Tag>}
          {record.pendingCount > 0 && <Tag color="default">{t('status.waiting')} {record.pendingCount}</Tag>}
        </Space>
      ),
    },
    { key: 'startTime', title: t('table.startTime'), dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: t('table.endTime'), dataIndex: 'endTime', width: 160 },
  ], [t]);

  // ========== 设备列表列定义 ==========
  const deviceColumns: DataTableColumn<BackupDeviceRow>[] = useMemo(() => [
    { key: 'deviceSn', title: t('device.stationCode'), dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: t('device.stationName'), dataIndex: 'deviceName', width: 150, ellipsis: true },
    { key: 'productClass', title: t('backup.productClass'), dataIndex: 'productClass', width: 100 },
    {
      key: 'backupType',
      title: t('table.type'),
      dataIndex: 'backupType',
      width: 100,
      render: (val) => <Tag color="blue">{t(BACKUP_TYPE_KEYS[String(val)] ?? '') || String(val)}</Tag>,
    },
    { key: 'taskName', title: t('table.taskName'), dataIndex: 'taskName', width: 160, ellipsis: true },
    {
      key: 'fileSize',
      title: t('backup.configFile'),
      dataIndex: 'fileSize',
      width: 180,
      render: (raw: unknown, record: BackupDeviceRow) => {
        const val = raw as number;
        return record.status === 'success' && val > 0
          ? <Button type="link" size="small" icon={<DownloadOutlined />} onClick={() => void message.success(t('backup.startDownload', { sn: record.deviceSn }))}>{record.deviceSn}_CFG.xml</Button>
          : '-';
      },
    },
    {
      key: 'status',
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const cfg = DEVICE_STATUS_KEYS[String(val)] ?? DEVICE_STATUS_KEYS.pending;
        return <Tag color={cfg.color}>{t(cfg.key)}</Tag>;
      },
    },
    {
      key: 'failureReason',
      title: t('table.failureReason'),
      dataIndex: 'failureReason',
      width: 150,
      ellipsis: true,
      render: (val: unknown) => val ? <span style={{ color: '#ff4d4f' }}>{val as string}</span> : '-',
    },
    { key: 'startTime', title: t('table.startTime'), dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: t('table.endTime'), dataIndex: 'endTime', width: 160 },
  ], [t]);

  // ========== 查找任务详情 ==========
  // T-0016 / R-102: detail Drawer reads from real task list (no separate
  // /backup/tasks/:id polling — useBackupTaskById exists but list is enough
  // for current display). Device-level detail still uses inline mock since
  // backend has no per-device sub-task model — out of scope per PRD §5.
  const taskDetail = useMemo(() => {
    if (!taskDetailId) return null;
    const task = realTaskData.find((row) => row.id === taskDetailId);
    if (!task) return null;
    const devices = mockDeviceData.filter((d) => d.taskId === taskDetailId);
    return { task, devices };
  }, [taskDetailId, realTaskData]);

  // ========== 页面头部按钮 ==========
  const headerExtra = (
    <Space>
      <Button
        type="primary"
        icon={<PlayCircleOutlined />}
        onClick={() => openBackupDrawer('manual')}
      >
        {t('backup.newBackup')}
      </Button>
      <Button
        icon={<ClockCircleOutlined />}
        onClick={() => openBackupDrawer('scheduled')}
      >
        {t('backup.scheduledBackup')}
        <Tag color={drawerCycleEnabled ? 'green' : 'default'} style={{ marginLeft: 6, marginRight: 0 }}>
          {drawerCycleEnabled ? t('common.enable') : t('common.disable')}
        </Tag>
      </Button>
      <Button
        icon={<DownloadOutlined />}
        onClick={handleExport}
      >
        {t('common.export')}
      </Button>
    </Space>
  );

  return (
    <ListPageLayout title={t('nav.backup.tasks')} extra={headerExtra}>
      {/* 页签选择 */}
      <Radio.Group
        value={activeTab}
        onChange={(e) => {
          setActiveTab(e.target.value);
          setFilters({});
          setDeviceFilters({});
          setPage(1);
          setDevicePage(1);
        }}
        optionType="button"
        buttonStyle="solid"
        style={{ marginBottom: 12 }}
      >
        <Radio.Button value="task">{t('backup.taskList')}</Radio.Button>
        <Radio.Button value="device">{t('backup.deviceList')}</Radio.Button>
      </Radio.Group>

      <FilterBar
        filterId={`backup-tasks-filter-${activeTab}`}
        fields={activeTab === 'task' ? taskFilterFields : deviceFilterFields}
        onSearch={(vals) => {
          if (activeTab === 'task') {
            setFilters(vals);
            setPage(1);
          } else {
            setDeviceFilters(vals);
            setDevicePage(1);
          }
        }}
        onReset={() => {
          if (activeTab === 'task') {
            setFilters({});
            setPage(1);
          } else {
            setDeviceFilters({});
            setDevicePage(1);
          }
        }}
      />

      {/* 列表 */}
      {activeTab === 'task' ? (
        <Card
          size="small"
          bordered
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
          <DataTable<BackupTaskRow>
            tableId="backup-tasks-list-task"
            columns={taskColumns}
            dataSource={filteredTaskData}
            loading={isLoading}
            rowKey="id"
            // T-0016 / R-102: server total drives pager; client-side keyword/
            // timeRange filters reduce the visible page but never inflate count.
            total={tasksResp?.total ?? filteredTaskData.length}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            onRefresh={() => void refetch()}
            scroll={{ x: 'max-content', y: 'calc(100vh - 510px)' }}
            showRowNumber
            rowNumberTitle={t('common.rowNumber')}
          />
        </Card>
      ) : (
        <Card
          size="small"
          bordered
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
          <DataTable<BackupDeviceRow>
            tableId="backup-tasks-list-device"
            columns={deviceColumns}
            dataSource={paginatedDeviceData}
            loading={isLoading}
            rowKey="id"
            total={filteredDeviceData.length}
            currentPage={devicePage}
            pageSize={devicePageSize}
            onPageChange={(p, s) => { setDevicePage(p); setDevicePageSize(s); }}
            onRefresh={() => void refetch()}
            scroll={{ x: 'max-content', y: 'calc(100vh - 510px)' }}
            showRowNumber
            rowNumberTitle={t('common.rowNumber')}
          />
        </Card>
      )}

      {/* 任务详情弹窗 */}
      <Modal
        title={t('backup.taskDetail')}
        open={!!taskDetailId}
        onCancel={() => setTaskDetailId(null)}
        footer={null}
        width={800}
      >
        {taskDetail && (
          <>
            <Descriptions bordered size="small" column={3} style={{ marginBottom: 16 }}>
              <Descriptions.Item label={t('table.taskName')} span={2}>{taskDetail.task.taskName}</Descriptions.Item>
              <Descriptions.Item label={t('table.operator')}>{taskDetail.task.creator}</Descriptions.Item>
              <Descriptions.Item label={t('backup.taskType')}>{t(TASK_TYPE_KEYS[taskDetail.task.taskType] ?? '')}</Descriptions.Item>
              <Descriptions.Item label={t('backup.backupType')}>{t(BACKUP_TYPE_KEYS[taskDetail.task.backupType] ?? '')}</Descriptions.Item>
              <Descriptions.Item label={t('backup.deviceRange')}>{t('backup.deviceCountUnit', { count: taskDetail.task.deviceCount })}</Descriptions.Item>
              <Descriptions.Item label={t('table.startTime')}>{taskDetail.task.startTime}</Descriptions.Item>
              <Descriptions.Item label={t('table.endTime')}>{taskDetail.task.endTime}</Descriptions.Item>
              <Descriptions.Item label={t('table.fileSize')}>{formatBytes(taskDetail.task.fileSize)}</Descriptions.Item>
            </Descriptions>

            <Typography.Text strong style={{ fontSize: 13, display: 'block', marginBottom: 12 }}>
              {t('backup.deviceExecution')}
            </Typography.Text>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                {taskDetail.task.successCount > 0 && <Tag color="success">{t('status.success')} {taskDetail.task.successCount}</Tag>}
                {taskDetail.task.failedCount > 0 && <Tag color="error">{t('status.failed')} {taskDetail.task.failedCount}</Tag>}
                {taskDetail.task.runningCount > 0 && <Tag color="processing">{t('status.inProgress')} {taskDetail.task.runningCount}</Tag>}
                {taskDetail.task.pendingCount > 0 && <Tag color="default">{t('status.waiting')} {taskDetail.task.pendingCount}</Tag>}
              </Space>
            </Space>

            <Typography.Text strong style={{ fontSize: 12, display: 'block', marginTop: 16, marginBottom: 8 }}>
              {t('backup.executionTimeline')}
            </Typography.Text>
            <Timeline
              items={[
                { color: 'blue', children: <span style={{ fontSize: 12 }}>[{taskDetail.task.startTime}] {t('backup.backupTaskStarted')}</span> },
                ...taskDetail.devices.slice(0, 3).map((d) => ({
                  color: d.status === 'success' ? 'green' : d.status === 'failed' ? 'red' : 'gray',
                  children: (
                    <span style={{ fontSize: 12 }}>
                      [{d.endTime || t('status.inProgress')}] {d.deviceSn} {d.deviceName} — {
                        d.status === 'success' ? t('backup.backupComplete', { size: formatBytes(d.fileSize) }) :
                        d.status === 'failed' ? t('backup.backupFailed', { reason: d.failureReason }) :
                        d.status === 'running' ? t('backup.backupRunning', { progress: d.progress }) :
                        d.status === 'cancelled' ? t('status.cancelled') : t('backup.backupWaiting')
                      }
                    </span>
                  ),
                })),
                taskDetail.devices.length > 3 ? {
                  color: 'gray',
                  children: <span style={{ fontSize: 12 }}>{t('backup.moreRecords', { count: taskDetail.devices.length - 3 })}</span>,
                } : null,
              ].filter(Boolean) as { color: string; children: React.ReactNode }[]}
            />
          </>
        )}
      </Modal>

      {/* 删除确认弹窗 */}
      <Modal
        title={t('backup.confirmDeleteTask')}
        open={!!deleteRecord}
        onCancel={() => setDeleteRecord(null)}
        onOk={() => {
          if (!deleteRecord) return;
          // T-0016 / R-102: real DELETE /backup/tasks/:id (batched).
          deleteTasks.mutate([deleteRecord.id], {
            onSuccess: () => {
              void message.success(t('backup.deleteSuccess'));
              setDeleteRecord(null);
            },
            onError: (e: unknown) => {
              void message.error(t('backup.deleteFailed', { error: getErrMsg(e) }));
            },
          });
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true, loading: deleteTasks.isPending }}
      >
        {deleteRecord && (
          <Typography.Text>
            {t('backup.deleteTaskConfirm', { name: deleteRecord.taskName })}
          </Typography.Text>
        )}
      </Modal>

      {/* 新建备份 / 周期备份抽屉 */}
      <Drawer
        title={backupDrawerMode === 'scheduled' ? t('backup.scheduledBackup') : t('backup.newBackup')}
        placement="right"
        width={600}
        open={!!backupDrawerMode}
        onClose={closeBackupDrawer}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={closeBackupDrawer}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              onClick={handleSubmitBackup}
              disabled={!canSubmit}
            >
              {t('common.confirm')}
            </Button>
          </Space>
        }
      >
        <Form layout="vertical" size="small">
          {/* 启用周期备份开关（仅周期备份模式，放在任务名称上面） */}
          {backupDrawerMode === 'scheduled' && (
            <Form.Item>
              <Space>
                <Switch
                  checked={drawerCycleEnabled}
                  onChange={(checked) => setDrawerCycleEnabled(checked)}
                  checkedChildren={t('common.on')}
                  unCheckedChildren={t('common.off')}
                />
                <span>{t('backup.enableScheduledBackup')}</span>
              </Space>
            </Form.Item>
          )}

          {/* 任务名称 */}
          <Form.Item label={t('table.taskName')} required>
            <Input
              value={drawerTaskName}
              onChange={(e) => setDrawerTaskName(e.target.value)}
              placeholder={t('backup.inputTaskName')}
              maxLength={100}
              showCount
            />
          </Form.Item>

          {/* 全选设备 */}
          <Form.Item>
            <Checkbox
              checked={selectAllDevices}
              onChange={(e) => setSelectAllDevices(e.target.checked)}
            >
              {t('backup.backupAllDevices')}
              <Tag color="blue" style={{ marginLeft: 8 }}>{t('backup.allDeviceCount', { count: mockDeviceData.length })}</Tag>
            </Checkbox>
          </Form.Item>

          {/* 已选备份设备 */}
          {!selectAllDevices && (
            <Form.Item label={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                <span>{t('backup.selectedBackupDevices')} <Tag color="blue">{t('backup.deviceCountUnit', { count: drawerDevices.length })}</Tag></span>
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={() => { setBatchInputValue(''); setBatchInputPreview({ matched: [], notFound: [] }); setBatchInputVisible(true); }}
                >
                  {t('backup.batchInput')}
                </Button>
              </div>
            }>
              <>
                <div style={{ maxHeight: 200, overflow: 'auto', border: '1px solid #d9d9d9', borderRadius: 6 }}>
                  <Table
                    size="small"
                    dataSource={drawerDevices}
                    rowKey="id"
                    pagination={false}
                    columns={[
                      { title: t('device.stationCode'), dataIndex: 'deviceSn', width: 100 },
                      { title: t('device.stationName'), dataIndex: 'deviceName', ellipsis: true },
                      { title: t('backup.productClass'), dataIndex: 'productClass', width: 80 },
                      {
                        title: '',
                        width: 40,
                        render: (_: unknown, record: BackupDeviceRow) => (
                          <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => setDrawerDevices((prev) => prev.filter((d) => d.id !== record.id))} />
                        ),
                      },
                    ]}
                  />
                </div>
                <div style={{ marginTop: 8 }}>
                  <Button
                    type="dashed"
                    icon={<PlusOutlined />}
                    onClick={() => { setAddDeviceSearch(''); setAddDeviceProductClasses([]); setSelectedNewDevices([]); setAddDeviceVisible(true); }}
                    style={{ width: '100%' }}
                  >
                    {t('common.addDevice')}
                  </Button>
                </div>
              </>
            </Form.Item>
          )}

          <Divider />

          {/* 执行方式（新建备份才有，周期备份固定为定时） */}
          {backupDrawerMode === 'manual' ? (
            <>
              <Form.Item label={t('backup.executionMethod')} required>
                <Radio.Group value={drawerExecutionMethod} onChange={(e) => setDrawerExecutionMethod(e.target.value)}>
                  <Radio value="immediate">{t('backup.immediateExecution')}</Radio>
                  <Radio value="scheduled">{t('backup.scheduledExecution')}</Radio>
                </Radio.Group>
              </Form.Item>

              {drawerExecutionMethod === 'scheduled' && (
                <Form.Item label={t('backup.executionTime')} required>
                  <DatePicker
                    showTime
                    format="YYYY-MM-DD HH:mm"
                    value={drawerScheduledTime}
                    onChange={setDrawerScheduledTime}
                    placeholder={t('backup.selectExecutionTime')}
                    style={{ width: '100%' }}
                    disabledDate={(current) => !!(current && current.isBefore(new Date()))}
                  />
                </Form.Item>
              )}
            </>
          ) : (
            <>
              {/* 周期配置 */}
              <Form.Item label={t('backup.executionCycle')} required>
                  <Radio.Group value={drawerCycleType} onChange={(e) => setDrawerCycleType(e.target.value)}>
                    <Radio value="daily">{t('backup.policy.everyday')}</Radio>
                    <Radio value="weekly">{t('backup.policy.weekday.sun').replace('周日', '每周')}</Radio>
                    <Radio value="monthly">{t('backup.selectMonth')}</Radio>
                  </Radio.Group>
                </Form.Item>

                {drawerCycleType === 'weekly' && (
                  <Form.Item label={t('backup.executionWeekday')} required>
                    <Checkbox.Group
                      value={drawerCycleWeekDays}
                      onChange={(vals) => setDrawerCycleWeekDays(vals as number[])}
                      options={[
                        { label: t('backup.policy.weekday.mon'), value: 1 },
                        { label: t('backup.policy.weekday.tue'), value: 2 },
                        { label: t('backup.policy.weekday.wed'), value: 3 },
                        { label: t('backup.policy.weekday.thu'), value: 4 },
                        { label: t('backup.policy.weekday.fri'), value: 5 },
                        { label: t('backup.policy.weekday.sat'), value: 6 },
                        { label: t('backup.policy.weekday.sun'), value: 7 },
                      ]}
                    />
                  </Form.Item>
                )}

                {drawerCycleType === 'monthly' && (
                  <>
                    <Form.Item label={t('backup.selectMonth')} required>
                      <Checkbox.Group
                        value={drawerCycleMonths}
                        onChange={(vals) => setDrawerCycleMonths(vals as number[])}
                        style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}
                      >
                        {Array.from({ length: 12 }, (_, i) => (
                          <Checkbox key={i + 1} value={i + 1} style={{ width: 72 }}>
                            {i + 1}
                          </Checkbox>
                        ))}
                      </Checkbox.Group>
                    </Form.Item>
                    <Form.Item label={t('backup.executionDate')} required>
                      <Checkbox.Group
                        value={drawerCycleMonthDays}
                        onChange={(vals) => setDrawerCycleMonthDays(vals as number[])}
                        style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}
                      >
                        {Array.from({ length: 31 }, (_, i) => (
                          <Checkbox key={i + 1} value={i + 1} style={{ width: 72 }}>
                            {i + 1}
                          </Checkbox>
                        ))}
                      </Checkbox.Group>
                    </Form.Item>
                  </>
                )}

                <Form.Item label={t('backup.executionTime')} required>
                  <DatePicker
                    picker="time"
                    format="HH:mm"
                    value={drawerCycleTime}
                    onChange={setDrawerCycleTime}
                    placeholder={t('backup.selectDailyExecutionTime')}
                    style={{ width: '100%' }}
                  />
                </Form.Item>
            </>
          )}

          {/* FTP Server 配置（仅周期备份） */}
          {backupDrawerMode === 'scheduled' && (
            <>
              <Divider />

              <Form.Item>
                <Checkbox
                  checked={ftpEnabled}
                  onChange={(e) => setFtpEnabled(e.target.checked)}
                >
                  {t('backup.ftpServerConfig')}
                </Checkbox>
              </Form.Item>

              {ftpEnabled && (
                <>
                  <Form.Item label={t('backup.ftpProtocol')} required>
                    <Select
                      value={ftpProtocol}
                      onChange={(val) => {
                        setFtpProtocol(val);
                        if (val === 'SFTP') setFtpPort('22');
                        else setFtpPort('21');
                      }}
                      style={{ width: '100%' }}
                      options={[
                        { label: 'FTP', value: 'FTP' },
                        { label: 'SFTP', value: 'SFTP' },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item label={t('backup.ipAddress')} required>
                    <Input
                      value={ftpHost}
                      onChange={(e) => setFtpHost(e.target.value)}
                      placeholder={t('backup.inputFtpIp')}
                    />
                  </Form.Item>
                  <Form.Item label={t('backup.port')} required>
                    <Input
                      value={ftpPort}
                      onChange={(e) => setFtpPort(e.target.value)}
                      placeholder={t('backup.inputPort')}
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                  <Form.Item label={t('backup.uploadPath')} required>
                    <Input
                      value={ftpPath}
                      onChange={(e) => setFtpPath(e.target.value)}
                      placeholder={t('backup.inputUploadPath')}
                    />
                  </Form.Item>
                  <Form.Item label={t('backup.inputFtpUser')} required>
                    <Input
                      value={ftpUser}
                      onChange={(e) => setFtpUser(e.target.value)}
                      placeholder={t('backup.inputFtpUser')}
                    />
                  </Form.Item>
                  <Form.Item label={t('backup.inputFtpPassword')} required>
                    <Input.Password
                      value={ftpPassword}
                      onChange={(e) => setFtpPassword(e.target.value)}
                      placeholder={t('backup.inputFtpPassword')}
                    />
                  </Form.Item>
                </>
              )}
            </>
          )}
        </Form>
      </Drawer>

      {/* 添加设备弹窗 */}
      <Modal
        title={t('common.addDevice')}
        open={addDeviceVisible}
        onCancel={() => setAddDeviceVisible(false)}
        onOk={() => {
          if (selectedNewDevices.length === 0) {
            void message.warning(t('backup.pleaseSelectDevices'));
            return;
          }
          const newDevices = availableDevices.filter((d) => selectedNewDevices.includes(d.id));
          setDrawerDevices((prev) => [...prev, ...newDevices]);
          setAddDeviceVisible(false);
          setSelectedNewDevices([]);
          setAddDeviceSearch('');
          void message.success(t('backup.deviceCountUnit', { count: newDevices.length }));
        }}
        okText={t('common.confirmAdd')}
        cancelText={t('common.cancel')}
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert type="info" showIcon message={t('backup.noDeviceToAdd')} />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={t('backup.availableDeviceCount', { total: availableDevices.length, selected: selectedNewDevices.length })}
              style={{ marginBottom: 16 }}
            />
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
              <Space>
                <Input.Search
                  placeholder={t('backup.searchDeviceSnOrName')}
                  value={addDeviceSearch}
                  onChange={(e) => setAddDeviceSearch(e.target.value)}
                  style={{ width: 220 }}
                  allowClear
                />
                <Select
                  mode="multiple"
                  placeholder={t('backup.productClass')}
                  value={addDeviceProductClasses}
                  onChange={setAddDeviceProductClasses}
                  options={[
                    { label: 'BBU', value: 'BBU' },
                    { label: 'RRU', value: 'RRU' },
                    { label: 'AAU', value: 'AAU' },
                  ]}
                  style={{ minWidth: 160 }}
                  allowClear
                  maxTagCount="responsive"
                />
              </Space>
              <Checkbox
                checked={selectedNewDevices.length === filteredAvailableDevices.length && filteredAvailableDevices.length > 0}
                indeterminate={selectedNewDevices.length > 0 && selectedNewDevices.length < filteredAvailableDevices.length}
                onChange={(e) => setSelectedNewDevices(e.target.checked ? filteredAvailableDevices.map((d) => d.id) : [])}
              >
                {t('backup.selectAllWithCount', { count: filteredAvailableDevices.length })}
              </Checkbox>
            </div>
            <Table
              size="small"
              dataSource={filteredAvailableDevices}
              rowKey="id"
              pagination={{ pageSize: 5, size: 'small', showSizeChanger: false }}
              scroll={{ y: 250 }}
              rowSelection={{
                selectedRowKeys: selectedNewDevices,
                onChange: (keys) => setSelectedNewDevices(keys),
              }}
              columns={[
                { title: t('device.stationCode'), dataIndex: 'deviceSn', width: 120 },
                { title: t('device.stationName'), dataIndex: 'deviceName', ellipsis: true },
                { title: t('backup.productClass'), dataIndex: 'productClass', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>

      {/* 批量输入弹窗 */}
      <Modal
        title={t('backup.batchInputDeviceSn')}
        open={batchInputVisible}
        onCancel={() => setBatchInputVisible(false)}
        onOk={handleBatchInputConfirm}
        okText={t('common.confirmAdd')}
        cancelText={t('common.cancel')}
        width={600}
        okButtonProps={{ disabled: batchInputPreview.matched.length === 0 }}
      >
        <div style={{ marginBottom: 16 }}>
          <Input.TextArea
            placeholder={t('backup.batchInputPlaceholder')}
            rows={6}
            value={batchInputValue}
            onChange={(e) => setBatchInputValue(e.target.value)}
            onBlur={handleBatchInputPreview}
          />
        </div>

        {batchInputPreview.matched.length > 0 && (
          <Alert
            type="success"
            showIcon
            message={t('backup.matchedDeviceCount', { matched: batchInputPreview.matched.length })}
            style={{ marginBottom: 8 }}
          />
        )}

        {batchInputPreview.notFound.length > 0 && (
          <Alert
            type="warning"
            showIcon
            message={
              <div>
                <div>{t('backup.notFoundSnCount', { count: batchInputPreview.notFound.length })}</div>
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
    </ListPageLayout>
  );
}
