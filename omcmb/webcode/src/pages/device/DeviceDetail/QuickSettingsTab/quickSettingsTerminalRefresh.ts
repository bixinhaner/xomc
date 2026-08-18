import type { QueryClient } from '@tanstack/react-query';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';

export async function refreshQuickSettingsRelatedLists(
  queryClient: QueryClient,
  deviceId: string,
): Promise<void> {
  deviceParameterApi.invalidateParameterSchemaCache(deviceId);
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: ['devices', 'parameter-schema', deviceId],
      refetchType: 'active',
    }),
    queryClient.invalidateQueries({
      queryKey: ['devices', 'parameters', deviceId],
      refetchType: 'active',
    }),
    queryClient.invalidateQueries({
      queryKey: ['devices', 'parameters', 'search', deviceId],
      refetchType: 'active',
    }),
    queryClient.invalidateQueries({
      queryKey: ['devices', 'parameter-tree', deviceId],
      refetchType: 'active',
    }),
  ]);
}
