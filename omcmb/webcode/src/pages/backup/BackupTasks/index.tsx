import { useState, useMemo } from 'react';
import {
  Button,
  Tag,
  message,
  Progress,
  Space,
  Radio,
  Card,
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
import { useBackupTasks } from '@/hooks/api/useBackup';
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

// 任务级别状态配置
const TASK_STATUS_CONFIG: Record<TaskStatus, { color: string; text: string }> = {
  1: { color: 'default', text: '等待' },
  2: { color: 'processing', text: '进行中' },
  3: { color: 'warning', text: '暂停' },
  4: { color: 'success', text: '已结束' },
  5: { color: 'error', text: '终止中' },
};

const STATUS_MAP_KEYS: Record<string, { color: string; key: string }> = {
  pending: { color: 'default', key: 'status.pending' },
  running: { color: 'processing', key: 'status.running' },
  success: { color: 'success', key: 'status.success' },
  failed: { color: 'error', key: 'status.failed' },
  cancelled: { color: 'warning', key: 'status.cancelled' },
};

// 设备级别状态配置（与任务列表风格一致）
const DEVICE_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  pending: { color: 'default', text: '等待' },
  running: { color: 'processing', text: '进行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  cancelled: { color: 'warning', text: '已取消' },
};

const BACKUP_TYPE_MAP: Record<string, string> = {
  full: '全量备份',
  incremental: '增量备份',
  'config-only': '配置备份',
};

const TASK_TYPE_MAP: Record<string, string> = {
  manual: '手动',
  scheduled: '计划',
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
  productType: string;
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
  { id: 'd1', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', productType: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:15:10', fileSize: 1024 * 1024 * 64, failureReason: '' },
  { id: 'd2', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', productType: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:18:30', fileSize: 1024 * 1024 * 58, failureReason: '' },
  { id: 'd3', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', productType: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:25:20', fileSize: 1024 * 1024 * 72, failureReason: '' },
  { id: 'd4', taskId: 'bkp-001', taskName: '全量备份-北京站点', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', productType: 'AAU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:45:32', fileSize: 1024 * 1024 * 62, failureReason: '' },
  { id: 'd5', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', productType: 'BBU', backupType: 'full', status: 'success', progress: 100, startTime: '2026-03-02 02:00:00', endTime: '2026-03-02 02:12:05', fileSize: 1024 * 1024 * 48, failureReason: '' },
  { id: 'd6', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', productType: 'BBU', backupType: 'full', status: 'running', progress: 80, startTime: '2026-03-02 02:00:00', endTime: '', fileSize: 0, failureReason: '' },
  { id: 'd7', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', productType: 'BBU', backupType: 'full', status: 'running', progress: 55, startTime: '2026-03-02 02:00:00', endTime: '', fileSize: 0, failureReason: '' },
  { id: 'd8', taskId: 'bkp-002', taskName: '计划备份-全网每日', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', productType: 'AAU', backupType: 'full', status: 'pending', progress: 0, startTime: '', endTime: '', fileSize: 0, failureReason: '' },
  { id: 'd9', taskId: 'bkp-003', taskName: '配置备份-5G基站', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', productType: 'AAU', backupType: 'config-only', status: 'failed', progress: 40, startTime: '2026-03-01 15:30:00', endTime: '2026-03-01 15:35:12', fileSize: 0, failureReason: '设备连接超时' },
  { id: 'd10', taskId: 'bkp-003', taskName: '配置备份-5G基站', deviceSn: 'GNB00002', deviceName: '上海5G基站01', deviceGroup: '上海移动', productType: 'AAU', backupType: 'config-only', status: 'failed', progress: 60, startTime: '2026-03-01 15:30:00', endTime: '2026-03-01 15:38:12', fileSize: 0, failureReason: '认证失败' },
  { id: 'd11', taskId: 'bkp-004', taskName: '增量备份-上海站点', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', productType: 'BBU', backupType: 'incremental', status: 'success', progress: 100, startTime: '2026-03-01 04:00:00', endTime: '2026-03-01 04:12:05', fileSize: 1024 * 1024 * 32, failureReason: '' },
  { id: 'd12', taskId: 'bkp-005', taskName: '手动备份-单台设备', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', productType: 'BBU', backupType: 'config-only', status: 'cancelled', progress: 20, startTime: '2026-02-28 09:00:00', endTime: '2026-02-28 09:05:00', fileSize: 0, failureReason: '' },
];

// Mock 任务级别数据
const mockTaskData: BackupTaskRow[] = [
  { id: 'bkp-001', taskName: '全量备份-北京站点', taskType: 'manual', backupType: 'full', deviceRange: 'ENB00001, ENB00002, ENB00003, GNB00001', deviceCount: 4, status: 4, progress: 100, startTime: '2026-03-01 02:00:00', endTime: '2026-03-01 02:45:32', fileSize: 1024 * 1024 * 256, creator: 'admin', operateTime: '2026-03-01 01:55:00', successCount: 4, failedCount: 0, runningCount: 0, pendingCount: 0 },
  { id: 'bkp-002', taskName: '计划备份-全网每日', taskType: 'scheduled', backupType: 'full', deviceRange: '全部设备 (18台)', deviceCount: 18, status: 2, progress: 65, startTime: '2026-03-02 02:00:00', endTime: '-', fileSize: 0, creator: 'system', operateTime: '2026-03-02 01:58:00', successCount: 1, failedCount: 0, runningCount: 2, pendingCount: 15 },
  { id: 'bkp-003', taskName: '配置备份-5G基站', taskType: 'manual', backupType: 'config-only', deviceRange: 'GNB00001, GNB00002', deviceCount: 2, status: 4, progress: 100, startTime: '2026-03-01 15:30:00', endTime: '2026-03-01 15:38:12', fileSize: 0, creator: 'operator1', operateTime: '2026-03-01 15:25:00', successCount: 0, failedCount: 2, runningCount: 0, pendingCount: 0 },
  { id: 'bkp-004', taskName: '增量备份-上海站点', taskType: 'scheduled', backupType: 'incremental', deviceRange: 'ENB00003', deviceCount: 1, status: 4, progress: 100, startTime: '2026-03-01 04:00:00', endTime: '2026-03-01 04:12:05', fileSize: 1024 * 1024 * 32, creator: 'system', operateTime: '2026-03-01 03:55:00', successCount: 1, failedCount: 0, runningCount: 0, pendingCount: 0 },
  { id: 'bkp-005', taskName: '手动备份-单台设备', taskType: 'manual', backupType: 'config-only', deviceRange: 'ENB00001', deviceCount: 1, status: 4, progress: 100, startTime: '2026-02-28 09:00:00', endTime: '2026-02-28 09:05:00', fileSize: 0, creator: 'operator2', operateTime: '2026-02-28 08:55:00', successCount: 0, failedCount: 0, runningCount: 0, pendingCount: 0 },
];

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
  const [addDeviceProductTypes, setAddDeviceProductTypes] = useState<string[]>([]);
  const [selectedNewDevices, setSelectedNewDevices] = useState<React.Key[]>([]);
  // 批量输入弹窗
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');
  const [batchInputPreview, setBatchInputPreview] = useState<{
    matched: BackupDeviceRow[];
    notFound: string[];
  }>({ matched: [], notFound: [] });

  const { data, isLoading, refetch } = useBackupTasks({ page, pageSize });

  void data;

  // ========== 导出 ==========
  const handleExport = () => {
    const data = activeTab === 'task' ? filteredTaskData : filteredDeviceData;
    if (data.length === 0) {
      void message.warning('没有可导出的数据');
      return;
    }

    let headers: string[];
    let rows: string[][];

    if (activeTab === 'task') {
      headers = ['任务名称', '操作人', '操作时间', '任务类型', '备份类型', '状态', '任务进度', '成功', '失败', '进行中', '等待', '开始时间', '结束时间'];
      rows = (data as BackupTaskRow[]).map((row) => [
        row.taskName,
        row.creator,
        row.operateTime,
        TASK_TYPE_MAP[row.taskType] ?? row.taskType,
        BACKUP_TYPE_MAP[row.backupType] ?? row.backupType,
        TASK_STATUS_CONFIG[row.status]?.text ?? String(row.status),
        `${row.progress}%`,
        String(row.successCount),
        String(row.failedCount),
        String(row.runningCount),
        String(row.pendingCount),
        row.startTime,
        row.endTime,
      ]);
    } else {
      headers = ['基站编码', '基站名称', '设备组', '备份类型', '任务名称', '状态', '进度', '失败原因', '开始时间', '结束时间', '文件大小'];
      rows = (data as BackupDeviceRow[]).map((row) => [
        row.deviceSn,
        row.deviceName,
        row.deviceGroup,
        BACKUP_TYPE_MAP[row.backupType] ?? row.backupType,
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
    link.download = `备份${activeTab === 'task' ? '任务' : '设备'}列表_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);

    void message.success(`已导出 ${data.length} 条记录`);
  };

  // ========== 任务操作处理 ==========
  const handleStartTask = (record: BackupTaskRow) => {
    void message.success(`已开始任务: ${record.taskName}`);
  };

  const handleStopTask = (record: BackupTaskRow) => {
    void message.success(`已终止任务: ${record.taskName}`);
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
    if (addDeviceProductTypes.length > 0) {
      result = result.filter((d) => addDeviceProductTypes.includes(d.productType));
    }
    if (addDeviceSearch.trim()) {
      const keyword = addDeviceSearch.toLowerCase();
      result = result.filter(
        (d) => d.deviceSn.toLowerCase().includes(keyword) || d.deviceName.toLowerCase().includes(keyword)
      );
    }
    return result;
  }, [availableDevices, addDeviceSearch, addDeviceProductTypes]);

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
      void message.warning('没有匹配到任何设备');
      return;
    }
    setDrawerDevices((prev) => [...prev, ...matched]);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [] });
    void message.success(`已添加 ${matched.length} 台设备`);
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
      void message.warning('请输入任务名称');
      return;
    }
    if (!selectAllDevices && drawerDevices.length === 0) {
      void message.warning('请选择要备份的设备');
      return;
    }

    const mode = backupDrawerMode === 'scheduled' ? '周期备份' : '新建备份';
    const deviceInfo = selectAllDevices ? `全部 ${mockDeviceData.length} 台设备` : `${drawerDevices.length} 台设备`;
    void message.success(`已创建${mode}任务「${drawerTaskName}」：${deviceInfo}`);
    closeBackupDrawer();
  };

  // ========== 任务列表筛选条件 ==========
  const taskFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '任务名称', type: 'input', placeholder: '请输入任务名称' },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: '全部', value: 'all' },
        { label: '等待', value: 1 },
        { label: '进行中', value: 2 },
        { label: '暂停', value: 3 },
        { label: '已结束', value: 4 },
        { label: '终止中', value: 5 },
      ],
    },
    {
      name: 'backupType',
      label: '备份类型',
      type: 'select',
      options: [
        { label: '全部', value: 'all' },
        { label: '全量备份', value: 'full' },
        { label: '增量备份', value: 'incremental' },
        { label: '配置备份', value: 'config-only' },
      ],
    },
    {
      name: 'taskType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '全部', value: 'all' },
        { label: '手动', value: 'manual' },
        { label: '计划', value: 'scheduled' },
      ],
    },
    {
      name: 'timeRange',
      label: '时间范围',
      type: 'date-range',
      placeholder: '请选择时间范围',
    },
  ], [t]);

  // ========== 设备列表筛选条件 ==========
  const deviceFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '基站编码/名称', type: 'input', placeholder: '请输入基站编码或名称' },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: '全部', value: 'all' },
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.running'), value: 'running' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('status.cancelled'), value: 'cancelled' },
      ],
    },
    {
      name: 'backupType',
      label: '备份类型',
      type: 'select',
      options: [
        { label: '全部', value: 'all' },
        { label: '全量备份', value: 'full' },
        { label: '增量备份', value: 'incremental' },
        { label: '配置备份', value: 'config-only' },
      ],
    },
    {
      name: 'taskName',
      label: '任务名称',
      type: 'input',
      placeholder: '请输入任务名称',
    },
  ], [t]);

  // ========== 任务列表过滤 ==========
  const filteredTaskData = useMemo(() => {
    return mockTaskData.filter((row) => {
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
  }, [filters]);

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
      title: '操作',
      width: 80,
      align: 'center',
      fixed: 'left',
      render: (_: unknown, record: BackupTaskRow) => {
        const status = record.status;
        // 等待(1): 开始、终止、删除
        // 进行中(2): 终止
        // 暂停(3): 开始、终止
        // 已结束(4): 删除
        // 终止中(5): 删除
        const showStart = status === 1 || status === 3;
        const showTerminate = status === 1 || status === 2 || status === 3;
        const showDelete = status === 1 || status === 4 || status === 5;

        const items: MenuProps['items'] = [
          showStart ? {
            key: 'start',
            label: '开始',
            icon: <PlayCircleOutlined />,
            onClick: () => handleStartTask(record),
          } : null,
          showTerminate ? {
            key: 'terminate',
            label: '终止',
            icon: <StopOutlined />,
            danger: true,
            onClick: () => handleStopTask(record),
          } : null,
          (showStart || showTerminate) && showDelete ? { type: 'divider' } : null,
          showDelete ? {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => handleDeleteTask(record),
          } : null,
        ].filter(Boolean) as MenuProps['items'];

        if (!items || items.length === 0) return null;

        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
        );
      },
    },
    {
      key: 'taskName',
      title: '任务名称',
      dataIndex: 'taskName',
      width: 180,
      ellipsis: true,
      render: (val: string, record: BackupTaskRow) => (
        <Button type="link" size="small" onClick={() => setTaskDetailId(record.id)} style={{ padding: 0 }}>
          {val || '-'}
        </Button>
      ),
    },
    { key: 'creator', title: '操作人', dataIndex: 'creator', width: 100 },
    { key: 'operateTime', title: '操作时间', dataIndex: 'operateTime', width: 160 },
    {
      key: 'taskType',
      title: '类型',
      dataIndex: 'backupType',
      width: 100,
      render: (val) => <Tag color="blue">{BACKUP_TYPE_MAP[String(val)] ?? String(val)}</Tag>,
    },
    {
      key: 'status',
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (val: TaskStatus) => {
        const cfg = TASK_STATUS_CONFIG[val] ?? { color: 'default', text: String(val) };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'progress',
      title: '任务进度',
      dataIndex: 'progress',
      width: 120,
      render: (val: number) => <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />,
    },
    {
      key: 'result',
      title: '执行结果',
      width: 200,
      render: (_: unknown, record: BackupTaskRow) => (
        <Space size={4}>
          {record.successCount > 0 && <Tag color="success">成功 {record.successCount}</Tag>}
          {record.failedCount > 0 && <Tag color="error">失败 {record.failedCount}</Tag>}
          {record.runningCount > 0 && <Tag color="processing">进行中 {record.runningCount}</Tag>}
          {record.pendingCount > 0 && <Tag color="default">等待 {record.pendingCount}</Tag>}
        </Space>
      ),
    },
    { key: 'startTime', title: '开始时间', dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: '结束时间', dataIndex: 'endTime', width: 160 },
  ], [t]);

  // ========== 设备列表列定义 ==========
  const deviceColumns: DataTableColumn<BackupDeviceRow>[] = useMemo(() => [
    { key: 'deviceSn', title: '基站编码', dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: '基站名称', dataIndex: 'deviceName', width: 150, ellipsis: true },
    { key: 'productType', title: '产品类型', dataIndex: 'productType', width: 100 },
    {
      key: 'backupType',
      title: '类型',
      dataIndex: 'backupType',
      width: 100,
      render: (val) => <Tag color="blue">{BACKUP_TYPE_MAP[String(val)] ?? String(val)}</Tag>,
    },
    { key: 'taskName', title: '任务名称', dataIndex: 'taskName', width: 160, ellipsis: true },
    {
      key: 'fileSize',
      title: '配置文件',
      dataIndex: 'fileSize',
      width: 180,
      render: (val: number, record: BackupDeviceRow) =>
        record.status === 'success' && val > 0
          ? <Button type="link" size="small" icon={<DownloadOutlined />} onClick={() => void message.success(`开始下载: ${record.deviceSn}`)}>{record.deviceSn}_CFG.xml</Button>
          : '-',
    },
    {
      key: 'status',
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const cfg = DEVICE_STATUS_CONFIG[String(val)] ?? DEVICE_STATUS_CONFIG.pending;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'failureReason',
      title: '失败原因',
      dataIndex: 'failureReason',
      width: 150,
      ellipsis: true,
      render: (val: string) => val ? <span style={{ color: '#ff4d4f' }}>{val}</span> : '-',
    },
    { key: 'startTime', title: '开始时间', dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: '结束时间', dataIndex: 'endTime', width: 160 },
  ], [t]);

  // ========== 查找任务详情 ==========
  const taskDetail = useMemo(() => {
    if (!taskDetailId) return null;
    const task = mockTaskData.find((t) => t.id === taskDetailId);
    if (!task) return null;
    const devices = mockDeviceData.filter((d) => d.taskId === taskDetailId);
    return { task, devices };
  }, [taskDetailId]);

  // ========== 页面头部按钮 ==========
  const headerExtra = (
    <Space>
      <Button
        type="primary"
        icon={<PlayCircleOutlined />}
        onClick={() => openBackupDrawer('manual')}
      >
        新建备份
      </Button>
      <Button
        icon={<ClockCircleOutlined />}
        onClick={() => openBackupDrawer('scheduled')}
      >
        周期备份
        <Tag color={drawerCycleEnabled ? 'green' : 'default'} style={{ marginLeft: 6, marginRight: 0 }}>
          {drawerCycleEnabled ? '已启用' : '未启用'}
        </Tag>
      </Button>
      <Button
        icon={<DownloadOutlined />}
        onClick={handleExport}
      >
        导出
      </Button>
    </Space>
  );

  return (
    <ListPageLayout title={t('nav.backup.tasks')} extra={headerExtra}>
      {/* 页签选择 */}
      <Card bordered={false} style={{ marginBottom: 16 }}>
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
        >
          <Radio.Button value="task">任务列表</Radio.Button>
          <Radio.Button value="device">设备列表</Radio.Button>
        </Radio.Group>
      </Card>

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
      <Card bordered={false} className="backup-tasks-list-wrapper">
        {activeTab === 'task' ? (
          <DataTable<BackupTaskRow>
            tableId="backup-tasks-list-task"
            columns={taskColumns}
            dataSource={filteredTaskData}
            loading={isLoading}
            rowKey="id"
            total={filteredTaskData.length}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            onRefresh={() => void refetch()}
            scroll={{ x: 'max-content', y: 'calc(100vh - 510px)' }}
            showRowNumber
            rowNumberTitle="序号"
          />
        ) : (
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
            rowNumberTitle="序号"
          />
        )}
      </Card>

      {/* 任务详情弹窗 */}
      <Modal
        title="任务详情"
        open={!!taskDetailId}
        onCancel={() => setTaskDetailId(null)}
        footer={null}
        width={800}
      >
        {taskDetail && (
          <>
            <Descriptions bordered size="small" column={3} style={{ marginBottom: 16 }}>
              <Descriptions.Item label="任务名称" span={2}>{taskDetail.task.taskName}</Descriptions.Item>
              <Descriptions.Item label="操作人">{taskDetail.task.creator}</Descriptions.Item>
              <Descriptions.Item label="任务类型">{TASK_TYPE_MAP[taskDetail.task.taskType]}</Descriptions.Item>
              <Descriptions.Item label="备份类型">{BACKUP_TYPE_MAP[taskDetail.task.backupType]}</Descriptions.Item>
              <Descriptions.Item label="设备范围">{taskDetail.task.deviceCount} 台</Descriptions.Item>
              <Descriptions.Item label="开始时间">{taskDetail.task.startTime}</Descriptions.Item>
              <Descriptions.Item label="结束时间">{taskDetail.task.endTime}</Descriptions.Item>
              <Descriptions.Item label="文件大小">{formatBytes(taskDetail.task.fileSize)}</Descriptions.Item>
            </Descriptions>

            <Typography.Text strong style={{ fontSize: 13, display: 'block', marginBottom: 12 }}>
              设备执行情况
            </Typography.Text>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                {taskDetail.task.successCount > 0 && <Tag color="success">成功 {taskDetail.task.successCount}</Tag>}
                {taskDetail.task.failedCount > 0 && <Tag color="error">失败 {taskDetail.task.failedCount}</Tag>}
                {taskDetail.task.runningCount > 0 && <Tag color="processing">进行中 {taskDetail.task.runningCount}</Tag>}
                {taskDetail.task.pendingCount > 0 && <Tag color="default">等待 {taskDetail.task.pendingCount}</Tag>}
              </Space>
            </Space>

            <Typography.Text strong style={{ fontSize: 12, display: 'block', marginTop: 16, marginBottom: 8 }}>
              执行时间线
            </Typography.Text>
            <Timeline
              items={[
                { color: 'blue', children: <span style={{ fontSize: 12 }}>[{taskDetail.task.startTime}] 备份任务开始执行</span> },
                ...taskDetail.devices.slice(0, 3).map((d) => ({
                  color: d.status === 'success' ? 'green' : d.status === 'failed' ? 'red' : 'gray',
                  children: (
                    <span style={{ fontSize: 12 }}>
                      [{d.endTime || '进行中'}] {d.deviceSn} {d.deviceName} — {
                        d.status === 'success' ? `备份完成 ${formatBytes(d.fileSize)}` :
                        d.status === 'failed' ? `备份失败: ${d.failureReason}` :
                        d.status === 'running' ? `备份中 ${d.progress}%` :
                        d.status === 'cancelled' ? '已取消' : '等待中'
                      }
                    </span>
                  ),
                })),
                taskDetail.devices.length > 3 ? {
                  color: 'gray',
                  children: <span style={{ fontSize: 12 }}>...还有 {taskDetail.devices.length - 3} 条记录</span>,
                } : null,
              ].filter(Boolean) as { color: string; children: React.ReactNode }[]}
            />
          </>
        )}
      </Modal>

      {/* 删除确认弹窗 */}
      <Modal
        title="确认删除"
        open={!!deleteRecord}
        onCancel={() => setDeleteRecord(null)}
        onOk={() => {
          if (deleteRecord) {
            void message.success(t('common.deleteSuccess'));
            setDeleteRecord(null);
          }
        }}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        {deleteRecord && (
          <Typography.Text>
            确定要删除任务「<strong>{deleteRecord.taskName}</strong>」吗？此操作不可恢复。
          </Typography.Text>
        )}
      </Modal>

      {/* 新建备份 / 周期备份抽屉 */}
      <Drawer
        title={backupDrawerMode === 'scheduled' ? '周期备份' : '新建备份'}
        placement="right"
        width={600}
        open={!!backupDrawerMode}
        onClose={closeBackupDrawer}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={closeBackupDrawer}>取消</Button>
            <Button
              type="primary"
              onClick={handleSubmitBackup}
              disabled={!canSubmit}
            >
              确认
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
                  checkedChildren="开"
                  unCheckedChildren="关"
                />
                <span>启用周期备份</span>
              </Space>
            </Form.Item>
          )}

          {/* 任务名称 */}
          <Form.Item label="任务名称" required>
            <Input
              value={drawerTaskName}
              onChange={(e) => setDrawerTaskName(e.target.value)}
              placeholder="请输入任务名称"
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
              备份全部设备
              <Tag color="blue" style={{ marginLeft: 8 }}>共 {mockDeviceData.length} 台</Tag>
            </Checkbox>
          </Form.Item>

          {/* 已选备份设备 */}
          {!selectAllDevices && (
            <Form.Item label={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                <span>已选备份设备 <Tag color="blue">{drawerDevices.length} 台</Tag></span>
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={() => { setBatchInputValue(''); setBatchInputPreview({ matched: [], notFound: [] }); setBatchInputVisible(true); }}
                >
                  批量输入
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
                      { title: '基站编码', dataIndex: 'deviceSn', width: 100 },
                      { title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
                      { title: '产品类型', dataIndex: 'productType', width: 80 },
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
                    onClick={() => { setAddDeviceSearch(''); setAddDeviceProductTypes([]); setSelectedNewDevices([]); setAddDeviceVisible(true); }}
                    style={{ width: '100%' }}
                  >
                    添加设备
                  </Button>
                </div>
              </>
            </Form.Item>
          )}

          <Divider />

          {/* 执行方式（新建备份才有，周期备份固定为定时） */}
          {backupDrawerMode === 'manual' ? (
            <>
              <Form.Item label="执行方式" required>
                <Radio.Group value={drawerExecutionMethod} onChange={(e) => setDrawerExecutionMethod(e.target.value)}>
                  <Radio value="immediate">立即执行</Radio>
                  <Radio value="scheduled">定时执行</Radio>
                </Radio.Group>
              </Form.Item>

              {drawerExecutionMethod === 'scheduled' && (
                <Form.Item label="执行时间" required>
                  <DatePicker
                    showTime
                    format="YYYY-MM-DD HH:mm"
                    value={drawerScheduledTime}
                    onChange={setDrawerScheduledTime}
                    placeholder="请选择执行时间"
                    style={{ width: '100%' }}
                    disabledDate={(current) => !!(current && current.isBefore(new Date()))}
                  />
                </Form.Item>
              )}
            </>
          ) : (
            <>
              {/* 周期配置 */}
              <Form.Item label="执行周期" required>
                  <Radio.Group value={drawerCycleType} onChange={(e) => setDrawerCycleType(e.target.value)}>
                    <Radio value="daily">每天</Radio>
                    <Radio value="weekly">每周</Radio>
                    <Radio value="monthly">每月</Radio>
                  </Radio.Group>
                </Form.Item>

                {drawerCycleType === 'weekly' && (
                  <Form.Item label="执行星期" required>
                    <Checkbox.Group
                      value={drawerCycleWeekDays}
                      onChange={(vals) => setDrawerCycleWeekDays(vals as number[])}
                      options={[
                        { label: '周一', value: 1 },
                        { label: '周二', value: 2 },
                        { label: '周三', value: 3 },
                        { label: '周四', value: 4 },
                        { label: '周五', value: 5 },
                        { label: '周六', value: 6 },
                        { label: '周日', value: 7 },
                      ]}
                    />
                  </Form.Item>
                )}

                {drawerCycleType === 'monthly' && (
                  <>
                    <Form.Item label="选择月份" required>
                      <Checkbox.Group
                        value={drawerCycleMonths}
                        onChange={(vals) => setDrawerCycleMonths(vals as number[])}
                        style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}
                      >
                        {Array.from({ length: 12 }, (_, i) => (
                          <Checkbox key={i + 1} value={i + 1} style={{ width: 72 }}>
                            {i + 1}月
                          </Checkbox>
                        ))}
                      </Checkbox.Group>
                    </Form.Item>
                    <Form.Item label="执行日期" required>
                      <Checkbox.Group
                        value={drawerCycleMonthDays}
                        onChange={(vals) => setDrawerCycleMonthDays(vals as number[])}
                        style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}
                      >
                        {Array.from({ length: 31 }, (_, i) => (
                          <Checkbox key={i + 1} value={i + 1} style={{ width: 72 }}>
                            {i + 1}日
                          </Checkbox>
                        ))}
                      </Checkbox.Group>
                    </Form.Item>
                  </>
                )}

                <Form.Item label="执行时间" required>
                  <DatePicker
                    picker="time"
                    format="HH:mm"
                    value={drawerCycleTime}
                    onChange={setDrawerCycleTime}
                    placeholder="请选择每日执行时间"
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
                  FTP Server 配置
                </Checkbox>
              </Form.Item>

              {ftpEnabled && (
                <>
                  <Form.Item label="FTP协议" required>
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
                  <Form.Item label="IP地址" required>
                    <Input
                      value={ftpHost}
                      onChange={(e) => setFtpHost(e.target.value)}
                      placeholder="请输入FTP服务器IP地址"
                    />
                  </Form.Item>
                  <Form.Item label="端口" required>
                    <Input
                      value={ftpPort}
                      onChange={(e) => setFtpPort(e.target.value)}
                      placeholder="请输入端口号"
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                  <Form.Item label="上传路径" required>
                    <Input
                      value={ftpPath}
                      onChange={(e) => setFtpPath(e.target.value)}
                      placeholder="请输入上传路径，如 /backup/"
                    />
                  </Form.Item>
                  <Form.Item label="用户名" required>
                    <Input
                      value={ftpUser}
                      onChange={(e) => setFtpUser(e.target.value)}
                      placeholder="请输入FTP用户名"
                    />
                  </Form.Item>
                  <Form.Item label="密码" required>
                    <Input.Password
                      value={ftpPassword}
                      onChange={(e) => setFtpPassword(e.target.value)}
                      placeholder="请输入FTP密码"
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
        title="添加设备"
        open={addDeviceVisible}
        onCancel={() => setAddDeviceVisible(false)}
        onOk={() => {
          if (selectedNewDevices.length === 0) {
            void message.warning('请选择要添加的设备');
            return;
          }
          const newDevices = availableDevices.filter((d) => selectedNewDevices.includes(d.id));
          setDrawerDevices((prev) => [...prev, ...newDevices]);
          setAddDeviceVisible(false);
          setSelectedNewDevices([]);
          setAddDeviceSearch('');
          void message.success(`已添加 ${newDevices.length} 台设备`);
        }}
        okText="确认添加"
        cancelText="取消"
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert type="info" showIcon message="没有可添加的设备" />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={`共 ${availableDevices.length} 台设备可选，已选择 ${selectedNewDevices.length} 台`}
              style={{ marginBottom: 16 }}
            />
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
              <Space>
                <Input.Search
                  placeholder="搜索基站编码或名称"
                  value={addDeviceSearch}
                  onChange={(e) => setAddDeviceSearch(e.target.value)}
                  style={{ width: 220 }}
                  allowClear
                />
                <Select
                  mode="multiple"
                  placeholder="产品类型"
                  value={addDeviceProductTypes}
                  onChange={setAddDeviceProductTypes}
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
                全选 ({filteredAvailableDevices.length} 台)
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
                { title: '基站编码', dataIndex: 'deviceSn', width: 120 },
                { title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
                { title: '产品类型', dataIndex: 'productType', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>

      {/* 批量输入弹窗 */}
      <Modal
        title="批量输入设备SN"
        open={batchInputVisible}
        onCancel={() => setBatchInputVisible(false)}
        onOk={handleBatchInputConfirm}
        okText="确认添加"
        cancelText="取消"
        width={600}
        okButtonProps={{ disabled: batchInputPreview.matched.length === 0 }}
      >
        <div style={{ marginBottom: 16 }}>
          <Input.TextArea
            placeholder="请输入设备SN，支持换行、逗号、分号、空格分隔&#10;例如：&#10;ENB00001&#10;ENB00002, ENB00003; GNB00001"
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
            message={`匹配到 ${batchInputPreview.matched.length} 个设备，可添加 ${batchInputPreview.matched.length} 台`}
            style={{ marginBottom: 8 }}
          />
        )}

        {batchInputPreview.notFound.length > 0 && (
          <Alert
            type="warning"
            showIcon
            message={
              <div>
                <div>以下 {batchInputPreview.notFound.length} 个设备SN未找到或已在列表中:</div>
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
