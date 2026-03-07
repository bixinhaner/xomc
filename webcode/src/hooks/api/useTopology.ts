import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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

// ── Group CRUD hooks ──

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; parent_id?: string; description?: string }) =>
      useMock
        ? topologyService.getDomains().then(() => data as any)
        : topologyApi.createGroup(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domain-tree'] });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name?: string; description?: string } }) =>
      useMock
        ? topologyService.getDomains().then(() => data as any)
        : topologyApi.updateGroup(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domain-tree'] });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock
        ? topologyService.getDomains().then(() => undefined as any)
        : topologyApi.deleteGroup(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domain-tree'] });
    },
  });
}

export function useGroupDevices(groupId: string) {
  return useQuery({
    queryKey: ['topology', 'group-devices', groupId],
    queryFn: () => topologyApi.getGroupDevices(groupId),
    enabled: Boolean(groupId),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAddDeviceToGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, deviceId }: { groupId: string; deviceId: string }) =>
      topologyApi.addDeviceToGroup(groupId, deviceId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ['topology', 'group-devices', variables.groupId],
      });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
    },
  });
}

export function useRemoveDeviceFromGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, deviceId }: { groupId: string; deviceId: string }) =>
      topologyApi.removeDeviceFromGroup(groupId, deviceId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ['topology', 'group-devices', variables.groupId],
      });
      void queryClient.invalidateQueries({ queryKey: ['topology', 'domains'] });
    },
  });
}
