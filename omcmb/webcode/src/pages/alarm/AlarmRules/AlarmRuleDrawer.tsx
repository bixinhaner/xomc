import React, { useCallback, useMemo, useState } from 'react';
import {
  App,
  Button,
  Checkbox,
  DatePicker,
  Drawer,
  Form,
  Input,
  Modal,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  theme,
} from 'antd';
import type { TableProps } from 'antd';
import { useT } from '@/hooks/useT';
import { useDeviceList, useDeviceGroups, useDevicesByIds } from '@core/hooks/api/useDevices';
import { useAllAlarmDefinitions } from '@core/hooks/api/useAlarmDefinitions';
import type { AlarmRule } from '@core/types/alarm';
import type { AlarmDefinition } from '@core/types/alarmDefinition';
import type { AlarmSeverity } from '@core/types/common';
import type { Device, DeviceGroup } from '@core/types/device';
import dayjs, { type Dayjs } from 'dayjs';

const { RangePicker } = DatePicker;

// 执行动作配置 (匹配后端 action 值)
const RULE_TYPE_OPTIONS: { value: string; labelKey: string }[] = [
  { value: 'ignore', labelKey: 'alarm.rule.action.ignore' },
  { value: 'auto_acknowledge', labelKey: 'alarm.rule.action.autoAck' },
  { value: 'auto_clear', labelKey: 'alarm.rule.action.autoClear' },
  { value: 'notify_email', labelKey: 'alarm.rule.action.notifyEmail' },
];

const MAX_EMAIL_RECIPIENTS = 50;
const EMAIL_ADDRESS_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// 事件类型配置
const EVENT_TYPE_OPTIONS: { value: string; labelKey: string }[] = [
  { value: '30000', labelKey: 'alarm.eventType.30000' },
  { value: '30001', labelKey: 'alarm.eventType.30001' },
  { value: '30002', labelKey: 'alarm.eventType.30002' },
  { value: '30003', labelKey: 'alarm.eventType.30003' },
  { value: '30004', labelKey: 'alarm.eventType.30004' },
  { value: '30006', labelKey: 'alarm.eventType.30006' },
];

// 告警级别配置
const SEVERITY_OPTIONS: { value: AlarmSeverity; labelKey: string; color: string }[] = [
  { value: 'critical', labelKey: 'alarm.severity.critical', color: 'red' },
  { value: 'major', labelKey: 'alarm.severity.major', color: 'orange' },
  { value: 'minor', labelKey: 'alarm.severity.minor', color: 'blue' },
  { value: 'warning', labelKey: 'alarm.severity.warning', color: 'gold' },
];

// 设备类型配置
const DEVICE_TYPE_OPTIONS = [
  { label: 'eNB', value: 'eNB' },
  { label: 'gNB', value: 'gNB' },
  { label: 'GSM', value: 'GSM' },
];

const DEVICE_TYPE_TO_NETWORK_TYPE: Record<string, string> = {
  eNB: 'lte',
  gNB: 'nr',
  GSM: 'gsm',
};

const DEVICE_TABLE_DEFAULT_PAGE_SIZE = 5;

const MAX_VISIBLE_SELECTED_ALARMS = 5;

interface AlarmRuleDrawerProps {
  open: boolean;
  mode: 'add' | 'edit' | 'view';
  rule?: AlarmRule | null;
  existingNames?: string[];
  onClose: () => void;
  onSubmit: (data: AlarmRuleFormData) => Promise<void>;
}

export interface AlarmRuleFormData {
  ruleName: string;
  status: boolean;
  ruleType: string;
  deviceSelectionMode: 'devices' | 'groups';
  selectedDevices: string[];
  selectedGroups: string[];
  selectedAlarms: string[];
  emailRecipients: string[];
  timeRange?: [string, string];
}

interface AlarmLibraryItem {
  alarmIdentifier: string;
  alarmName: string;
  eventType?: string;
  severity: AlarmSeverity;
}

function isAlarmLibraryItem(item: AlarmLibraryItem | undefined): item is AlarmLibraryItem {
  return item !== undefined;
}

function mergeVisibleSelection(
  previousKeys: string[],
  visibleKeys: string[],
  nextVisibleKeys: string[],
): string[] {
  const visibleKeySet = new Set(visibleKeys);
  const mergedKeySet = new Set(previousKeys.filter((key) => !visibleKeySet.has(key)));

  nextVisibleKeys.forEach((key) => {
    mergedKeySet.add(key);
  });

  return Array.from(mergedKeySet);
}

function toggleVisibleSelection(
  previousKeys: string[],
  visibleKeys: string[],
  checked: boolean,
): string[] {
  if (checked) {
    return mergeVisibleSelection(previousKeys, visibleKeys, visibleKeys);
  }

  const visibleKeySet = new Set(visibleKeys);
  return previousKeys.filter((key) => !visibleKeySet.has(key));
}

function sortSelectedFirst<T>(items: T[], isSelected: (item: T) => boolean): T[] {
  return [...items].sort((left, right) => Number(isSelected(right)) - Number(isSelected(left)));
}

