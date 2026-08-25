import React, { useCallback, useEffect, useMemo, useReducer, useRef, useState } from 'react';
import { Badge, Button, Card, Dropdown, Space, Tag, App } from 'antd';
import {
  CheckOutlined,
  ClearOutlined,
  ExportOutlined,
  MoreOutlined,
  EyeOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';

import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { alarmService } from '@core/mock/services/alarmService';
import { deviceService } from '@core/mock/services/deviceService';
import { useCurrentAlarms, useAcknowledgeAlarms, useAlarmById, useClearAlarms, useAlarmCount, useMarkAlarmRead, useUnacknowledgeAlarms } from '@core/hooks/api/useAlarms';
import { alarmApi } from '@core/services/api/alarmApi';
import { deviceApi } from '@core/services/api/deviceApi';
import { createApiSwitch } from '@core/services/apiSwitch';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import type { AlarmFilter } from '@core/types/alarm';
import { parseAlarmId, parseDrillDownParams, withoutAlarmId } from '../drillDown';
import { useSearchParams } from 'react-router-dom';
import AlarmDetail from '../AlarmDetail';
import ExportModal from './ExportModal';
import AutoRefreshDropdown from '../components/AutoRefreshDropdown';
import ConfirmWithNoteModal from '../components/ConfirmWithNoteModal';
import { useAlarmListExport } from '../hooks/useAlarmListExport';
import { buildAlarmExportFieldDefinitions, type AlarmExportFieldKey } from '../utils/alarmExportFields';
import { getBaseStationTypeOptions, formatBaseStationTypeLabel } from '../utils/baseStationType';
import { renderSnWithTooltip } from '../utils/snTooltip';
import styles from './CurrentAlarms.module.css';
import { formatSystemTime } from '@core/utils/systemTime';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';

// 告警级别颜色 - 专业配色方案
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  critical: { color: '#E53935', bgColor: '#FFEBEE' },
  major: { color: '#FB8C00', bgColor: '#FFF3E0' },
  minor: { color: '#FDD835', bgColor: '#FFFDE7' },
  warning: { color: '#42A5F5', bgColor: '#E3F2FD' },
};

// 告警状态配置 - 四种状态使用不同颜色区分
const DEAL_STATE_CONFIG: Record<DealState, { label: string; color: string; icon: string }> = {
  '0': { label: 'alarm.dealState.unconfirmedUncleared', color: '#E53935', icon: 'unconfirmInactive' },
  '1': { label: 'alarm.dealState.confirmedUncleared', color: '#FB8C00', icon: 'confirmInactive' },
  '2': { label: 'alarm.dealState.unconfirmedCleared', color: '#42A5F5', icon: 'unconfirmActive' },
  '3': { label: 'alarm.dealState.confirmedCleared', color: '#67D972', icon: 'confirmActive' },
};

// 事件类型配置
const EVENT_TYPE_CONFIG: Record<EventType, string> = {
  communication: 'alarm.eventType.communication',
  qualityOfService: 'alarm.eventType.qualityOfService',
  processingError: 'alarm.eventType.processingError',
  device: 'alarm.eventType.device',
  environment: 'alarm.eventType.environment',
  performance: 'alarm.eventType.performance',
};

// 自动刷新间隔选项 - 移至组件内 useMemo

const exportAlarmApi: typeof alarmApi = createApiSwitch(
  alarmService as unknown as typeof alarmApi,
  alarmApi,
);
const exportDeviceApi = createApiSwitch(deviceService as unknown as typeof deviceApi, deviceApi);
const EXPORT_PAGE_SIZE = 500;

