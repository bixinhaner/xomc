import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Button, Col, Drawer, Form, Input, Modal, Radio, Row, Select, Space, Statistic, Tag, Typography, App, Tree } from 'antd';
import {
  AlertOutlined,
  CheckOutlined,
  ClearOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  EnvironmentOutlined,
  ExportOutlined,
  EyeOutlined,
  FilterOutlined,
  MinusCircleOutlined,
  PlusOutlined,
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

// 告警级别颜色
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  critical: { color: '#E53935', bgColor: '#FFEBEE' },
  major: { color: '#FB8C00', bgColor: '#FFF3E0' },
  minor: { color: '#FDD835', bgColor: '#FFFDE7' },
  warning: { color: '#42A5F5', bgColor: '#E3F2FD' },
};

// 告警状态配置
const DEAL_STATE_CONFIG: Record<DealState, { label: string; color: string }> = {
  '0': { label: 'alarm.dealState.unconfirmedUncleared', color: '#E53935' },
  '1': { label: 'alarm.dealState.confirmedUncleared', color: '#FB8C00' },
  '2': { label: 'alarm.dealState.unconfirmedCleared', color: '#42A5F5' },
  '3': { label: 'alarm.dealState.confirmedCleared', color: '#67D972' },
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

// 筛选规则 - 匹配模式
type MatchMode = 'contains' | 'startsWith' | 'endsWith' | 'equals';

// 筛选规则项
interface FilterRule {
  field: 'equipInfo' | 'alarmName' | 'alarmIdentifier';  // 筛选字段
  mode: MatchMode;                                        // 匹配模式
  value: string;                                          // 匹配值
}

// 自定义告警分组项
interface CustomAlarmGroup {
  id: string;
  name: string;                    // 分组名称，如"北京告警"、"上海告警"
  alarmType: 'active' | 'historical';
  filterRules: FilterRule[];       // 筛选规则
  severity?: string[];             // 告警级别过滤
  createdAt: string;
  // 统计信息
  stats?: {
    total: number;
    critical: number;
    major: number;
    minor: number;
    warning: number;
  };
}

// Mock 数据 - 自定义告警分组（按地区示例）
const DEFAULT_GROUPS: CustomAlarmGroup[] = [
  {
    id: 'group-beijing',
    name: '北京告警',
    alarmType: 'active',
    filterRules: [{ field: 'equipInfo', mode: 'contains', value: '北京' }],
    createdAt: '2026-03-01',
    stats: { total: 128, critical: 12, major: 35, minor: 48, warning: 33 },
  },
  {
    id: 'group-shanghai',
    name: '上海告警',
    alarmType: 'active',
    filterRules: [{ field: 'equipInfo', mode: 'contains', value: '上海' }],
    createdAt: '2026-03-01',
    stats: { total: 256, critical: 24, major: 68, minor: 98, warning: 66 },
  },
  {
    id: 'group-tianjin',
    name: '天津告警',
    alarmType: 'active',
    filterRules: [{ field: 'equipInfo', mode: 'contains', value: '天津' }],
    createdAt: '2026-03-01',
    stats: { total: 64, critical: 6, major: 18, minor: 24, warning: 16 },
  },
  {
    id: 'group-history',
    name: '历史告警',
    alarmType: 'historical',
    filterRules: [],
    createdAt: '2026-03-01',
    stats: { total: 1024, critical: 89, major: 256, minor: 412, warning: 267 },
  },
];

// 匹配模式选项
const MATCH_MODE_OPTIONS = [
  { label: '包含', value: 'contains' },
  { label: '开头为', value: 'startsWith' },
  { label: '结尾为', value: 'endsWith' },
  { label: '等于', value: 'equals' },
];

// 筛选字段选项
const FILTER_FIELD_OPTIONS = [
  { label: '告警源/设备信息', value: 'equipInfo' },
  { label: '告警名称', value: 'alarmName' },
  { label: '告警标识', value: 'alarmIdentifier' },
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
  const [groups, setGroups] = useState<CustomAlarmGroup[]>(DEFAULT_GROUPS);
  const [selectedGroupId, setSelectedGroupId] = useState<string>('group-beijing');
  const [searchText, setSearchText] = useState('');

  // 添加分组弹窗状态
  const [addGroupDrawerOpen, setAddGroupDrawerOpen] = useState(false);
  const [addGroupForm] = Form.useForm<Omit<CustomAlarmGroup, 'id' | 'createdAt' | 'stats'>>();

  // 编辑分组弹窗状态
  const [editGroupDrawerOpen, setEditGroupDrawerOpen] = useState(false);
  const [editGroupId, setEditGroupId] = useState<string | null>(null);
  const [editGroupForm] = Form.useForm<Partial<CustomAlarmGroup>>();

  // 确认/清除/删除弹窗状态
  const [ackModalOpen, setAckModalOpen] = useState(false);
  const [ackTargetIds, setAckTargetIds] = useState<string[]>([]);
  const [ackLoading, setAckLoading] = useState(false);
  const [clearModalOpen, setClearModalOpen] = useState(false);
  const [clearTargetIds, setClearTargetIds] = useState<string[]>([]);
  const [clearLoading, setClearLoading] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteTargetIds, setDeleteTargetIds] = useState<string[]>([]);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // 当前选中的分组
  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );

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

  // 查询参数 - 合并分组的筛选规则
  const queryParams = useMemo(() => {
    const baseParams: Record<string, unknown> = {
      ...filterParams,
      page: currentPage,
      pageSize,
    };

    // 应用分组的筛选规则
    if (selectedGroup?.filterRules && selectedGroup.filterRules.length > 0) {
      selectedGroup.filterRules.forEach((rule) => {
        if (rule.value.trim()) {
          // 简化处理：所有匹配模式都转为包含查询
          baseParams[rule.field] = rule.value;
        }
      });
    }

    // 应用告警级别过滤
    if (selectedGroup?.severity && selectedGroup.severity.length > 0) {
      baseParams.severity = selectedGroup.severity;
    }

    return baseParams;
  }, [filterParams, currentPage, pageSize, selectedGroup]);

  // 根据分组类型使用不同的 hook
  const isHistorical = selectedGroup?.alarmType === 'historical';
  const currentAlarmsQuery = useCurrentAlarms(!isHistorical ? queryParams : ({} as Parameters<typeof useCurrentAlarms>[0]));
  const historicalAlarmsQuery = useHistoricalAlarms(isHistorical ? queryParams : ({} as Parameters<typeof useHistoricalAlarms>[0]));

  const activeResult = !isHistorical ? currentAlarmsQuery : { data: undefined, isLoading: false, refetch: () => Promise.resolve() };
  const historicalResult = isHistorical ? historicalAlarmsQuery : { data: undefined, isLoading: false, refetch: () => Promise.resolve() };

  const isLoading = activeResult.isLoading || historicalResult.isLoading;
  const data = !isHistorical ? activeResult.data : historicalResult.data;
  const refetch = !isHistorical ? activeResult.refetch : historicalResult.refetch;

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

  // 过滤分组列表
  const filteredGroups = useMemo(() => {
    if (!searchText.trim()) return groups;
    return groups.filter((g) =>
      g.name.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [groups, searchText]);

  // 添加分组
  const handleAddGroup = useCallback(async () => {
    try {
      const values = await addGroupForm.validateFields();
      const newGroup: CustomAlarmGroup = {
        id: `group-${Date.now()}`,
        name: values.name,
        alarmType: values.alarmType,
        filterRules: values.filterRules || [],
        severity: values.severity,
        createdAt: new Date().toISOString().split('T')[0],
      };
      setGroups((prev) => [...prev, newGroup]);
      setAddGroupDrawerOpen(false);
      addGroupForm.resetFields();
      message.success(t('common.success'));
    } catch {
      // validation error
    }
  }, [addGroupForm, message, t]);

  // 编辑分组
  const handleEditGroup = useCallback((groupId: string) => {
    const group = groups.find((g) => g.id === groupId);
    if (group) {
      setEditGroupId(groupId);
      editGroupForm.setFieldsValue({
        name: group.name,
        alarmType: group.alarmType,
        filterRules: group.filterRules.length > 0 ? group.filterRules : [undefined],
        severity: group.severity,
      });
      setEditGroupDrawerOpen(true);
    }
  }, [groups, editGroupForm]);

  const handleSaveEditGroup = useCallback(async () => {
    try {
      const values = await editGroupForm.validateFields();
      setGroups((prev) =>
        prev.map((g) => (g.id === editGroupId ? {
          ...g,
          name: values.name || g.name,
          alarmType: values.alarmType || g.alarmType,
          filterRules: values.filterRules || g.filterRules,
          severity: values.severity,
        } : g))
      );
      setEditGroupDrawerOpen(false);
      setEditGroupId(null);
      editGroupForm.resetFields();
      message.success(t('common.success'));
    } catch {
      // validation error
    }
  }, [editGroupId, editGroupForm, message, t]);

  // 删除分组
  const handleDeleteGroup = useCallback((groupId: string) => {
    const group = groups.find((g) => g.id === groupId);
    if (groups.length <= 1) {
      message.warning('至少保留一个分组');
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      content: `确定要删除分组「${group?.name}」吗？`,
      okText: t('common.confirm'),
      okType: 'danger',
      onOk: () => {
        setGroups((prev) => prev.filter((g) => g.id !== groupId));
        if (selectedGroupId === groupId) {
          setSelectedGroupId(groups.find((g) => g.id !== groupId)?.id || '');
        }
        message.success(t('common.deleteSuccess'));
      },
    });
  }, [groups, selectedGroupId, modal, message, t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    const keyword = values.keyword as string;
    setFilterParams({
      severity: values.severity as AlarmFilter['severity'],
      eventType: values.eventType as AlarmFilter['eventType'],
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

  const handleAcknowledge = useCallback((ids: string[]) => {
    setAckTargetIds(ids);
    setAckModalOpen(true);
  }, []);

  const handleAcknowledgeConfirm = useCallback(async (note: string) => {
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
  }, [acknowledgeAlarms, ackTargetIds, refetch, t, message]);

  const handleUnacknowledge = useCallback((ids: string[]) => {
    modal.confirm({
      title: t('alarm.unacknowledge'),
      content: t('common.unackConfirmMsg', { count: ids.length }),
      okText: t('common.confirm'),
      onOk: async () => {
        setSelectedRowKeys([]);
        void refetch();
        message.success(t('common.unackSuccess'));
      },
    });
  }, [refetch, t, modal, message]);

  const handleClear = useCallback((ids: string[]) => {
    setClearTargetIds(ids);
    setClearModalOpen(true);
  }, []);

  const handleClearConfirm = useCallback(async (note: string) => {
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
  }, [clearAlarms, clearTargetIds, refetch, t, message]);

  const handleDelete = useCallback((ids: string[]) => {
    setDeleteTargetIds(ids);
    setDeleteModalOpen(true);
  }, []);

  const handleDeleteConfirm = useCallback(async () => {
    setDeleteLoading(true);
    try {
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
  }, [deleteTargetIds, refetch, t, message]);

  const handleMarkRead = useCallback((ids: string[]) => {
    setSelectedRowKeys([]);
    void refetch();
    message.success(t('common.markReadSuccess'));
  }, [refetch, t, message]);

  const handleExport = useCallback(async (params: ExportParams) => {
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
  }, [message, t]);

  const handleShowDetail = useCallback((alarm: Alarm) => {
    setDetailAlarm(alarm);
    setDetailOpen(true);
  }, []);

  const handleCloseDetail = useCallback(() => {
    setDetailOpen(false);
    setDetailAlarm(null);
  }, []);

  const columns = useMemo((): DataTableColumn<Alarm>[] => [
    {
      key: 'alarmId',
      title: t('alarm.alarmId'),
      dataIndex: 'alarmId',
      width: 100,
      render: (val, record) => (
        <Space size={4}>
          {record.unread === '1' && <Badge status="error" style={{ marginLeft: -4 }} />}
          <Button type="link" size="small" style={{ padding: 0, height: 'auto' }} onClick={() => handleShowDetail(record)}>
            {val}
          </Button>
        </Space>
      ),
    },
    {
      key: 'severity',
      title: t('alarm.severity'),
      dataIndex: 'severity',
      width: 80,
      render: (_val, record) => {
        const config = SEVERITY_CONFIG[record.severity] || SEVERITY_CONFIG.warning;
        return (
          <Tag style={{ color: config.color, backgroundColor: config.bgColor, border: 'none' }}>
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
      key: 'equipInfo',
      title: t('alarm.equipInfo'),
      dataIndex: 'equipInfo',
      width: 200,
      ellipsis: true,
    },
    {
      key: 'eventType',
      title: t('alarm.eventType'),
      dataIndex: 'eventType',
      width: 120,
      ellipsis: true,
      render: (val: EventType) => t(EVENT_TYPE_CONFIG[val] || 'common.unknown'),
    },
    {
      key: 'dealState',
      title: t('alarm.dealState'),
      dataIndex: 'dealState',
      width: 150,
      ellipsis: true,
      render: (val: DealState) => {
        const config = DEAL_STATE_CONFIG[val];
        return <span style={{ color: config?.color || '#666' }}>{t(config?.label || 'common.unknown')}</span>;
      },
    },
    {
      key: 'eventTime',
      title: t('alarm.eventTime'),
      dataIndex: 'eventTime',
      width: 150,
      render: (v) => (v ? new Date(String(v)).toLocaleString('zh-CN') : '-'),
    },
    {
      key: 'alarmCount',
      title: t('alarm.alarmCount'),
      dataIndex: 'alarmCount',
      width: 80,
    },
  ], [t, SEVERITY_LABEL, handleShowDetail]);

  const batchActions = useMemo((): BatchAction[] => {
    const actions: BatchAction[] = [];
    if (!isHistorical) {
      actions.push(
        { key: 'batch-ack', label: t('alarm.acknowledge'), icon: <CheckOutlined />, onClick: (keys) => handleAcknowledge(keys as string[]) },
        { key: 'batch-unack', label: t('alarm.unacknowledge'), icon: <MinusCircleOutlined />, onClick: (keys) => handleUnacknowledge(keys as string[]) },
        { key: 'batch-clear', label: t('alarm.clear'), icon: <ClearOutlined />, danger: true, onClick: (keys) => handleClear(keys as string[]) },
        { key: 'batch-read', label: t('alarm.markRead'), icon: <EyeOutlined />, onClick: (keys) => handleMarkRead(keys as string[]) }
      );
    }
    actions.push({ key: 'batch-delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: (keys) => handleDelete(keys as string[]) });
    return actions;
  }, [isHistorical, handleAcknowledge, handleUnacknowledge, handleClear, handleMarkRead, handleDelete, t]);

  // 计算统计信息
  const groupStats = selectedGroup?.stats || { total, critical: 0, major: 0, minor: 0, warning: 0 };

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
          自定义告警分组
        </Text>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => {
            addGroupForm.resetFields();
            addGroupForm.setFieldsValue({ alarmType: 'active', filterRules: [{}] });
            setAddGroupDrawerOpen(true);
          }}
        >
          添加
        </Button>
      </div>
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
        <Input
          placeholder="搜索分组"
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
          size="small"
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '8px 4px' }}>
        <Tree
          treeData={filteredGroups.map((group) => {
            const stats = group.stats;
            return {
              key: group.id,
              title: (
                <div
                  className="alarm-group-node"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    width: '100%',
                    padding: '2px 0',
                  }}
                >
                  <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'flex', alignItems: 'center' }}>
                    <EnvironmentOutlined style={{ marginRight: 6, color: '#1890ff' }} />
                    <span>{group.name}</span>
                  </span>
                  {stats && stats.total > 0 && (
                    <Badge
                      count={stats.total}
                      size="small"
                      style={{ marginRight: 8, minWidth: 20 }}
                    />
                  )}
                  <Space size={0} className="node-actions" style={{ opacity: 0, transition: 'opacity 0.2s' }}>
                    <Button
                      type="text"
                      size="small"
                      icon={<EditOutlined />}
                      onClick={(e) => { e.stopPropagation(); handleEditGroup(group.id); }}
                      style={{ flexShrink: 0, padding: '0 4px' }}
                    />
                    <Button
                      type="text"
                      size="small"
                      icon={<DeleteOutlined />}
                      onClick={(e) => { e.stopPropagation(); handleDeleteGroup(group.id); }}
                      style={{ flexShrink: 0, padding: '0 4px' }}
                      danger
                    />
                  </Space>
                </div>
              ),
              isLeaf: true,
            };
          })}
          selectedKeys={[selectedGroupId]}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (key) {
              setSelectedGroupId(key);
              setCurrentPage(1);
              setFilterParams({}); // 切换分组时重置筛选
            }
          }}
          blockNode
          style={{ fontSize: 13 }}
        />
      </div>
      <style>{`
        .alarm-group-node:hover .node-actions { opacity: 1 !important; }
      `}</style>
    </div>
  );

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
          {selectedGroup?.name || '自定义告警'}
        </Text>
        <Space>
          <Button icon={<ExportOutlined />} onClick={() => setExportOpen(true)}>
            导出
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', background: '#fafafa' }}>
        <Row gutter={16}>
          <Col span={4}>
            <Statistic title="总数" value={groupStats.total} />
          </Col>
          <Col span={4}>
            <Statistic title="严重" value={groupStats.critical} valueStyle={{ color: '#E53935', fontSize: 18 }} />
          </Col>
          <Col span={4}>
            <Statistic title="主要" value={groupStats.major} valueStyle={{ color: '#FB8C00', fontSize: 18 }} />
          </Col>
          <Col span={4}>
            <Statistic title="次要" value={groupStats.minor} valueStyle={{ color: '#FDD835', fontSize: 18 }} />
          </Col>
          <Col span={4}>
            <Statistic title="警告" value={groupStats.warning} valueStyle={{ color: '#42A5F5', fontSize: 18 }} />
          </Col>
          <Col span={4}>
            <Statistic
              title="类型"
              value={isHistorical ? '历史' : '活动'}
              prefix={isHistorical ? <ClockCircleOutlined /> : <AlertOutlined />}
              valueStyle={{ fontSize: 16 }}
            />
          </Col>
        </Row>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        <FilterBar filterId="custom-alarm-stats" fields={FILTER_FIELDS} onSearch={handleSearch} onReset={handleReset} collapsedRows={1} />
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
          onPageChange={(page, size) => { setCurrentPage(page); setPageSize(size); }}
          batchActions={batchActions}
          onRefresh={() => void refetch()}
          defaultDensity="compact"
        />
      </div>
    </div>
  );

  // 渲染筛选规则表单项
  const renderFilterRulesFormItems = (form: typeof addGroupForm) => (
    <Form.List name="filterRules">
      {(fields, { add, remove }) => (
        <div>
          <div style={{ marginBottom: 8, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              <FilterOutlined style={{ marginRight: 4 }} />
              筛选规则
            </Text>
            <Button type="dashed" size="small" onClick={() => add({ field: 'equipInfo', mode: 'contains', value: '' })} icon={<PlusOutlined />}>
              添加规则
            </Button>
          </div>
          {fields.map(({ key, name, ...restField }) => (
            <Row key={key} gutter={8} style={{ marginBottom: 8 }}>
              <Col span={7}>
                <Form.Item {...restField} name={[name, 'field']} noStyle>
                  <Select size="small" options={FILTER_FIELD_OPTIONS} placeholder="字段" />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item {...restField} name={[name, 'mode']} noStyle>
                  <Select size="small" options={MATCH_MODE_OPTIONS} placeholder="匹配" />
                </Form.Item>
              </Col>
              <Col span={9}>
                <Form.Item {...restField} name={[name, 'value']} noStyle>
                  <Input size="small" placeholder="值" />
                </Form.Item>
              </Col>
              <Col span={2}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
              </Col>
            </Row>
          ))}
          {fields.length === 0 && (
            <Text type="secondary" style={{ fontSize: 12, color: '#999' }}>
              暂无筛选规则，将显示所有告警
            </Text>
          )}
        </div>
      )}
    </Form.List>
  );

  return (
    <>
      <TreeListPageLayout tree={treePanel} defaultTreeWidth={220}>
        {rightPanel}
      </TreeListPageLayout>

      <AlarmDetail alarm={detailAlarm} open={detailOpen} onClose={handleCloseDetail} />

      <ExportModal open={exportOpen} onClose={() => setExportOpen(false)} onConfirm={handleExport} confirmLoading={exportLoading} />

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

      {/* 添加分组抽屉 */}
      <Drawer
        title="添加自定义告警分组"
        open={addGroupDrawerOpen}
        onClose={() => setAddGroupDrawerOpen(false)}
        width={480}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setAddGroupDrawerOpen(false)}>取消</Button>
              <Button type="primary" onClick={handleAddGroup}>确定</Button>
            </Space>
          </div>
        }
      >
        <Form form={addGroupForm} layout="vertical" size="small">
          <Form.Item name="name" label="分组名称" rules={[{ required: true, message: '请输入分组名称' }]}>
            <Input placeholder="如：北京告警、上海告警" maxLength={50} showCount />
          </Form.Item>
          <Form.Item name="alarmType" label="告警类型" rules={[{ required: true }]} initialValue="active">
            <Radio.Group>
              <Radio value="active">活动告警</Radio>
              <Radio value="historical">历史告警</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="severity" label="告警级别">
            <Select mode="multiple" placeholder="选择告警级别（可多选）" allowClear>
              <Select.Option value="critical">严重</Select.Option>
              <Select.Option value="major">主要</Select.Option>
              <Select.Option value="minor">次要</Select.Option>
              <Select.Option value="warning">警告</Select.Option>
            </Select>
          </Form.Item>
          {renderFilterRulesFormItems(addGroupForm)}
        </Form>
      </Drawer>

      {/* 编辑分组抽屉 */}
      <Drawer
        title="编辑自定义告警分组"
        open={editGroupDrawerOpen}
        onClose={() => { setEditGroupDrawerOpen(false); setEditGroupId(null); }}
        width={480}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setEditGroupDrawerOpen(false)}>取消</Button>
              <Button type="primary" onClick={handleSaveEditGroup}>确定</Button>
            </Space>
          </div>
        }
      >
        <Form form={editGroupForm} layout="vertical" size="small">
          <Form.Item name="name" label="分组名称" rules={[{ required: true, message: '请输入分组名称' }]}>
            <Input placeholder="如：北京告警、上海告警" maxLength={50} showCount />
          </Form.Item>
          <Form.Item name="alarmType" label="告警类型" rules={[{ required: true }]}>
            <Radio.Group>
              <Radio value="active">活动告警</Radio>
              <Radio value="historical">历史告警</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="severity" label="告警级别">
            <Select mode="multiple" placeholder="选择告警级别（可多选）" allowClear>
              <Select.Option value="critical">严重</Select.Option>
              <Select.Option value="major">主要</Select.Option>
              <Select.Option value="minor">次要</Select.Option>
              <Select.Option value="warning">警告</Select.Option>
            </Select>
          </Form.Item>
          {renderFilterRulesFormItems(editGroupForm)}
        </Form>
      </Drawer>
    </>
  );
}
