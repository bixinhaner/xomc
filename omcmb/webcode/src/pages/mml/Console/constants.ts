import type { ConsoleDevice } from './types';
import type { MMLCommand } from '@/types/mml';

// 模拟设备列表
export const DEVICE_LIST: ConsoleDevice[] = [
  { sn: 'ENB00001', name: '北京朝阳基站01', type: 'eNB', productType: 'PM-B4860', status: 'online' },
  { sn: 'ENB00002', name: '北京海淀基站01', type: 'eNB', productType: 'PM-B4860', status: 'online' },
  { sn: 'ENB00003', name: '上海浦东基站01', type: 'eNB', productType: 'QAFA', status: 'alarm' },
  { sn: 'ENB00004', name: '上海徐汇基站01', type: 'eNB', productType: 'PM-B4860', status: 'online' },
  { sn: 'ENB00005', name: '广州天河基站01', type: 'eNB', productType: 'QAFA', status: 'online' },
  { sn: 'ENB00006', name: '深圳南山基站01', type: 'eNB', productType: 'PM-B4860', status: 'offline' },
  { sn: 'ENB00007', name: '杭州西湖基站01', type: 'eNB', productType: 'BaiBNX', status: 'online' },
  { sn: 'ENB00008', name: '南京鼓楼基站01', type: 'eNB', productType: 'PM-B4860', status: 'alarm' },
  { sn: 'GNB00001', name: '北京5G基站01', type: 'gNB', productType: 'BaiBS5163', status: 'online' },
  { sn: 'GNB00002', name: '北京5G基站02', type: 'gNB', productType: 'BaiBS5163', status: 'offline' },
  { sn: 'GNB00003', name: '上海5G基站01', type: 'gNB', productType: 'BaiBS5263', status: 'online' },
  { sn: 'GNB00004', name: '广州5G基站01', type: 'gNB', productType: 'BaiBS5163', status: 'online' },
  { sn: 'GNB00005', name: '深圳5G基站01', type: 'gNB', productType: 'BaiBS5263', status: 'alarm' },
  { sn: 'GSM00001', name: '北京GSM基站01', type: 'GSM', productType: 'BTS', status: 'online' },
  { sn: 'GSM00002', name: '上海GSM基站01', type: 'GSM', productType: 'BTS', status: 'offline' },
  { sn: 'GSM00003', name: '广州GSM基站01', type: 'GSM', productType: 'BSC', status: 'online' },
];

// 模拟命令列表
export const MOCK_COMMANDS: MMLCommand[] = [
  // 总览
  {
    id: '1',
    commandName: '基本信息',
    commandCode: 'LST BASIC_INFO',
    category: '总览',
    description: '查询设备基本信息',
    params: [],
    productTypes: ['eNB', 'gNB', 'GSM'],
  },
  {
    id: '2',
    commandName: '状态信息',
    commandCode: 'LST STATUS_INFO',
    category: '总览',
    description: '查询设备状态信息',
    params: [],
    productTypes: ['eNB', 'gNB', 'GSM'],
  },
  {
    id: '3',
    commandName: '修改状态',
    commandCode: 'MOD STATUS_INFO',
    category: '总览',
    description: '修改设备状态信息配置',
    params: [
      {
        name: 'STATUS',
        type: 'enum',
        required: true,
        description: '状态',
        options: [
          { label: '启用', value: 1 },
          { label: '禁用', value: 0 },
        ],
      },
    ],
    productTypes: ['eNB', 'gNB'],
  },
  // 快速设置
  {
    id: '4',
    commandName: 'eNB配置查询',
    commandCode: 'LST eNB_CONFIG',
    category: '快速设置',
    description: '查询eNB快速配置信息',
    params: [],
    productTypes: ['eNB'],
  },
  {
    id: '5',
    commandName: 'eNB配置修改',
    commandCode: 'MOD eNB_CONFIG',
    category: '快速设置',
    description: '修改eNB快速配置',
    params: [
      { name: 'FREQ', type: 'number', required: true, description: '频点', minValue: 0, maxValue: 65535 },
      { name: 'PCI', type: 'number', required: true, description: '物理小区标识', minValue: 0, maxValue: 503 },
      { name: 'PWR', type: 'number', required: false, description: '发射功率(dBm)', minValue: -30, maxValue: 50 },
    ],
    productTypes: ['eNB'],
  },
  {
    id: '6',
    commandName: '小区配置查询',
    commandCode: 'LST CELL',
    category: '快速设置',
    description: '查询小区配置信息',
    params: [
      { name: 'CELLID', type: 'number', required: false, description: '小区ID，不填则查询全部', minValue: 0, maxValue: 65535 },
    ],
    productTypes: ['eNB', 'gNB'],
  },
  // 告警管理
  {
    id: '7',
    commandName: '告警查询',
    commandCode: 'LST ALARM',
    category: '告警管理',
    description: '查询设备当前告警',
    params: [
      { name: 'ALARM_LEVEL', type: 'enum', required: false, description: '告警级别', options: [
        { label: '紧急', value: 1 },
        { label: '重要', value: 2 },
        { label: '一般', value: 3 },
        { label: '提示', value: 4 },
      ]},
    ],
    productTypes: ['eNB', 'gNB', 'GSM'],
  },
  {
    id: '8',
    commandName: '告警清除',
    commandCode: 'CLR ALARM',
    category: '告警管理',
    description: '清除指定告警',
    params: [
      { name: 'ALARM_ID', type: 'string', required: true, description: '告警ID' },
    ],
    productTypes: ['eNB', 'gNB', 'GSM'],
  },
  // 性能统计
  {
    id: '9',
    commandName: '性能统计查询',
    commandCode: 'LST PM',
    category: '性能统计',
    description: '查询设备性能统计信息',
    params: [
      { name: 'START_TIME', type: 'string', required: true, description: '开始时间' },
      { name: 'END_TIME', type: 'string', required: true, description: '结束时间' },
    ],
    productTypes: ['eNB', 'gNB', 'GSM'],
  },
  // 设备控制
  {
    id: '10',
    commandName: '设备重启',
    commandCode: 'RST DEVICE',
    category: '设备控制',
    description: '重启指定设备',
    params: [
      { name: 'DELAY', type: 'number', required: false, description: '延迟秒数', minValue: 0, maxValue: 3600 },
    ],
    productTypes: ['eNB', 'gNB'],
  },
  {
    id: '11',
    commandName: '软件版本查询',
    commandCode: 'LST VERSION',
    category: '设备控制',
    description: '查询设备软件版本',
    params: [],
    productTypes: ['eNB', 'gNB', 'GSM'],
  },
];

// 分页大小
export const DEVICE_PAGE_SIZE = 8;
