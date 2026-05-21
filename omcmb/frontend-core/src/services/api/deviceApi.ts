import http from '../http';
import type { Device, NE, DeviceFilter, DeviceGroup, DeviceListResponse, DeviceListStats, DeviceStats, DeviceParameter, CreateDeviceInput, NameFilterItem, BatchImportRequest, BatchImportResponse } from '../../types/device';
import type { PageRequest, PageResponse } from '../../types/pagination';

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
  // T-0162 DEPRECATED: status 列已 DROP；后端读 lifecycle_state+is_online，scan
  // 后 populateDeviceCompat() 派生 status 回老前端读侧。新前端读 lifecycle_state
  // 和 is_online 字段，不要再用 status。
  status: string;
  // T-0162: 业务生命周期（与后端 model.DeviceLifecycle 1:1）
  lifecycle_state: string;
  // T-0162: 实时在线
  is_online: boolean;
  firmware_version: string;
  ip_address: string;
  connection_request_url: string;
  last_inform_at?: string;
  inform_interval: number;
  device_name: string;
  site_id: string;
  latitude: number;
  longitude: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  deleted_by?: string;
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
  // T-0162: 后端 service.ListDevicesWithInfo 在主查询同样筛选下跑 group-by
  // 聚合，填充本字段；前端直读 stats 而不是用 items.filter() 自行估算（修
  // Q2 分析里的"page-only 偏差"bug）。
  stats?: {
    total: number;
    by_lifecycle?: Record<string, number>;
    online_count: number;
    offline_count: number;
    alarmed: number;
  };
}

// T-0162: 老 mapStatus 已删除（"乐观归类"5 种 status 全归 online 与筛选侧
// 不对称引起 Q1 bug 的根因）。新 mapBackendDevice 直接读 bd.is_online
// 派生 connStatus，与后端语义 1:1。

function mapBackendDevice(bd: BackendDevice): Device {
  // T-0162: 直读新字段；connStatus 由 isOnline 派生供老 UI 代码兼容（DEPRECATED）
  const lifecycleState = (bd.lifecycle_state || 'registered') as Device['lifecycleState'];
  const isOnline = Boolean(bd.is_online);

  return {
    id: bd.id,
    sn: bd.serial_number,
    name: bd.device_name || bd.serial_number,
    vendor: bd.manufacturer,
    productType: bd.product_class,
    networkType: bd.technology,
    deviceModel: bd.model_name,
    region: bd.device_name,
    stationId: bd.site_id,

    // T-0162 新字段
    lifecycleState,
    isOnline,

    // T-0162 DEPRECATED: 派生 connStatus（is_online → online/offline 直翻；
    // 不再用老 mapStatus 那种 5 种状态全归 online 的"乐观归类"做法）
    connStatus: isOnline ? 'online' : 'offline',

    alarmLevel: 'none',
    engStatus: 'commissioned',
    mgmtStatus: 'managed',
    lastOnlineTime: bd.last_inform_at || '',
    ipAddress: bd.ip_address,
    subnet: '',
    site: bd.device_name,
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
    deviceName: bd.device_name || '',
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

    // 回收站扩展字段
    deletedAt: bd.deleted_at,
    deletedBy: bd.deleted_by,
  };
}

