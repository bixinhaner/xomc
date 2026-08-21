import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge, Button, Card, Checkbox, Divider, Drawer, Form, Input, Modal, Pagination, Radio, Space, Table, Tag, Typography, App, Tree } from 'antd';
import {
  CheckOutlined,
  ClearOutlined,
  DeleteOutlined,
  EditOutlined,
  ExportOutlined,
  EyeOutlined,
  AlertOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useCurrentAlarms, useHistoricalAlarms, useAcknowledgeAlarms, useClearAlarms } from '@core/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm, DealState, EventType } from '@core/types/alarm';
import type { AlarmFilter } from '@core/types/alarm';
import AlarmDetail from '../AlarmDetail';
import ConfirmWithNoteModal from '../components/ConfirmWithNoteModal';
import styles from './CustomAlarmStats.module.css';
import { formatSystemTime } from '@core/utils/systemTime';

const { Text } = Typography;

// 告警级别颜色
const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string }> = {
  critical: { color: '#E53935', bgColor: '#FFEBEE' },
  major: { color: '#FB8C00', bgColor: '#FFF3E0' },
  minor: { color: '#FDD835', bgColor: '#FFFDE7' },
  warning: { color: '#42A5F5', bgColor: '#E3F2FD' },
};

// 告警级别标签 - use i18n keys
const SEVERITY_LABEL_KEYS: Record<string, string> = {
  critical: 'alarm.severity.critical',
  major: 'alarm.severity.major',
  minor: 'alarm.severity.minor',
  warning: 'alarm.severity.warning',
};

// 告警状态配置
const DEAL_STATE_CONFIG: Record<DealState, { label: string; color: string }> = {
  '0': { label: 'alarm.dealState.unconfirmedUncleared', color: '#E53935' },
  '1': { label: 'alarm.dealState.confirmedUncleared', color: '#FB8C00' },
  '2': { label: 'alarm.dealState.unconfirmedCleared', color: '#42A5F5' },
  '3': { label: 'alarm.dealState.confirmedCleared', color: '#67D972' },
};

