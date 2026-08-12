import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
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
  Input,
  Modal,
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
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  EditOutlined,
  MoreOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import ErrorBoundary from '@/components/common/ErrorBoundary';
import { useParameterSchema, useSyncStatus } from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { deviceTaskApi, isAbortError } from '@core/services/api/deviceTaskApi';
import { useDeviceBySn, useDeviceGroups, useRenameDevice, useSyncDeviceParams } from '@core/hooks/api/useDevices';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import { deviceApi } from '@core/services/api/deviceApi';
import { isDeviceTaskTerminal, type DeviceTaskStatus } from '@core/types/deviceTask';
import { useDictionary } from '@core/hooks/api/useSystem';
import {
  activationStatusLabelOf,
  activationStatusOf,
  displayActivationStatusLabelOf,
  displayActivationStatusOf,
} from '@core/utils/activationStatus';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useResolvedCellInstances } from '@core/hooks/api/useResolvedCellInstances';
import { useAcknowledgeAlarms, useClearAlarms, useCurrentAlarms, useTriggerAlarmSync, useUnacknowledgeAlarms } from '@core/hooks/api/useAlarms';
import { useAggregatedMetricsByDevices, useMetricObjects } from '@core/hooks/api/usePmQuery';
import { formatObjectLdn } from '@core/types/pmObject';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@core/types/alarm';
import type { Device } from '@core/types/device';
import { connectionStatusMessageId } from '@core/utils/connectionStatus';
import { buildKpiCharts, buildKpiCompareData } from './kpiSeries';
import {
  buildKpiQueryWindow,
  buildKpiTooltipRangeLabels,
  formatKpiAxisLabel,
  formatKpiBucketRangeLabel,
  previousKpiQueryWindow,
} from './kpiTime';
import { runDeviceAlarmRefresh } from './alarmRefresh';
import ParameterTreeTab from './ParameterTreeTab';
import QuickSettingsTab from './QuickSettingsTab';
import LicenseParamsTab from './LicenseParamsTab';
import PasswordManagementTab from './PasswordManagementTab';
import { formatLteBandwidthDisplay } from './QuickSettingsTab/validators';
import AlarmDetail from '@/pages/alarm/AlarmDetail';
import AutoRefreshDropdown from '@/pages/alarm/components/AutoRefreshDropdown';
import ConfirmWithNoteModal from '@/pages/alarm/components/ConfirmWithNoteModal';
import { formatSystemTime } from '@core/utils/systemTime';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import { useAppStore } from '@core/store/appStore';
import { formatDeviceSyncStatus, getDeviceSyncStatusKind, normalizeDeviceSyncStatus } from '@core/utils/deviceSyncStatus';
import { computeCumulativeOnlineDurationSeconds, computeCurrentOnlineDurationSeconds } from '@core/utils/onlineDuration';
import { displayRFStatusLabelOf, displayRFStatusOf } from '@core/utils/rfStatus';
import { buildDeviceGroupDisplayName } from '@core/utils/deviceGroupDisplay';
import { resolveNetworkTypeLabel } from '@core/utils/networkType';
import { localizeDeviceProductName } from '@core/utils/deviceDisplay';
import type { Locale } from '@core/utils/i18nText';

const { Title, Text } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

const passwordTaskStorageKey = (deviceSn: string) => `xomc:device-password-task:${deviceSn}`;

function passwordTaskStatusTagSpec(status: DeviceTaskStatus | undefined): {
  color: string;
  icon: ReactNode;
} {
  switch (status) {
    case 'completed':
      return { color: 'success', icon: <CheckCircleOutlined /> };
    case 'failed':
      return { color: 'error', icon: <CloseCircleOutlined /> };
    case 'expired':
      return { color: 'warning', icon: <ClockCircleOutlined /> };
    case 'cancelled':
      return { color: 'default', icon: <CloseCircleOutlined /> };
    case 'sent':
    case 'pending':
    default:
      return { color: 'processing', icon: <SyncOutlined spin /> };
  }
}

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

const resolveConnectedBscIp = (bscSelect?: string, omlRemoteIp?: string, omlRemoteIpBak?: string): string => {
  const selected = bscSelect?.trim();
  if (selected === '0') return omlRemoteIp?.trim() || '';
  if (selected === '1') return omlRemoteIpBak?.trim() || '';
  return '';
};

const isBlankDetailValue = (value: string | number | null | undefined): boolean => {
  if (value == null) return true;
  const text = String(value).trim();
  return !text || text === '-' || text === '--';
};

const firstDetailValue = (...values: Array<string | number | null | undefined>): string => {
  for (const value of values) {
    if (!isBlankDetailValue(value)) return String(value).trim();
  }
  return '';
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
  gpsHeight?: number | null;
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
  bscSelect?: string;
  omlRemoteIp?: string;
  omlRemoteIpBak?: string;
  connectedBscIp?: string;
}

