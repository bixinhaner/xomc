import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useTabStore } from '@core/store/tabStore';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import http from '@core/services/http';
import {
  Badge,
  Button,
  Card,
  Col,
  Descriptions,
  Dropdown,
  Empty,
  Radio,
  Row,
  Select,
  Skeleton,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  Alert,
  App,
} from 'antd';
import {
  ArrowLeftOutlined,
  MoreOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import StatusIndicator from '@/components/StatusIndicator';
import { useSyncStatus } from '@core/hooks/api/useDeviceParameters';
import { useDeviceBySn, useSyncDeviceParams } from '@core/hooks/api/useDevices';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useResolvedCellInstances } from '@core/hooks/api/useResolvedCellInstances';
import { useAcknowledgeAlarms, useClearAlarms, useCurrentAlarms, useUnacknowledgeAlarms } from '@core/hooks/api/useAlarms';
import { useAggregatedMetricsByDevices, useMetricObjects } from '@core/hooks/api/usePmQuery';
import { formatObjectLdn } from '@core/types/pmObject';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@core/types/alarm';
import type { Device } from '@core/types/device';
import { buildKpiCharts, buildKpiCompareData } from './kpiSeries';
import ParameterTreeTab from './ParameterTreeTab';
import QuickSettingsTab from './QuickSettingsTab';
import LicenseParamsTab from './LicenseParamsTab';
import { formatLteBandwidthDisplay } from './QuickSettingsTab/validators';
import AlarmDetail from '@/pages/alarm/AlarmDetail';
import ConfirmWithNoteModal from '@/pages/alarm/components/ConfirmWithNoteModal';

const { Title, Text } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

// ─── KPI 指标配置 ────────────────────────────────────────────────────────

interface KPIConfig {
  key: string;
  label: string;
  unit: string;
  category: string;
}

interface KPIConfigRaw {
  key: string;
  labelKey: string;
  unit: string;
  category: string;
}

// eNB KPI 配置 (8 项) —— key 为真实 K 编号（pm_metrics.metric_path），与 spec §3 LTE 映射表一致。
const ENB_KPI_CONFIG_RAW: KPIConfigRaw[] = [
  { key: 'K900010002', labelKey: 'kpi.K900010002', unit: '%', category: 'accessibility' },
  { key: 'K900010005', labelKey: 'kpi.K900010005', unit: '%', category: 'accessibility' },
  { key: 'K900010014', labelKey: 'kpi.K900010014', unit: '%', category: 'utilization' },
  { key: 'K900010013', labelKey: 'kpi.K900010013', unit: '%', category: 'utilization' },
  { key: 'K900010015', labelKey: 'kpi.K900010015', unit: '', category: 'traffic' },
  { key: 'K900010016', labelKey: 'kpi.K900010016', unit: '', category: 'traffic' },
  { key: 'K900010021', labelKey: 'kpi.K900010021', unit: '%', category: 'mobility' },
  { key: 'K900010027', labelKey: 'kpi.K900010027', unit: '%', category: 'retainability' },
];

// gNB KPI 配置 (4 项) —— 与 spec §3 NR/5G 映射表一致。
const GNB_KPI_CONFIG_RAW: KPIConfigRaw[] = [
  { key: 'KGNB0517', labelKey: 'kpi.KGNB0517', unit: '', category: 'traffic' },
  { key: 'KGNB0516', labelKey: 'kpi.KGNB0516', unit: '', category: 'traffic' },
  { key: 'KGNB0506', labelKey: 'kpi.KGNB0506', unit: '%', category: 'utilization' },
  { key: 'KGNB0505', labelKey: 'kpi.KGNB0505', unit: '%', category: 'utilization' },
];

// GSM KPI 配置 (3 项) —— 与 spec §3 GSM/2G 映射表一致。
const GSM_KPI_CONFIG_RAW: KPIConfigRaw[] = [
  { key: 'KGSM0102', labelKey: 'kpi.KGSM0102', unit: '%', category: 'accessibility' },
  { key: 'KGSM0103', labelKey: 'kpi.KGSM0103', unit: '%', category: 'retainability' },
  { key: 'KGSM0101', labelKey: 'kpi.KGSM0101', unit: '%', category: 'mobility' },
];

function translateKPIConfigs(raw: KPIConfigRaw[], t: (id: string) => string): KPIConfig[] {
  return raw.map((r) => ({ ...r, label: t(r.labelKey) }));
}

// 根据 networkType 获取 KPI 配置
const getKPIConfig = (networkType: string, t: (id: string) => string): KPIConfig[] => {
  switch (networkType) {
    case 'eNB':
      return translateKPIConfigs(ENB_KPI_CONFIG_RAW, t);
    case 'gNB':
      return translateKPIConfigs(GNB_KPI_CONFIG_RAW, t);
    case 'GSM':
      return translateKPIConfigs(GSM_KPI_CONFIG_RAW, t);
    default:
      return [];
  }
};

// 「按天/按周」→ 聚合查询粒度 + 时间窗（spec §4）。
//   按天 = hourly 最近 24 小时；按周 = daily 最近 7 天。
//   X 轴用返回行的真实 time（不再硬造标签）。
function kpiQueryWindow(mode: 'day' | 'week'): {
  granularity: 'hourly' | 'daily';
  startTime: string;
  endTime: string;
} {
  const now = new Date();
  const end = now.toISOString();
  if (mode === 'day') {
    const start = new Date(now.getTime() - 24 * 3_600_000).toISOString();
    return { granularity: 'hourly', startTime: start, endTime: end };
  }
  const start = new Date(now.getTime() - 7 * 86_400_000).toISOString();
  return { granularity: 'daily', startTime: start, endTime: end };
}

// X 轴时间桶 → 人类可读标签：hourly 显示「MM-DD HH:00」，daily 显示「MM-DD」。
function formatKpiAxisLabel(iso: string, granularity: 'hourly' | 'daily'): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  if (granularity === 'hourly') {
    const hh = String(d.getHours()).padStart(2, '0');
    return `${mm}-${dd} ${hh}:00`;
  }
  return `${mm}-${dd}`;
}

// 周期对比上一周期窗口（spec §11.1，长度守恒）：当前 [start,end]，L=end-start → 上周期 [start-L, start]。
// 复刻 dashboardFilterUtils.previousWindow 语义，这里直接用 ISO 串/毫秒实现（不引入 dayjs）。
function previousKpiWindow(startTime: string, endTime: string): { startTime: string; endTime: string } {
  const startMs = Date.parse(startTime);
  const endMs = Date.parse(endTime);
  const lengthMs = endMs - startMs;
  return {
    startTime: new Date(startMs - lengthMs).toISOString(),
    endTime: startTime,
  };
}

// tooltip 上一周期文案：「真实起 ~ 止」；缺结束时间只显示起点。粒度决定时分显示与否。
function formatCompareLabel(
  startIso: string,
  endIso: string,
  granularity: 'hourly' | 'daily',
): string | undefined {
  if (!startIso) return undefined;
  const start = formatKpiAxisLabel(startIso, granularity);
  const end = endIso ? formatKpiAxisLabel(endIso, granularity) : '';
  return end && end !== start ? `${start} ~ ${end}` : start;
}

// ─── 字段定义组件 ────────────────────────────────────────────────────────

interface FieldItem {
  key: string;
  label: string;
  render: (device: DetailDevice) => React.ReactNode;
}

interface FieldGroup {
  title: string;
  fields: FieldItem[];
}

interface CellRecord {
  key: string;
  index: number;
  values: Partial<Device>;
}

type DetailDevice = Device & {
  ppsTimeMode?: string;
};

interface DeviceDetailCell {
  index: number;
  cellId?: string;
  eci?: string;
  pci?: string;
  freqPoint?: string;
  bandwidth?: string;
  band?: string;
  opState?: string;
  rfTxStatus?: string;
  adminState?: string;
  lac?: string;
  arfcn?: string;
  btsNum?: number;
}

interface DeviceDetailInfo {
  deviceName?: string;
  remark?: string;
  eci?: string;
  pci?: string;
  cellId?: string;
  freqPoint?: string;
  bandwidth?: number;
  transmitPower?: number;
  plmn?: string;
  rfStatus?: string;
  mmeStatus?: string;
  syncStatus?: string;
  firstOnlineTime?: string;
  lastOnlineTime?: string;
  lastOfflineTime?: string;
  runTime?: number;
  cumulativeOnlineDuration?: number;
  tac?: string;
  band?: string;
  ulEarfcn?: string;
  subframeAssignment?: string;
  specialSubframe?: string;
  rootIndex?: string;
  gpsSatellites?: number;
  gpsHeight?: number;
  lockStatus?: string;
  enbId?: string;
  networkModel?: string;
  mac?: string;
  amfStatus?: string;
  multiPlmnEnable?: string;
  gpsVersion?: string;
  ppsTimeMode?: string;
  rollbackVersion?: string;
  wanSpeed?: string;
}

