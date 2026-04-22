import React, { useMemo, useState } from 'react';
import { Card, Col, Row, Typography, DatePicker, Radio, Space } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import BarChart from '@/components/Charts/BarChart';
import { useT } from '@/hooks/useT';

const { Title, _Text } = Typography;

// 告警级别颜色 - 专业配色方案
export const SEVERITY_COLORS = {
  critical: '#E53935', // 紧急 - 深红色
  major: '#FB8C00',    // 重要 - 明亮橙色
  minor: '#FDD835',    // 次要 - 金黄色
  warning: '#42A5F5',  // 警告 - 亮蓝色
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
  const [dateType, setDateType] = useState<'day' | 'month'>('day');
  const [selectedDate, setSelectedDate] = useState<Dayjs>(dayjs());

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
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity', borderRadius: 4 },
    ];
  }, [t, selectedDate, dateType]);

  // Series for cleared alarms (清除告警)
  const clearedAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(0.8);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity', borderRadius: 4 },
    ];
  }, [t, selectedDate, dateType]);

  // Series for active alarms (活动告警)
  const activeAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(1.5);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity', borderRadius: 4 },
    ];
  }, [t, selectedDate, dateType]);

  // Series for all alarms (所有告警)
  const allAlarmSeries = useMemo(() => {
    const data = generateMockStackedData(2);
    return [
      { name: t('alarm.severity.critical'), data: data.critical, color: SEVERITY_COLORS.critical, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.major'), data: data.major, color: SEVERITY_COLORS.major, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.minor'), data: data.minor, color: SEVERITY_COLORS.minor, stack: 'severity', borderRadius: 0 },
      { name: t('alarm.severity.warning'), data: data.warning, color: SEVERITY_COLORS.warning, stack: 'severity', borderRadius: 4 },
    ];
  }, [t, selectedDate, dateType]);

  const handleDateChange = (date: Dayjs | null) => {
    if (date) {
      setSelectedDate(date);
    }
  };

  const handleDateTypeChange = (value: 'day' | 'month') => {
    setDateType(value);
  };

  // 图表卡片配置
  const chartCards = [
    { key: 'new', title: t('alarm.stats.new'), series: newAlarmSeries },
    { key: 'cleared', title: t('alarm.stats.cleared'), series: clearedAlarmSeries },
    { key: 'active', title: t('alarm.stats.active'), series: activeAlarmSeries },
    { key: 'all', title: t('alarm.stats.all'), series: allAlarmSeries },
  ];

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* 顶部标题栏 */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '0 0 16px 0',
        flexShrink: 0,
      }}>
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

      {/* 四个图表卡片 - 填满剩余空间 */}
      <div style={{ flex: 1, minHeight: 0 }}>
        <Row gutter={[16, 16]} style={{ height: '100%' }}>
          {chartCards.map(({ key, title, series }) => (
            <Col key={key} xs={24} sm={12} style={{ height: '100%', display: 'flex' }}>
              <Card
                title={title}
                size="small"
                styles={{
                  body: { padding: '12px', flex: 1, display: 'flex', flexDirection: 'column' },
                }}
                style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
              >
                <div style={{ flex: 1, minHeight: 0 }}>
                  <BarChart
                    title=""
                    xData={xData}
                    series={series}
                    height="100%"
                    stacked
                    barWidth={10}
                    borderRadius={0}
                  />
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      </div>
    </div>
  );
}
