import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { AlarmFilter } from '../../types/alarm';
import type { PageRequest } from '../../types/pagination';
import { alarmService } from '../../mock/services/alarmService';
import { alarmApi } from '../../services/api/alarmApi';
import { useMock } from '../../services/apiSwitch';

// Real API is the canonical surface; mock is best-effort and adapted at runtime.
const api: typeof alarmApi = useMock
  ? (alarmService as unknown as typeof alarmApi)
  : alarmApi;

function scheduleAlarmRefresh(queryClient: ReturnType<typeof useQueryClient>) {
  const retryDelays = [3_000, 8_000, 15_000, 30_000];

  retryDelays.forEach((delayMs) => {
    window.setTimeout(() => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    }, delayMs);
  });
}

interface AlarmQueryOptions {
  refetchIntervalMs?: number | false;
  refetchIntervalInBackground?: boolean;
  refetchOnWindowFocus?: boolean;
}

export function useCurrentAlarms(
  params: AlarmFilter & PageRequest,
  options?: AlarmQueryOptions
) {
  return useQuery({
    queryKey: ['alarms', 'current', params],
    queryFn: () => api.getCurrentAlarms(params),
    refetchInterval: options?.refetchIntervalMs ?? 30000,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? true,
    refetchOnWindowFocus: options?.refetchOnWindowFocus ?? true,
  });
}

export function useHistoricalAlarms(params: AlarmFilter & PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'historical', params],
    queryFn: () => api.getHistoricalAlarms(params),
    refetchOnWindowFocus: true,
  });
}

export function useAlarmList(params: AlarmFilter & PageRequest & { isActive?: boolean }) {
  return useQuery({
    queryKey: ['alarms', 'list', params],
    queryFn: () => api.getList(params),
    refetchInterval: params.isActive !== false ? 30000 : undefined,
    refetchIntervalInBackground: params.isActive !== false,
    refetchOnWindowFocus: true,
  });
}

export function useAlarmById(id: string) {
  return useQuery({
    queryKey: ['alarms', 'detail', id],
    queryFn: () => api.getById(id),
    enabled: Boolean(id),
  });
}

export function useAlarmCount() {
  return useQuery({
    queryKey: ['alarms', 'count'],
    queryFn: () => api.getAlarmCount(),
    refetchInterval: 15000,
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: true,
  });
}

export function useHistoryAlarmCount() {
  return useQuery({
    queryKey: ['alarms', 'history-count'],
    queryFn: () => api.getHistoryAlarmCount(),
  });
}

export function useAlarmRules(params: PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'rules', params],
    queryFn: () => api.getRules(params),
  });
}

export function useAcknowledgeAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, note }: { ids: string[]; note?: string }) =>
      api.acknowledgeAlarms(ids, note),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useClearAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, note }: { ids: string[]; note?: string }) => api.clearAlarms(ids, note),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useUnacknowledgeAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.unacknowledgeAlarms(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useAcknowledgeHistoryAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, note }: { ids: string[]; note?: string }) =>
      api.acknowledgeHistoryAlarms(ids, note),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useUnacknowledgeHistoryAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.unacknowledgeHistoryAlarms(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useDeleteHistoryAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteHistoryAlarms(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useCreateAlarmRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof api.createRule>[0]) =>
      api.createRule(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}

export function useUpdateAlarmRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof api.updateRule>[1] }) =>
      api.updateRule(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}

export function useDeleteAlarmRules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteRules(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}

export function useMarkAlarmRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.markAlarmRead(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useToggleAlarmRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.toggleRule(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}

// T-0098-P5-06：旧 useAlarmLibraries / useCreate|Update|DeleteAlarmLibrary 已下线，
// 治理 hook 走 useAlarmDefinitions（@core/hooks/api/useAlarmDefinitions）。

export function useTriggerAlarmSync() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (deviceSN: string) => api.triggerAlarmSync(deviceSN),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
      scheduleAlarmRefresh(queryClient);
    },
  });
}
