import { describe, it, expect } from 'vitest';
import dayjs from 'dayjs';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  buildCompareSeries,
  dimSelectionToParams,
  dimensionFilterMeta,
  extendChartAxis,
  extendChartsAxis,
  filterRowsByWeekdayHour,
  parseBandNumber,
  previousWindow,
} from './dashboardFilterUtils';
import type { MetricChart, MetricSeries, MetricSeriesValue } from './taskDashboardUtils';

// 本地时区构造行：用 dayjs 本地构造再取 ISO，过滤口径与读回口径一致，不受 CI TZ 影响。
function rowAt(local: string) {
  // local 形如 '2026-05-25 08:30'（按本地时区解释）
  return { startTime: dayjs(local).toISOString() };
}

describe('filterRowsByWeekdayHour', () => {
  it('全选（默认）不过滤，原样返回', () => {
    const rows = [rowAt('2026-05-25 08:00'), rowAt('2026-05-26 13:00')];
    const out = filterRowsByWeekdayHour(rows, new Set(ALL_WEEKDAYS), new Set(ALL_HOURS));
    expect(out).toBe(rows); // 同一引用，未拷贝
  });

  it('只勾某星期 + 某小时段命中正确', () => {
    // 2026-05-25 是周一（dayjs().day()===1），下面用 dayjs 校验后断言。
    const monday0800 = rowAt('2026-05-25 08:30');
    const monday0900 = rowAt('2026-05-25 09:30');
    const tuesday0800 = rowAt('2026-05-26 08:30');
    const monday1000 = rowAt('2026-05-25 10:30');
    // 确认构造的星期/小时与预期一致（防 TZ 把日期推走）。
    expect(dayjs(monday0800.startTime).day()).toBe(1);
    expect(dayjs(monday0800.startTime).hour()).toBe(8);

    const rows = [monday0800, monday0900, tuesday0800, monday1000];
    const out = filterRowsByWeekdayHour(rows, new Set([1]), new Set([8, 9]));
    // 只剩周一且 8/9 点 → monday0800 + monday0900
    expect(out).toEqual([monday0800, monday0900]);
  });

  it('只勾星期、小时全选 → 只按星期筛', () => {
    const monday = rowAt('2026-05-25 03:00');
    const tuesday = rowAt('2026-05-26 03:00');
    const out = filterRowsByWeekdayHour([monday, tuesday], new Set([1]), new Set(ALL_HOURS));
    expect(out).toEqual([monday]);
  });

  it('空数据不抛错，返回空', () => {
    expect(() => filterRowsByWeekdayHour([], new Set([1]), new Set([8]))).not.toThrow();
    expect(filterRowsByWeekdayHour([], new Set([1]), new Set([8]))).toEqual([]);
  });

  it('非法 startTime 行被丢弃（非全选时）', () => {
    const bad = { startTime: 'not-a-time' };
    const out = filterRowsByWeekdayHour([bad], new Set([1]), new Set([8]));
    expect(out).toEqual([]);
  });
});

describe('previousWindow', () => {
  it('上一周期长度守恒（= 当前窗口长）', () => {
    const start = dayjs('2026-05-23T00:00:00Z');
    const end = dayjs('2026-05-30T00:00:00Z');
    const [ps, pe] = previousWindow([start, end]);
    const curLen = end.valueOf() - start.valueOf();
    const prevLen = pe.valueOf() - ps.valueOf();
    expect(prevLen).toBe(curLen);
    // 上一周期终点 = 当前起点
    expect(pe.valueOf()).toBe(start.valueOf());
    // 上一周期起点 = 当前起点 - L
    expect(ps.valueOf()).toBe(start.valueOf() - curLen);
  });
});

