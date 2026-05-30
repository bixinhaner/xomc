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
  it('上一周期值按 +offsetMs 偏移对齐当前轴', () => {
    // 当前桶 10:00 / 11:00；上一周期桶（前 1 小时窗口）09:00 / 10:00。
    const offsetMs = dayjs('2026-05-25T10:00:00Z').valueOf() - dayjs('2026-05-25T09:00:00Z').valueOf();
    const currentBuckets = ['2026-05-25T10:00:00Z', '2026-05-25T11:00:00Z'];
    const prevBuckets = ['2026-05-25T09:00:00Z', '2026-05-25T10:00:00Z'];
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [5, 7] }];

    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs);
    // prev 09:00(=5) → +1h → 10:00（当前轴位置0）；prev 10:00(=7) → +1h → 11:00（当前轴位置1）
    expect(out).toHaveLength(1);
    expect(out[0].key).toBe('SN-A__prev');
    expect(out[0].name).toContain('上一周期');
    expect(out[0].values).toEqual([5, 7]);
  });

  it('当前桶在上一周期无对应 → "-"（断线）', () => {
    const offsetMs = 3600_000;
    const currentBuckets = ['2026-05-25T10:00:00Z', '2026-05-25T12:00:00Z'];
    const prevBuckets = ['2026-05-25T09:00:00Z']; // 偏移后只覆盖 10:00
    const prevSeries: MetricSeries[] = [{ key: 'SN-A', name: 'SN-A', values: [5] }];
    const out = buildCompareSeries(currentBuckets, prevSeries, prevBuckets, offsetMs);
    expect(out[0].values).toEqual([5, '-']);
  });

  it('空数据不抛错', () => {
    expect(() => buildCompareSeries([], [], [], 0)).not.toThrow();
    expect(buildCompareSeries([], [], [], 0)).toEqual([]);
  });
});

describe('attachCompareSeries', () => {
  function chart(metricPath: string, buckets: string[], series: MetricSeries[]): MetricChart {
    return { metricPath, displayName: metricPath, buckets, series };
  }

  it('按 metricPath 匹配并挂上一周期虚线系列', () => {
    const offsetMs = 3600_000;
    const cur = [
      chart('M1', ['2026-05-25T10:00:00Z'], [{ key: 'SN-A', name: 'SN-A', values: [1] }]),
    ];
    const prev = [
      chart('M1', ['2026-05-25T09:00:00Z'], [{ key: 'SN-A', name: 'SN-A', values: [9] }]),
    ];
    const out = attachCompareSeries(cur, prev, offsetMs);
    expect(out[0].compareSeries).toBeDefined();
    expect(out[0].compareSeries![0].values).toEqual([9]);
  });

  it('上一周期无对应 metric → 不挂 compareSeries', () => {
    const cur = [chart('M1', ['2026-05-25T10:00:00Z'], [{ key: 'SN-A', name: 'SN-A', values: [1] }])];
    const out = attachCompareSeries(cur, [], 3600_000);
    expect(out[0].compareSeries).toBeUndefined();
  });

  it('空集合不抛错', () => {
    expect(() => attachCompareSeries([], [], 0)).not.toThrow();
    expect(attachCompareSeries([], [], 0)).toEqual([]);
  });
});
