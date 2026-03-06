import type { Device } from '@/types/device';
import type { AlarmSeverity } from '@/types/common';
import { generateIP, generateSN } from '../utils';

const vendors = ['华为', '中兴', '爱立信', '大唐', '京信'];
const productTypes = ['eNB', 'gNB', 'CPE', 'eGW'];
const networkTypes = ['LTE', 'NR', 'LTE/NR'];
const deviceModels: Record<string, string[]> = {
  eNB: ['BBU3910', 'DBS3900', 'BTS3900', 'eNodeB3900', 'ZXSDR B8200'],
  gNB: ['BBU5900', 'AAU5239', 'RRU5258', 'ZXRAN B8300', 'AIR6449'],
  CPE: ['CPE Pro 2', 'MC801A', 'H112-370', 'CPE B2351', 'FWA01'],
  eGW: ['EGW200', 'EGW500', 'GW-B1000', 'eGW3000', 'vEPC-200'],
};

const cities = [
  { name: '北京', region: '华北', lat: 39.9042, lng: 116.4074, subnets: ['10.1.0.0/16', '10.2.0.0/16'] },
  { name: '上海', region: '华东', lat: 31.2304, lng: 121.4737, subnets: ['10.3.0.0/16', '10.4.0.0/16'] },
  { name: '广州', region: '华南', lat: 23.1291, lng: 113.2644, subnets: ['10.5.0.0/16', '10.6.0.0/16'] },
  { name: '深圳', region: '华南', lat: 22.5431, lng: 114.0579, subnets: ['10.7.0.0/16', '10.8.0.0/16'] },
  { name: '成都', region: '西南', lat: 30.5728, lng: 104.0668, subnets: ['10.9.0.0/16', '10.10.0.0/16'] },
  { name: '西安', region: '西北', lat: 34.3416, lng: 108.9398, subnets: ['10.11.0.0/16', '10.12.0.0/16'] },
  { name: '杭州', region: '华东', lat: 30.2741, lng: 120.1551, subnets: ['10.13.0.0/16', '10.14.0.0/16'] },
  { name: '武汉', region: '华中', lat: 30.5928, lng: 114.3055, subnets: ['10.15.0.0/16', '10.16.0.0/16'] },
  { name: '南京', region: '华东', lat: 32.0603, lng: 118.7969, subnets: ['10.17.0.0/16', '10.18.0.0/16'] },
  { name: '长沙', region: '华中', lat: 28.2278, lng: 112.9388, subnets: ['10.19.0.0/16', '10.20.0.0/16'] },
];

const alarmLevels: Array<AlarmSeverity | 'none'> = ['none', 'none', 'none', 'warning', 'warning', 'minor', 'major', 'critical'];
const engStatuses: Array<Device['engStatus']> = ['commissioned', 'commissioned', 'commissioned', 'uncommissioned', 'decommissioned'];
const mgmtStatuses: Array<Device['mgmtStatus']> = ['managed', 'managed', 'managed', 'managed', 'unmanaged', 'pre-managed'];
const softwareVersions = ['V100R011C10SPC100', 'V100R011C10SPC200', 'V200R001C00SPC100', 'V200R001C10SPC100', 'V300R001C00'];

function randomOffset(base: number, range: number): number {
  return parseFloat((base + (Math.random() - 0.5) * range).toFixed(6));
}

function randomDate(daysAgo: number): string {
  const d = new Date(Date.now() - Math.random() * daysAgo * 86400000);
  return d.toISOString();
}

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function generateDevice(index: number): Device {
  const city = cities[index % cities.length];
  const type = productTypes[index % productTypes.length] as keyof typeof deviceModels;
  const vendor = vendors[index % vendors.length];
  const sn = generateSN(type) + String(index).padStart(3, '0');
  const isOnline = Math.random() < 0.85;
  const alarmLevel = isOnline ? pickRandom(alarmLevels) : 'none';
  const models = deviceModels[type];
  const model = models[Math.floor(Math.random() * models.length)];

  return {
    id: `dev-${String(index + 1).padStart(4, '0')}`,
    sn,
    name: `${city.name}-${type}-${String(index + 1).padStart(4, '0')}`,
    vendor,
    productType: type,
    networkType: type === 'gNB' ? 'NR' : type === 'eNB' ? 'LTE' : pickRandom(networkTypes),
    deviceModel: model,
    region: city.region,
    stationId: `ST${String(index + 1).padStart(6, '0')}`,
    connStatus: isOnline ? 'online' : 'offline',
    alarmLevel,
    engStatus: pickRandom(engStatuses),
    mgmtStatus: pickRandom(mgmtStatuses),
    lastOnlineTime: isOnline ? new Date().toISOString() : randomDate(30),
    ipAddress: generateIP(),
    subnet: pickRandom(city.subnets),
    site: `${city.name}站点${String((index % 20) + 1).padStart(2, '0')}`,
    longitude: randomOffset(city.lng, 0.5),
    latitude: randomOffset(city.lat, 0.5),
    softwareVersion: pickRandom(softwareVersions),
    createTime: randomDate(365),
  };
}

export const mockDevices: Device[] = Array.from({ length: 200 }, (_, i) => generateDevice(i));