function mapListResponse(resp: BackendListResponse<BackendDevice>): DeviceListResponse {
  const items = (resp.items || []).map(mapBackendDevice);
  // T-0162: 优先用后端 stats（T-0162 P3 已实装 backend 真返回）；fallback
  // 用当前页 items 估算是过渡兜底，新部署后绝大多数请求走前一路径。
  const stats: DeviceListStats = resp.stats
    ? {
        total: resp.stats.total,
        online_count: resp.stats.online_count,
        offline_count: resp.stats.offline_count,
        // T-0162 alias for backward compat
        online: resp.stats.online_count,
        offline: resp.stats.offline_count,
        by_lifecycle: resp.stats.by_lifecycle as DeviceListStats['by_lifecycle'],
        alarmed: resp.stats.alarmed,
      }
    : {
        total: resp.total,
        online_count: items.filter((d) => d.isOnline).length,
        offline_count: items.filter((d) => !d.isOnline).length,
        online: items.filter((d) => d.isOnline).length,
        offline: items.filter((d) => !d.isOnline).length,
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
    // productType → product_class
    if (params.productType) query.product_class = params.productType;
    // networkType: T-0162 后 network_type 字典已直接给 'lte'/'nr'（与后端
    // devices.technology 字段值一致），不再需要 eNB/gNB → lte/nr 翻译。但
    // 历史前端 / 老 link 可能仍传 eNB/gNB，做向下兼容映射。
    if (params.networkType) {
      const legacyMap: Record<string, string> = { eNB: 'lte', gNB: 'nr' };
      const tech = legacyMap[params.networkType] ?? params.networkType;
      query.technology = tech;
    }
    // groupId → group_id (device group filter)
    if (params.groupId) query.group_id = params.groupId;

    // T-0162: 新筛选维度，直接 1:1 传给后端，前端不再翻译
    if (params.lifecycleState && params.lifecycleState.length > 0) {
      query.lifecycle_state = params.lifecycleState.join(','); // CSV 多选
    }
    // isOnline 兼容三种来源：
    //   - 程序化调用：boolean true/false
    //   - 表单/字典选择：字符串 'true'/'false'（is_online 字典 value 即字符串）
    //   - URL deep-link：?isOnline=true 也是字符串
    // 类型定义里 isOnline 是 boolean，但运行时来自表单时是字符串 —— 不能只
    // 看 typeof boolean，否则字典选 "在线" 会被静默丢弃。
    if (params.isOnline === true || (params.isOnline as unknown) === 'true') {
      query.is_online = 'true';
    } else if (params.isOnline === false || (params.isOnline as unknown) === 'false') {
      query.is_online = 'false';
    }
    // 多选下拉 → CSV：FilterField type='multi-select' 返回数组，axios 默认
    // 把数组序列化为 key[]=a&key[]=b，gin 的 c.Query("model_name") 只认
    // 'model_name'，不认 'model_name[]'，过滤会被静默丢弃。统一 join 成
    // CSV，对应后端按 strings.Split(",") 解析（与 lifecycle_state 同模式）。
    const csv = (v: unknown): string | undefined => {
      if (Array.isArray(v)) return v.length > 0 ? v.join(',') : undefined;
      return v ? String(v) : undefined;
    };
    if (params.modelName) query.model_name = csv(params.modelName);
    if (params.softwareVersion) query.software_version = csv(params.softwareVersion);
    if (params.firmwareVersion) query.firmware_version = csv(params.firmwareVersion);

    // T-0162 DEPRECATED: 老 connStatus 仍兼容，但仅在新字段都没传时才用
    // （新前端代码直接用 lifecycleState/isOnline）
    if (params.connStatus && typeof params.isOnline !== 'boolean' && !params.lifecycleState) {
      // connStatus '1'(在线) → is_online=true，'0'(离线) → is_online=false
      const onlineMap: Record<string, boolean> = {
        '1': true, '0': false, '2': false, '3': true,
        'online': true, 'offline': false,
      };
      const v = onlineMap[params.connStatus as string];
      if (typeof v === 'boolean') {
        query.is_online = v ? 'true' : 'false';
      }
    }
    if (params.opState) query.op_state = params.opState;
    // productModel / productType 都映射到 product_class（前者是 multi-select，
    // 后者是历史单值字段），统一走 csv() 处理。
    if (params.productModel) query.product_class = csv(params.productModel);

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

  async create(input: CreateDeviceInput): Promise<Device> {
    // payload 字段名严格对齐后端 device.CreateDeviceRequest（device_handler.go）。
    // 后端 binding:"required,oneof=cmcc ctcc cucc"/oneof=lte nr，前端 dropdown 提交
    // 同一组枚举值，非法值会被 422 拦在 BE。
    const payload: Record<string, unknown> = {
      serial_number: input.serialNumber,
      oui: input.oui,
      carrier: input.carrier,
      technology: input.technology,
      product_class: input.productClass || undefined,
      manufacturer: input.manufacturer || undefined,
      model_name: input.modelName || undefined,
      ip_address: input.ipAddress || undefined,
      device_name: input.deviceName || undefined,
      site_id: input.siteId || undefined,
      latitude: input.latitude,
      longitude: input.longitude,
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
    if (data.site !== undefined) payload.device_name = data.site;
    if (data.name !== undefined) payload.device_name = data.name;
    if (data.stationId !== undefined) payload.site_id = data.stationId;
    if (data.latitude !== undefined) payload.latitude = data.latitude;
    if (data.longitude !== undefined) payload.longitude = data.longitude;
    if (data.remark !== undefined) payload.remark = data.remark;
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

  // 批量导入：前端解析 CSV → 结构化 JSON → 后端逐行 CreateDevice。
  // payload 字段已是 wire 层 snake_case（与 device.CreateDeviceRequest 对齐），
  // axios 请求拦截器只转 query params，不动 body，因此这里不要再写 camelCase。
  async batchImportDevices(payload: BatchImportRequest): Promise<BatchImportResponse> {
    const { data } = await http.post<BatchImportResponse>('/devices/batch-import', payload);
    return data;
  },

  async getGroups(): Promise<{ groups: DeviceGroup[]; stats: { totalDevices: number } }> {
    // Backend GET /device-groups/tree returns nested tree with device counts.
    // 我们摊平为列表给前端树构造器；同时**保留 matching rule 字段**（matching_mode /
    // name_rule_list / lac_list / tac_list），让 L2 编辑入口能回填原规则。
    interface BackendGroupItem {
      id: string;
      name: string;
      parent_id: string | null;
      device_count: number;
      description: string;
      remark: string;
      is_default: boolean;
      level: number;
      // 匹配规则字段（service.go DeviceGroup 反序列化）
      matching_mode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber';
      name_rule_list?: NameFilterItem[];
      lac_list?: number[];
      tac_list?: number[];
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
          matchingMode: g.matching_mode,
          nameRuleList: g.name_rule_list,
          lacList: g.lac_list,
          tacList: g.tac_list,
        });
        if (g.children?.length) walk(g.children);
      }
    }
    walk(data.items || []);
    // 计算总设备数 = 已分组设备 + 未分组设备
    const totalDevices = (data.stats?.grouped_devices ?? 0) + (data.stats?.ungrouped_devices ?? 0);
    return { groups: flat, stats: { totalDevices } };
  },

  async createGroup(data: {
    name: string;
    parent_id?: string;
    remark?: string;
    matching_mode?: 'deviceName' | 'lac' | 'tac';
    name_rule_list?: NameFilterItem[];
    lac_list?: number[];
    tac_list?: number[];
  }): Promise<DeviceGroup> {
    const { data: created } = await http.post<DeviceGroup>('/device-groups', data);
    return created;
  },

  async updateGroup(id: string, data: {
    name?: string;
    parent_id?: string;
    remark?: string;
    matching_mode?: 'deviceName' | 'lac' | 'tac';
    name_rule_list?: NameFilterItem[];
    lac_list?: number[];
    tac_list?: number[];
  }): Promise<DeviceGroup> {
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

  // T-0126: 手动触发 Path B 全量参数同步（reason="manual"）。
  // 替代旧 deviceParameterApi.syncParameters（Path A 已下线）。
  // 后端 POST /api/v1/devices/:id/sync-params 响应 202 {status, source_id, device_id, serial_number, force}
  // force: 预留供未来节流绕过；当前 manual 端点天然不走节流。
  async syncDeviceParams(
    id: string,
    options?: { force?: boolean }
  ): Promise<{
    status: string;
    sourceId: string;
    deviceId: string;
    serialNumber: string;
    force: boolean;
  }> {
    const { data } = await http.post<{
      status: string;
      source_id: string;
      device_id: string;
      serial_number: string;
      force: boolean;
    }>(`/devices/${id}/sync-params`, options ?? {});
    return {
      status: data.status,
      sourceId: data.source_id,
      deviceId: data.device_id,
      serialNumber: data.serial_number,
      force: data.force,
    };
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
        neName: bd.device_name || bd.serial_number,
        sn: bd.serial_number,
        neType: bd.product_class,
        vendor: bd.manufacturer,
        region: bd.device_name,
        subnet: '',
        site: bd.device_name,
        // T-0162: 直接用 is_online 派生（与 mapBackendDevice 一致）
        connStatus: bd.is_online ? ('online' as const) : ('offline' as const),
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

  // ========== Recycle Bin APIs ==========

  async listRecycleBin(params: {
    search?: string;
    carrier?: string;
    technology?: string;
    group_id?: string;
    deleted_by?: string;
    page?: number;
    pageSize?: number;
    sortField?: string;
    sortOrder?: string;
  }): Promise<DeviceListResponse> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
      sortField: params.sortField,
      sortOrder: params.sortOrder,
    };
    if (params.search) query.search = params.search;
    if (params.carrier) query.carrier = params.carrier;
    if (params.technology) query.technology = params.technology;
    if (params.group_id) query.group_id = params.group_id;
    if (params.deleted_by) query.deleted_by = params.deleted_by;

    const { data } = await http.get<BackendListResponse<BackendDevice>>('/devices/recycle', {
      params: query,
    });
    return mapListResponse(data);
  },

  async restoreDevices(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.patch<BatchOperationResult>('/devices/recycle/restore', { ids });
    return data;
  },

  async permanentDeleteDevices(ids: string[]): Promise<BatchOperationResult> {
    const { data } = await http.delete<BatchOperationResult>('/devices/recycle/permanent', { data: { ids } });
    return data;
  },

  async getProductClasses(): Promise<string[]> {
    const { data } = await http.get<string[]>('/devices/product-classes');
    return data;
  },
};
