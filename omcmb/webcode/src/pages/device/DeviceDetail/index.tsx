import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Badge,
  Button,
  Card,
  Col,
  Descriptions,
  Radio,
  Row,
  Skeleton,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  Alert,
} from 'antd';
import {
  ArrowLeftOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import StatusIndicator from '@/components/StatusIndicator';
import { useDeviceBySn } from '@core/hooks/api/useDevices';
import { useCurrentAlarms } from '@core/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@core/types/alarm';
import type { Device } from '@core/types/device';
import ParameterTreeTab from './ParameterTreeTab';

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
    { key: 'productType', label: t('device.productType'), render: (d) => d.productType || '-' },
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
    { key: 'firstOnlineTime', label: t('device.firstOnlineTime'), render: (d) => fmtTime(d.firstOnlineTime) },
    { key: 'lastInformTime', label: t('device.lastInformTime'), render: (d) => fmtTime(d.lastInformTime) },
    // 站址信息
    { key: 'siteName', label: t('device.siteName'), render: (d) => d.siteName || '-' },
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
  const { sn = '' } = useParams<{ sn: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState('basic');

  const { data: device, isLoading, refetch } = useDeviceBySn(sn);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);

  const alarmParams = useMemo(
    () => ({ deviceSn: sn, page: 1, pageSize: 20 } as Parameters<typeof useCurrentAlarms>[0]),
    [sn]
  );
  const { data: alarmData, isLoading: alarmsLoading } = useCurrentAlarms(alarmParams);
  const alarms: Alarm[] = alarmData?.items ?? [];

  const alarmColumns = useMemo(
    (): DataTableColumn<Alarm>[] => [
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
    [t, SEVERITY_LABEL]
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
            <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
              {t('common.refresh')}
            </Button>
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
              label: t('nav.license.list').replace('管理', '').trim(),
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <Table
                    size="small"
                    dataSource={[
                      {
                        key: '1',
                        version: 'V2.1.0',
                        generatedTime: '2025-01-15 10:30:00',
                        mode: '永久',
                        featureId: 'BASIC-001',
                        description: '基础功能授权',
                        capacity: '不限',
                        expiryDate: '永久',
                        remainingDays: '-',
                      },
                      {
                        key: '2',
                        version: 'V2.1.0',
                        generatedTime: '2025-01-15 10:30:00',
                        mode: '时间限制',
                        featureId: 'HALOB-001',
                        description: 'HaloB License',
                        capacity: '100',
                        expiryDate: '2027-12-31',
                        remainingDays: 652,
                      },
                      {
                        key: '3',
                        version: 'V2.1.0',
                        generatedTime: '2025-06-01 14:20:00',
                        mode: '时间限制',
                        featureId: '5GNR-001',
                        description: '5G NR 授权',
                        capacity: '50',
                        expiryDate: '2027-06-30',
                        remainingDays: 468,
                      },
                      {
                        key: '4',
                        version: 'V1.5.0',
                        generatedTime: '2024-12-10 09:00:00',
                        mode: '时间限制',
                        featureId: 'LTE-ADV-001',
                        description: 'LTE Advanced 功能',
                        capacity: '200',
                        expiryDate: '2026-06-30',
                        remainingDays: 103,
                      },
                    ]}
                    rowKey="key"
                    pagination={false}
                    scroll={{ x: 1100 }}
                    columns={[
                      { title: 'License版本', dataIndex: 'version', key: 'version', width: 100, fixed: 'left' },
                      { title: '生成时间', dataIndex: 'generatedTime', key: 'generatedTime', width: 160 },
                      { title: '模式', dataIndex: 'mode', key: 'mode', width: 100, render: (v: string) => <Tag color={v === '永久' ? 'success' : 'processing'}>{v}</Tag> },
                      { title: '特性ID', dataIndex: 'featureId', key: 'featureId', width: 120, render: (v: string) => <Text style={{ fontFamily: 'monospace' }}>{v}</Text> },
                      { title: '描述', dataIndex: 'description', key: 'description', width: 150 },
                      { title: '数量', dataIndex: 'capacity', key: 'capacity', width: 80 },
                      { title: '有效期', dataIndex: 'expiryDate', key: 'expiryDate', width: 120 },
                      {
                        title: '剩余天数',
                        dataIndex: 'remainingDays',
                        key: 'remainingDays',
                        width: 100,
                        render: (v: number | string) => {
                          if (v === '-') return <Text type="secondary">-</Text>;
                          const days = Number(v);
                          let color = 'success';
                          if (days <= 30) color = 'error';
                          else if (days <= 90) color = 'warning';
                          return <Tag color={color}>{days} 天</Tag>;
                        },
                      },
                    ]}
                  />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}
