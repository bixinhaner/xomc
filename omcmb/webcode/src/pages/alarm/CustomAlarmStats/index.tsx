import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge, Button, Card, Col, Dropdown, Form, Input, Menu, Modal, Radio, Row, Select, Space, Statistic, Tag, Typography, App, Tree, message, Tooltip } from 'antd';
import {
  AlertOutlined,
  CalendarOutlined,
  CheckOutlined,
  ClearOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EnvironmentOutlined,
  ExportOutlined,
  EyeOutlined,
  FilterOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
  SearchOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import LineChart from '@/components/Charts/LineChart';
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

// 快捷时间选项
const QUICK_TIME_OPTIONS = [
  { label: '今日', value: 'today' },
  { label: '昨日', value: 'yesterday' },
  { label: '本周', value: 'thisWeek' },
  { label: '本月', value: 'thisMonth' },
  { label: '最近7天', value: 'last7days' },
  { label: '最近30天', value: 'last30days' },
];

// 快捷筛选选项
const QUICK_FILTER_OPTIONS = [
  { label: '全部', key: 'all' },
  { label: '严重告警', key: 'critical', severity: ['critical'] },
  { label: '未确认', key: 'unacked', dealState: ['0'] },
  { label: '未读', key: 'unread', unread: '1' },
];

// 筛选模板存储 key
const FILTER_TEMPLATES_KEY = 'custom-alarm-filter-templates';
const LAST_FILTER_KEY = 'custom-alarm-last-filter';

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

// 筛选模板
interface FilterTemplate {
  id: string;
  name: string;
  params: AlarmFilter;
  createdAt: string;
}

// Mock 数据 - 自定义告警分组
const DEFAULT_GROUPS: CustomAlarmGroup[] = [
  { id: 'group-beijing', name: '北京告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 128, critical: 12, major: 35, minor: 48, warning: 33 } },
  { id: 'group-shanghai', name: '上海告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 256, critical: 24, major: 68, minor: 98, warning: 66 } },
  { id: 'group-tianjin', name: '天津告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 64, critical: 6, major: 18, minor: 24, warning: 16 } },
  { id: 'group-guangzhou', name: '广州告警', alarmType: 'active', createdAt: '2026-03-01', stats: { total: 192, critical: 18, major: 52, minor: 72, warning: 50 } },
];

// 生成趋势图数据
function generateTrendData(): { dates: string[]; series: { name: string; data: number[]; color?: string }[] } {
  const dates: string[] = [];
  const criticalData: number[] = [];
  const majorData: number[] = [];
  const minorData: number[] = [];
  const warningData: number[] = [];

  for (let i = 6; i >= 0; i--) {
    const date = dayjs().subtract(i, 'day');
    dates.push(date.format('MM-DD'));
    criticalData.push(Math.floor(Math.random() * 20) + 5);
    majorData.push(Math.floor(Math.random() * 40) + 15);
    minorData.push(Math.floor(Math.random() * 60) + 20);
    warningData.push(Math.floor(Math.random() * 50) + 10);
  }

  return {
    dates,
    series: [
      { name: '严重', data: criticalData, color: '#E53935' },
      { name: '主要', data: majorData, color: '#FB8C00' },
      { name: '次要', data: minorData, color: '#FDD835' },
      { name: '警告', data: warningData, color: '#42A5F5' },
    ],
  };
}

// 快捷时间转日期范围
function quickTimeToRange(value: string): [string, string] | undefined {
  const now = dayjs();
  switch (value) {
    case 'today':
      return [now.startOf('day').toISOString(), now.endOf('day').toISOString()];
    case 'yesterday':
      const yesterday = now.subtract(1, 'day');
      return [yesterday.startOf('day').toISOString(), yesterday.endOf('day').toISOString()];
    case 'thisWeek':
      return [now.startOf('week').toISOString(), now.endOf('week').toISOString()];
    case 'thisMonth':
      return [now.startOf('month').toISOString(), now.endOf('month').toISOString()];
    case 'last7days':
      return [now.subtract(7, 'day').startOf('day').toISOString(), now.endOf('day').toISOString()];
    case 'last30days':
      return [now.subtract(30, 'day').startOf('day').toISOString(), now.endOf('day').toISOString()];
    default:
      return undefined;
  }
}

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
  const [exportMode, setExportMode] = useState<'all' | 'selected'>('all');

  // 左侧树状态
  const [groups, setGroups] = useState<CustomAlarmGroup[]>(DEFAULT_GROUPS);
  const [selectedGroupId, setSelectedGroupId] = useState<string>('group-beijing');
  const [searchText, setSearchText] = useState('');

  // 快捷筛选状态
  const [activeQuickFilter, setActiveQuickFilter] = useState<string>('all');
  const [activeQuickTime, setActiveQuickTime] = useState<string | undefined>(undefined);

  // 筛选模板状态
  const [filterTemplates, setFilterTemplates] = useState<FilterTemplate[]>([]);
  const [saveTemplateModalOpen, setSaveTemplateModalOpen] = useState(false);
  const [templateForm] = Form.useForm<{ name: string }>();

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

  // 趋势图数据
  const trendData = useMemo(() => generateTrendData(), []);

  // 加载筛选模板
  useEffect(() => {
    try {
      const saved = localStorage.getItem(FILTER_TEMPLATES_KEY);
      if (saved) {
        setFilterTemplates(JSON.parse(saved));
      }
    } catch {
      // ignore
    }
  }, []);

  // 加载上次筛选条件
  useEffect(() => {
    try {
      const saved = localStorage.getItem(LAST_FILTER_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        setFilterParams(parsed);
      }
    } catch {
      // ignore
    }
  }, []);

  // 保存筛选条件
  useEffect(() => {
    try {
      localStorage.setItem(LAST_FILTER_KEY, JSON.stringify(filterParams));
    } catch {
      // ignore
    }
  }, [filterParams]);

  // 当前选中的分组
  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId),
    [groups, selectedGroupId]
  );

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

  // 根据分组名称提取地区关键词，用于过滤告警
  const groupRegion = useMemo(() => {
    if (!selectedGroup?.name) return '';
    const match = selectedGroup.name.match(/^(.+?)告警$/);
    return match ? match[1] : '';
  }, [selectedGroup?.name]);

  // 查询参数 - 包含分组ID和地区过滤，切换分组时触发重新查询
  const queryParams = useMemo(
    () => ({
      ...filterParams,
      page: currentPage,
      pageSize,
      groupId: selectedGroupId,
      ...(groupRegion && !filterParams.equipInfo ? { equipInfo: groupRegion } : {}),
    }),
    [filterParams, currentPage, pageSize, selectedGroupId, groupRegion]
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

  // 过滤分组列表
  const filteredGroups = useMemo(() => {
    if (!searchText.trim()) return groups;
    return groups.filter((g) =>
      g.name.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [groups, searchText]);

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
        const { severity, dealState, unread, ...rest } = prev as any;
        return rest;
      });
    }
    setCurrentPage(1);
  }, []);

  // 快捷时间处理
  const handleQuickTime = useCallback((value: string) => {
    setActiveQuickTime(value);
    const range = quickTimeToRange(value);
    if (range) {
      setFilterParams((prev) => ({
        ...prev,
        timeRange: range,
      }));
    } else {
      setFilterParams((prev) => {
        const { timeRange, ...rest } = prev as any;
        return rest;
      });
    }
    setCurrentPage(1);
  }, []);

  // 保存筛选模板
  const handleSaveTemplate = useCallback(async () => {
    try {
      const values = await templateForm.validateFields();
      const newTemplate: FilterTemplate = {
        id: `template-${Date.now()}`,
        name: values.name,
        params: filterParams,
        createdAt: new Date().toISOString(),
      };
      const updated = [...filterTemplates, newTemplate];
      setFilterTemplates(updated);
      localStorage.setItem(FILTER_TEMPLATES_KEY, JSON.stringify(updated));
      setSaveTemplateModalOpen(false);
      templateForm.resetFields();
      message.success('模板保存成功');
    } catch {
      // validation error
    }
  }, [filterParams, filterTemplates, templateForm, message]);

  // 应用筛选模板
  const handleApplyTemplate = useCallback((template: FilterTemplate) => {
    setFilterParams(template.params);
    setCurrentPage(1);
    message.success(`已应用模板「${template.name}」`);
  }, [message]);

  // 删除筛选模板
  const handleDeleteTemplate = useCallback((templateId: string) => {
    const updated = filterTemplates.filter((t) => t.id !== templateId);
    setFilterTemplates(updated);
    localStorage.setItem(FILTER_TEMPLATES_KEY, JSON.stringify(updated));
    message.success('模板已删除');
  }, [filterTemplates, message]);

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
    setFilterParams((prev) => ({
      ...prev,
      severity: values.severity as AlarmFilter['severity'],
      eventType: values.eventType as AlarmFilter['eventType'],
      neType: values.neType as string,
      unread: values.unread as '0' | '1',
      dealState: values.dealState as AlarmFilter['dealState'],
      alarmIdentifier: keyword,
      alarmName: keyword,
      equipInfo: keyword,
    }));
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    setActiveQuickFilter('all');
    setActiveQuickTime(undefined);
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

  // 导出处理 - 支持导出选中数据
  const handleExport = useCallback(async (params: ExportParams) => {
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
  }, [exportMode, selectedRowKeys, message, t]);

  const handleShowDetail = useCallback((alarm: Alarm) => {
    setDetailAlarm(alarm);
    setDetailOpen(true);
  }, []);

  const handleCloseDetail = useCallback(() => {
    setDetailOpen(false);
    setDetailAlarm(null);
  }, []);

  // 表格列
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
      render: () => (
        <Tag color={isHistorical ? 'default' : 'red'}>
          {isHistorical ? '历史告警' : '活动告警'}
        </Tag>
      ),
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
  ], [t, handleShowDetail, isHistorical]);

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
              setFilterParams({});
              setCurrentPage(1);
              setSelectedRowKeys([]);
              setActiveQuickFilter('all');
              setActiveQuickTime(undefined);
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
            <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
              刷新
            </Button>
            <Dropdown
              menu={{
                items: [
                  { key: 'all', label: '导出全部', icon: <DownloadOutlined /> },
                  { key: 'selected', label: `导出选中 (${selectedRowKeys.length})`, icon: <DownloadOutlined />, disabled: selectedRowKeys.length === 0 },
                ],
                onClick: ({ key }) => {
                  setExportMode(key as 'all' | 'selected');
                  setExportOpen(true);
                },
              }}
            >
              <Button icon={<ExportOutlined />}>
                导出
              </Button>
            </Dropdown>
          </Space>
        </div>
      </Card>

      {/* 统计卡片 - 使用实时计算的数据 */}
      <Card size="small" styles={{ body: { padding: '12px 16px' } }}>
        <Row gutter={24}>
          <Col span={4}>
            <Statistic title="总数" value={realStats.total} valueStyle={{ fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="严重" value={realStats.critical} valueStyle={{ color: '#E53935', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="主要" value={realStats.major} valueStyle={{ color: '#FB8C00', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="次要" value={realStats.minor} valueStyle={{ color: '#FDD835', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="警告" value={realStats.warning} valueStyle={{ color: '#42A5F5', fontSize: 20 }} />
          </Col>
          <Col span={4}>
            <Statistic title="未确认" value={realStats.unacked} valueStyle={{ color: '#E53935', fontSize: 20 }} />
          </Col>
        </Row>
      </Card>

      {/* 趋势图卡片 */}
      <Card size="small" title="告警趋势（近7天）" styles={{ body: { padding: '12px 16px' } }}>
        <LineChart
          xData={trendData.dates}
          series={trendData.series}
          height={160}
          smooth
        />
      </Card>

      {/* 搜索和列表卡片 */}
      <Card size="small" styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}>
        <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', background: '#fafafa' }}>
          {/* 快捷筛选按钮 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, flexWrap: 'wrap' }}>
            {/* 快捷类型筛选 */}
            <Space.Compact size="small">
              {QUICK_FILTER_OPTIONS.map((opt) => (
                <Button
                  key={opt.key}
                  type={activeQuickFilter === opt.key ? 'primary' : 'default'}
                  onClick={() => handleQuickFilter(opt.key)}
                >
                  {opt.label}
                </Button>
              ))}
            </Space.Compact>

            <div style={{ flex: 1 }} />

            {/* 快捷时间选择 */}
            <Select
              size="small"
              placeholder="快捷时间"
              value={activeQuickTime}
              onChange={handleQuickTime}
              allowClear
              style={{ width: 120 }}
              suffixIcon={<CalendarOutlined />}
            >
              {QUICK_TIME_OPTIONS.map((opt) => (
                <Select.Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Select.Option>
              ))}
            </Select>

            {/* 筛选模板 */}
            {filterTemplates.length > 0 && (
              <Dropdown
                menu={{
                  items: [
                    { type: 'group', label: '应用筛选模板', key: 'template-group' },
                    ...filterTemplates.map((t) => ({
                      key: t.id,
                      label: (
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <span>{t.name}</span>
                          <Button
                            type="text"
                            size="small"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={(e) => { e.stopPropagation(); handleDeleteTemplate(t.id); }}
                          />
                        </div>
                      ),
                    })),
                  ],
                  onClick: ({ key }) => {
                    const template = filterTemplates.find((t) => t.id === key);
                    if (template) handleApplyTemplate(template);
                  },
                }}
              >
                <Button size="small" icon={<FilterOutlined />}>
                  模板
                </Button>
              </Dropdown>
            )}

            {/* 保存筛选模板 */}
            <Tooltip title="保存当前筛选条件为模板">
              <Button size="small" icon={<SaveOutlined />} onClick={() => setSaveTemplateModalOpen(true)}>
                保存
              </Button>
            </Tooltip>
          </div>

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

      {/* 添加分组弹窗 */}
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

      {/* 编辑分组弹窗 */}
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

      {/* 保存筛选模板弹窗 */}
      <Modal
        open={saveTemplateModalOpen}
        title="保存筛选模板"
        onCancel={() => setSaveTemplateModalOpen(false)}
        onOk={handleSaveTemplate}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form form={templateForm} layout="vertical" size="small">
          <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
            <Input placeholder="如：严重未确认告警" maxLength={30} autoFocus />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