interface BackendDeviceDetailCell {
  index: number;
  cell_id?: string;
  eci?: string;
  pci?: string;
  freq_point?: string;
  bandwidth?: string;
  band?: string;
  op_state?: string;
  rf_tx_status?: string;
  admin_state?: string;
  lac?: string;
  arfcn?: string;
  bts_num?: number;
}

interface BackendDeviceDetailInfo {
  device_name?: string;
  remark?: string;
  eci?: string;
  pci?: string;
  cell_id?: string;
  freq_point?: string;
  bandwidth?: number;
  transmit_power?: number;
  plmn?: string;
  rf_status?: string;
  mme_status?: string;
  amf_status?: string;
  sync_status?: string;
  first_online_time?: string;
  last_online_time?: string;
  last_offline_time?: string;
  run_time?: number;
  cumulative_online_duration?: number;
  tac?: string;
  band?: string;
  ul_earfcn?: string;
  subframe_assignment?: string;
  special_subframe?: string;
  root_index?: string;
  gps_satellites?: number;
  gps_height?: number;
  lock_status?: string;
  enb_id?: string;
  network_model?: string;
  mac?: string;
  multi_plmn_enable?: string;
  gps_version?: string;
  pps_time_mode?: string;
  rollback_version?: string;
  wan_status?: string;
}

interface BackendDeviceDetailCompositeResponse {
  info?: BackendDeviceDetailInfo;
  cells?: BackendDeviceDetailCell[];
  gsm_cells?: BackendDeviceDetailCell[];
}

interface DeviceDetailCompositeResponse {
  info?: DeviceDetailInfo;
  cells?: DeviceDetailCell[];
  gsmCells?: DeviceDetailCell[];
}

function mapDeviceDetailInfo(info?: BackendDeviceDetailInfo): DeviceDetailInfo | undefined {
  if (!info) return undefined;

  return {
    deviceName: info.device_name,
    remark: info.remark,
    eci: info.eci,
    pci: info.pci,
    cellId: info.cell_id,
    freqPoint: info.freq_point,
    bandwidth: info.bandwidth,
    transmitPower: info.transmit_power,
    plmn: info.plmn,
    rfStatus: info.rf_status,
    mmeStatus: info.mme_status,
    amfStatus: info.amf_status,
    syncStatus: info.sync_status,
    firstOnlineTime: info.first_online_time,
    lastOnlineTime: info.last_online_time,
    lastOfflineTime: info.last_offline_time,
    runTime: info.run_time,
    cumulativeOnlineDuration: info.cumulative_online_duration,
    tac: info.tac,
    band: info.band,
    ulEarfcn: info.ul_earfcn,
    subframeAssignment: info.subframe_assignment,
    specialSubframe: info.special_subframe,
    rootIndex: info.root_index,
    gpsSatellites: info.gps_satellites,
    gpsHeight: info.gps_height,
    lockStatus: info.lock_status,
    enbId: info.enb_id,
    networkModel: info.network_model,
    mac: info.mac,
    multiPlmnEnable: info.multi_plmn_enable,
    gpsVersion: info.gps_version,
    ppsTimeMode: info.pps_time_mode,
    rollbackVersion: info.rollback_version,
    wanSpeed: info.wan_status,
  };
}

function mapDeviceDetailCell(cell: BackendDeviceDetailCell): DeviceDetailCell {
  return {
    index: cell.index,
    cellId: cell.cell_id,
    eci: cell.eci,
    pci: cell.pci,
    freqPoint: cell.freq_point,
    bandwidth: cell.bandwidth,
    band: cell.band,
    opState: cell.op_state,
    rfTxStatus: cell.rf_tx_status,
    adminState: cell.admin_state,
    lac: cell.lac,
    arfcn: cell.arfcn,
    btsNum: cell.bts_num,
  };
}

function mapDeviceDetailCompositeResponse(data: BackendDeviceDetailCompositeResponse): DeviceDetailCompositeResponse {
  return {
    info: mapDeviceDetailInfo(data.info),
    cells: data.cells?.map(mapDeviceDetailCell),
    gsmCells: data.gsm_cells?.map(mapDeviceDetailCell),
  };
}

function mergeDeviceDetailInfo(device: Device, info?: DeviceDetailInfo): DetailDevice {
  if (!info) return device;

  return {
    ...device,
    deviceName: info.deviceName || device.deviceName,
    remark: info.remark || device.remark,
    macAddress: info.mac || device.macAddress,
    gpsVersion: info.gpsVersion || device.gpsVersion,
    eci: info.eci || device.eci,
    pci: info.pci || device.pci,
    cellId: info.cellId || device.cellId,
    plmnId: info.plmn || device.plmnId,
    tac: info.tac || device.tac,
    subframeAssignment: info.subframeAssignment || device.subframeAssignment,
    specialSubframe: info.specialSubframe || device.specialSubframe,
    rootIndex: info.rootIndex || device.rootIndex,
    bandwidth: info.bandwidth != null ? String(info.bandwidth) : device.bandwidth,
    dlEarfcn: info.freqPoint || device.dlEarfcn,
    ulEarfcn: info.ulEarfcn || device.ulEarfcn,
    networkModel: info.networkModel || device.networkModel,
    txPower: info.transmitPower != null ? String(info.transmitPower) : device.txPower,
    band: info.band || device.band,
    mmeStatus: info.mmeStatus || device.mmeStatus,
    amfStatus: info.amfStatus || info.mmeStatus || device.amfStatus,
    rfStatus: info.rfStatus || device.rfStatus,
    syncStatus: info.syncStatus || device.syncStatus,
    lockStatus: info.lockStatus || device.lockStatus,
    multiPlmnEnable: info.multiPlmnEnable || device.multiPlmnEnable,
    firstOnlineTime: info.firstOnlineTime || device.firstOnlineTime,
    onlineTime: info.lastOnlineTime || device.onlineTime,
    offlineTime: info.lastOfflineTime || device.offlineTime,
    upTime: info.runTime ?? device.upTime,
    cumulativeOnlineDuration: info.cumulativeOnlineDuration ?? device.cumulativeOnlineDuration,
    gpsHeight: info.gpsHeight ?? device.gpsHeight,
    gpsSatelliteCount: info.gpsSatellites ?? device.gpsSatelliteCount,
    ppsTimeMode: info.ppsTimeMode,
    enbId: info.enbId || device.enbId,
    rollbackVersion: info.rollbackVersion || device.rollbackVersion,
    wanSpeed: info.wanSpeed || device.wanSpeed,
  };
}

function shouldShowGpsLocation(device: DetailDevice): boolean {
  return device.ppsTimeMode?.trim().toUpperCase().includes('GPS') ?? false;
}

interface CellSummaryColumn {
  title: string;
  key: string;
  dataIndex?: string[];
  width?: number;
}

type BmCellTech = 'LTE' | 'GSM';

const normalizeNetworkType = (networkType: string | undefined): string => {
  switch (networkType) {
    case 'lte':
      return 'eNB';
    case 'nr':
      return 'gNB';
    case 'gsm':
      return 'GSM';
    default:
      return networkType ?? '';
  }
};

const normalizeQuickSettingsNetworkType = (networkType: string | undefined): string => {
  switch (networkType) {
    case 'eNB':
      return 'lte';
    case 'gNB':
      return 'nr';
    case 'GSM':
      return 'gsm';
    default:
      return networkType ?? '';
  }
};

// 格式化时间
const fmtTime = (v: string | undefined | null) => (v ? new Date(v).toLocaleString('zh-CN') : '-');

// 格式化时长(秒)
const fmtDuration = (seconds: number | string | undefined | null) => {
  if (!seconds) return '-';
  const sec = typeof seconds === 'string' ? parseInt(seconds, 10) : seconds;
  if (isNaN(sec)) return '-';
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return d > 0 ? `${d}d ${h}h ${m}m` : h > 0 ? `${h}h ${m}m` : `${m}m`;
};

