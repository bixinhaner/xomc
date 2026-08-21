import http from '../http';
import type { Device, NE, DeviceFilter, DeviceGroup, DeviceListResponse, DeviceListStats, DeviceStats, DeviceParameter, CreateDeviceInput, NameFilterItem, BatchImportRequest, BatchImportResponse, BatchPreRegisterRequest, BatchPreRegisterResponse, LocationSync, ReportedLocation, DeviceControlSummary, DeviceControlActionHistoryList } from '../../types/device';
import type { AntennaSector } from '../../types/map';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { normalizeDeviceSyncStatus } from '../../utils/deviceSyncStatus';
import { normalizeAlarmSeverity } from '../../utils/alarmSeverity';

// Backend device model from Go struct
interface BackendDevice {
  id: string;
  serial_number: string;
  oui: string;
  product_class: string;
  device_type?: string;
  manufacturer: string;
  model_name: string;
  carrier: string;
  technology: string;
  data_model_id?: string;
  // T-0162 DEPRECATED: status 列已 DROP；后端读 lifecycle_state+is_online，scan
  // 后 populateDeviceCompat() 派生 status 回老前端读侧。新前端读 lifecycle_state
  // 和 is_online 字段，不要再用 status。
  status: string;
  // T-0162: 业务生命周期（与后端 model.DeviceLifecycle 1:1）
  lifecycle_state?: string;
  // T-0162: 实时在线
  is_online?: boolean;
  firmware_version: string;
  ip_address: string;
  connection_request_url: string;
  last_inform_at?: string;
  inform_interval: number;
  device_name: string;
  info_device_name?: string;
  site_id: string;
  latitude: number;
  longitude: number;
  location_source_mode?: 'tr069' | 'external';
  location_sync?: BackendLocationSync;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  deleted_by?: string;
  recycle_type?: 'manual' | 'auto';
  recycle_executor?: string;

  // --- 监控扩展字段 ---
  host_name?: string;
  product_name?: string;
  // T-XXX (Phase 5): 字段名对齐后端 DTO。mac_address / tx_power / gps_satellite_count
  // 是早期前端假定的字段名，后端实际返回的 json tag 为 mac / transmit_power / gps_satellites。
  // 旧 *? 字段保留作为兼容（Mock / 自填值场景），mapper 优先读正确字段。
  mac?: string;
  mac_address?: string;
  group_id?: string;
  group_name?: string;
  source_type?: 'manual' | 'rule' | 'auto';
  // T-XXX (Phase 0)：字段名对齐后端 DeviceWithInfo DTO（json tag），
  // 修复"其他信息组"接入/断开/首次接入/运行时长 5 字段全空白 bug。
  // 原 mapper 用的 `online_at / offline_at / first_online_at / up_time` 在后端从不存在。
  last_online_time?: string;
  last_offline_time?: string;
  first_online_time?: string;
  // run_time: 设备本次开机后运行秒数（来自 Device.DeviceInfo.UpTime），int64 not string
  run_time?: number;
  // online_duration: SQL 派生 — 当前在线/上次在线区间秒数
  online_duration?: number;
  // T-0173: OMC 视角累计在线时长（秒）。
  // 后端 device_info.cumulative_online_duration，由 DeviceStatusReconciler
  // 在 online→offline 边沿事务性累加 (NOW - last_online_time)。
  cumulative_online_duration?: number;
  // T-0173: 最近一次离线原因（诊断字段，可空）。
  // 取值：heartbeat_timeout / manual / reboot / null（从未离线或当前在线）
  last_offline_reason?: string | null;
  offline_seconds?: number;
  offline_days?: number;
  offline_hours?: number;
  offline_minutes?: number;
  gps_version?: string;
  rom?: string;
  remark?: string;
  gnb_id?: string;

  // Issue #758：设备名称同步
  name_sync_pending?: boolean;
  lmt_device_name?: string;
  param_sync_running?: boolean;
  last_param_sync_at?: string;

  // Cell
  enb_id?: string;
  cell_id?: string;
  eci?: string;
  nr_cell_id?: string;
  pci?: string;
  plmn_id?: string;
  tac?: string;
  subframe_assignment?: string;
  special_subframe?: string;
  root_index?: string;
  bandwidth?: string;
  dl_earfcn?: string;
  ul_earfcn?: string;
  // #177: 列表 List DTO 不输出 dl_earfcn（恒空），真实下行频点源是
  // device_info.freq_point（详情页 DeviceDetail/index.tsx:411 已用 freq_point）。
  // 列表 mapper 让 dlEarfcn 兜底到 freq_point，与详情页口径统一。
  freq_point?: string;
  network_model?: string;
  // T-XXX (Phase 5): transmit_power 后端 numeric (NUMERIC(8,2))；
  // tx_power 是旧字段名,保留兼容,mapper 优先读 transmit_power。
  transmit_power?: number;
  tx_power?: string;
  band?: string;
  lac?: string;
  arfcn?: string;
  uplink_frequency?: string;
  downlink_frequency?: string;

  // Status
  // #361: 告警级别来自后端 alarms_active 实时聚合（未 cleared 活动告警 MIN(severity)
  // 映成 critical/major/minor/warning 文本），无活动告警 → null → 前端归 'none'。
  // 原 mapper 写死 'none' 且 interface 未声明该字段，即便后端给值也被丢弃。
  alarm_severity?: string | null;
  // #361: 该设备未 cleared 活动告警数；无活动告警 → null → 前端归 0。
  active_alarm_count?: number | null;
  op_state?: string;
  control_summary?: BackendDeviceControlSummary | null;
  // T-2026-06-18: device_info.cell_status 透传（后端 CalcCellStatus 派生 ：任一 cell active → "active"，
  // 全部 inactive / 无 cell 数据 → "inactive"）。之前 T-0162 误以为后端不再透出此列。
  cell_status?: string;
  mme_status?: string;
  mme_pool?: Array<{
    index: number;
    ip?: string;
    status?: string;
    plmn_id?: string;
  }>;
  amf_status?: string;
  rf_status?: string;
  pm_report_status?: string;
  halob_enabled?: boolean;
  sync_status?: string;
  validity?: string;
  lock_status?: string;
  ue_count?: number;
  eu_count?: string;
  ru_count?: string;
  cpe_count?: number;
  wan_speed?: string;
  service_status?: string;
  admin_state?: string;
  multi_plmn_enable?: string;
  bsc_link_status?: string;
  bsc_select?: string;
  bsc_serial_number?: string;
  bts_num?: number;
  // GSM 设备详情「Status Info」截图新增字段。后端按设备是 BTS 时才下发,其余制式为 undefined.
  wan_link_status?: string;
  omc_status?: string;
  bsic?: string;
  vswr?: string;

  // Network
  ipsec_addr?: string;
  mmepool_ipsec_addr?: string;
  ipa_unit_id?: string;
  oml_remote_ip?: string;
  oml_remote_ip_bak?: string;

