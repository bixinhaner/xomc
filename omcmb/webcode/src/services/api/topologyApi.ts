import http from '../http';
import type { Domain, DomainLevel, Site, TopoNode, TopoEdge, NodeType, NodeStatus, SiteStatus, EdgeStatus } from '@/types/topology';
import type { PageResponse } from '@/types/pagination';
import type {
  DeviceGeo,
  DeviceCluster,
  MapStats,
  MapFilterParams,
  DeviceSearchResult,
  MapBounds,
  DeviceStatus,
  DeviceType,
  BackendDeviceGeo,
  BackendDeviceCluster,
  BackendMapStats,
  BackendSearchResult,
} from '@/types/map';
import { toDisplayStatus } from '@/types/map';

// Backend device group model
interface BackendDeviceGroup {
  id: string;
  name: string;
  parent_id: string | null;
  carrier: string;
  description: string;
  sort_order: number;
  children?: BackendDeviceGroup[];
  created_at: string;
  updated_at: string;
}

// Backend site model (snake_case from Go)
interface BackendSite {
  id: string;
  name: string;
  domain_id: string | null;
  address: string;
  longitude: number | null;
  latitude: number | null;
  device_count: number;
  status: SiteStatus;
  created_at: string;
  updated_at: string;
}

// Backend topo node model (snake_case from Go)
interface BackendTopoNode {
  id: string;
  label: string;
  node_type: string;
  x: number;
  y: number;
  status: NodeStatus;
  device_sn: string;
  site_id: string | null;
  domain_id: string | null;
  created_at: string;
  updated_at: string;
}

// Backend topo edge model (snake_case from Go)
interface BackendTopoEdge {
  id: string;
  source_id: string;
  target_id: string;
  label: string;
  status: EdgeStatus;
  created_at: string;
}

function mapBackendSite(bs: BackendSite): Site {
  return {
    id: bs.id,
    name: bs.name,
    domainId: bs.domain_id ?? '',
    address: bs.address ?? '',
    longitude: bs.longitude ?? 0,
    latitude: bs.latitude ?? 0,
    deviceCount: bs.device_count,
    status: bs.status,
  };
}

function mapBackendTopoNode(bn: BackendTopoNode): TopoNode {
  return {
    id: bn.id,
    label: bn.label,
    type: bn.node_type as NodeType,
    x: bn.x,
    y: bn.y,
    status: bn.status,
    deviceSn: bn.device_sn || undefined,
  };
}

function mapBackendTopoEdge(be: BackendTopoEdge): TopoEdge {
  return {
    id: be.id,
    source: be.source_id,
    target: be.target_id,
    label: be.label || undefined,
    status: be.status,
  };
}

function mapGroupToDomain(
  bg: BackendDeviceGroup,
  level: DomainLevel = 1
): Domain {
  return {
    id: bg.id,
    name: bg.name,
    level,
    parentId: bg.parent_id,
    children: bg.children?.map((c) =>
      mapGroupToDomain(c, Math.min(level + 1, 5) as DomainLevel)
    ),
    deviceCount: 0, // not available from group endpoint directly
  };
}

function flattenTree(
  groups: BackendDeviceGroup[],
  level: DomainLevel = 1
): Domain[] {
  const result: Domain[] = [];
  for (const g of groups) {
    result.push(mapGroupToDomain(g, level));
    if (g.children?.length) {
      result.push(
        ...flattenTree(
          g.children,
          Math.min(level + 1, 5) as DomainLevel
        )
      );
    }
  }
  return result;
}