// T-0173: 计算"总在线时长"。
//   后端 cumulativeOnlineDuration 字段在 online→offline 边沿才事务性追加,
//   在线期间它不会随时间增长。展示时把"本次在线区间长度（NOW - onlineTime)
//   "加上,让用户看到的总时长持续变化。
//   onlineTime 是 ISO 字符串（device.last_online_time）。
const computeTotalOnline = (
  cumulative: number | null | undefined,
  isOnline: boolean | undefined,
  onlineTime: string | undefined,
): number | null => {
  const cum = typeof cumulative === 'number' ? cumulative : 0;
  if (!isOnline || !onlineTime) return cum > 0 ? cum : null;
  const start = Date.parse(onlineTime);
  if (Number.isNaN(start)) return cum > 0 ? cum : null;
  const currentSegment = Math.max(0, Math.floor((Date.now() - start) / 1000));
  const total = cum + currentSegment;
  return total > 0 ? total : null;
};

// 状态渲染
const renderStatusTag = (value: string | undefined, map: Record<string, { label: string; color: string }>) => {
  if (!value) return '-';
  const raw = value.trim();
  const entry = map[raw] ?? map[raw.toLowerCase()];
  if (!entry) return value;
  return <Tag color={entry.color}>{entry.label}</Tag>;
};

// ─── 基站信息组 ────────────────────────────────────────────────────────

const getStationFields = (t: ReturnType<typeof useT>, networkType: string): FieldGroup => {
  const fields: FieldItem[] = [
    // 公共字段
    { key: 'sn', label: t('device.sn'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.sn}</Text> },
    { key: 'name', label: t('device.hostName'), render: (d) => d.name || '-' },
    { key: 'networkType', label: t('device.radioMode'), render: (d) => <Tag color={{ eNB: 'blue', gNB: 'green', GSM: 'orange' }[d.networkType ?? '']}>{d.networkType || '-'}</Tag> },
    { key: 'productClass', label: t('device.productClass'), render: (d) => d.productClass || '-' },
    { key: 'deviceModel', label: t('device.model'), render: (d) => d.deviceModel || '-' },
    { key: 'softwareVersion', label: t('device.softwareVersion'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.softwareVersion || '-'}</Text> },
    { key: 'firmwareVersion', label: t('device.firmwareVersion'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.firmwareVersion || '-'}</Text> },
    { key: 'macAddress', label: t('device.macAddress'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.macAddress || '-'}</Text> },
    { key: 'groupName', label: t('device.groupName'), render: (d) => d.groupName || '-' },
    { key: 'ipAddress', label: t('device.ipAddress'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.ipAddress || '-'}</Text> },
  ];

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      { key: 'gpsVersion', label: t('device.gpsVersion'), render: (d) => d.gpsVersion ?? '-' },
    );
  }

  // gNB 独有字段
  if (networkType === 'gNB') {
    fields.push(
      { key: 'rollbackVersion', label: t('device.rollbackVersion'), render: (d) => d.rollbackVersion ?? '-' },
    );
  }

  // GSM 独有字段
  if (networkType === 'GSM') {
    fields.push(
      { key: 'ipaUnitId', label: 'IPA Unit ID', render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.ipaUnitId ?? '-'}</Text> },
      { key: 'omlRemoteIp', label: 'OML Remote IP', render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.omlRemoteIp ?? '-'}</Text> },
      { key: 'omlRemoteIpBak', label: 'OML Remote IP Bak', render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.omlRemoteIpBak ?? '-'}</Text> },
      { key: 'bscSelect', label: 'BSC Select', render: (d) => d.bscSelect === '0' ? t('device.bscPrimary') : d.bscSelect === '1' ? t('device.bscBackup') : d.bscSelect ?? '-' },
    );
  }

  return { title: t('device.group.station'), fields };
};

// ─── 小区信息组 ────────────────────────────────────────────────────────

const getCellFields = (t: ReturnType<typeof useT>, networkType: string): FieldGroup => {
  const fields: FieldItem[] = [];

  // eNB/gNB 共享字段
  if (networkType === 'eNB' || networkType === 'gNB') {
    fields.push(
      { key: 'pci', label: 'PCI', render: (d) => d.pci ?? '-' },
      { key: 'tac', label: 'TAC', render: (d) => d.tac ?? '-' },
      { key: 'band', label: 'Band', render: (d) => d.band ?? '-' },
      { key: 'dlEarfcn', label: t('device.dlEarfcn'), render: (d) => d.dlEarfcn ?? '-' },
      { key: 'ulEarfcn', label: t('device.ulEarfcn'), render: (d) => d.ulEarfcn ?? '-' },
      { key: 'networkModel', label: t('device.networkModel'), render: (d) => d.networkModel ?? '-' },
      { key: 'txPower', label: 'Tx Power', render: (d) => d.txPower ?? '-' },
    );
  }

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      { key: 'enbId', label: 'eNodeB ID', render: (d) => d.enbId ?? '-' },
      { key: 'cellId', label: t('device.cellId'), render: (d) => d.cellId ?? '-' },
      { key: 'eci', label: 'ECI', render: (d) => d.eci ?? '-' },
      { key: 'plmnId', label: 'PLMN', render: (d) => d.plmnId ?? '-' },
      { key: 'subframeAssignment', label: t('device.subframeAssignment'), render: (d) => d.subframeAssignment ?? '-' },
      { key: 'specialSubframe', label: t('device.specialSubframe'), render: (d) => d.specialSubframe ?? '-' },
      { key: 'rootIndex', label: t('device.rootIndex'), render: (d) => d.rootIndex ?? '-' },
      { key: 'siteId', label: 'Site ID', render: (d) => d.siteId ?? '-' },
      { key: 'bandwidth', label: t('device.bandwidth'), render: (d) => formatLteBandwidthDisplay(d.bandwidth) },
    );
  }

  // gNB 独有字段
  if (networkType === 'gNB') {
    fields.push(
      { key: 'gnbId', label: 'gNB ID', render: (d) => d.gnbId ?? '-' },
      { key: 'nrCellId', label: 'NR Cell ID', render: (d) => d.nrCellId ?? '-' },
    );
  }

  // GSM 独有字段
  if (networkType === 'GSM') {
    fields.push(
      { key: 'lac', label: 'LAC', render: (d) => d.lac ?? '-' },
      { key: 'arfcn', label: t('device.arfcn'), render: (d) => d.arfcn ?? '-' },
      { key: 'uplinkFrequency', label: t('device.uplinkFrequency'), render: (d) => d.uplinkFrequency ? `${d.uplinkFrequency} MHz` : '-' },
      { key: 'downlinkFrequency', label: t('device.downlinkFrequency'), render: (d) => d.downlinkFrequency ? `${d.downlinkFrequency} MHz` : '-' },
      { key: 'btsNum', label: t('device.btsNum'), render: (d) => d.btsNum ?? '-' },
    );
  }

  return { title: t('device.group.cell'), fields };
};

// ─── 状态信息组 ────────────────────────────────────────────────────────

const getStatusFields = (t: ReturnType<typeof useT>, networkType: string): FieldGroup => {
  const fields: FieldItem[] = [
    // 公共字段
    { key: 'ueCount', label: t('device.ueCount'), render: (d) => d.ueCount ?? '-' },
    { key: 'syncStatus', label: t('device.syncStatus'), render: (d) => d.syncStatus || '-' },
  ];

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      { key: 'mmeStatus', label: t('device.mmeStatus'), render: (d) => d.mmeStatus ?? '-' },
      { key: 'lockStatus', label: t('device.lockStatus'), render: (d) => renderStatusTag(d.lockStatus, { locked: { label: t('status.locked'), color: 'warning' }, unlocked: { label: t('status.unlocked'), color: 'success' } }) },
      { key: 'wanSpeed', label: t('device.wanSpeed'), render: (d) => d.wanSpeed ?? '-' },
    );
  }

  // gNB 独有字段
  if (networkType === 'gNB') {
    fields.push(
      { key: 'mmeStatus', label: t('device.amfStatus'), render: (d) => d.mmeStatus ?? d.amfStatus ?? '-' },
      { key: 'wanSpeed', label: t('device.wanSpeed'), render: (d) => d.wanSpeed ?? '-' },
      { key: 'multiPlmnEnable', label: 'Multi PLMN', render: (d) => renderStatusTag(d.multiPlmnEnable, { enabled: { label: t('status.enabled'), color: 'success' }, disabled: { label: t('status.disabled'), color: 'default' } }) },
    );
  }

  // GSM 独有字段
  if (networkType === 'GSM') {
    fields.push(
      { key: 'bscLinkStatus', label: t('device.bscLinkStatus'), render: (d) => renderStatusTag(d.bscLinkStatus, { connected: { label: t('status.connected'), color: 'success' }, disconnected: { label: t('status.disconnected'), color: 'error' } }) },
      { key: 'bscSerialNumber', label: t('device.bscSerialNumber'), render: (d) => d.bscSerialNumber ?? '-' },
    );
  }

  return { title: t('device.group.status'), fields };
};

