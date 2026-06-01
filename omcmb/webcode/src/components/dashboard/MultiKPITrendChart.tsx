/**
 * 多指标 KPI 趋势图组件 - 使用 LineChart 实现
 */

import React, { useMemo } from 'react';
import { Card, Button, Space, Typography, Spin, Empty } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

const { Text } = Typography;

export interface MultiKPIConfig {
  key: string;
  label: string;
  color: string;
  unit: string;
}

export interface MultiKPITrendChartProps {
  title: string;
  kpis: MultiKPIConfig[];
  trendDataMap: Record<string, any>;
  height?: number;
  loading?: boolean;
  timeRange?: 'yesterday' | 'last_week'; // 当前选择的时间范围
  onTimeRangeChange?: (range: 'yesterday' | 'last_week') => void; // 时间范围变更回调
  className?: string;
}

export function MultiKPITrendChart({
  title,
  kpis,
  trendDataMap,
  height = 260,
  loading = false,
  timeRange = 'yesterday',
  onTimeRangeChange,
  className,
}: MultiKPITrendChartProps) {
  const t = useT();
  const token = useThemeToken();

  // 转换数据格式给 LineChart 使用
  const { xData, series } = useMemo(() => {
    // 辅助函数：从可能的数据结构中提取 current 数组
    // 兼容两种数据格式：
    // 1. React Query 结果格式: { data: { current: [...], compare: [...] } }
    // 2. 直接数据格式: { current: [...], compare: [...] }
    const getCurrentData = (kpiKey: string) => {
      const trendData = trendDataMap[kpiKey];
      if (!trendData) return [];
      // 优先使用 .data.current（React Query 格式）
      if (trendData.data?.current) return trendData.data.current;
      // 否则直接使用 .current（直接数据格式）
      if (trendData.current) return trendData.current;
      return [];
    };

    // 找到第一个有效的 KPI 数据来提取时间轴
    const firstValidKPI = kpis.find(kpi => getCurrentData(kpi.key).length > 0);

    if (!firstValidKPI) {
      return { xData: [], series: [] };
    }

    // 提取时间轴数据 - 根据时间范围格式化
    const currentData = getCurrentData(firstValidKPI.key);
    const xData = currentData.map((d: any) => {
      const date = new Date(d.time);
      if (timeRange === 'last_week') {
        // 上周对比显示日期 (MM/DD)
        return `${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`;
      } else {
        // 昨日对比显示时间 (HH:mm)
        return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
      }
    });

    // 转换每个 KPI 的数据为 LineChart 格式
    const series = kpis.map(kpi => {
      const kpiCurrentData = getCurrentData(kpi.key);
      const data = kpiCurrentData.map((d: any) => d.value);
      return {
        name: kpi.label,
        data,
        color: kpi.color,
      };
    });

    return { xData, series };
  }, [kpis, trendDataMap, timeRange]);

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
    >
      {loading ? (
        <div style={{ height: height - 40, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin indicator={<LoadingOutlined spin />} tip={t('common.loading')} />
        </div>
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
          unit={kpis[0]?.unit}
        />
      )}
    </Card>
  );
}