export const topologyApi = {
  async getDomains(): Promise<Domain[]> {
    const { data } = await http.get<{ items: BackendDeviceGroup[] }>('/groups');
    return flattenTree(data.items || []);
  },

  async getDomainTree(): Promise<Domain[]> {
    const { data } = await http.get<{ items: BackendDeviceGroup[] }>('/groups');
    return (data.items || []).map((g) => mapGroupToDomain(g));
  },

  async createGroup(data: { name: string; parent_id?: string; description?: string }): Promise<BackendDeviceGroup> {
    const { data: result } = await http.post<BackendDeviceGroup>('/groups', data);
    return result;
  },

  async updateGroup(id: string, data: { name?: string; description?: string }): Promise<BackendDeviceGroup> {
    const { data: result } = await http.put<BackendDeviceGroup>(`/groups/${id}`, data);
    return result;
  },

  async deleteGroup(id: string): Promise<void> {
    await http.delete(`/groups/${id}`);
  },

  async addDeviceToGroup(groupId: string, deviceId: string): Promise<void> {
    await http.post(`/groups/${groupId}/devices`, { device_id: deviceId });
  },

  async removeDeviceFromGroup(groupId: string, deviceId: string): Promise<void> {
    await http.delete(`/groups/${groupId}/devices/${deviceId}`);
  },

  async getGroupDevices(groupId: string, params?: Record<string, unknown>): Promise<PageResponse<Record<string, unknown>>> {
    const { data } = await http.get<{ items: Record<string, unknown>[]; total: number; page: number; page_size: number }>(`/groups/${groupId}/devices`, { params });
    return {
      items: data.items || [],
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getSites(params?: { domainId?: string }): Promise<PageResponse<Site>> {
    const { data } = await http.get<{ items: BackendSite[]; total: number; page: number; page_size: number }>('/sites', {
      params: { domain_id: params?.domainId },
    });
    return {
      items: (data.items || []).map(mapBackendSite),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getSiteById(id: string): Promise<Site | null> {
    try {
      const { data } = await http.get<BackendSite>(`/sites/${id}`);
      return mapBackendSite(data);
    } catch {
      return null;
    }
  },

  async getTopoNodes(params?: { domainId?: string }): Promise<PageResponse<TopoNode>> {
    const { data } = await http.get<{ items: BackendTopoNode[]; total: number; page: number; page_size: number }>('/topology/nodes', {
      params: { domain_id: params?.domainId },
    });
    return {
      items: (data.items || []).map(mapBackendTopoNode),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTopoEdges(): Promise<PageResponse<TopoEdge>> {
    const { data } = await http.get<{ items: BackendTopoEdge[]; total: number; page: number; page_size: number }>('/topology/edges');
    return {
      items: (data.items || []).map(mapBackendTopoEdge),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTopoGraph(params?: { domainId?: string }): Promise<{ nodes: TopoNode[]; edges: TopoEdge[] }> {
    const { data } = await http.get<{ nodes: BackendTopoNode[]; edges: BackendTopoEdge[] }>('/topology/graph', {
      params: { domain_id: params?.domainId },
    });
    return {
      nodes: (data.nodes || []).map(mapBackendTopoNode),
      edges: (data.edges || []).map(mapBackendTopoEdge),
    };
  },

  async getGeoData(): Promise<{ sites: Site[]; nodes: TopoNode[] }> {
    const { data } = await http.get<{ sites: BackendSite[]; nodes: BackendTopoNode[] }>('/topology/geo');
    return {
      sites: (data.sites || []).map(mapBackendSite),
      nodes: (data.nodes || []).map(mapBackendTopoNode),
    };
  },

  // ============ Map API Methods ============

  /**
   * 获取设备地理数据（支持筛选）
   */
  async getDevicesGeo(params?: MapFilterParams): Promise<{ items: DeviceGeo[]; total: number }> {
    const { data } = await http.get<{ items: BackendDeviceGeo[]; total: number }>('/devices/geo', {
      params: {
        group_ids: params?.groupIds?.join(','),
        status: params?.status?.join(','),
        type: params?.type?.join(','),
        keyword: params?.keyword,
        bounds: params?.bounds,
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    return {
      items: (data.items || []).map(mapBackendDeviceGeo),
      total: data.total,
    };
  },

  /**
   * 获取聚合数据
   */
  async getAggregation(params: {
    bounds: MapBounds;
    zoom: number;
    gridSize?: number;
    filters?: MapFilterParams;
  }): Promise<{ clusters: DeviceCluster[] }> {
    const { data } = await http.post<{ clusters: BackendDeviceCluster[] }>('/devices/geo/aggregate', {
      bounds: {
        min_lng: params.bounds.minLng,
        max_lng: params.bounds.maxLng,
        min_lat: params.bounds.minLat,
        max_lat: params.bounds.maxLat,
      },
      zoom: params.zoom,
      grid_size: params.gridSize || 50,
      filters: {
        group_ids: params.filters?.groupIds,
        status: params.filters?.status,
      },
    });
    return {
      clusters: (data.clusters || []).map(mapBackendCluster),
    };
  },

  /**
   * 获取地图统计数据
   */
  async getMapStats(params?: { groupIds?: string[]; bounds?: string }): Promise<MapStats> {
    const { data } = await http.get<BackendMapStats>('/devices/geo/stats', {
      params: {
        group_ids: params?.groupIds?.join(','),
        bounds: params?.bounds,
      },
    });
    return mapBackendStats(data);
  },

  /**
   * 搜索设备（节点查找）
   */
  async searchDevices(keyword: string): Promise<DeviceSearchResult[]> {
    const { data } = await http.get<{ items: BackendSearchResult[] }>('/devices/search', {
      params: { keyword },
    });
    return (data.items || []).map(mapBackendSearchResult);
  },

  /**
   * 获取设备组树（用于地图筛选）
   */
  async getGroupTree(): Promise<Domain[]> {
    const { data } = await http.get<{ items: BackendDeviceGroup[] }>('/groups');
    return (data.items || []).map((g) => mapGroupToDomain(g));
  },
};

// ============ Map Backend Type Mappers ============

function mapBackendDeviceGeo(bd: BackendDeviceGeo): DeviceGeo {
  return {
    id: bd.id,
    name: bd.name,
    sn: bd.sn,
    longitude: bd.longitude,
    latitude: bd.latitude,
    status: toDisplayStatus(bd.status),
    type: bd.type as DeviceType | undefined,
    groupId: bd.group_id,
    groupName: bd.group_name,
    address: bd.address,
    alarmCount: bd.alarm_count,
  };
}

function mapBackendCluster(bc: BackendDeviceCluster): DeviceCluster {
  return {
    id: bc.id,
    longitude: bc.longitude,
    latitude: bc.latitude,
    count: bc.count,
    statusCount: bc.status_count as Record<DeviceStatus, number>,
    alarmCount: bc.alarm_count,
    bounds: bc.bounds ? {
      minLng: bc.bounds.min_lng,
      maxLng: bc.bounds.max_lng,
      minLat: bc.bounds.min_lat,
      maxLat: bc.bounds.max_lat,
    } : undefined,
  };
}

function mapBackendStats(bs: BackendMapStats): MapStats {
  return {
    total: bs.total,
    statusCount: bs.status_count as Record<DeviceStatus, number>,
    alarmCount: bs.alarm_count,
    typeCount: bs.type_count as Record<DeviceType, number> | undefined,
    viewportCount: bs.viewport_count,
    center: bs.center ? { lat: bs.center.lat, lng: bs.center.lng } : undefined,
  };
}

function mapBackendSearchResult(bs: BackendSearchResult): DeviceSearchResult {
  return {
    id: bs.id,
    name: bs.name,
    sn: bs.sn,
    status: toDisplayStatus(bs.status),
    longitude: bs.longitude,
    latitude: bs.latitude,
    groupName: bs.group_name,
  };
}
