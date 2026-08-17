import { useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { AlarmFilter } from '../../types/alarm';
import type { PageRequest } from '../../types/pagination';
import { alarmService } from '../../mock/services/alarmService';
import { alarmApi } from '../../services/api/alarmApi';
import { createApiSwitchWithMock } from '../../services/apiSwitch';
import { useAppStore } from '../../store/appStore';

// Real API is the canonical surface; mock is best-effort and adapted at runtime.
const api = createApiSwitchWithMock(alarmService, alarmApi);

interface AlarmQueryOptions {
  refetchIntervalMs?: number | false;
  refetchIntervalInBackground?: boolean;
  refetchOnWindowFocus?: boolean;
}

export function useCurrentAlarms(
  params: AlarmFilter & PageRequest,
  options?: AlarmQueryOptions
) {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: ['alarms', 'current', params, locale],
    queryFn: () => api.getCurrentAlarms(params),
    // 实时页（性能 #15）：比全局 30s staleTime 更短，进入页面/切回前台更快补刷
    staleTime: 5_000,
    refetchInterval: options?.refetchIntervalMs ?? 30000,
    // 性能 #15：默认后台标签暂停轮询，切回前台经 refetchOnWindowFocus 立即补刷，
    // 不丢实时性。调用方可显式传 true 覆盖（如确需后台常驻轮询的场景）。
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
    refetchOnWindowFocus: options?.refetchOnWindowFocus ?? true,
  });
}

export function useHistoricalAlarms(
  params: AlarmFilter & PageRequest,
  options?: AlarmQueryOptions
) {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: ['alarms', 'historical', params, locale],
    queryFn: () => api.getHistoricalAlarms(params),
    refetchInterval: options?.refetchIntervalMs ?? false,
    refetchIntervalInBackground: options?.refetchIntervalInBackground ?? false,
    refetchOnWindowFocus: options?.refetchOnWindowFocus ?? true,
  });
}

export function useAlarmList(params: AlarmFilter & PageRequest & { isActive?: boolean }) {
  return useQuery({
    queryKey: ['alarms', 'list', params],
    queryFn: () => api.getList(params),
    staleTime: 5_000,
    refetchInterval: params.isActive !== false ? 30000 : undefined,
    // 性能 #15：后台标签暂停轮询（不再随 isActive 在后台常驻）
    refetchIntervalInBackground: false,
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

// 告警计数从常驻 Header 触发（每 15s 轮询徽标）。性能（#15）：标签页隐藏时
// 暂停轮询（refetchIntervalInBackground: false），切回前台 + refetchOnWindowFocus
// 会立即补刷一次，徽标即时回到最新值，不丢实时性。
export function useAlarmCount() {
  return useQuery({
    queryKey: ['alarms', 'count'],
    queryFn: () => api.getAlarmCount(),
    refetchInterval: 15000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });
}

export function useAlarmCountWithDeviceListInvalidation() {
  const queryClient = useQueryClient();
  const alarmCountQuery = useAlarmCount();
  const stateVersion = alarmCountQuery.data?.stateVersion ?? alarmCountQuery.data?.total_active ?? 0;

  useEffect(() => {
    void queryClient.invalidateQueries({ queryKey: ['devices'] });
  }, [queryClient, stateVersion]);

  return alarmCountQuery;
}

export function useHistoryAlarmCount() {
  return useQuery({
    queryKey: ['alarms', 'history-count'],
    queryFn: () => api.getHistoryAlarmCount(),
  });
}

export function useAlarmRules(
  params: PageRequest & { keyword?: string; filterType?: string[]; action?: string; enabled?: string }
) {
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

export function useAlarmEmailSetting() {
  return useQuery({
    queryKey: ['alarms', 'email-setting'],
    queryFn: () => api.getAlarmEmailSetting(),
  });
}

export function useUpdateAlarmEmailSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.updateAlarmEmailSetting,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['alarms', 'email-setting'] }),
  });
}

export function useAlarmEmailSubscriptions() {
  return useQuery({
    queryKey: ['alarms', 'email-subscriptions'],
    queryFn: () => api.getAlarmEmailSubscriptions(),
  });
}

export function useCreateAlarmEmailSubscription() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.createAlarmEmailSubscription,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['alarms', 'email-subscriptions'] }),
  });
}

export function useUpdateAlarmEmailSubscription() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Parameters<typeof api.updateAlarmEmailSubscription>[1] }) =>
      api.updateAlarmEmailSubscription(id, input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['alarms', 'email-subscriptions'] }),
  });
}

export function useDeleteAlarmEmailSubscription() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.deleteAlarmEmailSubscription,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['alarms', 'email-subscriptions'] }),
  });
}

// T-0098-P5-06：旧 useAlarmLibraries / useCreate|Update|DeleteAlarmLibrary 已下线，
// 治理 hook 走 useAlarmDefinitions（@core/hooks/api/useAlarmDefinitions）。

export function useTriggerAlarmSync() {
  return useMutation({
    mutationFn: (deviceSN: string) => api.triggerAlarmSync(deviceSN),
  });
}