describe('buildCompareSeries', () => {
  it('上一周期值按整数粒度步长偏移对齐当前轴（干净 offset）', () => {
    // 当前桶 10:00 / 11:00；上一周期桶（前 1 小时窗口）09:00 / 10:00。
    const offsetMs = dayjs('2026-05-25T10:00:00Z').valueOf() - dayjs('2026-05-25T09:00:00Z').valueOf();
    const currentBuckets = ['2026-05-25T10:00:00Z', '2026-05-25T11:00:00Z'];
    const prevBuckets = ['2026-05-25T09:00:00Z', '2026-05-25T10:00:00Z'];
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [5, 7] }];

    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs, 'hourly');
    // prev 09:00(=5) → +1h → 10:00（当前轴位置0）；prev 10:00(=7) → +1h → 11:00（当前轴位置1）
    expect(out.series).toHaveLength(1);
    expect(out.series[0].key).toBe('SN-A__prev');
    expect(out.series[0].name).toContain('上一周期');
    expect(out.series[0].values).toEqual([5, 7]);
  });

  it('核心回归：offsetMs 带亚秒/分钟零头仍精确对齐（吸附整步长）', () => {
    // hourly：窗口长 = 1h + 643ms 零头（默认 end=dayjs() 带毫秒典型场景）。
    const offsetMs = 3600_000 + 643;
    const currentBuckets = ['2026-05-25T10:00:00Z', '2026-05-25T11:00:00Z'];
    const prevBuckets = ['2026-05-25T09:00:00Z', '2026-05-25T10:00:00Z'];
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [5, 7] }];
    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs, 'hourly');
    // 吸附到 3600000，整点桶精确落位，不再全 '-'。
    expect(out.series[0].values).toEqual([5, 7]);
  });

  it('daily 粒度按整日步长吸附（offset 带零头）', () => {
    const offsetMs = 86_400_000 + 12_345; // 1 天 + 零头
    const currentBuckets = ['2026-05-25T00:00:00Z', '2026-05-26T00:00:00Z'];
    const prevBuckets = ['2026-05-24T00:00:00Z', '2026-05-25T00:00:00Z'];
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [3, 4] }];
    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs, 'daily');
    expect(out.series[0].values).toEqual([3, 4]);
  });

  it('month 粒度按整数日历月平移（不按固定毫秒漂移出月界）', () => {
    // 上一周期 3/31、4/30；offset≈1 个月，日历月平移后应落到当前轴 4/30、5/31。
    const offsetMs = 30 * 86_400_000 + 999; // ≈1 月 + 零头
    const currentBuckets = ['2026-04-30T00:00:00Z', '2026-05-31T00:00:00Z'];
    const prevBuckets = ['2026-03-30T00:00:00Z', '2026-04-30T00:00:00Z'];
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [11, 22] }];
    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs, 'monthly');
    // 3/30 +1月 → 4/30（位置0）；4/30 +1月 → 5/30 ≠ 5/31 → 位置1 无对应为 '-'。
    expect(out.series[0].values[0]).toBe(11);
    expect(out.series[0].values[1]).toBe('-');
  });

  it('当前桶在上一周期无对应 → "-"（断线，不串位）', () => {
    const offsetMs = 3600_000;
    const currentBuckets = ['2026-05-25T10:00:00Z', '2026-05-25T12:00:00Z'];
    const prevBuckets = ['2026-05-25T09:00:00Z']; // 偏移后只覆盖 10:00
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [5] }];
    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs, 'hourly');
    expect(out.series[0].values).toEqual([5, '-']);
  });

  it('compareBuckets 与当前轴索引对齐，留上一周期真实起止时间（无对应处空串）', () => {
    const offsetMs = 3600_000;
    const currentBuckets = ['2026-05-25T10:00:00Z', '2026-05-25T11:00:00Z'];
    const prevBuckets = ['2026-05-25T09:00:00Z']; // 只覆盖当前轴位置0
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [5] }];
    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs, 'hourly');
    // 位置0 对应上一周期真实桶 09:00（非当前轴 10:00 标签）；位置1 无对应 → 空串。
    expect(out.compareBuckets).toEqual(['2026-05-25T09:00:00Z', '']);
  });

  it('空数据不抛错', () => {
    expect(() => buildCompareSeries([], [], [], 0, 'hourly')).not.toThrow();
    expect(buildCompareSeries([], [], [], 0, 'hourly').series).toEqual([]);
  });
});

