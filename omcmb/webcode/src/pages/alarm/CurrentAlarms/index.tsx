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
import { useCurrentAlarms, useAcknowledgeAlarms, useClearAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@/types/alarm';
import type { AlarmFilter } from '@/types/alarm';
import AlarmDetail from '../AlarmDetail';
import ExportModal, { type ExportParams } from './ExportModal';
import ConfirmWithNoteModal from '../components/ConfirmWithNoteModal';

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
  '30000': 'alarm.eventType.communication',
  '30001': 'alarm.eventType.qualityOfService',
  '30002': 'alarm.eventType.processingError',
  '30003': 'alarm.eventType.device',
  '30004': 'alarm.eventType.environment',
  '30006': 'alarm.eventType.performance',
};

// 基站制式配置
const NE_TYPE_CONFIG: Record<string, string> = {
  'eNB': 'eNB',
  'gNB': 'gNB',
  'GSM': 'GSM',
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

// 自动刷新间隔选项
const AUTO_REFRESH_INTERVALS = [
  { label: '15秒', value: 15 },
  { label: '30秒', value: 30 },
  { label: '1分钟', value: 60 },
  { label: '5分钟', value: 300 },
];

// 统计项组件
function StatItem({
  label,
  value,
  color,
  active,
  onClick,
}: {
  label: string;
  value: number;
  color?: string;
  active?: boolean;
  onClick?: () => void;
}) {
  return (
    <div
      onClick={onClick}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        padding: '8px 16px',
        borderRadius: 6,
        cursor: onClick ? 'pointer' : 'default',
        background: active ? '#E6F4FF' : 'transparent',
        transition: 'all 0.2s',
        minWidth: 70,
      }}
    >
      <Text type="secondary" style={{ fontSize: 12 }}>{label}</Text>
      <Text strong style={{ fontSize: 20, color: color || '#1F1F1F', lineHeight: 1.2 }}>{value}</Text>
    </div>
  );
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

  // 实时统计（从查询结果计算）
  const realStats = useMemo(() => {
    const stats = { total, critical: 0, major: 0, minor: 0, warning: 0, unacked: 0 };
    rawAlarms.forEach((alarm) => {
      if (alarm.severity === 'critical') stats.critical++;
      else if (alarm.severity === 'major') stats.major++;
      else if (alarm.severity === 'minor') stats.minor++;
      else if (alarm.severity === 'warning') stats.warning++;
      if (alarm.dealState === '0') stats.unacked++;
    });
    return stats;
  }, [rawAlarms, total]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    const keyword = values.keyword as string;
    setFilterParams({
      severity: values.severity as AlarmFilter['severity'],
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      unread: values.unread as '0' | '1',
      dealState: values.dealState as AlarmFilter['dealState'],
      alarmIdentifier: keyword,
      alarmName: keyword,
      equipInfo: keyword,
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
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
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
            setSelectedRowKeys([]);
            refetch();
            message.success(t('common.unackSuccess'));
          } catch {
            message.error(t('common.unackFailed'));
          }
        },
      });
    },
    [refetch, t, message, modal]
  );

  const handleClear = useCallback(
    (ids: string[]) => {
      setClearTargetIds(ids);
      setClearModalOpen(true);
    },
    []
  );

  const handleClearConfirm = useCallback(
    async () => {
      setClearLoading(true);
      try {
        await clearAlarms.mutateAsync(clearTargetIds);
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


  const handleMarkRead = useCallback(
    () => {
      setSelectedRowKeys([]);
      refetch();
      message.success(t('common.markReadSuccess'));
    },
    [refetch, t, message]
  );

  // 导出告警
  const handleExport = useCallback(
    async (params: ExportParams) => {
      setExportLoading(true);
      try {
        const exportData = exportMode === 'selected' ? selectedRowKeys : undefined;
        console.log('Export params:', { ...params, mode: exportMode, selectedIds: exportData });
        void message.info(t('common.exportInProgress'));
        setExportOpen(false);
      } catch {
        message.error(t('common.exportFailed'));
      } finally {
        setExportLoading(false);
      }
    },
    [exportMode, selectedRowKeys, message, t]
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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    (_record: Alarm): 'critical' | 'major' | 'minor' | 'warning' | null => {
      return null;
    },
    []
  );

  const columns = useMemo(
    (): DataTableColumn<Alarm>[] => [
      {
        key: 'alarmId',
        title: t('alarm.alarmId'),
        dataIndex: 'alarmId',
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
      label: autoRefresh ? '关闭自动刷新' : '开启自动刷新',
      icon: <SyncOutlined spin={autoRefresh} />,
    },
    ...(autoRefresh ? AUTO_REFRESH_INTERVALS.map((opt) => ({
      key: `interval-${opt.value}`,
      label: opt.label,
    })) : []),
  ], [autoRefresh]);

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
      {/* 统计卡片 */}
      <Card
        size="small"
        bordered
        style={{ marginBottom: 12 }}
        styles={{ body: { padding: '12px 16px' } }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <StatItem
            label="总数"
            value={realStats.total}
            active={activeQuickFilter === 'all'}
            onClick={() => handleQuickFilter('all')}
          />
          <StatItem
            label="严重"
            value={realStats.critical}
            color="#E53935"
            active={activeQuickFilter === 'critical'}
            onClick={() => handleQuickFilter('critical')}
          />
          <StatItem
            label="主要"
            value={realStats.major}
            color="#FB8C00"
            active={activeQuickFilter === 'major'}
            onClick={() => handleQuickFilter('major')}
          />
          <StatItem
            label="次要"
            value={realStats.minor}
            color="#FDD835"
            active={activeQuickFilter === 'minor'}
            onClick={() => handleQuickFilter('minor')}
          />
          <StatItem
            label="警告"
            value={realStats.warning}
            color="#42A5F5"
            active={activeQuickFilter === 'warning'}
            onClick={() => handleQuickFilter('warning')}
          />
          <StatItem
            label="未确认"
            value={realStats.unacked}
            color="#E53935"
            active={activeQuickFilter === 'unacked'}
            onClick={() => handleQuickFilter('unacked')}
          />
          <StatItem
            label="未读"
            value={rawAlarms.filter(a => a.unread === '1').length}
            color="#722ED1"
            active={activeQuickFilter === 'unread'}
            onClick={() => handleQuickFilter('unread')}
          />
        </div>
      </Card>

      {/* 搜索卡片 */}
      <Card
        size="small"
        bordered
        style={{ marginBottom: 12 }}
        styles={{ body: { padding: '12px 16px 0' } }}
      >
        <FilterBar
          filterId="current-alarms"
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
          noDefaultStyle
        />
      </Card>

      {/* 列表卡片 */}
      <Card
        size="small"
        bordered
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
          defaultDensity="compact"
          showRowNumber
          rowNumberTitle="序号"
          scroll={{ y: 'calc(100vh - 510px)' }}
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
