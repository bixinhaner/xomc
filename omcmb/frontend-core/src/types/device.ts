import type { AlarmSeverity } from './common';

// T-0162: ConnStatus 老类型保留过渡兼容；新代码请用 `lifecycleState` (业务进度)
// 与 `isOnline` (实时连接) 两个正交字段。详见
// docs/design/device-lifecycle-online-status-decouple-20260520.md
//
// 旧 `connStatus` 是"乐观归类"（5 种后端 status 全归 online），与筛选侧
// 只发 status=active 的语义不对称，造成 device list 页统计数与列表数不一致 bug。
export type ConnStatus = 'online' | 'offline';

// T-0162: 业务生命周期 6 状态，与后端 model.DeviceLifecycle 1:1。
export type DeviceLifecycle =
  | 'discovered'
  | 'registered'
  | 'provisioning'
  | 'commissioned'
  | 'maintenance'
  | 'decommissioned';

export type EngStatus = 'commissioned' | 'uncommissioned' | 'decommissioned';

export type MgmtStatus = 'managed' | 'unmanaged' | 'pre-managed';

// 运营商代码 — 与后端 global.CarrierCode 对齐（cmcc/ctcc/cucc）。
export type CarrierCode = 'cmcc' | 'ctcc' | 'cucc';

// 制式 — 与后端 model.Technology 对齐（lte/nr）。
export type DeviceTechnology = 'lte' | 'nr';

// 手动注册设备入参（与后端 device.CreateDeviceRequest 一一对应）。
// 字段语义：
//   - serialNumber / oui / carrier / technology 后端 required + oneof 校验
//   - productClass / manufacturer / modelName 自由文本，可空（CPE Bootstrap 时 RegisterFromInform 会用真实值 UPDATE 覆盖）
//   - 其余字段为站点 / 经纬度 / IP，可空
export interface CreateDeviceInput {
  serialNumber: string;
  oui: string;
  carrier: CarrierCode;
  technology: DeviceTechnology;
  productClass?: string;
  manufacturer?: string;
  modelName?: string;
  ipAddress?: string;
  deviceName?: string;
  siteId?: string;
  latitude?: number;
  longitude?: number;
}

// 批量导入单行数据（wire 层使用 snake_case，与后端 device.CreateDeviceRequest 一一对应）。
// 与 CreateDeviceInput 同语义，区别仅在序列化命名空间：
//   - CreateDeviceInput 走单次 create，由 deviceApi.create() 内部 camelCase → snake_case 手动映射；
//   - BatchImportDevice 走批量 POST，由 BatchImportModal 解析 CSV 后直接以 snake_case 提交，
//     避免每行做一次 mapping helper 调用。
export interface BatchImportDevice {
  serial_number: string;
  oui: string;
  carrier: CarrierCode;
  technology: DeviceTechnology;
  product_class?: string;
  manufacturer?: string;
  model_name?: string;
  ip_address?: string;
  device_name?: string;
  site_id?: string;
  latitude?: number;
  longitude?: number;
}

export interface BatchImportRequest {
  devices: BatchImportDevice[];
  /** 可选：导入后批量归入此分组（设备分组页操作上下文）。未传则保持未分组。 */
  group_id?: string;
}

// 单行错误回执（row 与 CSV 用户视角行号一致，1-based 不含 header）。
export interface BatchImportRowError {
  row: number;
  sn?: string;
  reason: string;
}

export interface BatchImportResponse {
  total: number;
  succeeded: number;
  failed: number;
  errors: BatchImportRowError[];
}

export interface Device {
  id: string;
  sn: string;
  name: string;
  vendor: string;
  productClass: string;
  networkType: string;
  deviceModel: string;
  region: string;
  stationId: string;
  /** 厂商 OUI（与后端 devices.oui 1:1）；批量导出 CSV 需要回写以支持 roundtrip 导入。 */
  oui: string;
  /** 运营商代码 'cmcc' / 'ctcc' / 'cucc'。同上 roundtrip 用。 */
  carrier: string;

  // T-0162: 新解耦字段
  lifecycleState: DeviceLifecycle;
  isOnline: boolean;

