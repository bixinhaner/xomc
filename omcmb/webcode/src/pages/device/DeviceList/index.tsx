import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { App, Badge, Button, Card, Checkbox, Drawer, Form, Input, InputNumber, Modal, Popover, Progress, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AlertOutlined,
  ClockCircleOutlined,
  CheckOutlined,
  CloseOutlined,
  EditOutlined,
  ExportOutlined,
  FileTextOutlined,
  LinkOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import StatisticsPanel from '@/components/StatisticsPanel';
import StatusIndicator from '@/components/StatusIndicator';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import AutoRefreshDropdown from '@/pages/alarm/components/AutoRefreshDropdown';
import { prefetchDeviceDetailContext, useDeviceList, useBatchRebootDevices, useDeviceGroups, useUpdateDevice, useSyncDeviceParams } from '@core/hooks/api/useDevices';
import { useProductList } from '@core/hooks/api/useProducts';
import { useBatchUpdateSysConfigs, useDictionaryBatch, useSysConfigsByCategory } from '@core/hooks/api/useSystem';
import { resolveNetworkTypeLabel } from '@core/utils/networkType';
import { activationStatusLabelOf, displayActivationStatusLabelOf, displayActivationStatusOf } from '@core/utils/activationStatus';
import { formatDeviceSyncStatus, getDeviceSyncStatusKind, normalizeDeviceSyncStatus } from '@core/utils/deviceSyncStatus';
import { expandSelectedGroupIds } from '@core/utils/deviceGroupFilter';
import { withDeviceGroupDisplayName } from '@core/utils/deviceGroupDisplay';
import { getI18nText } from '@core/utils/i18nText';
import { containsHan, localizeDeviceProductName } from '@core/utils/deviceDisplay';
import { hasAlarmSeverity } from '@core/utils/alarmSeverity';
import { useAlarmCountWithDeviceListInvalidation } from '@core/hooks/api/useAlarms';
import { useTriggerAlarmSync } from '@core/hooks/api/useAlarms';
import { deviceTaskApi, isAbortError } from '@core/services/api/deviceTaskApi';
import { useCreateUnifiedFileTransferTask } from '@core/hooks/api/useUnifiedFileTransfer';
import { deviceApi } from '@core/services/api/deviceApi';
import { createApiSwitch } from '@core/services/apiSwitch';
import { deviceService } from '@core/mock/services/deviceService';
import { mapWithConcurrencyLimit } from '@core/utils/asyncPool';
import {
  resolveVisibleExportColumns,
  buildCsvContent,
  triggerCsvDownload,
  triggerXlsxDownload,
  exportTimestamp,
  fetchAllPaged,
} from './deviceExport';
import { useT } from '@/hooks/useT';
import { useUserStore } from '@core/store/userStore';
import { useAppStore } from '@core/store/appStore';
import { buildDefaultUfteTaskName } from '@/pages/transfer/shared';
import dayjs from 'dayjs';
import { buildBatchTaskTypeMap, batchActionHasDetail, removeParamSyncOptimisticDeviceId } from './deviceBatchTask';
import { getDeviceListParamSyncPaths } from './deviceListParamSync';
import { formatDeviceRadioField, formatDeviceRFStatus } from './deviceRadioFieldSupport';
import {
  amfStatusMessageIdForDevice,
  bscLinkStatusMessageIdForDevice,
  mmeStatusMessageIdForDevice,
} from './deviceCoreNetworkStatus';
import { shouldShowLocationSyncIndicator } from './deviceGpsSyncIndicator';
import GpsSyncConfirmModal from './GpsSyncConfirmModal';
import GpsSyncTrigger from './GpsSyncTrigger';
import { applyLocationSyncResult, applyLocationSyncResultToList } from './deviceLocationSync';
import type { Device, DeviceListResponse } from '@core/types/device';
import { formatSystemTime } from '@core/utils/systemTime';
import { computeCumulativeOnlineDurationSeconds, computeCurrentOnlineDurationSeconds } from '@core/utils/onlineDuration';
import type { SysConfigItem } from '@core/types/system';
import { buildBatchItems } from '@/pages/system/SystemConfig/sysConfigSerialize';

const { Link } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

// 列表导出复用 useDeviceList 背后同一套 mock/real 切换,保证导出与展示口径一致。
const exportDeviceApi = createApiSwitch(deviceService as unknown as typeof deviceApi, deviceApi);
// DataTable tableId,导出时据此读取"列设置"localStorage(须与 <DataTable tableId> 一致)。
const DEVICE_LIST_TABLE_ID = 'device-list-table';
const ALARM_SYNC_BATCH_CONCURRENCY = 4;
const PARAM_SYNC_BATCH_CONCURRENCY = 4;
const PARAM_SYNC_ACTIVE_REFETCH_MS = 3000;
const PERIODIC_SYNC_WATCH_MS = 2 * 60 * 1000;
const PERIODIC_PARAM_SYNC_DEFAULTS = {
  periodicSyncEnabled: false,
  periodicSyncIntervalMinutes: 1440,
  periodicSyncBatchSize: 200,
  periodicSyncMaxConcurrent: 10,
  periodicSyncStaggerWindowMinutes: 0,
};

function decodeSysConfigValue(item: SysConfigItem): unknown {
  switch (item.valueType) {
    case 'bool':
      return item.value === 'true' || item.value === '1';
    case 'int': {
      const n = parseInt(item.value, 10);
      return Number.isFinite(n) ? n : 0;
    }
    case 'float': {
      const n = parseFloat(item.value);
      return Number.isFinite(n) ? n : 0;
    }
    case 'json':
      try {
        return JSON.parse(item.value) as unknown;
      } catch {
        return item.value;
      }
    default:
      return item.value;
  }
}

function createBatchAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError');
  }
  const error = new Error('The operation was aborted.');
  error.name = 'AbortError';
  return error;
}

function throwIfBatchAborted(signal: AbortSignal): void {
  if (signal.aborted) {
    throw createBatchAbortError();
  }
}

function isParamSyncAlreadyRunningError(err: unknown): boolean {
  if (!(err instanceof Error)) return false;
  const message = err.message.toLowerCase();
  return message.includes('active_sync_exists')
    || message.includes('already running')
    || message.includes('already exists')
    || message.includes('已有')
    || message.includes('正在同步');
}

// 筛选下拉框 name → 表格列 key 映射:列设置隐藏该列时,对应筛选下拉一并隐藏
// (用户决策 2026-06-09)。searchText 无对应列、不入表 → 始终显示。
const FILTER_COLUMN_MAP: Record<string, string> = {
  isOnline: 'connStatus',
  opState: 'opState',
  networkType: 'networkType',
  productId: 'deviceModel',
  productModel: 'productClass',
  modelName: 'deviceModel',
  softwareVersion: 'softwareVersion',
  groupId: 'groupName',
};

type TFn = (id: string, values?: Record<string, string | number>) => string;

function offlineDurationText(t: TFn, days?: number, hours?: number, minutes?: number): string {
  if (days === undefined || days === null) return '-';
  if (days >= 365) {
    const years = Math.floor(days / 365);
    const remainDays = days % 365;
    return remainDays > 0
      ? t('device.duration.yearsDays', { years, days: remainDays })
      : t('device.duration.years', { years });
  }
  if (days >= 30) {
    const months = Math.floor(days / 30);
    const remainDays = days % 30;
    return remainDays > 0
      ? t('device.duration.monthsDays', { months, days: remainDays })
      : t('device.duration.months', { months });
  }
  if (days > 0) {
    return hours && hours > 0
      ? t('device.duration.daysHours', { days, hours })
      : t('device.duration.days', { days });
  }
  if (hours && hours > 0) {
    return minutes && minutes > 0
      ? t('device.duration.hoursMinutes', { hours, minutes })
      : t('device.duration.hours', { hours });
  }
  if (minutes && minutes > 0) return t('device.duration.minutes', { minutes });
  return t('device.duration.lessThanMinute');
}

function offlineDurationPartsFromSeconds(seconds: number | null | undefined): { days: number; hours: number; minutes: number } | null {
  if (seconds === null || seconds === undefined) return null;
  const safeSeconds = Math.max(0, Math.floor(seconds));
  return {
    days: Math.floor(safeSeconds / 86400),
    hours: Math.floor((safeSeconds % 86400) / 3600),
    minutes: Math.floor((safeSeconds % 3600) / 60),
  };
}

function offlineDurationTextFromSeconds(t: TFn, seconds: number | null | undefined): string {
  const parts = offlineDurationPartsFromSeconds(seconds);
  if (!parts) return '-';
  return offlineDurationText(t, parts.days, parts.hours, parts.minutes);
}

function formatOfflineDuration(t: TFn, days?: number, hours?: number, minutes?: number): React.ReactNode {
  if (days === undefined || days === null) return '-';
  const text = offlineDurationText(t, days, hours, minutes);
  let color = 'default';
  if (days >= 365) color = 'red';
  else if (days >= 30) color = 'orange';
  else if (days > 0) color = days >= 7 ? 'orange' : 'gold';
  else if (hours && hours > 0) color = 'gold';
  return <Tag color={color}>{text}</Tag>;
}

function formatOfflineDurationFromSeconds(t: TFn, seconds: number | null | undefined): React.ReactNode {
  const parts = offlineDurationPartsFromSeconds(seconds);
  if (!parts) return '-';
  return formatOfflineDuration(t, parts.days, parts.hours, parts.minutes);
}

const URL_ARRAY_FIELDS = new Set<string>([
  'productModel',
  'modelName',
  'softwareVersion',
  'groupId',
]);

function parseUrlValue(key: string, value: string): unknown {
  return URL_ARRAY_FIELDS.has(key) ? value.split(',').filter(Boolean) : value;
}