  // Location
  gps_height?: number | null;
  mechanical_downtilt?: string;
  electronic_downtilt?: string;
  vertical_beam_width?: string;
  horizontal_azimuth?: string;
  install_address?: string;
  device_address?: string;
  // T-XXX (Phase 5): gps_satellites 是后端实际字段（migration 000181 列名）；
  // gps_satellite_count 是旧前端假定名，保留兼容，mapper 优先 gps_satellites。
  gps_satellites?: number;
  gps_satellite_count?: number;

  // 5G NR Others
  rollback_version?: string;
  sas_param?: string;
  eu_ru?: string;
  halob_license?: string;
  energy_saving?: string;
  gnb_topo_cellmgr?: string;
  ssl_cert_validity?: string;
  ups_summary?: BackendUPSSummary | null;
}

interface BackendUPSSummary {
  external_ip?: string;
  total_voltage?: string;
  total_temperature?: string;
  total_current?: string;
  software_version?: string;
  hardware_version?: string;
  manufacturer?: string;
  manufacturer_oui?: string;
  up_time_seconds?: number;
  bms_charging?: string;
  ac_power?: string;
  ac_voltage?: string;
  dc_voltage?: string;
  dc_current?: string;
  board_temperature?: string;
  sfp_state?: string;
  port0_state?: string;
  port1_state?: string;
  port2_state?: string;
  port3_state?: string;
  average_soc?: number;
  pack_counts?: number;
  last_inform_at?: string;
}

interface BackendDeviceControlSummary {
  source_type: 'geofence';
  source_id?: string;
  source_name: string;
  reason_code: string;
  phase: DeviceControlSummary['phase'];
  action_id: string;
  triggered_at: string;
  completed_at?: string;
  last_error?: string;
}

interface BackendDeviceControlActionHistoryList {
  items: Array<{
    id: string;
    parent_action_id?: string;
    source_type: 'geofence';
    source_id?: string;
    source_name: string;
    reason_code: string;
    observation_version?: number;
    effective_state_version: number;
    action_type: 'activate' | 'deactivate';
    status: string;
    before_state: Array<{ path: string; value: string }>;
    requested_state: Array<{ path: string; value: string }>;
    verified_state: Array<{ path: string; value: string }>;
    evaluation?: {
      id: string;
      observation_version: number;
      latitude: number;
      longitude: number;
      observed_at: string;
      rule_type: string;
      signed_distance_meters?: number;
      confirmed_state: string;
      reason_code: string;
    };
    last_error?: string;
    created_at: string;
    updated_at: string;
    completed_at?: string;
  }>;
  total: number;
  page: number;
  page_size: number;
}

function mapDeviceControlSummary(summary?: BackendDeviceControlSummary | null): DeviceControlSummary | null {
  if (!summary) return null;
  return {
    sourceType: summary.source_type,
    sourceId: summary.source_id,
    sourceName: summary.source_name,
    reasonCode: summary.reason_code,
    phase: summary.phase,
    actionId: summary.action_id,
    triggeredAt: summary.triggered_at,
    completedAt: summary.completed_at,
    lastError: summary.last_error,
  };
}

interface BackendLocationSync {
  status: LocationSync['status'];
  accepted?: { latitude: number; longitude: number } | null;
  reported?: {
    latitude: number;
    longitude: number;
    gps_height?: number | null;
    observed_at: string;
    version: number;
    source_path: string;
  } | null;
  distance_meters?: number | null;
  height_diff_meters?: number | null;
}

function mapBackendLocationSync(value?: BackendLocationSync): LocationSync {
  if (!value) {
    return { status: 'no_report', accepted: null, reported: null, distanceMeters: null, heightDiffMeters: null };
  }
  const reported: ReportedLocation | null = value.reported
    ? {
        latitude: value.reported.latitude,
        longitude: value.reported.longitude,
        gpsHeight: value.reported.gps_height ?? null,
        observedAt: value.reported.observed_at,
        version: value.reported.version,
        sourcePath: value.reported.source_path,
      }
    : null;
  return {
    status: value.status,
    accepted: value.accepted ? { latitude: value.accepted.latitude, longitude: value.accepted.longitude } : null,
    reported,
    distanceMeters: value.distance_meters ?? null,
    heightDiffMeters: value.height_diff_meters ?? null,
  };
}

export interface ParameterSyncRequest {
  id: string;
  deviceId: string;
  deviceSn: string;
  triggerReason: string;
  syncScope: string;
  requestedPaths: string[];
  status: string;
  runId?: string;
  activeRunId?: string;
  resultCode?: string;
  errorMessage?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  updatedAt: string;
}

interface BackendParameterSyncRequest {
  id: string;
  device_id: string;
  device_sn: string;
  trigger_reason: string;
  sync_scope: string;
  requested_paths?: string[];
  status: string;
  run_id?: string;
  active_run_id?: string;
  result_code?: string;
  error_message?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  updated_at: string;
}

const PARAMETER_SYNC_TERMINAL_STATUSES = new Set([
  'succeeded',
  'failed',
  'timed_out',
  'cancelled',
  'deduplicated',
  'rejected',
]);

function mapParameterSyncRequest(req: BackendParameterSyncRequest): ParameterSyncRequest {
  return {
    id: req.id,
    deviceId: req.device_id,
    deviceSn: req.device_sn,
    triggerReason: req.trigger_reason,
    syncScope: req.sync_scope,
    requestedPaths: req.requested_paths ?? [],
    status: req.status,
    runId: req.run_id,
    activeRunId: req.active_run_id,
    resultCode: req.result_code,
    errorMessage: req.error_message,
    createdAt: req.created_at,
    startedAt: req.started_at,
    completedAt: req.completed_at,
    updatedAt: req.updated_at,
  };
}

function isParameterSyncTerminalStatus(status: string): boolean {
  return PARAMETER_SYNC_TERMINAL_STATUSES.has(status);
}

function createAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError');
  }
  const error = new Error('The operation was aborted.');
  error.name = 'AbortError';
  return error;
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) {
    throw createAbortError();
  }
}

async function delay(delayMs: number, signal?: AbortSignal): Promise<void> {
  if (!signal) {
    await new Promise<void>((resolve) => {
      globalThis.setTimeout(resolve, delayMs);
    });
    return;
  }

  await new Promise<void>((resolve, reject) => {
    const timerId = globalThis.setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, delayMs);

    const onAbort = () => {
      globalThis.clearTimeout(timerId);
      signal.removeEventListener('abort', onAbort);
      reject(createAbortError());
    };

    signal.addEventListener('abort', onAbort, { once: true });
  });
}

export interface BatchOperationResult {
  total: number;
  succeeded: number;
  failed: number;
  errors?: Array<{ id: string; message: string }>;
}

// #378: 回收站批量恢复结果——可部分成功。SN+carrier 与某活跃设备冲突的回收站行
// 被跳过（不再触发部分唯一索引 23505 整批回滚 500），其余正常恢复。
export interface RestoreConflict {
  id: string;
  serialNumber: string;
  reason: string;
}
export interface RestoreDevicesResult {
  restored: number;
  skipped: number;
  conflicts: RestoreConflict[];
}

export interface RenameDeviceResult {
  taskId?: string;
}

