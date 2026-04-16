import { useState, useMemo } from 'react';
import {
  Tag,
  message,
  Progress,
  Checkbox,
  Alert,
  Space,
  Modal,
  Input,
  Button,
  Select,
  Radio,
  DatePicker,
  Drawer,
  Form,
  Divider,
  Table,
  Card,
  Dropdown,
  Descriptions,
} from 'antd';
import type { MenuProps } from 'antd';
import type { Dayjs } from 'dayjs';
import { PlayCircleOutlined, WarningOutlined, PlusOutlined, ReloadOutlined, DownloadOutlined, DeleteOutlined, DesktopOutlined, PauseOutlined, StopOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// 升级类型枚举
type UpgradeType = 'immediate' | 'scheduled' | 'manual';
// 升级结果枚举
type UpgradeResult = 'success' | 'failed' | 'partial' | 'running' | 'pending';
// 执行方式枚举
type ExecutionMethod = 'immediate' | 'suspend' | 'scheduled';
// 任务状态枚举（1-6）
type TaskStatus = 1 | 2 | 3 | 4 | 5 | 6;

// 任务状态颜色配置
const TASK_STATUS_COLORS: Record<TaskStatus, string> = {
  1: 'default',
  2: 'processing',
  3: 'warning',
  4: 'success',
  5: 'error',
  6: 'warning',
};

interface UpgradePlanRow extends Record<string, unknown> {
  id: string;
  deviceSn: string;
  deviceName: string;
  deviceGroup: string;
  sourceVersion: string;
  targetVersion: string;
  upgradeType: UpgradeType;
  productType: string;
  keepConfig: boolean;
  progress: number;
  result: UpgradeResult;
  failureReason: string;
  operator: string;
  operateTime: string;
  startTime: string;
  endTime: string;
  taskName: string;
  status: TaskStatus; // 任务状态：1-等待, 2-进行中, 3-暂停, 4-已结束, 5-终止中, 6-暂停中
}

// 回退类型颜色
const UPGRADE_TYPE_COLORS: Record<UpgradeType, string> = {
  immediate: 'green',
  scheduled: 'blue',
  manual: 'orange',
};

// 升级结果颜色
const UPGRADE_RESULT_COLORS: Record<UpgradeResult, string> = {
  success: 'success',
  failed: 'error',
  partial: 'warning',
  running: 'processing',
  pending: 'default',
};

// Mock 数据
const mockData: UpgradePlanRow[] = [
  // status: 1=等待, 2=进行中, 3=暂停, 4=已结束, 5=终止中, 6=暂停中
  { id: '1', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 09:50:00', startTime: '2026-03-25 10:00:00', endTime: '2026-03-25 10:15:00', taskName: 'PM-B4860批量升级任务', status: 4 },
  { id: '2', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 09:50:00', startTime: '2026-03-25 10:00:00', endTime: '2026-03-25 10:12:00', taskName: 'PM-B4860批量升级任务', status: 4 },
  { id: '3', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFA', keepConfig: true, progress: 75, result: 'running', failureReason: '', operator: 'zhangsan', operateTime: '2026-03-25 10:55:00', startTime: '2026-03-25 11:00:00', endTime: '', taskName: 'QAFA定时升级任务', status: 2 },
  { id: '4', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'immediate', productType: 'BaiBNX', keepConfig: false, progress: 100, result: 'failed', failureReason: '固件校验失败', operator: 'lisi', operateTime: '2026-03-25 09:25:00', startTime: '2026-03-25 09:30:00', endTime: '2026-03-25 09:45:00', taskName: 'BaiBNX固件升级', status: 4 },
  { id: '5', deviceSn: 'GNB00002', deviceName: '上海5G基站01', deviceGroup: '上海移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'manual', productType: 'BaiBNX', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'admin', operateTime: '2026-03-25 11:00:00', startTime: '', endTime: '', taskName: 'BaiBNX手动升级', status: 1 },
  { id: '6', deviceSn: 'ENB00004', deviceName: '广州天河基站01', deviceGroup: '广东移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QATA', keepConfig: true, progress: 100, result: 'partial', failureReason: '部分配置恢复失败', operator: 'wangwu', operateTime: '2026-03-25 07:55:00', startTime: '2026-03-25 08:00:00', endTime: '2026-03-25 08:30:00', taskName: 'QATA紧急升级', status: 4 },
  { id: '7', deviceSn: 'ENB00005', deviceName: '深圳南山基站01', deviceGroup: '广东移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFB', keepConfig: true, progress: 50, result: 'running', failureReason: '', operator: 'zhangsan', operateTime: '2026-03-25 11:25:00', startTime: '2026-03-25 11:30:00', endTime: '', taskName: 'QAFB定时升级任务', status: 3 },
  { id: '8', deviceSn: 'GNB00003', deviceName: '广州5G基站01', deviceGroup: '广东移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'immediate', productType: 'BaiBNQ', keepConfig: false, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 08:55:00', startTime: '2026-03-25 09:00:00', endTime: '2026-03-25 09:20:00', taskName: 'BaiBNQ批量升级', status: 4 },
  { id: '9', deviceSn: 'ENB00006', deviceName: '杭州西湖基站01', deviceGroup: '浙江移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'manual', productType: 'RTD', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'lisi', operateTime: '2026-03-25 12:00:00', startTime: '', endTime: '', taskName: 'RTD手动升级任务', status: 1 },
  { id: '10', deviceSn: 'ENB00007', deviceName: '南京鼓楼基站01', deviceGroup: '江苏移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'wangwu', operateTime: '2026-03-25 10:25:00', startTime: '2026-03-25 10:30:00', endTime: '2026-03-25 10:45:00', taskName: 'PM-B4860批量升级任务', status: 4 },
  { id: '11', deviceSn: 'GNB00004', deviceName: '深圳5G基站01', deviceGroup: '广东移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'scheduled', productType: 'BaiBNX', keepConfig: true, progress: 30, result: 'running', failureReason: '', operator: 'admin', operateTime: '2026-03-25 11:55:00', startTime: '2026-03-25 12:00:00', endTime: '', taskName: 'BaiBNX定时升级', status: 5 },
  { id: '12', deviceSn: 'ENB00008', deviceName: '成都武侯基站01', deviceGroup: '四川移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QAFA', keepConfig: false, progress: 100, result: 'failed', failureReason: '网络连接超时', operator: 'zhangsan', operateTime: '2026-03-25 08:25:00', startTime: '2026-03-25 08:30:00', endTime: '2026-03-25 08:50:00', taskName: 'QAFA紧急升级', status: 4 },
  { id: '13', deviceSn: 'ENB00009', deviceName: '武汉洪山基站01', deviceGroup: '湖北移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QATA', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'lisi', operateTime: '2026-03-25 08:55:00', startTime: '2026-03-25 09:00:00', endTime: '2026-03-25 09:18:00', taskName: 'QATA批量升级任务', status: 4 },
  { id: '14', deviceSn: 'GNB00005', deviceName: '成都5G基站01', deviceGroup: '四川移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'manual', productType: 'BaiBNQ', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'wangwu', operateTime: '2026-03-25 12:00:00', startTime: '', endTime: '', taskName: 'BaiBNQ手动升级', status: 6 },
  { id: '15', deviceSn: 'ENB00010', deviceName: '西安雁塔基站01', deviceGroup: '陕西移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFB', keepConfig: true, progress: 60, result: 'running', failureReason: '', operator: 'admin', operateTime: '2026-03-25 10:55:00', startTime: '2026-03-25 11:00:00', endTime: '', taskName: 'QAFB定时升级任务', status: 2 },
];

export default function UpgradePlan() {
  const t = useT();

  // 任务状态配置（带 i18n）
  const TASK_STATUS_CONFIG = useMemo(() => ({
    1: { color: TASK_STATUS_COLORS[1], text: t('software.status.waiting') },
    2: { color: TASK_STATUS_COLORS[2], text: t('software.status.inProgress') },
    3: { color: TASK_STATUS_COLORS[3], text: t('software.status.paused') },
    4: { color: TASK_STATUS_COLORS[4], text: t('software.status.ended') },
    5: { color: TASK_STATUS_COLORS[5], text: t('software.status.terminating') },
    6: { color: TASK_STATUS_COLORS[6], text: t('software.status.pausing') },
  }), [t]);

  // 回退类型映射（带 i18n）
  const UPGRADE_TYPE_MAP = useMemo(() => ({
    immediate: { color: UPGRADE_TYPE_COLORS.immediate, text: t('software.rollback.immediateRollback') },
    scheduled: { color: UPGRADE_TYPE_COLORS.scheduled, text: t('software.rollback.scheduledRollback') },
    manual: { color: UPGRADE_TYPE_COLORS.manual, text: t('software.rollback.manualRollback') },
  }), [t]);

  // 升级结果映射（带 i18n）
  const UPGRADE_RESULT_MAP = useMemo(() => ({
    success: { color: UPGRADE_RESULT_COLORS.success, text: t('status.success') },
    failed: { color: UPGRADE_RESULT_COLORS.failed, text: t('status.failed') },
    partial: { color: UPGRADE_RESULT_COLORS.partial, text: t('software.status.partialSuccess') },
    running: { color: UPGRADE_RESULT_COLORS.running, text: t('software.status.rollingBack') },
    pending: { color: UPGRADE_RESULT_COLORS.pending, text: t('software.status.pendingStatus') },
  }), [t]);

  // 页签状态
  const [activeTab, setActiveTab] = useState<'task' | 'device'>('task');
  // 任务列表状态
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  // 批量输入相关状态
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');
  const [batchInputPreview, setBatchInputPreview] = useState<{
    matched: UpgradePlanRow[];
    notFound: string[];
    mixedTypes: string[];
  }>({ matched: [], notFound: [], mixedTypes: [] });
  // 重新执行确认状态
  const [retryRecord, setRetryRecord] = useState<UpgradePlanRow | null>(null);
  // 批量升级抽屉状态
  const [upgradeDrawerVisible, setUpgradeDrawerVisible] = useState(false);
  const [taskName, setTaskName] = useState('');
  const [executionMethod, setExecutionMethod] = useState<ExecutionMethod>('immediate');
  const [scheduledTime, setScheduledTime] = useState<Dayjs | null>(null);
  const [drawerDevices, setDrawerDevices] = useState<UpgradePlanRow[]>([]);
  const [drawerProductType, setDrawerProductType] = useState<string>('');
  // 抽屉中添加设备的状态
  const [addDeviceModalVisible, setAddDeviceModalVisible] = useState(false);
  const [selectedNewDevices, setSelectedNewDevices] = useState<React.Key[]>([]);
  const [addDeviceKeyword, setAddDeviceKeyword] = useState('');
  // 全选该产品类型设备的状态
  const [selectAllOfType, setSelectAllOfType] = useState(false);

  // 可添加的设备列表（同产品类型且不在已选列表中）
  const availableDevices = useMemo(() => {
    if (!drawerProductType) return [];
    const existingIds = new Set(drawerDevices.map((d) => d.id));
    return mockData.filter((d) => d.productType === drawerProductType && !existingIds.has(d.id));
  }, [drawerProductType, drawerDevices]);

  // 根据关键字过滤的可添加设备列表
  const filteredAvailableDevices = useMemo(() => {
    if (!addDeviceKeyword.trim()) return availableDevices;
    const keyword = addDeviceKeyword.toLowerCase();
    return availableDevices.filter(
      (d) => d.deviceSn.toLowerCase().includes(keyword) || d.deviceName.toLowerCase().includes(keyword)
    );
  }, [availableDevices, addDeviceKeyword]);

  // 该产品类型的全部设备总数（用于全选功能）
  const allDevicesCountOfType = useMemo(() => {
    if (!drawerProductType) return 0;
    return mockData.filter((d) => d.productType === drawerProductType).length;
  }, [drawerProductType]);

  // 解析批量输入的设备SN
  const parseBatchInput = (input: string): string[] => {
    // 支持换行、逗号、分号、空格分隔
    return input
      .split(/[\n,;,\s]+/)
      .map((s) => s.trim().toUpperCase())
      .filter((s) => s.length > 0);
  };

  // 批量输入预览
  const handleBatchInputPreview = () => {
    const sns = parseBatchInput(batchInputValue);
    if (sns.length === 0) {
      setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
      return;
    }

    const matched: UpgradePlanRow[] = [];
    const notFound: string[] = [];
    const foundSns = new Set<string>();

    // 在过滤后的数据中查找匹配的设备
    for (const sn of sns) {
      const row = filteredData.find(
        (r) => r.deviceSn.toUpperCase() === sn
      );
      if (row) {
        matched.push(row);
        foundSns.add(sn);
      } else {
        notFound.push(sn);
      }
    }

    // 检查产品类型是否一致
    const typeSet = new Set(matched.map((r) => r.productType));
    const mixedTypes = Array.from(typeSet);

    setBatchInputPreview({ matched, notFound, mixedTypes });
  };

  // 确认批量输入（添加到抽屉设备列表）
  const handleBatchInputConfirm = () => {
    const { matched } = batchInputPreview;

    if (matched.length === 0) {
      void message.warning(t('software.upgrade.noMatchedDevices'));
      return;
    }

    // 过滤出符合当前产品类型且不在已选列表中的设备
    const existingIds = new Set(drawerDevices.map((d) => d.id));
    const validDevices = matched.filter(
      (d) => d.productType === drawerProductType && !existingIds.has(d.id)
    );

    if (validDevices.length === 0) {
      void message.warning(t('software.upgrade.noAddableDevices'));
      return;
    }

    // 添加到抽屉设备列表
    setDrawerDevices((prev) => [...prev, ...validDevices]);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });

    void message.success(t('software.upgrade.addedDevices', { count: validDevices.length }));
  };

  // 打开批量输入弹窗
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
    setBatchInputVisible(true);
  };

  // 重新执行回退 - 打开确认弹窗
  const handleRetry = (record: UpgradePlanRow) => {
    setRetryRecord(record);
  };

  // 确认重新执行回退
  const handleRetryConfirm = () => {
    if (retryRecord) {
      void message.success(t('software.rollback.rerunSuccess', { sn: retryRecord.deviceSn }));
      setRetryRecord(null);
    }
  };

  // 直接导出
  const handleExport = () => {
    if (filteredData.length === 0) {
      void message.warning(t('software.upgrade.exportNoData'));
      return;
    }

    const headers = t('software.rollback.exportHeaders').split(',');
    const rows = filteredData.map((row) => [
      row.deviceSn,
      row.deviceName,
      row.deviceGroup,
      row.sourceVersion,
      row.targetVersion,
      UPGRADE_TYPE_MAP[row.upgradeType]?.text ?? row.upgradeType,
      row.productType,
      row.keepConfig ? t('common.yes') : t('common.no'),
      `${row.progress}%`,
      UPGRADE_RESULT_MAP[row.result]?.text ?? row.result,
      row.failureReason,
      row.operator,
      row.operateTime,
      row.startTime,
      row.endTime,
    ]);

    const csvContent = '\uFEFF' + [headers.join(','), ...rows.map((r) => r.map((c) => `"${c}"`).join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${t('software.rollback.exportFileName')}_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);

    void message.success(t('software.upgrade.exportSuccess', { count: filteredData.length }));
  };

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.stationCodeOrName'), type: 'input', placeholder: t('software.inputStationCodeOrName') },
    {
      name: 'productType',
      label: t('software.upgrade.productType'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: [
        { label: t('common.all'), value: 'all' },
        { label: 'PM-B4860', value: 'PM-B4860' },
        { label: 'QAFA', value: 'QAFA' },
        { label: 'QATA', value: 'QATA' },
        { label: 'QAFB', value: 'QAFB' },
        { label: 'RTD', value: 'RTD' },
        { label: 'BaiBNX', value: 'BaiBNX' },
        { label: 'BaiBNQ', value: 'BaiBNQ' },
        { label: 'BSC', value: 'BSC' },
        { label: 'BTS', value: 'BTS' },
      ],
    },
    {
      name: 'upgradeType',
      label: t('software.rollback.rollbackType'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('software.rollback.immediateRollback'), value: 'immediate' },
        { label: t('software.rollback.scheduledRollback'), value: 'scheduled' },
        { label: t('software.rollback.manualRollback'), value: 'manual' },
      ],
    },
    {
      name: 'result',
      label: t('table.result'),
      type: 'select',
      placeholder: t('common.pleaseSelect'),
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failed' },
        { label: t('software.status.partialSuccess'), value: 'partial' },
        { label: t('software.status.rollingBack'), value: 'running' },
        { label: t('software.status.pendingStatus'), value: 'pending' },
      ],
    },
    {
      name: 'timeRange',
      label: t('common.timeRange'),
      type: 'date-range',
      placeholder: t('common.selectTimeRange'),
    },
  ], [t]);

  // 任务列表筛选条件（只有任务名称和时间）
  const taskFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.taskName'), type: 'input', placeholder: t('software.upgrade.inputTaskName') },
    {
      name: 'timeRange',
      label: t('common.timeRange'),
      type: 'date-range',
      placeholder: t('common.selectTimeRange'),
    },
  ], [t]);

  // 过滤数据
  const filteredData = useMemo(() => {
    return mockData.filter((row) => {
      // 关键字搜索（基站编码或名称）
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.deviceSn.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      // 产品类型
      if (filters.productType && filters.productType !== 'all') {
        if (row.productType !== filters.productType) return false;
      }
      // 初始版本
      if (filters.sourceVersion && filters.sourceVersion !== 'all') {
        if (row.sourceVersion !== filters.sourceVersion) return false;
      }
      // 升级版本
      if (filters.targetVersion && filters.targetVersion !== 'all') {
        if (row.targetVersion !== filters.targetVersion) return false;
      }
      // 设备组
      if (filters.deviceGroup && filters.deviceGroup !== 'all') {
        if (row.deviceGroup !== filters.deviceGroup) return false;
      }
      // 升级类型
      if (filters.upgradeType && filters.upgradeType !== 'all') {
        if (row.upgradeType !== filters.upgradeType) return false;
      }
      // 结果
      if (filters.result && filters.result !== 'all') {
        if (row.result !== filters.result) return false;
      }
      // 时间范围（筛选开始时间或结束时间在范围内的记录）
      if (filters.timeRange && Array.isArray(filters.timeRange)) {
        const [start, end] = filters.timeRange as [Dayjs, Dayjs];
        if (start && end) {
          const startDate = start.toDate();
          const endDate = end.endOf('day').toDate();
          const rowStartTime = row.startTime ? new Date(row.startTime) : null;
          const rowEndTime = row.endTime ? new Date(row.endTime) : null;
          // 如果开始时间和结束时间都为空，则不匹配
          if (!rowStartTime && !rowEndTime) return false;
          // 检查是否有交集：记录的开始时间或结束时间在筛选范围内
          const hasOverlap =
            (rowStartTime && rowStartTime >= startDate && rowStartTime <= endDate) ||
            (rowEndTime && rowEndTime >= startDate && rowEndTime <= endDate) ||
            (rowStartTime && rowEndTime && rowStartTime <= startDate && rowEndTime >= endDate);
          if (!hasOverlap) return false;
        }
      }
      return true;
    });
  }, [filters]);

  // 直接打开升级抽屉（不需要预选设备）
  const handleOpenUpgradeDrawer = () => {
    setTaskName('');
    setDrawerDevices([]);
    setDrawerProductType('');
    setExecutionMethod('immediate');
    setScheduledTime(null);
    setSelectAllOfType(false);
    setUpgradeDrawerVisible(true);
  };

  // 从抽屉设备列表中移除设备
  const handleRemoveDevice = (deviceId: string) => {
    setDrawerDevices((prev) => prev.filter((d) => d.id !== deviceId));
  };

  // 打开添加设备弹窗
  const handleOpenAddDeviceModal = () => {
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    setAddDeviceModalVisible(true);
  };

  // 全选/取消全选
  const handleSelectAllDevices = (checked: boolean) => {
    if (checked) {
      setSelectedNewDevices(filteredAvailableDevices.map((d) => d.id));
    } else {
      setSelectedNewDevices([]);
    }
  };

  // 确认添加选中的设备
  const handleConfirmAddDevices = () => {
    if (selectedNewDevices.length === 0) {
      void message.warning(t('software.upgrade.selectDeviceToAdd'));
      return;
    }

    const newDevices = availableDevices.filter((d) => selectedNewDevices.includes(d.id));
    setDrawerDevices((prev) => [...prev, ...newDevices]);
    setAddDeviceModalVisible(false);
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    void message.success(t('software.upgrade.addedDevices', { count: newDevices.length }));
  };

  // 提交批量回退
  const handleSubmitUpgrade = () => {
    // 验证任务名称
    if (!taskName.trim()) {
      void message.warning(t('software.upgrade.inputTaskNameWarning'));
      return;
    }
    // 验证设备选择
    if (!selectAllOfType && drawerDevices.length === 0) {
      void message.warning(t('software.rollback.selectDeviceOrAll'));
      return;
    }
    if (selectAllOfType && allDevicesCountOfType === 0) {
      void message.warning(t('software.upgrade.noDevicesOfType'));
      return;
    }
    if (executionMethod === 'scheduled' && !scheduledTime) {
      void message.warning(t('software.upgrade.selectScheduleTime'));
      return;
    }

    // 提交回退任务
    const execMethodText = executionMethod === 'immediate' ? t('software.upgrade.immediateExecText') :
                          executionMethod === 'suspend' ? t('software.upgrade.suspendExecText') : t('software.upgrade.scheduledExecText', { time: scheduledTime?.format('YYYY-MM-DD HH:mm') });
    const deviceCount = selectAllOfType ? allDevicesCountOfType : drawerDevices.length;
    const deviceInfo = selectAllOfType
      ? t('software.upgrade.productTypeAll', { type: drawerProductType, count: deviceCount })
      : t('software.upgrade.deviceCountInfo', { count: deviceCount });

    void message.success(t('software.rollback.createRollbackSuccess', { name: taskName, deviceInfo, execMethod: execMethodText }));

    // 关闭抽屉并清空选择
    setUpgradeDrawerVisible(false);
    setTaskName('');
    setSelectAllOfType(false);
  };

  // 任务操作处理函数
  const handlePauseTask = (record: UpgradePlanRow) => {
    void message.success(t('software.upgrade.pausedTask', { name: record.deviceName }));
  };

  const handleStopTask = (record: UpgradePlanRow) => {
    void message.success(t('software.upgrade.stoppedTask', { name: record.deviceName }));
  };

  const handleStartTask = (record: UpgradePlanRow) => {
    void message.success(t('software.upgrade.startedTask', { name: record.deviceName }));
  };

  const handleEditTask = (record: UpgradePlanRow) => {
    void message.success(t('software.upgrade.editTask', { name: record.deviceName }));
  };

  const handleViewTaskDetail = (record: UpgradePlanRow) => {
    setTaskDetailRecord(record);
  };

  // 删除任务确认状态
  const [deleteTaskRecord, setDeleteTaskRecord] = useState<UpgradePlanRow | null>(null);
  // 任务详情抽屉状态
  const [taskDetailRecord, setTaskDetailRecord] = useState<UpgradePlanRow | null>(null);

  const handleDeleteTaskConfirm = () => {
    if (deleteTaskRecord) {
      void message.success(t('software.upgrade.deletedTask', { name: deleteTaskRecord.deviceName }));
      setDeleteTaskRecord(null);
    }
  };

  // 任务列表列定义
  // 状态操作矩阵：
  // | 状态码 | 状态名称 | 信息 | 修改 | 开始 | 暂停 | 终止 | 删除 |
  // |--------|----------|------|------|------|------|------|------|
  // | 1      | 等待     | 可用 | 可用 | 可用 | 隐藏 | 可用 | 可用 |
  // | 2      | 进行中   | 可用 | 隐藏 | 隐藏 | 可用 | 可用 | 隐藏 |
  // | 3      | 暂停     | 可用 | 隐藏 | 可用 | 隐藏 | 可用 | 可用 |
  // | 4      | 已结束   | 可用 | 隐藏 | 隐藏 | 隐藏 | 隐藏 | 可用 |
  // | 5      | 终止中   | 可用 | 隐藏 | 隐藏 | 隐藏 | 隐藏 | 可用 |
  // | 6      | 暂停中   | 可用 | 隐藏 | 可用 | 隐藏 | 可用 | 可用 |
  const taskColumns: DataTableColumn<UpgradePlanRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: UpgradePlanRow) => {
        const status = record.status;

        // 根据状态操作矩阵计算每个操作的可见性
        // 状态：1=等待, 2=进行中, 3=暂停, 4=已结束, 5=终止中, 6=暂停中
        const showEdit = status === 1;
        const showStart = status === 1 || status === 3 || status === 6;
        const showPause = status === 2;
        const showTerminate = status === 1 || status === 2 || status === 3 || status === 6;
        const showDelete = status !== 2;

        const items: MenuProps['items'] = [
          showEdit ? {
            key: 'edit',
            label: t('common.modify'),
            icon: <DesktopOutlined />,
            onClick: () => handleEditTask(record),
          } : null,
          showStart ? {
            key: 'start',
            label: t('common.start'),
            icon: <PlayCircleOutlined />,
            onClick: () => handleStartTask(record),
          } : null,
          showPause ? {
            key: 'pause',
            label: t('common.pause'),
            icon: <PauseOutlined />,
            onClick: () => handlePauseTask(record),
          } : null,
          showTerminate ? {
            key: 'terminate',
            label: t('common.terminate'),
            icon: <StopOutlined />,
            danger: true,
            onClick: () => handleStopTask(record),
          } : null,
          (showEdit || showStart || showPause || showTerminate) && showDelete ? { type: 'divider' } : null,
          showDelete ? {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => setDeleteTaskRecord(record),
          } : null,
        ].filter(Boolean) as MenuProps['items'];

        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleViewTaskDetail(record)}>{t('common.details')}</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'taskName',
      title: t('software.taskName'),
      dataIndex: 'deviceName',
      width: 150,
      ellipsis: true,
      render: (val: string, record: UpgradePlanRow) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleViewTaskDetail(record)}
          style={{ padding: 0 }}
        >
          {val || '-'}
        </Button>
      ),
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    { key: 'operateTime', title: t('software.operateTime'), dataIndex: 'operateTime', width: 160 },
    {
      key: 'status',
      title: t('software.taskStatus'),
      dataIndex: 'status',
      width: 100,
      render: (val: TaskStatus) => {
        const cfg = TASK_STATUS_CONFIG[val] ?? { color: 'default', text: String(val) };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productType', title: t('software.upgrade.productType'), dataIndex: 'productType', width: 100 },
    {
      key: 'progress',
      title: t('software.rollback.rollbackProgress'),
      dataIndex: 'progress',
      width: 120,
      render: (val: number) => <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />,
    },
    {
      key: 'result',
      title: t('table.result'),
      dataIndex: 'result',
      width: 100,
      render: (val: UpgradeResult) => {
        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'startTime', title: t('software.startTime'), dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: t('software.endTime'), dataIndex: 'endTime', width: 160 },
  ], [t, TASK_STATUS_CONFIG, UPGRADE_RESULT_MAP]);

  const columns: DataTableColumn<UpgradePlanRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: UpgradePlanRow) => {
        if (record.result === 'failed') {
          return (
            <Button
              type="link"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => handleRetry(record)}
            >
              {t('software.upgrade.rerun')}
            </Button>
          );
        }
        return null;
      },
    },
    { key: 'deviceSn', title: t('software.stationCode'), dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: t('software.stationName'), dataIndex: 'deviceName', width: 150, ellipsis: true },
    { key: 'taskName', title: t('software.taskName'), dataIndex: 'taskName', width: 150, ellipsis: true },
    { key: 'targetVersion', title: t('software.rollback.originalVersion'), dataIndex: 'targetVersion', width: 100 },
    { key: 'productType', title: t('software.upgrade.productType'), dataIndex: 'productType', width: 100 },
    {
      key: 'progress',
      title: t('software.rollback.rollbackProgress'),
      dataIndex: 'progress',
      width: 120,
      render: (val: number) => <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />,
    },
    {
      key: 'result',
      title: t('table.result'),
      dataIndex: 'result',
      width: 100,
      render: (val: UpgradeResult) => {
        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'failureReason', title: t('software.failureReason'), dataIndex: 'failureReason', width: 150, ellipsis: true },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    { key: 'operateTime', title: t('software.operateTime'), dataIndex: 'operateTime', width: 160 },
    { key: 'startTime', title: t('software.startTime'), dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: t('software.endTime'), dataIndex: 'endTime', width: 160 },
  ], [t, UPGRADE_RESULT_MAP]);

  // 页面头部按钮
  const headerExtra = useMemo(() => (
    <Space>
      <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleOpenUpgradeDrawer}>
        {t('software.rollback.rollback')}
      </Button>
      <Button icon={<DownloadOutlined />} onClick={handleExport}>
        {t('common.export')}
      </Button>
    </Space>
  ), [t]);

  return (
    <ListPageLayout title={t('nav.software.versionRollback')} extra={headerExtra}>
      {/* 页签选择 */}
      <Radio.Group
        value={activeTab}
        onChange={(e) => {
          setActiveTab(e.target.value);
          setFilters({});
          setPage(1);
        }}
        optionType="button"
        buttonStyle="solid"
        style={{ marginBottom: 12 }}
      >
        <Radio.Button value="task">{t('software.upgrade.taskList')}</Radio.Button>
        <Radio.Button value="device">{t('software.upgrade.deviceList')}</Radio.Button>
      </Radio.Group>

      {/* 搜索表单 */}
      <FilterBar
        filterId={`upgrade-plan-filter-${activeTab}`}
        fields={activeTab === 'task' ? taskFilterFields : filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
          onReset={() => { setFilters({}); setPage(1); }}
        />

      {/* 列表 */}
      {activeTab === 'task' ? (
        <DataTable<UpgradePlanRow>
          tableId="upgrade-plan-list-task"
          columns={taskColumns}
          dataSource={filteredData}
          rowKey="id"
          total={filteredData.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          scroll={{ x: 'max-content', y: 'calc(100vh - 540px)' }}
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
        />
      ) : (
        <DataTable<UpgradePlanRow>
          tableId="upgrade-plan-list-device"
          columns={columns}
          dataSource={filteredData}
          rowKey="id"
          total={filteredData.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          scroll={{ x: 'max-content', y: 'calc(100vh - 540px)' }}
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
        />
      )}

      {/* 批量输入弹窗 */}
      <Modal
        title={t('software.upgrade.batchInputTitle', { type: drawerProductType || t('software.upgrade.selectProductTypeFirst') })}
        open={batchInputVisible}
        onCancel={() => setBatchInputVisible(false)}
        onOk={handleBatchInputConfirm}
        okText={t('software.upgrade.confirmAdd')}
        cancelText={t('common.cancel')}
        width={600}
        okButtonProps={{
          disabled: batchInputPreview.matched.length === 0 || !drawerProductType,
        }}
      >
        {!drawerProductType && (
          <Alert
            type="warning"
            showIcon
            message={t('software.upgrade.selectProductTypeFirst')}
            style={{ marginBottom: 16 }}
          />
        )}

        <div style={{ marginBottom: 16 }}>
          <Input.TextArea
            placeholder={`${t('software.upgrade.batchInputPlaceholder')}\n${t('software.upgrade.batchInputExample')}`}
            rows={6}
            value={batchInputValue}
            onChange={(e) => setBatchInputValue(e.target.value)}
            onBlur={handleBatchInputPreview}
          />
        </div>

        {/* 预览结果 */}
        {batchInputPreview.matched.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Alert
              type={batchInputPreview.mixedTypes.includes(drawerProductType) ? 'success' : 'warning'}
              showIcon
              icon={!batchInputPreview.mixedTypes.includes(drawerProductType) ? <WarningOutlined /> : undefined}
              message={
                <Space direction="vertical" size="small">
                  <span>
                    {t('software.upgrade.matchedDevices', { count: batchInputPreview.matched.length })}
                    {batchInputPreview.mixedTypes.includes(drawerProductType) && (
                      <Tag color="blue" style={{ marginLeft: 8 }}>{t('software.upgrade.addableCount', { count: batchInputPreview.matched.filter(d => d.productType === drawerProductType).length })}</Tag>
                    )}
                  </span>
                  {!batchInputPreview.mixedTypes.includes(drawerProductType) && drawerProductType && (
                    <span style={{ color: '#faad14' }}>
                      <WarningOutlined style={{ marginRight: 4 }} />
                      {t('software.upgrade.noMatchedType', { type: drawerProductType })}
                    </span>
                  )}
                </Space>
              }
              style={{ marginBottom: 8 }}
            />
          </div>
        )}

        {/* 未找到的设备 */}
        {batchInputPreview.notFound.length > 0 && (
          <Alert
            type="warning"
            showIcon
            message={
              <div>
                <div>{t('software.upgrade.notFoundSns', { count: batchInputPreview.notFound.length })}</div>
                <div style={{ maxHeight: 80, overflow: 'auto', marginTop: 4 }}>
                  {batchInputPreview.notFound.map((sn) => (
                    <Tag key={sn} style={{ margin: '2px' }}>{sn}</Tag>
                  ))}
                </div>
              </div>
            }
            style={{ marginBottom: 8 }}
          />
        )}
      </Modal>

      {/* 重新执行确认弹窗 */}
      <Modal
        title={t('software.upgrade.confirmRerun')}
        open={!!retryRecord}
        onCancel={() => setRetryRecord(null)}
        onOk={handleRetryConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                {t('software.rollback.confirmRerunRollback')}
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>{t('software.upgrade.stationCodeLabel')}</strong>{retryRecord?.deviceSn}<br />
                <strong>{t('software.upgrade.stationNameLabel')}</strong>{retryRecord?.deviceName}<br />
                <strong>{t('software.upgrade.targetVersionLabel')}</strong>{retryRecord?.targetVersion}
              </p>
            </div>
          }
        />
      </Modal>

      {/* 删除任务确认弹窗 */}
      <Modal
        title={t('common.confirmDelete')}
        open={!!deleteTaskRecord}
        onCancel={() => setDeleteTaskRecord(null)}
        onOk={handleDeleteTaskConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                {t('software.rollback.confirmDeleteRollback')}
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>{t('software.upgrade.taskNameLabel')}</strong>{deleteTaskRecord?.deviceName}<br />
                <strong>{t('software.upgrade.operatorLabel')}</strong>{deleteTaskRecord?.operator}<br />
                <strong>{t('software.upgrade.rollbackVersionLabel')}</strong>{deleteTaskRecord?.targetVersion}
              </p>
            </div>
          }
        />
      </Modal>

      {/* 批量回退抽屉 */}
      <Drawer
        title={t('software.rollback.batchRollback')}
        placement="right"
        width={600}
        open={upgradeDrawerVisible}
        onClose={() => setUpgradeDrawerVisible(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setUpgradeDrawerVisible(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              onClick={handleSubmitUpgrade}
              disabled={!selectAllOfType && drawerDevices.length === 0}
            >
              {t('software.rollback.confirmRollback')}
            </Button>
          </Space>
        }
      >
        <Form layout="vertical" size="small">
          {/* 任务名称 */}
          <Form.Item label={t('software.upgrade.taskName')} required>
            <Input
              value={taskName}
              onChange={(e) => setTaskName(e.target.value)}
              placeholder={t('software.upgrade.inputTaskName')}
              maxLength={100}
              showCount
            />
          </Form.Item>

          {/* 已选产品类型 */}
          <Form.Item label={t('software.upgrade.productType')} required>
            <Select
              value={drawerProductType}
              onChange={(val) => {
                setDrawerProductType(val);
                // 切换产品类型时，重置全选状态并清空不匹配的设备
                setSelectAllOfType(false);
                setDrawerDevices((prev) => prev.filter((d) => d.productType === val));
              }}
              options={[
                { label: 'PM-B4860', value: 'PM-B4860' },
                { label: 'QAFA', value: 'QAFA' },
                { label: 'QATA', value: 'QATA' },
                { label: 'QAFB', value: 'QAFB' },
                { label: 'RTD', value: 'RTD' },
                { label: 'BaiBNX', value: 'BaiBNX' },
                { label: 'BaiBNQ', value: 'BaiBNQ' },
              ]}
              style={{ width: '100%' }}
            />
          </Form.Item>

          {/* 全选该产品类型设备 */}
          <Form.Item>
            <Checkbox
              checked={selectAllOfType}
              onChange={(e) => setSelectAllOfType(e.target.checked)}
              disabled={!drawerProductType}
            >
              {t('software.rollback.rollbackAllOfType')}
              {drawerProductType && (
                <Tag color="blue" style={{ marginLeft: 8 }}>{t('software.upgrade.totalDevices', { count: allDevicesCountOfType })}</Tag>
              )}
            </Checkbox>
          </Form.Item>

          {/* 已选回退设备 */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
              <span>
                {t('software.rollback.selectedRollbackDevices')}
                {' '}
                <Tag color="blue">{selectAllOfType ? allDevicesCountOfType : drawerDevices.length} {t('common.devices')}</Tag>
              </span>
              {!selectAllOfType && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleOpenBatchInput}
                  disabled={!drawerProductType}
                >
                  {t('software.upgrade.batchInput')}
                </Button>
              )}
            </div>
          }>
            {selectAllOfType ? (
              <Alert
                type="info"
                showIcon
                message={t('software.rollback.selectedAllRollbackInfo', { type: drawerProductType, count: allDevicesCountOfType })}
                description={t('software.rollback.selectedAllRollbackDesc')}
              />
            ) : (
              <>
                <div style={{ maxHeight: 200, overflow: 'auto', border: '1px solid #d9d9d9', borderRadius: 6 }}>
                  <Table
                    size="small"
                    dataSource={drawerDevices}
                    rowKey="id"
                    pagination={false}
                    columns={[
                      { title: t('software.stationCode'), dataIndex: 'deviceSn', width: 100 },
                      { title: t('software.stationName'), dataIndex: 'deviceName', ellipsis: true },
                      { title: t('software.upgrade.currentVersion'), dataIndex: 'sourceVersion', width: 80 },
                      {
                        title: '',
                        width: 40,
                        render: (_: unknown, record: UpgradePlanRow) => (
                          <Button
                            type="text"
                            size="small"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => handleRemoveDevice(record.id)}
                          />
                        ),
                      },
                    ]}
                  />
                </div>
                {/* 添加设备按钮 */}
                <div style={{ marginTop: 8 }}>
                  <Button
                    type="dashed"
                    icon={<PlusOutlined />}
                    onClick={handleOpenAddDeviceModal}
                    style={{ width: '100%' }}
                    disabled={!drawerProductType}
                  >
                    {t('software.upgrade.addDevice')}
                  </Button>
                </div>
              </>
            )}
          </Form.Item>

          <Divider />

          {/* 执行方式 */}
          <Form.Item label={t('software.upgrade.executionMethod')} required>
            <Radio.Group value={executionMethod} onChange={(e) => setExecutionMethod(e.target.value)}>
              <Radio value="immediate">{t('software.upgrade.immediateExecute')}</Radio>
              <Radio value="suspend">{t('software.upgrade.suspendExecute')}</Radio>
              <Radio value="scheduled">{t('software.upgrade.scheduledExecute')}</Radio>
            </Radio.Group>
          </Form.Item>

          {/* 定时执行时间 */}
          {executionMethod === 'scheduled' && (
            <Form.Item label={t('software.upgrade.executeTime')} required>
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm"
                value={scheduledTime}
                onChange={setScheduledTime}
                placeholder={t('software.upgrade.selectExecuteTime')}
                style={{ width: '100%' }}
                disabledDate={(current) => current && current.isBefore(new Date(), 'day')}
              />
            </Form.Item>
          )}
        </Form>
      </Drawer>

      {/* 任务详情抽屉 */}
      <Drawer
        title={t('software.upgrade.taskDetail')}
        placement="right"
        width={720}
        open={!!taskDetailRecord}
        onClose={() => setTaskDetailRecord(null)}
        footer={null}
      >
        {taskDetailRecord && (() => {
          // 根据任务名称查找该任务下的所有设备
          const taskName = taskDetailRecord.taskName || taskDetailRecord.deviceName;
          const taskDevices = mockData.filter(d => (d.taskName || d.deviceName) === taskName);
          const totalDevices = taskDevices.length;

          // 统计执行结果
          const resultStats = {
            success: taskDevices.filter(d => d.result === 'success').length,
            failed: taskDevices.filter(d => d.result === 'failed').length,
            running: taskDevices.filter(d => d.result === 'running').length,
            pending: taskDevices.filter(d => d.result === 'pending').length,
            partial: taskDevices.filter(d => d.result === 'partial').length,
          };

          // 计算总体进度
          const totalProgress = taskDevices.reduce((sum, d) => sum + d.progress, 0);
          const avgProgress = Math.round(totalProgress / totalDevices);

          return (
            <>
              {/* 任务基本信息 */}
              <Descriptions column={2} bordered size="small" style={{ marginBottom: 16 }}>
                <Descriptions.Item label={t('software.taskName')} span={2}>{taskName}</Descriptions.Item>
                <Descriptions.Item label={t('software.upgrade.productType')}>{taskDetailRecord.productType}</Descriptions.Item>
                <Descriptions.Item label={t('software.rollback.rollbackType')}>
                  <Tag color={UPGRADE_TYPE_MAP[taskDetailRecord.upgradeType]?.color}>
                    {UPGRADE_TYPE_MAP[taskDetailRecord.upgradeType]?.text || taskDetailRecord.upgradeType}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('software.rollback.originalVersion')}>{taskDetailRecord.targetVersion}</Descriptions.Item>
                <Descriptions.Item label={t('software.upgrade.keepConfig')}>
                  <Checkbox checked={taskDetailRecord.keepConfig} disabled />
                </Descriptions.Item>
                <Descriptions.Item label={t('table.operator')}>{taskDetailRecord.operator}</Descriptions.Item>
                <Descriptions.Item label={t('software.operateTime')}>{taskDetailRecord.operateTime || '-'}</Descriptions.Item>
              </Descriptions>

              {/* 执行进度概览 */}
              <Card title={t('software.upgrade.progressOverview')} size="small" style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                  <div style={{ flex: '0 0 120px', textAlign: 'center' }}>
                    <Progress
                      type="circle"
                      percent={avgProgress}
                      size={80}
                      status={avgProgress === 100 ? 'success' : 'active'}
                    />
                    <div style={{ marginTop: 8, color: '#666' }}>{t('software.upgrade.overallProgress')}</div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <div>
                        <Tag color="success">{t('status.success')}</Tag>
                        <span>{resultStats.success} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="error">{t('status.failed')}</Tag>
                        <span>{resultStats.failed} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="processing">{t('software.status.rollingBack')}</Tag>
                        <span>{resultStats.running} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="warning">{t('software.status.partialSuccess')}</Tag>
                        <span>{resultStats.partial} {t('common.devices')}</span>
                      </div>
                      <div>
                        <Tag color="default">{t('software.status.pendingStatus')}</Tag>
                        <span>{resultStats.pending} {t('common.devices')}</span>
                      </div>
                    </Space>
                  </div>
                  <Divider type="vertical" style={{ height: 120 }} />
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 32, fontWeight: 'bold', color: '#1890ff' }}>{totalDevices}</div>
                    <div style={{ color: '#666' }}>{t('software.upgrade.deviceTotal')}</div>
                  </div>
                </div>
              </Card>

              {/* 设备列表 */}
              <Card title={t('software.upgrade.deviceListTitle', { count: totalDevices })} size="small">
                <Table
                  size="small"
                  dataSource={taskDevices}
                  rowKey="id"
                  pagination={totalDevices > 10 ? { pageSize: 10 } : false}
                  scroll={{ y: 300 }}
                  columns={[
                    {
                      title: t('software.stationCode'),
                      dataIndex: 'deviceSn',
                      width: 100,
                    },
                    {
                      title: t('software.stationName'),
                      dataIndex: 'deviceName',
                      ellipsis: true,
                    },
                    {
                      title: t('software.upgrade.currentVersion'),
                      dataIndex: 'sourceVersion',
                      width: 80,
                    },
                    {
                      title: t('software.progress'),
                      dataIndex: 'progress',
                      width: 100,
                      render: (val: number) => (
                        <Progress
                          percent={val}
                          size="small"
                          status={val === 100 ? 'success' : 'active'}
                        />
                      ),
                    },
                    {
                      title: t('table.result'),
                      dataIndex: 'result',
                      width: 80,
                      render: (val: UpgradeResult) => {
                        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
                        return <Tag color={cfg.color}>{cfg.text}</Tag>;
                      },
                    },
                    {
                      title: t('software.failureReason'),
                      dataIndex: 'failureReason',
                      width: 120,
                      ellipsis: true,
                      render: (val: string) => val ? (
                        <span style={{ color: '#ff4d4f' }}>{val}</span>
                      ) : '-',
                    },
                  ]}
                />
              </Card>
            </>
          );
        })()}
      </Drawer>

      {/* 添加设备弹窗 */}
      <Modal
        title={t('software.upgrade.addDeviceTitle', { type: drawerProductType })}
        open={addDeviceModalVisible}
        onCancel={() => setAddDeviceModalVisible(false)}
        onOk={handleConfirmAddDevices}
        okText={t('software.upgrade.confirmAdd')}
        cancelText={t('common.cancel')}
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message={t('software.upgrade.noAvailableDevices')}
            description={t('software.upgrade.allDevicesInList', { type: drawerProductType })}
          />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={t('software.upgrade.deviceSelectable', { total: availableDevices.length, shown: filteredAvailableDevices.length, selected: selectedNewDevices.length })}
              style={{ marginBottom: 16 }}
            />

            {/* 搜索和全选 */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input.Search
                placeholder={t('software.upgrade.searchStationCodeOrName')}
                value={addDeviceKeyword}
                onChange={(e) => setAddDeviceKeyword(e.target.value)}
                style={{ width: 250 }}
                allowClear
              />
              <Checkbox
                checked={selectedNewDevices.length === filteredAvailableDevices.length && filteredAvailableDevices.length > 0}
                indeterminate={selectedNewDevices.length > 0 && selectedNewDevices.length < filteredAvailableDevices.length}
                onChange={(e) => handleSelectAllDevices(e.target.checked)}
              >
                {t('common.selectAll')} ({filteredAvailableDevices.length} {t('common.devices')})
              </Checkbox>
            </div>

            <Table
              size="small"
              dataSource={filteredAvailableDevices}
              rowKey="id"
              pagination={false}
              scroll={{ y: 250 }}
              rowSelection={{
                selectedRowKeys: selectedNewDevices,
                onChange: (keys) => setSelectedNewDevices(keys),
              }}
              columns={[
                { title: t('software.stationCode'), dataIndex: 'deviceSn', width: 120 },
                { title: t('software.stationName'), dataIndex: 'deviceName', ellipsis: true },
                { title: t('software.deviceGroup'), dataIndex: 'deviceGroup', width: 100 },
                { title: t('software.upgrade.currentVersion'), dataIndex: 'sourceVersion', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>
    </ListPageLayout>
  );
}
