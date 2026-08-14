import type { Device, NE, DeviceFilter, DeviceGroup, DeviceListResponse, DeviceControlActionHistoryList } from '../../types/device';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mockDevices } from '../data/devices';
import { mockNEs } from '../data/nes';
import { delay, paginate, sortBy, filterByText, generateId } from '../utils';
import { normalizeNetworkTypeFilter } from '../../utils/networkType';

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
  const online = items.filter((d) => d.connStatus === 'online').length;
  const offline = items.filter((d) => d.connStatus === 'offline').length;
  const currentUECount = items
    .filter((d) => d.connStatus === 'online')
    .reduce((sum, d) => sum + d.ueCount, 0);
  return {
    total: items.length,
    online,
    offline,
    online_count: online,
    offline_count: offline,
    current_ue_count: currentUECount,
    alarmed: items.filter((d) => d.alarmLevel !== 'none').length,
  };
}

function matchesDeviceSearch(device: Device, rawSearch: string): boolean {
  const keywords = rawSearch
    .split(',')
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean)
    .slice(0, 50);
  if (keywords.length === 0) return true;

  const values = [
    device.sn,
    device.name,
    device.hostName,
    device.deviceName,
    device.ipAddress,
    device.macAddress,
    device.pci,
    device.site,
    device.deviceModel,
    device.vendor,
  ].map((v) => String(v ?? '').toLowerCase());

  return keywords.some((kw) => values.some((v) => v.includes(kw)));
}

export const deviceService = {
  async getList(
    params: DeviceFilter & PageRequest
  ): Promise<DeviceListResponse> {
    await delay(100, 250);
    let filtered = [...devices];

    if (params.name) filtered = filterByText(filtered, 'name', params.name);
    if (params.sn) filtered = filterByText(filtered, 'sn', params.sn);
    if (params.searchText) {
      filtered = filtered.filter((d) => matchesDeviceSearch(d, params.searchText!));
    }
    // 批量输入：按 SN 列表精确过滤（与后端 ?sn_list= 对齐）。
    if (params.snList && params.snList.length > 0) {
      const set = new Set(params.snList);
      filtered = filtered.filter((d) => set.has(d.sn));
    }
    // 实时在线过滤（与后端 ?is_online= 对齐）。
    if (typeof params.isOnline === 'boolean') {
      filtered = filtered.filter((d) =>
        typeof d.isOnline === 'boolean' ? d.isOnline === params.isOnline : (d.connStatus === 'online') === params.isOnline,
      );
    }
    if (params.vendor) filtered = filtered.filter((d) => d.vendor === params.vendor);
    if (params.productClass) filtered = filtered.filter((d) => d.productClass === params.productClass);
    // #443: 入参的「制式」（lte/nr/gsm 或老链路 eNB/gNB/GSM）先归一到
    // device.networkType 实际存储的基站类型码（eNB/gNB/GSM）再比对，与真实
    // deviceApi 的过滤语义对齐，否则传 'lte' 永不命中 'eNB'。
    if (params.networkType) {
      const want = normalizeNetworkTypeFilter(params.networkType);
      filtered = filtered.filter((d) => d.networkType === want);
    }
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
    if (params.controlSource) {
      filtered = filtered.filter((d) => d.controlSummary?.sourceType === params.controlSource);
    }
    if (params.controlPhase && params.controlPhase.length > 0) {
      const phases = new Set(params.controlPhase);
      filtered = filtered.filter((d) => d.controlSummary && phases.has(d.controlSummary.phase));
    }
    if (params.productModel) filtered = filtered.filter((d) => d.productClass === params.productModel);
    if (params.alarmLevel) filtered = filtered.filter((d) => d.alarmLevel === params.alarmLevel);
    if (params.region) filtered = filtered.filter((d) => d.region === params.region);
    if (params.subnet) filtered = filtered.filter((d) => d.subnet === params.subnet);
    if (params.engStatus) filtered = filtered.filter((d) => d.engStatus === params.engStatus);
    // 按 groupId 过滤：与真实接口保持一致，按 membership 做多选 OR。
    if (params.groupId) {
      const selectedGroupIDs = Array.isArray(params.groupId) ? params.groupId : [params.groupId];
      const selectedSet = new Set(selectedGroupIDs);
      filtered = filtered.filter((d) => d.groupId && selectedSet.has(d.groupId));
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

  async getControlActions(id: string): Promise<DeviceControlActionHistoryList> {
    await delay(80, 150);
    const device = devices.find((item) => item.id === id);
    if (!device?.controlSummary) {
      return { items: [], total: 0, page: 1, pageSize: 20 };
    }
    return {
      items: [{
        id: 'mock-control-action-1',
        sourceType: 'geofence',
        sourceId: 'mock-geofence-1',
        sourceName: device.controlSummary.sourceName,
        reasonCode: device.controlSummary.reasonCode,
        observationVersion: 12,
        effectiveStateVersion: 4,
        actionType: 'deactivate',
        status: 'verified',
        beforeState: [
          { path: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE', value: '1' },
          { path: 'Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus', value: '1' },
        ],
        requestedState: [
          { path: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE', value: '0' },
          { path: 'Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus', value: '0' },
        ],
        verifiedState: [
          { path: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE', value: '0' },
          { path: 'Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus', value: '0' },
        ],
        evaluation: {
          id: 'mock-evaluation-1', observationVersion: 12,
          latitude: 39.9042, longitude: 116.4074,
          observedAt: '2026-08-13T10:14:55+08:00',
          ruleType: 'polygon_allow_zone', signedDistanceMeters: 18.6,
          confirmedState: 'outside', reasonCode: 'confirmed_exit',
        },
        createdAt: device.controlSummary.triggeredAt,
        updatedAt: device.controlSummary.completedAt ?? device.controlSummary.triggeredAt,
        completedAt: device.controlSummary.completedAt,
      }],
      total: 1,
      page: 1,
      pageSize: 20,
    };
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

  async getGroups(): Promise<{ groups: DeviceGroup[]; stats: { totalDevices: number } }> {
    await delay(80, 150);
    // 计算 totalDevices 作为所有设备的数量
    const totalDevices = devices.length;
    return { groups: [...groups], stats: { totalDevices } };
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

  /**
   * T-0129 mock 类型对齐：与 deviceApi.syncDeviceParams (T-0126) 签名一致。
   * 返回 fake task envelope, force 参数透传无效果。
   */
  async syncDeviceParams(
    _deviceId: string,
    _options?: { force?: boolean },
  ): Promise<{
    status: string;
    sourceId: string;
    deviceId: string;
    serialNumber: string;
    force: boolean;
  }> {
    await delay(200, 500);
    return {
      status: 'queued',
      sourceId: `manual:mock-${Date.now()}`,
      deviceId: _deviceId,
      serialNumber: 'MOCK-SN',
      force: _options?.force ?? false,
    };
  },
};
