/**
 * #444 渲染隔离：ChartCard 包 React.memo + 内部 option useMemo。
 *
 * 现象背景：设备性能查看页勾选「小时段/星期」只改 filter（不改 submitted），图表数据不重算；
 * 但 ChartCard 未 memo 化时，DeviceListPane 因 setFilter 重渲染会连带把每张 ChartCard 重渲染，
 * 配合 <ReactECharts notMerge /> 把数据没变的图强制全量重绘 → 卡顿。
 *
 * 验证：
 *   - 成功路径（memo 隔离）：父组件重渲染但 chart prop 引用不变时，ChartCard 不重渲染、
 *     ReactECharts 不收到新 option（即不重绘）。
 *   - 重画路径：chart 换成新对象（模拟点「出图」上游产新 chart）时，ChartCard 重渲染、
 *     ReactECharts 收到反映新数据的 option（series 数据随之更新）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { useState } from 'react';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';
import type { MetricChart } from './taskDashboardUtils';

// 捕获 ReactECharts 每次收到的 option（render 次数 + 内容），断言重绘行为。
const echartsRenderSpy = vi.fn();
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: { series: { data: number[] }[] } }) => {
    echartsRenderSpy(option);
    const first = option.series[0]?.data ?? [];
    return <div data-testid="echart" data-first-series={JSON.stringify(first)} />;
  },
}));

import ChartCard from './ChartCard';

function wrapIntl(node: React.ReactElement) {
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      {node}
    </IntlProvider>
  );
}

function makeChart(values: number[]): MetricChart {
  return {
    metricPath: 'm1',
    displayName: '指标A',
    buckets: ['2026-06-16T00:00:00Z', '2026-06-16T00:15:00Z'],
    bucketEnds: ['2026-06-16T00:15:00Z', '2026-06-16T00:30:00Z'],
    series: [{ key: 'dev1', name: 'dev1', values }],
  };
}

/** 受控父：用按钮触发父重渲染；prop chart 引用是否换由 swap 标志决定（模拟「勾选」vs「出图」）。 */
function Harness({ initial, swapped }: { initial: MetricChart; swapped: MetricChart }) {
  const [tick, setTick] = useState(0);
  const [swap, setSwap] = useState(false);
  return (
    <div>
      <button onClick={() => setTick((n) => n + 1)}>rerender-parent</button>
      <button onClick={() => setSwap(true)}>swap-chart</button>
      <span data-testid="tick">{tick}</span>
      <ChartCard chart={swap ? swapped : initial} />
    </div>
  );
}

/**
 * 受控父：每次父重渲染都把 chart 换成**新对象但内容相同**（真实复现：勾选只 setFilter，
 * 但上游 useAggregatedMetricsByDevices flatMap 每 render 产新数组 → charts useMemo 重算
 * → chart prop 引用每次都变，内容不变）。验证内容级 areEqual 跳过重绘。
 */
function NewRefSameContentHarness({ values }: { values: number[] }) {
  const [tick, setTick] = useState(0);
  // 每次 render 都新建 chart 对象（引用必变），内容由 values 决定。
  const chart = makeChart(values);
  return (
    <div>
      <button onClick={() => setTick((n) => n + 1)}>rerender-parent</button>
      <span data-testid="tick">{tick}</span>
      <ChartCard chart={chart} />
    </div>
  );
}

describe('ChartCard 渲染隔离 (#444)', () => {
  beforeEach(() => echartsRenderSpy.mockClear());

  it('成功路径：父重渲染但 chart 引用不变时不重绘 ECharts（memo 隔离）', async () => {
    const userEventMod = await import('@testing-library/user-event');
    const user = userEventMod.default.setup();
    const chart = makeChart([1, 2]);
    render(wrapIntl(<Harness initial={chart} swapped={makeChart([9, 9])} />));

    expect(echartsRenderSpy).toHaveBeenCalledTimes(1); // 初次渲染一次

    // 触发父组件多次重渲染（模拟反复勾选/取消小时段、星期 → setFilter）
    await user.click(screen.getByText('rerender-parent'));
    await user.click(screen.getByText('rerender-parent'));
    await user.click(screen.getByText('rerender-parent'));

    expect(screen.getByTestId('tick').textContent).toBe('3'); // 父确实重渲染了 3 次
    expect(echartsRenderSpy).toHaveBeenCalledTimes(1); // 但 ECharts 一次都没再收到 option → 零重绘
    expect(screen.getByTestId('echart').getAttribute('data-first-series')).toBe('[1,2]');
  });

  it('真实复现路径：父每次重渲染都传新 chart 对象但内容相同时不重绘（内容级 memo）', async () => {
    const userEventMod = await import('@testing-library/user-event');
    const user = userEventMod.default.setup();
    // 复现运行栈实测：勾选星期 → setFilter → 上游 flatMap 产新 rawRows → charts 重算
    // → chart prop 引用每次都变（但内容一致）。纯引用浅比较会放行重绘，内容级比较应跳过。
    render(wrapIntl(<NewRefSameContentHarness values={[3, 4]} />));

    expect(echartsRenderSpy).toHaveBeenCalledTimes(1); // 初次渲染一次

    // 反复触发父重渲染（每次都把 chart 换成新对象，内容相同）
    await user.click(screen.getByText('rerender-parent'));
    await user.click(screen.getByText('rerender-parent'));
    await user.click(screen.getByText('rerender-parent'));

    expect(screen.getByTestId('tick').textContent).toBe('3'); // 父确实重渲染 3 次，每次传新 chart 对象
    expect(echartsRenderSpy).toHaveBeenCalledTimes(1); // 内容不变 → ECharts 零重绘（修复目标）
    expect(screen.getByTestId('echart').getAttribute('data-first-series')).toBe('[3,4]');
  });

  it('重画路径：chart 换新对象时重渲染并按新数据重绘（模拟点出图）', async () => {
    const userEventMod = await import('@testing-library/user-event');
    const user = userEventMod.default.setup();
    render(wrapIntl(<Harness initial={makeChart([1, 2])} swapped={makeChart([7, 8])} />));

    expect(echartsRenderSpy).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('echart').getAttribute('data-first-series')).toBe('[1,2]');

    await user.click(screen.getByText('swap-chart')); // 上游产新 chart 对象 → memo 失效

    expect(echartsRenderSpy).toHaveBeenCalledTimes(2); // 重绘一次
    expect(screen.getByTestId('echart').getAttribute('data-first-series')).toBe('[7,8]');
  });
});
