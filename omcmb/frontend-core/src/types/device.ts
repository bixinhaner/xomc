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

// 批量导入单行数据（wire 层 snake_case，与后端 device.BatchImportDeviceRow 一一对应）。
//
// 产品语义：导入【只更新已注册设备】的 名称 / 备注，并把这批 SN 划入当前所选分组
// （不新建设备——设备由 TR-069 inform 注册，carrier 是不可变分区键无法凭 SN 新建）。
// 故仅需 serial_number（必填）+ device_name / remark（可选）。
export interface BatchImportDevice {
  serial_number: string;
  device_name?: string;
  remark?: string;
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
  /** 机器可读错误码（如 "device_not_found"），前端据此查 i18n；为空时降级展示 reason。 */
  errorCode?: string;
  reason: string;
}

export interface BatchImportResponse {
  total: number;
  succeeded: number;
  failed: number;
  errors: BatchImportRowError[];
}

// ─── 批量预登记 ──────────────────────────────────────────────────────────────

/** 单行预登记请求（对应后端 BatchPreRegisterRow）。 */
export interface BatchPreRegisterDevice {
  serial_number: string;
  device_name?: string;
  remark?: string;
  /** 运营商：cmcc / ctcc / cucc。为空时后端从 SN 前6位推断 OUI → CarrierRegistry。 */
  carrier?: 'cmcc' | 'ctcc' | 'cucc';
  /** 制式：lte / nr。为空时后端默认 lte。 */
  technology?: 'lte' | 'nr';
  /** OUI（可选）。为空时后端取 SN 前6位。 */
  oui?: string;
}

export interface BatchPreRegisterRequest {
  devices: BatchPreRegisterDevice[];
}

/** 单行预登记结果（后端 BatchPreRegisterRowResult）。 */
export interface BatchPreRegisterRowResult {
  row: number;
  sn: string;
  action?: 'created' | 'updated' | 'skipped';
  error_code?: string;
  reason?: string;
}

export interface BatchPreRegisterResponse {
  total: number;
  created: number;
  updated: number;
  failed: number;
  errors: BatchPreRegisterRowResult[];
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
  locationSourceMode?: 'tr069' | 'external';

  // T-0162: 新解耦字段
  lifecycleState: DeviceLifecycle;
  isOnline: boolean;

  // T-0162 DEPRECATED: 老 connStatus 派生自 isOnline；新代码请直接读 isOnline
  // 与 lifecycleState。保留过渡期到所有调用点迁完后整体删除。
  connStatus: ConnStatus;

  alarmLevel: AlarmSeverity | 'none';
  // #361: 该设备未 cleared 活动告警数（来自后端 alarms_active 聚合）；无告警 → 0。
  activeAlarmCount?: number;
  engStatus: EngStatus;
  mgmtStatus: MgmtStatus;
  lastOnlineTime: string;
  ipAddress: string;
  subnet: string;
  site: string;
  longitude: number | null;
  latitude: number | null;
  locationSync: LocationSync;
  softwareVersion: string;
  createTime: string;

  // --- 监控页面扩展字段 (来自 LTE/GSM/5G NR 三个监控页面合并) ---

  // 设备信息组 (Device)
  hostName: string;
  productName: string;
  firmwareVersion: string;
  macAddress: string;
  groupId?: string;
  groupName: string;
  // T-0027 D9：分组归属来源（PRD §12.6 跨模块联合变更）
  // backend 通过 device_group_members.source_type 列传出（snake_case → camelCase 自动）
  // 'manual' = 用户手工指派；'rule' = 规则匹配；'auto' = 系统默认组视图/自动归属
  // optional 因后端列表 API 尚未全部 JOIN device_group_members 暴露此字段
  sourceType?: 'manual' | 'rule' | 'auto';
  onlineTime: string;
  offlineTime: string;
  // T-XXX (Phase 0)：后端 SQL 派生秒数（NULL 表示设备从未上线/无法计算）。
  // 之前 `number` + mapper `?? 0` 会让"从未上线"显示为"0m"误导，改 null。
  onlineDuration: number | null;
  // T-XXX (Phase 0)：后端 device_info.run_time int64 秒（来自 Device.DeviceInfo.UpTime）。
  // 之前类型是 string + mapper 读不存在字段 → 显示空。
  upTime: number | null;
  // T-0173: OMC 视角累计在线时长（秒）。
  // 由 DeviceStatusReconciler 在 online→offline 边沿事务性累加 (NOW - last_online_time)。
  // 总在线时长 = cumulativeOnlineDuration + (is_online ? NOW - last_online_time : 0)。
  // null 表示设备从未上线 / 后端旧版无该字段。
  cumulativeOnlineDuration: number | null;
  // T-0173: 最近一次离线原因（诊断字段，仅 isOnline=false 时有意义)。
  // 取值：'heartbeat_timeout' | 'manual' | 'reboot' | null
  lastOfflineReason: string | null;
  firstOnlineTime: string;
  lastInformTime: string;
  deviceName: string;
  gpsVersion: string;
  rom: string;
  remark: string;
  gnbId: string;

