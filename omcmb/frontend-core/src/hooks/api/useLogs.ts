import { useQuery, useMutation } from '@tanstack/react-query';
import type { OperationType, OperationResult } from '../../types/system';
import type { PageRequest } from '../../types/pagination';
import type { SystemLog } from '../../mock/data/logs';
import { logService } from '../../mock/services/logService';
import { adminApi } from '../../services/api/adminApi';
import { logApi } from '../../services/api/logApi';
import { useMock } from '../../services/apiSwitch';

export function useOperationLogs(
  params: {
    operator?: string;
    clientIp?: string;
    module?: string;
    action?: string;
    operationType?: OperationType;
    result?: OperationResult;
    reason?: string;
    timeRange?: [string, string];
    keyword?: string;
  } & PageRequest,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ['logs', 'operation', params],
    queryFn: () =>
      useMock
        ? logService.getOperationLogs(params)
        : adminApi.getOperationLogs(params),
    enabled: options?.enabled ?? true,
  });
}

export function useNorthboundAPIInvocationLogs(
  params: {
    apiKey?: string;
    name?: string;
    method?: string;
    path?: string;
    status?: string;
    createUser?: string;
    ipAddress?: string;
    timeRange?: [string, string];
  } & PageRequest,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ['logs', 'northbound-api-invocation', params],
    queryFn: () =>
      useMock
        ? logService.getNorthboundAPIInvocationLogs(params)
        : adminApi.getNorthboundAPIInvocationLogs(params),
    enabled: options?.enabled ?? true,
  });
}

export function useSystemLogs(
  params: { level?: SystemLog['level']; source?: string; keyword?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['logs', 'system', params],
    queryFn: () =>
      useMock
        ? logService.getSystemLogs(params)
        : logApi.getSystemLogs(params),
    refetchInterval: 30000,
  });
}

export function useNEMessageLogs(
  params: { deviceSn?: string; messageType?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['logs', 'ne-messages', params],
    queryFn: () =>
      useMock
        ? logService.getNEMessageLogs(params)
        : logApi.getNEMessageLogs(params),
    refetchInterval: 10000,
  });
}

export function useExportLogs() {
  return useMutation({
    mutationFn: ({ type, params }: { type: 'operation' | 'system' | 'ne'; params: Record<string, unknown> }) =>
      logService.exportLogs(type, params),
  });
}
