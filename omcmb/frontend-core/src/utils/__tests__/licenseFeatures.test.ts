import { describe, expect, it } from 'vitest';

import type { SystemLicense } from '../../services/api/systemLicenseApi';
import {
  filterDeviceScopedItemsByLicense,
  filterDeviceStandardOptionsByLicense,
  getLicenseControlledDeviceStandard,
  getLicenseControlledDeviceStandardFromScope,
  isDeviceStandardLicensed,
  isDeviceStandardValueVisibleByLicense,
  isUPSLicensed,
} from '../licenseFeatures';

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

describe('isUPSLicensed', () => {
  it('accepts normalized feature objects', () => {
    expect(isUPSLicensed(license({
      features: [{ featureCode: 'CODE_UPS', nameZh: 'UPS', nameEn: 'UPS', source: 'raw_code', recognized: true, licensed: true }],
    }))).toBe(true);
  });

  it('accepts legacy code/id and authorization tree shapes', () => {
    expect(isUPSLicensed(license({ legacy_feature_codes: ['CODE_UPS'] }))).toBe(true);
    expect(isUPSLicensed(license({ legacy_feature_ids: ['81'] }))).toBe(true);
    expect(isUPSLicensed(license({ authorization_tree: { UPS: { Monitor: 'All' } } }))).toBe(true);
    expect(isUPSLicensed(license({ authorization_tree: { UPS: 'All' } }))).toBe(true);
  });

  it('rejects absent or expired UPS authorization', () => {
    expect(isUPSLicensed(license({ legacy_feature_codes: ['CODE_ENB_MONITOR'] }))).toBe(false);
    expect(isUPSLicensed(license({ legacy_feature_codes: ['CODE_UPS'] }, true))).toBe(false);
    expect(isUPSLicensed(null)).toBe(false);
  });
});

describe('device standard license helpers', () => {
  const unlicensed = license({ legacy_feature_codes: ['CODE_ENB_MONITOR'] });
  const upsLicensed = license({ legacy_feature_codes: ['CODE_UPS'] });

  it('detects controlled device standards from exact values and product-class prefixes', () => {
    expect(getLicenseControlledDeviceStandard('ups')).toBe('UPS');
    expect(getLicenseControlledDeviceStandard('UPS_M3_BMU')).toBeUndefined();
    expect(getLicenseControlledDeviceStandard('UPS_M3_BMU', { matchProductClassPrefix: true })).toBe('UPS');
    expect(getLicenseControlledDeviceStandardFromScope({ productClass: 'UPS_M3_BMU' })).toBe('UPS');
    expect(getLicenseControlledDeviceStandardFromScope({ patterns: ['UPS.*'] })).toBe('UPS');
    expect(getLicenseControlledDeviceStandardFromScope({ loadedFrom: 'datamodels/alarm/UPS.xml' })).toBe('UPS');
  });

  it('uses the generic license check for controlled standards', () => {
    expect(isDeviceStandardLicensed(upsLicensed, 'UPS')).toBe(true);
    expect(isDeviceStandardLicensed(unlicensed, 'UPS')).toBe(false);
    expect(isDeviceStandardValueVisibleByLicense('UPS', unlicensed, false)).toBe(false);
    expect(isDeviceStandardValueVisibleByLicense('LTE', unlicensed, false)).toBe(true);
    expect(isDeviceStandardValueVisibleByLicense('UPS', unlicensed, true)).toBe(true);
  });

  it('filters controlled select options and scoped data rows', () => {
    const options = [
      { label: 'LTE', value: 'lte' },
      { label: 'UPS', value: 'ups' },
    ];
    expect(filterDeviceStandardOptionsByLicense(options, unlicensed, false)).toEqual([options[0]]);
    expect(filterDeviceStandardOptionsByLicense(options, unlicensed, true)).toEqual(options);
    expect(filterDeviceStandardOptionsByLicense(options, upsLicensed, false)).toEqual(options);

    const rows = [
      { name: 'LTE', patterns: ['LTE.*'] },
      { name: 'UPS', patterns: ['UPS.*'] },
      { name: 'FSU', productClass: 'UPS_M3_BMU' },
      { neType: 'UPS', loadedFrom: 'datamodels/UPS.xml' },
    ];
    expect(filterDeviceScopedItemsByLicense(rows, unlicensed, false)).toEqual([rows[0]]);
    expect(filterDeviceScopedItemsByLicense(rows, upsLicensed, false)).toEqual(rows);
  });
});
