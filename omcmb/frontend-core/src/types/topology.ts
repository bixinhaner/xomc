export type DomainLevel = 1 | 2 | 3 | 4 | 5;
export type SiteStatus = 'active' | 'inactive' | 'maintenance';
export type NodeStatus = 'online' | 'offline' | 'alarm' | 'maintenance';
export type EdgeStatus = 'active' | 'inactive' | 'degraded';
export type NodeType = 'eNB' | 'gNB' | 'CPE' | 'eGW' | 'domain' | 'site' | 'router' | 'switch';

export interface Domain {
  id: string;
  name: string;
  level: DomainLevel;
  parentId: string | null;
  children?: Domain[];
  deviceCount: number;
}

export interface Site {
  id: string;
  name: string;
  domainId: string;
  address: string;
  longitude: number;
  latitude: number;
  deviceCount: number;
  status: SiteStatus;
}

export interface TopoNode {
  id: string;
  label: string;
  type: NodeType;
  x: number;
  y: number;
  status: NodeStatus;
  deviceSn?: string;
  siteId?: string;
  domainId?: string;
}

export interface TopoEdge {
  id: string;
  source: string;
  target: string;
  label?: string;
  status: EdgeStatus;
}

export interface TopoStatistics {
  totalNodes: number;
  onlineNodes: number;
  offlineNodes: number;
  alarmNodes: number;
  maintenanceNodes: number;
  totalEdges: number;
  activeEdges: number;
  inactiveEdges: number;
  degradedEdges: number;
  nodeTypeCounts: Record<string, number>;
}

export interface TopoGraph {
  nodes: TopoNode[];
  edges: TopoEdge[];
  statistics?: TopoStatistics;
}
