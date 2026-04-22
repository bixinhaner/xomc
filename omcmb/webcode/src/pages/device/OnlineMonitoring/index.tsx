import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Badge, Button, Card, Col, Input, Progress, Row, Select, Space, Tag, Typography } from 'antd';
import {
  EyeOutlined,
  ReloadOutlined,
  SearchOutlined,
  WifiOutlined,
} from '@ant-design/icons';
import StatusIndicator from '@/components/StatusIndicator';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device } from '@core/types/device';

const { Text, Title } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: '#F5222D', major: '#FA8C16', minor: '#FADB14', warning: '#1677FF', none: '#52C41A',
};

function getMetricColor(value: number, warn = 70, crit = 90): string {
  if (value >= crit) return '#F5222D';
  if (value >= warn) return '#FA8C16';
  return '#52C41A';
}

// Generate mock per-device metrics deterministically from SN
function getMockMetrics(sn: string) {
  const hash = sn.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0);
  const cpu = ((hash * 37) % 60) + 20;
  const memory = ((hash * 53) % 50) + 40;
  const temperature = ((hash * 17) % 25) + 30;
  const uptime = ((hash * 7) % 720) + 24;
  return { cpu, memory, temperature, uptime };
}

function DeviceCard({
  device,
  onView,
  t,
}: {
  device: Device;
  onView: (sn: string) => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}) {
  const metrics = useMemo(() => getMockMetrics(device.sn), [device.sn]);
  const isOnline = device.connStatus === 'online';

  return (
    <Card
      size="small"
      hoverable
      style={{
        borderRadius: 8,
        border: device.alarmLevel !== 'none'
          ? `1px solid ${SEVERITY_COLOR[device.alarmLevel]}`
          : '1px solid #f0f0f0',
      }}
      styles={{ body: { padding: 12 } }}
    >
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Text
            strong
            style={{ display: 'block', fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
            title={device.name}
          >
            {device.name}
          </Text>
          <Text
            type="secondary"
            style={{ fontFamily: 'monospace', fontSize: 11, display: 'block' }}
          >
            {device.sn}
          </Text>
        </div>
        <Badge
          status={isOnline ? 'success' : 'default'}
          text={isOnline ? t('status.online') : t('status.offline')}
          style={{ flexShrink: 0, marginLeft: 8, fontSize: 12 }}
        />
      </div>

      {/* Metrics (only show for online devices) */}
      {isOnline ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 10 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Text style={{ fontSize: 11, color: '#8c8c8c', width: 50, flexShrink: 0 }}>CPU</Text>
            <Progress
              percent={metrics.cpu}
              size="small"
              strokeColor={getMetricColor(metrics.cpu)}
              style={{ flex: 1, margin: 0 }}
              format={(p) => <span style={{ fontSize: 10 }}>{p}%</span>}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Text style={{ fontSize: 11, color: '#8c8c8c', width: 50, flexShrink: 0 }}>Memory</Text>
            <Progress
              percent={metrics.memory}
              size="small"
              strokeColor={getMetricColor(metrics.memory)}
              style={{ flex: 1, margin: 0 }}
              format={(p) => <span style={{ fontSize: 10 }}>{p}%</span>}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Text style={{ fontSize: 11, color: '#8c8c8c', width: 50, flexShrink: 0 }}>Temp</Text>
            <Text
              style={{
                fontSize: 12,
                color: getMetricColor(metrics.temperature, 45, 55),
                fontWeight: 600,
              }}
            >
              {metrics.temperature}C
            </Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Text style={{ fontSize: 11, color: '#8c8c8c', width: 50, flexShrink: 0 }}>{t('status.online')}</Text>
            <Text style={{ fontSize: 12 }}>{metrics.uptime}h</Text>
          </div>
        </div>
      ) : (
        <div style={{ textAlign: 'center', padding: '12px 0', color: '#8c8c8c', fontSize: 12 }}>
          {t('status.offline')}
        </div>
      )}

      {/* Tags */}
      <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 8 }}>
        <Tag style={{ margin: 0, fontSize: 11 }}>{device.productType}</Tag>
        <Tag style={{ margin: 0, fontSize: 11 }}>{device.networkType}</Tag>
        {device.alarmLevel !== 'none' && (
          <Tag
            color={
              device.alarmLevel === 'critical'
                ? 'red'
                : device.alarmLevel === 'major'
                ? 'orange'
                : 'gold'
            }
            style={{ margin: 0, fontSize: 11 }}
          >
            {t('common.hasAlarm')}
          </Tag>
        )}
      </div>

      {/* Footer */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text type="secondary" style={{ fontSize: 11 }}>
          {device.region}
        </Text>
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          style={{ padding: '0 4px', height: 'auto', fontSize: 12 }}
          onClick={() => onView(device.sn)}
        >
          {t('common.detail')}
        </Button>
      </div>
    </Card>
  );
}

