import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ParameterFilter, ParameterUpdateRequest, ParameterSyncOptions } from '../../types/deviceParameter';
import type { PageRequest } from '../../types/pagination';
import { deviceParameterService } from '../../mock/services/deviceParameterService';
import { deviceParameterApi } from '../../services/api/deviceParameterApi';
import { createApiSwitch } from '../../services/apiSwitch';

const api = createApiSwitch(deviceParameterService, deviceParameterApi);

export function useDeviceParameters(
  deviceId: string,
  params?: ParameterFilter & PageRequest
) {
  return useQuery({
    queryKey: ['devices', 'parameters', deviceId, params],
    queryFn: () => api.getParameters(deviceId, params),
    enabled: Boolean(deviceId),
  });
}

export function useDeviceParameterTree(deviceId: string) {
  return useQuery({
    queryKey: ['devices', 'parameter-tree', deviceId],
    queryFn: () => api.getParameterTree(deviceId),
    enabled: Boolean(deviceId),
  });
}

export function useUpdateParameters() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      parameters,
    }: {
      deviceId: string;
      parameters: ParameterUpdateRequest[];
    }) => api.updateParameters(deviceId, parameters),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'children', deviceId],
      });
    },
  });
}

export function useSyncParameters() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      options,
    }: {
      deviceId: string;
      options?: ParameterSyncOptions;
    }) => api.syncParameters(deviceId, options),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'sync-status', deviceId],
      });
    },
  });
}

export function useDiscoverParameters() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (deviceId: string) => api.discoverParameters(deviceId),
    onSuccess: (_result, deviceId) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'sync-status', deviceId],
      });
    },
  });
}

export function useSyncStatus(deviceId: string, enabled: boolean) {
  return useQuery({
    queryKey: ['devices', 'sync-status', deviceId],
    queryFn: () => api.getSyncStatus(deviceId),
    enabled: Boolean(deviceId) && enabled,
    refetchInterval: enabled ? 3000 : false,
  });
}

export function useObjectTree(deviceId: string) {
  return useQuery({
    queryKey: ['devices', 'object-tree', deviceId],
    queryFn: () => api.getObjectTree(deviceId),
    enabled: Boolean(deviceId),
  });
}

export function useDirectChildren(deviceId: string, pathPrefix: string, page: number, pageSize: number) {
  return useQuery({
    queryKey: ['devices', 'children', deviceId, pathPrefix, page, pageSize],
    queryFn: () => api.getDirectChildren(deviceId, pathPrefix, { page, pageSize }),
    enabled: Boolean(deviceId) && Boolean(pathPrefix),
  });
}

export function useParameterSchema(deviceId: string, pathPrefix?: string) {
  return useQuery({
    queryKey: ['devices', 'parameter-schema', deviceId, pathPrefix],
    queryFn: () => api.getParameterSchema(deviceId, pathPrefix),
    enabled: Boolean(deviceId),
  });
}

export function useAddObject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      objectPath,
    }: {
      deviceId: string;
      objectPath: string;
    }) => api.addObject(deviceId, objectPath),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-schema', deviceId],
      });
    },
  });
}

export function useDeleteObject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceId,
      objectPath,
    }: {
      deviceId: string;
      objectPath: string;
    }) => api.deleteObject(deviceId, objectPath),
    onSuccess: (_result, { deviceId }) => {
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-tree', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameters', deviceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['devices', 'parameter-schema', deviceId],
      });
    },
  });
}

export function useSyncConfigFile() {
  return useMutation({
    mutationFn: (deviceId: string) => api.syncConfigFile(deviceId),
  });
}
