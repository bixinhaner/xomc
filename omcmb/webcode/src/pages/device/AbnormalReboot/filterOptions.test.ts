import { describe, expect, it } from 'vitest';

import { rebootRecordDeviceTypeOptions } from './filterOptions';

describe('rebootRecordDeviceTypeOptions', () => {
  it('includes GSM so reboot records can be filtered by GSM/BSC devices', () => {
    expect(rebootRecordDeviceTypeOptions).toEqual([
      { label: 'eNB', value: 'eNB' },
      { label: 'gNB', value: 'gNB' },
      { label: 'GSM', value: 'GSM' },
    ]);
  });
});