// ─── 其他信息组 ────────────────────────────────────────────────────────

const getOtherFields = (t: ReturnType<typeof useT>, networkType: string, device: DetailDevice): FieldGroup => {
  const fields: FieldItem[] = [
    // 时间信息
    { key: 'onlineTime', label: t('device.onlineTime'), render: (d) => fmtTime(d.onlineTime) },
    { key: 'offlineTime', label: t('device.offlineTime'), render: (d) => fmtTime(d.offlineTime) },
    { key: 'onlineDuration', label: t('device.onlineDuration'), render: (d) => fmtDuration(d.onlineDuration) },
    { key: 'upTime', label: t('device.upTime'), render: (d) => fmtDuration(d.upTime) },
    // T-0173: 累计在线时长 = 后端 cumulative_online_duration + 本次在线区间。
    // 在线时由 DeviceStatusReconciler 在 online→offline 边沿才追加,所以前端补一个
    // 当前段 (NOW - onlineTime) 让展示精确到当前。
    { key: 'cumulativeOnlineDuration', label: t('device.cumulativeOnlineDuration'),
      render: (d) => fmtDuration(computeTotalOnline(d.cumulativeOnlineDuration, d.isOnline, d.onlineTime)) },
    // T-0173: 离线原因（仅离线时显示有意义,在线时也展示便于追溯上次掉线原因)。
    { key: 'lastOfflineReason', label: t('device.lastOfflineReason'),
      render: (d) => d.lastOfflineReason ? t(`device.lastOfflineReason.${d.lastOfflineReason}`) : '-' },
    { key: 'firstOnlineTime', label: t('device.firstOnlineTime'), render: (d) => fmtTime(d.firstOnlineTime) },
    { key: 'lastInformTime', label: t('device.lastInformTime'), render: (d) => fmtTime(d.lastInformTime) },
    // 站址信息
    { key: 'siteName', label: t('device.siteName'), render: (d) => d.deviceName || '-' },
  ];

  if (shouldShowGpsLocation(device)) {
    fields.push(
      { key: 'longitude', label: t('device.longitude'), render: (d) => d.longitude?.toFixed(4) || '-' },
      { key: 'latitude', label: t('device.latitude'), render: (d) => d.latitude?.toFixed(4) || '-' },
      { key: 'gpsHeight', label: t('device.gpsHeight'), render: (d) => d.gpsHeight ?? '-' },
    );
  }

  // GSM 独有字段
  if (networkType === 'GSM') {
    fields.push(
      { key: 'gpsSatelliteCount', label: t('device.gpsSatelliteCount'), render: (d) => d.gpsSatelliteCount ?? '-' },
    );
  }

  return { title: t('device.group.other'), fields };
};

// ─── 渲染字段组 ────────────────────────────────────────────────────────

const renderFieldGroup = (group: FieldGroup, device: DetailDevice) => (
  <Descriptions
    key={group.title}
    title={group.title}
    bordered
    column={{ xs: 1, sm: 2, md: 3, lg: 4 }}
    size="small"
    style={{ marginBottom: 16 }}
  >
    {group.fields.map((field) => (
      <Descriptions.Item key={field.key} label={field.label}>
        {field.render(device)}
      </Descriptions.Item>
    ))}
  </Descriptions>
);

const buildCellRecords = (device: Device, detailCells?: DeviceDetailCell[]): CellRecord[] => {
  if (Array.isArray(detailCells) && detailCells.length > 0) {
    return detailCells.map((cell, idx) => ({
      key: `${device.id}-cell-${cell.index || idx + 1}`,
      index: cell.index || idx + 1,
      values: {
        ...device,
        cellId: cell.cellId ?? cell.eci ?? '',
        nrCellId: cell.cellId ?? cell.eci ?? '',
        eci: cell.eci ?? '',
        pci: cell.pci ?? '',
        freqPoint: cell.freqPoint ?? '',
        bandwidth: cell.bandwidth ?? '',
        band: cell.band ?? device.band ?? '',
        opState: cell.opState ?? '',
        rfStatus: cell.rfTxStatus ?? '',
        adminState: cell.adminState ?? '',
        lac: cell.lac ?? device.lac ?? '',
        arfcn: cell.arfcn ?? device.arfcn ?? '',
        btsNum: cell.btsNum ?? device.btsNum ?? 0,
      },
    }));
  }

  const ext = device as Device & {
    cellList?: Array<Partial<Device>>;
    cells?: Array<Partial<Device>>;
    cellInfos?: Array<Partial<Device>>;
    cellInfoList?: Array<Partial<Device>>;
  };

  const source = ext.cellList ?? ext.cells ?? ext.cellInfos ?? ext.cellInfoList;
  const rows = Array.isArray(source) && source.length > 0 ? source : [device];

  return rows.map((row, idx) => ({
    key: String((row as { id?: string }).id ?? `${device.id}-cell-${idx + 1}`),
    index: idx + 1,
    values: row,
  }));
};

const renderCellOpState = (value: string | undefined, t: ReturnType<typeof useT>) =>
  renderStatusTag(value, {
    '1': { label: t('status.active'), color: 'success' },
    '0': { label: t('status.inactive'), color: 'error' },
    true: { label: t('status.active'), color: 'success' },
    false: { label: t('status.inactive'), color: 'error' },
    active: { label: t('status.active'), color: 'success' },
    inactive: { label: t('status.inactive'), color: 'error' },
  });

const renderCellRfStatus = (value: string | undefined, t: ReturnType<typeof useT>) =>
  renderStatusTag(value, {
    on: { label: t('status.rfOn'), color: 'success' },
    off: { label: t('status.rfOff'), color: 'error' },
    '1': { label: t('status.rfOn'), color: 'success' },
    '0': { label: t('status.rfOff'), color: 'error' },
    true: { label: t('status.rfOn'), color: 'success' },
    false: { label: t('status.rfOff'), color: 'error' },
  });

const renderCellAdminState = (
  value: string | undefined,
  networkType: string,
  t: ReturnType<typeof useT>,
) => {
  const normalizedNetworkType = normalizeNetworkType(networkType);

  if (normalizedNetworkType === 'gNB') {
    return renderStatusTag(value, {
      '1': { label: 'Locked', color: 'warning' },
      '2': { label: 'Unlocked', color: 'success' },
      '3': { label: 'ShuttingDown', color: 'error' },
      locked: { label: 'Locked', color: 'warning' },
      unlocked: { label: 'Unlocked', color: 'success' },
      shuttingdown: { label: 'ShuttingDown', color: 'error' },
    });
  }

  return renderStatusTag(value, {
    '1': { label: t('status.enabled'), color: 'success' },
    '0': { label: t('status.disabled'), color: 'default' },
    true: { label: t('status.enabled'), color: 'success' },
    false: { label: t('status.disabled'), color: 'default' },
    enabled: { label: t('status.enabled'), color: 'success' },
    disabled: { label: t('status.disabled'), color: 'default' },
  });
};

