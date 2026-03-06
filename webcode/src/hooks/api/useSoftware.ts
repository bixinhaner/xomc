import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { SoftwareVersion, UpgradePlan } from '@/mock/data/software';
import type { PageRequest } from '@/types/pagination';
import { softwareService } from '@/mock/services/softwareService';
import { softwareApi } from '@/services/api/softwareApi';
import { createApiSwitch } from '@/services/apiSwitch';

const api = createApiSwitch(softwareService, softwareApi);

export function useSoftwareVersions(
  params: { deviceType?: string; status?: string; vendor?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'versions', params],
    queryFn: () => api.getVersions(params),
  });
}

export function useSoftwareVersionById(id: string) {
  return useQuery({
    queryKey: ['software', 'versions', 'detail', id],
    queryFn: () => api.getVersionById(id),
    enabled: Boolean(id),
  });
}

export function useUploadSoftwareVersion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<SoftwareVersion, 'id' | 'releaseDate'>) =>
      api.uploadVersion(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useDeleteSoftwareVersions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteVersions(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useUpgradePlans(params: { status?: string } & PageRequest) {
  return useQuery({
    queryKey: ['software', 'plans', params],
    queryFn: () => api.getUpgradePlans(params),
  });
}

export function useUpgradePlanById(id: string) {
  return useQuery({
    queryKey: ['software', 'plans', 'detail', id],
    queryFn: () => api.getUpgradePlanById(id),
    enabled: Boolean(id),
  });
}

export function useCreateUpgradePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<UpgradePlan, 'id' | 'status' | 'progress' | 'successCount' | 'failCount' | 'createdAt' | 'updatedAt'>) =>
      api.createUpgradePlan(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'plans'] });
    },
  });
}

export function useCancelUpgradePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancelUpgradePlan(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'plans'] });
    },
  });
}

export function useUpgradePrecheck() {
  return useMutation({
    mutationFn: ({ deviceSns, versionId }: { deviceSns: string[]; versionId: string }) =>
      api.precheck(deviceSns, versionId),
  });
}
