import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  DatePicker,
  Drawer,
  Form,
  Input,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  message,
} from 'antd';
import type { TableProps } from 'antd';
import { useT } from '@/hooks/useT';
import { useDeviceList, useDeviceGroups } from '@/hooks/api/useDevices';
import type { AlarmRule } from '@/types/alarm';
import type { AlarmSeverity } from '@/types/common';
import type { Device, DeviceGroup } from '@/types/device';
import dayjs, { Dayjs } from 'dayjs';

const { RangePicker } = DatePicker;

// 执行动作配置
const RULE_TYPE_OPTIONS = [
  { value: '1', label: '不入库不显示' },
  { value: '3', label: '自动确认' },
];

// 事件类型配置
const EVENT_TYPE_OPTIONS = [
  { value: '30000', label: '通信告警' },
  { value: '30001', label: '服务质量告警' },
  { value: '30002', label: '处理失败告警' },
  { value: '30003', label: '设备告警' },
  { value: '30004', label: '环境告警' },
  { value: '30006', label: '性能溢出告警' },
];

// 告警级别配置
const SEVERITY_OPTIONS: { value: AlarmSeverity; label: string; color: string }[] = [
  { value: 'critical', label: '紧急', color: 'red' },
  { value: 'major', label: '重要', color: 'orange' },
  { value: 'minor', label: '次要', color: 'blue' },
  { value: 'warning', label: '警告', color: 'gold' },
];

// 设备类型配置
const DEVICE_TYPE_OPTIONS = [
  { label: 'eNB', value: 'eNB' },
  { label: 'gNB', value: 'gNB' },
  { label: 'GSM', value: 'GSM' },
];

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
  timeRange?: [string, string];
}

// Mock 告警库数据
const mockAlarmLibrary = [
  { id: 'alarm-001', alarmIdentifier: 'A0001', alarmName: '小区不可用', eventType: '30003', severity: 'critical' as const },
  { id: 'alarm-002', alarmIdentifier: 'A0002', alarmName: 'S1链路中断', eventType: '30000', severity: 'major' as const },
  { id: 'alarm-003', alarmIdentifier: 'A0003', alarmName: 'X2链路中断', eventType: '30000', severity: 'major' as const },
  { id: 'alarm-004', alarmIdentifier: 'A0004', alarmName: '温度过高', eventType: '30004', severity: 'warning' as const },
  { id: 'alarm-005', alarmIdentifier: 'A0005', alarmName: 'gNB射频异常', eventType: '30003', severity: 'critical' as const },
  { id: 'alarm-006', alarmIdentifier: 'A0006', alarmName: 'gNB时钟失锁', eventType: '30003', severity: 'major' as const },
  { id: 'alarm-007', alarmIdentifier: 'A0007', alarmName: 'GSM功率异常', eventType: '30003', severity: 'warning' as const },
  { id: 'alarm-008', alarmIdentifier: 'A0008', alarmName: 'GSM链路告警', eventType: '30000', severity: 'minor' as const },
  { id: 'alarm-009', alarmIdentifier: 'A0009', alarmName: '电源电压异常', eventType: '30004', severity: 'critical' as const },
  { id: 'alarm-010', alarmIdentifier: 'A0010', alarmName: '风扇故障', eventType: '30003', severity: 'warning' as const },
];

// 设备数据类型（包含设备类型字段）
interface DeviceWithType extends Device {
  deviceType: 'eNB' | 'gNB' | 'GSM';
}

// 设备组数据类型（包含层级结构）
interface DeviceGroupWithLevel extends DeviceGroup {
  level: number;
  fullName: string;
}

