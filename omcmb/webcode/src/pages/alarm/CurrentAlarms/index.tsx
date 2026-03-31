import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge, Button, Card, Col, Dropdown, Form, Input, Menu, Modal, Radio, Row, Select, Space, Statistic, Tag, Tooltip, Typography, App } from 'antd';
import {
  CheckOutlined,
  ClearOutlined,
  DownloadOutlined,
  ExportOutlined,
  EyeOutlined,
  FilterOutlined,
  MinusCircleOutlined,
  ReloadOutlined,
  SaveOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
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

// 自动刷新间隔选项
const AUTO_REFRESH_INTERVALS = [
  { label: '15秒', value: 15 },
  { label: '30秒', value: 30 },
  { label: '1分钟', value: 60 },
  { label: '5分钟', value: 300 },
];

// 筛选模板存储 key
const FILTER_TEMPLATES_KEY = 'active-alarm-filter-templates';
const LAST_FILTER_KEY = 'active-alarm-last-filter';

// 筛选模板
interface FilterTemplate {
  id: string;
  name: string;
  params: AlarmFilter;
  createdAt: string;
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
  const [activeQuickTime, setActiveQuickTime] = useState<string | undefined>(undefined);

  // 筛选模板状态
  const [filterTemplates, setFilterTemplates] = useState<FilterTemplate[]>([]);
  const [saveTemplateModalOpen, setSaveTemplateModalOpen] = useState(false);
  const [templateForm] = Form.useForm<{ name: string }>();

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

  const rawAlarms: Alarm[] = data?.items ?? [];
  const total = data?.total ?? 0;

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
    setActiveQuickTime(undefined);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    setActiveQuickFilter('all');
    setActiveQuickTime(undefined);
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
  const handleSaveTemplate = useCallback(() => {
    templateForm.validateFields().then((values) => {
      const newTemplate: FilterTemplate = {
        id: `template-${Date.now()}`,
        name: values.name,
        params: filterParams,
        createdAt: new Date().toISOString(),
      };
      const updatedTemplates = [...filterTemplates, newTemplate];
      setFilterTemplates(updatedTemplates);
      localStorage.setItem(FILTER_TEMPLATES_KEY, JSON.stringify(updatedTemplates));
      setSaveTemplateModalOpen(false);
      templateForm.resetFields();
      message.success('筛选模板保存成功');
    });
  }, [filterParams, filterTemplates, templateForm, message]);

  // 应用筛选模板
  const handleApplyTemplate = useCallback((template: FilterTemplate) => {
    setFilterParams(template.params);
    setCurrentPage(1);
    message.success(`已应用模板：${template.name}`);
  }, [message]);

  // 删除筛选模板
  const handleDeleteTemplate = useCallback((id: string) => {
    const updatedTemplates = filterTemplates.filter((t) => t.id !== id);
    setFilterTemplates(updatedTemplates);
    localStorage.setItem(FILTER_TEMPLATES_KEY, JSON.stringify(updatedTemplates));
    message.success('模板已删除');
  }, [filterTemplates, message]);

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


  const handleMarkRead = useCallback(
    (ids: string[]) => {
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
        onClick: (keys) => handleMarkRead(keys as string[]),
      },
    ],
    [handleAcknowledge, handleUnacknowledge, handleClear, handleMarkRead]
  );

  return (
    <ListPageLayout
      title={t('nav.alarm.current')}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
            刷新
          </Button>
          <Space.Compact>
            <Button
              type={autoRefresh ? 'primary' : 'default'}
              icon={<SyncOutlined spin={autoRefresh} />}
              onClick={() => setAutoRefresh(!autoRefresh)}
            >
              自动刷新
            </Button>
            {autoRefresh && (
              <Select
                value={refreshInterval}
                onChange={setRefreshInterval}
                style={{ width: 90 }}
                options={AUTO_REFRESH_INTERVALS.map((opt) => ({
                  label: opt.label,
                  value: opt.value,
                }))}
              />
            )}
          </Space.Compact>
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
            <Button type="primary" icon={<ExportOutlined />}>
              导出
            </Button>
          </Dropdown>
        </Space>
      }
    >
      {/* 统计卡片 */}
      <Card size="small" styles={{ body: { padding: '12px 16px', marginBottom: 12 } }}>
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

      {/* 快捷操作栏 */}
      <Card size="small" styles={{ body: { padding: '12px 16px', marginBottom: 12 } }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
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
                  ...filterTemplates.map((tpl) => ({
                    key: tpl.id,
                    label: (
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span>{tpl.name}</span>
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<ClearOutlined />}
                          onClick={(e) => { e.stopPropagation(); handleDeleteTemplate(tpl.id); }}
                        />
                      </div>
                    ),
                  })),
                ],
                onClick: ({ key }) => {
                  const template = filterTemplates.find((tpl) => tpl.id === key);
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
      </Card>

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
    </ListPageLayout>
  );
}