const getCellSummaryColumns = (networkType: string, t: ReturnType<typeof useT>): CellSummaryColumn[] => {
  const base: CellSummaryColumn[] = [
    { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
    { title: t('device.opState'), key: 'opState', width: 120 },
  ];

  switch (networkType) {
    case 'eNB':
      return [
        { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
        { title: t('device.cellId'), dataIndex: ['values', 'cellId'], key: 'cellId', width: 120 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.opState'), key: 'opState', width: 120 },
        { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
        { title: 'PCI', dataIndex: ['values', 'pci'], key: 'pci', width: 100 },
        { title: 'Freq Point', dataIndex: ['values', 'freqPoint'], key: 'freqPoint', width: 140 },
        { title: t('device.bandwidth'), dataIndex: ['values', 'bandwidth'], key: 'bandwidth', width: 120 },
        { title: 'band', dataIndex: ['values', 'band'], key: 'band', width: 100 },
      ];
    case 'gNB':
      return [
        { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
        { title: t('device.opState'), key: 'opState', width: 120 },
        { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
        { title: 'PCI', dataIndex: ['values', 'pci'], key: 'pci', width: 100 },
        { title: 'NRARFCN', dataIndex: ['values', 'freqPoint'], key: 'freqPoint', width: 140 },
        { title: 'band', dataIndex: ['values', 'band'], key: 'band', width: 100 },
        { title: 'NR Cell ID', dataIndex: ['values', 'nrCellId'], key: 'nrCellId', width: 160 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.bandwidth'), dataIndex: ['values', 'bandwidth'], key: 'bandwidth', width: 120 },
      ];
    case 'GSM':
      return [
        { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
        { title: t('device.cellId'), dataIndex: ['values', 'cellId'], key: 'cellId', width: 120 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.opState'), key: 'opState', width: 120 },
        { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
        { title: 'LAC', dataIndex: ['values', 'lac'], key: 'lac', width: 120 },
        { title: t('device.arfcn'), dataIndex: ['values', 'arfcn'], key: 'arfcn', width: 120 },
        { title: t('device.btsNum'), dataIndex: ['values', 'btsNum'], key: 'btsNum', width: 120 },
      ];
    default:
      return [
        ...base,
        { title: t('device.cellId'), dataIndex: ['values', 'cellId'], key: 'cellId', width: 120 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
        { title: 'PCI', dataIndex: ['values', 'pci'], key: 'pci', width: 100 },
        { title: 'band', dataIndex: ['values', 'band'], key: 'band', width: 100 },
      ];
  }
};

// ─── KPI Tab 组件─────────────────────────────────────────────────────────

interface KPITabContentProps {
  device: Device;
  t: ReturnType<typeof useT>;
}

function KPITabContent({ device, t }: KPITabContentProps) {
  const [timeMode, setTimeMode] = useState<'day' | 'week'>('day');
  // 下钻对象：'' = 设备级（全部）；否则为某 objectLdn。客户端侧按 objectLdn 过滤聚合行。
  const [objectLdn, setObjectLdn] = useState<string>('');

  const networkType = device.networkType ?? '';
  const kpiConfig = useMemo(() => getKPIConfig(networkType, t), [networkType, t]);
  const sn = device.sn;
  const technology = normalizeQuickSettingsNetworkType(networkType); // eNB→lte / gNB→nr / GSM→gsm

  const queryWindow = useMemo(() => kpiQueryWindow(timeMode), [timeMode]);
  const metricPaths = useMemo(() => kpiConfig.map((c) => c.key), [kpiConfig]);

  // 下钻对象清单（该设备 PM 数据里实际出现过的小区/PLMN）。
  const { data: metricObjects = [] } = useMetricObjects(
    sn ? [sn] : [],
    technology || undefined,
  );

  // 真实聚合查询：单设备传 [sn]，dimension=device、metricType=kpi、fillEmpty=true。
  const enabled = Boolean(sn) && kpiConfig.length > 0;
  const baseParams = useMemo(
    () => ({
      granularity: queryWindow.granularity,
      dimension: 'device' as const,
      metricPaths,
      metricType: 'kpi' as const,
      startTime: queryWindow.startTime,
      endTime: queryWindow.endTime,
      fillEmpty: true,
    }),
    [queryWindow.granularity, queryWindow.startTime, queryWindow.endTime, metricPaths],
  );
  const { data: rows, isLoading, isFetching, isError, refetch } =
    useAggregatedMetricsByDevices(baseParams, sn ? [sn] : [], enabled);

  // 周期对比（spec §11，常驻开启、无开关）：再发一个上一周期窗口查询，其余参数完全一致。
  const prevWindow = useMemo(
    () => previousKpiWindow(queryWindow.startTime, queryWindow.endTime),
    [queryWindow.startTime, queryWindow.endTime],
  );
  const prevParams = useMemo(
    () => ({ ...baseParams, startTime: prevWindow.startTime, endTime: prevWindow.endTime }),
    [baseParams, prevWindow.startTime, prevWindow.endTime],
  );
  // 原始平移毫秒（= 当前窗口长 L），对齐时吸附到整数粒度步长（防带零头整条虚线全空）。
  const offsetMs = useMemo(
    () => Date.parse(queryWindow.endTime) - Date.parse(queryWindow.startTime),
    [queryWindow.startTime, queryWindow.endTime],
  );
  const { data: prevRows } = useAggregatedMetricsByDevices(prevParams, sn ? [sn] : [], enabled);

  // 聚合行 → 每 K 编号一张图（纯函数，按 objectLdn 过滤、null 占位不画点）。
  const charts = useMemo(
    () => buildKpiCharts(rows, kpiConfig, objectLdn || null),
    [rows, kpiConfig, objectLdn],
  );
  // 上一周期图（同口径、同 objectLdn 过滤），再吸附对齐到当前周期 X 轴。
  const prevCharts = useMemo(
    () => buildKpiCharts(prevRows, kpiConfig, objectLdn || null),
    [prevRows, kpiConfig, objectLdn],
  );
  const compareDatas = useMemo(
    () =>
      charts.map((c, i) =>
        buildKpiCompareData(c, prevCharts[i], offsetMs, queryWindow.granularity),
      ),
    [charts, prevCharts, offsetMs, queryWindow.granularity],
  );

  if (kpiConfig.length === 0) {
    return (
      <div style={{ padding: '0 0 16px' }}>
        <Alert
          type="info"
          message={t('common.noData')}
          description={t('device.kpi.noConfig', { networkType: networkType || '-' })}
          showIcon
        />
      </div>
    );
  }

  const objectOptions = [
    { label: t('device.kpi.deviceLevel'), value: '' },
    ...metricObjects.map((o) => ({
      label: formatObjectLdn(o.objectLdn),
      value: o.objectLdn,
    })),
  ];

  return (
    <div style={{ padding: '0 0 16px' }}>
      {/* 工具栏：下钻对象选择 + 时间维度切换 + 刷新 */}
      <div
        style={{
          marginBottom: 16,
          display: 'flex',
          justifyContent: 'flex-end',
          alignItems: 'center',
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <Space size={4}>
          <Text type="secondary" style={{ fontSize: 13 }}>{t('device.kpi.object')}</Text>
          <Select
            size="small"
            value={objectLdn}
            options={objectOptions}
            onChange={setObjectLdn}
            style={{ minWidth: 200 }}
          />
        </Space>
        <Radio.Group
          value={timeMode}
          onChange={(e) => setTimeMode(e.target.value)}
          optionType="button"
          buttonStyle="solid"
          size="small"
        >
          <Radio.Button value="day">{t('device.kpi.byDay')}</Radio.Button>
          <Radio.Button value="week">{t('device.kpi.byWeek')}</Radio.Button>
        </Radio.Group>
        <Button
          size="small"
          icon={<ReloadOutlined />}
          loading={isFetching}
          onClick={() => refetch()}
        >
          {t('common.refresh')}
        </Button>
      </div>

      {/* 出错提示（可重试） */}
      {isError && (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={t('device.kpi.loadFailed')}
          action={
            <Button size="small" onClick={() => refetch()}>
              {t('common.retry')}
            </Button>
          }
        />
      )}

      {/* 每个 KPI 指标单独显示趋势图 */}
      <Row gutter={[16, 16]}>
        {kpiConfig.map((kpi, idx) => {
          const chart = charts[idx];
          const compare = compareDatas[idx];
          const xLabels = chart.xData.map((iso) => formatKpiAxisLabel(iso, queryWindow.granularity));
          // 上一周期 tooltip 文案（每点对应上周期真实起止），仅当上周期有数据时挂线。
          const hasCompare = !compare.isEmpty;
          const compareLabels = hasCompare
            ? compare.compareBuckets.map((s, i) =>
                formatCompareLabel(s, compare.compareBucketEnds[i] ?? '', queryWindow.granularity),
              )
            : undefined;
          const seriesName = chart.displayName || kpi.label;
          const lineSeries = hasCompare
            ? [
                { name: seriesName, data: chart.values },
                { name: `${seriesName}（上一周期）`, data: compare.values, dashed: true },
              ]
            : [{ name: seriesName, data: chart.values }];

          return (
            <Col key={kpi.key} xs={24} sm={12}>
              <div style={{
                borderRadius: 8,
                padding: '12px 12px 8px',
                border: '1px solid var(--color-gray-200, #f0f0f0)',
              }}>
                <div style={{
                  fontSize: 13,
                  fontWeight: 500,
                  color: 'var(--color-gray-700, #525252)',
                  marginBottom: 8,
                }}>
                  {chart.displayName || kpi.label}
                </div>
                {isLoading ? (
                  <Skeleton.Node active style={{ width: '100%', height: 220 }}>
                    <div style={{ width: 1, height: 220 }} />
                  </Skeleton.Node>
                ) : chart.isEmpty ? (
                  <div style={{
                    height: 220,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}>
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description={t('common.noData')}
                    />
                  </div>
                ) : (
                  <LineChart
                    title=""
                    xData={xLabels}
                    series={lineSeries}
                    compareLabels={compareLabels}
                    unit={kpi.unit || undefined}
                    height={220}
                    areaFill
                  />
                )}
              </div>
            </Col>
          );
        })}
      </Row>
    </div>
  );
}

// ─── 主组件 ─────────────────────────────────────────────────────────────

export default function DeviceDetail() {
  const t = useT();
  const { modal, message } = App.useApp();
  const { sn = '' } = useParams<{ sn: string }>();
  const detailTabKey = sn ? `device-detail:${sn}` : 'device-detail';
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const openTab = useTabStore((s) => s.openTab);
  const closeTab = useTabStore((s) => s.closeTab);
  const [searchParams, setSearchParams] = useSearchParams();

  const { data: device, isLoading, refetch } = useDeviceBySn(sn);
  const syncMutation = useSyncDeviceParams();
  const { data: paramSyncStatus, refetch: refetchParamSyncStatus } = useSyncStatus(device?.id ?? '');
  const [quickSettingsSyncPending, setQuickSettingsSyncPending] = useState(false);
  const quickSettingsSyncBaselineRef = useRef<{ lastParamSyncAt?: string; lastParamSyncFailedAt?: string }>({});
  const { data: detailComposite } = useQuery({
    queryKey: ['devices', 'detail-composite-v2', device?.id],
    queryFn: async () => {
      const { data } = await http.get<BackendDeviceDetailCompositeResponse>(`/devices/${device?.id}/detail`);
      return mapDeviceDetailCompositeResponse(data);
    },
    enabled: Boolean(device?.id),
  });

  const displayDevice = useMemo(() => {
    if (!device) return null;
    return mergeDeviceDetailInfo(device, detailComposite?.info);
  }, [detailComposite?.info, device]);

  useEffect(() => {
    if (!device?.id || !device.macAddress) return;
    void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
  }, [device?.id, device?.macAddress, queryClient]);

  const handleBackToList = useCallback(() => {
    openTab({
      key: 'device-list',
      label: 'nav.device.list',
      labelRaw: false,
      path: '/device/list',
      closable: true,
    });
    closeTab(detailTabKey);
    void navigate('/device/list');
  }, [closeTab, detailTabKey, navigate, openTab]);

  // 内部 tab 以 URL ?tab= 作为单一真相源 ——
  // 1) 离开详情页（组件卸载）再切回时，能从 URL 还原内部 tab，不丢状态；
  // 2) TabBar 上注册的详情 tab path 也带 ?tab=，确保切回时 navigate 的 URL 正确；
  // 3) 列表里点告警 Tag/GPS Link 跳 ?tab=alarm/gps 的深链接同样生效。
  const urlTab = searchParams.get('tab') ?? 'basic';
  const activeTab = urlTab;
  const setActiveTab = useCallback((key: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('tab', key);
      return next;
    }, { replace: true });
  }, [setSearchParams]);

  // 将设备详情页注册为按 SN 唯一的 TabBar 项。
  // 同一设备复用同 key，并用 path 刷新当前 ?tab=alarm/gps 等深链接；
  // 不同设备使用不同 key，允许从设备列表同时打开多个详情页。
  useEffect(() => {
    if (!sn) return;
    const displayName = device?.name || sn;
    openTab({
      key: detailTabKey,
      label: `${t('common.detail')} · ${displayName}`,
      labelRaw: true,
      path: `/device/detail/${sn}${location.search}`,
      closable: true,
    });
  }, [sn, detailTabKey, location.search, device?.name, openTab, t]);

  // 关闭当前设备详情页（用户点 × 关当前详情 tab）时清掉该设备的 form 草稿 + 反馈状态。
  // 与"切到其他 tab"区分：切走时当前设备 tab 仍存在；关闭后才被移除。
  // unmount 时检查 tabStore 当前 state，按存在性判断意图。
  const did = device?.id;
  useEffect(() => {
    if (!did) return;
    return () => {
      const hasTab = useTabStore.getState().tabs.some((tb) => tb.key === detailTabKey);
      if (!hasTab) {
        useQuickSettingsFeedbackStore.getState().clearByDevice(did);
      }
    };
  }, [detailTabKey, did]);

  // T-0138:快速设置 tab 显示规则 —— 只在该设备对应 paramModel 有 quicksettings XML 时显示
  // (后端 GET /quicksettings/groups?device_id=... 返回空 groups 即视为未配置)
  const {
    data: quickSettingsData,
    isLoading: quickSettingsLoading,
  } = useQuickSettingsGroups(device?.id);
  const showQuickSettingsTab = !quickSettingsLoading && (quickSettingsData?.groups?.length ?? 0) > 0;
  const [activeBmTech, setActiveBmTech] = useState<BmCellTech>('LTE');

  const detailQuickSettingsNetworkType = normalizeQuickSettingsNetworkType(displayDevice?.networkType);
  const isBmProduct = ((quickSettingsData?.paramModel
    ?? displayDevice?.productClass
    ?? device?.productClass
    ?? '')
    .trim()
    .toUpperCase()
    .startsWith('BM'));

  // 概览页小区列表的实例过滤规则与「快速设置」tab 完全一致，统一走 useResolvedCellInstances。
  const detailResolved = useResolvedCellInstances({
    deviceId: device?.id ?? '',
    networkType: detailQuickSettingsNetworkType,
    paramModel: quickSettingsData?.paramModel ?? '',
    bmTech: activeBmTech,
  });

  const bmCellTechOptions = useMemo<BmCellTech[]>(() => {
    if (!isBmProduct) return [];
    if (detailResolved.bmTechOptions.length > 0) return detailResolved.bmTechOptions;
    return ['LTE', 'GSM'];
  }, [detailResolved.bmTechOptions, isBmProduct]);

  useEffect(() => {
    if (!isBmProduct) {
      if (activeBmTech !== 'LTE') setActiveBmTech('LTE');
      return;
    }
    if (bmCellTechOptions.length === 0) return;
    if (!bmCellTechOptions.includes(activeBmTech)) {
      setActiveBmTech(bmCellTechOptions[0]);
    }
  }, [activeBmTech, bmCellTechOptions, isBmProduct]);

  const detailQuickSettingsCellInstances = detailResolved.instances;
  const detailQuickSettingsRuleReady = showQuickSettingsTab && detailResolved.ready;

  useEffect(() => {
    if (activeTab !== 'quickSettings' || !quickSettingsSyncPending) return;
    const timer = window.setInterval(() => {
      void refetchParamSyncStatus();
    }, 2000);
    return () => {
      window.clearInterval(timer);
    };
  }, [activeTab, quickSettingsSyncPending, refetchParamSyncStatus]);

  useEffect(() => {
    const deviceId = device?.id;
    if (!deviceId || activeTab !== 'quickSettings' || !quickSettingsSyncPending || !paramSyncStatus) return;
    if (paramSyncStatus.status === 'syncing') return;

    const { lastParamSyncAt: baselineSyncAt, lastParamSyncFailedAt: baselineFailedAt } = quickSettingsSyncBaselineRef.current;
    const hasNewSuccess = Boolean(
      paramSyncStatus.lastParamSyncAt && paramSyncStatus.lastParamSyncAt !== baselineSyncAt,
    );
    const hasNewFailure = Boolean(
      paramSyncStatus.lastParamSyncFailedAt && paramSyncStatus.lastParamSyncFailedAt !== baselineFailedAt,
    );

    if (!hasNewSuccess && !hasNewFailure) return;

    setQuickSettingsSyncPending(false);

    if (hasNewFailure && !hasNewSuccess) {
      message.error(paramSyncStatus.lastParamSyncError || '设备侧取数失败');
      return;
    }

    useQuickSettingsFeedbackStore.getState().clearByDevice(deviceId);
    useQuickSettingsFeedbackStore.getState().bumpRefreshTick(deviceId);
    void queryClient.invalidateQueries({ queryKey: ['devices', 'detail-composite-v2', deviceId] });
    void queryClient.invalidateQueries({ queryKey: ['quicksettings', 'groups', deviceId] });
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
    message.success('已获取设备侧最新数据');
  }, [activeTab, device?.id, message, paramSyncStatus, queryClient, quickSettingsSyncPending]);

  const handleHeaderRefresh = useCallback(() => {
    void refetch();
    const deviceId = device?.id;
    if (deviceId) {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'detail-composite-v2', deviceId] });
    }
    switch (activeTab) {
      case 'parameters':
        if (deviceId) {
          void queryClient.invalidateQueries({ queryKey: ['devices', 'object-tree', deviceId] });
          void queryClient.invalidateQueries({ queryKey: ['devices', 'children', deviceId] });
          void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', deviceId] });
        }
        break;
      case 'alarms':
        void queryClient.invalidateQueries({ queryKey: ['alarms', 'current'] });
        break;
      case 'quickSettings':
        if (deviceId) {
          quickSettingsSyncBaselineRef.current = {
            lastParamSyncAt: paramSyncStatus?.lastParamSyncAt,
            lastParamSyncFailedAt: paramSyncStatus?.lastParamSyncFailedAt,
          };
          setQuickSettingsSyncPending(true);
          syncMutation.mutate(
            { deviceId },
            {
              onSuccess: (data) => {
                message.success(`设备取数已入队（${data.sourceId}）`);
                void refetchParamSyncStatus();
              },
              onError: (err) => {
                setQuickSettingsSyncPending(false);
                const errMsg = err instanceof Error ? err.message : '设备取数触发失败';
                message.error(errMsg);
              },
            },
          );
        }
        break;
      default:
        break;
    }
  }, [activeTab, device?.id, message, paramSyncStatus?.lastParamSyncAt, paramSyncStatus?.lastParamSyncFailedAt, queryClient, refetch, refetchParamSyncStatus, syncMutation]);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);

  const [detailAlarm, setDetailAlarm] = useState<Alarm | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [ackTargetIds, setAckTargetIds] = useState<string[]>([]);
  const [ackModalOpen, setAckModalOpen] = useState(false);
  const [ackLoading, setAckLoading] = useState(false);
  const [clearTargetIds, setClearTargetIds] = useState<string[]>([]);
  const [clearModalOpen, setClearModalOpen] = useState(false);
  const [clearLoading, setClearLoading] = useState(false);

  const acknowledgeAlarms = useAcknowledgeAlarms();
  const unacknowledgeAlarms = useUnacknowledgeAlarms();
  const clearAlarms = useClearAlarms();
  const [alarmPage, setAlarmPage] = useState(1);
  const [alarmPageSize, setAlarmPageSize] = useState(20);

  const alarmParams = useMemo(
    () => ({ deviceSn: sn, page: alarmPage, pageSize: alarmPageSize } as Parameters<typeof useCurrentAlarms>[0]),
    [alarmPage, alarmPageSize, sn]
  );
  const { data: alarmData, isLoading: alarmsLoading, refetch: refetchAlarms } = useCurrentAlarms(alarmParams);
  const alarms: Alarm[] = alarmData?.items ?? [];

  useEffect(() => {
    setAlarmPage(1);
    setAlarmPageSize(20);
  }, [sn]);

  const handleAlarmPageChange = useCallback((page: number, size: number) => {
    setAlarmPage(page);
    setAlarmPageSize(size);
  }, []);

  const handleShowAlarmDetail = useCallback((alarm: Alarm) => {
    setDetailAlarm(alarm);
    setDetailOpen(true);
  }, []);

  const handleCloseAlarmDetail = useCallback(() => {
    setDetailOpen(false);
    setDetailAlarm(null);
  }, []);

  const handleAcknowledgeAlarm = useCallback((ids: string[]) => {
    setAckTargetIds(ids);
    setAckModalOpen(true);
  }, []);

  const handleAcknowledgeConfirm = useCallback(async (note: string) => {
    setAckLoading(true);
    try {
      await acknowledgeAlarms.mutateAsync({ ids: ackTargetIds, note });
      setAckModalOpen(false);
      await refetchAlarms();
      void message.success(t('common.ackSuccess'));
    } catch {
      void message.error(t('common.ackFailed'));
    } finally {
      setAckLoading(false);
    }
  }, [ackTargetIds, acknowledgeAlarms, message, refetchAlarms, t]);

  const handleUnacknowledgeAlarm = useCallback((ids: string[]) => {
    modal.confirm({
      title: t('alarm.unacknowledge'),
      content: t('common.unackConfirmMsg', { count: ids.length }),
      okText: t('common.confirm'),
      onOk: async () => {
        try {
          await unacknowledgeAlarms.mutateAsync(ids);
          await refetchAlarms();
          void message.success(t('common.unackSuccess'));
        } catch {
          void message.error(t('common.unackFailed'));
        }
      },
    });
  }, [message, modal, refetchAlarms, t, unacknowledgeAlarms]);

  const handleClearAlarm = useCallback((ids: string[]) => {
    setClearTargetIds(ids);
    setClearModalOpen(true);
  }, []);

  const handleClearConfirm = useCallback(async (note: string) => {
    setClearLoading(true);
    try {
      await clearAlarms.mutateAsync({ ids: clearTargetIds, note });
      setClearModalOpen(false);
      await refetchAlarms();
      void message.success(t('common.clearSuccess'));
    } catch {
      void message.error(t('common.clearFailed'));
    } finally {
      setClearLoading(false);
    }
  }, [clearAlarms, clearTargetIds, message, refetchAlarms, t]);

  const alarmColumns = useMemo(
    (): DataTableColumn<Alarm>[] => [
      {
        key: 'actions',
        title: t('common.operation'),
        width: 72,
        fixed: 'left',
        render: (_val, record) => {
          const isConfirmed = record.dealState === '1' || record.dealState === '3';
          return (
            <Dropdown
              trigger={['click']}
              menu={{
                items: [
                  { key: 'detail', label: t('common.detail') },
                  { key: 'ack', label: t(isConfirmed ? 'alarm.unacknowledge' : 'alarm.acknowledge') },
                  { key: 'clear', label: t('alarm.clear'), danger: true },
                ],
                onClick: ({ key, domEvent }) => {
                  domEvent.stopPropagation();
                  if (key === 'detail') handleShowAlarmDetail(record);
                  if (key === 'ack') {
                    if (isConfirmed) {
                      handleUnacknowledgeAlarm([record.id]);
                    } else {
                      handleAcknowledgeAlarm([record.id]);
                    }
                  }
                  if (key === 'clear') handleClearAlarm([record.id]);
                },
              }}
            >
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          );
        },
      },
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 90,
        render: (_val, record) => (
          <Tag color={SEVERITY_COLOR[record.severity] ?? 'default'}>
            {SEVERITY_LABEL[record.severity] ?? record.severity}
          </Tag>
        ),
      },
      { key: 'alarmIdentifier', title: t('alarm.code'), dataIndex: 'alarmIdentifier', width: 100, mono: true },
      { key: 'alarmName', title: t('alarm.name'), dataIndex: 'alarmName', width: 180, ellipsis: true },
      { key: 'description', title: t('alarm.possibleCause'), dataIndex: 'description', width: 220, ellipsis: true },
      {
        key: 'eventTime',
        title: t('alarm.time'),
        dataIndex: 'eventTime',
        width: 160,
        render: (_val, record) => new Date(record.eventTime).toLocaleString('zh-CN'),
      },
      {
        key: 'dealState',
        title: t('alarm.status'),
        dataIndex: 'dealState',
        width: 100,
        render: (_val, record) => (
          <Tag color={record.dealState === '1' || record.dealState === '3' ? 'success' : 'warning'}>
            {record.dealState === '1' || record.dealState === '3' ? t('alarm.dealState.confirmedUncleared') : t('alarm.dealState.unconfirmedUncleared')}
          </Tag>
        ),
      },
    ],
    [SEVERITY_LABEL, handleAcknowledgeAlarm, handleClearAlarm, handleShowAlarmDetail, handleUnacknowledgeAlarm, t]
  );

  // 根据设备制式获取字段组
  const detailGroups = useMemo((): FieldGroup[] => {
    if (!displayDevice) return [];
    const networkType = normalizeNetworkType(displayDevice.networkType);

    return [
      getStationFields(t, networkType),
      getStatusFields(t, networkType),
      getOtherFields(t, networkType, displayDevice),
    ];
  }, [displayDevice, t]);

  const cellGroup = useMemo((): FieldGroup | null => {
    if (!displayDevice) return null;
    const networkType = isBmProduct && activeBmTech === 'GSM'
      ? 'GSM'
      : normalizeNetworkType(displayDevice.networkType);
    return getCellFields(t, networkType);
  }, [activeBmTech, displayDevice, isBmProduct, t]);

  const displayCellNetworkType = useMemo(() => {
    if (isBmProduct) {
      return activeBmTech === 'GSM' ? 'GSM' : 'eNB';
    }
    return normalizeNetworkType(displayDevice?.networkType);
  }, [activeBmTech, displayDevice?.networkType, isBmProduct]);

  const activeDetailCells = isBmProduct && activeBmTech === 'GSM'
    ? detailComposite?.gsmCells
    : detailComposite?.cells;
  const useCompositeCellRecords = Array.isArray(activeDetailCells) && activeDetailCells.length > 0;

  const cellRecords = useMemo(() => {
    if (!displayDevice) return [];
    return buildCellRecords(displayDevice, useCompositeCellRecords ? activeDetailCells : undefined);
  }, [activeDetailCells, displayDevice, useCompositeCellRecords]);

  const displayCellRecords = useMemo(() => {
    if (!detailQuickSettingsRuleReady || !useCompositeCellRecords) {
      return cellRecords;
    }
    const allowed = new Set(detailQuickSettingsCellInstances);
    return cellRecords.filter((row) => allowed.has(Number(row.index)));
  }, [cellRecords, detailQuickSettingsCellInstances, detailQuickSettingsRuleReady, useCompositeCellRecords]);

  const cellColumns = useMemo(
    () => getCellSummaryColumns(displayCellNetworkType, t).map((column) => {
      // LTE 小区列表里的带宽列与详情字段、快速设置共享同一个枚举映射（仅 eNB）。
      const isLteBandwidth = column.key === 'bandwidth' && displayCellNetworkType === 'eNB';
      return {
        ...column,
        render: column.key === 'opState'
          ? (_: unknown, row: CellRecord) => renderCellOpState(row.values.opState as string | undefined, t)
          : column.key === 'adminState'
            ? (_: unknown, row: CellRecord) => renderCellAdminState(
              row.values.adminState as string | undefined,
              displayCellNetworkType,
              t,
            )
          : column.key === 'rfStatus'
            ? (_: unknown, row: CellRecord) => renderCellRfStatus(row.values.rfStatus as string | undefined, t)
          : isLteBandwidth
            ? (_: unknown, row: CellRecord) => formatLteBandwidthDisplay(row.values.bandwidth as string | undefined)
            : (value: string | number | undefined) => value ?? '-',
      };
    }),
    [displayCellNetworkType, t],
  );

  if (isLoading) {
    return (
      <div style={{ padding: 24 }}>
        <Skeleton active paragraph={{ rows: 8 }} />
      </div>
    );
  }

  if (!device || !displayDevice) {
    return (
      <div style={{ padding: 24 }}>
        <Alert
          type="error"
          message={t('common.noData')}
          description={`SN: "${sn}"`}
          action={
            <Button onClick={handleBackToList}>{t('common.back')}</Button>
          }
        />
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header */}
      <Card size="small">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={handleBackToList}
            >
              {t('common.back')}
            </Button>
            <div>
              <Title level={4} style={{ margin: 0 }}>
                {displayDevice.name}
              </Title>
              <Text type="secondary" style={{ fontFamily: 'monospace', fontSize: 13 }}>
                {displayDevice.sn}
              </Text>
            </div>
            <StatusIndicator
              status={displayDevice.connStatus === 'online' ? 'online' : 'offline'}
              variant="tag"
            />
            {displayDevice.alarmLevel !== 'none' && (
              <Tag color={SEVERITY_COLOR[displayDevice.alarmLevel]}>
                {SEVERITY_LABEL[displayDevice.alarmLevel]}
              </Tag>
            )}
          </div>
          <Space>
            {/* license tab 自带"刷新"按钮，此处头部刷新隐藏，避免同页两个刷新按钮 */}
            {activeTab !== 'license' && (
              <Button
                icon={<ReloadOutlined />}
                onClick={handleHeaderRefresh}
                loading={activeTab === 'quickSettings' && (quickSettingsSyncPending || syncMutation.isPending)}
              >
                {t('common.refresh')}
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* Tabs */}
      <Card
        styles={{ body: { padding: 0 } }}
        style={{ flex: 1 }}
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          style={{ padding: '0 16px' }}
          items={[
            {
              key: 'basic',
              label: t('common.detail'),
              children: (
                <div style={{ padding: '16px 0' }}>
                  {detailGroups[0] && renderFieldGroup(detailGroups[0], displayDevice)}
                  {cellGroup && (
                    <Card
                      size="small"
                      title={cellGroup.title}
                      extra={bmCellTechOptions.length > 1 ? (
                        <Space size={8}>
                          <Text type="secondary">{t('device.cellViewTech')}</Text>
                          <Radio.Group
                            size="small"
                            optionType="button"
                            buttonStyle="solid"
                            value={activeBmTech}
                            onChange={(event) => setActiveBmTech(event.target.value as BmCellTech)}
                          >
                            {bmCellTechOptions.map((tech) => (
                              <Radio.Button key={tech} value={tech}>
                                {tech === 'LTE' ? t('device.cellViewTech.lte') : t('device.cellViewTech.gsm')}
                              </Radio.Button>
                            ))}
                          </Radio.Group>
                        </Space>
                      ) : undefined}
                    >
                      <Table<CellRecord>
                        rowKey="key"
                        columns={cellColumns}
                        dataSource={displayCellRecords}
                        pagination={false}
                        size="small"
                        scroll={{ x: 'max-content' }}
                      />
                    </Card>
                  )}
                  {detailGroups.slice(1).map((group) => renderFieldGroup(group, displayDevice))}
                </div>
              ),
            },
            {
              key: 'parameters',
              label: t('device.parameterTree'),
              children: <ParameterTreeTab deviceId={device.id} />,
            },
            ...(showQuickSettingsTab
              ? [{
                  key: 'quickSettings',
                  label: t('device.quickSettings.tabTitle'),
                  // 保活：用户在此 tab 改参数后看到"入队成功 / 基站应答"Tag,
                  // 切到其他内部 tab 再切回时必须保留反馈状态(state 在子组件 useState 中)。
                  forceRender: true,
                  children: <QuickSettingsTab deviceId={device.id} networkType={device.networkType} />,
                }]
              : []),
            {
              key: 'alarms',
              label: (
                <span>
                  {t('nav.alarm.current')}
                  {alarms.length > 0 && (
                    <Badge count={alarms.length} size="small" style={{ marginLeft: 6 }} />
                  )}
                </span>
              ),
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <DataTable<Alarm>
                    tableId="device-detail-alarms"
                    columns={alarmColumns}
                    dataSource={alarms}
                    loading={alarmsLoading}
                    rowKey="id"
                    total={alarmData?.total ?? 0}
                    currentPage={alarmPage}
                    pageSize={alarmPageSize}
                    onPageChange={handleAlarmPageChange}
                    showPagination
                    defaultDensity="default"
                    alarmRowStyle={(record) => record.severity as 'critical' | 'major' | 'minor' | 'warning'}
                  />
                </div>
              ),
            },
            {
              key: 'performance',
              label: 'KPI',
              children: <KPITabContent device={displayDevice} t={t} />,
            },
            {
              key: 'license',
              label: t('device.licenseParam.title'),
              forceRender: true,
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <LicenseParamsTab deviceId={device.id} />
                </div>
              ),
            },
          ]}
        />
      </Card>

      <AlarmDetail alarm={detailAlarm} open={detailOpen} onClose={handleCloseAlarmDetail} />
      <ConfirmWithNoteModal
        open={ackModalOpen}
        title={t('alarm.acknowledge')}
        message={t('common.ackConfirmMsg', { count: ackTargetIds.length })}
        confirmText={t('alarm.acknowledge')}
        loading={ackLoading}
        onConfirm={handleAcknowledgeConfirm}
        onCancel={() => setAckModalOpen(false)}
      />
      <ConfirmWithNoteModal
        open={clearModalOpen}
        title={t('alarm.clear')}
        message={t('common.clearConfirmMsg', { count: clearTargetIds.length })}
        confirmText={t('alarm.clear')}
        confirmType="danger"
        loading={clearLoading}
        onConfirm={handleClearConfirm}
        onCancel={() => setClearModalOpen(false)}
      />
    </div>
  );
}
