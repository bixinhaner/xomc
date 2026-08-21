import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge, Button, Card, Dropdown, Space, Tag, App } from 'antd';
import {
  CheckOutlined,
  DeleteOutlined,
  ExportOutlined,
  MoreOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { alarmService } from '@core/mock/services/alarmService';
import { deviceService } from '@core/mock/services/deviceService';
import { useHistoricalAlarms, useAcknowledgeHistoryAlarms, useUnacknowledgeHistoryAlarms, useDeleteHistoryAlarms, useHistoryAlarmCount } from '@core/hooks/api/useAlarms';
import { alarmApi } from '@core/services/api/alarmApi';
import { deviceApi } from '@core/services/api/deviceApi';
import { createApiSwitch } from '@core/services/apiSwitch';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import type { AlarmFilter } from '@core/types/alarm';
import AlarmDetail from '../AlarmDetail';
import AutoRefreshDropdown from '../components/AutoRefreshDropdown';
import ExportModal from '../CurrentAlarms/ExportModal';
import ConfirmWithNoteModal from '../components/ConfirmWithNoteModal';
import { useAlarmListExport } from '../hooks/useAlarmListExport';
import { buildAlarmExportFieldDefinitions, type AlarmExportFieldKey } from '../utils/alarmExportFields';
import { getBaseStationTypeOptions, formatBaseStationTypeLabel } from '../utils/baseStationType';
import { renderSnWithTooltip } from '../utils/snTooltip';
import styles from './HistoricalAlarms.module.css';
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
  '0': { label: 'alarm.dealState.unconfirmedUncleared', color: '#E53935', icon: 'unconfirmInactive' },  // 红色 - 未确认未清除
  '1': { label: 'alarm.dealState.confirmedUncleared', color: '#FB8C00', icon: 'confirmInactive' },      // 橙色 - 已确认未清除
  '2': { label: 'alarm.dealState.unconfirmedCleared', color: '#42A5F5', icon: 'unconfirmActive' },       // 蓝色 - 未确认已清除
  '3': { label: 'alarm.dealState.confirmedCleared', color: '#67D972', icon: 'confirmActive' },           // 绿色 - 已确认已清除
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

export default function HistoricalAlarms() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<AlarmFilter>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30);

  // 确认告警弹窗状态
  const [ackModalOpen, setAckModalOpen] = useState(false);
  const [ackTargetIds, setAckTargetIds] = useState<string[]>([]);
  const [ackLoading, setAckLoading] = useState(false);

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
        { label: t('alarm.dealState.unconfirmedCleared'), value: '2' },
        { label: t('alarm.dealState.confirmedCleared'), value: '3' },
      ],
    },
    { name: 'timeRange', label: t('alarm.eventTime'), type: 'date-range', showTime: true },
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

  const { data, isLoading, refetch } = useHistoricalAlarms(
    queryParams as unknown as Parameters<typeof useHistoricalAlarms>[0],
    {
      refetchIntervalMs: autoRefresh ? refreshInterval * 1000 : false,
      refetchIntervalInBackground: autoRefresh,
      refetchOnWindowFocus: true,
    }
  );
  const acknowledgeHistoryAlarms = useAcknowledgeHistoryAlarms();
  const unacknowledgeHistoryAlarms = useUnacknowledgeHistoryAlarms();
  const deleteHistoryAlarms = useDeleteHistoryAlarms();
  const { data: alarmCount } = useHistoryAlarmCount();

  const rawAlarms: Alarm[] = useMemo(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;

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
  }), [alarmCount, total]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams({
      deviceSn: values.deviceSn as string,
      severity: values.severity as AlarmFilter['severity'],
      alarmIdentifier: values.alarmIdentifier as string,
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      dealState: values.dealState as AlarmFilter['dealState'],
      timeRange: values.timeRange as [string, string] | undefined,
    });
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
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
        await acknowledgeHistoryAlarms.mutateAsync({ ids: ackTargetIds, note });
        setSelectedRowKeys([]);
        setAckModalOpen(false);
        message.success(t('common.ackSuccess'));
      } catch {
        message.error(t('common.ackFailed'));
      } finally {
        setAckLoading(false);
      }
    },
    [acknowledgeHistoryAlarms, ackTargetIds, t, message]
  );

  const handleUnacknowledge = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.unacknowledge'),
        content: t('common.unackConfirmMsg', { count: ids.length }),
        okText: t('common.confirm'),
        onOk: async () => {
          try {
            await unacknowledgeHistoryAlarms.mutateAsync(ids);
            setSelectedRowKeys([]);
            message.success(t('common.unackSuccess'));
          } catch {
            message.error(t('common.unackFailed'));
          }
        },
      });
    },
    [unacknowledgeHistoryAlarms, t, message, modal]
  );

  // 删除历史告警
  const handleDelete = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.deleteAlarm'),
        content: t('alarm.deleteConfirmMsg', { count: ids.length }),
        okText: t('alarm.deleteAlarm'),
        okType: 'danger',
        icon: <DeleteOutlined />,
        onOk: async () => {
          try {
            await deleteHistoryAlarms.mutateAsync(ids);
            setSelectedRowKeys([]);
            message.success(t('common.deleteSuccess'));
          } catch {
            message.error(t('common.deleteFailed'));
          }
        },
      });
    },
    [deleteHistoryAlarms, t, message, modal]
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

  const fetchAllHistoricalAlarmsForExport = useCallback(async (filters: AlarmFilter) => {
    const alarmsForExport: Alarm[] = [];
    let page = 1;

    while (true) {
      const response = await exportAlarmApi.getHistoricalAlarms({
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
    triggerCsvDownload(csv, `historical-alarms-${datePart}.csv`);
  }, [exportFields]);

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
    exportMessageKey: 'history-alarm-export',
    filterParams,
    selectedRowKeys,
    setSelectedRowKeys,
    fetchAllAlarmsForExport: fetchAllHistoricalAlarmsForExport,
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
                  { key: 'delete', label: t('alarm.deleteAlarm'), danger: true },
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
                  if (key === 'delete') handleDelete([record.id]);
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
        key: 'clearTime',
        title: t('alarm.clearTime'),
        dataIndex: 'clearTime',
        width: 150,
        render: (v) => v ? formatSystemTime(String(v)) : '-',
      },
      {
        key: 'specificProblem',
        title: t('alarm.specificProblem'),
        dataIndex: 'specificProblem',
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
    [t, SEVERITY_LABEL, handleAcknowledge, handleDelete, handleShowDetail, handleUnacknowledge]
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
        key: 'batch-delete',
        label: t('alarm.deleteAlarm'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (keys) => handleDelete(keys as string[]),
      },
    ],
    [handleAcknowledge, handleUnacknowledge, handleDelete, t]
  );

  return (
    <ListPageLayout
      title={t('nav.alarm.history')}
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
            </Space>
          </div>
        </div>
      </Card>

      <FilterBar
        filterId="historical-alarms"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={2}
      />

      {/* 列表卡片 */}
      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<Alarm>
          tableId="historical-alarms-table"
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
            setCurrentPage(page);
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
        alarm={detailAlarm}
        open={detailOpen}
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
    </ListPageLayout>
  );
}
