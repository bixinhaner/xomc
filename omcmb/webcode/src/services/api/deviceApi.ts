import http from '../http';
import type { Device, NE, DeviceFilter, DeviceGroup, DeviceListResponse, DeviceListStats, DeviceStats, DeviceParameter } from '@/types/device';
import type { PageRequest, PageResponse } from '@/types/pagination';

// Backend device model from Go struct
interface BackendDevice {
  id: string;
  serial_number: string;
  oui: string;
  product_class: string;
  manufacturer: string;
  model_name: string;
  carrier: string;
  technology: string;
  data_model_id?: string;
  status: string;
  firmware_version: string;
  ip_address: string;
  connection_request_url: string;
  last_inform_at?: string;
  inform_interval: number;
  site_name: string;
  site_id: string;
  latitude: number;
  longitude: number;
  created_at: string;
  updated_at: string;
  platform_type?: string;

  // --- 监控扩展字段 ---
  host_name?: string;
  product_name?: string;
  mac_address?: string;
  group_name?: string;
  online_at?: string;
  offline_at?: string;
  online_duration?: number;
  up_time?: string;
  first_online_at?: string;
  gps_version?: string;
  rom?: string;
  remark?: string;
  gnb_id?: string;

  // Cell
  enb_id?: string;
  cell_id?: string;
  eci?: string;
  nr_cell_id?: string;
  pci?: string;
  plmn_id?: string;
  tac?: string;
  subframe_assignment?: string;
  special_subframe?: string;
  root_index?: string;
  bandwidth?: string;
  dl_earfcn?: string;
  ul_earfcn?: string;
  network_model?: string;
  tx_power?: string;
  band?: string;
  lac?: string;
  arfcn?: string;
  uplink_frequency?: string;
  downlink_frequency?: string;

  // Status
  op_state?: string;
  mme_status?: string;
  amf_status?: string;
  rf_status?: string;
  pm_report_status?: string;
  halob_enabled?: boolean;
  sync_status?: string;
  validity?: string;
  lock_status?: string;
  ue_count?: number;
  eu_count?: string;
  ru_count?: string;
  cpe_count?: number;
  wan_speed?: string;
  service_status?: string;
  admin_state?: string;
  multi_plmn_enable?: string;
  bsc_link_status?: string;
  bsc_select?: string;
  bsc_serial_number?: string;
  bts_num?: number;

  // Network
  ipsec_addr?: string;
  mmepool_ipsec_addr?: string;
  ipa_unit_id?: string;
  oml_remote_ip?: string;
  oml_remote_ip_bak?: string;

  // Location
  gps_height?: number;
  mechanical_downtilt?: string;
  electronic_downtilt?: string;
  vertical_beam_width?: string;
  horizontal_azimuth?: string;
  install_address?: string;
  gps_satellite_count?: number;

  // 5G NR Others
  rollback_version?: string;
  sas_param?: string;
  eu_ru?: string;
  halob_license?: string;
  energy_saving?: string;
  gnb_topo_cellmgr?: string;
  ssl_cert_validity?: string;
}

export interface BatchOperationResult {
  total: number;
  succeeded: number;
  failed: number;
  errors?: Array<{ id: string; message: string }>;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
  // 统计字段 — 后端返回筛选条件下的全量统计
  stats?: {
    total: number;
    online: number;
    offline: number;
    alarmed: number;
  };
}

// Map backend device status to frontend connStatus
// Backend statuses: discovered, registered, provisioning, active, maintenance, offline, decommissioned
function mapStatus(status: string): Device['connStatus'] {
  switch (status) {
    case 'online':
    case 'active':
      return 'online';
    case 'offline':
    case 'registered':
    case 'discovered':
    case 'provisioning':
    case 'maintenance':
    case 'decommissioned':
    default:
      return 'offline';
  }
}