  // T-0162 DEPRECATED: 老 connStatus 派生自 isOnline；新代码请直接读 isOnline
  // 与 lifecycleState。保留过渡期到所有调用点迁完后整体删除。
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
  // T-0027 D9：分组归属来源（PRD §12.6 跨模块联合变更）
  // backend 通过 device_group_members.source_type 列传出（snake_case → camelCase 自动）
  // 'manual' = 用户手工指派；'rule' = 规则自动评估命中（D5.B SQL 守护已在后端）
  // optional 因后端列表 API 尚未全部 JOIN device_group_members 暴露此字段
  sourceType?: 'manual' | 'rule';
  sourceRuleId?: string; // 当 sourceType='rule' 时关联的 device_rules.id
  onlineTime: string;
  offlineTime: string;
  onlineDuration: number;
  upTime: string;
  firstOnlineTime: string;
  lastInformTime: string;
  deviceName: string;
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
  productClass?: string;
  /**
   * 设备分组匹配规则字段（来自后端 device_groups 表）。L2 子分组编辑入口
   * （DeviceGrouping/useGroupActions.tsx openEditLevel2）需要这些字段才能
   * 把原规则回填到表单 — 缺失就是 bug 入口（用户改"匹配规则"但表单显示空）。
   */
  matchingMode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber';
  nameRuleList?: NameFilterItem[];
  lacList?: number[];
  tacList?: number[];
}

export interface DeviceFilter {
  name?: string;
  /** 通用搜索文本（覆盖 SN/名称/IP/MAC 等） */
  searchText?: string;
  sn?: string;
  vendor?: string;
  productClass?: string;
  networkType?: string;
  /**
   * T-0162 DEPRECATED: 用 `lifecycleState` + `isOnline` 替代。
   * 保留过渡期供老 UI 代码工作。
   */
  connStatus?: ConnStatus;
  /** T-0162: 业务生命周期多选，对应后端 ?lifecycle_state= CSV 多选 */
  lifecycleState?: DeviceLifecycle[];
  /** T-0162: 实时在线，对应后端 ?is_online=true|false */
  isOnline?: boolean;
  alarmLevel?: AlarmSeverity | 'none';
  region?: string;
  subnet?: string;
  engStatus?: EngStatus;
  groupId?: string;
  /** 激活状态：'1'=激活，'0'=未激活 */
  opState?: string;
  /** 产品型号（如 PM-B4860, QAFA 等），对应后端 product_class */
  productModel?: string;
  /** T-0162: 设备型号（字典 device_model 提供下拉），后端 ?model_name= */
  modelName?: string;
  /** T-0162: 软件版本（字典 software_version 提供下拉），后端 ?software_version= */
  softwareVersion?: string;
  /** T-0162: 固件版本（字典 firmware_version 提供下拉），后端 ?firmware_version= */
  firmwareVersion?: string;
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

/**
 * 设备列表统计数据 — 基于筛选条件的全量统计（非当前页）。
 *
 * T-0162 解耦后字段命名：
 *   - online_count / offline_count: 实时在线/离线（is_online 双值统计）
 *   - by_lifecycle: 按业务生命周期分组的计数（commissioned/maintenance/...）
 *   - alarmed: 含 active 告警的设备数（占位字段，由后续 follow-up commit
 *     真正 JOIN alarms 表统计，当前固定 0）
 *   - online / offline: T-0162 DEPRECATED 别名，仍由后端兼容写入，新 UI 用
 *     online_count / offline_count
 */
export interface DeviceListStats {
  total: number;
  /** T-0162 DEPRECATED: 用 onlineCount 替代 */
  online?: number;
  /** T-0162 DEPRECATED: 用 offlineCount 替代 */
  offline?: number;
  /** T-0162: is_online=true 的设备数 */
  online_count: number;
  /** T-0162: is_online=false 的设备数 */
  offline_count: number;
  /** T-0162: 按 lifecycle_state 分组计数 */
  by_lifecycle?: Partial<Record<DeviceLifecycle, number>>;
  /** 有 active 告警的设备数（任意级别） */
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

/** 设备分组/规则 的"设备名匹配"条件项 */
export interface NameFilterItem {
  id: string;
  condition: 'contain' | 'notContain' | 'startWith' | 'endWith';
  value: string;
  andOr?: 'and' | 'or';
}
