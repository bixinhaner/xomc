import { useState, useMemo, useCallback } from 'react';
import { Tabs, Button, Space, DatePicker, Select, Input, Radio, Typography, Dropdown, Modal, Form, Switch, InputNumber, TimePicker, List, Spin, Segmented, Collapse, Card, Tag, Tooltip, Divider, App, Drawer } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  DownloadOutlined,
  TableOutlined,
  BarChartOutlined,
  EditOutlined,
  DeleteOutlined,
  CopyOutlined,
  StarOutlined,
  StarFilled,
  SettingOutlined,
  ClockCircleOutlined,
  MoreOutlined,
  FileOutlined,
  TeamOutlined,
  UserOutlined,
  InfoCircleOutlined,
  ExportOutlined,
  LineChartOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import TemplateDrawer from './components/TemplateDrawer';
import ExportDrawer from './components/ExportDrawer';

// Simple ID generator
const generateId = () => `chart-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;

const { RangePicker } = DatePicker;
const { Title, Text } = Typography;

// Mock template data
const PUBLIC_TEMPLATES: TemplateItem[] = [
  { id: 'tpl-1', name: 'eNB基础KPI', isDefault: true, reportSwitch: '0', isPublic: '1', isOneSelf: '0', isAdmin: '0' },
  { id: 'tpl-2', name: 'gNB性能指标', isDefault: false, reportSwitch: '1', isPublic: '1', isOneSelf: '0', isAdmin: '0' },
  { id: 'tpl-3', name: '小区吞吐量', isDefault: false, reportSwitch: '0', isPublic: '1', isOneSelf: '0', isAdmin: '0' },
  { id: 'tpl-6', name: '切换成功率分析', isDefault: false, reportSwitch: '0', isPublic: '1', isOneSelf: '1', isAdmin: '0' },
  { id: 'tpl-7', name: '用户数统计', isDefault: false, reportSwitch: '1', isPublic: '1', isOneSelf: '1', isAdmin: '1' },
];

const PRIVATE_TEMPLATES: TemplateItem[] = [
  { id: 'tpl-4', name: '我的eNB模板', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '1', creator: 'admin', isAdmin: '0' },
  { id: 'tpl-5', name: '测试模板', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '1', creator: 'admin', isAdmin: '0' },
  { id: 'tpl-8', name: '自定义报表', isDefault: false, reportSwitch: '1', isPublic: '0', isOneSelf: '1', creator: 'admin', isAdmin: '0' },
  { id: 'tpl-9', name: '性能监控模板', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '0', creator: 'zhangsan', isAdmin: '0' },
  { id: 'tpl-10', name: '告警分析', isDefault: false, reportSwitch: '1', isPublic: '0', isOneSelf: '0', creator: 'zhangsan', isAdmin: '0' },
  { id: 'tpl-11', name: '容量规划', isDefault: false, reportSwitch: '0', isPublic: '0', isOneSelf: '0', creator: 'lisi', isAdmin: '0' },
];

// Mock KPI data for each template - Device query data
const TEMPLATE_DEVICE_DATA: Record<string, Record<string, unknown>[]> = {
  'tpl-1': [ // eNB基础KPI
    { key: '1', serialNumber: 'ENB00001', hostName: '北京朝阳基站01', subStationName: '朝阳站址001', enodeId: '100001', cellId: '1', eci: '100001001', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.2%', ERAB_SR: '98.5%', DL_THP: '145.6', UL_THP: '32.1' },
    { key: '2', serialNumber: 'ENB00002', hostName: '北京海淀基站01', subStationName: '海淀站址001', enodeId: '100002', cellId: '1', eci: '100002001', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.9%', ERAB_SR: '97.8%', DL_THP: '132.1', UL_THP: '28.5' },
    { key: '3', serialNumber: 'ENB00003', hostName: '北京东城基站01', subStationName: '东城站址001', enodeId: '100003', cellId: '1', eci: '100003001', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.5%', ERAB_SR: '98.9%', DL_THP: '156.3', UL_THP: '35.7' },
    { key: '4', serialNumber: 'ENB00004', hostName: '北京西城基站01', subStationName: '西城站址001', enodeId: '100004', cellId: '1', eci: '100004001', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.7%', ERAB_SR: '97.5%', DL_THP: '128.9', UL_THP: '26.4' },
    { key: '5', serialNumber: 'ENB00005', hostName: '上海浦东基站01', subStationName: '浦东站址001', enodeId: '200001', cellId: '1', eci: '200001001', groupName: '上海区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.1%', ERAB_SR: '98.3%', DL_THP: '142.5', UL_THP: '31.2' },
    { key: '6', serialNumber: 'ENB00006', hostName: '上海徐汇基站01', subStationName: '徐汇站址001', enodeId: '200002', cellId: '1', eci: '200002001', groupName: '上海区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.6%', ERAB_SR: '97.2%', DL_THP: '125.8', UL_THP: '25.6' },
    { key: '7', serialNumber: 'ENB00007', hostName: '上海静安基站01', subStationName: '静安站址001', enodeId: '200003', cellId: '1', eci: '200003001', groupName: '上海区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.3%', ERAB_SR: '98.7%', DL_THP: '148.2', UL_THP: '33.9' },
    { key: '8', serialNumber: 'ENB00008', hostName: '广州天河基站01', subStationName: '天河站址001', enodeId: '300001', cellId: '1', eci: '300001001', groupName: '广州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.4%', ERAB_SR: '97.1%', DL_THP: '118.6', UL_THP: '24.3' },
    { key: '9', serialNumber: 'ENB00009', hostName: '广州越秀基站01', subStationName: '越秀站址001', enodeId: '300002', cellId: '1', eci: '300002001', groupName: '广州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.0%', ERAB_SR: '98.2%', DL_THP: '135.4', UL_THP: '29.8' },
    { key: '10', serialNumber: 'ENB00010', hostName: '深圳南山基站01', subStationName: '南山站址001', enodeId: '400001', cellId: '1', eci: '400001001', groupName: '深圳区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.8%', ERAB_SR: '97.6%', DL_THP: '129.7', UL_THP: '27.5' },
    { key: '11', serialNumber: 'ENB00011', hostName: '深圳福田基站01', subStationName: '福田站址001', enodeId: '400002', cellId: '1', eci: '400002001', groupName: '深圳区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.4%', ERAB_SR: '98.8%', DL_THP: '152.1', UL_THP: '34.6' },
    { key: '12', serialNumber: 'ENB00012', hostName: '杭州西湖基站01', subStationName: '西湖站址001', enodeId: '500001', cellId: '1', eci: '500001001', groupName: '杭州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.2%', ERAB_SR: '96.9%', DL_THP: '115.3', UL_THP: '23.1' },
    { key: '13', serialNumber: 'ENB00013', hostName: '成都武侯基站01', subStationName: '武侯站址001', enodeId: '600001', cellId: '1', eci: '600001001', groupName: '成都区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.1%', ERAB_SR: '98.4%', DL_THP: '141.8', UL_THP: '30.9' },
    { key: '14', serialNumber: 'ENB00014', hostName: '武汉洪山基站01', subStationName: '洪山站址001', enodeId: '700001', cellId: '1', eci: '700001001', groupName: '武汉区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.5%', ERAB_SR: '97.3%', DL_THP: '126.7', UL_THP: '26.2' },
    { key: '15', serialNumber: 'ENB00015', hostName: '南京鼓楼基站01', subStationName: '鼓楼站址001', enodeId: '800001', cellId: '1', eci: '800001001', groupName: '南京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.3%', ERAB_SR: '98.6%', DL_THP: '147.5', UL_THP: '33.2' },
  ],
  'tpl-2': [ // gNB性能指标
    { key: '1', serialNumber: 'GNB00001', hostName: '北京5G基站01', subStationName: '5G站址001', enodeId: '200001', cellId: '1', eci: '200001001', groupName: '北京5G区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.5%', ERAB_SR: '99.1%', DL_THP: '856.2', UL_THP: '125.3' },
    { key: '2', serialNumber: 'GNB00002', hostName: '上海5G基站01', subStationName: '5G站址002', enodeId: '200002', cellId: '1', eci: '200002001', groupName: '上海5G区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.3%', ERAB_SR: '98.8%', DL_THP: '923.5', UL_THP: '132.1' },
  ],
  'tpl-3': [ // 小区吞吐量
    { key: '1', serialNumber: 'ENB00101', hostName: '广州天河基站01', subStationName: '天河站址001', enodeId: '300101', cellId: '1', eci: '300101001', groupName: '广州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', DL_THP: '256.8', UL_THP: '45.2' },
    { key: '2', serialNumber: 'ENB00102', hostName: '广州越秀基站01', subStationName: '越秀站址001', enodeId: '300102', cellId: '1', eci: '300102001', groupName: '广州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', DL_THP: '312.5', UL_THP: '52.8' },
  ],
  'tpl-4': [ // 我的eNB模板
    { key: '1', serialNumber: 'ENB00201', hostName: '深圳南山基站01', subStationName: '南山站址001', enodeId: '400201', cellId: '1', eci: '400201001', groupName: '深圳区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '98.5%', ERAB_SR: '97.2%' },
  ],
  'tpl-5': [ // 测试模板
    { key: '1', serialNumber: 'TEST001', hostName: '测试设备01', subStationName: '测试站址', enodeId: '999001', cellId: '1', eci: '999001001', groupName: '测试组', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '100%', ERAB_SR: '100%', DL_THP: '999.9', UL_THP: '99.9' },
  ],
  'tpl-6': [ // 切换成功率分析
    { key: '1', serialNumber: 'ENB00301', hostName: '杭州西湖基站01', subStationName: '西湖站址001', enodeId: '500301', cellId: '1', eci: '500301001', groupName: '杭州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', HO_SR: '98.5%', HO_SUCCESS: '1250', HO_FAIL: '18' },
    { key: '2', serialNumber: 'ENB00302', hostName: '杭州滨江基站01', subStationName: '滨江站址001', enodeId: '500302', cellId: '1', eci: '500302001', groupName: '杭州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', HO_SR: '97.8%', HO_SUCCESS: '980', HO_FAIL: '22' },
  ],
  'tpl-7': [ // 用户数统计
    { key: '1', serialNumber: 'ENB00401', hostName: '成都武侯基站01', subStationName: '武侯站址001', enodeId: '600401', cellId: '1', eci: '600401001', groupName: '成都区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', ACTIVE_USER: '1250', IDLE_USER: '320', MAX_USER: '1500' },
    { key: '2', serialNumber: 'ENB00402', hostName: '成都锦江基站01', subStationName: '锦江站址001', enodeId: '600402', cellId: '1', eci: '600402001', groupName: '成都区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', ACTIVE_USER: '980', IDLE_USER: '280', MAX_USER: '1200' },
  ],
  'tpl-8': [ // 自定义报表
    { key: '1', serialNumber: 'ENB00501', hostName: '武汉洪山基站01', subStationName: '洪山站址001', enodeId: '700501', cellId: '1', eci: '700501001', groupName: '武汉区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', RRC_SR: '99.0%', ERAB_SR: '98.5%', DL_THP: '180.5', UL_THP: '35.2' },
  ],
  'tpl-9': [ // 性能监控模板
    { key: '1', serialNumber: 'ENB00601', hostName: '南京鼓楼基站01', subStationName: '鼓楼站址001', enodeId: '800601', cellId: '1', eci: '800601001', groupName: '南京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', CPU_USAGE: '45%', MEM_USAGE: '62%', DISK_USAGE: '38%' },
    { key: '2', serialNumber: 'ENB00602', hostName: '南京玄武基站01', subStationName: '玄武站址001', enodeId: '800602', cellId: '1', eci: '800602001', groupName: '南京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', CPU_USAGE: '52%', MEM_USAGE: '58%', DISK_USAGE: '42%' },
  ],
  'tpl-10': [ // 告警分析
    { key: '1', serialNumber: 'ENB00701', hostName: '西安雁塔基站01', subStationName: '雁塔站址001', enodeId: '900701', cellId: '1', eci: '900701001', groupName: '西安区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', ALARM_COUNT: '5', CRITICAL: '1', MAJOR: '2', MINOR: '2' },
  ],
  'tpl-11': [ // 容量规划
    { key: '1', serialNumber: 'ENB00801', hostName: '重庆渝北基站01', subStationName: '渝北站址001', enodeId: '1000801', cellId: '1', eci: '1000801001', groupName: '重庆区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', PRB_USAGE: '68%', CPU_LOAD: '55%', CAPACITY_WARN: '正常' },
    { key: '2', serialNumber: 'ENB00802', hostName: '重庆江北基站01', subStationName: '江北站址001', enodeId: '1000802', cellId: '1', eci: '1000802001', groupName: '重庆区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', PRB_USAGE: '85%', CPU_LOAD: '72%', CAPACITY_WARN: '预警' },
  ],
};

// Mock KPI data for each template - Device group query data
const TEMPLATE_GROUP_DATA: Record<string, Record<string, unknown>[]> = {
  'tpl-1': [ // eNB基础KPI
    { key: '1', groupName: '北京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '99.1%', avgErabSr: '98.2%', avgDlThp: '138.9', avgUlThp: '30.3' },
    { key: '2', groupName: '上海区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '98.8%', avgErabSr: '97.9%', avgDlThp: '125.6', avgUlThp: '27.8' },
  ],
  'tpl-2': [ // gNB性能指标
    { key: '1', groupName: '北京5G区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '99.5%', avgErabSr: '99.1%', avgDlThp: '856.2', avgUlThp: '125.3' },
    { key: '2', groupName: '上海5G区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '99.3%', avgErabSr: '98.8%', avgDlThp: '923.5', avgUlThp: '132.1' },
  ],
  'tpl-3': [ // 小区吞吐量
    { key: '1', groupName: '广州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgDlThp: '284.7', avgUlThp: '49.0' },
    { key: '2', groupName: '深圳区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgDlThp: '310.2', avgUlThp: '55.6' },
  ],
  'tpl-4': [ // 我的eNB模板
    { key: '1', groupName: '深圳区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '98.5%', avgErabSr: '97.2%' },
  ],
  'tpl-5': [ // 测试模板
    { key: '1', groupName: '测试组', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '100%', avgErabSr: '100%', avgDlThp: '999.9', avgUlThp: '99.9' },
  ],
  'tpl-6': [ // 切换成功率分析
    { key: '1', groupName: '杭州区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgHoSr: '98.2%', totalHoSuccess: '2230', totalHoFail: '40' },
  ],
  'tpl-7': [ // 用户数统计
    { key: '1', groupName: '成都区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgActiveUser: '1115', avgIdleUser: '300', totalMaxUser: '2700' },
  ],
  'tpl-8': [ // 自定义报表
    { key: '1', groupName: '武汉区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgRrcSr: '99.0%', avgErabSr: '98.5%', avgDlThp: '180.5', avgUlThp: '35.2' },
  ],
  'tpl-9': [ // 性能监控模板
    { key: '1', groupName: '南京区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgCpuUsage: '48.5%', avgMemUsage: '60%', avgDiskUsage: '40%' },
  ],
  'tpl-10': [ // 告警分析
    { key: '1', groupName: '西安区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', totalAlarmCount: '5', totalCritical: '1', totalMajor: '2', totalMinor: '2' },
  ],
  'tpl-11': [ // 容量规划
    { key: '1', groupName: '重庆区域', timeLevel: '15Min', startTime: '2026-04-01 10:00', endTime: '2026-04-01 10:15', avgPrbUsage: '76.5%', avgCpuLoad: '63.5%', capacityWarnCount: '1' },
  ],
};

// Default empty data for templates without specific data
const DEFAULT_DEVICE_DATA: Record<string, unknown>[] = [];
const DEFAULT_GROUP_DATA: Record<string, unknown>[] = [];

interface TemplateItem {
  id: string;
  name: string;
  isDefault?: boolean;
  reportSwitch?: string;
  isPublic: string;
  isOneSelf?: string;
  isAdmin?: string;
  creator?: string;
  description?: string;
}

// Time range type for chart display
type TimeRangeType = 'today' | 'yesterday' | 'thisWeek' | 'lastWeek' | 'thisMonth' | 'lastMonth';

// Chart configuration interface
interface ChartConfig {
  id: string;
  name: string;
  devices: string[]; // Max 5 devices
  kpis: string[]; // Max 3 KPIs
  timeRange: TimeRangeType; // Time range for chart display
  createdAt: number;
}

// Available devices for selection
const AVAILABLE_DEVICES = [
  { value: 'ENB00001', label: 'ENB00001 北京朝阳基站01' },
  { value: 'ENB00002', label: 'ENB00002 北京海淀基站01' },
  { value: 'GNB00001', label: 'GNB00001 北京5G基站01' },
  { value: 'GNB00002', label: 'GNB00002 上海5G基站01' },
  { value: 'ENB00101', label: 'ENB00101 广州天河基站01' },
  { value: 'ENB00102', label: 'ENB00102 广州越秀基站01' },
  { value: 'ENB00201', label: 'ENB00201 深圳南山基站01' },
  { value: 'ENB00301', label: 'ENB00301 杭州西湖基站01' },
  { value: 'ENB00302', label: 'ENB00302 杭州滨江基站01' },
  { value: 'ENB00401', label: 'ENB00401 成都武侯基站01' },
];

// Available device groups for selection
const AVAILABLE_GROUPS = [
  { value: 'group-1', label: '北京区域' },
  { value: 'group-1-1', label: '北京朝阳' },
  { value: 'group-1-2', label: '北京海淀' },
  { value: 'group-2', label: '上海区域' },
  { value: 'group-2-1', label: '上海浦东' },
  { value: 'group-2-2', label: '上海徐汇' },
  { value: 'group-3', label: '广州区域' },
  { value: 'group-4', label: '深圳区域' },
];

// Available KPIs for selection
const AVAILABLE_KPIS = [
  { value: 'RRC_SR', label: 'RRC建立成功率', category: 'access' },
  { value: 'ERAB_SR', label: 'ERAB建立成功率', category: 'access' },
  { value: 'DL_THP', label: '下行吞吐量', category: 'throughput' },
  { value: 'UL_THP', label: '上行吞吐量', category: 'throughput' },
  { value: 'HO_SR', label: '切换成功率', category: 'handover' },
  { value: 'ACTIVE_USER', label: '活跃用户数', category: 'user' },
];

// Unified chart time type
type ChartTimeType = 'day' | 'week' | 'month';

// Unified chart time configuration
const CHART_TIME_CONFIG: Record<ChartTimeType, {
  interval: number;
  intervalUnit: dayjs.ManipulateType;
  format: string;
  defaultDays: number;
}> = {
  day: { interval: 1, intervalUnit: 'hour', format: 'HH:mm', defaultDays: 1 },
  week: { interval: 1, intervalUnit: 'day', format: 'MM-DD', defaultDays: 7 },
  month: { interval: 1, intervalUnit: 'day', format: 'MM-DD', defaultDays: 30 },
};

// Generate mock chart data for a device and KPI with unified time filter
const generateMockChartDataUnified = (
  device: string,
  kpi: string,
  timeType: ChartTimeType,
  dateRange: [dayjs.Dayjs, dayjs.Dayjs] | null
) => {
  const data: number[] = [];
  const xData: string[] = [];

  const config = CHART_TIME_CONFIG[timeType];
  const now = dayjs();

  // Use date range if provided, otherwise use default based on time type
  const start = dateRange ? dateRange[0].startOf('day') : now.subtract(config.defaultDays - 1, 'day').startOf('day');
  const end = dateRange ? dateRange[1].endOf('day') : now.endOf('day');

  // Use device and kpi to generate consistent base value
  const seed = device.charCodeAt(0) + kpi.charCodeAt(0);
  const baseValue = (seed % 50) + 50; // 50-100 base

  let current = start;
  while (current.isBefore(end) || current.isSame(end, config.intervalUnit)) {
    xData.push(current.format(config.format));
    data.push(Number((baseValue + Math.random() * 10 - 5).toFixed(1)));
    current = current.add(config.interval, config.intervalUnit);
    // Prevent infinite loop
    if (xData.length > 1000) break;
  }

  return { data, xData };
};

export default function KPIQuery() {
  const t = useT();
  const token = useThemeToken();
  const { modal, message } = App.useApp();
  const [searchText, setSearchText] = useState('');
  const [templateTab, setTemplateTab] = useState<'public' | 'private'>('public');
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>('tpl-1');
  const [defaultTemplateId, setDefaultTemplateId] = useState<string>('tpl-1');
  const [openTabs, setOpenTabs] = useState<string[]>(['tpl-1']);
  const [activeTab, setActiveTab] = useState<string>('tpl-1');
  const [viewMode, setViewMode] = useState<'table' | 'chart'>('table');
  const [granularity, setGranularity] = useState('15');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [deviceType, setDeviceType] = useState<'1' | '2'>('2');
  const [deviceSearch, setDeviceSearch] = useState('');
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(30);

  // Template drawer state
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false);
  const [templateDrawerMode, setTemplateDrawerMode] = useState<'add' | 'edit' | 'view' | 'copy'>('add');
  const [editingTemplate, setEditingTemplate] = useState<TemplateItem | null>(null);
  const [templateSaving, setTemplateSaving] = useState(false);

  // Report config drawer state
  const [reportDrawerOpen, setReportDrawerOpen] = useState(false);
  const [reportForm] = Form.useForm();
  const [reportTargetTemplateId, setReportTargetTemplateId] = useState<string | null>(null);
  const [reportSwitchMap, setReportSwitchMap] = useState<Record<string, string>>(() => {
    const map: Record<string, string> = {};
    for (const tpl of PUBLIC_TEMPLATES) map[tpl.id] = tpl.reportSwitch ?? '0';
    for (const tpl of PRIVATE_TEMPLATES) map[tpl.id] = tpl.reportSwitch ?? '0';
    return map;
  });

  // Export drawer state
  const [exportDrawerOpen, setExportDrawerOpen] = useState(false);

  // Chart management state
  const [charts, setCharts] = useState<ChartConfig[]>([]);
  const [chartConfigModalOpen, setChartConfigModalOpen] = useState(false);
  const [editingChart, setEditingChart] = useState<ChartConfig | null>(null);
  const [chartForm] = Form.useForm();
  const [chartSaving, setChartSaving] = useState(false);
  const [chartDeviceSelectType, setChartDeviceSelectType] = useState<'device' | 'group'>('device');
  const [chartKpiCategory, setChartKpiCategory] = useState<string>('all');

  // Unified chart time filter state
  const [chartTimeType, setChartTimeType] = useState<'day' | 'week' | 'month'>('day');
  const [chartDateRange, setChartDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);

  // All templates combined
  const allTemplates = useMemo(() => [...PUBLIC_TEMPLATES, ...PRIVATE_TEMPLATES], []);

  // Time granularity options with i18n
  const granularityOptions = useMemo(() => [
    { label: t('perf.query.granularity15min'), value: '15' },
    { label: t('perf.query.granularity60min'), value: '60' },
    { label: t('perf.query.granularity24hour'), value: '1440' },
    { label: t('perf.query.granularityWeek'), value: '10080' },
    { label: t('perf.query.granularityMonth'), value: '43200' },
  ], [t]);

  // KPI categories for filtering
  const kpiCategories = useMemo(() => [
    { label: t('perf.query.allCategories'), value: 'all' },
    { label: t('perf.query.accessKpiCategory'), value: 'access' },
    { label: t('perf.query.throughputKpiCategory'), value: 'throughput' },
    { label: t('perf.query.handoverKpiCategory'), value: 'handover' },
    { label: t('perf.query.userKpiCategory'), value: 'user' },
  ], [t]);

  // Filtered KPIs based on selected category
  const filteredKpis = useMemo(() => {
    if (chartKpiCategory === 'all') {
      return AVAILABLE_KPIS;
    }
    return AVAILABLE_KPIS.filter(kpi => kpi.category === chartKpiCategory);
  }, [chartKpiCategory]);

  // Chart management functions
  const handleAddChart = useCallback(() => {
    setEditingChart(null);
    chartForm.resetFields();
    // Use setTimeout to ensure form is ready after reset
    setTimeout(() => {
      chartForm.setFieldsValue({
        name: t('perf.query.chartName', { index: charts.length + 1 }),
        devices: [],
        kpis: [],
      });
    }, 0);
    setChartConfigModalOpen(true);
  }, [chartForm, charts.length, t]);

  const handleEditChart = useCallback((chart: ChartConfig) => {
    setEditingChart(chart);
    chartForm.setFieldsValue({
      name: chart.name,
      devices: chart.devices,
      kpis: chart.kpis,
    });
    setChartConfigModalOpen(true);
  }, [chartForm]);

  const handleDeleteChart = useCallback((chartId: string) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: t('perf.query.deleteChartConfirm'),
      okType: 'danger',
      onOk: () => {
        setCharts(prev => prev.filter(c => c.id !== chartId));
        void message.success(t('common.deleteSuccess'));
      },
    });
  }, [t, modal, message]);

  const handleSaveChart = useCallback(async () => {
    setChartSaving(true);
    try {
      const values = await chartForm.validateFields();

      if (editingChart) {
        // Update existing chart
        setCharts(prev => prev.map(c =>
          c.id === editingChart.id
            ? { ...c, name: values.name, devices: values.devices || [], kpis: values.kpis || [], timeRange: c.timeRange || 'today' }
            : c
        ));
        void message.success(t('common.updateSuccess'));
      } else {
        // Add new chart
        const newChart: ChartConfig = {
          id: generateId(),
          name: values.name,
          devices: values.devices || [],
          kpis: values.kpis || [],
          timeRange: 'today',
          createdAt: Date.now(),
        };
        setCharts(prev => [...prev, newChart]);
        void message.success(t('common.addSuccess'));
      }
      setChartConfigModalOpen(false);
    } catch {
      // Form validation failed - errors are displayed inline
    } finally {
      setChartSaving(false);
    }
  }, [chartForm, editingChart, t, message]);

  // Generate chart series data based on configuration
  const generateChartSeries = useCallback((chart: ChartConfig, timeType: ChartTimeType, dateRange: [dayjs.Dayjs, dayjs.Dayjs] | null) => {
    const series: { name: string; data: number[] }[] = [];

    chart.devices.forEach((device) => {
      const deviceLabel = AVAILABLE_DEVICES.find(d => d.value === device)?.label?.split(' ').slice(1).join(' ') || device;

      chart.kpis.forEach((kpi) => {
        const kpiLabel = AVAILABLE_KPIS.find(k => k.value === kpi)?.label || kpi;
        const mockData = generateMockChartDataUnified(device, kpi, timeType, dateRange);

        series.push({
          name: `${deviceLabel} - ${kpiLabel}`,
          data: mockData.data,
        });
      });
    });

    return series;
  }, []);

  // Generate chart xData based on unified time filter
  const generateChartXData = useCallback((timeType: ChartTimeType, dateRange: [dayjs.Dayjs, dayjs.Dayjs] | null) => {
    const config = CHART_TIME_CONFIG[timeType];
    const now = dayjs();

    const start = dateRange ? dateRange[0].startOf('day') : now.subtract(config.defaultDays - 1, 'day').startOf('day');
    const end = dateRange ? dateRange[1].endOf('day') : now.endOf('day');

    const xData: string[] = [];
    let current = start;
    while (current.isBefore(end) || current.isSame(end, config.intervalUnit)) {
      xData.push(current.format(config.format));
      current = current.add(config.interval, config.intervalUnit);
      if (xData.length > 1000) break;
    }
    return xData;
  }, []);

  // Get template menu items - all templates have the same menu items
  const getTemplateMenuItems = useCallback((template: TemplateItem): MenuProps['items'] => {
    const items: MenuProps['items'] = [];
    const isDefault = template.id === defaultTemplateId;

    // 已为默认 / 设为默认（仅公共模板支持设为默认）
    if (template.isPublic === '1') {
      if (isDefault) {
        items.push({
          key: 'isDefault',
          label: t('perf.query.isDefault'),
          icon: <StarFilled style={{ color: '#faad14' }} />,
          disabled: true,
        });
      } else {
        items.push({
          key: 'setDefault',
          label: t('perf.query.setAsDefault'),
          icon: <StarOutlined />,
        });
      }
    }

    // 详情
    items.push({
      key: 'detail',
      label: t('common.detail'),
      icon: <InfoCircleOutlined />,
    });

    // 修改
    items.push({
      key: 'edit',
      label: t('common.edit'),
      icon: <EditOutlined />,
    });

    // 导出指标
    items.push({
      key: 'exportKpi',
      label: t('perf.query.exportKpi'),
      icon: <ExportOutlined />,
    });

    // 定时报表
    items.push({
      key: 'report',
      label: t('perf.query.reportConfig'),
      icon: <SettingOutlined />,
    });

    // 删除 - 非默认模板才显示
    if (!isDefault) {
      items.push({ type: 'divider' });
      items.push({
        key: 'delete',
        label: t('common.delete'),
        icon: <DeleteOutlined />,
        danger: true,
      });
    }

    // 复制模板
    items.push({
      key: 'copy',
      label: t('common.copy'),
      icon: <CopyOutlined />,
    });

    return items;
  }, [t, defaultTemplateId]);

  // 导出模版数据
  const handleExportTemplate = useCallback((template: TemplateItem) => {
    // 构建模版导出数据
    const exportData = {
      id: template.id,
      name: template.name,
      isPublic: template.isPublic,
      isDefault: template.isDefault,
      reportSwitch: template.reportSwitch,
      exportedAt: new Date().toISOString(),
    };

    // 创建 Blob 并下载
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `template_${template.id}_${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    void message.success(t('common.exportSuccess'));
  }, [t, message]);

  // Handle template menu click
  const handleTemplateMenuClick = useCallback((key: string, template: TemplateItem) => {
    switch (key) {
      case 'detail':
        // 使用 TemplateDrawer 的 view 模式查看模板详情
        setEditingTemplate(template);
        setTemplateDrawerMode('view');
        setTemplateDialogOpen(true);
        break;
      case 'edit':
        setEditingTemplate(template);
        setTemplateDrawerMode('edit');
        setTemplateDialogOpen(true);
        break;
      case 'copy':
        setEditingTemplate(template);
        setTemplateDrawerMode('copy');
        setTemplateDialogOpen(true);
        break;
      case 'setDefault':
        modal.confirm({
          title: t('perf.query.setAsDefault'),
          content: t('perf.query.setAsDefaultConfirm', { name: template.name }),
          onOk: () => {
            setDefaultTemplateId(template.id);
            void message.success(t('common.success'));
          },
        });
        break;
      case 'exportKpi':
        // 直接导出模版数据
        handleExportTemplate(template);
        break;
      case 'report':
        setReportTargetTemplateId(template.id);
        reportForm.setFieldsValue({
          reportStatus: reportSwitchMap[template.id] === '1',
        });
        setReportDrawerOpen(true);
        break;
      case 'delete':
        modal.confirm({
          title: t('common.confirmDelete'),
          content: t('common.deleteConfirmMsg'),
          okType: 'danger',
          onOk: () => void message.success(t('common.deleteSuccess')),
        });
        break;
    }
  }, [t, modal, message, handleExportTemplate, reportForm, reportSwitchMap]);

  // Filter templates by search text
  const filteredPublicTemplates = useMemo(() => {
    if (!searchText.trim()) return PUBLIC_TEMPLATES;
    return PUBLIC_TEMPLATES.filter((tpl) =>
      tpl.name.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [searchText]);

  const filteredPrivateTemplates = useMemo(() => {
    if (!searchText.trim()) return PRIVATE_TEMPLATES;
    return PRIVATE_TEMPLATES.filter((tpl) =>
      tpl.name.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [searchText]);

  // Group private templates by creator
  const groupedPrivateTemplates = useMemo(() => {
    const groups: Record<string, TemplateItem[]> = {};
    filteredPrivateTemplates.forEach((tpl) => {
      const creator = tpl.creator || 'unknown';
      if (!groups[creator]) {
        groups[creator] = [];
      }
      groups[creator].push(tpl);
    });
    return Object.entries(groups).map(([creator, templates]) => ({
      creator,
      templates,
      count: templates.length,
    }));
  }, [filteredPrivateTemplates]);

  // Handle template selection
  const handleTemplateSelect = useCallback((template: TemplateItem) => {
    setSelectedTemplateId(template.id);
    if (!openTabs.includes(template.id)) {
      setOpenTabs([...openTabs, template.id]);
    }
    setActiveTab(template.id);
  }, [openTabs]);

  // Handle tab close
  const handleTabClose = useCallback((targetKey: string) => {
    const newTabs = openTabs.filter((tab) => tab !== targetKey);
    setOpenTabs(newTabs);
    if (activeTab === targetKey && newTabs.length > 0) {
      setActiveTab(newTabs[newTabs.length - 1]);
    }
  }, [openTabs, activeTab]);

  // Handle query
  const handleQuery = useCallback(() => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      void message.success(t('common.success'));
    }, 1000);
  }, [t, message]);

  // Handle pagination change
  const handlePageChange = useCallback((page: number, size: number) => {
    setCurrentPage(page);
    setPageSize(size);
  }, []);

  // Get template name by id
  const getTemplateName = useCallback((id: string): string => {
    const found = allTemplates.find((t) => t.id === id);
    return found?.name ?? id;
  }, [allTemplates]);

  // Render template item
  const renderTemplateItem = useCallback((template: TemplateItem) => (
    <List.Item
      key={template.id}
      onClick={() => handleTemplateSelect(template)}
      style={{
        padding: '8px 16px',
        cursor: 'pointer',
        background: selectedTemplateId === template.id ? token.colorPrimaryBg : 'transparent',
        borderRadius: 4,
        marginBottom: 4,
        transition: 'background 0.2s',
      }}
      onMouseEnter={(e) => {
        if (selectedTemplateId !== template.id) {
          e.currentTarget.style.background = token.colorBgTextHover;
        }
      }}
      onMouseLeave={(e) => {
        if (selectedTemplateId !== template.id) {
          e.currentTarget.style.background = 'transparent';
        }
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, overflow: 'hidden' }}>
          <FileOutlined style={{ color: '#8c8c8c', flexShrink: 0 }} />
          <Text ellipsis style={{ flex: 1 }}>{template.name}</Text>
          {template.isPublic === '1' && template.id === defaultTemplateId && <Tooltip title={t('perf.query.isDefault')}><StarOutlined style={{ color: '#faad14', fontSize: 12, flexShrink: 0 }} /></Tooltip>}
          {reportSwitchMap[template.id] === '1' && <Tooltip title={t('perf.query.reportConfig')}><ClockCircleOutlined style={{ color: '#52c41a', fontSize: 12, flexShrink: 0 }} /></Tooltip>}
        </div>
        <Dropdown
          menu={{
            items: getTemplateMenuItems(template),
            onClick: (info) => {
              info.domEvent.stopPropagation();
              handleTemplateMenuClick(info.key, template);
            },
          }}
          trigger={['click']}
        >
          <Button
            type="text"
            size="small"
            icon={<MoreOutlined />}
            onClick={(e) => e.stopPropagation()}
            style={{ padding: '0 4px', flexShrink: 0 }}
          />
        </Dropdown>
      </div>
    </List.Item>
  ), [selectedTemplateId, defaultTemplateId, reportSwitchMap, token, t, handleTemplateSelect, handleTemplateMenuClick, getTemplateMenuItems]);

  // Table columns - dynamic based on query object type (device or device group)
  const columns: DataTableColumn<Record<string, unknown>>[] = useMemo(() => {
    // Base columns shared by both device and device group
    const baseColumns: DataTableColumn<Record<string, unknown>>[] = [
      { key: 'groupName', title: t('perf.query.groupName'), dataIndex: 'groupName', width: 160, fixed: 'left' },
      { key: 'timeLevel', title: t('perf.query.timeLevel'), dataIndex: 'timeLevel', width: 100 },
      { key: 'startTime', title: t('perf.query.startTime'), dataIndex: 'startTime', width: 160 },
      { key: 'endTime', title: t('perf.query.endTime'), dataIndex: 'endTime', width: 160 },
    ];

    // Device query columns (deviceType = '2')
    if (deviceType === '2') {
      return [
        { key: 'serialNumber', title: t('perf.query.serialNumber'), dataIndex: 'serialNumber', width: 160, fixed: 'left', mono: true },
        { key: 'hostName', title: t('perf.query.hostName'), dataIndex: 'hostName', width: 180, fixed: 'left' },
        { key: 'enodeId', title: t('perf.query.enodeId'), dataIndex: 'enodeId', width: 120, mono: true },
        { key: 'cellId', title: t('perf.query.cellId'), dataIndex: 'cellId', width: 100 },
        { key: 'eci', title: t('perf.query.eci'), dataIndex: 'eci', width: 140, mono: true },
        { key: 'groupName', title: t('perf.query.groupName'), dataIndex: 'groupName', width: 160 },
        { key: 'timeLevel', title: t('perf.query.timeLevel'), dataIndex: 'timeLevel', width: 100 },
        { key: 'startTime', title: t('perf.query.startTime'), dataIndex: 'startTime', width: 160 },
        { key: 'endTime', title: t('perf.query.endTime'), dataIndex: 'endTime', width: 160 },
        { key: 'RRC_SR', title: `${t('kpi.rrcSetupSuccessRate')}(%)`, dataIndex: 'RRC_SR', width: 140 },
        { key: 'ERAB_SR', title: `${t('kpi.erabSetupSuccessRate')}(%)`, dataIndex: 'ERAB_SR', width: 150 },
        { key: 'DL_THP', title: `${t('kpi.dlThroughput')}(Mbps)`, dataIndex: 'DL_THP', width: 140 },
        { key: 'UL_THP', title: `${t('kpi.ulThroughput')}(Mbps)`, dataIndex: 'UL_THP', width: 140 },
      ];
    }

    // Device group query columns (deviceType = '1')
    return [
      ...baseColumns,
      { key: 'avgRrcSr', title: `${t('kpi.rrcSetupSuccessRate')}(%)`, dataIndex: 'avgRrcSr', width: 140 },
      { key: 'avgErabSr', title: `${t('kpi.erabSetupSuccessRate')}(%)`, dataIndex: 'avgErabSr', width: 150 },
      { key: 'avgDlThp', title: `${t('kpi.dlThroughput')}(Mbps)`, dataIndex: 'avgDlThp', width: 140 },
      { key: 'avgUlThp', title: `${t('kpi.ulThroughput')}(Mbps)`, dataIndex: 'avgUlThp', width: 140 },
    ];
  }, [t, deviceType]);

  // Get mock data based on selected template and query object type
  const mockData = useMemo(() => {
    if (!selectedTemplateId) return deviceType === '2' ? DEFAULT_DEVICE_DATA : DEFAULT_GROUP_DATA;

    if (deviceType === '2') {
      return TEMPLATE_DEVICE_DATA[selectedTemplateId] || DEFAULT_DEVICE_DATA;
    }
    return TEMPLATE_GROUP_DATA[selectedTemplateId] || DEFAULT_GROUP_DATA;
  }, [selectedTemplateId, deviceType]);

  // Paginated data based on current page and page size
  const paginatedData = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    return mockData.slice(start, end);
  }, [mockData, currentPage, pageSize]);

  // Left panel with tabs
  const leftPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <div
        style={{
          padding: '12px 12px 8px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <Title level={5} style={{ margin: 0, fontSize: 14 }}>
          {t('perf.query.templateTitle')}
        </Title>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingTemplate(null);
            setTemplateDrawerMode('add');
            setTemplateDialogOpen(true);
          }}
        >
          {t('common.add')}
        </Button>
      </div>

      {/* Search */}
      <div style={{ padding: '8px 12px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
        <Input
          placeholder={t('perf.query.searchTemplate')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
          size="small"
        />
      </div>

      {/* Tabs for Public/Private */}
      <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
        {/* Segmented Control */}
        <div style={{
          padding: '12px 12px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}>
          <Radio.Group
            value={templateTab}
            onChange={(e) => setTemplateTab(e.target.value)}
            optionType="button"
            buttonStyle="solid"
            size="small"
            style={{ width: '100%', display: 'flex' }}
          >
            <Radio.Button value="public" style={{ flex: 1, textAlign: 'center' }}>
              <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                <TeamOutlined />
                <span>{t('perf.query.publicTemplate')}</span>
                <span style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  minWidth: 18,
                  height: 18,
                  padding: '0 4px',
                  borderRadius: 9,
                  fontSize: 11,
                  backgroundColor: 'rgba(0,0,0,0.15)',
                }}>
                  {filteredPublicTemplates.length}
                </span>
              </div>
            </Radio.Button>
            <Radio.Button value="private" style={{ flex: 1, textAlign: 'center' }}>
              <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                <UserOutlined />
                <span>{t('perf.query.privateTemplate')}</span>
                <span style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  minWidth: 18,
                  height: 18,
                  padding: '0 4px',
                  borderRadius: 9,
                  fontSize: 11,
                  backgroundColor: 'rgba(0,0,0,0.15)',
                }}>
                  {filteredPrivateTemplates.length}
                </span>
              </div>
            </Radio.Button>
          </Radio.Group>
        </div>

        {/* Tab Content */}
        <div style={{ flex: 1, overflow: 'auto', padding: '8px 8px' }}>
          {templateTab === 'public' ? (
            filteredPublicTemplates.length > 0 ? (
              filteredPublicTemplates.map(renderTemplateItem)
            ) : (
              <div style={{ textAlign: 'center', padding: 24, color: token.colorTextSecondary }}>
                {t('common.noData')}
              </div>
            )
          ) : (
            groupedPrivateTemplates.length > 0 ? (
              <Collapse
                defaultActiveKey={groupedPrivateTemplates[0]?.creator}
                ghost
                expandIconPosition="end"
                style={{ background: 'transparent' }}
                items={groupedPrivateTemplates.map((group) => ({
                  key: group.creator,
                  label: (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
                      <UserOutlined style={{ color: token.colorPrimary }} />
                      <span style={{ flex: 1 }}>{group.creator}</span>
                      <span style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        minWidth: 18,
                        height: 18,
                        padding: '0 4px',
                        borderRadius: 9,
                        fontSize: 11,
                        backgroundColor: token.colorPrimaryBg,
                        color: token.colorPrimary,
                      }}>
                        {group.count}
                      </span>
                    </div>
                  ),
                  children: (
                    <>
                      {group.templates.map((template) => renderTemplateItem(template))}
                    </>
                  ),
                }))}
              />
            ) : (
              <div style={{ textAlign: 'center', padding: 24, color: token.colorTextSecondary }}>
                {t('common.noData')}
              </div>
            )
          )}
        </div>
      </div>
    </div>
  );

  // Right content panel - wrapped in useCallback to avoid changing on every render
  const renderRightPanel = useCallback((tabId: string) => {
    return (
      <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        {/* Query conditions */}
        <div
          style={{
            padding: '12px 16px',
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
            <span style={{ fontSize: 14, fontWeight: 600, whiteSpace: 'nowrap' }}>{t('perf.query.queryObjectType')}</span>
            <Radio.Group
              value={deviceType}
              onChange={(e) => setDeviceType(e.target.value as '1' | '2')}
              optionType="button"
              buttonStyle="solid"
              size="small"
            >
              <Radio.Button value="2">{t('perf.query.device')}</Radio.Button>
              <Radio.Button value="1">{t('device.group')}</Radio.Button>
            </Radio.Group>
            <Input
              placeholder={deviceType === '2' ? t('perf.query.deviceSearchPlaceholder') : t('perf.query.groupSearchPlaceholder')}
              prefix={<SearchOutlined />}
              value={deviceSearch}
              onChange={(e) => setDeviceSearch(e.target.value)}
              allowClear
              style={{ width: 220 }}
              size="small"
            />
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.granularity')}</Text>
              <Select
                value={granularity}
                onChange={setGranularity}
                options={granularityOptions}
                style={{ width: 100 }}
                size="small"
              />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.timeRange')}</Text>
              <RangePicker
                value={dateRange}
                onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs] | null)}
                showTime={{ format: 'HH:mm' }}
                format="YYYY-MM-DD HH:mm"
                style={{ width: 360 }}
                size="small"
              />
            </div>
            <div style={{ flex: 1 }} />
            <Space size="small">
              <Button type="primary" onClick={handleQuery} loading={loading} size="small">
                {t('common.query')}
              </Button>
              <Button
                icon={<DownloadOutlined />}
                size="small"
                onClick={() => setExportDrawerOpen(true)}
              >
                {t('common.export')}
              </Button>
            </Space>
          </div>
        </div>

        {/* View mode toggle */}
        <div style={{ padding: '8px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text strong>{getTemplateName(tabId)}</Text>
          <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} size="small">
            <Radio.Button value="table"><TableOutlined /> {t('perf.query.tableView')}</Radio.Button>
            <Radio.Button value="chart"><BarChartOutlined /> {t('perf.query.chartView')}</Radio.Button>
          </Radio.Group>
        </div>

        {/* Result area */}
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          {viewMode === 'table' ? (
            <div className="kpi-query-table-wrapper" style={{ flex: 1, padding: '0 16px 16px', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
              <style>{`
                .kpi-query-table-wrapper .ant-table-thead > tr > th,
                .kpi-query-table-wrapper .ant-table-tbody > tr > td {
                  white-space: nowrap !important;
                }
                .kpi-query-table-wrapper .ant-spin-nested-loading,
                .kpi-query-table-wrapper .ant-spin-container {
                  height: 100%;
                }
                .kpi-query-table-wrapper [class*="dataTableWrapper"] {
                  height: 100%;
                }
                .kpi-query-table-wrapper [class*="tableContainer"] {
                  flex: 1;
                  min-height: 0;
                  overflow: hidden;
                }
                .kpi-query-table-wrapper [class*="paginationWrapper"] {
                  flex-shrink: 0;
                }
              `}</style>
              <Spin spinning={loading} style={{ height: '100%' }}>
                <DataTable
                  tableId={`kpi-query-${tabId}`}
                  columns={columns}
                  dataSource={paginatedData}
                  rowKey="key"
                  scroll={{ x: 'max-content', y: 'calc(100vh - 400px)' }}
                  showRowNumber
                  rowNumberTitle={t('table.rowNumber')}
                  showPagination
                  currentPage={currentPage}
                  pageSize={pageSize}
                  total={mockData.length}
                  onPageChange={handlePageChange}
                />
              </Spin>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, overflow: 'hidden' }}>
              {/* Chart toolbar - fixed at top with unified time filter */}
              <div style={{ padding: '12px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexShrink: 0, borderBottom: `1px solid ${token.colorBorderSecondary}`, gap: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                  {/* Unified time type selector */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.chartTimeType')}</Text>
                    <Segmented
                      value={chartTimeType}
                      onChange={(value) => setChartTimeType(value as ChartTimeType)}
                      options={[
                        { label: t('perf.query.chartTimeDay'), value: 'day' },
                        { label: t('perf.query.chartTimeWeek'), value: 'week' },
                        { label: t('perf.query.chartTimeMonth'), value: 'month' },
                      ]}
                      size="small"
                    />
                  </div>
                  {/* Unified date range picker */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Text style={{ whiteSpace: 'nowrap' }}>{t('perf.query.chartDateRange')}</Text>
                    <RangePicker
                      value={chartDateRange}
                      onChange={(dates) => setChartDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs] | null)}
                      format="YYYY-MM-DD"
                      style={{ width: 260 }}
                      size="small"
                      allowClear
                      placeholder={[t('perf.query.startDate'), t('perf.query.endDate')]}
                    />
                  </div>
                </div>
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAddChart} size="small">
                  {t('perf.query.addChart')}
                </Button>
              </div>

              {/* Charts grid - scrollable */}
              <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
                {charts.length === 0 ? (
                  <div style={{
                    height: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: token.colorTextSecondary,
                  }}>
                    <LineChartOutlined style={{ fontSize: 48, marginBottom: 16, opacity: 0.3 }} />
                    <div>{t('perf.query.noChart')}</div>
                    <Button type="link" onClick={handleAddChart}>{t('perf.query.addFirstChart')}</Button>
                  </div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                    {charts.map((chart) => (
                      <Card
                        key={chart.id}
                        size="small"
                        title={
                          <Space>
                            <LineChartOutlined />
                            <span>{chart.name}</span>
                            <Tag color="blue">{t('perf.query.deviceCount', { count: chart.devices.length })}</Tag>
                            <Tag color="green">{t('perf.query.kpiCount', { count: chart.kpis.length })}</Tag>
                          </Space>
                        }
                        extra={
                          <Space size="small">
                            <Tooltip title={t('common.edit')}>
                              <Button
                                type="text"
                                size="small"
                                icon={<EditOutlined />}
                                onClick={() => handleEditChart(chart)}
                              />
                            </Tooltip>
                            <Tooltip title={t('common.delete')}>
                              <Button
                                type="text"
                                size="small"
                                danger
                                icon={<DeleteOutlined />}
                                onClick={() => handleDeleteChart(chart.id)}
                              />
                            </Tooltip>
                          </Space>
                        }
                        styles={{ body: { padding: '12px 16px 16px' } }}
                      >
                        <div style={{ marginBottom: 12 }}>
                          <Space wrap size={[4, 8]}>
                            {chart.devices.map(device => (
                              <Tag key={device} color="processing" style={{ margin: 0 }}>
                                {AVAILABLE_DEVICES.find(d => d.value === device)?.label?.split(' ').slice(1).join(' ') || device}
                              </Tag>
                            ))}
                            <Divider type="vertical" style={{ height: 20, margin: '0 4px' }} />
                            {chart.kpis.map(kpi => (
                              <Tag key={kpi} color="success" style={{ margin: 0 }}>
                                {AVAILABLE_KPIS.find(k => k.value === kpi)?.label || kpi}
                              </Tag>
                            ))}
                          </Space>
                        </div>
                        <div style={{ height: 350 }}>
                          <LineChart
                            series={generateChartSeries(chart, chartTimeType, chartDateRange)}
                            xData={generateChartXData(chartTimeType, chartDateRange)}
                            height={350}
                            showLegend={true}
                          />
                        </div>
                      </Card>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }, [
    token, t, deviceType, setDeviceType, deviceSearch,
    granularity, setGranularity, granularityOptions, dateRange, setDateRange,
    handleQuery, loading, getTemplateName,
    viewMode, setViewMode, columns, paginatedData, currentPage, pageSize,
    mockData.length, handlePageChange, charts, handleAddChart,
    handleEditChart, handleDeleteChart,
    generateChartSeries, generateChartXData, chartTimeType, chartDateRange, setChartTimeType, setChartDateRange,
    setExportDrawerOpen,
  ]);

  // Tab items
  const tabItems = useMemo(() => {
    return openTabs.map((tabId) => ({
      key: tabId,
      label: getTemplateName(tabId),
      closable: openTabs.length > 1,
      children: renderRightPanel(tabId),
    }));
  }, [openTabs, getTemplateName, renderRightPanel]);

  return (
    <>
      <TreeListPageLayout tree={leftPanel} defaultTreeWidth={280}>
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          {openTabs.length > 0 ? (
            <Tabs
              type="editable-card"
              hideAdd
              activeKey={activeTab}
              onChange={setActiveTab}
              onEdit={(targetKey, action) => {
                if (action === 'remove' && typeof targetKey === 'string') {
                  handleTabClose(targetKey);
                }
              }}
              items={tabItems}
              className="mml-console-tabs"
              style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
              tabBarStyle={{ margin: 0, padding: '0 16px', background: token.colorBgContainer }}
            />
          ) : (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: token.colorTextSecondary }}>
              <div style={{ textAlign: 'center' }}>
                <FileOutlined style={{ fontSize: 48, marginBottom: 16, opacity: 0.3 }} />
                <div>{t('perf.query.selectTemplate')}</div>
              </div>
            </div>
          )}
        </div>
      </TreeListPageLayout>

      {/* Template drawer */}
      <TemplateDrawer
        open={templateDialogOpen}
        onClose={() => {
          setTemplateDialogOpen(false);
          setEditingTemplate(null);
          setTemplateDrawerMode('add');
        }}
        mode={templateDrawerMode}
        initialValues={editingTemplate ? {
          tempName: editingTemplate.name,
          isPublic: editingTemplate.isPublic as '0' | '1',
          description: editingTemplate.description,
        } : undefined}
        loading={templateSaving}
        onSubmit={() => {
          // TODO: Call API to save template with values
          setTemplateSaving(true);
          setTimeout(() => {
            setTemplateSaving(false);
            setTemplateDialogOpen(false);
            setEditingTemplate(null);
            setTemplateDrawerMode('add');
            void message.success(t('common.success'));
          }, 500);
        }}
      />

      {/* Report config drawer */}
      <Drawer
        title={t('perf.query.reportConfig')}
        open={reportDrawerOpen}
        onClose={() => setReportDrawerOpen(false)}
        width={480}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setReportDrawerOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={() => {
              reportForm.validateFields().then((values) => {
                if (reportTargetTemplateId) {
                  setReportSwitchMap((prev) => ({
                    ...prev,
                    [reportTargetTemplateId]: values.reportStatus ? '1' : '0',
                  }));
                }
                void message.success(t('common.success'));
                setReportDrawerOpen(false);
                setReportTargetTemplateId(null);
              }).catch(() => {});
            }}>{t('common.confirm')}</Button>
          </div>
        }
      >
        <Form form={reportForm} layout="vertical">
          <Form.Item label={t('perf.query.enableReport')} name="reportStatus" valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
          <Form.Item shouldUpdate={(prev, cur) => prev.reportStatus !== cur.reportStatus} noStyle>
            {({ getFieldValue }) => {
              const reportEnabled = getFieldValue('reportStatus');
              return (
                <>
                  <Form.Item
                    label={t('perf.query.reportPeriod')}
                    name="reportPeriod"
                    rules={reportEnabled ? [{ required: true, message: t('common.pleaseSelect') }] : []}
                  >
                    <Select mode="multiple" placeholder={t('common.pleaseSelect')} options={granularityOptions} />
                  </Form.Item>
                  <Form.Item
                    label={t('perf.query.reportTime')}
                    name="reportTime"
                    rules={reportEnabled ? [{ required: true, message: t('common.pleaseSelect') }] : []}
                  >
                    <TimePicker format="HH:mm" style={{ width: '100%' }} />
                  </Form.Item>
                </>
              );
            }}
          </Form.Item>
          <Form.Item label={t('perf.query.enableEmail')} name="mailStatus" valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
          <Form.Item shouldUpdate={(prev, cur) => prev.mailStatus !== cur.mailStatus} noStyle>
            {({ getFieldValue }) => {
              if (!getFieldValue('mailStatus')) return null;
              return (
                <Form.Item
                  label={t('perf.query.emailAddress')}
                  name="mailAddress"
                  rules={[{ required: true, message: t('common.pleaseInput') }]}
                >
                  <Input placeholder={t('perf.query.emailPlaceholder')} />
                </Form.Item>
              );
            }}
          </Form.Item>
          <Form.Item label={t('perf.query.enableFtp')} name="ftpSwitch" valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
          <Form.Item shouldUpdate={(prev, cur) => prev.ftpSwitch !== cur.ftpSwitch} noStyle>
            {({ getFieldValue }) => {
              if (!getFieldValue('ftpSwitch')) return null;
              return (
                <>
                  <Form.Item
                    label={t('perf.query.ftpProtocol')}
                    name="ftpProtocol"
                    rules={[{ required: true, message: t('common.pleaseSelect') }]}
                  >
                    <Radio.Group>
                      <Radio value="sftp">SFTP</Radio>
                      <Radio value="ftp">FTP</Radio>
                    </Radio.Group>
                  </Form.Item>
                  <Form.Item
                    label={t('perf.query.ftpPath')}
                    name="ftpPath"
                    rules={[{ required: true, message: t('common.pleaseInput') }]}
                  >
                    <Input placeholder="/data/reports" />
                  </Form.Item>
                  <Form.Item
                    label={t('perf.query.ftpIp')}
                    name="ftpIp"
                    rules={[{ required: true, message: t('common.pleaseInput') }]}
                  >
                    <Input placeholder="192.168.1.100" />
                  </Form.Item>
                  <Form.Item
                    label={t('perf.query.ftpPort')}
                    name="ftpPort"
                    rules={[{ required: true, message: t('common.pleaseInput') }]}
                  >
                    <InputNumber min={1} max={65535} placeholder="22" style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item
                    label={t('perf.query.ftpUser')}
                    name="ftpUser"
                    rules={[{ required: true, message: t('common.pleaseInput') }]}
                  >
                    <Input placeholder="admin" />
                  </Form.Item>
                  <Form.Item
                    label={t('perf.query.ftpPassword')}
                    name="ftpPassword"
                    rules={[{ required: true, message: t('common.pleaseInput') }]}
                  >
                    <Input.Password placeholder="******" />
                  </Form.Item>
                </>
              );
            }}
          </Form.Item>
        </Form>
      </Drawer>

      {/* Chart configuration modal */}
      <Modal
        title={editingChart ? t('perf.query.editChart') : t('perf.query.addChart')}
        open={chartConfigModalOpen}
        onCancel={() => setChartConfigModalOpen(false)}
        onOk={handleSaveChart}
        confirmLoading={chartSaving}
        width={800}
        destroyOnHidden
      >
        <Form form={chartForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label={t('perf.query.chartName')}
            rules={[{ required: true, message: t('common.pleaseInput') }]}
          >
            <Input placeholder={t('perf.query.chartNamePlaceholder')} />
          </Form.Item>

          <Form.Item label={t('perf.query.deviceSelectType')} required>
            <Radio.Group
              value={chartDeviceSelectType}
              onChange={(e) => {
                setChartDeviceSelectType(e.target.value);
                chartForm.setFieldsValue({ devices: [] });
              }}
              optionType="button"
              buttonStyle="solid"
            >
              <Radio.Button value="device">{t('perf.query.selectByDevice')}</Radio.Button>
              <Radio.Button value="group">{t('perf.query.selectByGroup')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            name="devices"
            label={chartDeviceSelectType === 'device' ? t('perf.query.selectDevices') : t('perf.query.selectGroups')}
            rules={[
              { required: true, message: t('common.pleaseSelect') },
              {
                validator: (_, value) =>
                  value && value.length > 5
                    ? Promise.reject(t('perf.query.maxDevices'))
                    : Promise.resolve(),
              },
            ]}
            extra={chartDeviceSelectType === 'device' ? t('perf.query.devicesExtra') : t('perf.query.groupsExtra')}
          >
            <Select
              mode="multiple"
              placeholder={chartDeviceSelectType === 'device' ? t('perf.query.selectDevicesPlaceholder') : t('perf.query.selectGroupsPlaceholder')}
              options={chartDeviceSelectType === 'device' ? AVAILABLE_DEVICES : AVAILABLE_GROUPS}
              maxTagCount={5}
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase()) ||
                (option?.value ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>

          <Form.Item label={t('perf.query.kpiSet')}>
            <Select
              value={chartKpiCategory}
              onChange={(value) => {
                setChartKpiCategory(value);
                // Clear selected KPIs when category changes
                chartForm.setFieldsValue({ kpis: [] });
              }}
              options={kpiCategories}
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item
            name="kpis"
            label={t('perf.query.selectKpis')}
            rules={[
              { required: true, message: t('common.pleaseSelect') },
              {
                validator: (_, value) =>
                  value && value.length > 3
                    ? Promise.reject(t('perf.query.maxKpis'))
                    : Promise.resolve(),
              },
            ]}
            extra={t('perf.query.kpisExtra')}
          >
            <Select
              mode="multiple"
              placeholder={t('perf.query.selectKpisPlaceholder')}
              options={filteredKpis}
              maxTagCount={3}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Export drawer */}
      <ExportDrawer
        open={exportDrawerOpen}
        onClose={() => setExportDrawerOpen(false)}
        templateName={getTemplateName(activeTab)}
        reportPeriod={granularity}
      />
    </>
  );
}
