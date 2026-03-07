import type { AlarmSeverity } from './common';

export type ConnStatus = 'online' | 'offline';

export type EngStatus = 'commissioned' | 'uncommissioned' | 'decommissioned';

export type MgmtStatus = 'managed' | 'unmanaged' | 'pre-managed';

export interface Device {
  id: string;
  sn: string;
  name: string;
  vendor: string;
  productType: string;
  networkType: string;
  deviceModel: string;
  region: string;
  stationId: string;
  connStatus: ConnStatus;
  alarmLevel: AlarmSeverity | 'none';
  engStatus: EngStatus;
  mgmtStatus: MgmtStatus;
  lastOnlineTime: string;
  ipAddress: string;
  subnet: string;
  site: string;
  longitude: number;
  latitude: number;
  softwareVersion: string;
  createTime: string;
}

export interface NE {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  vendor: string;
  region: string;
  subnet: string;
  site: string;
  connStatus: ConnStatus;
  alarmLevel: AlarmSeverity | 'none';
}

export interface DeviceGroup {
  id: string;
  name: string;
  parentId: string | null;
  deviceCount: number;
  description: string;
}

export interface DeviceFilter {
  name?: string;
  sn?: string;
  vendor?: string;
  productType?: string;
  networkType?: string;
  connStatus?: ConnStatus;
  alarmLevel?: AlarmSeverity | 'none';
  region?: string;
  subnet?: string;
  engStatus?: EngStatus;
}
