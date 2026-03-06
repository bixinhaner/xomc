import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { DeviceFilter } from '@/types/device';
import type { PageRequest } from '@/types/pagination';
import { deviceService } from '@/mock/services/deviceService';

export function useDeviceList(params: DeviceFilter & PageRequest) {
  return useQuery({
    queryKey: ['devices', 'list', params],
    queryFn: () => deviceService.getList(params),
  });
}

export function useDeviceById(id: string) {
  return useQuery({
    queryKey: ['devices', 'detail', id],
    queryFn: () => deviceService.getById(id),
    enabled: Boolean(id),
  });
}

export function useDeviceBySn(sn: string) {
  return useQuery({
    queryKey: ['devices', 'sn', sn],
    queryFn: () => deviceService.getBySn(sn),
    enabled: Boolean(sn),
  });
}

export function useDeviceGroups() {
  return useQuery({
    queryKey: ['devices', 'groups'],
    queryFn: () => deviceService.getGroups(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof deviceService.create>[0]) =>
      deviceService.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices'] });
    },
  });
}

export function useUpdateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof deviceService.update>[1] }) =>
      deviceService.update(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useDeleteDevices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => deviceService.delete(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
    },
  });
}

export function useNEList(params: { keyword?: string } & PageRequest) {
  return useQuery({
    queryKey: ['devices', 'ne-list', params],
    queryFn: () => deviceService.getNEList(params),
  });
}

export function useNEBySn(sn: string) {
  return useQuery({
    queryKey: ['devices', 'ne-sn', sn],
    queryFn: () => deviceService.getNEBySn(sn),
    enabled: Boolean(sn),
  });
}
