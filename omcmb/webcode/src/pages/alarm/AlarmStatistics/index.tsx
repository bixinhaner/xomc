import React, { useMemo, useState } from 'react';
import { Card, Col, Row, Statistic, Typography, DatePicker, Radio, Space } from 'antd';
import {
  AlertOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import BarChart from '@/components/Charts/BarChart';
import { useAlarmCount } from '@/hooks/api/useAlarms';
import { useT } from '@/hooks/useT';

const { Title, Text } = Typography;

const SEVERITY_COLORS = {
  critical: '#FC5959',
  major: '#FF973E',
  minor: '#FFDA41',
  warning: '#67DFF8',
};

function generateHourLabels(): string[] {
  return Array.from({ length: 24 }, (_, i) => `${i}:00`);
}

function generateDayLabels(count: number): string[] {
  const days: string[] = [];
  for (let i = count - 1; i >= 0; i--) {
    const d = dayjs().subtract(i, 'day');
    days.push(d.format('MM/DD'));
  }
  return days;
}

// Mock data generator for stacked bar chart
function generateMockStackedData(baseMultiplier: number = 1) {
  return {
    critical: Array.from({ length: 24 }, () => Math.floor(Math.random() * 5 * baseMultiplier)),
    major: Array.from({ length: 24 }, () => Math.floor(Math.random() * 10 * baseMultiplier)),
    minor: Array.from({ length: 24 }, () => Math.floor(Math.random() * 8 * baseMultiplier)),
    warning: Array.from({ length: 24 }, () => Math.floor(Math.random() * 6 * baseMultiplier)),
  };
}

export default function AlarmStatistics() {
  const t = useT();
  const { data: alarmCount } = useAlarmCount();
  const [dateType, setDateType] = useState<'day' | 'month'>('day');
  const [selectedDate, setSelectedDate] = useState<Dayjs>(dayjs());

  const critical = alarmCount?.critical ?? 8;
  const major = alarmCount?.major ?? 15;
  const minor = alarmCount?.minor ?? 12;
  const warning = alarmCount?.warning ?? 8;
  const totalActive = critical + major + minor + warning;

  // X-axis labels based on date type
  const xData = useMemo(() => {
    if (dateType === 'day') {
      return generateHourLabels();
    }
    return generateDayLabels(30);
  }, [dateType]);

  // Series for new alarms (新增告警)
  const newAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(1.2);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity' },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity' },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity' },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity' },
    ];
  }, [t, selectedDate, dateType]);

  // Series for cleared alarms (清除告警)
  const clearedAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(0.8);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity' },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity' },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity' },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity' },
    ];
  }, [t, selectedDate, dateType]);

  // Series for active alarms (活动告警)
  const activeAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(1.5);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity' },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity' },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity' },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity' },
    ];
  }, [t, selectedDate, dateType]);

  // Series for all alarms (所有告警)
  const allAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(2);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity' },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity' },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity' },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity' },
    ];
  }, [t, selectedDate, dateType]);

  // Summary cards
  const summaryCards = [
    {
      title: t('alarm.severity.critical'),
      value: critical,
      color: SEVERITY_COLORS.critical,
      bg: '#fff2f0',
      icon: <AlertOutlined />,
    },
    {
      title: t('alarm.severity.major'),
      value: major,
      color: SEVERITY_COLORS.major,
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

  const handleDateChange = (date: Dayjs | null) => {
    if (date) {
      setSelectedDate(date);
    }
  };

  const handleDateTypeChange = (value: 'day' | 'month') => {
    setDateType(value);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('nav.alarm.statistics')}
        </Title>
        <Space>
          <DatePicker
            value={selectedDate}
            onChange={handleDateChange}
            picker={dateType === 'day' ? 'date' : 'month'}
            allowClear={false}
            disabledDate={(current) => current && current > dayjs().endOf('day')}
          />
          <Radio.Group
            value={dateType}
            onChange={(e) => handleDateTypeChange(e.target.value)}
            optionType="button"
            buttonStyle="solid"
            size="small"
          >
            <Radio.Button value="day">{t('alarm.stats.hour')}</Radio.Button>
            <Radio.Button value="month">{t('alarm.stats.day')}</Radio.Button>
          </Radio.Group>
        </Space>
      </div>

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

      {/* 告警变化趋势 - 新增告警 & 清除告警 */}
      <Card
        title={t('alarm.stats.trend')}
        size="small"
        styles={{ body: { padding: '12px' } }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text strong>{t('alarm.stats.new')}</Text>
            </div>
            <BarChart
              title=""
              xData={xData}
              series={newAlarmSeries}
              height={280}
              stacked
            />
          </Col>
          <Col span={12}>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text strong>{t('alarm.stats.cleared')}</Text>
            </div>
            <BarChart
              title=""
              xData={xData}
              series={clearedAlarmSeries}
              height={280}
              stacked
            />
          </Col>
        </Row>
      </Card>

      {/* 告警存量分布 - 活动告警 & 所有告警 */}
      <Card
        title={t('alarm.stats.distribution')}
        size="small"
        styles={{ body: { padding: '12px' } }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text strong>{t('alarm.stats.active')}</Text>
            </div>
            <BarChart
              title=""
              xData={xData}
              series={activeAlarmSeries}
              height={280}
              stacked
            />
          </Col>
          <Col span={12}>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text strong>{t('alarm.stats.all')}</Text>
            </div>
            <BarChart
              title=""
              xData={xData}
              series={allAlarmSeries}
              height={280}
              stacked
            />
          </Col>
        </Row>
      </Card>
    </div>
  );
}
