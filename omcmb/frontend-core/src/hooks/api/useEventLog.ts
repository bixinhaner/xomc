import { useQuery } from '@tanstack/react-query';
import { eventLogApi, type EventLogListParams } from '../../services/api/eventLogApi';

export const eventLogKeys = {
  all: ['event-logs'] as const,
  lists: () => [...eventLogKeys.all, 'list'] as const,
  list: (params: EventLogListParams) => [...eventLogKeys.lists(), params] as const,
  detail: (id: string) => [...eventLogKeys.all, 'detail', id] as const,
};

export function useEventLogList(params: EventLogListParams, enabled = true) {
  return useQuery({
    queryKey: eventLogKeys.list(params),
    queryFn: () => eventLogApi.list(params),
    enabled,
    staleTime: 15_000,
  });
}

export function useEventLog(id: string | undefined) {
  return useQuery({
    queryKey: id ? eventLogKeys.detail(id) : eventLogKeys.all,
    queryFn: () => eventLogApi.getById(id as string),
    enabled: !!id,
  });
}