function getConditionValues(rule: AlarmRule | null | undefined, field: string): string[] {
  if (!rule) {
    return [];
  }

  return rule.conditions
    .filter((condition) => condition.field === field)
    .flatMap((condition) => (Array.isArray(condition.value) ? condition.value : [condition.value]))
    .map((value) => String(value))
    .filter((value) => value.length > 0);
}

function mapSeverityCodeToAlarmSeverity(code: number): AlarmSeverity {
  switch (code) {
    case 1:
    case 31001:
      return 'critical';
    case 2:
    case 31002:
      return 'major';
    case 3:
    case 31003:
      return 'minor';
    case 4:
    case 31004:
    default:
      return 'warning';
  }
}

function mapAlarmDefinitionEventType(eventType: AlarmDefinition['eventType']): string | undefined {
  if (eventType === undefined || eventType === null || eventType === '') {
    return undefined;
  }

  const value = String(eventType);
  switch (value) {
    case 'communication':
      return '30000';
    case 'qualityOfService':
      return '30001';
    case 'processingError':
      return '30002';
    case 'device':
    case 'equipment':
      return '30003';
    case 'environment':
      return '30004';
    case 'performance':
    case 'service':
      return '30006';
    default:
      return value;
  }
}

function mapAlarmDefinitionToLibraryItem(definition: AlarmDefinition): AlarmLibraryItem {
  return {
    alarmIdentifier: definition.identifier,
    alarmName:
      definition.cnProbableCause ||
      definition.cnName ||
      definition.enProbableCause ||
      definition.enName ||
      definition.identifier,
    eventType: mapAlarmDefinitionEventType(definition.eventType),
    severity: mapSeverityCodeToAlarmSeverity(definition.severityCode),
  };
}

function attachDeviceType(device: Device): DeviceWithType {
  let deviceType: 'eNB' | 'gNB' | 'GSM' = 'eNB';
  const name = device.name?.toLowerCase() || '';
  const networkType = device.networkType?.toLowerCase() || '';

  if (
    name.includes('gnb') ||
    networkType.includes('gnb') ||
    networkType.includes('5g') ||
    networkType.includes('nr')
  ) {
    deviceType = 'gNB';
  } else if (name.includes('gsm') || networkType.includes('gsm')) {
    deviceType = 'GSM';
  }

  return { ...device, deviceType };
}

// 设备数据类型（包含设备类型字段）
interface DeviceWithType extends Device {
  deviceType: 'eNB' | 'gNB' | 'GSM';
}

interface SelectedDeviceRow {
  id: string;
  sn: string;
  name?: string;
  connStatus?: string;
  deviceType: 'eNB' | 'gNB' | 'GSM';
}

// 设备组数据类型（包含层级结构）
interface DeviceGroupWithLevel extends DeviceGroup {
  level: number;
  fullName: string;
}

