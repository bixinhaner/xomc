import type { SystemLicense } from '@core/services/api/systemLicenseApi';
import { filterDeviceStandardOptionsByLicense } from '@core/utils/licenseFeatures';

export const rebootRecordDeviceTypeOptions = [
  { label: 'eNB', value: 'eNB' },
  { label: 'gNB', value: 'gNB' },
  { label: 'GSM', value: 'GSM' },
  { label: 'UPS', value: 'UPS' },
] as const;

export function getRebootRecordDeviceTypeOptions(
  license?: SystemLicense | null,
  isLicenseLoading = false,
): Array<{ label: string; value: string }> {
  return filterDeviceStandardOptionsByLicense(rebootRecordDeviceTypeOptions, license, isLicenseLoading);
}
