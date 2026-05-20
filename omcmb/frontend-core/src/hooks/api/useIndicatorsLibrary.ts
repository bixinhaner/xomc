import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { indicatorLibraryApi } from '../../services/api/indicatorLibraryApi';
import { indicatorLibraryService } from '../../mock/services/indicatorLibraryService';
import { createApiSwitch } from '../../services/apiSwitch';
import type {
  DeviceType,
  IndicatorListFilter,
  CreateIndicatorInput,
  UpdateIndicatorInput,
  CreateGroupInput,
  UpdateGroupInput,
  EnabledIndicatorsRequest,
  UnitInput,
} from '../../types/indicatorLibrary';

const api = createApiSwitch(indicatorLibraryService, indicatorLibraryApi);

const IL_KEY = ['indicator-library'] as const;

export function useIndicatorList(deviceType: DeviceType, filter?: IndicatorListFilter) {
  return useQuery({
    queryKey: [...IL_KEY, 'list', deviceType, filter ?? {}],
    queryFn: () => api.list(deviceType, filter),
  });
}

export function usePlatformList(deviceType: DeviceType) {
  return useQuery({
    queryKey: [...IL_KEY, 'platforms', deviceType],
    queryFn: () => api.listPlatforms(deviceType),
    staleTime: 5 * 60 * 1000,
  });
}

export function useIndicatorDetail(deviceType: DeviceType, id: string | undefined) {
  return useQuery({
    queryKey: [...IL_KEY, 'detail', deviceType, id],
    queryFn: () => api.get(deviceType, id as string),
    enabled: Boolean(id),
  });
}

export function useCreateIndicator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, input }: { deviceType: DeviceType; input: CreateIndicatorInput }) =>
      api.create(deviceType, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useUpdateIndicator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      id,
      input,
    }: {
      deviceType: DeviceType;
      id: string;
      input: UpdateIndicatorInput;
    }) => api.update(deviceType, id, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'detail', vars.deviceType, vars.id] });
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useDeleteIndicator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, id }: { deviceType: DeviceType; id: string }) =>
      api.delete(deviceType, id),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useFormulas(deviceType: DeviceType, indicatorId: string | undefined) {
  return useQuery({
    queryKey: [...IL_KEY, 'formulas', deviceType, indicatorId],
    queryFn: () => api.listFormulas(deviceType, indicatorId as string),
    enabled: Boolean(indicatorId),
  });
}

export function useUpsertFormula() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      indicatorId,
      platform,
      formula,
    }: {
      deviceType: DeviceType;
      indicatorId: string;
      platform: string;
      formula: string;
    }) => api.upsertFormula(deviceType, indicatorId, platform, formula),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({
        queryKey: [...IL_KEY, 'formulas', vars.deviceType, vars.indicatorId],
      });
    },
  });
}

export function useDeleteFormula() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      indicatorId,
      platform,
    }: {
      deviceType: DeviceType;
      indicatorId: string;
      platform: string;
    }) => api.deleteFormula(deviceType, indicatorId, platform),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({
        queryKey: [...IL_KEY, 'formulas', vars.deviceType, vars.indicatorId],
      });
    },
  });
}

export function useIndicatorGroups(deviceType: DeviceType, operatorCode?: string) {
  return useQuery({
    queryKey: [...IL_KEY, 'groups', deviceType, operatorCode ?? ''],
    queryFn: () => api.listGroups(deviceType, operatorCode),
  });
}

export function useCreateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, input }: { deviceType: DeviceType; input: CreateGroupInput }) =>
      api.createGroup(deviceType, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'groups', vars.deviceType] });
    },
  });
}

export function useUpdateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      id,
      input,
    }: {
      deviceType: DeviceType;
      id: string;
      input: UpdateGroupInput;
    }) => api.updateGroup(deviceType, id, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'groups', vars.deviceType] });
    },
  });
}

export function useDeleteGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, id }: { deviceType: DeviceType; id: string }) =>
      api.deleteGroup(deviceType, id),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'groups', vars.deviceType] });
    },
  });
}

export function useEnabledIndicators(deviceType: DeviceType, operatorCode: string) {
  return useQuery({
    queryKey: [...IL_KEY, 'enabled', deviceType, operatorCode],
    queryFn: () => api.listEnabled(deviceType, operatorCode),
    enabled: Boolean(operatorCode),
  });
}

export function useSetEnabledIndicators() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (request: EnabledIndicatorsRequest) => api.setEnabled(request),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({
        queryKey: [...IL_KEY, 'enabled', vars.deviceType, vars.operatorCode],
      });
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useIndicatorUnits() {
  return useQuery({
    queryKey: [...IL_KEY, 'units'],
    queryFn: () => api.listUnits(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useUpsertUnit() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UnitInput) => api.upsertUnit(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'units'] });
    },
  });
}

export function useUpdateUnit() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UnitInput }) => api.updateUnit(id, input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'units'] });
    },
  });
}

export function useDeleteUnit() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteUnit(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'units'] });
    },
  });
}

export function useIndicatorCacheRefresh() {
  return useMutation({
    mutationFn: () => api.cacheRefresh(),
  });
}

export function useIndicatorImportDirectory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.importDirectory(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: IL_KEY });
    },
  });
}
