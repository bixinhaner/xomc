import { describe, it, expect } from 'vitest';
import {
  formatSystemTime,
  formatSystemDate,
  formatSystemTimeOnly,
  nowInSystemTimezone,
  systemTimezoneLabel,
  toSystemTimezoneRFC3339,
} from '../systemTime';

describe('formatSystemTime — 保留后端系统时区钟面，不做浏览器本地转换', () => {
  // 关键：带偏移 ISO 的钟面必须原样呈现，不被运行环境（浏览器/CI）的 OS 时区二次转换。
  it('带 +09:00 偏移的 ISO 原样显示东京钟面', () => {
    expect(formatSystemTime('2026-06-16T10:00:00+09:00')).toBe('2026-06-16 10:00:00');
  });

  it('带 +08:00 偏移的 ISO 原样显示上海钟面', () => {
    expect(formatSystemTime('2026-06-16T18:30:45+08:00')).toBe('2026-06-16 18:30:45');
  });

  it('Z（UTC）偏移原样显示 UTC 钟面', () => {
    expect(formatSystemTime('2026-06-16T10:00:00Z')).toBe('2026-06-16 10:00:00');
  });

  it('负偏移 -05:00 原样显示', () => {
    expect(formatSystemTime('2026-06-16T07:00:00-05:00')).toBe('2026-06-16 07:00:00');
  });

  it('model.Time 无偏移钟面串（YYYY/M/D H:m:s）按字面原样显示', () => {
    expect(formatSystemTime('2026/6/16 10:00:00')).toBe('2026-06-16 10:00:00');
  });

  it('无偏移 YYYY-MM-DD HH:mm:ss 钟面串按字面原样显示', () => {
    expect(formatSystemTime('2026-06-16 09:05:03')).toBe('2026-06-16 09:05:03');
  });

  it('支持自定义格式（含毫秒）', () => {
    expect(
      formatSystemTime('2026-06-16T10:00:00.123+09:00', { format: 'YYYY-MM-DD HH:mm:ss.SSS' })
    ).toBe('2026-06-16 10:00:00.123');
  });

  // 失败路径
  it('null 返回默认占位 -', () => {
    expect(formatSystemTime(null)).toBe('-');
  });

  it('undefined / 空串返回占位', () => {
    expect(formatSystemTime(undefined)).toBe('-');
    expect(formatSystemTime('')).toBe('-');
  });

  it('非法字符串返回自定义占位', () => {
    expect(formatSystemTime('not-a-date', { placeholder: '—' })).toBe('—');
  });

  it('epoch 毫秒按系统时区落钟面（Asia/Tokyo）', () => {
    // 2026-06-16T01:00:00Z → 东京 +09:00 → 10:00
    const ms = Date.UTC(2026, 5, 16, 1, 0, 0);
    expect(formatSystemTime(ms, { systemTimezone: 'Asia/Tokyo' })).toBe('2026-06-16 10:00:00');
  });

  it('epoch 毫秒无系统时区时回落 UTC', () => {
    const ms = Date.UTC(2026, 5, 16, 1, 0, 0);
    expect(formatSystemTime(ms)).toBe('2026-06-16 01:00:00');
  });
});

describe('formatSystemDate / formatSystemTimeOnly', () => {
  it('仅日期', () => {
    expect(formatSystemDate('2026-06-16T10:00:00+09:00')).toBe('2026-06-16');
  });
  it('仅时间', () => {
    expect(formatSystemTimeOnly('2026-06-16T10:00:00+09:00')).toBe('10:00:00');
  });
  it('空值占位', () => {
    expect(formatSystemDate(null)).toBe('-');
  });
});

describe('systemTimezoneLabel', () => {
  it('UTC / 空 → UTC', () => {
    expect(systemTimezoneLabel('UTC')).toBe('UTC');
    expect(systemTimezoneLabel(undefined)).toBe('UTC');
    expect(systemTimezoneLabel('')).toBe('UTC');
  });
  it('IANA 名原样返回', () => {
    expect(systemTimezoneLabel('Asia/Tokyo')).toBe('Asia/Tokyo');
  });
});

describe('nowInSystemTimezone', () => {
  it('返回有效 dayjs（系统时区）', () => {
    expect(nowInSystemTimezone('Asia/Tokyo').isValid()).toBe(true);
  });
  it('无时区回落 UTC 仍有效', () => {
    expect(nowInSystemTimezone(undefined).isValid()).toBe(true);
  });
});

describe('toSystemTimezoneRFC3339 — 筛选输入按系统时区附加偏移', () => {
  it('用户选的钟面按 Asia/Tokyo 解释，输出带 +09:00 偏移', () => {
    const out = toSystemTimezoneRFC3339('2026-06-16 10:00:00', 'Asia/Tokyo');
    expect(out).toMatch(/\+09:00$/);
    expect(out).toContain('2026-06-16T10:00:00');
  });

  it('Asia/Shanghai 输出 +08:00 偏移', () => {
    const out = toSystemTimezoneRFC3339('2026-06-16 18:00:00', 'Asia/Shanghai');
    expect(out).toMatch(/\+08:00$/);
  });

  it('无系统时区回落 UTC（Z 或 +00:00）', () => {
    const out = toSystemTimezoneRFC3339('2026-06-16 10:00:00', undefined);
    expect(out).toMatch(/(Z|\+00:00)$/);
  });

  // 失败路径
  it('null / 空返回 null', () => {
    expect(toSystemTimezoneRFC3339(null, 'Asia/Tokyo')).toBeNull();
    expect(toSystemTimezoneRFC3339('', 'Asia/Tokyo')).toBeNull();
  });

  it('非法钟面返回 null', () => {
    expect(toSystemTimezoneRFC3339('garbage', 'Asia/Tokyo')).toBeNull();
  });

  it('同一钟面在不同系统时区附加不同偏移（语义正确性）', () => {
    const tokyo = toSystemTimezoneRFC3339('2026-06-16 10:00:00', 'Asia/Tokyo');
    const shanghai = toSystemTimezoneRFC3339('2026-06-16 10:00:00', 'Asia/Shanghai');
    expect(tokyo).not.toBe(shanghai);
    expect(tokyo).toMatch(/\+09:00$/);
    expect(shanghai).toMatch(/\+08:00$/);
  });
});
