/**
 * #200 buildUnionTimeAxis 单测。
 *
 * 验收点（死判 unit）：
 *  - 多设备/多指标稀疏时间点 → 并集去重 + 升序正确
 *  - 缺失位补 null
 *  - 每个 series 对齐后值数组长度 == xData 长度
 *  - 时间点错位场景对齐正确（按时间值索引取值，不按位置 zip）
 */

import { describe, it, expect } from 'vitest';
import { buildUnionTimeAxis } from '../buildUnionTimeAxis';

describe('buildUnionTimeAxis', () => {
  it('空输入 → 空 xData 与空 values', () => {
    const out = buildUnionTimeAxis([]);
    expect(out.xData).toEqual([]);
    expect(out.values).toEqual([]);
  });

  it('单 series 时间点直接成为 xData（升序）', () => {
    const out = buildUnionTimeAxis([
      [
        { time: '2026-06-13T00:15:00Z', value: 2 },
        { time: '2026-06-13T00:00:00Z', value: 1 },
      ],
    ]);
    expect(out.xData).toEqual(['2026-06-13T00:00:00Z', '2026-06-13T00:15:00Z']);
    expect(out.values).toEqual([[1, 2]]);
  });

  it('多 series 时间点取并集（去重 + 升序）', () => {
    const out = buildUnionTimeAxis([
      [
        { time: '2026-06-13T00:00:00Z', value: 10 },
        { time: '2026-06-13T00:30:00Z', value: 30 },
      ],
      [
        { time: '2026-06-13T00:15:00Z', value: 15 },
        { time: '2026-06-13T00:30:00Z', value: 31 },
      ],
    ]);
    // 并集：00:00 / 00:15 / 00:30，去重升序
    expect(out.xData).toEqual([
      '2026-06-13T00:00:00Z',
      '2026-06-13T00:15:00Z',
      '2026-06-13T00:30:00Z',
    ]);
    // series0 缺 00:15 → null
    expect(out.values[0]).toEqual([10, null, 30]);
    // series1 缺 00:00 → null
    expect(out.values[1]).toEqual([null, 15, 31]);
  });

  it('稀疏多设备：每个 series 对齐后长度均 == xData 长度', () => {
    const out = buildUnionTimeAxis([
      [{ time: 't1', value: 1 }],
      [{ time: 't2', value: 2 }],
      [{ time: 't3', value: 3 }],
    ]);
    expect(out.xData).toEqual(['t1', 't2', 't3']);
    out.values.forEach((vals) => {
      expect(vals.length).toBe(out.xData.length);
    });
    expect(out.values[0]).toEqual([1, null, null]);
    expect(out.values[1]).toEqual([null, 2, null]);
    expect(out.values[2]).toEqual([null, null, 3]);
  });

  it('时间点错位：按时间值对齐而非按位置 zip', () => {
    // series0 与 series1 各 2 点但时间完全错位，朴素 zip 会把不同时刻的值放同列
    const out = buildUnionTimeAxis([
      [
        { time: 'a', value: 100 },
        { time: 'c', value: 300 },
      ],
      [
        { time: 'b', value: 200 },
        { time: 'd', value: 400 },
      ],
    ]);
    expect(out.xData).toEqual(['a', 'b', 'c', 'd']);
    expect(out.values[0]).toEqual([100, null, 300, null]);
    expect(out.values[1]).toEqual([null, 200, null, 400]);
  });

  it('同 series 内同一时间点重复 → 后者覆盖前者（不影响并集去重）', () => {
    const out = buildUnionTimeAxis([
      [
        { time: 't1', value: 1 },
        { time: 't1', value: 9 },
        { time: 't2', value: 2 },
      ],
    ]);
    expect(out.xData).toEqual(['t1', 't2']);
    expect(out.values[0]).toEqual([9, 2]);
  });

  it('某 series 为空数组 → 该 series 全 null 且长度对齐', () => {
    const out = buildUnionTimeAxis([
      [
        { time: 't1', value: 1 },
        { time: 't2', value: 2 },
      ],
      [],
    ]);
    expect(out.xData).toEqual(['t1', 't2']);
    expect(out.values[0]).toEqual([1, 2]);
    expect(out.values[1]).toEqual([null, null]);
  });

  it('字符串时间按字典序升序（ISO 时间天然字典序==时间序）', () => {
    const out = buildUnionTimeAxis([
      [
        { time: '2026-06-13T09:00:00Z', value: 9 },
        { time: '2026-06-13T10:00:00Z', value: 10 },
      ],
      [{ time: '2026-06-13T08:00:00Z', value: 8 }],
    ]);
    expect(out.xData).toEqual([
      '2026-06-13T08:00:00Z',
      '2026-06-13T09:00:00Z',
      '2026-06-13T10:00:00Z',
    ]);
    expect(out.values[0]).toEqual([null, 9, 10]);
    expect(out.values[1]).toEqual([8, null, null]);
  });
});
