import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { DeviceFilter } from '@/types/device';
import type { PageRequest } from '@/types/pagination';
import { deviceService } from '@/mock/services/deviceService';
import { deviceApi } from '@/services/api/deviceApi';
import { createApiSwitch } from '@/services/apiSwitch';

const api = createApiSwitch(deviceService, deviceApi);

export function useDeviceList(params: DeviceFilter & PageRequest) {
  return useQuery({
    queryKey: ['devices', 'list', params],
    queryFn: () => api.getList(params),
  });
}

export function useDeviceById(id: string) {
  return useQuery({
    queryKey: ['devices', 'detail', id],
    queryFn: () => api.getById(id),
    enabled: Boolean(id),
  });
}

export function useDeviceBySn(sn: string) {
  return useQuery({
    queryKey: ['devices', 'sn', sn],
    queryFn: () => api.getBySn(sn),
    enabled: Boolean(sn),
  });
}

export function useDeviceGroups() {
  return useQuery({
    queryKey: ['devices', 'groups'],
    queryFn: () => api.getGroups(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; parent_id?: string; remark?: string }) =>
      api.createGroup(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name?: string; remark?: string } }) =>
      api.updateGroup(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteGroup(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useMoveDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { device_ids: string[]; target_group_id: string }) =>
      api.moveDevices(params),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useAddDevicesToGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, deviceIds }: { groupId: string; deviceIds: string[] }) =>
      api.addDevicesToGroup(groupId, deviceIds),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'groups'] });
    },
  });
}

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof api.create>[0]) =>
      api.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices'] });
    },
  });
}

export function useUpdateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof api.update>[1] }) =>
      api.update(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useDeleteDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.delete(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useNEList(params: { keyword?: string } & PageRequest) {
  return useQuery({
    queryKey: ['devices', 'ne-list', params],
    queryFn: () => api.getNEList(params),
  });
}

export function useNEBySn(sn: string) {
  return useQuery({
    queryKey: ['devices', 'ne-sn', sn],
    queryFn: () => api.getNEBySn(sn),
    enabled: Boolean(sn),
  });
}

export function useRebootDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.reboot(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useBatchRebootDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      const results = await Promise.allSettled(ids.map((id) => api.reboot(id)));
      const failed = results.filter((r) => r.status === 'rejected');
      if (failed.length > 0) {
        throw new Error(`${failed.length}/${ids.length} devices failed to reboot`);
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}