function mapBackendDevice(bd: BackendDevice): Device {
  return {
    id: bd.id,
    sn: bd.serial_number,
    name: bd.site_name || bd.serial_number,
    vendor: bd.manufacturer,
    productType: bd.product_class,
    networkType: bd.technology,
    deviceModel: bd.model_name,
    region: bd.site_name,
    stationId: bd.site_id,
    connStatus: mapStatus(bd.status),
    alarmLevel: 'none',
    engStatus: 'commissioned',
    mgmtStatus: 'managed',
    lastOnlineTime: bd.last_inform_at || '',
    ipAddress: bd.ip_address,
    subnet: '',
    site: bd.site_name,
    longitude: bd.longitude,
    latitude: bd.latitude,
    softwareVersion: bd.firmware_version,
    createTime: bd.created_at,

    // 监控扩展字段
    platformType: bd.platform_type || '',
    hostName: bd.host_name || '',
    productName: bd.product_name || '',
    firmwareVersion: bd.firmware_version || '',
    macAddress: bd.mac_address || '',
    groupName: bd.group_name || '',
    onlineTime: bd.online_at || '',
    offlineTime: bd.offline_at || '',
    onlineDuration: bd.online_duration ?? 0,
    upTime: bd.up_time || '',
    firstOnlineTime: bd.first_online_at || '',
    lastInformTime: bd.last_inform_at || '',
    siteName: bd.site_name || '',
    gpsVersion: bd.gps_version || '',
    rom: bd.rom || '',
    remark: bd.remark || '',
    gnbId: bd.gnb_id || '',

    enbId: bd.enb_id || '',
    cellId: bd.cell_id || '',
    eci: bd.eci || '',
    nrCellId: bd.nr_cell_id || '',
    pci: bd.pci || '',
    plmnId: bd.plmn_id || '',
    tac: bd.tac || '',
    subframeAssignment: bd.subframe_assignment || '',
    specialSubframe: bd.special_subframe || '',
    rootIndex: bd.root_index || '',
    siteId: bd.site_id || '',
    bandwidth: bd.bandwidth || '',
    dlEarfcn: bd.dl_earfcn || '',
    ulEarfcn: bd.ul_earfcn || '',
    networkModel: bd.network_model || '',
    txPower: bd.tx_power || '',
    band: bd.band || '',
    lac: bd.lac || '',
    arfcn: bd.arfcn || '',
    uplinkFrequency: bd.uplink_frequency || '',
    downlinkFrequency: bd.downlink_frequency || '',

    opState: bd.op_state || 'unknown',
    mmeStatus: bd.mme_status || '',
    amfStatus: bd.amf_status || '',
    rfStatus: bd.rf_status || '',
    pmReportStatus: bd.pm_report_status || '',
    halobFlag: bd.halob_enabled ?? false,
    syncStatus: bd.sync_status || '',
    validity: bd.validity || '',
    lockStatus: bd.lock_status || '',
    ueCount: bd.ue_count ?? 0,
    euCount: bd.eu_count || '',
    ruCount: bd.ru_count || '',
    cpeCount: bd.cpe_count ?? 0,
    wanSpeed: bd.wan_speed || '',
    serviceStatus: bd.service_status || '',
    adminState: bd.admin_state || '',
    multiPlmnEnable: bd.multi_plmn_enable || '',
    bscLinkStatus: bd.bsc_link_status || '',
    bscSelect: bd.bsc_select || '',
    bscSerialNumber: bd.bsc_serial_number || '',
    btsNum: bd.bts_num ?? 0,

    ipsecAddr: bd.ipsec_addr || '',
    mmepoolIpsecAddr: bd.mmepool_ipsec_addr || '',
    ipaUnitId: bd.ipa_unit_id || '',
    omlRemoteIp: bd.oml_remote_ip || '',
    omlRemoteIpBak: bd.oml_remote_ip_bak || '',

    gpsHeight: bd.gps_height ?? 0,
    mechanicalDowntilt: bd.mechanical_downtilt || '',
    electronicDowntilt: bd.electronic_downtilt || '',
    verticalBeamWidth: bd.vertical_beam_width || '',
    horizontalAzimuth: bd.horizontal_azimuth || '',
    installAddress: bd.install_address || '',
    gpsSatelliteCount: bd.gps_satellite_count ?? 0,

    rollbackVersion: bd.rollback_version || '',
    sasParam: bd.sas_param || '',
    euRu: bd.eu_ru || '',
    halobLicense: bd.halob_license || '',
    energySaving: bd.energy_saving || '',
    gnbTopoCellmgr: bd.gnb_topo_cellmgr || '',
    sslCertValidity: bd.ssl_cert_validity || '',
  };
}

