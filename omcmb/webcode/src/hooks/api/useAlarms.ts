import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { AlarmFilter } from '@/types/alarm';
import type { PageRequest } from '@/types/pagination';
import { alarmService } from '@/mock/services/alarmService';
import { alarmApi } from '@/services/api/alarmApi';
import { createApiSwitch } from '@/services/apiSwitch';

const api = createApiSwitch(alarmService, alarmApi);

export function useCurrentAlarms(params: AlarmFilter & PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'current', params],
    queryFn: () => api.getCurrentAlarms(params),
    refetchInterval: 30000,
  });
}

export function useHistoricalAlarms(params: AlarmFilter & PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'historical', params],
    queryFn: () => api.getHistoricalAlarms(params),
  });
}

export function useAlarmList(params: AlarmFilter & PageRequest & { isActive?: boolean }) {
  return useQuery({
    queryKey: ['alarms', 'list', params],
    queryFn: () => api.getList(params),
    refetchInterval: params.isActive !== false ? 30000 : undefined,
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
    mutationFn: (ids: string[]) => api.clearAlarms(ids),
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
