import { describe, it, expect } from 'vitest';
import dayjs from 'dayjs';
import {
  ALL_HOURS,
  ALL_WEEKDAYS,
  attachCompareSeries,
  buildCompareSeries,
  filterRowsByWeekdayHour,
  previousWindow,
} from './dashboardFilterUtils';
import type { MetricChart, MetricSeries } from './taskDashboardUtils';

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
