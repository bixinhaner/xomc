import { describe, expect, it } from 'vitest';
import type { Device, DeviceListResponse, LocationSync } from '@core/types/device';
import { applyLocationSyncResult, applyLocationSyncResultToList } from './deviceLocationSync';

describe('applyLocationSyncResult', () => {
  it('确认成功后立即更新坐标和 in_sync 状态', () => {
    const device = {
      id: 'device-1',
      longitude: 115.366462,
      latitude: 25.92416,
      locationSync: {
        status: 'pending',
        reported: { longitude: 115.376, latitude: 25.9341, observedAt: '', version: 3, sourcePath: 'Device.FAP.GPS' },
      },
    } as Device;
    const result = {
      status: 'in_sync',
      accepted: { longitude: 115.376, latitude: 25.9341 },
      reported: device.locationSync.reported,
      distanceMeters: 0,
      heightDiffMeters: null,
    } as LocationSync;

    expect(applyLocationSyncResult(device, result)).toMatchObject({
      longitude: 115.376,
      latitude: 25.9341,
      locationSync: { status: 'in_sync' },
    });
  });

  it('更新命中的列表设备并保留分页元数据', () => {
    const target = {
      id: 'device-1',
      sn: 'SN-1',
      longitude: 115.366462,
      latitude: 25.92416,
      locationSync: {
        status: 'pending',
        reported: { longitude: 115.376, latitude: 25.9341, observedAt: '', version: 3, sourcePath: 'Device.FAP.GPS' },
      },
    } as Device;
    const other = { ...target, id: 'device-2', sn: 'SN-2' };
    const list = {
      items: [target, other],
      total: 2,
      page: 1,
      pageSize: 20,
      stats: {},
    } as DeviceListResponse;
    const result = {
      status: 'in_sync',
      accepted: { longitude: 115.376, latitude: 25.9341 },
      reported: target.locationSync.reported,
      distanceMeters: 0,
      heightDiffMeters: null,
    } as LocationSync;

    const updated = applyLocationSyncResultToList(list, target.id, result);

    expect(updated).not.toBe(list);
    expect(updated?.items[0].locationSync.status).toBe('in_sync');
    expect(updated?.items[1]).toBe(other);
    expect(updated).toMatchObject({ total: 2, page: 1, pageSize: 20 });
  });
});
