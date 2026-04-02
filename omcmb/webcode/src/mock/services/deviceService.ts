import type { Device, NE, DeviceFilter, DeviceGroup, DeviceListResponse } from '@/types/device';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockDevices } from '../data/devices';
import { mockNEs } from '../data/nes';
import { delay, paginate, sortBy, filterByText, generateId } from '../utils';

let devices = [...mockDevices];
let groups: DeviceGroup[] = [
  { id: 'grp-default', name: '默认设备组', parentId: null, deviceCount: 0, description: '默认设备组', builtIn: 1 },
  { id: 'grp-east', name: '华东大区', parentId: null, deviceCount: 0, description: '华东区所有设备', builtIn: 1 },
  { id: 'grp-south', name: '华南大区', parentId: null, deviceCount: 0, description: '华南区所有设备', builtIn: 1 },
  { id: 'grp-north', name: '华北大区', parentId: null, deviceCount: 0, description: '华北区所有设备', builtIn: 1 },
  { id: 'grp-west', name: '西部大区', parentId: null, deviceCount: 0, description: '西部区所有设备', builtIn: 1 },
  { id: 'grp-bj', name: '北京', parentId: 'grp-default', deviceCount: 20, description: '北京市设备', builtIn: 1 },
  { id: 'grp-sh', name: '上海', parentId: 'grp-east', deviceCount: 20, description: '上海市设备', builtIn: 1 },
  { id: 'grp-nj', name: '南京', parentId: 'grp-east', deviceCount: 15, description: '南京市设备', builtIn: 1 },
  { id: 'grp-hz', name: '杭州', parentId: 'grp-east', deviceCount: 18, description: '杭州市设备', builtIn: 1 },
  { id: 'grp-gz', name: '广州', parentId: 'grp-south', deviceCount: 20, description: '广州市设备', builtIn: 1 },
  { id: 'grp-sz', name: '深圳', parentId: 'grp-south', deviceCount: 25, description: '深圳市设备', builtIn: 1 },
  { id: 'grp-dg', name: '东莞', parentId: 'grp-south', deviceCount: 12, description: '东莞市设备', builtIn: 1 },
  { id: 'grp-tj', name: '天津', parentId: 'grp-north', deviceCount: 16, description: '天津市设备', builtIn: 1 },
  { id: 'grp-sjz', name: '石家庄', parentId: 'grp-north', deviceCount: 10, description: '石家庄市设备', builtIn: 1 },
  { id: 'grp-cd', name: '成都', parentId: 'grp-west', deviceCount: 22, description: '成都市设备', builtIn: 1 },
  { id: 'grp-xa', name: '西安', parentId: 'grp-west', deviceCount: 14, description: '西安市设备', builtIn: 1 },
  { id: 'grp-cq', name: '重庆', parentId: 'grp-west', deviceCount: 19, description: '重庆市设备', builtIn: 1 },
];

function computeStats(items: Device[]) {
  return {
    total: items.length,
    online: items.filter((d) => d.connStatus === 'online').length,
    offline: items.filter((d) => d.connStatus === 'offline').length,
    alarmed: items.filter((d) => d.alarmLevel !== 'none').length,
  };
}

