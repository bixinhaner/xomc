import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import {
  deviceAccessApi,
  type AccessListType,
  type AccessState,
  type ActionStatus,
} from '../../services/api/deviceAccessApi';

const rootKey = ['device-access'] as const;

// Cache refresh is follow-up work, not part of the command acknowledgement.
// Returning invalidateQueries() from a mutation onSuccess keeps mutateAsync and
// every confirmLoading Modal pending until all active device-access queries
// settle. A slow detail/candidate request must not leave a successful command
// stuck behind a loading overlay.
export function invalidateDeviceAccessQueries(qc: Pick<QueryClient, 'invalidateQueries'>): void {
  void qc.invalidateQueries({ queryKey: rootKey });
}

export function useAccessStates(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; state?: AccessState }) {
  return useQuery({ queryKey: [...rootKey, 'states', params], queryFn: () => deviceAccessApi.listStates(params), enabled: Boolean(params.operatorCode) });
}

export function useDeviceAccessRuntimeSettings(operatorCode: string) {
  return useQuery({
    queryKey: [...rootKey, 'settings', operatorCode],
    queryFn: () => deviceAccessApi.getRuntimeSettings(operatorCode),
    enabled: Boolean(operatorCode),
  });
}

export function useUpdateDeviceAccessRuntimeSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, enabled }: { operatorCode: string; enabled: boolean }) =>
      deviceAccessApi.updateRuntimeSettings(operatorCode, enabled),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}

export function useAccessDetail(operatorCode: string, serialNumber?: string) {
  return useQuery({
    queryKey: [...rootKey, 'detail', operatorCode, serialNumber],
    queryFn: () => deviceAccessApi.getDetail(operatorCode, serialNumber!),
    enabled: Boolean(operatorCode && serialNumber),
  });
}

export function useReevaluateAccessDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, serialNumber }: { operatorCode: string; serialNumber: string }) =>
      deviceAccessApi.reevaluateDevice(operatorCode, serialNumber),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}

export function useAccessPolicies(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string }) {
  return useQuery({ queryKey: [...rootKey, 'policies', params], queryFn: () => deviceAccessApi.listPolicies(params), enabled: Boolean(params.operatorCode) });
}

export function useAccessPolicy(operatorCode: string, versionId?: string) {
  return useQuery({
    queryKey: [...rootKey, 'policy', operatorCode, versionId],
    queryFn: () => deviceAccessApi.getPolicy(operatorCode, versionId!),
    enabled: Boolean(operatorCode && versionId),
  });
}

export function useCreateAccessPolicyDraft() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.createPolicyDraft, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function usePublishAccessPolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, versionId }: { operatorCode: string; versionId: string }) => deviceAccessApi.publishPolicy(operatorCode, versionId),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}

export function useDeleteAccessPolicyDraft() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, versionId }: { operatorCode: string; versionId: string }) =>
      deviceAccessApi.deletePolicyDraft(operatorCode, versionId),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}

export function useAccessEntries(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; entryType?: AccessListType }) {
  return useQuery({ queryKey: [...rootKey, 'entries', params], queryFn: () => deviceAccessApi.listEntries(params), enabled: Boolean(params.operatorCode) });
}

export function useUpsertAccessEntry() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.upsertEntry, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useAccessCandidates(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; reviewStatus?: string }) {
  return useQuery({ queryKey: [...rootKey, 'candidates', params], queryFn: () => deviceAccessApi.listCandidates(params), enabled: Boolean(params.operatorCode) });
}

export function useReviewAccessCandidate() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.reviewCandidate, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useAccessActions(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; status?: ActionStatus }) {
  return useQuery({ queryKey: [...rootKey, 'actions', params], queryFn: () => deviceAccessApi.listActions(params), enabled: Boolean(params.operatorCode) });
}

export function useRetryAccessAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, actionId }: { operatorCode: string; actionId: string }) => deviceAccessApi.retryAction(operatorCode, actionId),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}
