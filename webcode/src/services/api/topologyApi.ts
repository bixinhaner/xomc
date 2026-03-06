import http from '../http';
import type { Domain, DomainLevel, Site, TopoNode, TopoEdge } from '@/types/topology';
import { topologyService } from '@/mock/services/topologyService';

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

  // No backend equivalents — delegate to mock
  getSites: topologyService.getSites.bind(topologyService),
  getSiteById: topologyService.getSiteById.bind(topologyService),
  getTopoNodes: topologyService.getTopoNodes.bind(topologyService),
  getTopoEdges: topologyService.getTopoEdges.bind(topologyService),
  getTopoGraph: topologyService.getTopoGraph.bind(topologyService),
  getGeoData: topologyService.getGeoData.bind(topologyService),
};
