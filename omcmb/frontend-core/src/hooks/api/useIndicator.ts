import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  IndicatorGroupTreeParams,
  IndicatorListParams,
  IndicatorCreateParams,
  EnableIndicatorsParams,
  CustNameUpdateParams,
} from '@core/types/indicator';
import { indicatorService } from '@core/mock/services/indicatorService';
import { indicatorApi } from '@core/services/api/indicatorApi';
import { useMock } from '@core/services/apiSwitch';

// ── Query key factory ────────────────────────────────────────────────────────

const indicatorKeys = {
  groupTree: (params: IndicatorGroupTreeParams) => ['indicator', 'groupTree', params] as const,
  list: (params: IndicatorListParams) => ['indicator', 'list', params] as const,
  detail: (id: string) => ['indicator', 'detail', id] as const,
  units: () => ['indicator', 'units'] as const,
  groupList: (deviceType: string) => ['indicator', 'groupList', deviceType] as const,
  types: (deviceType: string) => ['indicator', 'types', deviceType] as const,
  inTemplate: (id: string) => ['indicator', 'inTemplate', id] as const,
};

// ── Group Tree ───────────────────────────────────────────────────────────────

export function useIndicatorGroupTree(params: IndicatorGroupTreeParams) {
  return useQuery({
    queryKey: indicatorKeys.groupTree(params),
    queryFn: () =>
      useMock
        ? indicatorService.getIndicatorGroupTree(params)
        : indicatorApi.getIndicatorGroupTree(params),
    staleTime: 5 * 60 * 1000,
  });
}

// ── Indicator List (paginated) ───────────────────────────────────────────────

export function useIndicatorList(params: IndicatorListParams) {
  return useQuery({
    queryKey: indicatorKeys.list(params),
    queryFn: () =>
      useMock
        ? indicatorService.getIndicatorListByPage(params)
        : indicatorApi.getIndicatorListByPage(params),
    staleTime: 2 * 60 * 1000,
  });
}

// ── Indicator Detail ─────────────────────────────────────────────────────────

export function useIndicatorInfo(indicatorId: string, deviceType?: string) {
  return useQuery({
    queryKey: indicatorKeys.detail(indicatorId),
    queryFn: () =>
      useMock
        ? indicatorService.getIndicatorInfo(indicatorId, deviceType)
        : indicatorApi.getIndicatorInfo(indicatorId, deviceType),
    enabled: Boolean(indicatorId),
  });
}

// ── Create Indicator ─────────────────────────────────────────────────────────

export function useCreateIndicator() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: IndicatorCreateParams & { deviceType?: string }) =>
      useMock
        ? indicatorService.addOrModifyIndicator(params)
        : indicatorApi.addOrModifyIndicator(params),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
    },
  });
}

// ── Update Indicator ─────────────────────────────────────────────────────────

export function useUpdateIndicator() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: IndicatorCreateParams & { deviceType?: string }) =>
      useMock
        ? indicatorService.addOrModifyIndicator(params)
        : indicatorApi.addOrModifyIndicator(params),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
      if (variables.kpiId) {
        void queryClient.invalidateQueries({ queryKey: indicatorKeys.detail(variables.kpiId) });
      }
    },
  });
}

// ── Delete Indicator ─────────────────────────────────────────────────────────

export function useDeleteIndicator() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, deviceType }: { id: string; deviceType?: string }) =>
      useMock
        ? indicatorService.deleteIndicator(id, deviceType)
        : indicatorApi.deleteIndicator(id, deviceType),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
    },
  });
}

// ── Enable Indicators ────────────────────────────────────────────────────────

export function useEnableIndicators() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: EnableIndicatorsParams) =>
      useMock
        ? indicatorService.enableIndicator(params)
        : indicatorApi.enableIndicator(params),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
    },
  });
}

// ── Disable Indicators ───────────────────────────────────────────────────────

export function useDisableIndicators() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: EnableIndicatorsParams) =>
      useMock
        ? indicatorService.disableIndicator(params)
        : indicatorApi.disableIndicator(params),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
    },
  });
}

// ── Is Indicator in Template ─────────────────────────────────────────────────

export function useIsIndicatorInTemplate(indicatorId: string) {
  return useQuery({
    queryKey: indicatorKeys.inTemplate(indicatorId),
    queryFn: () =>
      useMock
        ? indicatorService.isIndicatorInTemplate(indicatorId)
        : indicatorApi.isIndicatorInTemplate(indicatorId),
    enabled: Boolean(indicatorId),
  });
}

// ── Export Indicators ────────────────────────────────────────────────────────

export function useExportIndicators() {
  return useMutation({
    mutationFn: ({
      deviceType,
      groupId,
      operatorCode,
    }: {
      deviceType: string;
      groupId?: string;
      operatorCode?: string;
    }) =>
      useMock
        ? indicatorService.exportAllIndicator(deviceType, groupId, operatorCode)
        : indicatorApi.exportAllIndicator(deviceType, groupId, operatorCode),
  });
}

// ── Update Custom Name ───────────────────────────────────────────────────────

export function useUpdateCustName() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: CustNameUpdateParams) =>
      useMock
        ? indicatorService.updateBaseKpiCustName(params)
        : indicatorApi.updateBaseKpiCustName(params),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
    },
  });
}

// ── Update Counter Name (ENB/GSM/GNB) ───────────────────────────────────────

export function useUpdateCounterName() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      indicatorId,
      name,
    }: {
      deviceType: 'ENB' | 'GSM' | 'GNB';
      indicatorId: string;
      name: string;
    }) => {
      if (useMock) {
        if (deviceType === 'GNB') return indicatorService.updateGnbIndicatorsName(indicatorId, name);
        if (deviceType === 'GSM') return indicatorService.updateGsmIndicatorsName(indicatorId, name);
        return indicatorService.updateEnbIndicatorsName(indicatorId, name);
      }
      if (deviceType === 'GNB') return indicatorApi.updateGnbIndicatorsName(indicatorId, name);
      if (deviceType === 'GSM') return indicatorApi.updateGsmIndicatorsName(indicatorId, name);
      return indicatorApi.updateEnbIndicatorsName(indicatorId, name);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['indicator', 'list'] });
    },
  });
}

// ── Indicator Units ──────────────────────────────────────────────────────────

export function useIndicatorUnits() {
  return useQuery({
    queryKey: indicatorKeys.units(),
    queryFn: () =>
      useMock
        ? indicatorService.getIndicatorUnitList()
        : indicatorApi.getIndicatorUnitList(),
    staleTime: 30 * 60 * 1000, // Units rarely change
  });
}

// ── Indicator Group List (for dropdown) ──────────────────────────────────────

export function useIndicatorGroupList(deviceType: string, operatorCode?: string) {
  return useQuery({
    queryKey: indicatorKeys.groupList(deviceType),
    queryFn: () =>
      useMock
        ? indicatorService.getIndicatorGroupList(deviceType, operatorCode)
        : indicatorApi.getIndicatorGroupList(deviceType, operatorCode),
    enabled: Boolean(deviceType),
    staleTime: 5 * 60 * 1000,
  });
}

// ── Indicator Types ──────────────────────────────────────────────────────────

export function useIndicatorTypes(deviceType: string) {
  return useQuery({
    queryKey: indicatorKeys.types(deviceType),
    queryFn: () =>
      useMock
        ? indicatorService.getIndicatorTypes(deviceType)
        : indicatorApi.getIndicatorTypes(deviceType),
    enabled: Boolean(deviceType),
    staleTime: 30 * 60 * 1000,
  });
}
