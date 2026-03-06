import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { License } from '@/mock/data/license';
import type { PageRequest } from '@/types/pagination';
import { licenseService } from '@/mock/services/licenseService';

export function useLicenses(
  params: { status?: string; licenseType?: string; deviceType?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['licenses', 'list', params],
    queryFn: () => licenseService.getList(params),
  });
}

export function useLicenseById(id: string) {
  return useQuery({
    queryKey: ['licenses', 'detail', id],
    queryFn: () => licenseService.getById(id),
    enabled: Boolean(id),
  });
}

export function useLicenseSummary() {
  return useQuery({
    queryKey: ['licenses', 'summary'],
    queryFn: () => licenseService.getSummary(),
    refetchInterval: 60000,
  });
}

export function useActivateLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (licenseCode: string) => licenseService.activate(licenseCode),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useRevokeLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => licenseService.revoke(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useImportLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<License, 'id'>) => licenseService.importLicense(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}
