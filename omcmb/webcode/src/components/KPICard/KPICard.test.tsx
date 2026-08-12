/**
 * KPICard loading 态（issue #370）：
 *   - loading=true 时渲染骨架占位（Skeleton），不渲染具体 value，也不渲染 trend/delta 行
 *     —— 这样硬刷新（React Query 缓存为空）首帧不会闪现任何占位/旧数字。
 *   - loading=false（或不传）时正常渲染 value 与 trend/delta。
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import KPICard from './index';

describe('KPICard loading 态', () => {
  it('loading=true 时显示骨架、不渲染 value 与 delta', () => {
    const { container } = render(
      <KPICard
        title="设备总数"
        value={1234}
        icon={<span>icon</span>}
        trend="up"
        delta="+5"
        deltaLabel="较昨日"
      />,
    );
    // 基线：loading 未传时正常渲染值。
    expect(screen.getByText('1234')).toBeTruthy();

    // 切到 loading 态：骨架出现，value/delta 消失。
    const { container: loadingContainer } = render(
      <KPICard
        title="设备总数"
        value={1234}
        icon={<span>icon</span>}
        trend="up"
        delta="+5"
        deltaLabel="较昨日"
        loading
      />,
    );
    expect(loadingContainer.querySelector('.ant-skeleton')).toBeTruthy();
    expect(loadingContainer.textContent).not.toContain('1234');
    expect(loadingContainer.textContent).not.toContain('+5');
    expect(loadingContainer.textContent).not.toContain('较昨日');

    // 非 loading 态无骨架。
    expect(container.querySelector('.ant-skeleton')).toBeFalsy();
  });

  it('loading=false 时正常渲染 value 与 trend/delta 行', () => {
    render(
      <KPICard
        title="在线设备"
        value={1137}
        icon={<span>icon</span>}
        trend="down"
        delta="-3"
        deltaLabel="较昨日"
        loading={false}
      />,
    );
    expect(screen.getByText('1137')).toBeTruthy();
    expect(screen.getByText('-3')).toBeTruthy();
    expect(screen.getByText('较昨日')).toBeTruthy();
  });

  it('无有效历史基线时显示 N/A 且隐藏趋势箭头和对比周期', () => {
    const { container } = render(
      <KPICard
        title="活跃告警"
        value={3}
        icon={<span>icon</span>}
        trend="up"
        delta="100.0%"
        deltaLabel="较昨日"
        hasComparison={false}
      />,
    );

    expect(screen.getByText('N/A')).toBeTruthy();
    expect(container.textContent).not.toContain('100.0%');
    expect(container.textContent).not.toContain('较昨日');
    expect(container.querySelector('.anticon-arrow-up')).toBeFalsy();
  });

  it('卡片拉伸到栅格单元高度，避免无趋势卡片变矮', () => {
    const { container } = render(
      <KPICard
        title="当前接入UE数"
        value={1}
        icon={<span>icon</span>}
      />,
    );

    expect(container.querySelector('.omc-kpi-card')).toHaveStyle({
      width: '100%',
      height: '100%',
    });
    expect(container.querySelector('.ant-card')).toHaveStyle({
      height: '100%',
    });
  });
});
