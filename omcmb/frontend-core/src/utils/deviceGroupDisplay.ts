import type { Device, DeviceGroup } from '../types/device';
import { getRecordI18n, type Locale } from './i18nText';
import { UNASSIGNED_GROUP_ID } from './deviceGroupTargets';

function getGroupDisplayName(group: DeviceGroup, locale: Locale): string {
  return getRecordI18n(group as unknown as Record<string, unknown>, 'name', locale) || group.name;
}

export function buildDeviceGroupPathName(
  group: DeviceGroup,
  groups: DeviceGroup[],
  locale: Locale,
): string {
  const childName = getGroupDisplayName(group, locale);
  if (!group.parentId) return childName;
  const parent = groups.find((item) => item.id === group.parentId);
  if (!parent) return childName;
  const parentName = getGroupDisplayName(parent, locale);
  return parentName ? `${parentName} / ${childName}` : childName;
}

export function buildDeviceGroupDisplayName(
  device: Pick<Device, 'groupId' | 'groupName'>,
  groups: DeviceGroup[],
  locale: Locale,
): string {
  const group = device.groupId
    ? groups.find((item) => item.id === device.groupId)
    : (!device.groupName ? groups.find((item) => item.id === UNASSIGNED_GROUP_ID) : undefined);
  if (!group) return device.groupName || '';
  return buildDeviceGroupPathName(group, groups, locale);
}

export function withDeviceGroupDisplayName<T extends Pick<Device, 'groupId' | 'groupName'>>(
  devices: T[],
  groups: DeviceGroup[],
  locale: Locale,
): T[] {
  return devices.map((device) => ({
    ...device,
    groupName: buildDeviceGroupDisplayName(device, groups, locale),
  }));
}
