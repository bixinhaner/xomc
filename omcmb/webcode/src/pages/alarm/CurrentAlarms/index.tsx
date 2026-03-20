import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Button, Space, Tag, Typography, App } from 'antd';
import {
  BellOutlined,
  CheckOutlined,
  ClearOutlined,
  EyeOutlined,
  FilterOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useCurrentAlarms, useAcknowledgeAlarms, useClearAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@/types/alarm';
import type { AlarmFilter } from '@/types/alarm';
import AlarmDetail from '../AlarmDetail';

const { Text } = Typography;

// 告警级别颜色
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  critical: { color: '#FC5959', bgColor: '#FFF1F0' },
  major: { color: '#FF973E', bgColor: '#FFF7E6' },
  minor: { color: '#FFDA41', bgColor: '#FFFBE6' },
  warning: { color: '#60BEFC', bgColor: '#E6F7FF' },
};

// 告警状态配置
const DEAL_STATE_CONFIG: Record<DealState, { label: string; color: string; icon: string }> = {
  '0': { label: 'alarm.dealState.unconfirmedUncleared', color: '#E88282', icon: 'unconfirmInactive' },
  '1': { label: 'alarm.dealState.confirmedUncleared', color: '#E88282', icon: 'confirmInactive' },
  '2': { label: 'alarm.dealState.unconfirmedCleared', color: '#67D972', icon: 'unconfirmActive' },
  '3': { label: 'alarm.dealState.confirmedCleared', color: '#67D972', icon: 'confirmActive' },
};

// 事件类型配置
const EVENT_TYPE_CONFIG: Record<EventType, string> = {
  '30000': 'alarm.eventType.communication',
  '30001': 'alarm.eventType.qualityOfService',
  '30002': 'alarm.eventType.processingError',
  '30003': 'alarm.eventType.device',
  '30004': 'alarm.eventType.environment',
  '30006': 'alarm.eventType.performance',
};

