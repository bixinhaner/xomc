import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { License } from '../../mock/data/license';
import type { PageRequest } from '../../types/pagination';
import { licenseService } from '../../mock/services/licenseService';
import { licenseApi } from '../../services/api/licenseApi';
import { useMock } from '../../services/apiSwitch';

export function useLicenses(
  params: { status?: string; licenseType?: string; deviceType?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['licenses', 'list', params],
    queryFn: () =>
      useMock
        ? licenseService.getList(params)
        : licenseApi.getLicenses(params),
  });
}

export function useLicenseById(id: string) {
  return useQuery({
    queryKey: ['licenses', 'detail', id],
    queryFn: () =>
      useMock
        ? licenseService.getById(id)
        : licenseApi.getLicenseById(id),
    enabled: Boolean(id),
  });
}

export function useLicenseSummary() {
  return useQuery({
    queryKey: ['licenses', 'summary'],
    queryFn: () =>
      useMock
        ? licenseService.getSummary()
        : licenseApi.getLicenseSummary(),
    refetchInterval: 60000,
  });
}

export function useActivateLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (licenseCode: string) =>
      useMock
        ? licenseService.activate(licenseCode)
        : licenseApi.activateLicense(licenseCode),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useRevokeLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock
        ? licenseService.revoke(id)
        : licenseApi.revokeLicense(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}

export function useImportLicense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<License, 'id'>) =>
      useMock
        ? licenseService.importLicense(data)
        : licenseApi.importLicense(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['licenses'] });
    },
  });
}
