import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import {
  deviceAccessApi,
  type AccessListType,
  type AccessAuditFilters,
  type AccessState,
  type ActionStatus,
  type ImportFailurePolicy,
  type ImportMode,
  type RuleDimension,
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

export function useAccessStates(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; state?: AccessState } & AccessAuditFilters) {
  return useQuery({ queryKey: [...rootKey, 'states', params], queryFn: () => deviceAccessApi.listStates(params), enabled: Boolean(params.operatorCode) });
}

export function useAccessStateSummary(params: { operatorCode: string; serialNumber?: string; state?: AccessState } & AccessAuditFilters) {
  return useQuery({ queryKey: [...rootKey, 'states-summary', params], queryFn: () => deviceAccessApi.summarizeStates(params), enabled: Boolean(params.operatorCode) });
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

export function useAccessDecisions(params: { operatorCode: string; serialNumber?: string; page: number; pageSize: number } & AccessAuditFilters) {
  return useQuery({
    queryKey: [...rootKey, 'decisions', params],
    queryFn: () => deviceAccessApi.listDecisions(params),
    enabled: Boolean(params.operatorCode && params.serialNumber),
  });
}

export function useAccessIdentitySnapshots(params: { operatorCode: string; serialNumber?: string; page: number; pageSize: number }) {
  return useQuery({ queryKey: [...rootKey, 'identity-snapshots', params], queryFn: () => deviceAccessApi.listIdentitySnapshots(params), enabled: Boolean(params.operatorCode && params.serialNumber) });
}

export function useAccessEvidence(params: { operatorCode: string; serialNumber?: string; page: number; pageSize: number }) {
  return useQuery({ queryKey: [...rootKey, 'evidence', params], queryFn: () => deviceAccessApi.listEvidence(params), enabled: Boolean(params.operatorCode && params.serialNumber) });
}

export function useAccessNotifications(params: { operatorCode: string; serialNumber?: string; page: number; pageSize: number }) {
  return useQuery({ queryKey: [...rootKey, 'notifications', params], queryFn: () => deviceAccessApi.listNotifications(params), enabled: Boolean(params.operatorCode && params.serialNumber) });
}

export function useAccessManualOperations(params: { operatorCode: string; serialNumber?: string; page: number; pageSize: number }) {
  return useQuery({ queryKey: [...rootKey, 'manual-operations', params], queryFn: () => deviceAccessApi.listManualOperations(params), enabled: Boolean(params.operatorCode && params.serialNumber) });
}

export function useArchiveAccessDecision() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, decisionId, reason }: { operatorCode: string; decisionId: string; reason: string }) =>
      deviceAccessApi.archiveDecision(operatorCode, decisionId, reason),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}

export function useRestoreAccessDecision() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, decisionId }: { operatorCode: string; decisionId: string }) =>
      deviceAccessApi.restoreDecision(operatorCode, decisionId),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
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

export function useUpdateAccessPolicyDraft() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.updatePolicyDraft, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function usePublishAccessPolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, versionId }: { operatorCode: string; versionId: string }) => deviceAccessApi.publishPolicy(operatorCode, versionId),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}

export function useAccessPolicyDifference() {
  return useMutation({
    mutationFn: ({ operatorCode, versionId, baseVersionId }: { operatorCode: string; versionId: string; baseVersionId?: string }) =>
      deviceAccessApi.getPolicyDifference(operatorCode, versionId, baseVersionId),
  });
}

export function useRollbackAccessPolicy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, versionId }: { operatorCode: string; versionId: string }) =>
      deviceAccessApi.rollbackPolicy(operatorCode, versionId),
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

export function useAccessEntries(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; productName?: string; entryType?: AccessListType; status?: 'active' | 'disabled' }) {
  return useQuery({ queryKey: [...rootKey, 'entries', params], queryFn: () => deviceAccessApi.listEntries(params), enabled: Boolean(params.operatorCode) });
}

export function useUpsertAccessEntries() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.upsertEntries, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useDisableAccessEntries() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.disableEntries, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useAccessListImports(params: { operatorCode: string; page: number; pageSize: number }) {
  return useQuery({ queryKey: [...rootKey, 'imports', params], queryFn: () => deviceAccessApi.listAccessListImports(params), enabled: Boolean(params.operatorCode) });
}

export function useRuleDimensionImports(params: { operatorCode: string; policyVersionId: string; ruleId: string; dimension: RuleDimension; page: number; pageSize: number; enabled?: boolean }) {
  return useQuery({
    queryKey: [...rootKey, 'rule-dimension-imports', params],
    queryFn: () => deviceAccessApi.listRuleDimensionImports(params),
    enabled: Boolean(params.enabled && params.operatorCode && params.policyVersionId && params.ruleId),
  });
}

export function usePreviewAccessListImport() {
  return useMutation({ mutationFn: deviceAccessApi.previewAccessListImport });
}

export function usePreviewRuleDimensionImport() {
  return useMutation({ mutationFn: deviceAccessApi.previewRuleDimensionImport });
}

export function useClearRuleDimension() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.clearRuleDimension, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useCommitAccessListImport() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.commitAccessListImport, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useRollbackAccessListImport() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.rollbackAccessListImport, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export type AccessListImportFormValues = {
  entryType: AccessListType;
  mode: ImportMode;
  failurePolicy: ImportFailurePolicy;
};

export function useUpsertAccessEntry() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.upsertEntry, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useAccessCandidates(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; reviewStatus?: string; candidateId?: string }) {
  return useQuery({ queryKey: [...rootKey, 'candidates', params], queryFn: () => deviceAccessApi.listCandidates(params), enabled: Boolean(params.operatorCode) });
}

export function useReviewAccessCandidate() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: deviceAccessApi.reviewCandidate, onSuccess: () => invalidateDeviceAccessQueries(qc) });
}

export function useAccessActions(params: { operatorCode: string; page: number; pageSize: number; serialNumber?: string; status?: ActionStatus } & AccessAuditFilters) {
  return useQuery({ queryKey: [...rootKey, 'actions', params], queryFn: () => deviceAccessApi.listActions(params), enabled: Boolean(params.operatorCode) });
}

export function useAccessActionAttempts(operatorCode: string, actionId?: string) {
  return useQuery({
    queryKey: [...rootKey, 'actions', actionId, 'attempts'],
    queryFn: () => deviceAccessApi.listActionAttempts(operatorCode, actionId!),
    enabled: Boolean(operatorCode && actionId),
  });
}

export function useRetryAccessAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ operatorCode, actionId, reason }: { operatorCode: string; actionId: string; reason: string }) =>
      deviceAccessApi.retryAction(operatorCode, actionId, reason),
    onSuccess: () => invalidateDeviceAccessQueries(qc),
  });
}