export default function CurrentAlarms() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<AlarmFilter>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('alarm.search'), type: 'input', placeholder: t('alarm.searchPlaceholder') },
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
        { label: t('alarm.eventType.communication'), value: '30000' },
        { label: t('alarm.eventType.qualityOfService'), value: '30001' },
        { label: t('alarm.eventType.processingError'), value: '30002' },
        { label: t('alarm.eventType.device'), value: '30003' },
        { label: t('alarm.eventType.environment'), value: '30004' },
      ],
    },
    {
      name: 'neType',
      label: t('alarm.neType'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        // TODO: 从 API 动态加载网元类型
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
  ], [t]);

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize }),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useCurrentAlarms(queryParams);
  const acknowledgeAlarms = useAcknowledgeAlarms();
  const clearAlarms = useClearAlarms();

  const rawAlarms: Alarm[] = data?.items ?? [];
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

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams({
      severity: values.severity as AlarmFilter['severity'],
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      unread: values.unread as '0' | '1',
      dealState: values.dealState as AlarmFilter['dealState'],
      keyword: values.keyword as string,
    });
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const handleAcknowledge = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.acknowledge'),
        content: t('common.ackConfirmMsg', { count: ids.length }),
        okText: t('common.confirm'),
        icon: <CheckOutlined style={{ color: 'var(--color-primary-600)' }} />,
        onOk: async () => {
          try {
            await acknowledgeAlarms.mutateAsync({ ids });
            setSelectedRowKeys([]);
            refetch();
            message.success(t('common.ackSuccess'));
          } catch {
            message.error(t('common.ackFailed'));
          }
        },
      });
    },
    [acknowledgeAlarms, refetch, t]
  );

  const handleUnacknowledge = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.unacknowledge'),
        content: t('common.unackConfirmMsg', { count: ids.length }),
        okText: t('common.confirm'),
        onOk: async () => {
          try {
            // TODO: 调用反确认 API
            setSelectedRowKeys([]);
            refetch();
            message.success(t('common.unackSuccess'));
          } catch {
            message.error(t('common.unackFailed'));
          }
        },
      });
    },
    [refetch, t]
  );

  const handleClear = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.clear'),
        content: t('common.clearConfirmMsg', { count: ids.length }),
        okText: t('alarm.clear'),
        okType: 'danger',
        icon: <ClearOutlined />,
        onOk: async () => {
          try {
            await clearAlarms.mutateAsync(ids);
            setSelectedRowKeys([]);
            refetch();
            message.success(t('common.clearSuccess'));
          } catch {
            message.error(t('common.clearFailed'));
          }
        },
      });
    },
    [clearAlarms, refetch, t]
  );

  const handleFilter = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('alarm.filterAlarm'),
        content: t('alarm.filterAlarmConfirm'),
        okText: t('common.confirm'),
        icon: <FilterOutlined />,
        onOk: async () => {
          // TODO: 调用过滤告警 API
          setSelectedRowKeys([]);
          refetch();
          message.success(t('common.success'));
        },
      });
    },
    [refetch, t]
  );

  const handleMarkRead = useCallback(
    (ids: string[]) => {
      // TODO: 调用标记已读 API
      setSelectedRowKeys([]);
      refetch();
      message.success(t('common.markReadSuccess'));
    },
    [refetch, t]
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
    // 去掉紧急告警的红色背景色，不再根据告警级别设置行样式
    (_record: Alarm): 'critical' | 'major' | 'minor' | 'warning' | null => {
      return null;
    },
    []
  );

  const columns = useMemo(
    (): DataTableColumn<Alarm>[] => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'left',
        render: (_, record) => (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleShowDetail(record)}
          >
            {t('alarm.detail')}
          </Button>
        ),
      },
      {
        key: 'alarmId',
        title: t('alarm.alarmId'),
        dataIndex: 'alarmId',
        width: 80,
        sorter: true,
        render: (val, record) => (
          <Space size={4}>
            {record.unread === '1' && <Badge status="error" style={{ marginLeft: -4 }} />}
            <span>{val}</span>
          </Space>
        ),
      },
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 100,
        sorter: true,
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
        title: t('alarm.neType'),
        dataIndex: 'neType',
        width: 120,
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
        sorter: true,
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
        key: 'alarmType',
        title: t('alarm.alarmType'),
        dataIndex: 'alarmType',
        width: 100,
        render: () => t('alarm.alarmType.active'),
      },
      {
        key: 'eventTime',
        title: t('alarm.eventTime'),
        dataIndex: 'eventTime',
        width: 150,
        sorter: true,
        render: (v) => v ? new Date(String(v)).toLocaleString('zh-CN') : '-',
      },
      {
        key: 'updTime',
        title: t('alarm.updTime'),
        dataIndex: 'updTime',
        width: 150,
        sorter: true,
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
        key: 'alarmCount',
        title: t('alarm.alarmCount'),
        dataIndex: 'alarmCount',
        width: 100,
        sorter: true,
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
        key: 'batch-filter',
        label: t('alarm.filterAlarm'),
        icon: <FilterOutlined />,
        onClick: (keys) => handleFilter(keys as string[]),
      },
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
        onClick: (keys) => handleMarkRead(keys as string[]),
      },
    ],
    [handleAcknowledge, handleUnacknowledge, handleClear, handleFilter, handleMarkRead]
  );

  // 统计未确认未清除告警
  const unackCount = alarms.filter((a) => a.dealState === '0' || a.dealState === '1').length;

  return (
    <ListPageLayout
      title={t('nav.alarm.current')}
      extra={
        <Space>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 13 }}>
            <Badge status="success" />
            <span style={{ color: '#52C41A' }}>{t('common.realTimeConn')}</span>
          </span>
          {unackCount > 0 && (
            <Tag color="red" icon={<BellOutlined />}>
              {unackCount} {t('common.unacked')}
            </Tag>
          )}
        </Space>
      }
    >
      <FilterBar
        filterId="current-alarms"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

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
        defaultDensity="compact"
      />

      <AlarmDetail
        alarm={detailAlarm}
        open={detailOpen}
        onClose={handleCloseDetail}
      />
    </ListPageLayout>
  );
}