export default function OnlineMonitoring() {
  const t = useT();
  const navigate = useNavigate();
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [typeFilter, setTypeFilter] = useState<string>('all');

  const { data, isLoading, refetch } = useDeviceList({ page: 1, pageSize: 100 } as Parameters<typeof useDeviceList>[0]);
  const allDevices: Device[] = data?.items ?? [];

  const filteredDevices = useMemo(() => {
    return allDevices.filter((d) => {
      const matchText =
        !searchText ||
        d.name.toLowerCase().includes(searchText.toLowerCase()) ||
        d.sn.toLowerCase().includes(searchText.toLowerCase());
      const matchStatus = statusFilter === 'all' || d.connStatus === statusFilter;
      const matchType = typeFilter === 'all' || d.productType === typeFilter;
      return matchText && matchStatus && matchType;
    });
  }, [allDevices, searchText, statusFilter, typeFilter]);

  const onlineCount = filteredDevices.filter((d) => d.connStatus === 'online').length;
  const offlineCount = filteredDevices.filter((d) => d.connStatus === 'offline').length;
  const alarmCount = filteredDevices.filter((d) => d.alarmLevel !== 'none').length;

  const handleView = useCallback(
    (sn: string) => void navigate(`/device/detail/${sn}`),
    [navigate]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16, flexWrap: 'wrap' }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('nav.device.monitor')}
          <WifiOutlined style={{ marginLeft: 8, fontSize: 16, color: '#52C41A' }} />
        </Title>
        <Space wrap>
          <Badge dot status="success" />
          <Text style={{ fontSize: 13 }}>{t('status.online')}: {onlineCount}</Text>
          <Badge dot status="default" />
          <Text style={{ fontSize: 13 }}>{t('status.offline')}: {offlineCount}</Text>
          <Badge dot status="warning" />
          <Text style={{ fontSize: 13 }}>{t('common.hasAlarm')}: {alarmCount}</Text>
        </Space>
      </div>

      {/* Filters */}
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
        <Input
          placeholder={t('common.placeholder')}
          prefix={<SearchOutlined />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
          style={{ width: 220 }}
        />
        <Select
          value={statusFilter}
          onChange={setStatusFilter}
          style={{ width: 120 }}
          options={[
            { label: t('common.all'), value: 'all' },
            { label: t('status.online'), value: 'online' },
            { label: t('status.offline'), value: 'offline' },
          ]}
        />
        <Select
          value={typeFilter}
          onChange={setTypeFilter}
          style={{ width: 120 }}
          options={[
            { label: t('common.all'), value: 'all' },
            { label: 'eNB', value: 'eNB' },
            { label: 'gNB', value: 'gNB' },
            { label: 'CPE', value: 'CPE' },
            { label: 'eGW', value: 'eGW' },
          ]}
        />
        <Button icon={<ReloadOutlined />} onClick={() => void refetch()} loading={isLoading}>
          {t('common.refresh')}
        </Button>
        <Text type="secondary" style={{ lineHeight: '32px', fontSize: 13 }}>
          {t('table.total')} {filteredDevices.length}
        </Text>
      </div>

      {/* Device Grid */}
      <Row gutter={[12, 12]}>
        {filteredDevices.map((device) => (
          <Col key={device.id} xs={24} sm={12} md={8} lg={6} xl={4}>
            <DeviceCard device={device} onView={handleView} t={t} />
          </Col>
        ))}
        {filteredDevices.length === 0 && !isLoading && (
          <Col span={24}>
            <div style={{ textAlign: 'center', padding: '60px 0', color: '#8c8c8c' }}>
              {t('common.noData')}
            </div>
          </Col>
        )}
      </Row>
    </div>
  );
}
