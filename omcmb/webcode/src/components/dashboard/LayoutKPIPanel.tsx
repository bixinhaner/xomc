/**
 * LayoutKPIPanel - 全局布局驱动的首页 KPI 折线图卡（issue #213 S2 / KPI-ALL-IND 阶段4）
 *
 * 与旧 KPIPanel 的区别：本卡不再各自发请求、不再按 panelType 读写死配置，
 * 而是接收「布局里这张图的 metrics（指标编号 / 旧 symbolic key）」+「整页一次批量取数得到的对比数据」，
 * 只负责渲染。
 *
 * KPI-ALL-IND 阶圻4：放开全部指标后面板存的是指标编号（K/C 编号），名字/单位优先从 KPI_CATALOG
 * （kpi-config.ts 单一数据源，含 dashboard.kpi.* i18n + i18n unit）取，未录入 catalog 的新
 * 指标回退指标库元数据；都缺时名字回退编号本身。线色用默认色。
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
import { generateDayAxisLabels, generateDayAxisTimestamps } from '@core/utils/format';
import type { KPILayoutPanel } from '@core/types/dashboard';
import type { MultiTrendComparisonData } from '@core/types/dashboard';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';
import { useMetricMetadata, resolveMetricMeta } from './useMetricMetadata';

const { Text } = Typography;

// 折线默认色：今日主色、昨日灰（KPI-ALL-IND 阶段4：线色用默认色，不再按指标取配置色）。
const COLOR_TODAY = '#1677FF';
const COLOR_YESTERDAY = '#999999';

export interface LayoutKPIPanelProps {
  /** 当前制式（用于稳定 key + 按制式拉指标库元数据）。 */
  technology: TechnologyType;
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
  conversion: number,
): { series: Array<{ name: string; data: (number | null)[]; color: string }>; currentValue: number | null } {
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
      { name: todayLabel, data: todayValues, color: COLOR_TODAY },
      { name: yesterdayLabel, data: yesterdayValues, color: COLOR_YESTERDAY },
    ],
    currentValue,
  };
}

export function LayoutKPIPanel({ technology, panel, trendData, isLoading, height = 280 }: LayoutKPIPanelProps) {
  const t = useT();
  const token = useThemeToken();

  // 按制式拉指标库元数据（编号 → 名字/单位）。放开全部指标后名字/单位来源在此。
  const meta = useMetricMetadata(technology);

  // 默认选中第一条指标。
  const [selectedMetric, setSelectedMetric] = useState<string>(panel.metrics[0] ?? '');

  // 布局变化（制式切换 / 配置变化）时把选中指标复位到首条。
  React.useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- Intentional reset when panel metrics change */
    setSelectedMetric(panel.metrics[0] ?? '');
    /* eslint-enable react-hooks/set-state-in-effect */
  }, [panel.metrics]);

  // 当前选中指标的展示元数据（名字/单位/换算）。
  const selectedMeta = useMemo(
    () => resolveMetricMeta(selectedMetric, meta, t),
    [selectedMetric, meta, t],
  );

  const todayLabel = t('dashboard.timeRange.today');
  const yesterdayLabel = t('dashboard.timeRange.yesterday');

  const xData = useMemo(() => generateDayAxisLabels(), []);
  const xDataFull = useMemo(() => generateDayAxisTimestamps(), []);

  const { series } = useMemo(
    () => buildSeries(selectedMetric, trendData, xData, todayLabel, yesterdayLabel, selectedMeta.conversion),
    [selectedMetric, trendData, xData, todayLabel, yesterdayLabel, selectedMeta.conversion],
  );

  // 选中指标在今日/昨日两条线里是否有任一真实数据点（issue #359）。
  // 全网线优先读每小时预聚合表、缺数据时后端已回退原始明细；若两边都拿不到（聚合任务尚未跑过且
  // 原表也无可聚数据），不再画裸空图，而是给出"暂无聚合数据/每小时整点更新"的明确提示，
  // 避免被误判为故障。
  const hasSeriesData = useMemo(
    () => series.some((s) => s.data.some((v) => v !== null && !Number.isNaN(v))),
    [series],
  );

  // 指标下拉：按编号取指标库名字；存量旧别名回退老配置 label；都缺退回编号本身。
  const indicatorOptions = panel.metrics.map((key) => ({
    label: resolveMetricMeta(key, meta, t).name,
    value: key,
  }));

  return (
    <Card
      style={{ width: '100%', height: '100%' }}
      styles={{ body: { padding: '8px 16px 0', display: 'flex', flexDirection: 'column', height: '100%' } }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
          <Text style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>
            {/* 卡片标题只显示配置的图标题（管理员在配置页起的名；默认图的 title 是 i18n key，
                t() 译成「流量/可用性」等用途名）。未设标题时回退展示当前选中指标的名字。
                不再在标题旁拼当前选中指标的「最新值」。 */}
            {panel.title ? t(panel.title) : selectedMeta.name}
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
        ) : !hasSeriesData ? (
          // 有指标但无任何数据点：区分"暂无聚合数据/每小时整点更新"与裸空白（issue #359）。
          <div style={{ height: height - 70, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div style={{ textAlign: 'center' }}>
                  <div style={{ color: token.colorText }}>{t('dashboard.kpiPanel.empty.title')}</div>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('dashboard.kpiPanel.empty.hint')}
                  </Text>
                </div>
              }
            />
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
            unit={selectedMeta.unit}
          />
        )}
      </div>
    </Card>
  );
}
