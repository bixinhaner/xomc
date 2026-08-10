import { useEffect, useRef } from 'react';
import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { geofenceApi } from '../../services/api/geofenceApi';
import { createApiSwitchWithMock } from '../../services/apiSwitch';
import { geofenceMockService } from '../../mock/services/geofenceMock';
import type {
  CreateGeofenceDefinitionInput,
  CreateGeofenceManualBindJobInput,
  CreateGeofenceVersionInput,
  GeofenceBatchItemFilter,
  GeofenceBindingFilter,
  GeofenceBindingInputs,
  GeofenceDefinitionFilter,
  GeofenceJobStatus,
  GeofenceLifecycleTarget,
  GeofenceMapFilter,
  GeofenceTransitionInput,
  UpdateGeofenceSettingsInput,
} from '../../types/geofence';

const geofenceService = createApiSwitchWithMock(
  geofenceMockService,
  geofenceApi,
);

export const geofenceKeys = {
  all: ['geofence'] as const,
  availability: () => [...geofenceKeys.all, 'availability'] as const,
  settings: () => [...geofenceKeys.all, 'settings'] as const,
  maps: () => [...geofenceKeys.all, 'map'] as const,
  map: (filter: GeofenceMapFilter) =>
    [...geofenceKeys.maps(), filter] as const,
  definitions: () =>
    [...geofenceKeys.all, 'definitions'] as const,
  definitionList: (filter: GeofenceDefinitionFilter) =>
    [...geofenceKeys.definitions(), 'list', filter] as const,
  detail: (id: string) =>
    [...geofenceKeys.definitions(), 'detail', id] as const,
  versions: (id: string) =>
    [...geofenceKeys.detail(id), 'versions'] as const,
  bindingLists: (id: string) =>
    [...geofenceKeys.detail(id), 'bindings'] as const,
  bindings: (id: string, filter: GeofenceBindingFilter) =>
    [...geofenceKeys.bindingLists(id), filter] as const,
  jobs: () => [...geofenceKeys.all, 'jobs'] as const,
  job: (id: string) =>
    [...geofenceKeys.jobs(), 'detail', id] as const,
  jobItemsRoot: (id: string) =>
    [...geofenceKeys.job(id), 'items'] as const,
  jobItems: (id: string, filter: GeofenceBatchItemFilter) =>
    [...geofenceKeys.jobItemsRoot(id), filter] as const,
};

export function geofenceJobRefetchInterval(
  status: GeofenceJobStatus | undefined,
  intervalMs = 2000,
): number | false {
  switch (status) {
    case 'pending':
    case 'running':
    case 'zombie':
      return intervalMs;
    default:
      return false;
  }
}

export function isGeofenceJobTerminal(
  status: GeofenceJobStatus | undefined,
): boolean {
  return (
    status === 'succeeded' ||
    status === 'failed' ||
    status === 'canceled'
  );
}

type GeofenceCacheInvalidator = Pick<
  ReturnType<typeof useQueryClient>,
  'invalidateQueries'
>;

export async function invalidateGeofenceSettingsCaches(
  queryClient: GeofenceCacheInvalidator,
) {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.settings(),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.availability(),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.definitions(),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.maps(),
    }),
  ]);
}

export async function invalidateGeofenceDefinitionCaches(
  queryClient: GeofenceCacheInvalidator,
  id: string,
) {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.definitions(),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.detail(id),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.versions(id),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.maps(),
    }),
  ]);
}

export async function invalidateGeofenceBindingCaches(
  queryClient: GeofenceCacheInvalidator,
  id: string,
) {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.bindingLists(id),
    }),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.maps(),
    }),
  ]);
}

export async function invalidateGeofenceCompletedJobCaches(
  queryClient: GeofenceCacheInvalidator,
  jobId: string,
  geofenceId: string,
) {
  await Promise.all([
    invalidateGeofenceBindingCaches(queryClient, geofenceId),
    queryClient.invalidateQueries({
      queryKey: geofenceKeys.jobItemsRoot(jobId),
    }),
  ]);
}

