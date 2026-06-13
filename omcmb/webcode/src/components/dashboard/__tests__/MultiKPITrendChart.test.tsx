/**
 * #200 MultiKPITrendChart 组件接入并集时间轴 + ECharts 续连 单测（死判 echarts-opt）。
 *
 * 验收点：
 *  - 用构造的稀疏多设备/多指标数据，经 buildMultiKpiTrendModel 装配 + buildLineChartOption
 *    组装出真实 ECharts option；
 *  - 断言 option.series 全部 connectNulls === true（KPI 趋势场景跨 null 续连）；
 *  - 断言 option.xAxis.data 长度 == 并集时间点数（非旧"只取第一个 KPI 时间点"逻辑）；
 *  - 打印 option / series data 供运行栈 agent 自判读（确认稀疏点因并集轴 + connectNulls 续连，
 *    非孤点 / 贴底平线）。本测试用构造数据验逻辑，非真机曲线。
 */

import { describe, it, expect } from 'vitest';
import { buildMultiKpiTrendModel } from '@core/utils/buildMultiKpiTrendModel';
import { buildLineChartOption } from '@/components/Charts/LineChart';

// 构造 3 个 KPI（模拟多设备/多指标）稀疏 + 时间点错位的数据：
// - KPI A 打点在 00/15/45（缺 30）
// - KPI B 打点在 15/30（与 A 部分重叠、部分错位）
// - KPI C 打点在 00/30/45（与 A/B 各错开）
// 三者时间点并集应为 {00,15,30,45} 共 4 个。旧"按索引 zip"会把 B 的 15 当成 00 列错位。
const T0 = '2026-06-13T00:00:00Z';
const T15 = '2026-06-13T00:15:00Z';
const T30 = '2026-06-13T00:30:00Z';
const T45 = '2026-06-13T00:45:00Z';

const sparseInputs = [
  {
    name: '上行速率',
    color: '#5470c6',
    points: [
      { time: T0, value: 10 },
      { time: T15, value: 12 },
      { time: T45, value: 18 }, // 缺 T30
    ],
  },
  {
    name: '下行速率',
    color: '#91cc75',
    points: [
      { time: T15, value: 20 }, // 起点就错位（不在 T0）
      { time: T30, value: 22 },
    ],
  },
  {
    name: 'PRB利用率',
    color: '#fac858',
    points: [
      { time: T0, value: 0 }, // 显式 value=0：必须保留为 0，不能被当 null 丢掉
      { time: T30, value: 30 },
      { time: T45, value: 35 },
    ],
  },
];

const UNION_LEN = 4; // {T0,T15,T30,T45}

function buildOptionFromInputs() {
  const model = buildMultiKpiTrendModel(sparseInputs, 'yesterday');
  return {
    model,
    option: buildLineChartOption({
      title: '',
      xData: model.xData,
      xDataFull: model.xDataFull,
      series: model.series,
      areaFill: true,
      smooth: true,
      showLegend: true,
      connectNulls: true, // KPI 趋势场景开启续连
      isDark: false,
      appTheme: 'classic',
      palette: ['#5470c6', '#91cc75', '#fac858', '#ee6666'],
    }),
  };
}

describe('MultiKPITrendChart option（并集时间轴 + connectNulls 续连）', () => {
  it('xAxis.data 长度 == 并集时间点数（非只取第一个 KPI 时间点）', () => {
    const { option } = buildOptionFromInputs();
    const xAxis = option.xAxis as { data: string[] };
    expect(xAxis.data).toHaveLength(UNION_LEN);
  });

  it('option.series 全部 connectNulls === true', () => {
    const { option } = buildOptionFromInputs();
    const series = option.series as Array<{ connectNulls?: boolean; data: Array<number | null> }>;
    expect(series).toHaveLength(3);
    series.forEach((s) => {
      expect(s.connectNulls).toBe(true);
    });
  });

  it('各 series 值数组长度 == 并集长度，缺采点为 null（稀疏续连的依据）', () => {
    const { option } = buildOptionFromInputs();
    const series = option.series as Array<{ name: string; data: Array<number | null> }>;
    series.forEach((s) => {
      expect(s.data).toHaveLength(UNION_LEN);
    });
    // KPI A: T0=10,T15=12,T30 缺=null,T45=18
    expect(series[0].data).toEqual([10, 12, null, 18]);
    // KPI B: T0 缺=null,T15=20,T30=22,T45 缺=null
    expect(series[1].data).toEqual([null, 20, 22, null]);
    // KPI C: T0=0(显式保留),T15 缺=null,T30=30,T45=35
    expect(series[2].data).toEqual([0, null, 30, 35]);
  });

  it('value=0 不被当作 null 丢失（PRB 起点 0 仍在）', () => {
    const { option } = buildOptionFromInputs();
    const series = option.series as Array<{ data: Array<number | null> }>;
    expect(series[2].data[0]).toBe(0);
  });

  it('打印 option / series data 供自判读（构造数据验逻辑，非真机曲线）', () => {
    const { model, option } = buildOptionFromInputs();
    const series = option.series as Array<{ name: string; connectNulls?: boolean; data: Array<number | null> }>;
    const xAxis = option.xAxis as { data: string[] };

    // 供运行栈 agent 读证据：稀疏点因并集轴 + connectNulls 而续连，非孤点 / 贴底平线
    // eslint-disable-next-line no-console
    console.log('[#200 自判证据] xAxis.data(展示标签) =', JSON.stringify(xAxis.data));
    // eslint-disable-next-line no-console
    console.log('[#200 自判证据] xDataFull(原始并集时间) =', JSON.stringify(model.xDataFull));
    series.forEach((s) => {
      // eslint-disable-next-line no-console
      console.log(
        `[#200 自判证据] series "${s.name}" connectNulls=${s.connectNulls} data=`,
        JSON.stringify(s.data),
      );
    });

    expect(xAxis.data.length).toBe(model.xDataFull.length);
  });
});
