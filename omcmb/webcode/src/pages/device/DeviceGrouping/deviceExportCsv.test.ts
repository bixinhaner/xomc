// @vitest-environment node

import { describe, expect, it } from 'vitest';
import type { Device, DeviceGroup } from '@core/types/device';
import { buildCsvForDeviceList, normalizeDevicesForExport } from './deviceExportCsv';

const enMessages: Record<string, string> = {
  'device.csv.column.sn': 'SN',
  'device.csv.column.deviceName': 'Device Name',
  'device.csv.column.connStatus': 'Connection Status',
  'device.csv.column.macAddress': 'MAC Address',
  'device.csv.column.deviceGrouping': 'Device Group',
  'device.csv.column.sourceType': 'Source',
  'device.csv.column.remark': 'Remark',
  'status.online': 'Online',
  'status.offline': 'Offline',
  'device.sourceType.manual': 'Manual',
};

function t(id: string): string {
  return enMessages[id] ?? id;
}

function mkGroup(overrides: Partial<DeviceGroup>): DeviceGroup {
  return {
    id: 'group',
    name: '默认设备组',
    parentId: null,
    deviceCount: 0,
    description: '',
    builtIn: 1,
    ...overrides,
  };
}

describe('deviceExportCsv', () => {
  it('英文环境导出英文表头，并补齐按 groupId 解析出的设备分组列', () => {
    const groups = [
      mkGroup({
        id: 'root',
        name: '默认设备组',
        nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
      }),
      mkGroup({
        id: 'child',
        name: '默认设备组',
        parentId: 'root',
        nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
      }),
    ];
    const device = {
      sn: 'SN001',
      name: 'Device A',
      connStatus: 'online',
      macAddress: '48:BF:74:24:12:44',
      groupId: 'child',
      groupName: '',
      sourceType: 'manual',
      remark: 'Remark A',
    } as Device;

    const rows = normalizeDevicesForExport([device], groups, 'en-US');
    const csv = buildCsvForDeviceList(rows, t).replace(/^\uFEFF/, '').trimEnd();

    expect(csv.split('\n')).toEqual([
      'SN,Device Name,Connection Status,MAC Address,Device Group,Source,Remark',
      'SN001,Device A,Online,48:BF:74:24:12:44,Default Group / Default Group,Manual,Remark A',
    ]);
  });
});
