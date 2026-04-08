import type { AlarmSeverity } from './common';

export type ConnStatus = 'online' | 'offline';

export type EngStatus = 'commissioned' | 'uncommissioned' | 'decommissioned';

export type MgmtStatus = 'managed' | 'unmanaged' | 'pre-managed';

export interface Device {
  id: string;
  sn: string;
  name: string;
  vendor: string;
  productType: string;
  networkType: string;
  deviceModel: string;
  region: string;
  stationId: string;
  connStatus: ConnStatus;
  alarmLevel: AlarmSeverity | 'none';
  engStatus: EngStatus;
  mgmtStatus: MgmtStatus;
  lastOnlineTime: string;
  ipAddress: string;
  subnet: string;
  site: string;
  longitude: number;
  latitude: number;
  softwareVersion: string;
  createTime: string;

  // --- 监控页面扩展字段 (来自 LTE/GSM/5G NR 三个监控页面合并) ---

  // 设备信息组 (Device)
  platformType: string;  // 平台类型标识 (Intel_CR_CA, MLN_CA, BaiBNX 等)
  hostName: string;
  productName: string;
  firmwareVersion: string;
  macAddress: string;
  groupName: string;
  onlineTime: string;
  offlineTime: string;
  onlineDuration: number;
  upTime: string;
  firstOnlineTime: string;
  lastInformTime: string;
  siteName: string;
  gpsVersion: string;
  rom: string;
  remark: string;
  gnbId: string;

  // 小区信息组 (Cell)
  enbId: string;
  cellId: string;
  eci: string;
  nrCellId: string;
  pci: string;
  plmnId: string;
  tac: string;
  subframeAssignment: string;
  specialSubframe: string;
  rootIndex: string;
  siteId: string;
  bandwidth: string;
  dlEarfcn: string;
  ulEarfcn: string;
  networkModel: string;
  txPower: string;
  band: string;
  lac: string;
  arfcn: string;
  uplinkFrequency: string;
  downlinkFrequency: string;

  // 状态信息组 (Status)
  opState: string;
  mmeStatus: string;
  amfStatus: string;
  rfStatus: string;
  pmReportStatus: string;
  halobFlag: boolean;
  syncStatus: string;
  validity: string;
  lockStatus: string;
  ueCount: number;
  euCount: string;
  ruCount: string;
  cpeCount: number;
  wanSpeed: string;
  serviceStatus: string;
  adminState: string;
  multiPlmnEnable: string;
  bscLinkStatus: string;
  bscSelect: string;
  bscSerialNumber: string;
  btsNum: number;

  // 网络信息组 (Network)
  ipsecAddr: string;
  mmepoolIpsecAddr: string;
  ipaUnitId: string;
  omlRemoteIp: string;
  omlRemoteIpBak: string;

  // 位置信息组 (Location)
  gpsHeight: number;
  mechanicalDowntilt: string;
  electronicDowntilt: string;
  verticalBeamWidth: string;
  horizontalAzimuth: string;
  installAddress: string;
  gpsSatelliteCount: number;

  // 5G NR 扩展 (Others)
  rollbackVersion: string;
  sasParam: string;
  euRu: string;
  halobLicense: string;
  energySaving: string;
  gnbTopoCellmgr: string;
  sslCertValidity: string;

  // 回收站扩展字段
  deletedAt?: string;
  deletedBy?: string;

  // ===== 离线时长（仅离线设备有值）=====

  /** 离线总秒数 */
  offlineSeconds?: number;
  /** 离线天数 */
  offlineDays?: number;
  /** 剩余小时数（0-23）*/
  offlineHours?: number;
  /** 剩余分钟数（0-59）*/
  offlineMinutes?: number;
}

export interface NE {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  vendor: string;
  region: string;
  subnet: string;
  site: string;
  connStatus: ConnStatus;
  alarmLevel: AlarmSeverity | 'none';
}

export interface DeviceGroup {
  id: string;
  name: string;
  parentId: string | null;
  deviceCount: number;
  description: string;
  /** 是否为内置设备组：1=内置, 0=自定义 */
  builtIn: number;
  /** 基站制式：LTE, 5G, GSM 等 */
  networkType?: string;
  /** 产品类型 */
  productType?: string;
}

export interface DeviceFilter {
  name?: string;
  /** 通用搜索文本（覆盖 SN/名称/IP/MAC 等） */
  searchText?: string;
  sn?: string;
  vendor?: string;
  productType?: string;
  networkType?: string;
  connStatus?: ConnStatus;
  alarmLevel?: AlarmSeverity | 'none';
  region?: string;
  subnet?: string;
  engStatus?: EngStatus;
  groupId?: string;
  /** 激活状态：'1'=激活，'0'=未激活 */
  opState?: string;
  /** 产品型号（如 PM-B4860, QAFA 等），对应后端 product_class */
  productModel?: string;
}

/** 设备统计数据 — 按状态分类的设备数量 */
export interface DeviceStats {
  counts: Record<string, number>;
}

/** 设备参数 — TR069 参数路径和值 */
export interface DeviceParameter {
  deviceId: string;
  parameterPath: string;
  parameterValue: string;
  parameterType: string;
  writable: boolean;
  lastUpdatedAt: string;
}

/** 设备列表统计数据 — 基于筛选条件的全量统计（非当前页） */
export interface DeviceListStats {
  total: number;
  online: number;
  offline: number;
  alarmed: number;
}

/** 设备列表响应 — 分页数据 + 筛选统计 */
export interface DeviceListResponse {
  items: Device[];
  total: number;
  page: number;
  pageSize: number;
  stats: DeviceListStats;
}