// ── PM-DASH-DIMFILTER 维度子集筛选纯函数 ───────────────────────────────

describe('parseBandNumber（频段可读化）', () => {
  it("'Band=42' → '42'（去前缀，配合 i18n 模板渲染「频段 42」)", () => {
    expect(parseBandNumber('Band=42')).toBe('42');
  });

  it("'Band=1' → '1'", () => {
    expect(parseBandNumber('Band=1')).toBe('1');
  });

  it('无 Band= 前缀（脏数据/非频段值）→ 原样返回，不崩、label 非空', () => {
    expect(parseBandNumber('42')).toBe('42');
    expect(parseBandNumber('')).toBe('');
  });
});

describe('dimensionFilterMeta（按维度决定是否渲染筛选框）', () => {
  it('product → 渲染「产品」框（返回产品标题/占位符键）', () => {
    expect(dimensionFilterMeta('product')).toEqual({
      titleId: 'perf.dashboard.filterProduct',
      placeholderId: 'perf.dashboard.allProducts',
    });
  });

  it('device_group → 渲染「设备组」框', () => {
    expect(dimensionFilterMeta('device_group')).toEqual({
      titleId: 'perf.dashboard.filterDeviceGroup',
      placeholderId: 'perf.dashboard.allDeviceGroups',
    });
  });

  it('band → 渲染「频段」框', () => {
    expect(dimensionFilterMeta('band')).toEqual({
      titleId: 'perf.dashboard.filterBand',
      placeholderId: 'perf.dashboard.allBands',
    });
  });

  it('device / aggregate_group / network / undefined → 不渲染（null）', () => {
    expect(dimensionFilterMeta('device')).toBeNull();
    expect(dimensionFilterMeta('aggregate_group')).toBeNull();
    expect(dimensionFilterMeta('network')).toBeNull();
    expect(dimensionFilterMeta(undefined)).toBeNull();
  });
});

describe('dimSelectionToParams（维度选中值→results 入参）', () => {
  it('product 维度选中 → productIds（不带 objectLdns）', () => {
    const out = dimSelectionToParams('product', ['p1', 'p2']);
    expect(out).toEqual({ productIds: ['p1', 'p2'] });
    expect(out.objectLdns).toBeUndefined();
  });

  it('device_group 维度选中 → objectLdns（值是 DeviceGroup=<uuid> 原值）', () => {
    const out = dimSelectionToParams('device_group', ['DeviceGroup=g1', 'DeviceGroup=g2']);
    expect(out).toEqual({ objectLdns: ['DeviceGroup=g1', 'DeviceGroup=g2'] });
    expect(out.productIds).toBeUndefined();
  });

  it('band 维度选中 → objectLdns（值是 Band=<值> 原值）', () => {
    const out = dimSelectionToParams('band', ['Band=42']);
    expect(out).toEqual({ objectLdns: ['Band=42'] });
  });

  it('空选中 → 不过滤（两者皆 undefined，无回归）', () => {
    expect(dimSelectionToParams('product', [])).toEqual({});
    expect(dimSelectionToParams('band', [])).toEqual({});
  });

  it('不可筛维度（network 等）即便误传选中也不映射出过滤入参', () => {
    expect(dimSelectionToParams('network', ['x'])).toEqual({});
    expect(dimSelectionToParams('device', ['x'])).toEqual({});
  });
});