  // Issue #758：设备名称同步
  nameSyncPending: boolean;
  lmtDeviceName: string;
  paramSyncRunning: boolean;
  /** 最近一次全量参数同步成功完成时间；manual / periodic / online-trigger 共用后端回写口径。 */
  lastParamSyncAt?: string;

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
  cellStatus: string;
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
  // GSM 设备详情『Status Info』新增字段（参考 BTS 友机 OMC 截图）。
  // 后端暂未提供时返回空串/0，前端渲染为 '-'。
  wanLinkStatus: string;   // WAN 链路状态 (Connected/Disconnected)
  omcStatus: string;       // OMC 连接状态 (Connected/Disconnected)
  bsic: string;            // GSM 基站识别码
  vswr: string;            // VSWR (例 "ch0:5.4 ch1:5.8")

  // 网络信息组 (Network)
  ipsecAddr: string;
  mmepoolIpsecAddr: string;
  ipaUnitId: string;
  omlRemoteIp: string;
  omlRemoteIpBak: string;

  // 位置信息组 (Location)
  gpsHeight: number | null;
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
  recycleType?: 'manual' | 'auto';
  recycleExecutor?: string;

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

export type LocationSyncStatus = 'no_report' | 'initialized' | 'in_sync' | 'pending';

export interface LocationCoordinate {
  latitude: number;
  longitude: number;
}

export interface ReportedLocation extends LocationCoordinate {
  gpsHeight?: number | null;
  observedAt: string;
  version: number;
  sourcePath: string;
}

export interface LocationSync {
  status: LocationSyncStatus;
  accepted: LocationCoordinate | null;
  reported: ReportedLocation | null;
  distanceMeters: number | null;
  heightDiffMeters: number | null;
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
  /** i18n JSONB 列(后端 migration 000003)。snake_case key 形如 'zh-CN' / 'en-US'。
   *  Axios camelCase 转换后字段名变 nameI18n,前端组件通过 useI18nText().fromRecord 自动兼容两种。 */
  nameI18n?: Record<string, string>;
  descriptionI18n?: Record<string, string>;
  remarkI18n?: Record<string, string>;
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
  sourceGroupId?: string;
  nameRuleList?: NameFilterItem[];
  lacList?: number[];
  tacList?: number[];
  serialNumberList?: string[];
}

export interface DeviceFilter {
  name?: string;
  /** 通用搜索文本（覆盖 SN/名称/IP/MAC 等） */
  searchText?: string;
  sn?: string;
  /** 批量输入：按 SN 列表精确过滤（对应后端 ?sn_list= CSV），与其它筛选条件正交 */
  snList?: string[];
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
  groupId?: string | string[];
  /** 激活状态：'1'=激活，'0'=未激活 */
  opState?: string;
  /** 产品装配件 UUID（下拉来自 /products），对应后端 ?product_id= → devices.product_id */
  productId?: string;
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
 *   - alarmed: active 告警总条数（任意级别）
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
  /** 当前在线设备上报的接入 UE 数之和；旧版后端可能暂不返回 */
  current_ue_count?: number;
  /** T-0162: 按 lifecycle_state 分组计数 */
  by_lifecycle?: Partial<Record<DeviceLifecycle, number>>;
  /** active 告警总条数（任意级别） */
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
