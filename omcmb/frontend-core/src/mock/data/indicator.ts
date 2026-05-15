import type { IndicatorGroup, PerfIndicator, IndicatorUnit, IndicatorType } from '../../types/indicator';

// ── Mock Indicator Group Trees ───────────────────────────────────────────────

const enbChildren: IndicatorGroup[] = [
  { id: 'enb-call', enName: 'Call', cnName: '呼叫类', operatorCode: 'cmcc', isBuildIn: true, description: 'RRC、ERAB等呼叫相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 15, children: [] },
  { id: 'enb-context', enName: 'Context', cnName: '上下文类', operatorCode: 'cmcc', isBuildIn: true, description: '上下文管理相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 8, children: [] },
  { id: 'enb-data', enName: 'Data', cnName: '数据类', operatorCode: 'cmcc', isBuildIn: true, description: '数据传输相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 12, children: [] },
  { id: 'enb-drb', enName: 'DRB', cnName: 'DRB类', operatorCode: 'cmcc', isBuildIn: true, description: 'DRB承载相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 6, children: [] },
  { id: 'enb-erab', enName: 'ERAB', cnName: 'ERAB类', operatorCode: 'cmcc', isBuildIn: true, description: 'ERAB承载相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 10, children: [] },
  { id: 'enb-ho', enName: 'HO', cnName: '切换类', operatorCode: 'cmcc', isBuildIn: true, description: '切换相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 8, children: [] },
  { id: 'enb-eqpt', enName: 'EQPT', cnName: '设备类', operatorCode: 'cmcc', isBuildIn: true, description: '设备相关指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 4, children: [] },
  { id: 'enb-custom', enName: 'Custom', cnName: '自定义', operatorCode: 'cmcc', isBuildIn: false, description: '用户自定义指标', parentId: 'enb-root', deviceType: 'ENB', indicatorCount: 2, children: [] },
];

const gsmChildren: IndicatorGroup[] = [
  { id: 'gsm-call', enName: 'Call', cnName: '呼叫类', operatorCode: 'cmcc', isBuildIn: true, description: 'GSM呼叫相关指标', parentId: 'gsm-root', deviceType: 'GSM', indicatorCount: 10, children: [] },
  { id: 'gsm-ho', enName: 'HO', cnName: '切换类', operatorCode: 'cmcc', isBuildIn: true, description: 'GSM切换相关指标', parentId: 'gsm-root', deviceType: 'GSM', indicatorCount: 6, children: [] },
  { id: 'gsm-data', enName: 'Data', cnName: '数据类', operatorCode: 'cmcc', isBuildIn: true, description: 'GSM数据相关指标', parentId: 'gsm-root', deviceType: 'GSM', indicatorCount: 8, children: [] },
  { id: 'gsm-custom', enName: 'Custom', cnName: '自定义', operatorCode: 'cmcc', isBuildIn: false, description: 'GSM自定义指标', parentId: 'gsm-root', deviceType: 'GSM', indicatorCount: 1, children: [] },
];

const gnbChildren: IndicatorGroup[] = [
  { id: 'gnb-call', enName: 'Call', cnName: '呼叫类', operatorCode: 'cmcc', isBuildIn: true, description: 'NR RRC等呼叫相关指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 12, children: [] },
  { id: 'gnb-context', enName: 'Context', cnName: '上下文类', operatorCode: 'cmcc', isBuildIn: true, description: 'NR上下文管理相关指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 6, children: [] },
  { id: 'gnb-data', enName: 'Data', cnName: '数据类', operatorCode: 'cmcc', isBuildIn: true, description: 'NR数据传输相关指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 14, children: [] },
  { id: 'gnb-drb', enName: 'DRB', cnName: 'DRB类', operatorCode: 'cmcc', isBuildIn: true, description: 'NR DRB承载相关指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 8, children: [] },
  { id: 'gnb-ho', enName: 'HO', cnName: '切换类', operatorCode: 'cmcc', isBuildIn: true, description: 'NR切换相关指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 10, children: [] },
  { id: 'gnb-endc-mn', enName: 'ENDC-MN', cnName: 'ENDC主节点类', operatorCode: 'cmcc', isBuildIn: true, description: 'ENDC主节点相关指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 5, children: [] },
  { id: 'gnb-custom', enName: 'Custom', cnName: '自定义', operatorCode: 'cmcc', isBuildIn: false, description: 'NR自定义指标', parentId: 'gnb-root', deviceType: 'GNB', indicatorCount: 2, children: [] },
];

export const mockIndicatorGroups: Record<string, IndicatorGroup> = {
  ENB: {
    id: 'enb-root',
    enName: 'ENB',
    cnName: 'ENB功能集',
    operatorCode: 'cmcc',
    isBuildIn: true,
    description: 'LTE基站指标集',
    parentId: '',
    deviceType: 'ENB',
    indicatorCount: 65,
    children: enbChildren,
  },
  GSM: {
    id: 'gsm-root',
    enName: 'GSM',
    cnName: 'GSM功能集',
    operatorCode: 'cmcc',
    isBuildIn: true,
    description: 'GSM基站指标集',
    parentId: '',
    deviceType: 'GSM',
    indicatorCount: 25,
    children: gsmChildren,
  },
  GNB: {
    id: 'gnb-root',
    enName: 'GNB',
    cnName: 'GNB功能集',
    operatorCode: 'cmcc',
    isBuildIn: true,
    description: 'NR基站指标集',
    parentId: '',
    deviceType: 'GNB',
    indicatorCount: 57,
    children: gnbChildren,
  },
};

// ── Mock Indicators ──────────────────────────────────────────────────────────

export const mockIndicators: PerfIndicator[] = [
  // ENB - Call (Counter)
  { kpiId: 'RRC_CONN_REQ', kpiName: 'RRC连接请求次数', catagoryId: 'enb-call', catagoryName: '呼叫类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'RRC连接建立请求次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-20 10:30:00', calculatingStatus: '' },
  { kpiId: 'RRC_CONN_SUCC', kpiName: 'RRC连接成功次数', catagoryId: 'enb-call', catagoryName: '呼叫类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'RRC连接建立成功次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-20 10:30:00', calculatingStatus: '' },
  // ENB - Call (KPI)
  { kpiId: 'RRC_SR', kpiName: 'RRC连接成功率', catagoryId: 'enb-call', catagoryName: '呼叫类', productType: 'BBU', custName: '接入成功率', indicatorLevel: 'plmn', unit: '%', isCustomize: true, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[RRC_CONN_SUCC]/[RRC_CONN_REQ]*100', definition: 'RRC连接成功率', statisType: 'pct', updater: 'user1', updateTime: '2024-03-21 14:20:00', calculatingStatus: 'active' },
  // ENB - ERAB (Counter)
  { kpiId: 'ERAB_SETUP_REQ', kpiName: 'ERAB建立请求次数', catagoryId: 'enb-erab', catagoryName: 'ERAB类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: false, indicatorType: 'counter', arithmetic: '', definition: 'ERAB建立尝试次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-19 09:15:00', calculatingStatus: '' },
  { kpiId: 'ERAB_SETUP_SUCC', kpiName: 'ERAB建立成功次数', catagoryId: 'enb-erab', catagoryName: 'ERAB类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'ERAB建立成功次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-19 09:15:00', calculatingStatus: '' },
  // ENB - ERAB (KPI)
  { kpiId: 'ERAB_SR', kpiName: 'ERAB建立成功率', catagoryId: 'enb-erab', catagoryName: 'ERAB类', productType: 'BBU', custName: '', indicatorLevel: 'plmn', unit: '%', isCustomize: false, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[ERAB_SETUP_SUCC]/[ERAB_SETUP_REQ]*100', definition: 'ERAB建立成功率', statisType: 'pct', updater: 'admin', updateTime: '2024-03-19 09:15:00', calculatingStatus: 'active' },
  // ENB - HO
  { kpiId: 'HO_EXEC', kpiName: '切换执行次数', catagoryId: 'enb-ho', catagoryName: '切换类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: '切换执行次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-18 16:45:00', calculatingStatus: '' },
  { kpiId: 'HO_SUCC', kpiName: '切换成功次数', catagoryId: 'enb-ho', catagoryName: '切换类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: '切换成功次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-18 16:45:00', calculatingStatus: '' },
  { kpiId: 'HO_SR', kpiName: '切换成功率', catagoryId: 'enb-ho', catagoryName: '切换类', productType: 'BBU', custName: '', indicatorLevel: 'plmn', unit: '%', isCustomize: false, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[HO_SUCC]/[HO_EXEC]*100', definition: '切换成功率', statisType: 'pct', updater: 'admin', updateTime: '2024-03-18 16:45:00', calculatingStatus: 'active' },
  // ENB - Data
  { kpiId: 'DL_DATA_VOL', kpiName: '下行数据量', catagoryId: 'enb-data', catagoryName: '数据类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: 'MB', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: '下行数据传输量', statisType: 'sum', updater: 'admin', updateTime: '2024-03-17 11:00:00', calculatingStatus: '' },
  { kpiId: 'UL_DATA_VOL', kpiName: '上行数据量', catagoryId: 'enb-data', catagoryName: '数据类', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: 'MB', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: '上行数据传输量', statisType: 'sum', updater: 'admin', updateTime: '2024-03-17 11:00:00', calculatingStatus: '' },
  // ENB - Custom
  { kpiId: 'CUSTOM_ENB_001', kpiName: '自定义接入指标', catagoryId: 'enb-custom', catagoryName: '自定义', productType: 'BBU', custName: '我的接入指标', indicatorLevel: 'device', unit: '%', isCustomize: true, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[RRC_SR]+[ERAB_SR]', definition: '自定义接入类综合指标', statisType: 'avg', updater: 'user1', updateTime: '2024-03-22 08:30:00', calculatingStatus: 'active' },

  // GNB - Call
  { kpiId: 'NR_RRC_CONN_REQ', kpiName: 'NR RRC连接请求次数', catagoryId: 'gnb-call', catagoryName: '呼叫类', productType: 'AAU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'NR RRC连接请求次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-15 09:00:00', calculatingStatus: '' },
  { kpiId: 'NR_RRC_CONN_SUCC', kpiName: 'NR RRC连接成功次数', catagoryId: 'gnb-call', catagoryName: '呼叫类', productType: 'AAU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'NR RRC连接成功次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-15 09:00:00', calculatingStatus: '' },
  { kpiId: 'NR_RRC_SR', kpiName: 'NR RRC连接成功率', catagoryId: 'gnb-call', catagoryName: '呼叫类', productType: 'AAU', custName: '', indicatorLevel: 'plmn', unit: '%', isCustomize: false, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[NR_RRC_CONN_SUCC]/[NR_RRC_CONN_REQ]*100', definition: 'NR RRC连接成功率', statisType: 'pct', updater: 'admin', updateTime: '2024-03-15 09:00:00', calculatingStatus: 'active' },
  // GNB - Data
  { kpiId: 'DL_THROUGHPUT', kpiName: '下行吞吐量', catagoryId: 'gnb-data', catagoryName: '数据类', productType: 'AAU', custName: '', indicatorLevel: 'plmn', unit: 'Mbps', isCustomize: false, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[DL_DATA_VOL]/Duration', definition: '下行吞吐量', statisType: 'avg', updater: 'admin', updateTime: '2024-03-17 11:00:00', calculatingStatus: 'active' },
  { kpiId: 'UL_THROUGHPUT', kpiName: '上行吞吐量', catagoryId: 'gnb-data', catagoryName: '数据类', productType: 'AAU', custName: '', indicatorLevel: 'plmn', unit: 'Mbps', isCustomize: false, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[UL_DATA_VOL]/Duration', definition: '上行吞吐量', statisType: 'avg', updater: 'admin', updateTime: '2024-03-17 11:00:00', calculatingStatus: 'active' },
  // GNB - HO
  { kpiId: 'NR_HO_EXEC', kpiName: 'NR切换执行次数', catagoryId: 'gnb-ho', catagoryName: '切换类', productType: 'AAU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'NR切换执行次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-13 10:30:00', calculatingStatus: '' },
  // GNB - Custom
  { kpiId: 'CUSTOM_GNB_001', kpiName: '自定义吞吐量指标', catagoryId: 'gnb-custom', catagoryName: '自定义', productType: 'AAU', custName: '我的吞吐量指标', indicatorLevel: 'plmn', unit: 'Mbps', isCustomize: true, isCounter: false, isEnable: true, indicatorType: 'kpi', arithmetic: '[DL_THROUGHPUT]+[UL_THROUGHPUT]', definition: '自定义吞吐量综合指标', statisType: 'avg', updater: 'user1', updateTime: '2024-03-22 09:00:00', calculatingStatus: 'active' },

  // GSM - Call
  { kpiId: 'GSM_CALL_REQ', kpiName: 'GSM呼叫请求次数', catagoryId: 'gsm-call', catagoryName: '呼叫类', productType: 'RRU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'GSM呼叫请求次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-11 15:00:00', calculatingStatus: '' },
  { kpiId: 'GSM_CALL_SUCC', kpiName: 'GSM呼叫成功次数', catagoryId: 'gsm-call', catagoryName: '呼叫类', productType: 'RRU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'GSM呼叫成功次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-11 15:00:00', calculatingStatus: '' },
  // GSM - HO
  { kpiId: 'GSM_HO_EXEC', kpiName: 'GSM切换执行次数', catagoryId: 'gsm-ho', catagoryName: '切换类', productType: 'RRU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: false, isCounter: true, isEnable: true, indicatorType: 'counter', arithmetic: '', definition: 'GSM切换执行次数', statisType: 'sum', updater: 'admin', updateTime: '2024-03-10 11:00:00', calculatingStatus: '' },
];

// ── Mock Units ───────────────────────────────────────────────────────────────

export const mockIndicatorUnits: IndicatorUnit[] = [
  { id: 'u-1', enName: 'percent', cnName: '%' },
  { id: 'u-2', enName: 'times', cnName: '次' },
  { id: 'u-3', enName: 'Mbps', cnName: 'Mbps' },
  { id: 'u-4', enName: 'Gbps', cnName: 'Gbps' },
  { id: 'u-5', enName: 'ms', cnName: 'ms' },
  { id: 'u-6', enName: 'dBm', cnName: 'dBm' },
  { id: 'u-7', enName: 'dB', cnName: 'dB' },
  { id: 'u-8', enName: 'W', cnName: 'W' },
  { id: 'u-9', enName: 'MB', cnName: 'MB' },
  { id: 'u-10', enName: 'KB', cnName: 'KB' },
  { id: 'u-11', enName: 'bytes', cnName: 'bytes' },
  { id: 'u-12', enName: 'count', cnName: '个' },
  { id: 'u-13', enName: 'none', cnName: '无' },
];

// ── Mock Indicator Types ─────────────────────────────────────────────────────

export const mockIndicatorTypes: IndicatorType[] = [
  { id: 'counter', name: 'Counter', description: '原始计数器' },
  { id: 'kpi', name: 'KPI', description: '关键绩效指标（由公式计算）' },
];