export default function AlarmRuleDrawer({ open, mode, rule, existingNames = [], onClose, onSubmit }: AlarmRuleDrawerProps) {
  const { message } = App.useApp();
  const t = useT();
  const { token } = theme.useToken();
  const [form] = Form.useForm<AlarmRuleFormData>();
  const [loading, setLoading] = useState(false);
  const [currentRuleType, setCurrentRuleType] = useState<string | undefined>(() => rule?.ruleType);
  const [deviceTablePage, setDeviceTablePage] = useState(1);
  const [deviceTablePageSize, setDeviceTablePageSize] = useState(DEVICE_TABLE_DEFAULT_PAGE_SIZE);
  const [alarmTablePage, setAlarmTablePage] = useState(1);
  const [alarmTablePageSize, setAlarmTablePageSize] = useState(5);
  const initialSelectedDevices = useMemo(() => getConditionValues(rule, 'device_id'), [rule]);
  const initialSelectedGroups = useMemo(() => getConditionValues(rule, 'device_group_id'), [rule]);
  const [deviceSelectionMode, setDeviceSelectionMode] = useState<'devices' | 'groups'>(() =>
    initialSelectedGroups.length > 0 && initialSelectedDevices.length === 0 ? 'groups' : 'devices',
  );
  const [selectedDevices, setSelectedDevices] = useState<string[]>(initialSelectedDevices);
  const [selectedGroups, setSelectedGroups] = useState<string[]>(initialSelectedGroups);
  const [selectedAlarms, setSelectedAlarms] = useState<string[]>(() => getConditionValues(rule, 'alarm_identifier'));
  const [selectedDeviceModalOpen, setSelectedDeviceModalOpen] = useState(false);
  const [selectedDeviceKeyword, setSelectedDeviceKeyword] = useState('');
  const [timeRange, setTimeRange] = useState<[Dayjs, Dayjs] | null>(() =>
    rule?.effectiveStart && rule.effectiveEnd
      ? [dayjs(rule.effectiveStart), dayjs(rule.effectiveEnd)]
      : null,
  );
  const [alarmFilter, setAlarmFilter] = useState({
    keyword: '',
    eventType: undefined as string | undefined,
    severity: undefined as AlarmSeverity | undefined,
  });
  const [alarmError, setAlarmError] = useState<string | null>(null);

  // 设备筛选状态
  const [deviceFilter, setDeviceFilter] = useState({
    deviceTypes: [] as string[],
    snKeyword: '',
  });

  const deviceListParams = useMemo(() => {
    const keyword = deviceFilter.snKeyword.trim();
    const networkType = deviceFilter.deviceTypes
      .map((deviceType) => DEVICE_TYPE_TO_NETWORK_TYPE[deviceType])
      .filter(Boolean)
      .join(',');

    return {
      page: deviceTablePage,
      pageSize: deviceTablePageSize,
      ...(keyword ? { searchText: keyword } : {}),
      ...(networkType ? { networkType } : {}),
    };
  }, [deviceFilter.deviceTypes, deviceFilter.snKeyword, deviceTablePage, deviceTablePageSize]);

  // 获取设备列表：按表格分页请求，避免只加载前 100 条导致最多 20 页。
  const { data: deviceData, isLoading: deviceLoading } = useDeviceList(deviceListParams);
  const selectedDeviceQueries = useDevicesByIds(selectedDevices);

  // 获取设备组列表
  const { data: groupsData, isLoading: groupsLoading } = useDeviceGroups();
  const { data: alarmDefinitionData, isLoading: alarmDefinitionLoading } = useAllAlarmDefinitions();

  const isViewMode = mode === 'view';
  const title = mode === 'add' ? t('common.add') : mode === 'edit' ? t('common.edit') : t('common.detail');
  const ruleTypeOptions = useMemo(() => {
    const base = RULE_TYPE_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) }));
    return rule?.ruleType === 'default'
      ? [...base, { value: 'default', label: t('alarm.ruleType.defaultLegacy'), disabled: true }]
      : base;
  }, [rule?.ruleType, t]);
  const eventTypeOptions = useMemo(
    () => EVENT_TYPE_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
    [t],
  );
  const severityOptions = useMemo(
    () => SEVERITY_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
    [t],
  );

  // 处理设备数据，添加设备类型
  const devicesWithType: DeviceWithType[] = useMemo(() => {
    return (deviceData?.items || []).map(attachDeviceType);
  }, [deviceData]);

  // 处理设备组数据，构建层级结构
  const groupsWithLevel: DeviceGroupWithLevel[] = useMemo(() => {
    const groups = groupsData?.groups || [];
    if (groups.length === 0) return [];

    // 构建父子关系映射
    const groupMap = new Map<string, DeviceGroupWithLevel>();
    const rootGroups: DeviceGroupWithLevel[] = [];

    // 第一遍：创建所有节点
    groups.forEach(g => {
      groupMap.set(g.id, {
        ...g,
        level: 0,
        fullName: g.name,
      });
    });

    // 第二遍：建立层级关系
    groups.forEach(g => {
      const node = groupMap.get(g.id)!;
      if (g.parentId && groupMap.has(g.parentId)) {
        const parent = groupMap.get(g.parentId)!;
        node.level = parent.level + 1;
        node.fullName = `${parent.name} / ${g.name}`;
      } else {
        node.level = 0;
        rootGroups.push(node);
      }
    });

    // 按层级排序
    const result: DeviceGroupWithLevel[] = [];
    const addToResult = (group: DeviceGroupWithLevel) => {
      result.push(group);
      // 添加子节点
      groups.forEach(g => {
        if (g.parentId === group.id) {
          const child = groupMap.get(g.id);
          if (child) addToResult(child);
        }
      });
    };
    rootGroups.forEach(g => addToResult(g));

    return result;
  }, [groupsData]);

  const filteredDevices = devicesWithType;
  const deviceTableTotal = deviceData?.total ?? 0;

  const selectedDeviceRows: SelectedDeviceRow[] = useMemo(() => {
    const deviceMap = new Map<string, SelectedDeviceRow>();
    devicesWithType.forEach((device) => deviceMap.set(device.id, device));
    selectedDeviceQueries.forEach((query) => {
      if (query.data) {
        const device = attachDeviceType(query.data);
        deviceMap.set(device.id, device);
      }
    });

    return selectedDevices.map((id) => deviceMap.get(id) ?? ({
      id,
      sn: id,
      name: id,
      connStatus: 'unknown',
      deviceType: 'eNB',
    }));
  }, [devicesWithType, selectedDeviceQueries, selectedDevices]);

  const selectedDeviceLoading = selectedDeviceQueries.some((query) => query.isLoading);

  const filteredSelectedDeviceRows = useMemo(() => {
    const keyword = selectedDeviceKeyword.trim().toLowerCase();
    if (!keyword) {
      return selectedDeviceRows;
    }

    return selectedDeviceRows.filter((device) =>
      device.sn.toLowerCase().includes(keyword) ||
      (device.name ?? '').toLowerCase().includes(keyword),
    );
  }, [selectedDeviceKeyword, selectedDeviceRows]);

  const alarmLibrary = useMemo<AlarmLibraryItem[]>(
    () => (alarmDefinitionData?.items || []).map(mapAlarmDefinitionToLibraryItem),
    [alarmDefinitionData]
  );

  // 过滤告警库
  const filteredAlarms = useMemo<AlarmLibraryItem[]>(() => {
    let result: AlarmLibraryItem[] = alarmLibrary;
    if (alarmFilter.keyword) {
      const kw = alarmFilter.keyword.toLowerCase();
      result = result.filter((alarm) =>
        alarm.alarmIdentifier.toLowerCase().includes(kw) ||
        alarm.alarmName.toLowerCase().includes(kw)
      );
    }
    if (alarmFilter.eventType) {
      result = result.filter((alarm) => alarm.eventType === alarmFilter.eventType);
    }
    if (alarmFilter.severity) {
      result = result.filter((alarm) => alarm.severity === alarmFilter.severity);
    }

    const selectedAlarmSet = new Set(selectedAlarms);
    return sortSelectedFirst(result, (alarm) => selectedAlarmSet.has(alarm.alarmIdentifier));
  }, [alarmFilter, alarmLibrary, selectedAlarms]);

  const selectedAlarmItems = useMemo<AlarmLibraryItem[]>(() => {
    const alarmMap = new Map(alarmLibrary.map((alarm) => [alarm.alarmIdentifier, alarm]));
    return selectedAlarms
      .map((alarmIdentifier) => alarmMap.get(alarmIdentifier))
      .filter(isAlarmLibraryItem);
  }, [alarmLibrary, selectedAlarms]);

  const visibleSelectedAlarmItems = useMemo(
    () => selectedAlarmItems.slice(0, MAX_VISIBLE_SELECTED_ALARMS),
    [selectedAlarmItems]
  );

  const hiddenSelectedAlarmItems = useMemo(
    () => selectedAlarmItems.slice(MAX_VISIBLE_SELECTED_ALARMS),
    [selectedAlarmItems]
  );

  const handleSubmit = useCallback(async () => {
    // 验证告警必填
    if (selectedAlarms.length === 0) {
      setAlarmError(t('alarm.selectAtLeastOne'));
      return;
    }
    setAlarmError(null);

    try {
      const values = await form.validateFields();
      setLoading(true);
      await onSubmit({
        ...values,
        deviceSelectionMode,
        selectedDevices,
        selectedGroups,
        selectedAlarms,
        timeRange: timeRange
          ? [timeRange[0].toISOString(), timeRange[1].toISOString()]
          : undefined,
      });
      message.success(t('common.operationSuccess'));
      onClose();
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) {
        return;
      }
      console.error('Submit failed:', error);
      message.error(t('common.operationFailed'));
    } finally {
      setLoading(false);
    }
  }, [form, message, onSubmit, deviceSelectionMode, selectedDevices, selectedGroups, selectedAlarms, timeRange, t, onClose]);

  // 规则名称验证器
  const validateRuleName = useCallback((_: unknown, value: string) => {
    if (!value || !value.trim()) {
      return Promise.reject(new Error(t('filter.enterField', { label: t('alarm.ruleName') })));
    }
    if (value.length > 100) {
      return Promise.reject(new Error(t('alarm.ruleNameMax100')));
    }
    const currentName = rule?.ruleName;
    const isDuplicate = existingNames.some(name => name !== currentName && name === value.trim());
    if (isDuplicate) {
      return Promise.reject(new Error(t('alarm.ruleNameDuplicate')));
    }
    return Promise.resolve();
  }, [existingNames, rule?.ruleName, t]);

  const validateEmailRecipients = useCallback((_: unknown, values: string[] | undefined) => {
    if (currentRuleType !== 'notify_email') {
      return Promise.resolve();
    }
    const recipients = (values || []).map((value) => value.trim()).filter(Boolean);
    if (recipients.length === 0) {
      return Promise.reject(new Error(t('alarm.rule.emailRecipientsRequired')));
    }
    if (recipients.length > MAX_EMAIL_RECIPIENTS) {
      return Promise.reject(new Error(t('alarm.rule.emailRecipientsMax', { max: MAX_EMAIL_RECIPIENTS })));
    }
    if (recipients.some((recipient) => !EMAIL_ADDRESS_PATTERN.test(recipient))) {
      return Promise.reject(new Error(t('alarm.rule.emailRecipientsInvalid')));
    }
    return Promise.resolve();
  }, [currentRuleType, t]);

  // 设备列表列配置
  const deviceColumns: TableProps<DeviceWithType>['columns'] = [
    {
      title: t('device.sn'),
      dataIndex: 'sn',
      width: 140,
      ellipsis: true,
    },
    {
      title: t('alarm.deviceType'),
      dataIndex: 'deviceType',
      width: 80,
      render: (deviceType: 'eNB' | 'gNB' | 'GSM') => {
        const colorMap = { eNB: 'blue', gNB: 'green', GSM: 'orange' };
        return <Tag color={colorMap[deviceType]}>{deviceType}</Tag>;
      },
    },
    {
      title: t('device.connStatus'),
      dataIndex: 'connStatus',
      width: 80,
      render: (status: string) => (
        <Tag color={status === 'online' ? 'green' : 'default'} style={{ margin: 0 }}>
          {status === 'online' ? t('device.online') : t('device.offline')}
        </Tag>
      ),
    },
  ];

  const selectedDeviceColumns: TableProps<SelectedDeviceRow>['columns'] = [
    {
      title: t('device.sn'),
      dataIndex: 'sn',
      width: 160,
      ellipsis: true,
    },
    {
      title: t('device.name'),
      dataIndex: 'name',
      ellipsis: true,
    },
    {
      title: t('alarm.deviceType'),
      dataIndex: 'deviceType',
      width: 80,
      render: (deviceType: 'eNB' | 'gNB' | 'GSM') => {
        const colorMap = { eNB: 'blue', gNB: 'green', GSM: 'orange' };
        return <Tag color={colorMap[deviceType]}>{deviceType}</Tag>;
      },
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 80,
      render: (_, record) => isViewMode ? null : (
        <Button
          type="link"
          size="small"
          onClick={() => setSelectedDevices((previousKeys) => previousKeys.filter((key) => key !== record.id))}
        >
          {t('common.delete')}
        </Button>
      ),
    },
  ];

  // 设备组列表列配置
  const groupColumns: TableProps<DeviceGroupWithLevel>['columns'] = [
    {
      title: t('device.groupName'),
      dataIndex: 'fullName',
      ellipsis: true,
      render: (_fullName: string, record) => (
        <span style={{ paddingLeft: record.level * 20 }}>
          {record.level > 0 && <span style={{ color: '#999' }}>└ </span>}
          {record.name}
        </span>
      ),
    },
    {
      title: t('device.count.total'),
      dataIndex: 'deviceCount',
      width: 100,
    },
  ];

  // 告警库列表列配置
  const alarmColumns: TableProps<AlarmLibraryItem>['columns'] = [
    {
      title: t('alarm.alarmIdentifier'),
      dataIndex: 'alarmIdentifier',
      width: 120,
    },
    {
      title: t('alarm.possibleCause'),
      dataIndex: 'alarmName',
      ellipsis: true,
    },
    {
      title: t('alarm.eventType'),
      dataIndex: 'eventType',
      width: 120,
      render: (type) => {
        const opt = EVENT_TYPE_OPTIONS.find(o => o.value === type);
        return opt ? t(opt.labelKey) : type;
      },
    },
    {
      title: t('alarm.severity'),
      dataIndex: 'severity',
      width: 80,
      render: (severity) => {
        const config = SEVERITY_OPTIONS.find(o => o.value === severity);
        return <Tag color={config?.color} style={{ margin: 0 }}>{config ? t(config.labelKey) : severity}</Tag>;
      },
    },
  ];

  const filteredDeviceKeys = useMemo(() => filteredDevices.map((device) => device.id), [filteredDevices]);
  const filteredAlarmKeys = useMemo(
    () => filteredAlarms.map((alarm) => alarm.alarmIdentifier),
    [filteredAlarms],
  );

  const filteredSelectedDeviceCount = useMemo(() => {
    const selectedDeviceSet = new Set(selectedDevices);
    return filteredDeviceKeys.filter((key) => selectedDeviceSet.has(key)).length;
  }, [filteredDeviceKeys, selectedDevices]);

  const filteredSelectedAlarmCount = useMemo(() => {
    const selectedAlarmSet = new Set(selectedAlarms);
    return filteredAlarmKeys.filter((key) => selectedAlarmSet.has(key)).length;
  }, [filteredAlarmKeys, selectedAlarms]);

  // 表格行选择配置
  const deviceRowSelection = {
    selectedRowKeys: selectedDevices,
    preserveSelectedRowKeys: true,
    onChange: (keys: React.Key[]) => {
      setSelectedDevices((previousKeys) => mergeVisibleSelection(previousKeys, filteredDeviceKeys, keys as string[]));
    },
    columnWidth: 40,
  };

  const groupRowSelection = {
    selectedRowKeys: selectedGroups,
    onChange: (keys: React.Key[]) => setSelectedGroups(keys as string[]),
    columnWidth: 40,
  };

  const alarmRowSelection: TableProps<AlarmLibraryItem>['rowSelection'] = {
    selectedRowKeys: selectedAlarms,
    preserveSelectedRowKeys: true,
    onChange: (keys: React.Key[]) => {
      setSelectedAlarms((previousKeys) => mergeVisibleSelection(previousKeys, filteredAlarmKeys, keys as string[]));
      setAlarmError(null);
    },
    columnWidth: 40,
  };

  // 设备全选/取消全选
  const handleDeviceSelectAll = useCallback((checked: boolean) => {
    setSelectedDevices((previousKeys) => toggleVisibleSelection(previousKeys, filteredDeviceKeys, checked));
  }, [filteredDeviceKeys]);

  // 设备组全选/取消全选
  const handleGroupSelectAll = useCallback((checked: boolean) => {
    if (checked) {
      setSelectedGroups(groupsWithLevel.map(g => g.id));
    } else {
      setSelectedGroups([]);
    }
  }, [groupsWithLevel]);

  // 告警全选/取消全选
  const handleAlarmSelectAll = useCallback((checked: boolean) => {
    setSelectedAlarms((previousKeys) => toggleVisibleSelection(previousKeys, filteredAlarmKeys, checked));
    setAlarmError(null);
  }, [filteredAlarmKeys]);

  // 计算全选状态
  const isAllDevicesSelected = filteredDevices.length > 0 && filteredSelectedDeviceCount === filteredDevices.length;
  const isAllGroupsSelected = groupsWithLevel.length > 0 && selectedGroups.length === groupsWithLevel.length;
  const isAllAlarmsSelected = filteredAlarms.length > 0 && filteredSelectedAlarmCount === filteredAlarms.length;
  const isIndeterminateDevices = filteredSelectedDeviceCount > 0 && filteredSelectedDeviceCount < filteredDevices.length;
  const isIndeterminateGroups = selectedGroups.length > 0 && selectedGroups.length < groupsWithLevel.length;
  const isIndeterminateAlarms = filteredSelectedAlarmCount > 0 && filteredSelectedAlarmCount < filteredAlarms.length;

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      size="large"
      destroyOnHidden
      footer={
        isViewMode ? null : (
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={onClose}>{t('common.cancel')}</Button>
            <Button type="primary" loading={loading} onClick={() => void handleSubmit()}>
              {t('common.confirm')}
            </Button>
          </div>
        )
      }
    >
      <Form
        form={form}
        layout="vertical"
        disabled={isViewMode}
        initialValues={{
          ruleName: rule?.ruleName,
          status: rule?.enabled ?? true,
          ruleType: rule?.ruleType,
          emailRecipients: rule?.emailRecipients ?? [],
        }}
      >
        <Form.Item
          name="ruleName"
          label={t('alarm.ruleName')}
          rules={[
            { required: true, message: t('filter.enterField', { label: t('alarm.ruleName') }) },
            { validator: validateRuleName },
          ]}
          validateTrigger="onBlur"
        >
          <Input placeholder={t('filter.enterField', { label: t('alarm.ruleName') })} maxLength={100} showCount />
        </Form.Item>

        <Form.Item
          name="status"
          label={t('alarm.status')}
          rules={[{ required: true }]}
        >
          <Radio.Group>
            <Radio value={true}>{t('status.enabled')}</Radio>
            <Radio value={false}>{t('status.disabled')}</Radio>
          </Radio.Group>
        </Form.Item>

        <Form.Item
          name="ruleType"
          label={t('alarm.ruleType')}
          rules={[{ required: true }]}
        >
          <Select
            options={ruleTypeOptions}
            placeholder={t('filter.selectField', { label: t('alarm.ruleType') })}
            onChange={setCurrentRuleType}
          />
        </Form.Item>

        <Form.Item
          name="emailRecipients"
          label={t('alarm.rule.emailRecipients')}
          extra={t('alarm.rule.emailRecipientsHint')}
          hidden={currentRuleType !== 'notify_email'}
          rules={[{ validator: validateEmailRecipients }]}
        >
          <Select
            mode="tags"
            tokenSeparators={[',', ';', ' ']}
            placeholder={t('alarm.rule.emailRecipientsPlaceholder')}
            maxTagCount="responsive"
          />
        </Form.Item>

        {/* 设备选择方式 */}
        <Form.Item label={t('alarm.deviceSelection')}>
          <Space orientation="vertical" style={{ width: '100%' }} size="small">
            <Radio.Group
              value={deviceSelectionMode}
              onChange={(e) => setDeviceSelectionMode(e.target.value)}
              optionType="button"
              buttonStyle="solid"
              size="small"
            >
              <Radio.Button value="devices">{t('device.sn')}</Radio.Button>
              <Radio.Button value="groups">{t('device.groupName')}</Radio.Button>
            </Radio.Group>

            {deviceSelectionMode === 'devices' ? (
              <>
                {/* 设备类型筛选 + 搜索 + 选择摘要 */}
                <Space wrap size="small" style={{ width: '100%', justifyContent: 'space-between' }}>
                  <Space wrap size="small">
                    <Checkbox.Group
                      options={DEVICE_TYPE_OPTIONS}
                      value={deviceFilter.deviceTypes}
                      onChange={(values) => {
                        setDeviceFilter(prev => ({ ...prev, deviceTypes: values as string[] }));
                        setDeviceTablePage(1);
                      }}
                    />
                    <Input.Search
                      placeholder={t('alarm.searchDeviceSnPlaceholder')}
                      style={{ width: 200 }}
                      value={deviceFilter.snKeyword}
                      onChange={(e) => {
                        setDeviceFilter(prev => ({ ...prev, snKeyword: e.target.value }));
                        setDeviceTablePage(1);
                      }}
                      allowClear
                      size="small"
                    />
                  </Space>
                  <Space wrap size="small">
                    <Checkbox
                      checked={isAllDevicesSelected}
                      indeterminate={isIndeterminateDevices}
                      onChange={(e) => handleDeviceSelectAll(e.target.checked)}
                      disabled={isViewMode || filteredDevices.length === 0}
                    >
                      {t('alarm.rule.selectCurrentPageDevices')}
                    </Checkbox>
                    <Button
                      type="link"
                      size="small"
                      style={{ padding: 0 }}
                      disabled={selectedDevices.length === 0}
                      onClick={() => setSelectedDeviceModalOpen(true)}
                    >
                      {t('alarm.rule.selectedDeviceCount', { count: selectedDevices.length })}
                    </Button>
                  </Space>
                </Space>
                <Table
                  rowSelection={deviceRowSelection}
                  columns={deviceColumns}
                  dataSource={filteredDevices}
                  rowKey="id"
                  size="small"
                  loading={deviceLoading}
                  pagination={{
                    current: deviceTablePage,
                    pageSize: deviceTablePageSize,
                    total: deviceTableTotal,
                    size: 'small',
                    showSizeChanger: true,
                    showTotal: (total) => t('alarm.rule.deviceTotal', { count: total }),
                    pageSizeOptions: [5, 10, 20, 50],
                    onChange: (page, nextPageSize) => {
                      setDeviceTablePage(page);
                      setDeviceTablePageSize(nextPageSize);
                    },
                    onShowSizeChange: (_current, nextPageSize) => {
                      setDeviceTablePage(1);
                      setDeviceTablePageSize(nextPageSize);
                    },
                  }}
                  scroll={{ y: 180 }}
                />
              </>
            ) : (
              <>
                {/* 设备组全选 */}
                <Checkbox
                  checked={isAllGroupsSelected}
                  indeterminate={isIndeterminateGroups}
                  onChange={(e) => handleGroupSelectAll(e.target.checked)}
                  disabled={isViewMode || groupsWithLevel.length === 0}
                >
                  {t('common.selectAll')}
                </Checkbox>
                <Table
                  rowSelection={groupRowSelection}
                  columns={groupColumns}
                  dataSource={groupsWithLevel}
                  rowKey="id"
                  size="small"
                  loading={groupsLoading}
                  pagination={{ pageSize: 5, size: 'small', showSizeChanger: false }}
                  scroll={{ y: 180 }}
                />
                {selectedGroups.length > 0 && (
                  <div style={{ color: token.colorTextSecondary, fontSize: token.fontSizeSM }}>
                    {t('table.selected', { count: selectedGroups.length })}
                  </div>
                )}
              </>
            )}
          </Space>
        </Form.Item>

        {/* 告警选择 */}
        <Form.Item
          label={t('alarm.filter.title')}
          required
          validateStatus={alarmError ? 'error' : ''}
          help={alarmError}
        >
          <Space orientation="vertical" style={{ width: '100%' }} size="small">
            <Space wrap size="small" style={{ width: '100%', justifyContent: 'space-between' }}>
              <Space wrap size="small">
                <Input.Search
                  placeholder={t('alarm.librarySearchPlaceholder')}
                  style={{ width: 220 }}
                  onChange={(e) => {
                    setAlarmFilter(prev => ({ ...prev, keyword: e.target.value }));
                    setAlarmTablePage(1);
                  }}
                  allowClear
                  size="small"
                />
                <Select
                  placeholder={t('alarm.eventType')}
                  style={{ width: 130 }}
                  options={eventTypeOptions}
                  onChange={(v) => {
                    setAlarmFilter(prev => ({ ...prev, eventType: v }));
                    setAlarmTablePage(1);
                  }}
                  allowClear
                  size="small"
                />
                <Select
                  placeholder={t('alarm.severity')}
                  style={{ width: 90 }}
                  options={severityOptions}
                  onChange={(v) => {
                    setAlarmFilter(prev => ({ ...prev, severity: v }));
                    setAlarmTablePage(1);
                  }}
                  allowClear
                  size="small"
                />
              </Space>
              <Space wrap size="small">
                <Typography.Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
                  {t('alarm.rule.selectedAlarmCount', { count: selectedAlarms.length })}
                </Typography.Text>
                <Checkbox
                  checked={isAllAlarmsSelected}
                  indeterminate={isIndeterminateAlarms}
                  onChange={(e) => handleAlarmSelectAll(e.target.checked)}
                  disabled={isViewMode || filteredAlarms.length === 0}
                >
                    {t('alarm.rule.selectFilteredAlarms')}
                </Checkbox>
              </Space>
            </Space>
            {selectedAlarmItems.length > 0 && (
              <div
                data-testid="selected-alarm-summary"
                style={{
                  padding: token.paddingSM,
                  border: `1px solid ${token.colorBorderSecondary}`,
                  borderRadius: token.borderRadiusLG,
                  background: token.colorFillAlter,
                }}
              >
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: token.marginSM,
                  marginBottom: token.marginXS,
                }}>
                  <Typography.Text strong style={{ fontSize: token.fontSizeSM }}>
                    {t('alarm.rule.selectedAlarmIds')}
                  </Typography.Text>
                  {!isViewMode && (
                    <Button
                      type="link"
                      size="small"
                      style={{ height: 'auto', padding: 0 }}
                      onClick={() => setSelectedAlarms([])}
                    >
                      {t('alarm.clearAll')}
                    </Button>
                  )}
                </div>
                <Space wrap size={[4, 8]}>
                  {visibleSelectedAlarmItems.map((alarm) => (
                    <Tooltip key={alarm.alarmIdentifier} title={alarm.alarmName}>
                      <Tag
                        closable={!isViewMode}
                        onClose={() => {
                          setSelectedAlarms((previousKeys) => previousKeys.filter((key) => key !== alarm.alarmIdentifier));
                        }}
                        style={{ marginInlineEnd: 0 }}
                      >
                        {alarm.alarmIdentifier}
                      </Tag>
                    </Tooltip>
                  ))}
                  {hiddenSelectedAlarmItems.length > 0 && (
                    <Tooltip title={hiddenSelectedAlarmItems.map((alarm) => alarm.alarmIdentifier).join(', ')}>
                      <Tag style={{ marginInlineEnd: 0 }}>{`+${hiddenSelectedAlarmItems.length}...`}</Tag>
                    </Tooltip>
                  )}
                </Space>
              </div>
            )}
            <Table<AlarmLibraryItem>
              rowSelection={alarmRowSelection}
              columns={alarmColumns}
              dataSource={filteredAlarms}
              rowKey="alarmIdentifier"
              loading={alarmDefinitionLoading}
              size="small"
              pagination={{
                current: alarmTablePage,
                pageSize: alarmTablePageSize,
                size: 'small',
                showSizeChanger: true,
                showTotal: (total) => t('alarm.rule.alarmTotal', { count: total }),
                pageSizeOptions: ['5', '10', '20', '50', '100'],
                onChange: (page, size) => {
                  setAlarmTablePage(page);
                  setAlarmTablePageSize(size);
                },
                onShowSizeChange: (page, size) => {
                  setAlarmTablePage(page);
                  setAlarmTablePageSize(size);
                },
              }}
              scroll={{ y: 180 }}
            />
          </Space>
        </Form.Item>

        {/* 故障时间范围 */}
        <Form.Item label={t('alarm.filter.timeRange')}>
          <RangePicker
            showTime={{ format: 'HH:mm:ss' }}
            format="YYYY-MM-DD HH:mm:ss"
            value={timeRange}
            onChange={(dates) => setTimeRange(dates as [Dayjs, Dayjs] | null)}
            style={{ width: '100%' }}
            placeholder={[t('dateRange.start'), t('dateRange.end')]}
            size="small"
            allowClear
          />
        </Form.Item>
      </Form>
      {selectedDeviceModalOpen && (
        <Modal
          title={t('alarm.rule.selectedDevicesTitle', { count: selectedDevices.length })}
          open={selectedDeviceModalOpen}
          onCancel={() => {
            setSelectedDeviceModalOpen(false);
            setSelectedDeviceKeyword('');
          }}
          footer={<Button type="primary" onClick={() => {
            setSelectedDeviceModalOpen(false);
            setSelectedDeviceKeyword('');
          }}>{t('common.confirm')}</Button>}
          width={640}
          destroyOnHidden
        >
          <Space orientation="vertical" size={12} style={{ width: '100%', marginBottom: 12 }}>
            <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
              <Input.Search
                placeholder={t('alarm.searchDeviceSnPlaceholder')}
                allowClear
                style={{ width: 240 }}
                value={selectedDeviceKeyword}
                onChange={(e) => setSelectedDeviceKeyword(e.target.value)}
              />
              <Space size={16} style={{ color: 'rgba(0,0,0,0.45)', fontSize: 12 }}>
                <span>{t('alarm.rule.selectedDeviceCount', { count: selectedDevices.length })}</span>
                <span>{t('alarm.rule.deviceTotal', { count: selectedDeviceRows.length })}</span>
              </Space>
            </Space>
          </Space>
          <Table<SelectedDeviceRow>
            columns={selectedDeviceColumns}
            dataSource={filteredSelectedDeviceRows}
            rowKey="id"
            size="small"
            loading={selectedDeviceLoading}
            pagination={{
              pageSize: 8,
              size: 'small',
              showSizeChanger: false,
              showTotal: (total) => t('alarm.rule.deviceTotal', { count: total }),
            }}
          />
        </Modal>
      )}
    </Drawer>
  );
}
