import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Button, Card, Col, Form, Input, Modal, Row, Space, Statistic, Tag, Typography, App, Tree } from 'antd';
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

// 告警级别标签
const SEVERITY_LABEL: Record<string, string> = {
  critical: '严重',
  major: '主要',
  minor: '次要',
  warning: '警告',
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

// 自定义告警分组项
interface CustomAlarmGroup {
  id: string;
  name: string;
  alarmType: 'active' | 'historical';
  createdAt: string;
  stats?: {
    total: number;
    critical: number;
    major: number;
    minor: number;
    warning: number;
  };
}

// Mock 数据 - 自定义告警分组
const DEFAULT_GROUPS: CustomAlarmGroup[] = [
  { id: 'group-beijing', name: '北京告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 128, critical: 12, major: 35, minor: 48, warning: 33 } },
  { id: 'group-shanghai', name: '上海告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 256, critical: 24, major: 68, minor: 98, warning: 66 } },
  { id: 'group-tianjin', name: '天津告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 64, critical: 6, major: 18, minor: 24, warning: 16 } },
  { id: 'group-guangzhou', name: '广州告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 192, critical: 18, major: 52, minor: 72, warning: 50 } },
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
  const [addGroupModalOpen, setAddGroupModalOpen] = useState(false);
  const [addGroupForm] = Form.useForm<{ name: string }>();

  // 编辑分组弹窗状态
  const [editGroupModalOpen, setEditGroupModalOpen] = useState(false);
  const [editGroupId, setEditGroupId] = useState<string | null>(null);
  const [editGroupForm] = Form.useForm<{ name: string }>();

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

  const SEVERITY_LABEL_I18N: Record<string, string> = useMemo(() => ({
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

  // 查询参数
  const queryParams = useMemo(
    () => ({
      ...filterParams,
      page: currentPage,
      pageSize,
    }),
    [filterParams, currentPage, pageSize]
  );

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
        alarmType: 'active',
        createdAt: new Date().toISOString().split('T')[0],
      };
      setGroups((prev) => [...prev, newGroup]);
      setAddGroupModalOpen(false);
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
      editGroupForm.setFieldsValue({ name: group.name });
      setEditGroupModalOpen(true);
    }
  }, [groups, editGroupForm]);

  const handleSaveEditGroup = useCallback(async () => {
    try {
      const values = await editGroupForm.validateFields();
      setGroups((prev) =>
        prev.map((g) => (g.id === editGroupId ? { ...g, name: values.name } : g))
      );
      setEditGroupModalOpen(false);
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

  // 表格列 - 与活动告警保持一致，增加告警类型列
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
      key: 'neType',
      title: t('alarm.neType'),
      dataIndex: 'neType',
      width: 100,
      render: (val: string) => NE_TYPE_CONFIG[val] || val || '-',
    },
    {
      key: 'equipInfo',
      title: t('alarm.equipInfo'),
      dataIndex: 'equipInfo',
      width: 220,
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
      key: 'alarmType',
      title: '告警类型',
      dataIndex: 'alarmType',
      width: 100,
      render: (_val, record) => {
        // 根据分组类型或告警本身属性判断
        const type = isHistorical ? 'historical' : 'active';
        return (
          <Tag color={type === 'active' ? 'red' : 'default'}>
            {type === 'active' ? '活动告警' : '历史告警'}
          </Tag>
        );
      },
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
      key: 'updTime',
      title: t('alarm.updTime'),
      dataIndex: 'updTime',
      width: 150,
      render: (v) => (v ? new Date(String(v)).toLocaleString('zh-CN') : '-'),
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
      width: 80,
    },
    {
      key: 'dealMemo',
      title: t('alarm.dealMemo'),
      dataIndex: 'dealMemo',
      width: 100,
      ellipsis: true,
    },
  ], [t, SEVERITY_LABEL, handleShowDetail, isHistorical]);

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
            setAddGroupModalOpen(true);
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
          treeData={filteredGroups.map((group) => ({
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
          }))}
          selectedKeys={[selectedGroupId]}
          onSelect={(keys) => {
            const key = keys[0] as string | undefined;
            if (key) {
              setSelectedGroupId(key);
              // 切换分组时重置搜索条件和页码
              setFilterParams({});
              setCurrentPage(1);
              setSelectedRowKeys([]);
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

  // 右侧面板 - 使用 Card 分区
  const rightPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', background: '#f5f5f5', padding: 12, gap: 12 }}>
      {/* 标题卡片 */}
      <Card size="small" styles={{ body: { padding: '12px 16px' } }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <EnvironmentOutlined style={{ color: '#1890ff' }} />
            <Text strong style={{ fontSize: 16 }}>{selectedGroup?.name || '自定义告警'}</Text>
            <Tag color={isHistorical ? 'default' : 'red'}>
              {isHistorical ? <><ClockCircleOutlined /> 历史</> : <><AlertOutlined /> 活动</>}
            </Tag>
          </Space>
          <Space>
            <Button icon={<ExportOutlined />} onClick={() => setExportOpen(true)}>
              导出
            </Button>
          </Space>
        </div>
      </Card>

      {/* 统计卡片 */}
      <Card size="small" styles={{ body: { padding: '12px 16px' } }}>
        <Row gutter={24}>
          <Col span={4}>
            <Statistic title="总数" value={groupStats.total} valueStyle={{ fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="严重" value={groupStats.critical} valueStyle={{ color: '#E53935', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="主要" value={groupStats.major} valueStyle={{ color: '#FB8C00', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="次要" value={groupStats.minor} valueStyle={{ color: '#FDD835', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="警告" value={groupStats.warning} valueStyle={{ color: '#42A5F5', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic
              title="未确认"
              value={Math.floor(groupStats.total * 0.3)}
              valueStyle={{ color: '#E53935', fontSize: 20 }}
            />
          </Col>
        </Row>
      </Card>

      {/* 搜索和列表卡片 */}
      <Card size="small" styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}>
        <div style={{ padding: '12px 16px 0' }}>
          <FilterBar filterId={`custom-alarm-stats-${selectedGroupId}`} fields={FILTER_FIELDS} onSearch={handleSearch} onReset={handleReset} collapsedRows={1} />
        </div>
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <DataTable<Alarm>
            tableId={`custom-alarm-stats-table-${selectedGroupId}`}
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
      </Card>
    </div>
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

      {/* 添加分组弹窗 - 只显示分组名称 */}
      <Modal
        open={addGroupModalOpen}
        title="添加自定义告警分组"
        onCancel={() => setAddGroupModalOpen(false)}
        onOk={handleAddGroup}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form form={addGroupForm} layout="vertical" size="small">
          <Form.Item name="name" label="分组名称" rules={[{ required: true, message: '请输入分组名称' }]}>
            <Input placeholder="请输入分组名称，如：北京告警" maxLength={50} showCount autoFocus />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑分组弹窗 - 只显示分组名称 */}
      <Modal
        open={editGroupModalOpen}
        title="编辑自定义告警分组"
        onCancel={() => { setEditGroupModalOpen(false); setEditGroupId(null); }}
        onOk={handleSaveEditGroup}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form form={editGroupForm} layout="vertical" size="small">
          <Form.Item name="name" label="分组名称" rules={[{ required: true, message: '请输入分组名称' }]}>
            <Input placeholder="请输入分组名称" maxLength={50} showCount autoFocus />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
