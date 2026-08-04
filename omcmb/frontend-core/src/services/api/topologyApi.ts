import http from '../http';
import type { Domain, DomainLevel, Site, TopoNode, TopoEdge, NodeType, NodeStatus, SiteStatus, EdgeStatus, TopoStatistics } from '../../types/topology';
import type { PageResponse } from '../../types/pagination';
import type {
  DeviceGeo,
  DeviceCluster,
  MapStats,
  MapFilterParams,
  DeviceSearchResult,
  MapBounds,
  DeviceStatus,
  DeviceType,
  RawDeviceStatus,
  MapGeoResponse,
  BackendDeviceGeo,
  BackendDeviceCluster,
  BackendMapStats,
  BackendSearchResult,
} from '../../types/map';
import { toDisplayStatus } from '../../types/map';

// Backend device group model
interface BackendDeviceGroup {
  id: string;
  name: string;
  name_i18n?: Record<string, string>;
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
    siteId: bn.site_id || undefined,
    domainId: bn.domain_id || undefined,
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

// Backend topology statistics model (snake_case from Go)
interface BackendTopoStatistics {
  total_nodes: number;
  online_nodes: number;
  offline_nodes: number;
  alarm_nodes: number;
  maintenance_nodes: number;
  total_edges: number;
  active_edges: number;
  inactive_edges: number;
  degraded_edges: number;
  node_type_counts: Record<string, number>;
}

function mapBackendTopoStatistics(bs: BackendTopoStatistics): TopoStatistics {
  return {
    totalNodes: bs.total_nodes,
    onlineNodes: bs.online_nodes,
    offlineNodes: bs.offline_nodes,
    alarmNodes: bs.alarm_nodes,
    maintenanceNodes: bs.maintenance_nodes,
    totalEdges: bs.total_edges,
    activeEdges: bs.active_edges,
    inactiveEdges: bs.inactive_edges,
    degradedEdges: bs.degraded_edges,
    nodeTypeCounts: bs.node_type_counts,
  };
}

function mapGroupToDomain(
  bg: BackendDeviceGroup,
  level: DomainLevel = 1
): Domain {
  return {
    id: bg.id,
    name: bg.name,
    nameI18n: bg.name_i18n,
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

  async getSites(params?: {
    domainId?: string;
    status?: SiteStatus;
    keyword?: string;
    page?: number;
    pageSize?: number;
  }): Promise<PageResponse<Site>> {
    const queryParams: Record<string, unknown> = {};
    if (params?.domainId) queryParams.domain_id = params.domainId;
    if (params?.status) queryParams.status = params.status;
    if (params?.keyword) queryParams.keyword = params.keyword;
    if (params?.page) queryParams.page = params.page;
    if (params?.pageSize) queryParams.page_size = params.pageSize;

    const { data } = await http.get<{ items: BackendSite[]; total: number; page: number; page_size: number }>('/sites', {
      params: queryParams,
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

  async getTopoNodes(params?: {
    domainId?: string;
    nodeType?: NodeType;
    status?: NodeStatus;
    page?: number;
    pageSize?: number;
  }): Promise<PageResponse<TopoNode>> {
    const queryParams: Record<string, unknown> = {};
    if (params?.domainId) queryParams.domain_id = params.domainId;
    if (params?.nodeType) queryParams.node_type = params.nodeType;
    if (params?.status) queryParams.status = params.status;
    if (params?.page) queryParams.page = params.page;
    if (params?.pageSize) queryParams.page_size = params.pageSize;

    const { data } = await http.get<{ items: BackendTopoNode[]; total: number; page: number; page_size: number }>('/topology/nodes', {
      params: queryParams,
    });
    return {
      items: (data.items || []).map(mapBackendTopoNode),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTopoEdges(params?: {
    status?: EdgeStatus;
    page?: number;
    pageSize?: number;
  }): Promise<PageResponse<TopoEdge>> {
    const queryParams: Record<string, unknown> = {};
    if (params?.status) queryParams.status = params.status;
    if (params?.page) queryParams.page = params.page;
    if (params?.pageSize) queryParams.page_size = params.pageSize;

    const { data } = await http.get<{ items: BackendTopoEdge[]; total: number; page: number; page_size: number }>('/topology/edges', {
      params: queryParams,
    });
    return {
      items: (data.items || []).map(mapBackendTopoEdge),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getTopoGraph(params?: { domainId?: string; layoutType?: string; nodeType?: NodeType; status?: NodeStatus; limit?: number }): Promise<{ nodes: TopoNode[]; edges: TopoEdge[]; statistics?: TopoStatistics }> {
    const { data } = await http.get<{ nodes: BackendTopoNode[]; edges: BackendTopoEdge[]; statistics?: BackendTopoStatistics }>('/topology/graph', {
      params: {
        domain_id: params?.domainId,
        layout_type: params?.layoutType,
        node_type: params?.nodeType,
        status: params?.status,
        limit: params?.limit ?? 500, // 默认限制 500 个节点，避免浏览器崩溃
      },
    });
    const result = {
      nodes: (data.nodes || []).map(mapBackendTopoNode),
      edges: (data.edges || []).map(mapBackendTopoEdge),
      statistics: data.statistics ? mapBackendTopoStatistics(data.statistics) : undefined,
    };
    return result;
  },

  /**
   * 创建拓扑节点
   */
  async createTopoNode(data: {
    label: string;
    nodeType: NodeType;
    x: number;
    y: number;
    status?: NodeStatus;
    deviceSn?: string;
    siteId?: string;
    domainId?: string;
  }): Promise<TopoNode> {
    const { data: result } = await http.post<BackendTopoNode>('/topology/nodes', {
      label: data.label,
      node_type: data.nodeType,
      x: data.x,
      y: data.y,
      status: data.status,
      device_sn: data.deviceSn,
      site_id: data.siteId,
      domain_id: data.domainId,
    });
    return mapBackendTopoNode(result);
  },

  /**
   * 更新拓扑节点
   */
  async updateTopoNode(id: string, data: {
    label?: string;
    nodeType?: NodeType;
    x?: number;
    y?: number;
    status?: NodeStatus;
    deviceSn?: string;
    siteId?: string;
    domainId?: string;
  }): Promise<TopoNode> {
    const { data: result } = await http.put<BackendTopoNode>(`/topology/nodes/${id}`, {
      label: data.label,
      node_type: data.nodeType,
      x: data.x,
      y: data.y,
      status: data.status,
      device_sn: data.deviceSn,
      site_id: data.siteId,
      domain_id: data.domainId,
    });
    return mapBackendTopoNode(result);
  },

  /**
   * 删除拓扑节点
   */
  async deleteTopoNode(id: string): Promise<void> {
    await http.delete(`/topology/nodes/${id}`);
  },

  /**
   * 创建拓扑边
   */
  async createTopoEdge(data: {
    sourceId: string;
    targetId: string;
    label?: string;
    status?: EdgeStatus;
  }): Promise<TopoEdge> {
    const { data: result } = await http.post<BackendTopoEdge>('/topology/edges', {
      source_id: data.sourceId,
      target_id: data.targetId,
      label: data.label,
      status: data.status,
    });
    return mapBackendTopoEdge(result);
  },

  /**
   * 更新拓扑边
   */
  async updateTopoEdge(id: string, data: {
    sourceId?: string;
    targetId?: string;
    label?: string;
    status?: EdgeStatus;
  }): Promise<TopoEdge> {
    const { data: result } = await http.put<BackendTopoEdge>(`/topology/edges/${id}`, {
      source_id: data.sourceId,
      target_id: data.targetId,
      label: data.label,
      status: data.status,
    });
    return mapBackendTopoEdge(result);
  },

  /**
   * 删除拓扑边
   */
  async deleteTopoEdge(id: string): Promise<void> {
    await http.delete(`/topology/edges/${id}`);
  },

  /**
   * 批量从设备生成拓扑节点
   */
  async batchCreateTopoNodes(params?: {
    siteId?: string;
    domainId?: string;
    nodeTypes?: NodeType[];
    limit?: number;
  }): Promise<{ created: number; nodes: TopoNode[] }> {
    const { data } = await http.post<{ created: number; nodes: BackendTopoNode[] }>('/topology/nodes/batch', {
      site_id: params?.siteId,
      domain_id: params?.domainId,
      node_types: params?.nodeTypes,
      limit: params?.limit,
    });
    return {
      created: data.created,
      nodes: (data.nodes || []).map(mapBackendTopoNode),
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
   * @param signal 可选 AbortSignal，由 React Query 注入；queryKey 变化时自动取消未完成请求
   */
  async getDevicesGeo(
    params?: MapFilterParams,
    signal?: AbortSignal,
  ): Promise<MapGeoResponse> {
    const { data } = await http.get<{
      items: BackendDeviceGeo[];
      total: number;
      has_more?: boolean;
      complete?: boolean;
      coordinate_count?: number;
    }>('/devices/geo', {
      params: {
        group_ids: params?.groupIds?.join(','),
        status: params?.status?.join(','),
        type: params?.type?.join(','),
        keyword: params?.keyword,
        bounds: params?.bounds,
        page: params?.page,
        page_size: params?.pageSize,
        ue_count_max: params?.ueCountMax,
      },
      signal,
    });
    return {
      items: (data.items || []).map(mapBackendDeviceGeo),
      total: data.total,
      hasMore: data.has_more ?? false,
      complete: data.complete ?? false,
      coordinateCount: data.coordinate_count ?? data.total,
    };
  },

  /**
   * 获取聚合数据
   */
  async getAggregation(
    params: {
      bounds: MapBounds;
      zoom: number;
      gridSize?: number;
      filters?: MapFilterParams;
    },
    signal?: AbortSignal,
  ): Promise<{ clusters: DeviceCluster[] }> {
    const { data } = await http.post<{ clusters: BackendDeviceCluster[] }>(
      '/devices/geo/aggregate',
      {
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
      },
      { signal },
    );
    return {
      clusters: (data.clusters || []).map(mapBackendCluster),
    };
  },

  /**
   * 获取地图统计数据
   */
  async getMapStats(
    params?: { groupIds?: string[]; status?: string[]; bounds?: string },
    signal?: AbortSignal,
  ): Promise<MapStats> {
    const { data } = await http.get<BackendMapStats>('/devices/geo/stats', {
      params: {
        group_ids: params?.groupIds?.join(','),
        status: params?.status?.join(','),
        bounds: params?.bounds,
      },
      signal,
    });
    const result = mapBackendStats(data);
    return result;
  },

  /**
   * 搜索设备（节点查找）
   */
  async searchDevices(keyword: string, signal?: AbortSignal): Promise<DeviceSearchResult[]> {
    const { data } = await http.get<{ items: BackendSearchResult[] }>('/devices/search', {
      params: { keyword },
      signal,
      timeout: 10_000,
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
    status: toDisplayStatus(bd.status as RawDeviceStatus),
    type: bd.type as DeviceType | undefined,
    groupId: bd.group_id,
    groupName: bd.group_name,
    address: bd.address,
    alarmCount: bd.alarm_count,
    ip_address: bd.ip_address,
    mac: bd.mac,
    pci: bd.pci,
    device_name: bd.device_name,
    ueCount: bd.ue_count,
    highestAlarmSeverity: bd.highest_alarm_severity ?? null,
    highestSeverityAlarmCount: bd.highest_severity_alarm_count ?? 0,
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
  // 处理后端状态统计，兼容新旧两种格式
  // 新格式：{ onlineActive: number, onlineInactive: number, offline: number }
  // 旧格式：{ online: number, offline: number } (后端统计不完整)
  let statusCount: Record<DeviceStatus, number>;

  // 防御性空值处理：后端可能在极端情况下返回 null 或缺省字段
  const rawCount = (bs.status_count ?? {}) as Record<string, number>;
  if ('onlineActive' in rawCount) {
    // 新格式：直接使用
    statusCount = rawCount as unknown as Record<DeviceStatus, number>;
  } else if ('online' in rawCount) {
    // 旧格式：后端统计不完整，需要计算
    const onlineCount = rawCount.online || 0;
    const offlineCount = rawCount.offline || 0;
    const total = bs.total;
    const sum = onlineCount + offlineCount;

    if (sum < total && onlineCount > 0) {
      // 后端统计不完整：registered设备被遗漏
      // 计算缺失的设备数量（这些是registered状态的设备）
      const registeredCount = total - onlineCount - offlineCount;

      statusCount = {
        onlineActive: onlineCount,
        onlineInactive: registeredCount,
        offline: offlineCount,
      };
    } else if (sum === total) {
      // 统计完整但无法区分active/registered
      // 将online全部当作onlineInactive
      statusCount = {
        onlineActive: 0,
        onlineInactive: onlineCount,
        offline: offlineCount,
      };
    } else {
      // 异常情况：所有设备计为offline
      statusCount = {
        onlineActive: 0,
        onlineInactive: 0,
        offline: total,
      };
    }
  } else {
    // 默认：初始化为0
    statusCount = {
      onlineActive: 0,
      onlineInactive: 0,
      offline: 0,
    };
  }

  return {
    total: bs.total,
    statusCount,
    alarmCount: bs.alarm_count,
    typeCount: bs.type_count as Record<DeviceType, number> | undefined,
    viewportCount: bs.viewport_count,
    center: bs.center ? { lat: bs.center.lat, lng: bs.center.lng } : undefined,
    ueZeroCount: bs.ue_zero_count,
  };
}

function mapBackendSearchResult(bs: BackendSearchResult): DeviceSearchResult {
  return {
    id: bs.id,
    name: bs.name,
    sn: bs.sn,
    status: toDisplayStatus(bs.status as RawDeviceStatus),
    longitude: bs.longitude,
    latitude: bs.latitude,
    groupId: bs.group_id,
    groupName: bs.group_name,
    ip_address: bs.ip_address,
    mac: bs.mac,
    pci: bs.pci,
    device_name: bs.device_name,
    ueCount: bs.ue_count,
    alarmCount: bs.alarm_count,
    highestAlarmSeverity: bs.highest_alarm_severity ?? null,
    highestSeverityAlarmCount: bs.highest_severity_alarm_count ?? 0,
  };
}
