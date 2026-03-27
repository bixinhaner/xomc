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
  InputNumber,
  Form,
  Divider,
  Table,
  Card,
  Dropdown,
  Descriptions,
} from 'antd';
import type { MenuProps } from 'antd';
import type { Dayjs } from 'dayjs';
import { PlayCircleOutlined, WarningOutlined, PlusOutlined, ReloadOutlined, DownloadOutlined, DeleteOutlined, UnorderedListOutlined, DesktopOutlined, PauseOutlined, StopOutlined, InfoCircleOutlined, MoreOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// 升级类型枚举
type UpgradeType = 'immediate' | 'scheduled' | 'manual';
// 升级结果枚举
type UpgradeResult = 'success' | 'failed' | 'partial' | 'running' | 'pending';
// 升级类别枚举
type UpgradeCategory = 'software' | 'patch' | 'fpga';
// 执行方式枚举
type ExecutionMethod = 'immediate' | 'suspend' | 'scheduled';

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
}

// 升级类型映射
const UPGRADE_TYPE_MAP: Record<UpgradeType, { color: string; text: string }> = {
  immediate: { color: 'green', text: '立即升级' },
  scheduled: { color: 'blue', text: '定时升级' },
  manual: { color: 'orange', text: '手动升级' },
};

// 升级结果映射
const UPGRADE_RESULT_MAP: Record<UpgradeResult, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  partial: { color: 'warning', text: '部分成功' },
  running: { color: 'processing', text: '升级中' },
  pending: { color: 'default', text: '等待中' },
};

