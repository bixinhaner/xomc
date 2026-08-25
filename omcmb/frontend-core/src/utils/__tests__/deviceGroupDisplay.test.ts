import { describe, expect, it } from 'vitest';
import {
  buildDeviceGroupDisplayName,
  buildDeviceGroupPathName,
  withDeviceGroupDisplayName,
} from '../deviceGroupDisplay';
import type { Device, DeviceGroup } from '../../types/device';

function mkGroup(overrides: Partial<DeviceGroup>): DeviceGroup {
  return {
    id: 'g1',
    name: '默认设备组',
    parentId: null,
    deviceCount: 0,
    description: '',
    builtIn: 1,
    ...overrides,
  };
}

function mkDevice(overrides: Partial<Device>): Device {
  return {
    id: 'd1',
    sn: 'SN001',
    name: '设备1',
    deviceName: '设备1',
    networkType: 'gNB',
    productClass: '',
    connStatus: 'online',
    alarmLevel: 'none',
    engStatus: 'commissioned',
    mgmtStatus: 'managed',
    lastOnlineTime: '',
    ipAddress: '',
    subnet: '',
    site: '',
    longitude: null,
    latitude: null,
    softwareVersion: '',
    createTime: '',
    hostName: '',
    productName: '',
    firmwareVersion: '',
    macAddress: '',
    groupName: '',
    onlineTime: '',
    offlineTime: '',
    onlineDuration: null,
    upTime: null,
    cumulativeOnlineDuration: null,
    lastOfflineReason: null,
    firstOnlineTime: '',
    lastInformTime: '',
    gpsVersion: '',
    rom: '',
    remark: '',
    gnbId: '',
    enbId: '',
    cellId: '',
    eci: '',
    nrCellId: '',
    pci: '',
    plmnId: '',
    tac: '',
    subframeAssignment: '',
    specialSubframe: '',
    nrarfcnDl: '',
    bandwidth: '',
    txPower: '',
    rfStatus: '',
    adminState: '',
    opState: '',
    cellStatus: '',
    syncStatus: '',
    isOnline: true,
    ...overrides,
  } as Device;
}

describe('deviceGroupDisplay', () => {
  it('为同名父子组生成可区分的层级路径', () => {
    const groups = [
      mkGroup({ id: 'root', name: '默认设备组' }),
      mkGroup({ id: 'child', name: '默认设备组', parentId: 'root' }),
    ];

    expect(buildDeviceGroupPathName(groups[0], groups, 'zh-CN')).toBe('默认设备组');
    expect(buildDeviceGroupPathName(groups[1], groups, 'zh-CN')).toBe('默认设备组 / 默认设备组');
  });

  it('按 groupId 组装父子层级名称', () => {
    const groups = [
      mkGroup({ id: 'root', name: '默认设备组', nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' } }),
      mkGroup({ id: 'child', name: '默认设备组', parentId: 'root', nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' } }),
    ];
    const device = mkDevice({ groupId: 'child', groupName: '默认设备组' });

    expect(buildDeviceGroupDisplayName(device, groups, 'zh-CN')).toBe('默认设备组 / 默认设备组');
    expect(buildDeviceGroupDisplayName(device, groups, 'en-US')).toBe('Default Group / Default Group');
  });

  it('缺 groupId 时回退原始 groupName', () => {
    expect(buildDeviceGroupDisplayName(mkDevice({ groupName: '默认设备组' }), [], 'zh-CN')).toBe('默认设备组');
  });

  it('未分组设备无归属记录时回退到默认二级节点路径', () => {
    const groups = [
      mkGroup({ id: 'root', name: '默认设备组' }),
      mkGroup({ id: '00000000-0000-0000-0000-000000000002', name: '默认设备组', parentId: 'root' }),
    ];

    expect(buildDeviceGroupDisplayName(mkDevice({ groupId: undefined, groupName: '' }), groups, 'zh-CN')).toBe('默认设备组 / 默认设备组');
  });

  it('批量包装设备分组展示名', () => {
    const groups = [mkGroup({ id: 'root' }), mkGroup({ id: 'child', parentId: 'root' })];
    const out = withDeviceGroupDisplayName([mkDevice({ groupId: 'child', groupName: '默认设备组' })], groups, 'zh-CN');

    expect(out[0].groupName).toBe('默认设备组 / 默认设备组');
  });
});