export const deviceService = {
  async getList(
    params: DeviceFilter & PageRequest
  ): Promise<DeviceListResponse> {
    await delay(100, 250);
    let filtered = [...devices];

    if (params.name) filtered = filterByText(filtered, 'name', params.name);
    if (params.sn) filtered = filterByText(filtered, 'sn', params.sn);
    if (params.vendor) filtered = filtered.filter((d) => d.vendor === params.vendor);
    if (params.productType) filtered = filtered.filter((d) => d.productType === params.productType);
    if (params.networkType) filtered = filtered.filter((d) => d.networkType === params.networkType);
    if (params.connStatus) {
      // 前端传数字值 '1'(在线)/'0'(离线)/'2'(同步失败)/'3'(同步中)，mock 数据用 'online'/'offline'
      const connMap: Record<string, string[]> = {
        '1': ['online'],
        '0': ['offline'],
        '2': ['offline'],
        '3': ['online'],
      };
      const mapped = connMap[params.connStatus];
      if (mapped) {
        filtered = filtered.filter((d) => mapped.includes(d.connStatus));
      } else {
        filtered = filtered.filter((d) => d.connStatus === params.connStatus);
      }
    }
    if (params.opState) {
      // 前端传 '1'=激活 / '0'=未激活
      // mock 数据: 'active'/'inactive' 或逗号分隔的多小区值 '1,0,1'
      filtered = filtered.filter((d) => {
        if (params.opState === '1') {
          // 激活：opState 为 'active' 或包含 '1' 的多小区值
          return d.opState === 'active' || /^1(,1)*$/.test(d.opState);
        } else if (params.opState === '0') {
          // 未激活：opState 为 'inactive' 或多小区值全不为 '1'
          return d.opState === 'inactive' || (/,/.test(d.opState) && !d.opState.includes('1'));
        }
        return true;
      });
    }
    if (params.productModel) filtered = filtered.filter((d) => d.productType === params.productModel);
    if (params.alarmLevel) filtered = filtered.filter((d) => d.alarmLevel === params.alarmLevel);
    if (params.region) filtered = filtered.filter((d) => d.region === params.region);
    if (params.subnet) filtered = filtered.filter((d) => d.subnet === params.subnet);
    if (params.engStatus) filtered = filtered.filter((d) => d.engStatus === params.engStatus);
    // 按 groupId 过滤（模拟：基于分组名称匹配城市）
    if (params.groupId) {
      const group = await this.getGroups().then((groups) => groups.find((g) => g.id === params.groupId));
      if (group) {
        // 根据分组名称过滤设备（通过设备名称包含城市名来判断）
        const groupCityMap: Record<string, string> = {
          'grp-bj': '北京',
          'grp-sh': '上海',
          'grp-gz': '广州',
        };
        const cityName = groupCityMap[params.groupId];
        if (cityName) {
          filtered = filtered.filter((d) => d.name.includes(cityName));
        }
      }
    }

    if (params.sortField) {
      filtered = sortBy(filtered, params.sortField as keyof Device, params.sortOrder ?? 'ascend');
    }

    const stats = computeStats(filtered);
    const page = paginate(filtered, params.page, params.pageSize);
    return { ...page, stats };
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

  async delete(ids: string[]): Promise<{ total: number; succeeded: number; failed: number }> {
    await delay(150, 300);
    const count = devices.length;
    devices = devices.filter((d) => !ids.includes(d.id));
    return { total: ids.length, succeeded: count - devices.length, failed: 0 };
  },

  async getGroups(): Promise<DeviceGroup[]> {
    await delay(80, 150);
    return [...groups];
  },

  async createGroup(data: { name: string; parent_id?: string; remark?: string }): Promise<DeviceGroup> {
    await delay(100, 200);
    const newGroup: DeviceGroup = {
      id: generateId('grp'),
      name: data.name,
      parentId: data.parent_id ?? null,
      deviceCount: 0,
      description: data.remark ?? '',
      builtIn: 0,
    };
    groups.push(newGroup);
    return newGroup;
  },

  async updateGroup(id: string, data: { name?: string; remark?: string }): Promise<DeviceGroup> {
    await delay(100, 200);
    const idx = groups.findIndex((g) => g.id === id);
    if (idx === -1) throw new Error(`Group ${id} not found`);
    groups[idx] = { ...groups[idx], name: data.name ?? groups[idx].name, description: data.remark ?? groups[idx].description };
    return groups[idx];
  },

  async deleteGroup(id: string): Promise<void> {
    await delay(100, 200);
    groups = groups.filter((g) => g.id !== id && g.parentId !== id);
  },

  async moveDevices(params: { device_ids: string[]; target_group_id: string }): Promise<void> {
    await delay(100, 200);
    // mock: no-op, just acknowledge
    void params;
  },

  async addDevicesToGroup(_groupId: string, _deviceIds: string[]): Promise<void> {
    await delay(100, 200);
    // mock: no-op
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

  async reboot(_id: string): Promise<void> {
    await delay(200, 500);
  },
};
