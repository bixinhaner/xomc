import type { Device, NE, DeviceFilter } from '@/types/device';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockDevices } from '../data/devices';
import { mockNEs } from '../data/nes';
import { delay, paginate, sortBy, filterByText, generateId } from '../utils';

let devices = [...mockDevices];

export const deviceService = {
  async getList(
    params: DeviceFilter & PageRequest
  ): Promise<PageResponse<Device>> {
    await delay(100, 250);
    let filtered = [...devices];

    if (params.name) filtered = filterByText(filtered, 'name', params.name);
    if (params.sn) filtered = filterByText(filtered, 'sn', params.sn);
    if (params.vendor) filtered = filtered.filter((d) => d.vendor === params.vendor);
    if (params.productType) filtered = filtered.filter((d) => d.productType === params.productType);
    if (params.networkType) filtered = filtered.filter((d) => d.networkType === params.networkType);
    if (params.connStatus) filtered = filtered.filter((d) => d.connStatus === params.connStatus);
    if (params.alarmLevel) filtered = filtered.filter((d) => d.alarmLevel === params.alarmLevel);
    if (params.region) filtered = filtered.filter((d) => d.region === params.region);
    if (params.subnet) filtered = filtered.filter((d) => d.subnet === params.subnet);
    if (params.engStatus) filtered = filtered.filter((d) => d.engStatus === params.engStatus);

    if (params.sortField) {
      filtered = sortBy(filtered, params.sortField as keyof Device, params.sortOrder ?? 'ascend');
    }

    return paginate(filtered, params.page, params.pageSize);
  },

  async getById(id: string): Promise<Device | null> {
    await delay(80, 150);
    return devices.find((d) => d.id === id) ?? null;
  },

  async getBySn(sn: string): Promise<Device | null> {
    await delay(80, 150);
    return devices.find((d) => d.sn === sn) ?? null;
  },

  async create(data: Omit<Device, 'id' | 'createTime'>): Promise<Device> {
    await delay(200, 400);
    const newDevice: Device = {
      ...data,
      id: generateId('dev'),
      createTime: new Date().toISOString(),
    };
    devices.push(newDevice);
    return newDevice;
  },

  async update(id: string, data: Partial<Device>): Promise<Device> {
    await delay(150, 300);
    const idx = devices.findIndex((d) => d.id === id);
    if (idx === -1) throw new Error(`Device ${id} not found`);
    devices[idx] = { ...devices[idx], ...data };
    return devices[idx];
  },

  async delete(ids: string[]): Promise<void> {
    await delay(150, 300);
    devices = devices.filter((d) => !ids.includes(d.id));
  },

  async getGroups() {
    await delay(80, 150);
    return [
      { id: 'grp-001', name: '华北大区', parentId: null, deviceCount: 45, description: '华北区所有设备' },
      { id: 'grp-002', name: '华东大区', parentId: null, deviceCount: 60, description: '华东区所有设备' },
      { id: 'grp-003', name: '华南大区', parentId: null, deviceCount: 50, description: '华南区所有设备' },
      { id: 'grp-004', name: '西南大区', parentId: null, deviceCount: 25, description: '西南区所有设备' },
      { id: 'grp-north', name: '北京', parentId: 'grp-001', deviceCount: 30, description: '北京市设备' },
      { id: 'grp-sh', name: '上海', parentId: 'grp-002', deviceCount: 35, description: '上海市设备' },
      { id: 'grp-gz', name: '广州', parentId: 'grp-003', deviceCount: 28, description: '广州市设备' },
      { id: 'grp-5g', name: '5G gNB设备', parentId: null, deviceCount: 60, description: '全国所有5G基站' },
    ];
  },

  // NE methods
  async getNEList(params: { keyword?: string } & PageRequest): Promise<PageResponse<NE>> {
    await delay(100, 250);
    let filtered = [...mockNEs];
    if (params.keyword) {
      filtered = filtered.filter(
        (n) =>
          n.neName.includes(params.keyword!) ||
          n.sn.includes(params.keyword!)
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getNEBySn(sn: string): Promise<NE | null> {
    await delay(80, 150);
    return mockNEs.find((n) => n.sn === sn) ?? null;
  },
};
