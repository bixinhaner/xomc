/**
 * Dashboard KPI Panel区域 - issue #213 S2：读全局布局（带回退）驱动渲染
 *
 * 流程：按当前制式读 S1 全局布局接口 → 读不到/为空/出错回退内置默认 →
 * 汇总所有可见图要画的指标做一次批量取数 → 按网格位置（行）渲染折线图。
 * 首页只读不可拖。
 */

import { useEffect, useMemo, useState } from 'react';
import { Row, Col } from 'antd';
import type { TechnologyType } from './kpi-config';
import { useKPILayout } from '@core/hooks/api/useDashboard';
import { useMultiKPITrendComparison } from '@core/hooks/api/useDashboard';
import { LayoutKPIPanel } from '@/components/dashboard/LayoutKPIPanel';
import { resolveLayout, collectMetrics, layoutToRows } from './layoutMapping';

interface DashboardKPIModulesProps {
  /** 当前制式 */
  technology: TechnologyType;
  /** 趋势对比时窗，默认 'yesterday' */
  compareWindow?: 'yesterday' | 'last_week';
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
  compareWindow = 'yesterday',
  enableScrollReveal = true,
  startDelay = 2,
}: DashboardKPIModulesProps) {
  const [isInitialLoad, setIsInitialLoad] = useState(true);

  // 读全局布局（按制式）；读不到 / 为空 / 出错由 resolveLayout 回退内置默认。
  const { data: remoteLayout } = useKPILayout(technology);
  const layout = useMemo(
    () => resolveLayout(technology, remoteLayout),
    [technology, remoteLayout],
  );

  // 汇总当前制式所有图要画的指标，去重，做一次批量取数（今日 vs compareWindow）。
  const metrics = useMemo(() => collectMetrics(layout.panels), [layout.panels]);
  const { data: trendData, isLoading } = useMultiKPITrendComparison(metrics, compareWindow, metrics.length > 0);

  // 按网格坐标把图排成行（首页只读不可拖）。
  const rows = useMemo(() => layoutToRows(layout.panels), [layout.panels]);

  // 第一次渲染后标记为非初始加载。
  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- One-time initialization after mount */
    setIsInitialLoad(false);
    /* eslint-enable react-hooks/set-state-in-effect */
  }, []);

  const shouldAnimate = enableScrollReveal && isInitialLoad;

  return (
    <>
      {rows.map((row, rowIndex) => (
        <Row
          key={rowIndex}
          gutter={[16, 16]}
          className={shouldAnimate ? 'omc-scroll-reveal omc-visible' : ''}
          data-delay={startDelay + rowIndex}
        >
          {row.panels.map((panel) => (
            <Col
              key={`${rowIndex}-${panel.x}-${panel.title}`}
              xs={24}
              // 12 列网格 → antd 24 栅格：lg = w*2（满宽 12→24，半宽 6→12）。
              lg={Math.min(24, panel.w * 2)}
              style={{ display: 'flex' }}
            >
              <LayoutKPIPanel
                key={`${technology}-${panel.title}`}  // 制式切换时重新挂载，重置状态
                technology={technology}
                panel={panel}
                trendData={trendData}
                isLoading={isLoading}
                height={280}
              />
            </Col>
          ))}
        </Row>
      ))}
    </>
  );
}