function buildUpdatedDeviceFallback(id: string, data: Partial<Device>, fallbackDevice?: Partial<Device>): Device {
  return {
    ...(fallbackDevice ?? {}),
    ...data,
    id,
    sn: data.sn ?? fallbackDevice?.sn ?? '',
    installAddress: data.installAddress ?? fallbackDevice?.installAddress ?? '',
    remark: data.remark ?? fallbackDevice?.remark ?? '',
  } as Device;
}

interface BackendAntennaSector {
  number: number;
  cell_id?: string;
  antenna_height?: number;
  mechanical_downtilt?: number;
  electronic_downtilt?: string;
  vertical_beamwidth?: number;
  horizontal_beamwidth?: number;
  azimuth?: number;
  near_radius_meters?: number;
  far_radius_meters?: number;
  field_sources?: Record<string, string>;
  direction_available?: boolean;
  coverage_available?: boolean;
  coverage_status?: 'available' | 'incomplete' | 'invalid_geometry';
  coverage_issue?: string;
  missing_fields?: string[];
}

export interface AntennaSectorPlanInput {
  azimuth?: number;
  antennaHeight?: number;
  mechanicalDowntilt?: number;
  horizontalBeamwidth?: number;
  verticalBeamwidth?: number;
}

function mapBackendAntennaSector(sector: BackendAntennaSector): AntennaSector {
  return {
    number: sector.number,
    cellId: sector.cell_id,
    antennaHeight: sector.antenna_height ?? undefined,
    mechanicalDowntilt: sector.mechanical_downtilt ?? undefined,
    electronicDowntilt: sector.electronic_downtilt,
    verticalBeamwidth: sector.vertical_beamwidth ?? undefined,
    horizontalBeamwidth: sector.horizontal_beamwidth ?? undefined,
    azimuth: sector.azimuth ?? undefined,
    nearRadiusMeters: sector.near_radius_meters ?? undefined,
    farRadiusMeters: sector.far_radius_meters ?? undefined,
    fieldSources: sector.field_sources ?? {},
    directionAvailable: sector.direction_available ?? false,
    coverageAvailable: sector.coverage_available ?? false,
    coverageStatus: sector.coverage_status
      ?? (sector.coverage_available ? 'available' : 'incomplete'),
    coverageIssue: sector.coverage_issue,
    missingFields: sector.missing_fields ?? [],
  };
}

// 后端 handler 返回的 JSON 形态（key 与 Go gin.H / 结构体 json tag 一致）。
interface BackendRestoreResult {
  restored?: number;
  skipped?: number;
  conflicts?: Array<{ id: string; serialNumber: string; reason: string }> | null;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
  // T-0162: 后端 service.ListDevicesWithInfo 在主查询同样筛选下跑 group-by
  // 聚合，填充本字段；前端直读 stats 而不是用 items.filter() 自行估算（修
  // Q2 分析里的"page-only 偏差"bug）。
  stats?: {
    total: number;
    by_lifecycle?: Record<string, number>;
    online_count: number;
    offline_count: number;
    current_ue_count?: number;
    alarmed: number;
  };
}

// T-0162: 老 mapStatus 已删除（"乐观归类"5 种 status 全归 online 与筛选侧
// 不对称引起 Q1 bug 的根因）。新 mapBackendDevice 直接读 bd.is_online
// 派生 connStatus，与后端语义 1:1。

function deriveLegacyLifecycle(status: string | undefined): Device['lifecycleState'] {
  switch (status) {
    case 'discovered':
    case 'registered':
    case 'provisioning':
    case 'maintenance':
    case 'decommissioned':
      return status;
    case 'active':
    case 'offline':
    default:
      return 'commissioned';
  }
}

// toRadioMode 把后端 technology 枚举（'lte' / 'nr' / 'gsm'）转换为
// BasicTab 期望的基站类型（'eNB' / 'gNB' / 'GSM'）。前端 BasicTab 内部
// 的"小区信息 / 状态信息（专属字段）"分组渲染条件依赖此枚举，否则整组
// 不显示。设计文档 §12。
function toRadioMode(technology: string): string {
  switch (technology) {
    case 'lte':
      return 'eNB';
    case 'nr':
      return 'gNB';
    case 'gsm':
      return 'GSM';
    default:
      return technology;
  }
}

function mapBackendUPSSummary(summary?: BackendUPSSummary | null): Device['upsSummary'] {
  if (!summary) return undefined;
  return {
    externalIp: summary.external_ip,
    totalVoltage: summary.total_voltage,
    totalTemperature: summary.total_temperature,
    totalCurrent: summary.total_current,
    softwareVersion: summary.software_version,
    hardwareVersion: summary.hardware_version,
    manufacturer: summary.manufacturer,
    manufacturerOui: summary.manufacturer_oui,
    upTimeSeconds: summary.up_time_seconds,
    bmsCharging: summary.bms_charging,
    acPower: summary.ac_power,
    acVoltage: summary.ac_voltage,
    dcVoltage: summary.dc_voltage,
    dcCurrent: summary.dc_current,
    boardTemperature: summary.board_temperature,
    sfpState: summary.sfp_state,
    port0State: summary.port0_state,
    port1State: summary.port1_state,
    port2State: summary.port2_state,
    port3State: summary.port3_state,
    averageSoc: summary.average_soc,
    packCounts: summary.pack_counts,
    lastInformAt: summary.last_inform_at,
  };
}

