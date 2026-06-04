/**
 * KPIPanel - 仪表板Panel组件
 *
 * 基于老系统设计，每个Panel包含：
 * - 指标下拉选择
 * - Day/Week视图切换
 * - Today/Yesterday时间对比（多选）
 * - 图表显示
 */

import React, { useState, useMemo, useCallback } from 'react';
import { Card, Select, Button, Space, Spin, Empty, Typography, Checkbox } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { formatKPIValue, generateDayAxisLabels, generateDayAxisTimestamps } from '@core/utils/format';
import type {
  TechnologyType,
  PanelType,
  ViewMode,
  TimeRangeOption,
  KPIConfig,
  PanelConfig,
} from '@/pages/dashboard/kpi-config';
import { getPanelConfig } from '@/pages/dashboard/kpi-config';
import { useKPIPanelData, type KPIDataPoint } from './useKPIPanelData';

const { Text } = Typography;

export interface KPIPanelProps {
  /** 当前制式 */
  technology: TechnologyType;
  /** Panel类型 */
  panelType: PanelType;
  /** 图表高度 */
  height?: number;
  /** 自定义className */
  className?: string;
}

/**
 * KPIPanel组件
 *
 * @example
 * ```tsx
 * <KPIPanel
 *   technology="lte"
 *   panelType="traffic"
 *   height={280}
 * />
 * ```
 */
