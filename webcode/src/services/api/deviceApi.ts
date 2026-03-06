import http from '../http';
import type { Device, NE, DeviceFilter, DeviceGroup } from '@/types/device';
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
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Map backend device status to frontend connStatus
function mapStatus(status: string): Device['connStatus'] {
  switch (status) {
    case 'online':
      return 'online';
    case 'offline':
    case 'registered':
    case 'unregistered':
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
  };
}

function mapListResponse(resp: BackendListResponse<BackendDevice>): PageResponse<Device> {
  return {
    items: (resp.items || []).map(mapBackendDevice),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

export const deviceApi = {
  async getList(params: DeviceFilter & PageRequest): Promise<PageResponse<Device>> {
    // Map frontend filter fields to backend query params
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
      sortField: params.sortField,
      sortOrder: params.sortOrder,
    };

    if (params.name) query.search = params.name;
    if (params.sn) query.sn = params.sn;
    if (params.vendor) query.oui = params.vendor;
    if (params.networkType) query.technology = params.networkType;
    if (params.connStatus) query.status = params.connStatus;

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
    // Sprint 4 — device CRUD; placeholder for now
    throw new Error('Device create not yet implemented (Sprint 4)');
  },

  async update(id: string, data: Partial<Device>): Promise<Device> {
    // Sprint 4 — device CRUD; placeholder for now
    throw new Error('Device update not yet implemented (Sprint 4)');
  },

  async delete(ids: string[]): Promise<void> {
    // Sprint 4 — device CRUD; placeholder for now
    throw new Error('Device delete not yet implemented (Sprint 4)');
  },

  async getGroups(): Promise<DeviceGroup[]> {
    const { data } = await http.get<BackendListResponse<{
      id: string;
      name: string;
      parent_id: string | null;
      device_count: number;
      description: string;
    }>>('/groups');
    return (data.items || []).map((g) => ({
      id: g.id,
      name: g.name,
      parentId: g.parent_id,
      deviceCount: g.device_count,
      description: g.description,
    }));
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
