import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { SoftwareVersion, UpgradePlan } from '@/mock/data/software';
import type { PageRequest } from '@/types/pagination';
import { softwareService } from '@/mock/services/softwareService';

export function useSoftwareVersions(
  params: { deviceType?: string; status?: string; vendor?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['software', 'versions', params],
    queryFn: () => softwareService.getVersions(params),
  });
}

export function useSoftwareVersionById(id: string) {
  return useQuery({
    queryKey: ['software', 'versions', 'detail', id],
    queryFn: () => softwareService.getVersionById(id),
    enabled: Boolean(id),
  });
}

export function useUploadSoftwareVersion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<SoftwareVersion, 'id' | 'releaseDate'>) =>
      softwareService.uploadVersion(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useDeleteSoftwareVersions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => softwareService.deleteVersions(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'versions'] });
    },
  });
}

export function useUpgradePlans(params: { status?: string } & PageRequest) {
  return useQuery({
    queryKey: ['software', 'plans', params],
    queryFn: () => softwareService.getUpgradePlans(params),
  });
}

export function useUpgradePlanById(id: string) {
  return useQuery({
    queryKey: ['software', 'plans', 'detail', id],
    queryFn: () => softwareService.getUpgradePlanById(id),
    enabled: Boolean(id),
  });
}

export function useCreateUpgradePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<UpgradePlan, 'id' | 'status' | 'progress' | 'successCount' | 'failCount' | 'createdAt' | 'updatedAt'>) =>
      softwareService.createUpgradePlan(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'plans'] });
    },
  });
}

export function useCancelUpgradePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => softwareService.cancelUpgradePlan(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['software', 'plans'] });
    },
  });
}

export function useUpgradePrecheck() {
  return useMutation({
    mutationFn: ({ deviceSns, versionId }: { deviceSns: string[]; versionId: string }) =>
      softwareService.precheck(deviceSns, versionId),
  });
}
