import { describe, expect, it } from 'vitest';
import { shouldShowLocationSyncIndicator } from './deviceGpsSyncIndicator';

describe('shouldShowLocationSyncIndicator', () => {
  const reported = {
    longitude: 115.366198,
    latitude: 25.92416,
    observedAt: '2026-07-17T10:00:00Z',
    version: 3,
    sourcePath: 'Device.DeviceInfo.SAS.FAP.GPS',
  };

  it('仅待确认差异显示提示', () => {
    expect(shouldShowLocationSyncIndicator({ status: 'pending', reported })).toBe(true);
  });

  it('确认后、无上报和初始化状态均不显示提示', () => {
    expect(shouldShowLocationSyncIndicator({ status: 'in_sync', reported })).toBe(false);
    expect(shouldShowLocationSyncIndicator({ status: 'no_report', reported: null })).toBe(false);
    expect(shouldShowLocationSyncIndicator({ status: 'initialized', reported })).toBe(false);
    expect(shouldShowLocationSyncIndicator({ status: 'pending', reported: null })).toBe(false);
  });
});
