import { describe, it, expect } from 'vitest';
import {
  buildTimezoneOptions,
  timezoneOffsetMinutes,
} from '../timezoneOptions';

// 固定参考时刻避免夏令时漂移影响断言。
const REF = new Date('2026-06-16T12:00:00Z');

describe('timezoneOffsetMinutes', () => {
  it('UTC 偏移为 0', () => {
    expect(timezoneOffsetMinutes('UTC', REF)).toBe(0);
  });

  it('Asia/Shanghai 偏移 +480 分钟', () => {
    expect(timezoneOffsetMinutes('Asia/Shanghai', REF)).toBe(480);
  });

  it('Asia/Kolkata 偏移 +330 分钟（半小时时区）', () => {
    expect(timezoneOffsetMinutes('Asia/Kolkata', REF)).toBe(330);
  });

  it('非法时区名返回 null', () => {
    expect(timezoneOffsetMinutes('Not/A_Zone', REF)).toBeNull();
  });
});

describe('buildTimezoneOptions', () => {
  const opts = buildTimezoneOptions(REF);

  it('生成完整 IANA 列表（数百项）', () => {
    // 现代运行环境 supportedValuesOf 通常 400+ 项；即便回退也 >=9 项。
    expect(opts.length).toBeGreaterThan(50);
  });

  it('每项 value 为 IANA 名、label 带 GMT 偏移', () => {
    const sh = opts.find((o) => o.value === 'Asia/Shanghai');
    expect(sh).toBeDefined();
    expect(sh!.label).toContain('GMT+08:00');
    expect(sh!.label).toContain('Asia/Shanghai');
  });

  it('包含 UTC', () => {
    expect(opts.some((o) => o.value === 'UTC')).toBe(true);
  });

  it('按偏移升序排序（解析偏移单调不降）', () => {
    const offsets = opts
      .map((o) => timezoneOffsetMinutes(o.value, REF))
      .filter((v): v is number => v !== null);
    for (let i = 1; i < offsets.length; i++) {
      expect(offsets[i]).toBeGreaterThanOrEqual(offsets[i - 1]);
    }
  });
});