describe('attachCompareSeries', () => {
  function chart(
    metricPath: string,
    buckets: string[],
    series: MetricSeries[],
    bucketEnds?: string[],
  ): MetricChart {
    return { metricPath, displayName: metricPath, buckets, bucketEnds: bucketEnds ?? [], series };
  }

  it('按 metricPath 匹配并挂上一周期虚线系列（带零头 offset 仍对齐）', () => {
    const offsetMs = 3600_000 + 500;
    const cur = [
      chart('M1', ['2026-05-25T10:00:00Z'], [{ key: 'SN-A', name: 'SN-A', values: [1] }]),
    ];
    const prev = [
      chart('M1', ['2026-05-25T09:00:00Z'], [{ key: 'SN-A', name: 'SN-A', values: [9] }],
        ['2026-05-25T10:00:00Z']),
    ];
    const out = attachCompareSeries(cur, prev, offsetMs, 'hourly');
    expect(out[0].compareSeries).toBeDefined();
    expect(out[0].compareSeries![0].values).toEqual([9]);
    // tooltip 用上一周期真实桶起止时间（与当前轴索引对齐）。
    expect(out[0].compareBuckets).toEqual(['2026-05-25T09:00:00Z']);
    expect(out[0].compareBucketEnds).toEqual(['2026-05-25T10:00:00Z']);
  });

  it('上一周期无对应 metric → 不挂 compareSeries', () => {
    const cur = [chart('M1', ['2026-05-25T10:00:00Z'], [{ key: 'SN-A', name: 'SN-A', values: [1] }])];
    const out = attachCompareSeries(cur, [], 3600_000, 'hourly');
    expect(out[0].compareSeries).toBeUndefined();
  });

  it('空集合不抛错', () => {
    expect(() => attachCompareSeries([], [], 0, 'hourly')).not.toThrow();
    expect(attachCompareSeries([], [], 0, 'hourly')).toEqual([]);
  });
});