function escapeCsvCell(value: unknown): string {
  const normalized = value == null ? '' : String(value);
  const escaped = normalized.replace(/"/g, '""');
  return `"${escaped}"`;
}

function triggerCsvDownload(content: string, filename: string) {
  const blob = new Blob(['\ufeff' + content], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

interface AlarmListState {
  currentPage: number;
  filterParams: AlarmFilter;
}

type AlarmListAction =
  | { type: 'applyDrillDown'; filter: AlarmFilter }
  | { type: 'search'; filter: AlarmFilter }
  | { type: 'reset' }
  | { type: 'page'; page: number };

function alarmListReducer(state: AlarmListState, action: AlarmListAction): AlarmListState {
  switch (action.type) {
    case 'applyDrillDown':
    case 'search':
      return { ...state, currentPage: 1, filterParams: action.filter };
    case 'reset':
      return { ...state, currentPage: 1, filterParams: {} };
    case 'page':
      return { ...state, currentPage: action.page };
    default:
      return state;
  }
}

export default function CurrentAlarms() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [searchParams, setSearchParams] = useSearchParams();
  const drillDown = useMemo(() => parseDrillDownParams(searchParams), [searchParams]);
  const deepLinkAlarmId = useMemo(() => parseAlarmId(searchParams), [searchParams]);
  const deepLinkNoticeRef = useRef('');
  const [pageSize, setPageSize] = useState(20);
  const [{ currentPage, filterParams }, dispatchList] = useReducer(alarmListReducer, drillDown.filter, (filter) => ({
    currentPage: 1,
    filterParams: filter,
  }));
  // 有钻取参数时把表单初始值传给 FilterBar（无则传 undefined，保留 sessionStorage 恢复行为）
  const drillDownInitialValues = useMemo(
    () => (Object.keys(drillDown.formValues).length > 0 ? drillDown.formValues : undefined),
    [drillDown]
  );
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const deepLinkAlarm = useAlarmById(deepLinkAlarmId ?? '');
  const activeDetailAlarm = deepLinkAlarm.data ?? detailAlarm;
  const activeDetailOpen = detailOpen || Boolean(deepLinkAlarm.data);

  // 确认告警弹窗状态
  const [ackModalOpen, setAckModalOpen] = useState(false);
  const [ackTargetIds, setAckTargetIds] = useState<string[]>([]);
  const [ackLoading, setAckLoading] = useState(false);

  // 清除告警弹窗状态
  const [clearModalOpen, setClearModalOpen] = useState(false);
  const [clearTargetIds, setClearTargetIds] = useState<string[]>([]);
  const [clearLoading, setClearLoading] = useState(false);

  // 自动刷新状态
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30);

  useEffect(() => {
    dispatchList({ type: 'applyDrillDown', filter: drillDown.filter });
  }, [drillDown]);

  useEffect(() => {
    if (!deepLinkAlarmId) {
      deepLinkNoticeRef.current = '';
      return;
    }
    if (deepLinkAlarm.data) return;
    const noticeKey = `${deepLinkAlarmId}:${deepLinkAlarm.isError ? 'error' : 'missing'}`;
    if (deepLinkNoticeRef.current === noticeKey) return;
    if (deepLinkAlarm.isError) {
      deepLinkNoticeRef.current = noticeKey;
      const status = (deepLinkAlarm.error as { response?: { status?: number } })?.response?.status;
      void message.error(t(status === 403 ? 'common.noPermission' : 'alarm.detailLoadFailed'));
    } else if (deepLinkAlarm.isFetched) {
      deepLinkNoticeRef.current = noticeKey;
      void message.warning(t('alarm.noLongerAvailable'));
    }
  }, [deepLinkAlarm.data, deepLinkAlarm.error, deepLinkAlarm.isError, deepLinkAlarm.isFetched, deepLinkAlarmId, message, t]);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const exportFields = useMemo(() => buildAlarmExportFieldDefinitions(t), [t]);
  const exportFieldOptions = useMemo(() => exportFields.map(({ key, label }) => ({ key, label })), [exportFields]);
  const defaultExportFieldKeys = useMemo(() => exportFields.map(({ key }) => key), [exportFields]);
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();
  const deviceStandardOptions = useMemo(
    () => getBaseStationTypeOptions(systemLicense, systemLicenseLoading),
    [systemLicense, systemLicenseLoading],
  );

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'deviceSn', label: t('alarm.deviceSn'), type: 'input', placeholder: t('alarm.searchSnPlaceholder'), width: 180 },
    {
      name: 'neType',
      label: t('alarm.neTypeCol'),
      type: 'select',
      width: 96,
      options: [
        { label: t('common.all'), value: '' },
        ...deviceStandardOptions,
      ],
    },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'multi-select',
      width: 120,
      options: [
        { label: t('alarm.severity.critical'), value: 'critical' },
        { label: t('alarm.severity.major'), value: 'major' },
        { label: t('alarm.severity.minor'), value: 'minor' },
        { label: t('alarm.severity.warning'), value: 'warning' },
      ],
    },
    { name: 'alarmIdentifier', label: t('alarm.alarmIdentifier'), type: 'input', width: 120 },
    {
      name: 'eventType',
      label: t('alarm.eventType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.eventType.communication'), value: 'communication' },
        { label: t('alarm.eventType.qualityOfService'), value: 'qualityOfService' },
        { label: t('alarm.eventType.processingError'), value: 'processingError' },
        { label: t('alarm.eventType.device'), value: 'device' },
        { label: t('alarm.eventType.environment'), value: 'environment' },
      ],
    },
    {
      name: 'dealState',
      label: t('alarm.dealState'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.dealState.unconfirmedUncleared'), value: '0' },
        { label: t('alarm.dealState.confirmedUncleared'), value: '1' },
      ],
    },
    { name: 'timeRange', label: t('alarm.eventTime'), type: 'date-range', showTime: true },
    {
      name: 'unread',
      label: t('alarm.readStatus'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.readStatus.read'), value: '0' },
        { label: t('alarm.readStatus.unread'), value: '1' },
      ],
    },
    {
      // 未识别告警过滤：识别状态 = identifier 是否在告警库中存在（设计 §3.3 治理闭环）
      name: 'isUnknown',
      label: t('alarm.recognizeStatus'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.recognizeStatus.unknown'), value: 'true' },
        { label: t('alarm.recognizeStatus.known'), value: 'false' },
      ],
    },
  ], [deviceStandardOptions, t]);

  const queryParams = useMemo(
    () => {
      const clean: Record<string, unknown> = { page: currentPage, pageSize };
      for (const [k, v] of Object.entries(filterParams)) {
        if (v !== undefined && v !== '') clean[k] = v;
      }
      return clean;
    },
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useCurrentAlarms(
    queryParams as unknown as Parameters<typeof useCurrentAlarms>[0],
    {
      refetchIntervalMs: autoRefresh ? refreshInterval * 1000 : false,
      refetchIntervalInBackground: autoRefresh,
      refetchOnWindowFocus: true,
    }
  );
  const acknowledgeAlarms = useAcknowledgeAlarms();
  const clearAlarms = useClearAlarms();
  const markAlarmRead = useMarkAlarmRead();
  const unacknowledgeAlarms = useUnacknowledgeAlarms();
  const { data: alarmCount } = useAlarmCount();

  const rawAlarms: Alarm[] = useMemo(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;

  // 开启自动刷新或切换间隔时立即拉一次，避免用户等待下一轮轮询。
  useEffect(() => {
    if (!autoRefresh) return;
    void refetch();
  }, [autoRefresh, refreshInterval, refetch]);

  // 未读告警排在最前面
  const alarms = useMemo(
    () => [...rawAlarms].sort((a, b) => {
      if (a.unread === '1' && b.unread !== '1') return -1;
      if (a.unread !== '1' && b.unread === '1') return 1;
      return 0;
    }),
    [rawAlarms]
  );

  // 统计数据来自后端 API
  const realStats = useMemo(() => ({
    total: alarmCount?.total_active ?? total,
    critical: alarmCount?.critical ?? 0,
    major: alarmCount?.major ?? 0,
    minor: alarmCount?.minor ?? 0,
    warning: alarmCount?.warning ?? 0,
    unacked: alarmCount?.unacknowledged ?? 0,
    unread: alarmCount?.unread ?? 0,
  }), [alarmCount, total]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    dispatchList({ type: 'search', filter: {
      deviceSn: values.deviceSn as string,
      severity: values.severity as AlarmFilter['severity'],
      alarmIdentifier: values.alarmIdentifier as string,
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      unread: values.unread as '0' | '1',
      dealState: values.dealState as AlarmFilter['dealState'],
      isUnknown: values.isUnknown as AlarmFilter['isUnknown'],
      timeRange: values.timeRange as [string, string] | undefined,
    } });
  }, []);

  const handleReset = useCallback(() => {
    dispatchList({ type: 'reset' });
  }, []);

  const handleAcknowledge = useCallback(
    (ids: string[]) => {
      setAckTargetIds(ids);
      setAckModalOpen(true);
    },
    []
  );

  const handleAcknowledgeConfirm = useCallback(
    async (note: string) => {
      setAckLoading(true);
      try {
        await acknowledgeAlarms.mutateAsync({ ids: ackTargetIds, note });
        setSelectedRowKeys([]);
        setAckModalOpen(false);
        refetch();
        message.success(t('common.ackSuccess'));
      } catch {
        message.error(t('common.ackFailed'));
      } finally {
        setAckLoading(false);
      }
    },
    [acknowledgeAlarms, ackTargetIds, refetch, t, message]
  );

  const handleUnacknowledge = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.unacknowledge'),
        content: t('common.unackConfirmMsg', { count: ids.length }),
        okText: t('common.confirm'),
        onOk: async () => {
          try {
            await unacknowledgeAlarms.mutateAsync(ids);
            setSelectedRowKeys([]);
            message.success(t('common.unackSuccess'));
          } catch {
            message.error(t('common.unackFailed'));
          }
        },
      });
    },
    [unacknowledgeAlarms, t, message, modal]
  );

  const handleClear = useCallback(
    (ids: string[]) => {
      setClearTargetIds(ids);
      setClearModalOpen(true);
    },
    []
  );

  const handleClearConfirm = useCallback(
    async (note: string) => {
      setClearLoading(true);
      try {
        await clearAlarms.mutateAsync({ ids: clearTargetIds, note });
        setSelectedRowKeys([]);
        setClearModalOpen(false);
        refetch();
        message.success(t('common.clearSuccess'));
      } catch {
        message.error(t('common.clearFailed'));
      } finally {
        setClearLoading(false);
      }
    },
    [clearAlarms, clearTargetIds, refetch, t, message]
  );

  const fetchDeviceSnsByGroups = useCallback(async (groupIds: string[]) => {
    const snSet = new Set<string>();

    await Promise.all(
      groupIds.map(async (groupId) => {
        let page = 1;

        while (true) {
          const response = await exportDeviceApi.getList({
            groupId,
            page,
            pageSize: EXPORT_PAGE_SIZE,
          });

          response.items.forEach((device) => {
            if (device.sn) {
              snSet.add(device.sn);
            }
          });

          if (response.items.length === 0 || response.page * response.pageSize >= response.total) {
            break;
          }

          page += 1;
        }
      })
    );

    return snSet;
  }, []);

  const fetchAllCurrentAlarmsForExport = useCallback(async (filters: AlarmFilter) => {
    const alarmsForExport: Alarm[] = [];
    let page = 1;

    while (true) {
      const response = await exportAlarmApi.getCurrentAlarms({
        ...filters,
        page,
        pageSize: EXPORT_PAGE_SIZE,
      });

      alarmsForExport.push(...response.items);

      if (response.items.length === 0 || response.page * response.pageSize >= response.total) {
        break;
      }

      page += 1;
    }

    return alarmsForExport;
  }, []);

  const downloadAlarmCsv = useCallback((items: Alarm[], fieldKeys: AlarmExportFieldKey[]) => {
    const selectedFieldSet = new Set(fieldKeys);
    const selectedFields = exportFields.filter((field) => selectedFieldSet.has(field.key));
    const headers = selectedFields.map((field) => field.label);
    const rows = items.map((alarm) => selectedFields.map((field) => field.getValue(alarm)));

    const csv = [headers, ...rows]
      .map((row) => row.map((cell) => escapeCsvCell(cell)).join(','))
      .join('\n');

    const datePart = new Date().toISOString().slice(0, 10);
    triggerCsvDownload(csv, `current-alarms-${datePart}.csv`);
  }, [exportFields]);


  const handleMarkRead = useCallback(
    async () => {
      try {
        await Promise.all(selectedRowKeys.map((id) => markAlarmRead.mutateAsync(id as string)));
        setSelectedRowKeys([]);
        message.success(t('common.markReadSuccess'));
      } catch {
        message.error(t('common.markReadFailed'));
      }
    },
    [selectedRowKeys, markAlarmRead, t, message]
  );

  const {
    clearSelection,
    exportLoading,
    exportOpen,
    handleExportConfirm,
    handleExportTrigger,
    handleSelectAllFiltered,
    selectAllLoading,
    setExportOpen,
  } = useAlarmListExport({
    exportMessageKey: 'current-alarm-export',
    filterParams,
    selectedRowKeys,
    setSelectedRowKeys,
    fetchAllAlarmsForExport: fetchAllCurrentAlarmsForExport,
    fetchDeviceSnsByGroups,
    downloadAlarmCsv,
    message,
    t,
  });

  const selectionActions = useMemo(() => (
    <Space size={8}>
      {total > 0 && (
        <Button
          size="small"
          loading={selectAllLoading}
          disabled={selectedRowKeys.length === total}
          onClick={() => { void handleSelectAllFiltered(); }}
        >
          {t('alarm.selectAllFiltered', { count: total })}
        </Button>
      )}
      {selectedRowKeys.length > 0 && (
        <Button size="small" onClick={clearSelection}>
          {t('common.unselectAll')}
        </Button>
      )}
    </Space>
  ), [clearSelection, handleSelectAllFiltered, selectAllLoading, selectedRowKeys.length, t, total]);

  // 打开告警详情
  const handleShowDetail = useCallback((alarm: Alarm) => {
    setDetailAlarm(alarm);
    setDetailOpen(true);
  }, []);

  // 关闭告警详情
  const handleCloseDetail = useCallback(() => {
    setDetailOpen(false);
    setDetailAlarm(null);
    if (deepLinkAlarmId) {
      setSearchParams(withoutAlarmId(searchParams), { replace: true });
    }
  }, [deepLinkAlarmId, searchParams, setSearchParams]);

  const alarmRowStyle = useCallback(
     
    (_record: Alarm): 'critical' | 'major' | 'minor' | 'warning' | null => {
      return null;
    },
    []
  );

  const columns = useMemo(
    (): DataTableColumn<Alarm>[] => [
      {
        key: 'actions',
        title: t('common.operation'),
        width: 56,
        fixed: 'left',
        render: (_val: unknown, record) => {
          const isConfirmed = record.dealState === '1' || record.dealState === '3';
          return (
            <Dropdown
              trigger={['click']}
              menu={{
                items: [
                  { key: 'detail', label: t('common.detail') },
                  { key: 'ack', label: t(isConfirmed ? 'alarm.unacknowledge' : 'alarm.acknowledge') },
                  { key: 'clear', label: t('alarm.clear'), danger: true },
                ],
                onClick: ({ key, domEvent }) => {
                  domEvent.stopPropagation();
                  if (key === 'detail') handleShowDetail(record);
                  if (key === 'ack') {
                    if (isConfirmed) {
                      handleUnacknowledge([record.id]);
                    } else {
                      handleAcknowledge([record.id]);
                    }
                  }
                  if (key === 'clear') handleClear([record.id]);
                },
              }}
            >
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          );
        },
      },
      {
        key: 'deviceSn',
        title: t('alarm.deviceSn'),
        dataIndex: 'deviceSn',
        width: 240,
        render: (val: unknown, record) =>
          renderSnWithTooltip(
            val,
            record.unread === '1' ? <Badge status="error" /> : undefined,
          ),
      },
      {
        key: 'neType',
        title: t('alarm.neTypeCol'),
        dataIndex: 'neType',
        width: 120,
        render: (val: unknown) => formatBaseStationTypeLabel(String(val ?? '')),
      },
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 100,
        render: (_val, record) => {
          const config = SEVERITY_CONFIG[record.severity] || SEVERITY_CONFIG.warning;
          return (
            <Tag
              style={{
                color: config.color,
                backgroundColor: config.bgColor,
                border: 'none',
              }}
            >
              {SEVERITY_LABEL[record.severity] ?? record.severity}
            </Tag>
          );
        },
      },
      {
        key: 'alarmIdentifier',
        title: t('alarm.alarmIdentifier'),
        dataIndex: 'alarmIdentifier',
        width: 130,
        mono: true,
      },
      {
        key: 'description',
        title: t('alarm.content'),
        dataIndex: 'description',
        width: 200,
        ellipsis: true,
      },
      {
        key: 'alarmName',
        title: t('alarm.possibleCause'),
        dataIndex: 'alarmName',
        width: 180,
        ellipsis: true,
      },
      {
        key: 'eventType',
        title: t('alarm.eventType'),
        dataIndex: 'eventType',
        width: 160,
        ellipsis: true,
        render: (val: unknown) => t(EVENT_TYPE_CONFIG[val as EventType] || 'common.unknown'),
      },
      {
        key: 'dealState',
        title: t('alarm.dealState'),
        dataIndex: 'dealState',
        width: 190,
        ellipsis: true,
        render: (val: unknown) => {
          const config = DEAL_STATE_CONFIG[val as DealState];
          return (
            <span style={{ color: config?.color || '#666' }}>
              {t(config?.label || 'common.unknown')}
            </span>
          );
        },
      },
      {
        key: 'eventTime',
        title: t('alarm.eventTime'),
        dataIndex: 'eventTime',
        width: 150,
        render: (v) => v ? formatSystemTime(String(v)) : '-',
      },
      {
        key: 'updTime',
        title: t('alarm.updTime'),
        dataIndex: 'updTime',
        width: 150,
        render: (v) => v ? formatSystemTime(String(v)) : '-',
      },
      {
        key: 'alarmCount',
        title: t('alarm.alarmCount'),
        dataIndex: 'alarmCount',
        width: 100,
      },
      {
        key: 'dealMemo',
        title: t('alarm.dealMemo'),
        dataIndex: 'dealMemo',
        width: 100,
        ellipsis: true,
      },
    ],
    [t, SEVERITY_LABEL, handleAcknowledge, handleClear, handleShowDetail, handleUnacknowledge]
  );

  const batchActions = useMemo(
    (): BatchAction[] => [
      {
        key: 'batch-ack',
        label: t('alarm.acknowledge'),
        icon: <CheckOutlined />,
        onClick: (keys) => handleAcknowledge(keys as string[]),
      },
      {
        key: 'batch-unack',
        label: t('alarm.unacknowledge'),
        icon: <MinusCircleOutlined />,
        onClick: (keys) => handleUnacknowledge(keys as string[]),
      },
      {
        key: 'batch-clear',
        label: t('alarm.clear'),
        icon: <ClearOutlined />,
        danger: true,
        onClick: (keys) => handleClear(keys as string[]),
      },
      {
        key: 'batch-read',
        label: t('alarm.markRead'),
        icon: <EyeOutlined />,
        onClick: () => handleMarkRead(),
      },
    ],
    [handleAcknowledge, handleUnacknowledge, handleClear, handleMarkRead, t]
  );

  return (
    <ListPageLayout
      title={t('nav.alarm.current')}
      extra={
        <Space>
          <AutoRefreshDropdown
            enabled={autoRefresh}
            intervalSeconds={refreshInterval}
            onEnabledChange={setAutoRefresh}
            onIntervalChange={setRefreshInterval}
          />
          <Button
            icon={<ExportOutlined />}
            loading={exportLoading}
            onClick={() => { void handleExportTrigger(); }}
          >
            {t('common.export')}
          </Button>
        </Space>
      }
    >
      {/* 统计卡片 - Pill Tabs 风格 */}
      <Card
        size="small"
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}
        style={{ marginBottom: 12 }}
      >
        <div className={styles.cardHeader}>
          <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', width: '100%' }}>
            <Space size={8}>
              <div className={`${styles.statsBadge} ${styles.statsCritical}`}>
                <span>{t('alarm.statistics.critical')}: {realStats.critical}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsMajor}`}>
                <span>{t('alarm.statistics.major')}: {realStats.major}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsMinor}`}>
                <span>{t('alarm.statistics.minor')}: {realStats.minor}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsWarning}`}>
                <span>{t('alarm.statistics.warning')}: {realStats.warning}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsUnacked}`}>
                <span>{t('alarm.statistics.unconfirmed')}: {realStats.unacked}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsUnread}`}>
                <span>{t('alarm.statistics.unread')}: {realStats.unread}</span>
              </div>
            </Space>
          </div>
        </div>
      </Card>

      <FilterBar
        filterId="current-alarms"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={2}
        initialValues={drillDownInitialValues}
      />

      {/* 列表卡片 */}
      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<Alarm>
          tableId="current-alarms-table"
          columns={columns}
          dataSource={alarms}
          loading={isLoading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={(keys) => setSelectedRowKeys(keys)}
          preserveSelectedRowKeys
          total={total}
          pageSize={pageSize}
          currentPage={currentPage}
          onPageChange={(page, size) => {
            dispatchList({ type: 'page', page });
            setPageSize(size);
          }}
          batchActions={batchActions}
          onRefresh={() => void refetch()}
          hideRealtime
          alarmRowStyle={alarmRowStyle as (record: Alarm) => 'critical' | 'major' | 'minor' | 'warning' | null}
          defaultDensity="default"
          extraToolbarAfterBatch={selectionActions}
          showRowNumber
          rowNumberTitle={t('common.rowNumber')}
          scroll={{ y: 'calc(100vh - 450px)' }}
        />
      </Card>

      <AlarmDetail
        alarm={activeDetailAlarm}
        open={activeDetailOpen}
        onClose={handleCloseDetail}
      />

      <ExportModal
        open={exportOpen}
        onClose={() => setExportOpen(false)}
        onConfirm={handleExportConfirm}
        confirmLoading={exportLoading}
        fieldOptions={exportFieldOptions}
        defaultFieldKeys={defaultExportFieldKeys}
        hasSelectedRows={selectedRowKeys.length > 0}
        selectedRowCount={selectedRowKeys.length}
      />

      <ConfirmWithNoteModal
        open={ackModalOpen}
        title={t('alarm.acknowledge')}
        message={t('common.ackConfirmMsg', { count: ackTargetIds.length })}
        onConfirm={handleAcknowledgeConfirm}
        onCancel={() => setAckModalOpen(false)}
        loading={ackLoading}
      />

      <ConfirmWithNoteModal
        open={clearModalOpen}
        title={t('alarm.clear')}
        message={t('common.clearConfirmMsg', { count: clearTargetIds.length })}
        confirmText={t('alarm.clear')}
        confirmType="danger"
        onConfirm={handleClearConfirm}
        onCancel={() => setClearModalOpen(false)}
        loading={clearLoading}
      />
    </ListPageLayout>
  );
}
