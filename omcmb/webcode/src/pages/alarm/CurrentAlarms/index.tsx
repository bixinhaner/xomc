import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge, Button, Card, Dropdown, Space, Tag, Typography, App } from 'antd';
import {
  CheckOutlined,
  ClearOutlined,
  ExportOutlined,
  EyeOutlined,
  MinusCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';

import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { alarmService } from '@core/mock/services/alarmService';
import { deviceService } from '@core/mock/services/deviceService';
import { useCurrentAlarms, useAcknowledgeAlarms, useClearAlarms, useAlarmCount, useMarkAlarmRead, useUnacknowledgeAlarms } from '@core/hooks/api/useAlarms';
import { alarmApi } from '@core/services/api/alarmApi';
import { deviceApi } from '@core/services/api/deviceApi';
import { createApiSwitch } from '@core/services/apiSwitch';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import type { AlarmFilter } from '@core/types/alarm';
import AlarmDetail from '../AlarmDetail';
import ExportModal, { type ExportParams } from './ExportModal';
import ConfirmWithNoteModal from '../components/ConfirmWithNoteModal';
import { BASE_STATION_TYPE_OPTIONS, formatBaseStationTypeLabel } from '../utils/baseStationType';
import styles from './CurrentAlarms.module.css';

const { Text } = Typography;

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

// 快捷筛选选项
const QUICK_FILTER_OPTIONS = [
  { label: '全部', key: 'all' },
  { label: '严重', key: 'critical', severity: ['critical'] },
  { label: '主要', key: 'major', severity: ['major'] },
  { label: '次要', key: 'minor', severity: ['minor'] },
  { label: '警告', key: 'warning', severity: ['warning'] },
  { label: '未确认', key: 'unacked', dealState: ['0'] },
  { label: '未读', key: 'unread', unread: '1' },
];

// 自动刷新间隔选项 - 移至组件内 useMemo

const exportAlarmApi: typeof alarmApi = createApiSwitch(
  alarmService as unknown as typeof alarmApi,
  alarmApi,
);
const exportDeviceApi = createApiSwitch(deviceService, deviceApi);
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

export default function CurrentAlarms() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<AlarmFilter>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);
  const [exportLoading, setExportLoading] = useState(false);
  const [exportMode, setExportMode] = useState<'all' | 'selected'>('all');

  // 确认告警弹窗状态
  const [ackModalOpen, setAckModalOpen] = useState(false);
  const [ackTargetIds, setAckTargetIds] = useState<string[]>([]);
  const [ackLoading, setAckLoading] = useState(false);

  // 清除告警弹窗状态
  const [clearModalOpen, setClearModalOpen] = useState(false);
  const [clearTargetIds, setClearTargetIds] = useState<string[]>([]);
  const [clearLoading, setClearLoading] = useState(false);

  // 快捷筛选状态
  const [activeQuickFilter, setActiveQuickFilter] = useState<string>('all');
  // 自动刷新状态
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30);

  // 自动刷新间隔选项（依赖 t）
  const AUTO_REFRESH_INTERVALS = useMemo(() => [
    { label: t('common.15seconds'), value: 15 },
    { label: t('common.30seconds'), value: 30 },
    { label: t('common.1minute'), value: 60 },
    { label: t('common.5minutes'), value: 300 },
  ], [t]);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('alarm.search'), type: 'input', placeholder: t('alarm.searchPlaceholderNew') },
    { name: 'timeRange', label: t('alarm.eventTime'), type: 'date-range' },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'multi-select',
      options: [
        { label: t('alarm.severity.critical'), value: 'critical' },
        { label: t('alarm.severity.major'), value: 'major' },
        { label: t('alarm.severity.minor'), value: 'minor' },
        { label: t('alarm.severity.warning'), value: 'warning' },
      ],
    },
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
      name: 'neType',
      label: t('alarm.neTypeCol'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        ...BASE_STATION_TYPE_OPTIONS,
      ],
    },
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
      name: 'dealState',
      label: t('alarm.dealState'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.dealState.unconfirmedUncleared'), value: '0' },
        { label: t('alarm.dealState.confirmedUncleared'), value: '1' },
      ],
    },
    {
      // 未识别告警过滤：识别状态 = identifier 是否在告警库中存在（设计 §3.3 治理闭环）
      name: 'isUnknown',
      label: t('alarm.recognizeStatus', { defaultMessage: '识别状态' }),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('alarm.recognizeStatus.unknown', { defaultMessage: '仅未识别' }), value: 'true' },
        { label: t('alarm.recognizeStatus.known', { defaultMessage: '仅已识别' }), value: 'false' },
      ],
    },
  ], [t]);

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

  const { data, isLoading, refetch } = useCurrentAlarms(queryParams);
  const acknowledgeAlarms = useAcknowledgeAlarms();
  const clearAlarms = useClearAlarms();
  const markAlarmRead = useMarkAlarmRead();
  const unacknowledgeAlarms = useUnacknowledgeAlarms();
  const { data: alarmCount } = useAlarmCount();

  const rawAlarms: Alarm[] = useMemo(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;

  // 自动刷新
  useEffect(() => {
    if (!autoRefresh) return;

    const timer = setInterval(() => {
      void refetch();
    }, refreshInterval * 1000);

    return () => clearInterval(timer);
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
    setFilterParams({
      severity: values.severity as AlarmFilter['severity'],
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      unread: values.unread as '0' | '1',
      dealState: values.dealState as AlarmFilter['dealState'],
      isUnknown: values.isUnknown as AlarmFilter['isUnknown'],
      keyword: values.keyword as string,
      timeRange: values.timeRange as [string, string] | undefined,
    });
    setCurrentPage(1);
    setActiveQuickFilter('all');
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    setActiveQuickFilter('all');
  }, []);

  // 快捷筛选处理
  const handleQuickFilter = useCallback((key: string) => {
    setActiveQuickFilter(key);
    const option = QUICK_FILTER_OPTIONS.find((o) => o.key === key);
    if (option && option.key !== 'all') {
      setFilterParams((prev) => ({
        ...prev,
        severity: option.severity as AlarmFilter['severity'],
        dealState: option.dealState as AlarmFilter['dealState'],
        unread: option.unread as '0' | '1',
      }));
    } else {
      setFilterParams((prev) => {
         
        const { severity, dealState, unread, ...rest } = prev;
        return rest;
      });
    }
    setCurrentPage(1);
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

  const downloadAlarmCsv = useCallback((items: Alarm[]) => {
    const headers = [
      t('alarm.alarmId'),
      t('alarm.alarmIdentifier'),
      t('alarm.severity'),
      t('alarm.possibleCause'),
      t('alarm.equipInfo'),
      t('alarm.eventType'),
      t('alarm.dealState'),
      t('alarm.eventTime'),
      t('alarm.updTime'),
      t('alarm.dealMemo'),
    ];

    const rows = items.map((alarm) => [
      alarm.id,
      alarm.alarmIdentifier,
      SEVERITY_LABEL[alarm.severity] ?? alarm.severity,
      alarm.alarmName,
      alarm.equipInfo,
      t(EVENT_TYPE_CONFIG[alarm.eventType] || 'common.unknown'),
      t(DEAL_STATE_CONFIG[alarm.dealState]?.label || 'common.unknown'),
      alarm.eventTime,
      alarm.updTime,
      alarm.dealMemo || '',
    ]);

    const csv = [headers, ...rows]
      .map((row) => row.map((cell) => escapeCsvCell(cell)).join(','))
      .join('\n');

    const datePart = new Date().toISOString().slice(0, 10);
    triggerCsvDownload(csv, `current-alarms-${datePart}.csv`);
  }, [t, SEVERITY_LABEL]);


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

  // 导出告警
  const handleExport = useCallback(
    async (params: ExportParams) => {
      setExportLoading(true);
      message.open({ key: 'current-alarm-export', type: 'loading', content: t('common.exportInProgress'), duration: 0 });
      try {
        if (params.deviceGroupIds.length === 0) {
          message.open({ key: 'current-alarm-export', type: 'warning', content: t('export.selectDeviceGroup') });
          return;
        }

        const selectedIdSet = new Set(selectedRowKeys.map((key) => String(key)));
        const effectiveFilter: AlarmFilter = params.timeRange
          ? { ...filterParams, timeRange: params.timeRange }
          : filterParams;

        const allowedDeviceSns = await fetchDeviceSnsByGroups(params.deviceGroupIds);
        if (allowedDeviceSns.size === 0) {
          message.open({ key: 'current-alarm-export', type: 'warning', content: t('common.noDataToExport') });
          return;
        }

        const alarmsForExport = await fetchAllCurrentAlarmsForExport(effectiveFilter);
        const filteredAlarms = alarmsForExport.filter((alarm) => allowedDeviceSns.has(alarm.deviceSn));
        const exportItems = exportMode === 'selected'
          ? filteredAlarms.filter((alarm) => selectedIdSet.has(alarm.id))
          : filteredAlarms;

        if (exportItems.length === 0) {
          message.open({ key: 'current-alarm-export', type: 'warning', content: t('common.noDataToExport') });
          return;
        }

        downloadAlarmCsv(exportItems);
        message.open({ key: 'current-alarm-export', type: 'success', content: t('common.exportSuccess') });
        setExportOpen(false);
      } catch {
        message.open({ key: 'current-alarm-export', type: 'error', content: t('common.exportFailed') });
      } finally {
        setExportLoading(false);
      }
    },
    [
      downloadAlarmCsv,
      exportMode,
      fetchAllCurrentAlarmsForExport,
      fetchDeviceSnsByGroups,
      filterParams,
      message,
      selectedRowKeys,
      t,
    ]
  );

  // 打开告警详情
  const handleShowDetail = useCallback((alarm: Alarm) => {
    setDetailAlarm(alarm);
    setDetailOpen(true);
  }, []);

  // 关闭告警详情
  const handleCloseDetail = useCallback(() => {
    setDetailOpen(false);
    setDetailAlarm(null);
  }, []);

  const alarmRowStyle = useCallback(
     
    (_record: Alarm): 'critical' | 'major' | 'minor' | 'warning' | null => {
      return null;
    },
    []
  );

  const columns = useMemo(
    (): DataTableColumn<Alarm>[] => [
      {
        key: 'id',
        title: t('alarm.alarmId'),
        dataIndex: 'id',
        width: 100,
        render: (val: unknown, record) => (
          <Space size={4}>
            {record.unread === '1' && <Badge status="error" style={{ marginLeft: -4 }} />}
            <Button
              type="link"
              size="small"
              style={{ padding: 0, height: 'auto' }}
              onClick={() => handleShowDetail(record)}
            >
              {String(val ?? '')}
            </Button>
          </Space>
        ),
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
        key: 'alarmName',
        title: t('alarm.possibleCause'),
        dataIndex: 'alarmName',
        width: 180,
        ellipsis: true,
      },
      {
        key: 'neType',
        title: t('alarm.neTypeCol'),
        dataIndex: 'neType',
        width: 120,
        render: (val: unknown) => formatBaseStationTypeLabel(String(val ?? '')),
      },
      {
        key: 'equipInfo',
        title: t('alarm.equipInfo'),
        dataIndex: 'equipInfo',
        width: 250,
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
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
      },
      {
        key: 'updTime',
        title: t('alarm.updTime'),
        dataIndex: 'updTime',
        width: 150,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
      },
      {
        key: 'specificProblem',
        title: t('alarm.specificProblem'),
        dataIndex: 'description',
        width: 150,
        ellipsis: true,
      },
      {
        key: 'description',
        title: t('alarm.content'),
        dataIndex: 'description',
        width: 200,
        ellipsis: true,
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
    [t, SEVERITY_LABEL, handleShowDetail]
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

  // 自动刷新下拉菜单
  const autoRefreshMenuItems = useMemo(() => [
    {
      key: 'toggle',
      label: autoRefresh ? t('alarm.autoRefreshOff') : t('alarm.autoRefreshOn'),
      icon: <SyncOutlined spin={autoRefresh} />,
    },
    ...(autoRefresh ? AUTO_REFRESH_INTERVALS.map((opt) => ({
      key: `interval-${opt.value}`,
      label: opt.label,
    })) : []),
  ], [autoRefresh, AUTO_REFRESH_INTERVALS, t]);

  const handleAutoRefreshMenuClick = useCallback(({ key }: { key: string }) => {
    if (key === 'toggle') {
      setAutoRefresh(!autoRefresh);
    } else if (key.startsWith('interval-')) {
      setRefreshInterval(parseInt(key.replace('interval-', ''), 10));
    }
  }, [autoRefresh]);

  return (
    <ListPageLayout
      title={t('nav.alarm.current')}
      extra={
        <Space>
          <Dropdown menu={{ items: autoRefreshMenuItems, onClick: handleAutoRefreshMenuClick }}>
            <Button icon={<SyncOutlined spin={autoRefresh} />} type={autoRefresh ? 'primary' : 'default'}>
              {autoRefresh ? `${refreshInterval}秒` : '自动刷新'}
            </Button>
          </Dropdown>
          <Button
            icon={<ExportOutlined />}
            onClick={() => {
              setExportMode('all');
              setExportOpen(true);
            }}
          >
            导出
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
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
            <Space size={16}>
              <Text className={styles.cardHeaderTitle}>{t('nav.alarm.statistics')}</Text>
              <div className={styles.pillTabs}>
                {[
                  { key: 'all', label: t('alarm.statistics.all') },
                  { key: 'critical', label: t('alarm.statistics.critical') },
                  { key: 'major', label: t('alarm.statistics.major') },
                  { key: 'minor', label: t('alarm.statistics.minor') },
                  { key: 'warning', label: t('alarm.statistics.warning') },
                  { key: 'unacked', label: t('alarm.statistics.unconfirmed') },
                  { key: 'unread', label: t('alarm.statistics.unread') },
                ].map(item => (
                  <span
                    key={item.key}
                    className={`${styles.pillTab} ${activeQuickFilter === item.key ? styles.pillTabActive : styles.pillTabInactive}`}
                    onClick={() => handleQuickFilter(item.key)}
                  >
                    {item.label}
                  </span>
                ))}
              </div>
            </Space>
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
        collapsedRows={1}
      />

      {/* 列表卡片 */}
      <Card
        size="small"
        bordered
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
          total={total}
          pageSize={pageSize}
          currentPage={currentPage}
          onPageChange={(page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          }}
          batchActions={batchActions}
          onRefresh={() => void refetch()}
          alarmRowStyle={alarmRowStyle as (record: Alarm) => 'critical' | 'major' | 'minor' | 'warning' | null}
          defaultDensity="default"
          showRowNumber
          rowNumberTitle={t('common.rowNumber')}
          scroll={{ y: 'calc(100vh - 450px)' }}
        />
      </Card>

      <AlarmDetail
        alarm={detailAlarm}
        open={detailOpen}
        onClose={handleCloseDetail}
      />

      <ExportModal
        open={exportOpen}
        onClose={() => setExportOpen(false)}
        onConfirm={handleExport}
        confirmLoading={exportLoading}
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
