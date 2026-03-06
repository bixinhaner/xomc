import React, { useMemo } from 'react';
import { Card, Col, Row, Statistic, Typography } from 'antd';
import {
  AlertOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import PieChart from '@/components/Charts/PieChart';
import BarChart from '@/components/Charts/BarChart';
import LineChart from '@/components/Charts/LineChart';
import { useAlarmCount } from '@/hooks/api/useAlarms';
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

export default function AlarmStatistics() {
  const t = useT();
  const { data: alarmCount } = useAlarmCount();

  const critical = alarmCount?.critical ?? 8;
  const major = alarmCount?.major ?? 15;
  const minor = alarmCount?.minor ?? 12;
  const warning = alarmCount?.warning ?? 8;
  const totalActive = critical + major + minor + warning;

  // Severity distribution donut
  const severityData = useMemo(
    () => [
      { name: t('alarm.severity.critical'), value: critical },
      { name: t('alarm.severity.major'), value: major },
      { name: t('alarm.severity.minor'), value: minor },
      { name: t('alarm.severity.warning'), value: warning },
    ],
    [critical, major, minor, warning, t]
  );

  // 7-day trend (4 series)
  const trendXData = useMemo(() => generateTrendDays(7), []);
  const trendSeries = useMemo(() => [
    { name: t('alarm.severity.critical'), data: [5, 8, 6, 9, 7, 10, 8], color: '#F5222D' },
    { name: t('alarm.severity.major'), data: [12, 15, 11, 18, 14, 17, 15], color: '#FA8C16' },
    { name: t('alarm.severity.minor'), data: [8, 10, 9, 12, 11, 13, 12], color: '#FADB14' },
    { name: t('alarm.severity.warning'), data: [6, 7, 5, 8, 6, 9, 8], color: '#1677FF' },
  ], [t]);

  // TOP10 devices (horizontal bar)
  const top10Devices = [
    '成都基站-005', '广州基站-003', '昆明基站-010', '哈尔滨-009',
    '北京基站-001', '上海基站-002', '西安基站-007', '南京基站-008',
    '深圳基站-004', '兰州基站-006',
  ];
  const top10Series = [
    {
      name: t('alarm.total'),
      data: [24, 21, 18, 16, 14, 12, 10, 8, 6, 4],
      color: '#FA8C16',
    },
  ];

  // Alarm type distribution
  const alarmTypeXData = useMemo(() => ['Link Down', 'Link Fault', 'Threshold', 'Config', 'Software', 'Hardware'], []);
  const alarmTypeSeries = [
    {
      name: t('alarm.total'),
      data: [23, 18, 34, 12, 9, 7],
      color: 'var(--color-primary-600)',
    },
  ];

  // Alarm by hour heatmap-style bar
  const hourXData = Array.from({ length: 24 }, (_, i) => `${i}:00`);
  const hourSeries = [
    {
      name: t('alarm.total'),
      data: [3, 2, 1, 1, 2, 4, 6, 8, 10, 9, 8, 7, 6, 5, 7, 9, 10, 12, 11, 8, 7, 5, 4, 3],
    },
  ];

  // Summary cards
  const summaryCards = [
    {
      title: t('alarm.severity.critical'),
      value: critical,
      color: '#F5222D',
      bg: '#fff2f0',
      icon: <AlertOutlined />,
    },
    {
      title: t('alarm.severity.major'),
      value: major,
      color: '#FA8C16',
      bg: '#fff7e6',
      icon: <ExclamationCircleOutlined />,
    },
    {
      title: t('alarm.severity.minor'),
      value: minor,
      color: '#D4B106',
      bg: '#feffe6',
      icon: <WarningOutlined />,
    },
    {
      title: t('alarm.severity.warning'),
      value: warning,
      color: '#1677FF',
      bg: '#e6f4ff',
      icon: <InfoCircleOutlined />,
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Title level={4} style={{ margin: 0 }}>
        {t('nav.alarm.statistics')}
      </Title>

      {/* Summary Row */}
      <Row gutter={[16, 16]}>
        {summaryCards.map(({ title, value, color, bg, icon }) => (
          <Col key={title} xs={12} sm={6}>
            <Card styles={{ body: { padding: '16px 20px' } }}>
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
                  <Statistic
                    value={value}
                    valueStyle={{ color, fontSize: 22, fontWeight: 700 }}
                  />
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* Charts Row 1 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={10}>
          <Card title={t('alarm.severity')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <PieChart
              title=""
              data={severityData}
              height={300}
              donut
            />
            <div style={{ textAlign: 'center', paddingBottom: 12 }}>
              <Text type="secondary" style={{ fontSize: 13 }}>
                {t('alarm.active')} <Text strong style={{ color: '#F5222D' }}>{totalActive}</Text>
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={14}>
          <Card title={t('alarm.severity')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <LineChart
              title=""
              xData={trendXData}
              series={trendSeries}
              height={320}
              areaFill
            />
          </Card>
        </Col>
      </Row>

      {/* Charts Row 2 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="TOP10" size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <BarChart
              title=""
              xData={top10Devices}
              series={top10Series}
              height={300}
              horizontal
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('alarm.type')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <BarChart
              title=""
              xData={alarmTypeXData}
              series={alarmTypeSeries}
              height={300}
            />
          </Card>
        </Col>
      </Row>

      {/* Charts Row 3 -- Hourly distribution */}
      <Row gutter={[16, 16]}>
        <Col span={24}>
          <Card title={t('alarm.time')} size="small" styles={{ body: { padding: '8px 0 0' } }}>
            <BarChart title="" xData={hourXData} series={hourSeries} height={220} />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