// ── T-AXISFILL 横轴铺满后处理（验收 6 条，行为非实现）───────────────────────
describe('extendChartAxis（横轴铺满 + 空点如实显示）', () => {
  function chart(
    buckets: string[],
    values: MetricSeriesValue[],
    bucketEnds?: string[],
  ): MetricChart {
    return {
      metricPath: 'M1',
      displayName: 'M1',
      buckets,
      bucketEnds: bucketEnds ?? buckets.map(() => ''),
      series: [{ key: 'SN-A', name: 'SN-A', values }],
    };
  }
  const ms = (local: string) => dayjs(local).valueOf();
  // 全选集合（不触发星期/小时裁）。
  const allWd = new Set(ALL_WEEKDAYS);
  const allHr = new Set(ALL_HOURS);

  it('① 连续生成：稀疏真实桶 + 跨多刻度 → 补齐范围内全部应有刻度，无数据刻度值为 "-"', () => {
    // daily 粒度，范围 5/25~5/28（含两端），真实只有 5/25 与 5/28。
    const b25 = dayjs('2026-05-25 00:00').toISOString();
    const b28 = dayjs('2026-05-28 00:00').toISOString();
    const out = extendChartAxis(chart([b25, b28], [10, 40]), {
      rangeStartMs: ms('2026-05-25 00:00'),
      rangeEndMs: ms('2026-05-28 00:00'),
      weekdays: allWd,
      hours: allHr,
      granularity: 'daily',
    });
    // 5/25 5/26 5/27 5/28 共 4 刻度。
    expect(out.buckets).toHaveLength(4);
    const days = out.buckets.map((b) => dayjs(b).date());
    expect(days).toEqual([25, 26, 27, 28]);
    // 真实刻度有值，中间两天补 '-'。
    expect(out.series[0].values).toEqual([10, '-', '-', 40]);
  });

  it('② 星期裁剪：只勾周一 → 只含范围内周一（含无数据周一），不含非周一', () => {
    // 范围 5/13~5/28（daily）。5/18、5/25 是周一；5/25 有数据、5/18 无数据。
    const b25 = dayjs('2026-05-25 00:00').toISOString();
    // 先确认构造的周一 day()===1（防 TZ）。
    expect(dayjs('2026-05-18 00:00').day()).toBe(1);
    expect(dayjs('2026-05-25 00:00').day()).toBe(1);
    const out = extendChartAxis(chart([b25], [99]), {
      rangeStartMs: ms('2026-05-13 00:00'),
      rangeEndMs: ms('2026-05-28 00:00'),
      weekdays: new Set([1]),
      hours: allHr,
      granularity: 'daily',
    });
    // 全部刻度都是周一。
    out.buckets.forEach((b) => expect(dayjs(b).day()).toBe(1));
    const dates = out.buckets.map((b) => dayjs(b).date()).sort((a, b) => a - b);
    // 范围内周一：5/18、5/25（5/11 在范围外，6/1 在范围外）。
    expect(dates).toEqual([18, 25]);
    // 5/18（空周一）如实在轴上、值为 '-'；5/25 有值 99。
    const idx18 = out.buckets.findIndex((b) => dayjs(b).date() === 18);
    const idx25 = out.buckets.findIndex((b) => dayjs(b).date() === 25);
    expect(out.series[0].values[idx18]).toBe('-');
    expect(out.series[0].values[idx25]).toBe(99);
  });

  it('③ 并集兜底：错相位真实桶（落生成刻度之外）仍在轴上、不丢点', () => {
    // daily 锚点 5/25 00:00；额外真实桶 5/26 06:30（非整日相位，生成刻度推不到它）。
    const b25 = dayjs('2026-05-25 00:00').toISOString();
    const bOdd = dayjs('2026-05-26 06:30').toISOString();
    const out = extendChartAxis(chart([b25, bOdd], [1, 2]), {
      rangeStartMs: ms('2026-05-25 00:00'),
      rangeEndMs: ms('2026-05-27 00:00'),
      weekdays: allWd,
      hours: allHr,
      granularity: 'daily',
    });
    // 错相位真实桶必须仍在轴上。
    const oddIdx = out.buckets.findIndex((b) => dayjs(b).valueOf() === ms('2026-05-26 06:30'));
    expect(oddIdx).toBeGreaterThanOrEqual(0);
    expect(out.series[0].values[oddIdx]).toBe(2);
    // 锚点也在。
    const anchorIdx = out.buckets.findIndex((b) => dayjs(b).valueOf() === ms('2026-05-25 00:00'));
    expect(out.series[0].values[anchorIdx]).toBe(1);
  });

  it('④ 月粒度：按自然月推刻度，不漂出月界', () => {
    // monthly 锚点 1/31；范围 1/31~4/30。日历平移应落到 1/31、2/28、3/31、4/30（不变成固定 30 天漂移）。
    const b0131 = dayjs('2026-01-31 00:00').toISOString();
    const out = extendChartAxis(chart([b0131], [7]), {
      rangeStartMs: ms('2026-01-31 00:00'),
      rangeEndMs: ms('2026-04-30 00:00'),
      weekdays: allWd,
      hours: allHr,
      granularity: 'monthly',
    });
    // 各刻度落在各自然月末（dayjs add month 对月末做钳位）。
    const months = out.buckets.map((b) => dayjs(b).month()); // 0=1月
    // 锚点 1月在，2/3/4 月各一刻度。
    expect(months).toContain(0);
    expect(months).toContain(1);
    expect(months).toContain(2);
    expect(months).toContain(3);
    // 不漂出月界：2 月那刻仍在 2 月（dayjs(1/31).add(1,'month') = 2/28）。
    const febIdx = months.indexOf(1);
    expect(dayjs(out.buckets[febIdx]).month()).toBe(1);
  });

  it('⑤ 空图跳过：空 buckets → 原样返回、不报错、不强造轴', () => {
    const empty = chart([], []);
    const out = extendChartAxis(empty, {
      rangeStartMs: ms('2026-05-25 00:00'),
      rangeEndMs: ms('2026-05-28 00:00'),
      weekdays: allWd,
      hours: allHr,
      granularity: 'daily',
    });
    expect(out).toBe(empty); // 同引用，原样返回
  });

  it('未知粒度退化为仅真实桶（安全兜底，不报错）', () => {
    const b25 = dayjs('2026-05-25 00:00').toISOString();
    const out = extendChartAxis(chart([b25], [5]), {
      rangeStartMs: ms('2026-05-20 00:00'),
      rangeEndMs: ms('2026-05-30 00:00'),
      weekdays: allWd,
      hours: allHr,
      granularity: 'unknown-gran',
    });
    // 不生成新刻度，轴 = 仅真实桶。
    expect(out.buckets).toHaveLength(1);
    expect(out.series[0].values).toEqual([5]);
  });

  it('空刻度的 bucketEnds 落「下一刻度」时间（供 tooltip），真实刻度保留原 end', () => {
    const b25 = dayjs('2026-05-25 00:00').toISOString();
    const b27 = dayjs('2026-05-27 00:00').toISOString();
    const out = extendChartAxis(
      chart([b25, b27], [10, 30], ['real-end-25', 'real-end-27']),
      {
        rangeStartMs: ms('2026-05-25 00:00'),
        rangeEndMs: ms('2026-05-27 00:00'),
        weekdays: allWd,
        hours: allHr,
        granularity: 'daily',
      },
    );
    // 真实刻度 end 原样保留。
    const idx25 = out.buckets.findIndex((b) => dayjs(b).valueOf() === ms('2026-05-25 00:00'));
    expect(out.bucketEnds[idx25]).toBe('real-end-25');
    // 空刻度 5/26 的 end = 下一刻度 5/27 的 ms（ISO 可解析）。
    const idx26 = out.buckets.findIndex((b) => dayjs(b).valueOf() === ms('2026-05-26 00:00'));
    expect(idx26).toBeGreaterThanOrEqual(0);
    expect(dayjs(out.bucketEnds[idx26]).valueOf()).toBe(ms('2026-05-27 00:00'));
  });
});

