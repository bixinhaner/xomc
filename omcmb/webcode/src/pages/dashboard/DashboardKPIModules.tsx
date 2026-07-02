/**
 * Dashboard KPI Panel区域 - issue #213 S2：读全局布局（带回退）驱动渲染
 *
 * 流程：按当前制式读 S1 全局布局接口 → 读不到/为空/出错回退内置默认 →
 * 汇总所有可见图要画的指标做一次批量取数 → 按网格位置（行）渲染折线图。
 * 首页只读不可拖。
 */

import { useEffect, useMemo, useState } from 'react';
import type { TechnologyType } from './kpi-config';
import { useKPILayout } from '@core/hooks/api/useDashboard';
import { useMultiKPITrendComparison } from '@core/hooks/api/useDashboard';
import { LayoutKPIPanel } from '@/components/dashboard/LayoutKPIPanel';
import { resolveLayout, collectMetrics } from './layoutMapping';
import type { KPILayoutPanel } from '@core/types/dashboard';

type CompareWindow = 'yesterday' | 'last_week';

interface DashboardKPIModulesProps {
  /** 当前制式 */
  technology: TechnologyType;
  /** 是否启用滚动显示动画 */
  enableScrollReveal?: boolean;
  /** 动画起始延迟值（在整个页面中的起始位置） */
  startDelay?: number;
}

/**
 * Dashboard KPI Panel区域组件
 */
export function DashboardKPIModules({
  technology,
  enableScrollReveal = true,
  startDelay = 2,
}: DashboardKPIModulesProps) {
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [compareWindows, setCompareWindows] = useState<Record<string, CompareWindow>>({});

  // 读全局布局（按制式）；读不到 / 为空 / 出错由 resolveLayout 回退内置默认。
  const { data: remoteLayout } = useKPILayout(technology);
  const layout = useMemo(
    () => resolveLayout(technology, remoteLayout),
    [technology, remoteLayout],
  );

  // 汇总当前制式所有图要画的指标，去重；默认拉 yesterday，last_week 按需启用。
  const metrics = useMemo(() => collectMetrics(layout.panels), [layout.panels]);
  const needsLastWeekData = useMemo(
    () => Object.values(compareWindows).some((window) => window === 'last_week'),
    [compareWindows],
  );
  const { data: yesterdayTrendData, isLoading: isYesterdayLoading } = useMultiKPITrendComparison(
    metrics,
    'yesterday',
    metrics.length > 0,
  );
  const { data: lastWeekTrendData, isLoading: isLastWeekLoading } = useMultiKPITrendComparison(
    metrics,
    'last_week',
    metrics.length > 0 && needsLastWeekData,
  );

  // 按网格坐标把图排成行（首页只读不可拖）。
  const panels = useMemo(() => layout.panels, [layout.panels]);

  // 第一次渲染后标记为非初始加载。
  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- One-time initialization after mount */
    setIsInitialLoad(false);
    /* eslint-enable react-hooks/set-state-in-effect */
  }, []);

  const shouldAnimate = enableScrollReveal && isInitialLoad;

  const getPanelKey = (panel: KPILayoutPanel) =>
    `${technology}:${panel.x}:${panel.y}:${panel.title}`;

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(2, 1fr)',
        gap: '16px',
      }}
      className={shouldAnimate ? 'omc-scroll-reveal omc-visible' : ''}
    >
      {panels.map((panel, index) => {
        const panelKey = getPanelKey(panel);
        const compareWindow = compareWindows[panelKey] ?? 'yesterday';
        const trendData = compareWindow === 'last_week' ? lastWeekTrendData : yesterdayTrendData;
        const isLoading = compareWindow === 'last_week'
          ? isLastWeekLoading
          : isYesterdayLoading;

        // 奇数个 panel 时，最后一个占满整行
        const isLastOdd = index === panels.length - 1 && panels.length % 2 === 1;

        return (
          <div
            key={`${technology}-${panel.x}-${panel.y}-${panel.title}`}
            style={isLastOdd ? { gridColumn: '1 / -1' } : {}}
            data-delay={startDelay + Math.floor(index / 2)}
          >
            <LayoutKPIPanel
              technology={technology}
              panel={panel}
              trendData={trendData}
              isLoading={isLoading}
              compareWindow={compareWindow}
              onCompareWindowChange={(nextWindow) => {
                setCompareWindows((prev) => (
                  prev[panelKey] === nextWindow ? prev : { ...prev, [panelKey]: nextWindow }
                ));
              }}
              height={280}
            />
          </div>
        );
      })}
    </div>
  );
}