// 事件类型配置（T-0136: keys 是 X.733 数字代码，非 EventType union；用 Record<string,string>）
const EVENT_TYPE_CONFIG: Record<string, string> = {
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

// 筛选模板存储 key
const FILTER_TEMPLATES_KEY = 'custom-alarm-filter-templates';
const LAST_FILTER_KEY = 'custom-alarm-last-filter';

// 自定义告警分组项
interface CustomAlarmGroup {
  id: string;
  name: string;
  description?: string;
  alarmType: 'active' | 'historical';
  alarmIds?: string[];
  deviceIds?: string[];
  enableNotification?: boolean;
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

export default function CustomAlarmStats() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<AlarmFilter>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 左侧树状态
  const [groups, setGroups] = useState<CustomAlarmGroup[]>(DEFAULT_GROUPS);
  const [selectedGroupId, setSelectedGroupId] = useState<string>('group-beijing');
  const [searchText, setSearchText] = useState('');

  // 筛选模板状态
  const [filterTemplates, setFilterTemplates] = useState<FilterTemplate[]>([]);
  const [saveTemplateModalOpen, setSaveTemplateModalOpen] = useState(false);
  const [templateForm] = Form.useForm<{ name: string }>();

  // 添加分组抽屉状态
  const [addGroupDrawerOpen, setAddGroupDrawerOpen] = useState(false);
  const [isEditingGroup, setIsEditingGroup] = useState(false);
  const [editGroupId, setEditGroupId] = useState<string | null>(null);
  const [addGroupForm] = Form.useForm<{
    name: string;
    description: string;
    alarmType?: 'active' | 'historical';
    alarmSources: string[];
    deviceIds: string[];
    enableNotification: boolean;
  }>();
  // 告警源设备列表（Mock数据）
  const [availableDevices, setAvailableDevices] = useState<{ id: string; name: string; neType: string }[]>([]);
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  // 全选网元类型状态
  const [selectAllENB, setSelectAllENB] = useState(false);
  const [selectAllGNB, setSelectAllGNB] = useState(false);
  const [selectAllGSM, setSelectAllGSM] = useState(false);
  // 添加设备弹窗状态
  const [addDeviceModalVisible, setAddDeviceModalVisible] = useState(false);
  const [addDeviceMode, setAddDeviceMode] = useState<'device' | 'deviceGroup'>('device');
  const [selectedNewDevices, setSelectedNewDevices] = useState<string[]>([]);
  const [selectedNewDeviceGroups, setSelectedNewDeviceGroups] = useState<string[]>([]);
  const [addDeviceKeyword, setAddDeviceKeyword] = useState('');
  const [addDeviceCurrentPage, setAddDeviceCurrentPage] = useState(1);
  const [addDevicePageSize, setAddDevicePageSize] = useState(10);
  // 设备组数据
  const [availableDeviceGroups, setAvailableDeviceGroups] = useState<{ id: string; name: string; deviceIds: string[] }[]>([]);

  // 已选告警状态
  const [availableAlarms, setAvailableAlarms] = useState<{ id: string; alarmIdentifier: string; alarmSource: string; possibleCause: string }[]>([]);
  const [selectedAlarms, setSelectedAlarms] = useState<string[]>([]);
  const [addAlarmModalVisible, setAddAlarmModalVisible] = useState(false);
  const [selectedNewAlarms, setSelectedNewAlarms] = useState<string[]>([]);
  const [addAlarmKeyword, setAddAlarmKeyword] = useState('');
  const [addAlarmCurrentPage, setAddAlarmCurrentPage] = useState(1);
  const [addAlarmPageSize, setAddAlarmPageSize] = useState(10);

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


  // 可添加的设备列表（不在已选列表中的设备，且根据关键字过滤）
  const filteredAvailableDevices = useMemo(() => {
    const existingIds = new Set(selectedDevices);
    let available = availableDevices.filter((d) => !existingIds.has(d.id));
    if (addDeviceKeyword.trim()) {
      const keyword = addDeviceKeyword.toLowerCase();
      available = available.filter(
        (d) => d.name.toLowerCase().includes(keyword) || d.neType.toLowerCase().includes(keyword)
      );
    }
    return available;
  }, [availableDevices, selectedDevices, addDeviceKeyword]);

  // 分页后的设备列表
  const paginatedDevices = useMemo(() => {
    const start = (addDeviceCurrentPage - 1) * addDevicePageSize;
    return filteredAvailableDevices.slice(start, start + addDevicePageSize);
  }, [filteredAvailableDevices, addDeviceCurrentPage, addDevicePageSize]);

  // 可添加的设备组列表（根据关键字过滤）
  const filteredAvailableDeviceGroups = useMemo(() => {
    let available = availableDeviceGroups;
    if (addDeviceKeyword.trim()) {
      const keyword = addDeviceKeyword.toLowerCase();
      available = available.filter((g) => g.name.toLowerCase().includes(keyword));
    }
    return available;
  }, [availableDeviceGroups, addDeviceKeyword]);

  // 分页后的设备组列表
  const paginatedDeviceGroups = useMemo(() => {
    const start = (addDeviceCurrentPage - 1) * addDevicePageSize;
    return filteredAvailableDeviceGroups.slice(start, start + addDevicePageSize);
  }, [filteredAvailableDeviceGroups, addDeviceCurrentPage, addDevicePageSize]);

  // 可添加的告警列表（不在已选列表中的告警，且根据关键字过滤）
  const filteredAvailableAlarms = useMemo(() => {
    const existingIds = new Set(selectedAlarms);
    let available = availableAlarms.filter((a) => !existingIds.has(a.id));
    if (addAlarmKeyword.trim()) {
      const keyword = addAlarmKeyword.toLowerCase();
      available = available.filter(
        (a) =>
          a.alarmIdentifier.toLowerCase().includes(keyword) ||
          a.alarmSource.toLowerCase().includes(keyword) ||
          a.possibleCause.toLowerCase().includes(keyword)
      );
    }
    return available;
  }, [availableAlarms, selectedAlarms, addAlarmKeyword]);

  // 分页后的告警列表
  const paginatedAlarms = useMemo(() => {
    const start = (addAlarmCurrentPage - 1) * addAlarmPageSize;
    return filteredAvailableAlarms.slice(start, start + addAlarmPageSize);
  }, [filteredAvailableAlarms, addAlarmCurrentPage, addAlarmPageSize]);

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
    { name: 'timeRange', label: t('alarm.eventTime'), type: 'date-range', showTime: true },
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

  // 同时获取活动告警和历史告警
  const currentAlarmsQuery = useCurrentAlarms(queryParams);
  const historicalAlarmsQuery = useHistoricalAlarms(queryParams);

  const isLoading = currentAlarmsQuery.isLoading || historicalAlarmsQuery.isLoading;
  const data = currentAlarmsQuery.data;
  const refetch = currentAlarmsQuery.refetch;

  // 活动告警数据
  const activeTotal = currentAlarmsQuery.data?.total ?? 0;

  // 历史告警数据
  const historicalTotal = historicalAlarmsQuery.data?.total ?? 0;

  const acknowledgeAlarms = useAcknowledgeAlarms();
  const clearAlarms = useClearAlarms();

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
      message.success(t('alarm.templateSaveSuccess'));
    } catch {
      // validation error
    }
  }, [filterParams, filterTemplates, templateForm, message]);


  // 添加/编辑分组
  const handleAddGroup = useCallback(async () => {
    try {
      const values = await addGroupForm.validateFields();

      if (isEditingGroup && editGroupId) {
        // 编辑模式
        setGroups((prev) =>
          prev.map((g) => (g.id === editGroupId ? {
            ...g,
            name: values.name,
            description: values.description,
            alarmIds: selectedAlarms,
            deviceIds: selectedDevices,
            enableNotification: values.enableNotification ?? false,
          } : g))
        );
      } else {
        // 新增模式
        const newGroup: CustomAlarmGroup = {
          id: `group-${Date.now()}`,
          name: values.name,
          description: values.description,
          alarmType: values.alarmType || 'active',
          createdAt: new Date().toISOString().split('T')[0],
          alarmIds: selectedAlarms,
          deviceIds: selectedDevices,
          enableNotification: values.enableNotification ?? false,
        };
        setGroups((prev) => [...prev, newGroup]);
      }

      setAddGroupDrawerOpen(false);
      setIsEditingGroup(false);
      setEditGroupId(null);
      addGroupForm.resetFields();
      setSelectedDevices([]);
      setSelectedAlarms([]);
      setSelectAllENB(false);
      setSelectAllGNB(false);
      setSelectAllGSM(false);
      message.success(t('common.success'));
    } catch {
      // validation error
    }
  }, [addGroupForm, selectedDevices, selectedAlarms, isEditingGroup, editGroupId, message, t]);

  // 打开添加分组抽屉
  const handleOpenAddGroupDrawer = useCallback(() => {
    setIsEditingGroup(false);
    setEditGroupId(null);
    addGroupForm.resetFields();
    setSelectedDevices([]);
    setSelectedAlarms([]);
    setSelectAllENB(false);
    setSelectAllGNB(false);
    setSelectAllGSM(false);
    // 模拟加载可用设备列表
    const mockDevices = [
      { id: 'dev-001', name: '北京基站-1', neType: 'eNB' },
      { id: 'dev-002', name: '北京基站-2', neType: 'eNB' },
      { id: 'dev-003', name: '上海基站-1', neType: 'gNB' },
      { id: 'dev-004', name: '上海基站-2', neType: 'gNB' },
      { id: 'dev-005', name: '广州基站-1', neType: 'GSM' },
      { id: 'dev-006', name: '广州基站-2', neType: 'GSM' },
      { id: 'dev-007', name: '深圳基站-1', neType: 'eNB' },
      { id: 'dev-008', name: '深圳基站-2', neType: 'gNB' },
      { id: 'dev-009', name: '成都基站-1', neType: 'eNB' },
      { id: 'dev-010', name: '成都基站-2', neType: 'gNB' },
      { id: 'dev-011', name: '杭州基站-1', neType: 'eNB' },
      { id: 'dev-012', name: '杭州基站-2', neType: 'gNB' },
      { id: 'dev-013', name: '南京基站-1', neType: 'GSM' },
      { id: 'dev-014', name: '南京基站-2', neType: 'eNB' },
      { id: 'dev-015', name: '武汉基站-1', neType: 'gNB' },
    ];
    setAvailableDevices(mockDevices);
    // 模拟加载可用告警列表
    const mockAlarms = [
      { id: 'alarm-001', alarmIdentifier: 'ALM-001', alarmSource: 'eNB', possibleCause: '射频单元功率异常' },
      { id: 'alarm-002', alarmIdentifier: 'ALM-002', alarmSource: 'eNB', possibleCause: '光模块信号丢失' },
      { id: 'alarm-003', alarmIdentifier: 'ALM-003', alarmSource: 'gNB', possibleCause: '时钟同步失败' },
      { id: 'alarm-004', alarmIdentifier: 'ALM-004', alarmSource: 'gNB', possibleCause: 'CPU利用率过高' },
      { id: 'alarm-005', alarmIdentifier: 'ALM-005', alarmSource: 'GSM', possibleCause: '传输链路故障' },
      { id: 'alarm-006', alarmIdentifier: 'ALM-006', alarmSource: 'GSM', possibleCause: '基站温度过高' },
      { id: 'alarm-007', alarmIdentifier: 'ALM-007', alarmSource: 'eNB', possibleCause: 'S1接口连接中断' },
      { id: 'alarm-008', alarmIdentifier: 'ALM-008', alarmSource: 'gNB', possibleCause: '电源模块故障' },
      { id: 'alarm-009', alarmIdentifier: 'ALM-009', alarmSource: 'eNB', possibleCause: '风扇转速异常' },
      { id: 'alarm-010', alarmIdentifier: 'ALM-010', alarmSource: 'gNB', possibleCause: '存储空间不足' },
      { id: 'alarm-011', alarmIdentifier: 'ALM-011', alarmSource: 'GSM', possibleCause: '驻波比告警' },
      { id: 'alarm-012', alarmIdentifier: 'ALM-012', alarmSource: 'eNB', possibleCause: 'X2接口连接中断' },
      { id: 'alarm-013', alarmIdentifier: 'ALM-013', alarmSource: 'gNB', possibleCause: 'F1接口异常' },
      { id: 'alarm-014', alarmIdentifier: 'ALM-014', alarmSource: 'GSM', possibleCause: 'Abis接口故障' },
      { id: 'alarm-015', alarmIdentifier: 'ALM-015', alarmSource: 'eNB', possibleCause: 'GPS信号丢失' },
    ];
    setAvailableAlarms(mockAlarms);
    setAddGroupDrawerOpen(true);
  }, [addGroupForm]);


  // 打开添加设备弹窗
  const handleOpenAddDeviceModal = useCallback(() => {
    setAddDeviceMode('device');
    setSelectedNewDevices([]);
    setSelectedNewDeviceGroups([]);
    setAddDeviceKeyword('');
    setAddDeviceCurrentPage(1);
    // 加载设备组数据（包含实际的设备ID列表）
    const mockDeviceGroups = [
      { id: 'group-001', name: '北京地区基站', deviceIds: ['dev-001', 'dev-002', 'dev-007', 'dev-009', 'dev-011', 'dev-014'] },
      { id: 'group-002', name: '上海地区基站', deviceIds: ['dev-003', 'dev-004', 'dev-008', 'dev-010', 'dev-012'] },
      { id: 'group-003', name: '广州地区基站', deviceIds: ['dev-005', 'dev-006', 'dev-013'] },
      { id: 'group-004', name: '深圳地区基站', deviceIds: ['dev-007', 'dev-008', 'dev-015'] },
      { id: 'group-005', name: '成都地区基站', deviceIds: ['dev-009', 'dev-010'] },
      { id: 'group-006', name: '杭州地区基站', deviceIds: ['dev-011', 'dev-012'] },
      { id: 'group-007', name: '南京地区基站', deviceIds: ['dev-013', 'dev-014'] },
      { id: 'group-008', name: '武汉地区基站', deviceIds: ['dev-015'] },
    ];
    setAvailableDeviceGroups(mockDeviceGroups);
    setAddDeviceModalVisible(true);
  }, []);

  // 编辑分组 - 打开抽屉并预填数据
  const handleEditGroup = useCallback((groupId: string) => {
    const group = groups.find((g) => g.id === groupId);
    if (!group) return;

    setIsEditingGroup(true);
    setEditGroupId(groupId);

    // 加载可用设备列表
    const mockDevices = [
      { id: 'dev-001', name: '北京基站-1', neType: 'eNB' },
      { id: 'dev-002', name: '北京基站-2', neType: 'eNB' },
      { id: 'dev-003', name: '上海基站-1', neType: 'gNB' },
      { id: 'dev-004', name: '上海基站-2', neType: 'gNB' },
      { id: 'dev-005', name: '广州基站-1', neType: 'GSM' },
      { id: 'dev-006', name: '广州基站-2', neType: 'GSM' },
      { id: 'dev-007', name: '深圳基站-1', neType: 'eNB' },
      { id: 'dev-008', name: '深圳基站-2', neType: 'gNB' },
      { id: 'dev-009', name: '成都基站-1', neType: 'eNB' },
      { id: 'dev-010', name: '成都基站-2', neType: 'gNB' },
      { id: 'dev-011', name: '杭州基站-1', neType: 'eNB' },
      { id: 'dev-012', name: '杭州基站-2', neType: 'gNB' },
      { id: 'dev-013', name: '南京基站-1', neType: 'GSM' },
      { id: 'dev-014', name: '南京基站-2', neType: 'eNB' },
      { id: 'dev-015', name: '武汉基站-1', neType: 'gNB' },
    ];
    setAvailableDevices(mockDevices);

    // 加载可用告警列表
    const mockAlarms = [
      { id: 'alarm-001', alarmIdentifier: 'ALM-001', alarmSource: 'eNB', possibleCause: '射频单元功率异常' },
      { id: 'alarm-002', alarmIdentifier: 'ALM-002', alarmSource: 'eNB', possibleCause: '光模块信号丢失' },
      { id: 'alarm-003', alarmIdentifier: 'ALM-003', alarmSource: 'gNB', possibleCause: '时钟同步失败' },
      { id: 'alarm-004', alarmIdentifier: 'ALM-004', alarmSource: 'gNB', possibleCause: 'CPU利用率过高' },
      { id: 'alarm-005', alarmIdentifier: 'ALM-005', alarmSource: 'GSM', possibleCause: '传输链路故障' },
      { id: 'alarm-006', alarmIdentifier: 'ALM-006', alarmSource: 'GSM', possibleCause: '基站温度过高' },
      { id: 'alarm-007', alarmIdentifier: 'ALM-007', alarmSource: 'eNB', possibleCause: 'S1接口连接中断' },
      { id: 'alarm-008', alarmIdentifier: 'ALM-008', alarmSource: 'gNB', possibleCause: '电源模块故障' },
      { id: 'alarm-009', alarmIdentifier: 'ALM-009', alarmSource: 'eNB', possibleCause: '风扇转速异常' },
      { id: 'alarm-010', alarmIdentifier: 'ALM-010', alarmSource: 'gNB', possibleCause: '存储空间不足' },
      { id: 'alarm-011', alarmIdentifier: 'ALM-011', alarmSource: 'GSM', possibleCause: '驻波比告警' },
      { id: 'alarm-012', alarmIdentifier: 'ALM-012', alarmSource: 'eNB', possibleCause: 'X2接口连接中断' },
      { id: 'alarm-013', alarmIdentifier: 'ALM-013', alarmSource: 'gNB', possibleCause: 'F1接口异常' },
      { id: 'alarm-014', alarmIdentifier: 'ALM-014', alarmSource: 'GSM', possibleCause: 'Abis接口故障' },
      { id: 'alarm-015', alarmIdentifier: 'ALM-015', alarmSource: 'eNB', possibleCause: 'GPS信号丢失' },
    ];
    setAvailableAlarms(mockAlarms);

    // 设置表单值
    addGroupForm.setFieldsValue({
      name: group.name,
      description: group.description || '',
    });

    // 设置已选设备和告警
    setSelectedDevices(group.deviceIds || []);
    setSelectedAlarms(group.alarmIds || []);

    // 更新全选状态
    const enbDevices = mockDevices.filter(d => d.neType === 'eNB').map(d => d.id);
    const gnbDevices = mockDevices.filter(d => d.neType === 'gNB').map(d => d.id);
    const gsmDevices = mockDevices.filter(d => d.neType === 'GSM').map(d => d.id);
    const groupDeviceIds = group.deviceIds || [];
    setSelectAllENB(enbDevices.every(id => groupDeviceIds.includes(id)));
    setSelectAllGNB(gnbDevices.every(id => groupDeviceIds.includes(id)));
    setSelectAllGSM(gsmDevices.every(id => groupDeviceIds.includes(id)));

    setAddGroupDrawerOpen(true);
  }, [groups, addGroupForm]);

  // 删除分组
  const handleDeleteGroup = useCallback((groupId: string) => {
    const group = groups.find((g) => g.id === groupId);
    if (groups.length <= 1) {
      message.warning(t('alarm.keepAtLeastOneGroup'));
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      content: t('alarm.confirmDeleteGroup', { name: group?.name ?? '' }),
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

  const handleMarkRead = useCallback(() => {
    setSelectedRowKeys([]);
    void refetch();
    message.success(t('common.markReadSuccess'));
  }, [refetch, t, message]);

  // 导出处理 - 直接导出当前分组数据
  const handleExport = useCallback(() => {
    try {
      console.log('Export data for group:', selectedGroupId);
      void message.info(t('common.exportInProgress'));
    } catch {
      message.error(t('common.exportFailed'));
    }
  }, [selectedGroupId, message, t]);

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
      key: 'id',
      title: t('alarm.alarmId'),
      dataIndex: 'id',
      width: 100,
      render: (val: unknown, record) => (
        <Space size={4}>
          {record.unread === '1' && <Badge status="error" style={{ marginLeft: -4 }} />}
          <Button type="link" size="small" style={{ padding: 0, height: 'auto' }} onClick={() => handleShowDetail(record)}>
            {String(val ?? '')}
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
            {t(SEVERITY_LABEL_KEYS[record.severity]) ?? record.severity}
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
      render: (val: unknown) => { const s = String(val ?? ''); return NE_TYPE_CONFIG[s] || s || '-'; },
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
      render: (val: unknown) => t(EVENT_TYPE_CONFIG[val as EventType] || 'common.unknown'),
    },
    {
      key: 'alarmType',
      title: t('alarm.alarmTypeCol'),
      dataIndex: 'alarmType',
      width: 100,
      render: () => (
        <Tag color="red">
          {t('alarm.activeAlarm')}
        </Tag>
      ),
    },
    {
      key: 'dealState',
      title: t('alarm.dealState'),
      dataIndex: 'dealState',
      width: 150,
      ellipsis: true,
      render: (val: unknown) => {
        const config = DEAL_STATE_CONFIG[val as DealState];
        return <span style={{ color: config?.color || '#666' }}>{t(config?.label || 'common.unknown')}</span>;
      },
    },
    {
      key: 'eventTime',
      title: t('alarm.eventTime'),
      dataIndex: 'eventTime',
      width: 150,
      render: (v) => (v ? formatSystemTime(String(v)) : '-'),
    },
    {
      key: 'updTime',
      title: t('alarm.updTime'),
      dataIndex: 'updTime',
      width: 150,
      render: (v) => (v ? formatSystemTime(String(v)) : '-'),
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
  ], [t, handleShowDetail]);

  const batchActions = useMemo((): BatchAction[] => {
    const actions: BatchAction[] = [];
    actions.push(
      { key: 'batch-ack', label: t('alarm.acknowledge'), icon: <CheckOutlined />, onClick: (keys) => handleAcknowledge(keys as string[]) },
      { key: 'batch-unack', label: t('alarm.unacknowledge'), icon: <MinusCircleOutlined />, onClick: (keys) => handleUnacknowledge(keys as string[]) },
      { key: 'batch-clear', label: t('alarm.clear'), icon: <ClearOutlined />, danger: true, onClick: (keys) => handleClear(keys as string[]) },
      { key: 'batch-read', label: t('alarm.markRead'), icon: <EyeOutlined />, onClick: () => handleMarkRead() }
    );
    actions.push({ key: 'batch-delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, onClick: (keys) => handleDelete(keys as string[]) });
    return actions;
  }, [handleAcknowledge, handleUnacknowledge, handleClear, handleMarkRead, handleDelete, t]);

  // 左侧树面板
  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className={styles.treePanelHeader}>
        <Text strong className={styles.treePanelTitle}>
          {t('alarm.customAlarmGroup')}
        </Text>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={handleOpenAddGroupDrawer}
        >
          {t('alarm.addGroupBtn')}
        </Button>
      </div>
      <div className={styles.treeSearchContainer}>
        <Input
          placeholder={t('alarm.searchGroup')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
          size="small"
          className={styles.treeSearchInput}
        />
      </div>
      <div className={styles.treeNodesContainer}>
        <Tree
          className="alarm-tree"
          treeData={filteredGroups.map((group) => ({
            key: group.id,
            title: (
              <div className={styles.alarmGroupNode}>
                <span className={styles.treeNodeContent}>
                  <AlertOutlined className={styles.alarmGroupNodeIcon} />
                  <span className={styles.treeNodeText}>{group.name}</span>
                </span>
                <span className={styles.treeNodeActions}>
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={(e) => { e.stopPropagation(); handleEditGroup(group.id); }}
                    className={styles.treeNodeActionButton}
                    title={t('common.edit')}
                  />
                  <Button
                    type="text"
                    size="small"
                    icon={<DeleteOutlined />}
                    onClick={(e) => { e.stopPropagation(); handleDeleteGroup(group.id); }}
                    className={styles.treeNodeActionButton}
                    danger
                    title={t('common.delete')}
                  />
                </span>
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
            }
          }}
          blockNode
          showLine={{ showLeafIcon: false }}
          style={{ fontSize: 13 }}
        />
      </div>
      <style>{`
        /* 列表卡片滚动条样式 - 页面特定，只滚动表格内容 */
        .custom-alarm-list-card .ant-card-body > div {
          position: absolute;
          inset: 0;
        }
        .custom-alarm-list-card .omc-data-table {
          overflow: hidden;
        }
        .custom-alarm-list-card .ant-table-body {
          overflow-y: auto !important;
          max-height: calc(100vh - 520px) !important;
        }
      `}</style>
    </div>
  );

  // 右侧面板
  const rightPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 12 }}>
      {/* 标题卡片 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Space>
          <Text strong style={{ fontSize: 16 }}>{selectedGroup?.name || t('alarm.customAlarmGroup')}</Text>
        </Space>
        <Space>
          <Button icon={<ExportOutlined />} onClick={handleExport}>
            {t('alarm.export')}
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Card size="small" styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}>
        <div className={styles.cardHeader}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
            <Space size={16}>
              <Text className={styles.cardHeaderTitle}>{t('alarm.statistics.title')}</Text>
            </Space>
            <Space size={8}>
              <div className={`${styles.statsBadge} ${styles.statsActive}`}>
                <span>{t('alarm.stat.activeAlarm')}: {activeTotal}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsHistorical}`}>
                <span>{t('alarm.stat.historicalAlarm')}: {historicalTotal}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsCritical}`}>
                <span>{t('alarm.stat.critical')}: {realStats.critical}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsMajor}`}>
                <span>{t('alarm.stat.major')}: {realStats.major}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsMinor}`}>
                <span>{t('alarm.stat.minor')}: {realStats.minor}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsWarning}`}>
                <span>{t('alarm.stat.warning')}: {realStats.warning}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsUnacked}`}>
                <span>{t('alarm.stat.unacked')}: {realStats.unacked}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsUnread}`}>
                <span>{t('alarm.stat.unread')}: {rawAlarms.filter(a => a.unread === '1').length}</span>
              </div>
            </Space>
          </div>
        </div>
      </Card>

      <div className="custom-alarm-filter-wrapper">
        <FilterBar
          filterId={`custom-alarm-stats-${selectedGroupId}`}
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />
      </div>

      {/* 列表卡片 */}
      <Card
        size="small"
        variant="outlined"
        className="custom-alarm-list-card"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden', position: 'relative' } }}
      >
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
          defaultDensity="default"
          scroll={{ x: 'max-content', y: 100 }}
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
        />
      </Card>
      <style>{`
        .custom-alarm-filter-wrapper [class*="_filterBarWrapper_"] {
          margin-bottom: 0 !important;
        }
        .custom-alarm-list-card .omc-data-table,
        .custom-alarm-list-card .ant-table-wrapper,
        .custom-alarm-list-card .ant-spin-nested-loading,
        .custom-alarm-list-card .ant-spin-nested-loading > div,
        .custom-alarm-list-card .ant-table,
        .custom-alarm-list-card .ant-table-container {
          display: flex !important;
          flex-direction: column !important;
          flex: 1 !important;
          min-height: 0 !important;
        }
        .custom-alarm-list-card .ant-table-body {
          flex: 1 !important;
          min-height: 0 !important;
          overflow: auto !important;
          max-height: none !important;
        }
      `}</style>
    </div>
  );

  return (
    <>
      <TreeListPageLayout tree={treePanel} defaultTreeWidth={220}>
        {rightPanel}
      </TreeListPageLayout>

      <AlarmDetail alarm={detailAlarm} open={detailOpen} onClose={handleCloseDetail} />

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

      {/* 添加/编辑分组抽屉 */}
      <Drawer
        open={addGroupDrawerOpen}
        title={isEditingGroup ? t('alarm.editCustomGroup') : t('alarm.addCustomGroup')}
        placement="right"
        size={520}
        onClose={() => setAddGroupDrawerOpen(false)}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setAddGroupDrawerOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleAddGroup}>{t('common.confirm')}</Button>
          </div>
        }
      >
        <Form form={addGroupForm} layout="vertical" size="small">
          {/* 模版名称 */}
          <Form.Item
            name="name"
            label={t('alarm.templateName')}
            rules={[
              { required: true, message: t('alarm.inputTemplateName') },
              { min: 1, max: 50, message: t('alarm.templateNameLength') },
              {
                validator: (_, value) => {
                  const otherGroups = isEditingGroup
                    ? groups.filter(g => g.id !== editGroupId)
                    : groups;
                  if (value && otherGroups.some(g => g.name === value)) {
                    return Promise.reject(new Error(t('alarm.templateNameExists')));
                  }
                  return Promise.resolve();
                },
              },
            ]}
          >
            <Input placeholder={t('alarm.inputTemplateName')} maxLength={50} showCount />
          </Form.Item>

          {/* 描述 */}
          <Form.Item name="description" label={t('alarm.description')}>
            <Input.TextArea placeholder={t('alarm.inputDescription')} rows={3} maxLength={200} showCount />
          </Form.Item>

          <Divider />

          {/* 告警源 - 全选复选框 */}
          <Form.Item label={t('alarm.alarmSource')}>
            <Space orientation="vertical" style={{ width: '100%' }}>
              <Checkbox
                checked={selectAllENB}
                onChange={(e) => {
                  setSelectAllENB(e.target.checked);
                  if (e.target.checked) {
                    // 选中eNB全部设备
                    const enbDevices = availableDevices.filter(d => d.neType === 'eNB').map(d => d.id);
                    setSelectedDevices(prev => [...new Set([...prev, ...enbDevices])]);
                  } else {
                    // 取消选中eNB全部设备
                    setSelectedDevices(prev => prev.filter(id => {
                      const device = availableDevices.find(d => d.id === id);
                      return device?.neType !== 'eNB';
                    }));
                  }
                }}
              >
                {t('alarm.allENBDevices')}
                <Tag color="blue" style={{ marginLeft: 8 }}>
                  {availableDevices.filter(d => d.neType === 'eNB').length} {t('alarm.deviceUnit')}
                </Tag>
              </Checkbox>
              <Checkbox
                checked={selectAllGNB}
                onChange={(e) => {
                  setSelectAllGNB(e.target.checked);
                  if (e.target.checked) {
                    const gnbDevices = availableDevices.filter(d => d.neType === 'gNB').map(d => d.id);
                    setSelectedDevices(prev => [...new Set([...prev, ...gnbDevices])]);
                  } else {
                    setSelectedDevices(prev => prev.filter(id => {
                      const device = availableDevices.find(d => d.id === id);
                      return device?.neType !== 'gNB';
                    }));
                  }
                }}
              >
                {t('alarm.allGNBDevices')}
                <Tag color="blue" style={{ marginLeft: 8 }}>
                  {availableDevices.filter(d => d.neType === 'gNB').length} {t('alarm.deviceUnit')}
                </Tag>
              </Checkbox>
              <Checkbox
                checked={selectAllGSM}
                onChange={(e) => {
                  setSelectAllGSM(e.target.checked);
                  if (e.target.checked) {
                    const gsmDevices = availableDevices.filter(d => d.neType === 'GSM').map(d => d.id);
                    setSelectedDevices(prev => [...new Set([...prev, ...gsmDevices])]);
                  } else {
                    setSelectedDevices(prev => prev.filter(id => {
                      const device = availableDevices.find(d => d.id === id);
                      return device?.neType !== 'GSM';
                    }));
                  }
                }}
              >
                {t('alarm.allGSMDevices')}
                <Tag color="blue" style={{ marginLeft: 8 }}>
                  {availableDevices.filter(d => d.neType === 'GSM').length} {t('alarm.deviceUnit')}
                </Tag>
              </Checkbox>
            </Space>
          </Form.Item>

          <Divider />

          {/* 已选设备 */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
              <span>
                {t('alarm.selectedDevices')}
                <Tag color="blue" style={{ marginLeft: 8 }}>{selectedDevices.length} {t('alarm.deviceUnit')}</Tag>
              </span>
              <Space size={8}>
                <Button
                  size="small"
                  icon={<ClearOutlined />}
                  disabled={selectedDevices.length === 0}
                  onClick={() => {
                    setSelectedDevices([]);
                    setSelectAllENB(false);
                    setSelectAllGNB(false);
                    setSelectAllGSM(false);
                  }}
                >
                                    {t('alarm.clearAll')}
                </Button>
                <Button
                  size="small"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={handleOpenAddDeviceModal}
                >
                  {t('alarm.addDevice')}
                </Button>
              </Space>
            </div>
          }>
            {selectedDevices.length === 0 ? (
              <div style={{ padding: 24, textAlign: 'center', color: '#999', border: '1px dashed #d9d9d9', borderRadius: 6 }}>
                {t('alarm.selectAlarmSource')}
              </div>
            ) : (
              <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, maxHeight: 300, overflow: 'auto' }}>
                <Table
                  size="small"
                  dataSource={availableDevices.filter(d => selectedDevices.includes(d.id))}
                  rowKey="id"
                  pagination={false}
                  columns={[
                    { title: t('alarm.deviceNameCol'), dataIndex: 'name', ellipsis: true },
                    { title: t('alarm.neTypeCol'), dataIndex: 'neType', width: 80, render: (val) => <Tag>{val}</Tag> },
                    {
                      title: '',
                      width: 40,
                      render: (_: unknown, record: { id: string; name: string; neType: string }) => (
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => {
                            setSelectedDevices(prev => prev.filter(id => id !== record.id));
                            // 同时更新全选状态
                            const enbCount = availableDevices.filter(d => d.neType === 'eNB').length;
                            const gnbCount = availableDevices.filter(d => d.neType === 'gNB').length;
                            const gsmCount = availableDevices.filter(d => d.neType === 'GSM').length;
                            const selectedEnbCount = selectedDevices.filter(id => {
                              const d = availableDevices.find(dev => dev.id === id);
                              return d?.neType === 'eNB';
                            }).length - (record.neType === 'eNB' ? 1 : 0);
                            const selectedGnbCount = selectedDevices.filter(id => {
                              const d = availableDevices.find(dev => dev.id === id);
                              return d?.neType === 'gNB';
                            }).length - (record.neType === 'gNB' ? 1 : 0);
                            const selectedGsmCount = selectedDevices.filter(id => {
                              const d = availableDevices.find(dev => dev.id === id);
                              return d?.neType === 'GSM';
                            }).length - (record.neType === 'GSM' ? 1 : 0);
                            setSelectAllENB(selectedEnbCount === enbCount && enbCount > 0);
                            setSelectAllGNB(selectedGnbCount === gnbCount && gnbCount > 0);
                            setSelectAllGSM(selectedGsmCount === gsmCount && gsmCount > 0);
                          }}
                        />
                      ),
                    },
                  ]}
                />
              </div>
            )}
          </Form.Item>

          <Divider />

          {/* 已选告警 */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
              <span>
                {t('alarm.selectedAlarms')}
                <Tag color="orange" style={{ marginLeft: 8 }}>{selectedAlarms.length} {t('alarm.alarmCountUnit')}</Tag>
              </span>
              <Space size={8}>
                <Button
                  size="small"
                  icon={<ClearOutlined />}
                  disabled={selectedAlarms.length === 0}
                  onClick={() => setSelectedAlarms([])}
                >
                  清空
                </Button>
                <Button
                  size="small"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setAddAlarmModalVisible(true)}
                >
                  {t('alarm.addAlarmBtn')}
                </Button>
              </Space>
            </div>
          }>
            {selectedAlarms.length === 0 ? (
              <div style={{ padding: 24, textAlign: 'center', color: '#999', border: '1px dashed #d9d9d9', borderRadius: 6 }}>
                {t('alarm.noSelectedAlarm')}
              </div>
            ) : (
              <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, maxHeight: 200, overflow: 'auto' }}>
                <Table
                  size="small"
                  dataSource={availableAlarms.filter(a => selectedAlarms.includes(a.id))}
                  rowKey="id"
                  pagination={false}
                  columns={[
                    { title: t('alarm.alarmIdentifierCol'), dataIndex: 'alarmIdentifier', ellipsis: true },
                    { title: t('alarm.alarmSourceCol'), dataIndex: 'alarmSource', width: 80, render: (val) => <Tag>{val}</Tag> },
                    { title: t('alarm.possibleCauseCol'), dataIndex: 'possibleCause', ellipsis: true },
                    {
                      title: '',
                      width: 40,
                      render: (_: unknown, record: { id: string }) => (
                        <Button
                          type="text"
          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => {
            setSelectedAlarms(prev => prev.filter(id => id !== record.id));
          }}
                        />
                      ),
                    }
                  ]}
                />
              </div>
            )}
          </Form.Item>
        </Form>
      </Drawer>

      {/* 添加设备弹窗 */}
      <Modal
        open={addDeviceModalVisible}
        title={t('alarm.addDeviceTitle')}
        onCancel={() => {
          setAddDeviceModalVisible(false);
          setSelectedNewDevices([]);
          setSelectedNewDeviceGroups([]);
          setAddDeviceKeyword('');
          setAddDeviceCurrentPage(1);
          setAddDeviceMode('device');
        }}
        onOk={() => {
          if (addDeviceMode === 'device') {
            setSelectedDevices(prev => [...new Set([...prev, ...selectedNewDevices])]);
            // 更新全选状态
            const allIds = [...new Set([...selectedDevices, ...selectedNewDevices])];
            const enbCount = availableDevices.filter(d => d.neType === 'eNB').length;
            const gnbCount = availableDevices.filter(d => d.neType === 'gNB').length;
            const gsmCount = availableDevices.filter(d => d.neType === 'GSM').length;
            const selectedEnbCount = allIds.filter(id => {
              const d = availableDevices.find(dev => dev.id === id);
              return d?.neType === 'eNB';
            }).length;
            const selectedGnbCount = allIds.filter(id => {
              const d = availableDevices.find(dev => dev.id === id);
              return d?.neType === 'gNB';
            }).length;
            const selectedGsmCount = allIds.filter(id => {
              const d = availableDevices.find(dev => dev.id === id);
              return d?.neType === 'GSM';
            }).length;
            setSelectAllENB(selectedEnbCount === enbCount && enbCount > 0);
            setSelectAllGNB(selectedGnbCount === gnbCount && gnbCount > 0);
            setSelectAllGSM(selectedGsmCount === gsmCount && gsmCount > 0);
          } else {
            // 按设备组添加：将设备组下的所有设备添加到已选设备
            const deviceIdsFromGroups = availableDeviceGroups
              .filter(g => selectedNewDeviceGroups.includes(g.id))
              .flatMap(g => g.deviceIds);
            setSelectedDevices(prev => [...new Set([...prev, ...deviceIdsFromGroups])]);
          }
          setAddDeviceModalVisible(false);
          setSelectedNewDevices([]);
          setSelectedNewDeviceGroups([]);
          setAddDeviceKeyword('');
          setAddDeviceCurrentPage(1);
          setAddDeviceMode('device');
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={700}
      >
        {/* 模式切换 */}
        <div style={{ marginBottom: 16 }}>
          <Radio.Group
            value={addDeviceMode}
            onChange={(e) => {
              setAddDeviceMode(e.target.value);
              setSelectedNewDevices([]);
              setSelectedNewDeviceGroups([]);
              setAddDeviceKeyword('');
              setAddDeviceCurrentPage(1);
            }}
            optionType="button"
            buttonStyle="solid"
          >
            <Radio.Button value="device">{t('alarm.addByDevice')}</Radio.Button>
            <Radio.Button value="deviceGroup">{t('alarm.addByDeviceGroup')}</Radio.Button>
          </Radio.Group>
        </div>

        {/* 搜索框 */}
        <div style={{ marginBottom: 12 }}>
          <Input
            placeholder={addDeviceMode === 'device' ? t('alarm.searchDeviceName') : t('alarm.searchDeviceGroupName')}
            prefix={<SearchOutlined />}
            value={addDeviceKeyword}
            onChange={(e) => {
              setAddDeviceKeyword(e.target.value);
              setAddDeviceCurrentPage(1);
            }}
            allowClear
          />
        </div>

        {addDeviceMode === 'device' ? (
          <>
            {/* 设备列表 */}
            <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Checkbox
                checked={selectedNewDevices.length === filteredAvailableDevices.length && filteredAvailableDevices.length > 0}
                indeterminate={selectedNewDevices.length > 0 && selectedNewDevices.length < filteredAvailableDevices.length}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelectedNewDevices(filteredAvailableDevices.map(d => d.id));
                  } else {
                    setSelectedNewDevices([]);
                  }
                }}
              >
                全选（{filteredAvailableDevices.length} 个可选设备）
              </Checkbox>
              <span style={{ color: '#666', fontSize: 12 }}>
                已选 {selectedNewDevices.length} 个
              </span>
            </div>
            <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, minHeight: 300, maxHeight: 300, overflow: 'auto' }}>
              {filteredAvailableDevices.length === 0 ? (
                <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>
                  暂无可添加的设备
                </div>
              ) : (
                <>
                  {/* 表头 */}
                  <div style={{ display: 'flex', padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0', fontWeight: 500, fontSize: 12 }}>
                    <div style={{ width: 40 }}></div>
                    <div style={{ flex: 1 }}>设备编码</div>
                    <div style={{ flex: 1 }}>设备名称</div>
                    <div style={{ width: 80, textAlign: 'center' }}>设备制式</div>
                  </div>
                  {/* 表体 */}
                  {paginatedDevices.map(device => (
                    <div
                      key={device.id}
                      onClick={() => {
                        setSelectedNewDevices(prev =>
                          prev.includes(device.id)
                            ? prev.filter(id => id !== device.id)
                            : [...prev, device.id]
                        );
                      }}
                      style={{
                        display: 'flex',
                        padding: '8px 12px',
                        cursor: 'pointer',
                        background: selectedNewDevices.includes(device.id) ? '#e6f4ff' : 'transparent',
                        borderBottom: '1px solid #f0f0f0',
                        alignItems: 'center',
                      }}
                    >
                      <div style={{ width: 40 }}>
                        <Checkbox
                          checked={selectedNewDevices.includes(device.id)}
                          onChange={() => {}}
                        />
                      </div>
                      <div style={{ flex: 1, fontSize: 12 }}>{device.id}</div>
                      <div style={{ flex: 1 }}>{device.name}</div>
                      <div style={{ width: 80, textAlign: 'center' }}>
                        <Tag>{device.neType}</Tag>
                      </div>
                    </div>
                  ))}
                </>
              )}
            </div>
            {filteredAvailableDevices.length > addDevicePageSize && (
              <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
                <Pagination
                  size="small"
                  current={addDeviceCurrentPage}
                  pageSize={addDevicePageSize}
                  total={filteredAvailableDevices.length}
                  onChange={(page, pageSize) => {
                    setAddDeviceCurrentPage(page);
                    setAddDevicePageSize(pageSize);
                  }}
                  showSizeChanger
                  showTotal={(total) => `共 ${total} 个`}
                />
              </div>
            )}
          </>
        ) : (
          <>
            {/* 设备组列表 */}
            <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Checkbox
                checked={selectedNewDeviceGroups.length === filteredAvailableDeviceGroups.length && filteredAvailableDeviceGroups.length > 0}
                indeterminate={selectedNewDeviceGroups.length > 0 && selectedNewDeviceGroups.length < filteredAvailableDeviceGroups.length}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelectedNewDeviceGroups(filteredAvailableDeviceGroups.map(g => g.id));
                  } else {
                    setSelectedNewDeviceGroups([]);
                  }
                }}
              >
                全选（{filteredAvailableDeviceGroups.length} 个可选设备组）
              </Checkbox>
              <span style={{ color: '#666', fontSize: 12 }}>
                已选 {selectedNewDeviceGroups.length} 个
              </span>
            </div>
            <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, minHeight: 300, maxHeight: 300, overflow: 'auto' }}>
              {filteredAvailableDeviceGroups.length === 0 ? (
                <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>
                  暂无可添加的设备组
                </div>
              ) : (
                <>
                  {/* 表头 */}
                  <div style={{ display: 'flex', padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0', fontWeight: 500, fontSize: 12 }}>
                    <div style={{ width: 40 }}></div>
                    <div style={{ flex: 1 }}>设备组名称</div>
                    <div style={{ width: 100, textAlign: 'center' }}>设备数量</div>
                  </div>
                  {/* 表体 */}
                  {paginatedDeviceGroups.map(group => (
                    <div
                      key={group.id}
                      onClick={() => {
                        setSelectedNewDeviceGroups(prev =>
                          prev.includes(group.id)
                            ? prev.filter(id => id !== group.id)
                            : [...prev, group.id]
                        );
                      }}
                      style={{
                        display: 'flex',
                        padding: '8px 12px',
                        cursor: 'pointer',
                        background: selectedNewDeviceGroups.includes(group.id) ? '#e6f4ff' : 'transparent',
                        borderBottom: '1px solid #f0f0f0',
                        alignItems: 'center',
                      }}
                    >
                      <div style={{ width: 40 }}>
                        <Checkbox
                          checked={selectedNewDeviceGroups.includes(group.id)}
                          onChange={() => {}}
                        />
                      </div>
                      <div style={{ flex: 1 }}>
                        <AlertOutlined style={{ marginRight: 8, color: '#FA8C16' }} />
                        {group.name}
                      </div>
                      <div style={{ width: 100, textAlign: 'center' }}>
                        <Text type="secondary">{group.deviceIds.length} 个</Text>
                      </div>
                    </div>
                  ))}
                </>
              )}
            </div>
            {filteredAvailableDeviceGroups.length > addDevicePageSize && (
              <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
                <Pagination
                  size="small"
                  current={addDeviceCurrentPage}
                  pageSize={addDevicePageSize}
                  total={filteredAvailableDeviceGroups.length}
                  onChange={(page, pageSize) => {
                    setAddDeviceCurrentPage(page);
                    setAddDevicePageSize(pageSize);
                  }}
                  showSizeChanger
                  showTotal={(total) => `共 ${total} 个`}
                />
              </div>
            )}
          </>
        )}
      </Modal>

      {/* 添加告警弹窗 */}
      <Modal
        open={addAlarmModalVisible}
        title="添加告警"
        onCancel={() => {
          setAddAlarmModalVisible(false);
          setSelectedNewAlarms([]);
          setAddAlarmKeyword('');
          setAddAlarmCurrentPage(1);
        }}
        onOk={() => {
          setSelectedAlarms(prev => [...new Set([...prev, ...selectedNewAlarms])]);
          setAddAlarmModalVisible(false);
          setSelectedNewAlarms([]);
          setAddAlarmKeyword('');
          setAddAlarmCurrentPage(1);
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={700}
      >
        <div style={{ marginBottom: 12 }}>
          <Input
            placeholder="搜索告警标识、告警源或可能原因"
            prefix={<SearchOutlined />}
            value={addAlarmKeyword}
            onChange={(e) => {
              setAddAlarmKeyword(e.target.value);
              setAddAlarmCurrentPage(1);
            }}
            allowClear
          />
        </div>
        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Checkbox
            checked={selectedNewAlarms.length === filteredAvailableAlarms.length && filteredAvailableAlarms.length > 0}
            indeterminate={selectedNewAlarms.length > 0 && selectedNewAlarms.length < filteredAvailableAlarms.length}
            onChange={(e) => {
              if (e.target.checked) {
                setSelectedNewAlarms(filteredAvailableAlarms.map(a => a.id));
              } else {
                setSelectedNewAlarms([]);
              }
            }}
          >
            全选（{filteredAvailableAlarms.length} 个可选告警）
          </Checkbox>
          <span style={{ color: '#666', fontSize: 12 }}>
            已选 {selectedNewAlarms.length} 个
          </span>
        </div>
        <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, minHeight: 300, maxHeight: 300, overflow: 'auto' }}>
          {filteredAvailableAlarms.length === 0 ? (
            <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>
              暂无可添加的告警
            </div>
          ) : (
            <Table
              size="small"
              dataSource={paginatedAlarms}
              rowKey="id"
              pagination={false}
              rowSelection={{
                selectedRowKeys: selectedNewAlarms,
                onChange: (keys) => setSelectedNewAlarms(keys as string[]),
              }}
              columns={[
                { title: '告警标识', dataIndex: 'alarmIdentifier', ellipsis: true },
                { title: '告警源', dataIndex: 'alarmSource', width: 80, render: (val) => <Tag>{val}</Tag> },
                { title: '可能原因', dataIndex: 'possibleCause', ellipsis: true },
              ]}
            />
          )}
        </div>
        {filteredAvailableAlarms.length > addAlarmPageSize && (
          <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
            <Pagination
              size="small"
              current={addAlarmCurrentPage}
              pageSize={addAlarmPageSize}
              total={filteredAvailableAlarms.length}
              onChange={(page, pageSize) => {
                setAddAlarmCurrentPage(page);
                setAddAlarmPageSize(pageSize);
              }}
              showSizeChanger
              showTotal={(total) => `共 ${total} 条`}
            />
          </div>
        )}
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
