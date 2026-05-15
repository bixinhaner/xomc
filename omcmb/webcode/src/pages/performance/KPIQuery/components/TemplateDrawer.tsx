import { useState, useMemo, useEffect } from 'react';
import {
  Drawer,
  Form,
  Input,
  Radio,
  Checkbox,
  Button,
  Space,
  Card,
  Select,
  App,
  Row,
  Col,
  Typography,
  Tag,
  Pagination,
  Modal,
  Table,
  Divider,
  Alert,
} from 'antd';
import {
  SearchOutlined,
  FolderOutlined,
  ClearOutlined,
  DeleteOutlined,
  PlusOutlined,
  EditOutlined,
} from '@ant-design/icons';
import type { DataNode, TreeProps } from 'antd/es/tree';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

const { Text } = Typography;
const { TextArea } = Input;

type DrawerMode = 'add' | 'edit' | 'view' | 'copy';

interface TemplateDrawerProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: TemplateFormValues) => void;
  loading?: boolean;
  mode?: DrawerMode;
  initialValues?: Partial<TemplateFormValues>;
  networkType?: 'enb' | 'gnb' | 'egw';
}

export interface TemplateFormValues {
  // 基本信息
  tempName: string;
  isPublic: '0' | '1';
  description: string;
  creator?: string;
  updator?: string;
  // 周期设定
  reportPeriod: '15' | '60' | '1440';
  checkAll: '0' | '1';
  week: string[];
  hour: string[];
  busyTime: '' | '6' | '8';
  // 设备选择
  selDeviceType: '1' | '2';
  selectedDevices: string[];
  selectedGroups: string[];
  // 指标选择
  indicatorLevel?: 'device' | 'plmn';
  selectedKpis: string[];
  // 设为默认
  isDefault: boolean;
}

// 小时选择选项
const HOUR_OPTIONS = Array.from({ length: 24 }, (_, i) => ({
  label: `${i}:00`,
  value: String(i),
}));

// 忙时预设
const BUSY_HOUR_PRESETS: Record<string, string[]> = {
  '6': ['8', '9', '10', '18', '19', '20'],
  '8': ['8', '9', '10', '11', '18', '19', '20', '21'],
};

// 默认表单值
const DEFAULT_FORM_VALUES: TemplateFormValues = {
  tempName: '',
  isPublic: '0',
  description: '',
  reportPeriod: '15',
  checkAll: '1',
  week: [],
  hour: [],
  busyTime: '',
  selDeviceType: '2',
  selectedDevices: [],
  selectedGroups: [],
  indicatorLevel: 'device',
  selectedKpis: [],
  isDefault: false,
};

// Mock 设备组数据 (示例数据，实际使用时从 API 获取)
const MOCK_DEVICE_GROUPS = [
  { id: 'group-1', name: '北京区域', deviceCount: 15 },
  { id: 'group-1-1', name: '北京朝阳', deviceCount: 8, parentId: 'group-1' },
  { id: 'group-1-2', name: '北京海淀', deviceCount: 7, parentId: 'group-1' },
  { id: 'group-2', name: '上海区域', deviceCount: 12 },
  { id: 'group-2-1', name: '上海浦东', deviceCount: 6, parentId: 'group-2' },
  { id: 'group-2-2', name: '上海徐汇', deviceCount: 6, parentId: 'group-2' },
  { id: 'group-3', name: '广州区域', deviceCount: 10 },
  { id: 'group-4', name: '深圳区域', deviceCount: 8 },
];

// Mock 设备数据 (示例数据)
const MOCK_DEVICES = [
  { id: 'ENB00001', name: '北京朝阳基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-1-1' },
  { id: 'ENB00002', name: '北京海淀基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-1-2' },
  { id: 'ENB00003', name: '北京东城基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-1' },
  { id: 'ENB00004', name: '北京西城基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-1' },
  { id: 'ENB00005', name: '上海浦东基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-2-1' },
  { id: 'ENB00006', name: '上海徐汇基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-2-2' },
  { id: 'GNB00001', name: '北京5G基站01', neType: 'gNB', productType: 'NR', groupId: 'group-1' },
  { id: 'GNB00002', name: '上海5G基站01', neType: 'gNB', productType: 'NR', groupId: 'group-2' },
  { id: 'GNB00003', name: '广州5G基站01', neType: 'gNB', productType: 'NR', groupId: 'group-3' },
  { id: 'ENB00007', name: '深圳南山基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-4' },
  { id: 'ENB00008', name: '深圳福田基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-4' },
  { id: 'ENB00009', name: '广州天河基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-3' },
  { id: 'ENB00010', name: '广州越秀基站01', neType: 'eNB', productType: 'LTE', groupId: 'group-3' },
  { id: 'GNB00004', name: '深圳5G基站01', neType: 'gNB', productType: 'NR', groupId: 'group-4' },
  { id: 'ENB00011', name: '北京朝阳基站02', neType: 'eNB', productType: 'LTE', groupId: 'group-1-1' },
];

// Mock KPI 数据 (示例数据)
const MOCK_KPIS = [
  // 接入类指标
  { id: 'kpi-rrc-sr', name: 'RRC建立成功率', primaryCategory: 'pc-1', secondaryCategory: 'sc-1-1', unit: '%' },
  { id: 'kpi-erab-sr', name: 'ERAB建立成功率', primaryCategory: 'pc-1', secondaryCategory: 'sc-1-1', unit: '%' },
  { id: 'kpi-radio-sr', name: '无线接通率', primaryCategory: 'pc-1', secondaryCategory: 'sc-1-2', unit: '%' },
  { id: 'kpi-srv-sr', name: '业务建立成功率', primaryCategory: 'pc-1', secondaryCategory: 'sc-1-2', unit: '%' },
  // 吞吐量指标
  { id: 'kpi-dl-thp', name: '下行吞吐量', primaryCategory: 'pc-2', secondaryCategory: 'sc-2-1', unit: 'Mbps' },
  { id: 'kpi-ul-thp', name: '上行吞吐量', primaryCategory: 'pc-2', secondaryCategory: 'sc-2-1', unit: 'Mbps' },
  { id: 'kpi-dl-prb', name: '下行PRB利用率', primaryCategory: 'pc-2', secondaryCategory: 'sc-2-2', unit: '%' },
  { id: 'kpi-ul-prb', name: '上行PRB利用率', primaryCategory: 'pc-2', secondaryCategory: 'sc-2-2', unit: '%' },
  // 移动性指标
  { id: 'kpi-ho-sr', name: '切换成功率', primaryCategory: 'pc-3', secondaryCategory: 'sc-3-1', unit: '%' },
  { id: 'kpi-ho-fail', name: '切换失败次数', primaryCategory: 'pc-3', secondaryCategory: 'sc-3-1', unit: '次' },
  { id: 'kpi-ho-in', name: '切入成功率', primaryCategory: 'pc-3', secondaryCategory: 'sc-3-2', unit: '%' },
  { id: 'kpi-ho-out', name: '切出成功率', primaryCategory: 'pc-3', secondaryCategory: 'sc-3-2', unit: '%' },
  // 用户数指标
  { id: 'kpi-max-user', name: '最大用户数', primaryCategory: 'pc-4', secondaryCategory: 'sc-4-1', unit: '个' },
  { id: 'kpi-active-user', name: '在线用户数', primaryCategory: 'pc-4', secondaryCategory: 'sc-4-1', unit: '个' },
  { id: 'kpi-idle-user', name: '空闲用户数', primaryCategory: 'pc-4', secondaryCategory: 'sc-4-2', unit: '个' },
  { id: 'kpi-avg-user', name: '平均用户数', primaryCategory: 'pc-4', secondaryCategory: 'sc-4-2', unit: '个' },
];

