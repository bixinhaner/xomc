/**
 * 多指标 KPI 趋势图组件 - 使用 LineChart 实现
 */

import React, { useMemo } from 'react';
import { Card, Button, Space, Typography, Spin, Empty } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import LineChart from '@/components/Charts/LineChart';
import type { ThresholdLine } from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { buildMultiKpiTrendModel } from '@core/utils/buildMultiKpiTrendModel';

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
  trendDataMap: Record<string, TrendDataValue>;
  height?: number;
  loading?: boolean;
  timeRange?: 'yesterday' | 'last_week'; // 当前选择的时间范围
  onTimeRangeChange?: (range: 'yesterday' | 'last_week') => void; // 时间范围变更回调
  className?: string;
  /** 阈值线配置（如PRB利用率告警线） */
  thresholdLines?: ThresholdLine[];
}

// 数据值类型定义
interface TrendDataValue {
  current?: Array<{ time: string; value: number }>;
  data?: { current?: Array<{ time: string; value: number }> };
  compare?: Array<{ time: string; value: number }>;
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
  thresholdLines,
}: MultiKPITrendChartProps) {
  const t = useT();
  const token = useThemeToken();

  // 转换数据格式给 LineChart 使用（#200：多 KPI 时间轴取并集 + 按时间值对齐，替换旧
  // "只取第一个 KPI 时间点 + 按索引 zip" 的错位逻辑）
  const { xData, xDataFull, series } = useMemo(() => {
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

    // 装配多 KPI 并集时间轴模型（name 先做 i18n 翻译后传入纯函数）
    return buildMultiKpiTrendModel(
      kpis.map((kpi) => ({
        name: t(kpi.label),
        points: getCurrentData(kpi.key),
        color: kpi.color,
      })),
      timeRange,
    );
  }, [kpis, trendDataMap, timeRange, t]);

  const hasData = xData.length > 0;
  const hasKPIs = kpis.length > 0;

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
        <Spin
          indicator={<LoadingOutlined spin />}
          spinning={loading}
          style={{ height: height - 40, width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
        >
          <div style={{ height: height - 40 }} />
        </Spin>
      ) : !hasKPIs || !hasData ? (
        <div style={{ height: height - 40, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('common.noData')} />
        </div>
      ) : (
        <LineChart
          title=""
          xData={xData}
          xDataFull={xDataFull}
          series={series}
          height={height - 40}
          areaFill
          smooth
          showLegend
          connectNulls
          unit={kpis[0]?.unit}
          thresholdLines={thresholdLines}
        />
      )}
    </Card>
  );
}
