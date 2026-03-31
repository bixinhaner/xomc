import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Button, Modal, Space, Tag, Typography, App, Tree, Input } from 'antd';
import {
  CheckOutlined,
  ClearOutlined,
  DeleteOutlined,
  ExportOutlined,
  EyeOutlined,
  FolderOutlined,
  MinusCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useCurrentAlarms, useHistoricalAlarms, useAcknowledgeAlarms, useClearAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@/types/alarm';
import type { AlarmFilter } from '@/types/alarm';
import AlarmDetail from '../AlarmDetail';
import ExportModal, { type ExportParams } from '../CurrentAlarms/ExportModal';
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

// 告警类型分组
type AlarmType = 'active' | 'historical';

// 一级节点数据（告警类型 + 严重级别分组）
const ALARM_TYPE_GROUPS: { id: AlarmType; name: string; icon: typeof FolderOutlined }[] = [
  { id: 'active', name: 'alarm.type.active', icon: FolderOutlined },
  { id: 'historical', name: 'alarm.type.historical', icon: FolderOutlined },
];

// 严重级别分组
const SEVERITY_GROUPS = [
  { id: 'all', name: 'common.all' },
  { id: 'critical', name: 'alarm.severity.critical' },
  { id: 'major', name: 'alarm.severity.major' },
  { id: 'minor', name: 'alarm.severity.minor' },
  { id: 'warning', name: 'alarm.severity.warning' },
];

export default function CustomAlarmStats() {
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

  // 左侧树状态
  const [selectedAlarmType, setSelectedAlarmType] = useState<AlarmType>('active');
  const [selectedSeverityId, setSelectedSeverityId] = useState<string>('all');
  const [searchText, setSearchText] = useState('');

  // 确认告警弹窗状态
  const [ackModalOpen, setAckModalOpen] = useState(false);
  const [ackTargetIds, setAckTargetIds] = useState<string[]>([]);
  const [ackLoading, setAckLoading] = useState(false);

  // 清除告警弹窗状态
  const [clearModalOpen, setClearModalOpen] = useState(false);
  const [clearTargetIds, setClearTargetIds] = useState<string[]>([]);
  const [clearLoading, setClearLoading] = useState(false);

  // 删除告警弹窗状态
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteTargetIds, setDeleteTargetIds] = useState<string[]>([]);
  const [deleteLoading, setDeleteLoading] = useState(false);

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

  // 根据选中的分组添加 severity 过滤
  const queryParams = useMemo(
    () => {
      const params: Parameters<typeof useCurrentAlarms>[0] = {
        ...filterParams,
        page: currentPage,
        pageSize,
      };
      // 根据选中的分组设置 severity 过滤
      if (selectedSeverityId !== 'all') {
        params.severity = [selectedSeverityId] as AlarmFilter['severity'];
      }
      return params;
    },
    [filterParams, currentPage, pageSize, selectedSeverityId]
  );

  // 根据告警类型使用不同的 hook
  const currentAlarmsQuery = useCurrentAlarms(selectedAlarmType === 'active' ? queryParams : {} as Parameters<typeof useCurrentAlarms>[0]);
  const historicalAlarmsQuery = useHistoricalAlarms(selectedAlarmType === 'historical' ? queryParams : {} as Parameters<typeof useHistoricalAlarms>[0]);

  const activeQuery = selectedAlarmType === 'active' ? currentAlarmsQuery : null;
  const historicalQuery = selectedAlarmType === 'historical' ? historicalAlarmsQuery : null;

  const { data: activeData, isLoading: isActiveLoading, refetch: refetchActive } = activeQuery || { data: undefined, isLoading: false, refetch: () => Promise.resolve() };
  const { data: historicalData, isLoading: isHistoricalLoading, refetch: refetchHistorical } = historicalQuery || { data: undefined, isLoading: false, refetch: () => Promise.resolve() };

  const isLoading = selectedAlarmType === 'active' ? isActiveLoading : isHistoricalLoading;
  const data = selectedAlarmType === 'active' ? activeData : historicalData;
  const refetch = selectedAlarmType === 'active' ? refetchActive : refetchHistorical;

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
        await acknowledgeAlarms.mutateAsync({ ids: ackTargetIds, note });
        setSelectedRowKeys([]);
        setAckModalOpen(false);
        void refetch();
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
            void refetch();
            message.success(t('common.unackSuccess'));
          } catch {
            message.error(t('common.unackFailed'));
          }
        },
      });
    },
    [refetch, t, modal, message]
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
        void refetch();
        message.success(t('common.clearSuccess'));
      } catch {
        message.error(t('common.clearFailed'));
      } finally {
        setClearLoading(false);
      }
    },
    [clearAlarms, clearTargetIds, refetch, t, message]
  );

  // 删除告警
  const handleDelete = useCallback(
    (ids: string[]) => {
      setDeleteTargetIds(ids);
      setDeleteModalOpen(true);
    },
    []
  );

  const handleDeleteConfirm = useCallback(
    async () => {
      setDeleteLoading(true);
      try {
        // TODO: 调用删除告警 API
        console.log('删除告警:', deleteTargetIds);
        setSelectedRowKeys([]);
        setDeleteModalOpen(false);
        void refetch();
        message.success(t('common.deleteSuccess'));
      } catch {
        message.error(t('common.deleteFailed'));
      } finally {
        setDeleteLoading(false);
      }
    },
    [deleteTargetIds, refetch, t, message]
  );

  const handleMarkRead = useCallback(
    (ids: string[]) => {
      setSelectedRowKeys([]);
      void refetch();
      message.success(t('common.markReadSuccess'));
    },
    [refetch, t, message]
  );

  // 导出告警
  const handleExport = useCallback(
    async (params: ExportParams) => {
      setExportLoading(true);
      try {
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

  // 批量操作 - 活动告警和历史告警有不同的操作
  const batchActions = useMemo(
    (): BatchAction[] => {
      const actions: BatchAction[] = [];

      // 活动告警操作
      if (selectedAlarmType === 'active') {
        actions.push(
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
          }
        );
      }

      // 删除操作 - 活动告警和历史告警都有
      actions.push({
        key: 'batch-delete',
        label: t('common.delete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (keys) => handleDelete(keys as string[]),
      });

      return actions;
    },
    [selectedAlarmType, handleAcknowledge, handleUnacknowledge, handleClear, handleMarkRead, handleDelete, t]
  );

  // 构建树形数据
  const treeData = useMemo(() => {
    return ALARM_TYPE_GROUPS.map((typeGroup) => ({
      key: typeGroup.id,
      title: (
        <span>
          <FolderOutlined style={{ marginRight: 6, color: '#1890FF' }} />
          {t(typeGroup.name)}
        </span>
      ),
      children: SEVERITY_GROUPS.map((severityGroup) => ({
        key: `${typeGroup.id}-${severityGroup.id}`,
        title: (
          <span>
            <FolderOutlined style={{ marginRight: 6, color: '#FA8C16' }} />
            {t(severityGroup.name)}
            <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
              ({severityGroup.id === 'all' ? total : alarms.filter((a) => a.severity === severityGroup.id).length})
            </Text>
          </span>
        ),
        isLeaf: true,
      })),
    }));
  }, [t, total, alarms]);

  // 当前选中的完整 key
  const selectedTreeKey = `${selectedAlarmType}-${selectedSeverityId}`;

  // 左侧树面板
  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div
        style={{
          padding: '12px 12px 8px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Text strong style={{ fontSize: 14 }}>
          {t('nav.alarm.customAlarm')}
        </Text>
      </div>
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
        <Input
          placeholder={t('common.search')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
          size="small"
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '8px 4px' }}>
        <Tree
          treeData={treeData.filter((node) =>
            searchText
              ? t(ALARM_TYPE_GROUPS.find((g) => g.id === node.key)?.name || '').toLowerCase().includes(searchText.toLowerCase()) ||
                node.children?.some((child) => {
                  const severityId = (child.key as string).split('-')[1];
                  return t(SEVERITY_GROUPS.find((g) => g.id === severityId)?.name || '').toLowerCase().includes(searchText.toLowerCase());
                })
              : true
          )}
          selectedKeys={[selectedTreeKey]}
          expandedKeys={[selectedAlarmType]}
          onExpand={(keys) => {
            // 保持展开状态
          }}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (key) {
              if (key.includes('-')) {
                // 子节点格式: "active-all", "historical-critical" 等
                const [alarmType, severityId] = key.split('-');
                setSelectedAlarmType(alarmType as AlarmType);
                setSelectedSeverityId(severityId);
              } else {
                // 父节点格式: "active", "historical"
                setSelectedAlarmType(key as AlarmType);
                setSelectedSeverityId('all');
              }
              setCurrentPage(1);
            }
          }}
          blockNode
          style={{ fontSize: 13 }}
        />
      </div>
    </div>
  );

  // 获取当前选中分组的显示名称
  const getCurrentTitle = useCallback(() => {
    const typeName = t(ALARM_TYPE_GROUPS.find((g) => g.id === selectedAlarmType)?.name || '');
    const severityName = selectedSeverityId === 'all'
      ? ''
      : ` - ${t(SEVERITY_GROUPS.find((g) => g.id === selectedSeverityId)?.name || '')}`;
    return `${typeName}${severityName}`;
  }, [selectedAlarmType, selectedSeverityId, t]);

  // 右侧面板
  const rightPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div
        style={{
          padding: '12px 16px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Text strong style={{ fontSize: 14 }}>
          {getCurrentTitle()}
        </Text>
        <Button type="primary" icon={<ExportOutlined />} onClick={() => setExportOpen(true)}>
          {t('common.export')}
        </Button>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        <FilterBar
          filterId="custom-alarm-stats"
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />
        <DataTable<Alarm>
          tableId="custom-alarm-stats-table"
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
      </div>
    </div>
  );

  return (
    <>
      <TreeListPageLayout tree={treePanel} defaultTreeWidth={220}>
        {rightPanel}
      </TreeListPageLayout>

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

      {/* 删除告警确认弹窗 */}
      <Modal
        open={deleteModalOpen}
        title={t('common.delete')}
        onCancel={() => setDeleteModalOpen(false)}
        onOk={handleDeleteConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true, loading: deleteLoading }}
      >
        <Text>{t('common.deleteConfirmMsg', { count: deleteTargetIds.length })}</Text>
      </Modal>
    </>
  );
}
