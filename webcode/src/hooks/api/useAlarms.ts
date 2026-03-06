import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { AlarmFilter } from '@/types/alarm';
import type { PageRequest } from '@/types/pagination';
import { alarmService } from '@/mock/services/alarmService';

export function useCurrentAlarms(params: AlarmFilter & PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'current', params],
    queryFn: () => alarmService.getCurrentAlarms(params),
    refetchInterval: 30000,
  });
}

export function useHistoricalAlarms(params: AlarmFilter & PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'historical', params],
    queryFn: () => alarmService.getHistoricalAlarms(params),
  });
}

export function useAlarmList(params: AlarmFilter & PageRequest & { isActive?: boolean }) {
  return useQuery({
    queryKey: ['alarms', 'list', params],
    queryFn: () => alarmService.getList(params),
    refetchInterval: params.isActive !== false ? 30000 : undefined,
  });
}

export function useAlarmById(id: string) {
  return useQuery({
    queryKey: ['alarms', 'detail', id],
    queryFn: () => alarmService.getById(id),
    enabled: Boolean(id),
  });
}

export function useAlarmCount() {
  return useQuery({
    queryKey: ['alarms', 'count'],
    queryFn: () => alarmService.getAlarmCount(),
    refetchInterval: 15000,
  });
}

export function useAlarmRules(params: PageRequest) {
  return useQuery({
    queryKey: ['alarms', 'rules', params],
    queryFn: () => alarmService.getRules(params),
  });
}

export function useAcknowledgeAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, note }: { ids: string[]; note?: string }) =>
      alarmService.acknowledgeAlarms(ids, note),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useClearAlarms() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => alarmService.clearAlarms(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}

export function useCreateAlarmRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof alarmService.createRule>[0]) =>
      alarmService.createRule(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}

export function useUpdateAlarmRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof alarmService.updateRule>[1] }) =>
      alarmService.updateRule(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}

export function useDeleteAlarmRules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => alarmService.deleteRules(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}