function isParameterSyncAlreadyRunningError(err: unknown) {
  const e = err as { bizCode?: number; response?: { data?: { biz_code?: number; code?: number } }; message?: string } | null;
  const code = e?.bizCode ?? e?.response?.data?.biz_code ?? e?.response?.data?.code;
  const msg = e?.message ?? '';
  return code === 1305 || code === 1205 || msg.includes('parameter sync already running');
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
  gps_height?: number | null;
  lock_status?: string;
  enb_id?: string;
  network_model?: string;
  mac?: string;
  multi_plmn_enable?: string;
  gps_version?: string;
  pps_time_mode?: string;
  rollback_version?: string;
  wan_status?: string;
  bsc_select?: string;
  oml_remote_ip?: string;
  oml_remote_ip_bak?: string;
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
    syncStatus: normalizeDeviceSyncStatus(info.sync_status),
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
    bscSelect: info.bsc_select,
    omlRemoteIp: info.oml_remote_ip,
    omlRemoteIpBak: info.oml_remote_ip_bak,
    connectedBscIp: resolveConnectedBscIp(info.bsc_select, info.oml_remote_ip, info.oml_remote_ip_bak),
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
    deviceName: firstDetailValue(info.deviceName, device.deviceName),
    remark: firstDetailValue(info.remark, device.remark),
    macAddress: firstDetailValue(info.mac, device.macAddress),
    gpsVersion: firstDetailValue(info.gpsVersion, device.gpsVersion),
    eci: firstDetailValue(info.eci, device.eci),
    pci: firstDetailValue(info.pci, device.pci),
    cellId: firstDetailValue(info.cellId, device.cellId),
    plmnId: firstDetailValue(info.plmn, device.plmnId),
    tac: firstDetailValue(info.tac, device.tac),
    subframeAssignment: firstDetailValue(info.subframeAssignment, device.subframeAssignment),
    specialSubframe: firstDetailValue(info.specialSubframe, device.specialSubframe),
    rootIndex: firstDetailValue(info.rootIndex, device.rootIndex),
    bandwidth: firstDetailValue(info.bandwidth, device.bandwidth),
    dlEarfcn: firstDetailValue(info.freqPoint, device.dlEarfcn),
    ulEarfcn: firstDetailValue(info.ulEarfcn, device.ulEarfcn),
    networkModel: firstDetailValue(info.networkModel, device.networkModel),
    txPower: firstDetailValue(info.transmitPower, device.txPower),
    band: firstDetailValue(info.band, device.band),
    mmeStatus: firstDetailValue(info.mmeStatus, device.mmeStatus),
    amfStatus: firstDetailValue(info.amfStatus, info.mmeStatus, device.amfStatus),
    rfStatus: firstDetailValue(info.rfStatus, device.rfStatus),
    syncStatus: firstDetailValue(info.syncStatus, device.syncStatus),
    lockStatus: firstDetailValue(info.lockStatus, device.lockStatus),
    multiPlmnEnable: firstDetailValue(info.multiPlmnEnable, device.multiPlmnEnable),
    firstOnlineTime: firstDetailValue(info.firstOnlineTime, device.firstOnlineTime),
    onlineTime: firstDetailValue(info.lastOnlineTime, device.onlineTime),
    offlineTime: firstDetailValue(info.lastOfflineTime, device.offlineTime),
    upTime: info.runTime ?? device.upTime,
    cumulativeOnlineDuration: info.cumulativeOnlineDuration ?? device.cumulativeOnlineDuration,
    gpsHeight: info.gpsHeight ?? device.gpsHeight,
    gpsSatelliteCount: info.gpsSatellites ?? device.gpsSatelliteCount,
    ppsTimeMode: firstDetailValue(info.ppsTimeMode),
    enbId: firstDetailValue(info.enbId, device.enbId),
    rollbackVersion: firstDetailValue(info.rollbackVersion, device.rollbackVersion),
    wanSpeed: firstDetailValue(info.wanSpeed, device.wanSpeed),
    bscSelect: firstDetailValue(info.bscSelect, device.bscSelect),
    omlRemoteIp: firstDetailValue(info.omlRemoteIp, device.omlRemoteIp),
    omlRemoteIpBak: firstDetailValue(info.omlRemoteIpBak, device.omlRemoteIpBak),
    bscSerialNumber: firstDetailValue(info.connectedBscIp, resolveConnectedBscIp(device.bscSelect, device.omlRemoteIp, device.omlRemoteIpBak), device.bscSerialNumber),
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
const fmtTime = (v: string | undefined | null) => (v ? formatSystemTime(v) : '-');

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

// 状态渲染
const renderStatusTag = (value: string | undefined, map: Record<string, { label: string; color: string }>) => {
  if (!value) return '-';
  const raw = value.trim();
  const entry = map[raw] ?? map[raw.toLowerCase()];
  if (!entry) return value;
  return <Tag color={entry.color}>{entry.label}</Tag>;
};

const renderDeviceSyncStatus = (value: string | undefined, t: ReturnType<typeof useT>) => {
  if (!value) return '-';
  const kind = getDeviceSyncStatusKind(value);
  if (!kind) return value;
  return (
    <Tag color={kind === 'error' ? 'error' : 'success'}>
      {formatDeviceSyncStatus(value, {
        synchronized: t('status.synchronized'),
        gps: `GPS ${t('status.synchronized')}`,
        beidou: `北斗${t('status.synchronized')}`,
        ntp: `NTP/1588 ${t('status.synchronized')}`,
        rem: `REM ${t('status.synchronized')}`,
        error: t('status.notSynchronized'),
      })}
    </Tag>
  );
};

// ─── 基站信息组 ────────────────────────────────────────────────────────

const getStationFields = (
  t: ReturnType<typeof useT>,
  networkType: string,
  onResolveNameSync?: (action: 'use_lmt' | 'use_omc' | 'ignore') => void,
  onEditOMCName?: () => void,
  renderNetworkType?: FieldItem['render'],
  locale: Locale = 'zh-CN',
): FieldGroup => {
  const fields: FieldItem[] = [
    // 公共字段
    { key: 'sn', label: t('device.sn'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.sn}</Text> },
    {
      key: 'name',
      label: t('device.hostName'),
      render: (d) => {
        // Issue #758: 名称同步待处理时显示对比 + 操作按钮
        if (d.nameSyncPending && d.lmtDeviceName) {
          return (
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
              <Space>
                <Badge status="processing" />
                <Text strong>{d.name || '-'}</Text>
                <Text type="secondary">({t('device.nameSyncPending.omcName')})</Text>
                {onEditOMCName && (
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    title={t('device.nameSync.editOmcName')}
                    aria-label={t('device.nameSync.editOmcName')}
                    onClick={onEditOMCName}
                  >
                    {t('common.edit')}
                  </Button>
                )}
              </Space>
              <Space>
                <Text>{d.lmtDeviceName}</Text>
                <Text type="secondary">({t('device.nameSyncPending.lmtName')})</Text>
              </Space>
              {onResolveNameSync && (
                <Space size={4}>
                  <Button size="small" type="primary" onClick={() => onResolveNameSync('use_lmt')}>
                    {t('device.nameSyncPending.useLmt')}
                  </Button>
                  <Button size="small" onClick={() => onResolveNameSync('use_omc')}>
                    {t('device.nameSyncPending.useOmc')}
                  </Button>
                  <Button size="small" onClick={() => onResolveNameSync('ignore')}>
                    {t('device.nameSyncPending.ignore')}
                  </Button>
                </Space>
              )}
            </Space>
          );
        }
        return (
          <Space size={4}>
            <Text>{d.name || '-'}</Text>
            {onEditOMCName && (
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                title={t('device.nameSync.editOmcName')}
                aria-label={t('device.nameSync.editOmcName')}
                onClick={onEditOMCName}
              >
                {t('common.edit')}
              </Button>
            )}
          </Space>
        );
      },
    },
    {
      key: 'networkType',
      label: t('device.radioMode'),
      render: renderNetworkType ?? ((d) => <Tag color={{ eNB: 'blue', gNB: 'green', GSM: 'orange' }[d.networkType ?? '']}>{d.networkType || '-'}</Tag>),
    },
    { key: 'productClass', label: t('device.productClass'), render: (d) => d.productClass || '-' },
    { key: 'deviceModel', label: t('device.model'), render: (d) => localizeDeviceProductName(d.deviceModel, locale) },
    { key: 'softwareVersion', label: t('device.softwareVersion'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.softwareVersion || '-'}</Text> },
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

  // GSM 独有字段（IPA Unit ID / OML Remote IP / OML Remote IP Bak / BSC Select 按需求隐藏）

  return { title: t('device.group.station'), fields };
};

// ─── 小区信息组 ────────────────────────────────────────────────────────

const getCellFields = (t: ReturnType<typeof useT>, networkType: string, isBtsProduct = false): FieldGroup => {
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
      { key: 'uplinkFrequency', label: t('device.uplinkFrequency'), render: (d) => d.uplinkFrequency ? `${d.uplinkFrequency} MHz` : '-' },
      { key: 'downlinkFrequency', label: t('device.downlinkFrequency'), render: (d) => d.downlinkFrequency ? `${d.downlinkFrequency} MHz` : '-' },
    );
    if (isBtsProduct) {
      fields.splice(1, 0, { key: 'arfcn', label: t('device.arfcn'), render: (d) => d.dlEarfcn || d.arfcn || '-' });
    } else {
      fields.splice(1, 0, { key: 'arfcn', label: t('device.arfcn'), render: (d) => d.arfcn ?? '-' });
      fields.push({ key: 'btsNum', label: t('device.btsNum'), render: (d) => d.btsNum ?? '-' });
    }
  }

  return { title: t('device.group.cell'), fields };
};

// ─── 状态信息组 ────────────────────────────────────────────────────────