export function KPIPanel({ technology, panelType, height = 280, className }: KPIPanelProps) {
  const t = useT();
  const token = useThemeToken();


  // 获取Panel配置
  const panelConfig: PanelConfig | undefined = useMemo(
    () => getPanelConfig(technology, panelType),
    [technology, panelType]
  );


  // Panel内部状态
  const [selectedIndicator, setSelectedIndicator] = useState<string>('');
  const [viewMode, setViewMode] = useState<ViewMode>('day');
  const [timeRangeSelection, setTimeRangeSelection] = useState<TimeRangeOption[]>(['today', 'yesterday']);

  // 当panelConfig变化时，重置所有Panel内部状态为默认值
  // 这确保了从LTE切换到NR再切换到GSM时，状态完全重置
  React.useEffect(() => {
    if (panelConfig) {
      /* eslint-disable react-hooks/set-state-in-effect -- Intentional state reset when panelConfig changes */
      setSelectedIndicator(panelConfig.defaultIndicator);
      setViewMode('day');
      setTimeRangeSelection(['today', 'yesterday']);
      /* eslint-enable react-hooks/set-state-in-effect */
    }
  }, [panelConfig]);

  // 获取选中指标的配置（用于获取颜色和单位）
  const selectedKPIConfig: KPIConfig | undefined = useMemo(() => {
    return panelConfig?.indicators.find(ind => ind.key === selectedIndicator);
  }, [panelConfig, selectedIndicator]);

  // 获取KPI数据
  const { data, isLoading } = useKPIPanelData(
    selectedIndicator,
    timeRangeSelection,
    viewMode,
    Boolean(selectedIndicator)
  );

  // 处理指标切换
  const handleIndicatorChange = useCallback((value: string) => {
    setSelectedIndicator(value);
  }, []);

  // 处理视图模式切换
  const handleViewModeChange = useCallback((mode: ViewMode) => {
    setViewMode(mode);
  }, []);

  // 转换图表数据
  const chartData = useMemo(() => {
    if (!data || !selectedKPIConfig) {
      return { xData: [], xDataFull: [], series: [], currentValue: null, trend: null };
    }

    const xData: string[] = [];
    const xDataFull: string[] = []; // 完整时间戳，用于tooltip显示
    const series: Array<{ name: string; data: number[]; color: string }> = [];
    let currentValue: number | null = null;
    let trend: 'up' | 'down' | 'flat' | null = null;

    // 构建完整的X轴
    if (viewMode === 'day') {
      // 对于Day模式，使用整点时间标签（0:00, 01:00, ..., 23:00）
      xData.push(...generateDayAxisLabels());
      xDataFull.push(...generateDayAxisTimestamps());
    } else {
      // Week模式使用数据中的日期
      const allDates = new Set<string>();
      if (data.today && data.today.length > 0) {
        data.today.forEach((point: { time: string }) => {
          const date = new Date(point.time);
          allDates.add(`${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`);
        });
      }
      if (data.yesterday && data.yesterday.length > 0) {
        data.yesterday.forEach((point: { time: string }) => {
          const date = new Date(point.time);
          allDates.add(`${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`);
        });
      }
      // 如果没有任何数据，使用本周的日期
      if (allDates.size === 0) {
        const now = new Date();
        const weekday = now.getDay() || 7;
        for (let i = 0; i < 7; i++) {
          const date = new Date(now);
          date.setDate(now.getDate() - weekday + 1 + i);
          allDates.add(`${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`);
        }
      }
      xData.push(...Array.from(allDates).sort());
      // Week模式下，xDataFull使用完整的日期时间格式
      const now = new Date();
      const weekday = now.getDay() || 7;
      for (let i = 0; i < 7; i++) {
        const date = new Date(now);
        date.setDate(now.getDate() - weekday + 1 + i);
        const year = date.getFullYear();
        const month = (date.getMonth() + 1).toString().padStart(2, '0');
        const day = date.getDate().toString().padStart(2, '0');
        xDataFull.push(`${year}-${month}-${day} 00:00:00`);
      }
    }

    // 处理Today数据 - 始终添加到series（即使数据为空也显示图例）
    {
      const todayValues = new Array<number>(xData.length).fill(null);
      const conversionFactor = selectedKPIConfig.unitConversion ?? 1;

      // 将数据点映射到对应的X轴位置（如果有数据）
      if (data.today && data.today.length > 0) {
        data.today.forEach((point: KPIDataPoint) => {
          const date = new Date(point.time);
          let index: number;
          if (viewMode === 'day') {
            // Day模式：根据HH:00格式查找索引
            const timeStr = `${date.getHours().toString().padStart(2, '0')}:00`;
            index = xData.indexOf(timeStr);
          } else {
            const dateStr = `${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`;
            index = xData.indexOf(dateStr);
          }
          if (index >= 0 && index < todayValues.length) {
            // 应用单位转换
            todayValues[index] = point.value * conversionFactor;
          }
        });
      }

      // 始终添加Today到series（确保图例显示）
      series.push({
        name: t('dashboard.timeRange.today'),
        data: todayValues,
        color: selectedKPIConfig.color,
      });

      // 获取最新值（最后一个非空值）
      const convertedTodayValues = todayValues.filter((v): v is number => v !== null && !Number.isNaN(v));
      currentValue = convertedTodayValues.pop() ?? null;

      // 计算趋势（与昨日对比）
      if (data.yesterday && data.yesterday.length > 0) {
        const yesterdayConversionFactor = selectedKPIConfig.unitConversion ?? 1;
        const yesterdayValues = data.yesterday.map((point: { time: string; value: number }) => point.value * yesterdayConversionFactor);
        const yesterdayValue = yesterdayValues.filter((v): v is number => v !== null && !Number.isNaN(v)).pop() ?? null;

        if (currentValue !== null && yesterdayValue !== null) {
          const diff = currentValue - yesterdayValue;
          const percentDiff = yesterdayValue !== 0 ? (diff / Math.abs(yesterdayValue)) * 100 : 0;
          if (percentDiff > 0.1) trend = 'up';
          else if (percentDiff < -0.1) trend = 'down';
          else trend = 'flat';
        }
      }
    }

    // 处理Yesterday数据 - 始终添加到series（即使数据为空也显示图例）
    {
      const yesterdayValues = new Array<number>(xData.length).fill(null);
      const conversionFactor = selectedKPIConfig.unitConversion ?? 1;

      // 将数据点映射到对应的X轴位置（如果有数据）
      if (data.yesterday && data.yesterday.length > 0) {
        data.yesterday.forEach((point: KPIDataPoint) => {
          const date = new Date(point.time);
          let index: number;
          if (viewMode === 'day') {
            // Day模式：根据HH:00格式查找索引
            const timeStr = `${date.getHours().toString().padStart(2, '0')}:00`;
            index = xData.indexOf(timeStr);
          } else {
            const dateStr = `${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getDate().toString().padStart(2, '0')}`;
            index = xData.indexOf(dateStr);
          }
          if (index >= 0 && index < yesterdayValues.length) {
            // 应用单位转换
            yesterdayValues[index] = point.value * conversionFactor;
          }
        });
      }

      // 始终添加Yesterday到series（确保图例显示）
      series.push({
        name: t('dashboard.timeRange.yesterday'),
        data: yesterdayValues,
        color: '#999999', // Yesterday使用灰色
      });
    }

    return { xData, xDataFull, series, currentValue, trend };
  }, [data, selectedKPIConfig, viewMode, t]);

  // 如果没有配置，显示错误提示
  if (!panelConfig) {
    return (
      <Card style={{ width: '100%', height: '100%' }}>
        <Empty
          description={t('dashboard.panelConfigNotFound')}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    );
  }

  const hasData = chartData.xData.length > 0;
  const hasToday = timeRangeSelection.includes('today');
  const hasYesterday = timeRangeSelection.includes('yesterday');
  const hasSelection = hasToday || hasYesterday;

  // 指标下拉选项
  const indicatorOptions = panelConfig.indicators.map(ind => ({
    label: t(ind.label),
    value: ind.key,
  }));

  return (
    <Card
      className={className}
      style={{ width: '100%', height: '100%' }}
      styles={{ body: { padding: '8px 16px 0', display: 'flex', flexDirection: 'column', height: '100%' } }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
          {/* 左侧：指标名称 + 当前值 */}
          <Space size={8}>
            <Text style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>
              {selectedKPIConfig ? t(selectedKPIConfig.label) : t(panelConfig.title)}
            </Text>
            {chartData.currentValue !== null && (
              <>
                <Text style={{
                  fontSize: 14,
                  fontWeight: 500,
                  color: token.colorText
                }}>
                  {formatKPIValue(chartData.currentValue, selectedKPIConfig?.unit || '', t)}
                </Text>
                {chartData.trend && chartData.trend !== 'flat' && (
                  <Text style={{
                    fontSize: 14,
                    fontWeight: 600,
                    color: chartData.trend === 'up' ? '#52C41A' : '#F5222D'
                  }}>
                    {chartData.trend === 'up' ? '↑' : '↓'}
                  </Text>
                )}
              </>
            )}
          </Space>

          {/* 右侧：控制区 - 指标下拉 + Day/Week */}
          <Space size="small" wrap={false}>
            <Select
              value={selectedIndicator}
              onChange={handleIndicatorChange}
              options={indicatorOptions}
              style={{ minWidth: 120 }}
              size="small"
            />
            <Space.Compact size="small">
              <Button
                type={viewMode === 'day' ? 'primary' : 'default'}
                onClick={() => handleViewModeChange('day')}
              >
                {t('dashboard.viewMode.day')}
              </Button>
              <Button
                type={viewMode === 'week' ? 'primary' : 'default'}
                onClick={() => handleViewModeChange('week')}
              >
                {t('dashboard.viewMode.week')}
              </Button>
            </Space.Compact>
          </Space>
        </div>
      }
    >
      {/* 图表区域 */}
      <div style={{ flex: 1, minHeight: height - 70 }}>
        {isLoading ? (
          <Spin
            indicator={<LoadingOutlined spin />}
            spinning={isLoading}
            style={{ height: height - 70, width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          >
            <div style={{ height: height - 70 }} />
          </Spin>
        ) : !hasSelection ? (
          <div style={{ height: height - 70, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Empty description={t('dashboard.selectTimeRange')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          </div>
        ) : !hasData ? (
          <div style={{ height: height - 70, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          </div>
        ) : (
          <LineChart
            key={`${technology}-${panelType}-${selectedIndicator}-${chartData.xData.length}`}
            title=""
            xData={chartData.xData}
            xDataFull={chartData.xDataFull}
            series={chartData.series}
            height={height - 70}
            areaFill={hasToday && !hasYesterday} // 只有Today时填充区域
            smooth
            showLegend={true} // 显示图表内的Today/Yesterday图例
            unit={selectedKPIConfig?.unit ? t(selectedKPIConfig.unit) : ''}
          />
        )}
      </div>
    </Card>
  );
}
