/**
 * LayoutKPIPanel - 全局布局驱动的首页 KPI 折线图卡（issue #213 S2 / KPI-ALL-IND 阶段4）
 *
 * 与旧 KPIPanel 的区别：本卡不再各自发请求、不再按 panelType 读写死配置，
 * 而是接收「布局里这张图的 metrics（指标编号 / 旧 symbolic key）」+「整页一次批量取数得到的对比数据」，
 * 只负责渲染。
 *
 * KPI-ALL-IND 阶圻4：放开全部指标后面板存的是指标编号（K/C 编号），名字/单位优先从 KPI_CATALOG
 * （kpi-config.ts 单一数据源，含 dashboard.kpi.* i18n + i18n unit）取，未录入 catalog 的新
 * 指标回退指标库元数据；都缺时名字回退编号本身。
 *
 * Issue B（多指标对比）：下拉支持多选，默认仅选 panel.metrics 第一项。
 *  - 选 1 个 → 同时画今日 + 昨日（昨日虚线），与旧单选行为对齐（默认即此态）。
 *  - 选 ≥ 2 个 → 仅画今日 N 条，按调色板循环上色，避免 2N 条线视觉爆炸（决策 D2）。
 *  - 选 0 个 → 显示“请选择指标”占位，不渲图（与 Grafana/DataDog 同类产品一致，不强制最少 1 个）。
 *  - tag 不带单个 ×：反选走下拉点 ✓，避免选 N 个时 N 个 × 的视觉杂象；保留 allowClear 一键清空。
 *  - 单位混选时，Y 轴单位留空（决策 D3 起步版，未来可升级双 Y 轴）。
 *
 * 容错：某指标在批量取数结果里缺失（指标库下线 / 无数据）时，那条线跳过、不让整图崩。
 * 首页只读不可拖。
 */

import { useCallback, useMemo, useState, useEffect, useRef } from 'react';
import type { ComponentProps, ReactNode } from 'react';
import { Card, Select, Segmented, Spin, Empty, Typography, Tooltip, Tag } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import LineChart from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';
import { useIntl } from 'react-intl';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { DashboardKPIGranularity, KPILayoutPanel } from '@core/types/dashboard';
import type {
  DashboardPeriodProgress,
  DashboardProgressState,
} from '@core/types/dashboard';
import type { MultiTrendComparisonData } from '@core/types/dashboard';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';
import { useMetricMetadata, resolveMetricMeta } from './useMetricMetadata';
import { buildSeries, shouldShowKPIChartLegend } from './LayoutKPIPanel.helpers';

const { Text } = Typography;

export interface LayoutKPIPanelProps {
  /** 当前制式（用于稳定 key + 按制式拉指标库元数据）。 */
  technology: TechnologyType;
  /** 这张图的布局（标题 + 指标列表）。 */
  panel: KPILayoutPanel;
  /** 整页一次批量取数得到的多指标今日/昨日对比数据。 */
  trendData: MultiTrendComparisonData | undefined;
  /** 批量取数是否加载中。 */
  isLoading: boolean;
  /** 当前整页聚合粒度。 */
  granularity: DashboardKPIGranularity;
  /** 切换整页聚合粒度。 */
  onGranularityChange: (granularity: DashboardKPIGranularity) => void;
  /** 后端聚合桶对应的固定横轴键。 */
  bucketKeys?: string[];
  /** 当前日/周自然周期覆盖率。 */
  periodProgress?: DashboardPeriodProgress[];
  /** 当前周期状态源是否可用。 */
  progressState?: DashboardProgressState;
  /** 图表高度。 */
  height?: number;
}

function isLocalMonday(datePart: string): boolean {
  const parts = datePart.split('-').map((value) => Number.parseInt(value, 10));
  if (parts.length !== 3 || parts.some((value) => Number.isNaN(value))) {
    return false;
  }
  const [year, month, day] = parts;
  return new Date(Date.UTC(year, month - 1, day)).getUTCDay() === 1;
}

