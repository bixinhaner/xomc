import http from '../http';
import type { Domain, DomainLevel, Site, TopoNode, TopoEdge, NodeType, NodeStatus, SiteStatus, EdgeStatus } from '@/types/topology';
import type { PageResponse } from '@/types/pagination';

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
};