const getStatusFields = (t: ReturnType<typeof useT>, networkType: string): FieldGroup => {
  const fields: FieldItem[] = [
    // 公共字段
    { key: 'ueCount', label: t('device.ueCount'), render: (d) => d.ueCount ?? '-' },
    { key: 'syncStatus', label: t('device.syncStatus'), render: (d) => renderDeviceSyncStatus(d.syncStatus, t) },
  ];

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      {
        key: 'mmeStatus',
        label: t('device.mmeStatus'),
        render: (d) => {
          const messageId = connectionStatusMessageId(d.mmeStatus, true);
          return messageId ? t(messageId) : '-';
        },
      },
      { key: 'lockStatus', label: t('device.lockStatus'), render: (d) => renderStatusTag(d.lockStatus, { locked: { label: t('status.locked'), color: 'warning' }, unlocked: { label: t('status.unlocked'), color: 'success' } }) },
      { key: 'wanSpeed', label: t('device.wanSpeed'), render: (d) => d.wanSpeed ?? '-' },
    );
  }

  // gNB 独有字段
  if (networkType === 'gNB') {
    fields.push(
      {
        key: 'mmeStatus',
        label: t('device.amfStatus'),
        render: (d) => {
          const messageId = connectionStatusMessageId(d.mmeStatus ?? d.amfStatus);
          return messageId ? t(messageId) : '-';
        },
      },
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

const getOtherFields = (
  t: ReturnType<typeof useT>,
  networkType: string,
  device: DetailDevice,
  onEditLocationSource?: () => void,
): FieldGroup => {
  const fields: FieldItem[] = [
    // 时间信息
    { key: 'onlineTime', label: t('device.onlineTime'), render: (d) => fmtTime(d.onlineTime) },
    { key: 'offlineTime', label: t('device.offlineTime'), render: (d) => fmtTime(d.offlineTime) },
    { key: 'onlineDuration', label: t('device.onlineDuration'), render: (d) => fmtDuration(computeCurrentOnlineDurationSeconds({
      isOnline: d.isOnline,
      onlineTime: d.onlineTime,
      offlineTime: d.offlineTime,
      fallbackOnlineDuration: d.onlineDuration,
    })) },
    { key: 'upTime', label: t('device.upTime'), render: (d) => fmtDuration(d.upTime) },
    // T-0173: 累计在线时长 = 后端 cumulative_online_duration + 本次在线区间。
    // 在线时由 DeviceStatusReconciler 在 online→offline 边沿才追加,所以前端补一个
    // 当前段 (NOW - onlineTime) 让展示精确到当前。
    { key: 'cumulativeOnlineDuration', label: t('device.cumulativeOnlineDuration'),
      render: (d) => fmtDuration(computeCumulativeOnlineDurationSeconds({
        isOnline: d.isOnline,
        onlineTime: d.onlineTime,
        offlineTime: d.offlineTime,
        fallbackOnlineDuration: d.onlineDuration,
        cumulativeOnlineDuration: d.cumulativeOnlineDuration,
      })) },
    // T-0173: 离线原因（仅离线时显示有意义,在线时也展示便于追溯上次掉线原因)。
    { key: 'lastOfflineReason', label: t('device.lastOfflineReason'),
      render: (d) => d.lastOfflineReason ? t(`device.lastOfflineReason.${d.lastOfflineReason}`) : '-' },
    { key: 'firstOnlineTime', label: t('device.firstOnlineTime'), render: (d) => fmtTime(d.firstOnlineTime) },
    { key: 'lastOnlineTime', label: t('device.lastOnline'), render: (d) => fmtTime(d.lastOnlineTime) },
    // 站址信息
    { key: 'siteName', label: t('device.siteName'), render: (d) => d.deviceName || '-' },
    { key: 'installAddress', label: t('device.installAddress'), render: (d) => d.installAddress || '-' },
    {
      key: 'locationSourceMode',
      label: t('device.locationSourceMode'),
      render: (d) => (
        <Space size={4}>
          <Text>
            {t(d.locationSourceMode === 'external' ? 'device.locationSourceExternal' : 'device.locationSourceTr069')}
          </Text>
          {onEditLocationSource && (
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              title={t('device.locationSourceMode')}
              aria-label={t('device.locationSourceMode')}
              onClick={onEditLocationSource}
            >
              {t('common.edit')}
            </Button>
          )}
        </Space>
      ),
    },
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

const BM_OPTICAL_MODULE_FIELDS = [
  ['identifier', 'device.optical.identifier'],
  ['connector', 'device.optical.connector'],
  ['transmissionMedia', 'device.optical.transmissionMedia'],
  ['encodeing', 'device.optical.encoding'],
  ['linkLength', 'device.optical.linkLength'],
  ['vendorName', 'device.optical.vendorName'],
  ['vendorPn', 'device.optical.vendorPn'],
  ['ethComplianceCodes', 'device.optical.complianceCodes'],
  ['wavelength', 'device.optical.wavelength'],
  ['options', 'device.optical.options'],
  ['bitRate', 'device.optical.bitRate'],
  ['transceiverTemperature', 'device.optical.temperature'],
  ['supplyVoltage', 'device.optical.supplyVoltage'],
  ['txBiasionCurrent', 'device.optical.txBiasCurrent'],
  ['txOpticalOutputPower', 'device.optical.txPower'],
  ['rxOpticalInputPower', 'device.optical.rxPower'],
  ['linkStatus', 'device.optical.linkStatus'],
] as const;

const firstText = (...values: Array<string | number | null | undefined>): string => {
  for (const value of values) {
    if (!isBlankDetailValue(value)) return String(value).trim();
  }
  return '';
};

const cellIndexedFallback = (
  value: string | number | null | undefined,
  index: number,
  total: number,
): string => {
  if (value == null) return '';
  const text = String(value).trim();
  if (isBlankDetailValue(text)) return '';
  if (!text.includes(',')) return index === 0 || total <= 1 ? text : '';
  return text.split(',').map((item) => item.trim()).filter((item) => !isBlankDetailValue(item))[index] ?? '';
};

const buildCellRecords = (device: Device, detailCells?: DeviceDetailCell[]): CellRecord[] => {
  if (Array.isArray(detailCells) && detailCells.length > 0) {
    const total = detailCells.length;
    return detailCells.map((cell, idx) => ({
      key: `${device.id}-cell-${cell.index || idx + 1}`,
      index: cell.index || idx + 1,
      values: {
        ...device,
        cellId: firstText(cell.cellId, cell.eci, cellIndexedFallback(device.cellId, idx, total), cellIndexedFallback(device.eci, idx, total)),
        nrCellId: firstText(cell.cellId, cell.eci, cellIndexedFallback(device.nrCellId, idx, total)),
        eci: firstText(cell.eci, cellIndexedFallback(device.eci, idx, total)),
        pci: firstText(cell.pci, cellIndexedFallback(device.pci, idx, total)),
        freqPoint: firstText(cell.freqPoint, cellIndexedFallback(device.dlEarfcn, idx, total)),
        bandwidth: firstText(cell.bandwidth, cellIndexedFallback(device.bandwidth, idx, total)),
        band: firstText(cell.band, cellIndexedFallback(device.band, idx, total)),
        opState: firstText(cell.opState, cellIndexedFallback(device.opState, idx, total)),
        rfStatus: firstText(cell.rfTxStatus, cellIndexedFallback(device.rfStatus, idx, total)),
        adminState: firstText(cell.adminState, cellIndexedFallback(device.adminState, idx, total)),
        lac: firstText(cell.lac, cellIndexedFallback(device.lac, idx, total)),
        arfcn: firstText(cell.arfcn, cellIndexedFallback(device.arfcn, idx, total)),
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

const renderCellOpState = (
  value: string | undefined,
  _isOnline: boolean | undefined | null,
  t: ReturnType<typeof useT>,
  details?: { label?: string; value?: string; status?: boolean }[],
  locale: 'zh-CN' | 'en-US' = 'zh-CN',
) => {
  const status = activationStatusOf(value);
  if (status == null) return '-';
  const label = activationStatusLabelOf(value, details, {
    active: t('status.active'),
    inactive: t('status.inactive'),
  }, locale);
  return <Tag color={status === 'active' ? 'success' : 'error'}>{label}</Tag>;
};

const renderCellRfStatus = (value: string | undefined, isOnline: boolean | undefined | null, t: ReturnType<typeof useT>) => {
  const kind = displayRFStatusOf(value, isOnline);
  if (!kind) return '-';
  const labels = {
    on: t('status.rfOn'),
    off: t('status.rfOff'),
    error: t('status.failed'),
  };
  const color = kind === 'on' ? 'success' : 'error';
  return <Tag color={color}>{displayRFStatusLabelOf(value, isOnline, labels)}</Tag>;
};

const renderCellAdminState = (
  value: string | undefined,
  _networkType: string,
  t: ReturnType<typeof useT>,
) => {
  return renderStatusTag(value, {
    // CellEnable.AdminState: 1 = 未锁定，0 = 锁定。
    '1': { label: t('status.unlocked'), color: 'success' },
    '0': { label: t('status.locked'), color: 'warning' },
    '2': { label: t('status.unlocked'), color: 'success' },
    '3': { label: t('status.shuttingDown'), color: 'error' },
    true: { label: t('status.locked'), color: 'warning' },
    false: { label: t('status.unlocked'), color: 'success' },
    enabled: { label: t('status.locked'), color: 'warning' },
    disabled: { label: t('status.unlocked'), color: 'success' },
    locked: { label: t('status.locked'), color: 'warning' },
    unlocked: { label: t('status.unlocked'), color: 'success' },
    shuttingdown: { label: t('status.shuttingDown'), color: 'error' },
    'shutting down': { label: t('status.shuttingDown'), color: 'error' },
  });
};

const getCellSummaryColumns = (networkType: string, t: ReturnType<typeof useT>, isBtsProduct = false): CellSummaryColumn[] => {
  // 表里每行是一个 cell，这里的「激活状态」是 cell.op_state（小区维度），
  // 与设备列表/详情头的 device.op_state（设备维度）是底层同名但完全不同的
  // 字段。为避免同名给用户造成「列表激活/详情未激活」的误解，列标题专用
  // device.cellOpState（'小区激活态'）。渲染仄 renderCellOpState 仍复用 cell op_state 语义。
  const base: CellSummaryColumn[] = [
    { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
    { title: t('device.cellOpState'), key: 'opState', width: 120 },
  ];

  switch (networkType) {
    case 'eNB':
      return [
        { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
        { title: t('device.cellId'), dataIndex: ['values', 'cellId'], key: 'cellId', width: 120 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.cellOpState'), key: 'opState', width: 120 },
        { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
        { title: 'PCI', dataIndex: ['values', 'pci'], key: 'pci', width: 100 },
        { title: 'Freq Point', dataIndex: ['values', 'freqPoint'], key: 'freqPoint', width: 140 },
        { title: t('device.bandwidth'), dataIndex: ['values', 'bandwidth'], key: 'bandwidth', width: 120 },
        { title: 'band', dataIndex: ['values', 'band'], key: 'band', width: 100 },
      ];
    case 'gNB':
      return [
        { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
        { title: t('device.cellOpState'), key: 'opState', width: 120 },
        { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
        { title: 'PCI', dataIndex: ['values', 'pci'], key: 'pci', width: 100 },
        { title: 'NRARFCN', dataIndex: ['values', 'freqPoint'], key: 'freqPoint', width: 140 },
        { title: 'band', dataIndex: ['values', 'band'], key: 'band', width: 100 },
        { title: 'NR Cell ID', dataIndex: ['values', 'nrCellId'], key: 'nrCellId', width: 160 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.bandwidth'), dataIndex: ['values', 'bandwidth'], key: 'bandwidth', width: 120 },
      ];
    case 'GSM':
      if (isBtsProduct) {
        return [
          { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
          { title: t('device.cellOpState'), key: 'opState', width: 120 },
          { title: t('device.rfStatus'), dataIndex: ['values', 'rfStatus'], key: 'rfStatus', width: 140 },
          { title: 'LAC', dataIndex: ['values', 'lac'], key: 'lac', width: 120 },
          { title: t('device.arfcn'), dataIndex: ['values', 'dlEarfcn'], key: 'arfcn', width: 120 },
        ];
      }
      return [
        { title: 'index', dataIndex: ['index'], key: 'index', width: 80 },
        { title: t('device.cellId'), dataIndex: ['values', 'cellId'], key: 'cellId', width: 120 },
        { title: 'Admin State', dataIndex: ['values', 'adminState'], key: 'adminState', width: 140 },
        { title: t('device.cellOpState'), key: 'opState', width: 120 },
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
  const appLocale = useAppStore((s) => s.locale);
  const systemTimezone = useSystemTimezoneValue();
  // 下钻对象集（多选，默认全选）：'' = 设备级伪项 + metricObjects 返回的实实在在 ldn。
  // 设备级行 (object_ldn='') 不在后端 metricObjects 返回集里（SQL 有 `object_ldn <> ''`），
  // 但 5G KGNB05xx 这类 KPI 原生在设备级行，不带上会全选后什么都不出，所以手动在集首加个 '' 伪项。
  const [objectLdns, setObjectLdns] = useState<string[]>([]);
  // 初始化一次全选；后续用户主动清空后不被 effect 覆盖。
  const objectLdnsInitializedRef = useRef(false);

  const networkType = device.networkType ?? '';
  const kpiConfig = useMemo(() => getKPIConfig(networkType, t), [networkType, t]);
  const sn = device.sn;
  const technology = normalizeQuickSettingsNetworkType(networkType); // eNB→lte / gNB→nr / GSM→gsm

  const queryWindow = useMemo(
    () => buildKpiQueryWindow(timeMode, systemTimezone),
    [timeMode, systemTimezone],
  );
  const metricPaths = useMemo(() => kpiConfig.map((c) => c.key), [kpiConfig]);

  // 下钻对象清单（该设备 PM 数据里实际出现过的小区/PLMN）。
  const { data: metricObjects = [] } = useMetricObjects(
    sn ? [sn] : [],
    technology || undefined,
  );

  // 全选集 = 设备级伪项 '' + metricObjects 全部 ldn。
  const allObjectLdns = useMemo(
    () => ['', ...metricObjects.map((o) => o.objectLdn)],
    [metricObjects],
  );

  // 首次成功拿到下拉项后默认全选（仅执行一次，不覆盖用户后续手动取消）。
  useEffect(() => {
    if (!objectLdnsInitializedRef.current && metricObjects.length > 0) {
      setObjectLdns(allObjectLdns);
      objectLdnsInitializedRef.current = true;
    }
  }, [metricObjects.length, allObjectLdns]);

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
    () => previousKpiQueryWindow(
      queryWindow.startTime,
      queryWindow.endTime,
      systemTimezone,
      queryWindow.granularity,
    ),
    [queryWindow.startTime, queryWindow.endTime, queryWindow.granularity, systemTimezone],
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

  // 聚合行 → 每 K 编号一张图（纯函数，按 objectLdns 集合过滤 + 多对象 series，null 占位不画点）。
  const charts = useMemo(
    () => buildKpiCharts(rows, kpiConfig, objectLdns),
    [rows, kpiConfig, objectLdns],
  );
  // 上一周期图（同口径、同 objectLdns 过滤），再吸附对齐到当前周期 X 轴。
  const prevCharts = useMemo(
    () => buildKpiCharts(prevRows, kpiConfig, objectLdns),
    [prevRows, kpiConfig, objectLdns],
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
      // 下拉标签走 formatObjectLdn 友好名；legend / series.name 却按拍板决定显示原始 LDN。
      label: formatObjectLdn(o.objectLdn, appLocale),
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
            mode="multiple"
            size="small"
            value={objectLdns}
            options={objectOptions}
            onChange={setObjectLdns}
            allowClear
            maxTagCount="responsive"
            placeholder={t('device.kpi.objectPlaceholder')}
            // 工具栏 flex-end 不让 Select 自然撑开 → 用固定 width 给足空间显示 LDN tag；
            // 窄屏靠 flexWrap 触发整行换行，不破布局。
            style={{ width: 720 }}
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
          const xLabels = chart.xData.map((iso) =>
            formatKpiAxisLabel(iso, queryWindow.granularity, systemTimezone),
          );
          const xDataFull = buildKpiTooltipRangeLabels(
            chart.xData,
            chart.xEnds,
            queryWindow.granularity,
            systemTimezone,
          );
          // 多对象：过滤掉「该对象本图全 null」的索引，无数据对象不出线。
          const visibleIdx = chart.series
            .map((s, i) => (s.values.every((v) => v == null) ? -1 : i))
            .filter((i) => i >= 0);
          // 上周期 tooltip 文案（首条可见对象的桶，多对象时不并出多行以保清爽）：
          // snapped 后多对象 buckets 一致，拿首条不失真。
          const hasCompareGlobal = !compare.isEmpty;
          const firstCompareIdx = hasCompareGlobal
            ? visibleIdx.find(
                (i) => !(compare.series[i]?.values ?? []).every((v) => v == null),
              )
            : undefined;
          const compareLabels =
            firstCompareIdx != null
              ? compare.series[firstCompareIdx].compareBuckets.map((s, i) =>
                  formatKpiBucketRangeLabel(
                    s,
                    compare.series[firstCompareIdx].compareBucketEnds[i] ?? '',
                    queryWindow.granularity,
                    systemTimezone,
                  ),
                )
              : undefined;
          // 实线 × 多对象；虚线 × 多对象（仅本对象上周期非全空才出）。
          const realSeries = visibleIdx.map((i) => ({
            name: chart.series[i].name,
            data: chart.series[i].values,
          }));
          const compareSeriesArr = hasCompareGlobal
            ? visibleIdx
                .filter(
                  (i) => !(compare.series[i]?.values ?? []).every((v) => v == null),
                )
                .map((i) => ({
                  name: t('device.detail.kpiPrevPeriod', { name: chart.series[i].name }),
                  data: compare.series[i].values,
                  dashed: true as const,
                  // 多对象时 legend 已经被实线占满；虚线只通过线型表达「上一周期」，不再
                  // 单独占 legend 项（避免与实线名重复 + 项过多触发 echart 翻页）。
                  showInLegend: false,
                }))
            : [];
          const lineSeries = [...realSeries, ...compareSeriesArr];

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
                      // gNB(5G/NR) 查空时给更明确文案（#187/#201）：5G 真机样本厂商错配会让
                      // KPI 算不出、该设备无 KPI 行，泛化「暂无数据」无法区分「指标库未注册」
                      // 与「时段无采样」。其它制式保持通用文案。
                      description={
                        chart.hasSamples
                          ? t('device.detail.kpiAllMissing')
                          : technology === 'nr'
                          ? t('device.detail.kpiNoDataNr')
                          : t('common.noData')
                      }
                    />
                  </div>
                ) : (
                  <LineChart
                    title=""
                    xData={xLabels}
                    xDataFull={xDataFull}
                    series={lineSeries}
                    compareLabels={compareLabels}
                    formatCompareLabel={(label) => `${t('pm.chart.tooltipPrevPeriod')} ${label}`}
                    unit={kpi.unit || undefined}
                    pmMetricValueFormat
                    height={220}
                    areaFill
                    connectNulls
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
  const { modal, message, notification } = App.useApp();
  const { sn = '' } = useParams<{ sn: string }>();
  const detailTabKey = sn ? `device-detail:${sn}` : 'device-detail';
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const appLocale = useAppStore((s) => s.locale);
  const openTab = useTabStore((s) => s.openTab);
  const closeTab = useTabStore((s) => s.closeTab);
  const [searchParams, setSearchParams] = useSearchParams();

  const { data: device, isLoading, refetch } = useDeviceBySn(sn);
  const { data: deviceGroupsData } = useDeviceGroups();
  const { data: opStateDict } = useDictionary('op_state');
  const { data: networkTypeDict } = useDictionary('network_type');
  const syncMutation = useSyncDeviceParams();
  const renameMutation = useRenameDevice(device?.id ?? '');
  const [omcNameEditorOpen, setOmcNameEditorOpen] = useState(false);
  const [omcNameDraft, setOmcNameDraft] = useState('');
  const [locationSourceEditorOpen, setLocationSourceEditorOpen] = useState(false);
  const [locationSourceDraft, setLocationSourceDraft] = useState<'tr069' | 'external'>('tr069');
  const [locationSourceSaving, setLocationSourceSaving] = useState(false);
  const { data: paramSyncStatus, refetch: refetchParamSyncStatus } = useSyncStatus(device?.id ?? '');
  const [quickSettingsSyncTargetPaths, setQuickSettingsSyncTargetPaths] = useState<string[]>([]);
  const [licenseSyncTargetPaths, setLicenseSyncTargetPaths] = useState<string[]>([]);
  const quickSettingsSync = useQuickSettingsFeedbackStore((s) => (device?.id ? s.quickSettingsSyncs[device.id] : undefined));
  const quickSettingsSyncPending = Boolean(quickSettingsSync);
  const lastQuickSettingsParamSync = useQuickSettingsFeedbackStore((s) => (device?.id ? s.lastScopedSyncs[device.id] : undefined)) ?? null;
  const observedParamSyncAtRef = useRef<Record<string, string>>({});
  const notifiedPasswordTaskRef = useRef<Record<string, true>>({});
  const [passwordTaskId, setPasswordTaskId] = useState<string | undefined>();
  const { data: passwordTask } = useDeviceTaskStatus(passwordTaskId);
  const triggerAlarmSync = useTriggerAlarmSync();
  const alarmRefreshAbortRef = useRef<AbortController | null>(null);
  const [alarmRefreshState, setAlarmRefreshState] = useState<{
    deviceSn: string;
    controller: AbortController;
  } | null>(null);
  const isDeviceParamSyncBusy = paramSyncStatus?.status === 'syncing' || quickSettingsSyncPending || syncMutation.isPending;
  const isQuickSettingsRefreshSubmitting = syncMutation.isPending;
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
    const merged = mergeDeviceDetailInfo(device, detailComposite?.info);
    const groups = deviceGroupsData?.groups ?? [];
    if (groups.length === 0) return merged;
    return {
      ...merged,
      groupName: buildDeviceGroupDisplayName(merged, groups, appLocale),
    };
  }, [appLocale, detailComposite?.info, device, deviceGroupsData?.groups]);
  const alarmRefreshPending = alarmRefreshState?.deviceSn === displayDevice?.sn;

  useEffect(() => {
    alarmRefreshAbortRef.current?.abort();
    alarmRefreshAbortRef.current = null;
    return () => {
      alarmRefreshAbortRef.current?.abort();
    };
  }, [sn]);

  useEffect(() => {
    const deviceSn = displayDevice?.sn;
    if (!deviceSn || typeof window === 'undefined') {
      setPasswordTaskId(undefined);
      return;
    }
    setPasswordTaskId(window.localStorage.getItem(passwordTaskStorageKey(deviceSn)) || undefined);
  }, [displayDevice?.sn]);

  const handlePasswordTaskSubmitted = useCallback((taskId: string) => {
    setPasswordTaskId(taskId);
    const deviceSn = displayDevice?.sn;
    if (deviceSn && typeof window !== 'undefined') {
      window.localStorage.setItem(passwordTaskStorageKey(deviceSn), taskId);
    }
  }, [displayDevice?.sn]);

  const handlePasswordTaskCleared = useCallback(() => {
    setPasswordTaskId(undefined);
    const deviceSn = displayDevice?.sn;
    if (deviceSn && typeof window !== 'undefined') {
      window.localStorage.removeItem(passwordTaskStorageKey(deviceSn));
    }
  }, [displayDevice?.sn]);

  useEffect(() => {
    if (!device?.id || !passwordTask || !isDeviceTaskTerminal(passwordTask.status)) return;

    const deviceSn = displayDevice?.sn;
    if (deviceSn && typeof window !== 'undefined') {
      window.localStorage.removeItem(passwordTaskStorageKey(deviceSn));
    }

    deviceParameterApi.invalidateParameterSchemaCache(device.id);
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', device.id] });
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', 'search', device.id] });
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', device.id] });

    if (notifiedPasswordTaskRef.current[passwordTask.id]) return;
    notifiedPasswordTaskRef.current[passwordTask.id] = true;

    if (passwordTask.status === 'completed') return;

    notification.error({
      message: t('device.password.taskFailed'),
      description: passwordTask.errorMessage || t('device.multi.unknownErrorHint'),
      duration: 8,
    });
  }, [device?.id, displayDevice?.sn, message, notification, passwordTask, queryClient, t]);

  const passwordTaskTag = useMemo(() => {
    if (!passwordTaskId) return null;
    const status = passwordTask?.status ?? 'pending';
    const spec = passwordTaskStatusTagSpec(status);
    return (
      <Tag icon={spec.icon} color={spec.color} style={{ marginInlineEnd: 0 }}>
        {t('device.password.statusPrefix')}: {t(`device.taskStatus.${status}`)}
      </Tag>
    );
  }, [passwordTask?.status, passwordTaskId, t]);

  useEffect(() => {
    const deviceId = device?.id;
    const syncedAt = paramSyncStatus?.lastParamSyncAt;
    if (!deviceId || !syncedAt) return;

    const prev = observedParamSyncAtRef.current[deviceId];
    observedParamSyncAtRef.current[deviceId] = syncedAt;
    if (prev === syncedAt) return;

    deviceParameterApi.invalidateParameterSchemaCache(deviceId);
    void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
  }, [device?.id, paramSyncStatus?.lastParamSyncAt, queryClient]);

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
    if (key === urlTab) return;
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('tab', key);
      return next;
    }, { replace: true });
  }, [setSearchParams, urlTab]);

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
      labelPrefixI18nKey: 'common.detail',
      labelSuffix: displayName,
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
  useEffect(() => {
    if (!device?.id) return;
    if (quickSettingsLoading) return;
    if (urlTab !== 'quickSettings' || showQuickSettingsTab) return;
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('tab', 'basic');
      return next;
    }, { replace: true });
  }, [device?.id, quickSettingsLoading, setSearchParams, showQuickSettingsTab, urlTab]);
  const [activeBmTech, setActiveBmTech] = useState<BmCellTech>('LTE');

  const detailQuickSettingsNetworkType = normalizeQuickSettingsNetworkType(displayDevice?.networkType);
  const isBmProduct = ((quickSettingsData?.paramModel
    ?? displayDevice?.productClass
    ?? device?.productClass
    ?? '')
    .trim()
    .toUpperCase()
    .startsWith('BM'));
  const isBtsProduct = ((quickSettingsData?.paramModel
    ?? displayDevice?.productClass
    ?? device?.productClass
    ?? '')
    .trim()
    .toUpperCase()) === 'BTS';
  const { data: opticalModuleSchema, isLoading: opticalModuleLoading } = useParameterSchema(
    device?.id ?? '',
    'Device.DeviceInfo.OpticalModInfo.1.',
    activeTab === 'basic' && isBmProduct,
  );
  const opticalModuleValues = useMemo(() => new Map(
    (opticalModuleSchema?.parameters ?? []).map((parameter) => [parameter.path, parameter.currentValue]),
  ), [opticalModuleSchema?.parameters]);

  // 概览页小区列表的实例过滤规则与「快速设置」tab 完全一致，统一走 useResolvedCellInstances。
  const detailResolved = useResolvedCellInstances({
    deviceId: device?.id ?? '',
    networkType: detailQuickSettingsNetworkType,
    paramModel: quickSettingsData?.paramModel ?? '',
    bmTech: activeBmTech,
    enabled: activeTab === 'basic',
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

  const submitScopedParamRefresh = useCallback((
    deviceId: string,
    targetPaths: string[],
    scope: 'quickSettings' | 'license' = 'quickSettings',
  ) => {
    const parameterPaths = targetPaths.length > 0 ? targetPaths : undefined;
    const targetCountHint = targetPaths.length;

    useQuickSettingsFeedbackStore.getState().startQuickSettingsSync(deviceId, {
      scope,
      lastParamSyncAt: paramSyncStatus?.lastParamSyncAt,
      lastParamSyncFailedAt: paramSyncStatus?.lastParamSyncFailedAt,
      targetCount: targetCountHint,
      gpvTaskCount: 0,
      startedAt: Date.now(),
    });
    syncMutation.mutate(
      { deviceId, parameterPaths },
      {
        onSuccess: (data) => {
          const targetCount = data.parameterPathsCount ?? targetCountHint;
          const gpvTaskCount = data.gpvTaskCount ?? 0;
          useQuickSettingsFeedbackStore.getState().patchQuickSettingsSync(deviceId, {
            sourceId: data.sourceId,
            requestId: data.requestId,
            runId: data.runId,
            targetCount,
            gpvTaskCount,
          });
          message.success(targetCount > 0
            ? t('device.detail.deviceFetchQueuedScoped', { id: data.sourceId, count: targetCount, gpvCount: gpvTaskCount })
            : t('device.detail.deviceFetchQueued', { id: data.sourceId }));
          void refetchParamSyncStatus();
        },
        onError: (err) => {
          useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
          if (isParameterSyncAlreadyRunningError(err)) {
            void refetchParamSyncStatus();
            return;
          }
          const errMsg = err instanceof Error ? err.message : t('device.detail.deviceFetchTriggerFailed');
          message.error(errMsg);
        },
      },
    );
  }, [message, paramSyncStatus?.lastParamSyncAt, paramSyncStatus?.lastParamSyncFailedAt, refetchParamSyncStatus, syncMutation, t]);

  const startAlarmRefresh = useCallback(() => {
    if (alarmRefreshPending) return;

    const deviceSn = displayDevice?.sn?.trim();
    if (!deviceSn) {
      void message.error(t('common.operationFailed'));
      return;
    }

    alarmRefreshAbortRef.current?.abort();
    const abortController = new AbortController();
    alarmRefreshAbortRef.current = abortController;
    setAlarmRefreshState({ deviceSn, controller: abortController });

    void runDeviceAlarmRefresh({
      deviceSn,
      trigger: (targetSn) => triggerAlarmSync.mutateAsync(targetSn),
      waitForTerminal: (taskId, signal) => deviceTaskApi.waitForTerminal(taskId, { signal }),
      refreshCurrentAlarms: () => queryClient.invalidateQueries({ queryKey: ['alarms', 'current'] }),
      signal: abortController.signal,
    }).then(() => {
      void message.success(t('status.success'));
    }).catch((error) => {
      if (!isAbortError(error)) {
        void message.error(t('common.operationFailed'));
      }
    }).finally(() => {
      if (alarmRefreshAbortRef.current === abortController) {
        alarmRefreshAbortRef.current = null;
      }
      setAlarmRefreshState((current) => current?.controller === abortController ? null : current);
    });
  }, [alarmRefreshPending, displayDevice?.sn, message, queryClient, t, triggerAlarmSync]);

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
        startAlarmRefresh();
        break;
      case 'quickSettings':
        if (deviceId) {
          const hasQuickSettingsDrafts = Object.keys(useQuickSettingsFeedbackStore.getState().drafts)
            .some((key) => key.startsWith(`${deviceId}::`));
          if (hasQuickSettingsDrafts) {
            modal.confirm({
              title: t('device.detail.quickSettingsRefreshConfirmTitle'),
              content: t('device.detail.quickSettingsRefreshConfirmContent'),
              okText: t('common.confirm'),
              cancelText: t('common.cancel'),
              onOk: () => {
                submitScopedParamRefresh(deviceId, quickSettingsSyncTargetPaths);
              },
            });
            break;
          }
          submitScopedParamRefresh(deviceId, quickSettingsSyncTargetPaths);
        }
        break;
      case 'license':
        if (deviceId) {
          submitScopedParamRefresh(deviceId, licenseSyncTargetPaths, 'license');
        }
        break;
      default:
        break;
    }
  }, [activeTab, device?.id, licenseSyncTargetPaths, modal, queryClient, quickSettingsSyncTargetPaths, refetch, startAlarmRefresh, submitScopedParamRefresh, t]);

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
  const [alarmAutoRefresh, setAlarmAutoRefresh] = useState(false);
  const [alarmRefreshInterval, setAlarmRefreshInterval] = useState(30);

  const alarmParams = useMemo(
    () => ({ deviceSn: sn, page: alarmPage, pageSize: alarmPageSize } as Parameters<typeof useCurrentAlarms>[0]),
    [alarmPage, alarmPageSize, sn]
  );
  const { data: alarmData, isLoading: alarmsLoading, refetch: refetchAlarms } = useCurrentAlarms(
    alarmParams,
    {
      refetchIntervalMs: alarmAutoRefresh ? alarmRefreshInterval * 1000 : false,
      refetchIntervalInBackground: alarmAutoRefresh,
      refetchOnWindowFocus: true,
    }
  );
  const alarms: Alarm[] = alarmData?.items ?? [];

  useEffect(() => {
    setAlarmPage(1);
    setAlarmPageSize(20);
  }, [sn]);

  useEffect(() => {
    if (!alarmAutoRefresh) return;
    void refetchAlarms();
  }, [alarmAutoRefresh, alarmRefreshInterval, refetchAlarms]);

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
        width: 56,
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
        key: 'deviceSn',
        title: t('alarm.deviceSn'),
        dataIndex: 'deviceSn',
        width: 240,
        mono: true,
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
      { key: 'alarmIdentifier', title: t('alarm.alarmIdentifier'), dataIndex: 'alarmIdentifier', width: 100, mono: true },
      { key: 'specificProblem', title: t('alarm.specificProblem'), dataIndex: 'specificProblem', width: 220, ellipsis: true },
      { key: 'alarmName', title: t('alarm.possibleCause'), dataIndex: 'alarmName', width: 260, ellipsis: true, copyable: true },
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
      {
        key: 'eventTime',
        title: t('alarm.time'),
        dataIndex: 'eventTime',
        width: 160,
        render: (_val, record) => formatSystemTime(record.eventTime),
      },
    ],
    [SEVERITY_LABEL, handleAcknowledgeAlarm, handleClearAlarm, handleShowAlarmDetail, handleUnacknowledgeAlarm, t]
  );

  // Issue #758: 设备名称同步 - 解决名称差异
  const handleResolveNameSync = useCallback(
    async (action: 'use_lmt' | 'use_omc' | 'ignore') => {
      if (!displayDevice?.id) return;
      try {
        await deviceApi.resolveNameSync(displayDevice.id, action);
        void message.success(t('common.operationSuccess'));
        // 刷新设备列表、SN 详情和 composite 详情缓存。
        void queryClient.invalidateQueries({ queryKey: ['devices'] });
      } catch (_err) {
        void message.error(t('common.operationFailed'));
      }
    },
    [displayDevice?.id, message, queryClient, t]
  );

  const handleOpenOMCNameEditor = useCallback(() => {
    if (!displayDevice) return;
    setOmcNameDraft(displayDevice.name || '');
    setOmcNameEditorOpen(true);
  }, [displayDevice]);

  const handleRenameOMCName = useCallback(async () => {
    const nextName = omcNameDraft.trim();
    if (!nextName) {
      void message.error(t('device.nameSync.omcNameRequired'));
      return;
    }
    try {
      await renameMutation.mutateAsync(nextName);
      setOmcNameEditorOpen(false);
      void message.success(t('device.nameSync.editOmcNameSuccess'));
    } catch {
      void message.error(t('common.operationFailed'));
    }
  }, [message, omcNameDraft, renameMutation, t]);

  const handleOpenLocationSourceEditor = useCallback(() => {
    if (!displayDevice) return;
    setLocationSourceDraft(displayDevice.locationSourceMode === 'external' ? 'external' : 'tr069');
    setLocationSourceEditorOpen(true);
  }, [displayDevice]);

  const handleSaveLocationSource = useCallback(async () => {
    if (!displayDevice?.id) return;
    setLocationSourceSaving(true);
    try {
      await deviceApi.update(
        displayDevice.id,
        { locationSourceMode: locationSourceDraft },
        displayDevice,
      );
      setLocationSourceEditorOpen(false);
      await queryClient.invalidateQueries({ queryKey: ['devices'] });
      void message.success(t('common.operationSuccess'));
    } catch {
      void message.error(t('common.operationFailed'));
    } finally {
      setLocationSourceSaving(false);
    }
  }, [displayDevice, locationSourceDraft, message, queryClient, t]);

  const renderDeviceNetworkType = useCallback<FieldItem['render']>(
    (d) => {
      const label = resolveNetworkTypeLabel(d.networkType, networkTypeDict?.sysDictionaryDetails, appLocale);
      return <Tag color={{ eNB: 'blue', gNB: 'green', GSM: 'orange' }[d.networkType ?? '']}>{label}</Tag>;
    },
    [appLocale, networkTypeDict?.sysDictionaryDetails],
  );

  // 根据设备制式获取字段组
  const detailGroups = useMemo((): FieldGroup[] => {
    if (!displayDevice) return [];
    const networkType = normalizeNetworkType(displayDevice.networkType);

    // BSC（独立 GSM 设备，paramModel === 'BSC'）按需求隐藏「状态信息」组；BTS 保留显示。
    const groups: FieldGroup[] = [getStationFields(
      t,
      networkType,
      handleResolveNameSync,
      handleOpenOMCNameEditor,
      renderDeviceNetworkType,
      appLocale,
    )];
    if (!detailResolved.isBSC) {
      groups.push(getStatusFields(t, networkType));
    }
    groups.push(getOtherFields(t, networkType, displayDevice, handleOpenLocationSourceEditor));
    return groups;
  }, [appLocale, detailResolved.isBSC, displayDevice, handleOpenLocationSourceEditor, handleOpenOMCNameEditor, handleResolveNameSync, renderDeviceNetworkType, t]);

  const cellGroup = useMemo((): FieldGroup | null => {
    if (!displayDevice) return null;
    const networkType = isBmProduct && activeBmTech === 'GSM'
      ? 'GSM'
      : normalizeNetworkType(displayDevice.networkType);
    // BSC（独立 GSM 设备）按需求隐藏「小区信息」表；BTS 与 BM 产品里的 GSM 小区视图均保留。
    if (detailResolved.isBSC) return null;
    return getCellFields(t, networkType, isBtsProduct);
  }, [activeBmTech, detailResolved.isBSC, displayDevice, isBmProduct, isBtsProduct, t]);

  const displayCellNetworkType = useMemo(() => {
    if (isBmProduct) {
      return activeBmTech === 'GSM' ? 'GSM' : 'eNB';
    }
    return normalizeNetworkType(displayDevice?.networkType);
  }, [activeBmTech, displayDevice?.networkType, isBmProduct]);
  const displayDeviceIsOnline = displayDevice?.isOnline;

  const activeDetailCells = isBmProduct && activeBmTech === 'GSM'
    ? detailComposite?.gsmCells
    : detailComposite?.cells;
  const useCompositeCellRecords = !isBtsProduct && Array.isArray(activeDetailCells) && activeDetailCells.length > 0;

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
    () => getCellSummaryColumns(displayCellNetworkType, t, isBtsProduct).map((column) => {
      // LTE 小区列表里的带宽列与详情字段、快速设置共享同一个枚举映射（仅 eNB）。
      const isLteBandwidth = column.key === 'bandwidth' && displayCellNetworkType === 'eNB';
      return {
        ...column,
        render: column.key === 'opState'
          ? (_: unknown, row: CellRecord) => renderCellOpState(
            row.values.opState as string | undefined,
            displayDeviceIsOnline,
            t,
            opStateDict?.sysDictionaryDetails,
            appLocale,
          )
          : column.key === 'adminState'
            ? (_: unknown, row: CellRecord) => renderCellAdminState(
              row.values.adminState as string | undefined,
              displayCellNetworkType,
              t,
            )
          : column.key === 'rfStatus'
            ? (_: unknown, row: CellRecord) => renderCellRfStatus(
              row.values.rfStatus as string | undefined,
              displayDeviceIsOnline,
              t,
            )
          : isLteBandwidth
            ? (_: unknown, row: CellRecord) => formatLteBandwidthDisplay(row.values.bandwidth as string | undefined)
            : (value: string | number | undefined) => value ?? '-',
      };
    }),
    [appLocale, displayCellNetworkType, displayDeviceIsOnline, isBtsProduct, opStateDict?.sysDictionaryDetails, t],
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
            {/*
              「激活状态」设备级 Tag —— 判定走 frontend-core/utils/activationStatus.ts,
              列表/详情/KV 全调同一函数，修改口径请只改 utility。

              注: 「小区信息』表里也有列名「激活状态」但那是 cell.op_state（小区维度），
              与这里的 device.op_state（设备维度）是后端同名不同事实的两个字段——
              详情页小区表列标题已拆为 device.cellOpState「小区激活态」，避免同名误解。
            */}
            {(() => {
              const status = displayActivationStatusOf(displayDevice.opState, displayDevice.isOnline);
              if (status == null) return '-';
              const isActive = status === 'active';
              const label = displayActivationStatusLabelOf(displayDevice.opState, displayDevice.isOnline, opStateDict?.sysDictionaryDetails, {
                active: t('status.active'),
                inactive: t('status.inactive'),
              }, appLocale);
              return (
                <Tag color={isActive ? 'success' : 'error'}>
                  {label}
                </Tag>
              );
            })()}
            {displayDevice.alarmLevel !== 'none' && (
              <Tag color={SEVERITY_COLOR[displayDevice.alarmLevel]}>
                {SEVERITY_LABEL[displayDevice.alarmLevel]}
              </Tag>
            )}
          </div>
          <Space>
            {passwordTaskTag}
            {/* parameters tab 自带全量同步入口；quickSettings/license 使用页头刷新触发同一套参数同步。 */}
            {activeTab !== 'parameters' && activeTab !== 'password' && (
              <Button
                icon={<ReloadOutlined />}
                onClick={handleHeaderRefresh}
                loading={((activeTab === 'quickSettings' || activeTab === 'license') && isQuickSettingsRefreshSubmitting)
                  || (activeTab === 'alarms' && alarmRefreshPending)}
                disabled={isDeviceParamSyncBusy || (activeTab === 'alarms' && alarmRefreshPending)}
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
                  {isBmProduct && (
                    <Descriptions
                      title={t('device.group.opticalModule')}
                      bordered
                      column={{ xs: 1, sm: 2, md: 3, lg: 4 }}
                      size="small"
                      style={{ marginBottom: 16 }}
                    >
                      {BM_OPTICAL_MODULE_FIELDS.map(([leaf, labelKey]) => {
                        const rawValue = opticalModuleValues.get(`Device.DeviceInfo.OpticalModInfo.1.${leaf}`);
                        const value = rawValue == null || String(rawValue).trim() === '' ? '-' : String(rawValue);
                        const content = leaf === 'linkStatus' && value !== '-'
                          ? (
                            <Tag color={value.toLowerCase() === 'true' || value === '1' ? 'success' : 'default'}>
                              {t(value.toLowerCase() === 'true' || value === '1' ? 'status.online' : 'status.offline')}
                            </Tag>
                          )
                          : value;
                        return (
                          <Descriptions.Item key={leaf} label={t(labelKey)}>
                            {opticalModuleLoading ? <Skeleton.Input active size="small" /> : content}
                          </Descriptions.Item>
                        );
                      })}
                    </Descriptions>
                  )}
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
              // 细粒度 ErrorBoundary：参数树独立拉取大量数据，渲染异常时仅此页签降级，
              // 不连累设备头部与其他页签。
              children: (
                <ErrorBoundary>
                  <ParameterTreeTab
                    deviceId={device.id}
                    lastScopedSync={lastQuickSettingsParamSync}
                    syncBusy={isDeviceParamSyncBusy}
                    onFullSyncStarted={() => {
                      if (device?.id) useQuickSettingsFeedbackStore.getState().clearLastScopedSync(device.id);
                    }}
                  />
                </ErrorBoundary>
              ),
            },
            ...(showQuickSettingsTab
              ? [{
                  key: 'quickSettings',
                  label: t('device.quickSettings.tabTitle'),
                  // 保活：用户在此 tab 改参数后看到"入队成功 / 基站应答"Tag,
                  // 切到其他内部 tab 再切回时必须保留反馈状态(state 在子组件 useState 中)。
                  forceRender: true,
                  children: (
                    <ErrorBoundary>
                      <QuickSettingsTab
                        deviceId={device.id}
                        networkType={device.networkType}
                        active={activeTab === 'quickSettings'}
                        onSyncTargetPathsChange={setQuickSettingsSyncTargetPaths}
                      />
                    </ErrorBoundary>
                  ),
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
                    extraToolbarRight={(
                      <AutoRefreshDropdown
                        enabled={alarmAutoRefresh}
                        intervalSeconds={alarmRefreshInterval}
                        onEnabledChange={setAlarmAutoRefresh}
                        onIntervalChange={setAlarmRefreshInterval}
                        size="small"
                      />
                    )}
                    hideRealtime
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
              children: (
                <ErrorBoundary>
                  <KPITabContent device={displayDevice} t={t} />
                </ErrorBoundary>
              ),
            },
            {
              key: 'license',
              label: t('device.licenseParam.title'),
              forceRender: true,
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <ErrorBoundary>
                    <LicenseParamsTab
                      deviceId={device.id}
                      onSyncTargetPathsChange={setLicenseSyncTargetPaths}
                    />
                  </ErrorBoundary>
                </div>
              ),
            },
            {
              key: 'password',
              label: t('device.password.title'),
              forceRender: true,
              children: (
                <ErrorBoundary>
                  <PasswordManagementTab
                    deviceId={device.id}
                    deviceSn={displayDevice.sn}
                    productClass={displayDevice.productClass}
                    deviceModel={displayDevice.deviceModel}
                    networkType={displayDevice.networkType}
                    onTaskSubmitted={handlePasswordTaskSubmitted}
                    onTaskCleared={handlePasswordTaskCleared}
                  />
                </ErrorBoundary>
              ),
            },
          ]}
        />
      </Card>

      <Modal
        title={t('device.nameSync.editOmcName')}
        open={omcNameEditorOpen}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        confirmLoading={renameMutation.isPending}
        onOk={() => void handleRenameOMCName()}
        onCancel={() => setOmcNameEditorOpen(false)}
        destroyOnHidden
      >
        <Input
          value={omcNameDraft}
          maxLength={128}
          autoFocus
          placeholder={t('device.nameSync.omcNamePlaceholder')}
          onChange={(event) => setOmcNameDraft(event.target.value)}
          onPressEnter={() => void handleRenameOMCName()}
        />
      </Modal>

      <Modal
        title={t('device.locationSourceMode')}
        open={locationSourceEditorOpen}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        confirmLoading={locationSourceSaving}
        onOk={() => void handleSaveLocationSource()}
        onCancel={() => setLocationSourceEditorOpen(false)}
        destroyOnHidden
      >
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Alert type="info" showIcon title={t('device.locationSourceModeHint')} />
          <Select
            value={locationSourceDraft}
            style={{ width: '100%' }}
            onChange={setLocationSourceDraft}
            options={[
              { value: 'tr069', label: t('device.locationSourceTr069') },
              { value: 'external', label: t('device.locationSourceExternal') },
            ]}
          />
        </Space>
      </Modal>

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