export default function AlarmRuleDrawer({ open, mode, rule, existingNames = [], onClose, onSubmit }: AlarmRuleDrawerProps) {
  const t = useT();
  const [form] = Form.useForm<AlarmRuleFormData>();
  const [loading, setLoading] = useState(false);
  const [deviceSelectionMode, setDeviceSelectionMode] = useState<'devices' | 'groups'>('devices');
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  const [selectedGroups, setSelectedGroups] = useState<string[]>([]);
  const [selectedAlarms, setSelectedAlarms] = useState<string[]>([]);
  const [timeRange, setTimeRange] = useState<[Dayjs, Dayjs] | null>(null);
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

  // 获取设备列表
  const { data: deviceData, isLoading: deviceLoading } = useDeviceList({
    page: 1,
    pageSize: 1000,
  });

  // 获取设备组列表
  const { data: groupsData, isLoading: groupsLoading } = useDeviceGroups();

  const isViewMode = mode === 'view';
  const title = mode === 'add' ? t('common.add') : mode === 'edit' ? t('common.edit') : t('common.detail');

  // 处理设备数据，添加设备类型
  const devicesWithType: DeviceWithType[] = useMemo(() => {
    const devices = deviceData?.items || [];
    return devices.map(device => {
      // 根据设备名称或网络类型判断设备类型
      let deviceType: 'eNB' | 'gNB' | 'GSM' = 'eNB';
      const name = device.name?.toLowerCase() || '';
      const networkType = device.networkType?.toLowerCase() || '';

      if (name.includes('gnb') || networkType.includes('5g') || networkType.includes('nr')) {
        deviceType = 'gNB';
      } else if (name.includes('gsm') || networkType.includes('gsm')) {
        deviceType = 'GSM';
      }

      return { ...device, deviceType };
    });
  }, [deviceData]);

  // 处理设备组数据，构建层级结构
  const groupsWithLevel: DeviceGroupWithLevel[] = useMemo(() => {
    const groups = groupsData || [];
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

  // 根据筛选条件过滤设备
  const filteredDevices = useMemo(() => {
    let result = devicesWithType;

    // 按设备类型筛选
    if (deviceFilter.deviceTypes.length > 0) {
      result = result.filter(d => deviceFilter.deviceTypes.includes(d.deviceType));
    }

    // 按SN搜索
    if (deviceFilter.snKeyword) {
      const kw = deviceFilter.snKeyword.toLowerCase();
      result = result.filter(d =>
        d.sn?.toLowerCase().includes(kw) ||
        d.name?.toLowerCase().includes(kw)
      );
    }

    return result;
  }, [devicesWithType, deviceFilter]);

  // 初始化表单数据
  useEffect(() => {
    if (open && rule) {
      form.setFieldsValue({
        ruleName: rule.ruleName,
        status: rule.enabled,
        ruleType: rule.ruleType,
      });
      setSelectedDevices([]);
      setSelectedGroups([]);
      setSelectedAlarms([]);
      setTimeRange(null);
    } else if (open) {
      form.resetFields();
      setSelectedDevices([]);
      setSelectedGroups([]);
      setSelectedAlarms([]);
      setTimeRange(null);
    }
    setAlarmError(null);
    setDeviceFilter({ deviceTypes: [], snKeyword: '' });
  }, [open, rule, form]);

  // 过滤告警库
  const filteredAlarms = useMemo(() => {
    let result = mockAlarmLibrary;
    if (alarmFilter.keyword) {
      const kw = alarmFilter.keyword.toLowerCase();
      result = result.filter(a =>
        a.alarmIdentifier.toLowerCase().includes(kw) ||
        a.alarmName.toLowerCase().includes(kw)
      );
    }
    if (alarmFilter.eventType) {
      result = result.filter(a => a.eventType === alarmFilter.eventType);
    }
    if (alarmFilter.severity) {
      result = result.filter(a => a.severity === alarmFilter.severity);
    }
    return result;
  }, [alarmFilter]);

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
  }, [form, onSubmit, deviceSelectionMode, selectedDevices, selectedGroups, selectedAlarms, timeRange, t, onClose]);

  // 规则名称验证器
  const validateRuleName = useCallback((_: unknown, value: string) => {
    if (!value || !value.trim()) {
      return Promise.reject(new Error(t('filter.enterField').replace('{label}', t('alarm.ruleName'))));
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

  // 设备列表列配置
  const deviceColumns: TableProps<DeviceWithType>['columns'] = [
    {
      title: t('device.sn'),
      dataIndex: 'sn',
      width: 140,
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
  const alarmColumns: TableProps<typeof mockAlarmLibrary[0]>['columns'] = [
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
      render: (type) => EVENT_TYPE_OPTIONS.find(o => o.value === type)?.label || type,
    },
    {
      title: t('alarm.severity'),
      dataIndex: 'severity',
      width: 80,
      render: (severity) => {
        const config = SEVERITY_OPTIONS.find(o => o.value === severity);
        return <Tag color={config?.color} style={{ margin: 0 }}>{config?.label || severity}</Tag>;
      },
    },
  ];

  // 表格行选择配置
  const deviceRowSelection = {
    selectedRowKeys: selectedDevices,
    onChange: (keys: React.Key[]) => setSelectedDevices(keys as string[]),
    columnWidth: 40,
  };

  const groupRowSelection = {
    selectedRowKeys: selectedGroups,
    onChange: (keys: React.Key[]) => setSelectedGroups(keys as string[]),
    columnWidth: 40,
  };

  const alarmRowSelection = {
    selectedRowKeys: selectedAlarms,
    onChange: (keys: React.Key[]) => {
      setSelectedAlarms(keys as string[]);
      setAlarmError(null);
    },
    columnWidth: 40,
  };

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      width={720}
      destroyOnClose
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
        initialValues={{ status: true, ruleType: '1' }}
      >
        <Form.Item
          name="ruleName"
          label={t('alarm.ruleName')}
          rules={[{ validator: validateRuleName }]}
          validateTrigger="onBlur"
        >
          <Input placeholder={t('filter.enterField').replace('{label}', t('alarm.ruleName'))} maxLength={100} showCount />
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
          <Select options={RULE_TYPE_OPTIONS} placeholder={t('filter.selectField').replace('{label}', t('alarm.ruleType'))} />
        </Form.Item>

        {/* 设备选择方式 */}
        <Form.Item label={t('alarm.deviceSelection')}>
          <Space direction="vertical" style={{ width: '100%' }} size="small">
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
                {/* 设备类型筛选 + SN搜索 */}
                <Space wrap size="small">
                  <Checkbox.Group
                    options={DEVICE_TYPE_OPTIONS}
                    value={deviceFilter.deviceTypes}
                    onChange={(values) => setDeviceFilter(prev => ({ ...prev, deviceTypes: values as string[] }))}
                  />
                  <Input.Search
                    placeholder={t('alarm.searchDeviceSnPlaceholder')}
                    style={{ width: 200 }}
                    value={deviceFilter.snKeyword}
                    onChange={(e) => setDeviceFilter(prev => ({ ...prev, snKeyword: e.target.value }))}
                    allowClear
                    size="small"
                  />
                </Space>
                <Table
                  rowSelection={deviceRowSelection}
                  columns={deviceColumns}
                  dataSource={filteredDevices}
                  rowKey="id"
                  size="small"
                  loading={deviceLoading}
                  pagination={{ pageSize: 5, size: 'small', showSizeChanger: false }}
                  scroll={{ y: 180 }}
                />
              </>
            ) : (
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
            )}
          </Space>
        </Form.Item>

        {/* 告警选择 */}
        <Form.Item
          label={
            <span>
              {t('alarm.filter.title')}
              <span style={{ color: '#ff4d4f', marginLeft: 4 }}>*</span>
            </span>
          }
          validateStatus={alarmError ? 'error' : ''}
          help={alarmError}
        >
          <Space direction="vertical" style={{ width: '100%' }} size="small">
            <Space wrap size="small">
              <Input.Search
                placeholder={t('alarm.librarySearchPlaceholder')}
                style={{ width: 200 }}
                onChange={(e) => setAlarmFilter(prev => ({ ...prev, keyword: e.target.value }))}
                allowClear
                size="small"
              />
              <Select
                placeholder={t('alarm.eventType')}
                style={{ width: 130 }}
                options={EVENT_TYPE_OPTIONS}
                onChange={(v) => setAlarmFilter(prev => ({ ...prev, eventType: v }))}
                allowClear
                size="small"
              />
              <Select
                placeholder={t('alarm.severity')}
                style={{ width: 90 }}
                options={SEVERITY_OPTIONS.map(s => ({ value: s.value, label: s.label }))}
                onChange={(v) => setAlarmFilter(prev => ({ ...prev, severity: v }))}
                allowClear
                size="small"
              />
            </Space>
            <Table
              rowSelection={alarmRowSelection}
              columns={alarmColumns}
              dataSource={filteredAlarms}
              rowKey="id"
              size="small"
              pagination={{ pageSize: 5, size: 'small' }}
              scroll={{ y: 180 }}
            />
            {selectedAlarms.length > 0 && (
              <div style={{ color: 'rgba(0,0,0,0.45)', fontSize: 12 }}>
                {t('table.selected', { count: selectedAlarms.length })}
              </div>
            )}
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
    </Drawer>
  );
}
