import { describe, it, expect } from 'vitest';
import {
  parseCellId,
  parsePlmn,
  formatObjectLdn,
  deviceSnTail,
  buildDeviceSeriesName,
} from '../pmObject';

describe('pmObject — object_ldn 拆解', () => {
  it('parseCellId 取 Cellid 段（大小写不敏感）', () => {
    expect(parseCellId('Cellid=111172245,PLMN=46068')).toBe('111172245');
    expect(parseCellId('CELLID=999,PLMN=1')).toBe('999');
    expect(parseCellId('PLMN=46068')).toBeUndefined();
    expect(parseCellId('')).toBeUndefined();
    expect(parseCellId(null)).toBeUndefined();
  });

  it('parsePlmn 取 PLMN 段', () => {
    expect(parsePlmn('Cellid=111172245,PLMN=46068')).toBe('46068');
    expect(parsePlmn('Cellid=999')).toBeUndefined();
    expect(parsePlmn(undefined)).toBeUndefined();
  });
});

describe('pmObject — formatObjectLdn 友好名', () => {
  it('两段都有 → 「小区X · PLMNY」', () => {
    expect(formatObjectLdn('Cellid=111172245,PLMN=46068')).toBe('小区111172245 · PLMN46068');
  });

  it('只有小区 → 「小区X」', () => {
    expect(formatObjectLdn('Cellid=111172245')).toBe('小区111172245');
  });

  it('只有 PLMN → 「PLMNY」', () => {
    expect(formatObjectLdn('PLMN=46068')).toBe('PLMN46068');
  });

  it('无法拆解 → 回退原串', () => {
    expect(formatObjectLdn('SomeOtherLdn')).toBe('SomeOtherLdn');
  });

  it('空 / null → 空串', () => {
    expect(formatObjectLdn('')).toBe('');
    expect(formatObjectLdn(null)).toBe('');
    expect(formatObjectLdn(undefined)).toBe('');
  });
});

describe('pmObject — deviceSnTail / buildDeviceSeriesName', () => {
  it('deviceSnTail 取末 6 位，短于 6 全量', () => {
    expect(deviceSnTail('1202000240194DP0015')).toBe('DP0015'); // 末6位
    expect(deviceSnTail('SN-A')).toBe('SN-A');
    expect(deviceSnTail('')).toBe('');
    expect(deviceSnTail(null)).toBe('');
  });

  it('buildDeviceSeriesName 有小区 → 尾号 · 友好名', () => {
    expect(buildDeviceSeriesName('SN-AAAAAA', 'Cellid=111,PLMN=222')).toBe(
      'AAAAAA · 小区111 · PLMN222',
    );
  });

  it('buildDeviceSeriesName 无小区 → 仅尾号（兜底单线）', () => {
    expect(buildDeviceSeriesName('SN-AAAAAA', null)).toBe('AAAAAA');
    expect(buildDeviceSeriesName('SN-AAAAAA', '')).toBe('AAAAAA');
  });
});
