import React, { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Badge,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
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
import { useDeviceBySn } from '@/hooks/api/useDevices';
import { useCurrentAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@/types/alarm';
import type { Device } from '@/types/device';

const { Title, Text } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

// Mock KPI trend data (7 days)
function generateTrendDays() {
  const days: string[] = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date();
    d.setDate(d.getDate() - i);
    days.push(`${d.getMonth() + 1}/${d.getDate()}`);
  }
  return days;
}

const MOCK_XDATA = generateTrendDays();
const MOCK_KPI_SERIES = [
  { name: 'CPU(%)', data: [45, 52, 48, 61, 55, 58, 53], color: 'var(--color-primary-600)' },
  { name: 'Memory(%)', data: [62, 65, 60, 68, 63, 67, 64], color: '#52C41A' },
  { name: 'Temp(C)', data: [38, 40, 37, 42, 39, 41, 40], color: '#FA8C16' },
];

const MOCK_CONFIG_PARAMS = [
  { key: 'heartbeatInterval', name: 'heartbeatInterval', value: '30', unit: 's', category: 'Connection' },
  { key: 'reconnectRetry', name: 'reconnectRetry', value: '3', unit: '-', category: 'Connection' },
  { key: 'maxBandwidth', name: 'maxBandwidth', value: '100', unit: 'Mbps', category: 'Network' },
  { key: 'qosLevel', name: 'QoS', value: '3', unit: '-', category: 'Network' },
  { key: 'logLevel', name: 'logLevel', value: 'INFO', unit: '-', category: 'System' },
  { key: 'ntpServer', name: 'NTP Server', value: '10.0.0.1', unit: '-', category: 'System' },
];

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
const fmtDuration = (seconds: number | undefined | null) => {
  if (!seconds) return '-';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
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

  const ENG_STATUS_LABEL: Record<string, string> = useMemo(() => ({
    commissioned: t('device.engStatus.commissioned'),
    uncommissioned: t('device.engStatus.uncommissioned'),
    decommissioned: t('device.engStatus.decommissioned'),
  }), [t]);

  const MGMT_STATUS_LABEL: Record<string, string> = useMemo(() => ({
    managed: t('status.managed'),
    unmanaged: t('status.unmanaged'),
    'pre-managed': t('status.pending'),
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
      { key: 'alarmCode', title: t('alarm.code'), dataIndex: 'alarmCode', width: 100, mono: true },
      { key: 'alarmName', title: t('alarm.name'), dataIndex: 'alarmName', width: 180, ellipsis: true },
      { key: 'alarmContent', title: t('alarm.possibleCause'), dataIndex: 'alarmContent', width: 220, ellipsis: true },
      {
        key: 'alarmTime',
        title: t('alarm.time'),
        dataIndex: 'alarmTime',
        width: 160,
        render: (_val, record) => new Date(record.alarmTime).toLocaleString('zh-CN'),
      },
      {
        key: 'ackStatus',
        title: t('alarm.status'),
        dataIndex: 'ackStatus',
        width: 100,
        render: (_val, record) => (
          <Tag color={record.ackStatus === 'acknowledged' ? 'success' : 'warning'}>
            {record.ackStatus === 'acknowledged' ? t('alarm.ackStatus.acknowledged') : t('alarm.ackStatus.unacknowledged')}
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
                    defaultDensity="compact"
                    alarmRowStyle={(record) => record.severity as 'critical' | 'major' | 'minor' | 'warning'}
                  />
                </div>
              ),
            },
            {
              key: 'performance',
              label: 'KPI',
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <Card size="small" title="KPI" style={{ marginBottom: 16 }}>
                    <LineChart
                      title=""
                      xData={MOCK_XDATA}
                      series={MOCK_KPI_SERIES}
                      height={300}
                      areaFill
                    />
                  </Card>
                  <Row gutter={[16, 16]}>
                    {[
                      { label: 'CPU', value: '53%', color: 'var(--color-primary-600)' },
                      { label: 'Memory', value: '64%', color: '#52C41A' },
                      { label: 'Disk', value: '45%', color: '#722ED1' },
                      { label: 'Temp', value: '40C', color: '#FA8C16' },
                      { label: 'Uplink', value: '12.3 Mbps', color: '#13C2C2' },
                      { label: 'Downlink', value: '45.6 Mbps', color: '#EB2F96' },
                    ].map(({ label, value, color }) => (
                      <Col key={label} xs={12} sm={8} md={6}>
                        <Card size="small" styles={{ body: { padding: '12px 16px' } }}>
                          <Text type="secondary" style={{ fontSize: 12, display: 'block' }}>
                            {label}
                          </Text>
                          <Text strong style={{ fontSize: 20, color }}>
                            {value}
                          </Text>
                        </Card>
                      </Col>
                    ))}
                  </Row>
                </div>
              ),
            },
            {
              key: 'config',
              label: t('table.description'),
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <Table
                    size="small"
                    dataSource={MOCK_CONFIG_PARAMS}
                    rowKey="key"
                    pagination={false}
                    columns={[
                      { title: t('table.name'), dataIndex: 'name', key: 'name', width: 200 },
                      { title: 'Key', dataIndex: 'key', key: 'key', width: 200, render: (v: string) => <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</Text> },
                      { title: t('table.result'), dataIndex: 'value', key: 'value', width: 150, render: (v: string) => <Text strong>{v}</Text> },
                      { title: t('table.type'), dataIndex: 'unit', key: 'unit', width: 80 },
                      { title: t('table.vendor'), dataIndex: 'category', key: 'category', width: 120, render: (v: string) => <Tag>{v}</Tag> },
                    ]}
                  />
                </div>
              ),
            },
            {
              key: 'license',
              label: t('nav.license.list').replace('管理', '').trim(),
              children: (
                <div style={{ padding: '0 0 16px' }}>
                  <Table
                    size="small"
                    dataSource={[
                      { key: 'basic', name: '基础功能授权', status: 'active', expiryDate: '永久', capacity: '不限', used: 128 },
                      { key: 'halob', name: 'HaloB License', status: 'active', expiryDate: '2027-12-31', capacity: '100', used: 45 },
                      { key: '5g', name: '5G NR 授权', status: 'active', expiryDate: '2027-06-30', capacity: '50', used: 12 },
                    ]}
                    rowKey="key"
                    pagination={false}
                    columns={[
                      { title: t('license.licenseName'), dataIndex: 'name', key: 'name', width: 200 },
                      {
                        title: t('status.status'),
                        dataIndex: 'status',
                        key: 'status',
                        width: 100,
                        render: (v: string) => <Tag color={v === 'active' ? 'success' : 'warning'}>{v === 'active' ? t('status.active') : t('status.inactive')}</Tag>,
                      },
                      { title: t('license.expiryDate'), dataIndex: 'expiryDate', key: 'expiryDate', width: 120 },
                      { title: t('license.capacity'), dataIndex: 'capacity', key: 'capacity', width: 100 },
                      {
                        title: t('license.used'),
                        dataIndex: 'used',
                        key: 'used',
                        width: 100,
                        render: (_v: number, record) => `${record.used} / ${record.capacity}`,
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
