import { describe, expect, it } from 'vitest';
import { getTimezoneAliasOptions } from '@core/utils/timezoneAliasConfig';
import { getParamConfigExportFields } from './paramConfigExportFields';

describe('parameter config export fields', () => {
  it.each(['eNB', 'gNB'] as const)('keeps %s dropdown controls aligned with quick settings', (deviceType) => {
    const fields = getParamConfigExportFields(deviceType);
    const dropdownControls = new Set([
      'select', 'multi-select', 'timezone', 'dl-bandwidth', 'ul-bandwidth',
    ]);

    for (const field of fields.filter((item) => dropdownControls.has(item.control ?? ''))) {
      expect(field.options, `${field.id} must expose its quick-setting options`).toBeDefined();
      expect(field.options!.length).toBeGreaterThan(0);
    }

    const timezone = fields.find((field) => field.control === 'timezone');
    expect(timezone?.options?.map((option) => option.value))
      .toEqual(getTimezoneAliasOptions().map((option) => option.value));
  });

  it('does not add an IPSEC display condition to exported notes', () => {
    for (const deviceType of ['eNB', 'gNB'] as const) {
      const ipsecFields = getParamConfigExportFields(deviceType).filter((field) => (
        field.id.includes('IKE') || field.id.includes('ESP') || field.id.includes('TUNNEL')
      ));
      expect(ipsecFields.length).toBeGreaterThan(0);
      expect(ipsecFields.every((field) => field.condition === undefined)).toBe(true);
    }
  });
});