function mapBackendDevice(bd: BackendDevice): Device {
  // 兼容旧后端：若尚未升级到 T-0162 双字段，回退到 status 口径。
  const lifecycleState = (bd.lifecycle_state || deriveLegacyLifecycle(bd.status)) as Device['lifecycleState'];
  const isOnline = typeof bd.is_online === 'boolean' ? bd.is_online : bd.status === 'active';
  const offlineSeconds = bd.offline_seconds;
  let offlineDays = bd.offline_days;
  let offlineHours = bd.offline_hours;
  let offlineMinutes = bd.offline_minutes;

  if (
    offlineSeconds !== undefined &&
    (offlineDays === undefined || offlineHours === undefined || offlineMinutes === undefined)
  ) {
    const totalMinutes = Math.floor(Math.max(0, offlineSeconds) / 60);
    const totalHours = Math.floor(totalMinutes / 60);
    offlineDays = Math.floor(totalHours / 24);
    offlineHours = totalHours % 24;
    offlineMinutes = totalMinutes % 60;
  }

  const deviceType = bd.device_type || (bd.product_class?.startsWith('UPS') ? 'UPS' : 'BASE_STATION');
  const networkType = deviceType === 'UPS' ? 'UPS' : toRadioMode(bd.technology);
  // 基站沿用 devices.device_name 口径；UPS 没有设备侧改名/同步流程，
  // 优先使用后端从 device_ups_info 透出的 info_device_name 作为本地运维展示名。
  const friendlyName = deviceType === 'UPS'
    ? (bd.info_device_name || bd.device_name || bd.serial_number)
    : (bd.device_name || bd.serial_number);

  return {
    id: bd.id,
    sn: bd.serial_number,
    name: friendlyName,
    vendor: bd.manufacturer,
    productClass: bd.product_class,
    networkType,
    deviceType,
    upsSummary: mapBackendUPSSummary(bd.ups_summary),
    deviceModel: bd.model_name,
    region: bd.device_name,
    stationId: bd.site_id,
    oui: bd.oui ?? '',
    carrier: bd.carrier ?? '',
    locationSourceMode: bd.location_source_mode === 'external' ? 'external' : 'tr069',

    // T-0162 新字段
    lifecycleState,
    isOnline,

    // T-0162 DEPRECATED: 派生 connStatus（is_online → online/offline 直翻；
    // 不再用老 mapStatus 那种 5 种状态全归 online 的"乐观归类"做法）
    connStatus: isOnline ? 'online' : 'offline',

    // #361: 读后端 alarm_severity（alarms_active 实时聚合的文本），归一化到
    // AlarmSeverity | 'none'；不再无条件写死 'none'。
    alarmLevel: normalizeAlarmSeverity(bd.alarm_severity),
    activeAlarmCount: bd.active_alarm_count ?? 0,
    engStatus: 'commissioned',
    mgmtStatus: 'managed',
    // "最后在线" 表示 OMC 最近一次收到设备 Inform 的时间。
    // 本次连接时间使用 last_online_time，供"本次在线时长"计算。
    lastOnlineTime: bd.last_inform_at || '',
    ipAddress: bd.ip_address,
    subnet: '',
    site: bd.device_name,
    longitude: bd.longitude,
    latitude: bd.latitude,
    locationSync: mapBackendLocationSync(bd.location_sync),
    softwareVersion: bd.firmware_version,
    createTime: bd.created_at,

    // 监控扩展字段
    // 回收站/列表等页面部分列读取 hostName；后端主字段是 device_name，需兜底避免空白。
    hostName: bd.host_name || bd.device_name || bd.serial_number || '',
    productName: bd.product_name || '',
    firmwareVersion: bd.firmware_version || '',
    // T-XXX (Phase 5)：优先后端实际字段 mac，兜底旧 mac_address
    macAddress: bd.mac || bd.mac_address || '',
    groupId: bd.group_id || undefined,
    // 分组未命中时前端统一按空值展示占位符；这里保留字符串兜底，避免 undefined。
    groupName: bd.group_name || '',
    sourceType: bd.source_type,
    // T-XXX (Phase 0)：字段名对齐后端 DTO。设计文档 §13。
    // onlineTime = 本次连接时间；本次在线时长按它作为起点计算。
    onlineTime: bd.last_online_time || '',
    offlineTime: bd.last_offline_time || '',
    onlineDuration: bd.online_duration ?? null,
    upTime: bd.run_time ?? null,
    offlineSeconds,
    offlineDays,
    offlineHours,
    offlineMinutes,
    // T-0173: OMC 视角累计在线时长（秒）。后端 device_info.cumulative_online_duration。
    cumulativeOnlineDuration: bd.cumulative_online_duration ?? null,
    // T-0173: 最近一次离线原因（诊断字段)。空串 → null,与 onlineDuration 一致。
    lastOfflineReason: bd.last_offline_reason ?? null,
    firstOnlineTime: bd.first_online_time || '',
    lastInformTime: bd.last_inform_at || '',
    deviceName: friendlyName,
    gpsVersion: bd.gps_version || '',
    rom: bd.rom || '',
    remark: bd.remark || '',
    gnbId: bd.gnb_id || '',

    // Issue #758：设备名称同步
    nameSyncPending: bd.name_sync_pending ?? false,
    lmtDeviceName: bd.lmt_device_name || '',
    paramSyncRunning: bd.param_sync_running ?? false,
    lastParamSyncAt: bd.last_param_sync_at,

    enbId: bd.enb_id || '',
    cellId: bd.cell_id || '',
    eci: bd.eci || '',
    nrCellId: bd.nr_cell_id || '',
    pci: bd.pci || '',
    plmnId: bd.plmn_id || '',
    tac: bd.tac || '',
    subframeAssignment: bd.subframe_assignment || '',
    specialSubframe: bd.special_subframe || '',
    rootIndex: bd.root_index || '',
    siteId: bd.site_id || '',
    bandwidth: bd.bandwidth || '',
    // #177: dl_earfcn 在 List DTO 不输出（恒空），兜底到 freq_point，
    // 与详情页 DeviceDetail/index.tsx:411 (info.freqPoint || device.dlEarfcn) 口径统一。
    dlEarfcn: bd.dl_earfcn || bd.freq_point || '',
    ulEarfcn: bd.ul_earfcn || '',
    networkModel: bd.network_model || '',
    // T-XXX (Phase 5)：transmit_power 是后端实际字段 (NUMERIC 转 number)
    // 部分设备用 -1 表示未知/未上报，列表不应把占位值展示成真实 Tx Power。
    // tx_power 兼容旧字段名；fmtDuration 等渲染器接受 string，转字符串展示。
    txPower: bd.transmit_power != null && bd.transmit_power !== -1 ? String(bd.transmit_power) : (bd.tx_power || ''),
    band: bd.band || '',
    lac: bd.lac || '',
    arfcn: bd.arfcn || '',
    uplinkFrequency: bd.uplink_frequency || '',
    downlinkFrequency: bd.downlink_frequency || '',

    // T-2026-06-18: 后端 device_info.cell_status 重新透传（CalcCellStatus 派生：任一 cell active → "active"，
    // 全部 inactive / 无 cell 数据 → "inactive"）。之前 T-0162 误以为后端不再透出此列写死 ''，导致 BTS 详情页「小区状态」恒为 '-'。
    cellStatus: bd.cell_status || '',
    opState: bd.op_state || 'unknown',
    controlSummary: mapDeviceControlSummary(bd.control_summary),
    // device_info.mme_status 是后端按制式归一的核心网状态：LTE=MME，NR=AMF。
    // 这里按制式分流，避免 5G/2G 设备误显示 MME。
    mmeStatus: networkType === 'eNB' ? bd.mme_status || '' : '',
    mmePool: networkType === 'eNB'
      ? (bd.mme_pool || []).map((entry) => ({
          index: entry.index,
          ip: entry.ip || '',
          status: entry.status || '',
          plmnId: entry.plmn_id || '',
        }))
      : [],
    amfStatus: networkType === 'gNB' ? bd.amf_status || bd.mme_status || '' : '',
    rfStatus: bd.rf_status || '',
    pmReportStatus: bd.pm_report_status || '',
    halobFlag: bd.halob_enabled ?? false,
    syncStatus: normalizeDeviceSyncStatus(bd.sync_status),
    validity: bd.validity || '',
    lockStatus: bd.lock_status || '',
    ueCount: bd.ue_count ?? 0,
    euCount: bd.eu_count || '',
    ruCount: bd.ru_count || '',
    cpeCount: bd.cpe_count ?? 0,
    wanSpeed: bd.wan_speed || '',
    serviceStatus: bd.service_status || '',
    adminState: bd.admin_state || '',
    multiPlmnEnable: bd.multi_plmn_enable || '',
    bscLinkStatus: bd.bsc_link_status || '',
    bscSelect: bd.bsc_select || '',
    bscSerialNumber: bd.bsc_serial_number || '',
    btsNum: bd.bts_num ?? 0,
    // GSM 设备详情『Status Info』新增字段(截图友机 OMC 风格)。
    // 后端仅 BTS 下发,其余制式为 undefined → 前端回退 ''(表格展示 '-')。
    wanLinkStatus: bd.wan_link_status || '',
    omcStatus: bd.omc_status || '',
    bsic: bd.bsic || '',
    vswr: bd.vswr || '',

    ipsecAddr: bd.ipsec_addr || '',
    mmepoolIpsecAddr: bd.mmepool_ipsec_addr || '',
    ipaUnitId: bd.ipa_unit_id || '',
    omlRemoteIp: bd.oml_remote_ip || '',
    omlRemoteIpBak: bd.oml_remote_ip_bak || '',

    gpsHeight: bd.gps_height ?? null,
    mechanicalDowntilt: bd.mechanical_downtilt || '',
    electronicDowntilt: bd.electronic_downtilt || '',
    verticalBeamWidth: bd.vertical_beam_width || '',
    horizontalAzimuth: bd.horizontal_azimuth || '',
    installAddress: bd.install_address || bd.device_address || '',
    // T-XXX (Phase 5)：gps_satellites 是后端实际字段；gps_satellite_count 兜底
    gpsSatelliteCount: bd.gps_satellites ?? bd.gps_satellite_count ?? 0,

    rollbackVersion: bd.rollback_version || '',
    sasParam: bd.sas_param || '',
    euRu: bd.eu_ru || '',
    halobLicense: bd.halob_license || '',
    energySaving: bd.energy_saving || '',
    gnbTopoCellmgr: bd.gnb_topo_cellmgr || '',
    sslCertValidity: bd.ssl_cert_validity || '',

    // 回收站扩展字段
    deletedAt: bd.deleted_at,
    deletedBy: bd.deleted_by,
    recycleType: bd.recycle_type,
    recycleExecutor: bd.recycle_executor,
  };
}