// 一级指标集
const PRIMARY_CATEGORIES = [
  { id: 'pc-1', name: '接入类指标' },
  { id: 'pc-2', name: '吞吐量指标' },
  { id: 'pc-3', name: '移动性指标' },
  { id: 'pc-4', name: '用户数指标' },
];

// 二级指标集（按一级分类分组）
const SECONDARY_CATEGORIES: Record<string, { id: string; name: string }[]> = {
  'pc-1': [
    { id: 'sc-1-1', name: '信令建立' },
    { id: 'sc-1-2', name: '业务建立' },
  ],
  'pc-2': [
    { id: 'sc-2-1', name: '吞吐量' },
    { id: 'sc-2-2', name: 'PRB利用率' },
  ],
  'pc-3': [
    { id: 'sc-3-1', name: '切换统计' },
    { id: 'sc-3-2', name: '切入切出' },
  ],
  'pc-4': [
    { id: 'sc-4-1', name: '用户统计' },
    { id: 'sc-4-2', name: '用户分布' },
  ],
};

export default function TemplateDrawer({
  open,
  onClose,
  onSubmit,
  loading = false,
  mode = 'add',
  initialValues,
  networkType = 'enb',
}: TemplateDrawerProps) {
  const t = useT();
  const token = useThemeToken();
  const { message } = App.useApp();
  const [form] = Form.useForm();

  // 周选择选项 - 使用 i18n
  const WEEK_OPTIONS = [
    { label: t('perf.query.week.sun'), value: '7' },
    { label: t('perf.query.week.mon'), value: '1' },
    { label: t('perf.query.week.tue'), value: '2' },
    { label: t('perf.query.week.wed'), value: '3' },
    { label: t('perf.query.week.thu'), value: '4' },
    { label: t('perf.query.week.fri'), value: '5' },
    { label: t('perf.query.week.sat'), value: '6' },
  ];

  // Mock KPI 分类数据 - 使用 i18n
  const MOCK_KPI_CATEGORIES: DataNode[] = useMemo(() => [
    {
      title: t('perf.query.accessKpi'),
      key: 'cat-1',
      children: [
        { title: t('kpi.rrcSetupSuccessRate'), key: 'kpi-rrc-sr' },
        { title: t('kpi.erabSetupSuccessRate'), key: 'kpi-erab-sr' },
        { title: t('kpi.accessRate'), key: 'kpi-radio-sr' },
      ],
    },
    {
      title: t('perf.query.throughputKpi'),
      key: 'cat-2',
      children: [
        { title: t('kpi.dlThroughput'), key: 'kpi-dl-thp' },
        { title: t('kpi.ulThroughput'), key: 'kpi-ul-thp' },
        { title: t('kpi.prbUtilization'), key: 'kpi-dl-prb' },
        { title: t('kpi.prbUtilization'), key: 'kpi-ul-prb' },
      ],
    },
    {
      title: t('perf.query.handoverKpi'),
      key: 'cat-3',
      children: [
        { title: t('kpi.handoverSuccessRate'), key: 'kpi-ho-sr' },
        { title: t('kpi.handoverFailures'), key: 'kpi-ho-fail' },
      ],
    },
    {
      title: t('perf.query.userKpi'),
      key: 'cat-4',
      children: [
        { title: t('kpi.maxUsers'), key: 'kpi-max-user' },
        { title: t('kpi.onlineUsers'), key: 'kpi-active-user' },
        { title: t('kpi.idleUsers'), key: 'kpi-idle-user' },
      ],
    },
  ], [t]);

  // 表单状态
  const [isPublic, setIsPublic] = useState<'0' | '1'>('0');
  const [reportPeriod, setReportPeriod] = useState<'15' | '60' | '1440'>('15');
  const [checkAll, setCheckAll] = useState<'0' | '1'>('1');
  const [week, setWeek] = useState<string[]>([]);
  const [hour, setHour] = useState<string[]>([]);
  const [busyTime, setBusyTime] = useState<'' | '6' | '8'>('');
  const [selDeviceType, setSelDeviceType] = useState<'1' | '2'>('2');
  const [indicatorLevel, setIndicatorLevel] = useState<'device' | 'plmn'>('device');
  const [selectedKpis, setSelectedKpis] = useState<string[]>([]);
  const [isDefault, setIsDefault] = useState(false);

  // 设备选择状态
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  const [selectedGroups, setSelectedGroups] = useState<string[]>([]);
  const [deviceSearchText, _setDeviceSearchText] = useState('');
  const [deviceProductType, _setDeviceProductType] = useState<string>('all');
  const [groupDeviceType, _setGroupDeviceType] = useState<string>('all');
  const [deviceCurrentPage, _setDeviceCurrentPage] = useState(1);
  const [devicePageSize, _setDevicePageSize] = useState(10);
  const [addDeviceModalVisible, setAddDeviceModalVisible] = useState(false);
  const [modalSearchText, setModalSearchText] = useState('');
  const [modalProductType, setModalProductType] = useState<string>('all');
  const [modalDeviceType, setModalDeviceType] = useState<string>('all');
  const [modalCurrentPage, setModalCurrentPage] = useState(1);
  const [modalPageSize, setModalPageSize] = useState(10);
  const [tempSelectedDevices, setTempSelectedDevices] = useState<string[]>([]);
  const [tempSelectedGroups, setTempSelectedGroups] = useState<string[]>([]);

  // 批量输入状态
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');

  // KPI 选择状态
  const [kpiSearchText, _setKpiSearchText] = useState('');
  const [_selectedCategory, _setSelectedCategory] = useState<string[]>([]);
  const [addKpiModalVisible, setAddKpiModalVisible] = useState(false);
  const [kpiModalSearchText, setKpiModalSearchText] = useState('');
  const [kpiModalPrimaryCategory, setKpiModalPrimaryCategory] = useState<string>('all');
  const [kpiModalSecondaryCategory, setKpiModalSecondaryCategory] = useState<string>('all');
  const [kpiCurrentPage, setKpiCurrentPage] = useState(1);
  const [kpiPageSize, setKpiPageSize] = useState(10);
  const [tempSelectedKpis, setTempSelectedKpis] = useState<string[]>([]);

  // KPI 批量输入状态
  const [kpiBatchInputVisible, setKpiBatchInputVisible] = useState(false);
  const [kpiBatchInputValue, setKpiBatchInputValue] = useState('');

  // 是否只读模式
  const isReadOnly = mode === 'view';

  // 初始化表单状态
  useEffect(() => {
    if (open && initialValues) {
      if (initialValues.isPublic) {
        setIsPublic(initialValues.isPublic);
      }
    } else if (open) {
      // 新增模式，重置为默认值
      setIsPublic('0');
    }
  }, [open, initialValues]);

  // 关闭抽屉
  const handleClose = () => {
    onClose();
  };

  // 提交表单
  const handleSubmit = () => {
    if (isReadOnly) {
      handleClose();
      return;
    }

    form.validateFields().then((values) => {
      const submitData: TemplateFormValues = {
        ...values,
        isPublic,
        reportPeriod,
        checkAll,
        week: checkAll === '1' ? [] : week,
        hour: checkAll === '1' ? [] : hour,
        busyTime,
        selDeviceType,
        indicatorLevel,
        selectedKpis,
        isDefault,
      };

      // 验证必填项
      if (selectedKpis.length === 0) {
        message.warning(t('perf.query.pleaseSelectKpi'));
        return;
      }

      onSubmit(submitData);
    });
  };

  // 查询粒度变化
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleReportPeriodChange = (e: any) => {
    const value = e.target.value as '15' | '60' | '1440';
    setReportPeriod(value);
    if (value === '1440') {
      setCheckAll('1');
      setWeek([]);
      setHour([]);
      setBusyTime('');
    }
  };

  // 时段模式变化
  const handleCheckAllChange = (e: { target: { checked?: boolean } }) => {
    const value = e.target.checked ? '1' : '0';
    setCheckAll(value);
    if (value === '1') {
      setWeek([]);
      setHour([]);
      setBusyTime('');
    }
  };

  // 忙时选择变化
  const handleBusyTimeChange = (value: '' | '6' | '8') => {
    setBusyTime(value);
    if (value) {
      setHour(BUSY_HOUR_PRESETS[value]);
      setWeek(['1', '2', '3', '4', '5', '6', '7']);
    }
  };

  // 周选择变化
  const handleWeekChange = (checkedValues: string[]) => {
    setWeek(checkedValues);
    setBusyTime(''); // 手动选择时清除忙时预设
  };

  // 小时选择变化
  const handleHourChange = (checkedValues: string[]) => {
    setHour(checkedValues);
    setBusyTime(''); // 手动选择时清除忙时预设
  };

  // KPI 树选择
  const _handleKpiTreeCheck: TreeProps['onCheck'] = (checkedKeys) => {
    const keys = Array.isArray(checkedKeys) ? checkedKeys : checkedKeys.checked;
    setSelectedKpis(keys as string[]);
  };
  void _handleKpiTreeCheck;

  // 打开批量输入弹窗
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputVisible(true);
  };

  // 解析批量输入的设备ID
  const parseBatchInput = (input: string): string[] => {
    // 支持换行、逗号、分号、空格分隔
    return input
      .split(/[\n,;，；\s]+/)
      .map(s => s.trim())
      .filter(s => s.length > 0);
  };

  // 批量输入确认
  const handleBatchInputConfirm = () => {
    const inputIds = parseBatchInput(batchInputValue);
    if (inputIds.length === 0) {
      message.warning(t('perf.query.pleaseInputDeviceId'));
      return;
    }

    if (selDeviceType === '2') {
      // 按设备添加：过滤出存在的设备ID
      const existingDeviceIds = new Set(MOCK_DEVICES.map(d => d.id));
      const validIds = inputIds.filter(id => existingDeviceIds.has(id));
      const invalidIds = inputIds.filter(id => !existingDeviceIds.has(id));

      if (invalidIds.length > 0) {
        message.warning(t('perf.query.batchInputNotFound', { count: invalidIds.length, ids: invalidIds.slice(0, 5).join(', ') }));
      }

      if (validIds.length > 0) {
        setSelectedDevices(prev => [...new Set([...prev, ...validIds])]);
        message.success(t('perf.query.batchInputSuccess', { count: validIds.length }));
      }
    } else {
      // 按设备组添加：过滤出存在的设备组ID
      const existingGroupIds = new Set(MOCK_DEVICE_GROUPS.map(g => g.id));
      const validIds = inputIds.filter(id => existingGroupIds.has(id));
      const invalidIds = inputIds.filter(id => !existingGroupIds.has(id));

      if (invalidIds.length > 0) {
        message.warning(t('perf.query.batchInputNotFound', { count: invalidIds.length, ids: invalidIds.slice(0, 5).join(', ') }));
      }

      if (validIds.length > 0) {
        setSelectedGroups(prev => [...new Set([...prev, ...validIds])]);
        message.success(t('perf.query.batchInputSuccess', { count: validIds.length }));
      }
    }

    setBatchInputVisible(false);
    setBatchInputValue('');
  };

  // 打开 KPI 批量输入弹窗
  const handleOpenKpiBatchInput = () => {
    setKpiBatchInputValue('');
    setKpiBatchInputVisible(true);
  };

  // KPI 批量输入确认
  const handleKpiBatchInputConfirm = () => {
    const inputIds = parseBatchInput(kpiBatchInputValue);
    if (inputIds.length === 0) {
      message.warning(t('perf.query.pleaseInputKpiId'));
      return;
    }

    // 过滤出存在的 KPI ID
    const existingKpiIds = new Set(MOCK_KPIS.map(k => k.id));
    const validIds = inputIds.filter(id => existingKpiIds.has(id));
    const invalidIds = inputIds.filter(id => !existingKpiIds.has(id));

    if (invalidIds.length > 0) {
      message.warning(t('perf.query.batchInputNotFound', { count: invalidIds.length, ids: invalidIds.slice(0, 5).join(', ') }));
    }

    if (validIds.length > 0) {
      setSelectedKpis(prev => [...new Set([...prev, ...validIds])]);
      message.success(t('perf.query.batchInputSuccess', { count: validIds.length }));
    }

    setKpiBatchInputVisible(false);
    setKpiBatchInputValue('');
  };

  // 获取抽屉标题
  const getDrawerTitle = () => {
    switch (mode) {
      case 'add':
        return t('perf.query.addTemplate');
      case 'edit':
        return t('perf.query.editTemplate');
      case 'view':
        return t('perf.query.viewTemplate');
      case 'copy':
        return t('perf.query.copyTemplate');
      default:
        return t('perf.query.addTemplate');
    }
  };

  // 是否显示时段配置
  const showTimeConfig = reportPeriod !== '1440';

  // 时段配置是否禁用
  const timeConfigDisabled = checkAll === '1' || reportPeriod === '1440';

  // 过滤后的 KPI 列表
  const _filteredKpiCategories = useMemo(() => {
    if (!kpiSearchText) return MOCK_KPI_CATEGORIES;

    const filterTree = (nodes: DataNode[]): DataNode[] => {
      return nodes.reduce((acc: DataNode[], node) => {
        const title = String(node.title);
        if (title.toLowerCase().includes(kpiSearchText.toLowerCase())) {
          acc.push(node);
        } else if (node.children) {
          const filteredChildren = filterTree(node.children);
          if (filteredChildren.length > 0) {
            acc.push({ ...node, children: filteredChildren });
          }
        }
        return acc;
      }, []);
    };

    return filterTree(MOCK_KPI_CATEGORIES);
  }, [kpiSearchText, MOCK_KPI_CATEGORIES]);
  void _filteredKpiCategories;

  // 过滤后的设备列表
  const filteredDevices = useMemo(() => {
    let result = MOCK_DEVICES;

    // 按产品类型筛选
    if (deviceProductType !== 'all') {
      result = result.filter(device => device.productType === deviceProductType);
    }

    // 按搜索文本筛选
    if (deviceSearchText) {
      const searchLower = deviceSearchText.toLowerCase();
      result = result.filter(
        device =>
          device.id.toLowerCase().includes(searchLower) ||
          device.name.toLowerCase().includes(searchLower) ||
          device.neType.toLowerCase().includes(searchLower)
      );
    }

    return result;
  }, [deviceSearchText, deviceProductType]);

  // 过滤后的设备组列表
  const filteredGroups = useMemo(() => {
    let result = MOCK_DEVICE_GROUPS;

    // 按设备类型筛选（筛选包含指定类型设备的组）
    if (groupDeviceType !== 'all') {
      const neTypeMap: Record<string, string> = {
        enb: 'eNB',
        gnb: 'gNB',
        gsm: 'GSM',
      };
      const targetNeType = neTypeMap[groupDeviceType];
      // 获取包含指定类型设备的组ID
      const groupIdsWithDeviceType = new Set(
        MOCK_DEVICES
          .filter(device => device.neType === targetNeType)
          .map(device => device.groupId)
      );
      result = result.filter(group => groupIdsWithDeviceType.has(group.id));
    }

    // 按搜索文本筛选
    if (deviceSearchText) {
      const searchLower = deviceSearchText.toLowerCase();
      result = result.filter(
        group =>
          group.id.toLowerCase().includes(searchLower) ||
          group.name.toLowerCase().includes(searchLower)
      );
    }

    return result;
  }, [deviceSearchText, groupDeviceType]);

  // 分页后的设备列表
  const _paginatedDevices = useMemo(() => {
    const start = (deviceCurrentPage - 1) * devicePageSize;
    const end = start + devicePageSize;
    return filteredDevices.slice(start, end);
  }, [filteredDevices, deviceCurrentPage, devicePageSize]);
  void _paginatedDevices;

  // 分页后的设备组列表
  const _paginatedGroups = useMemo(() => {
    const start = (deviceCurrentPage - 1) * devicePageSize;
    const end = start + devicePageSize;
    return filteredGroups.slice(start, end);
  }, [filteredGroups, deviceCurrentPage, devicePageSize]);
  void _paginatedGroups;

  // Modal 中过滤后的设备列表（排除已选设备）
  const filteredModalDevices = useMemo(() => {
    let result = MOCK_DEVICES.filter(d => !selectedDevices.includes(d.id));

    // 按产品类型筛选
    if (modalProductType !== 'all') {
      result = result.filter(device => device.productType === modalProductType);
    }

    // 按搜索文本筛选
    if (modalSearchText) {
      const searchLower = modalSearchText.toLowerCase();
      result = result.filter(
        device =>
          device.id.toLowerCase().includes(searchLower) ||
          device.name.toLowerCase().includes(searchLower)
      );
    }

    return result;
  }, [selectedDevices, modalProductType, modalSearchText]);

  // Modal 中过滤后的设备组列表（排除已选组）
  const filteredModalGroups = useMemo(() => {
    let result = MOCK_DEVICE_GROUPS.filter(g => !selectedGroups.includes(g.id));

    // 按设备类型筛选
    if (modalDeviceType !== 'all') {
      const neTypeMap: Record<string, string> = {
        enb: 'eNB',
        gnb: 'gNB',
        gsm: 'GSM',
      };
      const targetNeType = neTypeMap[modalDeviceType];
      const groupIdsWithDeviceType = new Set(
        MOCK_DEVICES
          .filter(device => device.neType === targetNeType)
          .map(device => device.groupId)
      );
      result = result.filter(group => groupIdsWithDeviceType.has(group.id));
    }

    // 按搜索文本筛选
    if (modalSearchText) {
      const searchLower = modalSearchText.toLowerCase();
      result = result.filter(
        group =>
          group.id.toLowerCase().includes(searchLower) ||
          group.name.toLowerCase().includes(searchLower)
      );
    }

    return result;
  }, [selectedGroups, modalDeviceType, modalSearchText]);

  // Modal 中分页后的设备列表
  const paginatedModalDevices = useMemo(() => {
    const start = (modalCurrentPage - 1) * modalPageSize;
    const end = start + modalPageSize;
    return filteredModalDevices.slice(start, end);
  }, [filteredModalDevices, modalCurrentPage, modalPageSize]);

  // Modal 中分页后的设备组列表
  const paginatedModalGroups = useMemo(() => {
    const start = (modalCurrentPage - 1) * modalPageSize;
    const end = start + modalPageSize;
    return filteredModalGroups.slice(start, end);
  }, [filteredModalGroups, modalCurrentPage, modalPageSize]);

  // KPI Modal 中过滤后的指标列表（排除已选指标）
  const filteredModalKpis = useMemo(() => {
    let result = MOCK_KPIS.filter(k => !selectedKpis.includes(k.id));

    // 按一级指标集筛选
    if (kpiModalPrimaryCategory !== 'all') {
      result = result.filter(k => k.primaryCategory === kpiModalPrimaryCategory);
    }

    // 按二级指标集筛选
    if (kpiModalSecondaryCategory !== 'all') {
      result = result.filter(k => k.secondaryCategory === kpiModalSecondaryCategory);
    }

    // 按搜索文本筛选
    if (kpiModalSearchText) {
      const searchLower = kpiModalSearchText.toLowerCase();
      result = result.filter(
        k =>
          k.id.toLowerCase().includes(searchLower) ||
          k.name.toLowerCase().includes(searchLower)
      );
    }

    return result;
  }, [selectedKpis, kpiModalPrimaryCategory, kpiModalSecondaryCategory, kpiModalSearchText]);

  // KPI Modal 中分页后的指标列表
  const paginatedModalKpis = useMemo(() => {
    const start = (kpiCurrentPage - 1) * kpiPageSize;
    const end = start + kpiPageSize;
    return filteredModalKpis.slice(start, end);
  }, [filteredModalKpis, kpiCurrentPage, kpiPageSize]);

  return (
    <Drawer
      title={getDrawerTitle()}
      open={open}
      onClose={handleClose}
      width={720}
      destroyOnClose
      styles={{
        body: { padding: 0 },
      }}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          {!isReadOnly && (
            <Checkbox
              checked={isDefault}
              onChange={(e) => setIsDefault(e.target.checked)}
            >
              {t('perf.query.setAsDefault')}
            </Checkbox>
          )}
          <Space>
            <Button onClick={handleClose}>{t('common.cancel')}</Button>
            {!isReadOnly && (
              <Button type="primary" loading={loading} onClick={handleSubmit}>
                {t('common.confirm')}
              </Button>
            )}
          </Space>
        </div>
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={DEFAULT_FORM_VALUES}
        disabled={isReadOnly}
      >
        {/* 基本信息 */}
        <Card
          title={t('perf.query.basicInfo')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          <Form.Item
            name="tempName"
            label={t('perf.query.templateName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input
              placeholder={t('perf.query.templateNamePlaceholder')}
              maxLength={50}
              showCount
            />
          </Form.Item>

          <Form.Item label={t('perf.query.templateType')}>
            <Radio.Group
              value={isPublic}
              onChange={(e) => setIsPublic(e.target.value)}
              optionType="button"
              buttonStyle="solid"
            >
              <Radio.Button value="0">{t('perf.query.privateTemplate')}</Radio.Button>
              <Radio.Button value="1">{t('perf.query.publicTemplate')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          {/* 详情模式下显示创建人/更新人 */}
          {isReadOnly && (
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item label={t('perf.query.creator')}>
                  <Input value={initialValues?.creator || '-'} disabled />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label={t('perf.query.updator')}>
                  <Input value={initialValues?.updator || '-'} disabled />
                </Form.Item>
              </Col>
            </Row>
          )}

          <Form.Item name="description" label={t('common.description')}>
            <TextArea
              placeholder={t('common.pleaseInput')}
              maxLength={500}
              rows={3}
              showCount
            />
          </Form.Item>
        </Card>

        {/* 周期设定 */}
        <Card
          title={t('perf.query.periodSetting')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          <Form.Item label={t('perf.query.granularity')} required>
            <Radio.Group value={reportPeriod} onChange={handleReportPeriodChange}>
              <Radio.Button value="15">15Min</Radio.Button>
              <Radio.Button value="60">60Min</Radio.Button>
              <Radio.Button value="1440">24hour</Radio.Button>
            </Radio.Group>
          </Form.Item>

          {showTimeConfig && (
            <>
              <Form.Item>
                <Space>
                  <Checkbox
                    checked={checkAll === '1'}
                    onChange={handleCheckAllChange}
                  >
                    {t('perf.query.allTimeSlots')}
                  </Checkbox>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('perf.query.allTimeSlotsTip')}
                  </Text>
                </Space>
              </Form.Item>

              <div
                style={{
                  padding: 16,
                  background: token.colorBgLayout,
                  borderRadius: 8,
                  opacity: timeConfigDisabled ? 0.5 : 1,
                }}
              >
                {/* 周选择 */}
                <div style={{ marginBottom: 16 }}>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    {t('perf.query.week')}
                  </Text>
                  <Checkbox.Group
                    value={week}
                    onChange={handleWeekChange}
                    disabled={timeConfigDisabled}
                  >
                    <Row>
                      {WEEK_OPTIONS.map((item) => (
                        <Col key={item.value} span={3}>
                          <Checkbox value={item.value}>{item.label}</Checkbox>
                        </Col>
                      ))}
                    </Row>
                  </Checkbox.Group>
                </div>

                {/* 忙时快捷选择 */}
                <div style={{ marginBottom: 16 }}>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    {t('perf.query.busyHour')}
                  </Text>
                  <Space>
                    <Select
                      value={busyTime}
                      onChange={handleBusyTimeChange}
                      placeholder={t('perf.query.selectBusyHour')}
                      style={{ width: 120 }}
                      disabled={timeConfigDisabled}
                      allowClear
                    >
                      <Select.Option value="6">{t('perf.query.busyHour6')}</Select.Option>
                      <Select.Option value="8">{t('perf.query.busyHour8')}</Select.Option>
                    </Select>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {t('perf.query.busyHourTip')}
                    </Text>
                  </Space>
                </div>

                {/* 小时选择 */}
                <div>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    {t('perf.query.hour')}
                  </Text>
                  <Checkbox.Group
                    value={hour}
                    onChange={handleHourChange}
                    disabled={timeConfigDisabled}
                  >
                    <Row>
                      {HOUR_OPTIONS.map((item) => (
                        <Col key={item.value} span={3}>
                          <Checkbox value={item.value}>{item.label}</Checkbox>
                        </Col>
                      ))}
                    </Row>
                  </Checkbox.Group>
                </div>
              </div>
            </>
          )}
        </Card>

        {/* 设备选择 */}
        <Card
          title={t('perf.query.deviceList')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          <Form.Item label={t('perf.query.deviceSelectType')} required>
            <Radio.Group
              value={selDeviceType}
              onChange={(e) => {
                setSelDeviceType(e.target.value);
                setSelectedDevices([]);
                setSelectedGroups([]);
              }}
              optionType="button"
              buttonStyle="solid"
            >
              <Radio.Button value="2">{t('perf.query.selectByDevice')}</Radio.Button>
              <Radio.Button value="1">{t('perf.query.selectByGroup')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Divider style={{ margin: '12px 0' }} />

          {/* 已选设备/设备组 */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: 620 }}>
              <span>
                {selDeviceType === '2' ? t('perf.query.selectedDevices') : t('perf.query.selectedGroups')}
                <Tag color="blue" style={{ marginLeft: 8 }}>
                  {selDeviceType === '2' ? selectedDevices.length : selectedGroups.length}
                  {selDeviceType === '2' ? t('perf.query.deviceUnit') : t('perf.query.groupUnit')}
                </Tag>
              </span>
              <Space size={8}>
                {selDeviceType === '2' && (
                  <Button
                    size="small"
                    icon={<EditOutlined />}
                    onClick={handleOpenBatchInput}
                  >
                    {t('perf.query.batchInput')}
                  </Button>
                )}
                <Button
                  size="small"
                  icon={<ClearOutlined />}
                  disabled={selDeviceType === '2' ? selectedDevices.length === 0 : selectedGroups.length === 0}
                  onClick={() => {
                    if (selDeviceType === '2') {
                      setSelectedDevices([]);
                    } else {
                      setSelectedGroups([]);
                    }
                  }}
                >
                  {t('common.clear')}
                </Button>
                <Button
                  size="small"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setAddDeviceModalVisible(true)}
                >
                  {selDeviceType === '2' ? t('perf.query.addDevice') : t('perf.query.addGroup')}
                </Button>
              </Space>
            </div>
          }>
            {selDeviceType === '2' ? (
              selectedDevices.length === 0 ? (
                <div style={{ padding: 24, textAlign: 'center', color: token.colorTextSecondary, border: `1px dashed ${token.colorBorder}`, borderRadius: 6 }}>
                  {t('perf.query.pleaseSelectDevice')}
                </div>
              ) : (
                <div style={{ border: `1px solid ${token.colorBorder}`, borderRadius: 6, maxHeight: 300, overflow: 'auto' }}>
                  <Table
                    size="small"
                    dataSource={MOCK_DEVICES.filter(d => selectedDevices.includes(d.id))}
                    rowKey="id"
                    pagination={false}
                    columns={[
                      { title: t('perf.query.baseStationCode'), dataIndex: 'id', ellipsis: true },
                      { title: t('perf.query.baseStationName'), dataIndex: 'name', ellipsis: true },
                      {
                        title: t('perf.query.productType'),
                        dataIndex: 'productType',
                        width: 80,
                        render: (val) => <Tag>{val}</Tag>
                      },
                      {
                        title: '',
                        width: 40,
                        render: (_: unknown, record: { id: string }) => (
                          <Button
                            type="text"
                            size="small"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => setSelectedDevices(prev => prev.filter(id => id !== record.id))}
                          />
                        ),
                      },
                    ]}
                  />
                </div>
              )
            ) : (
              selectedGroups.length === 0 ? (
                <div style={{ padding: 24, textAlign: 'center', color: token.colorTextSecondary, border: `1px dashed ${token.colorBorder}`, borderRadius: 6 }}>
                  {t('perf.query.pleaseSelectGroup')}
                </div>
              ) : (
                <div style={{ border: `1px solid ${token.colorBorder}`, borderRadius: 6, maxHeight: 300, overflow: 'auto' }}>
                  <Table
                    size="small"
                    dataSource={MOCK_DEVICE_GROUPS.filter(g => selectedGroups.includes(g.id))}
                    rowKey="id"
                    pagination={false}
                    columns={[
                      {
                        title: t('perf.query.groupName'),
                        dataIndex: 'name',
                        ellipsis: true,
                        render: (val) => (
                          <span>
                            <FolderOutlined style={{ marginRight: 8, color: '#FA8C16' }} />
                            {val}
                          </span>
                        )
                      },
                      {
                        title: '',
                        width: 40,
                        render: (_: unknown, record: { id: string }) => (
                          <Button
                            type="text"
                            size="small"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => setSelectedGroups(prev => prev.filter(id => id !== record.id))}
                          />
                        ),
                      },
                    ]}
                  />
                </div>
              )
            )}
          </Form.Item>
        </Card>

        {/* 指标列表 */}
        <Card
          title={t('perf.query.kpiList')}
          size="small"
          style={{ marginBottom: 16, borderRadius: 0 }}
          styles={{ body: { padding: '16px 24px' } }}
        >
          {/* 等级选择（仅 eNB） */}
          {networkType === 'enb' && (
            <Form.Item label={t('perf.query.indicatorLevel')}>
              <Radio.Group
                value={indicatorLevel}
                onChange={(e) => setIndicatorLevel(e.target.value)}
              >
                <Radio value="device">{t('perf.query.indicatorLevelDevice')}</Radio>
                <Radio value="plmn">{t('perf.query.indicatorLevelPlmn')}</Radio>
              </Radio.Group>
            </Form.Item>
          )}

          <Divider style={{ margin: '12px 0' }} />

          {/* 已选指标 */}
          <Form.Item label={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: 620 }}>
              <span>
                {t('perf.query.selectedKpis')}
                <Tag color="green" style={{ marginLeft: 8 }}>
                  {selectedKpis.length}
                  {t('perf.query.kpiUnit')}
                </Tag>
              </span>
              <Space size={8}>
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={handleOpenKpiBatchInput}
                >
                  {t('perf.query.batchInput')}
                </Button>
                <Button
                  size="small"
                  icon={<ClearOutlined />}
                  disabled={selectedKpis.length === 0}
                  onClick={() => setSelectedKpis([])}
                >
                  {t('common.clear')}
                </Button>
                <Button
                  size="small"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setAddKpiModalVisible(true)}
                >
                  {t('perf.query.addKpi')}
                </Button>
              </Space>
            </div>
          }>
            {selectedKpis.length === 0 ? (
              <div style={{ padding: 24, textAlign: 'center', color: token.colorTextSecondary, border: `1px dashed ${token.colorBorder}`, borderRadius: 6 }}>
                {t('perf.query.pleaseSelectKpi')}
              </div>
            ) : (
              <div style={{ border: `1px solid ${token.colorBorder}`, borderRadius: 6, maxHeight: 300, overflow: 'auto' }}>
                <Table
                  size="small"
                  dataSource={MOCK_KPIS.filter(k => selectedKpis.includes(k.id))}
                  rowKey="id"
                  pagination={false}
                  columns={[
                    { title: t('perf.query.kpiId'), dataIndex: 'id', ellipsis: true },
                    { title: t('perf.query.kpiName'), dataIndex: 'name', ellipsis: true },
                    {
                      title: '',
                      width: 40,
                      render: (_: unknown, record: { id: string }) => (
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => setSelectedKpis(prev => prev.filter(id => id !== record.id))}
                        />
                      ),
                    },
                  ]}
                />
              </div>
            )}
          </Form.Item>
        </Card>
      </Form>

      {/* 添加设备/设备组弹窗 */}
      <Modal
        open={addDeviceModalVisible}
        title={selDeviceType === '2' ? t('perf.query.addDevice') : t('perf.query.addGroup')}
        onCancel={() => {
          setAddDeviceModalVisible(false);
          setTempSelectedDevices([]);
          setTempSelectedGroups([]);
          setModalSearchText('');
          setModalCurrentPage(1);
        }}
        onOk={() => {
          if (selDeviceType === '2') {
            // 按设备添加：去重合并
            setSelectedDevices(prev => [...new Set([...prev, ...tempSelectedDevices])]);
          } else {
            // 按设备组添加
            setSelectedGroups(prev => [...new Set([...prev, ...tempSelectedGroups])]);
          }
          setAddDeviceModalVisible(false);
          setTempSelectedDevices([]);
          setTempSelectedGroups([]);
          setModalSearchText('');
          setModalCurrentPage(1);
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={700}
      >
        {/* 筛选 */}
        {selDeviceType === '2' ? (
          <div style={{ marginBottom: 12 }}>
            <Select
              style={{ width: 150, marginRight: 12 }}
              value={modalProductType}
              onChange={(value) => {
                setModalProductType(value);
                setModalCurrentPage(1);
              }}
              options={[
                { label: t('perf.query.allProductTypes'), value: 'all' },
                { label: 'LTE', value: 'LTE' },
                { label: 'NR', value: 'NR' },
              ]}
            />
            <Input
              style={{ width: 200 }}
              placeholder={t('perf.query.deviceSearchPlaceholder')}
              prefix={<SearchOutlined />}
              value={modalSearchText}
              onChange={(e) => {
                setModalSearchText(e.target.value);
                setModalCurrentPage(1);
              }}
              allowClear
            />
          </div>
        ) : (
          <div style={{ marginBottom: 12 }}>
            <Select
              style={{ width: 150, marginRight: 12 }}
              value={modalDeviceType}
              onChange={(value) => {
                setModalDeviceType(value);
                setModalCurrentPage(1);
              }}
              options={[
                { label: t('perf.query.allDevices'), value: 'all' },
                { label: t('perf.query.enbDevices'), value: 'enb' },
                { label: t('perf.query.gnbDevices'), value: 'gnb' },
                { label: t('perf.query.gsmDevices'), value: 'gsm' },
              ]}
            />
            <Input
              style={{ width: 200 }}
              placeholder={t('perf.query.groupSearchPlaceholder')}
              prefix={<SearchOutlined />}
              value={modalSearchText}
              onChange={(e) => {
                setModalSearchText(e.target.value);
                setModalCurrentPage(1);
              }}
              allowClear
            />
          </div>
        )}

        {selDeviceType === '2' ? (
          <>
            <Alert
              type="info"
              showIcon
              message={t('perf.query.deviceSelectionInfo', {
                total: filteredModalDevices.length,
                selected: tempSelectedDevices.length,
              })}
              style={{ marginBottom: 16 }}
            />
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input
                placeholder={t('perf.query.deviceSearchPlaceholder')}
                prefix={<SearchOutlined />}
                value={modalSearchText}
                onChange={(e) => {
                  setModalSearchText(e.target.value);
                  setModalCurrentPage(1);
                }}
                allowClear
                style={{ width: 250 }}
              />
              <Checkbox
                checked={tempSelectedDevices.length === filteredModalDevices.length && filteredModalDevices.length > 0}
                indeterminate={tempSelectedDevices.length > 0 && tempSelectedDevices.length < filteredModalDevices.length}
                onChange={(e) => {
                  if (e.target.checked) {
                    setTempSelectedDevices(filteredModalDevices.map(d => d.id));
                  } else {
                    setTempSelectedDevices([]);
                  }
                }}
              >
                {t('perf.query.selectAllDevices', { count: filteredModalDevices.length })}
              </Checkbox>
            </div>
            <Table
              size="small"
              dataSource={paginatedModalDevices}
              rowKey="id"
              pagination={false}
              scroll={{ y: 250 }}
              rowSelection={{
                selectedRowKeys: tempSelectedDevices,
                onChange: (keys) => setTempSelectedDevices(keys as string[]),
              }}
              columns={[
                { title: t('perf.query.baseStationCode'), dataIndex: 'id', width: 120 },
                { title: t('perf.query.baseStationName'), dataIndex: 'name', ellipsis: true },
                {
                  title: t('perf.query.productType'),
                  dataIndex: 'productType',
                  width: 80,
                  render: (val) => <Tag>{val}</Tag>,
                },
              ]}
            />
            {filteredModalDevices.length > modalPageSize && (
              <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
                <Pagination
                  size="small"
                  current={modalCurrentPage}
                  pageSize={modalPageSize}
                  total={filteredModalDevices.length}
                  onChange={(page, pageSize) => {
                    setModalCurrentPage(page);
                    setModalPageSize(pageSize);
                  }}
                  showSizeChanger
                  showTotal={(total) => t('perf.query.totalCount', { count: total })}
                />
              </div>
            )}
          </>
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={t('perf.query.groupSelectionInfo', {
                total: filteredModalGroups.length,
                selected: tempSelectedGroups.length,
              })}
              style={{ marginBottom: 16 }}
            />
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input
                placeholder={t('perf.query.groupSearchPlaceholder')}
                prefix={<SearchOutlined />}
                value={modalSearchText}
                onChange={(e) => {
                  setModalSearchText(e.target.value);
                  setModalCurrentPage(1);
                }}
                allowClear
                style={{ width: 250 }}
              />
              <Checkbox
                checked={tempSelectedGroups.length === filteredModalGroups.length && filteredModalGroups.length > 0}
                indeterminate={tempSelectedGroups.length > 0 && tempSelectedGroups.length < filteredModalGroups.length}
                onChange={(e) => {
                  if (e.target.checked) {
                    setTempSelectedGroups(filteredModalGroups.map(g => g.id));
                  } else {
                    setTempSelectedGroups([]);
                  }
                }}
              >
                {t('perf.query.selectAllGroups', { count: filteredModalGroups.length })}
              </Checkbox>
            </div>
            <Table
              size="small"
              dataSource={paginatedModalGroups}
              rowKey="id"
              pagination={false}
              scroll={{ y: 250 }}
              rowSelection={{
                selectedRowKeys: tempSelectedGroups,
                onChange: (keys) => setTempSelectedGroups(keys as string[]),
              }}
              columns={[
                {
                  title: t('perf.query.groupName'),
                  dataIndex: 'name',
                  ellipsis: true,
                  render: (val) => (
                    <span>
                      <FolderOutlined style={{ marginRight: 8, color: '#FA8C16' }} />
                      {val}
                    </span>
                  ),
                },
              ]}
            />
            {filteredModalGroups.length > modalPageSize && (
              <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
                <Pagination
                  size="small"
                  current={modalCurrentPage}
                  pageSize={modalPageSize}
                  total={filteredModalGroups.length}
                  onChange={(page, pageSize) => {
                    setModalCurrentPage(page);
                    setModalPageSize(pageSize);
                  }}
                  showSizeChanger
                  showTotal={(total) => t('perf.query.totalCount', { count: total })}
                />
              </div>
            )}
          </>
        )}
      </Modal>

      {/* 添加指标弹窗 */}
      <Modal
        open={addKpiModalVisible}
        title={t('perf.query.addKpi')}
        onCancel={() => {
          setAddKpiModalVisible(false);
          setTempSelectedKpis([]);
          setKpiModalSearchText('');
          setKpiModalPrimaryCategory('all');
          setKpiModalSecondaryCategory('all');
          setKpiCurrentPage(1);
        }}
        onOk={() => {
          // 去重合并
          setSelectedKpis(prev => [...new Set([...prev, ...tempSelectedKpis])]);
          setAddKpiModalVisible(false);
          setTempSelectedKpis([]);
          setKpiModalSearchText('');
          setKpiModalPrimaryCategory('all');
          setKpiModalSecondaryCategory('all');
          setKpiCurrentPage(1);
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={700}
      >
        {/* 筛选 */}
        <div style={{ marginBottom: 12, display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <Select
            style={{ width: 150 }}
            value={kpiModalPrimaryCategory}
            onChange={(value) => {
              setKpiModalPrimaryCategory(value);
              setKpiModalSecondaryCategory('all'); // 切换一级分类时重置二级分类
              setKpiCurrentPage(1);
            }}
            options={[
              { label: t('perf.query.allPrimaryCategories'), value: 'all' },
              ...PRIMARY_CATEGORIES.map(pc => ({ label: pc.name, value: pc.id })),
            ]}
          />
          <Select
            style={{ width: 150 }}
            value={kpiModalSecondaryCategory}
            onChange={(value) => {
              setKpiModalSecondaryCategory(value);
              setKpiCurrentPage(1);
            }}
            disabled={kpiModalPrimaryCategory === 'all'}
            options={[
              { label: t('perf.query.allSecondaryCategories'), value: 'all' },
              ...(kpiModalPrimaryCategory !== 'all' && SECONDARY_CATEGORIES[kpiModalPrimaryCategory]
                ? SECONDARY_CATEGORIES[kpiModalPrimaryCategory].map(sc => ({ label: sc.name, value: sc.id }))
                : []),
            ]}
          />
          <Input
            style={{ width: 200 }}
            placeholder={t('perf.query.kpiSearchPlaceholder')}
            prefix={<SearchOutlined />}
            value={kpiModalSearchText}
            onChange={(e) => {
              setKpiModalSearchText(e.target.value);
              setKpiCurrentPage(1);
            }}
            allowClear
          />
        </div>

        {/* 指标列表 */}
        <Alert
          type="info"
          showIcon
          message={t('perf.query.kpiSelectionInfo', {
            total: filteredModalKpis.length,
            selected: tempSelectedKpis.length,
          })}
          style={{ marginBottom: 16 }}
        />
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Input
            placeholder={t('perf.query.kpiSearchPlaceholder')}
            prefix={<SearchOutlined />}
            value={kpiModalSearchText}
            onChange={(e) => {
              setKpiModalSearchText(e.target.value);
              setKpiCurrentPage(1);
            }}
            allowClear
            style={{ width: 250 }}
          />
          <Checkbox
            checked={tempSelectedKpis.length === filteredModalKpis.length && filteredModalKpis.length > 0}
            indeterminate={tempSelectedKpis.length > 0 && tempSelectedKpis.length < filteredModalKpis.length}
            onChange={(e) => {
              if (e.target.checked) {
                setTempSelectedKpis(filteredModalKpis.map(k => k.id));
              } else {
                setTempSelectedKpis([]);
              }
            }}
          >
            {t('perf.query.selectAllKpis', { count: filteredModalKpis.length })}
          </Checkbox>
        </div>
        <Table
          size="small"
          dataSource={paginatedModalKpis}
          rowKey="id"
          pagination={false}
          scroll={{ y: 250 }}
          rowSelection={{
            selectedRowKeys: tempSelectedKpis,
            onChange: (keys) => setTempSelectedKpis(keys as string[]),
          }}
          columns={[
            { title: t('perf.query.kpiId'), dataIndex: 'id', width: 120 },
            { title: t('perf.query.kpiName'), dataIndex: 'name', ellipsis: true },
            {
              title: t('perf.query.unit'),
              dataIndex: 'unit',
              width: 80,
              render: (val) => <Tag>{val}</Tag>,
            },
          ]}
        />
        {filteredModalKpis.length > kpiPageSize && (
          <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
            <Pagination
              size="small"
              current={kpiCurrentPage}
              pageSize={kpiPageSize}
              total={filteredModalKpis.length}
              onChange={(page, pageSize) => {
                setKpiCurrentPage(page);
                setKpiPageSize(pageSize);
              }}
              showSizeChanger
              showTotal={(total) => t('perf.query.totalCount', { count: total })}
            />
          </div>
        )}
      </Modal>

      {/* 批量输入弹窗 */}
      <Modal
        title={selDeviceType === '2' ? t('perf.query.batchInputDeviceTitle') : t('perf.query.batchInputGroupTitle')}
        open={batchInputVisible}
        onCancel={() => {
          setBatchInputVisible(false);
          setBatchInputValue('');
        }}
        onOk={handleBatchInputConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={600}
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {selDeviceType === '2'
              ? t('perf.query.batchInputDevicePlaceholder')
              : t('perf.query.batchInputGroupPlaceholder')
            }
          </Text>
          <TextArea
            placeholder={selDeviceType === '2'
              ? 'ENB00001\nENB00002, ENB00003; GNB00001'
              : 'group-1\ngroup-2, group-3'
            }
            rows={6}
            value={batchInputValue}
            onChange={(e) => setBatchInputValue(e.target.value)}
          />
        </div>

        {/* 预览结果 */}
        {batchInputValue.trim() && (
          <Alert
            type="info"
            showIcon
            message={
              <span>
                {t('perf.query.batchInputPreview', {
                  count: parseBatchInput(batchInputValue).length,
                  type: selDeviceType === '2' ? t('perf.query.deviceUnit') : t('perf.query.groupUnit')
                })}
              </span>
            }
          />
        )}
      </Modal>

      {/* KPI 批量输入弹窗 */}
      <Modal
        title={t('perf.query.batchInputKpiTitle')}
        open={kpiBatchInputVisible}
        onCancel={() => {
          setKpiBatchInputVisible(false);
          setKpiBatchInputValue('');
        }}
        onOk={handleKpiBatchInputConfirm}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={600}
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {t('perf.query.batchInputKpiPlaceholder')}
          </Text>
          <TextArea
            placeholder="KPI_001\nKPI_002, KPI_003; KPI_004"
            rows={6}
            value={kpiBatchInputValue}
            onChange={(e) => setKpiBatchInputValue(e.target.value)}
          />
        </div>

        {/* 预览结果 */}
        {kpiBatchInputValue.trim() && (
          <Alert
            type="info"
            showIcon
            message={
              <span>
                {t('perf.query.batchInputPreview', {
                  count: parseBatchInput(kpiBatchInputValue).length,
                  type: t('perf.query.kpiUnit')
                })}
              </span>
            }
          />
        )}
      </Modal>
    </Drawer>
  );
}