// Mock 数据
const mockData: UpgradePlanRow[] = [
  { id: '1', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 09:50:00', startTime: '2026-03-25 10:00:00', endTime: '2026-03-25 10:15:00', taskName: 'PM-B4860批量升级任务' },
  { id: '2', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 09:50:00', startTime: '2026-03-25 10:00:00', endTime: '2026-03-25 10:12:00', taskName: 'PM-B4860批量升级任务' },
  { id: '3', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFA', keepConfig: true, progress: 75, result: 'running', failureReason: '', operator: 'zhangsan', operateTime: '2026-03-25 10:55:00', startTime: '2026-03-25 11:00:00', endTime: '', taskName: 'QAFA定时升级任务' },
  { id: '4', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'immediate', productType: 'BaiBNX', keepConfig: false, progress: 100, result: 'failed', failureReason: '固件校验失败', operator: 'lisi', operateTime: '2026-03-25 09:25:00', startTime: '2026-03-25 09:30:00', endTime: '2026-03-25 09:45:00', taskName: 'BaiBNX固件升级' },
  { id: '5', deviceSn: 'GNB00002', deviceName: '上海5G基站01', deviceGroup: '上海移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'manual', productType: 'BaiBNX', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'admin', operateTime: '2026-03-25 11:00:00', startTime: '', endTime: '', taskName: 'BaiBNX手动升级' },
  { id: '6', deviceSn: 'ENB00004', deviceName: '广州天河基站01', deviceGroup: '广东移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QATA', keepConfig: true, progress: 100, result: 'partial', failureReason: '部分配置恢复失败', operator: 'wangwu', operateTime: '2026-03-25 07:55:00', startTime: '2026-03-25 08:00:00', endTime: '2026-03-25 08:30:00', taskName: 'QATA紧急升级' },
  { id: '7', deviceSn: 'ENB00005', deviceName: '深圳南山基站01', deviceGroup: '广东移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFB', keepConfig: true, progress: 50, result: 'running', failureReason: '', operator: 'zhangsan', operateTime: '2026-03-25 11:25:00', startTime: '2026-03-25 11:30:00', endTime: '', taskName: 'QAFB定时升级任务' },
  { id: '8', deviceSn: 'GNB00003', deviceName: '广州5G基站01', deviceGroup: '广东移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'immediate', productType: 'BaiBNQ', keepConfig: false, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 08:55:00', startTime: '2026-03-25 09:00:00', endTime: '2026-03-25 09:20:00', taskName: 'BaiBNQ批量升级' },
  { id: '9', deviceSn: 'ENB00006', deviceName: '杭州西湖基站01', deviceGroup: '浙江移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'manual', productType: 'RTD', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'lisi', operateTime: '2026-03-25 12:00:00', startTime: '', endTime: '', taskName: 'RTD手动升级任务' },
  { id: '10', deviceSn: 'ENB00007', deviceName: '南京鼓楼基站01', deviceGroup: '江苏移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'wangwu', operateTime: '2026-03-25 10:25:00', startTime: '2026-03-25 10:30:00', endTime: '2026-03-25 10:45:00', taskName: 'PM-B4860批量升级任务' },
  { id: '11', deviceSn: 'GNB00004', deviceName: '深圳5G基站01', deviceGroup: '广东移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'scheduled', productType: 'BaiBNX', keepConfig: true, progress: 30, result: 'running', failureReason: '', operator: 'admin', operateTime: '2026-03-25 11:55:00', startTime: '2026-03-25 12:00:00', endTime: '', taskName: 'BaiBNX定时升级' },
  { id: '12', deviceSn: 'ENB00008', deviceName: '成都武侯基站01', deviceGroup: '四川移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QAFA', keepConfig: false, progress: 100, result: 'failed', failureReason: '网络连接超时', operator: 'zhangsan', operateTime: '2026-03-25 08:25:00', startTime: '2026-03-25 08:30:00', endTime: '2026-03-25 08:50:00', taskName: 'QAFA紧急升级' },
  { id: '13', deviceSn: 'ENB00009', deviceName: '武汉洪山基站01', deviceGroup: '湖北移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QATA', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'lisi', operateTime: '2026-03-25 08:55:00', startTime: '2026-03-25 09:00:00', endTime: '2026-03-25 09:18:00', taskName: 'QATA批量升级任务' },
  { id: '14', deviceSn: 'GNB00005', deviceName: '成都5G基站01', deviceGroup: '四川移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'manual', productType: 'BaiBNQ', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'wangwu', operateTime: '2026-03-25 12:00:00', startTime: '', endTime: '', taskName: 'BaiBNQ手动升级' },
  { id: '15', deviceSn: 'ENB00010', deviceName: '西安雁塔基站01', deviceGroup: '陕西移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFB', keepConfig: true, progress: 60, result: 'running', failureReason: '', operator: 'admin', operateTime: '2026-03-25 10:55:00', startTime: '2026-03-25 11:00:00', endTime: '', taskName: 'QAFB定时升级任务' },
];

export default function UpgradePlan() {
  const t = useT();
  // 页签状态
  const [activeTab, setActiveTab] = useState<'task' | 'device'>('task');
  // 任务列表状态
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  // 设备列表状态
  const [deviceFilters, setDeviceFilters] = useState<Record<string, unknown>>({});
  const [devicePage, setDevicePage] = useState(1);
  const [devicePageSize, setDevicePageSize] = useState(20);
  // 批量输入相关状态
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');
  const [batchInputPreview, setBatchInputPreview] = useState<{
    matched: UpgradePlanRow[];
    notFound: string[];
    mixedTypes: string[];
  }>({ matched: [], notFound: [], mixedTypes: [] });
  // 导出相关状态
  const [exportVisible, setExportVisible] = useState(false);
  const [exportType, setExportType] = useState<'all' | 'range'>('all');
  const [exportTimeRange, setExportTimeRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  // 重新执行确认状态
  const [retryRecord, setRetryRecord] = useState<UpgradePlanRow | null>(null);
  // 批量升级抽屉状态
  const [upgradeDrawerVisible, setUpgradeDrawerVisible] = useState(false);
  const [taskName, setTaskName] = useState('');
  const [upgradeCategory, setUpgradeCategory] = useState<UpgradeCategory>('software');
  const [executionMethod, setExecutionMethod] = useState<ExecutionMethod>('immediate');
  const [scheduledTime, setScheduledTime] = useState<Dayjs | null>(null);
  const [upgradeFile, setUpgradeFile] = useState<string | undefined>(undefined);
  const [drawerKeepConfig, setDrawerKeepConfig] = useState(true);
  const [retryOffline, setRetryOffline] = useState(true);
  const [batchSize, setBatchSize] = useState(20);
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

  // Mock 升级文件列表
  const upgradeFiles = useMemo(() => [
    { label: 'V1.3.0_full.bin', value: 'V1.3.0_full.bin', category: 'software', productType: 'PM-B4860' },
    { label: 'V1.3.0_patch.bin', value: 'V1.3.0_patch.bin', category: 'patch', productType: 'PM-B4860' },
    { label: 'V2.1.0_full.bin', value: 'V2.1.0_full.bin', category: 'software', productType: 'BaiBNX' },
    { label: 'V2.1.0_fpga.bin', value: 'V2.1.0_fpga.bin', category: 'fpga', productType: 'BaiBNX' },
  ], []);

  // 根据产品类型和升级类别过滤文件
  const filteredFiles = useMemo(() => {
    return upgradeFiles.filter(
      (f) => f.productType === drawerProductType && f.category === upgradeCategory
    );
  }, [upgradeFiles, drawerProductType, upgradeCategory]);

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
      void message.warning('没有匹配到任何设备');
      return;
    }

    // 过滤出符合当前产品类型且不在已选列表中的设备
    const existingIds = new Set(drawerDevices.map((d) => d.id));
    const validDevices = matched.filter(
      (d) => d.productType === drawerProductType && !existingIds.has(d.id)
    );

    if (validDevices.length === 0) {
      void message.warning('没有可添加的设备（设备类型不匹配或已存在）');
      return;
    }

    // 添加到抽屉设备列表
    setDrawerDevices((prev) => [...prev, ...validDevices]);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });

    void message.success(`已添加 ${validDevices.length} 台设备`);
  };

  // 打开批量输入弹窗
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
    setBatchInputVisible(true);
  };

  // 重新执行升级 - 打开确认弹窗
  const handleRetry = (record: UpgradePlanRow) => {
    setRetryRecord(record);
  };

  // 确认重新执行升级
  const handleRetryConfirm = () => {
    if (retryRecord) {
      void message.success(`已重新发起 ${retryRecord.deviceSn} 的升级任务`);
      setRetryRecord(null);
    }
  };

  // 打开导出弹窗
  const handleOpenExport = () => {
    setExportType('all');
    setExportTimeRange(null);
    setExportVisible(true);
  };

  // 执行导出
  const handleExport = () => {
    let dataToExport: UpgradePlanRow[];

    if (exportType === 'all') {
      dataToExport = filteredData;
    } else {
      if (!exportTimeRange || !exportTimeRange[0] || !exportTimeRange[1]) {
        void message.warning('请选择时间范围');
        return;
      }
      const startDate = exportTimeRange[0].toDate();
      const endDate = exportTimeRange[1].endOf('day').toDate();
      dataToExport = filteredData.filter((row) => {
        const rowStartTime = row.startTime ? new Date(row.startTime) : null;
        const rowEndTime = row.endTime ? new Date(row.endTime) : null;
        if (!rowStartTime && !rowEndTime) return false;
        return (
          (rowStartTime && rowStartTime >= startDate && rowStartTime <= endDate) ||
          (rowEndTime && rowEndTime >= startDate && rowEndTime <= endDate) ||
          (rowStartTime && rowEndTime && rowStartTime <= startDate && rowEndTime >= endDate)
        );
      });
    }

    if (dataToExport.length === 0) {
      void message.warning('没有可导出的数据');
      return;
    }

    // 生成 CSV 内容
    const headers = ['基站编码', '基站名称', '设备组', '初始版本', '升级版本', '升级类型', '产品类型', '保留配置', '升级进度', '结果', '失败原因', '操作人', '操作时间', '开始时间', '结束时间'];
    const rows = dataToExport.map((row) => [
      row.deviceSn,
      row.deviceName,
      row.deviceGroup,
      row.sourceVersion,
      row.targetVersion,
      UPGRADE_TYPE_MAP[row.upgradeType]?.text ?? row.upgradeType,
      row.productType,
      row.keepConfig ? '是' : '否',
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
    link.download = `升级计划_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);

    setExportVisible(false);
    void message.success(`已导出 ${dataToExport.length} 条记录`);
  };

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '基站编码/名称', type: 'input', placeholder: '请输入基站编码或名称' },
    {
      name: 'productType',
      label: '产品类型',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        // eNB 产品类型
        { label: 'PM-B4860', value: 'PM-B4860' },
        { label: 'QAFA', value: 'QAFA' },
        { label: 'QATA', value: 'QATA' },
        { label: 'QAFB', value: 'QAFB' },
        { label: 'RTD', value: 'RTD' },
        // gNB 产品类型
        { label: 'BaiBNX', value: 'BaiBNX' },
        { label: 'BaiBNQ', value: 'BaiBNQ' },
        // GSM 产品类型
        { label: 'BSC', value: 'BSC' },
        { label: 'BTS', value: 'BTS' },
      ],
    },
    {
      name: 'sourceVersion',
      label: '初始版本',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: 'V1.1.5', value: 'V1.1.5' },
        { label: 'V1.2.0', value: 'V1.2.0' },
        { label: 'V2.0.0', value: 'V2.0.0' },
      ],
    },
    {
      name: 'targetVersion',
      label: '升级版本',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: 'V1.3.0', value: 'V1.3.0' },
        { label: 'V2.1.0', value: 'V2.1.0' },
      ],
    },
    {
      name: 'deviceGroup',
      label: '设备组',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '北京移动', value: '北京移动' },
        { label: '上海移动', value: '上海移动' },
        { label: '广东移动', value: '广东移动' },
        { label: '浙江移动', value: '浙江移动' },
        { label: '江苏移动', value: '江苏移动' },
        { label: '四川移动', value: '四川移动' },
        { label: '湖北移动', value: '湖北移动' },
        { label: '陕西移动', value: '陕西移动' },
      ],
    },
    {
      name: 'upgradeType',
      label: '升级类型',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '立即升级', value: 'immediate' },
        { label: '定时升级', value: 'scheduled' },
        { label: '手动升级', value: 'manual' },
      ],
    },
    {
      name: 'result',
      label: '结果',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '成功', value: 'success' },
        { label: '失败', value: 'failed' },
        { label: '部分成功', value: 'partial' },
        { label: '升级中', value: 'running' },
        { label: '等待中', value: 'pending' },
      ],
    },
    {
      name: 'timeRange',
      label: '时间范围',
      type: 'date-range',
      placeholder: '请选择时间范围',
    },
  ], []);

  // 任务列表筛选条件（只有任务名称和时间）
  const taskFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '任务名称', type: 'input', placeholder: '请输入任务名称' },
    {
      name: 'timeRange',
      label: '时间范围',
      type: 'date-range',
      placeholder: '请选择时间范围',
    },
  ], []);

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

  // 设备列表筛选条件
  const deviceFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '基站编码/名称', type: 'input', placeholder: '请输入基站编码或名称' },
    {
      name: 'productType',
      label: '产品类型',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
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
      name: 'deviceGroup',
      label: '设备组',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '北京移动', value: '北京移动' },
        { label: '上海移动', value: '上海移动' },
        { label: '广东移动', value: '广东移动' },
        { label: '浙江移动', value: '浙江移动' },
        { label: '江苏移动', value: '江苏移动' },
        { label: '四川移动', value: '四川移动' },
        { label: '湖北移动', value: '湖北移动' },
        { label: '陕西移动', value: '陕西移动' },
      ],
    },
    {
      name: 'onlineStatus',
      label: '在线状态',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '在线', value: 'online' },
        { label: '离线', value: 'offline' },
      ],
    },
  ], []);

  // 设备列表过滤数据
  const filteredDeviceData = useMemo(() => {
    return mockData.filter((row) => {
      // 关键字搜索
      if (deviceFilters.keyword && typeof deviceFilters.keyword === 'string') {
        const keyword = deviceFilters.keyword.toLowerCase();
        if (!row.deviceSn.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      // 产品类型
      if (deviceFilters.productType && deviceFilters.productType !== 'all') {
        if (row.productType !== deviceFilters.productType) return false;
      }
      // 设备组
      if (deviceFilters.deviceGroup && deviceFilters.deviceGroup !== 'all') {
        if (row.deviceGroup !== deviceFilters.deviceGroup) return false;
      }
      // 在线状态（mock: 根据结果模拟）
      if (deviceFilters.onlineStatus && deviceFilters.onlineStatus !== 'all') {
        // 简单模拟：success/running 为在线，其他为离线
        const isOnline = row.result === 'success' || row.result === 'running';
        if (deviceFilters.onlineStatus === 'online' && !isOnline) return false;
        if (deviceFilters.onlineStatus === 'offline' && isOnline) return false;
      }
      return true;
    });
  }, [deviceFilters]);

  // 设备列表列定义
  const deviceColumns: DataTableColumn<UpgradePlanRow>[] = useMemo(() => [
    { key: 'deviceSn', title: '基站编码', dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
    { key: 'deviceGroup', title: '设备组', dataIndex: 'deviceGroup', width: 100 },
    { key: 'productType', title: '产品类型', dataIndex: 'productType', width: 100 },
    { key: 'sourceVersion', title: '当前版本', dataIndex: 'sourceVersion', width: 100 },
    {
      key: 'onlineStatus',
      title: '在线状态',
      width: 100,
      render: (_: unknown, record: UpgradePlanRow) => {
        const isOnline = record.result === 'success' || record.result === 'running';
        return <Tag color={isOnline ? 'green' : 'default'}>{isOnline ? '在线' : '离线'}</Tag>;
      },
    },
    { key: 'taskName', title: '任务名称', dataIndex: 'taskName', width: 150, ellipsis: true },
  ], []);

  // 直接打开升级抽屉（不需要预选设备）
  const handleOpenUpgradeDrawer = () => {
    setTaskName('');
    setDrawerDevices([]);
    setDrawerProductType('');
    setUpgradeCategory('software');
    setExecutionMethod('immediate');
    setScheduledTime(null);
    setUpgradeFile(undefined);
    setDrawerKeepConfig(true);
    setRetryOffline(true);
    setBatchSize(20);
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
      void message.warning('请选择要添加的设备');
      return;
    }

    const newDevices = availableDevices.filter((d) => selectedNewDevices.includes(d.id));
    setDrawerDevices((prev) => [...prev, ...newDevices]);
    setAddDeviceModalVisible(false);
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    void message.success(`已添加 ${newDevices.length} 台设备`);
  };

  // 提交批量升级
  const handleSubmitUpgrade = () => {
    // 验证任务名称
    if (!taskName.trim()) {
      void message.warning('请输入任务名称');
      return;
    }
    // 验证设备选择
    if (!selectAllOfType && drawerDevices.length === 0) {
      void message.warning('请选择要升级的设备或勾选"升级该产品类型的全部设备"');
      return;
    }
    if (selectAllOfType && allDevicesCountOfType === 0) {
      void message.warning('该产品类型下没有设备');
      return;
    }
    if (!upgradeFile) {
      void message.warning('请选择升级文件');
      return;
    }
    if (executionMethod === 'scheduled' && !scheduledTime) {
      void message.warning('请选择定时执行时间');
      return;
    }

    // 提交升级任务
    const execMethodText = executionMethod === 'immediate' ? '立即执行' :
                          executionMethod === 'suspend' ? '挂起' : `定时执行 (${scheduledTime?.format('YYYY-MM-DD HH:mm')})`;
    const deviceCount = selectAllOfType ? allDevicesCountOfType : drawerDevices.length;
    const deviceInfo = selectAllOfType
      ? `产品类型「${drawerProductType}」全部 ${deviceCount} 台设备`
      : `${deviceCount} 台设备`;

    void message.success(`已创建批量升级任务「${taskName}」：${deviceInfo}，${execMethodText}`);

    // 关闭抽屉并清空选择
    setUpgradeDrawerVisible(false);
    setTaskName('');
    setSelectAllOfType(false);
  };

  // 任务操作处理函数
  const handlePauseTask = (record: UpgradePlanRow) => {
    void message.success(`已暂停任务: ${record.deviceName}`);
  };

  const handleStopTask = (record: UpgradePlanRow) => {
    void message.success(`已终止任务: ${record.deviceName}`);
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
      void message.success(`已删除任务: ${deleteTaskRecord.deviceName}`);
      setDeleteTaskRecord(null);
    }
  };

  // 任务列表列定义
  const taskColumns: DataTableColumn<UpgradePlanRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: '操作',
      width: 80,
      align: 'center',
      fixed: 'left',
      render: (_: unknown, record: UpgradePlanRow) => {
        const isRunning = record.result === 'running';
        const items: MenuProps['items'] = [
          {
            key: 'pause',
            label: '暂停',
            icon: <PauseOutlined />,
            disabled: !isRunning,
            onClick: () => handlePauseTask(record),
          },
          {
            key: 'stop',
            label: '终止',
            icon: <StopOutlined />,
            danger: true,
            disabled: !isRunning,
            onClick: () => handleStopTask(record),
          },
          { type: 'divider' },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => setDeleteTaskRecord(record),
          },
        ];
        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
        );
      },
    },
    {
      key: 'taskName',
      title: '任务名称',
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
    { key: 'operator', title: '操作人', dataIndex: 'operator', width: 100 },
    { key: 'operateTime', title: '操作时间', dataIndex: 'operateTime', width: 160 },
    { key: 'targetVersion', title: '升级版本', dataIndex: 'targetVersion', width: 100 },
    {
      key: 'upgradeType',
      title: '升级类型',
      dataIndex: 'upgradeType',
      width: 100,
      render: (val: UpgradeType) => {
        const cfg = UPGRADE_TYPE_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productType', title: '产品类型', dataIndex: 'productType', width: 100 },
    {
      key: 'progress',
      title: '升级进度',
      dataIndex: 'progress',
      width: 120,
      render: (val: number) => <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />,
    },
    {
      key: 'result',
      title: '结果',
      dataIndex: 'result',
      width: 100,
      render: (val: UpgradeResult) => {
        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'startTime', title: '开始时间', dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: '结束时间', dataIndex: 'endTime', width: 160 },
  ], []);

  const columns: DataTableColumn<UpgradePlanRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: '操作',
      width: 80,
      align: 'center',
      render: (_: unknown, record: UpgradePlanRow) => {
        if (record.result === 'failed') {
          return (
            <Button
              type="link"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => handleRetry(record)}
            >
              重新执行
            </Button>
          );
        }
        return null;
      },
    },
    { key: 'deviceSn', title: '基站编码', dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
    { key: 'deviceGroup', title: '设备组', dataIndex: 'deviceGroup', width: 100 },
    { key: 'sourceVersion', title: '初始版本', dataIndex: 'sourceVersion', width: 100 },
    { key: 'targetVersion', title: '升级版本', dataIndex: 'targetVersion', width: 100 },
    {
      key: 'upgradeType',
      title: '升级类型',
      dataIndex: 'upgradeType',
      width: 100,
      render: (val: UpgradeType) => {
        const cfg = UPGRADE_TYPE_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productType', title: '产品类型', dataIndex: 'productType', width: 100 },
    {
      key: 'keepConfig',
      title: '保留配置',
      dataIndex: 'keepConfig',
      width: 90,
      align: 'center',
      render: (val: boolean) => <Checkbox checked={val} />,
    },
    {
      key: 'progress',
      title: '升级进度',
      dataIndex: 'progress',
      width: 120,
      render: (val: number) => <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />,
    },
    {
      key: 'result',
      title: '结果',
      dataIndex: 'result',
      width: 100,
      render: (val: UpgradeResult) => {
        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'failureReason', title: '失败原因', dataIndex: 'failureReason', width: 150, ellipsis: true },
    { key: 'operator', title: '操作人', dataIndex: 'operator', width: 100 },
    { key: 'operateTime', title: '操作时间', dataIndex: 'operateTime', width: 160 },
    { key: 'startTime', title: '开始时间', dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: '结束时间', dataIndex: 'endTime', width: 160 },
  ], []);

  // 页面头部按钮
  const headerExtra = useMemo(() => (
    <Space>
      <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleOpenUpgradeDrawer}>
        升级
      </Button>
      <Button icon={<DownloadOutlined />} onClick={handleOpenExport}>
        导出
      </Button>
    </Space>
  ), []);

  return (
    <ListPageLayout title={t('nav.software.upgradePlan')} extra={headerExtra}>
      {/* 覆盖 FilterBar 样式 */}
      <style>{`
        .upgrade-plan-filter-wrapper [class*="_filterBarWrapper_"] {
          padding: 0 !important;
          margin-bottom: 0 !important;
        }
      `}</style>

      {/* 页签选择 */}
      <Card bordered={false} style={{ marginBottom: 16 }}>
        <Radio.Group
          value={activeTab}
          onChange={(e) => {
            setActiveTab(e.target.value);
            setFilters({});
            setPage(1);
          }}
          optionType="button"
          buttonStyle="solid"
        >
          <Radio.Button value="task">任务列表</Radio.Button>
          <Radio.Button value="device">设备列表</Radio.Button>
        </Radio.Group>
      </Card>

      {/* 搜索表单 */}
      <Card bordered={false} style={{ marginBottom: 16 }} className="upgrade-plan-filter-wrapper">
        <FilterBar
          filterId={`upgrade-plan-filter-${activeTab}`}
          fields={activeTab === 'task' ? taskFilterFields : filterFields}
          onSearch={(vals) => { setFilters(vals); setPage(1); }}
          onReset={() => { setFilters({}); setPage(1); }}
        />
      </Card>

      {/* 列表 */}
      <Card bordered={false}>
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
            scroll={{ x: 1600 }}
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
            scroll={{ x: 2000 }}
          />
        )}
      </Card>

      {/* 批量输入弹窗 */}
      <Modal
        title={`批量输入设备SN - ${drawerProductType || '请先选择产品类型'}`}
        open={batchInputVisible}
        onCancel={() => setBatchInputVisible(false)}
        onOk={handleBatchInputConfirm}
        okText="确认添加"
        cancelText="取消"
        width={600}
        okButtonProps={{
          disabled: batchInputPreview.matched.length === 0 || !drawerProductType,
        }}
      >
        {!drawerProductType && (
          <Alert
            type="warning"
            showIcon
            message="请先选择产品类型"
            style={{ marginBottom: 16 }}
          />
        )}

        <div style={{ marginBottom: 16 }}>
          <Input.TextArea
            placeholder="请输入设备SN，支持换行、逗号、分号、空格分隔&#10;例如：&#10;ENB00001&#10;ENB00002, ENB00003; GNB00001"
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
                    匹配到 <strong>{batchInputPreview.matched.length}</strong> 个设备
                    {batchInputPreview.mixedTypes.includes(drawerProductType) && (
                      <Tag color="blue" style={{ marginLeft: 8 }}>可添加: {batchInputPreview.matched.filter(d => d.productType === drawerProductType).length} 台</Tag>
                    )}
                  </span>
                  {!batchInputPreview.mixedTypes.includes(drawerProductType) && drawerProductType && (
                    <span style={{ color: '#faad14' }}>
                      <WarningOutlined style={{ marginRight: 4 }} />
                      没有产品类型为「{drawerProductType}」的设备，请重新输入
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
                <div>以下 {batchInputPreview.notFound.length} 个设备SN未找到:</div>
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

      {/* 导出弹窗 */}
      <Modal
        title="导出升级计划"
        open={exportVisible}
        onCancel={() => setExportVisible(false)}
        onOk={handleExport}
        okText="导出"
        cancelText="取消"
        width={500}
        okButtonProps={{
          disabled: exportType === 'range' && (!exportTimeRange || !exportTimeRange[0] || !exportTimeRange[1]),
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Radio.Group
            value={exportType}
            onChange={(e) => setExportType(e.target.value)}
          >
            <Radio value="all">导出全部</Radio>
            <Radio value="range">按时间导出</Radio>
          </Radio.Group>
        </div>

        {exportType === 'range' && (
          <div style={{ marginBottom: 16 }}>
            <DatePicker.RangePicker
              style={{ width: '100%' }}
              value={exportTimeRange}
              onChange={(dates) => setExportTimeRange(dates)}
              placeholder={['开始时间', '结束时间']}
            />
          </div>
        )}

        <Alert
          type="info"
          showIcon
          message={`将导出 ${exportType === 'all' ? filteredData.length : '符合时间范围的'} 条记录，格式为 CSV`}
        />
      </Modal>

      {/* 重新执行确认弹窗 */}
      <Modal
        title="确认重新执行"
        open={!!retryRecord}
        onCancel={() => setRetryRecord(null)}
        onOk={handleRetryConfirm}
        okText="确认"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                确定要重新执行以下设备的升级任务吗？
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>基站编码：</strong>{retryRecord?.deviceSn}<br />
                <strong>基站名称：</strong>{retryRecord?.deviceName}<br />
                <strong>目标版本：</strong>{retryRecord?.targetVersion}
              </p>
            </div>
          }
        />
      </Modal>

      {/* 删除任务确认弹窗 */}
      <Modal
        title="确认删除"
        open={!!deleteTaskRecord}
        onCancel={() => setDeleteTaskRecord(null)}
        onOk={handleDeleteTaskConfirm}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                确定要删除以下升级任务吗？此操作不可恢复。
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>任务名称：</strong>{deleteTaskRecord?.deviceName}<br />
                <strong>操作人：</strong>{deleteTaskRecord?.operator}<br />
                <strong>升级版本：</strong>{deleteTaskRecord?.targetVersion}
              </p>
            </div>
          }
        />
      </Modal>

      {/* 批量升级抽屉 */}
      <Drawer
        title="批量升级"
        placement="right"
        width={600}
        open={upgradeDrawerVisible}
        onClose={() => setUpgradeDrawerVisible(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setUpgradeDrawerVisible(false)}>取消</Button>
            <Button
              type="primary"
              onClick={handleSubmitUpgrade}
              disabled={(!selectAllOfType && drawerDevices.length === 0) || !upgradeFile}
            >
              确认升级
            </Button>
          </Space>
        }
      >
        <Form layout="vertical" size="small">
          {/* 任务名称 */}
          <Form.Item label="任务名称" required>
            <Input
              value={taskName}
              onChange={(e) => setTaskName(e.target.value)}
              placeholder="请输入任务名称"
              maxLength={100}
              showCount
            />
          </Form.Item>

          {/* 已选产品类型 */}
          <Form.Item label="产品类型" required>
            <Select
              value={drawerProductType}
              onChange={(val) => {
                setDrawerProductType(val);
                setUpgradeFile(undefined);
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
              升级该产品类型的全部设备
              {drawerProductType && (
                <Tag color="blue" style={{ marginLeft: 8 }}>共 {allDevicesCountOfType} 台</Tag>
              )}
            </Checkbox>
          </Form.Item>

          {/* 升级类型 */}
          <Form.Item label="升级类型" required>
            <Radio.Group
              value={upgradeCategory}
              onChange={(e) => {
                setUpgradeCategory(e.target.value);
                setUpgradeFile(undefined);
              }}
            >
              <Radio value="software">软件升级</Radio>
              <Radio value="patch">PATCH升级</Radio>
              <Radio value="fpga">FPGA升级</Radio>
            </Radio.Group>
          </Form.Item>

          {/* 已选升级设备 */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
              <span>
                已选升级设备
                {' '}
                <Tag color="blue">{selectAllOfType ? allDevicesCountOfType : drawerDevices.length} 台</Tag>
              </span>
              {!selectAllOfType && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleOpenBatchInput}
                  disabled={!drawerProductType}
                >
                  批量输入
                </Button>
              )}
            </div>
          }>
            {selectAllOfType ? (
              <Alert
                type="info"
                showIcon
                message={`已选择产品类型「${drawerProductType}」的全部设备，共 ${allDevicesCountOfType} 台`}
                description="执行时将自动查询该产品类型的所有设备进行升级"
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
                      { title: '基站编码', dataIndex: 'deviceSn', width: 100 },
                      { title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
                      { title: '当前版本', dataIndex: 'sourceVersion', width: 80 },
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
                    添加设备
                  </Button>
                </div>
              </>
            )}
          </Form.Item>

          <Divider />

          {/* 升级文件 */}
          <Form.Item label="升级文件" required>
            <Select
              value={upgradeFile}
              onChange={setUpgradeFile}
              placeholder="请选择升级文件"
              style={{ width: '100%' }}
              options={filteredFiles.map((f) => ({ label: f.label, value: f.value }))}
            />
          </Form.Item>

          {/* 是否保留配置 */}
          <Form.Item>
            <Checkbox
              checked={drawerKeepConfig}
              onChange={(e) => setDrawerKeepConfig(e.target.checked)}
            >
              保留配置
            </Checkbox>
          </Form.Item>

          <Divider />

          {/* 执行方式 */}
          <Form.Item label="执行方式" required>
            <Radio.Group value={executionMethod} onChange={(e) => setExecutionMethod(e.target.value)}>
              <Radio value="immediate">立即执行</Radio>
              <Radio value="suspend">挂起</Radio>
              <Radio value="scheduled">定时执行</Radio>
            </Radio.Group>
          </Form.Item>

          {/* 定时执行时间 */}
          {executionMethod === 'scheduled' && (
            <Form.Item label="执行时间" required>
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm"
                value={scheduledTime}
                onChange={setScheduledTime}
                placeholder="请选择执行时间"
                style={{ width: '100%' }}
                disabledDate={(current) => current && current < new Date()}
              />
            </Form.Item>
          )}

          <Divider />

          {/* 任务配置 */}
          <Form.Item label="任务配置">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Checkbox
                checked={retryOffline}
                onChange={(e) => setRetryOffline(e.target.checked)}
              >
                离线设备等上线后重试
              </Checkbox>
              <Space>
                <span>每次批量执行设备数：</span>
                <InputNumber
                  min={1}
                  max={100}
                  value={batchSize}
                  onChange={(val) => setBatchSize(val ?? 20)}
                  style={{ width: 80 }}
                />
                <span>台</span>
              </Space>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>

      {/* 任务详情抽屉 */}
      <Drawer
        title="任务详情"
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
                <Descriptions.Item label="任务名称" span={2}>{taskName}</Descriptions.Item>
                <Descriptions.Item label="产品类型">{taskDetailRecord.productType}</Descriptions.Item>
                <Descriptions.Item label="升级类型">
                  <Tag color={UPGRADE_TYPE_MAP[taskDetailRecord.upgradeType]?.color}>
                    {UPGRADE_TYPE_MAP[taskDetailRecord.upgradeType]?.text || taskDetailRecord.upgradeType}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="目标版本">{taskDetailRecord.targetVersion}</Descriptions.Item>
                <Descriptions.Item label="保留配置">
                  <Checkbox checked={taskDetailRecord.keepConfig} disabled />
                </Descriptions.Item>
                <Descriptions.Item label="操作人">{taskDetailRecord.operator}</Descriptions.Item>
                <Descriptions.Item label="操作时间">{taskDetailRecord.operateTime || '-'}</Descriptions.Item>
              </Descriptions>

              {/* 执行进度概览 */}
              <Card title="执行进度概览" size="small" style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                  <div style={{ flex: '0 0 120px', textAlign: 'center' }}>
                    <Progress
                      type="circle"
                      percent={avgProgress}
                      size={80}
                      status={avgProgress === 100 ? 'success' : 'active'}
                    />
                    <div style={{ marginTop: 8, color: '#666' }}>总体进度</div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <div>
                        <Tag color="success">成功</Tag>
                        <span>{resultStats.success} 台</span>
                      </div>
                      <div>
                        <Tag color="error">失败</Tag>
                        <span>{resultStats.failed} 台</span>
                      </div>
                      <div>
                        <Tag color="processing">升级中</Tag>
                        <span>{resultStats.running} 台</span>
                      </div>
                      <div>
                        <Tag color="warning">部分成功</Tag>
                        <span>{resultStats.partial} 台</span>
                      </div>
                      <div>
                        <Tag color="default">等待中</Tag>
                        <span>{resultStats.pending} 台</span>
                      </div>
                    </Space>
                  </div>
                  <Divider type="vertical" style={{ height: 120 }} />
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 32, fontWeight: 'bold', color: '#1890ff' }}>{totalDevices}</div>
                    <div style={{ color: '#666' }}>设备总数</div>
                  </div>
                </div>
              </Card>

              {/* 设备列表 */}
              <Card title={`设备列表 (${totalDevices} 台)`} size="small">
                <Table
                  size="small"
                  dataSource={taskDevices}
                  rowKey="id"
                  pagination={totalDevices > 10 ? { pageSize: 10 } : false}
                  scroll={{ y: 300 }}
                  columns={[
                    {
                      title: '基站编码',
                      dataIndex: 'deviceSn',
                      width: 100,
                    },
                    {
                      title: '基站名称',
                      dataIndex: 'deviceName',
                      ellipsis: true,
                    },
                    {
                      title: '当前版本',
                      dataIndex: 'sourceVersion',
                      width: 80,
                    },
                    {
                      title: '进度',
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
                      title: '结果',
                      dataIndex: 'result',
                      width: 80,
                      render: (val: UpgradeResult) => {
                        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
                        return <Tag color={cfg.color}>{cfg.text}</Tag>;
                      },
                    },
                    {
                      title: '失败原因',
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
        title={`添加设备 - ${drawerProductType}`}
        open={addDeviceModalVisible}
        onCancel={() => setAddDeviceModalVisible(false)}
        onOk={handleConfirmAddDevices}
        okText="确认添加"
        cancelText="取消"
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message="没有可添加的设备"
            description={`产品类型为 ${drawerProductType} 的设备已全部在列表中`}
          />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={`共 ${availableDevices.length} 台设备可选，当前显示 ${filteredAvailableDevices.length} 台，已选择 ${selectedNewDevices.length} 台`}
              style={{ marginBottom: 16 }}
            />

            {/* 搜索和全选 */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input.Search
                placeholder="搜索基站编码或名称"
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
                全选 ({filteredAvailableDevices.length} 台)
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
                { title: '基站编码', dataIndex: 'deviceSn', width: 120 },
                { title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
                { title: '设备组', dataIndex: 'deviceGroup', width: 100 },
                { title: '当前版本', dataIndex: 'sourceVersion', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>
    </ListPageLayout>
  );
}