function mapListResponse(resp: BackendListResponse<BackendDevice>): DeviceListResponse {
  const items = (resp.items || []).map(mapBackendDevice);
  // T-0162: 优先用后端 stats（T-0162 P3 已实装 backend 真返回）；fallback
  // 用当前页 items 估算是过渡兜底，新部署后绝大多数请求走前一路径。
  const stats: DeviceListStats = resp.stats
    ? {
        total: resp.stats.total,
        online_count: resp.stats.online_count,
        offline_count: resp.stats.offline_count,
        current_ue_count: resp.stats.current_ue_count,
        // T-0162 alias for backward compat
        online: resp.stats.online_count,
        offline: resp.stats.offline_count,
        by_lifecycle: resp.stats.by_lifecycle as DeviceListStats['by_lifecycle'],
        alarmed: resp.stats.alarmed,
      }
    : {
        total: resp.total,
        online_count: items.filter((d) => d.isOnline).length,
        offline_count: items.filter((d) => !d.isOnline).length,
        online: items.filter((d) => d.isOnline).length,
        offline: items.filter((d) => !d.isOnline).length,
        alarmed: items.filter((d) => d.alarmLevel !== 'none').length,
      };
  return {
    items,
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
    stats,
  };
}

export const deviceApi = {
  async getAntennaSectors(id: string): Promise<AntennaSector[]> {
    const { data } = await http.get<BackendAntennaSector[]>(`/devices/${id}/antenna-sectors`);
    return data.map(mapBackendAntennaSector);
  },

  async updateAntennaSectorPlan(
    id: string,
    sectorNumber: number,
    plan: AntennaSectorPlanInput,
  ): Promise<AntennaSector> {
    const { data } = await http.put<BackendAntennaSector>(
      `/devices/${id}/antenna-sectors/${sectorNumber}`,
      {
        azimuth: plan.azimuth,
        antenna_height: plan.antennaHeight,
        mechanical_downtilt: plan.mechanicalDowntilt,
        horizontal_beamwidth: plan.horizontalBeamwidth,
        vertical_beamwidth: plan.verticalBeamwidth,
      },
    );
    return mapBackendAntennaSector(data);
  },

  async getList(params: DeviceFilter & PageRequest): Promise<DeviceListResponse> {
    // Map frontend filter fields to backend query params
    const query: Record<string, unknown> = {
      page: params.page,
      page_size: params.pageSize,
      sort_by: params.sortField,
      sort_dir: params.sortOrder === 'ascend' ? 'asc' : params.sortOrder === 'descend' ? 'desc' : undefined,
    };

    if (params.name) query.search = params.name;
    if (params.searchText) query.search = params.searchText;
    if (params.sn) query.sn = params.sn;
    // 批量输入：SN 列表 → CSV，对应后端 ?sn_list=（serial_number IN (...)）。
    if (params.snList && params.snList.length > 0) query.sn_list = params.snList.join(',');
    if (params.vendor) query.oui = params.vendor;
    if (params.deviceType) query.device_type = params.deviceType;
    // productId → product_id（产品装配件 UUID 过滤，下拉来自 /products）
    if (params.productId) query.product_id = params.productId;
    // productClass → product_class
    if (params.productClass) query.product_class = params.productClass;
    // networkType 统一归一到 devices.technology canonical 值（lte/nr/gsm）。
    // 兼容字典值（lte/nr/gsm）与历史链路值（eNB/gNB/GSM，含大小写变体）。
    if (params.networkType) {
      const rawNetworkType = String(params.networkType).trim();
      const normalized = rawNetworkType.toLowerCase();
      let tech = rawNetworkType;
      if (normalized === 'enb' || normalized === 'lte') {
        tech = 'lte';
      } else if (normalized === 'gnb' || normalized === 'nr') {
        tech = 'nr';
      } else if (normalized === 'gsm') {
        tech = 'gsm';
      }
      query.technology = tech;
    }
    // T-0162: 新筛选维度，直接 1:1 传给后端，前端不再翻译
    if (params.lifecycleState && params.lifecycleState.length > 0) {
      query.lifecycle_state = params.lifecycleState.join(','); // CSV 多选
    }
    // isOnline 兼容三种来源：
    //   - 程序化调用：boolean true/false
    //   - 表单/字典选择：字符串 'true'/'false'（is_online 字典 value 即字符串）
    //   - URL deep-link：?isOnline=true 也是字符串
    // 类型定义里 isOnline 是 boolean，但运行时来自表单时是字符串 —— 不能只
    // 看 typeof boolean，否则字典选 "在线" 会被静默丢弃。
    if (params.isOnline === true || (params.isOnline as unknown) === 'true') {
      query.is_online = 'true';
    } else if (params.isOnline === false || (params.isOnline as unknown) === 'false') {
      query.is_online = 'false';
    }
    // 多选下拉 → CSV：FilterField type='multi-select' 返回数组，axios 默认
    // 把数组序列化为 key[]=a&key[]=b，gin 的 c.Query("model_name") 只认
    // 'model_name'，不认 'model_name[]'，过滤会被静默丢弃。统一 join 成
    // CSV，对应后端按 strings.Split(",") 解析（与 lifecycle_state 同模式）。
    const csv = (v: unknown): string | undefined => {
      if (Array.isArray(v)) return v.length > 0 ? v.join(',') : undefined;
      return v ? String(v) : undefined;
    };
    // groupId 多选 → CSV。FilterBar 的 multi-select 运行时返回数组，若直接透传
    // axios 会序列化成 group_id[]=a&group_id[]=b，后端 c.Query("group_id") 读不到。
    if (params.groupId) query.group_id = csv(params.groupId);
    if (params.modelName) query.model_name = csv(params.modelName);
    if (params.softwareVersion) query.software_version = csv(params.softwareVersion);
    if (params.firmwareVersion) query.firmware_version = csv(params.firmwareVersion);

    // T-0162 DEPRECATED: 老 connStatus 仍兼容，但仅在新字段都没传时才用
    // （新前端代码直接用 lifecycleState/isOnline）
    if (params.connStatus && typeof params.isOnline !== 'boolean' && !params.lifecycleState) {
      // connStatus '1'(在线) → is_online=true，'0'(离线) → is_online=false
      const onlineMap: Record<string, boolean> = {
        '1': true, '0': false, '2': false, '3': true,
        'online': true, 'offline': false,
      };
      const v = onlineMap[params.connStatus as string];
      if (typeof v === 'boolean') {
        query.is_online = v ? 'true' : 'false';
      }
    }
    if (params.opState) query.op_state = params.opState;
    if (params.controlSource) query.control_source = params.controlSource;
    if (params.controlPhase && params.controlPhase.length > 0) {
      query.control_phase = params.controlPhase.join(',');
    }
    // productModel / productClass 都映射到 product_class（前者是 multi-select，
    // 后者是历史单值字段），统一走 csv() 处理。
    if (params.productModel) query.product_class = csv(params.productModel);

    const { data } = await http.get<BackendListResponse<BackendDevice>>('/devices', {
      params: query,
    });
    return mapListResponse(data);
  },

  async getById(id: string): Promise<Device | null> {
    try {
      const { data } = await http.get<BackendDevice>(`/devices/${id}`);
      return mapBackendDevice(data);
    } catch {
      return null;
    }
  },

  async getControlActions(
    id: string,
    page = 1,
    pageSize = 20,
  ): Promise<DeviceControlActionHistoryList> {
    const { data } = await http.get<BackendDeviceControlActionHistoryList>(
      `/devices/${id}/control-actions`,
      { params: { page, page_size: pageSize } },
    );
    return {
      items: data.items.map((item) => ({
        id: item.id,
        parentActionId: item.parent_action_id,
        sourceType: item.source_type,
        sourceId: item.source_id,
        sourceName: item.source_name,
        reasonCode: item.reason_code,
        observationVersion: item.observation_version,
        effectiveStateVersion: item.effective_state_version,
        actionType: item.action_type,
        status: item.status,
        beforeState: item.before_state ?? [],
        requestedState: item.requested_state ?? [],
        verifiedState: item.verified_state ?? [],
        evaluation: item.evaluation ? {
          id: item.evaluation.id,
          observationVersion: item.evaluation.observation_version,
          latitude: item.evaluation.latitude,
          longitude: item.evaluation.longitude,
          observedAt: item.evaluation.observed_at,
          ruleType: item.evaluation.rule_type,
          signedDistanceMeters: item.evaluation.signed_distance_meters,
          confirmedState: item.evaluation.confirmed_state,
          reasonCode: item.evaluation.reason_code,
        } : undefined,
        lastError: item.last_error,
        createdAt: item.created_at,
        updatedAt: item.updated_at,
        completedAt: item.completed_at,
      })),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async acceptLocationSync(id: string, reportedVersion: number): Promise<LocationSync> {
    const { data } = await http.post<BackendLocationSync>(`/devices/${id}/location-sync/accept`, {
      reported_version: reportedVersion,
    });
    return mapBackendLocationSync(data);
  },

  async getBySn(sn: string): Promise<Device | null> {
    const result = await deviceApi.getList({
      sn,
      page: 1,
      pageSize: 1,
    });
    const listDevice = result.items.length > 0 ? result.items[0] : null;
    if (!listDevice?.id) return listDevice;

    const detailDevice = await deviceApi.getById(listDevice.id);
    if (!detailDevice) return listDevice;

    // 详情页按 SN 直达或浏览器刷新时没有列表页预热缓存。部分部署上的单设备详情
    // 接口可能不带部分 list/device_info 字段，不能让空值覆盖列表接口已经解析出的有效值。
    const merged: Device = { ...listDevice, ...detailDevice };
    for (const [key, value] of Object.entries(detailDevice) as Array<[keyof Device, unknown]>) {
      if (value === '' || value === null || value === undefined) {
        (merged as Record<keyof Device, unknown>)[key] = listDevice[key];
      }
    }
    return merged;
  },

  async create(input: CreateDeviceInput): Promise<Device> {
    // payload 字段名严格对齐后端 device.CreateDeviceRequest（device_handler.go）。
    // 后端 binding:"required,oneof=cmcc ctcc cucc"/oneof=lte nr，前端 dropdown 提交
    // 同一组枚举值，非法值会被 422 拦在 BE。
    const payload: Record<string, unknown> = {
      serial_number: input.serialNumber,
      oui: input.oui,
      carrier: input.carrier,
      technology: input.technology,
      product_class: input.productClass || undefined,
      manufacturer: input.manufacturer || undefined,
      model_name: input.modelName || undefined,
      ip_address: input.ipAddress || undefined,
      device_name: input.deviceName || undefined,
      site_id: input.siteId || undefined,
      latitude: input.latitude,
      longitude: input.longitude,
    };
    const { data: created } = await http.post<BackendDevice>('/devices', payload);
    return mapBackendDevice(created);
  },

  async update(id: string, data: Partial<Device>, fallbackDevice?: Partial<Device>): Promise<Device> {
    const devicePayload: Record<string, unknown> = {};
    const deviceInfoPayload: Record<string, unknown> = {};
    const isUPSUpdate = data.deviceType === 'UPS'
      || fallbackDevice?.deviceType === 'UPS'
      || data.networkType === 'UPS'
      || fallbackDevice?.networkType === 'UPS'
      || data.productClass?.startsWith('UPS')
      || fallbackDevice?.productClass?.startsWith('UPS');

    if (data.sn !== undefined) devicePayload.serial_number = data.sn;
    if (data.vendor !== undefined) devicePayload.manufacturer = data.vendor;
    if (data.productClass !== undefined) devicePayload.product_class = data.productClass;
    if (data.networkType !== undefined) devicePayload.technology = data.networkType;
    if (data.deviceModel !== undefined) devicePayload.model_name = data.deviceModel;
    if (data.connStatus !== undefined) devicePayload.status = data.connStatus === 'online' ? 'active' : 'offline';
    if (data.softwareVersion !== undefined) devicePayload.firmware_version = data.softwareVersion;
    if (data.ipAddress !== undefined) devicePayload.ip_address = data.ipAddress;
    if (data.site !== undefined) devicePayload.device_name = data.site;
    if (data.name !== undefined) {
      if (isUPSUpdate) deviceInfoPayload.device_name = data.name;
      else devicePayload.device_name = data.name;
    }
    if (data.stationId !== undefined) {
      if (isUPSUpdate) deviceInfoPayload.site_id = data.stationId;
      else devicePayload.site_id = data.stationId;
    }
    if (data.latitude !== undefined) devicePayload.latitude = data.latitude;
    if (data.longitude !== undefined) devicePayload.longitude = data.longitude;
    if (data.locationSourceMode !== undefined) devicePayload.location_source_mode = data.locationSourceMode;

    if (data.remark !== undefined) deviceInfoPayload.remark = data.remark;
    if (data.installAddress !== undefined) deviceInfoPayload.address = data.installAddress;
    if (data.deviceName !== undefined) deviceInfoPayload.device_name = data.deviceName;

    if (Object.keys(devicePayload).length > 0) {
      await http.put<BackendDevice>(`/devices/${id}`, devicePayload);
    }
    if (Object.keys(deviceInfoPayload).length > 0) {
      await http.put(`/devices/${id}/info`, deviceInfoPayload);
    }

    try {
      const { data: updated } = await http.get<BackendDevice>(`/devices/${id}`);
      return mapBackendDevice(updated);
    } catch {
      if (Object.keys(devicePayload).length > 0 || Object.keys(deviceInfoPayload).length > 0) {
        return buildUpdatedDeviceFallback(id, data, fallbackDevice);
      }
      throw new Error(`device ${id} update was skipped`);
    }
  },

  async updateInfo(id: string, data: { deviceName?: string; siteId?: string; remark?: string; address?: string }): Promise<void> {
    const payload: Record<string, unknown> = {};
    if (data.deviceName !== undefined) payload.device_name = data.deviceName;
    if (data.siteId !== undefined) payload.site_id = data.siteId;
    if (data.remark !== undefined) payload.remark = data.remark;
    if (data.address !== undefined) payload.address = data.address;
    if (Object.keys(payload).length === 0) return;
    await http.put(`/devices/${id}/info`, payload);
  },

  async delete(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.delete<BatchOperationResult>('/devices/batch', { data: { ids } });
    return data;
  },

  async batchReboot(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.post<BatchOperationResult>('/devices/batch-reboot', { ids });
    return data;
  },

  // 批量导入：前端解析 CSV → 结构化 JSON → 后端逐行 CreateDevice。
  // payload 字段已是 wire 层 snake_case（与 device.CreateDeviceRequest 对齐），
  // axios 请求拦截器只转 query params，不动 body，因此这里不要再写 camelCase。
  async batchImportDevices(payload: BatchImportRequest): Promise<BatchImportResponse> {
    const { data } = await http.post<BatchImportResponse>('/devices/batch-import', payload);
    return data;
  },

  /**
   * 批量预登记：在设备 Bootstrap 到达前，按 SN 列表预先录入设备名称。
   * 已存在的 SN 只更新名称/备注；不存在的 SN 新建设备（lifecycle='registered'）。
   * 新建设备由后端归入默认设备组或按规则归组，source_type 显示 Auto。
   */
  async batchPreRegisterDevices(payload: BatchPreRegisterRequest): Promise<BatchPreRegisterResponse> {
    const { data } = await http.post<BatchPreRegisterResponse>('/devices/batch-preregister', payload);
    return data;
  },

  async getGroups(): Promise<{ groups: DeviceGroup[]; stats: { totalDevices: number } }> {
    // Backend GET /device-groups/tree returns nested tree with device counts.
    // 我们摊平为列表给前端树构造器；同时**保留 matching rule 字段**（matching_mode /
    // name_rule_list / lac_list / tac_list），让 L2 编辑入口能回填原规则。
    interface BackendGroupItem {
      id: string;
      name: string;
      parent_id: string | null;
      device_count: number;
      description: string;
      remark: string;
      is_default: boolean;
      level: number;
      // i18n JSONB (migration 000003 + topology repo READ)
      name_i18n?: Record<string, string>;
      description_i18n?: Record<string, string>;
      remark_i18n?: Record<string, string>;
      // 匹配规则字段（service.go DeviceGroup 反序列化）
      matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber';
      source_group_id?: string;
      name_rule_list?: NameFilterItem[];
      lac_list?: number[];
      tac_list?: number[];
      serial_number_list?: string[];
      children?: BackendGroupItem[];
    }
    interface TreeResponse {
      items: BackendGroupItem[];
      stats?: { total_groups: number; grouped_devices: number; ungrouped_devices: number };
    }
    const { data } = await http.get<TreeResponse>('/device-groups/tree');
    const flat: DeviceGroup[] = [];
    function walk(items: BackendGroupItem[]) {
      for (const g of items) {
        flat.push({
          id: g.id,
          name: g.name,
          nameI18n: g.name_i18n,
          descriptionI18n: g.description_i18n,
          remarkI18n: g.remark_i18n,
          parentId: g.parent_id ?? null,
          deviceCount: g.device_count ?? 0,
          description: g.remark || g.description || '',
          builtIn: g.is_default ? 1 : 0,
          matchingMode: g.matching_mode,
          sourceGroupId: g.source_group_id,
          nameRuleList: g.name_rule_list,
          lacList: g.lac_list,
          tacList: g.tac_list,
          serialNumberList: g.serial_number_list,
        });
        if (g.children?.length) walk(g.children);
      }
    }
    walk(data.items || []);
    // 计算总设备数 = 已分组设备 + 历史无归属设备
    const totalDevices = (data.stats?.grouped_devices ?? 0) + (data.stats?.ungrouped_devices ?? 0);
    return { groups: flat, stats: { totalDevices } };
  },

  async createGroup(data: {
    name: string;
    /** i18n 三件套 (i18n B1 WRITE 路径)。{"zh-CN":"...","en-US":"..."}。 */
    name_i18n?: Record<string, string>;
    description_i18n?: Record<string, string>;
    remark_i18n?: Record<string, string>;
    parent_id?: string;
    remark?: string;
    matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber' | '';
    source_group_id?: string;
    name_rule_list?: NameFilterItem[];
    lac_list?: number[];
    tac_list?: number[];
    serial_number_list?: string[];
  }): Promise<DeviceGroup> {
    const { data: created } = await http.post<DeviceGroup>('/device-groups', data);
    return created;
  },

  async updateGroup(id: string, data: {
    name?: string;
    /** i18n 三件套 (i18n B1 WRITE 路径)。 */
    name_i18n?: Record<string, string>;
    description_i18n?: Record<string, string>;
    remark_i18n?: Record<string, string>;
    parent_id?: string;
    remark?: string;
    matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber' | '';
    source_group_id?: string;
    name_rule_list?: NameFilterItem[];
    lac_list?: number[];
    tac_list?: number[];
    serial_number_list?: string[];
  }): Promise<DeviceGroup> {
    const { data: updated } = await http.put<DeviceGroup>(`/device-groups/${id}`, data);
    return updated;
  },

  async deleteGroup(id: string): Promise<void> {
    await http.delete(`/device-groups/${id}`);
  },

  async moveDevices(params: { device_ids: string[]; target_group_id: string }): Promise<void> {
    await http.post('/device-groups/move-devices', params);
  },

  async addDevicesToGroup(groupId: string, deviceIds: string[]): Promise<void> {
    await http.post(`/device-groups/${groupId}/devices`, { device_ids: deviceIds });
  },

  async getStats(): Promise<DeviceStats> {
    const { data } = await http.get<DeviceStats>('/devices/stats');
    return data;
  },

  async getParameters(id: string): Promise<{ items: DeviceParameter[]; total: number }> {
    const { data } = await http.get<{ items: DeviceParameter[]; total: number }>(`/devices/${id}/parameters`);
    return data;
  },

  async reboot(id: string): Promise<void> {
    await http.post(`/devices/${id}/reboot`);
  },

  // 手动触发 durable paramsync 参数同步（reason="manual"）。
  // 后端当前复用 /devices/:id/sync-params 兼容端点；运行时由 paramSyncStarter 提交到 paramsync。
  // 后端 POST /api/v1/devices/:id/sync-params 响应 202 {status, source_id, device_id, serial_number, force}
  // force: 预留供未来节流绕过；当前 manual 端点天然不走节流。
  async syncDeviceParams(
    id: string,
    options?: { force?: boolean; parameterPaths?: string[] }
  ): Promise<{
    status: string;
    sourceId: string;
    requestId?: string;
    runId?: string;
    resultCode?: string;
    deviceId: string;
    serialNumber: string;
    force: boolean;
    parameterPathsCount?: number;
    taskCount?: number;
    gpvTaskCount?: number;
  }> {
    const { data } = await http.post<{
      status: string;
      source_id: string;
      request_id?: string;
      run_id?: string;
      result_code?: string;
      device_id: string;
      serial_number: string;
      force: boolean;
      parameter_paths_count?: number;
      gpv_task_count?: number;
    }>(`/devices/${id}/sync-params`, options
      ? { force: options.force, parameter_paths: options.parameterPaths }
      : {});
    return {
      status: data.status,
      sourceId: data.source_id,
      requestId: data.request_id,
      runId: data.run_id,
      resultCode: data.result_code,
      deviceId: data.device_id,
      serialNumber: data.serial_number,
      force: data.force,
      parameterPathsCount: data.parameter_paths_count,
      taskCount: data.gpv_task_count,
      gpvTaskCount: data.gpv_task_count,
    };
  },

  async getParameterSyncRequest(requestId: string): Promise<ParameterSyncRequest> {
    const { data } = await http.get<BackendParameterSyncRequest>(`/parameter-sync/requests/${requestId}`);
    return mapParameterSyncRequest(data);
  },

  async waitForParameterSyncRequest(
    requestId: string,
    options?: { timeoutMs?: number; intervalMs?: number; signal?: AbortSignal; onPoll?: (request: ParameterSyncRequest) => void },
  ): Promise<ParameterSyncRequest> {
    throwIfAborted(options?.signal);
    const timeoutMs = options?.timeoutMs ?? 10 * 60 * 1000;
    const intervalMs = options?.intervalMs ?? 2000;
    const deadlineAt = Date.now() + timeoutMs;

    while (true) {
      throwIfAborted(options?.signal);
      const request = await this.getParameterSyncRequest(requestId);
      options?.onPoll?.(request);
      if (isParameterSyncTerminalStatus(request.status)) {
        return request;
      }
      if (Date.now() >= deadlineAt) {
        throw new Error(`parameter sync request ${requestId} timed out`);
      }
      await delay(intervalMs, options?.signal);
    }
  },

  async getNEList(params: { keyword?: string } & PageRequest): Promise<PageResponse<NE>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.keyword) query.search = params.keyword;

    const { data } = await http.get<BackendListResponse<BackendDevice>>('/devices', {
      params: query,
    });
    return {
      items: (data.items || []).map((bd) => ({
        id: bd.id,
        neName: bd.device_name || bd.serial_number,
        sn: bd.serial_number,
        neType: bd.product_class,
        vendor: bd.manufacturer,
        region: bd.device_name,
        subnet: '',
        site: bd.device_name,
        // T-0162: 直接用 is_online 派生（与 mapBackendDevice 一致）
        connStatus: bd.is_online ? ('online' as const) : ('offline' as const),
        alarmLevel: 'none' as const,
      })),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getNEBySn(sn: string): Promise<NE | null> {
    const result = await deviceApi.getNEList({ keyword: sn, page: 1, pageSize: 1 });
    return result.items.length > 0 ? result.items[0] : null;
  },

  // ========== Recycle Bin APIs ==========

  async listRecycleBin(params: {
    search?: string;
    carrier?: string;
    technology?: string;
    group_id?: string;
    deleted_by?: string;
    page?: number;
    pageSize?: number;
    sortField?: string;
    sortOrder?: string;
  }): Promise<DeviceListResponse> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
      sortField: params.sortField,
      sortOrder: params.sortOrder,
    };
    if (params.search) query.search = params.search;
    if (params.carrier) query.carrier = params.carrier;
    if (params.technology) query.technology = params.technology;
    if (params.group_id) query.group_id = params.group_id;
    if (params.deleted_by) query.deleted_by = params.deleted_by;

    const { data } = await http.get<BackendListResponse<BackendDevice>>('/devices/recycle', {
      params: query,
    });
    return mapListResponse(data);
  },

  // #378: 返回 RestoreDevicesResult（恢复数 + 跳过数 + 冲突明细），可部分成功。
  async restoreDevices(ids: string[]): Promise<RestoreDevicesResult> {
    const { data } = await http.patch<BackendRestoreResult>('/devices/recycle/restore', { ids });
    return {
      restored: data?.restored ?? 0,
      skipped: data?.skipped ?? 0,
      conflicts: (data?.conflicts ?? []).map((c) => ({
        id: c.id,
        serialNumber: c.serialNumber,
        reason: c.reason,
      })),
    };
  },

  async permanentDeleteDevices(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.delete<BatchOperationResult>('/devices/recycle/permanent', { data: { ids } });
    return data;
  },

  async getProductClasses(): Promise<string[]> {
    const { data } = await http.get<string[]>('/devices/product-classes');
    return data;
  },

  // Issue #758: 设备名称同步 - 解决名称差异
  async resolveNameSync(deviceId: string, action: 'use_lmt' | 'use_omc' | 'ignore'): Promise<void> {
    await http.post(`/devices/${deviceId}/resolve-name-sync`, { action });
  },

  // 网管侧手动改基站名（即时下发）- 按 nameSyncMode 策略决定是否下发
  async renameDevice(deviceId: string, name: string): Promise<RenameDeviceResult> {
    const { data } = await http.post<{ task_id?: string }>(`/devices/${deviceId}/rename`, { name });
    return data?.task_id ? { taskId: data.task_id } : {};
  },
};
