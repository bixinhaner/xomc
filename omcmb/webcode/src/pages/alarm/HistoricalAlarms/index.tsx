import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Button, Card, Space, Tag, Typography, App } from 'antd';
import {
  CheckOutlined,
  DeleteOutlined,
  ExportOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useHistoricalAlarms, useAcknowledgeHistoryAlarms, useUnacknowledgeHistoryAlarms, useDeleteHistoryAlarms, useHistoryAlarmCount } from '@core/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import type { AlarmFilter } from '@core/types/alarm';
import AlarmDetail from '../AlarmDetail';
import ExportModal, { type ExportParams } from '../CurrentAlarms/ExportModal';
import ConfirmWithNoteModal from '../components/ConfirmWithNoteModal';
import styles from './HistoricalAlarms.module.css';

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

// 基站制式配置
const NE_TYPE_CONFIG: Record<string, string> = {
  'eNB': 'eNB',
  'gNB': 'gNB',
  'GSM': 'GSM',
};

export default function HistoricalAlarms() {
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
  const [activeQuickFilter, setActiveQuickFilter] = useState<string>('all');

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
      label: t('alarm.neType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'GSM', value: 'GSM' },
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

  const { data, isLoading, refetch } = useHistoricalAlarms(queryParams);
  const acknowledgeHistoryAlarms = useAcknowledgeHistoryAlarms();
  const unacknowledgeHistoryAlarms = useUnacknowledgeHistoryAlarms();
  const deleteHistoryAlarms = useDeleteHistoryAlarms();
  const { data: alarmCount } = useHistoryAlarmCount();

  const rawAlarms: Alarm[] = useMemo(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;

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
      severity: values.severity as AlarmFilter['severity'],
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      dealState: values.dealState as AlarmFilter['dealState'],
      keyword: values.keyword as string,
      timeRange: values.timeRange as [string, string] | undefined,
    });
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    setActiveQuickFilter('all');
  }, []);

  // 快捷筛选处理
  const handleQuickFilter = useCallback((key: string) => {
    setActiveQuickFilter(key);
    if (key === 'all') {
      setFilterParams((prev) => {
         
        const { severity, ...rest } = prev;
        return rest;
      });
    } else if (key === 'cleared') {
      setFilterParams((prev) => ({
        ...prev,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        dealState: ['2', '3'] as any,
      }));
    } else if (key === 'confirmed') {
      setFilterParams((prev) => ({
        ...prev,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        dealState: ['1', '3'] as any,
      }));
    } else {
      // severity: critical, major, minor, warning
      setFilterParams((prev) => ({
        ...prev,
        severity: [key] as AlarmFilter['severity'],
      }));
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

  // 删除告警
  const handleDelete = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.deleteAlarm'),
        content: t('alarm.deleteConfirmMsg', { count: ids.length }),
        okText: t('common.delete'),
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

  // 导出告警
  const handleExport = useCallback(
    async (params: ExportParams) => {
      setExportLoading(true);
      try {
        // TODO: 调用导出 API
        console.log('Export params:', params);
        void message.info(t('common.exportInProgress'));
        setExportOpen(false);
      } catch {
        message.error(t('common.exportFailed'));
      } finally {
        setExportLoading(false);
      }
    },
    [message, t]
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
        render: (val, record) => (
          <Space size={4}>
            {record.unread === '1' && <Badge status="error" style={{ marginLeft: -4 }} />}
            <Button
              type="link"
              size="small"
              style={{ padding: 0, height: 'auto' }}
              onClick={() => handleShowDetail(record)}
            >
              {val}
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
        key: 'alarmCode',
        title: t('alarm.alarmIdentifier'),
        dataIndex: 'alarmCode',
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
        title: t('alarm.neType'),
        dataIndex: 'neType',
        width: 120,
        render: (val: string) => NE_TYPE_CONFIG[val] || val || '-',
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
        render: (val: EventType) => t(EVENT_TYPE_CONFIG[val] || 'common.unknown'),
      },
      {
        key: 'dealState',
        title: t('alarm.dealState'),
        dataIndex: 'dealState',
        width: 190,
        ellipsis: true,
        render: (val: DealState) => {
          const config = DEAL_STATE_CONFIG[val];
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
        key: 'clearTime',
        title: t('alarm.clearTime'),
        dataIndex: 'clearTime',
        width: 150,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
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
        <Button
          icon={<ExportOutlined />}
          onClick={() => {
            setExportOpen(true);
          }}
        >
          导出
        </Button>
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
              <Text className={styles.cardHeaderTitle}>{t('alarm.statistics.title')}</Text>
              <div className={styles.pillTabs}>
                {[
                  { key: 'all', label: t('alarm.statistics.all') },
                  { key: 'critical', label: t('alarm.statistics.critical') },
                  { key: 'major', label: t('alarm.statistics.major') },
                  { key: 'minor', label: t('alarm.statistics.minor') },
                  { key: 'warning', label: t('alarm.statistics.warning') },
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
            </Space>
          </div>
        </div>
      </Card>

      <FilterBar
        filterId="historical-alarms"
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
          tableId="historical-alarms-table"
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
    </ListPageLayout>
  );
}
