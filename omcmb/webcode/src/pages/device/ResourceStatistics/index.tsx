import React, { useMemo } from 'react';
import { Card, Col, Row, Statistic, Typography } from 'antd';
import {
  AppstoreOutlined,
  CloudServerOutlined,
  GlobalOutlined,
  WifiOutlined,
} from '@ant-design/icons';
import PieChart from '@/components/Charts/PieChart';
import BarChart from '@/components/Charts/BarChart';
import LineChart from '@/components/Charts/LineChart';
import { useDashboardData } from '@core/hooks/api/useDashboard';
import { useT } from '@/hooks/useT';

const { Title, Text } = Typography;

function generateTrendDays(count: number): string[] {
  const days: string[] = [];
  for (let i = count - 1; i >= 0; i--) {
    const d = new Date();
    d.setDate(d.getDate() - i);
    days.push(`${d.getMonth() + 1}/${d.getDate()}`);
  }
  return days;
}

export default function ResourceStatistics() {
  const t = useT();
  const { data: dashboardData, isLoading } = useDashboardData();

  const totalDevices = (dashboardData as Record<string, unknown> | undefined)?.totalDevices as number | undefined ?? 1284;
  const onlineDevices = (dashboardData as Record<string, unknown> | undefined)?.onlineDevices as number | undefined ?? 1137;
  const offlineDevices = totalDevices - onlineDevices;
  const onlineRate = Math.round((onlineDevices / totalDevices) * 100);

  // Device count by type
  const typeData = useMemo(
    () => [
      { name: 'eNB', value: 489 },
      { name: 'gNB', value: 376 },
      { name: 'CPE', value: 298 },
      { name: 'eGW', value: 121 },
    ],
    []
  );

  // Device count by vendor
  const vendorData = useMemo(
    () => [
      { name: '华为', value: 512 },
      { name: '中兴', value: 356 },
      { name: '爱立信', value: 248 },
      { name: '大唐', value: 104 },
      { name: '京信', value: 64 },
    ],
    []
  );

  // Region bar chart
  const regionXData = ['华北区', '华东区', '华南区', '西南区', '西北区', '东北区'];
  const regionSeries = useMemo(() => [
    { name: t('status.online'), data: [218, 312, 245, 167, 98, 97], color: '#52C41A' },
    { name: t('status.offline'), data: [18, 25, 21, 14, 12, 10], color: '#8C8C8C' },
  ], [t]);

  // 30-day online/offline trend
  const trendXData = useMemo(() => generateTrendDays(14), []);
  const trendSeries = useMemo(() => [
    {
      name: t('status.online'),
      data: [1120, 1125, 1118, 1130, 1132, 1128, 1135, 1133, 1138, 1137, 1140, 1137, 1139, 1137],
      color: '#52C41A',
    },
    {
      name: t('status.offline'),
      data: [164, 159, 166, 154, 152, 156, 149, 151, 146, 147, 144, 147, 145, 147],
      color: '#8C8C8C',
    },
  ], [t]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Title level={4} style={{ margin: 0 }}>
        {t('nav.device.stats')}
      </Title>

      {/* Summary Stats */}
      <Row gutter={[16, 16]}>
        {[
          {
            title: t('device.count.total'),
            value: totalDevices,
            icon: <AppstoreOutlined />,
            color: 'var(--color-primary-600)',
            bg: '#e6f4ff',
          },
          {
            title: t('status.online'),
            value: onlineDevices,
            icon: <WifiOutlined />,
            color: '#52C41A',
            bg: '#f6ffed',
          },
          {
            title: t('status.offline'),
            value: offlineDevices,
            icon: <CloudServerOutlined />,
            color: '#8C8C8C',
            bg: '#f5f5f5',
          },
          {
            title: t('common.realTimeConn'),
            value: `${onlineRate}%`,
            icon: <GlobalOutlined />,
            color: '#722ED1',
            bg: '#f9f0ff',
          },
        ].map(({ title, value, icon, color, bg }) => (
          <Col key={title} xs={12} sm={6}>
            <Card loading={isLoading} styles={{ body: { padding: '16px 20px' } }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div
                  style={{
                    width: 40,
                    height: 40,
                    borderRadius: 8,
                    background: bg,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 18,
                    color,
                    flexShrink: 0,
                  }}
                >
                  {icon}
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: 13, display: 'block' }}>
                    {title}
                  </Text>
                  <Text strong style={{ fontSize: 22, color }}>
                    {value}
                  </Text>
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* Charts Row 1 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={t('device.productType')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <PieChart title="" data={typeData} height={280} donut />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('device.vendor')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <PieChart title="" data={vendorData} height={280} />
          </Card>
        </Col>
      </Row>

      {/* Charts Row 2 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Card title={t('device.region')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <BarChart title="" xData={regionXData} series={regionSeries} height={280} />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card title={t('table.time')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <LineChart title="" xData={trendXData} series={trendSeries} height={280} />
          </Card>
        </Col>
      </Row>

      {/* Network Type Distribution */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={t('device.networkType')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <PieChart
              title=""
              data={[
                { name: 'LTE-FDD', value: 456 },
                { name: 'LTE-TDD', value: 312 },
                { name: 'NR (5G)', value: 386 },
                { name: 'NB-IoT', value: 130 },
              ]}
              height={260}
              donut
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('device.engStatus')} size="small" styles={{ body: { padding: '20px' } }}>
            <Row gutter={[16, 16]}>
              {[
                { label: t('device.engStatus.commissioned'), value: 1156, color: '#52C41A' },
                { label: t('device.engStatus.uncommissioned'), value: 98, color: '#FA8C16' },
                { label: t('device.engStatus.decommissioned'), value: 30, color: '#8C8C8C' },
              ].map(({ label, value, color }) => (
                <Col key={label} span={8}>
                  <Statistic
                    title={label}
                    value={value}
                    valueStyle={{ color, fontSize: 28, fontWeight: 700 }}
                  />
                </Col>
              ))}
            </Row>
            <div style={{ marginTop: 16, height: 140 }}>
              <BarChart
                title=""
                xData={[t('device.engStatus.commissioned'), t('device.engStatus.uncommissioned'), t('device.engStatus.decommissioned')]}
                series={[{ name: t('table.total'), data: [1156, 98, 30] }]}
                height={140}
              />
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
