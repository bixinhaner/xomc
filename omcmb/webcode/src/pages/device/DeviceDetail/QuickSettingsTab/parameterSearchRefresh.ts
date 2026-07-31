import type { QueryClient } from '@tanstack/react-query';

interface ParameterSearchCacheItem {
  parameterPath: string;
  parameterValue?: unknown;
  [key: string]: unknown;
}

interface DeviceParameterSearchReadback {
  deviceId: string;
  searchQuery: string;
  replacePathPrefix: string;
  parameters: ParameterSearchCacheItem[];
}

export function applyDeviceParameterSearchReadback(
  queryClient: QueryClient,
  readback: DeviceParameterSearchReadback,
): void {
  queryClient.setQueriesData<ParameterSearchCacheItem[]>(
    {
      queryKey: [
        'devices',
        'parameters',
        'search',
        readback.deviceId,
        readback.searchQuery,
      ],
    },
    (current) => {
      if (!current) return current;
      return [
        ...current.filter(
          (item) => !item.parameterPath.startsWith(readback.replacePathPrefix),
        ),
        ...readback.parameters,
      ];
    },
  );
}

export async function refreshDeviceParameterSearchQueries(
  queryClient: QueryClient,
  deviceId: string,
): Promise<void> {
  await queryClient.invalidateQueries({
    queryKey: ['devices', 'parameters', 'search', deviceId],
    refetchType: 'active',
  });
}
