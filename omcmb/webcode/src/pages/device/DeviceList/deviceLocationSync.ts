import type { Device, DeviceListResponse, LocationSync } from '@core/types/device';

export function applyLocationSyncResult(device: Device, result: LocationSync): Device {
  const accepted = result.accepted ?? (result.status === 'in_sync' ? result.reported : null);
  return {
    ...device,
    longitude: accepted?.longitude ?? device.longitude,
    latitude: accepted?.latitude ?? device.latitude,
    locationSync: result,
  };
}

export function applyLocationSyncResultToList(
  list: DeviceListResponse | undefined,
  deviceId: string,
  result: LocationSync,
): DeviceListResponse | undefined {
  if (!list) return list;
  const index = list.items.findIndex((device) => device.id === deviceId);
  if (index < 0) return list;

  const items = [...list.items];
  items[index] = applyLocationSyncResult(items[index], result);
  return { ...list, items };
}
