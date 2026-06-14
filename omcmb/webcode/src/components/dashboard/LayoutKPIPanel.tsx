/**
 * LayoutKPIPanel - 全局布局驱动的首页 KPI 折线图卡（issue #213 S2）
 *
 * 与旧 KPIPanel 的区别：本卡不再各自发请求、不再按 panelType 读写死配置，
 * 而是接收「布局里这张图的 metrics（symbolic key）」+「整页一次批量取数得到的对比数据」，
 * 只负责渲染。指标 label / 线色 / 单位按 key 反查 kpi-config。
 *
 * 容错：某指标在批量取数结果里缺失（指标库下线 / 无数据）时，那条线跳过、不让整图崩。
 * 首页只读不可拖。
 */

import React, { useMemo, useState } from 'react';
import { Card, Select, Spin, Empty, Typography } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';
import { formatKPIValue, generateDayAxisLabels, generateDayAxisTimestamps } from '@core/utils/format';
import type { KPILayoutPanel } from '@core/types/dashboard';
import type { MultiTrendComparisonData } from '@core/types/dashboard';
import { getKPIConfigByKey } from '@/pages/dashboard/kpi-config';

const { Text } = Typography;

export interface LayoutKPIPanelProps {
  /** 当前制式（仅用于稳定 key，渲染不依赖）。 */
  technology: string;
  /** 这张图的布局（标题 + 指标列表）。 */
  panel: KPILayoutPanel;
  /** 整页一次批量取数得到的多指标今日/昨日对比数据。 */
  trendData: MultiTrendComparisonData | undefined;
  /** 批量取数是否加载中。 */
  isLoading: boolean;
  /** 图表高度。 */
  height?: number;
}

/**
 * 把一个指标的 current/compare 序列折算成 Day 模式（24 整点）两条线。
 */
function buildSeries(
  metricKey: string,
  trendData: MultiTrendComparisonData | undefined,
  xData: string[],
  todayLabel: string,
  yesterdayLabel: string,
): { series: Array<{ name: string; data: (number | null)[]; color: string }>; currentValue: number | null } {
  const config = getKPIConfigByKey(metricKey);
  const color = config?.color ?? '#1677FF';
  const conversion = config?.unitConversion ?? 1;
  const comparison = trendData?.[metricKey];

  const todayValues = new Array<number | null>(xData.length).fill(null);
  const yesterdayValues = new Array<number | null>(xData.length).fill(null);

  if (comparison) {
    comparison.current.forEach((point) => {
      const date = new Date(point.time);
      const timeStr = `${date.getHours().toString().padStart(2, '0')}:00`;
      const index = xData.indexOf(timeStr);
      if (index >= 0) todayValues[index] = point.value * conversion;
    });
    comparison.compare.forEach((point) => {
      const date = new Date(point.time);
      const timeStr = `${date.getHours().toString().padStart(2, '0')}:00`;
      const index = xData.indexOf(timeStr);
      if (index >= 0) yesterdayValues[index] = point.value * conversion;
    });
  }

  const nonEmptyToday = todayValues.filter((v): v is number => v !== null && !Number.isNaN(v));
  const currentValue = nonEmptyToday.length > 0 ? nonEmptyToday[nonEmptyToday.length - 1] : null;

  return {
    series: [
      { name: todayLabel, data: todayValues, color },
      { name: yesterdayLabel, data: yesterdayValues, color: '#999999' },
    ],
    currentValue,
  };
}

export function LayoutKPIPanel({ technology, panel, trendData, isLoading, height = 280 }: LayoutKPIPanelProps) {
  const t = useT();
  const token = useThemeToken();

  // 默认选中第一条指标。
  const [selectedMetric, setSelectedMetric] = useState<string>(panel.metrics[0] ?? '');

  // 布局变化（制式切换 / 配置变化）时把选中指标复位到首条。
  React.useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- Intentional reset when panel metrics change */
    setSelectedMetric(panel.metrics[0] ?? '');
    /* eslint-enable react-hooks/set-state-in-effect */
  }, [panel.metrics]);

  const selectedConfig = useMemo(() => getKPIConfigByKey(selectedMetric), [selectedMetric]);

  const todayLabel = t('dashboard.timeRange.today');
  const yesterdayLabel = t('dashboard.timeRange.yesterday');

  const xData = useMemo(() => generateDayAxisLabels(), []);
  const xDataFull = useMemo(() => generateDayAxisTimestamps(), []);

  const { series, currentValue } = useMemo(
    () => buildSeries(selectedMetric, trendData, xData, todayLabel, yesterdayLabel),
    [selectedMetric, trendData, xData, todayLabel, yesterdayLabel],
  );

  // 指标下拉：按 key 反查标签；缺登记时退回展示 key 本身。
  const indicatorOptions = panel.metrics.map((key) => {
    const cfg = getKPIConfigByKey(key);
    return { label: cfg ? t(cfg.label) : key, value: key };
  });

  const unitLabel = selectedConfig?.unit ? t(selectedConfig.unit) : '';

  return (
    <Card
      style={{ width: '100%', height: '100%' }}
      styles={{ body: { padding: '8px 16px 0', display: 'flex', flexDirection: 'column', height: '100%' } }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
          <Text style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>
            {/* 卡片标题显示配置的图标题（管理员在配置页起的名；默认图的 title 是 i18n key，
                t() 译成「流量/可用性」等用途名）。未设标题时回退展示当前选中指标的 label。 */}
            {panel.title ? t(panel.title) : (selectedConfig ? t(selectedConfig.label) : '')}
            {currentValue !== null && (
              <Text style={{ fontSize: 14, fontWeight: 500, marginLeft: 8 }}>
                {formatKPIValue(currentValue, selectedConfig?.unit || '', t)}
              </Text>
            )}
          </Text>
          <Select
            value={selectedMetric}
            onChange={setSelectedMetric}
            options={indicatorOptions}
            style={{ minWidth: 120 }}
            size="small"
          />
        </div>
      }
    >
      <div style={{ flex: 1, minHeight: height - 70 }}>
        {isLoading ? (
          <Spin
            indicator={<LoadingOutlined spin />}
            spinning={isLoading}
            style={{ height: height - 70, width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          >
            <div style={{ height: height - 70 }} />
          </Spin>
        ) : panel.metrics.length === 0 ? (
          <div style={{ height: height - 70, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          </div>
        ) : (
          <LineChart
            key={`${technology}-${panel.title}-${selectedMetric}`}
            title=""
            xData={xData}
            xDataFull={xDataFull}
            series={series}
            height={height - 70}
            smooth
            showLegend={true}
            unit={unitLabel}
          />
        )}
      </div>
    </Card>
  );
}
