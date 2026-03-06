import React, { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Badge,
  Button,
  Card,
  Col,
  Descriptions,
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
  EditOutlined,
  ReloadOutlined,
  WifiOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import StatusIndicator from '@/components/StatusIndicator';
import { useDeviceBySn } from '@/hooks/api/useDevices';
import { useCurrentAlarms } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';
import type { Alarm } from '@/types/alarm';

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
  { key: 'snmpCommunity', name: 'SNMP Community', value: 'public', unit: '-', category: 'Alarm' },
  { key: 'alarmThreshold', name: 'alarmThreshold', value: '80', unit: '%', category: 'Alarm' },
];

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
      { key: 'alarmContent', title: t('alarm.content'), dataIndex: 'alarmContent', width: 220, ellipsis: true },
      {
        key: 'alarmTime',
        title: t('alarm.time'),
        dataIndex: 'alarmTime',
        width: 160,
        render: (_val, record) => new Date(record.alarmTime).toLocaleString('zh-CN'),
      },
      {
        key: 'ackStatus',
        title: t('alarm.ackStatus'),
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
            <Button
              type="primary"
              icon={<EditOutlined />}
              onClick={() => void navigate(`/device/edit/${device.id}`)}
            >
              {t('common.edit')}
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
                <div style={{ padding: '0 0 16px' }}>
                  <Row gutter={[16, 16]}>
                    <Col span={24}>
                      <Descriptions
                        title={t('common.detail')}
                        bordered
                        column={{ xs: 1, sm: 2, md: 3 }}
                        size="small"
                      >
                        <Descriptions.Item label={t('device.sn')}>
                          <Text style={{ fontFamily: 'monospace' }}>{device.sn}</Text>
                        </Descriptions.Item>
                        <Descriptions.Item label={t('device.name')}>{device.name}</Descriptions.Item>
                        <Descriptions.Item label={t('device.vendor')}>{device.vendor}</Descriptions.Item>
                        <Descriptions.Item label={t('device.productType')}>{device.productType}</Descriptions.Item>
                        <Descriptions.Item label={t('device.networkType')}>{device.networkType}</Descriptions.Item>
                        <Descriptions.Item label={t('device.model')}>{device.deviceModel}</Descriptions.Item>
                        <Descriptions.Item label={t('device.connStatus')}>
                          <StatusIndicator
                            status={device.connStatus === 'online' ? 'online' : 'offline'}
                          />
                        </Descriptions.Item>
                        <Descriptions.Item label={t('device.engStatus')}>
                          <Tag color={device.engStatus === 'commissioned' ? 'success' : 'default'}>
                            {ENG_STATUS_LABEL[device.engStatus] ?? device.engStatus}
                          </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label={t('table.status')}>
                          <Tag color={device.mgmtStatus === 'managed' ? 'processing' : 'default'}>
                            {MGMT_STATUS_LABEL[device.mgmtStatus] ?? device.mgmtStatus}
                          </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label={t('device.ipAddress')}>
                          <Text style={{ fontFamily: 'monospace' }}>{device.ipAddress}</Text>
                        </Descriptions.Item>
                        <Descriptions.Item label={t('table.description')}>
                          <Text style={{ fontFamily: 'monospace' }}>{device.subnet}</Text>
                        </Descriptions.Item>
                        <Descriptions.Item label={t('device.region')}>{device.region}</Descriptions.Item>
                        <Descriptions.Item label={t('table.site')}>{device.site}</Descriptions.Item>
                        <Descriptions.Item label={t('device.softwareVersion')}>
                          <Text style={{ fontFamily: 'monospace' }}>{device.softwareVersion}</Text>
                        </Descriptions.Item>
                        <Descriptions.Item label={t('device.lastOnline')}>
                          {device.lastOnlineTime
                            ? new Date(device.lastOnlineTime).toLocaleString('zh-CN')
                            : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('table.createTime')}>
                          {device.createTime
                            ? new Date(device.createTime).toLocaleString('zh-CN')
                            : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('alarm.location')}>
                          {device.longitude.toFixed(4)}, {device.latitude.toFixed(4)}
                        </Descriptions.Item>
                      </Descriptions>
                    </Col>
                  </Row>
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
              label: t('table.result'),
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
                      {
                        title: t('table.operation'),
                        key: 'actions',
                        width: 80,
                        render: () => (
                          <Button type="link" size="small" icon={<EditOutlined />}>
                            {t('common.edit')}
                          </Button>
                        ),
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
