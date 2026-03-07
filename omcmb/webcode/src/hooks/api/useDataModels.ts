import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PageRequest } from '@/types/pagination';
import {
  datamodelApi,
  type CreateDataModelRequest,
  type UpdateDataModelRequest,
  type CreateOUIRequest,
} from '@/services/api/datamodelApi';

export function useDataModelList(
  params: {
    carrier?: string;
    technology?: string;
    oui?: string;
    productClass?: string;
    scope?: string;
    status?: string;
  } & PageRequest
) {
  return useQuery({
    queryKey: ['datamodels', 'list', params],
    queryFn: () => datamodelApi.getDataModels(params),
  });
}

export function useDataModel(id: string) {
  return useQuery({
    queryKey: ['datamodels', 'detail', id],
    queryFn: () => datamodelApi.getDataModel(id),
    enabled: Boolean(id),
  });
}

export function useDataModelStats() {
  return useQuery({
    queryKey: ['datamodels', 'statistics'],
    queryFn: () => datamodelApi.getStatistics(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCreateDataModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateDataModelRequest) => datamodelApi.createDataModel(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['datamodels'] });
    },
  });
}

export function useUpdateDataModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateDataModelRequest }) =>
      datamodelApi.updateDataModel(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['datamodels'] });
    },
  });
}

export function useDeleteDataModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => datamodelApi.deleteDataModel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['datamodels'] });
    },
  });
}

export function useActivateDataModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => datamodelApi.activateDataModel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['datamodels'] });
    },
  });
}

export function useDeprecateDataModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => datamodelApi.deprecateDataModel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['datamodels'] });
    },
  });
}

export function useRefreshDataModelCache() {
  return useMutation({
    mutationFn: () => datamodelApi.refreshCache(),
  });
}

// --- OUI hooks ---

export function useOUIList() {
  return useQuery({
    queryKey: ['oui', 'list'],
    queryFn: () => datamodelApi.getOUIList(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useCreateOUI() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateOUIRequest) => datamodelApi.createOUI(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['oui'] });
    },
  });
}
