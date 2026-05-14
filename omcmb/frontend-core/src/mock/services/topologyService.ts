import type { Domain, Site, TopoNode, TopoEdge, TopoStatistics, TopoGraph } from '../../types/topology';
import { mockDomains, mockSites, mockTopoNodes, mockTopoEdges } from '../data/topology';
import { delay } from '../utils';

export const topologyService = {
  async getDomains(): Promise<Domain[]> {
    await delay(80, 150);
    return mockDomains;
  },

  async getDomainTree(): Promise<Domain[]> {
    await delay(100, 200);
    const roots = mockDomains.filter((d) => d.parentId === null);
    function buildTree(parent: Domain): Domain {
      const children = mockDomains.filter((d) => d.parentId === parent.id);
      return {
        ...parent,
        children: children.length > 0 ? children.map(buildTree) : undefined,
      };
    }
    return roots.map(buildTree);
  },

  async getSites(params?: { domainId?: string }): Promise<Site[]> {
    await delay(80, 150);
    let filtered = [...mockSites];
    if (params?.domainId) {
      filtered = filtered.filter((s) => s.domainId === params.domainId);
    }
    return filtered;
  },

  async getSiteById(id: string): Promise<Site | null> {
    await delay(80, 150);
    return mockSites.find((s) => s.id === id) ?? null;
  },

  async getTopoNodes(params?: { domainId?: string }): Promise<TopoNode[]> {
    await delay(80, 150);
    void params;
    return mockTopoNodes;
  },

  async getTopoEdges(): Promise<TopoEdge[]> {
    await delay(80, 150);
    return mockTopoEdges;
  },

  async getTopoGraph(): Promise<TopoGraph> {
    await delay(100, 200);
    // Calculate mock statistics
    const nodeTypeCounts: Record<string, number> = {};
    mockTopoNodes.forEach((n) => {
      nodeTypeCounts[n.type] = (nodeTypeCounts[n.type] || 0) + 1;
    });
    const statistics: TopoStatistics = {
      totalNodes: mockTopoNodes.length,
      onlineNodes: mockTopoNodes.filter((n) => n.status === 'online').length,
      offlineNodes: mockTopoNodes.filter((n) => n.status === 'offline').length,
      alarmNodes: mockTopoNodes.filter((n) => n.status === 'alarm').length,
      maintenanceNodes: mockTopoNodes.filter((n) => n.status === 'maintenance').length,
      totalEdges: mockTopoEdges.length,
      activeEdges: mockTopoEdges.filter((e) => e.status === 'active').length,
      inactiveEdges: mockTopoEdges.filter((e) => e.status === 'inactive').length,
      degradedEdges: mockTopoEdges.filter((e) => e.status === 'degraded').length,
      nodeTypeCounts: nodeTypeCounts,
    };
    return {
      nodes: mockTopoNodes,
      edges: mockTopoEdges,
      statistics,
    };
  },

  async getGeoData(): Promise<{ sites: Site[]; nodes: TopoNode[] }> {
    await delay(100, 200);
    return {
      sites: mockSites,
      nodes: mockTopoNodes.filter((n) => n.deviceSn),
    };
  },
};