export function useGeofenceAvailability() {
  return useQuery({
    queryKey: geofenceKeys.availability(),
    queryFn: () => geofenceService.getAvailability(),
    staleTime: 30_000,
  });
}

export function useGeofenceSettings(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: geofenceKeys.settings(),
    queryFn: () => geofenceService.getSettings(),
    enabled: options?.enabled ?? true,
  });
}

export function usePreviewGeofenceSettings() {
  return useMutation({
    mutationFn: (settings: UpdateGeofenceSettingsInput) =>
      geofenceService.previewSettings(settings),
  });
}

export function useUpdateGeofenceSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (settings: UpdateGeofenceSettingsInput) =>
      geofenceService.updateSettings(settings),
    onSuccess: () =>
      invalidateGeofenceSettingsCaches(queryClient),
  });
}

export function useGeofenceMap(
  filter: GeofenceMapFilter,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: geofenceKeys.map(filter),
    queryFn: () => geofenceService.listMapDefinitions(filter),
    enabled: options?.enabled ?? true,
  });
}

export function useGeofenceDefinitions(
  filter: GeofenceDefinitionFilter = {},
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: geofenceKeys.definitionList(filter),
    queryFn: () => geofenceService.listDefinitions(filter),
    enabled: options?.enabled ?? true,
  });
}

export function useGeofenceDefinition(
  id: string | undefined,
) {
  return useQuery({
    queryKey: geofenceKeys.detail(id ?? ''),
    queryFn: () => geofenceService.getDefinition(id as string),
    enabled: Boolean(id),
  });
}

export function useGeofenceVersions(id: string | undefined) {
  return useQuery({
    queryKey: geofenceKeys.versions(id ?? ''),
    queryFn: () => geofenceService.listVersions(id as string),
    enabled: Boolean(id),
  });
}

export function useCreateGeofenceDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateGeofenceDefinitionInput) =>
      geofenceService.createDefinition(input),
    onSuccess: (result) =>
      invalidateGeofenceDefinitionCaches(
        queryClient,
        result.definition.id,
      ),
  });
}

export function useCreateGeofenceDraftVersion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string;
      input: CreateGeofenceVersionInput;
    }) => geofenceService.createDraftVersion(id, input),
    onSuccess: (_result, { id }) =>
      invalidateGeofenceDefinitionCaches(queryClient, id),
  });
}

export function useRenameGeofenceDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      geofenceService.renameDefinition(id, name),
    onSuccess: (_, input) => {
      void queryClient.invalidateQueries({ queryKey: geofenceKeys.maps() });
      void queryClient.invalidateQueries({ queryKey: geofenceKeys.detail(input.id) });
    },
  });
}

export function usePublishGeofenceDraft() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      versionId,
    }: {
      id: string;
      versionId: string;
    }) => geofenceService.publishDraft(id, versionId),
    onSuccess: (_result, { id }) =>
      invalidateGeofenceDefinitionCaches(queryClient, id),
  });
}

export function usePreviewGeofenceLifecycle() {
  return useMutation({
    mutationFn: ({
      id,
      target,
    }: {
      id: string;
      target: GeofenceLifecycleTarget;
    }) =>
      geofenceService.previewLifecycleTransition(id, target),
  });
}

export function useTransitionGeofenceLifecycle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      target,
      input,
    }: {
      id: string;
      target: GeofenceLifecycleTarget;
      input: GeofenceTransitionInput;
    }) =>
      geofenceService.transitionLifecycle(id, target, input),
    onSuccess: (_result, { id }) =>
      invalidateGeofenceDefinitionCaches(queryClient, id),
  });
}

export function useGeofenceBindings(
  id: string | undefined,
  filter: GeofenceBindingFilter,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: geofenceKeys.bindings(id ?? '', filter),
    queryFn: () =>
      geofenceService.listBindings(id as string, filter),
    enabled: Boolean(id) && (options?.enabled ?? true),
  });
}

