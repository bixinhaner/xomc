/**
 * Dashboard KPI Panel区域 - v2.0 Panel化设计
 *
 * 根据制式显示不同的Panel布局：
 * - LTE (eNB): 6个Panel（2×3网格）
 * - NR (gNB): 2个Panel（1×2布局）
 * - GSM: 3个Panel（第一行2个，第二行1个占满）
 */

import React, { useEffect, useState } from 'react';
import { Row, Col } from 'antd';
import type { TechnologyType, PanelType } from './kpi-config';
import { getPanelLayout } from './kpi-config';
import { KPIPanel } from '@/components/dashboard';

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
 *
 * @example
 * ```tsx
 * <DashboardKPIModules
 *   technology="lte"
 *   enableScrollReveal
 * />
 * ```
 */
export function DashboardKPIModules({ technology, enableScrollReveal = true, startDelay = 2 }: DashboardKPIModulesProps) {
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  // 获取当前制式的Panel布局
  const panelLayout = getPanelLayout(technology);

  // 第一次渲染后标记为非初始加载
  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- One-time initialization after mount */
    setIsInitialLoad(false);
    /* eslint-enable react-hooks/set-state-in-effect */
  }, []);

  // 初始加载时启用动画，制式切换时禁用动画直接显示
  const shouldAnimate = enableScrollReveal && isInitialLoad;

  return (
    <>
      {panelLayout.map((rowPanels, rowIndex) => (
        <Row
          key={rowIndex}
          gutter={[16, 16]}
          className={shouldAnimate ? 'omc-scroll-reveal omc-visible' : ''}
          data-delay={startDelay + rowIndex}
        >
          {rowPanels.map((panelType: PanelType) => (
            <Col
              key={`${rowIndex}-${panelType}`}
              xs={24}
              lg={rowPanels.length === 1 ? 24 : 12}
              style={{ display: 'flex' }}
            >
              <KPIPanel
                key={`${technology}-${panelType}`}  // 制式切换时重新挂载，重置状态
                technology={technology}
                panelType={panelType}
                height={280}
              />
            </Col>
          ))}
        </Row>
      ))}
    </>
  );
}
