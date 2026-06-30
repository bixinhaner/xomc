import { describe, expect, it } from 'vitest';

import {
  DEVICE_SYNC_STATUS_LABELS_EN,
  DEVICE_SYNC_STATUS_LABELS_ZH,
  formatDeviceSyncStatus,
  getDeviceSyncStatusKind,
  normalizeDeviceSyncStatus,
} from '../deviceSyncStatus';

describe('deviceSyncStatus', () => {
  it('归一化历史别名与后端规范值，避免列表/详情分叉', () => {
    expect(normalizeDeviceSyncStatus('gps')).toBe('gps');
    expect(normalizeDeviceSyncStatus('GPS synchronized')).toBe('gps');
    expect(normalizeDeviceSyncStatus('1588 synchronized')).toBe('ntp');
    expect(normalizeDeviceSyncStatus('not_synchronized')).toBe('error');
  });

  it('保留后端透传的未知文本态，避免误折叠原始状态', () => {
    expect(normalizeDeviceSyncStatus('HOLDOVER')).toBe('HOLDOVER');
    expect(getDeviceSyncStatusKind('HOLDOVER')).toBeNull();
  });

  it('按皮肤需要输出统一文案', () => {
    expect(formatDeviceSyncStatus('beidou', DEVICE_SYNC_STATUS_LABELS_ZH)).toBe('北斗已同步');
    expect(formatDeviceSyncStatus('not synchronized', DEVICE_SYNC_STATUS_LABELS_EN)).toBe('Not synchronized');
  });
});