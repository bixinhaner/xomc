import type { Domain, Site, TopoNode, TopoEdge } from '../../types/topology';
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

  async getTopoGraph(): Promise<{ nodes: TopoNode[]; edges: TopoEdge[] }> {
    await delay(100, 200);
    return {
      nodes: mockTopoNodes,
      edges: mockTopoEdges,
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
