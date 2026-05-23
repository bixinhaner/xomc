/**
 * useDeviceLicense (T-0165) — React Query hooks for device_licenses.
 *
 * 后端 API：/api/v1/backup/device-licenses/*
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  deviceLicenseApi,
  type DeviceLicense,
  type LicenseListParams,
  type LicenseImportResult,
  type BatchGetLicensesResult,
} from '../../services/api/deviceLicenseApi';

export function useDeviceLicenses(params: LicenseListParams) {
  return useQuery({
    queryKey: ['device-licenses', 'list', params],
    queryFn: () => deviceLicenseApi.list(params),
  });
}

export function useDeviceLicense(sn: string | undefined) {
  return useQuery<DeviceLicense | null>({
    queryKey: ['device-licenses', 'detail', sn],
    queryFn: () => deviceLicenseApi.getBySerialNumber(sn!),
    enabled: Boolean(sn),
  });
}

export function useBatchGetDeviceLicenses(serialNumbers: string[]) {
  return useQuery<BatchGetLicensesResult>({
    queryKey: ['device-licenses', 'batch-get', [...serialNumbers].sort()],
    queryFn: () => deviceLicenseApi.batchGet(serialNumbers),
    enabled: serialNumbers.length > 0,
  });
}

export function useImportDeviceLicenses() {
  const qc = useQueryClient();
  return useMutation<LicenseImportResult, Error, File[]>({
    mutationFn: (files: File[]) => deviceLicenseApi.import(files),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['device-licenses'] });
    },
  });
}

export function useDeleteDeviceLicense() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (sn: string) => deviceLicenseApi.delete(sn),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['device-licenses'] });
    },
  });
}

export function useBatchDeleteDeviceLicenses() {
  const qc = useQueryClient();
  return useMutation<
    { succeeded: string[]; failed: string[] },
    Error,
    string[]
  >({
    mutationFn: (sns: string[]) => deviceLicenseApi.batchDelete(sns),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['device-licenses'] });
    },
  });
}
