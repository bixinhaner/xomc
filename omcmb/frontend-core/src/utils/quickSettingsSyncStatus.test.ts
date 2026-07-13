import { describe, expect, it } from 'vitest';
import type { QuickSettingsSyncMonitor } from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSyncStatus } from '@core/types/deviceParameter';
import { isCurrentSyncFailure, isCurrentSyncSuccess } from './quickSettingsSyncStatus';

describe('isCurrentSyncSuccess', () => {
  it('does not accept another source completion while the current sync is still running', () => {
    const sync: QuickSettingsSyncMonitor = {
      sourceId: 'manual:quick-settings-a',
      lastParamSyncAt: '2026-07-13T01:00:00.000Z',
      targetCount: 3,
      gpvTaskCount: 1,
      startedAt: Date.parse('2026-07-13T01:01:00.000Z'),
    };
    const status: ParameterSyncStatus = {
      deviceId: 'device-1',
      status: 'syncing',
      totalParameters: 10,
      pendingCommands: 1,
      lastSyncGpv: {
        sourceId: 'manual:another-sync',
        taskCount: 1,
        lastCompletedAt: '2026-07-13T01:01:01.000Z',
      },
      lastParamSyncAt: '2026-07-13T01:01:01.000Z',
    };

    expect(isCurrentSyncSuccess(status, sync)).toBe(false);
  });
});

describe('isCurrentSyncFailure', () => {
  it('does not accept an unattributed failure while commands are still running', () => {
    const sync: QuickSettingsSyncMonitor = {
      sourceId: 'manual:quick-settings-a',
      lastParamSyncFailedAt: '2026-07-13T01:00:00.000Z',
      targetCount: 3,
      gpvTaskCount: 1,
      startedAt: Date.parse('2026-07-13T01:01:00.000Z'),
    };
    const status: ParameterSyncStatus = {
      deviceId: 'device-1',
      status: 'syncing',
      totalParameters: 10,
      pendingCommands: 1,
      lastParamSyncFailedAt: '2026-07-13T01:01:01.000Z',
      lastParamSyncError: 'another sync failed',
    };

    expect(isCurrentSyncFailure(status, sync)).toBe(false);
  });

  it('accepts a new failure after the device leaves syncing state', () => {
    const sync: QuickSettingsSyncMonitor = {
      sourceId: 'manual:quick-settings-a',
      lastParamSyncFailedAt: '2026-07-13T01:00:00.000Z',
      targetCount: 3,
      gpvTaskCount: 1,
      startedAt: Date.parse('2026-07-13T01:01:00.000Z'),
    };
    const status: ParameterSyncStatus = {
      deviceId: 'device-1',
      status: 'idle',
      totalParameters: 10,
      pendingCommands: 0,
      lastParamSyncFailedAt: '2026-07-13T01:01:01.000Z',
    };

    expect(isCurrentSyncFailure(status, sync)).toBe(true);
  });
});