function isNaturalWeeklyProgress(item: DashboardPeriodProgress): boolean {
  const start = item.windowStart.match(/^(\d{4}-\d{2}-\d{2})T00:00:00/);
  const end = item.windowEnd.match(/^(\d{4}-\d{2}-\d{2})T00:00:00/);
  return Boolean(start && end && isLocalMonday(start[1]) && isLocalMonday(end[1]));
}

function selectActivePeriodProgress(
  periodProgress: DashboardPeriodProgress[],
  granularity: DashboardKPIGranularity,
): DashboardPeriodProgress | undefined {
  const candidates = periodProgress.filter((item) => item.granularity === granularity);
  const prioritized = granularity === 'weekly'
    ? candidates.filter(isNaturalWeeklyProgress)
    : candidates;
  return (prioritized.length > 0 ? prioritized : candidates)
    .sort((left, right) => {
      const startOrder = right.windowStart.localeCompare(left.windowStart);
      if (startOrder !== 0) return startOrder;
      return right.expectedSlots - left.expectedSlots;
    })[0];
}

export function LayoutKPIPanel({
  technology,
  panel,
  trendData,
  isLoading,
  granularity,
  onGranularityChange,
  bucketKeys = [],
  periodProgress = [],
  progressState = 'not_applicable',
  height = 280,
}: LayoutKPIPanelProps) {
  const t = useT();
  const intl = useIntl();
  const token = useThemeToken();

  // 按制式拉指标库元数据（编号 → 名字/单位）。放开全部指标后名字/单位来源在此。
  const meta = useMetricMetadata(technology);

  // 默认仅选 panel.metrics 第一项（决策 D1 修订：默认就是单指标 today+yesterday 对比视图，
  // 用户主动添加才进入多选叠加态，符合大多数日常巡检场景）。
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>(() =>
    panel.metrics.length > 0 ? [panel.metrics[0]] : [],
  );

  const hasDataRef = useRef(panel.metrics.length > 0);

  // 应对异步 Layout 数据的更新：从内置兜底配置（例如 K编号）切换到接口返回的真实配置（可能带别名或删减了指标）
  useEffect(() => {
    // 场景一：初始兜底为空，现在接口首次返回有效指标。直接选第一项；
    // ref 修改放在 setState 之外，避免 StrictMode 下 updater 双调导致副作用重复。
    if (panel.metrics.length > 0 && !hasDataRef.current) {
      hasDataRef.current = true;
      setSelectedMetrics([panel.metrics[0]]);
      return;
    }

    // 场景二：接口返回的指标列表跟当前选中不一致（别名替换 / 删减）。
    // 纯函数过滤，updater 不带副作用。
    setSelectedMetrics((prev) => {
      const valid = prev.filter((m) => panel.metrics.includes(m));
      if (valid.length === prev.length) return prev;
      if (valid.length > 0) return valid;
      // 全军覆没且原本有选中：自动回退到新的第一项，避免 0 选空图态。
      if (prev.length > 0 && panel.metrics.length > 0) return [panel.metrics[0]];
      return prev;
    });
  }, [panel.metrics]);

  // 单 metric 解析器（在 buildSeries 与 unit 推断里共用）。
  const resolveOne = useCallback(
    (key: string) => resolveMetricMeta(key, meta, t),
    [meta, t],
  );

  // Y 轴单位：仅当所有选中指标单位相同时才显示（决策 D3 起步版，避免不同量纲共用一个单位标签误导）。
  const sharedUnit = useMemo(() => {
    if (selectedMetrics.length === 0) return '';
    const units = new Set(selectedMetrics.map((k) => resolveOne(k).unit));
    return units.size === 1 ? [...units][0] : '';
  }, [selectedMetrics, resolveOne]);

  // 卡片标题 fallback：未配 title 时，选 1 个用该指标名，选多个用 panel.metrics 第一个的名。
  const titleFallbackKey = selectedMetrics[0] ?? panel.metrics[0] ?? '';
  const titleFallback = titleFallbackKey ? resolveOne(titleFallbackKey).name : '';

  const { series, weekXData, weekXDataFull, weekXDataEndFull } = useMemo(
    () => buildSeries(
      selectedMetrics,
      trendData,
      [],
      '',
      '',
      resolveOne,
      undefined,
      granularity,
      bucketKeys,
    ),
    [selectedMetrics, trendData, resolveOne, granularity, bucketKeys],
  );

  const chartXData = weekXData ?? [];
  const chartXDataFull = weekXDataFull ?? [];
  const chartXDataEndFull = weekXDataEndFull ?? [];
  const activeProgress = useMemo(
    () => selectActivePeriodProgress(periodProgress, granularity),
    [granularity, periodProgress],
  );
  const progressLabel = useMemo(() => {
    if (granularity === 'hourly' || progressState === 'not_applicable') return undefined;
    if (progressState === 'unavailable') {
      return {
        color: 'warning',
        text: t('dashboard.kpiPanel.progress.unavailable'),
        detail: t('dashboard.kpiPanel.progress.unavailableDetail'),
      };
    }
    if (!activeProgress || activeProgress.expectedSlots <= 0) return undefined;
    const coverage = Math.min(
      100,
      (activeProgress.receivedSlots / activeProgress.expectedSlots) * 100,
    ).toFixed(1);
    const complete = t('dashboard.kpiPanel.progress.complete');
    const incomplete = t('dashboard.kpiPanel.progress.incomplete');
    return {
      color: 'processing',
      text: t('dashboard.kpiPanel.progress.partial', {
        received: activeProgress.receivedSlots,
        expected: activeProgress.expectedSlots,
        coverage,
      }),
      detail: (
        <div>
          <div>{t('dashboard.kpiPanel.progress.windowStart', {
            time: activeProgress.windowStart,
          })}</div>
          <div>{t('dashboard.kpiPanel.progress.windowEnd', {
            time: activeProgress.windowEnd,
          })}</div>
          <div>{t('dashboard.kpiPanel.progress.detail', {
            version: activeProgress.taskVersionId,
            from: activeProgress.versionEffectiveFrom,
            to: activeProgress.versionEffectiveTo
              ?? t('dashboard.kpiPanel.progress.effectiveOngoing'),
            revision: activeProgress.revision,
          })}</div>
          <div>{t('dashboard.kpiPanel.progress.versionSlice', {
            received: activeProgress.receivedSlots,
            expected: activeProgress.versionExpectedSlots,
            status: activeProgress.versionSliceComplete ? complete : incomplete,
          })}</div>
          <div>{t('dashboard.kpiPanel.progress.naturalPeriod', {
            received: activeProgress.receivedSlots,
            expected: activeProgress.expectedSlots,
            status: activeProgress.periodComplete ? complete : incomplete,
          })}</div>
        </div>
      ),
    };
  }, [activeProgress, granularity, progressState, t]);

  // 至少一条 series 有真实数据点？无任何点时给"暂无聚合数据"提示（issue #359 保留）。
  const hasSeriesData = useMemo(
    () => series.some((s) => s.data.some((v) => v !== null && !Number.isNaN(v))),
    [series],
  );

  // 指标下拉：按编号取指标库名字；存量旧别名回退老配置 label；都缺退回编号本身。
  const indicatorOptions = useMemo(
    () => panel.metrics.map((key) => ({ label: resolveOne(key).name, value: key })),
    [panel.metrics, resolveOne],
  );

  // 自定义 tag：去掉逐个 ×，与 Grafana / Kibana 多选 trigger 一致，避免选 N 个时 N 个 × 的视觉杂象。
  // 反选走“打开下拉 → 点选中项 ✓”；底部还有 allowClear 提供“一键清空”。
  const tagRender = useCallback<NonNullable<ComponentProps<typeof Select>['tagRender']>>(
    (props) => <Tag style={{ marginInlineEnd: 4, marginInlineStart: 0 }}>{props.label}</Tag>,
    [],
  );

  // 超出可见 tag 数量时的占位渲染：用 Tooltip 暴露完整名单，避免 "+2 ..." 用户不知道选了啥。
  const renderMaxTagPlaceholder = useCallback(
    (omittedValues: { label?: ReactNode; value?: string | number }[]) => {
      const names = omittedValues.map((o) =>
        typeof o.label === 'string' ? o.label : resolveOne(String(o.value ?? '')).name,
      );
      return (
        <Tooltip title={names.join('、')} placement="top">
          <span>{t('dashboard.kpi.moreMetricsCount', { count: omittedValues.length })}</span>
        </Tooltip>
      );
    },
    [resolveOne, t],
  );

  return (
    <Card
      style={{ width: '100%', height: '100%' }}
      styles={{ body: { padding: '8px 16px 0', display: 'flex', flexDirection: 'column', height: '100%' } }}
      title={
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            minHeight: 32,
            flexWrap: 'wrap',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              flexWrap: 'wrap',
              minWidth: 0,
              flex: '1 1 auto',
            }}
          >
            <Text
              style={{ fontSize: 15, fontWeight: 600, color: token.colorText, flexShrink: 0 }}
            >
              {panel.title
                ? panel.title in intl.messages
                  ? t(panel.title)
                  : panel.title
                : titleFallback}
            </Text>
            <Segmented
              size="small"
              value={granularity}
              onChange={(value) => onGranularityChange(value as DashboardKPIGranularity)}
              options={[
                { label: t('dashboard.viewMode.hour'), value: 'hourly' },
                { label: t('dashboard.viewMode.day'), value: 'daily' },
                { label: t('dashboard.viewMode.week'), value: 'weekly' },
              ]}
            />
            {progressLabel && (
              <Tooltip title={progressLabel.detail}>
                <Tag color={progressLabel.color}>{progressLabel.text}</Tag>
              </Tooltip>
            )}
          </div>
          {/*
           * 宽度分档 + responsive tag：
           *  - 只有 1 个可选项（如可用性/移动性）→ 160px，避免“只装一个 tag 却拉很长”的空荡感。
           *  - ≥2 个可选项 → 260px，够容 1–2 个 tag + “+N 项”折叠占位。
           *  - allowClear 保留：用户主动清空是明确意图，不拦截，0 选时画面给友好占位。
           *  - tagRender：去掉逐 tag 的 ×，反选靠下拉点 ✓，与 Grafana / Kibana 一致的心智模型。
           */}
          <Select
            mode="multiple"
            value={selectedMetrics}
            onChange={setSelectedMetrics}
            options={indicatorOptions}
            tagRender={tagRender}
            maxTagCount="responsive"
            maxTagPlaceholder={renderMaxTagPlaceholder}
            allowClear
            placeholder={t('dashboard.kpi.selectMetricsPlaceholder')}
            style={{ width: panel.metrics.length <= 1 ? 180 : 240, flexShrink: 0, marginLeft: 'auto' }}
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
        ) : selectedMetrics.length === 0 ? (
          // 用户主动清空选择：与 Grafana/DataDog 一致允许 0 选态，给友好提示而非拦截。
          <div style={{ height: height - 70, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Empty
              description={t('dashboard.kpi.noMetricSelected')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          </div>
        ) : !hasSeriesData ? (
          // 有选择但无任何数据点：区分"暂无聚合数据/每小时整点更新"与裸空白（issue #359）。
          <div style={{ height: height - 70, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div style={{ textAlign: 'center' }}>
                  <div style={{ color: token.colorText }}>{t('dashboard.kpiPanel.empty.title')}</div>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('dashboard.kpiPanel.empty.rollupHint')}
                  </Text>
                </div>
              }
            />
          </div>
        ) : (
          <LineChart
            // 粒度变化时横轴格式随之变化，强制 remount 清空旧 ECharts 实例。
            key={`${technology}-${panel.title}-${granularity}`}
            title=""
            xData={chartXData}
            xDataFull={chartXDataFull}
            xDataEndFull={chartXDataEndFull}
            formatTooltipStart={(time) => intl.formatMessage({
              id: 'dashboard.kpiPanel.progress.windowStart',
            }, { time })}
            formatTooltipEnd={(time) => intl.formatMessage({
              id: 'dashboard.kpiPanel.progress.windowEnd',
            }, { time })}
            series={series}
            height={height - 70}
            smooth
            showLegend={shouldShowKPIChartLegend(granularity, selectedMetrics.length)}
            unit={sharedUnit}
          />
        )}
      </div>
    </Card>
  );
}
