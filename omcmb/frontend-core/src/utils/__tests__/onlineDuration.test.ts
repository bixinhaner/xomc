import { describe, expect, it } from 'vitest';

import { computeCumulativeOnlineDurationSeconds, computeCurrentOnlineDurationSeconds } from '../onlineDuration';

describe('computeCurrentOnlineDurationSeconds', () => {
  it('在线时按 now - onlineTime 计算', () => {
    expect(computeCurrentOnlineDurationSeconds({
      isOnline: true,
      onlineTime: '2026-07-06T00:00:00.000Z',
      offlineTime: null,
      fallbackOnlineDuration: 10,
      nowMs: Date.parse('2026-07-06T01:00:00.000Z'),
    })).toBe(3600);
  });

  it('离线时按 offlineTime - onlineTime 计算', () => {
    expect(computeCurrentOnlineDurationSeconds({
      isOnline: false,
      onlineTime: '2026-07-06T00:00:00.000Z',
      offlineTime: '2026-07-06T00:30:00.000Z',
      fallbackOnlineDuration: null,
    })).toBe(1800);
  });

  it('时间戳缺失时回退 fallbackOnlineDuration', () => {
    expect(computeCurrentOnlineDurationSeconds({
      isOnline: true,
      onlineTime: null,
      offlineTime: null,
      fallbackOnlineDuration: 77,
    })).toBe(77);
  });
});

describe('computeCumulativeOnlineDurationSeconds', () => {
  it('在线时累计 = 已持久累计 + 当前在线段', () => {
    expect(computeCumulativeOnlineDurationSeconds({
      isOnline: true,
      onlineTime: '2026-07-06T00:00:00.000Z',
      offlineTime: null,
      fallbackOnlineDuration: null,
      cumulativeOnlineDuration: 120,
      nowMs: Date.parse('2026-07-06T00:10:00.000Z'),
    })).toBe(720);
  });

  it('离线时仅显示持久累计', () => {
    expect(computeCumulativeOnlineDurationSeconds({
      isOnline: false,
      onlineTime: '2026-07-06T00:00:00.000Z',
      offlineTime: '2026-07-06T00:10:00.000Z',
      fallbackOnlineDuration: null,
      cumulativeOnlineDuration: 120,
    })).toBe(120);
  });
});
