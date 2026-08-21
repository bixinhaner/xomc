import { describe, expect, it } from 'vitest';

import {
  getRebootRecordDeviceTypeOptions,
  rebootRecordDeviceTypeOptions,
} from './filterOptions';
import type { SystemLicense } from '@core/services/api/systemLicenseApi';

function license(featureList: SystemLicense['featureList'], isExpired = false): SystemLicense {
  return {
    id: 'lic-id',
    licenseId: 'LIC-1',
    licenseType: 'Commercial',
    issuer: null,
    licensee: null,
    issuedAt: '2026-01-01T00:00:00Z',
    expiryDate: null,
    devicesSupport: {},
    featureList,
    rawContent: '',
    signature: null,
    signatureKeyId: null,
    signatureStatus: 'verified',
    uploadedAt: '2026-01-01T00:00:00Z',
    uploadedByUserId: null,
    isCurrent: true,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    isExpired,
  };
}

describe('rebootRecordDeviceTypeOptions', () => {
  it('includes all reboot record device formats', () => {
    expect(rebootRecordDeviceTypeOptions).toEqual([
      { label: 'eNB', value: 'eNB' },
      { label: 'gNB', value: 'gNB' },
      { label: 'GSM', value: 'GSM' },
      { label: 'UPS', value: 'UPS' },
    ]);
  });

  it('filters UPS by license', () => {
    expect(getRebootRecordDeviceTypeOptions(license({ legacy_feature_codes: ['CODE_ENB_MONITOR'] }), false).map((item) => item.value)).toEqual([
      'eNB',
      'gNB',
      'GSM',
    ]);
    expect(getRebootRecordDeviceTypeOptions(license({ legacy_feature_codes: ['CODE_UPS'] }), false).map((item) => item.value)).toContain('UPS');
    expect(getRebootRecordDeviceTypeOptions(undefined, true).map((item) => item.value)).toContain('UPS');
  });
});