describe('extendChartsAxis（批量 + 周期对比回归）', () => {
  const ms = (local: string) => dayjs(local).valueOf();
  const allWd = new Set(ALL_WEEKDAYS);
  const allHr = new Set(ALL_HOURS);

  it('空集合不抛错', () => {
    expect(() =>
      extendChartsAxis([], {
        rangeStartMs: 0,
        rangeEndMs: 1,
        weekdays: allWd,
        hours: allHr,
        granularity: 'daily',
      }),
    ).not.toThrow();
    expect(
      extendChartsAxis([], {
        rangeStartMs: 0,
        rangeEndMs: 1,
        weekdays: allWd,
        hours: allHr,
        granularity: 'daily',
      }),
    ).toEqual([]);
  });

  it('⑥ 周期对比回归：扩轴后 attachCompareSeries 仍按毫秒对齐，空刻度处对比线为 "-"', () => {
    // 当前真实桶稀疏（仅 5/25、5/27，daily），扩轴后补 5/26 空刻度。
    const b25 = dayjs('2026-05-25 00:00').toISOString();
    const b27 = dayjs('2026-05-27 00:00').toISOString();
    const cur: MetricChart = {
      metricPath: 'M1',
      displayName: 'M1',
      buckets: [b25, b27],
      bucketEnds: ['', ''],
      series: [{ key: 'SN-A', name: 'SN-A', values: [10, 30] }],
    };
    const extended = extendChartsAxis([cur], {
      rangeStartMs: ms('2026-05-25 00:00'),
      rangeEndMs: ms('2026-05-27 00:00'),
      weekdays: allWd,
      hours: allHr,
      granularity: 'daily',
    });
    // 扩轴后轴 = 5/25 5/26 5/27。
    expect(extended[0].buckets).toHaveLength(3);

    // 上一周期 = 前 2 天窗口，真实桶 5/23、5/25（offset = 2 天）。
    const offsetMs = ms('2026-05-25 00:00') - ms('2026-05-23 00:00');
    const prev: MetricChart = {
      metricPath: 'M1',
      displayName: 'M1',
      buckets: [dayjs('2026-05-23 00:00').toISOString(), dayjs('2026-05-25 00:00').toISOString()],
      bucketEnds: ['', ''],
      series: [{ key: 'SN-A', name: 'SN-A', values: [7, 9] }],
    };
    const out = attachCompareSeries(extended, [prev], offsetMs, 'daily');
    // prev 5/23(+2d=5/25 轴位0)=7；prev 5/25(+2d=5/27 轴位2)=9；轴位1（5/26 空刻度）无对应 → '-'。
    expect(out[0].compareSeries).toBeDefined();
    expect(out[0].compareSeries![0].values).toEqual([7, '-', 9]);
  });
});
