import { describe, expect, it } from 'vitest';
import { buildPlugAndPlayLicenseListParams } from './licenseListParams';

describe('buildPlugAndPlayLicenseListParams', () => {
  it('lists every pre-provisioned license without a product constraint', () => {
    expect(buildPlugAndPlayLicenseListParams('')).toEqual({
      page: 1,
      pageSize: 1000,
      serialNumber: undefined,
    });
  });

  it('filters licenses only by serial number', () => {
    expect(buildPlugAndPlayLicenseListParams('SN-001')).toEqual({
      page: 1,
      pageSize: 1000,
      serialNumber: 'SN-001',
    });
  });
});
