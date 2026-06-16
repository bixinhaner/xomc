import { describe, it, expect } from 'vitest';
import {
  buildRegularTimeGrid,
  gridStepMs,
  alignPointsToGrid,
} from '../buildRegularTimeGrid';

// issue #429：v2/v3 用"查询窗口起止 + 粒度"铺规整网格，空槽 null、connectNulls=false 断开线。
// 必须按窗口起止铺（非数据 min/max），否则尾部连续断档铺不出占位槽。

const MIN = 60 * 1000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;

describe('gridStepMs', () => {
  it('三种粒度步长正确', () => {
    expect(gridStepMs('15min')).toBe(15 * MIN);
    expect(gridStepMs('hourly')).toBe(HOUR);
    expect(gridStepMs('daily')).toBe(DAY);
  });
  it('未知粒度返回 undefined', () => {
    expect(gridStepMs('weekly')).toBeUndefined();
    expect(gridStepMs('monthly')).toBeUndefined();
    expect(gridStepMs('bogus')).toBeUndefined();
  });
});

describe('buildRegularTimeGrid — 槽位数', () => {
  const start = 0;

  it('15min：4 小时窗 → 17 槽（含两端）', () => {
    const grid = buildRegularTimeGrid(start, start + 4 * HOUR, '15min');
    expect(grid.length).toBe(17); // 4h/15min = 16 步 + 起点
    expect(grid[0]).toBe(0);
    expect(grid[grid.length - 1]).toBe(4 * HOUR);
  });

  it('hourly：24 小时窗 → 25 槽', () => {
    const grid = buildRegularTimeGrid(start, start + 24 * HOUR, 'hourly');
    expect(grid.length).toBe(25);
  });

  it('daily：7 天窗 → 8 槽', () => {
    const grid = buildRegularTimeGrid(start, start + 7 * DAY, 'daily');
    expect(grid.length).toBe(8);
  });

  it('未知粒度 → 空', () => {
    expect(buildRegularTimeGrid(start, start + DAY, 'weekly')).toEqual([]);
  });

  it('非法窗口 / 非数 → 空', () => {
    expect(buildRegularTimeGrid(100, 50, '15min')).toEqual([]);
    expect(buildRegularTimeGrid(NaN, 100, '15min')).toEqual([]);
    expect(buildRegularTimeGrid(0, NaN, '15min')).toEqual([]);
  });
});

describe('alignPointsToGrid — 正常铺满', () => {
  it('每个槽位都有数 → values 全非 null', () => {
    const start = 0;
    const end = 4 * HOUR;
    const grid = buildRegularTimeGrid(start, end, '15min');
    const points = grid.map((ms, i) => ({ timeMs: ms, value: i }));
    const { grid: g, values } = alignPointsToGrid(points, start, end, '15min');
    expect(g.length).toBe(17);
    expect(values.length).toBe(17);
    expect(values.every((v) => v !== null)).toBe(true);
    expect(values[0]).toBe(0);
    expect(values[16]).toBe(16);
  });
});

describe('alignPointsToGrid — 尾部断档能铺出', () => {
  it('只有前 2 个槽有数 → 后续槽位 null（断档可见）', () => {
    const start = 0;
    const end = 4 * HOUR; // 17 槽
    const points = [
      { timeMs: 0, value: 10 },
      { timeMs: 15 * MIN, value: 20 },
      // 之后全断档（设备停报），网格仍按窗口铺满
    ];
    const { grid, values } = alignPointsToGrid(points, start, end, '15min');
    expect(grid.length).toBe(17);
    expect(values[0]).toBe(10);
    expect(values[1]).toBe(20);
    // 第 3 槽起到尾部全 null
    expect(values.slice(2).every((v) => v === null)).toBe(true);
  });
});

describe('alignPointsToGrid — 中段断档 + 边界', () => {
  it('中间缺采槽位置 null，两端有数', () => {
    const start = 0;
    const end = 4 * HOUR;
    const points = [
      { timeMs: 0, value: 1 },
      { timeMs: 2 * HOUR, value: 2 }, // 第 8 槽
      { timeMs: 4 * HOUR, value: 3 }, // 第 16 槽
    ];
    const { values } = alignPointsToGrid(points, start, end, '15min');
    expect(values[0]).toBe(1);
    expect(values[8]).toBe(2);
    expect(values[16]).toBe(3);
    expect(values[1]).toBeNull();
    expect(values[7]).toBeNull();
  });

  it('value=null 的占位点不写入（仍为 null 缺采）', () => {
    const start = 0;
    const end = HOUR;
    const points = [
      { timeMs: 0, value: 5 },
      { timeMs: 15 * MIN, value: null },
    ];
    const { values } = alignPointsToGrid(points, start, end, '15min');
    expect(values[0]).toBe(5);
    expect(values[1]).toBeNull();
  });

  it('窗口外的点被忽略', () => {
    const start = HOUR;
    const end = 2 * HOUR;
    const points = [
      { timeMs: 0, value: 99 }, // 窗口前
      { timeMs: HOUR, value: 7 },
      { timeMs: 3 * HOUR, value: 88 }, // 窗口后
    ];
    const { grid, values } = alignPointsToGrid(points, start, end, '15min');
    expect(grid.length).toBe(5); // 1h/15min = 4 步 + 起点
    expect(values[0]).toBe(7);
    expect(values.filter((v) => v !== null)).toEqual([7]);
  });

  it('非整点桶 floor 对齐到所属槽位', () => {
    const start = 0;
    const end = HOUR;
    const points = [{ timeMs: 20 * MIN, value: 42 }]; // 落在第 2 槽 [15min,30min)
    const { values } = alignPointsToGrid(points, start, end, '15min');
    expect(values[1]).toBe(42);
    expect(values[0]).toBeNull();
  });
});
