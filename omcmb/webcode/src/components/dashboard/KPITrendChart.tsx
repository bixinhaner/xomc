/**
 * KPI 趋势图组件 - 使用 LineChart 实现
 */

import React, { useMemo } from 'react';
import { Card, Button, Space, Typography, Empty } from 'antd';
import LineChart from '@/components/Charts/LineChart';
import { LoadingSpinner } from '@/components/LoadingSpinner';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

const { Text } = Typography;

const COLORS = {
  normal: '#52C41A',
  warning: '#FAAD14',
  critical: '#F5222D',
  today: '#3B82F6',
  yesterday: '#FAAD14',
};

export interface KPITrendChartProps {
  title: string;
  /** @deprecated Reserved for future use. KPI code for data fetching. */
  kpiCode?: string;
  /** @deprecated Reserved for future use. Human-readable KPI label. */
  kpiLabel?: string;
  unit: string;
  value?: number;
  status?: 'normal' | 'warning' | 'critical';
  height?: number;
  loading?: boolean;
  trendData?: { current?: Array<{ time: string; value: number }>; compare?: Array<{ time: string; value: number }> };
  timeRange?: 'yesterday' | 'last_week'; // 当前选择的时间范围
  onTimeRangeChange?: (range: 'yesterday' | 'last_week') => void; // 时间范围变更回调
  className?: string;
}

export function KPITrendChart({
  title,
  unit,
  value = 0,
  status,
  height = 220,
  loading = false,
  trendData,
  timeRange = 'yesterday',
  onTimeRangeChange,
  className,
}: KPITrendChartProps) {
  const t = useT();
  const token = useThemeToken();

  const statusColor = status ? COLORS[status] : COLORS.today;
  const statusText = status ? { normal: t('dashboard.status.normal'), warning: t('dashboard.status.warning'), critical: t('dashboard.status.critical') }[status] : '';

  // 转换数据格式给 LineChart 使用
  const { xData, series } = useMemo(() => {
    const currentData = trendData?.current ?? [];
    const compareData = trendData?.compare ?? [];

    // 根据时间范围格式化时间轴
    const xData = currentData.map((d) => {
      const date = new Date(d.time);
      if (timeRange === 'last_week') {
        // 上周对比显示日期 (MM/DD)
        return `${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`;
      } else {
        // 昨日对比显示时间 (HH:mm)
        return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
      }
    });

    const series = [
      {
        name: t('dashboard.timeRange.today'),
        data: currentData.map((d) => d.value),
        color: COLORS.today,
      },
      {
        name: timeRange === 'last_week' ? t('dashboard.timeRange.lastWeek') : t('dashboard.timeRange.yesterday'),
        data: compareData.map((d) => d.value),
        color: COLORS.yesterday,
      },
    ];

    return { xData, series };
  }, [trendData, t, timeRange]);

  const hasData = xData.length > 0;

  return (
    <Card
      className={className}
      style={{ width: '100%' }}
      styles={{ body: { padding: '8px 0 0' } }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Text style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>{title}</Text>
          <Space size="small">
            <Button
              size="small"
              type={timeRange === 'yesterday' ? 'primary' : 'default'}
              style={{ fontSize: 11 }}
              onClick={() => onTimeRangeChange?.('yesterday')}
            >
              {t('dashboard.timeRange.yesterday')}
            </Button>
            <Button
              size="small"
              type={timeRange === 'last_week' ? 'primary' : 'default'}
              style={{ fontSize: 11 }}
              onClick={() => onTimeRangeChange?.('last_week')}
            >
              {t('dashboard.timeRange.lastWeek')}
            </Button>
          </Space>
        </div>
      }
      extra={status ? <Text style={{ fontSize: 12, color: statusColor }}>{statusText}</Text> : null}
    >
      {loading ? (
        <LoadingSpinner tip={t('common.loading')} style={{ height: height - 40 }} />
      ) : !hasData ? (
        <div style={{ height: height - 40, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('common.noData')} />
        </div>
      ) : (
        <LineChart
          title=""
          xData={xData}
          series={series}
          height={height - 40}
          areaFill
          smooth
          showLegend
          unit={unit}
        />
      )}
    </Card>
  );
}