function mapListResponse(resp: BackendListResponse<BackendDevice>): DeviceListResponse {
  const items = (resp.items || []).map(mapBackendDevice);
  // 优先使用后端返回的统计；如果后端未返回则从当前页数据估算
  const stats: DeviceListStats = resp.stats ?? {
    total: resp.total,
    online: items.filter((d) => d.connStatus === 'online').length,
    offline: items.filter((d) => d.connStatus === 'offline').length,
    alarmed: items.filter((d) => d.alarmLevel !== 'none').length,
  };
  return {
    items,
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
    stats,
  };
}

export const deviceApi = {
  async getList(params: DeviceFilter & PageRequest): Promise<DeviceListResponse> {
    // Map frontend filter fields to backend query params
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
      sortField: params.sortField,
      sortOrder: params.sortOrder,
    };

    if (params.name) query.search = params.name;
    if (params.searchText) query.search = params.searchText;
    if (params.sn) query.sn = params.sn;
    if (params.vendor) query.oui = params.vendor;
    if (params.networkType) query.technology = params.networkType;
    // connStatus: 前端值 '1'(在线)→'active', '0'(离线)→'offline', '2'(同步失败)→'offline', '3'(同步中)→'active'
    if (params.connStatus) {
      const connStatusMap: Record<string, string> = {
        '1': 'active',
        '0': 'offline',
        '2': 'offline',  // 同步失败视为离线
        '3': 'active',   // 同步中视为在线
      };
      query.status = connStatusMap[params.connStatus] ?? params.connStatus;
    }
    if (params.opState) query.op_state = params.opState;
    // productModel → product_class
    if (params.productModel) query.product_class = params.productModel;

    const { data } = await http.get<BackendListResponse<BackendDevice>>('/devices', {
      params: query,
    });
    return mapListResponse(data);
  },

  async getById(id: string): Promise<Device | null> {
    try {
      const { data } = await http.get<BackendDevice>(`/devices/${id}`);
      return mapBackendDevice(data);
    } catch {
      return null;
    }
  },

  async getBySn(sn: string): Promise<Device | null> {
    const result = await deviceApi.getList({
      sn,
      page: 1,
      pageSize: 1,
    });
    return result.items.length > 0 ? result.items[0] : null;
  },

  async create(data: Omit<Device, 'id' | 'createTime'>): Promise<Device> {
    const payload: Record<string, unknown> = {
      serial_number: data.sn,
      oui: '',
      product_class: data.productType,
      manufacturer: data.vendor,
      model_name: data.deviceModel,
      carrier: '',
      technology: data.networkType,
      status: data.connStatus === 'online' ? 'active' : 'offline',
      firmware_version: data.softwareVersion,
      ip_address: data.ipAddress,
      site_name: data.site || data.name,
      site_id: data.stationId,
      latitude: data.latitude,
      longitude: data.longitude,
    };
    const { data: created } = await http.post<BackendDevice>('/devices', payload);
    return mapBackendDevice(created);
  },

  async update(id: string, data: Partial<Device>): Promise<Device> {
    const payload: Record<string, unknown> = {};
    if (data.sn !== undefined) payload.serial_number = data.sn;
    if (data.vendor !== undefined) payload.manufacturer = data.vendor;
    if (data.productType !== undefined) payload.product_class = data.productType;
    if (data.networkType !== undefined) payload.technology = data.networkType;
    if (data.deviceModel !== undefined) payload.model_name = data.deviceModel;
    if (data.connStatus !== undefined) payload.status = data.connStatus === 'online' ? 'active' : 'offline';
    if (data.softwareVersion !== undefined) payload.firmware_version = data.softwareVersion;
    if (data.ipAddress !== undefined) payload.ip_address = data.ipAddress;
    if (data.site !== undefined) payload.site_name = data.site;
    if (data.name !== undefined) payload.site_name = data.name;
    if (data.stationId !== undefined) payload.site_id = data.stationId;
    if (data.latitude !== undefined) payload.latitude = data.latitude;
    if (data.longitude !== undefined) payload.longitude = data.longitude;
    const { data: updated } = await http.put<BackendDevice>(`/devices/${id}`, payload);
    return mapBackendDevice(updated);
  },

  async delete(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.delete<BatchOperationResult>('/devices/batch', { data: { ids } });
    return data;
  },

  async batchReboot(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.post<BatchOperationResult>('/devices/batch-reboot', { ids });
    return data;
  },

  async getGroups(): Promise<DeviceGroup[]> {
    // Backend GET /device-groups/tree returns nested tree with device counts.
    // We flatten it to a flat list so the frontend tree builder works.
    interface BackendGroupItem {
      id: string;
      name: string;
      parent_id: string | null;
      device_count: number;
      description: string;
      remark: string;
      is_default: boolean;
      level: number;
      children?: BackendGroupItem[];
    }
    interface TreeResponse {
      items: BackendGroupItem[];
      stats?: { total_groups: number; grouped_devices: number; ungrouped_devices: number };
    }
    const { data } = await http.get<TreeResponse>('/device-groups/tree');
    const flat: DeviceGroup[] = [];
    function walk(items: BackendGroupItem[]) {
      for (const g of items) {
        flat.push({
          id: g.id,
          name: g.name,
          parentId: g.parent_id,
          deviceCount: g.device_count ?? 0,
          description: g.remark || g.description || '',
          builtIn: g.is_default ? 1 : 0,
        });
        if (g.children?.length) walk(g.children);
      }
    }
    walk(data.items || []);
    return flat;
  },

  async createGroup(data: { name: string; parent_id?: string; remark?: string }): Promise<DeviceGroup> {
    const { data: created } = await http.post<DeviceGroup>('/device-groups', data);
    return created;
  },

  async updateGroup(id: string, data: { name?: string; remark?: string }): Promise<DeviceGroup> {
    const { data: updated } = await http.put<DeviceGroup>(`/device-groups/${id}`, data);
    return updated;
  },

  async deleteGroup(id: string): Promise<void> {
    await http.delete(`/device-groups/${id}`);
  },

  async moveDevices(params: { device_ids: string[]; target_group_id: string }): Promise<void> {
    await http.post('/device-groups/move-devices', params);
  },

  async addDevicesToGroup(groupId: string, deviceIds: string[]): Promise<void> {
    await http.post(`/device-groups/${groupId}/devices`, { device_ids: deviceIds });
  },

  async getStats(): Promise<DeviceStats> {
    const { data } = await http.get<DeviceStats>('/devices/stats');
    return data;
  },

  async getParameters(id: string): Promise<{ items: DeviceParameter[]; total: number }> {
    const { data } = await http.get<{ items: DeviceParameter[]; total: number }>(`/devices/${id}/parameters`);
    return data;
  },

  async reboot(id: string): Promise<void> {
    await http.post(`/devices/${id}/reboot`);
  },

  async getNEList(params: { keyword?: string } & PageRequest): Promise<PageResponse<NE>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.keyword) query.search = params.keyword;

    const { data } = await http.get<BackendListResponse<BackendDevice>>('/devices', {
      params: query,
    });
    return {
      items: (data.items || []).map((bd) => ({
        id: bd.id,
        neName: bd.site_name || bd.serial_number,
        sn: bd.serial_number,
        neType: bd.product_class,
        vendor: bd.manufacturer,
        region: bd.site_name,
        subnet: '',
        site: bd.site_name,
        connStatus: mapStatus(bd.status),
        alarmLevel: 'none' as const,
      })),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getNEBySn(sn: string): Promise<NE | null> {
    const result = await deviceApi.getNEList({ keyword: sn, page: 1, pageSize: 1 });
    return result.items.length > 0 ? result.items[0] : null;
  },
};