export default function DeviceList() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { message, modal } = App.useApp();

  const prefetchDeviceDetailEntry = useCallback((device: Device) => {
    void import('@/pages/device/DeviceDetail');
    void prefetchDeviceDetailContext(queryClient, device);
  }, [queryClient]);

  const openDeviceDetail = useCallback((device: Device, tab?: string) => {
    prefetchDeviceDetailEntry(device);
    const suffix = tab ? `?tab=${tab}` : '';
    void navigate(`/device/detail/${device.sn}${suffix}`);
  }, [navigate, prefetchDeviceDetailEntry]);

  const resolveNameSyncFromList = useCallback(async (device: Device, action: 'use_lmt' | 'use_omc' | 'ignore') => {
    try {
      await deviceApi.resolveNameSync(device.id, action);
      void message.success(t('common.operationSuccess'));
      void queryClient.invalidateQueries({ queryKey: ['devices'] });
    } catch {
      void message.error(t('common.operationFailed'));
    }
  }, [message, queryClient, t]);

  const renderNameSyncActions = useCallback((device: Device) => (
    <Space direction="vertical" size={8}>
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {t('device.nameSyncPending')}
      </Typography.Text>
      <Space size={4}>
        <Button size="small" type="primary" onClick={() => void resolveNameSyncFromList(device, 'use_lmt')}>
          {t('device.nameSyncPending.useLmt')}
        </Button>
        <Button size="small" onClick={() => void resolveNameSyncFromList(device, 'use_omc')}>
          {t('device.nameSyncPending.useOmc')}
        </Button>
        <Button size="small" onClick={() => void resolveNameSyncFromList(device, 'ignore')}>
          {t('device.nameSyncPending.ignore')}
        </Button>
      </Space>
    </Space>
  ), [resolveNameSyncFromList, t]);

  const [currentPage, setCurrentPage] = useState(() => {
    const page = searchParams.get('page');
    return page ? parseInt(page, 10) : 1;
  });
  const [pageSize, setPageSize] = useState(() => {
    const size = searchParams.get('pageSize');
    return size ? parseInt(size, 10) : 20;
  });
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>(() => {
    const params: Record<string, unknown> = {};
    searchParams.forEach((value, key) => {
      if (key !== 'page' && key !== 'pageSize') {
        params[key] = parseUrlValue(key, value);
      }
    });
    return params;
  });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const [periodicSyncModalOpen, setPeriodicSyncModalOpen] = useState(false);
  const [periodicSyncForm] = Form.useForm();
  const { data: periodicSyncConfigs, isFetching: periodicSyncLoading } = useSysConfigsByCategory('device', periodicSyncModalOpen);
  const batchUpdateSysConfigs = useBatchUpdateSysConfigs();
  const [optimisticParamSyncDeviceIds, setOptimisticParamSyncDeviceIds] = useState<Set<string>>(() => new Set());
  const [optimisticAlarmSyncDeviceIds, setOptimisticAlarmSyncDeviceIds] = useState<Set<string>>(() => new Set());
  const [periodicSyncWatchUntil, setPeriodicSyncWatchUntil] = useState(0);

  useEffect(() => {
    const params: Record<string, unknown> = {};
    searchParams.forEach((value, key) => {
      if (key !== 'page' && key !== 'pageSize') {
        params[key] = parseUrlValue(key, value);
      }
    });
    const currentKeys = Object.keys(filterParams);
    const newKeys = Object.keys(params);
    if (currentKeys.length !== newKeys.length) {
      setFilterParams(params);
      return;
    }
    let changed = false;
    for (const key of newKeys) {
      if (params[key] !== filterParams[key]) {
        changed = true;
        break;
      }
    }
    if (changed) {
      setFilterParams(params);
    }
  }, [searchParams]);

  useEffect(() => {
    if (!periodicSyncModalOpen) return;
    const values: Record<string, unknown> = { ...PERIODIC_PARAM_SYNC_DEFAULTS };
    const hasMinuteInterval = (periodicSyncConfigs || []).some((cfg) => cfg.key === 'periodicSyncIntervalMinutes');
    for (const item of periodicSyncConfigs || []) {
      if (item.key in PERIODIC_PARAM_SYNC_DEFAULTS) {
        values[item.key] = decodeSysConfigValue(item);
      } else if (item.key === 'periodicSyncIntervalHours' && !hasMinuteInterval) {
        const hours = decodeSysConfigValue(item);
        if (typeof hours === 'number' && hours > 0) {
          values.periodicSyncIntervalMinutes = hours * 60;
        }
      }
    }
    periodicSyncForm.setFieldsValue(values);
  }, [periodicSyncModalOpen, periodicSyncConfigs, periodicSyncForm]);

  type TaskStatus = 'pending' | 'running' | 'success' | 'failed';
  interface LocalTask {
    id: string;
    sn: string;
    deviceName: string;
    type: string;
    status: TaskStatus;
    progress: number;
    message?: string;
    logContent?: string;
    hasDetail?: boolean;
  }

  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30);

  const [collectDrawerOpen, setCollectDrawerOpen] = useState(false);
  const [collectTasks, setCollectTasks] = useState<LocalTask[]>([]);
  const [collectDrawerTitle, setCollectDrawerTitle] = useState('');
  const [batchAlarmSyncRunning, setBatchAlarmSyncRunning] = useState(false);
  const [batchParamSyncRunning, setBatchParamSyncRunning] = useState(false);
  const [gpsSyncConfirmDevice, setGpsSyncConfirmDevice] = useState<Device | null>(null);

  const clearOptimisticParamSyncDevice = useCallback((deviceId: string) => {
    setOptimisticParamSyncDeviceIds((prev) => removeParamSyncOptimisticDeviceId(prev, deviceId));
  }, []);

  const clearOptimisticAlarmSyncDevice = useCallback((deviceId: string) => {
    setOptimisticAlarmSyncDeviceIds((prev) => removeParamSyncOptimisticDeviceId(prev, deviceId));
  }, []);

  const [logModalOpen, setLogModalOpen] = useState(false);
  const [currentLogTask, setCurrentLogTask] = useState<LocalTask | null>(null);
  const alarmSyncBatchAbortRef = useRef<AbortController | null>(null);
  const paramSyncBatchAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    return () => {
      alarmSyncBatchAbortRef.current?.abort();
      paramSyncBatchAbortRef.current?.abort();
    };
  }, []);

  const handleViewLog = useCallback((task: LocalTask) => {
    setCurrentLogTask(task);
    setLogModalOpen(true);
  }, []);

  const waitForDeviceTaskTerminal = useCallback(async (
    taskId: string,
    taskRowId: string,
    signal?: AbortSignal,
  ) => {
    return deviceTaskApi.waitForTerminal(taskId, {
      signal,
      onPoll: (task) => {
        setCollectTasks((prev) => prev.map((item) => {
          if (item.id !== taskRowId) return item;
          if (task.status === 'sent') {
            return { ...item, status: 'running', progress: Math.max(item.progress, 70) };
          }
          if (task.status === 'pending') {
            return { ...item, status: 'running', progress: Math.max(item.progress, 30) };
          }
          return item;
        }));
      },
    });
  }, []);

  // Remark 列头自定义标签
  const [remarkLabel, setRemarkLabel] = useState(() => {
    return localStorage.getItem('omc_remark_label') || 'Remark';
  });
  const [editingRemark, setEditingRemark] = useState(false);
  const [remarkInput, setRemarkInput] = useState('');

  const handleRemarkLabelSave = useCallback(() => {
    const val = remarkInput.trim();
    if (!val) return;
    setRemarkLabel(val);
    setEditingRemark(false);
    localStorage.setItem('omc_remark_label', val);
    // TODO: 接入 POST /cell/cpeinfos/updateColumnAlias.action
    // params: { columnName: 'remark', columnAlias: val }
  }, [remarkInput]);

  const handleRemarkLabelCancel = useCallback(() => {
    setEditingRemark(false);
  }, []);

  const [editingInstallAddressId, setEditingInstallAddressId] = useState<string | null>(null);
  const [editingInstallAddressValue, setEditingInstallAddressValue] = useState('');
  const [savingInstallAddressId, setSavingInstallAddressId] = useState<string | null>(null);

  // remarkHeaderRender 保持 useMemo，因为 headerRender 需要 ReactNode 而非函数
  // 编辑状态变化不频繁，性能开销可接受
  const remarkHeaderRender = useMemo(() => {
    if (editingRemark) {
      return (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }} onClick={(e) => e.stopPropagation()}>
          <Input
            size="small"
            value={remarkInput}
            onChange={(e) => setRemarkInput(e.target.value)}
            onPressEnter={handleRemarkLabelSave}
            style={{ width: 120 }}
            maxLength={30}
            autoFocus
          />
          <CheckOutlined
            style={{ fontSize: 12, color: '#52c41a', cursor: 'pointer' }}
            onClick={handleRemarkLabelSave}
          />
          <CloseOutlined
            style={{ fontSize: 12, color: '#ff4d4f', cursor: 'pointer' }}
            onClick={handleRemarkLabelCancel}
          />
        </span>
      );
    }
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
        <Tooltip title={remarkLabel}>
          <span style={{ maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {remarkLabel}
          </span>
        </Tooltip>
        <EditOutlined
          style={{ fontSize: 12, color: '#8c8c8c', cursor: 'pointer' }}
          onClick={(e) => {
            e.stopPropagation();
            setRemarkInput(remarkLabel);
            setEditingRemark(true);
          }}
        />
      </span>
    );
  }, [editingRemark, remarkInput, remarkLabel, handleRemarkLabelSave, handleRemarkLabelCancel]);
  // remark 列定义当前注释隐藏（见下方 columns 定义处说明）；remarkHeaderRender 有意保留以便恢复列时直接接回。
  // 此处显式引用消除 TS6133「声明未使用」，恢复 remark 列时删除本行即可。
  void remarkHeaderRender;

  const currentUser = useUserStore((s) => s.currentUser);
  const appLocale = useAppStore((s) => s.locale);
  // R6b: 设备分组下拉接入 device/group API（device-list-and-group-improvements-20260520.md R6b）
  const { data: groupsResp } = useDeviceGroups();
  const queryParams = useMemo(() => {
    const expandedGroupIDs = expandSelectedGroupIds(filterParams.groupId as string | string[] | undefined, groupsResp?.groups ?? []);
    return {
      ...filterParams,
      ...(expandedGroupIDs ? { groupId: expandedGroupIDs } : {}),
      page: currentPage,
      pageSize,
    } as Parameters<typeof useDeviceList>[0];
  }, [filterParams, groupsResp?.groups, currentPage, pageSize]);

  const paramSyncPolling = periodicSyncWatchUntil > 0 || optimisticParamSyncDeviceIds.size > 0;
  const { data, isLoading, isFetching, refetch } = useDeviceList(queryParams, {
    refetchInterval: autoRefresh ? refreshInterval * 1000 : (paramSyncPolling ? PARAM_SYNC_ACTIVE_REFETCH_MS : undefined),
  });
  const acceptLocationSync = useCallback(async (record: Device) => {
    const reportedVersion = record.locationSync.reported?.version;
    if (reportedVersion == null) {
      void message.error(t('device.gpsSyncNoReport'));
      return;
    }
    try {
      const result = await deviceApi.acceptLocationSync(record.id, reportedVersion);
      queryClient.setQueriesData<DeviceListResponse>(
        { queryKey: ['devices', 'list'] },
        (previous) => applyLocationSyncResultToList(previous, record.id, result),
      );
      queryClient.setQueryData<Device | null>(['devices', 'detail', record.id], (previous) => (
        previous ? applyLocationSyncResult(previous, result) : previous
      ));
      queryClient.setQueryData<Device | null>(['devices', 'sn', record.sn], (previous) => (
        previous ? applyLocationSyncResult(previous, result) : previous
      ));
      void message.success(t('device.gpsSyncSuccess'));
    } catch (error: unknown) {
      const status = (error as { response?: { status?: number } })?.response?.status;
      void message.error(
        status === 409
          ? t('device.gpsSyncConflict')
          : status === 403
            ? t('device.gpsSyncForbidden')
            : t('device.gpsSyncFailed'),
      );
      return;
    }

    // The promotion has already succeeded at this point. A list refresh is
    // best-effort so a transient query failure cannot be reported as a failed
    // GPS synchronization or leave the user unsure whether the action applied.
    try {
      const refreshResult = await refetch();
      if (refreshResult.isError) {
        void message.warning(t('device.gpsSyncRefreshFailed'));
      }
    } catch {
      void message.warning(t('device.gpsSyncRefreshFailed'));
    }
  }, [message, queryClient, refetch, t]);

  const renderLocationCell = useCallback((value: number | null | undefined, record: Device) => {
    const showSyncIndicator = shouldShowLocationSyncIndicator(record.locationSync);
    const displayValue = value == null ? '--' : value;
    if (!showSyncIndicator) return displayValue;
    return (
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 2,
          whiteSpace: 'nowrap',
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        <GpsSyncTrigger
          label={t('device.gpsSyncAction')}
          onClick={() => setGpsSyncConfirmDevice(record)}
        />
        {displayValue}
      </span>
    );
  }, [t]);
  const [refreshSpinnerActive, setRefreshSpinnerActive] = useState(false);
  const refreshSpinStartedAtRef = useRef<number | null>(null);
  const refreshSpinTimeoutRef = useRef<number | null>(null);
  const batchReboot = useBatchRebootDevices();
  const updateDevice = useUpdateDevice();
  const triggerAlarmSync = useTriggerAlarmSync();
  const syncDeviceParams = useSyncDeviceParams();
  useAlarmCountWithDeviceListInvalidation();
  const createUfteTask = useCreateUnifiedFileTransferTask();
  const taskNameUser = currentUser?.username || currentUser?.displayName || 'user';
  // 性能优化：使用 useMemo 避免每次渲染创建新引用，防止下游 callback/useMemo 依赖变化
  const devices = useMemo(
    () => withDeviceGroupDisplayName(
      data?.items ?? [],
      groupsResp?.groups ?? [],
      appLocale,
    ),
    [data?.items, groupsResp?.groups, appLocale]
  );
  const total = data?.total ?? 0;
  const stats = useMemo(() => data?.stats ?? { total: 0, online: 0, offline: 0, alarmed: 0, online_count: 0, offline_count: 0 }, [data?.stats]);

  useEffect(() => {
    if (periodicSyncWatchUntil <= Date.now()) return;
    const timeout = window.setTimeout(() => setPeriodicSyncWatchUntil(0), periodicSyncWatchUntil - Date.now());
    return () => window.clearTimeout(timeout);
  }, [periodicSyncWatchUntil]);

  useEffect(() => {
    if (!autoRefresh) return;
    void refetch();
  }, [autoRefresh, refreshInterval, refetch]);

  const startInstallAddressEdit = useCallback((device: Device) => {
    setEditingInstallAddressId(device.id);
    setEditingInstallAddressValue(device.installAddress || '');
  }, []);

  const cancelInstallAddressEdit = useCallback(() => {
    setEditingInstallAddressId(null);
    setEditingInstallAddressValue('');
    setSavingInstallAddressId(null);
  }, []);

  const saveInstallAddressEdit = useCallback((device: Device) => {
    const nextValue = editingInstallAddressValue.trim();
    const currentValue = (device.installAddress || '').trim();
    if (savingInstallAddressId === device.id) return;
    if (nextValue === currentValue) {
      cancelInstallAddressEdit();
      return;
    }

    setSavingInstallAddressId(device.id);
    updateDevice.mutate(
      {
        id: device.id,
        data: { installAddress: nextValue },
        fallbackDevice: {
          id: device.id,
          sn: device.sn,
          installAddress: device.installAddress,
          remark: device.remark,
        },
      },
      {
        onSuccess: (updatedDevice) => {
          queryClient.setQueryData(['devices', 'list', queryParams], (prev: typeof data) => {
            if (!prev) return prev;
            return {
              ...prev,
              items: prev.items.map((item) => item.id === updatedDevice.id ? { ...item, installAddress: updatedDevice.installAddress } : item),
            };
          });
          void queryClient.invalidateQueries({ queryKey: ['devices', 'sn', updatedDevice.sn] });
          void message.success(t('common.saveSuccess'));
          setEditingInstallAddressId(null);
          setEditingInstallAddressValue('');
          setSavingInstallAddressId(null);
        },
        onError: (err) => {
          setSavingInstallAddressId(null);
          void message.error(err instanceof Error ? err.message : t('common.saveFailed'));
        },
      },
    );
  }, [cancelInstallAddressEdit, data, editingInstallAddressValue, message, queryClient, queryParams, savingInstallAddressId, t, updateDevice]);

  const handleManualRefresh = useCallback(() => {
    if (refreshSpinTimeoutRef.current !== null) {
      window.clearTimeout(refreshSpinTimeoutRef.current);
      refreshSpinTimeoutRef.current = null;
    }
    if (refreshSpinStartedAtRef.current === null) {
      refreshSpinStartedAtRef.current = Date.now();
    }
    setRefreshSpinnerActive(true);
    void refetch();
  }, [refetch]);

  useEffect(() => {
    if (isFetching) {
      if (refreshSpinTimeoutRef.current !== null) {
        window.clearTimeout(refreshSpinTimeoutRef.current);
        refreshSpinTimeoutRef.current = null;
      }
      if (refreshSpinStartedAtRef.current === null) {
        refreshSpinStartedAtRef.current = Date.now();
      }
      setRefreshSpinnerActive(true);
      return;
    }

    if (refreshSpinStartedAtRef.current === null) {
      setRefreshSpinnerActive(false);
      return;
    }

    const elapsedMs = Date.now() - refreshSpinStartedAtRef.current;
    const remainingMs = Math.max(0, 1000 - elapsedMs);

    refreshSpinTimeoutRef.current = window.setTimeout(() => {
      refreshSpinStartedAtRef.current = null;
      refreshSpinTimeoutRef.current = null;
      setRefreshSpinnerActive(false);
    }, remainingMs);

    return () => {
      if (refreshSpinTimeoutRef.current !== null) {
        window.clearTimeout(refreshSpinTimeoutRef.current);
        refreshSpinTimeoutRef.current = null;
      }
    };
  }, [isFetching]);

  useEffect(() => () => {
    if (refreshSpinTimeoutRef.current !== null) {
      window.clearTimeout(refreshSpinTimeoutRef.current);
    }
  }, []);
  const groupOptions = useMemo(() => {
    const groups = groupsResp?.groups ?? [];
    // 仅 L2 子分组可作为设备过滤目标（L1 是容器）；用 parentName / name 双层展示便于辨识
    const byId = new Map(groups.map((g) => [g.id, g]));
    return groups
      .filter((g) => g.parentId !== null) // 排除 L1 根分组
      .map((g) => {
        const parent = g.parentId ? byId.get(g.parentId) : null;
        const name = getI18nText((g as { nameI18n?: Record<string, string> }).nameI18n, appLocale, g.name);
        const parentName = parent
          ? getI18nText((parent as { nameI18n?: Record<string, string> }).nameI18n, appLocale, parent.name)
          : '';
        return {
          label: parentName ? `${parentName} / ${name}` : name,
          value: g.id,
        };
      });
  }, [appLocale, groupsResp]);

  // R6c: 字典驱动 — 在线状态 / 激活状态 / 网络制式 / 产品类型
  // 字典 code 与种子数据在 migrations/000136 / 000137 维护。
  //
  // T-0162: conn_status 字典已废弃，替换为：
  //   - lifecycle_state（生命周期 6 状态）
  //   - is_online（在线 2 状态：true/false）
  // 新增 3 个字典（device_model / software_version / firmware_version）由
  // seed/000137 初始化为现有 devices/device_parameters 表的 distinct 值。
  // 性能优化：使用批量查询一次获取所有字典，减少 HTTP 请求
  const { data: batchDicts } = useDictionaryBatch([
    'is_online',
    'op_state',
    'network_type',
    'product_class',
    'device_model',
    'software_version',
    'firmware_version',
  ]);

  const isOnlineDict = batchDicts?.['is_online'];
  const opStateDict = batchDicts?.['op_state'];
  const networkTypeDict = batchDicts?.['network_type'];
  const productClassDict = batchDicts?.['product_class'];
  const deviceModelDict = batchDicts?.['device_model'];
  const softwareVersionDict = batchDicts?.['software_version'];

  type DictOptionDetail = { label: string; labelI18n?: Record<string, string>; value: string };

  const dictToOptions = useCallback(
    (
      dict: { sysDictionaryDetails?: DictOptionDetail[] } | undefined,
      formatLabel?: (detail: DictOptionDetail) => string,
    ) =>
      (dict?.sysDictionaryDetails ?? []).map((d) => ({
        label: formatLabel?.(d) ?? getI18nText(d.labelI18n, appLocale, d.label),
        value: d.value,
      })),
    [appLocale],
  );

  const formatOnlineOptionLabel = useCallback(
    (detail: DictOptionDetail) => {
      const label = getI18nText(detail.labelI18n, appLocale, detail.label);
      if (appLocale !== 'en-US' || !containsHan(label)) return label;
      const normalized = String(detail.value ?? '').trim().toLowerCase();
      return ['1', 'true', 'online', 'connected', 'active'].includes(normalized)
        ? t('status.online')
        : t('status.offline');
    },
    [appLocale, t],
  );

  const formatActivationOptionLabel = useCallback(
    (detail: DictOptionDetail) =>
      activationStatusLabelOf(detail.value, [detail], {
        active: t('status.active'),
        inactive: t('status.inactive'),
      }, appLocale) ?? getI18nText(detail.labelI18n, appLocale, detail.label),
    [appLocale, t],
  );

  // 产品名称下拉：选项来自 /products（label=产品名称，value=产品 UUID → devices.product_id）。
  // 与「产品类型」(product_class 字典) 不同，此处按产品装配件主键过滤。
  const { data: productListResp } = useProductList();
  const productOptions = useMemo(
    () => (productListResp?.items ?? []).map((p) => ({ label: localizeDeviceProductName(p.name, appLocale), value: p.id })),
    [appLocale, productListResp],
  );

  // 性能优化：SEVERITY_LABEL 改为函数调用，移除 useMemo
  // 仅 5 个字符串映射，计算开销可忽略，避免依赖 t 函数导致频繁重建
  const getSeverityLabel = useCallback((severity: string): string => {
    const labels: Record<string, string> = {
      critical: t('alarm.severity.critical'),
      major: t('alarm.severity.major'),
      minor: t('alarm.severity.minor'),
      warning: t('alarm.severity.warning'),
      none: t('alarm.severity.none'),
    };
    return labels[severity] || severity;
  }, [t]);

  // 状态值映射：兼容多种可能的数据格式，修复乱码问题
  const mapConnStatus = useCallback((status: string | undefined | null): 'online' | 'offline' => {
    if (!status) return 'offline';

    // 标准化处理：转小写、去空格
    const normalized = String(status).toLowerCase().trim();

    // 兼容多种可能的状态值
    const onlineValues = ['online', '1', 'true', 'yes', 'connected', 'active'];
    const offlineValues = ['offline', '0', 'false', 'no', 'disconnected', 'inactive'];

    if (onlineValues.includes(normalized)) return 'online';
    if (offlineValues.includes(normalized)) return 'offline';

    // 兜底：无法识别时返回 offline
    return 'offline';
  }, []);


  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    // --- 搜索项：文本搜索覆盖 SN/名称/IP/MAC/ECI/PCI ---
    // 后端 BuildSearchOR 支持英文逗号分隔多关键字（最多 50 个），
    // placeholder 提示用户可粘多 SN 一次搜。
    {
      name: 'searchText',
      label: t('filter.searchText'),
      type: 'input',
      placeholder: t('filter.searchText.multiHint'),
      width: 400,
    },

    // --- 筛选项：三制式公共（默认显示） ---
    // T-0162: 「生命周期」筛选已下线（用户反馈业务场景里实际只关心实时
    // 在线/离线，6 状态生命周期对前端筛选过细）。字典 lifecycle_state 在
    // 后端仍保留供详情页与统计 by_lifecycle 使用。
    {
      name: 'isOnline',
      label: t('device.connStatus'),
      type: 'select',
      width: 160,
      options: dictToOptions(isOnlineDict, formatOnlineOptionLabel),
    },
    {
      name: 'opState',
      label: t('device.opState'),
      type: 'select',
      width: 160,
      options: dictToOptions(opStateDict, formatActivationOptionLabel),
    },
    {
      name: 'networkType',
      label: t('device.radioMode'),
      type: 'select',
      width: 160,
      options: dictToOptions(networkTypeDict),
    },
    // 产品名称（按 devices.product_id 过滤，下拉来自 /products，value=产品 UUID）——置于产品类型之前。
    {
      name: 'productId',
      label: t('device.productName'),
      type: 'select',
      width: 160,
      options: productOptions,
    },
    {
      name: 'productModel',
      label: t('device.productClass'),
      type: 'multi-select',
      width: 160,
      options: dictToOptions(productClassDict),
    },

    // --- 筛选项：三制式公共（默认折叠） ---
    // T-0162: 3 个原本写死 [] 的下拉接入字典：device_model / software_version /
    // firmware_version。字典初始化数据由 seed/000137 从现有 devices /
    // device_parameters 表 distinct 灌入，保证用户任意选一项后端 filter 一定
    // 命中至少一条设备。后续新版本进来时管理员手动补字典词条。
    {
      name: 'modelName',
      label: t('device.model'),
      type: 'multi-select',
      width: 160,
      options: dictToOptions(deviceModelDict, (detail) => localizeDeviceProductName(getI18nText(detail.labelI18n, appLocale, detail.label), appLocale)),
    },
    {
      name: 'softwareVersion',
      label: t('device.softwareVersion'),
      type: 'multi-select',
      width: 160,
      options: dictToOptions(softwareVersionDict),
    },
    // R4 + R6b: groupId 接入 useDeviceGroups
    // width:400 与第一行 searchText 输入框对齐——分组名称长度普遍超过 160（含层级
    // "Region A / Subgroup B" 形式），160 时多选 chip 被截断成 "...""，体验差。
    {
      name: 'groupId',
      label: t('device.groupName'),
      type: 'multi-select',
      width: 400,
      options: groupOptions,
    },
  ], [
    t,
    productOptions,
    isOnlineDict,
    formatOnlineOptionLabel,
    opStateDict,
    formatActivationOptionLabel,
    networkTypeDict,
    productClassDict,
    deviceModelDict,
    softwareVersionDict,
    groupOptions,
    dictToOptions,
  ]);

  // 2026-06-03 用户决策:下拉(select/multi-select)提示统一在前面加「请选择」(无显式 placeholder 时回退 label,此处前置请选择)。
  const filterFields = useMemo(
    () =>
      FILTER_FIELDS.map((f) =>
        (f.type === 'select' || f.type === 'multi-select') && !f.placeholder
          ? { ...f, placeholder: `${t('common.pleaseSelect')}${f.label ?? ''}` }
          : f
      ),
    [FILTER_FIELDS, t]
  );

  // 列设置隐藏某列 → 对应搜索下拉框一并隐藏(FILTER_COLUMN_MAP 映射;searchText 无映射,始终显示)。
  // hiddenColumnKeys 由 <DataTable onHiddenColumnsChange> 在列设置变化时抬上来。
  const [hiddenColumnKeys, setHiddenColumnKeys] = useState<string[]>([]);
  const [tableMigrationVersion, setTableMigrationVersion] = useState(0);
  const visibleFilterFields = useMemo(
    () =>
      filterFields.filter((f) => {
        const colKey = FILTER_COLUMN_MAP[f.name];
        return !colKey || !hiddenColumnKeys.includes(colKey);
      }),
    [filterFields, hiddenColumnKeys]
  );

  useEffect(() => {
    const storageKey = 'omc_col_vis_device-list-table';
    try {
      const stored = localStorage.getItem(storageKey);
      if (!stored) return;
      const parsed = JSON.parse(stored) as unknown;
      if (!Array.isArray(parsed) || !parsed.includes('offlineDuration')) return;
      const next = parsed.filter((key): key is string => key !== 'offlineDuration');
      localStorage.setItem(storageKey, JSON.stringify(next));
      setHiddenColumnKeys((prev) => prev.filter((key) => key !== 'offlineDuration'));
      setTableMigrationVersion((prev) => prev + 1);
    } catch {
      // ignore malformed column settings and keep current table behavior
    }
  }, []);

  // 统计面板 — 基于筛选条件的全量统计（由后端 stats 字段返回，非当前页）
  // T-0162: 优先用 online_count / offline_count（与 backend DeviceListStats 1:1）；
  // 老 stats.online / stats.offline 字段在新前端不再使用（仅 mapListResponse 内部
  // 当 fallback 保留），新 UI 直读 stats.online_count。
  const activeAlarmCount = stats.alarmed;
  const statsItems = useMemo(() => [
    { label: t('device.count.total'), value: stats.total },
    { label: t('status.online'), value: stats.online_count ?? stats.online ?? 0, color: '#52C41A' },
    { label: t('status.offline'), value: stats.offline_count ?? stats.offline ?? 0, color: '#8C8C8C' },
    { label: t('alarm.stat.activeAlarm'), value: activeAlarmCount, color: '#FA8C16' },
  ], [activeAlarmCount, stats, t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    // 关键字上限校验：后端 BuildSearchOR 限定 ≤50 keyword × 6 fields = 300 ILIKE
    // 子句，超过会被静默截断。前端这里做截断 + 用户提示，让感知明确。
    let effective: Record<string, unknown> = values;
    const raw = values.searchText;
    if (typeof raw === 'string' && raw.includes(',')) {
      const keywords = raw
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      if (keywords.length > 50) {
        void message.warning(t('filter.searchText.tooManyKeywords'));
        effective = { ...values, searchText: keywords.slice(0, 50).join(',') };
      }
    }

    setFilterParams(effective);
    setCurrentPage(1);
    // 同步到 URL
    const newParams = new URLSearchParams();
    Object.entries(effective).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        if (Array.isArray(value)) {
          if (value.length > 0) {
            newParams.set(key, value.join(','));
          }
        } else {
          newParams.set(key, String(value));
        }
      }
    });
    setSearchParams(newParams);
  }, [setSearchParams, message, t]);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    // 清空 URL 参数
    setSearchParams(new URLSearchParams());
  }, [setSearchParams]);

  // 批量操作通用确认弹窗
  const handleBatchAction = useCallback(
    (actionLabel: string, ids: React.Key[], actionKey?: string) => {
      if (actionKey === 'batch-alarm-sync' && batchAlarmSyncRunning) {
        void message.warning(t('task.status.running'));
        return;
      }
      if (actionKey === 'batch-param-sync' && batchParamSyncRunning) {
        void message.warning(t('task.status.running'));
        return;
      }
      modal.confirm({
        title: t('common.confirm'),
        content: t('device.batch.actionConfirm', { action: actionLabel, count: ids.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: () => {
          void (async () => {
            const selectedDevices = devices.filter((d) => ids.includes(d.id));
            const taskTypeMap = buildBatchTaskTypeMap(t);
            const newTasks: LocalTask[] = selectedDevices.map((device, index) => ({
              id: `${actionKey}-${device.sn}-${Date.now()}-${index}`,
              sn: device.sn,
              deviceName: device.name || device.hostName || device.sn,
              type: taskTypeMap[actionKey ?? ''] || actionLabel,
              status: 'pending' as TaskStatus,
              progress: 0,
              hasDetail: batchActionHasDetail(actionKey),
            }));
            const alarmSyncEntries = actionKey === 'batch-alarm-sync'
              ? selectedDevices.map((device, index) => ({
                device,
                task: newTasks[index],
              }))
              : [];
            const paramSyncEntries = actionKey === 'batch-param-sync'
              ? selectedDevices.map((device, index) => ({
                device,
                task: newTasks[index],
                parameterPaths: getDeviceListParamSyncPaths(device),
              }))
              : [];
            const runnableAlarmSyncEntries = alarmSyncEntries.filter(({ device, task }) => device.isOnline && Boolean(task.sn));
            const runnableAlarmSyncRowIds = new Set(runnableAlarmSyncEntries.map(({ task }) => task.id));
            const runnableParamSyncEntries = paramSyncEntries.filter(({ device, task, parameterPaths }) =>
              device.isOnline && Boolean(task.sn) && parameterPaths.length > 0
            );
            const runnableParamSyncRowIds = new Set(runnableParamSyncEntries.map(({ task }) => task.id));

            if (actionKey === 'batch-log-collect') {
              try {
                const taskName = buildDefaultUfteTaskName(
                  'RUNTIME_LOG_COLLECT',
                  taskNameUser,
                  appLocale,
                  dayjs().format('YYYY-MM-DD HH:mm:ss'),
                );
                await createUfteTask.mutateAsync({
                  taskName,
                  typeCode: 'RUNTIME_LOG_COLLECT',
                  deviceIds: selectedDevices.map((d) => d.id),
                  deviceCount: selectedDevices.length,
                  executionMode: 'immediate',
                });
                void message.success({
                  content: t('device.action.logCollectTriggered'),
                  duration: 5,
                });
              } catch {
                void message.error(t('common.operationFailed'));
              }
              setSelectedRowKeys([]);
              return;
            }

            setCollectDrawerTitle(t('task.taskProgress'));
            if (actionKey === 'batch-alarm-sync') {
              const timestamp = new Date().toISOString();
              setCollectTasks(newTasks.map((task) => {
                if (!runnableAlarmSyncRowIds.has(task.id)) {
                  return {
                    ...task,
                    status: 'failed' as TaskStatus,
                    progress: 100,
                    message: t('device.batch.alarmSync.onlyOnline'),
                    logContent: `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${task.sn}\n[${timestamp}] ERROR: ${t('device.batch.alarmSync.onlyOnline')}`,
                  };
                }
                return {
                  ...task,
                  status: 'running' as TaskStatus,
                  progress: 10,
                  message: t('task.status.running'),
                };
              }));
            } else if (actionKey === 'batch-param-sync') {
              const timestamp = new Date().toISOString();
              setCollectTasks(newTasks.map((task) => {
                if (!runnableParamSyncRowIds.has(task.id)) {
                  const entry = paramSyncEntries.find(({ task: rowTask }) => rowTask.id === task.id);
                  const reason = entry?.device.isOnline === false
                    ? t('device.batch.paramSync.onlyOnline')
                    : t('device.batch.paramSync.noSupportedParams');
                  return {
                    ...task,
                    status: 'failed' as TaskStatus,
                    progress: 100,
                    message: reason,
                    logContent: `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${task.sn}\n[${timestamp}] ERROR: ${reason}`,
                  };
                }
                return {
                  ...task,
                  status: 'running' as TaskStatus,
                  progress: 10,
                  message: t('task.status.running'),
                };
              }));
            } else {
              setCollectTasks(newTasks);
            }
            setCollectDrawerOpen(true);

            if (actionKey !== 'batch-alarm-sync' && actionKey !== 'batch-param-sync') {
              newTasks.forEach((task, index) => {
                setTimeout(() => {
                  setCollectTasks((prev) => prev.map((item) =>
                    item.id === task.id ? { ...item, status: 'running', progress: 10 } : item
                  ));

                  const progressInterval = setInterval(() => {
                    setCollectTasks((prev) => prev.map((item) => {
                      if (item.id !== task.id) return item;
                      if (item.progress >= 100) {
                        clearInterval(progressInterval);
                        return item;
                      }
                      const randomProgress = Math.random() * 15 + 10;
                      return { ...item, progress: Math.min(item.progress + randomProgress, 90) };
                    }));
                  }, 200);

                  const completeTime = 1000 + Math.random() * 1000;
                  setTimeout(() => {
                    clearInterval(progressInterval);
                    const success = Math.random() > 0.1;
                    const timestamp = new Date().toISOString();
                    const logContent = success
                      ? `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${task.sn}\n[${timestamp}] INFO: ${t('task.log.success')}`
                      : `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${task.sn}\n[${timestamp}] ERROR: ${t('task.log.failed')}`;
                    setCollectTasks((prev) => prev.map((item) =>
                      item.id === task.id ? {
                        ...item,
                        status: success ? 'success' : 'failed',
                        progress: 100,
                        message: success ? t('task.status.completed') : t('common.failed'),
                        logContent,
                      } : item
                    ));
                  }, completeTime);
                }, index * 200);
              });
            }

            if (actionKey === 'batch-reboot') {
              try {
                await batchReboot.mutateAsync(ids.map(String));
                void message.success(t('common.commandSent'));
              } catch {
                void message.error(t('common.operationFailed'));
              }
            } else if (actionKey === 'batch-alarm-sync') {
              if (runnableAlarmSyncEntries.length === 0) {
                void message.warning(t('device.batch.alarmSync.onlyOnline'));
                setSelectedRowKeys([]);
                return;
              }

              setBatchAlarmSyncRunning(true);
              setOptimisticAlarmSyncDeviceIds((prev) => {
                const next = new Set(prev);
                for (const { device } of runnableAlarmSyncEntries) {
                  next.add(device.id);
                }
                return next;
              });
              const abortController = new AbortController();
              alarmSyncBatchAbortRef.current = abortController;

              try {
                const results = await mapWithConcurrencyLimit(
                  runnableAlarmSyncEntries,
                  ALARM_SYNC_BATCH_CONCURRENCY,
                  async ({ device, task }) => {
                    const taskId = task.id;
                    const sn = task.sn;
                    try {
                      const triggerResult = await triggerAlarmSync.mutateAsync(sn);
                      if (!triggerResult.taskId || !taskId) {
                        throw new Error(t('device.batch.alarmSync.taskUnavailable'));
                      }
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === taskId ? {
                          ...item,
                          status: 'running',
                          progress: 30,
                          message: t('task.status.running'),
                        } : item
                      ));
                      const completedTask = await waitForDeviceTaskTerminal(
                        triggerResult.taskId,
                        taskId,
                        abortController.signal,
                      );
                      if (completedTask.status !== 'completed') {
                        throw new Error(completedTask.errorMessage || completedTask.status);
                      }
                      const timestamp = new Date().toISOString();
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === taskId ? {
                          ...item,
                          status: 'success',
                          progress: 100,
                          message: t('task.status.completed'),
                          logContent: `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${sn}\n[${timestamp}] INFO: alarm sync completed and persisted\n[${timestamp}] INFO: ${t('task.log.success')}`,
                        } : item
                      ));
                      clearOptimisticAlarmSyncDevice(device.id);
                      return true;
                    } catch (err) {
                      if (isAbortError(err)) {
                        throw err;
                      }
                      clearOptimisticAlarmSyncDevice(device.id);
                      const timestamp = new Date().toISOString();
                      const errMsg = err instanceof Error ? err.message : t('task.log.failed');
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === taskId ? {
                          ...item,
                          status: 'failed',
                          progress: 100,
                          message: t('common.failed'),
                          logContent: `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${sn}\n[${timestamp}] ERROR: ${errMsg}`,
                        } : item
                      ));
                      throw err;
                    }
                  },
                );

                const aborted = abortController.signal.aborted
                  || results.some((result) => result.status === 'rejected' && isAbortError(result.reason));
                if (aborted) {
                  return;
                }

                const succeededCount = results.filter((result) => result.status === 'fulfilled').length;
                const failedCount = newTasks.length - succeededCount;
                if (succeededCount > 0) {
                  await queryClient.invalidateQueries({ queryKey: ['alarms'] });
                }
                if (succeededCount > 0 && failedCount === 0) {
                  void message.success(t('common.commandSent'));
                } else if (succeededCount > 0) {
                  void message.warning(t('device.batch.alarmSync.partialResult', { success: succeededCount, failed: failedCount }));
                } else {
                  void message.error(t('common.operationFailed'));
                }
              } finally {
                setBatchAlarmSyncRunning(false);
                setOptimisticAlarmSyncDeviceIds(new Set());
                if (alarmSyncBatchAbortRef.current === abortController) {
                  alarmSyncBatchAbortRef.current = null;
                }
              }
            } else if (actionKey === 'batch-param-sync') {
              if (runnableParamSyncEntries.length === 0) {
                void message.warning(t('device.batch.paramSync.noRunnableDevices'));
                setSelectedRowKeys([]);
                return;
              }

              setBatchParamSyncRunning(true);
              setOptimisticParamSyncDeviceIds((prev) => {
                const next = new Set(prev);
                for (const { device } of runnableParamSyncEntries) {
                  next.add(device.id);
                }
                return next;
              });
              const abortController = new AbortController();
              paramSyncBatchAbortRef.current = abortController;

              try {
                const results = await mapWithConcurrencyLimit(
                  runnableParamSyncEntries,
                  PARAM_SYNC_BATCH_CONCURRENCY,
                  async ({ device, task, parameterPaths }) => {
                    const timestamp = new Date().toISOString();
                    try {
                      throwIfBatchAborted(abortController.signal);
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === task.id ? {
                          ...item,
                          status: 'running',
                          progress: 30,
                          message: t('task.status.running'),
                        } : item
                      ));
                      const result = await syncDeviceParams.mutateAsync({ deviceId: device.id, parameterPaths });
                      throwIfBatchAborted(abortController.signal);
                      if (!result.requestId) {
                        throw new Error(t('device.batch.paramSync.requestUnavailable'));
                      }
                      const taskCountText = result.taskCount !== undefined
                        ? `, queued ${result.taskCount} task(s)`
                        : '';
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === task.id ? {
                          ...item,
                          status: 'running',
                          progress: 60,
                          message: t('device.batch.paramSync.waitingDevice'),
                          logContent: `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${device.sn}\n[${timestamp}] INFO: paramsync request accepted, ${parameterPaths.length} list path(s) requested${taskCountText}`,
                        } : item
                      ));
                      const terminalRequest = await deviceApi.waitForParameterSyncRequest(result.requestId, {
                        signal: abortController.signal,
                        onPoll: (request) => {
                          setCollectTasks((prev) => prev.map((item) => {
                            if (item.id !== task.id) return item;
                            if (request.status === 'running') {
                              return { ...item, status: 'running', progress: Math.max(item.progress, 70), message: t('device.batch.paramSync.waitingDevice') };
                            }
                            return item;
                          }));
                        },
                      });
                      if (terminalRequest.status !== 'succeeded' || terminalRequest.resultCode === 'NO_STORABLE_PATH') {
                        throw new Error(terminalRequest.errorMessage || terminalRequest.resultCode || terminalRequest.status);
                      }
                      throwIfBatchAborted(abortController.signal);
                      await queryClient.invalidateQueries({ queryKey: ['devices'] });
                      throwIfBatchAborted(abortController.signal);
                      clearOptimisticParamSyncDevice(device.id);
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === task.id ? {
                          ...item,
                          status: 'success',
                          progress: 100,
                          message: t('task.status.completed'),
                          logContent: `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${device.sn}\n[${timestamp}] INFO: paramsync completed, ${parameterPaths.length} list path(s) requested${taskCountText}\n[${timestamp}] INFO: ${t('task.log.success')}`,
                        } : item
                      ));
                      return true;
                    } catch (err) {
                      if (isAbortError(err)) {
                        throw err;
                      }
                      const alreadyRunning = isParamSyncAlreadyRunningError(err);
                      if (alreadyRunning) {
                        const skippedMessage = t('device.batch.paramSync.alreadyRunning');
                        clearOptimisticParamSyncDevice(device.id);
                        setCollectTasks((prev) => prev.map((item) =>
                          item.id === task.id ? {
                            ...item,
                            status: 'success',
                            progress: 100,
                            message: skippedMessage,
                            logContent: `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${device.sn}\n[${timestamp}] INFO: ${skippedMessage}`,
                          } : item
                        ));
                        return true;
                      }
                      const errMsg = err instanceof Error ? err.message : t('task.log.failed');
                      clearOptimisticParamSyncDevice(device.id);
                      setCollectTasks((prev) => prev.map((item) =>
                        item.id === task.id ? {
                          ...item,
                          status: 'failed',
                          progress: 100,
                          message: t('common.failed'),
                          logContent: `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${device.sn}\n[${timestamp}] ERROR: ${errMsg}`,
                        } : item
                      ));
                      throw err;
                    }
                  },
                );

                const aborted = abortController.signal.aborted
                  || results.some((result) => result.status === 'rejected' && isAbortError(result.reason));
                if (aborted) {
                  return;
                }

                const succeededCount = results.filter((result) => result.status === 'fulfilled').length;
                const failedCount = newTasks.length - succeededCount;
                if (succeededCount > 0 && failedCount === 0) {
                  void message.success(t('device.batch.paramSync.success', { count: succeededCount }));
                } else if (succeededCount > 0) {
                  void message.warning(t('device.batch.paramSync.partialResult', { success: succeededCount, failed: failedCount }));
                } else {
                  void message.error(t('common.operationFailed'));
                }
              } finally {
                setBatchParamSyncRunning(false);
                setOptimisticParamSyncDeviceIds(new Set());
                if (paramSyncBatchAbortRef.current === abortController) {
                  paramSyncBatchAbortRef.current = null;
                }
              }
            } else {
              void message.success(t('common.commandSent'));
            }
            setSelectedRowKeys([]);
          })();
        },
      });
    },
    [
      appLocale,
      batchAlarmSyncRunning,
      batchParamSyncRunning,
      batchReboot,
      clearOptimisticAlarmSyncDevice,
      clearOptimisticParamSyncDevice,
      createUfteTask,
      devices,
      message,
      modal,
      queryClient,
      t,
      taskNameUser,
      triggerAlarmSync,
      syncDeviceParams,
      waitForDeviceTaskTerminal,
    ]
  );

  // 导出实现见 columns 定义之后的 handleExport(依赖 columns,需在其后声明)。

  /** 解析多小区逗号分隔值为 cell 数组 */
  const parseCellValues = useCallback((v: string | undefined | null): string[] => {
    if (!v || v === '--') return [];
    return String(v).split(',').map((s) => s.trim()).filter(Boolean);
  }, []);

  // 格式化时间戳
  const fmtTime = useCallback((v: string) => (v ? formatSystemTime(v) : '-'), []);

  // 格式化在线时长(秒)
  const fmtDuration = useCallback((seconds: number | null | undefined) => {
    if (!seconds) return '-';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return d > 0 ? `${d}d ${h}h ${m}m` : h > 0 ? `${h}h ${m}m` : `${m}m`;
  }, []);

  const onlineDurationOf = useCallback((record: Device) => computeCurrentOnlineDurationSeconds({
    isOnline: record.isOnline,
    onlineTime: record.onlineTime,
    offlineTime: record.offlineTime,
    fallbackOnlineDuration: record.onlineDuration,
  }), []);

  const cumulativeOnlineDurationOf = useCallback((record: Device) => computeCumulativeOnlineDurationSeconds({
    isOnline: record.isOnline,
    onlineTime: record.onlineTime,
    offlineTime: record.offlineTime,
    fallbackOnlineDuration: record.onlineDuration,
    cumulativeOnlineDuration: record.cumulativeOnlineDuration,
  }), []);

  const offlineDurationOf = useCallback((record: Device) => {
    if (record.connStatus !== 'offline') return null;
    const fallbackOfflineSeconds =
      typeof record.offlineSeconds === 'number' && !Number.isNaN(record.offlineSeconds)
        ? Math.max(0, Math.floor(record.offlineSeconds))
        : null;
    if (!record.offlineTime) return fallbackOfflineSeconds;
    const offlineAt = dayjs(record.offlineTime);
    if (!offlineAt.isValid()) return fallbackOfflineSeconds;
    return Math.max(0, dayjs().diff(offlineAt, 'second'));
  }, []);

  // 状态值渲染辅助
  const fmtStatus = useCallback(
    (value: string | number | boolean | undefined | null, map: Record<string, { label: string; color: string }>) => {
      const v = String(value ?? '');
      const entry = map[v] ?? map[v.trim().toLowerCase()];
      if (!entry) return v || '-';
      return <Tag color={entry.color}>{entry.label}</Tag>;
    },
    []
  );

  const adminStateStatusMap = useMemo<Record<string, { label: string; color: string }>>(() => ({
    // CellEnable.AdminState: 1 = 未锁定，0 = 锁定。
    '1': { label: t('status.unlocked'), color: 'success' },
    '0': { label: t('status.locked'), color: 'warning' },
    '2': { label: t('status.unlocked'), color: 'success' },
    '3': { label: t('status.shuttingDown'), color: 'error' },
    true: { label: t('status.locked'), color: 'warning' },
    false: { label: t('status.unlocked'), color: 'success' },
    enabled: { label: t('status.locked'), color: 'warning' },
    disabled: { label: t('status.unlocked'), color: 'success' },
    locked: { label: t('status.locked'), color: 'warning' },
    unlocked: { label: t('status.unlocked'), color: 'success' },
    shuttingdown: { label: t('status.shuttingDown'), color: 'error' },
    'shutting down': { label: t('status.shuttingDown'), color: 'error' },
  }), [t]);

  const adminStateLabelOf = useCallback((value: string | number | boolean | undefined | null) => {
    const raw = String(value ?? '').trim();
    if (!raw) return '-';
    return adminStateStatusMap[raw]?.label ?? adminStateStatusMap[raw.toLowerCase()]?.label ?? raw;
  }, [adminStateStatusMap]);

  // ── 多小区/多连接状态渲染辅助 ──
  // 原始 JSP: 逗号分隔 "on,off,on" / "1,0,1" 表示多小区状态
  // 汇总显示 + 可点击 [N/M] Popover 查看逐小区明细

  /** 判断多小区汇总状态: 'all_on' | 'mixed' | 'all_off' */
  const getCellSummary = useCallback((cells: string[], onValues: string[]): 'all_on' | 'mixed' | 'all_off' => {
    const hasOn = cells.some((c) => onValues.includes(c));
    const hasOff = cells.some((c) => !onValues.includes(c));
    if (hasOn && hasOff) return 'mixed';
    if (hasOn) return 'all_on';
    return 'all_off';
  }, []);

  /** 渲染多小区状态: 汇总 Tag + [N/M] Popover */
  const renderMultiCellStatus = useCallback(
    (
      value: string | undefined | null,
      onValues: string[],
      labels: { on: string; off: string; title: string },
      colors: { on: string; off: string; mixed: string },
    ) => {
      if (!value || value === '--') return '-';
      const cells = parseCellValues(value);
      if (cells.length === 0) return '-';

      // 单小区 — 直接显示 Tag
      if (cells.length === 1) {
        const isOn = onValues.includes(cells[0]);
        return <Tag color={isOn ? colors.on : colors.off}>{isOn ? labels.on : labels.off}</Tag>;
      }

      // 多小区 — 汇总 + Popover
      const summary = getCellSummary(cells, onValues);
      const activeCount = cells.filter((c) => onValues.includes(c)).length;
      const summaryColor = summary === 'all_on' ? colors.on : summary === 'all_off' ? colors.off : colors.mixed;
      const summaryLabel = summary === 'all_off' ? labels.off : labels.on;

      const popoverContent = (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px 16px', padding: '8px 0' }}>
          {cells.map((cell, idx) => {
            const isOn = onValues.includes(cell);
            return (
              <span key={idx} style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                Cell {idx + 1}:
                <Tag color={isOn ? colors.on : colors.off} style={{ margin: 0 }}>
                  {isOn ? labels.on : labels.off}
                </Tag>
              </span>
            );
          })}
        </div>
      );

      return (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <Tag color={summaryColor}>{summaryLabel}</Tag>
          <Popover title={labels.title} content={popoverContent} trigger="click">
            <span style={{ color: '#4d84ff', cursor: 'pointer' }}>
              [{activeCount}/{cells.length}]
            </span>
          </Popover>
        </span>
      );
    },
    [parseCellValues, getCellSummary]
  );

  // 激活状态 = 设备是否曾首次上线（后端 op_state = DeriveOpStateActivated(first_online_time)：
  // '1'=激活 / '0'=未激活）。与"在线(实时连接 is_online)""小区状态(cell_status)"是三个不同维度——
  // 此前误用 cell_status==='normal' 当激活，导致在线设备因小区 inactive 显示未激活。
  //
  // ™ 判定口径由 frontend-core/utils/activationStatus.ts 统一控管——
  // V1 列表与详情统一调用同一函数，
  // 在上层各自渲染 Tag/文本。修改判定请只改 utility。
  const renderActivationStatus = useCallback((opState: string | undefined | null, isOnline: boolean | undefined | null) => {
    const status = displayActivationStatusOf(opState, isOnline);
    if (status == null) return '-';
    const isActive = status === 'active';
    const label = displayActivationStatusLabelOf(opState, isOnline, opStateDict?.sysDictionaryDetails, {
      active: t('status.active'),
      inactive: t('status.inactive'),
    }, appLocale);
    return <Tag color={isActive ? 'success' : 'error'}>{label}</Tag>;
  }, [appLocale, opStateDict?.sysDictionaryDetails, t]);

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      // =====================================================================
      // 公共字段 (common) — 三制式共有或多制式共享
      // =====================================================================
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 180,
        fixed: 'left',
        mono: true,
        copyable: true,
        group: 'common',
        render: (_val, record) => (
          <Link
            style={{ fontFamily: 'monospace' }}
            onMouseEnter={() => prefetchDeviceDetailEntry(record)}
            onFocus={() => prefetchDeviceDetailEntry(record)}
            onClick={() => openDeviceDetail(record)}
          >
            {record.sn}
          </Link>
        ),
      },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 125,
        fixed: 'left',
        group: 'common',
        render: (_val, record) => {
          const mappedStatus = mapConnStatus(record.connStatus);
          const syncing = mappedStatus === 'online' && (
            record.paramSyncRunning
            || optimisticParamSyncDeviceIds.has(record.id)
            || optimisticAlarmSyncDeviceIds.has(record.id)
          );
          return (
            <Space size={6} wrap={false}>
              <StatusIndicator
                status={mappedStatus}
                text={mappedStatus === 'online' ? t('status.online') : t('status.offline')}
              />
              {syncing && (
                <Tooltip title={t('device.periodicParamSync.running')}>
                  <Tag
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      width: 22,
                      height: 22,
                      padding: 0,
                      marginInlineEnd: 0,
                      color: '#1677ff',
                      background: '#e6f4ff',
                      borderColor: '#91caff',
                    }}
                  >
                    <span
                      style={{
                        position: 'relative',
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: 16,
                        height: 16,
                      }}
                    >
                      <LinkOutlined style={{ fontSize: 11, color: '#0958d9' }} />
                      <SyncOutlined
                        spin
                        style={{
                          position: 'absolute',
                          inset: 0,
                          fontSize: 16,
                          color: '#1677ff',
                        }}
                      />
                    </span>
                  </Tag>
                </Tooltip>
              )}
            </Space>
          );
        },
      },
      {
        key: 'alarmLevel',
        title: t('device.alarmLevel'),
        dataIndex: 'alarmLevel',
        width: 100,
        // 2026-06-04 用户决策:前三列(SN/连接状态/告警级别)固定左侧,横向滚动时
        // 保持不透明(固定列背景由 DataTable.module.css 的 .ant-table-cell-fix-left 统一处理)。
        fixed: 'left',
        group: 'common',
        render: (_val, record) => {
          const color = SEVERITY_COLOR[record.alarmLevel] ?? 'default';
          const label = getSeverityLabel(record.alarmLevel);
          if (hasAlarmSeverity(record.alarmLevel)) {
            // #361: 告警级别 Tag 旁拼接活动告警数（如「重要 · 3」）。
            const count = record.activeAlarmCount ?? 0;
            const display = count > 0 ? `${label} · ${count}` : label;
            // 点击告警跳转到设备详情当前告警 tab
            return (
              <Tag
                color={color}
                style={{ cursor: 'pointer' }}
                onMouseEnter={() => prefetchDeviceDetailEntry(record)}
                onClick={() => openDeviceDetail(record, 'alarms')}
              >
                {display}
              </Tag>
            );
          }
          return <Tag color={color}>{label}</Tag>;
        },
      },
      // "名称" 列绑定 device_name（设备名称），而非 host_name。
      // Issue #758: nameSyncPending=true 时显示小红点提示名称待同步
      {
        key: 'hostName',
        title: t('device.hostName'),
        dataIndex: 'deviceName',
        width: 170,
        ellipsis: true,
        group: 'common',
        render: (_val, record) => (
          <Space size={4}>
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {record.deviceName || '-'}
            </span>
            {record.nameSyncPending && (
              <Popover content={renderNameSyncActions(record)} trigger="click" placement="bottomLeft">
                <Badge status="error" style={{ cursor: 'pointer' }} onClick={(event) => event.stopPropagation()} />
              </Popover>
            )}
          </Space>
        ),
      },
      {
        key: 'networkType',
        title: t('device.radioMode'),
        dataIndex: 'networkType',
        width: 100,
        group: 'common',
        render: (_val, record) => {
          const colorMap: Record<string, string> = { eNB: 'blue', gNB: 'green', GSM: 'orange' };
          // issue #223: 列文案与「基站制式」筛选下拉同源——走 network_type 字典
          // value→label 映射（与回收站统一），不再直接显示原始 eNB/gNB。
          const label = resolveNetworkTypeLabel(record.networkType, networkTypeDict?.sysDictionaryDetails, appLocale);
          return <Tag color={colorMap[record.networkType] ?? 'default'}>{label}</Tag>;
        },
      },
      // 产品名称（= device.model_name，inform 命中产品后回填 product.Name）显示在产品类型前面。
      {
        key: 'deviceModel',
        title: t('device.productName'),
        dataIndex: 'deviceModel',
        width: 120,
        ellipsis: true,
        group: 'common',
        render: (_val, record) => localizeDeviceProductName(record.deviceModel, appLocale),
      },
      {
        key: 'productClass',
        title: t('device.productClass'),
        dataIndex: 'productClass',
        width: 120,
        group: 'common',
        // 原始 JSP: product 字段 — PM-B4860/QAFA/BaiBNX/BSC/BTS 等
        render: (_val, record) => record.productClass || '-',
      },
      { key: 'softwareVersion', title: t('device.softwareVersion'), dataIndex: 'softwareVersion', width: 140, ellipsis: true, group: 'common' },
      {
        key: 'ipAddress',
        title: t('device.ipAddress'),
        dataIndex: 'ipAddress',
        width: 140,
        mono: true,
        copyable: true,
        group: 'common',
        // 原始 JSP: IP 地址可点击，打开设备 Web UI
        render: (_val, record) => {
          const ip = record.ipAddress;
          if (!ip) return '-';
          return (
            <a href={`https://${ip}`} target="_blank" rel="noopener noreferrer"
              style={{ fontFamily: 'monospace', color: '#4d84ff' }}
            >
              {ip}
            </a>
          );
        },
      },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true, copyable: true, group: 'common' },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 120, group: 'common' },
      {
        key: 'firstOnlineTime',
        title: t('device.firstOnlineTime'),
        dataIndex: 'firstOnlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.firstOnlineTime),
      },
      {
        key: 'onlineTime',
        title: t('device.onlineTime'),
        dataIndex: 'onlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.onlineTime),
      },
      {
        key: 'lastOnlineTime',
        title: t('device.lastOnline'),
        dataIndex: 'lastOnlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.lastOnlineTime),
      },
      {
        key: 'offlineTime',
        title: t('device.offlineTime'),
        dataIndex: 'offlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.offlineTime),
      },
      {
        key: 'lastParamSyncAt',
        title: t('device.lastParamSyncAt'),
        dataIndex: 'lastParamSyncAt',
        width: 170,
        group: 'common',
        render: (_val, record) => {
          const syncText = record.lastParamSyncAt ? fmtTime(record.lastParamSyncAt) : t('device.paramTree.neverSynced');
          return (
            <Tooltip title={record.lastParamSyncAt ? t('device.paramSync.lastSyncedAt', { time: fmtTime(record.lastParamSyncAt) }) : syncText}>
              <span
                style={{
                  display: 'inline-block',
                  maxWidth: 150,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  verticalAlign: 'middle',
                }}
              >
                {syncText}
              </span>
            </Tooltip>
          );
        },
      },
      {
        key: 'onlineDuration',
        title: t('device.onlineDuration'),
        dataIndex: 'onlineDuration',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtDuration(onlineDurationOf(record)),
      },
      {
        key: 'cumulativeOnlineDuration',
        title: t('device.cumulativeOnlineDuration'),
        dataIndex: 'cumulativeOnlineDuration',
        width: 140,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtDuration(cumulativeOnlineDurationOf(record)),
      },
      {
        key: 'offlineDuration',
        title: t('device.offlineDuration'),
        width: 120,
        group: 'common',
        render: (_val, record) => {
          // 仅离线设备显示
          if (record.connStatus !== 'offline') return '-';
          return formatOfflineDurationFromSeconds(t, offlineDurationOf(record));
        },
      },
      { key: 'upTime', title: t('device.upTime'), dataIndex: 'upTime', width: 120, hidden: true, group: 'common', render: (_val, record) => fmtDuration(record.upTime) },
      {
        key: 'opState',
        title: t('device.opState'),
        dataIndex: 'opState',
        width: 140,
        group: 'common',
        // 激活状态 = 设备是否曾首次上线（op_state），与在线/小区状态正交。
        render: (_val, record) => renderActivationStatus(record.opState, record.isOnline),
      },
      {
        key: 'mmeStatus',
        title: t('device.mmeStatus'),
        dataIndex: 'mmeStatus',
        width: 130,
        group: 'common',
        render: (_val, record) => {
          const messageId = mmeStatusMessageIdForDevice(record);
          return messageId ? t(messageId) : '-';
        },
      },
      {
        key: 'amfStatus',
        title: t('device.amfStatus'),
        dataIndex: 'amfStatus',
        width: 130,
        group: 'common',
        render: (_val, record) => {
          const messageId = amfStatusMessageIdForDevice(record);
          return messageId ? t(messageId) : '-';
        },
      },
      {
        key: 'bscLinkStatus',
        title: t('device.bscLinkStatus'),
        dataIndex: 'bscLinkStatus',
        width: 140,
        group: 'common',
        render: (_val, record) => {
          const messageId = bscLinkStatusMessageIdForDevice(record);
          return messageId ? t(messageId) : '-';
        },
      },
      {
        key: 'ueCount',
        title: t('device.ueCount'),
        dataIndex: 'ueCount',
        width: 80,
        group: 'common',
        // JSP 行为: eNB >0 可点击(跳转 UE 详情页)；gNB/GSM 不可点击
        render: (_val, record) => {
          const v = record.ueCount;
          if (v === -1 || v === null || v === undefined) return '--';
          if (v === 0) return '0';
          // 仅 eNB 支持点击跳转 UE 详情页
          const isEnb = record.networkType === 'eNB';
          if (isEnb) {
            return (
              <Link onClick={() => navigate(`/device/ue-detail/${record.sn}?name=${encodeURIComponent(record.name || record.sn)}&ueCount=${record.ueCount}`)}>
                {v}
              </Link>
            );
          }
          return String(v);
        },
      },
      {
        key: 'rfStatus',
        title: t('device.rfStatus'),
        dataIndex: 'rfStatus',
        width: 150,
        hidden: true,
        group: 'common',
        // 原始 JSP: 支持多小区 "on,off,on"，汇总 + [N/M] Popover
        render: (_val, record) => {
          const rfStatus = formatDeviceRFStatus(record, record.rfStatus);
          if (rfStatus === '-') return '-';
          return renderMultiCellStatus(
            rfStatus,
            ['on', '1', '3'],
            { on: t('status.rfOn'), off: t('status.rfOff'), title: t('device.multiCellStatus') },
            { on: 'success', off: 'error', mixed: 'warning' },
          );
        },
      },
      {
        key: 'syncStatus',
        title: t('device.syncStatus'),
        dataIndex: 'syncStatus',
        width: 130,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          const normalized = normalizeDeviceSyncStatus(record.syncStatus);
          if (!normalized) return '-';
          const kind = getDeviceSyncStatusKind(normalized);
          if (!kind) return normalized;
          return (
            <Tag color={kind === 'error' ? 'error' : 'success'} style={{ fontWeight: 600 }}>
              {formatDeviceSyncStatus(normalized, {
                synchronized: t('status.synchronized'),
                gps: `GPS ${t('status.synchronized')}`,
                beidou: t('status.beidouSynchronized'),
                ntp: `NTP/1588 ${t('status.synchronized')}`,
                rem: `REM ${t('status.synchronized')}`,
                error: t('status.notSynchronized'),
              })}
            </Tag>
          );
        },
      },
      {
        key: 'siteName',
        title: t('device.cellName'),
        dataIndex: 'cellId',
        width: 130,
        hidden: true,
        ellipsis: true,
        group: 'common',
        // cellId 为空/null/空串 → 显占位符 '--'(绝不回退显 SN/设备编码,#184)。
        render: (val) => {
          const v = val as string | null | undefined;
          return v == null || v === '' ? '--' : v;
        },
      },
      // Remark 列暂时隐藏（用户反馈：含义不明 + 表头自定义编辑能力暂未对接后端持久化）。
      // 恢复方式：取消下行注释并保留 remarkHeaderRender / handleRemarkLabel* state（已保留）。
      // { key: 'remark', title: t('device.remark'), dataIndex: 'remark', width: 185, hidden: true, ellipsis: true, group: 'common', headerRender: remarkHeaderRender },
      {
        key: 'longitude',
        title: t('device.longitude'),
        dataIndex: 'longitude',
        width: 130,
        hidden: true,
        group: 'common',
        render: (_val, record) => renderLocationCell(record.longitude, record),
      },
      {
        key: 'latitude',
        title: t('device.latitude'),
        dataIndex: 'latitude',
        width: 130,
        hidden: true,
        group: 'common',
        render: (_val, record) => renderLocationCell(record.latitude, record),
      },
      {
        key: 'gpsHeight',
        title: t('device.gpsHeight'),
        dataIndex: 'gpsHeight',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => renderLocationCell(record.gpsHeight, record),
      },
      {
        key: 'gpsSatelliteCount',
        title: t('device.gpsSatelliteCount'),
        dataIndex: 'gpsSatelliteCount',
        width: 110,
        hidden: true,
        group: 'common',
        // 原始 JSP: 有卫星详情时可点击查看信号表（卫星号、信号强度）
        render: (_val, record) => {
          const v = record.gpsSatelliteCount;
          if (v === null || v === undefined) return '-';
          // TODO: 判断 hasSatelliteDetail 并点击打开卫星详情面板 (getSatellitesDataList.action)
          return v > 0
            ? (
              <Link
                onMouseEnter={() => prefetchDeviceDetailEntry(record)}
                onFocus={() => prefetchDeviceDetailEntry(record)}
                onClick={() => openDeviceDetail(record, 'gps')}
              >
                {v}
              </Link>
            )
            : String(v);
        },
      },
      {
        key: 'installAddress',
        title: t('device.installAddress'),
        dataIndex: 'installAddress',
        width: 180,
        hidden: true,
        ellipsis: true,
        group: 'common',
        render: (_val, record) => {
          const isEditing = editingInstallAddressId === record.id;
          const isSaving = savingInstallAddressId === record.id;
          const isEmptyAddress = !record.installAddress;
          if (isEditing) {
            return (
              <Input
                size="small"
                value={editingInstallAddressValue}
                autoFocus
                maxLength={256}
                placeholder={t('device.installAddress')}
                onClick={(e) => e.stopPropagation()}
                onChange={(e) => setEditingInstallAddressValue(e.target.value)}
                onPressEnter={() => saveInstallAddressEdit(record)}
                onBlur={() => saveInstallAddressEdit(record)}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    e.preventDefault();
                    cancelInstallAddressEdit();
                  }
                }}
                suffix={isSaving ? <ReloadOutlined spin /> : null}
              />
            );
          }

          return (
            <div
              title={record.installAddress || t('device.installAddress.doubleClickEdit')}
              onDoubleClick={(e) => {
                e.stopPropagation();
                startInstallAddressEdit(record);
              }}
              style={{
                cursor: 'text',
                minHeight: 22,
                color: isEmptyAddress ? 'rgba(0, 0, 0, 0.45)' : undefined,
              }}
            >
              {record.installAddress || t('device.installAddress.doubleClickEdit')}
            </div>
          );
        },
      },
      { key: 'pci', title: 'PCI', dataIndex: 'pci', width: 80, hidden: true, group: 'common', render: (value, record) => formatDeviceRadioField(record, 'pci', value) },
      { key: 'tac', title: 'TAC', dataIndex: 'tac', width: 80, hidden: true, group: 'common', render: (value, record) => formatDeviceRadioField(record, 'tac', value) },
      { key: 'band', title: 'Band', dataIndex: 'band', width: 100, hidden: true, group: 'common', render: (value, record) => formatDeviceRadioField(record, 'band', value) },
      { key: 'dlEarfcn', title: t('device.dlEarfcn'), dataIndex: 'dlEarfcn', width: 110, hidden: true, group: 'common', render: (value, record) => formatDeviceRadioField(record, 'dlEarfcn', value) },
      { key: 'ulEarfcn', title: t('device.ulEarfcn'), dataIndex: 'ulEarfcn', width: 110, hidden: true, group: 'common', render: (value, record) => formatDeviceRadioField(record, 'ulEarfcn', value) },
      // 基站类型(networkModel)暂时隐藏：当前 LTE/NR/双模 推导口径未与产品对齐;恢复时取消下行注释。
      // { key: 'networkModel', title: t('device.networkModel'), dataIndex: 'networkModel', width: 110, hidden: true, group: 'common' },
      { key: 'txPower', title: 'Tx Power', dataIndex: 'txPower', width: 100, hidden: true, group: 'common', render: (value, record) => formatDeviceRadioField(record, 'txPower', value) },
      {
        key: 'halobFlag',
        title: 'HaloB',
        dataIndex: 'halobFlag',
        width: 90,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          if (record.halobFlag === undefined || record.halobFlag === null) return '-';
          return <Tag color={record.halobFlag ? 'success' : 'default'}>{record.halobFlag ? t('status.enabled') : t('status.disabled')}</Tag>;
        },
      },
      {
        key: 'adminState',
        title: 'Admin State',
        dataIndex: 'adminState',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtStatus(record.adminState, adminStateStatusMap),
      },
      { key: 'ipsecAddr', title: t('device.ipsecAddr'), dataIndex: 'ipsecAddr', width: 140, hidden: true, mono: true, group: 'common' },
    ],
    // remarkHeaderRender 暂从 dep 列表移除：remark 列定义已注释，恢复时同步加回。
    [navigate, openDeviceDetail, prefetchDeviceDetailEntry, t, fmtTime, fmtDuration, fmtStatus, adminStateStatusMap, renderMultiCellStatus, renderActivationStatus, message, mapConnStatus, getSeverityLabel, networkTypeDict?.sysDictionaryDetails, editingInstallAddressId, editingInstallAddressValue, savingInstallAddressId, saveInstallAddressEdit, cancelInstallAddressEdit, startInstallAddressEdit, optimisticParamSyncDeviceIds, optimisticAlarmSyncDeviceIds, renderLocationCell]
  );

  // ─── 列表导出(用户决策 2026-06-02) ──────────────────────────────────────
  // 1) 仅导出当前筛选条件命中的全部数据(跨分页拉全量,受 EXPORT_ROW_CAP 上限保护);
  // 2) 导出列与列顺序严格以"列设置"(ColumnVisibility)为准 —— 即与表格当前展示一致。

  // 单元格取值：与列 render 的可读文案对齐;特殊列(状态/告警/时间/时长/枚举)单独
  // 格式化,其余列回退到 record[dataIndex] 原值(数组以逗号拼接,空值 → 空串)。
  const formatExportCell = useCallback(
    (record: Device, key: string, dataIndex?: string): string => {
      switch (key) {
        case 'connStatus':
          return mapConnStatus(record.connStatus) === 'online' ? t('status.online') : t('status.offline');
        case 'lastParamSyncAt':
          return record.lastParamSyncAt ? fmtTime(record.lastParamSyncAt) : t('device.paramTree.neverSynced');
        case 'alarmLevel':
          return getSeverityLabel(record.alarmLevel);
        case 'onlineTime':
          return fmtTime(record.onlineTime);
        case 'offlineTime':
          return fmtTime(record.offlineTime);
        case 'firstOnlineTime':
          return fmtTime(record.firstOnlineTime);
        case 'lastOnlineTime':
          return fmtTime(record.lastOnlineTime);
        case 'onlineDuration':
          return fmtDuration(onlineDurationOf(record));
        case 'cumulativeOnlineDuration':
          return fmtDuration(cumulativeOnlineDurationOf(record));
        case 'offlineDuration':
          return record.connStatus === 'offline'
            ? offlineDurationTextFromSeconds(t, offlineDurationOf(record))
            : '-';
        case 'halobFlag':
          return record.halobFlag == null
            ? '-'
            : record.halobFlag
              ? t('status.enabled')
              : t('status.disabled');
        case 'adminState': {
          return adminStateLabelOf(record.adminState);
        }
        case 'ueCount': {
          const v = record.ueCount;
          return v === -1 || v == null ? '--' : String(v);
        }
        case 'opState': {
          return displayActivationStatusLabelOf(record.opState, record.isOnline, opStateDict?.sysDictionaryDetails, {
            active: t('status.active'),
            inactive: t('status.inactive'),
          }, appLocale) || '-';
        }
        case 'rfStatus':
          return record.rfStatus || '';
        case 'pci':
        case 'tac':
        case 'band':
        case 'dlEarfcn':
        case 'ulEarfcn':
        case 'txPower':
          return formatDeviceRadioField(record, key, record[key]);
        default: {
          const v = dataIndex ? (record as unknown as Record<string, unknown>)[dataIndex] : undefined;
          if (Array.isArray(v)) return v.join(', ');
          return v == null ? '' : String(v);
        }
      }
    },
    [adminStateLabelOf, appLocale, mapConnStatus, getSeverityLabel, fmtTime, fmtDuration, onlineDurationOf, cumulativeOnlineDurationOf, offlineDurationOf, opStateDict?.sysDictionaryDetails, t]
  );

  // 按当前筛选条件并发分页拉取全部命中数据(不受列表当前页/页大小限制)。
  // 全表无筛选时可达 3W+ 行,串行逐页会慢到像"点了没反应",故用并发池 + 进度回调。
  const fetchAllFilteredDevices = useCallback(
    (onProgress?: (loaded: number, total: number) => void) =>
      fetchAllPaged<Device>(
        async (page, pageSize) => {
          const resp = await exportDeviceApi.getList({
            ...filterParams,
            page,
            pageSize,
          } as Parameters<typeof useDeviceList>[0]);
          return { items: resp.items, total: resp.total ?? resp.items.length };
        },
        { onProgress },
      ),
    [filterParams]
  );

  // 导出 — 选择格式(xlsx/csv)后触发：筛选生效 + 列以"列设置"为准。
  const handleExport = useCallback(
    async (format: 'xlsx' | 'csv') => {
      setExportModalOpen(false);
      const exportCols = resolveVisibleExportColumns(columns, DEVICE_LIST_TABLE_ID);
      if (exportCols.length === 0) {
        void message.warning(t('common.noColumnsToExport'));
        return;
      }
      const msgKey = 'device-list-export';
      message.open({ key: msgKey, type: 'loading', content: t('common.exportInProgress'), duration: 0 });
      try {
        // 进度更新做节流(全表 34 页会在 ~1s 内回调 34 次):同一 key 高频 message.open
        // 在部分机器上会出现"内容还没渲染就被下一次替换"的空白闪烁。限制为最多每 300ms
        // 刷新一次,并始终渲染最后一次(loaded===total),既给反馈又不抖动。
        let lastProgressAt = 0;
        const { items: rowsData, capped } = await fetchAllFilteredDevices((loaded, total) => {
          const now = Date.now();
          if (loaded < total && now - lastProgressAt < 300) return;
          lastProgressAt = now;
          const content = t('common.exportProgress', { loaded, total }) || `${loaded}/${total}`;
          message.open({ key: msgKey, type: 'loading', content, duration: 0 });
        });
        if (rowsData.length === 0) {
          message.open({ key: msgKey, type: 'warning', content: t('common.noDataToExport') });
          return;
        }
        const headers = exportCols.map((c) => c.title);
        const matrix = rowsData.map((dev) => exportCols.map((c) => formatExportCell(dev, c.key, c.dataIndex)));
        const filename = `device-list-${exportTimestamp()}.${format}`;
        if (format === 'csv') {
          triggerCsvDownload(buildCsvContent(headers, matrix), filename);
        } else {
          triggerXlsxDownload(headers, matrix, filename, 'devices');
        }
        // 命中上限时显式告知用户被截断,避免"以为导全了实际只导了一部分"。
        message.open({
          key: msgKey,
          type: capped ? 'warning' : 'success',
          content: capped
            ? t('common.exportCapped', { count: rowsData.length })
            : t('common.exportSuccess', { count: rowsData.length }),
        });
      } catch (e) {
        message.open({
          key: msgKey,
          type: 'error',
          content: t('common.exportFailed', { reason: e instanceof Error ? e.message : String(e) }),
        });
      }
    },
    [columns, message, t, fetchAllFilteredDevices, formatExportCell]
  );

  const handleSavePeriodicSync = useCallback(async () => {
    if (periodicSyncLoading) {
      void message.warning(t('common.loading'));
      return;
    }
    let values: Record<string, unknown>;
    try {
      values = (await periodicSyncForm.validateFields()) as Record<string, unknown>;
    } catch {
      void message.error(t('common.formValidationFailed'));
      return;
    }
    try {
      await batchUpdateSysConfigs.mutateAsync({
        category: 'device',
        items: buildBatchItems(values, periodicSyncConfigs),
      });
      if (values.periodicSyncEnabled === true) {
        setPeriodicSyncWatchUntil(Date.now() + PERIODIC_SYNC_WATCH_MS);
        void refetch();
      }
      void message.success(t('device.periodicParamSync.saveSuccess'));
      setPeriodicSyncModalOpen(false);
    } catch (err) {
      void message.error(err instanceof Error ? err.message : t('sysconfig.error.saveFailed'));
    }
  }, [batchUpdateSysConfigs, message, periodicSyncConfigs, periodicSyncForm, periodicSyncLoading, refetch, t]);

  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'batch-reboot',
      label: t('common.batchReboot'),
      icon: <ReloadOutlined />,
      onClick: (keys) => handleBatchAction(t('common.batchReboot'), keys, 'batch-reboot'),
    },
    // batch-tr069-collect 批量按钮已移除（#179）：按需抓包统一以「运维管理 - TR069
    // 报文跟踪」菜单为唯一入口；设备页该入口冗余且对 NAT 后设备主动呼叫超时易误触失败。
    {
      key: 'batch-log-collect',
      label: t('device.action.logCollect'),
      icon: <FileTextOutlined />,
      onClick: (keys) => handleBatchAction(t('device.action.logCollect'), keys, 'batch-log-collect'),
    },
    {
      key: 'batch-alarm-sync',
      label: t('device.action.alarmSync'),
      icon: <AlertOutlined />,
      disabled: batchAlarmSyncRunning,
      onClick: (keys) => handleBatchAction(t('device.action.alarmSync'), keys, 'batch-alarm-sync'),
    },
    {
      key: 'batch-param-sync',
      label: t('device.action.paramSync'),
      icon: <SyncOutlined />,
      disabled: batchParamSyncRunning,
      onClick: (keys) => handleBatchAction(t('device.action.paramSync'), keys, 'batch-param-sync'),
    },
    // 恢复默认配置已隐藏
    // {
    //   key: 'batch-reset-config',
    //   label: t('device.action.resetConfig'),
    //   icon: <ExclamationCircleOutlined />,
    //   danger: true,
    //   onClick: (keys) => handleBatchAction(t('device.action.resetConfig'), keys, 'batch-reset-config'),
    // },
  ], [batchAlarmSyncRunning, batchParamSyncRunning, handleBatchAction, t]);

  // 任务面板表格列定义
  const taskColumns: ColumnsType<LocalTask> = useMemo(() => [
    {
      title: 'SN',
      dataIndex: 'sn',
      key: 'sn',
      width: 140,
      ellipsis: true,
    },
    {
      title: t('alarm.deviceName'),
      dataIndex: 'deviceName',
      key: 'deviceName',
      ellipsis: true,
      width: 120,
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: TaskStatus) => {
        const statusConfig: Record<TaskStatus, { color: string; text: string }> = {
          pending: { color: 'default', text: t('status.pending') },
          running: { color: 'processing', text: t('task.status.running') },
          success: { color: 'success', text: t('task.status.completed') },
          failed: { color: 'error', text: t('task.status.failed') },
        };
        const cfg = statusConfig[status];
        return (
          <Tag color={cfg.color} style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
            {cfg.text}
          </Tag>
        );
      },
    },
    {
      title: t('task.progress'),
      dataIndex: 'progress',
      key: 'progress',
      width: 120,
      render: (progress: number, record) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <Progress
            percent={Math.round(progress)}
            size="small"
            status={record.status === 'failed' ? 'exception' : record.status === 'success' ? 'success' : 'active'}
            showInfo={false}
            style={{ flex: 1, minWidth: 60 }}
          />
          <span style={{ fontSize: 11, color: 'var(--color-neutral-600)', whiteSpace: 'nowrap' }}>
            {Math.round(progress)}%
          </span>
        </div>
      ),
    },
  ], [t, handleViewLog]);

  // R3: 导出按钮挪到 FilterBar 搜索按钮右侧（device-list-and-group-improvements-20260520.md R3）。
  // ListPageLayout.extra 不再承载，让筛选区与导出动作在视觉上一行对齐。
  const exportButton = (
    <Button
      type="primary"
      icon={<ExportOutlined />}
      onClick={() => setExportModalOpen(true)}
    >
      {t('common.export')}
    </Button>
  );

  // 导出确认弹窗：用 antd Modal（主题感知，不会再白底白字）。固定 CSV 格式，
  // 用户决策 2026-06-02：导出只支持 CSV，不再提供格式选择。
  const exportConfirmModal = (
    <Modal
      open={exportModalOpen}
      title={t('common.exportConfirmTitle')}
      onCancel={() => setExportModalOpen(false)}
      onOk={() => handleExport('csv')}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
    >
      {t('common.exportConfirmContent')}
    </Modal>
  );

  const periodicSyncModal = (
    <Modal
      open={periodicSyncModalOpen}
      title={t('device.periodicParamSync.title')}
      onCancel={() => setPeriodicSyncModalOpen(false)}
      onOk={() => void handleSavePeriodicSync()}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={batchUpdateSysConfigs.isPending}
      okButtonProps={{ disabled: periodicSyncLoading }}
      destroyOnHidden
    >
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16, fontSize: 12 }}>
        {t('device.periodicParamSync.desc')}
      </Typography.Paragraph>
      <Form
        form={periodicSyncForm}
        layout="vertical"
        size="small"
        disabled={periodicSyncLoading || batchUpdateSysConfigs.isPending}
        initialValues={PERIODIC_PARAM_SYNC_DEFAULTS}
      >
        <Form.Item name="periodicSyncEnabled" valuePropName="checked">
          <Checkbox>{t('system.device.periodicSync.enabledLabel')}</Checkbox>
        </Form.Item>
        <Form.Item label={t('system.device.periodicSync.intervalPrefix')} name="periodicSyncIntervalMinutes" rules={[{ required: true, type: 'number', min: 1, max: 10080 }]}>
          <InputNumber min={1} max={10080} addonAfter={t('device.periodicParamSync.minuteUnit')} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item label={t('system.device.periodicSync.batchSizePrefix')} name="periodicSyncBatchSize" rules={[{ required: true, type: 'number', min: 1, max: 1000 }]}>
          <InputNumber min={1} max={1000} addonAfter={t('device.periodicParamSync.deviceUnit')} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item label={t('system.device.periodicSync.maxConcurrentPrefix')} name="periodicSyncMaxConcurrent" rules={[{ required: true, type: 'number', min: 1, max: 50 }]}>
          <InputNumber min={1} max={50} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item label={t('system.device.periodicSync.staggerPrefix')} name="periodicSyncStaggerWindowMinutes" rules={[{ required: true, type: 'number', min: 0, max: 120 }]} style={{ marginBottom: 0 }}>
          <InputNumber min={0} max={120} addonAfter={t('device.periodicParamSync.minuteUnit')} style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Modal>
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      <div style={{ flex: '1 1 100%', minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <ListPageLayout>
          <FilterBar
            filterId="device-list"
            fields={visibleFilterFields}
            onSearch={handleSearch}
            onReset={handleReset}
            collapsedRows={1}
            initialValues={filterParams}
            extra={exportButton}
          />

          <StatisticsPanel items={statsItems} style={{ marginBottom: 8 }} />

          {/* 设备列表卡片。
              注意:不要在外层套 <Spin>。Antd Spin 内部的
              .ant-spin-nested-loading + .ant-spin-container 是 display:block
              不是 flex,会断 ListPageLayout → Card 的 flex 高度链,导致 Card
              收缩到内容自然高度(~161px),DataTable.autoFitHeight 算出 y=1px,
              所有列表行被压到 1px 高度内不可见(分页栏看似贴上来盖住列表)。
              DataTable 自己已接 loading={isLoading},无需外层 Spin。 */}
          <Card
            size="small"
            variant="outlined"
            style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
            styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
          >
            <DataTable<Device>
              key={tableMigrationVersion}
              tableId="device-list-table"
              columns={columns}
              onHiddenColumnsChange={setHiddenColumnKeys}
              dataSource={devices}
              loading={isLoading}
              rowKey="id"
              selectable
              selectedRowKeys={selectedRowKeys}
              onSelectionChange={(keys) => setSelectedRowKeys(keys)}
              getCheckboxProps={(record) => ({
                disabled: !record.isOnline,
              })}
              total={total}
              pageSize={pageSize}
              currentPage={currentPage}
              onPageChange={(page, size) => {
                setCurrentPage(page);
                setPageSize(size);
              }}
              batchActions={batchActions}
              onRefresh={handleManualRefresh}
              extraToolbarAfterBatch={(
                <Button
                  size="small"
                  icon={<ClockCircleOutlined />}
                  onClick={() => setPeriodicSyncModalOpen(true)}
                >
                  {t('device.action.autoParamSync')}
                </Button>
              )}
              extraToolbarRight={(
                <Space size={8}>
                  <Button
                    size="small"
                    icon={<ReloadOutlined />}
                    loading={refreshSpinnerActive}
                    onClick={handleManualRefresh}
                  >
                    {t('common.refresh')}
                  </Button>
                  <AutoRefreshDropdown
                    enabled={autoRefresh}
                    intervalSeconds={refreshInterval}
                    onEnabledChange={setAutoRefresh}
                    onIntervalChange={setRefreshInterval}
                    spinning={autoRefresh && refreshSpinnerActive}
                    size="small"
                  />
                </Space>
              )}
              defaultDensity="default"
              scroll={{ x: 'max-content' }}
              autoFitHeight
              hideRealtime
              hideRefresh
              showRowNumber
              rowNumberTitle={t('table.rowNumber')}
            />
          </Card>
        </ListPageLayout>
      </div>

      {/* 收集任务抽屉 */}
      <Drawer
        title={collectDrawerTitle || t('task.collectProgress')}
        placement="right"
        size={480}
        open={collectDrawerOpen}
        onClose={() => setCollectDrawerOpen(false)}
        styles={{
          body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
        }}
      >
        {/* 任务统计 */}
        <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--color-border)', flexShrink: 0 }}>
          <Space size={16}>
            <span>
              {t('task.total')}: {collectTasks.length}
            </span>
            <span style={{ color: 'var(--color-primary-600)' }}>
              {t('task.status.running')}: {collectTasks.filter((item) => item.status === 'running').length}
            </span>
            <span style={{ color: '#52c41a' }}>
              {t('task.status.completed')}: {collectTasks.filter((item) => item.status === 'success').length}
            </span>
            <span style={{ color: '#ff4d4f' }}>
              {t('task.status.failed')}: {collectTasks.filter((item) => item.status === 'failed').length}
            </span>
          </Space>
        </div>

        {/* 任务列表 */}
        <div style={{ flex: 1, overflow: 'hidden', padding: 8 }}>
          <Table<LocalTask>
            dataSource={collectTasks}
            columns={taskColumns}
            rowKey="id"
            size="small"
            pagination={false}
            scroll={{ x: 630, y: 'calc(100vh - 180px)' }}
            locale={{ emptyText: t('common.noData') }}
            style={{ fontSize: 12 }}
          />
        </div>
      </Drawer>

      {/* 日志详情弹窗 */}
      <Modal
        title={t('task.logDetail')}
        open={logModalOpen}
        onCancel={() => setLogModalOpen(false)}
        footer={[
          <Button key="close" onClick={() => setLogModalOpen(false)}>
            {t('common.close')}
          </Button>,
        ]}
        width={640}
      >
        {currentLogTask && (
          <div>
            <div style={{ marginBottom: 12, display: 'flex', gap: 16, fontSize: 13 }}>
              <span><strong>SN:</strong> <code style={{ fontFamily: 'monospace' }}>{currentLogTask.sn}</code></span>
              <span><strong>{t('alarm.deviceName')}:</strong> {currentLogTask.deviceName}</span>
              <span>
                <strong>{t('table.status')}:</strong>{' '}
                <Tag color={currentLogTask.status === 'success' ? 'success' : 'error'} style={{ marginLeft: 4 }}>
                  {currentLogTask.status === 'success' ? t('task.status.completed') : t('task.status.failed')}
                </Tag>
              </span>
            </div>
            <div
              style={{
                backgroundColor: '#1e1e1e',
                color: '#d4d4d4',
                padding: 16,
                borderRadius: 6,
                fontFamily: 'monospace',
                fontSize: 12,
                lineHeight: 1.6,
                maxHeight: 400,
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
              }}
            >
              {currentLogTask.logContent || t('common.noData')}
            </div>
          </div>
        )}
      </Modal>

      {exportConfirmModal}
      {periodicSyncModal}
      <GpsSyncConfirmModal
        open={gpsSyncConfirmDevice != null}
        device={gpsSyncConfirmDevice}
        onCancel={() => setGpsSyncConfirmDevice(null)}
        onConfirm={() => {
          const device = gpsSyncConfirmDevice;
          setGpsSyncConfirmDevice(null);
          if (device) void acceptLocationSync(device);
        }}
      />
    </div>
  );
}
