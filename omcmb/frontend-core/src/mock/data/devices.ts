import type { Device } from '../../types/device';
import type { AlarmSeverity } from '../../types/common';
import { generateIP, generateSN } from '../utils';

const vendors = ['华为', '中兴', '爱立信', '大唐', '京信'];
const networkTypes: Array<'eNB' | 'gNB' | 'GSM'> = ['eNB', 'gNB', 'GSM'];

// 产品类型标识 — 与原始 JSP product 字段对应
const productModels: Record<string, string[]> = {
  eNB: ['PM-B4860', 'QAFA', 'QATA', 'QAFB'],
  gNB: ['BaiBNX', 'BaiBNQ'],
  GSM: ['BSC', 'BTS'],
};

const deviceModels: Record<string, string[]> = {
  eNB: ['BBU3910', 'DBS3900', 'BTS3900', 'eNodeB3900', 'ZXSDR B8200'],
  gNB: ['BBU5900', 'AAU5239', 'RRU5258', 'ZXRAN B8300', 'AIR6449'],
  GSM: ['BTS3012', 'RBS2206', 'ZXSDR BS8700', 'BTS3900G', 'Flexi BTS'],
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
const firmwareVersions = ['HW-V1.0.0', 'HW-V1.1.0', 'HW-V2.0.0', 'ZTE-V3.0.0', 'ER-V4.0.0'];
const groupNames = ['华北大区', '华东大区', '华南大区', '西南大区', '西北大区', '东北大区', '5G gNB设备'];
const opStates = ['active', 'inactive'];
const rfStatuses = ['on', 'off', ''];
const syncStatuses = ['synchronized', 'not synchronized', ''];
const bands = ['Band 1', 'Band 3', 'Band 5', 'Band 7', 'Band 38', 'Band 40', 'Band 41', 'n41', 'n78', 'n79'];

const cityGroupIDs: Record<string, string> = {
  北京: 'grp-bj',
  上海: 'grp-sh',
  南京: 'grp-nj',
  杭州: 'grp-hz',
  广州: 'grp-gz',
  深圳: 'grp-sz',
  成都: 'grp-cd',
  西安: 'grp-xa',
};

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

function generateMAC(): string {
  const hex = () => Math.floor(Math.random() * 256).toString(16).padStart(2, '0').toUpperCase();
  return `${hex()}:${hex()}:${hex()}:${hex()}:${hex()}:${hex()}`;
}

function generateDevice(index: number): Device {
  const city = cities[index % cities.length];
  const groupId = cityGroupIDs[city.name] ?? 'grp-default';
  const type = networkTypes[index % networkTypes.length];
  const vendor = vendors[index % vendors.length];
  const sn = generateSN(type) + String(index).padStart(3, '0');
  const isOnline = Math.random() < 0.85;
  const alarmLevel = isOnline ? pickRandom(alarmLevels) : 'none';
  const models = deviceModels[type];
  const model = models[Math.floor(Math.random() * models.length)];
  const isNR = type === 'gNB';
  const isLTE = type === 'eNB';
  const productModel = pickRandom(productModels[type]);
  const enbIdVal = isLTE ? String(10000 + index) : '';
  const gnbIdVal = isNR ? String(50000 + index) : '';
  const pciVal = String(Math.floor(Math.random() * 504));
  const earfcn = String(Math.floor(Math.random() * 65535));
  const longitude = randomOffset(city.lng, 0.5);
  const latitude = randomOffset(city.lat, 0.5);
  const mmePool = isLTE
    ? Array.from({ length: 2 + (index % 3) }, (_, mmeIndex) => ({
        index: mmeIndex + 1,
        ip: `10.${20 + mmeIndex}.${index % 255}.${mmeIndex + 1}`,
        status: mmeIndex === 0 && index % 4 !== 3 ? 'active' : 'inactive',
        plmnId: '46000',
      }))
    : [];

  return {
    id: `dev-${String(index + 1).padStart(4, '0')}`,
    sn,
    name: `${city.name}-${type}-${String(index + 1).padStart(4, '0')}`,
    vendor,
    productClass: productModel,
    networkType: type,
    deviceModel: model,
    region: city.region,
    stationId: `ST${String(index + 1).padStart(6, '0')}`,
    connStatus: isOnline ? 'online' : 'offline',
    isOnline,
    lifecycleState: isOnline ? 'commissioned' : 'decommissioned',
    alarmLevel,
    engStatus: pickRandom(engStatuses),
    mgmtStatus: pickRandom(mgmtStatuses),
    lastOnlineTime: isOnline ? new Date().toISOString() : randomDate(30),
    ipAddress: generateIP(),
    subnet: pickRandom(city.subnets),
    site: `${city.name}站点${String((index % 20) + 1).padStart(2, '0')}`,
    longitude,
    latitude,
    locationSync: {
      status: 'initialized',
      accepted: { longitude, latitude },
      reported: null,
      distanceMeters: null,
      heightDiffMeters: null,
    },
    softwareVersion: pickRandom(softwareVersions),
    createTime: randomDate(365),
    oui: `${vendors[0]}-OUI`,
    carrier: pickRandom(['cmcc', 'ctcc', 'cucc']),

    // 监控扩展字段
    hostName: `${type}-${city.name}-${String(index + 1).padStart(3, '0')}`,
    productName: `${vendor} ${model}`,
    firmwareVersion: pickRandom(firmwareVersions),
    macAddress: generateMAC(),
    groupId,
    groupName: city.name,
    onlineTime: isOnline ? randomDate(7) : '',
    offlineTime: isOnline ? '' : randomDate(3),
    onlineDuration: isOnline ? Math.floor(Math.random() * 864000) : 0,
    upTime: isOnline ? Math.floor(Math.random() * 720 * 3600) : 0,
    // T-0173: 累计在线时长（OMC 视角）mock 取 1~30 天范围,验证 Detail 展示。
    cumulativeOnlineDuration: Math.floor(Math.random() * 30 * 86400),
    // T-0173: 离线原因 mock,在线设备 null,离线随机选三种之一。
    lastOfflineReason: isOnline ? null : pickRandom(['heartbeat_timeout', 'manual', 'reboot']),
    firstOnlineTime: randomDate(365),
    lastInformTime: isOnline ? randomDate(1) : randomDate(30),
    deviceName: `${city.name}站点${String((index % 20) + 1).padStart(2, '0')}`,
    gpsVersion: isLTE ? 'GPS V2.0' : '',
    rom: isLTE ? `ROM-${Math.floor(Math.random() * 100)}` : '',
    remark: index % 5 === 0 ? `备注-${index}` : '',
    gnbId: gnbIdVal,

    // Issue #758: 设备名称同步
    nameSyncPending: index % 10 === 0, // 每 10 个设备模拟一个待同步
    lmtDeviceName: index % 10 === 0 ? `LMT-${city.name}-${index}` : '',
    paramSyncRunning: index % 17 === 0,

    enbId: enbIdVal,
    cellId: String(Math.floor(Math.random() * 256)),
    eci: isLTE ? `${enbIdVal}${String(Math.floor(Math.random() * 256)).padStart(2, '0')}` : '',
    nrCellId: isNR ? `${gnbIdVal}-${Math.floor(Math.random() * 16)}` : '',
    pci: pciVal,
    plmnId: '46000',
    tac: String(Math.floor(Math.random() * 65535)),
    subframeAssignment: isLTE ? String(Math.floor(Math.random() * 7)) : '',
    specialSubframe: isLTE ? String(Math.floor(Math.random() * 10)) : '',
    rootIndex: isLTE ? String(Math.floor(Math.random() * 838)) : '',
    siteId: `SITE-${String(index + 1).padStart(6, '0')}`,
    bandwidth: isLTE ? `${pickRandom([5, 10, 15, 20])}MHz` : isNR ? `${pickRandom([20, 40, 60, 80, 100])}MHz` : '',
    dlEarfcn: earfcn,
    ulEarfcn: isNR ? String(parseInt(earfcn) + 18000) : '',
    networkModel: isLTE ? pickRandom(['TDD', 'FDD']) : isNR ? pickRandom(['SA', 'NSA']) : '',
    txPower: `${Math.floor(Math.random() * 46)}dBm`,
    band: pickRandom(bands),
    lac: isLTE ? '' : String(Math.floor(Math.random() * 65535)),
    arfcn: earfcn,
    uplinkFrequency: `${(1920 + Math.random() * 100).toFixed(1)}MHz`,
    downlinkFrequency: `${(2110 + Math.random() * 100).toFixed(1)}MHz`,

    cellStatus: isOnline ? '正常' : '离线',
    opState: (() => {
      // 多小区场景: 70% 概率生成多小区值 "1,0,1" / "active,inactive"
      if ((isLTE || isNR) && Math.random() < 0.7) {
        const cellCount = Math.floor(Math.random() * 3) + 2; // 2-4 cells
        return Array.from({ length: cellCount }, () => pickRandom(['1', '0'])).join(',');
      }
      return pickRandom(opStates);
    })(),
    controlSummary: index === 0 ? {
      sourceType: 'geofence',
      sourceId: 'mock-geofence-1',
      sourceName: '北京测试围栏',
      reasonCode: 'confirmed_exit',
      phase: 'deactivated',
      actionId: 'mock-control-action-1',
      triggeredAt: '2026-08-13T10:15:00+08:00',
      completedAt: '2026-08-13T10:15:08+08:00',
    } : null,
    mmeStatus: isLTE && mmePool.some((entry) => entry.status === 'active') ? 'connected' : isLTE ? 'disconnected' : '',
    mmePool,
    amfStatus: (() => {
      if (!isNR) return '';
      // 多 AMF 场景: 70% JSON 数组, 15% 旧格式, 15% 空
      const r = Math.random();
      if (r < 0.7) {
        const count = Math.floor(Math.random() * 3) + 2; // 2-4 AMFs
        return JSON.stringify(Array.from({ length: count }, (_, i) => ({
          AmfIP1: `10.${30 + i}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
          Status: Math.random() > 0.3 ? '1' : '0',
          PLMNID: '46000',
        })));
      }
      if (r < 0.85) return pickRandom(['1', '0']);
      return '';
    })(),
    rfStatus: (() => {
      // 多小区场景: 70% 概率生成多小区值 "on,off,on"
      if ((isLTE || isNR) && Math.random() < 0.7) {
        const cellCount = Math.floor(Math.random() * 3) + 2; // 2-4 cells
        return Array.from({ length: cellCount }, () => pickRandom(['on', 'off'])).join(',');
      }
      return pickRandom(rfStatuses);
    })(),
    pmReportStatus: isLTE ? pickRandom(['off', 'normal', 'broken', '']) : '',
    halobFlag: (isNR || isLTE) ? Math.random() > 0.5 : false,
    syncStatus: pickRandom(syncStatuses),
    validity: isLTE && Math.random() > 0.8 ? randomDate(-90) : '',
    lockStatus: Math.random() > 0.9 ? 'locked' : '',
    ueCount: isOnline ? Math.floor(Math.random() * 200) : 0,
    euCount: isNR ? (() => {
      const total = Math.floor(Math.random() * 4) + 1;
      const conn = Math.floor(Math.random() * (total + 1));
      return `${conn}/${total}`;
    })() : '',
    ruCount: isNR ? (() => {
      const total = Math.floor(Math.random() * 8) + 1;
      const conn = Math.floor(Math.random() * (total + 1));
      return `${conn}/${total}`;
    })() : '',
    cpeCount: isLTE ? Math.floor(Math.random() * 32) : 0,
    wanSpeed: isLTE ? `${Math.floor(Math.random() * 1000)}Mbps` : '',
    serviceStatus: '',
    adminState: isNR ? pickRandom(['1', '2', '3', '2', '2']) : '',
    multiPlmnEnable: isNR ? pickRandom(['true', 'false', '']) : '',
    bscLinkStatus: '',
    bscSelect: '',
    bscSerialNumber: '',
    btsNum: 0,
    wanLinkStatus: '',
    omcStatus: '',
    bsic: '',
    vswr: '',

    ipsecAddr: Math.random() > 0.7 ? generateIP() : '',
    mmepoolIpsecAddr: isLTE && Math.random() > 0.8 ? generateIP() : '',
    ipaUnitId: '',
    omlRemoteIp: '',
    omlRemoteIpBak: '',

    gpsHeight: parseFloat((Math.random() * 500).toFixed(1)),
    mechanicalDowntilt: isLTE ? `${Math.floor(Math.random() * 15)}°` : '',
    electronicDowntilt: isLTE ? `${Math.floor(Math.random() * 10)}°` : '',
    verticalBeamWidth: isLTE ? `${(5 + Math.random() * 10).toFixed(1)}°` : '',
    horizontalAzimuth: isLTE ? `${Math.floor(Math.random() * 360)}°` : '',
    installAddress: index % 10 === 0 ? `${city.name}市某区某路${index}号` : '',
    gpsSatelliteCount: Math.floor(Math.random() * 16),

    rollbackVersion: isNR && Math.random() > 0.7 ? pickRandom(softwareVersions) : '',
    sasParam: isNR && Math.random() > 0.8 ? 'SAS-Config-1' : '',
    euRu: isNR ? `${Math.floor(Math.random() * 4)}/${Math.floor(Math.random() * 8)}` : '',
    halobLicense: isNR && Math.random() > 0.5 ? 'valid' : '',
    energySaving: isNR ? pickRandom(['enabled', 'disabled', '']) : '',
    gnbTopoCellmgr: isNR ? `TOPO-${index}` : '',
    sslCertValidity: isNR && Math.random() > 0.5 ? randomDate(-365) : '',
  };
}

export const mockDevices: Device[] = Array.from({ length: 200 }, (_, i) => generateDevice(i));

// 为北京节点额外生成 50 条设备数据
const beijingCity = cities.find(c => c.name === '北京')!;
const beijingDevices: Device[] = Array.from({ length: 50 }, (_, i) => {
  const index = 200 + i;  // 从 200 开始，避免 ID 冲突
  const type = networkTypes[i % networkTypes.length];
  const vendor = vendors[i % vendors.length];
  const sn = generateSN(type) + String(index).padStart(3, '0');
  const isOnline = Math.random() < 0.85;
  const alarmLevel = isOnline ? pickRandom(alarmLevels) : 'none';
  const models = deviceModels[type];
  const model = models[Math.floor(Math.random() * models.length)];
  const isNR = type === 'gNB';
  const isLTE = type === 'eNB';
  const productModel = pickRandom(productModels[type]);
  const enbIdVal = isLTE ? String(10000 + index) : '';
  const gnbIdVal = isNR ? String(50000 + index) : '';
  const pciVal = String(Math.floor(Math.random() * 504));
  const earfcn = String(Math.floor(Math.random() * 65535));
  const longitude = randomOffset(beijingCity.lng, 0.5);
  const latitude = randomOffset(beijingCity.lat, 0.5);

  return {
    id: `dev-${String(index + 1).padStart(4, '0')}`,
    sn,
    name: `${beijingCity.name}-${type}-${String(index + 1).padStart(4, '0')}`,
    vendor,
    productClass: productModel,
    networkType: type,
    deviceModel: model,
    region: beijingCity.region,
    stationId: `ST${String(index + 1).padStart(6, '0')}`,
    connStatus: isOnline ? 'online' : 'offline',
    isOnline,
    lifecycleState: isOnline ? 'commissioned' : 'decommissioned',
    alarmLevel,
    engStatus: pickRandom(engStatuses),
    mgmtStatus: pickRandom(mgmtStatuses),
    lastOnlineTime: isOnline ? new Date().toISOString() : randomDate(30),
    ipAddress: generateIP(),
    subnet: pickRandom(beijingCity.subnets),
    site: `${beijingCity.name}站点${String((i % 20) + 1).padStart(2, '0')}`,
    longitude,
    latitude,
    locationSync: {
      status: 'initialized',
      accepted: { longitude, latitude },
      reported: null,
      distanceMeters: null,
      heightDiffMeters: null,
    },
    softwareVersion: pickRandom(softwareVersions),
    createTime: randomDate(365),
    oui: `${vendors[0]}-OUI`,
    carrier: pickRandom(['cmcc', 'ctcc', 'cucc']),

    // 监控扩展字段
    hostName: `${type}-${beijingCity.name}-${String(index + 1).padStart(3, '0')}`,
    productName: `${vendor} ${model}`,
    firmwareVersion: pickRandom(firmwareVersions),
    macAddress: generateMAC(),
    groupName: pickRandom(groupNames),
    onlineTime: isOnline ? randomDate(7) : '',
    offlineTime: isOnline ? '' : randomDate(3),
    onlineDuration: isOnline ? Math.floor(Math.random() * 864000) : 0,
    upTime: isOnline ? Math.floor(Math.random() * 720 * 3600) : 0,
    // T-0173: 累计在线时长（OMC 视角）mock 取 1~30 天范围。
    cumulativeOnlineDuration: Math.floor(Math.random() * 30 * 86400),
    // T-0173: 离线原因 mock。
    lastOfflineReason: isOnline ? null : pickRandom(['heartbeat_timeout', 'manual', 'reboot']),
    firstOnlineTime: randomDate(365),
    lastInformTime: isOnline ? randomDate(1) : randomDate(30),
    deviceName: `${beijingCity.name}站点${String((i % 20) + 1).padStart(2, '0')}`,
    gpsVersion: isLTE ? 'GPS V2.0' : '',
    rom: isLTE ? `ROM-${Math.floor(Math.random() * 100)}` : '',
    remark: i % 5 === 0 ? `备注-北京-${i}` : '',
    gnbId: gnbIdVal,

    // Issue #758: 设备名称同步
    nameSyncPending: i % 10 === 0,
    lmtDeviceName: i % 10 === 0 ? `LMT-北京-${i}` : '',
    paramSyncRunning: i % 17 === 0,

    enbId: enbIdVal,
    cellId: String(Math.floor(Math.random() * 256)),
    eci: isLTE ? `${enbIdVal}${String(Math.floor(Math.random() * 256)).padStart(2, '0')}` : '',
    nrCellId: isNR ? `${gnbIdVal}-${Math.floor(Math.random() * 16)}` : '',
    pci: pciVal,
    plmnId: '46000',
    tac: String(Math.floor(Math.random() * 65535)),
    subframeAssignment: isLTE ? String(Math.floor(Math.random() * 7)) : '',
    specialSubframe: isLTE ? String(Math.floor(Math.random() * 10)) : '',
    rootIndex: isLTE ? String(Math.floor(Math.random() * 838)) : '',
    siteId: `SITE-${String(index + 1).padStart(6, '0')}`,
    bandwidth: isLTE ? `${pickRandom([5, 10, 15, 20])}MHz` : isNR ? `${pickRandom([20, 40, 60, 80, 100])}MHz` : '',
    dlEarfcn: earfcn,
    ulEarfcn: isNR ? String(parseInt(earfcn) + 18000) : '',
    networkModel: isLTE ? pickRandom(['TDD', 'FDD']) : isNR ? pickRandom(['SA', 'NSA']) : '',
    txPower: `${Math.floor(Math.random() * 46)}dBm`,
    band: pickRandom(bands),
    lac: isLTE ? '' : String(Math.floor(Math.random() * 65535)),
    arfcn: earfcn,
    uplinkFrequency: `${(1920 + Math.random() * 100).toFixed(1)}MHz`,
    downlinkFrequency: `${(2110 + Math.random() * 100).toFixed(1)}MHz`,

    cellStatus: isOnline ? '正常' : '离线',
    opState: (() => {
      if ((isLTE || isNR) && Math.random() < 0.7) {
        const cellCount = Math.floor(Math.random() * 3) + 2;
        return Array.from({ length: cellCount }, () => pickRandom(['1', '0'])).join(',');
      }
      return pickRandom(opStates);
    })(),
    mmeStatus: (() => {
      if (!isLTE) return '';
      const r = Math.random();
      if (r < 0.7) {
        const count = Math.floor(Math.random() * 3) + 2;
        return JSON.stringify(Array.from({ length: count }, (_, i) => ({
          mmeIp: `10.${20 + i}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
          status: Math.random() > 0.3 ? '1' : '0',
          plmnId: '46000',
        })));
      }
      if (r < 0.85) return pickRandom(['1', '0']);
      return '';
    })(),
    amfStatus: (() => {
      if (!isNR) return '';
      const r = Math.random();
      if (r < 0.7) {
        const count = Math.floor(Math.random() * 3) + 2;
        return JSON.stringify(Array.from({ length: count }, (_, i) => ({
          AmfIP1: `10.${30 + i}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
          Status: Math.random() > 0.3 ? '1' : '0',
          PLMNID: '46000',
        })));
      }
      if (r < 0.85) return pickRandom(['1', '0']);
      return '';
    })(),
    rfStatus: (() => {
      if ((isLTE || isNR) && Math.random() < 0.7) {
        const cellCount = Math.floor(Math.random() * 3) + 2;
        return Array.from({ length: cellCount }, () => pickRandom(['on', 'off'])).join(',');
      }
      return pickRandom(rfStatuses);
    })(),
    pmReportStatus: isLTE ? pickRandom(['off', 'normal', 'broken', '']) : '',
    halobFlag: (isNR || isLTE) ? Math.random() > 0.5 : false,
    syncStatus: pickRandom(syncStatuses),
    validity: isLTE && Math.random() > 0.8 ? randomDate(-90) : '',
    lockStatus: Math.random() > 0.9 ? 'locked' : '',
    ueCount: isOnline ? Math.floor(Math.random() * 200) : 0,
    euCount: isNR ? (() => {
      const total = Math.floor(Math.random() * 4) + 1;
      const conn = Math.floor(Math.random() * (total + 1));
      return `${conn}/${total}`;
    })() : '',
    ruCount: isNR ? (() => {
      const total = Math.floor(Math.random() * 8) + 1;
      const conn = Math.floor(Math.random() * (total + 1));
      return `${conn}/${total}`;
    })() : '',
    cpeCount: isLTE ? Math.floor(Math.random() * 32) : 0,
    wanSpeed: isLTE ? `${Math.floor(Math.random() * 1000)}Mbps` : '',
    serviceStatus: '',
    adminState: isNR ? pickRandom(['1', '2', '3', '2', '2']) : '',
    multiPlmnEnable: isNR ? pickRandom(['true', 'false', '']) : '',
    bscLinkStatus: '',
    bscSelect: '',
    bscSerialNumber: '',
    btsNum: 0,
    wanLinkStatus: '',
    omcStatus: '',
    bsic: '',
    vswr: '',

    ipsecAddr: Math.random() > 0.7 ? generateIP() : '',
    mmepoolIpsecAddr: isLTE && Math.random() > 0.8 ? generateIP() : '',
    ipaUnitId: '',
    omlRemoteIp: '',
    omlRemoteIpBak: '',

    gpsHeight: parseFloat((Math.random() * 500).toFixed(1)),
    mechanicalDowntilt: isLTE ? `${Math.floor(Math.random() * 15)}°` : '',
    electronicDowntilt: isLTE ? `${Math.floor(Math.random() * 10)}°` : '',
    verticalBeamWidth: isLTE ? `${(5 + Math.random() * 10).toFixed(1)}°` : '',
    horizontalAzimuth: isLTE ? `${Math.floor(Math.random() * 360)}°` : '',
    installAddress: i % 10 === 0 ? `${beijingCity.name}市某区某路${i}号` : '',
    gpsSatelliteCount: Math.floor(Math.random() * 16),

    rollbackVersion: isNR && Math.random() > 0.7 ? pickRandom(softwareVersions) : '',
    sasParam: isNR && Math.random() > 0.8 ? 'SAS-Config-1' : '',
    euRu: isNR ? `${Math.floor(Math.random() * 4)}/${Math.floor(Math.random() * 8)}` : '',
    halobLicense: isNR && Math.random() > 0.5 ? 'valid' : '',
    energySaving: isNR ? pickRandom(['enabled', 'disabled', '']) : '',
    gnbTopoCellmgr: isNR ? `TOPO-${index}` : '',
    sslCertValidity: isNR && Math.random() > 0.5 ? randomDate(-365) : '',
  };
});

// 合并数据：原有 200 条 + 北京额外 50 条
export const allMockDevices: Device[] = [...mockDevices, ...beijingDevices];
