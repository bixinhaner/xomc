import { useQuery } from '@tanstack/react-query';
import { topologyService } from '@/mock/services/topologyService';
import { topologyApi } from '@/services/api/topologyApi';
import { useMock } from '@/services/apiSwitch';

export function useDomains() {
  return useQuery({
    queryKey: ['topology', 'domains'],
    queryFn: () =>
      useMock ? topologyService.getDomains() : topologyApi.getDomains(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useDomainTree() {
  return useQuery({
    queryKey: ['topology', 'domain-tree'],
    queryFn: () =>
      useMock ? topologyService.getDomainTree() : topologyApi.getDomainTree(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useSites(params?: { domainId?: string }) {
  return useQuery({
    queryKey: ['topology', 'sites', params],
    queryFn: () => topologyService.getSites(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useSiteById(id: string) {
  return useQuery({
    queryKey: ['topology', 'sites', 'detail', id],
    queryFn: () => topologyService.getSiteById(id),
    enabled: Boolean(id),
  });
}

export function useTopoNodes(params?: { domainId?: string }) {
  return useQuery({
    queryKey: ['topology', 'nodes', params],
    queryFn: () => topologyService.getTopoNodes(params),
    refetchInterval: 30000,
  });
}

export function useTopoEdges() {
  return useQuery({
    queryKey: ['topology', 'edges'],
    queryFn: () => topologyService.getTopoEdges(),
    refetchInterval: 30000,
  });
}

export function useTopoGraph(params?: { domainId?: string }) {
  return useQuery({
    queryKey: ['topology', 'graph', params],
    queryFn: () => topologyService.getTopoGraph(),
    refetchInterval: 30000,
  });
}

export function useGeoData() {
  return useQuery({
    queryKey: ['topology', 'geo'],
    queryFn: () => topologyService.getGeoData(),
    staleTime: 60 * 1000,
  });
}
