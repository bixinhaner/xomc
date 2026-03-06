import { useQuery, useMutation } from '@tanstack/react-query';
import type { OperationType, OperationResult } from '@/types/system';
import type { PageRequest } from '@/types/pagination';
import type { SystemLog } from '@/mock/data/logs';
import { logService } from '@/mock/services/logService';

export function useOperationLogs(
  params: {
    operator?: string;
    module?: string;
    operationType?: OperationType;
    result?: OperationResult;
    timeRange?: [string, string];
    keyword?: string;
  } & PageRequest
) {
  return useQuery({
    queryKey: ['logs', 'operation', params],
    queryFn: () => logService.getOperationLogs(params),
  });
}

export function useSystemLogs(
  params: { level?: SystemLog['level']; source?: string; keyword?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['logs', 'system', params],
    queryFn: () => logService.getSystemLogs(params),
    refetchInterval: 30000,
  });
}

export function useNEMessageLogs(
  params: { deviceSn?: string; messageType?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['logs', 'ne-messages', params],
    queryFn: () => logService.getNEMessageLogs(params),
    refetchInterval: 10000,
  });
}

export function useExportLogs() {
  return useMutation({
    mutationFn: ({ type, params }: { type: 'operation' | 'system' | 'ne'; params: Record<string, unknown> }) =>
      logService.exportLogs(type, params),
  });
}
