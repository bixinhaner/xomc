import type { NE } from '@/types/device';
import type { AlarmSeverity } from '@/types/common';
import { generateIP } from '../utils';

const neTypes = ['eNB', 'gNB', 'CPE', 'eGW'];
const vendors = ['华为', '中兴', '爱立信', '大唐', '京信'];
const connStatuses: Array<NE['connStatus']> = ['online', 'online', 'online', 'online', 'offline'];
const alarmLevels: Array<AlarmSeverity | 'none'> = ['none', 'none', 'none', 'warning', 'minor', 'major', 'critical'];

const cities = [
  { name: '北京', region: '华北', subnets: ['10.1.0.0/16', '10.2.0.0/16'] },
  { name: '上海', region: '华东', subnets: ['10.3.0.0/16', '10.4.0.0/16'] },
  { name: '广州', region: '华南', subnets: ['10.5.0.0/16', '10.6.0.0/16'] },
  { name: '深圳', region: '华南', subnets: ['10.7.0.0/16', '10.8.0.0/16'] },
  { name: '成都', region: '西南', subnets: ['10.9.0.0/16', '10.10.0.0/16'] },
  { name: '西安', region: '西北', subnets: ['10.11.0.0/16', '10.12.0.0/16'] },
  { name: '杭州', region: '华东', subnets: ['10.13.0.0/16', '10.14.0.0/16'] },
  { name: '武汉', region: '华中', subnets: ['10.15.0.0/16', '10.16.0.0/16'] },
  { name: '南京', region: '华东', subnets: ['10.17.0.0/16', '10.18.0.0/16'] },
  { name: '长沙', region: '华中', subnets: ['10.19.0.0/16', '10.20.0.0/16'] },
];

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function generateNE(index: number): NE {
  const city = cities[index % cities.length];
  const neType = neTypes[index % neTypes.length];
  const vendor = vendors[index % vendors.length];
  const cellNum = (index % 3) + 1;
  const siteNum = Math.floor(index / 3) + 1;

  const snPrefixes: Record<string, string> = {
    eNB: 'ENB',
    gNB: 'GNB',
    CPE: 'CPE',
    eGW: 'EGW',
  };
  const snPrefix = snPrefixes[neType] || 'DEV';
  const sn = `${snPrefix}${String(index + 1).padStart(6, '0')}`;

  return {
    id: `ne-${String(index + 1).padStart(6, '0')}`,
    neName: `${city.name}-${neType}-${String(siteNum).padStart(3, '0')}-Cell${cellNum}`,
    sn,
    neType,
    vendor,
    region: city.region,
    subnet: pickRandom(city.subnets),
    site: `${city.name}站点${String((index % 20) + 1).padStart(2, '0')}`,
    connStatus: pickRandom(connStatuses),
    alarmLevel: pickRandom(alarmLevels),
  };
}

export const mockNEs: NE[] = Array.from({ length: 500 }, (_, i) => generateNE(i));

// suppress unused import warning
void generateIP;