export function useExportGeofenceBindings() {
  return useMutation({
    mutationFn: ({
      id,
      filter,
    }: {
      id: string;
      filter: Pick<
        GeofenceBindingFilter,
        'status' | 'keyword'
      >;
    }) => geofenceService.exportBindings(id, filter),
  });
}

export function usePreviewGeofenceManualBindings() {
  return useMutation({
    mutationFn: ({
      id,
      inputs,
    }: {
      id: string;
      inputs: GeofenceBindingInputs;
    }) => geofenceService.previewManualBindings(id, inputs),
  });
}

export function usePreviewGeofenceCandidates() {
  return useMutation({
    mutationFn: (id: string) => geofenceService.previewCandidates(id),
  });
}

export function useGeofenceControlActions(
  id: string | undefined,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: ['geofence-control-actions', id],
    queryFn: () => geofenceService.listControlActions(id as string),
    enabled: Boolean(id) && (options?.enabled ?? true),
  });
}

export function useCreateGeofenceManualBindJob() {
  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string;
      input: CreateGeofenceManualBindJobInput;
    }) => geofenceService.createManualBindJob(id, input),
  });
}

export function useGeofenceManualBindJob(
  id: string | undefined,
  geofenceId: string | undefined,
  options?: { intervalMs?: number },
) {
  const queryClient = useQueryClient();
  const invalidatedTerminalRef = useRef<string | undefined>(
    undefined,
  );
  const query = useQuery({
    queryKey: geofenceKeys.job(id ?? ''),
    queryFn: () =>
      geofenceService.getManualBindJob(id as string),
    enabled: Boolean(id),
    retry: false,
    refetchInterval: (currentQuery) =>
      geofenceJobRefetchInterval(
        currentQuery.state.data?.status,
        options?.intervalMs,
      ),
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });

  useEffect(() => {
    const status = query.data?.status;
    if (!id || !geofenceId || !isGeofenceJobTerminal(status)) {
      return;
    }
    const terminalKey = `${id}:${status}`;
    if (invalidatedTerminalRef.current === terminalKey) {
      return;
    }
    invalidatedTerminalRef.current = terminalKey;
    void invalidateGeofenceCompletedJobCaches(
      queryClient,
      id,
      geofenceId,
    );
  }, [geofenceId, id, query.data?.status, queryClient]);

  return query;
}

export function useGeofenceManualBindItems(
  id: string | undefined,
  filter: GeofenceBatchItemFilter,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: geofenceKeys.jobItems(id ?? '', filter),
    queryFn: () =>
      geofenceService.listManualBindItems(
        id as string,
        filter,
      ),
    enabled: Boolean(id) && (options?.enabled ?? true),
  });
}

function useBindingTransitionInvalidation() {
  const queryClient = useQueryClient();
  return (geofenceId: string) =>
    invalidateGeofenceBindingCaches(queryClient, geofenceId);
}

export function useSuspendGeofenceBinding() {
  const invalidate = useBindingTransitionInvalidation();
  return useMutation({
    mutationFn: ({
      id,
      reason,
    }: {
      id: string;
      reason: string;
    }) => geofenceService.suspendBinding(id, reason),
    onSuccess: (binding) => invalidate(binding.geofenceId),
  });
}

export function useResumeGeofenceBinding() {
  const invalidate = useBindingTransitionInvalidation();
  return useMutation({
    mutationFn: ({
      id,
      reason,
    }: {
      id: string;
      reason: string;
    }) => geofenceService.resumeBinding(id, reason),
    onSuccess: (binding) => invalidate(binding.geofenceId),
  });
}

export function useRemoveGeofenceBinding() {
  const invalidate = useBindingTransitionInvalidation();
  return useMutation({
    mutationFn: ({
      id,
      reason,
    }: {
      id: string;
      reason: string;
    }) => geofenceService.removeBinding(id, reason),
    onSuccess: (binding) => invalidate(binding.geofenceId),
  });
}
