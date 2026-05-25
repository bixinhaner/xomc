import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useTabStore } from '@core/store/tabStore';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import {
  Badge,
  Button,
  Card,
  Col,
  Descriptions,
  Dropdown,
  Radio,
  Row,
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
import { useDeviceBySn } from '@core/hooks/api/useDevices';
import { useQuickSettingsGroups } from '@core/hooks/api/useQuickSettings';
import { useAcknowledgeAlarms, useClearAlarms, useCurrentAlarms, useUnacknowledgeAlarms } from '@core/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@core/types/alarm';
import type { Device } from '@core/types/device';
import ParameterTreeTab from './ParameterTreeTab';
import QuickSettingsTab from './QuickSettingsTab';
import LicenseParamsTab from './LicenseParamsTab';
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

// eNB KPI 配置 (8 项)
const ENB_KPI_CONFIG_RAW: KPIConfigRaw[] = [
  { key: 'enbDownlinkPRBUtilizationRate', labelKey: 'kpi.enbDlPrbUtil', unit: '%', category: 'utilization' },
  { key: 'enbUplinkPRBUtilizationRate', labelKey: 'kpi.enbUlPrbUtil', unit: '%', category: 'utilization' },
  { key: 'enbHoS1SuccRate', labelKey: 'kpi.enbHoS1SuccRate', unit: '%', category: 'mobility' },
  { key: 'enbHoX2SuccRate', labelKey: 'kpi.enbHoX2SuccRate', unit: '%', category: 'mobility' },
  { key: 'enbHoInterEnbSuccRate', labelKey: 'kpi.enbHoInterSuccRate', unit: '%', category: 'mobility' },
  { key: 'enbRrcSetupSuccessRate', labelKey: 'kpi.enbRrcSetupSuccRate', unit: '%', category: 'accessibility' },
  { key: 'enbAvgThroughputDL', labelKey: 'kpi.enbAvgDlThroughput', unit: 'Mbps', category: 'traffic' },
  { key: 'enbAvgThroughputUL', labelKey: 'kpi.enbAvgUlThroughput', unit: 'Mbps', category: 'traffic' },
];

// gNB KPI 配置 (4 项)
const GNB_KPI_CONFIG_RAW: KPIConfigRaw[] = [
  { key: 'gnbThroughputDL', labelKey: 'kpi.dlThroughput', unit: 'Mbps', category: 'traffic' },
  { key: 'gnbThroughputUL', labelKey: 'kpi.ulThroughput', unit: 'Mbps', category: 'traffic' },
  { key: 'gnbDownlinkPRBUtilizationRate', labelKey: 'kpi.enbDlPrbUtil', unit: '%', category: 'utilization' },
  { key: 'gnbUplinkPRBUtilizationRate', labelKey: 'kpi.enbUlPrbUtil', unit: '%', category: 'utilization' },
];

// GSM KPI 配置 (3 项)
const GSM_KPI_CONFIG_RAW: KPIConfigRaw[] = [
  { key: 'gsmCallSetupSuccRate', labelKey: 'kpi.accessRate', unit: '%', category: 'accessibility' },
  { key: 'gsmCallDropRate', labelKey: 'kpi.dropRate', unit: '%', category: 'retainability' },
  { key: 'gsmHandoverSuccessRate', labelKey: 'kpi.handoverSuccessRate', unit: '%', category: 'mobility' },
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

// 生成趋势日期标签（天/周）
function generateTrendLabels(mode: 'day' | 'week'): string[] {
  if (mode === 'day') {
    // 按天：显示24小时整点
    return Array.from({ length: 24 }, (_, i) => `${i}:00`);
  } else {
    // 按周：显示最近7天
    const labels: string[] = [];
    const today = new Date();
    for (let i = 6; i >= 0; i--) {
      const d = new Date(today);
      d.setDate(d.getDate() - i);
      labels.push(`${d.getMonth() + 1}/${d.getDate()}`);
    }
    return labels;
  }
};

// 预定义颜色数组
const CHART_COLORS = [
  '#1677FF',
  '#52C41A',
  '#FA8C16',
  '#722ED1',
  '#13C2C2',
  '#EB2F96',
  '#1890FF',
  '#FAAD14',
];

// 根据KPI配置生成类别趋势数据
const generateCategoryTrendSeries = (kpis: KPIConfig[], mode: 'day' | 'week') => {
  const count = mode === 'day' ? 24 : 7; // 按天24个点，按周7个点
  const randomData = (base: number = 50, range: number = 40) =>
    Array.from({ length: count }, () => Math.floor(Math.random() * range) + base);

  return kpis.map((kpi, index) => ({
    name: kpi.label,
    data: randomData(),
    color: CHART_COLORS[index % CHART_COLORS.length],
  }));
};

// ─── 字段定义组件 ────────────────────────────────────────────────────────

interface FieldItem {
  key: string;
  label: string;
  render: (device: Device) => React.ReactNode;
}

interface FieldGroup {
  title: string;
  fields: FieldItem[];
}

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
  const entry = map[value];
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

  // eNB/gNB 共享字段
  if (networkType === 'eNB' || networkType === 'gNB') {
    fields.push(
      { key: 'ipsecAddr', label: t('device.ipsecAddr'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.ipsecAddr || '-'}</Text> },
      { key: 'halobFlag', label: 'HaloB', render: (d) => <Tag color={d.halobFlag ? 'success' : 'default'}>{d.halobFlag ? t('status.enabled') : t('status.disabled')}</Tag> },
      { key: 'adminState', label: 'Admin State', render: (d) => renderStatusTag(String(d.adminState), { '1': { label: 'Locked', color: 'warning' }, '2': { label: 'Unlocked', color: 'success' }, '3': { label: 'ShuttingDown', color: 'error' } }) },
    );
  }

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      { key: 'gpsVersion', label: t('device.gpsVersion'), render: (d) => d.gpsVersion ?? '-' },
      { key: 'rom', label: 'ROM', render: (d) => d.rom ?? '-' },
      { key: 'mmepoolIpsecAddr', label: t('device.mmepoolIpsecAddr'), render: (d) => <Text style={{ fontFamily: 'monospace' }}>{d.mmepoolIpsecAddr || '-'}</Text> },
    );
  }

  // gNB 独有字段
  if (networkType === 'gNB') {
    fields.push(
      { key: 'rollbackVersion', label: t('device.rollbackVersion'), render: (d) => d.rollbackVersion ?? '-' },
      { key: 'sasParam', label: t('device.sasParam'), render: (d) => d.sasParam ?? '-' },
      { key: 'euRu', label: t('device.euRu'), render: (d) => d.euRu ?? '-' },
      { key: 'halobLicense', label: t('device.halobLicense'), render: (d) => d.halobLicense ?? '-' },
      { key: 'energySaving', label: t('device.energySaving'), render: (d) => d.energySaving ?? '-' },
      { key: 'gnbTopoCellmgr', label: t('device.gnbTopoCellmgr'), render: (d) => d.gnbTopoCellmgr ?? '-' },
      { key: 'sslCertValidity', label: t('device.sslCertValidity'), render: (d) => d.sslCertValidity ?? '-' },
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
      { key: 'bandwidth', label: t('device.bandwidth'), render: (d) => d.bandwidth ?? '-' },
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
    { key: 'connStatus', label: t('device.connStatus'), render: (d) => <StatusIndicator status={d.connStatus === 'online' ? 'online' : 'offline'} /> },
    { key: 'opState', label: t('device.opState'), render: (d) => renderStatusTag(d.opState, { '1': { label: t('status.active'), color: 'success' }, '0': { label: t('status.inactive'), color: 'error' }, active: { label: t('status.active'), color: 'success' }, inactive: { label: t('status.inactive'), color: 'error' } }) },
    { key: 'rfStatus', label: t('device.rfStatus'), render: (d) => renderStatusTag(d.rfStatus, { on: { label: t('status.rfOn'), color: 'success' }, off: { label: t('status.rfOff'), color: 'error' }, '1': { label: t('status.rfOn'), color: 'success' }, '0': { label: t('status.rfOff'), color: 'error' } }) },
    { key: 'ueCount', label: t('device.ueCount'), render: (d) => d.ueCount ?? '-' },
    { key: 'syncStatus', label: t('device.syncStatus'), render: (d) => d.syncStatus || '-' },
  ];

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      { key: 'mmeStatus', label: t('device.mmeStatus'), render: (d) => d.mmeStatus ?? '-' },
      { key: 'pmReportStatus', label: t('device.pmReportStatus'), render: (d) => d.pmReportStatus ?? '-' },
      { key: 'cpeCount', label: t('device.cpeCount'), render: (d) => d.cpeCount ?? '-' },
      { key: 'lockStatus', label: t('device.lockStatus'), render: (d) => renderStatusTag(d.lockStatus, { locked: { label: t('status.locked'), color: 'warning' }, unlocked: { label: t('status.unlocked'), color: 'success' } }) },
      { key: 'wanSpeed', label: t('device.wanSpeed'), render: (d) => d.wanSpeed ?? '-' },
      { key: 'serviceStatus', label: t('device.serviceStatus'), render: (d) => d.serviceStatus ?? '-' },
      { key: 'validity', label: t('device.validity'), render: (d) => d.validity ?? '-' },
    );
  }

  // gNB 独有字段
  if (networkType === 'gNB') {
    fields.push(
      { key: 'amfStatus', label: t('device.amfStatus'), render: (d) => d.amfStatus ?? '-' },
      { key: 'multiPlmnEnable', label: 'Multi PLMN', render: (d) => renderStatusTag(d.multiPlmnEnable, { enabled: { label: t('status.enabled'), color: 'success' }, disabled: { label: t('status.disabled'), color: 'default' } }) },
      { key: 'euCount', label: t('device.euCount'), render: (d) => d.euCount ?? '-' },
      { key: 'ruCount', label: t('device.ruCount'), render: (d) => d.ruCount ?? '-' },
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

const getOtherFields = (t: ReturnType<typeof useT>, networkType: string): FieldGroup => {
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
    // 位置信息
    { key: 'longitude', label: t('device.longitude'), render: (d) => d.longitude?.toFixed(4) || '-' },
    { key: 'latitude', label: t('device.latitude'), render: (d) => d.latitude?.toFixed(4) || '-' },
    { key: 'gpsHeight', label: t('device.gpsHeight'), render: (d) => d.gpsHeight ?? '-' },
    // 备注
    { key: 'remark', label: t('device.remark'), render: (d) => d.remark || '-' },
  ];

  // eNB 独有字段
  if (networkType === 'eNB') {
    fields.push(
      { key: 'mechanicalDowntilt', label: t('device.mechanicalDowntilt'), render: (d) => d.mechanicalDowntilt ?? '-' },
      { key: 'electronicDowntilt', label: t('device.electronicDowntilt'), render: (d) => d.electronicDowntilt ?? '-' },
      { key: 'verticalBeamWidth', label: t('device.verticalBeamWidth'), render: (d) => d.verticalBeamWidth ?? '-' },
      { key: 'horizontalAzimuth', label: t('device.horizontalAzimuth'), render: (d) => d.horizontalAzimuth ?? '-' },
      { key: 'installAddress', label: t('device.installAddress'), render: (d) => d.installAddress || '-' },
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

const renderFieldGroup = (group: FieldGroup, device: Device) => (
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

// ─── KPI Tab 组件─────────────────────────────────────────────────────────

interface KPITabContentProps {
  device: Device;
  t: ReturnType<typeof useT>;
}

function KPITabContent({ device, t }: KPITabContentProps) {
  const [timeMode, setTimeMode] = useState<'day' | 'week'>('day');

  const networkType = device.networkType ?? '';
  const kpiConfig = getKPIConfig(networkType, t);

  if (kpiConfig.length === 0) {
    return (
      <div style={{ padding: '0 0 16px' }}>
        <Alert
          type="info"
          message={t('common.noData')}
          description={`暂无 ${networkType || '未知制式'} 的 KPI 指标配置`}
          showIcon
        />
      </div>
    );
  }

  const trendLabels = generateTrendLabels(timeMode);

  return (
    <div style={{ padding: '0 0 16px' }}>
      {/* 时间维度切换 */}
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
        <Radio.Group
          value={timeMode}
          onChange={(e) => setTimeMode(e.target.value)}
          optionType="button"
          buttonStyle="solid"
          size="small"
        >
          <Radio.Button value="day">按天</Radio.Button>
          <Radio.Button value="week">按周</Radio.Button>
        </Radio.Group>
      </div>

      {/* 每个 KPI 指标单独显示趋势图 */}
      <Row gutter={[16, 16]}>
        {kpiConfig.map((kpi) => {
          const trendSeries = generateCategoryTrendSeries([kpi], timeMode);

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
                  {kpi.label}
                </div>
                <LineChart
                  title=""
                  xData={trendLabels}
                  series={trendSeries}
                  height={220}
                  areaFill
                />
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
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const openTab = useTabStore((s) => s.openTab);
  const [searchParams, setSearchParams] = useSearchParams();

  const { data: device, isLoading, refetch } = useDeviceBySn(sn);

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

  // 将设备详情页注册为 TabBar 的共享 tab（key='device-detail'）。
  // 同 key 复用槽位、按 sn 替换 path/label —— 避免每台设备各占一个 tab；
  // path 带 location.search 以保留 ?tab=alarm/gps 等深链接进入时的子 tab。
  useEffect(() => {
    if (!sn) return;
    const displayName = device?.name || sn;
    openTab({
      key: 'device-detail',
      label: `${t('common.detail')} · ${displayName}`,
      labelRaw: true,
      path: `/device/detail/${sn}${location.search}`,
      closable: true,
    });
  }, [sn, location.search, device?.name, openTab, t]);

  // 关闭"详情"页（用户点 × 关 device-detail tab）时清掉该设备的 form 草稿 + 反馈状态。
  // 与"切到其他 tab"区分：切走时 tabStore 里 device-detail tab 仍存在；关闭后才被移除。
  // unmount 时检查 tabStore 当前 state，按存在性判断意图。
  const did = device?.id;
  useEffect(() => {
    if (!did) return;
    return () => {
      const hasTab = useTabStore.getState().tabs.some((tb) => tb.key === 'device-detail');
      if (!hasTab) {
        useQuickSettingsFeedbackStore.getState().clearByDevice(did);
      }
    };
  }, [did]);

  // T-0138:快速设置 tab 显示规则 —— 只在该设备对应 paramModel 有 quicksettings XML 时显示
  // (后端 GET /quicksettings/groups?device_id=... 返回空 groups 即视为未配置)
  const { data: quickSettingsData } = useQuickSettingsGroups(device?.id);
  const showQuickSettingsTab = (quickSettingsData?.groups?.length ?? 0) > 0;

  const handleHeaderRefresh = useCallback(() => {
    void refetch();
    const deviceId = device?.id;
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
          // 方案 B：刷新 = 回到服务器状态，丢前端临时编辑（drafts + 组件内 form/rowEdits 全清）
          useQuickSettingsFeedbackStore.getState().clearByDevice(deviceId);
          // bump refreshTick → QuickSettingsTab 拼进子组件 key 触发 CellParameterForm / MultiInstanceTable 整体 remount，
          // 清掉 form.isFieldTouched / rowEdits 等组件内 state；React Query schema 失效后会重拉最新值
          useQuickSettingsFeedbackStore.getState().bumpRefreshTick(deviceId);
          void queryClient.invalidateQueries({ queryKey: ['quicksettings', 'groups', deviceId] });
          // CellParameterForm + MultiInstanceTable 都用 useParameterSchema(deviceId, prefix) — 按 deviceId 前缀失效
          void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
        }
        break;
      default:
        break;
    }
  }, [activeTab, refetch, queryClient, device?.id]);

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

  const alarmParams = useMemo(
    () => ({ deviceSn: sn, page: 1, pageSize: 20 } as Parameters<typeof useCurrentAlarms>[0]),
    [sn]
  );
  const { data: alarmData, isLoading: alarmsLoading, refetch: refetchAlarms } = useCurrentAlarms(alarmParams);
  const alarms: Alarm[] = alarmData?.items ?? [];

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
    if (!device) return [];
    const networkType = device.networkType ?? '';

    return [
      getStationFields(t, networkType),
      getCellFields(t, networkType),
      getStatusFields(t, networkType),
      getOtherFields(t, networkType),
    ];
  }, [device, t]);

  if (isLoading) {
    return (
      <div style={{ padding: 24 }}>
        <Skeleton active paragraph={{ rows: 8 }} />
      </div>
    );
  }

  if (!device) {
    return (
      <div style={{ padding: 24 }}>
        <Alert
          type="error"
          message={t('common.noData')}
          description={`SN: "${sn}"`}
          action={
            <Button onClick={() => void navigate('/device/list')}>{t('common.back')}</Button>
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
              onClick={() => void navigate('/device/list')}
            >
              {t('common.back')}
            </Button>
            <div>
              <Title level={4} style={{ margin: 0 }}>
                {device.name}
              </Title>
              <Text type="secondary" style={{ fontFamily: 'monospace', fontSize: 13 }}>
                {device.sn}
              </Text>
            </div>
            <StatusIndicator
              status={device.connStatus === 'online' ? 'online' : 'offline'}
              variant="tag"
            />
            {device.alarmLevel !== 'none' && (
              <Tag color={SEVERITY_COLOR[device.alarmLevel]}>
                {SEVERITY_LABEL[device.alarmLevel]}
              </Tag>
            )}
          </div>
          <Space>
            {/* license tab 自带"刷新"按钮，此处头部刷新隐藏，避免同页两个刷新按钮 */}
            {activeTab !== 'license' && (
              <Button icon={<ReloadOutlined />} onClick={handleHeaderRefresh}>
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
                  {detailGroups.map((group) => renderFieldGroup(group, device))}
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
              children: <KPITabContent device={device} t={t} />,
            },
            {
              key: 'license',
              label: t('device.licenseParam.title'),
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
